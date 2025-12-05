package holepunch

import (
	"fmt"
	"net"
	"time"

	"github.com/saintparish4/altair/pkg/types"
)

const (
	// DefaultAttempts is the number of punch attempts
	DefaultAttempts = 5

	// DefaultInterval between punch attempts
	DefaultInterval = 400 * time.Millisecond // 5 attempts over ~2 seconds

	// DefaultTimeout for overall connection establishment
	DefaultTimeout = 10 * time.Second
)

// ConnectionConfig holds configuration for connection establishment
type ConnectionConfig struct {
	Attempts int
	Interval time.Duration
	Timeout  time.Duration
}

// DefaultConfig returns the default connection configuration
func DefaultConfig() ConnectionConfig {
	return ConnectionConfig{
		Attempts: DefaultAttempts,
		Interval: DefaultInterval,
		Timeout:  DefaultTimeout,
	}
}

// EstablishConnection establishes a direct UDP connection through NAT hole punching
// localEndpoint: the local private endpoint (before NAT)
// remotePublicEndpoint: the remote peer's public endpoint (after their NAT)
// Returns an established UDP connection ready for data exchange
func EstablishConnection(localEndpoint, remotePublicEndpoint *types.Endpoint) (*net.UDPConn, error) {
	return EstablishConnectionWithConfig(localEndpoint, remotePublicEndpoint, DefaultConfig())
}

// EstablishConnectionWithConfig establishes a connection with custom configuration
func EstablishConnectionWithConfig(localEndpoint, remotePublicEndpoint *types.Endpoint, config ConnectionConfig) (*net.UDPConn, error) {
	// Prepare local endpoint - bind to specific port if provided, or random port
	var conn *net.UDPConn
	var err error

	if localEndpoint != nil && localEndpoint.Port != 0 {
		// Bind to specific local port
		addr := &net.UDPAddr{
			IP:   net.ParseIP(localEndpoint.IP),
			Port: localEndpoint.Port,
		}
		conn, err = net.ListenUDP("udp", addr)
	} else {
		// Bind to random port
		conn, _, err = PrepareLocalEndpoint()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to prepare local endpoint: %w", err)
	}

	// Set overall deadline for connection establishment
	deadline := time.Now().Add(config.Timeout)

	// Perform hole punching with retry logic
	fmt.Printf("Attempting hole punch to %s...\n", remotePublicEndpoint)

	// Try multiple times with exponential backoff
	for attempt := 1; attempt <= config.Attempts; attempt++ {
		if time.Now().After(deadline) {
			conn.Close()
			return nil, fmt.Errorf("connection timeout after %v", config.Timeout)
		}

		fmt.Printf(" Attempt %d/%d: Sending punch packets...\n", attempt, config.Attempts)

		// Perform simultaneous punch
		err = SimultaneousPunch(conn, remotePublicEndpoint, config.Attempts, config.Interval)
		if err == nil {
			fmt.Println(" Hole punched successfully!")
			return conn, nil
		}

		// If not the last attempt, wait before retry
		if attempt < config.Attempts {
			waitTime := config.Interval * time.Duration(attempt) // Exponential backoff
			fmt.Printf(" Failed: %v. Retrying in %v...\n", err, waitTime)
			time.Sleep(waitTime)
		}
	}

	conn.Close()
	return nil, fmt.Errorf("failed to establish connection after %d attempts: %w", config.Attempts, err)
}

// ValidateConnection tests if the connection is working by sending a test message
func ValidateConnection(conn *net.UDPConn, remoteEndpoint *types.Endpoint) error {
	// Send test message
	testMsg := "PING"
	if err := SendMessage(conn, testMsg, remoteEndpoint); err != nil {
		return fmt.Errorf("failed to send test message: %w", err)
	}

	// Wait for response
	response, _, err := ReceiveMessage(conn, 3*time.Second)
	if err != nil {
		return fmt.Errorf("failed to receive response: %w", err)
	}

	if response != "PONG" {
		return fmt.Errorf("unexpected response: %s (expected PONG)", response)
	}

	return nil
}

// PingPong performs a simple ping-pong exchange to verify connection
func PingPong(conn *net.UDPConn, remoteEndpoint *types.Endpoint, initiator bool) error {
	if initiator {
		// Send PING
		fmt.Println("\nSending PING...")
		if err := SendMessage(conn, "PING", remoteEndpoint); err != nil {
			return fmt.Errorf("failed to send PING: %w", err)
		}

		// Wait for PONG
		fmt.Println("Waiting for PONG...")
		response, _, err := ReceiveMessage(conn, 5*time.Second)
		if err != nil {
			return fmt.Errorf("failed to receive PONG: %w", err)
		}

		if response != "PONG" {
			return fmt.Errorf("unexpected response: %s", response)
		}

		fmt.Println(" Received PONG!")
		fmt.Println(" Round-trip successful!")
	} else {
		// Wait for PING
		fmt.Println("\nWaiting for PING...")
		message, sender, err := ReceiveMessage(conn, 10*time.Second)
		if err != nil {
			return fmt.Errorf("failed to receive PING: %w", err)
		}

		if message != "PING" {
			return fmt.Errorf("unexpected message: %s", message)
		}

		fmt.Printf(" Received PING from %s!\n", sender)

		// Send PONG back
		fmt.Println("Sending PONG...")
		if err := SendMessage(conn, "PONG", sender); err != nil {
			return fmt.Errorf("failed to send PONG: %w", err)
		}

		fmt.Println(" Sent PONG!")
		fmt.Println(" Round-trip successful!")
	}

	return nil
}
