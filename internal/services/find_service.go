package services

import (
	"main/pkg/logger"
	"main/pkg/lru"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const connTimeout = 10 * time.Second
const gameLimit = 2

type FindRequest struct {
	cid      uuid.UUID
	username string
	lastAck  time.Time
}

type FindService struct {
	requests    *lru.LRUList[uuid.UUID, FindRequest]
	clients     map[uuid.UUID]*websocket.Conn
	mu          sync.Mutex
	gameService *GameService
}

func (s *FindService) AddClient(client *websocket.Conn, cid uuid.UUID, username string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[cid] = client
	req := FindRequest{
		cid:      cid,
		username: username,
		lastAck:  time.Now(),
	}
	s.requests.Put(cid, req)
	logger.DPrintf(logger.DInfo, "Added client %v", cid)
}

func (s *FindService) RemoveClient(client *websocket.Conn, cid uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, cid)
	s.requests.Remove(cid)
	logger.DPrintf(logger.DInfo, "Removed client %v", cid)
}

func (s *FindService) UpdateClient(cid uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.requests.Get(cid)
	if !ok {
		return
	}
	req.lastAck = time.Now()
	s.requests.Put(cid, req)
}

func (s *FindService) cleanExpired() {
	for {
		s.mu.Lock()
		for s.requests.Len() > 0 {
			cid, req, ok := s.requests.Back()
			if !ok {
				logger.DPrintf(logger.DError, "Failed to get last request")
				break
			}
			if time.Since(req.lastAck) > connTimeout {
				ok := s.requests.Remove(cid)
				if !ok {
					logger.DPrintf(logger.DError, "Failed to remove request %v", cid)
					break
				}
			} else {
				break
			}
		}
		s.mu.Unlock()
		time.Sleep(connTimeout)
	}
}

func (s *FindService) matchClients() {
	for {
		s.mu.Lock()
		for s.requests.Len() >= gameLimit {
			cids := make([]uuid.UUID, 0, gameLimit)
			for i := 0; i < gameLimit; i++ {
				cid, _, ok := s.requests.Pop_front()
				if !ok {
					logger.DPrintf(logger.DError, "Failed to get request")
					break
				}
				cids = append(cids, cid)
			}
			if len(cids) != gameLimit {
				break
			}
			logger.DPrintf(logger.DInfo, "Creating game for %v", cids)
			sessionId := uuid.New()
			s.gameService.CreateSession(sessionId)

			// send "found:<sessionId>" message to clients
			for _, cid := range cids {
				client, ok := s.clients[cid]
				if !ok {
					logger.DPrintf(logger.DError, "Failed to get client %v", cid)
					continue
				}
				err := client.WriteMessage(websocket.TextMessage, []byte("found:"+sessionId.String()))
				if err != nil {
					logger.DPrintf(logger.DError, "Failed to send message to client %v: %v", cid, err)
					continue
				}
			}
		}
		s.mu.Unlock()
		time.Sleep(time.Second)
	}
}

func NewFindService(gameService *GameService) *FindService {
	s := &FindService{}
	s.mu = sync.Mutex{}
	s.clients = make(map[uuid.UUID]*websocket.Conn)
	s.requests = lru.NewLRUList[uuid.UUID, FindRequest]()
	s.gameService = gameService
	go s.matchClients()
	go s.cleanExpired()
	return s
}
