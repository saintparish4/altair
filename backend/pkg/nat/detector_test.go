package nat

import (
	"net"
	"testing"
	"time"
)

func TestTypeString(t *testing.T) {
	tests := []struct {
		natType  Type
		expected string
	}{
		{TypeUnknown, "Unknown"},
		{TypeOpenInternet, "Open Internet"},
		{TypeFullCone, "Full Cone"},
		{TypeRestrictedCone, "Restricted Cone"},
		{TypePortRestrictedCone, "Port Restricted Cone"},
		{TypeSymmetric, "Symmetric"},
		{TypeBlocked, "Blocked"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.natType.String()
			if result != tt.expected {
				t.Errorf("Type.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTypeSupportsP2P(t *testing.T) {
	tests := []struct {
		natType  Type
		expected bool
	}{
		{TypeUnknown, false},
		{TypeOpenInternet, true},
		{TypeFullCone, true},
		{TypeRestrictedCone, true},
		{TypePortRestrictedCone, true},
		{TypeSymmetric, false},
		{TypeBlocked, false},
	}

	for _, tt := range tests {
		t.Run(tt.natType.String(), func(t *testing.T) {
			result := tt.natType.SupportsP2P()
			if result != tt.expected {
				t.Errorf("%s.SupportsP2P() = %v, want %v",
					tt.natType, result, tt.expected)
			}
		})
	}
}

func TestTypeDifficulty(t *testing.T) {
	tests := []struct {
		natType       Type
		maxDifficulty int
	}{
		{TypeOpenInternet, 0},
		{TypeFullCone, 2},
		{TypeRestrictedCone, 4},
		{TypePortRestrictedCone, 6},
		{TypeSymmetric, 10},
		{TypeBlocked, 10},
		{TypeUnknown, 10},
	}

	for _, tt := range tests {
		t.Run(tt.natType.String(), func(t *testing.T) {
			difficulty := tt.natType.Difficulty()
			if difficulty > tt.maxDifficulty {
				t.Errorf("%s.Difficulty() = %d, want <= %d",
					tt.natType, difficulty, tt.maxDifficulty)
			}

			// Ensure difficulty is in valid range
			if difficulty < 0 || difficulty > 10 {
				t.Errorf("Difficulty %d out of range [0, 10]", difficulty)
			}
		})
	}
}

func TestTypeDifficultyOrdering(t *testing.T) {
	// Verify difficulty increases with NAT complexity
	if TypeOpenInternet.Difficulty() >= TypeFullCone.Difficulty() {
		t.Error("Open Internet should be easier than Full Cone")
	}

	if TypeFullCone.Difficulty() >= TypeRestrictedCone.Difficulty() {
		t.Error("Full Cone should be easier than Restricted Cone")
	}

	if TypeRestrictedCone.Difficulty() >= TypePortRestrictedCone.Difficulty() {
		t.Error("Restricted Cone should be easier than Port Restricted Cone")
	}

	if TypePortRestrictedCone.Difficulty() >= TypeSymmetric.Difficulty() {
		t.Error("Port Restricted Cone should be easier than Symmetric")
	}
}

func TestMappingString(t *testing.T) {
	// Test nil mapping
	var m *Mapping
	result := m.String()
	if result != "<nil mapping>" {
		t.Errorf("nil mapping String() = %q, want \"<nil mapping>\"", result)
	}

	// Test valid mapping (just check it doesn't panic)
	m = &Mapping{
		Type:       TypeFullCone,
		DetectedAt: time.Now(),
	}
	_ = m.String()
}

func TestMappingIsValid(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		mapping  *Mapping
		maxAge   time.Duration
		expected bool
	}{
		{
			name:     "nil mapping",
			mapping:  nil,
			maxAge:   1 * time.Minute,
			expected: false,
		},
		{
			name: "nil public addr",
			mapping: &Mapping{
				PublicAddr: nil,
				DetectedAt: now,
			},
			maxAge:   1 * time.Minute,
			expected: false,
		},
		{
			name: "fresh mapping",
			mapping: &Mapping{
				PublicAddr: &mockUDPAddr,
				DetectedAt: now,
			},
			maxAge:   1 * time.Minute,
			expected: true,
		},
		{
			name: "expired mapping",
			mapping: &Mapping{
				PublicAddr: &mockUDPAddr,
				DetectedAt: now.Add(-2 * time.Minute),
			},
			maxAge:   1 * time.Minute,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mapping.IsValid(tt.maxAge)
			if result != tt.expected {
				t.Errorf("IsValid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.PrimaryServer == "" {
		t.Error("DefaultConfig should set PrimaryServer")
	}

	if config.SecondaryServer == "" {
		t.Error("DefaultConfig should set SecondaryServer")
	}

	if config.Timeout == 0 {
		t.Error("DefaultConfig should set Timeout")
	}

	if config.RetryCount == 0 {
		t.Error("DefaultConfig should set RetryCount")
	}

	// Verify servers are different
	if config.PrimaryServer == config.SecondaryServer {
		t.Error("Primary and secondary servers should be different")
	}
}

func TestCanHolePunch(t *testing.T) {
	tests := []struct {
		name     string
		type1    Type
		type2    Type
		expected bool
	}{
		// Open Internet cases
		{"Open <-> Open", TypeOpenInternet, TypeOpenInternet, true},
		{"Open <-> Full Cone", TypeOpenInternet, TypeFullCone, true},
		{"Open <-> Symmetric", TypeOpenInternet, TypeSymmetric, true},
		{"Open <-> Blocked", TypeOpenInternet, TypeBlocked, false},

		// Full Cone cases
		{"Full Cone <-> Full Cone", TypeFullCone, TypeFullCone, true},
		{"Full Cone <-> Restricted", TypeFullCone, TypeRestrictedCone, true},
		{"Full Cone <-> Symmetric", TypeFullCone, TypeSymmetric, true},
		{"Full Cone <-> Blocked", TypeFullCone, TypeBlocked, false},

		// Restricted Cone cases
		{"Restricted <-> Restricted", TypeRestrictedCone, TypeRestrictedCone, true},
		{"Restricted <-> Port Restricted", TypeRestrictedCone, TypePortRestrictedCone, false},
		{"Restricted <-> Symmetric", TypeRestrictedCone, TypeSymmetric, false},

		// Port Restricted Cone cases
		{"Port Restricted <-> Port Restricted", TypePortRestrictedCone, TypePortRestrictedCone, true},
		{"Port Restricted <-> Symmetric", TypePortRestrictedCone, TypeSymmetric, false},

		// Symmetric cases
		{"Symmetric <-> Symmetric", TypeSymmetric, TypeSymmetric, false},
		{"Symmetric <-> Blocked", TypeSymmetric, TypeBlocked, false},

		// Blocked cases
		{"Blocked <-> Blocked", TypeBlocked, TypeBlocked, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanHolePunch(tt.type1, tt.type2)
			if result != tt.expected {
				t.Errorf("CanHolePunch(%s, %s) = %v, want %v",
					tt.type1, tt.type2, result, tt.expected)
			}

			// Verify symmetry
			result2 := CanHolePunch(tt.type2, tt.type1)
			if result != result2 {
				t.Errorf("CanHolePunch is not symmetric: (%s, %s) = %v but (%s, %s) = %v",
					tt.type1, tt.type2, result, tt.type2, tt.type1, result2)
			}
		})
	}
}

func TestCanHolePunchConsistency(t *testing.T) {
	// If SupportsP2P returns false for both types, CanHolePunch should be false
	allTypes := []Type{
		TypeUnknown, TypeOpenInternet, TypeFullCone,
		TypeRestrictedCone, TypePortRestrictedCone,
		TypeSymmetric, TypeBlocked,
	}

	for _, type1 := range allTypes {
		for _, type2 := range allTypes {
			canPunch := CanHolePunch(type1, type2)

			// If neither supports P2P, should not be able to hole punch
			if !type1.SupportsP2P() && !type2.SupportsP2P() {
				if canPunch {
					t.Errorf("CanHolePunch(%s, %s) = true, but neither supports P2P",
						type1, type2)
				}
			}

			// If one is blocked, should not be able to hole punch
			if type1 == TypeBlocked || type2 == TypeBlocked {
				if canPunch {
					t.Errorf("CanHolePunch(%s, %s) = true, but one is blocked",
						type1, type2)
				}
			}
		}
	}
}

// Mock UDP address for testing
var mockUDPAddr = net.UDPAddr{
	IP:   net.ParseIP("203.0.113.1"),
	Port: 12345,
}

func TestNewDetectorNilConfig(t *testing.T) {
	// Should use default config when nil is passed
	detector, err := NewDetector(nil)
	if err != nil {
		t.Fatalf("NewDetector(nil) failed: %v", err)
	}
	defer detector.Close()

	if detector.timeout == 0 {
		t.Error("Detector should have non-zero timeout from default config")
	}
}

func TestNewDetectorInvalidServers(t *testing.T) {
	config := &DetectorConfig{
		PrimaryServer:   "invalid:server:address",
		SecondaryServer: "also:invalid",
		Timeout:         5 * time.Second,
		RetryCount:      3,
	}

	_, err := NewDetector(config)
	if err == nil {
		t.Error("NewDetector should fail with invalid server addresses")
	}
}

func TestDetectorClose(t *testing.T) {
	detector, err := NewDetector(DefaultConfig())
	if err != nil {
		t.Fatalf("NewDetector failed: %v", err)
	}

	err = detector.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Closing again should not panic
	err = detector.Close()
	if err != nil {
		t.Logf("Second Close() returned error (acceptable): %v", err)
	}
}

func BenchmarkTypeString(b *testing.B) {
	natType := TypeFullCone
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = natType.String()
	}
}

func BenchmarkTypeDifficulty(b *testing.B) {
	natType := TypeRestrictedCone
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = natType.Difficulty()
	}
}

func BenchmarkCanHolePunch(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = CanHolePunch(TypeRestrictedCone, TypeFullCone)
	}
}

// Integration test note:
// Full detector testing requires actual STUN servers and network access.
// These tests cover the API and logic without network dependencies.
// For integration tests, see test/integration/nat_test.go
