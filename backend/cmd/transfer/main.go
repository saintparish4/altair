// P2P File Transfer - Send files directly between peers using Altair
//
// Usage:
//
//	# Send a file (connects to receiver)
//	altair-transfer send --file document.pdf --peer 192.168.1.100:9001
//
//	# Receive files (listens for sender)
//	altair-transfer receive --listen :9001 --output ./downloads
//
//	# With signaling server
//	altair-transfer send --file doc.pdf --room transfers --signaling ws://server:8080/ws
//	altair-transfer receive --room transfers --signaling ws://server:8080/ws
package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/saintparish4/altair/pkg/transfer"
)

const banner = `
   _   _ _        _        _____                   __         
  /_\ | | |_ __ _(_)_ _   |_   _| _ __ _ _ _  ___ / _|___ _ _ 
 / _ \| |  _/ _' | | '_|    | || '_/ _' | ' \(_-<|  _/ -_) '_|
/_/ \_\_|\__\__,_|_|_|      |_||_| \__,_|_||_/__/|_| \___|_|  
                                                              
            P2P File Transfer over NAT Traversal
`

// Terminal colors
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "send":
		sendCmd()
	case "receive":
		receiveCmd()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("%sUnknown command: %s%s\n", colorRed, command, colorReset)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(colorCyan + banner + colorReset)
	fmt.Println()
	fmt.Println(colorBold + "Usage:" + colorReset)
	fmt.Println("  altair-transfer <command> [options]")
	fmt.Println()
	fmt.Println(colorBold + "Commands:" + colorReset)
	fmt.Println("  send      Send a file to a peer")
	fmt.Println("  receive   Receive files from a peer")
	fmt.Println()
	fmt.Println(colorBold + "Examples:" + colorReset)
	fmt.Println("  # Send a file")
	fmt.Println("  altair-transfer send --file document.pdf --peer 192.168.1.100:9001")
	fmt.Println()
	fmt.Println("  # Receive files")
	fmt.Println("  altair-transfer receive --listen :9001 --output ./downloads")
	fmt.Println()
	fmt.Println("Run 'altair-transfer <command> --help' for command-specific options.")
}

