package websocket

import (
	"encoding/json"
	"log"
	"sync"
)

type Hub struct {
	Clients    map[*Client]bool
	Broadcast  chan Event
	Register   chan *Client
	Unregister chan *Client
	mu         sync.Mutex
}

// Global hub instance
var GlobalHub = &Hub{
	Clients:    make(map[*Client]bool),
	Broadcast:  make(chan Event, 256),
	Register:   make(chan *Client),
	Unregister: make(chan *Client),
}

// Run must be started once with: go ws.GlobalHub.Run()
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()
			log.Println("✅ WebSocket Client Connected")

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Println("❌ WebSocket Client Disconnected")

		case evt := <-h.Broadcast:
			data, err := json.Marshal(evt)
			if err != nil {
				continue
			}

			h.mu.Lock()
			for client := range h.Clients {
				// Only the same workspace
				if evt.OrgID != "" && client.OrgID != evt.OrgID {
					continue
				}
				// Only the targeted user (e.g. FORCE_LOGOUT)
				if evt.TargetUser != "" && client.UserID != evt.TargetUser {
					continue
				}

				select {
				case client.Send <- data:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}
