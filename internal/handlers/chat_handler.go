package handlers

import (
	"main/internal/services"
	"net/http"

	"github.com/gorilla/websocket"
)

type ChatHandler struct {
	service  *services.ChatService
	upgrader websocket.Upgrader
}

func NewChatHandler(service *services.ChatService) *ChatHandler {
	return &ChatHandler{
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

func (s *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	s.service.AddClient(ws)

	for {
		var msg services.Message
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			s.service.RemoveClient(ws)
			break
		}
		// Send the newly received message to the broadcast channel
		s.service.BroadcastMessage(msg)
	}
}
