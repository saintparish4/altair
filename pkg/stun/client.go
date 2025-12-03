package stun

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/saintparish4/altair/pkg/types"
)

const (
	// STUN message constants from RFC 5389
	magicCookie         = 0x2112A442
	bindingRequest      = 0x0001
	bindingResponse     = 0x0101
	xorMappedAddress    = 0x0020
	messageHeaderSize   = 20
	transactionIDSize   = 12
	attributeHeaderSize = 4

	// Address family constants
	familyIPv4 = 0x01
	familyIPv6 = 0x02
)

// Client represents a STUN client
type Client struct {
	ServerAddr string
	Timeout    time.Duration
}

// NewClient creates a new STUN client
func NewClient(serverAddr string) *Client {
	return &Client{
		ServerAddr: serverAddr,
		Timeout:    5 * time.Second,
	}
}

// Discover performs STUN discovery to find the public endpoint
func (c *Client) Discover() (*types.Endpoint, error) {
	// Resolve STUN server address
	serverAddr, err := net.ResolveUDPAddr("udp", c.ServerAddr)
	if err != nil {
		return nil, types.NewSTUNError("resolve address", err)
	}

	// Create a UDP connection
	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return nil, types.NewSTUNError("dial", err)
	}
	defer conn.Close()

	// Set deadline
	if err := conn.SetDeadline(time.Now().Add(c.Timeout)); err != nil {
		return nil, types.NewSTUNError("set_deadline", err)
	}

	// Generate transaction ID
	transactionID := make([]byte, transactionIDSize)
	if _, err := rand.Read(transactionID); err != nil {
		return nil, types.NewSTUNError("generate_transaction_id", err)
	}

	// Build and send Binding Request
	request := buildBindingRequest(transactionID)
	if _, err := conn.Write(request); err != nil {
		return nil, types.NewSTUNError("send_request", err)
	}

	// Read response
	response := make([]byte, 1500) // MTU size
	n, err := conn.Read(response)
	if err != nil {
		return nil, types.NewSTUNError("read_response", err)
	}

	// Parse response
	endpoint, err := parseBindingResponse(response[:n], transactionID)
	if err != nil {
		return nil, types.NewSTUNError("parse_response", err)
	}

	return endpoint, nil
}

// buildBindingRequest creates a STUN Binding Request message
func buildBindingRequest(transactionID []byte) []byte {
	// STUN message header (20 bytes):
	// 0-1: Message Type
	// 2-3: Message Length (0 for no attributes)
	// 4-7: Magic Cookie
	// 8-19: Transaction ID

	msg := make([]byte, messageHeaderSize)

	// Message Type (Binding Request = 0x0001)
	binary.BigEndian.PutUint16(msg[0:2], bindingRequest)

	// Message Length (0 for no attributes)
	binary.BigEndian.PutUint16(msg[2:4], 0)

	// Magic Cookie
	binary.BigEndian.PutUint32(msg[4:8], magicCookie)

	// Transaction ID (12 bytes)
	copy(msg[8:20], transactionID)

	return msg
}

// parseBindingResponse parses a STUN Binding Response and extracts the XOR-MAPPED-ADDRESS
func parseBindingResponse(response []byte, expectedTransactionID []byte) (*types.Endpoint, error) {
	// Validate minimum message size
	if len(response) < messageHeaderSize {
		return nil, fmt.Errorf("response too short: %d bytes", len(response))
	}

	// Parse header
	messageType := binary.BigEndian.Uint16(response[0:2])
	messageLength := binary.BigEndian.Uint16(response[2:4])
	receivedMagicCookie := binary.BigEndian.Uint32(response[4:8])
	receivedTransactionID := response[8:20]

	// Verify this is a Binding Response
	if messageType != bindingResponse {
		return nil, fmt.Errorf("unexpected message type: 0x%04x (expected 0x%04x)", messageType, bindingResponse)
	}

	// Verify magic cookie
	if receivedMagicCookie != magicCookie {
		return nil, fmt.Errorf("invalid magic cookie: 0x%08x", receivedMagicCookie)
	}

	// Verify transaction ID matches
	if !bytesEqual(receivedTransactionID, expectedTransactionID) {
		return nil, fmt.Errorf("transaction ID mismatch")
	}

	// Verify message length
	if len(response) < messageHeaderSize+int(messageLength) {
		return nil, fmt.Errorf("incomplete message: got %d bytes, expected %d", len(response), messageHeaderSize+int(messageLength))
	}

	// Parse attributes
	payload := response[messageHeaderSize : messageHeaderSize+int(messageLength)]
	endpoint, err := parseAttributes(payload, receivedTransactionID)
	if err != nil {
		return nil, fmt.Errorf("parse attributes: %w", err)
	}

	if endpoint == nil {
		return nil, fmt.Errorf("XOR-MAPPED-ADDRESS attribute not found")
	}

	return endpoint, nil
}

