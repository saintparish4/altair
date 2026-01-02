// Command signaling runs the Altair signaling server for P2P coordination.
//
// The signaling server enables peers to discover each other and exchange
// endpoint information for establishing direct P2P connections through NAT.
//
// Usage:
//
//	altair-signaling [flags]
//
// Flags:
//
//	-addr string    Listen address (default ":8080")
//	-verbose        Enable verbose logging
//
// Endpoints:
//
//	WebSocket: ws://host:port/ws
//	Health:    GET /health
//	Stats:     GET /api/stats
//	Rooms:     GET /api/rooms
//	Room:      GET /api/rooms/{id}
//
// Build with gorilla/websocket:
//
//	go get github.com/gorilla/websocket
//	go build -tags websocket ./cmd/signaling
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/saintparish4/altair/internal/signaling"
)

var (
	version = "dev" // Set via ldflags
)

func main() {
	// Parse command line flags
	addr := flag.String("addr", ":8080", "Listen address (e.g., :8080 or 0.0.0.0:8080)")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("altair-signaling %s\n", version)
		os.Exit(0)
	}

	// Configure logging
	var logger *log.Logger
	if *verbose {
		logger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)
	} else {
		logger = log.New(io.Discard, "", 0)
	}

	// Create server configuration
	cfg := signaling.Config{
		Addr:            *addr,
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		CleanupInterval: 1 * time.Minute,
		StaleTimeout:    5 * time.Minute,
		Logger:          logger,
	}

	// Create and start server
	server := signaling.NewServer(cfg)

	// NOTE: WebSocket upgrader must be set before starting.
	// When building with gorilla/websocket, use:
	//
	//   import "github.com/gorilla/websocket"
	//   upgrader := &GorillaUpgrader{websocket.Upgrader{
	//       ReadBufferSize:  1024,
	//       WriteBufferSize: 1024,
	//       CheckOrigin: func(r *http.Request) bool { return true },
	//   }}
	//   server.Handler().SetUpgrader(upgrader)
	//
	// See internal/signaling/gorilla.go for the adapter implementation.

	// Print startup banner
	printBanner(*addr, *verbose)

	if err := server.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func printBanner(addr string, verbose bool) {
	fmt.Println()
	fmt.Println("  █████╗ ██╗  ████████╗ █████╗ ██╗██████╗ ")
	fmt.Println(" ██╔══██╗██║  ╚══██╔══╝██╔══██╗██║██╔══██╗")
	fmt.Println(" ███████║██║     ██║   ███████║██║██████╔╝")
	fmt.Println(" ██╔══██║██║     ██║   ██╔══██║██║██╔══██╗")
	fmt.Println(" ██║  ██║███████╗██║   ██║  ██║██║██║  ██║")
	fmt.Println(" ╚═╝  ╚═╝╚══════╝╚═╝   ╚═╝  ╚═╝╚═╝╚═╝  ╚═╝")
	fmt.Println("          Signaling Server")
	fmt.Println()
	fmt.Printf(" WebSocket:  ws://localhost%s/ws\n", addr)
	fmt.Printf(" Health:     http://localhost%s/health\n", addr)
	fmt.Printf(" Stats:      http://localhost%s/api/stats\n", addr)
	fmt.Printf(" Rooms:      http://localhost%s/api/rooms\n", addr)
	fmt.Println()
	if verbose {
		fmt.Println(" Verbose logging: enabled")
	}
	fmt.Println(" Press Ctrl+C to stop")
	fmt.Println()
}
