package config

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

func ConnectRedis() {
	RDB = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	if err := RDB.Ping(context.Background()).Err(); err != nil {
		log.Fatal("❌ Redis connect error: ", err)
	}

	log.Println("✅ Redis connected")
}