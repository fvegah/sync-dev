package tray

import (
	_ "embed"
)

// Template icons for macOS menu bar
// These are black + transparent PNGs, 22x22 pixels
// macOS automatically adapts them for light/dark mode
// Icons created by generate.go in internal/tray/icons/

//go:embed icons/tray-idle.png
var IconIdle []byte

//go:embed icons/tray-syncing.png
var IconSyncing []byte

//go:embed icons/tray-error.png
var IconError []byte
