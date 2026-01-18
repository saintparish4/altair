// Package chat implements a simple P2P chat protocol over UDP connections
package chat

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MessageType identifies the type of chat message
type MessageType string

const (
	MessageTypeText        MessageType = "TEXT"
	MessageTypeJoin        MessageType = "JOIN"
	MessageTypeLeave       MessageType = "LEAVE"
	MessageTypePing        MessageType = "PING"
	MessageTypePong        MessageType = "PONG"
	MessageTypeAck         MessageType = "ACK"
	MessageTypeHistory     MessageType = "HISTORY"
	MessageTypeFileInfo    MessageType = "FILE_INFO"
	MessageTypeFileAccept  MessageType = "FILE_ACCEPT"
	MessageTypeFileReject  MessageType = "FILE_REJECT"
	MessageTypeFileChunk   MessageType = "FILE_CHUNK"
	MessageTypeFileChunkAck MessageType = "FILE_CHUNK_ACK"
	MessageTypeFileComplete MessageType = "FILE_COMPLETE"
	MessageTypeFileCancel   MessageType = "FILE_CANCEL"
)

// File transfer constants
const (
	FileChunkSize = 64 * 1024 // 64KB chunks
	MaxFileSize   = 10 * 1024 * 1024 * 1024 // 10GB max
)

// Message represents a chat message
type Message struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id"`
	From      string      `json:"from"`
	Content   string      `json:"content,omitempty"`
	Timestamp int64       `json:"timestamp"`
	
	// File transfer fields
	FileName   string `json:"file_name,omitempty"`
	FileSize   int64  `json:"file_size,omitempty"`
	FileHash   string `json:"file_hash,omitempty"`
	ChunkIndex int    `json:"chunk_index,omitempty"`
	ChunkHash  string `json:"chunk_hash,omitempty"`
	ChunkData  []byte `json:"chunk_data,omitempty"`
	ChunkSize  int    `json:"chunk_size,omitempty"`
	TotalChunks int   `json:"total_chunks,omitempty"`
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

	onMessage  func(*Message)
	onError    func(error)
	onFileInfo func(*FileInfo) bool
	outputDir  string

	// File transfer state
	fileTransferMu sync.Mutex
	activeTransfer *activeFileTransfer
	fileProgress   *FileProgress
	fileResponseCh chan *Message // Channel for file transfer responses

	done   chan struct{}
	sendMu sync.Mutex
}

type activeFileTransfer struct {
	fileInfo   *FileInfo
	file       *os.File
	outputPath string
	chunks     map[int]bool // Track received chunks
}

// SessionConfig holds configuration for a chat session
type SessionConfig struct {
	Username    string
	PeerName    string
	MaxMessages int
	OnMessage   func(*Message)
	OnError     func(error)
	OnFileInfo  func(*FileInfo) bool // Return true to accept file transfer
	OutputDir   string               // Directory to save received files
}

// FileInfo describes a file being transferred
type FileInfo struct {
	Name      string
	Size      int64
	Hash      string
	ChunkSize int
	Chunks    int
}

// FileProgress tracks file transfer progress
type FileProgress struct {
	FileName    string
	FileSize    int64
	BytesSent   int64
	ChunksDone  int
	ChunksTotal int
	StartTime   time.Time
	Speed       float64 // bytes per second
}

