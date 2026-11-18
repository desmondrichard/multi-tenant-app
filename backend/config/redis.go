package config

import (
	"context"
	"crypto/tls"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

func ConnectRedis() {
	addr := os.Getenv("REDIS_ADDR")
	password := os.Getenv("REDIS_PASSWORD")

	// If no address is provided, skip gracefully (useful for quick local tests)
	if addr == "" {
		log.Println("⚠️ Redis not configured, skipping connection")
		return
	}

	// Upstash requires TLS and a password
	RDB = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	})

	if err := RDB.Ping(context.Background()).Err(); err != nil {
		// We use log.Println instead of log.Fatal here so that if Redis goes down,
		// your whole app doesn't crash. It will just log a warning.
		log.Println("⚠️ Redis connect error: ", err)
		return
	}

	log.Println("✅ Redis connected to Upstash")
}
