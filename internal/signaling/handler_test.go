package signaling

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandlerServeHTTPWithoutUpgrader(t *testing.T) {
	registry := NewRegistry()
	rooms := NewRoomManager()
	handler := NewHandler(registry, rooms)
	// Don't set upgrader

	req := httptest.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestHandlerMessageTypes(t *testing.T) {
	registry := NewRegistry()
	rooms := NewRoomManager()
	handler := NewHandler(registry, rooms)

	// Create a peer with mock connection
	mockConn := NewMockConn()
	peer := NewPeer("test-peer", mockConn)
	registry.Register(peer)

	tests := []struct {
		name     string
		msg      *Message
		wantErr  bool
		errCode  string
		setup    func()
		validate func(t *testing.T)
	}{
		{
			name: "join without room_id",
			msg: &Message{
				Type:   MessageTypeJoin,
				PeerID: "test-peer",
			},
			wantErr: true,
			errCode: ErrorCodeInvalidMessage,
		},
		{
			name: "join with room_id",
			msg: &Message{
				Type:   MessageTypeJoin,
				PeerID: "test-peer",
				RoomID: "test-room",
			},
			wantErr: false,
			validate: func(t *testing.T) {
				if peer.GetRoomID() != "test-room" {
					t.Errorf("peer should be in test-room, got %s", peer.GetRoomID())
				}
				if rooms.Get("test-room") == nil {
					t.Error("room should exist")
				}
			},
		},
		{
			name: "join already in same room",
			msg: &Message{
				Type:   MessageTypeJoin,
				PeerID: "test-peer",
				RoomID: "test-room",
			},
			setup: func() {
				rooms.JoinRoom(peer, "test-room")
			},
			wantErr: true,
			errCode: ErrorCodeAlreadyInRoom,
		},
		{
			name: "discover without room",
			msg: &Message{
				Type:   MessageTypeDiscover,
				PeerID: "test-peer",
			},
			setup: func() {
				peer.SetRoomID("")
			},
			wantErr: true,
			errCode: ErrorCodeNotInRoom,
		},
		{
			name: "discover with room",
			msg: &Message{
				Type:   MessageTypeDiscover,
				PeerID: "test-peer",
				RoomID: "discover-room",
			},
			setup: func() {
				rooms.GetOrCreate("discover-room")
			},
			wantErr: false,
		},
		{
			name: "leave without being in room",
			msg: &Message{
				Type:   MessageTypeLeave,
				PeerID: "test-peer",
			},
			setup: func() {
				peer.SetRoomID("")
			},
			wantErr: true,
			errCode: ErrorCodeNotInRoom,
		},
		{
			name: "offer without target_id",
			msg: &Message{
				Type:   MessageTypeOffer,
				PeerID: "test-peer",
			},
			wantErr: true,
			errCode: ErrorCodeInvalidMessage,
		},
		{
			name: "offer with nonexistent target",
			msg: &Message{
				Type:     MessageTypeOffer,
				PeerID:   "test-peer",
				TargetID: "nonexistent",
			},
			wantErr: true,
			errCode: ErrorCodePeerNotFound,
		},
		{
			name: "answer without target_id",
			msg: &Message{
				Type:   MessageTypeAnswer,
				PeerID: "test-peer",
			},
			wantErr: true,
			errCode: ErrorCodeInvalidMessage,
		},
		{
			name: "candidate without target_id",
			msg: &Message{
				Type:   MessageTypeCandidate,
				PeerID: "test-peer",
			},
			wantErr: true,
			errCode: ErrorCodeInvalidMessage,
		},
		{
			name: "keep_alive",
			msg: &Message{
				Type:   MessageTypeKeepAlive,
				PeerID: "test-peer",
			},
			wantErr: false,
		},
		{
			name: "unknown message type",
			msg: &Message{
				Type:   "UNKNOWN",
				PeerID: "test-peer",
			},
			wantErr: true,
			errCode: ErrorCodeInvalidMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock connection
			mockConn = NewMockConn()
			peer.conn = mockConn

			if tt.setup != nil {
				tt.setup()
			}

			err := handler.handleMessage(peer, tt.msg)

			if tt.wantErr {
				// Check that error was sent to peer
				written := mockConn.GetWritten()
				if len(written) == 0 {
					t.Fatal("expected error message to be sent")
				}
				var response Message
				if err := json.Unmarshal(written[len(written)-1], &response); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}
				if response.Type != MessageTypeError {
					t.Errorf("expected error message, got %s", response.Type)
				}
				if tt.errCode != "" {
					var errPayload ErrorPayload
					if err := response.ParsePayload(&errPayload); err == nil {
						if errPayload.Code != tt.errCode {
							t.Errorf("expected error code %s, got %s", tt.errCode, errPayload.Code)
						}
					}
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t)
			}
		})
	}
}

