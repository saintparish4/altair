package relay

import (
	"net"
	"testing"
	"time"
)

func TestAllocationString(t *testing.T) {
	// Test nil allocation
	var alloc *Allocation
	result := alloc.String()
	if result != "<nil allocation>" {
		t.Errorf("nil allocation String() = %q, want \"<nil allocation>\"", result)
	}

	// Test valid allocation
	alloc = &Allocation{
		RelayAddr: &net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 3478},
		ExpiresAt: time.Now().Add(5 * time.Minute),
		ID:        "test-alloc",
	}

	result = alloc.String()
	if result == "" {
		t.Error("Allocation.String() should not be empty")
	}
}

func TestAllocationIsValid(t *testing.T) {
	// Test nil allocation
	var alloc *Allocation
	if alloc.IsValid() {
		t.Error("nil allocation should not be valid")
	}

	// Test expired allocation
	alloc = &Allocation{
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	}
	if alloc.IsValid() {
		t.Error("expired allocation should not be valid")
	}

	// Test valid allocation
	alloc = &Allocation{
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	if !alloc.IsValid() {
		t.Error("future expiration should be valid")
	}
}

func TestAllocationTimeRemaining(t *testing.T) {
	// Test expired allocation
	alloc := &Allocation{
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	}
	remaining := alloc.TimeRemaining()
	if remaining != 0 {
		t.Errorf("expired allocation TimeRemaining() = %v, want 0", remaining)
	}

	// Test valid allocation
	alloc = &Allocation{
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	remaining = alloc.TimeRemaining()
	if remaining <= 0 {
		t.Error("valid allocation should have positive time remaining")
	}
	if remaining > 5*time.Minute {
		t.Errorf("time remaining %v should be less than 5 minutes", remaining)
	}
}

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig("relay.example.com:3478")

	if config.ServerAddr == "" {
		t.Error("DefaultClientConfig should set ServerAddr")
	}

	if config.Lifetime == 0 {
		t.Error("DefaultClientConfig should set Lifetime")
	}

	if config.Timeout == 0 {
		t.Error("DefaultClientConfig should set Timeout")
	}

	// Verify reasonable values
	if config.Lifetime < 1*time.Minute {
		t.Error("Default lifetime should be at least 1 minute")
	}

	if config.Timeout < 1*time.Second {
		t.Error("Default timeout should be at least 1 second")
	}
}

func TestNewClientNilConfig(t *testing.T) {
	_, err := NewClient(nil)
	if err == nil {
		t.Error("NewClient should fail with nil config")
	}
}

func TestNewClientInvalidServer(t *testing.T) {
	config := &ClientConfig{
		ServerAddr: "invalid::server::address",
		Timeout:    5 * time.Second,
	}

	_, err := NewClient(config)
	if err == nil {
		t.Error("NewClient should fail with invalid server address")
	}
}

func TestNewClientSuccess(t *testing.T) {
	config := DefaultClientConfig("127.0.0.1:3478")

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	if client.conn == nil {
		t.Error("Client should have a connection")
	}

	if client.serverAddr == nil {
		t.Error("Client should have server address")
	}
}

func TestNewClientWithExistingConn(t *testing.T) {
	// Create a connection
	existingConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		t.Fatalf("Failed to create UDP connection: %v", err)
	}
	defer existingConn.Close()

	config := &ClientConfig{
		ServerAddr: "127.0.0.1:3478",
		Conn:       existingConn,
		Timeout:    5 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	// Don't close client - it would close the existing conn

	if client.conn != existingConn {
		t.Error("Client should use the provided connection")
	}
}

func TestAllocate(t *testing.T) {
	config := DefaultClientConfig("127.0.0.1:3478")

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Allocate
	lifetime := 10 * time.Minute
	allocation, err := client.Allocate(lifetime)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}

	if allocation == nil {
		t.Fatal("Allocation should not be nil")
	}

	if allocation.RelayAddr == nil {
		t.Error("Allocation should have relay address")
	}

	if allocation.ID == "" {
		t.Error("Allocation should have ID")
	}

	if !allocation.IsValid() {
		t.Error("New allocation should be valid")
	}

	// Verify client stored the allocation
	if client.Allocation() != allocation {
		t.Error("Client should store the allocation")
	}
}

func TestAllocateWhenClosed(t *testing.T) {
	config := DefaultClientConfig("127.0.0.1:3478")

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	client.Close()

	_, err = client.Allocate(10 * time.Minute)
	if err == nil {
		t.Error("Allocate should fail when client is closed")
	}
}

