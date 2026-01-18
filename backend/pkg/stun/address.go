package stun

import (
	"encoding/binary"
	"fmt"
	"net"
)

// DecodeXORMappedAddress decodes an XOR-MAPPED-ADDRESS attribute
func DecodeXORMappedAddress(attr *Attribute, transactionID [TransactionIDSize]byte) (*net.UDPAddr, error) {
	if attr.Type != AttrXORMappedAddress {
		return nil, fmt.Errorf("attribute is not XOR-MAPPED-ADDRESS")
	}

	if len(attr.Value) < 4 {
		return nil, fmt.Errorf("XOR-MAPPED-ADDRESS value too short: %d bytes", len(attr.Value))
	}

	// Read family
	family := binary.BigEndian.Uint16(attr.Value[0:2])
	xorPort := binary.BigEndian.Uint16(attr.Value[2:4])

	// XOR port with most significant 16 bits of magic cookie
	port := xorPort ^ uint16(MagicCookie>>16)

	var ip net.IP

	switch family {
	case FamilyIPv4:
		if len(attr.Value) < 8 {
			return nil, fmt.Errorf("IPv4 address too short: %d bytes", len(attr.Value))
		}

		// Read XOR'd address
		xorAddr := binary.BigEndian.Uint32(attr.Value[4:8])

		//XOR with magic cookie
		addr := xorAddr ^ MagicCookie

		ip = make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, addr)

	case FamilyIPv6:
		if len(attr.Value) < 20 {
			return nil, fmt.Errorf("IPv6 address too short: %d bytes", len(attr.Value))
		}

		// XOR with magic cookie (first 4 bytes) and transaction ID (remaining 12 bytes)
		xorKey := make([]byte, 16)
		binary.BigEndian.PutUint32(xorKey[0:4], MagicCookie)
		copy(xorKey[4:16], transactionID[:])

		ip = make(net.IP, 16)
		for i := 0; i < 16; i++ {
			ip[i] = attr.Value[4+i] ^ xorKey[i]
		}

	default:
		return nil, fmt.Errorf("unsupported address family: 0x%02x", family)
	}

	return &net.UDPAddr{
		IP:   ip,
		Port: int(port),
	}, nil
}

// DecodeMappedAddress decode a MAPPED-ADDRESS attribute
func DecodeMappedAddress(attr *Attribute) (*net.UDPAddr, error) {
	if attr.Type != AttrMappedAddress {
		return nil, fmt.Errorf("attribute is not MAPPED-ADDRESS")
	}

	if len(attr.Value) < 4 {
		return nil, fmt.Errorf("MAPPED-ADDRESS value too short: %d bytes", len(attr.Value))
	}

	// Read family and port
	family := binary.BigEndian.Uint16(attr.Value[0:2])
	port := binary.BigEndian.Uint16(attr.Value[2:4])

	var ip net.IP

	switch family {
	case FamilyIPv4:
		if len(attr.Value) < 8 {
			return nil, fmt.Errorf("IPv4 address too short: %d bytes", len(attr.Value))
		}

		ip = make(net.IP, 4)
		copy(ip, attr.Value[4:8])

	case FamilyIPv6:
		if len(attr.Value) < 20 {
			return nil, fmt.Errorf("IPv6 address too short: %d bytes", len(attr.Value))
		}

		ip = make(net.IP, 16)
		copy(ip, attr.Value[4:20])

	default:
		return nil, fmt.Errorf("unsupported address family: 0x%04X", family)
	}

	return &net.UDPAddr{
		IP:   ip,
		Port: int(port),
	}, nil
}

// EncodeXORMappedAddress creates an XOR-MAPPED-ADDRESS attribute from an address
func EncodeXORMappedAddress(addr *net.UDPAddr, transactionID [TransactionIDSize]byte) Attribute {
	var value []byte
	var family uint16

	ip := addr.IP.To4()
	if ip != nil {
		// IPv4
		family = FamilyIPv4
		value = make([]byte, 8)

		// XOR port
		xorPort := uint16(addr.Port) ^ uint16(MagicCookie>>16)
		binary.BigEndian.PutUint16(value[2:4], xorPort)

		// XOR address
		addrInt := binary.BigEndian.Uint32(ip)
		xorAddr := addrInt ^ MagicCookie
		binary.BigEndian.PutUint32(value[4:8], xorAddr)

	} else {
		// IPv6
		family = FamilyIPv6
		value = make([]byte, 20)
		ip = addr.IP.To16()

		// XOR port
		xorPort := uint16(addr.Port) ^ uint16(MagicCookie>>16)
		binary.BigEndian.PutUint16(value[2:4], xorPort)

		// XOR address with magic cookie and transaction ID
		xorKey := make([]byte, 16)
		binary.BigEndian.PutUint32(xorKey[0:4], MagicCookie)
		copy(xorKey[4:16], transactionID[:])

		for i := 0; i < 16; i++ {
			value[4+i] = ip[i] ^ xorKey[i]
		}
	}

	// Set family
	binary.BigEndian.PutUint16(value[0:2], family)

	return Attribute{
		Type:   AttrXORMappedAddress,
		Length: uint16(len(value)),
		Value:  value,
	}
}

// EncodeMappedAddress creates a MAPPED-ADDRESS attribute from an address
func EncodeMappedAddress(addr *net.UDPAddr) Attribute {
	var value []byte
	var family uint16

	ip := addr.IP.To4()
	if ip != nil {
		// IPv4
		family = FamilyIPv4
		value = make([]byte, 8)
		binary.BigEndian.PutUint16(value[2:4], uint16(addr.Port))
		copy(value[4:8], ip)
	} else {
		// IPv6
		family = FamilyIPv6
		value = make([]byte, 20)
		ip = addr.IP.To16()
		binary.BigEndian.PutUint16(value[2:4], uint16(addr.Port))
		copy(value[4:20], ip)
	}

	// Set family
	binary.BigEndian.PutUint16(value[0:2], family)

	return Attribute{
		Type:   AttrMappedAddress,
		Length: uint16(len(value)),
		Value:  value,
	}
}
