package stun

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestNewMessage(t *testing.T) {
	msg, err := NewMessage(TypeBindingRequest)
	if err != nil {
		t.Fatalf("NewMessage failed: %v", err)
	}

	if msg.Type != TypeBindingRequest {
		t.Errorf("expected type %v, got %v", TypeBindingRequest, msg.Type)
	}

	// Verify transaction ID is not all zeros
	allZeros := true
	for _, b := range msg.TransactionID {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		t.Error("transaction ID should not be all zeros")
	}
}

func TestMessageEncodeDecodeRoundtrip(t *testing.T) {
	// Create a message
	msg, err := NewMessage(TypeBindingRequest)
	if err != nil {
		t.Fatalf("NewMessage failed: %v", err)
	}

	// Add some attributes
	msg.AddAttribute(Attribute{
		Type:   AttrSoftware,
		Length: 6,
		Value:  []byte("Altair"),
	})

	// Encode
	encoded, err := msg.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Decode
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify
	if decoded.Type != msg.Type {
		t.Errorf("type mismatch: expected %v, got %v", msg.Type, decoded.Type)
	}

	if decoded.TransactionID != msg.TransactionID {
		t.Error("transaction ID mismatch")
	}

	if len(decoded.Attributes) != len(msg.Attributes) {
		t.Errorf("attribute count mismatch: expected %d, got %d",
			len(msg.Attributes), len(decoded.Attributes))
	}

	// Verify attribute
	if attr, found := decoded.GetAttribute(AttrSoftware); found {
		if !bytes.Equal(attr.Value, []byte("Altair")) {
			t.Errorf("attribute value mismatch: expected 'Altair', got %q", attr.Value)
		}
	} else {
		t.Error("SOFTWARE attribute not found")
	}
}

func TestDecodeXORMappedAddressIPv4(t *testing.T) {
	// Create a known IPv4 address
	expectedIP := net.ParseIP("192.0.2.1")
	expectedPort := 32853

	// Create transaction ID
	var txID [TransactionIDSize]byte
	copy(txID[:], []byte("test12345678"))

	// Encode the address
	addr := &net.UDPAddr{
		IP:   expectedIP,
		Port: expectedPort,
	}
	attr := EncodeXORMappedAddress(addr, txID)

	// Decode it back
	decoded, err := DecodeXORMappedAddress(&attr, txID)
	if err != nil {
		t.Fatalf("DecodeXORMappedAddress failed: %v", err)
	}

	// Verify
	if !decoded.IP.Equal(expectedIP) {
		t.Errorf("IP mismatch: expected %v, got %v", expectedIP, decoded.IP)
	}

	if decoded.Port != expectedPort {
		t.Errorf("port mismatch: expected %d, got %d", expectedPort, decoded.Port)
	}
}

func TestDecodeXORMappedAddressIPv6(t *testing.T) {
	// Create a known IPv6 address
	expectedIP := net.ParseIP("2001:db8::1")
	expectedPort := 32853

	// Create transaction ID
	var txID [TransactionIDSize]byte
	copy(txID[:], []byte("test12345678"))

	// Encode the address
	addr := &net.UDPAddr{
		IP:   expectedIP,
		Port: expectedPort,
	}
	attr := EncodeXORMappedAddress(addr, txID)

	// Decode it back
	decoded, err := DecodeXORMappedAddress(&attr, txID)
	if err != nil {
		t.Fatalf("DecodeXORMappedAddress failed: %v", err)
	}

	// Verify
	if !decoded.IP.Equal(expectedIP) {
		t.Errorf("IP mismatch: expected %v, got %v", expectedIP, decoded.IP)
	}

	if decoded.Port != expectedPort {
		t.Errorf("port mismatch: expected %d, got %d", expectedPort, decoded.Port)
	}
}

