package signaling

import (
	"sync"
	"testing"
	"time"
)

func TestRoomAddAndRemove(t *testing.T) {
	room := NewRoom("test-room")
	peer := &Peer{ID: "peer1"}

	// Add peer
	if err := room.Add(peer); err != nil {
		t.Fatalf("failed to add peer: %v", err)
	}

	// Verify peer is in room
	if !room.Contains("peer1") {
		t.Error("peer should be in room")
	}
	if peer.GetRoomID() != "test-room" {
		t.Errorf("peer's room ID should be 'test-room', got '%s'", peer.GetRoomID())
	}

	// Remove peer
	room.Remove("peer1")
	if room.Contains("peer1") {
		t.Error("peer should not be in room after removal")
	}
	if peer.GetRoomID() != "" {
		t.Errorf("peer's room ID should be empty, got '%s'", peer.GetRoomID())
	}
}

func TestRoomMaxPeers(t *testing.T) {
	room := NewRoom("limited-room")
	room.MaxPeers = 2

	// Add first two peers
	if err := room.Add(&Peer{ID: "p1"}); err != nil {
		t.Fatalf("failed to add p1: %v", err)
	}
	if err := room.Add(&Peer{ID: "p2"}); err != nil {
		t.Fatalf("failed to add p2: %v", err)
	}

	// Third peer should fail
	err := room.Add(&Peer{ID: "p3"})
	if err == nil {
		t.Error("expected error when room is full")
	}

	// After removing one, should be able to add again
	room.Remove("p1")
	if err := room.Add(&Peer{ID: "p3"}); err != nil {
		t.Fatalf("failed to add p3 after removal: %v", err)
	}
}

func TestRoomUnlimitedPeers(t *testing.T) {
	room := NewRoom("unlimited-room")
	// MaxPeers = 0 means unlimited

	for i := 0; i < 100; i++ {
		peer := &Peer{ID: generatePeerID()}
		if err := room.Add(peer); err != nil {
			t.Fatalf("failed to add peer %d: %v", i, err)
		}
	}

	if room.Count() != 100 {
		t.Errorf("expected 100 peers, got %d", room.Count())
	}
}

func TestRoomGet(t *testing.T) {
	room := NewRoom("test-room")
	peer := &Peer{ID: "target"}
	room.Add(peer)

	// Found
	got := room.Get("target")
	if got == nil || got.ID != "target" {
		t.Error("failed to get existing peer")
	}

	// Not found
	got = room.Get("nonexistent")
	if got != nil {
		t.Error("should return nil for nonexistent peer")
	}
}

func TestRoomPeers(t *testing.T) {
	room := NewRoom("test-room")
	room.Add(&Peer{ID: "p1"})
	room.Add(&Peer{ID: "p2"})

	peers := room.Peers()
	if len(peers) != 2 {
		t.Errorf("expected 2 peers, got %d", len(peers))
	}

	// Verify it's a copy
	peers = append(peers, &Peer{ID: "p3"})
	if room.Count() != 2 {
		t.Error("Peers() should return a copy")
	}
}

func TestRoomPeerInfos(t *testing.T) {
	room := NewRoom("test-room")
	now := time.Now()

	p1 := &Peer{ID: "p1", DisplayName: "Alice", JoinedAt: now}
	p2 := &Peer{ID: "p2", DisplayName: "Bob", JoinedAt: now}
	room.Add(p1)
	room.Add(p2)

	infos := room.PeerInfos()
	if len(infos) != 2 {
		t.Errorf("expected 2 peer infos, got %d", len(infos))
	}

	// Verify info contains correct data
	found := false
	for _, info := range infos {
		if info.PeerID == "p1" && info.DisplayName == "Alice" {
			found = true
			break
		}
	}
	if !found {
		t.Error("peer info should contain Alice")
	}
}

func TestRoomIsEmpty(t *testing.T) {
	room := NewRoom("test-room")

	if !room.IsEmpty() {
		t.Error("new room should be empty")
	}

	room.Add(&Peer{ID: "p1"})
	if room.IsEmpty() {
		t.Error("room with peer should not be empty")
	}

	room.Remove("p1")
	if !room.IsEmpty() {
		t.Error("room after removing all peers should be empty")
	}
}

func TestRoomConcurrentAccess(t *testing.T) {
	room := NewRoom("concurrent-room")
	var wg sync.WaitGroup
	const numOps = 100

	// Concurrent adds
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			room.Add(&Peer{ID: generatePeerID()})
		}()
	}

	// Concurrent reads
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = room.Count()
			_ = room.Peers()
			_ = room.IsEmpty()
		}()
	}

	wg.Wait()

	if room.Count() != numOps {
		t.Errorf("expected %d peers, got %d", numOps, room.Count())
	}
}

// --- Room Manager Tests ---

func TestRoomManagerGetOrCreate(t *testing.T) {
	rm := NewRoomManager()

	// First call creates room
	room1 := rm.GetOrCreate("room-a")
	if room1 == nil {
		t.Fatal("GetOrCreate returned nil")
	}
	if room1.ID != "room-a" {
		t.Errorf("expected room ID 'room-a', got '%s'", room1.ID)
	}

	// Second call returns same room
	room2 := rm.GetOrCreate("room-a")
	if room1 != room2 {
		t.Error("GetOrCreate should return same room instance")
	}

	// Different ID creates different room
	room3 := rm.GetOrCreate("room-b")
	if room3 == room1 {
		t.Error("different room IDs should create different rooms")
	}
}

