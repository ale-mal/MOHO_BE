package services

import "github.com/gorilla/websocket"

type ChatService struct {
	clients   map[*websocket.Conn]bool
	broadcast chan Message
}

type Message struct {
	Username string `json:"username"`
	Message  string `json:"message"`
}

func (s *ChatService) AddClient(client *websocket.Conn) {
	s.clients[client] = true
}

func (s *ChatService) RemoveClient(client *websocket.Conn) {
	delete(s.clients, client)
}

func (s *ChatService) BroadcastMessage(msg Message) {
	s.broadcast <- msg
}

func (s *ChatService) ticker() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-s.broadcast
		// Send it out to every client that is currently connected
		for client := range s.clients {
			err := client.WriteJSON(msg)
			if err != nil {
				client.Close()
				delete(s.clients, client)
			}
		}
	}
}

func NewChatService() *ChatService {
	s := &ChatService{}
	s.clients = make(map[*websocket.Conn]bool)
	s.broadcast = make(chan Message)
	go s.ticker()
	return s
}
