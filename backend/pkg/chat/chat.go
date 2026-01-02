// Package chat implements a simple P2P chat protocol over UDP connections
package chat

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// MessageType identifies the type of chat message
type MessageType string

const (
	MessageTypeText    MessageType = "TEXT"
	MessageTypeJoin    MessageType = "JOIN"
	MessageTypeLeave   MessageType = "LEAVE"
	MessageTypePing    MessageType = "PING"
	MessageTypePong    MessageType = "PONG"
	MessageTypeAck     MessageType = "ACK"
	MessageTypeHistory MessageType = "HISTORY"
)

// Message represents a chat message
type Message struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id"`
	From      string      `json:"from"`
	Content   string      `json:"content,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// NewTextMessage creates a new text message
func NewTextMessage(from, content string) *Message {
	return &Message{
		Type:      MessageTypeText,
		ID:        generateID(),
		From:      from,
		Content:   content,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewJoinMessage creates a join notification
func NewJoinMessage(username string) *Message {
	return &Message{
		Type:      MessageTypeJoin,
		ID:        generateID(),
		From:      username,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewLeaveMessage creates a leave notification
func NewLeaveMessage(username string) *Message {
	return &Message{
		Type:      MessageTypeLeave,
		ID:        generateID(),
		From:      username,
		Timestamp: time.Now().UnixMilli(),
	}
}

// Encode serializes the message to JSON bytes
func (m *Message) Encode() ([]byte, error) {
	return json.Marshal(m)
}

// DecodeMessage deserializes a message from JSON bytes
func DecodeMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// FormatTime returns a formatted timestamp string
func (m *Message) FormatTime() string {
	t := time.UnixMilli(m.Timestamp)
	return t.Format("15:04:05")
}

// --- Chat Session ---

// Session manages a P2P chat connection
type Session struct {
	conn     net.Conn
	username string
	peerName string

	messages    []*Message
	messagesMu  sync.RWMutex
	maxMessages int

	onMessage func(*Message)
	onError   func(error)

	done   chan struct{}
	sendMu sync.Mutex
}

// SessionConfig holds configuration for a chat session
type SessionConfig struct {
	Username    string
	PeerName    string
	MaxMessages int
	OnMessage   func(*Message)
	OnError     func(error)
}

// NewSession creates a new chat session over the given connection
func NewSession(conn net.Conn, cfg SessionConfig) *Session {
	if cfg.MaxMessages <= 0 {
		cfg.MaxMessages = 1000
	}
	return &Session{
		conn:        conn,
		username:    cfg.Username,
		peerName:    cfg.PeerName,
		messages:    make([]*Message, 0),
		maxMessages: cfg.MaxMessages,
		onMessage:   cfg.OnMessage,
		onError:     cfg.OnError,
		done:        make(chan struct{}),
	}
}

// Start begins the chat session, listening for incoming messages
func (s *Session) Start() error {
	// Send join message
	joinMsg := NewJoinMessage(s.username)
	if err := s.sendMessage(joinMsg); err != nil {
		return fmt.Errorf("failed to send join message: %w", err)
	}

	// Start receive loop
	go s.receiveLoop()

	// Start keepalive
	go s.keepAliveLoop()

	return nil
}

// Stop ends the chat session.
func (s *Session) Stop() {
	// Send leave message (best effort)
	leaveMsg := NewLeaveMessage(s.username)
	s.sendMessage(leaveMsg)

	close(s.done)
	s.conn.Close()
}

// Send sends a text message to the peer.
func (s *Session) Send(content string) error {
	msg := NewTextMessage(s.username, content)
	s.addMessage(msg)
	return s.sendMessage(msg)
}

// Messages returns a copy of the message history.
func (s *Session) Messages() []*Message {
	s.messagesMu.RLock()
	defer s.messagesMu.RUnlock()

	result := make([]*Message, len(s.messages))
	copy(result, s.messages)
	return result
}

// Username returns the local username.
func (s *Session) Username() string {
	return s.username
}

// PeerName returns the peer's username.
func (s *Session) PeerName() string {
	s.messagesMu.RLock()
	defer s.messagesMu.RUnlock()
	return s.peerName
}

func (s *Session) sendMessage(msg *Message) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	data, err := msg.Encode()
	if err != nil {
		return err
	}

	// Add newline delimiter
	data = append(data, '\n')

	s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err = s.conn.Write(data)
	return err
}

func (s *Session) receiveLoop() {
	reader := bufio.NewReader(s.conn)

	for {
		select {
		case <-s.done:
			return
		default:
		}

		s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				if s.onError != nil {
					s.onError(fmt.Errorf("connection closed"))
				}
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Timeout is ok, just retry
			}
			if s.onError != nil {
				s.onError(err)
			}
			return
		}

		msg, err := DecodeMessage(line)
		if err != nil {
			continue // Skip malformed messages
		}

		s.handleMessage(msg)
	}
}

func (s *Session) handleMessage(msg *Message) {
	switch msg.Type {
	case MessageTypeText:
		s.addMessage(msg)
		if s.onMessage != nil {
			s.onMessage(msg)
		}

	case MessageTypeJoin:
		s.messagesMu.Lock()
		s.peerName = msg.From
		s.messagesMu.Unlock()
		s.addMessage(msg)
		if s.onMessage != nil {
			s.onMessage(msg)
		}

	case MessageTypeLeave:
		s.addMessage(msg)
		if s.onMessage != nil {
			s.onMessage(msg)
		}

	case MessageTypePing:
		// Respond with pong
		pong := &Message{
			Type:      MessageTypePong,
			ID:        msg.ID,
			From:      s.username,
			Timestamp: time.Now().UnixMilli(),
		}
		s.sendMessage(pong)

	case MessageTypePong:
		// Peer is alive, nothing to do

	case MessageTypeAck:
		// Message acknowledged
	}
}

func (s *Session) addMessage(msg *Message) {
	s.messagesMu.Lock()
	defer s.messagesMu.Unlock()

	s.messages = append(s.messages, msg)

	// Trim old messages if needed
	if len(s.messages) > s.maxMessages {
		s.messages = s.messages[len(s.messages)-s.maxMessages:]
	}
}

func (s *Session) keepAliveLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			ping := &Message{
				Type:      MessageTypePing,
				ID:        generateID(),
				From:      s.username,
				Timestamp: time.Now().UnixMilli(),
			}
			s.sendMessage(ping)
		}
	}
}

// --- Utilities ---

var idCounter int64
var idMu sync.Mutex

func generateID() string {
	idMu.Lock()
	defer idMu.Unlock()
	idCounter++
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), idCounter)
}

// --- Terminal UI Helpers ---

// Colors for terminal output
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
	ColorBold   = "\033[1m"
)

// FormatMessage formats a message for terminal display.
func FormatMessage(msg *Message, isLocal bool) string {
	timeStr := ColorGray + msg.FormatTime() + ColorReset

	switch msg.Type {
	case MessageTypeText:
		if isLocal {
			return fmt.Sprintf("%s %s%sYou:%s %s",
				timeStr, ColorBold, ColorCyan, ColorReset, msg.Content)
		}
		return fmt.Sprintf("%s %s%s%s:%s %s",
			timeStr, ColorBold, ColorGreen, msg.From, ColorReset, msg.Content)

	case MessageTypeJoin:
		return fmt.Sprintf("%s %s* %s joined the chat%s",
			timeStr, ColorYellow, msg.From, ColorReset)

	case MessageTypeLeave:
		return fmt.Sprintf("%s %s* %s left the chat%s",
			timeStr, ColorYellow, msg.From, ColorReset)

	default:
		return ""
	}
}

// ClearLine clears the current terminal line.
func ClearLine() {
	fmt.Print("\r\033[K")
}

// MoveCursorUp moves the cursor up n lines.
func MoveCursorUp(n int) {
	fmt.Printf("\033[%dA", n)
}

// SaveCursor saves the current cursor position.
func SaveCursor() {
	fmt.Print("\033[s")
}

// RestoreCursor restores the saved cursor position.
func RestoreCursor() {
	fmt.Print("\033[u")
}
