# Codebase Structure

**Analysis Date:** 2026-01-22

## Directory Layout

```
sync-dev/
├── app.go                      # Application struct with public API methods
├── main.go                     # Wails app bootstrap and entry point
├── systray.go                  # System tray menu (disabled on macOS)
├── go.mod                      # Go module definition
├── go.sum                      # Go dependency checksums
├── wails.json                  # Wails framework configuration
├── frontend/                   # Svelte frontend application
│   ├── src/
│   │   ├── App.svelte         # Root component with navigation tabs
│   │   ├── main.js            # Frontend entry point
│   │   ├── lib/               # Reusable Svelte components
│   │   │   ├── PeerList.svelte      # Device discovery and pairing UI
│   │   │   ├── FolderPairs.svelte   # Folder sync pair management UI
│   │   │   ├── SyncStatus.svelte    # Sync progress and event display
│   │   │   └── Settings.svelte      # Configuration UI
│   │   ├── stores/            # Svelte reactive stores
│   │   │   └── app.js         # Global application state (peers, folders, sync status)
│   │   └── assets/            # Static assets (images, fonts)
│   ├── wailsjs/               # Auto-generated Wails bindings to Go methods
│   ├── dist/                  # Built frontend output
│   ├── node_modules/          # npm dependencies
│   ├── package.json           # Node.js dependencies and build scripts
│   └── jsconfig.json          # JavaScript configuration
├── internal/                   # Private Go packages
│   ├── config/                # Configuration management
│   │   ├── config.go          # Config struct definition and helpers
│   │   └── store.go           # Persistent config store with thread-safety
│   ├── models/                # Shared data structures
│   │   ├── peer.go            # Peer, PairingRequest, PairingResponse types
│   │   └── file.go            # File operations: FileInfo, FolderPair, SyncAction, TransferProgress
│   ├── network/               # Network communication layer
│   │   ├── discovery.go       # mDNS-based peer discovery
│   │   ├── server.go          # TCP server accepting peer connections
│   │   ├── client.go          # TCP client for initiating peer connections
│   │   └── protocol.go        # Message format and encoding
│   └── sync/                  # File synchronization engine
│       ├── engine.go          # Main SyncEngine orchestrator
│       ├── scanner.go         # Directory scanner with exclusion patterns
│       ├── index.go           # File index management and comparison
│       └── transfer.go        # File transfer operations and FileReceiver
├── build/                     # Build outputs (platform-specific)
│   ├── darwin/                # macOS build artifacts
│   ├── windows/               # Windows build artifacts
│   └── bin/                   # Compiled application binaries
├── scripts/                   # Build and utility scripts
└── .planning/                 # GSD planning documents
    └── codebase/              # Architecture and structure docs
```

## Directory Purposes

**Root Level:**
- Purpose: Go application entry point and Wails configuration
- Contains: Main application files (app.go, main.go), module definition, configuration
- Key files: `app.go` (API), `main.go` (bootstrap), `wails.json` (framework config)

