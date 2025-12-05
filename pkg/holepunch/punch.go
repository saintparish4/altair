package holepunch

import (
	"fmt"
	"net"
	"time"

	"github.com/saintparish4/altair/pkg/types"
)

const (
	// PunchMessage is sent to punch through NAT
	PunchMessage = "PUNCH"

	// BufferSize for receiving UDP packets
	BufferSize = 1500
)

// PrepareLocalEndpoint binds to a random UDP port and returns the connection and local address
func PrepareLocalEndpoint() (*net.UDPConn, *types.Endpoint, error) {
	// Bind to random port (0 means 0S assigns random port)
	// Bind to 0.0.0.0 to accept connections on all interfaces
	addr := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 0,
	}

	conn, err := net.ListenUDP("udp4", addr) // Use udp4 to force IPv4
	if err != nil {
		return nil, nil, fmt.Errorf("failed to bind UDP socket: %w", err)
	}

	// Get the actual local address assigned
	localAddr := conn.LocalAddr().(*net.UDPAddr)

	// Use a proper IP for the endpoint
	ip := localAddr.IP.String()
	if ip == "0.0.0.0" {
		// For binding to all interfaces, report localhost for local connections
		ip = "127.0.0.1"
	}

	endpoint := &types.Endpoint{
		IP:   ip,
		Port: localAddr.Port,
	}

	return conn, endpoint, nil
}

// SendPunch sends a punch packet to the remote endpoint
func SendPunch(conn *net.UDPConn, remoteEndpoint *types.Endpoint) error {
	remoteAddr, err := net.ResolveUDPAddr("udp", remoteEndpoint.String())
	if err != nil {
		return fmt.Errorf("failed to resolve remote address: %w", err)
	}

	_, err = conn.WriteToUDP([]byte(PunchMessage), remoteAddr)
	if err != nil {
		return fmt.Errorf("failed to send punch: %w", err)
	}

	return nil
}

// ListenForPunch waits to receive a punch packet from any remote endpoint
// Returns the remote endpoint and the message received
func ListenForPunch(conn *net.UDPConn, timeout time.Duration) (*types.Endpoint, string, error) {
	buffer := make([]byte, BufferSize)

	// Set read deadline
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, "", fmt.Errorf("failed to set read deadline: %w", err)
	}

	n, remoteAddr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, "", fmt.Errorf("timeout waiting for punch: %w", err)
		}
		return nil, "", fmt.Errorf("failed to receieve punch: %w", err)
	}

	endpoint := &types.Endpoint{
		IP:   remoteAddr.IP.String(),
		Port: remoteAddr.Port,
	}

	message := string(buffer[:n])
	return endpoint, message, nil
}

// SimultaneousPunch performs simultaneous UDP hole punching
// Both peers send to each other while listening for incoming packets
func SimultaneousPunch(conn *net.UDPConn, remotePublicEP *types.Endpoint, attempts int, interval time.Duration) error {
	// Channel to signal success
	success := make(chan bool, 1)
	errChan := make(chan error, 1)

	// Start listening in background
	go func() {
		// Listen for any packet (first valid packet wins)
		_, message, err := ListenForPunch(conn, time.Duration(attempts)*interval+time.Second)
		if err != nil {
			errChan <- err
			return
		}

		// Check if it's a valid punch message
		if message == PunchMessage || message == "PING" || message == "PONG" {
			success <- true
		} else {
			errChan <- fmt.Errorf("unexpected message: %s", message)
		}
	}()

	// Send punch packets repeatedly
	go func() {
		for i := 0; i < attempts; i++ {
			if err := SendPunch(conn, remotePublicEP); err != nil {
				// Log but continue trying
				continue
			}
			time.Sleep(interval)
		}
	}()

	// Wait for success or error
	select {
	case <-success:
		return nil
	case err := <-errChan:
		return err
	case <-time.After(time.Duration(attempts)*interval + 2*time.Second):
		return fmt.Errorf("simultaneous punch timeout after %d attempts", attempts)
	}
}

// ReceiveMessage receives a message from the connection with timeout
func ReceiveMessage(conn *net.UDPConn, timeout time.Duration) (string, *types.Endpoint, error) {
	buffer := make([]byte, BufferSize)

	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return "", nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	n, remoteAddr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return "", nil, fmt.Errorf("timeout waiting for message: %w", err)
		}
		return "", nil, fmt.Errorf("failed to receive message: %w", err)
	}

	endpoint := &types.Endpoint{
		IP:   remoteAddr.IP.String(),
		Port: remoteAddr.Port,
	}

	return string(buffer[:n]), endpoint, nil
}

// SendMessage sends a message to a specific endpoint
func SendMessage(conn *net.UDPConn, message string, remoteEndpoint *types.Endpoint) error {
	remoteAddr, err := net.ResolveUDPAddr("udp", remoteEndpoint.String())
	if err != nil {
		return fmt.Errorf("failed to resolve remote address: %w", err)
	}

	_, err = conn.WriteToUDP([]byte(message), remoteAddr)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
