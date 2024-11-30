package test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/ory/dockertest"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/ghaniswara/dating-app/internal"
	"github.com/ghaniswara/dating-app/internal/config"
)

var gormDB *gorm.DB
var redisClient *redis.Client

func TestMain(m *testing.M) {
	ctx := context.Background()

	config, err := config.NewConfig("TEST")
	if err != nil {
		log.Fatalf("Could not load configuration: %s", err)
	}

	pool, dbResource, redisResource, err := setupDockerResources(config)
	if err != nil {
		log.Fatalf("Could not set up Docker resources: %s", err)
	}

	// Run migrations and seed data
	runMigrationsAndSeeds(gormDB)

	// Run the server
	args := []string{"dev"}
	internal.Run(ctx, os.Stdout, args)

	// Run tests
	code := m.Run()

	// Cleanup
	if err := pool.Purge(dbResource); err != nil {
		log.Fatalf("Could not purge PostgreSQL: %s", err)
	}
	if err := pool.Purge(redisResource); err != nil {
		log.Fatalf("Could not purge Redis: %s", err)
	}

	os.Exit(code)
}

func setupDockerResources(config *config.Config) (*dockertest.Pool, *dockertest.Resource, *dockertest.Resource, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Could not connect to Docker: %s", err)
	}

	dbResource, err := pool.Run("postgres", "14", []string{
		fmt.Sprintf("POSTGRES_USER=%s", config.Get("POSTGRES_USER")),
		fmt.Sprintf("POSTGRES_PASSWORD=%s", config.Get("POSTGRES_PASSWORD")),
		fmt.Sprintf("POSTGRES_DB=%s", config.Get("POSTGRES_DB_NAME")),
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Could not start PostgreSQL: %s", err)
	}

	redisResource, err := pool.Run("redis", "7", nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Could not start Redis: %s", err)
	}

	pool.MaxWait = 120 * time.Second
	if err := pool.Retry(func() error {
		return connectToPostgres(dbResource, config)
	}); err != nil {
		return nil, nil, nil, fmt.Errorf("Could not connect to PostgreSQL: %s", err)
	}

	if err := pool.Retry(func() error {
		return connectToRedis(redisResource)
	}); err != nil {
		return nil, nil, nil, fmt.Errorf("Could not connect to Redis: %s", err)
	}

	return pool, dbResource, redisResource, nil
}

func connectToPostgres(dbResource *dockertest.Resource, config *config.Config) error {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		dbResource.GetHostPort("5432/tcp"),
		config.Get("POSTGRES_USER"),
		config.Get("POSTGRES_PASSWORD"),
		config.Get("POSTGRES_DB_NAME"))
	var err error
	gormDB, err = gorm.Open(postgres.Open(dsn))

	if err != nil {
		return err
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Ping()
}

func connectToRedis(redisResource *dockertest.Resource) error {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisResource.GetPort("6379/tcp"),
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	err := redisClient.Ping().Err()
	if err != nil {
		return err
	}

	return nil
}

func runMigrationsAndSeeds(gormDB *gorm.DB) {
	// Run database migrations and seed data
}
