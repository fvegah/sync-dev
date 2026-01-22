# Coding Conventions

**Analysis Date:** 2026-01-22

## Naming Patterns

**Files:**
- Go package files: lowercase with underscores (e.g., `config.go`, `store.go`, `scanner.go`)
- Svelte components: PascalCase (e.g., `PeerList.svelte`, `FolderPairs.svelte`, `SyncStatus.svelte`)
- JavaScript modules: camelCase (e.g., `app.js`, `main.js`)
- Test files: No test files present in codebase (not currently practiced)

**Functions & Methods:**
- Go: PascalCase for exported functions, camelCase for unexported (standard Go convention)
  - Example exported: `NewEngine()`, `GetConfig()`, `SyncNow()`
  - Example unexported: `acceptLoop()`, `handleMessage()`, `startScheduler()`
- JavaScript/Svelte: camelCase for all functions
  - Example: `loadPeers()`, `addPair()`, `getStatusColor()`, `submitPairing()`

**Variables:**
- Go: camelCase for all variables (following Go conventions)
  - Example: `pairingCode`, `syncEngine`, `configStore`, `fileReceivers`
- JavaScript: camelCase
  - Example: `myPairingCode`, `selectedPeer`, `showAddForm`, `filteredPeers`

**Types & Constants:**
- Go types: PascalCase (e.g., `Engine`, `Config`, `Peer`, `SyncStatus`, `Message`)
- Go constants: UPPER_CASE or PascalCase depending on scope
  - Example: `StatusIdle`, `DefaultPort`, `ChunkSize`, `ProtocolVersion`
  - Example const types: `MsgTypeHello`, `FileActionPush`

**Struct Tags:**
- JSON tags: camelCase in serialized form (e.g., `json:"deviceId"`, `json:"folderPair"`)
- Export visibility: lowercase field names in Go, PascalCase for exported fields

## Code Style

**Formatting:**
- No explicit formatter configured (no .prettierrc or eslintrc in root)
- Go code follows standard `gofmt` conventions
- Svelte/JavaScript: No formal style enforcement

**Linting:**
- Go: No explicit linting configuration detected
- JavaScript: jsconfig.json present with strict settings (checkJs: true, importsNotUsedAsValues: error)

**Import Organization (Go):**

Order:
1. Standard library imports (e.g., `"context"`, `"fmt"`, `"log"`)
2. Third-party imports (e.g., `"github.com/wailsapp/wails/v2"`)
3. Internal imports (e.g., `"SyncDev/internal/config"`)

Example from `app.go`:
```go
import (
	"SyncDev/internal/config"
	"SyncDev/internal/models"
	"SyncDev/internal/sync"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)
```

**Import Organization (JavaScript):**

Typically organized as:
1. Svelte lifecycle imports (`import { onMount, onDestroy } from 'svelte'`)
2. Store imports (`import { currentTab, ... } from '../stores/app.js'`)
3. External API imports (`import { GetPeers, ... } from '../../wailsjs/go/main/App.js'`)
4. Runtime imports (`import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js'`)

## Error Handling

**Go Patterns:**

1. **Return errors explicitly:**
   - Functions return `error` as last return value
   - Check errors immediately after calls
   - Example from `config/store.go`:
   ```go
   store, err := config.NewStore()
   if err != nil {
       log.Printf("Failed to initialize config store: %v", err)
       return
   }
   ```

2. **Error wrapping:**
   - Use `fmt.Errorf()` with `%w` for error wrapping
   - Example from `internal/network/server.go`:
   ```go
   return fmt.Errorf("failed to start TCP server: %w", err)
   ```

3. **Error messaging:**
   - Prefix with descriptive context
   - Example: `"failed to parse config file: %w"`

4. **Nil checks:**
   - Check for nil before using pointers
   - Example from `app.go`:
   ```go
   if a.syncEngine == nil {
       return fmt.Errorf("sync engine not initialized")
   }
   ```

**JavaScript/Svelte Patterns:**

