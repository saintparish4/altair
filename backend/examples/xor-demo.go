package main

import (
	"encoding/binary"
	"fmt"
)

// This example demonstrates how STUN XOR-MAPPED-ADDRESS encoding works
// according to RFC 5389 Section 15.2

const (
	magicCookie = 0x2112A442
)

func main() {
	fmt.Println("=== STUN XOR-MAPPED-ADDRESS Encoding Example ===")
	fmt.Println()

	// Example 1: Simple case
	fmt.Println("Example 1: IP 192.168.1.100, Port 5000")
	fmt.Println("----------------------------------------")
	demonstrateXOR("192.168.1.100", 5000)
	fmt.Println()

	// Example 2: Public IP example
	fmt.Println("Example 2: IP 203.0.113.1, Port 54321")
	fmt.Println("----------------------------------------")
	demonstrateXOR("203.0.113.1", 54321)
	fmt.Println()

	// Example 3: Another common case
	fmt.Println("Example 3: IP 192.0.2.1, Port 32768")
	fmt.Println("----------------------------------------")
	demonstrateXOR("192.0.2.1", 32768)
	fmt.Println()

	fmt.Println("=== Why XOR Encoding? ===")
	fmt.Println("NAT Application Layer Gateways (ALGs) sometimes inspect and")
	fmt.Println("modify IP addresses in packet payloads. XOR encoding prevents")
	fmt.Println("this by obfuscating the actual IP address and port values.")
}

func demonstrateXOR(ipStr string, port int) {
	// Parse IP address
	var ipBytes [4]byte
	fmt.Sscanf(ipStr, "%d.%d.%d.%d", &ipBytes[0], &ipBytes[1], &ipBytes[2], &ipBytes[3])
	ipAddr := binary.BigEndian.Uint32(ipBytes[:])

	fmt.Printf("Original Values:\n")
	fmt.Printf("  IP:   %s (0x%08X)\n", ipStr, ipAddr)
	fmt.Printf("  Port: %d (0x%04X)\n", port, port)
	fmt.Println()

	// XOR encoding for port
	// Port is XORed with the most significant 16 bits of the magic cookie
	magicCookieHigh := uint16(magicCookie >> 16)
	xorPort := uint16(port) ^ magicCookieHigh

	fmt.Printf("Port Encoding:\n")
	fmt.Printf("  Magic Cookie (high 16 bits): 0x%04X (%d)\n", magicCookieHigh, magicCookieHigh)
	fmt.Printf("  Port:          0x%04X (%d)\n", port, port)
	fmt.Printf("  XOR:           0x%04X (%d)\n", magicCookieHigh, magicCookieHigh)
	fmt.Printf("  X-Port:        0x%04X (%d)\n", xorPort, xorPort)
	fmt.Println()

	// XOR encoding for IPv4 address
	// Address is XORed with the full magic cookie
	xorAddr := ipAddr ^ magicCookie

	fmt.Printf("Address Encoding:\n")
	fmt.Printf("  Magic Cookie:  0x%08X\n", magicCookie)
	fmt.Printf("  IP Address:    0x%08X\n", ipAddr)
	fmt.Printf("  XOR:           0x%08X\n", magicCookie)
	fmt.Printf("  X-Address:     0x%08X\n", xorAddr)

	// Show X-Address as IP format (for illustration)
	xorIP := fmt.Sprintf("%d.%d.%d.%d",
		byte(xorAddr>>24),
		byte(xorAddr>>16),
		byte(xorAddr>>8),
		byte(xorAddr))
	fmt.Printf("  X-Address (as IP): %s\n", xorIP)
	fmt.Println()

	// Demonstrate decoding
	fmt.Printf("Decoding Process:\n")

	// Decode port
	decodedPort := xorPort ^ magicCookieHigh
	fmt.Printf("  X-Port 0x%04X XOR 0x%04X = %d ✓\n", xorPort, magicCookieHigh, decodedPort)

	// Decode address
	decodedAddr := xorAddr ^ magicCookie
	decodedIP := fmt.Sprintf("%d.%d.%d.%d",
		byte(decodedAddr>>24),
		byte(decodedAddr>>16),
		byte(decodedAddr>>8),
		byte(decodedAddr))
	fmt.Printf("  X-Address 0x%08X XOR 0x%08X = %s ✓\n", xorAddr, magicCookie, decodedIP)
	fmt.Println()

	// Show the complete XOR-MAPPED-ADDRESS attribute value
	fmt.Printf("Complete XOR-MAPPED-ADDRESS Attribute:\n")
	fmt.Printf("  Byte 0:    0x00 (Reserved)\n")
	fmt.Printf("  Byte 1:    0x01 (Family: IPv4)\n")
	fmt.Printf("  Bytes 2-3: 0x%04X (X-Port)\n", xorPort)
	fmt.Printf("  Bytes 4-7: 0x%08X (X-Address)\n", xorAddr)

	// Show as hex dump
	attrValue := make([]byte, 8)
	attrValue[0] = 0x00 // Reserved
	attrValue[1] = 0x01 // IPv4
	binary.BigEndian.PutUint16(attrValue[2:4], xorPort)
	binary.BigEndian.PutUint32(attrValue[4:8], xorAddr)

	fmt.Printf("  Hex dump:  ")
	for i, b := range attrValue {
		fmt.Printf("%02X ", b)
		if i == 3 {
			fmt.Printf("\n             ")
		}
	}
	fmt.Println()
}
