# Signaling Server Documentation

The signaling server enables automatic peer discovery and coordination for establishing P2P connections through NAT. It replaces the manual endpoint exchange required in Layer 2 with an automated WebSocket-based system.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Signaling Server                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Registry  │  │    Room     │  │   WebSocket Handler     │  │
│  │             │  │   Manager   │  │                         │  │
│  │  peer_id →  │  │             │  │  - Message routing      │  │
│  │    *Peer    │  │  room_id →  │  │  - Protocol handling    │  │
│  │             │  │    *Room    │  │  - Connection lifecycle │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
│                                                                 │
│  REST API: /health, /api/stats, /api/rooms                      │
│  WebSocket: /ws                                                 │
└─────────────────────────────────────────────────────────────────┘
         │                                        │
         │ ws://                                  │ ws://
         ▼                                        ▼
    ┌─────────┐                              ┌─────────┐
    │ Peer A  │◄──────── P2P ────────────────│ Peer B  │
    └─────────┘     (after coordination)     └─────────┘
```

## Protocol

### Message Format

All messages use JSON with this envelope structure:

```json
{
  "type": "JOIN",
  "peer_id": "a1b2c3d4",
  "target_id": "e5f6g7h8",
  "room_id": "my-room",
  "payload": { ... },
  "timestamp": 1703894400000,
  "request_id": "req-123"
}
```

### Message Types

#### Client → Server

| Type | Description | Required Fields |
|------|-------------|-----------------|
| `JOIN` | Join a room | `room_id` |
| `LEAVE` | Leave current room | - |
| `DISCOVER` | List peers in room | `room_id` (optional if in room) |
| `OFFER` | Send connection offer | `target_id`, `payload` |
| `ANSWER` | Respond to offer | `target_id`, `payload` |
| `CANDIDATE` | Exchange ICE candidate | `target_id`, `payload` |
| `KEEP_ALIVE` | Keep connection alive | - |

#### Server → Client

| Type | Description |
|------|-------------|
| `PEER_JOINED` | Notification: peer joined room |
| `PEER_LEFT` | Notification: peer left room |
| `PEER_LIST` | Response to DISCOVER |
| `ERROR` | Error response |
| `ACK` | Acknowledgment |

### Connection Flow

```
Peer A                    Server                    Peer B
  │                         │                         │
  │──── JOIN room-1 ───────▶│                         │
  │◀─── ACK + peer list ────│                         │
  │                         │                         │
  │                         │◀──── JOIN room-1 ───────│
  │◀─── PEER_JOINED ────────│──── ACK + peer list ───▶│
  │                         │                         │
  │──── OFFER ─────────────▶│                         │
  │                         │──── OFFER ─────────────▶│
  │                         │                         │
  │                         │◀──── ANSWER ────────────│
  │◀──── ANSWER ────────────│                         │
  │                         │                         │
  │◄═══════════════ P2P Connection ═══════════════════│
