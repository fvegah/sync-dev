# Architecture

**Analysis Date:** 2026-01-22

## Pattern Overview

**Overall:** Desktop application using Wails (Go/Svelte) with peer-to-peer file synchronization

**Key Characteristics:**
- Three-tier architecture: UI layer (Svelte), application layer (Go), and network/sync layer
- Event-driven communication between frontend and backend via Wails runtime
- Distributed peer discovery using mDNS over local network
- State management through shared JSON configuration persisted to disk
- Callback-based architecture for status updates, progress tracking, and events

## Layers

**Frontend (UI Layer):**
- Purpose: Render user interface and handle user interactions
- Location: `frontend/src/`
- Contains: Svelte components, stores (Svelte writable stores), styling
- Depends on: Wails runtime for backend communication
- Used by: Application window display; invokes Go methods via Wails bindings

**Application/API Layer:**
- Purpose: Expose public API to frontend and coordinate sync operations
- Location: `app.go`
- Contains: App struct with exported methods for configuration, peers, folder pairs, sync operations
- Depends on: ConfigStore, SyncEngine, Wails runtime
- Used by: Frontend via Wails runtime bindings; receives events from SyncEngine

**Sync Engine Layer:**
- Purpose: Orchestrate file synchronization and peer lifecycle
- Location: `internal/sync/`
- Contains: Engine (main orchestrator), Scanner (filesystem scanning), IndexManager (file tracking), Transfer (file transfer operations)
- Depends on: ConfigStore, Network components (Server, Client, Discovery)
- Used by: Application layer (app.go); connects to Network layer

**Network Layer:**
- Purpose: Handle peer discovery, connections, and message protocol
- Location: `internal/network/`
- Contains: Discovery (mDNS), Server (TCP listener), Client (peer connections), Protocol (message format)
- Depends on: mDNS library (hashicorp/mdns), TCP/IP stack
- Used by: SyncEngine; responds to handlers set on Server

**Configuration Layer:**
- Purpose: Persist and manage application state
- Location: `internal/config/`
- Contains: Store (thread-safe wrapper), Config (data structure), constants
- Depends on: JSON marshaling
- Used by: All layers for reading/writing config; SyncEngine for peer and folder pair data

**Models Layer:**
- Purpose: Define shared data structures
- Location: `internal/models/`
- Contains: Peer, FileInfo, FolderPair, TransferProgress, SyncAction
- Used by: All layers for consistent data representation

## Data Flow

**Peer Discovery Flow:**

1. Application starts → SyncEngine.Start() → Discovery.Start()
2. Discovery.startAdvertising() publishes device via mDNS with device ID, name, port
3. Discovery.scanLoop() continuously queries for `_syncdev._tcp` services
4. On peer found → engine.handlePeerFound() creates Peer instance
5. engine.discovery.SetPeerFoundCallback() notifies Engine of new peer
6. Engine calls onPeerChange() callback → App emits "peers:changed" to frontend
7. Frontend receives event → updates peers store from GetPeers()

**Pairing Flow:**

1. User clicks "Pair" on remote peer in UI
2. Frontend calls App.RequestPairing(peerID, code)
3. App.RequestPairing() → SyncEngine.RequestPairing()
4. SyncEngine creates Client connection to peer, sends PairingRequest with code
5. Remote peer's Server receives message → handlers process pairing logic
6. Both peers validate code and establish SharedSecret
7. Paired peer stored in ConfigStore and propagated via event callbacks

**Sync Flow:**

1. User triggers SyncNow() or scheduler fires
2. App.SyncNow() → SyncEngine.SyncAllPairs()
3. Engine.SyncFolderPair(pairID):
   - Scanner.ScanDirectory(localPath) generates current FileIndex
   - IndexManager retrieves previous FileIndex
   - Compares to determine FileAction (push/pull/delete)
   - Creates SyncAction list
