package transfer

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteReadMessage(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Write message in goroutine
	done := make(chan error, 1)
	go func() {
		payload := map[string]string{"test": "data"}
		done <- WriteMessage(client, MessageTypeHandshake, payload)
	}()

	// Read message
	msgType, data, err := ReadMessage(server, 5*time.Second)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if msgType != MessageTypeHandshake {
		t.Errorf("expected MessageTypeHandshake, got %d", msgType)
	}

	var payload map[string]string
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if payload["test"] != "data" {
		t.Errorf("expected 'data', got '%s'", payload["test"])
	}

	// Check write completed
	if err := <-done; err != nil {
		t.Errorf("write error: %v", err)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		got := FormatBytes(tt.bytes)
		if got != tt.want {
			t.Errorf("FormatBytes(%d) = %s, want %s", tt.bytes, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m 30s"},
		{3661 * time.Second, "1h 1m"},
	}

	for _, tt := range tests {
		got := FormatDuration(tt.duration)
		if got != tt.want {
			t.Errorf("FormatDuration(%v) = %s, want %s", tt.duration, got, tt.want)
		}
	}
}

func TestProgress(t *testing.T) {
	p := Progress{
		FileSize:       1000,
		BytesSent:      500,
		BytesPerSecond: 100,
	}

	if p.Percent() != 50.0 {
		t.Errorf("expected 50%%, got %.1f%%", p.Percent())
	}

	eta := p.ETA()
	if eta != 5*time.Second {
		t.Errorf("expected 5s ETA, got %v", eta)
	}
}

func TestProgressZeroSize(t *testing.T) {
	p := Progress{
		FileSize:  0,
		BytesSent: 0,
	}

	if p.Percent() != 0 {
		t.Errorf("expected 0%%, got %.1f%%", p.Percent())
	}
}

func TestProgressBarRender(t *testing.T) {
	pb := NewProgressBar(10)

	tests := []struct {
		percent float64
		wantMin string
	}{
		{0, "░░░░░░░░░░"},
		{50, "█████"},
		{100, "██████████"},
	}

	for _, tt := range tests {
		p := Progress{
			FileSize:  100,
			BytesSent: int64(tt.percent),
		}
		result := pb.Render(p)
		if result == "" {
			t.Error("render should not return empty string")
		}
	}
}

func TestNewProgressBar(t *testing.T) {
	pb1 := NewProgressBar(0)
	if pb1.width != 40 {
		t.Errorf("expected default width 40, got %d", pb1.width)
	}

	pb2 := NewProgressBar(50)
	if pb2.width != 50 {
		t.Errorf("expected width 50, got %d", pb2.width)
	}
}

func TestHashBytes(t *testing.T) {
	data := []byte("test data")
	hash1 := hashBytes(data)
	hash2 := hashBytes(data)

	if hash1 != hash2 {
		t.Error("same data should produce same hash")
	}

	hash3 := hashBytes([]byte("different data"))
	if hash1 == hash3 {
		t.Error("different data should produce different hash")
	}

	// Check hash format (should be hex string of SHA-256)
	if len(hash1) != 64 {
		t.Errorf("expected 64 char hash, got %d", len(hash1))
	}
}

func TestFileInfoJSON(t *testing.T) {
	info := FileInfo{
		Name:      "test.txt",
		Size:      1234,
		Hash:      "abc123",
		ChunkSize: ChunkSize,
		Chunks:    10,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded FileInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Name != info.Name {
		t.Errorf("name mismatch: got %s, want %s", decoded.Name, info.Name)
	}
	if decoded.Size != info.Size {
		t.Errorf("size mismatch: got %d, want %d", decoded.Size, info.Size)
	}
}

func TestChunkMsgJSON(t *testing.T) {
	chunk := ChunkMsg{
		Index: 5,
		Hash:  "abc123",
		Size:  1024,
		Data:  []byte("chunk data"),
	}

	data, err := json.Marshal(chunk)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded ChunkMsg
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Index != chunk.Index {
		t.Errorf("index mismatch: got %d, want %d", decoded.Index, chunk.Index)
	}
	if !bytes.Equal(decoded.Data, chunk.Data) {
		t.Error("data mismatch")
	}
}

func TestHandshakeMsg(t *testing.T) {
	msg := HandshakeMsg{
		Version:  ProtocolVersion,
		SenderID: "test-sender",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded HandshakeMsg
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Version != ProtocolVersion {
		t.Errorf("version mismatch: got %d, want %d", decoded.Version, ProtocolVersion)
	}
}

func TestErrorMsg(t *testing.T) {
	msg := ErrorMsg{
		Code:    ErrCodeHashMismatch,
		Message: "hash verification failed",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded ErrorMsg
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Code != ErrCodeHashMismatch {
		t.Errorf("code mismatch: got %d, want %d", decoded.Code, ErrCodeHashMismatch)
	}
}

func TestChunkAck(t *testing.T) {
	ack := ChunkAck{
		Index:   10,
		Success: true,
	}

	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded ChunkAck
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Index != 10 || !decoded.Success {
		t.Error("ack fields mismatch")
	}
}

func TestMessageTypes(t *testing.T) {
	// Verify all message types are unique
	types := []MessageType{
		MessageTypeHandshake,
		MessageTypeHandshakeAck,
		MessageTypeFileInfo,
		MessageTypeFileAccept,
		MessageTypeFileReject,
		MessageTypeChunk,
		MessageTypeChunkAck,
		MessageTypeComplete,
		MessageTypeError,
		MessageTypeCancel,
	}

	seen := make(map[MessageType]bool)
	for _, mt := range types {
		if seen[mt] {
			t.Errorf("duplicate message type: %d", mt)
		}
		seen[mt] = true
	}
}

func TestConstants(t *testing.T) {
	if ChunkSize <= 0 {
		t.Error("ChunkSize should be positive")
	}
	if MaxFileSize <= 0 {
		t.Error("MaxFileSize should be positive")
	}
	if ProtocolVersion <= 0 {
		t.Error("ProtocolVersion should be positive")
	}
}

// Integration test for sender/receiver
func TestSenderReceiverIntegration(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("Hello, this is test content for file transfer!")

	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create output directory
	outputDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	// Create pipe for testing
	server, client := net.Pipe()

	// Track progress
	senderProgress := make([]Progress, 0)
	receiverProgress := make([]Progress, 0)

	// Run receiver in goroutine
	receiveDone := make(chan error, 1)
	var receivedPath string

	go func() {
		receiver := NewReceiver(server, outputDir, func(p Progress) {
			receiverProgress = append(receiverProgress, p)
		})
		receiver.SetFileInfoHandler(func(info *FileInfo) bool {
			return true // Accept all
		})
		path, err := receiver.Receive()
		receivedPath = path
		receiveDone <- err
		server.Close()
	}()

	// Run sender
	sender := NewSender(client, testFile, func(p Progress) {
		senderProgress = append(senderProgress, p)
	})

	sendErr := sender.Send()
	client.Close()

	if sendErr != nil {
		t.Fatalf("sender error: %v", sendErr)
	}

	// Wait for receiver
	select {
	case err := <-receiveDone:
		if err != nil {
			t.Fatalf("receiver error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("receiver timeout")
	}

	// Verify received file
	receivedContent, err := os.ReadFile(receivedPath)
	if err != nil {
		t.Fatalf("failed to read received file: %v", err)
	}

	if !bytes.Equal(receivedContent, testContent) {
		t.Error("received content does not match original")
	}

	// Verify progress was tracked
	if len(senderProgress) == 0 {
		t.Error("sender progress not tracked")
	}
	if len(receiverProgress) == 0 {
		t.Error("receiver progress not tracked")
	}
}

func TestHashFile(t *testing.T) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "test-hash-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := []byte("test file content for hashing")
	tmpFile.Write(content)
	tmpFile.Seek(0, io.SeekStart)

	hash, err := hashFile(tmpFile)
	if err != nil {
		t.Fatalf("hashFile error: %v", err)
	}

	if len(hash) != 64 {
		t.Errorf("expected 64 char hash, got %d", len(hash))
	}

	// Hash again should be same
	tmpFile.Seek(0, io.SeekStart)
	hash2, _ := hashFile(tmpFile)

	if hash != hash2 {
		t.Error("same file should produce same hash")
	}
}

func TestReceiverSetFileInfoHandler(t *testing.T) {
	server, _ := net.Pipe()
	defer server.Close()

	receiver := NewReceiver(server, ".", nil)

	// Default should accept
	if !receiver.onFileInfo(&FileInfo{Name: "test.txt"}) {
		t.Error("default handler should accept")
	}

	// Set custom handler
	receiver.SetFileInfoHandler(func(info *FileInfo) bool {
		return info.Size < 1000
	})

	if !receiver.onFileInfo(&FileInfo{Name: "small.txt", Size: 500}) {
		t.Error("should accept small file")
	}

	if receiver.onFileInfo(&FileInfo{Name: "large.txt", Size: 5000}) {
		t.Error("should reject large file")
	}
}
