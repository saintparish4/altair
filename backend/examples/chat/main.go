package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/saintparish4/altair/pkg/nat"
	"github.com/saintparish4/altair/pkg/netutil"
	"github.com/saintparish4/altair/pkg/punch"
	"github.com/saintparish4/altair/pkg/relay"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
)

var (
	peerAddr    = flag.String("peer", "", "Peer address (IP:PORT)")
	relayServer = flag.String("relay", "", "Relay server address (optional)")
	username    = flag.String("user", "Anonymous", "Your username")
	stunServer  = flag.String("stun", "stun.l.google.com:19302", "STUN address")
)

type ChatConnection struct {
	conn        *net.UDPConn
	remoteAddr  *net.UDPAddr
	isRelayed   bool
	relayClient *relay.Client
}

func main() {
	flag.Parse()

	printBanner()

	// Step 1: Detect NAT type
	fmt.Printf("%s[1/4] Detecting NAT type...%s\n", colorCyan, colorReset)
	mapping, err := detectNAT()
	if err != nil {
		log.Fatalf("%sNAT detection failed: %v%s\n", colorRed, err, colorReset)
	}

	fmt.Printf("%s✓ NAT Type: %s%s\n", colorGreen, mapping.Type, colorReset)
	fmt.Printf("%s✓ Public Address: %s%s\n", colorGreen, mapping.PublicAddr, colorReset)

	// Get local addresses for LAN detection
	localAddrs, _ := getLocalAddresses()
	fmt.Printf("%s✓ Local Addresses: %v%s\n", colorGreen, localAddrs, colorReset)

	// Step 2: Establish connection
	var chatConn *ChatConnection

	if *peerAddr != "" {
		// Connect to specified peer
		fmt.Printf("\n%s[2/4] Connecting to peer %s...%s\n", colorCyan, *peerAddr, colorReset)
		chatConn, err = connectToPeer(*peerAddr, mapping, localAddrs)
		if err != nil {
			log.Fatalf("%sConnection failed: %v%s\n", colorRed, err, colorReset)
		}
	} else {
		// Listen mode
		fmt.Printf("\n%s[2/4] Waiting for incoming connection...%s\n", colorCyan, colorReset)
		printConnectionInfo(mapping, localAddrs)
		chatConn, err = waitForPeer(mapping)
		if err != nil {
			log.Fatalf("%sFailed to accept connection: %v%s\n", colorRed, err, colorReset)
		}
	}

	if chatConn.isRelayed {
		fmt.Printf("%s✓ Connected via relay%s\n", colorGreen, colorReset)
	} else {
		fmt.Printf("%s✓ Direct P2P connection established%s\n", colorGreen, colorReset)
	}

	// Step 3: Start chat
	fmt.Printf("\n%s[3/4] Starting chat session...%s\n", colorCyan, colorReset)
	fmt.Printf("%sType your messages and press Enter. Type '/quit' to exit.%s\n\n",
		colorYellow, colorReset)

	// Start message receiver
	go receiveMessages(chatConn)

	// Start message sender
	sendMessages(chatConn)
}

func printBanner() {
	banner := fmt.Sprintf(`%s
╔═══════════════════════════════════════════════════╗
║                                                   ║
║        %s✦ Altair P2P Chat - Demo ✦%s            ║
║                                                   ║
╚═══════════════════════════════════════════════════╝%s
`, colorPurple, colorYellow, colorPurple, colorReset)
	fmt.Print(banner)
}

func detectNAT() (*nat.Mapping, error) {
	config := &nat.DetectorConfig{
		PrimaryServer:   *stunServer,
		SecondaryServer: "stun1.l.google.com:19302",
		Timeout:         10 * time.Second,
		RetryCount:      3,
	}

	detector, err := nat.NewDetector(config)
	if err != nil {
		return nil, err
	}
	defer detector.Close()

	return detector.DetectWithRetry()
}

func getLocalAddresses() ([]*net.UDPAddr, error) {
	ips, err := netutil.GetPrivateAddresses()
	if err != nil {
		return nil, err
	}

	// Convert to UDP addresses (use ephemeral port)
	addrs := make([]*net.UDPAddr, len(ips))
	for i, ip := range ips {
		addrs[i] = &net.UDPAddr{IP: ip, Port: 0}
	}

	return addrs, nil
}

func printConnectionInfo(mapping *nat.Mapping, localAddrs []*net.UDPAddr) {
	fmt.Printf("\n%s╔═══ Connection Information ═══╗%s\n", colorBlue, colorReset)
	fmt.Printf("%sTo connect to you, peer should use:%s\n", colorYellow, colorReset)
	fmt.Printf("  Public: %s-peer %s%s\n", colorGreen, mapping.PublicAddr, colorReset)

	if len(localAddrs) > 0 {
		fmt.Printf("  LAN:    %s-peer %s:%d%s (if on same network)\n",
			colorGreen, localAddrs[0].IP, mapping.PublicAddr.Port, colorReset)
	}
	fmt.Printf("%s╚═══════════════════════════════╝%s\n\n", colorBlue, colorReset)
}

