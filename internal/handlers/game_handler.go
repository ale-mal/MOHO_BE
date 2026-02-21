package handlers

import (
	"main/internal/services"
	"main/pkg/logger"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type GameHandler struct {
	service  *services.GameService
	upgrader websocket.Upgrader
}

func NewGameHandler(service *services.GameService) *GameHandler {
	return &GameHandler{
		service: service,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *GameHandler) getParams(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	sessionId, err := uuid.Parse(r.URL.Query().Get("sessionId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	cid, err := uuid.Parse(r.URL.Query().Get("cid"))
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return sessionId, cid, nil
}

func (h *GameHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.DPrintf(logger.DError, "Failed to upgrade connection: %v", err)
		return
	}
	defer ws.Close()

	sessionId, _, err := h.getParams(r)
	if err != nil {
		logger.DPrintf(logger.DError, "Failed to get params: %v", err)
		return
	}

	if err := h.service.JoinSession(sessionId, ws); err != nil {
		logger.DPrintf(logger.DError, "Failed to join session %v: %v", sessionId, err)
		return
	}
	defer h.service.LeaveSession(sessionId, ws)

	for {
		var msg services.Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			break
		}
		h.service.BroadcastToSession(sessionId, msg)
	}
}
