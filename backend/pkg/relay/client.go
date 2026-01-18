package relay

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Allocation represents a relay allocation (similar to TURN)
type Allocation struct {
	// Relay address (the address others should send to)
	RelayAddr *net.UDPAddr

	// Client's reflexive address (as seen by relay server)
	ReflexiveAddr *net.UDPAddr

	// Lifetime of the allocation
	Lifetime time.Duration

	// When the allocation expires
	ExpiresAt time.Time

	// Allocation ID (for refresh requests)
	ID string
}

// String returns a human-readable representation of the allocation
func (a *Allocation) String() string {
	if a == nil {
		return "<nil allocation>"
	}
	remaining := time.Until(a.ExpiresAt)
	return fmt.Sprintf("Relay: %s, Expires in: %v (ID: %s)", a.RelayAddr, remaining, a.ID)
}

// IsValid checks if the allocation is still valid
func (a *Allocation) IsValid() bool {
	if a == nil {
		return false
	}
	return time.Now().Before(a.ExpiresAt)
}

// TimeRemaining returns the time remaining before expiration
func (a *Allocation) TimeRemaining() time.Duration {
	if !a.IsValid() {
		return 0
	}
	return time.Until(a.ExpiresAt)
}

// Client is a relay client for establishing relayed connections
type Client struct {
	serverAddr *net.UDPAddr
	conn       *net.UDPConn
	allocation *Allocation

	timeout time.Duration

	// Receive buffer and handlers
	recvBuf     []byte
	recvHandlers map[string]func([]byte, *net.UDPAddr)
	recvMu      sync.RWMutex

	// State
	closed bool
	mu     sync.RWMutex
}

// ClientConfig holds configuration for the relay client
type ClientConfig struct {
	// Relay server address
	ServerAddr string

	// Allocation lifetime to request
	Lifetime time.Duration

	// Timeout for relay operations
	Timeout time.Duration

	// Optional existing connection
	Conn *net.UDPConn
}

// DefaultClientConfig returns a configuration with sensible defaults
func DefaultClientConfig(serverAddr string) *ClientConfig {
	return &ClientConfig{
		ServerAddr: serverAddr,
		Lifetime:   10 * time.Minute,
		Timeout:    5 * time.Second,
	}
}

// NewClient creates a new relay client
func NewClient(config *ClientConfig) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Resolve server address
	serverAddr, err := net.ResolveUDPAddr("udp", config.ServerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server address: %w", err)
	}

	// Create or use existing connection
	var conn *net.UDPConn
	if config.Conn != nil {
		conn = config.Conn
	} else {
		conn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
		if err != nil {
			return nil, fmt.Errorf("failed to create UDP connection: %w", err)
		}
	}

	client := &Client{
		serverAddr:   serverAddr,
		conn:         conn,
		timeout:      config.Timeout,
		recvBuf:      make([]byte, 65536),
		recvHandlers: make(map[string]func([]byte, *net.UDPAddr)),
	}

	return client, nil
}

// Allocate requests a relay allocation from the server
func (c *Client) Allocate(lifetime time.Duration) (*Allocation, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, fmt.Errorf("client is closed")
	}

	// In a real implementation, this would send a TURN Allocate request
	// For this simplified version, we simulate the allocation
	
	// Generate allocation ID
	allocID := fmt.Sprintf("alloc-%d", time.Now().UnixNano())

	// In real TURN, the server would assign us a relay address
	// For simulation, we use a derived address
	relayAddr := &net.UDPAddr{
		IP:   c.serverAddr.IP,
		Port: c.serverAddr.Port + 1, // Simulated relay port
	}

	allocation := &Allocation{
		RelayAddr:     relayAddr,
		ReflexiveAddr: c.conn.LocalAddr().(*net.UDPAddr),
		Lifetime:      lifetime,
		ExpiresAt:     time.Now().Add(lifetime),
		ID:            allocID,
	}

	c.allocation = allocation
	return allocation, nil
}

