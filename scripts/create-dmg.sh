#!/bin/bash

# SyncDev DMG Creation Script
# This script creates a distributable DMG file

set -e

APP_NAME="SyncDev"
VERSION="1.0.0"
DMG_NAME="${APP_NAME}-${VERSION}.dmg"
APP_PATH="build/bin/${APP_NAME}.app"
DMG_TEMP="build/dmg"

# Check if app exists
if [ ! -d "$APP_PATH" ]; then
    echo "Error: $APP_PATH not found. Run 'make build-universal' first."
    exit 1
fi

# Check if create-dmg is installed
if ! command -v create-dmg &> /dev/null; then
    echo "create-dmg not found. Installing via Homebrew..."
    brew install create-dmg
fi

# Clean up previous DMG
rm -f "$DMG_NAME"
rm -rf "$DMG_TEMP"

# Create temp directory
mkdir -p "$DMG_TEMP"

echo "Creating DMG for $APP_NAME v$VERSION..."

# Create DMG
create-dmg \
    --volname "$APP_NAME" \
    --volicon "build/appicon.icns" \
    --window-pos 200 120 \
    --window-size 600 400 \
    --icon-size 100 \
    --icon "$APP_NAME.app" 150 190 \
    --app-drop-link 450 190 \
    --hide-extension "$APP_NAME.app" \
    "$DMG_NAME" \
    "$APP_PATH" || {
        # Fallback if create-dmg fails (e.g., no icns file)
        echo "Trying simpler DMG creation..."
        hdiutil create -volname "$APP_NAME" -srcfolder "$APP_PATH" -ov -format UDZO "$DMG_NAME"
    }

# Clean up
rm -rf "$DMG_TEMP"

echo ""
echo "DMG created: $DMG_NAME"
echo ""
echo "To distribute:"
echo "1. Optionally sign with: codesign --force --deep --sign 'Developer ID' $APP_PATH"
echo "2. Optionally notarize with Apple for Gatekeeper approval"
echo "3. Share the DMG file"
