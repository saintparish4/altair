package netutil

import (
	"fmt"
	"net"
	"sync"
)

// GetLocalAddresses returns all non-loopback local IP addresses
func GetLocalAddresses() ([]net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	var addresses []net.IP
	for _, iface := range ifaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Skip loopback addresses
			if ip == nil || ip.IsLoopback() {
				continue
			}

			addresses = append(addresses, ip)
		}
	}

	return addresses, nil
}

// GetPrivateAddresses returns all private local IP addresses
func GetPrivateAddresses() ([]net.IP, error) {
	all, err := GetLocalAddresses()
	if err != nil {
		return nil, err
	}

	var private []net.IP
	for _, ip := range all {
		if IsPrivateIP(ip) {
			private = append(private, ip)
		}
	}

	return private, nil
}

// GetPublicAddresses returns all public (non-private) local IP addresses
func GetPublicAddresses() ([]net.IP, error) {
	all, err := GetLocalAddresses()
	if err != nil {
		return nil, err
	}

	var public []net.IP
	for _, ip := range all {
		if !IsPrivateIP(ip) {
			public = append(public, ip)
		}
	}

	return public, nil
}

// IsPrivateIP checks if an IP address is in a private range
func IsPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// Check for IPv4 private ranges
	if ip4 := ip.To4(); ip4 != nil {
		// 10.0.0.0/8
		if ip4[0] == 10 {
			return true
		}
		// 172.16.0.0/12
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		// 169.254.0.0/16 (link-local)
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		return false
	}

	// Check for IPv6 private ranges
	// fc00::/7 (unique local addresses)
	if len(ip) == net.IPv6len && ip[0] >= 0xfc && ip[0] <= 0xfd {
		return true
	}

	// fe80::/10 (link-local)
	if len(ip) == net.IPv6len && ip[0] == 0xfe && ip[1] >= 0x80 && ip[1] <= 0xbf {
		return true
	}

	return false
}

// IsPublicIP checks if an IP address is routable on the public internet
func IsPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}

	return !IsPrivateIP(ip)
}

// FindAvailablePort finds an available UDP port by binding to :0
func FindAvailablePort() (int, error) {
	addr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return 0, fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return 0, fmt.Errorf("failed to listen on UDP port: %w", err)
	}
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).Port, nil
}

// CreateUDPSocket creates a UDP socket bound to the specificed port
// If port is 0, the system assigns an available port
func CreateUDPSocket(port int) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP socket: %w", err)
	}

	return conn, nil
}

// CreateUDPSocketWithAddress creates a UDP socket bound to a specific address and port
func CreateUDPSocketWithAddress(address string, port int) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP socket: %w", err)
	}

	return conn, nil
}

// PortScanner helps find multiple available ports
type PortScanner struct {
	mu    sync.Mutex
	used  map[int]bool
	start int
	end   int
}

// NewPortScanner creates a port scanner for a given range
func NewPortScanner(start, end int) *PortScanner {
	if start < 1024 {
		start = 1024 // Avoid privileged ports
	}
	if end > 65535 {
		end = 65535
	}
	if start > end {
		start, end = end, start
	}

	return &PortScanner{
		used:  make(map[int]bool),
		start: start,
		end:   end,
	}
}

// FindPort finds an available port in the range
func (ps *PortScanner) FindPort() (int, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	for port := ps.start; port <= ps.end; port++ {
		if ps.used[port] {
			continue
		}

		// Try to bind to the port
		conn, err := CreateUDPSocket(port)
		if err == nil {
			conn.Close()
			ps.used[port] = true
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", ps.start, ps.end)
}

// ReleasePort marks a port as available again
func (ps *PortScanner) ReleasePort(port int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.used, port)
}

// GetPreferredLocalAddress returns the preferred local address for external communication
// It attempts to determine which local address would be used for internet connectivity
func GetPreferredLocalAddress() (net.IP, error) {
	// Try to connect to a public address (doesn't actually send data)
	// This helps determine which interface would be used for internet access
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		// Fallback to first non-loopback address
		addresses, err := GetLocalAddresses()
		if err != nil {
			return nil, err
		}
		if len(addresses) > 0 {
			return addresses[0], nil
		}
		return nil, fmt.Errorf("no local addresses found")
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}

// SameNetwork checks if two IP addresses are on the same network
func SameNetwork(ip1, ip2 net.IP, mask net.IPMask) bool {
	if ip1 == nil || ip2 == nil || mask == nil {
		return false
	}

	// Ensure both are IPv4 or both are IPv6
	if ip1.To4() != nil && ip2.To4() == nil {
		return false
	}
	if ip1.To4() == nil && ip2.To4() != nil {
		return false
	}

	network1 := ip1.Mask(mask)
	network2 := ip2.Mask(mask)

	return network1.Equal(network2)
}

// ValidateUDPAddr checks if a UDP address string is valid
func ValidateUDPAddr(addr string) error {
	if addr == "" {
		return fmt.Errorf("invalid UDP address: empty address")
	}
	_, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("invalid UDP address %q: %w", addr, err)
	}
	return nil
}

// ResolveUDPAddr resolves a UDP address with better error messages
func ResolveUDPAddr(addr string) (*net.UDPAddr, error) {
	resolved, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address %q: %w", addr, err)
	}

	if resolved.IP == nil {
		return nil, fmt.Errorf("resolved address has no IP: %s", addr)
	}

	return resolved, nil
}
