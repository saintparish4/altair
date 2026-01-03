# Altair - P2P NAT Traversal Library

The main Altair API package providing a simple interface for P2P connections.

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/saintparish4/altair/pkg/altair"
)

func main() {
    // Create client
    client := altair.NewClient()
    
    // Discover public endpoint
    endpoint, err := client.DiscoverPublicEndpoint()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Public endpoint: %s\n", endpoint)
}
```

## API

### Client Creation

- `NewClient()` - Create client with default configuration
- `NewClientWithConfig(config)` - Create client with custom configuration

### NAT Traversal

- `DiscoverPublicEndpoint()` - Discover your public IP and port via STUN
- `Connect(remoteEndpoint, initiator)` - Establish P2P connection
- `ConnectWithLocalEndpoint(local, remote, initiator)` - Connect with specific local endpoint

### Messaging

- `SendMessage(conn, message, endpoint)` - Send string message
- `ReceiveMessage(conn, timeout)` - Receive string message

## Configuration

```go
config := altair.Config{
    STUNServer:         "stun.l.google.com:19302",
    ConnectionAttempts: 5,
    ConnectionInterval: 400 * time.Millisecond,
    ConnectionTimeout:  10 * time.Second,
    STUNTimeout:        5 * time.Second,
}

client := altair.NewClientWithConfig(config)
```

## See Also

- [STUN Package](../stun/) - Low-level STUN implementation
- [Holepunch Package](../holepunch/) - Low-level hole punching
- [Examples](../../../examples/) - Complete examples