1. **Try-catch for async operations:**
   - Example from `FolderPairs.svelte`:
   ```javascript
   try {
       await AddFolderPair(selectedPeerId, localPath, remotePath);
       showAddForm = false;
       await loadData();
   } catch (err) {
       alert('Failed to add folder pair: ' + err);
   }
   ```

2. **Basic fallback handling:**
   - Uses alert() for user-facing errors
   - Empty arrays as fallback: `folderPairs.set(pairs || [])`

## Logging

**Framework:** `log` package (Go standard library) and `console` (JavaScript)

**Go Patterns:**

- Use `log.Printf()` for formatted messages with context
- Use `log.Println()` for simple messages
- No custom logger wrapper; direct `log` calls throughout

Example patterns:
```go
log.Printf("Failed to initialize config store: %v", err)
log.Println("SyncDev started successfully")
log.Printf("Sync engine started")
log.Printf("Peer connected: %s (%s)", conn.PeerName, conn.PeerID)
```

**When to log:**
- Initialization/startup/shutdown events
- Error conditions
- Major state transitions (peer connect/disconnect)
- Business logic milestones (sync started, completed)

**JavaScript Patterns:**
- Direct `console` logging not observed; errors handled via try-catch/alert
- Frontend favors event-driven updates over logging

## Comments

**When to Comment:**
- Top of file: brief description or purpose (Go files use comment above package declaration)
- Before exported functions: documentation comment explaining purpose
- Complex business logic: explain "why" not "what"
- TODO/FIXME: mark incomplete work or known issues

Example from `app.go`:
```go
// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
    // ... implementation
}
```

**JSDoc/TSDoc:**
- Not currently used in JavaScript/Svelte components
- Go uses simple comment conventions above exported functions

Example from `internal/sync/protocol.go`:
```go
// Message is the base structure for all protocol messages
type Message struct {
    Type      MessageType     `json:"type"`
    Timestamp int64           `json:"timestamp"`
    Payload   json.RawMessage `json:"payload,omitempty"`
    HMAC      string          `json:"hmac,omitempty"`
}
```

## Function Design

**Size:**
- Go functions are typically 10-50 lines
- Svelte functions are typically 5-30 lines
- Longer functions broken into helper functions (e.g., `ScanDirectory` in scanner.go is ~60 lines but well-structured)

**Parameters:**
- Go: Receivers on methods (pointer receivers for mutating operations)
  - Example: `func (s *Store) Update(fn func(*Config)) error`
- Function parameters passed in order: context first (when needed), then primary arguments
- JavaScript: positional parameters, sometimes with destructuring

**Return Values:**
- Go: Explicit error returns as last value
- JavaScript: Single returns or void (event-driven updates via stores)

**Callbacks in Go:**
- Use function types for callbacks
- Example from `sync/engine.go`:
```go
onStatusChange func(SyncStatus, string)
onProgress     func(*models.TransferProgress)
onEvent        func(*SyncEvent)
onPeerChange   func()
```

## Module Design

**Exports (Go):**

Packages export public types and functions via PascalCase naming:
- `internal/config/`: Config, Store, DefaultConfig(), NewStore()
- `internal/models/`: Peer, FileInfo, FolderPair, etc.
- `internal/sync/`: Engine, Scanner, IndexManager, SyncStatus, SyncEvent

**Barrel Files:**
- Not used in Go (packages are the module boundary)
- In JavaScript: stores export functions and writables (e.g., `export const peers = writable([])`)

**Package Organization:**
- `internal/config/`: Configuration and persistence
- `internal/models/`: Shared data structures
- `internal/network/`: Network protocol, discovery, server/client
- `internal/sync/`: Sync logic, file transfer, indexing
- Root level: Main app structure, UI bindings (app.go, main.go)

**Dependency Direction:**
- Higher levels depend on lower levels
- app.go depends on internal/sync, internal/config
- internal/sync depends on internal/models, internal/network
- No circular dependencies observed

---

*Convention analysis: 2026-01-22*
