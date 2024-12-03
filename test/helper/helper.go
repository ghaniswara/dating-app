package helper_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ghaniswara/dating-app/internal"
	"github.com/ghaniswara/dating-app/internal/config"
	"github.com/ghaniswara/dating-app/internal/entity"
	"github.com/ghaniswara/dating-app/pkg/http_util"
	"github.com/ghaniswara/dating-app/pkg/path"
	"github.com/go-faker/faker/v4"
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
	Redis         *redis.Client
}

// setupTestServer sets up the test environment including Docker resources and server
func SetupTestServer(ctx context.Context) (*TestServerResources, error) {
	ctx, cancel := context.WithCancel(ctx)
	var gormDB *gorm.DB
	var redisClient *redis.Client
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
		redisClient, err = connectToRedis(redisResource)
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
		Redis:         redisClient,
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

	redisOptions := &dockertest.RunOptions{
		Repository: "redis",
		Tag:        "7",
		PortBindings: map[docker.Port][]docker.PortBinding{
			"6379/tcp": {{HostIP: "127.0.0.1", HostPort: fmt.Sprintf("%s/tcp", config.Get("REDIS_PORT"))}},
		},
	}

	redisResource, err := pool.RunWithOptions(redisOptions)

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

func runMigrationsAndSeeds(db *sql.DB, _ string) error {

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

func SignUpUser(t *testing.T, username, password, email string) (entity.SignUpResponse, error) {
	reqBody := entity.CreateUserRequest{
		Name:     "testname",
		Username: username,
		Password: password,
		Email:    email,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request body: %v", err)
	}

	// Create a new HTTP client
	client := &http.Client{}

	// Make a normal HTTP request
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/v1/auth/sign-up", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Assert the response
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	response := http_util.HTTPResponse[entity.SignUpResponse]{}
	response, err = http_util.DecodeBody[http_util.HTTPResponse[entity.SignUpResponse]](bodyBytes, response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	return response.Data, nil
}

func SignInUser(t *testing.T, email, username, password string) (token string, err error) {
	reqBody := entity.SignInRequest{
		Email:    email,
		Username: username,
		Password: password,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request body: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/v1/auth/sign-in", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body) // Read the response body

	response := http_util.HTTPResponse[entity.SignInResponse]{}
	response, err = http_util.DecodeBody[http_util.HTTPResponse[entity.SignInResponse]](bodyBytes, response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	return response.Data.Token, nil
}

func PopulateUsers(db *gorm.DB, count int) (users []entity.User, err error) {
	for i := 0; i < count; i++ {
		user := entity.User{
			Name:      faker.Name(),
			Email:     faker.Email(),
			Username:  faker.Username(),
			Password:  faker.Password(),
			IsPremium: false,
		}
		db.Create(&user)
		users = append(users, user)
	}
	return users, nil
}
