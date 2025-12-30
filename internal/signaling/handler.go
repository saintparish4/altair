package signaling

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Upgrader abstracts WebSocket upgrade functionality.
// This interface is satisfied by websocket.Upgrader from gorilla/websocket.
type Upgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (Conn, error)
}

// Handler processes WebSocket connections and signaling messages.
type Handler struct {
	registry *Registry
	rooms    *RoomManager
	upgrader Upgrader

	// Configuration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PingInterval time.Duration
	PongWait     time.Duration

	// Logging
	Logger *log.Logger
}

// NewHandler creates a new WebSocket handler.
// Pass nil for upgrader to create a handler without WebSocket support (for testing).
func NewHandler(registry *Registry, rooms *RoomManager) *Handler {
	return &Handler{
		registry:     registry,
		rooms:        rooms,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 10 * time.Second,
		PingInterval: 30 * time.Second,
		PongWait:     60 * time.Second,
		Logger:       log.Default(),
	}
}

// SetUpgrader sets the WebSocket upgrader.
func (h *Handler) SetUpgrader(u Upgrader) {
	h.upgrader = u
}

// ServeHTTP upgrades HTTP connections to WebSocket and handles the connection.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.upgrader == nil {
		http.Error(w, "WebSocket upgrader not configured", http.StatusInternalServerError)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log("upgrade error: %v", err)
		return
	}

	// Create and register peer
	peer := NewPeer("", conn)
	peer = h.registry.Register(peer)
	h.log("peer %s connected", peer.ID)

	// Send welcome message with assigned peer ID
	welcome := NewMessage(MessageTypeAck).
		WithPeerID(peer.ID).
		WithPayload(AckPayload{Message: "connected"})
	peer.Send(welcome)

	// Handle connection lifecycle
	defer h.handleDisconnect(peer)

	// Start ping/pong handler
	go h.pingLoop(peer)

	// Configure connection
	conn.SetReadLimit(32 * 1024) // 32KB max message size
	conn.SetReadDeadline(time.Now().Add(h.PongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(h.PongWait))
		peer.UpdateLastSeen()
		return nil
	})

	// Message read loop
	h.readLoop(peer)
}

// readLoop reads and processes messages from a peer.
func (h *Handler) readLoop(peer *Peer) {
	conn := peer.Connection()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			// Connection closed or error - log and exit
			if !peer.IsClosed() {
				h.log("peer %s read error: %v", peer.ID, err)
			}
			return
		}

		peer.UpdateLastSeen()

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			peer.SendError(ErrorCodeInvalidMessage, "invalid JSON")
			continue
		}

		// Set peer ID on incoming messages
		msg.PeerID = peer.ID

		if err := h.handleMessage(peer, &msg); err != nil {
			h.log("peer %s message error: %v", peer.ID, err)
		}
	}
}

// pingLoop sends periodic pings to keep the connection alive.
func (h *Handler) pingLoop(peer *Peer) {
	ticker := time.NewTicker(h.PingInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		if peer.IsClosed() {
			return
		}

		conn := peer.Connection()
		conn.SetWriteDeadline(time.Now().Add(h.WriteTimeout))
		if err := conn.WriteMessage(PingMessage, nil); err != nil {
			return
		}
	}
}

// handleDisconnect cleans up when a peer disconnects.
func (h *Handler) handleDisconnect(peer *Peer) {
	// Leave room and notify others
	roomID := peer.GetRoomID()
	if roomID != "" {
		if room := h.rooms.Get(roomID); room != nil {
			room.Remove(peer.ID)

			// Notify remaining peers
			notification := NewMessage(MessageTypePeerLeft).
				WithPeerID(peer.ID).
				WithRoomID(roomID)
			room.Broadcast(notification)
		}
	}

	// Close connection and unregister
	peer.Close()
	h.registry.Unregister(peer.ID)
	h.log("peer %s disconnected", peer.ID)
}

// handleMessage routes messages to appropriate handlers.
func (h *Handler) handleMessage(peer *Peer, msg *Message) error {
	switch msg.Type {
	case MessageTypeJoin:
		return h.handleJoin(peer, msg)
	case MessageTypeLeave:
		return h.handleLeave(peer, msg)
	case MessageTypeDiscover:
		return h.handleDiscover(peer, msg)
	case MessageTypeOffer:
		return h.handleOffer(peer, msg)
	case MessageTypeAnswer:
		return h.handleAnswer(peer, msg)
	case MessageTypeCandidate:
		return h.handleCandidate(peer, msg)
	case MessageTypeKeepAlive:
		return h.handleKeepAlive(peer, msg)
	default:
		return peer.SendError(ErrorCodeInvalidMessage, fmt.Sprintf("unknown message type: %s", msg.Type))
	}
}

