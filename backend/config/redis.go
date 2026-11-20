package config

import (
	"context"
	"crypto/tls"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// RDB is nil when Redis is not configured or not reachable.
// Code that uses it must check for nil (pub/sub falls back to local delivery).
var RDB *redis.Client

func ConnectRedis() {
	addr := os.Getenv("REDIS_ADDR")
	password := os.Getenv("REDIS_PASSWORD")

	// If no address is provided, skip gracefully (useful for quick local tests)
	if addr == "" {
		log.Println("⚠️ Redis not configured (REDIS_ADDR empty) - real-time events will be local only")
		return
	}

	opts := &redis.Options{
		Addr:     addr,
		Password: password,
	}

	// Upstash / managed Redis needs TLS. For a local Redis set REDIS_TLS=false
	if os.Getenv("REDIS_TLS") != "false" {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		// Don't crash the app if Redis is down - just run without pub/sub
		log.Println("⚠️ Redis connect error:", err, "- real-time events will be local only")
		client.Close()
		return
	}

	RDB = client
	log.Println("✅ Redis connected")
}
