// Copyright (c) 2025
// SPDX-License-Identifier: MIT

package altair

import (
	"fmt"
	"net"
	"time"

	"github.com/saintparish4/altair/pkg/holepunch"
	"github.com/saintparish4/altair/pkg/stun"
	"github.com/saintparish4/altair/pkg/types"
)

const (
	// DefaultSTUNServer is the default STUN server for endpoint discovery
	DefaultSTUNServer = "stun.l.google.com:19302"

	// Version is the library version
	Version = "1.0.0"
)

// Client represents an Altair P2P client
type Client struct {
	stunServer string
	stunClient *stun.Client
	config     Config
}

// Config holds configuration for the Altair client
type Config struct {
	// STUNServer is the STUN server address for endpoint discovery
	STUNServer string

	// ConnectionAttempts is the number of hole punching attempts
	ConnectionAttempts int

	// ConnectionInterval is the interval between hole punching attempts
	ConnectionInterval time.Duration

	// ConnectionTimeout is the overall timeout for connection establishment
	ConnectionTimeout time.Duration

	// STUNTimeout is the timeout for STUN discovery requests
	STUNTimeout time.Duration
}

// DefaultConfig returns the default client configuration
func DefaultConfig() Config {
	return Config{
		STUNServer:         DefaultSTUNServer,
		ConnectionAttempts: 5,
		ConnectionInterval: 400 * time.Millisecond,
		ConnectionTimeout:  10 * time.Second,
		STUNTimeout:        5 * time.Second,
	}
}

// NewClient creates a new Altair client with default configuration
func NewClient() *Client {
	return NewClientWithConfig(DefaultConfig())
}

// NewClientWithConfig creates a new Altair client with custom configuration
func NewClientWithConfig(config Config) *Client {
	stunClient := stun.NewClient(config.STUNServer)
	stunClient.Timeout = config.STUNTimeout

	return &Client{
		stunServer: config.STUNServer,
		stunClient: stunClient,
		config:     config,
	}
}

// DiscoverPublicEndpoint discovers the public IP and port using STUN
//
// This method contacts a STUN server to discover the client's public-facing
// IP address and port after NAT translation.
//
// Returns:
//   - *types.Endpoint: The discovered public endpoint
//   - error: Any error encountered during discovery
//
// Example:
//
//	client := altair.NewClient()
//	endpoint, err := client.DiscoverPublicEndpoint()
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Public endpoint: %s:%d\n", endpoint.IP, endpoint.Port)
func (c *Client) DiscoverPublicEndpoint() (*types.Endpoint, error) {
	endpoint, err := c.stunClient.Discover()
	if err != nil {
		return nil, fmt.Errorf("failed to discover public endpoint: %w", err)
	}
	return endpoint, nil
}

// Connect establishes a direct P2P connection to a remote peer
//
// This method performs UDP hole punching to establish a direct connection
// through NAT. Both peers must call this method with each other's public
// endpoints (obtained through DiscoverPublicEndpoint).
//
// Parameters:
//   - remoteEndpoint: The remote peer's public endpoint
//   - initiator: Whether this peer should initiate the ping-pong validation
//
// Returns:
//   - *net.UDPConn: An established UDP connection ready for data exchange
//   - error: Any error encountered during connection establishment
//
// Example:
//
//	// Exchange endpoints through signaling server (not shown)
//	// remoteEndpoint := ... (obtained from peer)
//
//	// Establish connection
//	conn, err := client.Connect(remoteEndpoint, true)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer conn.Close()
//
//	// Send data
//	_, err = conn.Write([]byte("Hello, peer!"))
func (c *Client) Connect(remoteEndpoint *types.Endpoint, initiator bool) (*net.UDPConn, error) {
	// Prepare local connection
	conn, localEndpoint, err := holepunch.PrepareLocalEndpoint()
	if err != nil {
		return nil, fmt.Errorf("failed to prepare local endpoint: %w", err)
	}

	// Create connection config from client config
	config := holepunch.ConnectionConfig{
		Attempts: c.config.ConnectionAttempts,
		Interval: c.config.ConnectionInterval,
		Timeout:  c.config.ConnectionTimeout,
	}

	// Perform hole punching
	err = holepunch.SimultaneousPunch(conn, remoteEndpoint, config.Attempts, config.Interval)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to punch through NAT: %w", err)
	}

	// Validate connection with ping-pong
	err = holepunch.PingPong(conn, remoteEndpoint, initiator)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to validate connection: %w", err)
	}

	fmt.Printf("✓ P2P connection established: %s <-> %s\n", localEndpoint, remoteEndpoint)
	return conn, nil
}

// ConnectWithLocalEndpoint establishes a P2P connection using a specific local endpoint
//
// This is useful when you want to reuse a specific local port or have already
// discovered your public endpoint.
//
// Parameters:
//   - localEndpoint: The local endpoint to bind to
//   - remoteEndpoint: The remote peer's public endpoint
//   - initiator: Whether this peer should initiate the ping-pong validation
//
// Returns:
//   - *net.UDPConn: An established UDP connection
//   - error: Any error encountered
func (c *Client) ConnectWithLocalEndpoint(localEndpoint, remoteEndpoint *types.Endpoint, initiator bool) (*net.UDPConn, error) {
	config := holepunch.ConnectionConfig{
		Attempts: c.config.ConnectionAttempts,
		Interval: c.config.ConnectionInterval,
		Timeout:  c.config.ConnectionTimeout,
	}

	conn, err := holepunch.EstablishConnectionWithConfig(localEndpoint, remoteEndpoint, config)
	if err != nil {
		return nil, fmt.Errorf("failed to establish connection: %w", err)
	}

	// Validate connection
	err = holepunch.PingPong(conn, remoteEndpoint, initiator)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to validate connection: %w", err)
	}

	return conn, nil
}

// SendMessage sends a message over an established connection
//
// This is a convenience method for sending string messages.
//
// Parameters:
//   - conn: An established UDP connection
//   - message: The message to send
//   - remoteEndpoint: The remote peer's endpoint
//
// Returns:
//   - error: Any error encountered during sending
func (c *Client) SendMessage(conn *net.UDPConn, message string, remoteEndpoint *types.Endpoint) error {
	return holepunch.SendMessage(conn, message, remoteEndpoint)
}

// ReceiveMessage receives a message from an established connection
//
// This is a convenience method for receiving string messages with a timeout.
//
// Parameters:
//   - conn: An established UDP connection
//   - timeout: Maximum time to wait for a message
//
// Returns:
//   - string: The received message
//   - *types.Endpoint: The sender's endpoint
//   - error: Any error encountered during receiving
func (c *Client) ReceiveMessage(conn *net.UDPConn, timeout time.Duration) (string, *types.Endpoint, error) {
	return holepunch.ReceiveMessage(conn, timeout)
}

// GetVersion returns the library version
func GetVersion() string {
	return Version
}
