// Package transfer implements a P2P file transfer protocol with chunking and verification.
package transfer

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Protocol constants
const (
	// ChunkSize is the size of each file chunk (64KB)
	ChunkSize = 64 * 1024

	// MaxFileSize is the maximum supported file size (10GB)
	MaxFileSize = 10 * 1024 * 1024 * 1024

	// ProtocolVersion is the current protocol version
	ProtocolVersion = 1
)

// MessageType identifies the type of transfer message.
type MessageType uint8

const (
	MessageTypeHandshake MessageType = iota + 1
	MessageTypeHandshakeAck
	MessageTypeFileInfo
	MessageTypeFileAccept
	MessageTypeFileReject
	MessageTypeChunk
	MessageTypeChunkAck
	MessageTypeComplete
	MessageTypeError
	MessageTypeCancel
)

// HandshakeMsg initiates a transfer session.
type HandshakeMsg struct {
	Version  uint8  `json:"version"`
	SenderID string `json:"sender_id"`
}

// FileInfo describes the file being transferred.
type FileInfo struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Hash      string `json:"hash"` // SHA-256 of entire file
	ChunkSize int    `json:"chunk_size"`
	Chunks    int    `json:"chunks"`
}

// ChunkMsg carries a piece of the file.
type ChunkMsg struct {
	Index int    `json:"index"`
	Hash  string `json:"hash"` // SHA-256 of chunk data
	Size  int    `json:"size"`
	Data  []byte `json:"data"`
}

// ChunkAck acknowledges receipt of a chunk.
type ChunkAck struct {
	Index   int  `json:"index"`
	Success bool `json:"success"`
}

// ErrorMsg reports an error.
type ErrorMsg struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error codes
const (
	ErrCodeGeneral       = 1
	ErrCodeFileNotFound  = 2
	ErrCodeAccessDenied  = 3
	ErrCodeFileTooLarge  = 4
	ErrCodeHashMismatch  = 5
	ErrCodeDiskFull      = 6
	ErrCodeTransferAbort = 7
)

// --- Wire Protocol ---

// WriteMessage writes a typed message to the connection.
func WriteMessage(conn net.Conn, msgType MessageType, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	// Header: 1 byte type + 4 bytes length
	header := make([]byte, 5)
	header[0] = byte(msgType)
	binary.BigEndian.PutUint32(header[1:], uint32(len(data)))

	conn.SetWriteDeadline(time.Now().Add(30 * time.Second))

	if _, err := conn.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	return nil
}

// ReadMessage reads a typed message from the connection.
func ReadMessage(conn net.Conn, timeout time.Duration) (MessageType, []byte, error) {
	header := make([]byte, 5)

	conn.SetReadDeadline(time.Now().Add(timeout))

	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, nil, fmt.Errorf("read header: %w", err)
	}

	msgType := MessageType(header[0])
	length := binary.BigEndian.Uint32(header[1:])

	if length > 100*1024*1024 { // 100MB max message
		return 0, nil, errors.New("message too large")
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(conn, data); err != nil {
		return 0, nil, fmt.Errorf("read payload: %w", err)
	}

	return msgType, data, nil
}

// --- Progress Tracking ---

// Progress represents transfer progress.
type Progress struct {
	FileName       string
	FileSize       int64
	BytesSent      int64
	ChunksTotal    int
	ChunksDone     int
	StartTime      time.Time
	LastUpdateTime time.Time
	BytesPerSecond float64
}

// Percent returns the completion percentage.
func (p *Progress) Percent() float64 {
	if p.FileSize == 0 {
		return 0
	}
	return float64(p.BytesSent) / float64(p.FileSize) * 100
}

// ETA returns estimated time remaining.
func (p *Progress) ETA() time.Duration {
	if p.BytesPerSecond <= 0 {
		return 0
	}
	remaining := p.FileSize - p.BytesSent
	return time.Duration(float64(remaining)/p.BytesPerSecond) * time.Second
}

// ProgressCallback is called with transfer progress updates.
type ProgressCallback func(Progress)

// --- Sender ---

