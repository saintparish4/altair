.PHONY: all build test test-layer1 test-layer2 test-chat test-transfer test-coverage clean run-discover run-xor-demo help-connect build-chat build-transfer fmt lint help

# Default target
all: build test

# Build the CLI binary
build:
	@echo "Building altair..."
	@cd backend && go build -o ../altair ./cmd/main
	@echo "✓ Binary created: ./altair"

# Run tests
test:
	@echo "Running all tests..."
	@cd backend && go test ./pkg/... -v

# Test only Layer 1 (STUN)
test-layer1:
	@echo "Running Layer 1 (STUN) tests..."
	@cd backend && go test ./pkg/stun -v

# Test only Layer 2 (Hole Punching)
test-layer2:
	@echo "Running Layer 2 (Hole Punching) tests..."
	@cd backend && go test ./pkg/holepunch -v

# Test chat package
test-chat:
	@echo "Running chat package tests..."
	@cd backend && go test ./pkg/chat -v

# Test transfer package
test-transfer:
	@echo "Running transfer package tests..."
	@cd backend && go test ./pkg/transfer -v

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@cd backend && go test ./pkg/... -cover -coverprofile=../coverage.out
	@cd backend && go tool cover -html=../coverage.out -o ../coverage.html
	@echo "✓ Coverage report: coverage.html"

# Build chat binary
build-chat:
	@echo "Building altair-chat..."
	@cd backend && go build -o ../altair-chat ./cmd/chat
	@echo "✓ Binary created: ./altair-chat"

# Build transfer binary
build-transfer:
	@echo "Building altair-transfer..."
	@cd backend && go build -o ../altair-transfer ./cmd/transfer
	@echo "✓ Binary created: ./altair-transfer"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f altair altair-chat altair-transfer coverage.out coverage.html
	@echo "✓ Clean complete"

# Run the discover command
run-discover: build
	@echo "Running STUN discovery..."
	@./altair discover

# Run the XOR encoding demo
run-xor-demo:
	@echo "Running XOR encoding demonstration..."
	@cd backend && go run examples/xor-demo.go

# Show connect command help
help-connect: build
	@echo "Showing connect command help..."
	@./altair connect --help

# Format code
fmt:
	@echo "Formatting code..."
	@cd backend && go fmt ./...
	@echo "✓ Formatting complete"

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@cd backend && golangci-lint run
	@echo "✓ Linting complete"

# Show help
help:
	@echo "Available targets:"
	@echo "  make build          - Build the altair binary"
	@echo "  make test           - Run all unit tests"
	@echo "  make test-layer1    - Run Layer 1 (STUN) tests only"
	@echo "  make test-layer2    - Run Layer 2 (Hole Punching) tests only"
	@echo "  make test-chat      - Run chat package tests only"
	@echo "  make test-transfer  - Run transfer package tests only"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make build-chat     - Build the altair-chat binary"
	@echo "  make build-transfer  - Build the altair-transfer binary"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make run-discover   - Run STUN discovery"
	@echo "  make run-xor-demo   - Run XOR encoding demo"
	@echo "  make help-connect   - Show connect command help"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Run linter (requires golangci-lint)"
	@echo "  make help           - Show this help message"
