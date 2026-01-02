# Layer 2: UDP Hole Punching - Implementation Guide

## Overview

Layer 2 implements UDP hole punching to establish direct peer-to-peer connections through NAT. This allows two peers behind different NATs to communicate directly without a relay server.

## How UDP Hole Punching Works

### The Problem: NAT Traversal

When a device behind a NAT sends a UDP packet to an external address, the NAT:
1. Creates a temporary mapping: `(internal_ip:internal_port) → (public_ip:public_port)`
2. Forwards the packet using the public endpoint
3. Allows responses from that specific external endpoint back through the NAT

The challenge: Each NAT only allows packets from endpoints the device has sent to.

### The Solution: Simultaneous Exchange

UDP hole punching solves this by having both peers send packets simultaneously:

```
Peer A (behind NAT A)          Peer B (behind NAT B)
└─ Private: 192.168.1.100:5000 └─ Private: 10.0.0.50:6000
└─ Public: 203.0.113.1:12345   └─ Public: 198.51.100.5:54321

Step 1: Both peers discover their public endpoints via STUN
Step 2: Peers exchange public endpoints (manually for now)
Step 3: Both peers send to each other's public endpoint simultaneously

Peer A sends to 198.51.100.5:54321 ───┐
└─ NAT A creates hole for responses   │
   from 198.51.100.5:54321            │
                                      │
                                      └──> Packet reaches NAT B
                                           └─ Allowed through because
                                              Peer B already sent to
                                              203.0.113.1:12345
                                              
Peer B sends to 203.0.113.1:12345 ───┐
└─ NAT B creates hole for responses   │
   from 203.0.113.1:12345             │
                                      │
                                      └──> Packet reaches NAT A
                                           └─ Allowed through because
                                              Peer A already sent to
                                              198.51.100.5:54321
```

Once both holes are punched, direct bidirectional communication is established!

## Implementation Components

### 1. Hole Punching Core (`pkg/holepunch/punch.go`)

Core functions for UDP operations:

- **`PrepareLocalEndpoint()`**: Binds to a random UDP port
  - Returns UDP connection and local endpoint
  - Used to prepare for hole punching

- **`SendPunch()`**: Sends a punch packet to remote endpoint
  - Packet content: "PUNCH"
  - Creates NAT mapping when sent

- **`ListenForPunch()`**: Waits for incoming punch packets
  - Accepts packets from any remote endpoint
  - Returns sender's endpoint and message

- **`SimultaneousPunch()`**: Orchestrates the hole punching
  - Sends multiple punch packets (retry logic)
  - Listens for incoming packets concurrently
  - Returns when first valid packet is received

- **`SendMessage()` / `ReceiveMessage()`**: General message exchange
  - Used after connection is established
  - Support for PING/PONG testing

### 2. Connection Manager (`pkg/holepunch/connection.go`)

High-level connection establishment:

- **`EstablishConnection(localEP, remoteEP)`**: Main connection function
  - Configures local UDP socket
  - Performs hole punching with retry logic
  - Returns established UDP connection

- **`ConnectionConfig`**: Configuration structure
  - `Attempts`: Number of punch attempts (default: 5)
  - `Interval`: Time between attempts (default: 400ms)
  - `Timeout`: Overall timeout (default: 10s)

- **`PingPong()`**: Connection validation
  - Initiator sends PING, waits for PONG
  - Responder waits for PING, sends PONG back
  - Verifies bidirectional communication

### 3. CLI Command (`cmd/main/connect.go`)

User-facing command for establishing connections:

```bash
altair connect --peer IP:PORT [options]
```

**Options:**
- `--peer`: Remote peer's public endpoint (required)
- `--initiator`: This peer sends PING first
- `--stun`: STUN server for discovery
- `--skip-test`: Skip PING/PONG test

**Workflow:**
1. Discovers own public endpoint via STUN
2. Displays connection information
3. Establishes P2P connection through hole punching
4. Optionally tests with PING/PONG
5. Keeps connection open

## Usage Examples

### Basic Connection Setup

**Peer A:**
```bash
# Step 1: Discover public endpoint
$ altair discover
✓ Discovered public endpoint: 203.0.113.1:12345

# Step 2: Connect as initiator
$ altair connect --peer 198.51.100.5:54321 --initiator
```

**Peer B:**
```bash
# Step 1: Discover public endpoint
$ altair discover
✓ Discovered public endpoint: 198.51.100.5:54321

# Step 2: Connect as responder
$ altair connect --peer 203.0.113.1:12345
```

### Connection Output

```
=== Altair P2P Connection ===

Step 1: Discovering public endpoint via STUN (stun.l.google.com:19302)...
✓ Your public endpoint: 203.0.113.1:12345

Step 2: Connection Information
  Your public IP:Port  : 203.0.113.1:12345
  Peer's public IP:Port: 198.51.100.5:54321

Step 3: Establishing P2P connection...
  This will punch through your NAT and connect directly to peer.
  Both peers should run this command simultaneously!

Attempting hole punch to 198.51.100.5:54321...
  Attempt 1/5: Sending punch packets...
✓ Hole punched successfully!

🎉 Connection established!
✓ Direct UDP connection to 198.51.100.5:54321

Step 4: Testing connection with PING-PONG...
  [You are the INITIATOR - sending PING first]

Sending PING...
Waiting for PONG...
✓ Received PONG!
✓ Round-trip successful!

✓ Connection verified and ready for data exchange!

Connection will remain open. Press Ctrl+C to close.
```

## Testing

