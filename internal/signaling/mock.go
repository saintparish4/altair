package signaling

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

// MockConn is a mock WebSocket connection for testing.
type MockConn struct {
	mu          sync.Mutex
	closed      bool
	readQueue   [][]byte
	writeQueue  [][]byte
	readErr     error
	writeErr    error
	readDelay   time.Duration
	pongHandler func(string) error
}

// NewMockConn creates a new mock connection.
func NewMockConn() *MockConn {
	return &MockConn{
		readQueue:  make([][]byte, 0),
		writeQueue: make([][]byte, 0),
	}
}

// WriteMessage implements Conn.
func (m *MockConn) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return errors.New("connection closed")
	}
	if m.writeErr != nil {
		return m.writeErr
	}

	// Store a copy of the data
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	m.writeQueue = append(m.writeQueue, dataCopy)
	return nil
}

// ReadMessage implements Conn.
func (m *MockConn) ReadMessage() (int, []byte, error) {
	if m.readDelay > 0 {
		time.Sleep(m.readDelay)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, nil, errors.New("connection closed")
	}
	if m.readErr != nil {
		return 0, nil, m.readErr
	}
	if len(m.readQueue) == 0 {
		return 0, nil, errors.New("no messages in queue")
	}

	data := m.readQueue[0]
	m.readQueue = m.readQueue[1:]
	return TextMessage, data, nil
}

// Close implements Conn.
func (m *MockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// SetWriteDeadline implements Conn.
func (m *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline implements Conn.
func (m *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetReadLimit implements Conn.
func (m *MockConn) SetReadLimit(limit int64) {
	// No-op for mock
}

// SetPongHandler implements Conn.
func (m *MockConn) SetPongHandler(h func(appData string) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pongHandler = h
}

// --- Mock-specific methods for testing ---

// EnqueueRead adds a message to be returned by ReadMessage.
func (m *MockConn) EnqueueRead(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readQueue = append(m.readQueue, data)
}

// GetWritten returns all messages written to the connection.
func (m *MockConn) GetWritten() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writeQueue
}

// LastWritten returns the last message written.
func (m *MockConn) LastWritten() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.writeQueue) == 0 {
		return nil
	}
	return m.writeQueue[len(m.writeQueue)-1]
}

// SetReadError sets an error to be returned by ReadMessage.
func (m *MockConn) SetReadError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readErr = err
}

// SetWriteError sets an error to be returned by WriteMessage.
func (m *MockConn) SetWriteError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeErr = err
}

// SetReadDelay sets a delay before ReadMessage returns.
func (m *MockConn) SetReadDelay(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readDelay = d
}

// IsClosed returns whether the connection is closed.
func (m *MockConn) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// SimulatePong simulates receiving a pong message.
func (m *MockConn) SimulatePong() error {
	m.mu.Lock()
	handler := m.pongHandler
	m.mu.Unlock()

	if handler != nil {
		return handler("")
	}
	return nil
}

// --- Mock Upgrader ---

// MockUpgrader is a mock WebSocket upgrader for testing.
type MockUpgrader struct {
	Connections []*MockConn
	mu          sync.Mutex
	nextConn    *MockConn
	err         error
}

// NewMockUpgrader creates a new mock upgrader.
func NewMockUpgrader() *MockUpgrader {
	return &MockUpgrader{
		Connections: make([]*MockConn, 0),
	}
}

// Upgrade implements Upgrader.
func (m *MockUpgrader) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (Conn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		return nil, m.err
	}

	var conn *MockConn
	if m.nextConn != nil {
		conn = m.nextConn
		m.nextConn = nil
	} else {
		conn = NewMockConn()
	}

	m.Connections = append(m.Connections, conn)
	return conn, nil
}

// SetNextConnection sets the connection to be returned by the next Upgrade call.
func (m *MockUpgrader) SetNextConnection(conn *MockConn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextConn = conn
}

// SetError sets an error to be returned by Upgrade.
func (m *MockUpgrader) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// LastConnection returns the last connection created.
func (m *MockUpgrader) LastConnection() *MockConn {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Connections) == 0 {
		return nil
	}
	return m.Connections[len(m.Connections)-1]
}
