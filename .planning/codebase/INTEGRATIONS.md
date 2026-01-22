# External Integrations

**Analysis Date:** 2026-01-22

## APIs & External Services

**Local Network Discovery:**
- mDNS (Multicast DNS) - Service discovery on local network
  - SDK/Client: github.com/hashicorp/mdns v1.0.5
  - Service name: `_syncdev._tcp`
  - Implementation: `internal/network/discovery.go`
  - Broadcast: All local IP addresses (IPv4/IPv6) to enable peer detection

**Peer-to-Peer Networking:**
- Custom TCP protocol for file synchronization
  - Protocol version: "1.0" (`internal/network/protocol.go`)
  - Message types: hello, pairing_request, sync_request, file_request, file_chunk, delete_file, etc.
  - Chunk size: 1MB for file transfers
  - Connection port: 52525 (configurable)
  - Implementation: `internal/network/server.go`, `internal/network/client.go`

## Data Storage

**Local File Storage:**
- File system only (no external database)
- User's home directory: `~/.syncdev/`
  - `config.json` - Application and peer configuration
  - `indices/` - Per-folder sync state indexes

**Configuration Structure:**
- Device ID, name, port
- Paired peer list with shared secrets
- Folder pair definitions (local/remote paths, exclusions)
- Global exclusion patterns

**Sync Index:**
- Location: `~/.syncdev/indices/`
- Purpose: Track file hashes and timestamps for change detection
- Format: Index manager stores file metadata for delta sync
  - Implementation: `internal/sync/index.go`

**File Storage:**
- No remote cloud storage
- Peer's local file system accessed via network protocol

## Authentication & Identity

**Auth Provider:**
- Custom local authentication
  - Implementation: `internal/network/server.go` (HMAC-SHA256 pairing)
  - Pairing flow: 6-digit code-based device verification
  - Shared secret: Generated during pairing, stored in peer config

**Pairing Process:**
- Code generation: `app.go` - GeneratePairingCode()
- Code validation: `internal/network/server.go` - handlePairingRequest()
- Shared secret: HMAC-SHA256 based on code (see `server.go` lines 6-7)
- Storage: Peer entry includes `SharedSecret` and `Paired` boolean

**Device Identity:**
- Device ID: UUID v4, generated once at first startup
- Device Name: User-configurable
- Both stored in `~/.syncdev/config.json`

## Monitoring & Observability

**Error Tracking:**
- None - All errors logged to stdout/stderr
- Error events emit via event system to UI (see `app.go` line 64)

**Logs:**
- Standard Go log package (`log` import in most files)
- Logs to console (visible in terminal when running `make dev`)
- Recent sync events stored in memory in engine
  - Structure: `SyncEvent` type in `internal/sync/engine.go`
  - Types: "push", "pull", "delete", "error"
  - Exposed via: `GetRecentEvents()` in `app.go`

**Event System:**
- Wails runtime events for UI updates:
  - `sync:status` - Sync status changes
  - `sync:progress` - File transfer progress
  - `sync:event` - Individual sync events (push/pull/delete)
  - `peers:changed` - Peer discovery updates
  - Implementation: `app.go` lines 52-69

## CI/CD & Deployment

**Hosting:**
- Standalone desktop application
- macOS distribution via DMG installer
- Binary packaging: Universal binary (Intel + Apple Silicon)

**Build Pipeline:**
- Makefile-based build (no external CI service detected)
- Targets: `make build`, `make build-universal`, `make dmg`
- DMG creation: `scripts/create-dmg.sh`

**Distribution:**
- DMG installer for macOS
- Direct .app installation to /Applications
- Version: 1.0.0 (defined in `Makefile` and `wails.json`)

## Environment Configuration

**Required env vars:**
- None detected - Application is self-contained
- All configuration stored in `~/.syncdev/config.json`

**Secrets location:**
- `~/.syncdev/config.json` contains:
  - Paired peers with shared secrets
  - Device ID and name
  - Folder pair local paths

**Configuration Management:**
- File-based persistence via `internal/config/store.go`
- Update callbacks ensure atomic writes
- Default exclusions for common files: `.DS_Store`, `.git`, `node_modules`, etc.

## Network Protocol Details

**Connection Handshake:**
1. Server listens on port 52525
2. Client connects and sends HELLO message with device ID/name
3. Server responds with device info
4. Pairing request includes 6-digit code
5. Receiving peer validates code against local state
6. Both peers exchange shared secret (HMAC-SHA256)

**Message Format:**
- JSON-encoded protocol messages
- Fields: type, timestamp, payload, HMAC
- HMAC calculated over message using shared secret for authentication
- Implementation: `internal/network/protocol.go`

**File Transfer Protocol:**
- File request initiates transfer
- File sent in 1MB chunks (ChunkSize constant)
- Checksum/completion message confirms receipt
- Delete operations with acknowledgment
- Folder pair sync for configuration propagation

## Webhooks & Callbacks

**Incoming:**
- TCP connections from peers (server accepts on port 52525)
- mDNS announcements from peer devices

**Outgoing:**
- TCP connections to peer devices for file sync
- mDNS advertisements of this device
- Event emissions to UI (Wails runtime events)

**Internal Callbacks:**
- `SetStatusCallback()` - Sync status changes
- `SetProgressCallback()` - Transfer progress updates
- `SetEventCallback()` - Individual sync events
- `SetPeerChangeCallback()` - Peer discovery changes
  - All set in `app.go` lines 52-69

---

*Integration audit: 2026-01-22*
