package redisClient

import "github.com/go-redis/redis"

type RedisClient struct {
	Client *redis.Client
}

func NewRedis(redisClient *redis.Client) *RedisClient {
	return &RedisClient{Client: redisClient}
}
