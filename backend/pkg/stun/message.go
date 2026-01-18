package stun

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

// MessageType represents STUn message type
type MessageType uint16

const (
	// Request types
	TypeBindingRequest MessageType = 0x0001

	// Success response types
	TypeBindingSuccess MessageType = 0x0101

	// Error response types
	TypeBindingError MessageType = 0x0111
)

// AttributeType represents STUN attribute type
type AttributeType uint16

const (
	AttrMappedAddress     AttributeType = 0x0001 // MAPPED-ADDRESS
	AttrXORMappedAddress  AttributeType = 0x0020 // XOR-MAPPED-ADDRESS
	AttrUsername          AttributeType = 0x0006 // USERNAME
	AttrMessageIntegrity  AttributeType = 0x0008 // MESSAGE-INTEGRITY
	AttrErrorCode         AttributeType = 0x0009 // ERROR-CODE
	AttrUnknownAttributes AttributeType = 0x000A // UNKNOWN-ATTRIBUTES
	AttrRealm             AttributeType = 0x0014 // REALM
	AttrNonce             AttributeType = 0x0015 // NONCE
	AttrSoftware          AttributeType = 0x8022 // SOFTWARE
	AttrAlternateServer   AttributeType = 0x8023 // ALTERNATE-SERVER
	AttrFingerprint       AttributeType = 0x8028 // FINGERPRINT
)

const (
	// Magic cookie
	MagicCookie uint32 = 0x2112A442

	// Header size in bytes
	HeaderSize = 20

	// Transaction ID size in bytes
	TransactionIDSize = 12

	// IPv4 address family
	FamilyIPv4 uint16 = 0x01

	// IPv6 address family
	FamilyIPv6 uint16 = 0x02
)

// Message represents a STUN message
type Message struct {
	Type          MessageType
	TransactionID [TransactionIDSize]byte
	Attributes    []Attribute
}

// Attribute represents a STUN attribute
type Attribute struct {
	Type   AttributeType
	Length uint16
	Value  []byte
}

// NewMessage creates a new STUN message with a random transaction ID
func NewMessage(msgType MessageType) (*Message, error) {
	msg := &Message{
		Type:       msgType,
		Attributes: make([]Attribute, 0),
	}

	// Generate random transaction ID
	if _, err := rand.Read(msg.TransactionID[:]); err != nil {
		return nil, fmt.Errorf("failed to generate transaction ID: %w", err)
	}

	return msg, nil
}

// AddAttribute adds an attribute to the message
func (m *Message) AddAttribute(attr Attribute) {
	m.Attributes = append(m.Attributes, attr)
}

// GetAttribute retrieves the first attribute of the given type
func (m *Message) GetAttribute(attrType AttributeType) (*Attribute, bool) {
	for i := range m.Attributes {
		if m.Attributes[i].Type == attrType {
			return &m.Attributes[i], true
		}
	}
	return nil, false
}

// Encode encodes thr STUN message to wire format
func (m *Message) Encode() ([]byte, error) {
	// Calculate message length
	msgLength := 0
	for _, attr := range m.Attributes {
		// Attribute header (4 bytes) + value + padding
		attrLen := 4 + int(attr.Length)
		// Attributes must be passed to 4-byte boundaries
		if pad := attrLen % 4; pad != 0 {
			attrLen += 4 - pad
		}
		msgLength += attrLen
	}

	// Allocate buffer: header (20 bytes) + attributes
	buf := make([]byte, HeaderSize+msgLength)

	// Encode header
	binary.BigEndian.PutUint16(buf[0:2], uint16(m.Type))
	binary.BigEndian.PutUint16(buf[2:4], uint16(msgLength))
	binary.BigEndian.PutUint32(buf[4:8], MagicCookie)
	copy(buf[8:20], m.TransactionID[:])

	// Encode attributes
	offset := HeaderSize
	for _, attr := range m.Attributes {
		binary.BigEndian.PutUint16(buf[offset:offset+2], uint16(attr.Type))
		binary.BigEndian.PutUint16(buf[offset+2:offset+4], uint16(attr.Length))
		copy(buf[offset+4:offset+4+int(attr.Length)], attr.Value)

		// Move to next attribute (with padding)
		attrLen := 4 + int(attr.Length)
		if pad := attrLen % 4; pad != 0 {
			attrLen += 4 - pad
		}
		offset += attrLen
	}

	return buf, nil
}

// Decode decodes a STUN message from wire format
func Decode(data []byte) (*Message, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("message too short: %d bytes", len(data))
	}

	// Decode header
	msg := &Message{
		Type: MessageType(binary.BigEndian.Uint16(data[0:2])),
	}

	msgLength := binary.BigEndian.Uint16(data[2:4])
	magicCookie := binary.BigEndian.Uint32(data[4:8])

	// Verify magic cookie
	if magicCookie != MagicCookie {
		return nil, fmt.Errorf("invalid magic cookie: 0x%X", magicCookie)
	}

	// Copy transaction ID
	copy(msg.TransactionID[:], data[8:20])

	// Verify message length
	if len(data) < HeaderSize+int(msgLength) {
		return nil, fmt.Errorf("incomplete message: expected %d bytes, got %d", HeaderSize+int(msgLength), len(data))
	}

	// Decode attributes
	offset := HeaderSize
	for offset < HeaderSize+int(msgLength) {
		if offset+4 > len(data) {
			return nil, fmt.Errorf("incomplete attribute header at offset %d", offset)
		}

		attr := Attribute{
			Type:   AttributeType(binary.BigEndian.Uint16(data[offset : offset+2])),
			Length: binary.BigEndian.Uint16(data[offset+2 : offset+4]),
		}

		if offset+4+int(attr.Length) > len(data) {
			return nil, fmt.Errorf("incomplete attribute value at offset %d", offset)
		}

		attr.Value = make([]byte, attr.Length)
		copy(attr.Value, data[offset+4:offset+4+int(attr.Length)])

		msg.AddAttribute(attr)

		// Move to next attribute (with padding)
		attrLen := 4 + int(attr.Length)
		if pad := attrLen % 4; pad != 0 {
			attrLen += 4 - pad
		}
		offset += attrLen
	}

	return msg, nil
}

// String returns a human-readable representation of the message type
func (t MessageType) String() string {
	switch t {
	case TypeBindingRequest:
		return "Binding Request"
	case TypeBindingSuccess:
		return "Binding Success Response"
	case TypeBindingError:
		return "Binding Error Response"
	default:
		return fmt.Sprintf("Unknown (0x%04X)", uint16(t))
	}
}

// String returns a human-readable representation of the attribute type
func (t AttributeType) String() string {
	switch t {
	case AttrMappedAddress:
		return "MAPPED-ADDRESS"
	case AttrXORMappedAddress:
		return "XOR-MAPPED-ADDRESS"
	case AttrUsername:
		return "USERNAME"
	case AttrMessageIntegrity:
		return "MESSAGE-INTEGRITY"
	case AttrErrorCode:
		return "ERROR-CODE"
	case AttrUnknownAttributes:
		return "UNKNOWN-ATTRIBUTES"
	case AttrRealm:
		return "REALM"
	case AttrNonce:
		return "NONCE"
	case AttrSoftware:
		return "SOFTWARE"
	case AttrAlternateServer:
		return "ALTERNATE-SERVER"
	case AttrFingerprint:
		return "FINGERPRINT"
	default:
		return fmt.Sprintf("Unknown (0x%04X)", uint16(t))
	}
}