func connectToPeer(peerAddrStr string, mapping *nat.Mapping, localAddrs []*net.UDPAddr) (*ChatConnection, error) {
	// Parse peer address
	peerUDP, err := net.ResolveUDPAddr("udp", peerAddrStr)
	if err != nil {
		return nil, fmt.Errorf("invalid peer address: %w", err)
	}

	// Determine if we can use hole punching
	// For simplicity, assume peer has compatible NAT if we do
	if mapping.Type != nat.TypeSymmetric || mapping.Type == nat.TypeBlocked {
		fmt.Printf("%sNAT type incompatible for P2P, using relay...%s\n", colorYellow, colorReset)
		return connectViaRelay(peerUDP)
	}

	// Try hole punching
	fmt.Printf("Attempting UDP hole punching...\n")

	config := &punch.PuncherConfig{
		Mapping:      mapping,
		Timeout:      15 * time.Second,
		PingInterval: 200 * time.Millisecond,
		MaxAttempts:  50,
	}

	puncher, err := punch.NewPuncher(config)
	if err != nil {
		return nil, err
	}
	// DO NOT CLOSE PUNCHER - we need its socket

	peerInfo := &punch.PeerInfo{
		PublicAddr: peerUDP,
		LocalAddrs: localAddrs,
		NATType:    nat.TypeFullCone, // Optimistic assumption
	}

	conn, err := puncher.PunchWithRetry(peerInfo, 2)
	if err != nil {
		fmt.Printf("%sHole punching failed: %v%s\n", colorYellow, err, colorReset)

		if *relayServer != "" {
			fmt.Printf("Falling back to relay...\n")
			return connectViaRelay(peerUDP)
		}

		return nil, fmt.Errorf("connection failed and no relay available")
	}

	return &ChatConnection{
		conn:       conn.Conn,
		remoteAddr: conn.RemoteAddr,
		isRelayed:  false,
	}, nil
}

func connectViaRelay(peerAddr *net.UDPAddr) (*ChatConnection, error) {
	if *relayServer == "" {
		return nil, fmt.Errorf("relay server not specified")
	}

	client, _, err := relay.QuickRelay(*relayServer, 10*time.Minute)
	if err != nil {
		return nil, err
	}

	// Create a dummy connection for the interface
	// We'll use relayClient.Send/Receive for actual communication
	dummyConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create dummy connection: %w", err)
	}

	return &ChatConnection{
		conn:        dummyConn,
		remoteAddr:  peerAddr,
		isRelayed:   true,
		relayClient: client,
	}, nil
}

func waitForPeer(mapping *nat.Mapping) (*ChatConnection, error) {
	// Create listener
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: mapping.PublicAddr.Port, // Try to use same port as STUN
	})
	if err != nil {
		// If that fails, use any port
		conn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
		if err != nil {
			return nil, err
		}
	}

	fmt.Printf("Listening on %s\n", conn.LocalAddr())
	fmt.Printf("Waiting for peer to connect...\n")

	// Wait for first message
	buf := make([]byte, 1500)
	conn.SetReadDeadline(time.Now().Add(5 * time.Minute)) // 5 minute timeout

	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			return nil, err
		}

		// Check if it's a PING (hole punching attempt)
		if n >= 4 && string(buf[:4]) == "PING" {
			// Send PONG back
			conn.WriteToUDP([]byte("PONG"), addr)
			continue
		}

		// First real message received
		fmt.Printf("\n%sReceived connection from %s%s\n", colorGreen, addr, colorReset)

		// Send confirmation
		conn.WriteToUDP([]byte("CONNECTED"), addr)

		return &ChatConnection{
			conn:       conn,
			remoteAddr: addr,
			isRelayed:  false,
		}, nil
	}
}

func sendMessages(chatConn *ChatConnection) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Printf("%s%s > %s", colorCyan, *username, colorReset)

		if !scanner.Scan() {
			break
		}

		message := strings.TrimSpace(scanner.Text())
		if message == "" {
			continue
		}

		if message == "/quit" || message == "/exit" {
			fmt.Printf("%sGoodbye!%s\n", colorYellow, colorReset)
			os.Exit(0)
		}

		// Format message
		fullMessage := fmt.Sprintf("%s: %s", *username, message)

		// Send message
		var err error
		if chatConn.isRelayed && chatConn.relayClient != nil {
			err = chatConn.relayClient.Send([]byte(fullMessage), chatConn.remoteAddr)
		} else {
			_, err = chatConn.conn.WriteToUDP([]byte(fullMessage), chatConn.remoteAddr)
		}
		if err != nil {
			fmt.Printf("%s✗ Failed to send: %v%s\n", colorRed, err, colorReset)
			continue
		}
	}
}

func receiveMessages(chatConn *ChatConnection) {
	buf := make([]byte, 1500)

	for {
		var n int
		var addr *net.UDPAddr
		var err error

		if chatConn.isRelayed && chatConn.relayClient != nil {
			// Use relay client's Receive method
			data, recvAddr, recvErr := chatConn.relayClient.Receive()
			if recvErr != nil {
				fmt.Printf("\n%s✗ Connection error: %v%s\n", colorRed, recvErr, colorReset)
				os.Exit(1)
			}
			n = len(data)
			copy(buf, data)
			addr = recvAddr
		} else {
			chatConn.conn.SetReadDeadline(time.Time{}) // No timeout
			n, addr, err = chatConn.conn.ReadFromUDP(buf)
			if err != nil {
				fmt.Printf("\n%s✗ Connection error: %v%s\n", colorRed, err, colorReset)
				os.Exit(1)
			}
		}

		// Verify it's from our peer
		if !addr.IP.Equal(chatConn.remoteAddr.IP) || addr.Port != chatConn.remoteAddr.Port {
			continue
		}

		message := string(buf[:n])

		// Skip protocol messages
		if message == "PING" || message == "PONG" || message == "CONNECTED" {
			continue
		}

		// Display message
		fmt.Printf("\r%s%s%s\n", colorGreen, message, colorReset)
		fmt.Printf("%s%s > %s", colorCyan, *username, colorReset)
	}
}
