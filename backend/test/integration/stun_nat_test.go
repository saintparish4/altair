//go:build integration
// +build integration

package integration

import (
	"testing"
	"time"

	"github.com/saintparish4/altair/pkg/nat"
	"github.com/saintparish4/altair/pkg/stun"
)

// TestSTUNRealServer tests STUN client against real STUN servers
func TestSTUNRealServer(t *testing.T) {
	servers := []string{
		"stun.l.google.com:19302",
		"stun1.l.google.com:19302",
		"stun2.l.google.com:19302",
	}

	for _, server := range servers {
		t.Run(server, func(t *testing.T) {
			client, err := stun.NewClient(&stun.ClientConfig{
				ServerAddr: server,
				Timeout:    10 * time.Second,
			})
			if err != nil {
				t.Fatalf("Failed to create STUN client: %v", err)
			}
			defer client.Close()

			endpoint, err := client.Discover()
			if err != nil {
				t.Fatalf("STUN discovery failed: %v", err)
			}

			if endpoint.PublicAddr == nil {
				t.Fatalf("Public address should not be nil")
			}

			if endpoint.PublicAddr.IP == nil {
				t.Fatalf("Public IP should not be nil")
			}

			if endpoint.PublicAddr.Port == 0 {
				t.Error("Public port should not be 0")
			}

			t.Logf("Local: %s, Public: %s", endpoint.LocalAddr, endpoint.PublicAddr)
		})
	}
}

// TestSTUNDiscoveryConsistency tests that multiple discoveries return consistent results
func TestSTUNDiscoveryConsistency(t *testing.T) {
	client, err := stun.NewClient(&stun.ClientConfig{
		ServerAddr: "stun.l.google.com:19302",
		Timeout:    10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create STUN client: %v", err)
	}
	defer client.Close()

	// First discovery
	endpoint1, err := client.Discover()
	if err != nil {
		t.Fatalf("First discovery failed: %v", err)
	}

	// Second discovery
	endpoint2, err := client.Discover()
	if err != nil {
		t.Fatalf("Second discovery failed: %v", err)
	}

	// IP should be the same
	if !endpoint1.PublicAddr.IP.Equal(endpoint2.PublicAddr.IP) {
		t.Errorf("Public IP changed between discoveries: %s -> %s",
			endpoint1.PublicAddr.IP, endpoint2.PublicAddr.IP)
	}

	// Port might change with symmetric NAT, but usually stays the same
	if endpoint1.PublicAddr.Port != endpoint2.PublicAddr.Port {
		t.Logf("Port changed (possible symmetric NAT): %d -> %d",
			endpoint1.PublicAddr.Port, endpoint2.PublicAddr.Port)
	}
}

// TestSTUNWithRetry tests the retry mechanism
func TestSTUNWithRetry(t *testing.T) {
	client, err := stun.NewClient(&stun.ClientConfig{
		ServerAddr: "stun.l.google.com:19302",
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create STUN client: %v", err)
	}
	defer client.Close()

	endpoint, err := client.DiscoverWithRetry(3)
	if err != nil {
		t.Fatalf("Discovery with retry failed: %v", err)
	}

	if endpoint.PublicAddr == nil {
		t.Fatal("Public address should not be nil")
	}

	t.Logf("Discovered: %s", endpoint)
}

// TestNATDetectionReal tests NAT detection against real STUN servers
func TestNATDetectionReal(t *testing.T) {
	config := &nat.DetectorConfig{
		PrimaryServer:   "stun.l.google.com:19302",
		SecondaryServer: "stun1.l.google.com:19302",
		Timeout:         10 * time.Second,
		RetryCount:      3,
	}

	detector, err := nat.NewDetector(config)
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}
	defer detector.Close()

	mapping, err := detector.Detect()
	if err != nil {
		t.Fatalf("NAT detection failed: %v", err)
	}

	if mapping == nil {
		t.Fatal("Mapping should not be nil")
	}

	t.Logf("NAT Type: %s", mapping.Type)
	t.Logf("Local Address: %s", mapping.LocalAddr)
	t.Logf("Public Address: %s", mapping.PublicAddr)
	t.Logf("Detected At: %s", mapping.DetectedAt)

	// Verify mapping is valid
	if !mapping.IsValid(1 * time.Hour) {
		t.Error("Mapping should be valid")
	}

	// Log P2P support
	if mapping.Type.SupportsP2P() {
		t.Logf("NAT type supports P2P (difficulty: %d/10)", mapping.Type.Difficulty())
	} else {
		t.Logf("NAT type does NOT support P2P easily")
	}
}

// TestNATDetectionWithRetry tests NAT detection with retry logic
func TestNATDetectionWithRetry(t *testing.T) {
	detector, err := nat.NewDetector(nat.DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}
	defer detector.Close()

	start := time.Now()
	mapping, err := detector.DetectWithRetry()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("NAT detection with retry failed: %v", err)
	}

	t.Logf("Detection took: %v", duration)
	t.Logf("NAT Type: %s", mapping.Type)
	t.Logf("Public: %s", mapping.PublicAddr)
}

