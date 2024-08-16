package handlers

import (
	"main/internal/services"
	"main/pkg/logger"
	"net/http"

	"github.com/google/uuid"
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

func (s *FindHandler) getParams(r *http.Request) (uuid.UUID, error) {
	cid, err := uuid.Parse(r.URL.Query().Get("cid"))
	if err != nil {
		return uuid.Nil, err
	}
	return cid, nil
}

func (s *FindHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.DPrintf(logger.DError, "Failed to upgrade connection: %v", err)
		return
	}
	defer ws.Close()

	cid, err := s.getParams(r)
	if err != nil {
		logger.DPrintf(logger.DError, "Failed to get params: %v", err)
		return
	}

	s.service.AddClient(ws, cid)
	defer s.service.RemoveClient(ws, cid)

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			break
		}

		if len(message) == 0 {
			s.service.UpdateClient(cid)
			continue
		}
	}
}
