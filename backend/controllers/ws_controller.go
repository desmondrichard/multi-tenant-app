package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/yourname/multi-tenant-pm/utils"
	ws "github.com/yourname/multi-tenant-pm/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // In production, restrict this to your frontend URL
	},
}

func HandleWS(c *gin.Context) {
	// Get token from query param (since browsers can't easily set headers for WS)
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		return
	}

	// Parse token to get OrgID
	claims, err := utils.ParseToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &ws.Client{
		Hub:   ws.GlobalHub,
		Conn:  conn,
		Send:  make(chan []byte, 256),
		OrgID: claims.OrganizationID,
	}

	client.Hub.Register <- client

	go client.WritePump()
	go client.ReadPump()
}
