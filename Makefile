.PHONY: build test lint install clean help

# Variables
BINARY_NAME=shiki
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)"

# Build binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/shiki
	@echo "Build complete: ./$(BINARY_NAME)"

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Lint code
lint:
	@echo "Running linters..."
	golangci-lint run

# Install binary to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) ./cmd/shiki

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out
	rm -rf vendor/
	go clean

# Help
help:
	@echo "Available targets:"
	@echo "  build    - Build binary with version info"
	@echo "  test     - Run tests with race detector"
	@echo "  lint     - Run golangci-lint"
	@echo "  install  - Install to GOPATH/bin"
	@echo "  clean    - Remove build artifacts"
