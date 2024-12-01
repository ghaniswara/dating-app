package helper_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ghaniswara/dating-app/internal"
	"github.com/ghaniswara/dating-app/internal/config"
	"github.com/ghaniswara/dating-app/internal/entity"
	"github.com/ghaniswara/dating-app/pkg/path"
	"github.com/go-redis/redis"
	"github.com/golang-migrate/migrate/v4"
	migratePostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestServerResources holds resources needed for test server setup
type TestServerResources struct {
	Cancel        context.CancelFunc
	Config        *config.Config
	Pool          *dockertest.Pool
	DBResource    *dockertest.Resource
	RedisResource *dockertest.Resource
	Address       string
	ORM           *gorm.DB
}

// setupTestServer sets up the test environment including Docker resources and server
func SetupTestServer(ctx context.Context) (*TestServerResources, error) {
	ctx, cancel := context.WithCancel(ctx)
	var gormDB *gorm.DB

	config, err := config.NewConfig("TEST")
	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not load configuration: %w", err)
	}

	pool, dbResource, redisResource, err := setupDockerResources(config)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not set up Docker resources: %w", err)
	}
	var dsn string
	pool.MaxWait = 10 * time.Second
	if err := pool.Retry(func() error {
		gormDB, dsn, err = connectToPostgres(dbResource, config)
		return err
	}); err != nil {
		cancel()
		return nil, fmt.Errorf("could not connect to postgreSQL: %s", err)
	}

	fmt.Println("ℹ️ Database Connected")

	if err := pool.Retry(func() error {
		_, err = connectToRedis(redisResource)
		return err
	}); err != nil {
		cancel()
		return nil, fmt.Errorf("could not connect to redis: %s", err)
	}

	fmt.Println("ℹ️ Redis Connected")

	// Run migrations and seed data
	dbConnection, err := gormDB.DB()
	if err != nil {
		cancel()
		return nil, err
	}

	err = runMigrationsAndSeeds(dbConnection, dsn)

	if err != nil {
		cancel()
		return nil, err
	}
	// Run the server
	args := []string{"test"}
	go internal.Run(ctx, os.Stdout, args)

	// Wait for server readiness
	if !waitForServer(ctx, config.Get("PORT")) {
		pool.Purge(redisResource)
		pool.Purge(dbResource)
		cancel()
		return nil, fmt.Errorf("server did not start within timeout")
	}

	return &TestServerResources{
		Cancel:        cancel,
		Config:        config,
		Pool:          pool,
		DBResource:    dbResource,
		RedisResource: redisResource,
		ORM:           gormDB,
	}, nil
}

// cleanupTestServer purges Docker resources
func (resources *TestServerResources) CleanupTestServer() {
	if resources == nil {
		return
	}

	// Cancel the context to stop the server
	if resources.Cancel != nil {
		resources.Cancel()
	}

	// Purge Docker resources
	if resources.Pool != nil {
		if resources.DBResource != nil {
			if err := resources.Pool.Purge(resources.DBResource); err != nil {
				log.Printf("Could not purge PostgreSQL: %s", err)
			}
		}

		if resources.RedisResource != nil {
			if err := resources.Pool.Purge(resources.RedisResource); err != nil {
				log.Printf("Could not purge Redis: %s", err)
			}
		}
	}
}

func setupDockerResources(config *config.Config) (*dockertest.Pool, *dockertest.Resource, *dockertest.Resource, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not connect to docker: %s", err)
	}

	dbOptions := &dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "14",
		Env: []string{
			fmt.Sprintf("POSTGRES_USER=%s", config.Get("POSTGRES_USER")),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", config.Get("POSTGRES_PASSWORD")),
			fmt.Sprintf("POSTGRES_DB=%s", config.Get("POSTGRES_DB_NAME")),
		},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432/tcp": {{HostIP: "127.0.0.1", HostPort: fmt.Sprintf("%s/tcp", config.Get("POSTGRES_PORT"))}},
		},
	}
	dbResource, err := pool.RunWithOptions(dbOptions)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not start postgres: %s", err)
	}

	redisResource, err := pool.Run("redis", "7", nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not start redis: %s", err)
	}

	return pool, dbResource, redisResource, nil
}

func connectToPostgres(dbResource *dockertest.Resource, config *config.Config) (*gorm.DB, string, error) {
	hostPort := strings.Split(dbResource.GetHostPort("5432/tcp"), ":")
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		hostPort[0],
		hostPort[1],
		config.Get("POSTGRES_USER"),
		config.Get("POSTGRES_PASSWORD"),
		config.Get("POSTGRES_DB_NAME"))
	var err error
	gormDB, err := gorm.Open(postgres.Open(dsn))

	if err != nil {
		return nil, "", err
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, "", err
	}

	return gormDB, dsn, sqlDB.Ping()
}

func connectToRedis(redisResource *dockertest.Resource) (*redis.Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:" + redisResource.GetPort("6379/tcp"),
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	err := redisClient.Ping().Err()

	return redisClient, err
}

func runMigrationsAndSeeds(db *sql.DB, dsn string) error {

	driver, err := migratePostgres.WithInstance(db, &migratePostgres.Config{})

	if err != nil {
		return err
	}

	basePath, err := os.Getwd()

	if err != nil {
		return err
	}

	migrationPath, err := path.FindRoot(basePath, "migrations", true)
	migrationPath = "file://" + migrationPath + "/migrations"
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationPath,
		"postgres", driver)
	if err != nil {
		return err
	}

	err = m.Up()

	return err
}

func waitForServer(ctx context.Context, port string) bool {
	loopContext, cancelLoopContext := context.WithTimeout(ctx, 120*time.Second)
	defer cancelLoopContext()

	for {
		select {
		case <-loopContext.Done():
			return false
		default:
			req, err := http.Get(fmt.Sprintf("http://localhost:%s/health", port))
			if err != nil {
				log.Printf("Failed to create HTTP request: %s", err)
				time.Sleep(1 * time.Second)
				continue
			}

			if req.StatusCode == http.StatusOK {
				log.Println("✅ Server is ready")
				return true
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func SignUpUser(username, password, email string) (entity.CreateUserRequest, error) {
	reqBody := entity.CreateUserRequest{
		Name:     "testname",
		Username: username,
		Password: password,
		Email:    email,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return entity.CreateUserRequest{}, err
	}

	// Create a new HTTP client
	client := &http.Client{}

	// Make a normal HTTP request
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/v1/auth/sign-up", bytes.NewBuffer(body))
	if err != nil {
		return entity.CreateUserRequest{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return entity.CreateUserRequest{}, err
	}
	defer resp.Body.Close()

	// Assert the response
	if resp.StatusCode != http.StatusOK {
		return entity.CreateUserRequest{}, fmt.Errorf("failed to sign up, status code: %d", resp.StatusCode)
	}

	return reqBody, nil
}
