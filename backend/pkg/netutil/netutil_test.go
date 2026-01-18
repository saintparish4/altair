package netutil

import (
	"net"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		// IPv4 private ranges
		{"10.0.0.0/8 start", "10.0.0.1", true},
		{"10.0.0.0/8 end", "10.255.255.254", true},
		{"172.16.0.0/12 start", "172.16.0.1", true},
		{"172.16.0.0/12 middle", "172.20.0.1", true},
		{"172.16.0.0/12 end", "172.31.255.254", true},
		{"192.168.0.0/16 start", "192.168.0.1", true},
		{"192.168.0.0/16 end", "192.168.255.254", true},
		{"Link-local 169.254.0.0/16", "169.254.1.1", true},

		// IPv4 public ranges
		{"Public IP 1", "8.8.8.8", false},
		{"Public IP 2", "1.1.1.1", false},
		{"Public IP 3", "203.0.113.1", false},
		{"Outside 172 range low", "172.15.255.254", false},
		{"Outside 172 range high", "172.32.0.1", false},

		// IPv6 private ranges
		{"IPv6 ULA fc00::/7", "fc00::1", true},
		{"IPv6 ULA fd00::/8", "fd00::1", true},
		{"IPv6 link-local fe80::/10", "fe80::1", true},

		// IPv6 public
		{"IPv6 public", "2001:db8::1", false},

		// Edge cases
		{"Nil IP", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ip net.IP
			if tt.ip != "" {
				ip = net.ParseIP(tt.ip)
				if ip == nil {
					t.Fatalf("failed to parse IP: %s", tt.ip)
				}
			}

			result := IsPrivateIP(ip)
			if result != tt.expected {
				t.Errorf("IsPrivateIP(%s) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsPublicIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"Google DNS", "8.8.8.8", true},
		{"Cloudflare DNS", "1.1.1.1", true},
		{"Private 10.x", "10.0.0.1", false},
		{"Private 192.168.x", "192.168.1.1", false},
		{"Loopback", "127.0.0.1", false},
		{"Link-local", "169.254.1.1", false},
		{"IPv6 public", "2001:4860:4860::8888", true},
		{"IPv6 link-local", "fe80::1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP: %s", tt.ip)
			}

			result := IsPublicIP(ip)
			if result != tt.expected {
				t.Errorf("IsPublicIP(%s) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestGetLocalAddresses(t *testing.T) {
	addresses, err := GetLocalAddresses()
	if err != nil {
		t.Fatalf("GetLocalAddresses() failed: %v", err)
	}

	// Should have at least one address on most systems
	if len(addresses) == 0 {
		t.Log("Warning: No local addresses found (might be normal in container)")
	}

	// Verify no loopback addresses
	for _, addr := range addresses {
		if addr.IsLoopback() {
			t.Errorf("GetLocalAddresses() returned loopback address: %s", addr)
		}
	}
}

func TestGetPrivateAddresses(t *testing.T) {
	addresses, err := GetPrivateAddresses()
	if err != nil {
		t.Fatalf("GetPrivateAddresses() failed: %v", err)
	}

	// Verify all returned addresses are private
	for _, addr := range addresses {
		if !IsPrivateIP(addr) {
			t.Errorf("GetPrivateAddresses() returned non-private address: %s", addr)
		}
	}
}

func TestGetPublicAddresses(t *testing.T) {
	addresses, err := GetPublicAddresses()
	if err != nil {
		t.Fatalf("GetPublicAddresses() failed: %v", err)
	}

	// Verify all returned addresses are public
	for _, addr := range addresses {
		if !IsPublicIP(addr) {
			t.Errorf("GetPublicAddresses() returned non-public address: %s", addr)
		}
	}
}

func TestFindAvailablePort(t *testing.T) {
	port, err := FindAvailablePort()
	if err != nil {
		t.Fatalf("FindAvailablePort() failed: %v", err)
	}

	if port <= 0 || port > 65535 {
		t.Errorf("FindAvailablePort() = %d, want valid port number", port)
	}

	// Verify we can actually bind to the port
	conn, err := CreateUDPSocket(port)
	if err != nil {
		t.Fatalf("Cannot bind to port returned by FindAvailablePort(): %v", err)
	}
	conn.Close()
}

func TestCreateUDPSocket(t *testing.T) {
	// Test with port 0 (system assigns)
	conn, err := CreateUDPSocket(0)
	if err != nil {
		t.Fatalf("CreateUDPSocket(0) failed: %v", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	if localAddr.Port == 0 {
		t.Error("System should have assigned a non-zero port")
	}
}

func TestCreateUDPSocketWithAddress(t *testing.T) {
	// Test with loopback
	conn, err := CreateUDPSocketWithAddress("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("CreateUDPSocketWithAddress() failed: %v", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	if !localAddr.IP.IsLoopback() {
		t.Errorf("Expected loopback address, got %s", localAddr.IP)
	}
}

func TestPortScanner(t *testing.T) {
	scanner := NewPortScanner(10000, 10010)

	// Find a port
	port, err := scanner.FindPort()
	if err != nil {
		t.Fatalf("FindPort() failed: %v", err)
	}

	if port < 10000 || port > 10010 {
		t.Errorf("FindPort() = %d, want port in range [10000, 10010]", port)
	}

	// Port should be marked as used
	if !scanner.used[port] {
		t.Error("Port should be marked as used")
	}

	// Release and find again
	scanner.ReleasePort(port)
	if scanner.used[port] {
		t.Error("Port should be released")
	}

	port2, err := scanner.FindPort()
	if err != nil {
		t.Fatalf("FindPort() after release failed: %v", err)
	}

	if port2 < 10000 || port2 > 10010 {
		t.Errorf("FindPort() = %d, want port in range [10000, 10010]", port2)
	}
}

func TestPortScannerExhaustion(t *testing.T) {
	// Create scanner with very small range
	scanner := NewPortScanner(10000, 10002)

	// Find all ports
	ports := []int{}
	for i := 0; i < 3; i++ {
		port, err := scanner.FindPort()
		if err != nil {
			t.Fatalf("FindPort() iteration %d failed: %v", i, err)
		}
		ports = append(ports, port)
	}

	// Next call should fail (exhausted)
	_, err := scanner.FindPort()
	if err == nil {
		t.Error("FindPort() should fail when range is exhausted")
	}
}

func TestGetPreferredLocalAddress(t *testing.T) {
	ip, err := GetPreferredLocalAddress()
	if err != nil {
		t.Fatalf("GetPreferredLocalAddress() failed: %v", err)
	}

	if ip == nil {
		t.Fatal("GetPreferredLocalAddress() returned nil IP")
	}

	if ip.IsLoopback() {
		t.Error("Preferred address should not be loopback")
	}
}

func TestSameNetwork(t *testing.T) {
	tests := []struct {
		name     string
		ip1      string
		ip2      string
		mask     string
		expected bool
	}{
		{
			name:     "Same /24 network",
			ip1:      "192.168.1.10",
			ip2:      "192.168.1.20",
			mask:     "255.255.255.0",
			expected: true,
		},
		{
			name:     "Different /24 network",
			ip1:      "192.168.1.10",
			ip2:      "192.168.2.10",
			mask:     "255.255.255.0",
			expected: false,
		},
		{
			name:     "Same /16 network",
			ip1:      "192.168.1.10",
			ip2:      "192.168.2.10",
			mask:     "255.255.0.0",
			expected: true,
		},
		{
			name:     "Different /16 network",
			ip1:      "192.168.1.10",
			ip2:      "192.169.1.10",
			mask:     "255.255.0.0",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip1 := net.ParseIP(tt.ip1)
			ip2 := net.ParseIP(tt.ip2)
			mask := net.IPMask(net.ParseIP(tt.mask).To4())

			result := SameNetwork(ip1, ip2, mask)
			if result != tt.expected {
				t.Errorf("SameNetwork(%s, %s, %s) = %v, want %v",
					tt.ip1, tt.ip2, tt.mask, result, tt.expected)
			}
		})
	}
}

func TestSameNetworkEdgeCases(t *testing.T) {
	ip := net.ParseIP("192.168.1.1")
	mask := net.CIDRMask(24, 32)

	// Nil IP cases
	if SameNetwork(nil, ip, mask) {
		t.Error("SameNetwork(nil, ip, mask) should be false")
	}
	if SameNetwork(ip, nil, mask) {
		t.Error("SameNetwork(ip, nil, mask) should be false")
	}
	if SameNetwork(ip, ip, nil) {
		t.Error("SameNetwork(ip, ip, nil) should be false")
	}

	// IPv4 vs IPv6
	ipv4 := net.ParseIP("192.168.1.1")
	ipv6 := net.ParseIP("2001:db8::1")
	if SameNetwork(ipv4, ipv6, mask) {
		t.Error("SameNetwork should be false for IPv4 vs IPv6")
	}
}

func TestValidateUDPAddr(t *testing.T) {
	tests := []struct {
		name      string
		addr      string
		wantError bool
	}{
		{"Valid IPv4", "192.168.1.1:8080", false},
		{"Valid IPv6", "[2001:db8::1]:8080", false},
		{"Valid hostname", "example.com:8080", false},
		{"Valid localhost", "localhost:8080", false},
		{"Invalid - no port", "192.168.1.1", true},
		{"Invalid - bad port", "192.168.1.1:99999", true},
		{"Invalid - empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUDPAddr(tt.addr)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateUDPAddr(%q) error = %v, wantError %v",
					tt.addr, err, tt.wantError)
			}
		})
	}
}

func TestResolveUDPAddr(t *testing.T) {
	// Test valid address
	addr, err := ResolveUDPAddr("127.0.0.1:8080")
	if err != nil {
		t.Fatalf("ResolveUDPAddr() failed: %v", err)
	}

	if addr.Port != 8080 {
		t.Errorf("Port = %d, want 8080", addr.Port)
	}

	if !addr.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Errorf("IP = %s, want 127.0.0.1", addr.IP)
	}

	// Test invalid address
	_, err = ResolveUDPAddr("invalid:address:format")
	if err == nil {
		t.Error("ResolveUDPAddr() should fail for invalid address")
	}
}

func TestPortScannerPrivilegedPorts(t *testing.T) {
	// Scanner should avoid privileged ports (< 1024)
	scanner := NewPortScanner(100, 2000)

	if scanner.start < 1024 {
		t.Errorf("PortScanner start = %d, should be >= 1024", scanner.start)
	}
}

func TestPortScannerInvalidRange(t *testing.T) {
	// Test with reversed range (should swap)
	scanner := NewPortScanner(10000, 9000)

	if scanner.start > scanner.end {
		t.Errorf("PortScanner should swap start and end when start > end")
	}
}

func BenchmarkIsPrivateIP(b *testing.B) {
	ip := net.ParseIP("192.168.1.1")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		IsPrivateIP(ip)
	}
}

func BenchmarkGetLocalAddresses(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetLocalAddresses()
	}
}

func BenchmarkCreateUDPSocket(b *testing.B) {
	for i := 0; i < b.N; i++ {
		conn, err := CreateUDPSocket(0)
		if err != nil {
			b.Fatal(err)
		}
		conn.Close()
	}
}
