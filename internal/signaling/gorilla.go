//go:build websocket
// +build websocket

// This file provides the gorilla/websocket adapter for the signaling server.
// Build with: go build -tags websocket ./cmd/signaling
//
// If you're seeing build errors, ensure you have gorilla/websocket installed:
//   go get github.com/gorilla/websocket

package signaling

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// GorillaUpgrader adapts websocket.Upgrader to our Upgrader interface.
type GorillaUpgrader struct {
	*websocket.Upgrader
}

// NewGorillaUpgrader creates a new GorillaUpgrader with sensible defaults.
func NewGorillaUpgrader() *GorillaUpgrader {
	return &GorillaUpgrader{
		Upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for development
				// In production, validate against allowed origins
				return true
			},
		},
	}
}

// Upgrade implements the Upgrader interface.
func (g *GorillaUpgrader) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (Conn, error) {
	conn, err := g.Upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// Note: websocket.Conn already implements our Conn interface:
// - WriteMessage(messageType int, data []byte) error
// - ReadMessage() (messageType int, p []byte, err error)
// - Close() error
// - SetWriteDeadline(t time.Time) error
// - SetReadDeadline(t time.Time) error
// - SetReadLimit(limit int64)
// - SetPongHandler(h func(appData string) error)
