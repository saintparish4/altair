// P2P Chat - A terminal-based chat application using Altair's hole punching
//
// Usage:
//
//	# Start chat and wait for peer (responder mode)
//	altair-chat --username Alice --listen :9000
//
//	# Connect to peer (initiator mode)
//	altair-chat --username Bob --peer 203.0.113.42:9000
//
//	# With signaling server (automatic coordination)
//	altair-chat --username Alice --room my-chat --signaling ws://server:8080/ws
package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/saintparish4/altair/pkg/chat"
)

const banner = `
   _   _ _        _        ___ _         _   
  /_\ | | |_ __ _(_)_ _   / __| |_  __ _| |_ 
 / _ \| |  _/ _' | | '_| | (__| ' \/ _' |  _|
/_/ \_\_|\__\__,_|_|_|    \___|_||_\__,_|\__|
                                             
         P2P Chat over NAT Traversal
`

func main() {
	// Parse flags
	username := flag.String("username", "", "Your display name (required)")
	listenAddr := flag.String("listen", "", "Address to listen on (responder mode)")
	peerAddr := flag.String("peer", "", "Peer address to connect to (initiator mode)")
	roomID := flag.String("room", "", "Room ID for signaling server coordination")
	signalingURL := flag.String("signaling", "", "Signaling server WebSocket URL")
	timeout := flag.Duration("timeout", 30*time.Second, "Connection timeout")
	flag.Parse()

	// Validate flags
	if *username == "" {
		fmt.Println("Error: --username is required")
		flag.Usage()
		os.Exit(1)
	}

	// Determine mode
	hasManual := *listenAddr != "" || *peerAddr != ""
	hasSignaling := *roomID != "" && *signalingURL != ""

	if !hasManual && !hasSignaling {
		fmt.Println("Error: specify either --listen/--peer for manual mode or --room/--signaling for automatic mode")
		flag.Usage()
		os.Exit(1)
	}

	// Print banner
	fmt.Print(chat.ColorCyan + banner + chat.ColorReset)
	fmt.Printf("\n%sUsername:%s %s\n", chat.ColorGray, chat.ColorReset, *username)

	var conn net.Conn
	var err error

	if hasSignaling {
		// Automatic mode with signaling server
		fmt.Printf("%sMode:%s Signaling (room: %s)\n", chat.ColorGray, chat.ColorReset, *roomID)
		fmt.Printf("%sConnecting to signaling server...%s\n", chat.ColorYellow, chat.ColorReset)

		// TODO: Integrate with actual signaling client and hole punching
		// For now, show placeholder
		fmt.Printf("\n%s[Signaling mode requires integration with altair holepunch package]%s\n", chat.ColorRed, chat.ColorReset)
		fmt.Println("Use manual mode with --listen or --peer for now.")
		os.Exit(1)

	} else if *listenAddr != "" {
		// Responder mode - listen for incoming connection
		fmt.Printf("%sMode:%s Listening on %s\n", chat.ColorGray, chat.ColorReset, *listenAddr)
		fmt.Printf("%sWaiting for peer to connect...%s\n\n", chat.ColorYellow, chat.ColorReset)

		listener, err := net.Listen("tcp", *listenAddr)
		if err != nil {
			fmt.Printf("%sError: failed to listen: %v%s\n", chat.ColorRed, err, chat.ColorReset)
			os.Exit(1)
		}
		defer listener.Close()

		// Set accept timeout
		if tcpListener, ok := listener.(*net.TCPListener); ok {
			tcpListener.SetDeadline(time.Now().Add(*timeout))
		}

		conn, err = listener.Accept()
		if err != nil {
			fmt.Printf("%sError: connection timeout or failed: %v%s\n", chat.ColorRed, err, chat.ColorReset)
			os.Exit(1)
		}

	} else if *peerAddr != "" {
		// Initiator mode - connect to peer
		fmt.Printf("%sMode:%s Connecting to %s\n", chat.ColorGray, chat.ColorReset, *peerAddr)
		fmt.Printf("%sEstablishing connection...%s\n\n", chat.ColorYellow, chat.ColorReset)

		conn, err = net.DialTimeout("tcp", *peerAddr, *timeout)
		if err != nil {
			fmt.Printf("%sError: failed to connect: %v%s\n", chat.ColorRed, err, chat.ColorReset)
			os.Exit(1)
		}
	}

	defer conn.Close()

	// Connection established
	fmt.Printf("%s✓ Connected to %s%s\n", chat.ColorGreen, conn.RemoteAddr(), chat.ColorReset)
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("%sType your message and press Enter. Type /quit to exit.%s\n\n", chat.ColorGray, chat.ColorReset)

	// Create downloads directory
	downloadsDir := "./downloads"
	os.MkdirAll(downloadsDir, 0755)

	// Create chat session
	session := chat.NewSession(conn, chat.SessionConfig{
		Username:  *username,
		OutputDir: downloadsDir,
		OnMessage: func(msg *chat.Message) {
			// Clear current input line, print message, restore prompt
			chat.ClearLine()
			formatted := chat.FormatMessage(msg, msg.From == *username)
			if formatted != "" {
				fmt.Println(formatted)
			}
			fmt.Print(chat.ColorCyan + "> " + chat.ColorReset)
		},
		OnError: func(err error) {
			chat.ClearLine()
			fmt.Printf("\n%s✗ Connection error: %v%s\n", chat.ColorRed, err, chat.ColorReset)
		},
		OnFileInfo: func(fileInfo *chat.FileInfo) bool {
			// Auto-accept files for now (could add prompt here)
			return true
		},
	})

	if err := session.Start(); err != nil {
		fmt.Printf("%sError starting session: %v%s\n", chat.ColorRed, err, chat.ColorReset)
		os.Exit(1)
	}

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Printf("\n%sDisconnecting...%s\n", chat.ColorYellow, chat.ColorReset)
		session.Stop()
		os.Exit(0)
	}()

	// Main input loop
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(chat.ColorCyan + "> " + chat.ColorReset)
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			parts := strings.Fields(input)
			cmd := strings.ToLower(parts[0])

			switch cmd {
			case "/quit", "/exit", "/q":
				fmt.Printf("%sDisconnecting...%s\n", chat.ColorYellow, chat.ColorReset)
				session.Stop()
				return

			case "/clear":
				fmt.Print("\033[H\033[2J") // Clear screen
				continue

			case "/help":
				printHelp()
				continue

			case "/status":
				printStatus(session)
				continue

			case "/send":
				if len(parts) < 2 {
					fmt.Printf("%sUsage: /send <filepath>%s\n", chat.ColorRed, chat.ColorReset)
					continue
				}
				filePath := parts[1]
				handleSendFile(session, filePath)
				continue

			default:
				fmt.Printf("%sUnknown command. Type /help for available commands.%s\n", chat.ColorGray, chat.ColorReset)
				continue
			}
		}

		// Send message
		if err := session.Send(input); err != nil {
			fmt.Printf("%sError sending message: %v%s\n", chat.ColorRed, err, chat.ColorReset)
		} else {
			// Show own message
			chat.ClearLine()
			chat.MoveCursorUp(1)
			chat.ClearLine()
			msg := chat.NewTextMessage(*username, input)
			fmt.Println(chat.FormatMessage(msg, true))
		}
	}
}