// NewSession creates a new chat session over the given connection
func NewSession(conn net.Conn, cfg SessionConfig) *Session {
	if cfg.MaxMessages <= 0 {
		cfg.MaxMessages = 1000
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "."
	}
	if cfg.OnFileInfo == nil {
		cfg.OnFileInfo = func(*FileInfo) bool { return true } // Accept all by default
	}
	return &Session{
		conn:          conn,
		username:      cfg.Username,
		peerName:      cfg.PeerName,
		messages:      make([]*Message, 0),
		maxMessages:   cfg.MaxMessages,
		onMessage:     cfg.OnMessage,
		onError:       cfg.OnError,
		onFileInfo:    cfg.OnFileInfo,
		outputDir:     cfg.OutputDir,
		fileResponseCh: make(chan *Message, 10),
		done:          make(chan struct{}),
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

// SendFile sends a file to the peer.
func (s *Session) SendFile(filePath string, onProgress func(*FileProgress)) error {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	if stat.Size() > MaxFileSize {
		return fmt.Errorf("file too large: %d bytes (max %d)", stat.Size(), MaxFileSize)
	}

	// Calculate file hash
	hash, err := hashFile(file)
	if err != nil {
		return fmt.Errorf("hash file: %w", err)
	}
	file.Seek(0, 0) // Reset to beginning

	// Calculate chunks
	chunks := int(stat.Size()/FileChunkSize) + 1
	if stat.Size()%FileChunkSize == 0 && stat.Size() > 0 {
		chunks = int(stat.Size() / FileChunkSize)
	}

	fileInfo := &FileInfo{
		Name:      filepath.Base(filePath),
		Size:      stat.Size(),
		Hash:      hash,
		ChunkSize: FileChunkSize,
		Chunks:    chunks,
	}

	// Send file info
	infoMsg := &Message{
		Type:        MessageTypeFileInfo,
		ID:          generateID(),
		From:        s.username,
		Timestamp:   time.Now().UnixMilli(),
		FileName:    fileInfo.Name,
		FileSize:    fileInfo.Size,
		FileHash:    fileInfo.Hash,
		ChunkSize:   fileInfo.ChunkSize,
		TotalChunks: fileInfo.Chunks,
	}

	if err := s.sendMessage(infoMsg); err != nil {
		return fmt.Errorf("send file info: %w", err)
	}

	// Wait for accept/reject
	select {
	case response := <-s.fileResponseCh:
		if response.Type == MessageTypeFileReject {
			return fmt.Errorf("file rejected by peer")
		}
		if response.Type != MessageTypeFileAccept {
			return fmt.Errorf("unexpected response: %s", response.Type)
		}
	case <-time.After(60 * time.Second):
		return fmt.Errorf("timeout waiting for file acceptance")
	case <-s.done:
		return fmt.Errorf("session closed")
	}

	// Send chunks
	progress := &FileProgress{
		FileName:    fileInfo.Name,
		FileSize:    fileInfo.Size,
		ChunksTotal: chunks,
		StartTime:   time.Now(),
	}

	buf := make([]byte, FileChunkSize)
	for i := 0; i < chunks; i++ {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("read chunk %d: %w", i, err)
		}
		if n == 0 {
			break
		}

		chunkHash := hashBytes(buf[:n])
		chunkMsg := &Message{
			Type:       MessageTypeFileChunk,
			ID:         generateID(),
			From:       s.username,
			Timestamp:  time.Now().UnixMilli(),
			ChunkIndex: i,
			ChunkHash:  chunkHash,
			ChunkData:  buf[:n],
			ChunkSize:  n,
		}

		if err := s.sendMessage(chunkMsg); err != nil {
			return fmt.Errorf("send chunk %d: %w", i, err)
		}

		// Wait for chunk ack
		select {
		case ack := <-s.fileResponseCh:
			if ack.Type != MessageTypeFileChunkAck || ack.ChunkIndex != i {
				return fmt.Errorf("invalid chunk ack for chunk %d", i)
			}
		case <-time.After(60 * time.Second):
			return fmt.Errorf("timeout waiting for chunk ack %d", i)
		case <-s.done:
			return fmt.Errorf("session closed")
		}

		// Update progress
		progress.BytesSent += int64(n)
		progress.ChunksDone = i + 1
		elapsed := time.Since(progress.StartTime).Seconds()
		if elapsed > 0 {
			progress.Speed = float64(progress.BytesSent) / elapsed
		}
		if onProgress != nil {
			onProgress(progress)
		}
	}

	// Send completion
	completeMsg := &Message{
		Type:      MessageTypeFileComplete,
		ID:        generateID(),
		From:      s.username,
		Timestamp: time.Now().UnixMilli(),
	}

	return s.sendMessage(completeMsg)
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

	case MessageTypeFileInfo:
		s.handleFileInfo(msg)

	case MessageTypeFileAccept, MessageTypeFileReject:
		// Send to response channel for SendFile to receive
		select {
		case s.fileResponseCh <- msg:
		default:
		}

	case MessageTypeFileChunk:
		s.handleFileChunk(msg)

	case MessageTypeFileChunkAck:
		// Send to response channel for SendFile to receive
		select {
		case s.fileResponseCh <- msg:
		default:
		}

	case MessageTypeFileComplete:
		s.handleFileComplete()

	case MessageTypeFileCancel:
		s.handleFileCancel()
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

	case MessageTypeFileInfo:
		if isLocal {
			return fmt.Sprintf("%s %s%sYou:%s 📁 Sending file: %s (%s)%s",
				timeStr, ColorBold, ColorCyan, ColorReset, msg.FileName, formatBytes(msg.FileSize), ColorReset)
		}
		return fmt.Sprintf("%s %s%s%s:%s 📁 Sending file: %s (%s)%s",
			timeStr, ColorBold, ColorGreen, msg.From, ColorReset, msg.FileName, formatBytes(msg.FileSize), ColorReset)

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

// --- File Transfer Handlers ---

func (s *Session) handleFileInfo(msg *Message) {
	fileInfo := &FileInfo{
		Name:      msg.FileName,
		Size:      msg.FileSize,
		Hash:      msg.FileHash,
		ChunkSize: msg.ChunkSize,
		Chunks:    msg.TotalChunks,
	}

	// Check if we should accept
	accept := s.onFileInfo != nil && s.onFileInfo(fileInfo)

	var response *Message
	if accept {
		// Create output file
		outputPath := filepath.Join(s.outputDir, "received_"+fileInfo.Name)
		file, err := os.Create(outputPath)
		if err != nil {
			// Send reject
			response = &Message{
				Type:      MessageTypeFileReject,
				ID:        generateID(),
				From:      s.username,
				Timestamp: time.Now().UnixMilli(),
				Content:   fmt.Sprintf("Failed to create file: %v", err),
			}
			s.sendMessage(response)
			return
		}

		// Start receiving
		s.fileTransferMu.Lock()
		s.activeTransfer = &activeFileTransfer{
			fileInfo:   fileInfo,
			file:       file,
			outputPath: outputPath,
			chunks:     make(map[int]bool),
		}
		s.fileProgress = &FileProgress{
			FileName:    fileInfo.Name,
			FileSize:    fileInfo.Size,
			ChunksTotal: fileInfo.Chunks,
			StartTime:   time.Now(),
		}
		s.fileTransferMu.Unlock()

		response = &Message{
			Type:      MessageTypeFileAccept,
			ID:        generateID(),
			From:      s.username,
			Timestamp: time.Now().UnixMilli(),
		}

		// Notify user
		if s.onMessage != nil {
			notifyMsg := &Message{
				Type:      MessageTypeText,
				ID:        generateID(),
				From:      msg.From,
				Content:   fmt.Sprintf("📁 Receiving file: %s (%s)", fileInfo.Name, formatBytes(fileInfo.Size)),
				Timestamp: time.Now().UnixMilli(),
			}
			s.onMessage(notifyMsg)
		}
	} else {
		response = &Message{
			Type:      MessageTypeFileReject,
			ID:        generateID(),
			From:      s.username,
			Timestamp: time.Now().UnixMilli(),
		}
	}

	s.sendMessage(response)
}

func (s *Session) handleFileChunk(msg *Message) {
	s.fileTransferMu.Lock()
	transfer := s.activeTransfer
	progress := s.fileProgress
	s.fileTransferMu.Unlock()

	if transfer == nil {
		return // No active transfer
	}

	// Verify chunk hash
	expectedHash := hashBytes(msg.ChunkData)
	if expectedHash != msg.ChunkHash {
		// Send reject ack
		ack := &Message{
			Type:       MessageTypeFileChunkAck,
			ID:         generateID(),
			From:       s.username,
			Timestamp:  time.Now().UnixMilli(),
			ChunkIndex: msg.ChunkIndex,
		}
		s.sendMessage(ack)
		return
	}

	// Write chunk
	offset := int64(msg.ChunkIndex) * int64(transfer.fileInfo.ChunkSize)
	if _, err := transfer.file.WriteAt(msg.ChunkData, offset); err != nil {
		ack := &Message{
			Type:       MessageTypeFileChunkAck,
			ID:         generateID(),
			From:       s.username,
			Timestamp:  time.Now().UnixMilli(),
			ChunkIndex: msg.ChunkIndex,
		}
		s.sendMessage(ack)
		return
	}

	// Send ack
	ack := &Message{
		Type:       MessageTypeFileChunkAck,
		ID:         generateID(),
		From:       s.username,
		Timestamp:  time.Now().UnixMilli(),
		ChunkIndex: msg.ChunkIndex,
	}
	s.sendMessage(ack)

	// Update progress
	s.fileTransferMu.Lock()
	transfer.chunks[msg.ChunkIndex] = true
	progress.BytesSent += int64(msg.ChunkSize)
	progress.ChunksDone = len(transfer.chunks)
	elapsed := time.Since(progress.StartTime).Seconds()
	if elapsed > 0 {
		progress.Speed = float64(progress.BytesSent) / elapsed
	}
	s.fileTransferMu.Unlock()
}

func (s *Session) handleFileComplete() {
	s.fileTransferMu.Lock()
	transfer := s.activeTransfer
	s.fileTransferMu.Unlock()

	if transfer == nil {
		return
	}

	transfer.file.Close()

	// Verify file hash
	file, err := os.Open(transfer.outputPath)
	if err == nil {
		hash, err := hashFile(file)
		file.Close()
		if err == nil && hash == transfer.fileInfo.Hash {
			// Success
			if s.onMessage != nil {
				notifyMsg := &Message{
					Type:      MessageTypeText,
					ID:        generateID(),
					From:      s.peerName,
					Content:   fmt.Sprintf("✓ File received: %s (saved to %s)", transfer.fileInfo.Name, transfer.outputPath),
					Timestamp: time.Now().UnixMilli(),
				}
				s.onMessage(notifyMsg)
			}
		} else {
			os.Remove(transfer.outputPath)
			if s.onError != nil {
				s.onError(fmt.Errorf("file hash mismatch"))
			}
		}
	}

	// Clean up
	s.fileTransferMu.Lock()
	s.activeTransfer = nil
	s.fileProgress = nil
	s.fileTransferMu.Unlock()
}

func (s *Session) handleFileCancel() {
	s.fileTransferMu.Lock()
	transfer := s.activeTransfer
	s.fileTransferMu.Unlock()

	if transfer != nil {
		transfer.file.Close()
		os.Remove(transfer.outputPath)
	}

	s.fileTransferMu.Lock()
	s.activeTransfer = nil
	s.fileProgress = nil
	s.fileTransferMu.Unlock()
}

// --- File Transfer Utilities ---

func hashFile(file *os.File) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func hashBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
