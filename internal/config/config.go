package config

import (
	"os"
	"strings"

	"github.com/ghaniswara/dating-app/pkg/path"
	"github.com/joho/godotenv"
)

type IConfig interface {
	Get(key string) string
}
type Config struct {
	Key map[string]string
	Env string
}

func NewConfig(env string) (*Config, error) {
	env = strings.ToUpper(env)

	basePath, err := os.Getwd()

	if err != nil {
		return nil, err
	}

	path, err := path.FindRoot(basePath, ".env", false)

	if err != nil {
		return nil, err
	}

	err = godotenv.Load(path + "/.env")

	if err != nil {
		return nil, err
	}

	return &Config{
		Key: map[string]string{
			"POSTGRES_DB_NAME":  getEnv(env+"_POSTGRES_DB_NAME", ""),
			"POSTGRES_USER":     getEnv(env+"_POSTGRES_USER", ""),
			"POSTGRES_PASSWORD": getEnv(env+"_POSTGRES_PASSWORD", ""),
			"POSTGRES_HOST":     getEnv(env+"_POSTGRES_HOST", ""),
			"POSTGRES_PORT":     getEnv(env+"_POSTGRES_PORT", ""),
			"REDIS_HOST":        getEnv(env+"_REDIS_HOST", ""),
			"REDIS_PORT":        getEnv(env+"_REDIS_PORT", ""),
			"JWT_SECRET":        getEnv(env+"_JWT_SECRET", ""),
			"PORT":              getEnv("PORT", "8080"),
		},
		Env: env,
	}, nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func (c *Config) Get(key string) string {
	return c.Key[key]
}