func printHelp() {
	fmt.Println()
	fmt.Println(chat.ColorBold + "Available Commands:" + chat.ColorReset)
	fmt.Println("  /quit, /exit, /q  - Disconnect and exit")
	fmt.Println("  /clear            - Clear the screen")
	fmt.Println("  /status           - Show connection status")
	fmt.Println("  /send <filepath>  - Send a file to peer")
	fmt.Println("  /help             - Show this help message")
	fmt.Println()
}

func handleSendFile(session *chat.Session, filePath string) {
	chat.ClearLine()
	fmt.Printf("%sSending file: %s...%s\n", chat.ColorYellow, filePath, chat.ColorReset)

	// Create progress bar
	lastProgress := int64(0)
	onProgress := func(progress *chat.FileProgress) {
		percent := float64(progress.BytesSent) / float64(progress.FileSize) * 100
		if progress.BytesSent-lastProgress > 1024*1024 || progress.BytesSent == progress.FileSize {
			// Update every MB or on completion
			chat.ClearLine()
			speed := progress.Speed
			speedStr := formatSpeed(speed)
			fmt.Printf("\r%sProgress: %.1f%% (%s) - %s%s", chat.ColorCyan, percent, formatBytes(progress.BytesSent), speedStr, chat.ColorReset)
			lastProgress = progress.BytesSent
		}
	}

	err := session.SendFile(filePath, onProgress)
	chat.ClearLine()
	if err != nil {
		fmt.Printf("%s✗ Error sending file: %v%s\n", chat.ColorRed, err, chat.ColorReset)
	} else {
		fmt.Printf("%s✓ File sent successfully!%s\n", chat.ColorGreen, chat.ColorReset)
	}
}

func formatSpeed(bytesPerSec float64) string {
	if bytesPerSec < 1024 {
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	}
	if bytesPerSec < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/1024)
	}
	return fmt.Sprintf("%.1f MB/s", bytesPerSec/(1024*1024))
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func printStatus(session *chat.Session) {
	fmt.Println()
	fmt.Println(chat.ColorBold + "Session Status:" + chat.ColorReset)
	fmt.Printf("  You: %s\n", session.Username())
	fmt.Printf("  Peer: %s\n", session.PeerName())
	fmt.Printf("  Messages: %d\n", len(session.Messages()))
	fmt.Println()
}
