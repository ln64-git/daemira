.PHONY: build install clean test run dev help start stop

# Binary name
BINARY_NAME=daemira
INSTALL_PATH=/usr/local/bin

# Build variables
BUILD_DIR=bin
GO_FILES=$(shell find . -name '*.go' -type f)

# Build the binary
build: $(GO_FILES)
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Install the binary to system
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@sudo install -Dm755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installation complete!"

# Run the binary
run: build
	@$(BUILD_DIR)/$(BINARY_NAME)

# Run in development mode (no build, direct execution)
dev:
	@echo "Running in development mode..."
	@go run main.go $(ARGS)

# Run with specific command
run-status: build
	@$(BUILD_DIR)/$(BINARY_NAME) status

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean
	@echo "Clean complete!"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Start daemon (both system updates and Google Drive sync)
start:
	@./scripts/start-daemira.sh

# Stop daemon
stop:
	@./scripts/stop-daemira.sh

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  dev          - Run in development mode (go run)"
	@echo "  dev ARGS=... - Run dev mode with arguments (e.g., 'make dev ARGS=status')"
	@echo "  install      - Install binary to system"
	@echo "  run          - Build and run the binary"
	@echo "  run-status   - Build and run 'daemira status'"
	@echo "  start        - Start daemon (system updates + Google Drive sync)"
	@echo "  stop         - Stop daemon"
	@echo "  clean        - Remove build artifacts"
	@echo "  test         - Run tests"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  help         - Show this help message"

# Default target
.DEFAULT_GOAL := build
