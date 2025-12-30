// Package signaling implements a WebSocket-based signaling server for P2P coordination.
// It enables peers to discover each other and exchange endpoint information for NAT traversal.

package signaling

import (
	"encoding/json"
	"fmt"
	"time"
)

// MessageType identifies the type of signaling message
type MessageType string

const (
	// Client -> Server messages
	MessageTypeJoin      MessageType = "JOIN"       // Join a room
	MessageTypeLeave     MessageType = "LEAVE"      // Leave current room
	MessageTypeOffer     MessageType = "OFFER"      // Send connection offer to peer
	MessageTypeAnswer    MessageType = "ANSWER"     // Respond to connection offer
	MessageTypeCandidate MessageType = "CANDIDATE"  // Exchange endpoint candidates
	MessageTypeDiscover  MessageType = "DISCOVER"   // Request list of peers in room
	MessageTypeKeepAlive MessageType = "KEEP_ALIVE" // Keep connection alive

	// Server -> Client messages
	MessageTypePeerJoined MessageType = "PEER_JOINED" // Notification: peer joined room
	MessageTypePeerLeft   MessageType = "PEER_LEFT"   // Notification: peer left room
	MessageTypePeerList   MessageType = "PEER_LIST"   // Response to DISCOVER
	MessageTypeError      MessageType = "ERROR"       // Error response
	MessageTypeAck        MessageType = "ACK"         // Acknowledgment
)

// Message represents a signaling protocol message.
// All communication between clients and server uses this envelope format.
type Message struct {
	Type      MessageType     `json:"type"`
	PeerID    string          `json:"peer_id,omitempty"`    // Sender's peer ID
	TargetID  string          `json:"target_id,omitempty"`  // Target peer for directed messages
	RoomID    string          `json:"room_id,omitempty"`    // Room identifier
	Payload   json.RawMessage `json:"payload,omitempty"`    // Type-specific payload
	Timestamp int64           `json:"timestamp,omitempty"`  // Unix timestamp in milliseconds
	RequestID string          `json:"request_id,omitempty"` // For request/response correlation
}

// NewMessage creates a new message with the current timestamp
func NewMessage(msgType MessageType) *Message {
	return &Message{
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
	}
}

// WithPeerID sets the peer ID and returns the message for chaining
func (m *Message) WithPeerID(id string) *Message {
	m.PeerID = id
	return m
}

// WithTargetID sets the target peer ID and returns the message for chaining.
func (m *Message) WithTargetID(id string) *Message {
	m.TargetID = id
	return m
}

// WithRoomID sets the room ID and returns the message for chaining
func (m *Message) WithRoomID(id string) *Message {
	m.RoomID = id
	return m
}

// WithPayload sets the payload from any serializable value
func (m *Message) WithPayload(v any) *Message {
	data, err := json.Marshal(v)
	if err != nil {
		// In practice, this shouldnt fail for our payloads
		m.Payload = json.RawMessage(fmt.Sprintf(`{"error":"marshal failed: %v"}`, err))
		return m
	}
	m.Payload = data
	return m
}

// WithRequestID sets the request ID for correlation.
func (m *Message) WithRequestID(id string) *Message {
	m.RequestID = id
	return m
}

// --- Payload Types ---

// JoinPayload is sent with JOIN messages.
type JoinPayload struct {
	DisplayName string    `json:"display_name,omitempty"` // Optional human-readable name
	Endpoint    *Endpoint `json:"endpoint,omitempty"`     // Public endpoint if already known
}

// Endpoint represents a network endpoint (IP:Port).
// Mirrors pkg/types.Endpoint but kept separate to avoid import cycles.
type Endpoint struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

func (e Endpoint) String() string {
	return fmt.Sprintf("%s:%d", e.IP, e.Port)
}

// OfferPayload is sent with OFFER messages to initiate a connection.
type OfferPayload struct {
	Endpoint    Endpoint `json:"endpoint"`     // Sender's public endpoint
	SessionID   string   `json:"session_id"`   // Unique session identifier
	InitiatorID string   `json:"initiator_id"` // Who initiated the connection
}

// AnswerPayload is sent in response to an OFFER.
type AnswerPayload struct {
	Endpoint  Endpoint `json:"endpoint"`   // Responder's public endpoint
	SessionID string   `json:"session_id"` // Must match the offer's session ID
	Accepted  bool     `json:"accepted"`   // Whether the connection is accepted
}

// CandidatePayload exchanges additional endpoint candidates.
// Useful when a peer discovers multiple possible endpoints.
type CandidatePayload struct {
	SessionID string   `json:"session_id"`
	Endpoint  Endpoint `json:"endpoint"`
	Priority  int      `json:"priority,omitempty"` // Higher = preferred
}

// PeerInfo describes a peer for PEER_LIST and PEER_JOINED messages.
type PeerInfo struct {
	PeerID      string    `json:"peer_id"`
	DisplayName string    `json:"display_name,omitempty"`
	Endpoint    *Endpoint `json:"endpoint,omitempty"`
	JoinedAt    int64     `json:"joined_at"` // Unix timestamp
}

// PeerListPayload is sent in response to DISCOVER.
type PeerListPayload struct {
	RoomID string     `json:"room_id"`
	Peers  []PeerInfo `json:"peers"`
}

// ErrorPayload provides error details.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error codes for ErrorPayload.
const (
	ErrorCodeInvalidMessage = "INVALID_MESSAGE"
	ErrorCodeRoomNotFound   = "ROOM_NOT_FOUND"
	ErrorCodePeerNotFound   = "PEER_NOT_FOUND"
	ErrorCodeNotInRoom      = "NOT_IN_ROOM"
	ErrorCodeAlreadyInRoom  = "ALREADY_IN_ROOM"
	ErrorCodeRoomFull       = "ROOM_FULL"
	ErrorCodeUnauthorized   = "UNAUTHORIZED"
	ErrorCodeInternal       = "INTERNAL_ERROR"
)

// NewErrorMessage creates an error message.
func NewErrorMessage(code, message string) *Message {
	return NewMessage(MessageTypeError).WithPayload(ErrorPayload{
		Code:    code,
		Message: message,
	})
}

// AckPayload confirms successful processing of a request.
type AckPayload struct {
	RequestID string `json:"request_id,omitempty"`
	Message   string `json:"message,omitempty"`
}

// ParsePayload unmarshals the message payload into the provided type.
func (m *Message) ParsePayload(v any) error {
	if m.Payload == nil {
		return fmt.Errorf("message has no payload")
	}
	return json.Unmarshal(m.Payload, v)
}