func TestHandlerOfferForwarding(t *testing.T) {
	registry := NewRegistry()
	rooms := NewRoomManager()
	handler := NewHandler(registry, rooms)

	// Create two peers
	mockConn1 := NewMockConn()
	peer1 := NewPeer("peer1", mockConn1)
	registry.Register(peer1)

	mockConn2 := NewMockConn()
	peer2 := NewPeer("peer2", mockConn2)
	registry.Register(peer2)

	// Peer1 sends offer to Peer2
	offer := &Message{
		Type:     MessageTypeOffer,
		PeerID:   "peer1",
		TargetID: "peer2",
		Payload:  json.RawMessage(`{"endpoint":{"ip":"1.2.3.4","port":5678}}`),
	}

	err := handler.handleMessage(peer1, offer)
	if err != nil {
		t.Fatalf("failed to handle offer: %v", err)
	}

	// Verify peer2 received the forwarded offer
	written := mockConn2.GetWritten()
	if len(written) == 0 {
		t.Fatal("peer2 should have received forwarded offer")
	}

	var forwarded Message
	if err := json.Unmarshal(written[0], &forwarded); err != nil {
		t.Fatalf("failed to parse forwarded message: %v", err)
	}

	if forwarded.Type != MessageTypeOffer {
		t.Errorf("expected OFFER, got %s", forwarded.Type)
	}
	if forwarded.PeerID != "peer1" {
		t.Errorf("expected PeerID 'peer1', got '%s'", forwarded.PeerID)
	}
}

func TestHandlerJoinWithPayload(t *testing.T) {
	registry := NewRegistry()
	rooms := NewRoomManager()
	handler := NewHandler(registry, rooms)

	mockConn := NewMockConn()
	peer := NewPeer("test-peer", mockConn)
	registry.Register(peer)

	// Join with display name and endpoint
	joinPayload := JoinPayload{
		DisplayName: "Alice",
		Endpoint:    &Endpoint{IP: "1.2.3.4", Port: 5678},
	}
	payloadBytes, _ := json.Marshal(joinPayload)

	msg := &Message{
		Type:    MessageTypeJoin,
		PeerID:  "test-peer",
		RoomID:  "test-room",
		Payload: payloadBytes,
	}

	err := handler.handleMessage(peer, msg)
	if err != nil {
		t.Fatalf("failed to join: %v", err)
	}

	// Verify peer info was updated
	info := peer.Info()
	if info.DisplayName != "Alice" {
		t.Errorf("expected DisplayName 'Alice', got '%s'", info.DisplayName)
	}
	if info.Endpoint == nil || info.Endpoint.IP != "1.2.3.4" {
		t.Error("endpoint should be set")
	}
}

func TestHandlerLeaveRoom(t *testing.T) {
	registry := NewRegistry()
	rooms := NewRoomManager()
	handler := NewHandler(registry, rooms)

	mockConn := NewMockConn()
	peer := NewPeer("test-peer", mockConn)
	registry.Register(peer)

	// First join a room
	rooms.JoinRoom(peer, "test-room")
	if peer.GetRoomID() != "test-room" {
		t.Fatal("peer should be in room after join")
	}

	// Leave the room
	msg := &Message{
		Type:   MessageTypeLeave,
		PeerID: "test-peer",
	}

	err := handler.handleMessage(peer, msg)
	if err != nil {
		t.Fatalf("failed to leave: %v", err)
	}

	if peer.GetRoomID() != "" {
		t.Error("peer should not be in any room after leave")
	}
}

