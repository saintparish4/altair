# STUN Client Implementation

A minimal STUN (Session Traversal Utilities for NAT) client implementation in Go, following RFC 5389.

## Overview

This project implements a basic STUN client that can discover your public IP address and port by querying a STUN server. This is useful for NAT traversal in peer-to-peer applications.

## Features

- ✅ STUN Binding Request/Response implementation
- ✅ XOR-MAPPED-ADDRESS attribute decoding
- ✅ IPv4 and IPv6 support
- ✅ RFC 5389 compliant
- ✅ Comprehensive unit tests
- ✅ Simple CLI interface

## Project Structure

```
punchthrough/
├── cmd/
│   └── punchthrough/
│       └── main.go           # CLI application
├── pkg/
│   ├── stun/
│   │   ├── client.go         # STUN client implementation
│   │   └── client_test.go    # Unit tests
│   └── types/
│       └── types.go          # Common types and errors
├── go.mod
└── README.md
```

## STUN Protocol Implementation Details

### Message Format (RFC 5389)

All STUN messages start with a 20-byte header:

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|0 0|     STUN Message Type     |         Message Length        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         Magic Cookie                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|                     Transaction ID (96 bits)                  |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

**Header Fields:**
- **Message Type** (2 bytes): 0x0001 for Binding Request, 0x0101 for Binding Response
- **Message Length** (2 bytes): Length of message payload (excluding 20-byte header)
- **Magic Cookie** (4 bytes): Fixed value 0x2112A442
- **Transaction ID** (12 bytes): Random identifier to match requests with responses

### XOR-MAPPED-ADDRESS Attribute

The XOR-MAPPED-ADDRESS attribute contains the client's public IP and port, encoded using XOR obfuscation to prevent NAT ALGs from modifying the payload.

**Format:**
```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|x x x x x x x x|    Family     |           X-Port              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                X-Address (Variable)                           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

**XOR Encoding:**
- **X-Port**: `port XOR (magic_cookie >> 16)`
- **X-Address (IPv4)**: `ip_address XOR magic_cookie`
- **X-Address (IPv6)**: `ip_address XOR (magic_cookie || transaction_id)`

### Implementation Flow

1. **Generate Transaction ID**: Create a random 96-bit (12-byte) transaction ID
2. **Build Binding Request**: Create a 20-byte STUN message header with:
   - Message Type: 0x0001 (Binding Request)
   - Message Length: 0 (no attributes in request)
   - Magic Cookie: 0x2112A442
   - Transaction ID: Random 12 bytes
3. **Send Request**: Send via UDP to STUN server
4. **Receive Response**: Read response and validate:
   - Message Type is 0x0101 (Binding Response)
   - Magic Cookie matches
   - Transaction ID matches
5. **Parse Attributes**: Extract XOR-MAPPED-ADDRESS attribute
6. **Decode Address**: XOR the port and IP address to get actual values

## Building

```bash
# Clone or create the project
cd punchthrough

# Initialize Go module (if not already done)
go mod init github.com/yourusername/punchthrough

# Build the CLI
go build -o punchthrough ./cmd/punchthrough

# Run tests
go test ./pkg/stun -v
```

## Usage

### CLI

```bash
# Discover your public endpoint using default STUN server (Google)
./altair discover

# Use a custom STUN server
STUN_SERVER=stun.ekiga.net:3478 ./altair discover

# Show help
./altair help
```

## Future Enhancements

Potential improvements for a production system:

- [ ] Add TURN support for relay
- [ ] Implement ICE for full NAT traversal
- [ ] Add authentication support
- [ ] TCP/TLS transport support
- [ ] Automatic server fallback
- [ ] Better timeout and retry logic
- [ ] Support for more STUN attributes
- [ ] Connection pooling
- [ ] Metrics and logging