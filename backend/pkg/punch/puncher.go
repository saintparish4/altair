package punch

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/saintparish4/altair/pkg/nat"
)

// PeerInfo contains information about a peer for hole punching
type PeerInfo struct {
	// Public endpoint discovered via STUN
	PublicAddr *net.UDPAddr

	// Local (private) addresses that might work if on same network
	LocalAddrs []*net.UDPAddr

	// NAT type of the peer
	NATType nat.Type
}

// Connection represents a successfully established P2P connection
type Connection struct {
	// Local address used for the connection
	LocalAddr *net.UDPAddr

	// Remote address of the peer
	RemoteAddr *net.UDPAddr

	// UDP connection
	Conn *net.UDPConn

	// Round-trip time measured during hole punching
	RTT time.Duration

	// Whether connection was established via relay
	IsRelayed bool

	// Timestamp when connection was established
	EstablishedAt time.Time
}

// Close closes the connection
func (c *Connection) Close() error {
	if c.Conn != nil {
		return c.Conn.Close()
	}
	return nil
}

// String returns a human-readable representation of the connection
func (c *Connection) String() string {
	relayed := ""
	if c.IsRelayed {
		relayed = " (relayed)"
	}
	return fmt.Sprintf("%s <-> %s (RTT: %v)%s", c.LocalAddr, c.RemoteAddr, c.RTT, relayed)
}

// Puncher performs UDP hole punching to establish P2P connections
type Puncher struct {
	localAddr *net.UDPAddr
	mapping   *nat.Mapping
	conn      *net.UDPConn

	timeout      time.Duration
	pingInterval time.Duration
	maxAttempts  int

	mu sync.Mutex
}

// PuncherConfig holds configuration for the hole puncher
type PuncherConfig struct {
	// Local address to bind to (optional, uses 0.0.0.0:0 if nil)
	LocalAddr *net.UDPAddr

	// NAT mapping information
	Mapping *nat.Mapping

	// Timeout for hole punching attempts
	Timeout time.Duration

	// Interval between ping packets during hole punching
	PingInterval time.Duration

	// Maximum number of punch attempts
	MaxAttempts int

	// Existing connection to use (optional)
	Conn *net.UDPConn
}

// DefaultPuncherConfig returns a configuration with sensible defaults
func DefaultPuncherConfig() *PuncherConfig {
	return &PuncherConfig{
		Timeout:      30 * time.Second,
		PingInterval: 200 * time.Millisecond,
		MaxAttempts:  50,
	}
}

// NewPuncher creates a new UDP hole puncher
func NewPuncher(config *PuncherConfig) (*Puncher, error) {
	if config == nil {
		config = DefaultPuncherConfig()
	}

	var conn *net.UDPConn
	var localAddr *net.UDPAddr
	var err error

	if config.Conn != nil {
		// Use existing connection
		conn = config.Conn
		localAddr = conn.LocalAddr().(*net.UDPAddr)
	} else {
		// Create new connection
		if config.LocalAddr != nil {
			localAddr = config.LocalAddr
		} else {
			localAddr = &net.UDPAddr{IP: net.IPv4zero, Port: 0}
		}

		conn, err = net.ListenUDP("udp", localAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to create UDP socket: %w", err)
		}
		localAddr = conn.LocalAddr().(*net.UDPAddr)
	}

	return &Puncher{
		localAddr:    localAddr,
		mapping:      config.Mapping,
		conn:         conn,
		timeout:      config.Timeout,
		pingInterval: config.PingInterval,
		maxAttempts:  config.MaxAttempts,
	}, nil
}

// PunchHole attempts to establish a P2P connection with a peer
// Uses simultaneous UDP hole punching technique
func (p *Puncher) PunchHole(peer *PeerInfo) (*Connection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if peer == nil {
		return nil, fmt.Errorf("peer info cannot be nil")
	}

	if peer.PublicAddr == nil {
		return nil, fmt.Errorf("peer public address cannot be nil")
	}

	// Check if hole punching is likely to succeed
	if p.mapping != nil && peer.NATType != nat.TypeUnknown {
		if !nat.CanHolePunch(p.mapping.Type, peer.NATType) {
			return nil, fmt.Errorf("hole punching unlikely to succeed: %s <-> %s",
				p.mapping.Type, peer.NATType)
		}
	}

	// Try local addresses first (in case on same network)
	for _, localAddr := range peer.LocalAddrs {
		conn, err := p.tryDirectConnection(localAddr, 2*time.Second)
		if err == nil {
			return conn, nil
		}
	}

	// Try public address with hole punching
	return p.simultaneousPunch(peer.PublicAddr)
}