// TestQuickDetect tests the convenience function
func TestQuickDetect(t *testing.T) {
	mapping, err := nat.QuickDetect()
	if err != nil {
		t.Fatalf("QuickDetect failed: %v", err)
	}

	if mapping == nil {
		t.Fatal("Mapping should not be nil")
	}

	t.Logf("Quick detection result: %s", mapping.Type)
}

// TestMultipleSTUNServers tests using different STUN servers
func TestMultipleSTUNServers(t *testing.T) {
	serverPairs := []struct {
		primary   string
		secondary string
	}{
		{"stun.l.google.com:19302", "stun1.l.google.com:19302"},
		{"stun.l.google.com:19302", "stun2.l.google.com:19302"},
		{"stun1.l.google.com:19302", "stun2.l.google.com:19302"},
	}

	for _, pair := range serverPairs {
		t.Run(pair.primary+"+"+pair.secondary, func(t *testing.T) {
			mapping, err := nat.DetectWithServers(pair.primary, pair.secondary)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}

			t.Logf("NAT: %s, Public: %s", mapping.Type, mapping.PublicAddr)
		})
	}
}

// TestNATDetectionPerformance benchmarks NAT detection
func TestNATDetectionPerformance(t *testing.T) {
	iterations := 5
	durations := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()

		mapping, err := nat.QuickDetect()
		if err != nil {
			t.Fatalf("Detection failed: %v", err)
		}

		durations[i] = time.Since(start)
		t.Logf("Iteration %d: %v (%s)", i+1, durations[i], mapping.Type)
	}

	// Calculate average
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	avg := total / time.Duration(iterations)

	t.Logf("Average detection time: %v", avg)

	// Sanity check - should complete within 30 seconds
	if avg > 30*time.Second {
		t.Errorf("Average detection time %v is too slow", avg)
	}
}

// TestSTUNIPv4 specifically tests IPv4 STUN
func TestSTUNIPv4(t *testing.T) {
	client, err := stun.NewClient(&stun.ClientConfig{
		ServerAddr: "stun.l.google.com:19302",
		Timeout:    10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create STUN client: %v", err)
	}
	defer client.Close()

	endpoint, err := client.Discover()
	if err != nil {
		t.Fatalf("Discovery failed: %v", err)
	}

	// Check if we got an IPv4 address
	if endpoint.PublicAddr.IP.To4() == nil {
		t.Log("Warning: Got IPv6 address, not IPv4")
	} else {
		t.Logf("Got IPv4 address: %s", endpoint.PublicAddr.IP)
	}
}

// TestConcurrentSTUNRequests tests multiple concurrent STUN requests
func TestConcurrentSTUNRequests(t *testing.T) {
	const numConcurrent = 5

	results := make(chan error, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		go func(id int) {
			client, err := stun.NewClient(&stun.ClientConfig{
				ServerAddr: "stun.l.google.com:19302",
				Timeout:    10 * time.Second,
			})
			if err != nil {
				results <- err
				return
			}
			defer client.Close()

			endpoint, err := client.Discover()
			if err != nil {
				results <- err
				return
			}

			t.Logf("Goroutine %d: %s", id, endpoint.PublicAddr)
			results <- nil
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < numConcurrent; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent request failed: %v", err)
		}
	}
}
