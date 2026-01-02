package signaling

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Server is the main signaling server that coordinates WebSocket and HTTP handlers.
type Server struct {
	registry *Registry
	rooms    *RoomManager
	handler  *Handler

	httpServer *http.Server
	mux        *http.ServeMux

	// Configuration
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	CleanupInterval time.Duration
	StaleTimeout    time.Duration

	// Lifecycle
	shutdownOnce sync.Once
	done         chan struct{}

	// Logging
	Logger *log.Logger
}

// Config holds server configuration options.
type Config struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	CleanupInterval time.Duration
	StaleTimeout    time.Duration
	Logger          *log.Logger
}

// DefaultConfig returns sensible default configuration.
func DefaultConfig() Config {
	return Config{
		Addr:            ":8080",
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		CleanupInterval: 1 * time.Minute,
		StaleTimeout:    5 * time.Minute,
		Logger:          log.Default(),
	}
}

// NewServer creates a new signaling server with the given configuration.
func NewServer(cfg Config) *Server {
	registry := NewRegistry()
	rooms := NewRoomManager()
	handler := NewHandler(registry, rooms)

	if cfg.Logger != nil {
		handler.Logger = cfg.Logger
	}

	s := &Server{
		registry:        registry,
		rooms:           rooms,
		handler:         handler,
		mux:             http.NewServeMux(),
		Addr:            cfg.Addr,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		CleanupInterval: cfg.CleanupInterval,
		StaleTimeout:    cfg.StaleTimeout,
		Logger:          cfg.Logger,
		done:            make(chan struct{}),
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures HTTP routes.
func (s *Server) setupRoutes() {
	// WebSocket endpoint
	s.mux.Handle("/ws", s.handler)

	// REST API endpoints
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/api/stats", s.handleStats)
	s.mux.HandleFunc("/api/rooms", s.handleRooms)
	s.mux.HandleFunc("/api/rooms/", s.handleRoom) // /api/rooms/{roomID}

	// CORS middleware wrapper for API endpoints
	s.mux.HandleFunc("/", s.handleNotFound)
}

// Start begins serving requests. Blocks until shutdown.
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:         s.Addr,
		Handler:      s.corsMiddleware(s.mux),
		ReadTimeout:  s.ReadTimeout,
		WriteTimeout: s.WriteTimeout,
	}

	// Start cleanup goroutine
	go s.cleanupLoop()

	// Handle graceful shutdown
	go s.handleShutdownSignals()

	s.log("starting server on %s", s.Addr)
	err := s.httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	var err error
	s.shutdownOnce.Do(func() {
		s.log("shutting down...")
		close(s.done)

		if s.httpServer != nil {
			err = s.httpServer.Shutdown(ctx)
		}

		// Close all peer connections
		s.registry.ForEach(func(peer *Peer) {
			peer.Close()
		})
	})
	return err
}

// cleanupLoop periodically cleans up stale peers and empty rooms.
func (s *Server) cleanupLoop() {
	ticker := time.NewTicker(s.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			stalePeers := s.registry.CleanupStale(s.StaleTimeout)
			emptyRooms := s.rooms.CleanupEmpty()
			if stalePeers > 0 || emptyRooms > 0 {
				s.log("cleanup: removed %d stale peers, %d empty rooms", stalePeers, emptyRooms)
			}
		}
	}
}

// handleShutdownSignals listens for OS signals and initiates graceful shutdown.
func (s *Server) handleShutdownSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		s.log("received signal: %v", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.Shutdown(ctx)
	case <-s.done:
		return
	}
}

// --- HTTP Handlers ---

// corsMiddleware adds CORS headers for cross-origin requests.
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleHealth returns server health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UnixMilli(),
	})
}

// handleStats returns server statistics.
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	registryStats := s.registry.Stats()
	roomStats := s.rooms.Stats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"peers": map[string]interface{}{
			"total":        registryStats.TotalPeers,
			"without_room": registryStats.PeersWithoutRoom,
		},
		"rooms": map[string]interface{}{
			"total":       roomStats.TotalRooms,
			"total_peers": roomStats.TotalPeers,
		},
		"timestamp": time.Now().UnixMilli(),
	})
}

// handleRooms returns list of rooms or creates a room.
func (s *Server) handleRooms(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		stats := s.rooms.Stats()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rooms": stats.Rooms,
		})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleRoom returns details for a specific room.
func (s *Server) handleRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract room ID from path: /api/rooms/{roomID}
	roomID := r.URL.Path[len("/api/rooms/"):]
	if roomID == "" {
		http.Error(w, "room ID required", http.StatusBadRequest)
		return
	}

	room := s.rooms.Get(roomID)
	if room == nil {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         room.ID,
		"peers":      room.PeerInfos(),
		"peer_count": room.Count(),
		"max_peers":  room.MaxPeers,
		"created_at": room.CreatedAt.UnixMilli(),
	})
}

// handleNotFound handles unknown routes.
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": "not found",
		"path":  r.URL.Path,
	})
}

// log writes a log message if a logger is configured.
func (s *Server) log(format string, args ...interface{}) {
	if s.Logger != nil {
		s.Logger.Printf("[server] "+format, args...)
	}
}

// Handler returns the WebSocket handler for configuration.
func (s *Server) Handler() *Handler {
	return s.handler
}

// Registry returns the peer registry for external access.
func (s *Server) Registry() *Registry {
	return s.registry
}

// Rooms returns the room manager for external access.
func (s *Server) Rooms() *RoomManager {
	return s.rooms
}

// HandlerFunc returns the handler as an http.HandlerFunc.
// Useful for embedding in custom routers.
func (s *Server) HandlerFunc() http.Handler {
	return s.corsMiddleware(s.mux)
}

// ListenAddr returns the address format string.
func (s *Server) ListenAddr() string {
	return fmt.Sprintf("http://localhost%s", s.Addr)
}