4. Engine contacts peer via Client connection with action list
5. Both peers execute transfer operations via Transfer component
6. FileReceiver writes incoming files; local files pushed to peer
7. Engine updates IndexManager with new state
8. Progress callbacks emit TransferProgress events to frontend
9. Completed sync event added to recent events list

**State Management:**

- **Configuration State**: ConfigStore held in App, persisted to `~/.syncdev/config.json`
- **Runtime State**: Engine holds connections, file receivers, peer list, sync status
- **UI State**: Frontend Svelte stores subscribe to backend event emissions
- **File Index State**: IndexManager persists per-folder-pair to `~/.syncdev/indices/`

## Key Abstractions

**Peer:**
- Purpose: Represents a remote device discoverable or paired
- Examples: `internal/models/peer.go`
- Pattern: Struct with ID, Name, Host, Port, PairingStatus, SharedSecret for encryption

**FolderPair:**
- Purpose: Represents a synced folder between local and remote
- Examples: `internal/models/file.go`
- Pattern: Struct with ID, PeerID, LocalPath, RemotePath, Enabled flag, per-pair Exclusions

**FileIndex:**
- Purpose: Snapshot of files in a folder at a point in time
- Examples: `internal/models/file.go`, `internal/sync/index.go`
- Pattern: Map of relative paths to FileInfo (hash, size, modtime, permissions)

**SyncEngine:**
- Purpose: Orchestrator combining discovery, file scanning, and transfer
- Examples: `internal/sync/engine.go`
- Pattern: Holds all components, manages lifecycle, exposes callbacks for status/progress

**Scanner:**
- Purpose: Filesystem traversal with exclusion pattern matching
- Examples: `internal/sync/scanner.go`
- Pattern: Uses glob library to match exclusion patterns; walks filesystem recursively

## Entry Points

**Application Entry Point:**
- Location: `main.go`
- Triggers: Go program start
- Responsibilities: Create Wails app with embedded frontend, bind Go methods, configure macOS window options

**Frontend Entry Point:**
- Location: `frontend/src/main.js`
- Triggers: Browser loads index.html
- Responsibilities: Initialize Svelte app, load root component (App.svelte), attach to DOM

**Sync Engine Startup:**
- Location: `app.go` startup() method
- Triggers: Wails OnStartup callback
- Responsibilities: Create config store, create sync engine, register callbacks, start engine

## Error Handling

**Strategy:** Hierarchical error propagation with logging and UI feedback

**Patterns:**

- **Config Errors**: Log to console, return early from startup, disable affected features
  - `config.NewStore()` failure → logs error, returns nil config, app continues but can't persist state

- **Network Errors**: Log error, update status to StatusError, emit error event
  - Connection failures during sync → addEvent(SyncEvent{Type: "error", Description: error})

- **File System Errors**: Skip files with access issues during scanning
  - filepath.Walk in Scanner skips on error rather than failing entire scan

- **Pairing Errors**: Return error from RequestPairing(), frontend displays in alert
  - Invalid code or peer unreachable → error message to user

- **Sync Engine Failures**: Set status to StatusError, emit event, allow retry
  - Transfer interrupted → create error event, reset to StatusIdle

## Cross-Cutting Concerns

**Logging:** Standard Go log package throughout; console output for development

**Validation:**
- Config: Sync interval constrained (1-60 mins), path existence checked before adding folder pairs
- Input: Pairing code verified, folder paths validated as directories

**Authentication:**
- Code-based pairing (6-digit code) generates SharedSecret for peer
- Messages authenticated via HMAC-SHA256 with SharedSecret
- Peer must be Paired flag = true for operations to proceed

**Concurrency Control:**
- ConfigStore uses RWMutex for thread-safe access
- Engine uses RWMutex for status/progress/events
- Connections map protected during add/remove operations
- Scheduler managed with channel-based shutdown

**Time Handling:**
- File ModTime used for change detection
- Sync events timestamped with time.Now()
- LastSeen and LastSyncTime tracked per peer
- Scheduler uses time.Ticker for periodic syncs

---

*Architecture analysis: 2026-01-22*
