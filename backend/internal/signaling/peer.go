package signaling

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Conn abstracts a WebSocket connection for testability.
// This interface is satisfied by *websocket.Conn from gorilla/websocket.
type Conn interface {
	WriteMessage(messageType int, data []byte) error
	ReadMessage() (messageType int, p []byte, err error)
	Close() error
	SetWriteDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetReadLimit(limit int64)
	SetPongHandler(h func(appData string) error)
}

// WebSocket message types (matching gorilla/websocket constants)
const (
	TextMessage   = 1
	BinaryMessage = 2
	CloseMessage  = 8
	PingMessage   = 9
	PongMessage   = 10
)

// Peer represents a connected client.
type Peer struct {
	ID          string
	DisplayName string
	Endpoint    *Endpoint
	RoomID      string
	JoinedAt    time.Time
	LastSeen    time.Time

	conn   Conn
	mu     sync.Mutex // Protects conn writes
	closed bool
}

// NewPeer creates a new peer with the given WebSocket connection.
func NewPeer(id string, conn Conn) *Peer {
	now := time.Now()
	return &Peer{
		ID:       id,
		conn:     conn,
		JoinedAt: now,
		LastSeen: now,
	}
}

// Send sends a message to the peer. Thread-safe.
func (p *Peer) Send(msg *Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("peer %s connection is closed", p.ID)
	}

	// Set write deadline to prevent blocking indefinitely
	if err := p.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	if err := p.conn.WriteMessage(TextMessage, data); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

// SendError sends an error message to the peer.
func (p *Peer) SendError(code, message string) error {
	return p.Send(NewErrorMessage(code, message))
}

// Close closes the peer's connection.
func (p *Peer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

// IsClosed returns whether the peer's connection is closed.
func (p *Peer) IsClosed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closed
}

// UpdateLastSeen updates the last seen timestamp.
func (p *Peer) UpdateLastSeen() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.LastSeen = time.Now()
}

// Info returns a PeerInfo snapshot for protocol messages.
func (p *Peer) Info() PeerInfo {
	p.mu.Lock()
	defer p.mu.Unlock()
	return PeerInfo{
		PeerID:      p.ID,
		DisplayName: p.DisplayName,
		Endpoint:    p.Endpoint,
		JoinedAt:    p.JoinedAt.UnixMilli(),
	}
}

// SetEndpoint updates the peer's public endpoint.
func (p *Peer) SetEndpoint(endpoint *Endpoint) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Endpoint = endpoint
}

// SetDisplayName updates the peer's display name.
func (p *Peer) SetDisplayName(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.DisplayName = name
}

// SetRoomID updates the peer's current room.
func (p *Peer) SetRoomID(roomID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.RoomID = roomID
}

// GetRoomID returns the peer's current room ID.
func (p *Peer) GetRoomID() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.RoomID
}

// Connection returns the underlying WebSocket connection.
// Use with caution - prefer using Send() for thread-safe writes.
func (p *Peer) Connection() Conn {
	return p.conn
}
