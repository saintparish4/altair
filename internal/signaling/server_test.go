package signaling

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServerHealthEndpoint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logger = nil
	server := NewServer(cfg)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.HandlerFunc().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%v'", response["status"])
	}

	if _, ok := response["timestamp"]; !ok {
		t.Error("response should include timestamp")
	}
}

func TestServerHealthMethodNotAllowed(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logger = nil
	server := NewServer(cfg)

	req := httptest.NewRequest("POST", "/health", nil)
	w := httptest.NewRecorder()

	server.HandlerFunc().ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestServerStatsEndpoint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logger = nil
	server := NewServer(cfg)

	// Add some peers and rooms for stats
	peer1 := &Peer{ID: "p1", RoomID: "room-a"}
	peer2 := &Peer{ID: "p2", RoomID: "room-a"}
	peer3 := &Peer{ID: "p3"}

	server.Registry().peers["p1"] = peer1
	server.Registry().peers["p2"] = peer2
	server.Registry().peers["p3"] = peer3

	room := server.Rooms().GetOrCreate("room-a")
	room.Add(peer1)
	room.Add(peer2)

	req := httptest.NewRequest("GET", "/api/stats", nil)
	w := httptest.NewRecorder()

	server.HandlerFunc().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	peers := response["peers"].(map[string]interface{})
	if peers["total"].(float64) != 3 {
		t.Errorf("expected 3 total peers, got %v", peers["total"])
	}
	if peers["without_room"].(float64) != 1 {
		t.Errorf("expected 1 peer without room, got %v", peers["without_room"])
	}

	rooms := response["rooms"].(map[string]interface{})
	if rooms["total"].(float64) != 1 {
		t.Errorf("expected 1 room, got %v", rooms["total"])
	}
}

func TestServerRoomsEndpoint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logger = nil
	server := NewServer(cfg)

	// Create some rooms
	room1 := server.Rooms().GetOrCreate("room-1")
	room1.Add(&Peer{ID: "p1"})

	server.Rooms().GetOrCreate("room-2")

	req := httptest.NewRequest("GET", "/api/rooms", nil)
	w := httptest.NewRecorder()

	server.HandlerFunc().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	rooms := response["rooms"].([]interface{})
	if len(rooms) != 2 {
		t.Errorf("expected 2 rooms, got %d", len(rooms))
	}
}

func TestServerRoomEndpoint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logger = nil
	server := NewServer(cfg)

	// Create a room with peers
	room := server.Rooms().GetOrCreate("test-room")
	peer1 := &Peer{ID: "p1", DisplayName: "Alice", JoinedAt: time.Now()}
	peer2 := &Peer{ID: "p2", DisplayName: "Bob", JoinedAt: time.Now()}
	room.Add(peer1)
	room.Add(peer2)

	req := httptest.NewRequest("GET", "/api/rooms/test-room", nil)
	w := httptest.NewRecorder()

	server.HandlerFunc().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["id"] != "test-room" {
		t.Errorf("expected room ID 'test-room', got '%v'", response["id"])
	}
	if response["peer_count"].(float64) != 2 {
		t.Errorf("expected 2 peers, got %v", response["peer_count"])
	}

	peers := response["peers"].([]interface{})
	if len(peers) != 2 {
		t.Errorf("expected 2 peer infos, got %d", len(peers))
	}
}

func TestServerRoomEndpointNotFound(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logger = nil
	server := NewServer(cfg)

	req := httptest.NewRequest("GET", "/api/rooms/nonexistent", nil)
	w := httptest.NewRecorder()

	server.HandlerFunc().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestServerRoomEndpointNoID(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logger = nil
	server := NewServer(cfg)

	req := httptest.NewRequest("GET", "/api/rooms/", nil)
	w := httptest.NewRecorder()

	server.HandlerFunc().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestServerNotFoundEndpoint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logger = nil
	server := NewServer(cfg)

	req := httptest.NewRequest("GET", "/unknown/path", nil)
	w := httptest.NewRecorder()

	server.HandlerFunc().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["error"] != "not found" {
		t.Errorf("expected error 'not found', got '%v'", response["error"])
	}
}

func TestServerCORSHeaders(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logger = nil
	server := NewServer(cfg)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.HandlerFunc().ServeHTTP(w, req)

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing Access-Control-Allow-Origin header")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("missing Access-Control-Allow-Methods header")
	}
}

func TestServerCORSPreflight(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logger = nil
	server := NewServer(cfg)

	req := httptest.NewRequest("OPTIONS", "/api/stats", nil)
	w := httptest.NewRecorder()

	server.HandlerFunc().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for OPTIONS, got %d", w.Code)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Addr != ":8080" {
		t.Errorf("expected default addr ':8080', got '%s'", cfg.Addr)
	}
	if cfg.ReadTimeout != 15*time.Second {
		t.Errorf("expected ReadTimeout 15s, got %v", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 15*time.Second {
		t.Errorf("expected WriteTimeout 15s, got %v", cfg.WriteTimeout)
	}
	if cfg.CleanupInterval != 1*time.Minute {
		t.Errorf("expected CleanupInterval 1m, got %v", cfg.CleanupInterval)
	}
	if cfg.StaleTimeout != 5*time.Minute {
		t.Errorf("expected StaleTimeout 5m, got %v", cfg.StaleTimeout)
	}
}

func TestServerAccessors(t *testing.T) {
	cfg := DefaultConfig()
	server := NewServer(cfg)

	if server.Registry() == nil {
		t.Error("Registry() should not return nil")
	}
	if server.Rooms() == nil {
		t.Error("Rooms() should not return nil")
	}
	if server.Handler() == nil {
		t.Error("Handler() should not return nil")
	}
	if server.HandlerFunc() == nil {
		t.Error("HandlerFunc() should not return nil")
	}

	addr := server.ListenAddr()
	if addr != "http://localhost:8080" {
		t.Errorf("expected 'http://localhost:8080', got '%s'", addr)
	}
}

func TestServerWebSocketEndpointWithoutUpgrader(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logger = nil
	server := NewServer(cfg)

	req := httptest.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()

	server.HandlerFunc().ServeHTTP(w, req)

	// Should fail because no upgrader is configured
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 without upgrader, got %d", w.Code)
	}
}

func TestServerWebSocketEndpointWithMockUpgrader(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logger = nil
	server := NewServer(cfg)

	// Set up mock upgrader
	mockUpgrader := NewMockUpgrader()
	server.Handler().SetUpgrader(mockUpgrader)

	// Note: httptest doesn't support actual WebSocket upgrades,
	// so this will still fail, but the upgrader should be called
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	w := httptest.NewRecorder()

	server.HandlerFunc().ServeHTTP(w, req)

	// The mock upgrader was called (it creates a connection)
	if len(mockUpgrader.Connections) != 1 {
		t.Errorf("expected 1 connection attempt, got %d", len(mockUpgrader.Connections))
	}
}
