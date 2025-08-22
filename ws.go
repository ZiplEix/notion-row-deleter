package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Progress struct {
	Running    bool  `json:"running"`
	Deleted    int64 `json:"deleted"`
	Total      int   `json:"total"`
	EtaSeconds int   `json:"etaSeconds"`
}

type Hub struct {
	clients    map[*websocket.Conn]bool
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  chan Progress
	last       Progress
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		broadcast:  make(chan Progress, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true
			// Re-broadcast the last state so the new client gets it via the hub loop.
			select {
			case h.broadcast <- h.last:
			default:
				// if buffer is full, drop; next worker update will refresh all clients anyway
			}
		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				c.Close()
				delete(h.clients, c)
			}
		case p := <-h.broadcast:
			h.last = p
			for c := range h.clients {
				if err := c.WriteJSON(p); err != nil {
					c.Close()
					delete(h.clients, c)
				}
			}
		}
	}
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func wsHandler(h *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("ws upgrade:", err)
			return
		}
		h.register <- conn
		// Reader goroutine to detect the closure on the client side
		go func() {
			defer func() { h.unregister <- conn }()
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()
	}
}