func TestRoomManagerGet(t *testing.T) {
	rm := NewRoomManager()

	// Room doesn't exist
	if rm.Get("nonexistent") != nil {
		t.Error("Get should return nil for nonexistent room")
	}

	// Create and get
	rm.GetOrCreate("exists")
	if rm.Get("exists") == nil {
		t.Error("Get should return room after creation")
	}
}

func TestRoomManagerDelete(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreate("to-delete")

	if rm.Get("to-delete") == nil {
		t.Fatal("room should exist before delete")
	}

	rm.Delete("to-delete")

	if rm.Get("to-delete") != nil {
		t.Error("room should not exist after delete")
	}
}

func TestRoomManagerList(t *testing.T) {
	rm := NewRoomManager()
	rm.GetOrCreate("room-a")
	rm.GetOrCreate("room-b")
	rm.GetOrCreate("room-c")

	list := rm.List()
	if len(list) != 3 {
		t.Errorf("expected 3 rooms, got %d", len(list))
	}
}

func TestRoomManagerCount(t *testing.T) {
	rm := NewRoomManager()

	if rm.Count() != 0 {
		t.Errorf("expected count 0, got %d", rm.Count())
	}

	rm.GetOrCreate("a")
	rm.GetOrCreate("b")

	if rm.Count() != 2 {
		t.Errorf("expected count 2, got %d", rm.Count())
	}
}

func TestRoomManagerStats(t *testing.T) {
	rm := NewRoomManager()

	room1 := rm.GetOrCreate("room-1")
	room1.Add(&Peer{ID: "p1"})
	room1.Add(&Peer{ID: "p2"})

	room2 := rm.GetOrCreate("room-2")
	room2.Add(&Peer{ID: "p3"})

	stats := rm.Stats()

	if stats.TotalRooms != 2 {
		t.Errorf("expected TotalRooms 2, got %d", stats.TotalRooms)
	}
	if stats.TotalPeers != 3 {
		t.Errorf("expected TotalPeers 3, got %d", stats.TotalPeers)
	}
}

func TestRoomManagerCleanupEmpty(t *testing.T) {
	rm := NewRoomManager()
	rm.EmptyRoomTTL = 0 // Immediate cleanup for testing

	// Create rooms with different states
	roomWithPeers := rm.GetOrCreate("has-peers")
	roomWithPeers.Add(&Peer{ID: "p1"})

	rm.GetOrCreate("empty-1")
	rm.GetOrCreate("empty-2")

	// Give rooms time to be "old enough"
	time.Sleep(10 * time.Millisecond)

	removed := rm.CleanupEmpty()

	if removed != 2 {
		t.Errorf("expected 2 removed, got %d", removed)
	}
	if rm.Count() != 1 {
		t.Errorf("expected 1 room remaining, got %d", rm.Count())
	}
	if rm.Get("has-peers") == nil {
		t.Error("room with peers should not be removed")
	}
}

func TestRoomManagerJoinRoom(t *testing.T) {
	rm := NewRoomManager()
	peer := &Peer{ID: "test-peer"}

	// Join first room
	room1, err := rm.JoinRoom(peer, "room-1")
	if err != nil {
		t.Fatalf("failed to join room-1: %v", err)
	}
	if !room1.Contains("test-peer") {
		t.Error("peer should be in room-1")
	}
	if peer.GetRoomID() != "room-1" {
		t.Errorf("peer's room should be room-1, got %s", peer.GetRoomID())
	}

	// Join different room (should leave first)
	room2, err := rm.JoinRoom(peer, "room-2")
	if err != nil {
		t.Fatalf("failed to join room-2: %v", err)
	}
	if room1.Contains("test-peer") {
		t.Error("peer should not be in room-1 anymore")
	}
	if !room2.Contains("test-peer") {
		t.Error("peer should be in room-2")
	}
}

func TestRoomManagerLeaveRoom(t *testing.T) {
	rm := NewRoomManager()
	peer := &Peer{ID: "test-peer"}

	rm.JoinRoom(peer, "test-room")
	if peer.GetRoomID() == "" {
		t.Fatal("peer should be in room after join")
	}

	rm.LeaveRoom(peer)
	if peer.GetRoomID() != "" {
		t.Error("peer should not be in room after leave")
	}
}

func TestRoomManagerDefaultMaxPeers(t *testing.T) {
	rm := NewRoomManager()
	rm.DefaultMaxPeers = 5

	room := rm.GetOrCreate("limited")
	if room.MaxPeers != 5 {
		t.Errorf("expected MaxPeers 5, got %d", room.MaxPeers)
	}
}

func TestRoomManagerConcurrentAccess(t *testing.T) {
	rm := NewRoomManager()
	var wg sync.WaitGroup
	const numOps = 50

	// Concurrent room creation
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			rm.GetOrCreate(generatePeerID())
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rm.Count()
			_ = rm.List()
			_ = rm.Stats()
		}()
	}

	wg.Wait()

	if rm.Count() != numOps {
		t.Errorf("expected %d rooms, got %d", numOps, rm.Count())
	}
}
