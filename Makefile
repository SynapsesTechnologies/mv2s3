.PHONY: build test clean install lint fmt vet

# Binary name
BINARY_NAME=mv2s3

# Build directory
BUILD_DIR=bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Build the binary
build:
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/mv2s3

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Install dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Install binary
install:
	$(GOBUILD) -o $(GOPATH)/bin/$(BINARY_NAME) ./cmd/mv2s3

# Format code
fmt:
	$(GOFMT) -s -w .

# Vet code
vet:
	$(GOVET) ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Run all checks
check: fmt vet lint test

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/mv2s3
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/mv2s3
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/mv2s3

# Development build with race detection
dev:
	$(GOBUILD) -race -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/mv2s3

# Show help
help:
	@echo "Available targets:"
	@echo "  build        Build the binary"
	@echo "  test         Run tests"
	@echo "  test-coverage Run tests with coverage"
	@echo "  clean        Clean build artifacts"
	@echo "  deps         Download dependencies"
	@echo "  install      Install binary to GOPATH/bin"
	@echo "  fmt          Format code"
	@echo "  vet          Vet code"
	@echo "  lint         Lint code"
	@echo "  check        Run all checks (fmt, vet, lint, test)"
	@echo "  build-all    Build for multiple platforms"
	@echo "  dev          Development build with race detection"
	@echo "  help         Show this help"
