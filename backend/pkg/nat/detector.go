package nat

import (
	"fmt"
	"net"
	"time"

	"github.com/saintparish4/altair/pkg/stun"
)

// Type represents the type of NAT detected
type Type int

const (
	// TypeUnknown indicates NAT type could not be determined
	TypeUnknown Type = iota

	// TypeOpenInternet indicates no NAT (direct public IP)
	TypeOpenInternet

	// TypeFullCone indicates full cone NAT
	// Same public IP:port for all destinations
	// Any external host can send to the public endpoint
	TypeFullCone

	// TypeRestrictedCone indicates address-restricted cone NAT
	// Same public IP:port for all destinations
	// Only hosts we've sent to can send back
	TypeRestrictedCone

	// TypePortRestrictedCone indicates port-restricted cone NAT
	// Same public IP:port for all destinations
	// Only specific IP:port pairs we've sent to can send back
	TypePortRestrictedCone

	// TypeSymmetric indicates symmetric NAT
	// Different public IP:port for each destination
	// Difficult for P2P connectivity
	TypeSymmetric

	// TypeBlocked indicates all UDP is blocked
	TypeBlocked
)

// String returns a human-readable name for the NAT type
func (t Type) String() string {
	switch t {
	case TypeUnknown:
		return "Unknown"
	case TypeOpenInternet:
		return "Open Internet"
	case TypeFullCone:
		return "Full Cone"
	case TypeRestrictedCone:
		return "Restricted Cone"
	case TypePortRestrictedCone:
		return "Port Restricted Cone"
	case TypeSymmetric:
		return "Symmetric"
	case TypeBlocked:
		return "Blocked"
	default:
		return fmt.Sprintf("Unknown(%d)", int(t))
	}
}

// SupportsP2P returns whether this NAT type generally supports P2P connections
func (t Type) SupportsP2P() bool {
	switch t {
	case TypeOpenInternet, TypeFullCone, TypeRestrictedCone, TypePortRestrictedCone:
		return true
	case TypeSymmetric:
		return false // Difficult but sometimes possible
	case TypeBlocked, TypeUnknown:
		return false
	default:
		return false
	}
}

// Difficulty returns a difficulty score (0-10) for establishing P2P connections
func (t Type) Difficulty() int {
	switch t {
	case TypeOpenInternet:
		return 0
	case TypeFullCone:
		return 1
	case TypeRestrictedCone:
		return 3
	case TypePortRestrictedCone:
		return 5
	case TypeSymmetric:
		return 9
	case TypeBlocked:
		return 10
	case TypeUnknown:
		return 10
	default:
		return 10
	}
}

// Mapping represents a NAT mapping (local to public address translation)
type Mapping struct {
	LocalAddr  *net.UDPAddr // Local (private) address
	PublicAddr *net.UDPAddr // Public (mapped) address
	Type       Type         // Detected NAT type
	DetectedAt time.Time    // When the mapping was discovered
}

// String returns a human-readable representation of the mapping
func (m *Mapping) String() string {
	if m == nil {
		return "<nil mapping>"
	}
	return fmt.Sprintf("%s (local: %s, public: %s, detected: %s)",
		m.Type, m.LocalAddr, m.PublicAddr, m.DetectedAt.Format(time.RFC3339))
}

// IsValid checks if the mapping is still likely valid
func (m *Mapping) IsValid(maxAge time.Duration) bool {
	if m == nil || m.PublicAddr == nil {
		return false
	}
	return time.Since(m.DetectedAt) < maxAge
}

// Detector performs NAT type detection using STUN
type Detector struct {
	primary    *stun.Client
	secondary  *stun.Client
	localConn  *net.UDPConn
	timeout    time.Duration
	retryCount int
}

// DetectorConfig holds configuration for NAT detection
type DetectorConfig struct {
	// Primary STUN server
	PrimaryServer string

	// Secondary STUN server (different IP than primary)
	SecondaryServer string

	// Timeout for STUN requests
	Timeout time.Duration

	// Number of retries for failed requests
	RetryCount int

	// Optional: existing UDP connection to use
	LocalConn *net.UDPConn
}

// DefaultConfig returns a detector configuration with sensible defaults
func DefaultConfig() *DetectorConfig {
	return &DetectorConfig{
		PrimaryServer:   "stun.l.google.com:19302",
		SecondaryServer: "stun1.l.google.com:19302",
		Timeout:         5 * time.Second,
		RetryCount:      3,
	}
}