func TestRefresh(t *testing.T) {
	config := DefaultClientConfig("127.0.0.1:3478")

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Try refresh without allocation
	err = client.Refresh(5 * time.Minute)
	if err == nil {
		t.Error("Refresh should fail without allocation")
	}

	// Create allocation
	allocation, err := client.Allocate(1 * time.Minute)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}

	oldExpires := allocation.ExpiresAt

	// Refresh
	time.Sleep(10 * time.Millisecond) // Ensure time passes
	err = client.Refresh(10 * time.Minute)
	if err != nil {
		t.Errorf("Refresh failed: %v", err)
	}

	// Verify expiration was extended
	if !allocation.ExpiresAt.After(oldExpires) {
		t.Error("Refresh should extend expiration time")
	}
}

func TestRefreshExpiredAllocation(t *testing.T) {
	config := DefaultClientConfig("127.0.0.1:3478")

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Create allocation and expire it manually
	client.allocation = &Allocation{
		ExpiresAt: time.Now().Add(-1 * time.Minute),
		ID:        "expired",
	}

	err = client.Refresh(5 * time.Minute)
	if err == nil {
		t.Error("Refresh should fail for expired allocation")
	}
}

func TestSend(t *testing.T) {
	config := DefaultClientConfig("127.0.0.1:3478")

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	peer := &net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 12345}

	// Try send without allocation
	err = client.Send([]byte("test"), peer)
	if err == nil {
		t.Error("Send should fail without allocation")
	}

	// Create allocation
	_, err = client.Allocate(10 * time.Minute)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}

	// Now send should work (though no peer is listening)
	err = client.Send([]byte("test"), peer)
	// We expect this to succeed even though peer isn't listening
	if err != nil {
		t.Errorf("Send failed: %v", err)
	}
}

func TestSendExpiredAllocation(t *testing.T) {
	config := DefaultClientConfig("127.0.0.1:3478")

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Create expired allocation
	client.allocation = &Allocation{
		ExpiresAt: time.Now().Add(-1 * time.Minute),
		ID:        "expired",
	}

	peer := &net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 12345}
	err = client.Send([]byte("test"), peer)
	if err == nil {
		t.Error("Send should fail with expired allocation")
	}
}

func TestCreatePermission(t *testing.T) {
	config := DefaultClientConfig("127.0.0.1:3478")

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	peer := &net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 12345}

	// Try without allocation
	err = client.CreatePermission(peer)
	if err == nil {
		t.Error("CreatePermission should fail without allocation")
	}

	// Create allocation
	_, err = client.Allocate(10 * time.Minute)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}

	// Now it should work
	err = client.CreatePermission(peer)
	if err != nil {
		t.Errorf("CreatePermission failed: %v", err)
	}
}

func TestClose(t *testing.T) {
	config := DefaultClientConfig("127.0.0.1:3478")

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Create allocation
	_, err = client.Allocate(10 * time.Minute)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify allocation is cleared
	if client.Allocation() != nil {
		t.Error("Close should clear allocation")
	}

	// Closing again should not error
	err = client.Close()
	if err != nil {
		t.Errorf("Second Close failed: %v", err)
	}
}

func TestLocalAddr(t *testing.T) {
	config := DefaultClientConfig("127.0.0.1:3478")

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	localAddr := client.LocalAddr()
	if localAddr == nil {
		t.Fatal("LocalAddr should not be nil")
	}

	if localAddr.Port == 0 {
		t.Error("LocalAddr should have non-zero port")
	}
}

func TestServerAddr(t *testing.T) {
	config := DefaultClientConfig("127.0.0.1:3478")

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	serverAddr := client.ServerAddr()
	if serverAddr == nil {
		t.Fatal("ServerAddr should not be nil")
	}

	if !serverAddr.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Errorf("ServerAddr IP = %s, want 127.0.0.1", serverAddr.IP)
	}

	if serverAddr.Port != 3478 {
		t.Errorf("ServerAddr Port = %d, want 3478", serverAddr.Port)
	}
}

func TestQuickRelay(t *testing.T) {
	client, allocation, err := QuickRelay("127.0.0.1:3478", 10*time.Minute)
	if err != nil {
		t.Fatalf("QuickRelay failed: %v", err)
	}
	defer client.Close()

	if client == nil {
		t.Fatal("QuickRelay should return client")
	}

	if allocation == nil {
		t.Fatal("QuickRelay should return allocation")
	}

	if !allocation.IsValid() {
		t.Error("QuickRelay allocation should be valid")
	}
}

func BenchmarkNewClient(b *testing.B) {
	config := DefaultClientConfig("127.0.0.1:3478")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client, err := NewClient(config)
		if err != nil {
			b.Fatal(err)
		}
		client.Close()
	}
}

func BenchmarkAllocate(b *testing.B) {
	config := DefaultClientConfig("127.0.0.1:3478")
	client, err := NewClient(config)
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client.Allocate(10 * time.Minute)
	}
}

func BenchmarkAllocationIsValid(b *testing.B) {
	alloc := &Allocation{
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = alloc.IsValid()
	}
}

// TODO: Integration test note:
// Full relay tests require a running TURN server.
// These tests cover the client API and logic.
// For real relay tests, see test/integration/relay_test.go