Run comprehensive tests:

```bash
# Test all hole punching functionality
go test ./pkg/holepunch -v

# Specific tests:
go test -run TestPrepareLocalEndpoint ./pkg/holepunch
go test -run TestSendAndReceiveMessage ./pkg/holepunch
go test -run TestPingPongSimulation ./pkg/holepunch
```

Test results:
```
✓ TestPrepareLocalEndpoint
✓ TestSendAndReceiveMessage
✓ TestPunchMessage
✓ TestSendPunch
✓ TestReceiveMessageTimeout
✓ TestParseEndpoint (4 sub-tests)
✓ TestDefaultConfig
✓ TestPingPongSimulation
```

## Retry Logic

The implementation includes robust retry logic:

1. **Multiple Attempts**: Sends 5 punch packets by default
2. **Timing**: 400ms intervals (~2 seconds total)
3. **Exponential Backoff**: Wait time increases with each attempt
4. **Timeout**: 10-second overall timeout
5. **First Success Wins**: Returns immediately when any packet gets through

This handles packet loss and timing mismatches between peers.

## Limitations & Considerations

### Current Limitations (Manual Coordination)

- **Manual Endpoint Exchange**: Users must manually share public endpoints
- **Timing Requirement**: Both peers must run connect within ~10 seconds
- **No Session Persistence**: Connection doesn't survive network changes
- **Single Connection**: Only establishes one connection at a time

### NAT Type Compatibility

Works with most NAT types:
- ✅ Full Cone NAT
- ✅ Restricted Cone NAT
- ✅ Port-Restricted Cone NAT
- ⚠️ Symmetric NAT (may require relay)

### Network Requirements

- **UDP Support**: Network must allow UDP traffic
- **Firewall**: Must allow outbound UDP
- **STUN Access**: Needs to reach STUN server for discovery
- **MTU**: Uses standard 1500-byte buffer

## Future Enhancements (Layer 3 & Beyond)

Layer 3 will add:
- **Signaling Server**: Automatic endpoint exchange
- **No Manual Coordination**: Peers find each other automatically
- **Session Management**: Persistent connections

Layer 4 will add:
- **NAT Type Detection**: Identify NAT characteristics
- **Relay Fallback**: Use relay for symmetric NATs
- **Keep-Alive**: Maintain NAT bindings

## Troubleshooting

### Connection Fails

1. **Timing**: Ensure both peers run connect within 10 seconds
2. **Endpoints**: Verify public endpoints are correct
3. **Firewall**: Check UDP traffic is allowed
4. **NAT**: Some symmetric NATs require relay (coming in Layer 4)

### Timeout Errors

- Increase timeout: Modify `DefaultTimeout` in `connection.go`
- Check network connectivity
- Verify STUN server is reachable

### PING/PONG Fails

- Ensure one peer uses `--initiator` flag
- Check connection was established successfully
- Verify no firewall blocks established connection

## Architecture Decisions

### Why UDP?

- Low latency (no connection handshake)
- Works through NATs with hole punching
- Suitable for real-time communication
- Standard for P2P applications

### Why Simultaneous Exchange?

- Both NATs must have outbound mappings
- Sending simultaneously ensures holes are ready
- Handles packet loss through retries
- Industry standard approach

### Why Manual Coordination (for now)?

- Simplifies initial implementation
- Focuses on hole punching mechanics
- Prepares foundation for automatic signaling (Layer 3)
- Educational: shows the core concept clearly

## Code Examples

### Programmatic Usage

```go
package main

import (
    "fmt"
    "github.com/saintparish4/altair/pkg/holepunch"
    "github.com/saintparish4/altair/pkg/types"
)

func main() {
    // Parse peer endpoint
    remoteEP := &types.Endpoint{
        IP:   "198.51.100.5",
        Port: 54321,
    }
    
    // Establish connection
    conn, err := holepunch.EstablishConnection(nil, remoteEP)
    if err != nil {
        panic(err)
    }
    defer conn.Close()
    
    fmt.Println("Connected!")
    
    // Send message
    holepunch.SendMessage(conn, "Hello!", remoteEP)
    
    // Receive message
    msg, _, _ := holepunch.ReceiveMessage(conn, 5*time.Second)
    fmt.Println("Received:", msg)
}
```

## Performance Characteristics

- **Connection Time**: 2-10 seconds (depends on retry logic)
- **Overhead**: Minimal (UDP header + payload)
- **Latency**: Near-optimal (direct peer-to-peer)
- **Throughput**: Limited by network, not implementation
- **CPU Usage**: Very low
- **Memory**: Minimal (small buffers)

## Security Considerations

### Current Security

- **No Encryption**: Data sent in clear text
- **No Authentication**: Anyone can send packets
- **Public Endpoints**: IP addresses are shared
- **Spoofing**: Vulnerable to packet injection

### Mitigations (Future)

- Add DTLS for encryption (Layer 4+)
- Implement shared secret authentication
- Use secure signaling channel
- Add packet sequence numbers

## References

- RFC 5389: STUN Protocol
- RFC 5766: TURN Protocol  
- RFC 8445: ICE Protocol
- "Peer-to-Peer Communication Across Network Address Translators" (STUN RFC)
- WebRTC specifications

## Summary

Layer 2 successfully implements UDP hole punching with:
- ✅ Direct P2P connections through NAT
- ✅ Robust retry logic
- ✅ PING/PONG validation
- ✅ Comprehensive testing
- ✅ User-friendly CLI
- ✅ Production-quality error handling
