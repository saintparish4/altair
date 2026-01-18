package punch

import (
	"net"
	"testing"
	"time"

	"github.com/saintparish4/altair/pkg/nat"
)

func TestConnectionString(t *testing.T) {
	conn := &Connection{
		LocalAddr:  &net.UDPAddr{IP: net.ParseIP("192.168.1.1"), Port: 12345},
		RemoteAddr: &net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 54321},
		RTT:        50 * time.Millisecond,
		IsRelayed:  false,
	}

	result := conn.String()
	if result == "" {
		t.Error("Connection.String() should not be empty")
	}

	// Check relayed connection
	conn.IsRelayed = true
	result = conn.String()
	if result == "" {
		t.Error("Relayed connection.String() should not be empty")
	}
}

func TestConnectionClose(t *testing.T) {
	// Test closing with nil conn
	conn := &Connection{}
	err := conn.Close()
	if err != nil {
		t.Errorf("Close() with nil conn should not error: %v", err)
	}

	// Test closing with real conn
	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		t.Fatalf("Failed to create UDP connection: %v", err)
	}

	conn = &Connection{Conn: udpConn}
	err = conn.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

func TestDefaultPuncherConfig(t *testing.T) {
	config := DefaultPuncherConfig()

	if config.Timeout == 0 {
		t.Error("DefaultPuncherConfig should set Timeout")
	}

	if config.PingInterval == 0 {
		t.Error("DefaultPuncherConfig should set PingInterval")
	}

	if config.MaxAttempts == 0 {
		t.Error("DefaultPuncherConfig should set MaxAttempts")
	}

	// Verify reasonable values
	if config.Timeout < 1*time.Second {
		t.Error("Default timeout should be at least 1 second")
	}

	if config.PingInterval > 1*time.Second {
		t.Error("Default ping interval should be less than 1 second")
	}

	if config.MaxAttempts < 10 {
		t.Error("Default max attempts should be at least 10")
	}
}

func TestNewPuncherNilConfig(t *testing.T) {
	// Should use default config when nil is passed
	puncher, err := NewPuncher(nil)
	if err != nil {
		t.Fatalf("NewPuncher(nil) failed: %v", err)
	}
	defer puncher.Close()

	if puncher.timeout == 0 {
		t.Error("Puncher should have non-zero timeout from default config")
	}

	if puncher.conn == nil {
		t.Error("Puncher should have created a connection")
	}
}

func TestNewPuncherWithLocalAddr(t *testing.T) {
	config := &PuncherConfig{
		LocalAddr: &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0},
		Timeout:   10 * time.Second,
	}

	puncher, err := NewPuncher(config)
	if err != nil {
		t.Fatalf("NewPuncher failed: %v", err)
	}
	defer puncher.Close()

	localAddr := puncher.LocalAddr()
	if !localAddr.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Errorf("LocalAddr IP = %s, want 127.0.0.1", localAddr.IP)
	}
}

func TestNewPuncherWithExistingConn(t *testing.T) {
	// Create a connection
	existingConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		t.Fatalf("Failed to create UDP connection: %v", err)
	}
	defer existingConn.Close()

	config := &PuncherConfig{
		Conn:    existingConn,
		Timeout: 10 * time.Second,
	}

	puncher, err := NewPuncher(config)
	if err != nil {
		t.Fatalf("NewPuncher failed: %v", err)
	}
	// Don't close puncher - it would close the existing conn

	if puncher.conn != existingConn {
		t.Error("Puncher should use the provided connection")
	}
}

func TestPunchHoleNilPeer(t *testing.T) {
	puncher, err := NewPuncher(nil)
	if err != nil {
		t.Fatalf("NewPuncher failed: %v", err)
	}
	defer puncher.Close()

	_, err = puncher.PunchHole(nil)
	if err == nil {
		t.Error("PunchHole should fail with nil peer")
	}
}

func TestPunchHoleNilPublicAddr(t *testing.T) {
	puncher, err := NewPuncher(nil)
	if err != nil {
		t.Fatalf("NewPuncher failed: %v", err)
	}
	defer puncher.Close()

	peer := &PeerInfo{
		PublicAddr: nil,
		NATType:    nat.TypeFullCone,
	}

	_, err = puncher.PunchHole(peer)
	if err == nil {
		t.Error("PunchHole should fail with nil public address")
	}
}