```

## Payload Types

### JoinPayload

```json
{
  "display_name": "Alice",
  "endpoint": {
    "ip": "203.0.113.1",
    "port": 12345
  }
}
```

### OfferPayload

```json
{
  "endpoint": {
    "ip": "203.0.113.1",
    "port": 12345
  },
  "session_id": "sess-abc123",
  "initiator_id": "a1b2c3d4"
}
```

### AnswerPayload

```json
{
  "endpoint": {
    "ip": "198.51.100.1",
    "port": 54321
  },
  "session_id": "sess-abc123",
  "accepted": true
}
```

### ErrorPayload

```json
{
  "code": "PEER_NOT_FOUND",
  "message": "target peer not found",
  "details": "peer_id: xyz123"
}
```

### Error Codes

| Code | Description |
|------|-------------|
| `INVALID_MESSAGE` | Malformed or missing required fields |
| `ROOM_NOT_FOUND` | Requested room doesn't exist |
| `PEER_NOT_FOUND` | Target peer doesn't exist |
| `NOT_IN_ROOM` | Action requires being in a room |
| `ALREADY_IN_ROOM` | Already in the requested room |
| `ROOM_FULL` | Room has reached max capacity |
| `UNAUTHORIZED` | Action not permitted |
| `INTERNAL_ERROR` | Server-side error |

## REST API

### GET /health

Health check endpoint.

**Response:**
```json
{
  "status": "ok",
  "timestamp": 1703894400000
}
```

### GET /api/stats

Server statistics.

**Response:**
```json
{
  "peers": {
    "total": 42,
    "without_room": 5
  },
  "rooms": {
    "total": 8,
    "total_peers": 37
  },
  "timestamp": 1703894400000
}
```

### GET /api/rooms

List all rooms.

**Response:**
```json
{
  "rooms": [
    {
      "id": "room-1",
      "peer_count": 5,
      "max_peers": 0,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### GET /api/rooms/{id}

Get room details.

**Response:**
```json
{
  "id": "room-1",
  "peers": [
    {
      "peer_id": "a1b2c3d4",
      "display_name": "Alice",
      "endpoint": {"ip": "203.0.113.1", "port": 12345},
      "joined_at": 1703894400000
    }
  ],
  "peer_count": 1,
  "max_peers": 0,
  "created_at": 1703894400000
}
```

## Usage

### Running the Server

```bash
# Build with websocket support
go get github.com/gorilla/websocket
go build -o altair-signaling ./cmd/signaling

# Run
./altair-signaling -addr :8080 -verbose
```

### Client Example (JavaScript)

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  // Join a room
  ws.send(JSON.stringify({
    type: 'JOIN',
    room_id: 'my-room',
    payload: {
      display_name: 'Alice',
      endpoint: { ip: '203.0.113.1', port: 12345 }
    }
  }));
};

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  
  switch (msg.type) {
    case 'ACK':
      console.log('Joined room, peers:', msg.payload.peers);
      break;
    case 'PEER_JOINED':
      console.log('New peer:', msg.peer_id);
      // Send offer to new peer
      ws.send(JSON.stringify({
        type: 'OFFER',
        target_id: msg.peer_id,
        payload: {
          endpoint: myEndpoint,
          session_id: crypto.randomUUID(),
          initiator_id: myPeerId
        }
      }));
      break;
    case 'OFFER':
      console.log('Received offer from:', msg.peer_id);
      // Send answer
      ws.send(JSON.stringify({
        type: 'ANSWER',
        target_id: msg.peer_id,
        payload: {
          endpoint: myEndpoint,
          session_id: msg.payload.session_id,
          accepted: true
        }
      }));
      break;
    case 'ANSWER':
      console.log('Received answer, begin hole punching');
      break;
  }
};
```

### Client Example (Go)

```go
package main

import (
    "encoding/json"
    "log"
    
    "github.com/gorilla/websocket"
    "github.com/saintparish4/altair/internal/signaling"
)

func main() {
    conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    
    // Join room
    join := signaling.NewMessage(signaling.MessageTypeJoin).
        WithRoomID("my-room").
        WithPayload(signaling.JoinPayload{
            DisplayName: "GoPeer",
        })
    
    data, _ := json.Marshal(join)
    conn.WriteMessage(websocket.TextMessage, data)
    
    // Read messages
    for {
        _, data, err := conn.ReadMessage()
        if err != nil {
            break
        }
        
        var msg signaling.Message
        json.Unmarshal(data, &msg)
        
        switch msg.Type {
        case signaling.MessageTypeAck:
            log.Println("Connected!")
        case signaling.MessageTypePeerJoined:
            log.Printf("Peer joined: %s\n", msg.PeerID)
        }
    }
}
```

## Design Decisions

### Interface-Based Architecture

The signaling package uses Go interfaces (`Conn`, `Upgrader`) instead of concrete types. This provides:

1. **Testability**: Mock implementations enable comprehensive unit testing without network I/O
2. **Flexibility**: Swap WebSocket libraries without changing business logic
3. **Dependency Injection**: External dependencies are injected at runtime

### Thread Safety

All shared state uses `sync.RWMutex` with careful lock ordering:

- Registry: Protects peer map
- Room: Protects peer membership
- Peer: Protects connection writes

Read operations use `RLock()` for concurrent access, writes use `Lock()`.

### Message Routing

Messages are routed through a single handler that:

1. Validates message structure
2. Routes to type-specific handlers
3. Forwards targeted messages (OFFER/ANSWER/CANDIDATE) directly
4. Broadcasts room events (JOIN/LEAVE) to room members

### Cleanup Strategy

- **Stale peers**: Removed after `StaleTimeout` without activity
- **Empty rooms**: Removed after `EmptyRoomTTL`
- **Cleanup runs**: Every `CleanupInterval`

## File Structure

```
internal/signaling/
├── protocol.go      # Message types and payloads
├── peer.go          # Connected peer representation
├── registry.go      # Peer tracking and lookup
├── room.go          # Room management
├── handler.go       # WebSocket message handling
├── server.go        # HTTP server orchestration
├── mock.go          # Test mocks (MockConn, MockUpgrader)
├── gorilla.go       # Gorilla/websocket adapter (build tag)
├── *_test.go        # Comprehensive test suites
```

## Testing

```bash
# Run all signaling tests
go test -v ./internal/signaling/

# Run with race detection
go test -race ./internal/signaling/

# Coverage report
go test -coverprofile=coverage.out ./internal/signaling/
go tool cover -html=coverage.out
```

The test suite includes:
- Protocol serialization tests
- Registry concurrency tests
- Room lifecycle tests
- Handler message routing tests
- Server HTTP API tests
- Mock connection utilities