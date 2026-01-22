.PHONY: all build build-universal dev clean dmg install

# Project settings
APP_NAME := SyncDev
VERSION := 1.0.0
BUILD_DIR := build/bin

# Wails commands
WAILS := wails

all: build

# Development mode with hot reload
dev:
	$(WAILS) dev

# Build for current platform
build:
	$(WAILS) build

# Build universal binary (Intel + Apple Silicon)
build-universal:
	$(WAILS) build -platform darwin/universal

# Build for specific platforms
build-arm64:
	$(WAILS) build -platform darwin/arm64

build-amd64:
	$(WAILS) build -platform darwin/amd64

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f *.dmg

# Create DMG for distribution
dmg: build-universal
	@echo "Creating DMG..."
	@./scripts/create-dmg.sh

# Install to /Applications
install: build
	@echo "Installing $(APP_NAME) to /Applications..."
	@rm -rf /Applications/$(APP_NAME).app
	@cp -R $(BUILD_DIR)/$(APP_NAME).app /Applications/
	@echo "$(APP_NAME) installed successfully!"

# Run the built app
run: build
	@open $(BUILD_DIR)/$(APP_NAME).app

# Generate Go bindings only
bindings:
	$(WAILS) generate

# Help
help:
	@echo "SyncDev - Folder Sync for Mac"
	@echo ""
	@echo "Available targets:"
	@echo "  make dev            - Start development mode with hot reload"
	@echo "  make build          - Build for current platform"
	@echo "  make build-universal- Build universal binary (Intel + Apple Silicon)"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make dmg            - Create DMG for distribution"
	@echo "  make install        - Install to /Applications"
	@echo "  make run            - Build and run the app"
	@echo "  make help           - Show this help message"
