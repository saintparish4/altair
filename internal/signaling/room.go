package signaling

import (
	"fmt"
	"sync"
	"time"
)

// Room represents a logical grouping of peers for discovery and coordination.
type Room struct {
	ID        string
	CreatedAt time.Time
	MaxPeers  int // 0 = unlimited

	peers map[string]*Peer // peerID -> Peer
	mu    sync.RWMutex
}

// NewRoom creates a new room with the given ID.
func NewRoom(id string) *Room {
	return &Room{
		ID:        id,
		CreatedAt: time.Now(),
		MaxPeers:  0, // unlimited by default
		peers:     make(map[string]*Peer),
	}
}

// Add adds a peer to the room.
// Returns an error if the room is full.
func (r *Room) Add(peer *Peer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.MaxPeers > 0 && len(r.peers) >= r.MaxPeers {
		return fmt.Errorf("room %s is full (max %d peers)", r.ID, r.MaxPeers)
	}

	r.peers[peer.ID] = peer
	peer.SetRoomID(r.ID)
	return nil
}

// Remove removes a peer from the room.
func (r *Room) Remove(peerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if peer, exists := r.peers[peerID]; exists {
		peer.SetRoomID("")
		delete(r.peers, peerID)
	}
}

// Get retrieves a peer from the room by ID.
func (r *Room) Get(peerID string) *Peer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.peers[peerID]
}

// Contains checks if a peer is in the room.
func (r *Room) Contains(peerID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.peers[peerID]
	return exists
}

// Peers returns a snapshot of all peers in the room.
func (r *Room) Peers() []*Peer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	peers := make([]*Peer, 0, len(r.peers))
	for _, p := range r.peers {
		peers = append(peers, p)
	}
	return peers
}

// PeerInfos returns PeerInfo for all peers (for protocol messages).
func (r *Room) PeerInfos() []PeerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]PeerInfo, 0, len(r.peers))
	for _, p := range r.peers {
		infos = append(infos, p.Info())
	}
	return infos
}

// Count returns the number of peers in the room.
func (r *Room) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.peers)
}

// IsEmpty returns true if the room has no peers.
func (r *Room) IsEmpty() bool {
	return r.Count() == 0
}

// Broadcast sends a message to all peers in the room except excluded ones.
func (r *Room) Broadcast(msg *Message, excludeIDs ...string) {
	excludeSet := make(map[string]bool, len(excludeIDs))
	for _, id := range excludeIDs {
		excludeSet[id] = true
	}

	r.mu.RLock()
	peers := make([]*Peer, 0, len(r.peers))
	for _, p := range r.peers {
		if !excludeSet[p.ID] {
			peers = append(peers, p)
		}
	}
	r.mu.RUnlock()

	for _, p := range peers {
		go p.Send(msg)
	}
}

// --- Room Manager ---

// RoomManager manages multiple rooms with automatic cleanup.
type RoomManager struct {
	rooms map[string]*Room
	mu    sync.RWMutex

	// Configuration
	DefaultMaxPeers int           // Default max peers per room (0 = unlimited)
	EmptyRoomTTL    time.Duration // How long to keep empty rooms
}

// NewRoomManager creates a new room manager.
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms:           make(map[string]*Room),
		DefaultMaxPeers: 0, // unlimited
		EmptyRoomTTL:    5 * time.Minute,
	}
}

// GetOrCreate gets an existing room or creates a new one.
func (rm *RoomManager) GetOrCreate(roomID string) *Room {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if room, exists := rm.rooms[roomID]; exists {
		return room
	}

	room := NewRoom(roomID)
	room.MaxPeers = rm.DefaultMaxPeers
	rm.rooms[roomID] = room
	return room
}

// Get retrieves a room by ID. Returns nil if not found.
func (rm *RoomManager) Get(roomID string) *Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.rooms[roomID]
}

// Delete removes a room.
func (rm *RoomManager) Delete(roomID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.rooms, roomID)
}

// List returns all room IDs.
func (rm *RoomManager) List() []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	ids := make([]string, 0, len(rm.rooms))
	for id := range rm.rooms {
		ids = append(ids, id)
	}
	return ids
}

// Count returns the number of rooms.
func (rm *RoomManager) Count() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return len(rm.rooms)
}

// Stats returns room statistics.
func (rm *RoomManager) Stats() RoomStats {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	stats := RoomStats{
		TotalRooms: len(rm.rooms),
		Rooms:      make([]RoomInfo, 0, len(rm.rooms)),
	}

	for _, room := range rm.rooms {
		count := room.Count()
		stats.TotalPeers += count
		stats.Rooms = append(stats.Rooms, RoomInfo{
			ID:        room.ID,
			PeerCount: count,
			MaxPeers:  room.MaxPeers,
			CreatedAt: room.CreatedAt,
		})
	}

	return stats
}

// RoomStats contains room manager statistics.
type RoomStats struct {
	TotalRooms int
	TotalPeers int
	Rooms      []RoomInfo
}

// RoomInfo contains information about a single room.
type RoomInfo struct {
	ID        string    `json:"id"`
	PeerCount int       `json:"peer_count"`
	MaxPeers  int       `json:"max_peers"`
	CreatedAt time.Time `json:"created_at"`
}

// CleanupEmpty removes rooms that have been empty for longer than EmptyRoomTTL.
// Returns the number of rooms removed.
func (rm *RoomManager) CleanupEmpty() int {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	removed := 0
	cutoff := time.Now().Add(-rm.EmptyRoomTTL)

	for id, room := range rm.rooms {
		if room.IsEmpty() && room.CreatedAt.Before(cutoff) {
			delete(rm.rooms, id)
			removed++
		}
	}

	return removed
}

// JoinRoom adds a peer to a room, creating the room if necessary.
// Handles removing the peer from their previous room.
func (rm *RoomManager) JoinRoom(peer *Peer, roomID string) (*Room, error) {
	// Leave current room if in one
	currentRoomID := peer.GetRoomID()
	if currentRoomID != "" && currentRoomID != roomID {
		if currentRoom := rm.Get(currentRoomID); currentRoom != nil {
			currentRoom.Remove(peer.ID)
		}
	}

	// Join new room
	room := rm.GetOrCreate(roomID)
	if err := room.Add(peer); err != nil {
		return nil, err
	}

	return room, nil
}

// LeaveRoom removes a peer from their current room.
func (rm *RoomManager) LeaveRoom(peer *Peer) {
	roomID := peer.GetRoomID()
	if roomID == "" {
		return
	}

	if room := rm.Get(roomID); room != nil {
		room.Remove(peer.ID)
	}
}
