package main

import (
	"fmt"
	"os"

	"github.com/saintparish4/altair/pkg/stun"
)

const (
	defaultSTUNServer = "stun.l.google.com:19302"
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
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func discoverCommand() error {
	// Get STUN server from environment or use default
	stunServer := os.Getenv("STUN_SERVER")
	if stunServer == "" {
		stunServer = defaultSTUNServer
	}

	fmt.Printf("Discovering public endpoint using STUN server: %s\n", stunServer)

	// Create STUN client
	client := stun.NewClient(stunServer)

	// Perform discovery
	endpoint, err := client.Discover()
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	// Print result
	fmt.Printf("\n Discovered public endpoint: %s\n", endpoint)
	fmt.Printf(" IP: %s\n", endpoint.IP)
	fmt.Printf(" Port: %d\n", endpoint.Port)

	return nil
}

func printUsage() {
	fmt.Println("Usage: altair <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println(" discover Discover your public IP and port using STUN")
	fmt.Println(" help, Show this help message")
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Printf(" STUN_SERVER   STUN server address (default: %s)\n", defaultSTUNServer)
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  Altair discover")
	fmt.Println(" STUN_SERVER=stun.ekiga.net:3478 altair discover")
}