// Sender handles sending a file to a peer.
type Sender struct {
	conn       net.Conn
	filePath   string
	fileInfo   *FileInfo
	progress   Progress
	onProgress ProgressCallback
	mu         sync.Mutex
}

// NewSender creates a new file sender.
func NewSender(conn net.Conn, filePath string, onProgress ProgressCallback) *Sender {
	return &Sender{
		conn:       conn,
		filePath:   filePath,
		onProgress: onProgress,
	}
}

// Send initiates and completes the file transfer.
func (s *Sender) Send() error {
	// Open and analyze file
	file, err := os.Open(s.filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

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

	// Prepare file info
	chunks := int(stat.Size()/ChunkSize) + 1
	if stat.Size()%ChunkSize == 0 && stat.Size() > 0 {
		chunks = int(stat.Size() / ChunkSize)
	}

	s.fileInfo = &FileInfo{
		Name:      filepath.Base(s.filePath),
		Size:      stat.Size(),
		Hash:      hash,
		ChunkSize: ChunkSize,
		Chunks:    chunks,
	}

	s.progress = Progress{
		FileName:    s.fileInfo.Name,
		FileSize:    s.fileInfo.Size,
		ChunksTotal: chunks,
		StartTime:   time.Now(),
	}

	// Perform handshake
	if err := s.handshake(); err != nil {
		return fmt.Errorf("handshake: %w", err)
	}

	// Send file info and wait for acceptance
	if err := s.sendFileInfo(); err != nil {
		return fmt.Errorf("send file info: %w", err)
	}

	// Send chunks
	if err := s.sendChunks(file); err != nil {
		return fmt.Errorf("send chunks: %w", err)
	}

	// Send completion
	if err := WriteMessage(s.conn, MessageTypeComplete, nil); err != nil {
		return fmt.Errorf("send complete: %w", err)
	}

	return nil
}

func (s *Sender) handshake() error {
	msg := HandshakeMsg{
		Version:  ProtocolVersion,
		SenderID: fmt.Sprintf("sender-%d", time.Now().UnixNano()),
	}

	if err := WriteMessage(s.conn, MessageTypeHandshake, msg); err != nil {
		return err
	}

	msgType, _, err := ReadMessage(s.conn, 30*time.Second)
	if err != nil {
		return err
	}

	if msgType != MessageTypeHandshakeAck {
		return fmt.Errorf("expected handshake ack, got %d", msgType)
	}

	return nil
}

func (s *Sender) sendFileInfo() error {
	if err := WriteMessage(s.conn, MessageTypeFileInfo, s.fileInfo); err != nil {
		return err
	}

	msgType, data, err := ReadMessage(s.conn, 60*time.Second)
	if err != nil {
		return err
	}

	switch msgType {
	case MessageTypeFileAccept:
		return nil
	case MessageTypeFileReject:
		return errors.New("file rejected by receiver")
	case MessageTypeError:
		var errMsg ErrorMsg
		json.Unmarshal(data, &errMsg)
		return fmt.Errorf("receiver error: %s", errMsg.Message)
	default:
		return fmt.Errorf("unexpected response: %d", msgType)
	}
}

func (s *Sender) sendChunks(file *os.File) error {
	buf := make([]byte, ChunkSize)

	for i := 0; i < s.fileInfo.Chunks; i++ {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("read chunk %d: %w", i, err)
		}

		if n == 0 {
			break
		}

		chunk := ChunkMsg{
			Index: i,
			Hash:  hashBytes(buf[:n]),
			Size:  n,
			Data:  buf[:n],
		}

		if err := WriteMessage(s.conn, MessageTypeChunk, chunk); err != nil {
			return fmt.Errorf("send chunk %d: %w", i, err)
		}

		// Wait for ack
		msgType, data, err := ReadMessage(s.conn, 60*time.Second)
		if err != nil {
			return fmt.Errorf("read chunk ack %d: %w", i, err)
		}

		if msgType == MessageTypeError {
			var errMsg ErrorMsg
			json.Unmarshal(data, &errMsg)
			return fmt.Errorf("chunk %d error: %s", i, errMsg.Message)
		}

		if msgType != MessageTypeChunkAck {
			return fmt.Errorf("expected chunk ack, got %d", msgType)
		}

		var ack ChunkAck
		if err := json.Unmarshal(data, &ack); err != nil {
			return fmt.Errorf("parse chunk ack: %w", err)
		}

		if !ack.Success {
			return fmt.Errorf("chunk %d rejected", i)
		}

		// Update progress
		s.mu.Lock()
		s.progress.BytesSent += int64(n)
		s.progress.ChunksDone = i + 1
		elapsed := time.Since(s.progress.StartTime).Seconds()
		if elapsed > 0 {
			s.progress.BytesPerSecond = float64(s.progress.BytesSent) / elapsed
		}
		s.progress.LastUpdateTime = time.Now()
		if s.onProgress != nil {
			s.onProgress(s.progress)
		}
		s.mu.Unlock()
	}

	return nil
}

