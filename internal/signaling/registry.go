package signaling

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Registry manages all connected peers and provides thread-safe operations.
// It uses a read-write mutex to allow concurrent reads while serializing writes.
type Registry struct {
	peers map[string]*Peer // peerID -> Peer
	mu    sync.RWMutex

	// Callbacks for lifecycle events (optional)
	OnPeerAdded   func(peer *Peer)
	OnPeerRemoved func(peer *Peer)
}

// NewRegistry creates an empty peer registry.
func NewRegistry() *Registry {
	return &Registry{
		peers: make(map[string]*Peer),
	}
}

// Register adds a new peer to the registry.
// Returns the peer with its assigned ID.
func (r *Registry) Register(peer *Peer) *Peer {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Generate unique ID if not set
	if peer.ID == "" {
		peer.ID = generatePeerID()
	}

	// Ensure uniqueness (shouldn't happen with random IDs, but be safe)
	for {
		if _, exists := r.peers[peer.ID]; !exists {
			break
		}
		peer.ID = generatePeerID()
	}

	r.peers[peer.ID] = peer

	// Trigger callback outside lock to prevent deadlocks
	if r.OnPeerAdded != nil {
		go r.OnPeerAdded(peer)
	}

	return peer
}

// Unregister removes a peer from the registry.
func (r *Registry) Unregister(peerID string) {
	r.mu.Lock()
	peer, exists := r.peers[peerID]
	if exists {
		delete(r.peers, peerID)
	}
	r.mu.Unlock()

	if exists && r.OnPeerRemoved != nil {
		go r.OnPeerRemoved(peer)
	}
}

// Get retrieves a peer by ID. Returns nil if not found.
func (r *Registry) Get(peerID string) *Peer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.peers[peerID]
}

// Exists checks if a peer exists in the registry.
func (r *Registry) Exists(peerID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.peers[peerID]
	return exists
}

// Count returns the total number of registered peers.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.peers)
}

// All returns a snapshot of all peers.
// The returned slice is safe to iterate without holding locks.
func (r *Registry) All() []*Peer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	peers := make([]*Peer, 0, len(r.peers))
	for _, p := range r.peers {
		peers = append(peers, p)
	}
	return peers
}

// ForEach iterates over all peers with the provided function.
// The function is called while holding a read lock - do not modify the registry.
func (r *Registry) ForEach(fn func(peer *Peer)) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.peers {
		fn(p)
	}
}

// Broadcast sends a message to all peers except the excluded ones.
func (r *Registry) Broadcast(msg *Message, excludeIDs ...string) {
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

	// Send outside the lock to prevent blocking
	for _, p := range peers {
		// Fire and forget - if send fails, peer will be cleaned up elsewhere
		go p.Send(msg)
	}
}

// CleanupStale removes peers that haven't been seen within the timeout.
// Returns the number of peers removed.
func (r *Registry) CleanupStale(timeout time.Duration) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-timeout)
	removed := 0

	for id, peer := range r.peers {
		if peer.LastSeen.Before(cutoff) {
			delete(r.peers, id)
			peer.Close()
			removed++
		}
	}

	return removed
}

// Stats returns registry statistics.
func (r *Registry) Stats() RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := RegistryStats{
		TotalPeers:  len(r.peers),
		PeersByRoom: make(map[string]int),
	}

	for _, p := range r.peers {
		roomID := p.GetRoomID()
		if roomID != "" {
			stats.PeersByRoom[roomID]++
		} else {
			stats.PeersWithoutRoom++
		}
	}

	return stats
}

// RegistryStats contains registry statistics.
type RegistryStats struct {
	TotalPeers       int
	PeersWithoutRoom int
	PeersByRoom      map[string]int
}

func (s RegistryStats) String() string {
	return fmt.Sprintf("TotalPeers=%d, WithoutRoom=%d, Rooms=%d",
		s.TotalPeers, s.PeersWithoutRoom, len(s.PeersByRoom))
}

// generatePeerID creates a random 8-character hex ID.
// Example: "a1b2c3d4"
func generatePeerID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}