// handleJoin processes a room join request.
func (h *Handler) handleJoin(peer *Peer, msg *Message) error {
	roomID := msg.RoomID
	if roomID == "" {
		return peer.SendError(ErrorCodeInvalidMessage, "room_id is required")
	}

	// Parse optional payload
	var payload JoinPayload
	if msg.Payload != nil {
		msg.ParsePayload(&payload)
	}

	// Update peer info
	if payload.DisplayName != "" {
		peer.SetDisplayName(payload.DisplayName)
	}
	if payload.Endpoint != nil {
		peer.SetEndpoint(payload.Endpoint)
	}

	// Check if already in this room
	if peer.GetRoomID() == roomID {
		return peer.SendError(ErrorCodeAlreadyInRoom, "already in this room")
	}

	// Join room
	room, err := h.rooms.JoinRoom(peer, roomID)
	if err != nil {
		return peer.SendError(ErrorCodeRoomFull, err.Error())
	}

	h.log("peer %s joined room %s", peer.ID, roomID)

	// Send ACK with peer list to joining peer
	ack := NewMessage(MessageTypeAck).
		WithPeerID(peer.ID).
		WithRoomID(roomID).
		WithRequestID(msg.RequestID).
		WithPayload(PeerListPayload{
			RoomID: roomID,
			Peers:  room.PeerInfos(),
		})
	peer.Send(ack)

	// Notify other peers in room
	notification := NewMessage(MessageTypePeerJoined).
		WithPeerID(peer.ID).
		WithRoomID(roomID).
		WithPayload(peer.Info())
	room.Broadcast(notification, peer.ID)

	return nil
}

// handleLeave processes a room leave request.
func (h *Handler) handleLeave(peer *Peer, msg *Message) error {
	roomID := peer.GetRoomID()
	if roomID == "" {
		return peer.SendError(ErrorCodeNotInRoom, "not in any room")
	}

	room := h.rooms.Get(roomID)
	if room != nil {
		room.Remove(peer.ID)

		// Notify remaining peers
		notification := NewMessage(MessageTypePeerLeft).
			WithPeerID(peer.ID).
			WithRoomID(roomID)
		room.Broadcast(notification)
	}

	h.log("peer %s left room %s", peer.ID, roomID)

	// Send ACK
	ack := NewMessage(MessageTypeAck).
		WithPeerID(peer.ID).
		WithRequestID(msg.RequestID).
		WithPayload(AckPayload{Message: "left room"})
	return peer.Send(ack)
}

// handleDiscover processes a peer discovery request.
func (h *Handler) handleDiscover(peer *Peer, msg *Message) error {
	roomID := msg.RoomID
	if roomID == "" {
		roomID = peer.GetRoomID()
	}

	if roomID == "" {
		return peer.SendError(ErrorCodeNotInRoom, "no room specified and not in any room")
	}

	room := h.rooms.Get(roomID)
	if room == nil {
		return peer.SendError(ErrorCodeRoomNotFound, "room not found")
	}

	response := NewMessage(MessageTypePeerList).
		WithPeerID(peer.ID).
		WithRoomID(roomID).
		WithRequestID(msg.RequestID).
		WithPayload(PeerListPayload{
			RoomID: roomID,
			Peers:  room.PeerInfos(),
		})

	return peer.Send(response)
}

// handleOffer forwards a connection offer to the target peer.
func (h *Handler) handleOffer(peer *Peer, msg *Message) error {
	if msg.TargetID == "" {
		return peer.SendError(ErrorCodeInvalidMessage, "target_id is required")
	}

	target := h.registry.Get(msg.TargetID)
	if target == nil {
		return peer.SendError(ErrorCodePeerNotFound, "target peer not found")
	}

	// Forward offer to target
	forward := NewMessage(MessageTypeOffer).
		WithPeerID(peer.ID).
		WithTargetID(msg.TargetID).
		WithRequestID(msg.RequestID)
	forward.Payload = msg.Payload

	h.log("forwarding offer from %s to %s", peer.ID, msg.TargetID)
	return target.Send(forward)
}

// handleAnswer forwards a connection answer to the target peer.
func (h *Handler) handleAnswer(peer *Peer, msg *Message) error {
	if msg.TargetID == "" {
		return peer.SendError(ErrorCodeInvalidMessage, "target_id is required")
	}

	target := h.registry.Get(msg.TargetID)
	if target == nil {
		return peer.SendError(ErrorCodePeerNotFound, "target peer not found")
	}

	// Forward answer to target
	forward := NewMessage(MessageTypeAnswer).
		WithPeerID(peer.ID).
		WithTargetID(msg.TargetID).
		WithRequestID(msg.RequestID)
	forward.Payload = msg.Payload

	h.log("forwarding answer from %s to %s", peer.ID, msg.TargetID)
	return target.Send(forward)
}

// handleCandidate forwards an ICE candidate to the target peer.
func (h *Handler) handleCandidate(peer *Peer, msg *Message) error {
	if msg.TargetID == "" {
		return peer.SendError(ErrorCodeInvalidMessage, "target_id is required")
	}

	target := h.registry.Get(msg.TargetID)
	if target == nil {
		return peer.SendError(ErrorCodePeerNotFound, "target peer not found")
	}

	// Forward candidate to target
	forward := NewMessage(MessageTypeCandidate).
		WithPeerID(peer.ID).
		WithTargetID(msg.TargetID).
		WithRequestID(msg.RequestID)
	forward.Payload = msg.Payload

	return target.Send(forward)
}

// handleKeepAlive processes a keep-alive message.
func (h *Handler) handleKeepAlive(peer *Peer, msg *Message) error {
	// Just update timestamp (already done in readLoop) and send ACK
	ack := NewMessage(MessageTypeAck).
		WithPeerID(peer.ID).
		WithRequestID(msg.RequestID)
	return peer.Send(ack)
}

// log writes a log message if a logger is configured.
func (h *Handler) log(format string, args ...interface{}) {
	if h.Logger != nil {
		h.Logger.Printf("[signaling] "+format, args...)
	}
}