// tryDirectConnection attempts a direct connection (for LAN peers)
func (p *Puncher) tryDirectConnection(addr *net.UDPAddr, timeout time.Duration) (*Connection, error) {
	start := time.Now()
	deadline := start.Add(timeout)

	// Send ping
	ping := []byte("PING")
	_, err := p.conn.WriteToUDP(ping, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to send ping: %w", err)
	}

	// Wait for pong
	p.conn.SetReadDeadline(deadline)
	defer p.conn.SetReadDeadline(time.Time{})

	buf := make([]byte, 1500)
	for time.Now().Before(deadline) {
		n, remoteAddr, err := p.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return nil, err
		}

		if n >= 4 && string(buf[:4]) == "PONG" {
			return &Connection{
				LocalAddr:     p.localAddr,
				RemoteAddr:    remoteAddr,
				Conn:          p.conn,
				RTT:           time.Since(start),
				IsRelayed:     false,
				EstablishedAt: time.Now(),
			}, nil
		}
	}

	return nil, fmt.Errorf("no response from peer")
}

// simultaneousPunch performs simultaneous UDP hole punching
func (p *Puncher) simultaneousPunch(peerAddr *net.UDPAddr) (*Connection, error) {
	start := time.Now()
	deadline := start.Add(p.timeout)

	// Channel to receive responses
	responses := make(chan *Connection, 1)
	errors := make(chan error, 1)

	// Start sender goroutine
	go func() {
		ping := []byte("PING")
		attempt := 0

		for time.Now().Before(deadline) && attempt < p.maxAttempts {
			// Send ping packet
			_, err := p.conn.WriteToUDP(ping, peerAddr)
			if err != nil {
				errors <- fmt.Errorf("failed to send ping: %w", err)
				return
			}

			attempt++
			time.Sleep(p.pingInterval)
		}
	}()

	// Start receiver goroutine
	go func() {
		buf := make([]byte, 1500)
		p.conn.SetReadDeadline(deadline)
		defer p.conn.SetReadDeadline(time.Time{})

		for time.Now().Before(deadline) {
			n, remoteAddr, err := p.conn.ReadFromUDP(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					errors <- fmt.Errorf("hole punching timed out")
					return
				}
				errors <- fmt.Errorf("read error: %w", err)
				return
			}

			// Check if it's a PING (peer is trying to punch to us)
			if n >= 4 && string(buf[:4]) == "PING" {
				// Send PONG back
				pong := []byte("PONG")
				p.conn.WriteToUDP(pong, remoteAddr)
				continue
			}

			// Check if it's a PONG (our punch succeeded)
			if n >= 4 && string(buf[:4]) == "PONG" {
				responses <- &Connection{
					LocalAddr:     p.localAddr,
					RemoteAddr:    remoteAddr,
					Conn:          p.conn,
					RTT:           time.Since(start),
					IsRelayed:     false,
					EstablishedAt: time.Now(),
				}
				return
			}
		}
	}()

	// Wait for success or timeout
	select {
	case conn := <-responses:
		return conn, nil
	case err := <-errors:
		return nil, err
	case <-time.After(p.timeout):
		return nil, fmt.Errorf("hole punching timed out after %v", p.timeout)
	}
}

// PunchWithRetry attempts hole punching with automatic retry
func (p *Puncher) PunchWithRetry(peer *PeerInfo, retries int) (*Connection, error) {
	var lastErr error

	for attempt := 0; attempt <= retries; attempt++ {
		conn, err := p.PunchHole(peer)
		if err == nil {
			return conn, nil
		}

		lastErr = err

		// Wait before retry (exponential backoff)
		if attempt < retries {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			if backoff > 10*time.Second {
				backoff = 10 * time.Second
			}
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("hole punching failed after %d attempts: %w", retries, lastErr)
}

// Close closes the hole puncher and releases resources
func (p *Puncher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

// LocalAddr returns the local address being used
func (p *Puncher) LocalAddr() *net.UDPAddr {
	return p.localAddr
}

// Mapping returns the NAT mapping information
func (p *Puncher) Mapping() *nat.Mapping {
	return p.mapping
}

// QuickPunch is a convenience function for one-off hole punching
func QuickPunch(peer *PeerInfo, mapping *nat.Mapping) (*Connection, error) {
	config := &PuncherConfig{
		Mapping:      mapping,
		Timeout:      30 * time.Second,
		PingInterval: 200 * time.Millisecond,
		MaxAttempts:  50,
	}

	puncher, err := NewPuncher(config)
	if err != nil {
		return nil, err
	}
	// Note: Don't close puncher - connection uses its socket
	// Caller is responsible for closing the returned connection

	return puncher.PunchWithRetry(peer, 3)
}
