package holepunch

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/saintparish4/altair/pkg/types"
)

func TestPrepareLocalEndpoint(t *testing.T) {
	conn, endpoint, err := PrepareLocalEndpoint()
	if err != nil {
		t.Fatalf("PrepareLocalEndpoint failed: %v", err)
	}
	// Verify connection is not nil
	if conn == nil {
		t.Fatal("connection is nil")
	}
	defer conn.Close()

	// Verify endpoint has valid port
	if endpoint.Port == 0 {
		t.Error("endpoint port is 0, expected random port assignment")
	}

	// Verify endpoint has IP
	if endpoint.IP == "" {
		t.Error("endpoint IP is empty")
	}

	t.Logf("Local endpoint: %s", endpoint)
}

func TestSendAndReceiveMessage(t *testing.T) {
	// Setup two UDP connections for testing
	conn1, endpoint1, err := PrepareLocalEndpoint()
	if err != nil {
		t.Fatalf("Failed to prepare endpoint 1: %v", err)
	}
	defer conn1.Close()

	conn2, endpoint2, err := PrepareLocalEndpoint()
	if err != nil {
		t.Fatalf("Failed to prepare endpoint 2: %v", err)
	}
	defer conn2.Close()

	// Test message
	testMsg := "HELLO"

	// Send message from conn1 to conn2
	err = SendMessage(conn1, testMsg, endpoint2)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Receive message on conn2
	receivedMsg, sender, err := ReceiveMessage(conn2, 1*time.Second)
	if err != nil {
		t.Fatalf("Failed to receive message: %v", err)
	}

	// Verify message
	if receivedMsg != testMsg {
		t.Errorf("Expected message %q, got %q", testMsg, receivedMsg)
	}

	// Verify sender port matches (IP might be 127.0.0.1 or [::1])
	if sender.Port != endpoint1.Port {
		t.Errorf("Expected sender port %d, got %d", endpoint1.Port, sender.Port)
	}

	t.Logf("Successfully sent and received message: %q", testMsg)
}

func TestPunchMessage(t *testing.T) {
	// Verify punch message constant
	if PunchMessage != "PUNCH" {
		t.Errorf("Expected PunchMessage to be 'PUNCH', got %q", PunchMessage)
	}
}

func TestSendPunch(t *testing.T) {
	conn, _, err := PrepareLocalEndpoint()
	if err != nil {
		t.Fatalf("Failed to prepare endpoint: %v", err)
	}
	defer conn.Close()

	// Create a target endpoint (doesn't need to be reachable for this test)
	target := &types.Endpoint{
		IP:   "127.0.0.1",
		Port: 9999,
	}

	// Send punch packet (won't error even if unreachable)
	err = SendPunch(conn, target)
	if err != nil {
		t.Fatalf("SendPunch failed: %v", err)
	}

	t.Log("Successfully sent punch packet")
}

func TestReceiveMessageTimeout(t *testing.T) {
	conn, _, err := PrepareLocalEndpoint()
	if err != nil {
		t.Fatalf("Failed to prepare endpoint: %v", err)
	}
	defer conn.Close()

	// Try to receive with short timeout (should timeout)
	_, _, err = ReceiveMessage(conn, 100*time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// Verify it's a timeout error
	if netErr, ok := err.(net.Error); ok {
		if !netErr.Timeout() {
			t.Error("Expected timeout error flag to be true")
		}
	}
}

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectError  bool
		expectedIP   string
		expectedPort int
	}{
		{
			name:         "valid IPv4 endpoint",
			input:        "192.168.1.1:5000",
			expectError:  false,
			expectedIP:   "192.168.1.1",
			expectedPort: 5000,
		},
		{
			name:        "missing port",
			input:       "192.168.1.1",
			expectError: true,
		},
		{
			name:        "invalid port",
			input:       "192.168.1.1:abc",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse endpoint using connect.go's parseEndpoint logic
			endpoint, err := parseEndpointHelper(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if endpoint.IP != tt.expectedIP {
					t.Errorf("Expected IP %q, got %q", tt.expectedIP, endpoint.IP)
				}
				if endpoint.Port != tt.expectedPort {
					t.Errorf("Expected port %d, got %d", tt.expectedPort, endpoint.Port)
				}
			}
		})
	}
}

// Helper function that mimics parseEndpoint from connect.go
func parseEndpointHelper(addr string) (*types.Endpoint, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("address must be in format IP:PORT")
	}

	var port int
	_, err := fmt.Sscanf(parts[1], "%d", &port)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	return &types.Endpoint{
		IP:   parts[0],
		Port: port,
	}, nil
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Attempts != DefaultAttempts {
		t.Errorf("Expected %d attempts, got %d", DefaultAttempts, config.Attempts)
	}

	if config.Interval != DefaultInterval {
		t.Errorf("Expected interval %v, got %v", DefaultInterval, config.Interval)
	}

	if config.Timeout != DefaultTimeout {
		t.Errorf("Expected timeout %v, got %v", DefaultTimeout, config.Timeout)
	}
}

func TestPingPongSimulation(t *testing.T) {
	// Create two UDP connections
	conn1, _, err := PrepareLocalEndpoint()
	if err != nil {
		t.Fatalf("Failed to prepare endpoint 1: %v", err)
	}
	defer conn1.Close()

	conn2, endpoint2, err := PrepareLocalEndpoint()
	if err != nil {
		t.Fatalf("Failed to prepare endpoint 2: %v", err)
	}
	defer conn2.Close()

	// Channel for synchronization
	done := make(chan error, 2)

	// Peer 1: Send PING, wait for PONG
	go func() {
		// Send PING
		if err := SendMessage(conn1, "PING", endpoint2); err != nil {
			done <- err
			return
		}

		// Wait for PONG
		msg, _, err := ReceiveMessage(conn1, 2*time.Second)
		if err != nil {
			done <- err
			return
		}

		if msg != "PONG" {
			done <- fmt.Errorf("expected PONG, got %s", msg)
			return
		}

		done <- nil
	}()

	// Peer 2: Wait for PING, send PONG
	go func() {
		// Wait for PING
		msg, sender, err := ReceiveMessage(conn2, 2*time.Second)
		if err != nil {
			done <- err
			return
		}

		if msg != "PING" {
			done <- fmt.Errorf("expected PING, got %s", msg)
			return
		}

		// Send PONG back
		if err := SendMessage(conn2, "PONG", sender); err != nil {
			done <- err
			return
		}

		done <- nil
	}()

	// Wait for both to complete
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatalf("Ping-pong failed: %v", err)
		}
	}

	t.Log("Ping-pong successful!")
}
