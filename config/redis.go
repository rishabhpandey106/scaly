package config

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var Ctx = context.Background()

func InitRedis(connectionString string) *redis.Client {
	opt, err := redis.ParseURL(connectionString)
	if err != nil {
		log.Fatal("Failed to parse Redis URL:", err)
	}

	client := redis.NewClient(opt)

	_, err = client.Ping(Ctx).Result()
	if err != nil {
		log.Fatal("Redis connection failed:", err)
	}

	log.Println("Redis connected")
	return client
}
