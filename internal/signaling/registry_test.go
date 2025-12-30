package signaling

import (
	"sync"
	"testing"
	"time"
)

// mockConn implements just enough of websocket.Conn for testing
// Since we can't easily mock gorilla/websocket, we'll test registry logic
// without actually using Peer objects that require connections

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()

	// Create a peer with pre-set ID (simulating already having an ID)
	peer := &Peer{ID: "test-peer-1"}
	r.peers["test-peer-1"] = peer

	// Verify we can retrieve it
	got := r.Get("test-peer-1")
	if got == nil {
		t.Fatal("expected to find peer, got nil")
	}
	if got.ID != "test-peer-1" {
		t.Errorf("expected ID 'test-peer-1', got '%s'", got.ID)
	}
}

func TestRegistryUnregister(t *testing.T) {
	r := NewRegistry()
	r.peers["test-peer"] = &Peer{ID: "test-peer"}

	if !r.Exists("test-peer") {
		t.Fatal("peer should exist before unregister")
	}

	r.Unregister("test-peer")

	if r.Exists("test-peer") {
		t.Error("peer should not exist after unregister")
	}
}

func TestRegistryCount(t *testing.T) {
	r := NewRegistry()

	if r.Count() != 0 {
		t.Errorf("expected count 0, got %d", r.Count())
	}

	r.peers["p1"] = &Peer{ID: "p1"}
	r.peers["p2"] = &Peer{ID: "p2"}
	r.peers["p3"] = &Peer{ID: "p3"}

	if r.Count() != 3 {
		t.Errorf("expected count 3, got %d", r.Count())
	}
}

func TestRegistryAll(t *testing.T) {
	r := NewRegistry()
	r.peers["p1"] = &Peer{ID: "p1"}
	r.peers["p2"] = &Peer{ID: "p2"}

	all := r.All()
	if len(all) != 2 {
		t.Errorf("expected 2 peers, got %d", len(all))
	}

	// Verify it's a copy (modifying shouldn't affect registry)
	all = append(all, &Peer{ID: "p3"})
	if r.Count() != 2 {
		t.Error("modifying All() result affected registry")
	}
}

func TestRegistryForEach(t *testing.T) {
	r := NewRegistry()
	r.peers["p1"] = &Peer{ID: "p1"}
	r.peers["p2"] = &Peer{ID: "p2"}

	var ids []string
	r.ForEach(func(p *Peer) {
		ids = append(ids, p.ID)
	})

	if len(ids) != 2 {
		t.Errorf("expected 2 IDs, got %d", len(ids))
	}
}

func TestRegistryStats(t *testing.T) {
	r := NewRegistry()

	// Add peers with different room states
	p1 := &Peer{ID: "p1", RoomID: "room-a"}
	p2 := &Peer{ID: "p2", RoomID: "room-a"}
	p3 := &Peer{ID: "p3", RoomID: "room-b"}
	p4 := &Peer{ID: "p4"} // no room

	r.peers["p1"] = p1
	r.peers["p2"] = p2
	r.peers["p3"] = p3
	r.peers["p4"] = p4

	stats := r.Stats()

	if stats.TotalPeers != 4 {
		t.Errorf("expected TotalPeers 4, got %d", stats.TotalPeers)
	}
	if stats.PeersWithoutRoom != 1 {
		t.Errorf("expected PeersWithoutRoom 1, got %d", stats.PeersWithoutRoom)
	}
	if stats.PeersByRoom["room-a"] != 2 {
		t.Errorf("expected 2 peers in room-a, got %d", stats.PeersByRoom["room-a"])
	}
	if stats.PeersByRoom["room-b"] != 1 {
		t.Errorf("expected 1 peer in room-b, got %d", stats.PeersByRoom["room-b"])
	}
}

func TestRegistryCleanupStale(t *testing.T) {
	r := NewRegistry()

	// Add peers with different LastSeen times
	now := time.Now()
	r.peers["fresh"] = &Peer{ID: "fresh", LastSeen: now}
	r.peers["stale1"] = &Peer{ID: "stale1", LastSeen: now.Add(-10 * time.Minute)}
	r.peers["stale2"] = &Peer{ID: "stale2", LastSeen: now.Add(-20 * time.Minute)}

	// Cleanup peers stale for more than 5 minutes
	removed := r.CleanupStale(5 * time.Minute)

	if removed != 2 {
		t.Errorf("expected 2 removed, got %d", removed)
	}
	if r.Count() != 1 {
		t.Errorf("expected 1 remaining, got %d", r.Count())
	}
	if !r.Exists("fresh") {
		t.Error("fresh peer should still exist")
	}
}

func TestGeneratePeerID(t *testing.T) {
	// Generate multiple IDs and verify they're unique and well-formed
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generatePeerID()

		// Should be 8 characters (4 bytes hex-encoded)
		if len(id) != 8 {
			t.Errorf("expected ID length 8, got %d for '%s'", len(id), id)
		}

		// Should be unique
		if ids[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		ids[id] = true

		// Should be valid hex
		for _, c := range id {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("invalid hex character '%c' in ID '%s'", c, id)
			}
		}
	}
}

func TestRegistryConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup
	const numGoroutines = 100

	// Concurrent registrations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			peer := &Peer{ID: generatePeerID()}
			r.mu.Lock()
			r.peers[peer.ID] = peer
			r.mu.Unlock()
		}(i)
	}

	// Concurrent reads while registrations happen
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Count()
			_ = r.All()
			_ = r.Stats()
		}()
	}

	wg.Wait()

	// Should have exactly numGoroutines peers
	if r.Count() != numGoroutines {
		t.Errorf("expected %d peers, got %d", numGoroutines, r.Count())
	}
}

func TestRegistryCallbacks(t *testing.T) {
	r := NewRegistry()

	var addedID, removedID string
	var addedWg, removedWg sync.WaitGroup
	addedWg.Add(1)
	removedWg.Add(1)

	r.OnPeerAdded = func(p *Peer) {
		addedID = p.ID
		addedWg.Done()
	}
	r.OnPeerRemoved = func(p *Peer) {
		removedID = p.ID
		removedWg.Done()
	}

	peer := &Peer{ID: "callback-test"}
	r.Register(peer)

	// Wait for async callback
	addedWg.Wait()
	if addedID != "callback-test" {
		t.Errorf("OnPeerAdded not called correctly, got ID '%s'", addedID)
	}

	r.Unregister("callback-test")

	// Wait for async callback
	removedWg.Wait()
	if removedID != "callback-test" {
		t.Errorf("OnPeerRemoved not called correctly, got ID '%s'", removedID)
	}
}

func TestRegistryStatsString(t *testing.T) {
	stats := RegistryStats{
		TotalPeers:       10,
		PeersWithoutRoom: 2,
		PeersByRoom:      map[string]int{"a": 4, "b": 4},
	}

	s := stats.String()
	if s == "" {
		t.Error("Stats.String() returned empty string")
	}
	// Just verify it doesn't panic and returns something
	t.Logf("Stats string: %s", s)
}