func TestDecodeMappedAddress(t *testing.T) {
	expectedIP := net.ParseIP("203.0.113.1")
	expectedPort := 19302

	// Encode the address
	addr := &net.UDPAddr{
		IP:   expectedIP,
		Port: expectedPort,
	}
	attr := EncodeMappedAddress(addr)

	// Decode it back
	decoded, err := DecodeMappedAddress(&attr)
	if err != nil {
		t.Fatalf("DecodeMappedAddress failed: %v", err)
	}

	// Verify
	if !decoded.IP.Equal(expectedIP) {
		t.Errorf("IP mismatch: expected %v, got %v", expectedIP, decoded.IP)
	}

	if decoded.Port != expectedPort {
		t.Errorf("port mismatch: expected %d, got %d", expectedPort, decoded.Port)
	}
}

func TestMessageEncodeWithPadding(t *testing.T) {
	msg, err := NewMessage(TypeBindingRequest)
	if err != nil {
		t.Fatalf("NewMessage failed: %v", err)
	}

	// Add attribute with value that needs padding (5 bytes)
	msg.AddAttribute(Attribute{
		Type:   AttrSoftware,
		Length: 5,
		Value:  []byte("Hello"),
	})

	encoded, err := msg.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Message length should be padded to 4-byte boundary
	// Header (20) + Attr header (4) + Value (5) + Padding (3) = 32
	if len(encoded) != 32 {
		t.Errorf("expected length 32, got %d", len(encoded))
	}

	// Verify we can decode it back
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	attr, found := decoded.GetAttribute(AttrSoftware)
	if !found {
		t.Fatal("SOFTWARE attribute not found")
	}

	if !bytes.Equal(attr.Value, []byte("Hello")) {
		t.Errorf("value mismatch: expected 'Hello', got %q", attr.Value)
	}
}

func TestClientLocalAddr(t *testing.T) {
	// Create client with ephemeral port
	client, err := NewClient(&ClientConfig{
		ServerAddr: "stun.l.google.com:19302",
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	localAddr := client.LocalAddr()
	if localAddr == nil {
		t.Fatal("LocalAddr returned nil")
	}

	if localAddr.Port == 0 {
		t.Error("local port should not be 0")
	}
}

func TestMessageTypeString(t *testing.T) {
	tests := []struct {
		msgType  MessageType
		expected string
	}{
		{TypeBindingRequest, "Binding Request"},
		{TypeBindingSuccess, "Binding Success Response"},
		{TypeBindingError, "Binding Error Response"},
		{MessageType(0x9999), "Unknown (0x9999)"},
	}

	for _, tt := range tests {
		result := tt.msgType.String()
		if result != tt.expected {
			t.Errorf("MessageType.String() = %q, want %q", result, tt.expected)
		}
	}
}

func TestAttributeTypeString(t *testing.T) {
	tests := []struct {
		attrType AttributeType
		expected string
	}{
		{AttrMappedAddress, "MAPPED-ADDRESS"},
		{AttrXORMappedAddress, "XOR-MAPPED-ADDRESS"},
		{AttrSoftware, "SOFTWARE"},
		{AttributeType(0x9999), "Unknown (0x9999)"},
	}

	for _, tt := range tests {
		result := tt.attrType.String()
		if result != tt.expected {
			t.Errorf("AttributeType.String() = %q, want %q", result, tt.expected)
		}
	}
}

func TestDecodeInvalidMessage(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "too short",
			data: []byte{0x00, 0x01},
		},
		{
			name: "invalid magic cookie",
			data: []byte{
				0x00, 0x01, // Type
				0x00, 0x00, // Length
				0xFF, 0xFF, 0xFF, 0xFF, // Invalid magic cookie
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Transaction ID
				0x00, 0x00, 0x00, 0x00,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode(tt.data)
			if err == nil {
				t.Error("expected error for invalid message")
			}
		})
	}
}

func TestEndpointString(t *testing.T) {
	endpoint := &Endpoint{
		LocalAddr:  &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 12345},
		PublicAddr: &net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 54321},
		ServerAddr: &net.UDPAddr{IP: net.ParseIP("stun.example.com"), Port: 19302},
	}

	result := endpoint.String()
	if result == "" {
		t.Error("Endpoint.String() returned empty string")
	}

	// Should contain all addresses
	if !bytes.Contains([]byte(result), []byte("192.168.1.100")) {
		t.Error("result should contain local address")
	}
	if !bytes.Contains([]byte(result), []byte("203.0.113.1")) {
		t.Error("result should contain public address")
	}
}