**frontend/**
- Purpose: Svelte UI application and build pipeline
- Contains: Source files, components, stores, built assets, dependencies
- Key files: `src/App.svelte` (root), `src/stores/app.js` (state management)

**frontend/src/lib/**
- Purpose: Reusable UI components with specific functionality
- Contains: Four main tab components for UI sections
- Key files: Component files for each feature area

**frontend/src/stores/**
- Purpose: Svelte reactive state containers
- Contains: Writable stores for app state management
- Key files: `app.js` (all global stores)

**internal/config/**
- Purpose: Configuration persistence and management
- Contains: Config data structure, thread-safe store, IO operations
- Key files: `config.go` (structure), `store.go` (persistence)

**internal/models/**
- Purpose: Shared type definitions used across layers
- Contains: Peer, File, Folder, Transfer data structures
- Key files: `peer.go`, `file.go`

**internal/network/**
- Purpose: Network communication and peer discovery
- Contains: mDNS discovery, TCP server/client, protocol handling
- Key files: `discovery.go` (mDNS), `server.go` (listener), `client.go` (connections)

**internal/sync/**
- Purpose: File synchronization orchestration
- Contains: Main engine, filesystem scanning, file indexing, transfer logic
- Key files: `engine.go` (orchestrator), `scanner.go` (filesystem), `index.go` (tracking)

**build/**
- Purpose: Platform-specific compiled outputs
- Contains: macOS app bundles, Windows installers, binaries
- Generated: Yes, committed: No

## Key File Locations

**Entry Points:**
- `main.go`: Go application bootstrap - initializes Wails framework
- `app.go`: Application struct with all public methods exported to frontend
- `frontend/src/main.js`: Svelte app initialization
- `frontend/src/App.svelte`: Root UI component with tab navigation

**Configuration:**
- `wails.json`: Wails framework configuration (frontend build, dev server)
- `go.mod`: Go dependencies (Wails, mDNS, UUID, glob)
- `frontend/package.json`: Node.js dependencies (Svelte, Vite)
- `internal/config/config.go`: Default configuration constants

**Core Logic:**
- `app.go`: Public API methods called from frontend
- `internal/sync/engine.go`: Main sync orchestration
- `internal/network/discovery.go`: Peer discovery via mDNS
- `internal/network/server.go`: Incoming peer connections
- `internal/config/store.go`: Persistent configuration

**Testing:**
- No test files present in repository

**Data Storage:**
- `~/.syncdev/config.json`: Persistent configuration (created at runtime)
- `~/.syncdev/indices/`: File indices per folder pair (created at runtime)

## Naming Conventions

**Files:**
- Go source: `lowercase_with_underscores.go` (e.g., `scanner.go`, `discovery.go`)
- Svelte components: `PascalCase.svelte` (e.g., `PeerList.svelte`)
- JavaScript modules: `lowercase.js` (e.g., `app.js`)
- Configuration: `lowercase.json` (e.g., `config.json`)

**Directories:**
- Private Go packages: `internal/` prefix for import encapsulation
- Feature-based: `sync/`, `network/`, `config/`, `models/` organize by domain
- Frontend: `src/` for source, `lib/` for components, `stores/` for state
- Generated: `wailsjs/` for auto-generated bindings, `dist/` for build output

**Functions & Types:**
- Go types: `PascalCase` (e.g., `Engine`, `Scanner`, `FileIndex`)
- Go functions: `camelCase` for private, `PascalCase` for exported (e.g., `Start()`, `GetPeers()`)
- Svelte scripts: `camelCase` for functions and variables (e.g., `setTab()`, `submitPairing()`)
- Constants: `ALL_CAPS` in Go (e.g., `StatusIdle`, `ServiceName`), UPPERCASE in JavaScript

**Variables & Properties:**
- Go: `camelCase` for private, `PascalCase` for exported struct fields with JSON tags
- Svelte: `camelCase` for all variables and properties
- JSON: `camelCase` in serialized format (enforced via struct tags)

## Where to Add New Code

**New Feature (full stack):**

1. **Data Model**: Add struct to `internal/models/` (e.g., `internal/models/newfeature.go`)
2. **Backend Logic**: Add package under `internal/` (e.g., `internal/newfeature/newfeature.go`)
3. **Public API**: Add methods to `App` struct in `app.go` to expose to frontend
4. **Frontend Component**: Create in `frontend/src/lib/` (e.g., `NewFeature.svelte`)
5. **Frontend State**: Add store to `frontend/src/stores/app.js` if needed
6. **Integration**: Call App methods from component using Wails-generated bindings

**New Sync-Related Feature:**
- Core logic: `internal/sync/` package
- Integration: Add methods to `Engine` struct, callback registration in `app.go`
- Events: Create new SyncEvent type or action in `internal/models/`
- Frontend: Create component importing from Wails bindings and subscribing to stores

**New Component/Module:**
- Implementation: Create `internal/[module]/[module].go`
- Integration: Instantiate in `NewEngine()` or `NewApp()` as needed
- Public API: Only export if needed by frontend; most internal modules stay private

**Utilities & Helpers:**
- Shared utilities: Add to respective package (e.g., string helpers in sync package near usage)
- Path utilities: Add to config or models
- Format functions: Add to app.go as utility methods (e.g., FormatBytes, FormatDuration)

**Frontend Utilities:**
- UI helpers: Add to component files or create `frontend/src/lib/utils.js`
- Store helpers: Extend `frontend/src/stores/app.js` with additional derived stores or helpers

## Special Directories

**wailsjs/:**
- Purpose: Auto-generated Wails bindings from Go methods
- Generated: Yes, automatically by Wails during build
- Committed: Yes, committed to simplify development without build step
- Update: Regenerated when Go API methods in `app.go` change

**dist/:**
- Purpose: Built frontend output (HTML, CSS, JS bundles)
- Generated: Yes, by Vite build system
- Committed: No, generated during `npm run build`
- Embedded: Frontend dist is embedded in Go binary via `//go:embed`

**build/bin/:**
- Purpose: Compiled application binaries and app bundles
- Generated: Yes, by Wails build system
- Committed: No, generated during `wails build`
- Contents: SyncDev.app for macOS, executable for Windows

**node_modules/:**
- Purpose: npm dependencies for frontend
- Generated: Yes, by `npm install`
- Committed: No, managed by package-lock.json
- Usage: Required for development and build

**~/.syncdev/ (at runtime):**
- Purpose: Application data directory
- Generated: Yes, created on first run
- Contains: `config.json` (configuration), `indices/` (file indices)

---

*Structure analysis: 2026-01-22*