// --- Receiver ---

// Receiver handles receiving a file from a peer.
type Receiver struct {
	conn       net.Conn
	outputDir  string
	fileInfo   *FileInfo
	progress   Progress
	onProgress ProgressCallback
	onFileInfo func(*FileInfo) bool // Return true to accept
	mu         sync.Mutex
}

// NewReceiver creates a new file receiver.
func NewReceiver(conn net.Conn, outputDir string, onProgress ProgressCallback) *Receiver {
	return &Receiver{
		conn:       conn,
		outputDir:  outputDir,
		onProgress: onProgress,
		onFileInfo: func(*FileInfo) bool { return true }, // Accept all by default
	}
}

// SetFileInfoHandler sets the callback for file info (return true to accept).
func (r *Receiver) SetFileInfoHandler(handler func(*FileInfo) bool) {
	r.onFileInfo = handler
}

// Receive waits for and receives a file transfer.
func (r *Receiver) Receive() (string, error) {
	// Wait for handshake
	if err := r.handleHandshake(); err != nil {
		return "", fmt.Errorf("handshake: %w", err)
	}

	// Wait for file info
	if err := r.handleFileInfo(); err != nil {
		return "", fmt.Errorf("file info: %w", err)
	}

	// Receive chunks
	outputPath := filepath.Join(r.outputDir, r.fileInfo.Name)
	if err := r.receiveChunks(outputPath); err != nil {
		os.Remove(outputPath) // Clean up partial file
		return "", fmt.Errorf("receive chunks: %w", err)
	}

	// Verify final hash
	file, err := os.Open(outputPath)
	if err != nil {
		return "", fmt.Errorf("open for verify: %w", err)
	}
	defer file.Close()

	hash, err := hashFile(file)
	if err != nil {
		return "", fmt.Errorf("hash verify: %w", err)
	}

	if hash != r.fileInfo.Hash {
		os.Remove(outputPath)
		return "", fmt.Errorf("file hash mismatch: expected %s, got %s", r.fileInfo.Hash, hash)
	}

	return outputPath, nil
}

func (r *Receiver) handleHandshake() error {
	msgType, data, err := ReadMessage(r.conn, 60*time.Second)
	if err != nil {
		return err
	}

	if msgType != MessageTypeHandshake {
		return fmt.Errorf("expected handshake, got %d", msgType)
	}

	var msg HandshakeMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("parse handshake: %w", err)
	}

	if msg.Version != ProtocolVersion {
		WriteMessage(r.conn, MessageTypeError, ErrorMsg{
			Code:    ErrCodeGeneral,
			Message: fmt.Sprintf("version mismatch: got %d, want %d", msg.Version, ProtocolVersion),
		})
		return fmt.Errorf("version mismatch")
	}

	return WriteMessage(r.conn, MessageTypeHandshakeAck, nil)
}

func (r *Receiver) handleFileInfo() error {
	msgType, data, err := ReadMessage(r.conn, 60*time.Second)
	if err != nil {
		return err
	}

	if msgType != MessageTypeFileInfo {
		return fmt.Errorf("expected file info, got %d", msgType)
	}

	var info FileInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return fmt.Errorf("parse file info: %w", err)
	}

	r.fileInfo = &info

	// Check if file should be accepted
	if !r.onFileInfo(&info) {
		WriteMessage(r.conn, MessageTypeFileReject, nil)
		return errors.New("file rejected")
	}

	r.progress = Progress{
		FileName:    info.Name,
		FileSize:    info.Size,
		ChunksTotal: info.Chunks,
		StartTime:   time.Now(),
	}

	return WriteMessage(r.conn, MessageTypeFileAccept, nil)
}

