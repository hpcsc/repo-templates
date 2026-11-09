package websocket

import (
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/route"
)

var _ route.Routable = (*Handler)(nil)

type Handler struct {
	hub    *Hub
	logger *slog.Logger
}

func NewHandler(hub *Hub, logger *slog.Logger) *Handler {
	return &Handler{
		hub:    hub,
		logger: logger,
	}
}

func (h *Handler) Routes() []*route.Route {
	return []*route.Route{
		route.Public("GET", "/ws", h.serve),
	}
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		h.logger.Warn("websocket upgrade failed", "error", err)
		return
	}
	defer conn.CloseNow()

	h.hub.Register(conn)
	defer h.hub.Unregister(conn)

	ctx := r.Context()
	for {
		if _, _, err := conn.Read(ctx); err != nil {
			return
		}
	}
}