// Refresh extends the lifetime of an existing allocation
func (c *Client) Refresh(duration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.allocation == nil {
		return fmt.Errorf("no allocation to refresh")
	}

	if !c.allocation.IsValid() {
		return fmt.Errorf("allocation has expired")
	}

	// Extend expiration time
	c.allocation.ExpiresAt = time.Now().Add(duration)
	c.allocation.Lifetime = duration

	return nil
}

// Send sends data to a peer through the relay
func (c *Client) Send(data []byte, peer *net.UDPAddr) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	if c.allocation == nil {
		return fmt.Errorf("no allocation - call Allocate() first")
	}

	if !c.allocation.IsValid() {
		return fmt.Errorf("allocation has expired")
	}

	// In real TURN, we would wrap data in a Send indication
	// For this simplified version, send directly
	_, err := c.conn.WriteToUDP(data, peer)
	if err != nil {
		return fmt.Errorf("failed to send data: %w", err)
	}

	return nil
}

// Receive receives data from a peer through the relay
func (c *Client) Receive() ([]byte, *net.UDPAddr, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, nil, fmt.Errorf("client is closed")
	}
	c.mu.RUnlock()

	// Set read deadline
	deadline := time.Now().Add(c.timeout)
	if err := c.conn.SetReadDeadline(deadline); err != nil {
		return nil, nil, fmt.Errorf("failed to set deadline: %w", err)
	}
	defer c.conn.SetReadDeadline(time.Time{})

	// Read data
	n, addr, err := c.conn.ReadFromUDP(c.recvBuf)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to receive data: %w", err)
	}

	// Return copy of data
	data := make([]byte, n)
	copy(data, c.recvBuf[:n])

	return data, addr, nil
}

// ReceiveFrom receives data from a specific peer
func (c *Client) ReceiveFrom(peer *net.UDPAddr, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		data, addr, err := c.Receive()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return nil, err
		}

		// Check if data is from expected peer
		if addr.IP.Equal(peer.IP) && addr.Port == peer.Port {
			return data, nil
		}
	}

	return nil, fmt.Errorf("timeout waiting for data from %s", peer)
}

// CreatePermission creates a permission for a peer to send through the relay
// In real TURN, this would send a CreatePermission request
func (c *Client) CreatePermission(peer *net.UDPAddr) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.allocation == nil {
		return fmt.Errorf("no allocation")
	}

	if !c.allocation.IsValid() {
		return fmt.Errorf("allocation has expired")
	}

	// In real TURN, we would send a CreatePermission request
	// For this simplified version, permissions are implicit

	return nil
}

// Close closes the relay client and releases the allocation
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	// In real TURN, we would send a Refresh with lifetime=0 to release
	c.allocation = nil

	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

// Allocation returns the current allocation
func (c *Client) Allocation() *Allocation {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.allocation
}

// LocalAddr returns the local address
func (c *Client) LocalAddr() *net.UDPAddr {
	if c.conn != nil {
		return c.conn.LocalAddr().(*net.UDPAddr)
	}
	return nil
}

// ServerAddr returns the relay server address
func (c *Client) ServerAddr() *net.UDPAddr {
	return c.serverAddr
}

// QuickRelay is a convenience function for creating a relay connection
func QuickRelay(serverAddr string, lifetime time.Duration) (*Client, *Allocation, error) {
	config := &ClientConfig{
		ServerAddr: serverAddr,
		Lifetime:   lifetime,
		Timeout:    5 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		return nil, nil, err
	}

	allocation, err := client.Allocate(lifetime)
	if err != nil {
		client.Close()
		return nil, nil, err
	}

	return client, allocation, nil
}

// TODO: Note: This is a simplified relay implementation for demonstration.
// A full TURN implementation would include:
// - Proper STUN message encoding for TURN messages
// - Channel bindings for efficiency
// - Authentication and authorization
// - Bandwidth management
// - Multiple relay address families (IPv4/IPv6)