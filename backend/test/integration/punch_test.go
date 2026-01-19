// +build integration

package integration

import (
	"net"
	"sync"
	"testing"	
	"time"

	"github.com/saintparish4/altair/pkg/nat"
	"github.com/saintparish4/altair/pkg/netutil"
	"github.com/saintparish4/altair/pkg/punch"
)

// TestLocalPeerConnection tests P2P connection on localhost (simulates LAN)
func TestLocalPeerConnection(t *testing.T) {
	// Create two punchers on localhost
	puncher1, err := punch.NewPuncher(&punch.PuncherConfig{
		Timeout:      5 * time.Second,
		PingInterval: 100 * time.Millisecond,
		MaxAttempts:  20,
	})
	if err != nil {
		t.Fatalf("Failed to create puncher1: %v", err)
	}
	defer puncher1.Close()

	puncher2, err := punch.NewPuncher(&punch.PuncherConfig{
		Timeout:      5 * time.Second,
		PingInterval: 100 * time.Millisecond,
		MaxAttempts:  20,
	})
	if err != nil {
		t.Fatalf("Failed to create puncher2: %v", err)
	}
	defer puncher2.Close()

	t.Logf("Puncher1 local: %s", puncher1.LocalAddr())
	t.Logf("Puncher2 local: %s", puncher2.LocalAddr())

	// Prepare peer info (simulating localhost connection)
	peer1Info := &punch.PeerInfo{
		PublicAddr: puncher1.LocalAddr(),
		LocalAddrs: []*net.UDPAddr{puncher1.LocalAddr()},
		NATType:    nat.TypeOpenInternet,
	}

	peer2Info := &punch.PeerInfo{
		PublicAddr: puncher2.LocalAddr(),
		LocalAddrs: []*net.UDPAddr{puncher2.LocalAddr()},
		NATType:    nat.TypeOpenInternet,
	}

	// Simultaneous connection attempt
	var wg sync.WaitGroup
	wg.Add(2)

	var conn1 *punch.Connection
	var conn2 *punch.Connection
	var err1, err2 error

	// Peer 1 tries to connect to Peer 2
	go func() {
		defer wg.Done()
		conn1, err1 = puncher1.PunchHole(peer2Info)
	}()

	// Peer 2 tries to connect to Peer 1
	go func() {
		defer wg.Done()
		conn2, err2 = puncher2.PunchHole(peer1Info)
	}()

	wg.Wait()

	// Check results
	if err1 != nil {
		t.Errorf("Peer 1 connection failed: %v", err1)
	}
	if err2 != nil {
		t.Errorf("Peer 2 connection failed: %v", err2)
	}

	if conn1 != nil {
		t.Logf("Peer 1 connected: %s (RTT: %v)", conn1, conn1.RTT)
	}
	if conn2 != nil {
		t.Logf("Peer 2 connected: %s (RTT: %v)", conn2, conn2.RTT)
	}
}

// TestPunchWithRealNAT tests hole punching with actual NAT detection
func TestPunchWithRealNAT(t *testing.T) {
	// Detect NAT type first
	mapping, err := nat.QuickDetect()
	if err != nil {
		t.Fatalf("NAT detection failed: %v", err)
	}

	t.Logf("Detected NAT: %s", mapping.Type)
	t.Logf("Public address: %s", mapping.PublicAddr)

	// Create puncher with mapping
	puncher, err := punch.NewPuncher(&punch.PuncherConfig{
		Mapping:      mapping,
		Timeout:      10 * time.Second,
		PingInterval: 200 * time.Millisecond,
		MaxAttempts:  30,
	})
	if err != nil {
		t.Fatalf("Failed to create puncher: %v", err)
	}
	defer puncher.Close()

	// For this test, we'll just verify the puncher is set up correctly
	// Real peer-to-peer would require coordination with another instance
	t.Logf("Puncher ready: local=%s, public=%s",
		puncher.LocalAddr(), mapping.PublicAddr)

	if puncher.Mapping() == nil {
		t.Error("Puncher should have mapping")
	}
}

// TestLANDetection tests detection of local network peers
func TestLANDetection(t *testing.T) {
	// Get local network addresses
	localAddrs, err := netutil.GetPrivateAddresses()
	if err != nil {
		t.Fatalf("Failed to get local addresses: %v", err)
	}

	if len(localAddrs) == 0 {
		t.Skip("No local addresses available")
	}

	t.Logf("Found %d local addresses", len(localAddrs))
	for i, addr := range localAddrs {
		t.Logf("  [%d] %s", i, addr)
	}

	// Verify they're all private
	for _, addr := range localAddrs {
		if !netutil.IsPrivateIP(addr) {
			t.Errorf("Address %s should be private", addr)
		}
	}
}

// TestPunchTimeout tests that hole punching times out appropriately
func TestPunchTimeout(t *testing.T) {
	puncher, err := punch.NewPuncher(&punch.PuncherConfig{
		Timeout:      1 * time.Second, // Short timeout
		PingInterval: 100 * time.Millisecond,
		MaxAttempts:  5,
	})
	if err != nil {
		t.Fatalf("Failed to create puncher: %v", err)
	}
	defer puncher.Close()

	// Try to connect to a non-existent peer
	peer := &punch.PeerInfo{
		PublicAddr: &net.UDPAddr{
			IP:   net.ParseIP("203.0.113.1"), // TEST-NET-3 (should not respond)
			Port: 12345,
		},
		NATType: nat.TypeFullCone,
	}

	start := time.Now()
	conn, err := puncher.PunchHole(peer)
	duration := time.Since(start)

	if err == nil {
		conn.Close()
		t.Error("Expected timeout error, got success")
	}

	t.Logf("Timed out after %v (expected ~1s)", duration)

	// Should timeout around 1 second
	if duration < 500*time.Millisecond || duration > 2*time.Second {
		t.Errorf("Timeout duration %v unexpected (wanted ~1s)", duration)
	}
}

