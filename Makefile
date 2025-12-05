.PHONY: all build test test-layer1 test-layer2 test-coverage clean run-discover run-xor-demo help-connect fmt lint help

# Default target
all: build test

# Build the CLI binary
build:
	@echo "Building altair..."
	@go build -o altair ./cmd/main
	@echo "✓ Binary created: ./altair"

# Run tests
test:
	@echo "Running all tests..."
	@go test ./pkg/... -v

# Test only Layer 1 (STUN)
test-layer1:
	@echo "Running Layer 1 (STUN) tests..."
	@go test ./pkg/stun -v

# Test only Layer 2 (Hole Punching)
test-layer2:
	@echo "Running Layer 2 (Hole Punching) tests..."
	@go test ./pkg/holepunch -v

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test ./pkg/... -cover -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f altair coverage.out coverage.html
	@echo "✓ Clean complete"

# Run the discover command
run-discover: build
	@echo "Running STUN discovery..."
	@./altair discover

# Run the XOR encoding demo
run-xor-demo:
	@echo "Running XOR encoding demonstration..."
	@go run examples/xor-demo.go

# Show connect command help
help-connect: build
	@echo "Showing connect command help..."
	@./altair connect --help

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Formatting complete"

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@golangci-lint run
	@echo "✓ Linting complete"

# Show help
help:
	@echo "Available targets:"
	@echo "  make build          - Build the altair binary"
	@echo "  make test           - Run all unit tests"
	@echo "  make test-layer1    - Run Layer 1 (STUN) tests only"
	@echo "  make test-layer2    - Run Layer 2 (Hole Punching) tests only"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make run-discover   - Run STUN discovery"
	@echo "  make run-xor-demo   - Run XOR encoding demo"
	@echo "  make help-connect   - Show connect command help"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Run linter (requires golangci-lint)"
	@echo "  make help           - Show this help message"
