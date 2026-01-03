// Copyright (c) 2025
// SPDX-License-Identifier: MIT

/*
Package altair provides a simple and efficient P2P NAT traversal library.

Altair enables direct peer-to-peer UDP connections between devices behind NAT
using STUN for endpoint discovery and UDP hole punching for NAT traversal.

# Quick Start

Discover your public endpoint:

	client := altair.NewClient()
	endpoint, err := client.DiscoverPublicEndpoint()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Public endpoint: %s\n", endpoint)

Establish a P2P connection:

	// Peer A (initiator)
	conn, err := client.Connect(remoteEndpoint, true)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Peer B (responder)
	conn, err := client.Connect(remoteEndpoint, false)

# Architecture

Altair consists of three layers:

1. STUN Client - Discovers public IP and port through STUN servers
2. UDP Hole Punching - Establishes direct P2P connections through NAT
3. Signaling - Coordinates peer discovery and connection setup

# Use Cases

- Real-time communication (voice, video, messaging)
- File transfer between devices
- Multiplayer gaming
- IoT device communication
- Distributed systems

# Features

- RFC 5389 compliant STUN implementation
- Automatic NAT traversal through UDP hole punching
- Configurable retry logic and timeouts
- IPv4 and IPv6 support
- Production-ready error handling
- Zero external dependencies (except signaling)

For detailed examples and documentation, see https://github.com/saintparish4/altair
*/
package altair