// TestPunchWithRetry tests the retry mechanism
func TestPunchWithRetry(t *testing.T) {
	puncher, err := punch.NewPuncher(&punch.PuncherConfig{
		Timeout:      500 * time.Millisecond, // Very short
		PingInterval: 50 * time.Millisecond,
		MaxAttempts:  3,
	})
	if err != nil {
		t.Fatalf("Failed to create puncher: %v", err)
	}
	defer puncher.Close()

	peer := &punch.PeerInfo{
		PublicAddr: &net.UDPAddr{
			IP:   net.ParseIP("203.0.113.1"),
			Port: 12345,
		},
		NATType: nat.TypeFullCone,
	}

	start := time.Now()
	_, err = puncher.PunchWithRetry(peer, 2) // 2 retries
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected error with non-existent peer")
	}

	t.Logf("Retry failed after %v", duration)

	// With 2 retries, should take roughly:
	// Initial attempt + 1s backoff + attempt + 2s backoff + attempt
	// = 3 attempts + 3s backoff + 3*0.5s timeouts = ~4.5s minimum
	if duration < 1*time.Second {
		t.Errorf("Retry duration %v too short", duration)
	}
}

// TestNATCompatibilityChecking tests the compatibility matrix
func TestNATCompatibilityChecking(t *testing.T) {
	tests := []struct {
		type1    nat.Type
		type2    nat.Type
		expected bool
	}{
		{nat.TypeOpenInternet, nat.TypeFullCone, true},
		{nat.TypeFullCone, nat.TypeRestrictedCone, true},
		{nat.TypeSymmetric, nat.TypeSymmetric, false},
		{nat.TypeRestrictedCone, nat.TypeRestrictedCone, true},
		{nat.TypeBlocked, nat.TypeFullCone, false},
	}

	for _, tt := range tests {
		result := nat.CanHolePunch(tt.type1, tt.type2)
		if result != tt.expected {
			t.Errorf("CanHolePunch(%s, %s) = %v, want %v",
				tt.type1, tt.type2, result, tt.expected)
		}
	}
}

// TestConcurrentPunching tests multiple simultaneous punch attempts
func TestConcurrentPunching(t *testing.T) {
	const numPeers = 3

	punchers := make([]*punch.Puncher, numPeers)
	for i := 0; i < numPeers; i++ {
		p, err := punch.NewPuncher(nil)
		if err != nil {
			t.Fatalf("Failed to create puncher %d: %v", i, err)
		}
		defer p.Close()
		punchers[i] = p
	}

	// Each puncher tries to connect to the next one
	var wg sync.WaitGroup
	errors := make(chan error, numPeers)

	for i := 0; i < numPeers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			nextIdx := (idx + 1) % numPeers
			peer := &punch.PeerInfo{
				PublicAddr: punchers[nextIdx].LocalAddr(),
				LocalAddrs: []*net.UDPAddr{punchers[nextIdx].LocalAddr()},
				NATType:    nat.TypeOpenInternet,
			}

			conn, err := punchers[idx].PunchHole(peer)
			if err != nil {
				errors <- err
				return
			}
			if conn != nil {
				t.Logf("Peer %d connected to peer %d", idx, nextIdx)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Logf("Connection error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Logf("%d/%d connections failed", errorCount, numPeers)
	}
}

// TestQuickPunch tests the convenience function
func TestQuickPunch(t *testing.T) {
	// This would need a real peer to connect to
	// For now, we just test that the function can be called

	mapping := &nat.Mapping{
		LocalAddr:  &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345},
		PublicAddr: &net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 12345},
		Type:       nat.TypeFullCone,
		DetectedAt: time.Now(),
	}

	peer := &punch.PeerInfo{
		PublicAddr: &net.UDPAddr{IP: net.ParseIP("203.0.113.2"), Port: 54321},
		NATType:    nat.TypeFullCone,
	}

	// This will timeout since peer doesn't exist
	conn, err := punch.QuickPunch(peer, mapping)
	if err == nil {
		conn.Close()
		t.Log("Unexpected success - peer doesn't exist")
	} else {
		t.Logf("Expected failure: %v", err)
	}
}

// TestPortAllocation tests that punchers use different ports
func TestPortAllocation(t *testing.T) {
	const numPunchers = 10

	punchers := make([]*punch.Puncher, numPunchers)
	ports := make(map[int]bool)

	for i := 0; i < numPunchers; i++ {
		p, err := punch.NewPuncher(nil)
		if err != nil {
			t.Fatalf("Failed to create puncher %d: %v", i, err)
		}
		defer p.Close()

		punchers[i] = p
		port := p.LocalAddr().Port

		if ports[port] {
			t.Errorf("Port %d reused", port)
		}
		ports[port] = true
	}

	t.Logf("Allocated %d unique ports", len(ports))
}






