package stun

import (
	"fmt"
	"net"
	"time"
)

// Endpoint represents a discovered network endpoint
type Endpoint struct {
	LocalAddr  *net.UDPAddr
	PublicAddr *net.UDPAddr
	ServerAddr *net.UDPAddr
}

// Client is a STUN client for discovering public endpoints
type Client struct {
	conn       *net.UDPConn
	serverAddr *net.UDPAddr
	timeout    time.Duration
}

// ClientConfig holds configuration for creating a STUN client
type ClientConfig struct {
	ServerAddr string        // STUN server address (host:port)
	LocalAddr  string        // Optional local address to bind to
	Timeout    time.Duration // Request timeout
}

// DefaultTimeout is the default timeout for STUN requests
const DefaultTimeout = 5 * time.Second

// NewClient creates a new STUN client
func NewClient(config *ClientConfig) (*Client, error) {
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}

	// Resolve server address
	serverAddr, err := net.ResolveUDPAddr("udp", config.ServerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server address: %w", err)
	}

	// Create UDP connection
	var localAddr *net.UDPAddr
	if config.LocalAddr != "" {
		localAddr, err = net.ResolveUDPAddr("udp", config.LocalAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve local address: %w", err)
		}
	}

	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP connection: %w", err)
	}

	return &Client{
		conn:       conn,
		serverAddr: serverAddr,
		timeout:    config.Timeout,
	}, nil
}

// Discover performs endpoint discovery using a STUN binding request
func (c *Client) Discover() (*Endpoint, error) {
	// Create binding request
	request, err := NewMessage(TypeBindingRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create binding request: %w", err)
	}

	// Encode message
	data, err := request.Encode()
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	// Send request
	_, err = c.conn.WriteToUDP(data, c.serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Set read deadline
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}
	defer c.conn.SetReadDeadline(time.Time{}) // Clear deadline

	// Wait for response
	buf := make([]byte, 1500) // MTU size
	n, _, err := c.conn.ReadFromUDP(buf)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, fmt.Errorf("STUN request timed out after %v", c.timeout)
		}
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Decode response
	response, err := Decode(buf[:n])
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Verify transaction ID matches
	if response.TransactionID != request.TransactionID {
		return nil, fmt.Errorf("transaction ID mismatch")
	}

	// Check response type
	if response.Type != TypeBindingSuccess {
		return nil, fmt.Errorf("received error response: %s", response.Type)
	}

	// Extract public address from XOR-MAPPED-ADDRESS
	attr, found := response.GetAttribute(AttrXORMappedAddress)
	if !found {
		// Fallback to MAPPED-ADDRESS
		attr, found = response.GetAttribute(AttrMappedAddress)
		if !found {
			return nil, fmt.Errorf("no address attribute in response")
		}
		publicAddr, err := DecodeMappedAddress(attr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode MAPPED-ADDRESS: %w", err)
		}

		return &Endpoint{
			LocalAddr:  c.conn.LocalAddr().(*net.UDPAddr),
			PublicAddr: publicAddr,
			ServerAddr: c.serverAddr,
		}, nil
	}

	publicAddr, err := DecodeXORMappedAddress(attr, request.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode XOR-MAPPED-ADDRESS: %w", err)
	}

	return &Endpoint{
		LocalAddr:  c.conn.LocalAddr().(*net.UDPAddr),
		PublicAddr: publicAddr,
		ServerAddr: c.serverAddr,
	}, nil
}

// DiscoverWithRetry attempts endpoint discovery with retry logic
func (c *Client) DiscoverWithRetry(maxRetries int) (*Endpoint, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		endpoint, err := c.Discover()
		if err == nil {
			return endpoint, nil
		}

		lastErr = err

		// Wait before retry (exponential backoff)
		if attempt < maxRetries {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			if backoff > 10*time.Second {
				backoff = 10 * time.Second
			}
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("discovery failed after %d attempts: %w", maxRetries, lastErr)
}

// Close closes the STUN client and releases resources
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// LocalAddr returns the local address the client is bound to
func (c *Client) LocalAddr() *net.UDPAddr {
	if c.conn != nil {
		return c.conn.LocalAddr().(*net.UDPAddr)
	}
	return nil
}

// ServerAddr returns the STUN server address
func (c *Client) ServerAddr() *net.UDPAddr {
	return c.serverAddr
}

// String returns a string representation of the endpoint
func (e *Endpoint) String() string {
	return fmt.Sprintf("Local: %s, Public: %s (via %s)",
		e.LocalAddr, e.PublicAddr, e.ServerAddr)
}
