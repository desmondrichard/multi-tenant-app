package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/yourname/multi-tenant-pm/config"
	"github.com/yourname/multi-tenant-pm/routes"
	ws "github.com/yourname/multi-tenant-pm/websocket"
)

func main() {
	_ = godotenv.Load()

	config.ConnectMongo()
	config.ConnectRedis()

	// Real-time: start the WebSocket hub and the Redis pub/sub listener
	go ws.GlobalHub.Run()
	ws.StartSubscriber()

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{os.Getenv("CLIENT_ORIGIN")},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	routes.Setup(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("🚀 Listening on :" + port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}