func (r *Receiver) receiveChunks(outputPath string) error {
	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	for {
		msgType, data, err := ReadMessage(r.conn, 120*time.Second)
		if err != nil {
			return fmt.Errorf("read message: %w", err)
		}

		switch msgType {
		case MessageTypeChunk:
			var chunk ChunkMsg
			if err := json.Unmarshal(data, &chunk); err != nil {
				return fmt.Errorf("parse chunk: %w", err)
			}

			// Verify chunk hash
			if hashBytes(chunk.Data) != chunk.Hash {
				WriteMessage(r.conn, MessageTypeChunkAck, ChunkAck{
					Index:   chunk.Index,
					Success: false,
				})
				return fmt.Errorf("chunk %d hash mismatch", chunk.Index)
			}

			// Write chunk
			offset := int64(chunk.Index) * int64(r.fileInfo.ChunkSize)
			if _, err := file.WriteAt(chunk.Data, offset); err != nil {
				return fmt.Errorf("write chunk %d: %w", chunk.Index, err)
			}

			// Send ack
			if err := WriteMessage(r.conn, MessageTypeChunkAck, ChunkAck{
				Index:   chunk.Index,
				Success: true,
			}); err != nil {
				return fmt.Errorf("send ack %d: %w", chunk.Index, err)
			}

			// Update progress
			r.mu.Lock()
			r.progress.BytesSent += int64(chunk.Size)
			r.progress.ChunksDone = chunk.Index + 1
			elapsed := time.Since(r.progress.StartTime).Seconds()
			if elapsed > 0 {
				r.progress.BytesPerSecond = float64(r.progress.BytesSent) / elapsed
			}
			r.progress.LastUpdateTime = time.Now()
			if r.onProgress != nil {
				r.onProgress(r.progress)
			}
			r.mu.Unlock()

		case MessageTypeComplete:
			return nil

		case MessageTypeCancel:
			return errors.New("transfer cancelled by sender")

		case MessageTypeError:
			var errMsg ErrorMsg
			json.Unmarshal(data, &errMsg)
			return fmt.Errorf("sender error: %s", errMsg.Message)

		default:
			return fmt.Errorf("unexpected message type: %d", msgType)
		}
	}
}

// --- Utilities ---

func hashFile(file *os.File) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// FormatBytes returns a human-readable byte size.
func FormatBytes(bytes int64) string {
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

// FormatDuration returns a human-readable duration.
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

// --- Progress Bar ---

// ProgressBar renders a terminal progress bar.
type ProgressBar struct {
	width int
}

// NewProgressBar creates a new progress bar with the given width.
func NewProgressBar(width int) *ProgressBar {
	if width <= 0 {
		width = 40
	}
	return &ProgressBar{width: width}
}

// Render returns the progress bar string.
func (pb *ProgressBar) Render(p Progress) string {
	percent := p.Percent()
	filled := int(percent / 100 * float64(pb.width))
	if filled > pb.width {
		filled = pb.width
	}

	bar := ""
	for i := 0; i < pb.width; i++ {
		if i < filled {
			bar += "█"
		} else if i == filled {
			bar += "▓"
		} else {
			bar += "░"
		}
	}

	eta := ""
	if p.BytesPerSecond > 0 && p.BytesSent < p.FileSize {
		eta = fmt.Sprintf(" ETA: %s", FormatDuration(p.ETA()))
	}

	speed := ""
	if p.BytesPerSecond > 0 {
		speed = fmt.Sprintf(" %s/s", FormatBytes(int64(p.BytesPerSecond)))
	}

	return fmt.Sprintf("\r[%s] %.1f%% %s/%s%s%s",
		bar,
		percent,
		FormatBytes(p.BytesSent),
		FormatBytes(p.FileSize),
		speed,
		eta,
	)
}
