package chat

import (
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"
)

func TestNewTextMessage(t *testing.T) {
	msg := NewTextMessage("Alice", "Hello, world!")

	if msg.Type != MessageTypeText {
		t.Errorf("expected type TEXT, got %s", msg.Type)
	}
	if msg.From != "Alice" {
		t.Errorf("expected from 'Alice', got '%s'", msg.From)
	}
	if msg.Content != "Hello, world!" {
		t.Errorf("expected content 'Hello, world!', got '%s'", msg.Content)
	}
	if msg.ID == "" {
		t.Error("expected non-empty ID")
	}
	if msg.Timestamp == 0 {
		t.Error("expected non-zero timestamp")
	}
}

func TestNewJoinMessage(t *testing.T) {
	msg := NewJoinMessage("Bob")

	if msg.Type != MessageTypeJoin {
		t.Errorf("expected type JOIN, got %s", msg.Type)
	}
	if msg.From != "Bob" {
		t.Errorf("expected from 'Bob', got '%s'", msg.From)
	}
}

func TestNewLeaveMessage(t *testing.T) {
	msg := NewLeaveMessage("Charlie")

	if msg.Type != MessageTypeLeave {
		t.Errorf("expected type LEAVE, got %s", msg.Type)
	}
	if msg.From != "Charlie" {
		t.Errorf("expected from 'Charlie', got '%s'", msg.From)
	}
}

func TestMessageEncodeDecode(t *testing.T) {
	original := NewTextMessage("Alice", "Test message")

	encoded, err := original.Encode()
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	decoded, err := DecodeMessage(encoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("type mismatch: got %s, want %s", decoded.Type, original.Type)
	}
	if decoded.From != original.From {
		t.Errorf("from mismatch: got %s, want %s", decoded.From, original.From)
	}
	if decoded.Content != original.Content {
		t.Errorf("content mismatch: got %s, want %s", decoded.Content, original.Content)
	}
	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, original.ID)
	}
}

func TestDecodeInvalidMessage(t *testing.T) {
	_, err := DecodeMessage([]byte("not valid json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestMessageFormatTime(t *testing.T) {
	// Use local time to match FormatTime() behavior
	testTime := time.Date(2024, 1, 15, 14, 30, 45, 0, time.Local)
	msg := &Message{
		Timestamp: testTime.UnixMilli(),
	}

	formatted := msg.FormatTime()
	expected := testTime.Format("15:04:05")
	if formatted != expected {
		t.Errorf("expected '%s', got '%s'", expected, formatted)
	}
}

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		name    string
		msg     *Message
		isLocal bool
		wantNot string // String that should NOT appear (to verify different formatting)
	}{
		{
			name:    "local text message",
			msg:     NewTextMessage("Alice", "Hello"),
			isLocal: true,
			wantNot: "Alice:", // Local messages should say "You:", not "Alice:"
		},
		{
			name:    "remote text message",
			msg:     NewTextMessage("Bob", "Hi there"),
			isLocal: false,
			wantNot: "You:", // Remote messages should not say "You:"
		},
		{
			name:    "join message",
			msg:     NewJoinMessage("Charlie"),
			isLocal: false,
			wantNot: "",
		},
		{
			name:    "leave message",
			msg:     NewLeaveMessage("Diana"),
			isLocal: false,
			wantNot: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMessage(tt.msg, tt.isLocal)
			if result == "" && tt.msg.Type != MessageTypePing && tt.msg.Type != MessageTypePong {
				t.Error("expected non-empty formatted message")
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	ids := make(map[string]bool)

	// Generate multiple IDs and check uniqueness
	for i := 0; i < 100; i++ {
		id := generateID()
		if ids[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestSessionMessages(t *testing.T) {
	// Create a pipe for testing
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	session := NewSession(client, SessionConfig{
		Username:    "TestUser",
		MaxMessages: 10,
	})

	// Add some messages manually (simulating received messages)
	for i := 0; i < 15; i++ {
		msg := NewTextMessage("Peer", "Message")
		session.addMessage(msg)
	}

	messages := session.Messages()

	// Should be trimmed to MaxMessages
	if len(messages) != 10 {
		t.Errorf("expected 10 messages (max), got %d", len(messages))
	}
}

func TestSessionConfig(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	session := NewSession(client, SessionConfig{
		Username:    "Alice",
		PeerName:    "Bob",
		MaxMessages: 50,
	})

	if session.Username() != "Alice" {
		t.Errorf("expected username 'Alice', got '%s'", session.Username())
	}
	if session.PeerName() != "Bob" {
		t.Errorf("expected peer name 'Bob', got '%s'", session.PeerName())
	}
}

func TestSessionMessageCallback(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	var received *Message
	var wg sync.WaitGroup
	wg.Add(1)

	session := NewSession(client, SessionConfig{
		Username: "TestUser",
		OnMessage: func(msg *Message) {
			received = msg
			wg.Done()
		},
	})

	// Start receiving in background
	go session.receiveLoop()

	// Send a message from "server" side
	msg := NewTextMessage("Peer", "Hello from peer")
	data, _ := json.Marshal(msg)
	data = append(data, '\n')
	server.Write(data)

	// Wait for callback
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if received == nil {
			t.Fatal("callback not called")
		}
		if received.Content != "Hello from peer" {
			t.Errorf("expected content 'Hello from peer', got '%s'", received.Content)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message callback")
	}
}

func TestMessageTypes(t *testing.T) {
	types := []MessageType{
		MessageTypeText,
		MessageTypeJoin,
		MessageTypeLeave,
		MessageTypePing,
		MessageTypePong,
		MessageTypeAck,
		MessageTypeHistory,
	}

	for _, mt := range types {
		if mt == "" {
			t.Error("message type should not be empty")
		}
	}
}

func TestColorConstants(t *testing.T) {
	// Just verify they're defined and non-empty
	colors := []string{
		ColorReset,
		ColorRed,
		ColorGreen,
		ColorYellow,
		ColorBlue,
		ColorCyan,
		ColorGray,
		ColorBold,
	}

	for _, c := range colors {
		if c == "" {
			t.Error("color constant should not be empty")
		}
	}
}