func TestHandlerPeerJoinNotification(t *testing.T) {
	registry := NewRegistry()
	rooms := NewRoomManager()
	handler := NewHandler(registry, rooms)

	// Peer1 is already in the room
	mockConn1 := NewMockConn()
	peer1 := NewPeer("peer1", mockConn1)
	registry.Register(peer1)
	rooms.JoinRoom(peer1, "test-room")

	// Clear any messages from joining
	mockConn1.writeQueue = nil

	// Peer2 joins
	mockConn2 := NewMockConn()
	peer2 := NewPeer("peer2", mockConn2)
	registry.Register(peer2)

	msg := &Message{
		Type:   MessageTypeJoin,
		PeerID: "peer2",
		RoomID: "test-room",
	}

	err := handler.handleMessage(peer2, msg)
	if err != nil {
		t.Fatalf("failed to join: %v", err)
	}

	// Give async broadcast time to complete
	time.Sleep(10 * time.Millisecond)

	// Peer1 should receive PEER_JOINED notification
	written := mockConn1.GetWritten()
	if len(written) == 0 {
		t.Fatal("peer1 should receive notification")
	}

	var notification Message
	if err := json.Unmarshal(written[0], &notification); err != nil {
		t.Fatalf("failed to parse notification: %v", err)
	}

	if notification.Type != MessageTypePeerJoined {
		t.Errorf("expected PEER_JOINED, got %s", notification.Type)
	}
	if notification.PeerID != "peer2" {
		t.Errorf("notification should be about peer2, got %s", notification.PeerID)
	}
}

func TestHandlerDiscover(t *testing.T) {
	registry := NewRegistry()
	rooms := NewRoomManager()
	handler := NewHandler(registry, rooms)

	// Create room with multiple peers
	room := rooms.GetOrCreate("test-room")

	for i := 0; i < 3; i++ {
		peer := &Peer{ID: generatePeerID(), DisplayName: "Peer" + string(rune('A'+i))}
		registry.peers[peer.ID] = peer
		room.Add(peer)
	}

	// Add requesting peer
	mockConn := NewMockConn()
	peer := NewPeer("requester", mockConn)
	registry.Register(peer)

	msg := &Message{
		Type:   MessageTypeDiscover,
		PeerID: "requester",
		RoomID: "test-room",
	}

	err := handler.handleMessage(peer, msg)
	if err != nil {
		t.Fatalf("failed to discover: %v", err)
	}

	// Check response
	written := mockConn.GetWritten()
	if len(written) == 0 {
		t.Fatal("should receive peer list response")
	}

	var response Message
	if err := json.Unmarshal(written[0], &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Type != MessageTypePeerList {
		t.Errorf("expected PEER_LIST, got %s", response.Type)
	}

	var payload PeerListPayload
	if err := response.ParsePayload(&payload); err != nil {
		t.Fatalf("failed to parse payload: %v", err)
	}

	if len(payload.Peers) != 3 {
		t.Errorf("expected 3 peers, got %d", len(payload.Peers))
	}
}

func TestMockConn(t *testing.T) {
	conn := NewMockConn()

	// Test write
	err := conn.WriteMessage(TextMessage, []byte("hello"))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	written := conn.GetWritten()
	if len(written) != 1 || string(written[0]) != "hello" {
		t.Error("written data mismatch")
	}

	// Test read
	conn.EnqueueRead([]byte("response"))
	msgType, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if msgType != TextMessage {
		t.Errorf("expected TextMessage, got %d", msgType)
	}
	if string(data) != "response" {
		t.Errorf("expected 'response', got '%s'", string(data))
	}

	// Test close
	conn.Close()
	if !conn.IsClosed() {
		t.Error("connection should be closed")
	}

	err = conn.WriteMessage(TextMessage, []byte("after close"))
	if err == nil {
		t.Error("write after close should fail")
	}
}

func TestMockUpgrader(t *testing.T) {
	upgrader := NewMockUpgrader()

	req := httptest.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}

	if conn == nil {
		t.Fatal("connection should not be nil")
	}

	if len(upgrader.Connections) != 1 {
		t.Errorf("expected 1 connection, got %d", len(upgrader.Connections))
	}
}
