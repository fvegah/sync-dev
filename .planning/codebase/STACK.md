# Technology Stack

**Analysis Date:** 2026-01-22

## Languages

**Primary:**
- Go 1.23 - Backend application, sync engine, peer networking
- JavaScript (ES6+) - Frontend UI and Wails bindings

**Secondary:**
- Svelte 3.49.0 - Component framework for reactive UI
- CSS3 - Styling (custom, no CSS framework detected)

## Runtime

**Environment:**
- macOS (primary platform, see `Makefile` for Darwin targets)
- Wails v2.11.0 - Go/JavaScript desktop application framework
- Embedded web view for frontend rendering

**Package Manager:**
- Go: Module-based (go.mod/go.sum)
- Node.js: npm (frontend dependencies managed in `frontend/package.json`)

## Frameworks

**Core:**
- Wails v2.11.0 - Desktop application framework combining Go backend with JS/Svelte frontend
  - Provides IPC bridge between Go and JavaScript
  - Asset embedding for frontend resources
  - Native macOS integration (titlebar, appearance, about dialog)

**Frontend:**
- Svelte 3.49.0 - Reactive component framework
- Vite 3.0.7 - Build tool and dev server
- @sveltejs/vite-plugin-svelte 1.0.1 - Svelte integration with Vite

**Networking:**
- hashicorp/mdns v1.0.5 - mDNS service discovery for local peer detection
- Custom TCP protocol implementation for peer-to-peer file transfer

**Storage:**
- File-based JSON configuration stored in `~/.syncdev/` directory
- Index files stored in `~/.syncdev/indices/` for sync state tracking

## Key Dependencies

**Critical:**
- github.com/wailsapp/wails/v2 v2.11.0 - Desktop framework backbone
- github.com/hashicorp/mdns v1.0.5 - Peer discovery on local network
- github.com/google/uuid v1.6.0 - Device and peer ID generation
- github.com/gobwas/glob v0.2.3 - File exclusion pattern matching

**Infrastructure:**
- fyne.io/systray v1.12.0 - System tray integration (currently disabled due to macOS/Wails conflict; see `main.go` line 36)
- github.com/miekg/dns v1.1.41 - DNS protocol support for mDNS

**Utilities:**
- github.com/google/uuid v1.6.0 - UUID generation
- github.com/samber/lo v1.49.1 - Go utilities/helpers
- golang.org/x/crypto v0.33.0 - Cryptography (HMAC-SHA256 for pairing)
- golang.org/x/net v0.35.0 - Network utilities

## Configuration

**Environment:**
- No .env files required
- Application data: `~/.syncdev/` (user's home directory, see `app.go` line 380)
- Default service port: 52525 (defined in `internal/config/config.go`)
- mDNS service name: `_syncdev._tcp`

**Build:**
- `wails.json` - Wails configuration
- `Makefile` - Build targets for development, universal/arm64/amd64, DMG distribution
- `frontend/vite.config.js` - Vite build configuration
- `frontend/package.json` - npm dependencies

**Development:**
- `make dev` - Hot reload development mode
- `make build` - Standard build
- `make build-universal` - Intel + Apple Silicon universal binary
- `make dmg` - Create distribution DMG file

## Platform Requirements

**Development:**
- Go 1.23+
- Node.js/npm (for frontend)
- Wails CLI (install via `go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- macOS (current build targets)

**Production:**
- macOS 10.11+ (typical Wails requirement)
- Deployment via DMG installer or direct .app installation to /Applications

## Runtime Ports

**Network:**
- TCP port 52525 (default) - Peer-to-peer connection server (`internal/network/server.go`)
- mDNS multicast (port 5353) - Service discovery via hashicorp/mdns

---

*Stack analysis: 2026-01-22*
