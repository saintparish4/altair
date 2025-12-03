.PHONY: all build test clean run-discover run-xor-demo help

# Default target
all: build test

# Build the CLI binary
build:
	@echo "Building altair..."
	@go build -o altair ./cmd/main
	@echo "✓ Binary created: ./altair"

# Run tests
test:
	@echo "Running tests..."
	@go test ./pkg/... -v

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
	@echo "  make test           - Run unit tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make run-discover   - Run STUN discovery"
	@echo "  make run-xor-demo   - Run XOR encoding demo"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Run linter (requires golangci-lint)"
	@echo "  make help           - Show this help message"