func sendCmd() {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	filePath := fs.String("file", "", "Path to the file to send (required)")
	peerAddr := fs.String("peer", "", "Peer address to connect to (required for manual mode)")
	roomID := fs.String("room", "", "Room ID for signaling")
	signalingURL := fs.String("signaling", "", "Signaling server URL")
	timeout := fs.Duration("timeout", 30*time.Second, "Connection timeout")
	fs.Parse(os.Args[2:])

	// Validate
	if *filePath == "" {
		fmt.Printf("%sError: --file is required%s\n", colorRed, colorReset)
		fs.Usage()
		os.Exit(1)
	}

	// Check file exists
	fileInfo, err := os.Stat(*filePath)
	if err != nil {
		fmt.Printf("%sError: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	hasManual := *peerAddr != ""
	hasSignaling := *roomID != "" && *signalingURL != ""

	if !hasManual && !hasSignaling {
		fmt.Printf("%sError: specify either --peer for manual mode or --room/--signaling for automatic mode%s\n", colorRed, colorReset)
		fs.Usage()
		os.Exit(1)
	}

	// Print banner
	fmt.Print(colorCyan + banner + colorReset)
	fmt.Println()
	fmt.Printf("%sFile:%s %s\n", colorGray, colorReset, filepath.Base(*filePath))
	fmt.Printf("%sSize:%s %s\n", colorGray, colorReset, transfer.FormatBytes(fileInfo.Size()))

	var conn net.Conn

	if hasSignaling {
		fmt.Printf("\n%s[Signaling mode requires integration with altair holepunch]%s\n", colorRed, colorReset)
		fmt.Println("Use --peer for manual mode.")
		os.Exit(1)
	}

	// Manual mode - connect to peer
	fmt.Printf("%sConnecting to %s...%s\n", colorYellow, *peerAddr, colorReset)

	conn, err = net.DialTimeout("tcp", *peerAddr, *timeout)
	if err != nil {
		fmt.Printf("%sError: failed to connect: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("%s✓ Connected%s\n", colorGreen, colorReset)
	fmt.Println(strings.Repeat("─", 60))

	// Create progress bar
	pb := transfer.NewProgressBar(40)

	// Create sender
	sender := transfer.NewSender(conn, *filePath, func(p transfer.Progress) {
		fmt.Print(pb.Render(p))
	})

	// Send file
	fmt.Printf("%sSending %s...%s\n", colorYellow, filepath.Base(*filePath), colorReset)
	startTime := time.Now()

	if err := sender.Send(); err != nil {
		fmt.Printf("\n%sError: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	duration := time.Since(startTime)
	avgSpeed := float64(fileInfo.Size()) / duration.Seconds()

	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("%s✓ Transfer complete!%s\n", colorGreen, colorReset)
	fmt.Printf("%sTime:%s %s\n", colorGray, colorReset, transfer.FormatDuration(duration))
	fmt.Printf("%sAverage speed:%s %s/s\n", colorGray, colorReset, transfer.FormatBytes(int64(avgSpeed)))
}

func receiveCmd() {
	fs := flag.NewFlagSet("receive", flag.ExitOnError)
	listenAddr := fs.String("listen", ":9001", "Address to listen on")
	outputDir := fs.String("output", ".", "Directory to save received files")
	roomID := fs.String("room", "", "Room ID for signaling")
	signalingURL := fs.String("signaling", "", "Signaling server URL")
	autoAccept := fs.Bool("auto", false, "Automatically accept all transfers")
	timeout := fs.Duration("timeout", 120*time.Second, "Connection timeout")
	fs.Parse(os.Args[2:])

	hasSignaling := *roomID != "" && *signalingURL != ""

	// Create output directory if needed
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("%sError creating output directory: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	// Print banner
	fmt.Print(colorCyan + banner + colorReset)
	fmt.Println()
	fmt.Printf("%sOutput directory:%s %s\n", colorGray, colorReset, *outputDir)

	var conn net.Conn
	var err error

	if hasSignaling {
		fmt.Printf("\n%s[Signaling mode requires integration with altair holepunch]%s\n", colorRed, colorReset)
		fmt.Println("Use --listen for manual mode.")
		os.Exit(1)
	}

	// Manual mode - listen for connection
	fmt.Printf("%sListening on %s...%s\n", colorYellow, *listenAddr, colorReset)
	fmt.Printf("%sWaiting for sender to connect...%s\n\n", colorGray, colorReset)

	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		fmt.Printf("%sError: failed to listen: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
	defer listener.Close()

	// Set accept timeout
	if tcpListener, ok := listener.(*net.TCPListener); ok {
		tcpListener.SetDeadline(time.Now().Add(*timeout))
	}

	conn, err = listener.Accept()
	if err != nil {
		fmt.Printf("%sError: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("%s✓ Sender connected from %s%s\n", colorGreen, conn.RemoteAddr(), colorReset)
	fmt.Println(strings.Repeat("─", 60))

	// Create progress bar
	pb := transfer.NewProgressBar(40)

	// Create receiver
	receiver := transfer.NewReceiver(conn, *outputDir, func(p transfer.Progress) {
		fmt.Print(pb.Render(p))
	})

	// Set up file acceptance callback
	if !*autoAccept {
		receiver.SetFileInfoHandler(func(info *transfer.FileInfo) bool {
			fmt.Printf("\n%sIncoming file:%s %s (%s)\n", colorYellow, colorReset, info.Name, transfer.FormatBytes(info.Size))
			fmt.Printf("Accept? [Y/n]: ")

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response == "" || response == "y" || response == "yes" {
				fmt.Printf("%sReceiving %s...%s\n", colorYellow, info.Name, colorReset)
				return true
			}
			fmt.Printf("%sTransfer rejected%s\n", colorRed, colorReset)
			return false
		})
	} else {
		receiver.SetFileInfoHandler(func(info *transfer.FileInfo) bool {
			fmt.Printf("%sReceiving %s (%s)...%s\n", colorYellow, info.Name, transfer.FormatBytes(info.Size), colorReset)
			return true
		})
	}

	// Receive file
	startTime := time.Now()
	outputPath, err := receiver.Receive()
	if err != nil {
		fmt.Printf("\n%sError: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	duration := time.Since(startTime)
	fileInfo, _ := os.Stat(outputPath)
	avgSpeed := float64(fileInfo.Size()) / duration.Seconds()

	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("%s✓ Transfer complete!%s\n", colorGreen, colorReset)
	fmt.Printf("%sSaved to:%s %s\n", colorGray, colorReset, outputPath)
	fmt.Printf("%sTime:%s %s\n", colorGray, colorReset, transfer.FormatDuration(duration))
	fmt.Printf("%sAverage speed:%s %s/s\n", colorGray, colorReset, transfer.FormatBytes(int64(avgSpeed)))
}
