.PHONY: all build build-signaling build-chat test test-coverage clean install examples fmt lint help chat

# Default target
all: build test

# Build the CLI binary
build:
	@echo "Building altair CLI..."
	@cd backend && go build -o ../bin/altair ./cmd/altair
	@echo "✓ Binary created: ./bin/altair"

# Build signaling server
build-signaling:
	@echo "Building signaling server..."
	@cd backend && go build -o ../bin/altair-signaling ./cmd/signaling
	@echo "✓ Binary created: ./bin/altair-signaling"

# Build chat application
build-chat:
	@echo "Building chat application..."
	@cd backend && go build -o ../bin/altair-chat ./cmd/chat
	@echo "✓ Binary created: ./bin/altair-chat"

# Build all examples
examples:
	@echo "Building examples..."
	@mkdir -p bin/examples
	@cd examples/simple-p2p && go build -o ../../bin/examples/simple-p2p
	@cd examples/chat && go build -o ../../bin/examples/chat
	@cd examples/file-transfer && go build -o ../../bin/examples/file-transfer
	@echo "✓ Examples built in ./bin/examples/"

# Run all tests
test:
	@echo "Running all tests..."
	@cd backend && go test ./... -v

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@cd backend && go test ./... -race -coverprofile=../coverage.out -covermode=atomic
	@cd backend && go tool cover -html=../coverage.out -o ../coverage.html
	@echo "✓ Coverage report: coverage.html"

# Install CLI globally
install: build
	@echo "Installing altair..."
	@cd backend && go install ./cmd/altair
	@echo "✓ Installed altair to GOPATH/bin"

# Run signaling server
serve: build-signaling
	@echo "Starting signaling server on :8080..."
	@./bin/altair-signaling

# Run chat application
# Usage: make chat ARGS="--username Alice --listen :9000"
# Usage: make chat ARGS="--username Bob --peer 127.0.0.1:9000"
# Usage: make chat ARGS="--username Alice --room my-room --signaling ws://localhost:8080/ws"
chat: build-chat
	@if [ -z "$(ARGS)" ]; then \
		echo ""; \
		echo "❌ Error: ARGS parameter is required"; \
		echo ""; \
		echo "Usage: make chat ARGS=\"--username NAME [options]\""; \
		echo ""; \
		echo "Examples:"; \
		echo "  make chat ARGS=\"--username Alice --listen :9000\""; \
		echo "  make chat ARGS=\"--username Bob --peer 127.0.0.1:9000\""; \
		echo "  make chat ARGS=\"--username Alice --room my-room --signaling ws://localhost:8080/ws\""; \
		echo ""; \
		echo "Required flags:"; \
		echo "  --username string    Your display name (required)"; \
		echo ""; \
		echo "Connection modes (choose one):"; \
		echo "  --listen string     Address to listen on (responder mode)"; \
		echo "  --peer string       Peer address to connect to (initiator mode)"; \
		echo "  --room string       Room ID for signaling server (requires --signaling)"; \
		echo "  --signaling string  Signaling server WebSocket URL"; \
		echo ""; \
		exit 1; \
	fi
	@echo "Starting chat application..."
	@./bin/altair-chat $(ARGS)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/ coverage.out coverage.html
	@rm -f altair altair-* received_* altair-chat
	@echo "✓ Clean complete"

# Format code
fmt:
	@echo "Formatting code..."
	@cd backend && go fmt ./...
	@cd examples && go fmt ./...
	@echo "✓ Formatting complete"

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@cd backend && golangci-lint run
	@echo "✓ Linting complete"

# Run quick discovery test
discover: build
	@echo "Running STUN discovery..."
	@./bin/altair discover

# Show version
version: build
	@./bin/altair version

# Show help
help:
	@echo "Altair - P2P NAT Traversal Library"
	@echo ""
	@echo "Available targets:"
	@echo "  make build           - Build the altair CLI binary"
	@echo "  make build-signaling - Build the signaling server"
	@echo "  make build-chat     - Build the chat application"
	@echo "  make examples       - Build all example applications"
	@echo "  make test           - Run all unit tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make install        - Install altair CLI to GOPATH/bin"
	@echo "  make serve          - Run signaling server"
	@echo "  make chat           - Run chat application (requires ARGS)"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make fmt            - Format all code"
	@echo "  make lint           - Run linter (requires golangci-lint)"
	@echo "  make discover       - Quick STUN discovery test"
	@echo "  make version        - Show version"
	@echo "  make help           - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build && ./bin/altair discover"
	@echo "  make examples && ./bin/examples/simple-p2p discover"
	@echo "  make serve  # Start signaling server"
	@echo "  make chat ARGS=\"--username Alice --listen :9000\"  # Start chat in responder mode"
	@echo "  make chat ARGS=\"--username Bob --peer 127.0.0.1:9000\"  # Connect to peer"
	@echo "  make chat ARGS=\"--username Alice --room my-room --signaling ws://localhost:8080/ws\"  # Use signaling server"
