package signaling

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewMessage(t *testing.T) {
	before := time.Now().UnixMilli()
	msg := NewMessage(MessageTypeJoin)
	after := time.Now().UnixMilli()

	if msg.Type != MessageTypeJoin {
		t.Errorf("expected type %s, got %s", MessageTypeJoin, msg.Type)
	}

	if msg.Timestamp < before || msg.Timestamp > after {
		t.Errorf("timestamp %d not in expected range [%d, %d]", msg.Timestamp, before, after)
	}
}

func TestMessageChaining(t *testing.T) {
	msg := NewMessage(MessageTypeOffer).
		WithPeerID("peer1").
		WithTargetID("peer2").
		WithRoomID("room1").
		WithRequestID("req123")

	if msg.PeerID != "peer1" {
		t.Errorf("expected PeerID 'peer1', got '%s'", msg.PeerID)
	}
	if msg.TargetID != "peer2" {
		t.Errorf("expected TargetID 'peer2', got '%s'", msg.TargetID)
	}
	if msg.RoomID != "room1" {
		t.Errorf("expected RoomID 'room1', got '%s'", msg.RoomID)
	}
	if msg.RequestID != "req123" {
		t.Errorf("expected RequestID 'req123', got '%s'", msg.RequestID)
	}
}

func TestMessageWithPayload(t *testing.T) {
	endpoint := Endpoint{IP: "1.2.3.4", Port: 5678}
	payload := OfferPayload{
		Endpoint:    endpoint,
		SessionID:   "session123",
		InitiatorID: "peer1",
	}

	msg := NewMessage(MessageTypeOffer).WithPayload(payload)

	var parsed OfferPayload
	if err := msg.ParsePayload(&parsed); err != nil {
		t.Fatalf("failed to parse payload: %v", err)
	}

	if parsed.Endpoint.IP != "1.2.3.4" {
		t.Errorf("expected IP '1.2.3.4', got '%s'", parsed.Endpoint.IP)
	}
	if parsed.Endpoint.Port != 5678 {
		t.Errorf("expected Port 5678, got %d", parsed.Endpoint.Port)
	}
	if parsed.SessionID != "session123" {
		t.Errorf("expected SessionID 'session123', got '%s'", parsed.SessionID)
	}
}

func TestMessageJSONSerialization(t *testing.T) {
	msg := NewMessage(MessageTypeJoin).
		WithPeerID("peer1").
		WithRoomID("test-room").
		WithPayload(JoinPayload{
			DisplayName: "Test User",
			Endpoint:    &Endpoint{IP: "192.168.1.1", Port: 12345},
		})

	// Serialize
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Deserialize
	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Type != MessageTypeJoin {
		t.Errorf("expected type %s, got %s", MessageTypeJoin, decoded.Type)
	}
	if decoded.PeerID != "peer1" {
		t.Errorf("expected PeerID 'peer1', got '%s'", decoded.PeerID)
	}
	if decoded.RoomID != "test-room" {
		t.Errorf("expected RoomID 'test-room', got '%s'", decoded.RoomID)
	}

	// Parse payload
	var payload JoinPayload
	if err := decoded.ParsePayload(&payload); err != nil {
		t.Fatalf("failed to parse payload: %v", err)
	}
	if payload.DisplayName != "Test User" {
		t.Errorf("expected DisplayName 'Test User', got '%s'", payload.DisplayName)
	}
}

func TestEndpointString(t *testing.T) {
	tests := []struct {
		endpoint Endpoint
		expected string
	}{
		{Endpoint{IP: "1.2.3.4", Port: 8080}, "1.2.3.4:8080"},
		{Endpoint{IP: "192.168.0.1", Port: 443}, "192.168.0.1:443"},
		{Endpoint{IP: "::1", Port: 80}, "::1:80"},
	}

	for _, tt := range tests {
		result := tt.endpoint.String()
		if result != tt.expected {
			t.Errorf("Endpoint{%s, %d}.String() = %s, want %s",
				tt.endpoint.IP, tt.endpoint.Port, result, tt.expected)
		}
	}
}

func TestNewErrorMessage(t *testing.T) {
	msg := NewErrorMessage(ErrorCodePeerNotFound, "peer xyz not found")

	if msg.Type != MessageTypeError {
		t.Errorf("expected type %s, got %s", MessageTypeError, msg.Type)
	}

	var payload ErrorPayload
	if err := msg.ParsePayload(&payload); err != nil {
		t.Fatalf("failed to parse error payload: %v", err)
	}

	if payload.Code != ErrorCodePeerNotFound {
		t.Errorf("expected code %s, got %s", ErrorCodePeerNotFound, payload.Code)
	}
	if payload.Message != "peer xyz not found" {
		t.Errorf("expected message 'peer xyz not found', got '%s'", payload.Message)
	}
}

func TestParsePayloadWithNilPayload(t *testing.T) {
	msg := NewMessage(MessageTypeJoin)
	// Don't set payload

	var payload JoinPayload
	err := msg.ParsePayload(&payload)
	if err == nil {
		t.Error("expected error for nil payload, got nil")
	}
}

func TestPeerListPayload(t *testing.T) {
	payload := PeerListPayload{
		RoomID: "test-room",
		Peers: []PeerInfo{
			{PeerID: "peer1", DisplayName: "Alice", JoinedAt: 1000},
			{PeerID: "peer2", DisplayName: "Bob", JoinedAt: 2000},
		},
	}

	msg := NewMessage(MessageTypePeerList).WithPayload(payload)

	var parsed PeerListPayload
	if err := msg.ParsePayload(&parsed); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(parsed.Peers) != 2 {
		t.Errorf("expected 2 peers, got %d", len(parsed.Peers))
	}
	if parsed.Peers[0].DisplayName != "Alice" {
		t.Errorf("expected first peer 'Alice', got '%s'", parsed.Peers[0].DisplayName)
	}
}

func TestAllMessageTypes(t *testing.T) {
	// Ensure all message types are distinct
	types := []MessageType{
		MessageTypeJoin,
		MessageTypeLeave,
		MessageTypeOffer,
		MessageTypeAnswer,
		MessageTypeCandidate,
		MessageTypeDiscover,
		MessageTypeKeepAlive,
		MessageTypePeerJoined,
		MessageTypePeerLeft,
		MessageTypePeerList,
		MessageTypeError,
		MessageTypeAck,
	}

	seen := make(map[MessageType]bool)
	for _, mt := range types {
		if seen[mt] {
			t.Errorf("duplicate message type: %s", mt)
		}
		seen[mt] = true
	}
}

func TestAllErrorCodes(t *testing.T) {
	// Ensure all error codes are distinct
	codes := []string{
		ErrorCodeInvalidMessage,
		ErrorCodeRoomNotFound,
		ErrorCodePeerNotFound,
		ErrorCodeNotInRoom,
		ErrorCodeAlreadyInRoom,
		ErrorCodeRoomFull,
		ErrorCodeUnauthorized,
		ErrorCodeInternal,
	}

	seen := make(map[string]bool)
	for _, code := range codes {
		if seen[code] {
			t.Errorf("duplicate error code: %s", code)
		}
		seen[code] = true
	}
}
