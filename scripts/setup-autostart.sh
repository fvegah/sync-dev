#!/bin/bash

# SyncDev Auto-Start Setup Script
# This script installs or removes the LaunchAgent for starting SyncDev at login

PLIST_NAME="com.syncdev.agent.plist"
PLIST_SOURCE="$(dirname "$0")/$PLIST_NAME"
PLIST_DEST="$HOME/Library/LaunchAgents/$PLIST_NAME"

usage() {
    echo "SyncDev Auto-Start Setup"
    echo ""
    echo "Usage: $0 [install|uninstall|status]"
    echo ""
    echo "Commands:"
    echo "  install   - Enable SyncDev to start automatically at login"
    echo "  uninstall - Disable auto-start"
    echo "  status    - Check if auto-start is enabled"
}

install() {
    # Check if SyncDev is installed
    if [ ! -d "/Applications/SyncDev.app" ]; then
        echo "Error: SyncDev.app not found in /Applications"
        echo "Please install SyncDev first."
        exit 1
    fi

    # Create LaunchAgents directory if needed
    mkdir -p "$HOME/Library/LaunchAgents"

    # Copy plist
    cp "$PLIST_SOURCE" "$PLIST_DEST"

    # Load the agent
    launchctl load "$PLIST_DEST" 2>/dev/null || true

    echo "SyncDev auto-start enabled!"
    echo "SyncDev will now start automatically when you log in."
}

uninstall() {
    if [ -f "$PLIST_DEST" ]; then
        # Unload the agent
        launchctl unload "$PLIST_DEST" 2>/dev/null || true

        # Remove plist
        rm -f "$PLIST_DEST"

        echo "SyncDev auto-start disabled."
    else
        echo "Auto-start was not enabled."
    fi
}

status() {
    if [ -f "$PLIST_DEST" ]; then
        echo "Auto-start: ENABLED"
        echo "Plist location: $PLIST_DEST"
    else
        echo "Auto-start: DISABLED"
    fi
}

case "$1" in
    install)
        install
        ;;
    uninstall)
        uninstall
        ;;
    status)
        status
        ;;
    *)
        usage
        exit 1
        ;;
esac
