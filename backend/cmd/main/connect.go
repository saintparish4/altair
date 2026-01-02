package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/saintparish4/altair/pkg/holepunch"
	"github.com/saintparish4/altair/pkg/stun"
	"github.com/saintparish4/altair/pkg/types"
)

func connectCommand(args []string) error {
	// Parse flags
	fs := flag.NewFlagSet("connect", flag.ExitOnError)
	peerAddr := fs.String("peer", "", "Remote peer's public endpoint (IP:PORT)")
	stunServer := fs.String("stun", defaultSTUNServer, "STUN server for discovering public IP")
	initiator := fs.Bool("initiator", false, "Whether this peer initiates the PING (default: false = responder)")
	skipPingPong := fs.Bool("skip-test", false, "Skip the ping-pong test")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate peer address
	if *peerAddr == "" {
		return fmt.Errorf("--peer flag is required (use --help for usage)")
	}

	// Parse peer endpoint
	remoteEndpoint, err := parseEndpoint(*peerAddr)
	if err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}

	fmt.Println("=== Altair P2P Connection ===")
	fmt.Println()

	// Step 1: Discover our public endpoint
	fmt.Printf("Step 1: Discovering public endpoint via STUN (%s)...\n", *stunServer)
	client := stun.NewClient(*stunServer)
	publicEndpoint, err := client.Discover()
	if err != nil {
		return fmt.Errorf("STUN discovery failed: %w", err)
	}
	fmt.Printf("✓ Your public endpoint: %s\n", publicEndpoint)
	fmt.Println()

	// Step 2: Display connection info
	fmt.Println("Step 2: Connection Information")
	fmt.Printf("  Your public IP:Port  : %s\n", publicEndpoint)
	fmt.Printf("  Peer's public IP:Port: %s\n", remoteEndpoint)
	fmt.Println()

	// Step 3: Establish connection through hole punching
	fmt.Println("Step 3: Establishing P2P connection...")
	fmt.Println("  This will punch through your NAT and connect directly to peer.")
	fmt.Println("  Both peers should run this command simultaneously!")
	fmt.Println()

	conn, err := holepunch.EstablishConnection(nil, remoteEndpoint)
	if err != nil {
		return fmt.Errorf("failed to establish connection: %w", err)
	}
	defer conn.Close()

	fmt.Println()
	fmt.Println("🎉 Connection established!")
	fmt.Printf("✓ Direct UDP connection to %s\n", remoteEndpoint)
	fmt.Println()

	// Step 4: Optional ping-pong test
	if !*skipPingPong {
		fmt.Println("Step 4: Testing connection with PING-PONG...")

		if *initiator {
			fmt.Println("  [You are the INITIATOR - sending PING first]")
		} else {
			fmt.Println("  [You are the RESPONDER - waiting for PING]")
		}

		if err := holepunch.PingPong(conn, remoteEndpoint, *initiator); err != nil {
			return fmt.Errorf("ping-pong test failed: %w", err)
		}
	}

	fmt.Println()
	fmt.Println("✓ Connection verified and ready for data exchange!")
	fmt.Println()
	fmt.Println("Connection will remain open. Press Ctrl+C to close.")

	// Keep connection open
	select {}
}

func parseEndpoint(addr string) (*types.Endpoint, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("address must be in format IP:PORT")
	}

	var port int
	_, err := fmt.Sscanf(parts[1], "%d", &port)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	return &types.Endpoint{
		IP:   parts[0],
		Port: port,
	}, nil
}

func printConnectUsage() {
	fmt.Println("Usage: altair connect --peer IP:PORT [options]")
	fmt.Println()
	fmt.Println("Establish a direct P2P connection through NAT hole punching.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --peer string       Remote peer's public endpoint (required)")
	fmt.Println("                      Format: IP:PORT")
	fmt.Println("                      Example: 203.0.113.5:54321")
	fmt.Println()
	fmt.Println("  --stun string       STUN server for IP discovery")
	fmt.Println("                      (default: stun.l.google.com:19302)")
	fmt.Println()
	fmt.Println("  --initiator         This peer sends PING first (one peer must set this)")
	fmt.Println("                      (default: false = responder mode)")
	fmt.Println()
	fmt.Println("  --skip-test         Skip the ping-pong connection test")
	fmt.Println()
	fmt.Println("Setup Instructions:")
	fmt.Println("  1. Both peers run: altair discover")
	fmt.Println("  2. Exchange discovered IP:PORT with peer (via chat, email, etc.)")
	fmt.Println("  3. One peer runs:  altair connect --peer PEER_IP:PORT --initiator")
	fmt.Println("  4. Other peer runs: altair connect --peer PEER_IP:PORT")
	fmt.Println("  5. Both commands should be run within a few seconds of each other")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  # Peer A discovers their public endpoint")
	fmt.Println("  $ altair discover")
	fmt.Println("  ✓ Discovered public endpoint: 192.0.2.100:12345")
	fmt.Println()
	fmt.Println("  # Peer B discovers their public endpoint")
	fmt.Println("  $ altair discover")
	fmt.Println("  ✓ Discovered public endpoint: 198.51.100.200:54321")
	fmt.Println()
	fmt.Println("  # Peer A connects (as initiator)")
	fmt.Println("  $ altair connect --peer 198.51.100.200:54321 --initiator")
	fmt.Println()
	fmt.Println("  # Peer B connects (as responder)")
	fmt.Println("  $ altair connect --peer 192.0.2.100:12345")
	fmt.Println()
	fmt.Println("Why both peers need to send simultaneously:")
	fmt.Println("  NAT hole punching works by having both peers send UDP packets")
	fmt.Println("  to each other's public endpoints at the same time. This creates")
	fmt.Println("  temporary holes in both NATs that allow the packets through.")
}
