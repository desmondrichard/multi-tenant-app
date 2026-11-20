package websocket

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/yourname/multi-tenant-pm/config"
)

const defaultChannel = "taskflow:events"

func channelName() string {
	if v := os.Getenv("REDIS_CHANNEL"); v != "" {
		return v
	}
	return defaultChannel
}

// Publish sends an event to ALL backend instances through Redis.
// Every instance (including this one) receives it in StartSubscriber and
// delivers it to its own WebSocket clients.
//
// If Redis is not available, the event is delivered to the clients of this
// instance only, so the app keeps working in single-server mode.
//
// It never blocks the HTTP request: the work runs in a goroutine.
func Publish(evt Event) {
	go func() {
		if config.RDB != nil {
			data, err := json.Marshal(evt)
			if err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				if err = config.RDB.Publish(ctx, channelName(), data).Err(); err == nil {
					return // delivered through Redis
				}
			}
			log.Println("⚠️ Redis publish failed, delivering locally:", err)
		}
		GlobalHub.Broadcast <- evt
	}()
}

// StartSubscriber listens to the Redis channel and hands every event to the
// local Hub. Call it once at startup (after config.ConnectRedis()).
func StartSubscriber() {
	if config.RDB == nil {
		log.Println("ℹ️ Redis pub/sub disabled - events stay on this instance")
		return
	}

	go func() {
		ctx := context.Background()
		for {
			sub := config.RDB.Subscribe(ctx, channelName())

			// Wait for the subscription to be confirmed
			if _, err := sub.Receive(ctx); err != nil {
				log.Println("⚠️ Redis subscribe error, retrying in 5s:", err)
				sub.Close()
				time.Sleep(5 * time.Second)
				continue
			}
			log.Println("✅ Redis pub/sub subscribed to channel:", channelName())

			for msg := range sub.Channel() {
				var evt Event
				if err := json.Unmarshal([]byte(msg.Payload), &evt); err != nil {
					log.Println("⚠️ Bad pub/sub message:", err)
					continue
				}
				GlobalHub.Broadcast <- evt
			}

			// Channel closed (connection lost) - subscribe again
			sub.Close()
			time.Sleep(time.Second)
		}
	}()
}