func TestPunchHoleIncompatibleNAT(t *testing.T) {
	mapping := &nat.Mapping{
		Type: nat.TypeSymmetric,
	}

	config := &PuncherConfig{
		Mapping: mapping,
		Timeout: 1 * time.Second,
	}

	puncher, err := NewPuncher(config)
	if err != nil {
		t.Fatalf("NewPuncher failed: %v", err)
	}
	defer puncher.Close()

	peer := &PeerInfo{
		PublicAddr: &net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 12345},
		NATType:    nat.TypeSymmetric,
	}

	_, err = puncher.PunchHole(peer)
	if err == nil {
		t.Error("PunchHole should fail for incompatible NAT types")
	}
}

func TestPuncherClose(t *testing.T) {
	puncher, err := NewPuncher(nil)
	if err != nil {
		t.Fatalf("NewPuncher failed: %v", err)
	}

	err = puncher.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Closing again should not panic
	err = puncher.Close()
	if err != nil {
		t.Logf("Second Close() returned error (acceptable): %v", err)
	}
}

func TestPuncherLocalAddr(t *testing.T) {
	puncher, err := NewPuncher(nil)
	if err != nil {
		t.Fatalf("NewPuncher failed: %v", err)
	}
	defer puncher.Close()

	localAddr := puncher.LocalAddr()
	if localAddr == nil {
		t.Fatal("LocalAddr() should not return nil")
	}

	if localAddr.Port == 0 {
		t.Error("LocalAddr() should have non-zero port")
	}
}

func TestPuncherMapping(t *testing.T) {
	mapping := &nat.Mapping{
		Type: nat.TypeFullCone,
	}

	config := &PuncherConfig{
		Mapping: mapping,
		Timeout: 10 * time.Second,
	}

	puncher, err := NewPuncher(config)
	if err != nil {
		t.Fatalf("NewPuncher failed: %v", err)
	}
	defer puncher.Close()

	result := puncher.Mapping()
	if result != mapping {
		t.Error("Mapping() should return the configured mapping")
	}
}

func TestPuncherConcurrentAccess(t *testing.T) {
	puncher, err := NewPuncher(nil)
	if err != nil {
		t.Fatalf("NewPuncher failed: %v", err)
	}
	defer puncher.Close()

	// Try concurrent PunchHole calls (should be safe with mutex)
	done := make(chan bool, 2)

	peer := &PeerInfo{
		PublicAddr: &net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 12345},
		NATType:    nat.TypeFullCone,
	}

	// Modify timeout to fail quickly
	puncher.timeout = 100 * time.Millisecond

	go func() {
		puncher.PunchHole(peer)
		done <- true
	}()

	go func() {
		puncher.PunchHole(peer)
		done <- true
	}()

	// Wait for both to complete
	<-done
	<-done
}

func TestPeerInfoValidation(t *testing.T) {
	// Test with valid peer info
	peer := &PeerInfo{
		PublicAddr: &net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 12345},
		LocalAddrs: []*net.UDPAddr{
			{IP: net.ParseIP("192.168.1.1"), Port: 12345},
		},
		NATType: nat.TypeFullCone,
	}

	if peer.PublicAddr == nil {
		t.Error("Valid peer should have public address")
	}

	if len(peer.LocalAddrs) == 0 {
		t.Error("Peer should have local addresses")
	}
}

func BenchmarkNewPuncher(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		puncher, err := NewPuncher(nil)
		if err != nil {
			b.Fatal(err)
		}
		puncher.Close()
	}
}

func BenchmarkPuncherLocalAddr(b *testing.B) {
	puncher, err := NewPuncher(nil)
	if err != nil {
		b.Fatal(err)
	}
	defer puncher.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = puncher.LocalAddr()
	}
}

// TODO: Integration test note:
// Full hole punching tests require two network endpoints.
// These tests cover the API and validation logic.
// For real hole punching tests, see test/integration/punch_test.go