// parseAttributes parses STUN attributes and extracts XOR-MAPPED-ADDRESS
func parseAttributes(payload []byte, transactionID []byte) (*types.Endpoint, error) {
	pos := 0

	for pos < len(payload) {
		// Need at least 4 bytes for attribute header
		if pos+attributeHeaderSize > len(payload) {
			break
		}

		// Parse attribute header
		attrType := binary.BigEndian.Uint16(payload[pos : pos+2])
		attrLength := binary.BigEndian.Uint16(payload[pos+2 : pos+4])
		pos += attributeHeaderSize

		// Verify we have enough data for attribute value
		if pos+int(attrLength) > len(payload) {
			return nil, fmt.Errorf("incomplete attribute: type=0x%04x, length=%d", attrType, attrLength)
		}

		// Extract attribute value
		attrValue := payload[pos : pos+int(attrLength)]

		// Check if this is XOR-MAPPED-ADDRESS
		if attrType == xorMappedAddress {
			endpoint, err := decodeXORMappedAddress(attrValue, transactionID)
			if err != nil {
				return nil, fmt.Errorf("decode XOR-MAPPED-ADDRESS: %w", err)
			}
			return endpoint, nil
		}

		// Move to next attribute (attributes are padded to 4-byte boundaries)
		pos += int(attrLength)
		// Add padding to align to 4-byte boundary
		if pad := int(attrLength) % 4; pad != 0 {
			pos += 4 - pad
		}
	}

	return nil, nil
}

// decodeXORMappedAddress decodes the XOR-MAPPED-ADDRESS attribute
func decodeXORMappedAddress(value []byte, transactionID []byte) (*types.Endpoint, error) {
	// XOR-MAPPED-ADDRESS format (RFC 5389 Section 15.2):
	// 0: Reserved (8 bits)
	// 1: Family (8 bits) - 0x01=IPv4, 0x02=IPv6
	// 2-3: X-Port (16 bits)
	// 4-7 (IPv4) or 4-19 (IPv6): X-Address

	if len(value) < 4 {
		return nil, fmt.Errorf("value too short: %d bytes", len(value))
	}

	// Parse family
	family := value[1]

	// Parse X-Port (XORed port)
	xorPort := binary.BigEndian.Uint16(value[2:4])

	// Decode port: XOR with most significant 16 bits of magic cookie
	port := int(xorPort ^ uint16(magicCookie>>16))

	var ip string

	switch family {
	case familyIPv4:
		if len(value) < 8 {
			return nil, fmt.Errorf("IPv4 address too short: %d bytes", len(value))
		}

		// Parse X-Address (XORed IPv4 address)
		xorAddr := binary.BigEndian.Uint32(value[4:8])

		// Decode address: XOR with magic cookie
		addr := xorAddr ^ magicCookie

		// Convert to IP string
		ip = fmt.Sprintf("%d.%d.%d.%d",
			byte(addr>>24),
			byte(addr>>16),
			byte(addr>>8),
			byte(addr))

	case familyIPv6:
		if len(value) < 20 {
			return nil, fmt.Errorf("IPv6 address too short: %d bytes", len(value))
		}

		// For IPv6, XOR with magic cookie (4 bytes) + transaction ID (12 bytes)
		xorKey := make([]byte, 16)
		binary.BigEndian.PutUint32(xorKey[0:4], magicCookie)
		copy(xorKey[4:16], transactionID)

		// XOR the address
		addr := make([]byte, 16)
		for i := 0; i < 16; i++ {
			addr[i] = value[4+i] ^ xorKey[i]
		}

		// Convert to IP
		ipAddr := net.IP(addr)
		ip = ipAddr.String()

	default:
		return nil, fmt.Errorf("unsupported address family: 0x%02x", family)
	}

	return &types.Endpoint{
		IP:   ip,
		Port: port,
	}, nil
}

// bytesEqual compares two byte slices
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
