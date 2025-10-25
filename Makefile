BINARY_NAME=dms
BINARY_NAME_INSTALL=dankinstall
SOURCE_DIR=cmd/dms
SOURCE_DIR_INSTALL=cmd/dankinstall
BUILD_DIR=bin
PREFIX ?= /usr/local
INSTALL_DIR=$(PREFIX)/bin

GO=go
GOFLAGS=-ldflags="-s -w"

# Version and build info
VERSION=$(shell git describe --tags --always 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

BUILD_LDFLAGS=-ldflags='-s -w -X main.Version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.commit=$(COMMIT)'

# Architecture to build for dist target (amd64, arm64, or all)
ARCH ?= all

.PHONY: all build dankinstall dist clean install uninstall test fmt vet deps help

# Default target
all: build

# Build the main binary (dms)
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build $(BUILD_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(SOURCE_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

dankinstall:
	@echo "Building $(BINARY_NAME_INSTALL)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build $(BUILD_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_INSTALL) ./$(SOURCE_DIR_INSTALL)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME_INSTALL)"

# Build distro binaries for amd64 and arm64 (Linux only, no update/greeter support)
dist:
ifeq ($(ARCH),all)
	@echo "Building $(BINARY_NAME) for distribution (amd64 and arm64)..."
	@mkdir -p $(BUILD_DIR)
	@echo "Building for linux/amd64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -tags distro_binary $(BUILD_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(SOURCE_DIR)
	@echo "Building for linux/arm64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -tags distro_binary $(BUILD_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./$(SOURCE_DIR)
	@echo "Distribution builds complete:"
	@echo "  $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"
	@echo "  $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64"
else
	@echo "Building $(BINARY_NAME) for distribution ($(ARCH))..."
	@mkdir -p $(BUILD_DIR)
	@echo "Building for linux/$(ARCH)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) $(GO) build -tags distro_binary $(BUILD_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-$(ARCH) ./$(SOURCE_DIR)
	@echo "Distribution build complete:"
	@echo "  $(BUILD_DIR)/$(BINARY_NAME)-linux-$(ARCH)"
endif

build-all: build dankinstall

install: build-all
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installing $(BINARY_NAME_INSTALL) to $(INSTALL_DIR)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME_INSTALL) $(INSTALL_DIR)/$(BINARY_NAME_INSTALL)
	@chmod +x $(INSTALL_DIR)/$(BINARY_NAME_INSTALL)
	@echo "Installation complete"

uninstall:
	@echo "Uninstalling $(BINARY_NAME) from $(INSTALL_DIR)..."
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Uninstalling $(BINARY_NAME_INSTALL) from $(INSTALL_DIR)..."
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME_INSTALL)
	@echo "Uninstall complete"

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

test:
	@echo "Running tests..."
	$(GO) test -v ./...

fmt:
	@echo "Formatting Go code..."
	$(GO) fmt ./...

vet:
	@echo "Running go vet..."
	$(GO) vet ./...

deps:
	@echo "Updating dependencies..."
	$(GO) mod tidy
	$(GO) mod download

dev:
	@echo "Building $(BINARY_NAME) for development..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(SOURCE_DIR)
	@echo "Development build complete: $(BUILD_DIR)/$(BINARY_NAME)"

check-go:
	@echo "Checking Go version..."
	@go version | grep -E "go1\.(2[2-9]|[3-9][0-9])" > /dev/null || (echo "ERROR: Go 1.22 or higher required" && exit 1)
	@echo "Go version OK"

version: check-go
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Commit: $(COMMIT)"

help:
	@echo "Available targets:"
	@echo "  all          - Build the main binary (dms) (default)"
	@echo "  build        - Build the main binary (dms)"
	@echo "  dankinstall  - Build dankinstall binary"
	@echo "  dist         - Build dms for linux amd64/arm64 (no update/greeter)"
	@echo "                 Use ARCH=amd64 or ARCH=arm64 to build only one"
	@echo "  build-all    - Build both binaries"
	@echo "  install      - Install both binaries to $(INSTALL_DIR)"
	@echo "  uninstall    - Remove binaries from $(INSTALL_DIR)"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  fmt          - Format Go code"
	@echo "  vet          - Run go vet"
	@echo "  deps         - Update dependencies"
	@echo "  dev          - Build with debug info"
	@echo "  check-go     - Check Go version compatibility"
	@echo "  version      - Show version information"
	@echo "  help         - Show this help message"
