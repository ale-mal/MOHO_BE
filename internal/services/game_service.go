package services

import (
	"fmt"
	"main/pkg/logger"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type GameSession struct {
	clients   map[*websocket.Conn]bool
	broadcast chan Message
	mu        sync.Mutex
}

func (s *GameSession) ticker() {
	for {
		msg := <-s.broadcast
		s.mu.Lock()
		for client := range s.clients {
			err := client.WriteJSON(msg)
			if err != nil {
				client.Close()
				delete(s.clients, client)
			}
		}
		s.mu.Unlock()
	}
}

type GameService struct {
	sessions map[uuid.UUID]*GameSession
	mu       sync.Mutex
}

func NewGameService() *GameService {
	return &GameService{
		sessions: make(map[uuid.UUID]*GameSession),
	}
}

func (s *GameService) CreateSession(id uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := &GameSession{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan Message, 16),
	}
	s.sessions[id] = session
	go session.ticker()
	logger.DPrintf(logger.DInfo, "Created game session %v", id)
}

func (s *GameService) JoinSession(id uuid.UUID, ws *websocket.Conn) error {
	s.mu.Lock()
	session, ok := s.sessions[id]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("session %v not found", id)
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	session.clients[ws] = true
	logger.DPrintf(logger.DInfo, "Client joined session %v", id)
	return nil
}

func (s *GameService) LeaveSession(id uuid.UUID, ws *websocket.Conn) {
	s.mu.Lock()
	session, ok := s.sessions[id]
	s.mu.Unlock()
	if !ok {
		return
	}
	session.mu.Lock()
	delete(session.clients, ws)
	empty := len(session.clients) == 0
	session.mu.Unlock()
	if empty {
		s.mu.Lock()
		delete(s.sessions, id)
		s.mu.Unlock()
		logger.DPrintf(logger.DInfo, "Deleted empty session %v", id)
	}
}

func (s *GameService) BroadcastToSession(id uuid.UUID, msg Message) {
	s.mu.Lock()
	session, ok := s.sessions[id]
	s.mu.Unlock()
	if !ok {
		return
	}
	session.broadcast <- msg
}
