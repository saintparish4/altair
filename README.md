# STUN Client & UDP Hole Punching Implementation

A NAT traversal toolkit implementing STUN client (RFC 5389) and UDP hole punching for peer-to-peer connections.

## Overview

This project implements two critical components for NAT traversal:

**Layer 1: STUN Client** - Discovers public IP and port through STUN servers  

**Layer 2: UDP Hole Punching** - Establishes direct P2P connections through NAT

Perfect for learning NAT traversal, building P2P applications, or understanding WebRTC internals.

## Features

### Layer 1: STUN Client

- ✅ RFC 5389 compliant STUN implementation

- ✅ XOR-MAPPED-ADDRESS attribute decoding

- ✅ IPv4 and IPv6 support

- ✅ Cryptographically secure transaction IDs

- ✅ Comprehensive unit tests

### Layer 2: UDP Hole Punching

- ✅ Direct peer-to-peer UDP connections

- ✅ Simultaneous packet exchange

- ✅ Retry logic with exponential backoff

- ✅ PING/PONG connection validation

- ✅ Works through most NAT types

- ✅ Production-ready error handling

## Quick Start

### Discover Your Public Endpoint

```bash

$ altair discover

✓ Discovered public endpoint: 203.0.113.1:12345

```

### Establish P2P Connection

**Peer A:**

```bash

$ altair connect --peer 198.51.100.5:54321 --initiator

```

**Peer B:**

```bash  

$ altair connect --peer 203.0.113.1:12345

```

Both peers will establish a direct UDP connection through their NATs!

## Project Structure

```

altair/

├── cmd/

│   └── altair/

│       ├── main.go          # CLI application

│       └── connect.go       # P2P connect command

├── pkg/

│   ├── stun/

│   │   ├── client.go        # STUN client (Layer 1)

│   │   └── client_test.go   # STUN tests

│   ├── holepunch/

│   │   ├── punch.go         # Hole punching core (Layer 2)

│   │   ├── connection.go    # Connection manager

│   │   └── punch_test.go    # Hole punching tests

│   └── types/

│       └── types.go         # Common types

├── examples/

│   └── xor-demo.go          # XOR encoding demo

├── LAYER2_GUIDE.md          # Detailed Layer 2 docs

├── QUICK_REFERENCE.md       # STUN protocol reference

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

cd altair 

# Initialize Go module (if not already done)

go mod init github.com/yourusername/altair

# Build the CLI

go build -o altair ./cmd/altair

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

### As a Library

```go

package main

import (

    "fmt"

    "log"

    

    "github.com/saintparish4/altair/pkg/stun"

)

func main() {

    // Create STUN client

    client := stun.NewClient("stun.l.google.com:19302")

    

    // Discover public endpoint

    endpoint, err := client.Discover()

    if err != nil {

        log.Fatalf("Discovery failed: %v", err)

    }

    

    fmt.Printf("Public IP: %s\n", endpoint.IP)

    fmt.Printf("Public Port: %d\n", endpoint.Port)

}

```

## Testing

The implementation includes comprehensive unit tests covering:

- ✅ STUN message header construction

- ✅ XOR-MAPPED-ADDRESS encoding/decoding (IPv4)

- ✅ Transaction ID validation

- ✅ Message parsing with various edge cases

Run tests:

```bash

go test ./pkg/stun -v

```

Test output:

```

=== RUN   TestBuildBindingRequest

--- PASS: TestBuildBindingRequest (0.00s)

=== RUN   TestDecodeXORMappedAddress_IPv4

--- PASS: TestDecodeXORMappedAddress_IPv4 (0.00s)

=== RUN   TestDecodeXORMappedAddress_GoogleExample

--- PASS: TestDecodeXORMappedAddress_GoogleExample (0.00s)

=== RUN   TestParseBindingResponse

--- PASS: TestParseBindingResponse (0.00s)

=== RUN   TestParseBindingResponse_WrongTransactionID

--- PASS: TestParseBindingResponse_WrongTransactionID (0.00s)

=== RUN   TestBytesEqual

--- PASS: TestBytesEqual (0.00s)

PASS

```

## Public STUN Servers

Common public STUN servers you can use:

- `stun.l.google.com:19302` (Google)

- `stun1.l.google.com:19302` (Google)

- `stun2.l.google.com:19302` (Google)

- `stun.ekiga.net:3478` (Ekiga)

- `stun.freeswitch.org:3478` (FreeSWITCH)

- `stun.voip.blackberry.com:3478` (BlackBerry)

## Error Handling

The implementation includes proper error handling for:

- Network connection failures

- DNS resolution errors

- Timeout handling (5-second default)

- Invalid STUN responses

- Transaction ID mismatches

- Incomplete or malformed messages

## Limitations

This is a minimal implementation focused on basic STUN Binding requests. It does not include:

- STUN authentication (MESSAGE-INTEGRITY)

- TURN (relay) functionality

- ICE (Interactive Connectivity Establishment)

- TCP/TLS support (UDP only)

- Retry logic with exponential backoff

- Multiple STUN server fallback

## References

- [RFC 5389: Session Traversal Utilities for NAT (STUN)](https://datatracker.ietf.org/doc/html/rfc5389)

- [RFC 8489: Session Traversal Utilities for NAT (STUN) - Updated](https://datatracker.ietf.org/doc/html/rfc8489)

- [RFC 8445: Interactive Connectivity Establishment (ICE)](https://datatracker.ietf.org/doc/html/rfc8445)

## License

This is an educational implementation for learning purposes.

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
