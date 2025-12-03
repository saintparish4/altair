package stun

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestBuildBindingRequest(t *testing.T) {
	transactionID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c}

	request := buildBindingRequest(transactionID)

	// Verify message header
	if len(request) != messageHeaderSize {
		t.Fatalf("expected header size %d, got %d", messageHeaderSize, len(request))
	}

	// Check message type (Binding Request = 0x0001)
	msgType := binary.BigEndian.Uint16(request[0:2])
	if msgType != bindingRequest {
		t.Errorf("expected message type 0x%04x, got 0x%04x", bindingRequest, msgType)
	}

	// Check message length (should be 0 for no attributes)
	msgLen := binary.BigEndian.Uint16(request[2:4])
	if msgLen != 0 {
		t.Errorf("expected message length 0, got %d", msgLen)
	}

	// Check magic cookie
	cookie := binary.BigEndian.Uint32(request[4:8])
	if cookie != magicCookie {
		t.Errorf("expected magic cookie 0x%08x, got 0x%08x", magicCookie, cookie)
	}

	// Check transaction ID
	if !bytes.Equal(request[8:20], transactionID) {
		t.Errorf("transaction ID mismatch")
	}
}

func TestDecodeXORMappedAddress_IPv4(t *testing.T) {
	// Test data: IP 192.168.1.100, Port 5000
	// XOR encoding (Magic Cookie = 0x2112A442):
	// Port: 5000 ^ (0x2112A442 >> 16) = 5000 ^ 0x2112 = 12954 (0x329A)
	// IP: 192.168.1.100 (0xC0A80164) ^ 0x2112A442 = 0xE1BAA526

	transactionID := make([]byte, 12)

	// Build XOR-MAPPED-ADDRESS attribute value
	value := make([]byte, 8)
	value[0] = 0x00 // Reserved
	value[1] = familyIPv4
	binary.BigEndian.PutUint16(value[2:4], 0x329A)     // X-Port (12954)
	binary.BigEndian.PutUint32(value[4:8], 0xE1BAA526) // X-Address

	endpoint, err := decodeXORMappedAddress(value, transactionID)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	expectedIP := "192.168.1.100"
	expectedPort := 5000

	if endpoint.IP != expectedIP {
		t.Errorf("expected IP %s, got %s", expectedIP, endpoint.IP)
	}

	if endpoint.Port != expectedPort {
		t.Errorf("expected port %d, got %d", expectedPort, endpoint.Port)
	}
}

func TestDecodeXORMappedAddress_GoogleExample(t *testing.T) {
	// Real example from Google STUN response
	// Public IP: 203.0.113.1, Port: 54321
	// Magic Cookie: 0x2112A442

	transactionID := make([]byte, 12)

	// XOR calculations:
	// X-Port = 54321 ^ (0x2112 >> 16) = 54321 ^ 0x2112 = 62803 (0xF553)
	// For reference, 0x2112 = 8466
	xorPort := uint16(54321 ^ 0x2112)

	// X-Address = 203.0.113.1 (0xCB007101) ^ 0x2112A442 = 0xEA12D543
	xorAddr := uint32(0xCB007101 ^ 0x2112A442)

	value := make([]byte, 8)
	value[0] = 0x00
	value[1] = familyIPv4
	binary.BigEndian.PutUint16(value[2:4], xorPort)
	binary.BigEndian.PutUint32(value[4:8], xorAddr)

	endpoint, err := decodeXORMappedAddress(value, transactionID)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if endpoint.IP != "203.0.113.1" {
		t.Errorf("expected IP 203.0.113.1, got %s", endpoint.IP)
	}

	if endpoint.Port != 54321 {
		t.Errorf("expected port 54321, got %d", endpoint.Port)
	}
}

func TestParseBindingResponse(t *testing.T) {
	transactionID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c}

	// Build a mock Binding Response with XOR-MAPPED-ADDRESS attribute
	// Header (20 bytes)
	response := make([]byte, 20)
	binary.BigEndian.PutUint16(response[0:2], bindingResponse)
	binary.BigEndian.PutUint16(response[2:4], 12) // Attribute length (4 header + 8 value)
	binary.BigEndian.PutUint32(response[4:8], magicCookie)
	copy(response[8:20], transactionID)

	// XOR-MAPPED-ADDRESS attribute (12 bytes: 4 header + 8 value)
	attr := make([]byte, 12)
	binary.BigEndian.PutUint16(attr[0:2], xorMappedAddress) // Attribute type
	binary.BigEndian.PutUint16(attr[2:4], 8)                // Attribute length
	attr[4] = 0x00                                          // Reserved
	attr[5] = familyIPv4                                    // Family

	// IP: 192.0.2.1 (0xC0000201), Port: 32768
	// X-Port: 32768 ^ (0x2112A442 >> 16) = 32768 ^ 0x2112 = 41234 (0xA112)
	// X-Address: 0xC0000201 ^ 0x2112A442 = 0xE112A643
	binary.BigEndian.PutUint16(attr[6:8], 0xA112)
	binary.BigEndian.PutUint32(attr[8:12], 0xE112A643)

	response = append(response, attr...)

	endpoint, err := parseBindingResponse(response, transactionID)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if endpoint.IP != "192.0.2.1" {
		t.Errorf("expected IP 192.0.2.1, got %s", endpoint.IP)
	}

	if endpoint.Port != 32768 {
		t.Errorf("expected port 32768, got %d", endpoint.Port)
	}
}

func TestParseBindingResponse_WrongTransactionID(t *testing.T) {
	transactionID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c}
	wrongTransactionID := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

	// Build minimal response with wrong transaction ID
	response := make([]byte, 20)
	binary.BigEndian.PutUint16(response[0:2], bindingResponse)
	binary.BigEndian.PutUint16(response[2:4], 0)
	binary.BigEndian.PutUint32(response[4:8], magicCookie)
	copy(response[8:20], wrongTransactionID)

	_, err := parseBindingResponse(response, transactionID)
	if err == nil {
		t.Fatal("expected error for mismatched transaction ID")
	}
}

func TestBytesEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        []byte
		b        []byte
		expected bool
	}{
		{"equal", []byte{1, 2, 3}, []byte{1, 2, 3}, true},
		{"not equal", []byte{1, 2, 3}, []byte{1, 2, 4}, false},
		{"different length", []byte{1, 2}, []byte{1, 2, 3}, false},
		{"both empty", []byte{}, []byte{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bytesEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
