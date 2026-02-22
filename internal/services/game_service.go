package services

import (
	"encoding/json"
	"fmt"
	"main/pkg/logger"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Physics constants
const (
	tickRate      = 30                    // Hz
	tickDuration  = time.Second / tickRate // ~33.3 ms
	gravity       = 1800.0               // px/s²
	jumpVelocity  = -650.0               // px/s
	moveSpeed     = 220.0                // px/s
	groundY       = 460.0               // top of ground
	worldWidth    = 800.0
	worldHeight   = 500.0
	playerW       = 40.0
	playerH       = 40.0
	stompThresh   = 20.0 // px
	bounceVY      = -300.0
	respawnDelay  = 1500 * time.Millisecond
)

var spawnPositions = [2][2]float64{
	{60.0, groundY},
	{700.0, groundY},
}

// InputMessage is sent by the client every frame.
type InputMessage struct {
	Left  bool `json:"left"`
	Right bool `json:"right"`
	Jump  bool `json:"jump"`
}

// PlayerSnap is the per-player data in a StateSnapshot.
type PlayerSnap struct {
	CID       string  `json:"cid"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	VX        float64 `json:"vx"`
	VY        float64 `json:"vy"`
	Alive     bool    `json:"alive"`
	SpawnSide int     `json:"spawn_side"`
	Kills     int     `json:"kills"`
}

// StateSnapshot is broadcast to all clients each tick.
type StateSnapshot struct {
	Tick    uint64       `json:"tick"`
	Players []PlayerSnap `json:"players"`
}

type InputState struct {
	Left  bool
	Right bool
	Jump  bool
}

type PlayerState struct {
	CID       uuid.UUID
	X, Y      float64
	VX, VY    float64
	Alive     bool
	SpawnSide int
	Kills     int
}

type GameSession struct {
	players   [2]*PlayerState
	inputs    [2]InputState
	playerIdx map[uuid.UUID]int
	clients   [2]*websocket.Conn
	broadcast chan []byte
	mu        sync.Mutex
}

func newGameSession() *GameSession {
	return &GameSession{
		playerIdx: make(map[uuid.UUID]int),
		broadcast: make(chan []byte, 4),
	}
}

// physicsTick runs the 30 Hz authoritative simulation for a session.
func (s *GameSession) physicsTick(stopCh <-chan struct{}) {
	ticker := time.NewTicker(tickDuration)
	defer ticker.Stop()
	var tick uint64
	respawnTimers := [2]*time.Timer{}

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
		}
		tick++
		dt := tickDuration.Seconds()

		s.mu.Lock()

		// Step each player
		for i, p := range s.players {
			if p == nil || !p.Alive {
				continue
			}
			inp := s.inputs[i]

			// Horizontal
			if inp.Left {
				p.VX = -moveSpeed
			} else if inp.Right {
				p.VX = moveSpeed
			} else {
				p.VX = 0
			}

			// Jump (only when on ground)
			onGround := p.Y >= groundY
			if inp.Jump && onGround {
				p.VY = jumpVelocity
				// clear jump so a held key doesn't double-jump next tick
				s.inputs[i].Jump = false
			}

			// Gravity
			p.VY += gravity * dt

			// Integrate
			p.X += p.VX * dt
			p.Y += p.VY * dt

			// Ground clamp
			if p.Y >= groundY {
				p.Y = groundY
				p.VY = 0
			}

			// Wall clamp
			if p.X < 0 {
				p.X = 0
			} else if p.X > worldWidth-playerW {
				p.X = worldWidth - playerW
			}
		}

		// Stomp detection
		for i, attacker := range s.players {
			if attacker == nil || !attacker.Alive || attacker.VY <= 0 {
				continue
			}
			for j, victim := range s.players {
				if j == i || victim == nil || !victim.Alive {
					continue
				}
				// Attacker top must be within stompThresh of victim top
				attackerTop := attacker.Y - playerH
				victimTop := victim.Y - playerH
				if attackerTop-victimTop > -stompThresh && attackerTop < victimTop+stompThresh {
					// Horizontal overlap check
					if attacker.X < victim.X+playerW && attacker.X+playerW > victim.X {
						victim.Alive = false
						attacker.VY = bounceVY
						attacker.Kills++

						// Schedule respawn
						jCopy := j
						if respawnTimers[jCopy] != nil {
							respawnTimers[jCopy].Stop()
						}
						respawnTimers[jCopy] = time.AfterFunc(respawnDelay, func() {
							s.mu.Lock()
							if s.players[jCopy] != nil {
								sp := s.players[jCopy].SpawnSide
								s.players[jCopy].X = spawnPositions[sp][0]
								s.players[jCopy].Y = spawnPositions[sp][1]
								s.players[jCopy].VX = 0
								s.players[jCopy].VY = 0
								s.players[jCopy].Alive = true
							}
							s.mu.Unlock()
						})
					}
				}
			}
		}

		// Build snapshot
		snap := StateSnapshot{Tick: tick}
		for _, p := range s.players {
			if p == nil {
				continue
			}
			snap.Players = append(snap.Players, PlayerSnap{
				CID:       p.CID.String(),
				X:         p.X,
				Y:         p.Y,
				VX:        p.VX,
				VY:        p.VY,
				Alive:     p.Alive,
				SpawnSide: p.SpawnSide,
				Kills:     p.Kills,
			})
		}
		s.mu.Unlock()

		data, err := json.Marshal(snap)
		if err != nil {
			logger.DPrintf(logger.DError, "Failed to marshal snapshot: %v", err)
			continue
		}

		// Non-blocking send; drop tick if channel is full (client too slow)
		select {
		case s.broadcast <- data:
		default:
		}
	}
}

// sender reads from broadcast and writes raw bytes to each connected client.
func (s *GameSession) sender(stopCh <-chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		case data := <-s.broadcast:
			s.mu.Lock()
			for i, conn := range s.clients {
				if conn == nil {
					continue
				}
				err := conn.WriteMessage(websocket.TextMessage, data)
				if err != nil {
					logger.DPrintf(logger.DError, "Write error to player %d: %v", i, err)
				}
			}
			s.mu.Unlock()
		}
	}
}

// GameService manages all active game sessions.
type GameService struct {
	sessions map[uuid.UUID]*GameSession
	stopChs  map[uuid.UUID]chan struct{}
	mu       sync.Mutex
}

func NewGameService() *GameService {
	return &GameService{
		sessions: make(map[uuid.UUID]*GameSession),
		stopChs:  make(map[uuid.UUID]chan struct{}),
	}
}

func (s *GameService) CreateSession(id uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := newGameSession()
	stopCh := make(chan struct{})
	s.sessions[id] = session
	s.stopChs[id] = stopCh
	go session.physicsTick(stopCh)
	go session.sender(stopCh)
	logger.DPrintf(logger.DInfo, "Created game session %v", id)
}

// JoinSession registers a player by cid into the session (index 0 or 1).
func (s *GameService) JoinSession(id uuid.UUID, cid uuid.UUID, ws *websocket.Conn) (int, error) {
	s.mu.Lock()
	session, ok := s.sessions[id]
	s.mu.Unlock()
	if !ok {
		return -1, fmt.Errorf("session %v not found", id)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Already registered (reconnect)?
	if idx, exists := session.playerIdx[cid]; exists {
		session.clients[idx] = ws
		logger.DPrintf(logger.DInfo, "Player %v reconnected as slot %d in session %v", cid, idx, id)
		return idx, nil
	}

	// Assign next free slot
	idx := -1
	for i, p := range session.players {
		if p == nil {
			idx = i
			break
		}
	}
	if idx == -1 {
		return -1, fmt.Errorf("session %v is full", id)
	}

	session.playerIdx[cid] = idx
	session.clients[idx] = ws
	session.players[idx] = &PlayerState{
		CID:       cid,
		X:         spawnPositions[idx][0],
		Y:         spawnPositions[idx][1],
		Alive:     true,
		SpawnSide: idx,
	}
	logger.DPrintf(logger.DInfo, "Player %v joined session %v as slot %d", cid, id, idx)
	return idx, nil
}

// SetInput stores the latest input for a player slot (called from handler goroutine).
func (s *GameService) SetInput(id uuid.UUID, playerIdx int, inp InputMessage) {
	s.mu.Lock()
	session, ok := s.sessions[id]
	s.mu.Unlock()
	if !ok {
		return
	}
	session.mu.Lock()
	prev := session.inputs[playerIdx]
	session.inputs[playerIdx] = InputState{
		Left:  inp.Left,
		Right: inp.Right,
		// Latch jump: once set true it stays until physics tick consumes it
		Jump: prev.Jump || inp.Jump,
	}
	session.mu.Unlock()
}

// LeaveSession removes a player's connection; deletes session when both gone.
func (s *GameService) LeaveSession(id uuid.UUID, cid uuid.UUID) {
	s.mu.Lock()
	session, ok := s.sessions[id]
	s.mu.Unlock()
	if !ok {
		return
	}

	session.mu.Lock()
	idx, exists := session.playerIdx[cid]
	if exists {
		session.clients[idx] = nil
	}
	allGone := true
	for _, c := range session.clients {
		if c != nil {
			allGone = false
			break
		}
	}
	session.mu.Unlock()

	if allGone {
		s.mu.Lock()
		stopCh, hasCh := s.stopChs[id]
		delete(s.sessions, id)
		delete(s.stopChs, id)
		s.mu.Unlock()
		if hasCh {
			close(stopCh)
		}
		logger.DPrintf(logger.DInfo, "Deleted session %v (all players disconnected)", id)
	}
}
