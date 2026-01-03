// Copyright (c) 2025
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"

	"github.com/saintparish4/altair/pkg/altair"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "discover":
		if err := discoverCommand(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "connect":
		// Check for help flag
		if len(os.Args) > 2 && (os.Args[2] == "-h" || os.Args[2] == "--help") {
			printConnectUsage()
			return
		}
		if err := connectCommand(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "version", "-v", "--version":
		fmt.Printf("altair version %s\n", altair.GetVersion())
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func discoverCommand() error {
	fmt.Println("Discovering public endpoint via STUN...")

	// Create client
	client := altair.NewClient()

	// Perform discovery
	endpoint, err := client.DiscoverPublicEndpoint()
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	// Print result
	fmt.Printf("\n✓ Discovered public endpoint: %s\n", endpoint)
	fmt.Printf("  IP: %s\n", endpoint.IP)
	fmt.Printf("  Port: %d\n", endpoint.Port)
	fmt.Println()
	fmt.Println("Share this endpoint with your peer to establish a connection.")

	return nil
}

func printUsage() {
	fmt.Println("Altair - P2P NAT Traversal Toolkit")
	fmt.Printf("Version: %s\n", altair.GetVersion())
	fmt.Println()
	fmt.Println("Usage: altair <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  discover        Discover your public IP and port using STUN")
	fmt.Println("  connect         Establish a P2P connection to a remote peer")
	fmt.Println("  version         Show version information")
	fmt.Println("  help            Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  altair discover")
	fmt.Println("  altair connect --peer 203.0.113.5:54321 --initiator")
	fmt.Println()
	fmt.Println("For detailed help on a command:")
	fmt.Println("  altair <command> --help")
}
