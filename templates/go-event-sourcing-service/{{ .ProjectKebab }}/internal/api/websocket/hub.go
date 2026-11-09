package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/coder/websocket"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}
	logger  *slog.Logger
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]struct{}),
		logger:  logger,
	}
}

func (h *Hub) Register(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()
	h.logger.Info("websocket client connected", "total", len(h.clients))
}

func (h *Hub) Unregister(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	h.logger.Info("websocket client disconnected", "total", len(h.clients))
}

func (h *Hub) Broadcast(ctx context.Context, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("failed to marshal websocket message", "error", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range h.clients {
		if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
			h.logger.Warn("failed to write to websocket client", "error", err)
		}
	}
}

func (h *Hub) Shutdown(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		conn.Close(websocket.StatusGoingAway, "server shutting down")
	}
	h.clients = make(map[*websocket.Conn]struct{})
	h.logger.Info("websocket hub shut down")
}

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}