// NewDetector creates a new NAT type detector
func NewDetector(config *DetectorConfig) (*Detector, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create primary STUN client
	primary, err := stun.NewClient(&stun.ClientConfig{
		ServerAddr: config.PrimaryServer,
		Timeout:    config.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create primary STUN client: %w", err)
	}

	// Create secondary STUN client
	secondary, err := stun.NewClient(&stun.ClientConfig{
		ServerAddr: config.SecondaryServer,
		Timeout:    config.Timeout,
	})
	if err != nil {
		primary.Close()
		return nil, fmt.Errorf("failed to create secondary STUN client: %w", err)
	}

	return &Detector{
		primary:    primary,
		secondary:  secondary,
		localConn:  config.LocalConn,
		timeout:    config.Timeout,
		retryCount: config.RetryCount,
	}, nil
}

// Detect performs NAT type detection using the RFC 3489 algorithm
func (d *Detector) Detect() (*Mapping, error) {
	// Test 1: Send request to primary server
	endpoint1, err := d.primary.Discover()
	if err != nil {
		return nil, fmt.Errorf("test 1 failed (primary server): %w", err)
	}

	// Check if we have a public IP (no NAT)
	if endpoint1.LocalAddr.IP.Equal(endpoint1.PublicAddr.IP) {
		return &Mapping{
			LocalAddr:  endpoint1.LocalAddr,
			PublicAddr: endpoint1.PublicAddr,
			Type:       TypeOpenInternet,
			DetectedAt: time.Now(),
		}, nil
	}

	// We're behind NAT, continue tests
	// Test 2: Send request to secondary server (different IP)
	endpoint2, err := d.secondary.Discover()
	if err != nil {
		return nil, fmt.Errorf("test 2 failed (secondary server): %w", err)
	}

	// Check if public IP and port are the same from both servers
	sameIP := endpoint1.PublicAddr.IP.Equal(endpoint2.PublicAddr.IP)
	samePort := endpoint1.PublicAddr.Port == endpoint2.PublicAddr.Port

	if !sameIP || !samePort {
		// Different public endpoint for different destination = Symmetric NAT
		return &Mapping{
			LocalAddr:  endpoint1.LocalAddr,
			PublicAddr: endpoint1.PublicAddr,
			Type:       TypeSymmetric,
			DetectedAt: time.Now(),
		}, nil
	}

	// Same public endpoint from both servers
	// Now we need to determine cone type (Full, Restricted, or Port-Restricted)
	// This requires STUN server features we don't have in basic implementation
	// For now, we classify as Restricted Cone (most common)

	// TODO: Implement full cone detection (requires CHANGE-REQUEST attribute support)
	// This would involve requesting server to respond from different IP/port
	// and checking if we receive the response

	return &Mapping{
		LocalAddr:  endpoint1.LocalAddr,
		PublicAddr: endpoint1.PublicAddr,
		Type:       TypeRestrictedCone, // Conservative estimate
		DetectedAt: time.Now(),
	}, nil
}

// DetectWithRetry performs NAT detection with automatic retry on failure
func (d *Detector) DetectWithRetry() (*Mapping, error) {
	var lastErr error

	for attempt := 0; attempt <= d.retryCount; attempt++ {
		mapping, err := d.Detect()
		if err == nil {
			return mapping, nil
		}

		lastErr = err

		// Wait before retry (exponential backoff)
		if attempt < d.retryCount {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			if backoff > 10*time.Second {
				backoff = 10 * time.Second
			}
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("detection failed after %d attempts: %w", d.retryCount, lastErr)
}

// Close releases resources used by the detector
func (d *Detector) Close() error {
	var err1, err2 error

	if d.primary != nil {
		err1 = d.primary.Close()
	}

	if d.secondary != nil {
		err2 = d.secondary.Close()
	}

	if err1 != nil {
		return err1
	}
	return err2
}

// QuickDetect is a convenience function for one-off NAT detection
func QuickDetect() (*Mapping, error) {
	detector, err := NewDetector(DefaultConfig())
	if err != nil {
		return nil, err
	}
	defer detector.Close()

	return detector.DetectWithRetry()
}

// DetectWithServers detects NAT type using specified STUN servers
func DetectWithServers(primary, secondary string) (*Mapping, error) {
	config := &DetectorConfig{
		PrimaryServer:   primary,
		SecondaryServer: secondary,
		Timeout:         5 * time.Second,
		RetryCount:      3,
	}

	detector, err := NewDetector(config)
	if err != nil {
		return nil, err
	}
	defer detector.Close()

	return detector.DetectWithRetry()
}

// CanHolePunch returns whether two NAT types can likely establish a P2P connection
func CanHolePunch(type1, type2 Type) bool {
	// Blocked can't connect to anything
	if type1 == TypeBlocked || type2 == TypeBlocked {
		return false
	}

	// Open internet can connect to anything (except blocked, already handled above)
	if type1 == TypeOpenInternet || type2 == TypeOpenInternet {
		return true
	}

	// Full cone can connect to anything (except blocked)
	if type1 == TypeFullCone || type2 == TypeFullCone {
		return true
	}

	// Restricted cone can connect to restricted cone
	if type1 == TypeRestrictedCone && type2 == TypeRestrictedCone {
		return true
	}

	// Port restricted cone can connect to port restricted cone
	if type1 == TypePortRestrictedCone && type2 == TypePortRestrictedCone {
		return true
	}

	// Symmetric NAT is difficult
	if type1 == TypeSymmetric || type2 == TypeSymmetric {
		// Symmetric <-> Symmetric is very difficult
		if type1 == TypeSymmetric && type2 == TypeSymmetric {
			return false
		}
		// Symmetric <-> Cone might work with port prediction
		return false // Conservative estimate
	}

	return false
}
