package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Key map[string]string
	Env string
}

func NewConfig(env string) (*Config, error) {
	env = strings.ToUpper(env)
	err := godotenv.Load("../../.env." + env)
	if err != nil {
		return nil, err
	}

	return &Config{
		Key: map[string]string{
			"POSTGRES_DB_NAME":  os.Getenv(env + "_POSTGRES_DB_NAME"),
			"POSTGRES_USER":     os.Getenv(env + "_POSTGRES_USER"),
			"POSTGRES_PASSWORD": os.Getenv(env + "_POSTGRES_PASSWORD"),
			"POSTGRES_HOST":     os.Getenv(env + "_POSTGRES_HOST"),
			"POSTGRES_PORT":     os.Getenv(env + "_POSTGRES_PORT"),
			"REDIS_HOST":        os.Getenv(env + "_REDIS_HOST"),
			"REDIS_PORT":        os.Getenv(env + "_REDIS_PORT"),
			"JWT_SECRET":        os.Getenv(env + "_JWT_SECRET"),
		},
		Env: env,
	}, nil
}

func (c *Config) Get(key string) string {
	return c.Key[key]
}
