package handlers

import (
	"main/internal/services"
	"net/http"

	"github.com/gorilla/websocket"
)

type FindHandler struct {
	service  *services.FindService
	upgrader websocket.Upgrader
}

func NewFindHandler(service *services.FindService) *FindHandler {
	return &FindHandler{
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

func (s *FindHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer ws.Close()
}
