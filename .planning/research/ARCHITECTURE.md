# Architecture Patterns: System Tray, Keychain, and Progress Tracking for Wails

**Domain:** Desktop app enhancement (SyncDev)
**Researched:** 2026-01-22
**Confidence:** MEDIUM-HIGH

## Executive Summary

This research addresses architectural integration of three capabilities into the existing SyncDev Wails app: system tray, macOS Keychain access, and enhanced progress tracking. The current architecture is event-driven (UI → App → SyncEngine → Network), which provides solid foundation for these enhancements.

**Key recommendations:**
1. **System Tray**: Stay on Wails v2 with workaround library, or migrate to Wails v3 alpha for native support
2. **Keychain**: Use zalando/go-keyring (no CGo) for cross-platform compatibility
3. **Progress Tracking**: Extend existing callback architecture with hierarchical progress aggregation

## Current Architecture Context

**Existing Pattern (from .planning/codebase/ARCHITECTURE.md):**
- Three-tier: UI (Svelte) → Application (app.go) → Sync/Network (internal/)
- Event-driven via Wails runtime.EventsEmit()
- Callbacks already exist: SetStatusCallback, SetProgressCallback, SetEventCallback
- State managed via ConfigStore (RWMutex-protected), Engine status/progress (RWMutex)

**Strengths to preserve:**
- Clean separation between UI and business logic
- Event-driven communication scales to multiple consumers
- Existing callback architecture is extensible

**Gaps to address:**
- No system tray presence (attempted with fyne.io/systray, conflicts with Wails)
- No secure credential storage (pairing codes, future auth tokens)
- Progress tracking is basic (single TransferProgress struct, no per-file granularity)

## Recommended Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                          System Tray                             │
│  (Native menu bar icon, click handlers, tooltip updates)        │
└────────────────────┬───────────────────────────────────┬─────────┘
                     │                                   │
                     │ (separate process IPC)            │ (show/hide window)
                     │                                   │
┌────────────────────▼───────────────────────────────────▼─────────┐
│                        Wails Runtime                             │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ UI Layer (Svelte)                                        │   │
│  │  • Progress components (per-file + aggregate displays)   │   │
│  │  • Theme-aware CSS (macOS native look)                   │   │
│  │  • Event subscribers (peers, progress, status)           │   │
│  └──────────────────────────────────────────────────────────┘   │
│                           ▲                                      │
│                           │ (runtime.EventsEmit)                 │
│                           │                                      │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Application Layer (app.go)                               │   │
│  │  • Wails method bindings                                 │   │
│  │  • Callback registration                                 │   │
│  │  • Window show/hide management                           │   │
│  └──────────────────────────────────────────────────────────┘   │
│                           ▲                                      │
│                           │ (method calls)                       │
│                           │                                      │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ SyncEngine (internal/sync/engine.go)                     │   │
│  │  • ProgressAggregator (NEW)                              │   │
│  │    - Collects per-file progress                          │   │
│  │    - Computes aggregate metrics                          │   │
│  │    - Emits hierarchical updates                          │   │
│  │  • Existing callbacks (status, progress, event, peer)    │   │
│  └──────────────────────────────────────────────────────────┘   │
│                           ▲                                      │
│                           │                                      │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Transfer/Network Layer                                   │   │
│  │  • Per-file transfer workers                             │   │
│  │  • Progress updates per operation                        │   │
│  └──────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                    Keychain Integration                          │
│  (zalando/go-keyring wrapping /usr/bin/security)                │
│  • Store: pairing codes, auth tokens, credentials               │
│  • Retrieve: on app startup, peer connection                    │
└──────────────────────────────────────────────────────────────────┘
```

## Component Boundaries

| Component | Responsibility | Communicates With | Data Flow Direction |
|-----------|---------------|-------------------|---------------------|
| **System Tray Manager** | Menu bar presence, icon, menu items, click handling | App struct (via IPC or direct), Wails runtime (window control) | Bidirectional: receives status updates, sends commands |
| **Keychain Service** | CRUD operations on macOS Keychain via go-keyring | ConfigStore (read credentials), SyncEngine (store/retrieve secrets) | Called by: Config/Engine. Returns: credentials or error |
| **ProgressAggregator** | Collects per-file progress, computes totals, emits updates | Transfer workers (receives), Engine callbacks (emits to UI) | Unidirectional: workers → aggregator → callbacks → UI |
| **UI Progress Components** | Render per-file list, aggregate progress bar, speed/ETA | Svelte stores subscribed to "sync:progress" events | Unidirectional: Engine → EventsEmit → stores → components |
| **Transfer Workers** | Execute file operations, report per-file progress | ProgressAggregator (reports to), Network layer (I/O) | Unidirectional: worker → aggregator |

## Architecture Decisions

### 1. System Tray Integration

**Decision: Use separate-process IPC approach with getlantern/systray for Wails v2, OR migrate to Wails v3 for native support**

**Options evaluated:**

| Option | Pros | Cons | Confidence |
|--------|------|------|------------|
| **fyne.io/systray** | Currently imported | Thread conflicts with Wails, doesn't work | HIGH (verified in main.go comments) |
| **getlantern/systray** | Cross-platform, well-maintained | Requires CGo, thread conflicts with Wails main loop | MEDIUM (community reports conflicts) |
| **energye/systray** | Fork of getlantern, removes GTK | Still requires CGo, thread conflicts | MEDIUM |
| **Separate process IPC** | Avoids thread conflicts, works with any systray library | Additional process to manage, IPC complexity | MEDIUM (proven workaround) |
| **Wails v3 native** | Native support, no conflicts, unified API | Alpha stability, migration effort | HIGH (official docs, but alpha) |

**Recommendation: Two-phase approach**

**Phase 1 (Short term)**: Implement separate-process IPC with getlantern/systray
- Why: Proven workaround, stays on stable Wails v2, avoids blocking progress
- How: Small Go binary using systray.Run(), communicates via Unix domain socket or HTTP localhost
- Trade-off: Two processes to manage, but isolated failure domain

**Phase 2 (Medium term)**: Migrate to Wails v3 when stable
- Why: Native system tray, better platform integration, cleaner architecture
- When: Wails v3 reaches stable (currently alpha as of 2026-01, "nearly ready")
- Effort: Migration guide exists at https://v3alpha.wails.io/migration/v2-to-v3/

**Implementation pattern (Phase 1):**

```go
// systray_manager.go
package systray

import (
    "net"
    "encoding/json"
    "github.com/getlantern/systray"
)

type TrayManager struct {
    listener net.Listener
    commands chan TrayCommand
}

func (t *TrayManager) Start() error {
    // Start IPC listener
    go t.listenForCommands()

    // Start systray in separate binary (spawned as child process)
    go systray.Run(t.onReady, t.onExit)

    return nil
}

func (t *TrayManager) onReady() {
    systray.SetIcon(getIcon())
    systray.SetTooltip("SyncDev")

    mShow := systray.AddMenuItem("Show", "")
    mSync := systray.AddMenuItem("Sync Now", "")
    systray.AddSeparator()
    mQuit := systray.AddMenuItem("Quit", "")

    go t.handleMenuClicks(mShow, mSync, mQuit)
}

func (t *TrayManager) handleMenuClicks(...) {
    // Send commands back to main app via IPC
}

func (t *TrayManager) UpdateStatus(status string) {
    // Called from main app, updates tray tooltip/icon
}
```

**Data flow:**
1. Main app spawns systray binary as child process
2. Systray establishes IPC channel (Unix socket or localhost HTTP)
3. Main app sends status updates → systray updates icon/tooltip
4. User clicks menu item → systray sends command → main app handles

**Sources:**
- [Wails v3 System Tray Documentation](https://v3alpha.wails.io/features/menus/systray/) (MEDIUM confidence - alpha docs)
- [Wails v2 System Tray Issues](https://github.com/wailsapp/wails/discussions/4514) (HIGH confidence - community discussions)
- [getlantern/systray GitHub](https://github.com/getlantern/systray) (HIGH confidence - official repo)

### 2. macOS Keychain Integration

**Decision: Use zalando/go-keyring for credential storage**

**Options evaluated:**

| Option | Pros | Cons | Confidence |
|--------|------|------|------------|
| **keybase/go-keychain** | Direct Security.framework binding, iOS support | Requires CGo, complicates builds | HIGH (official docs) |
| **zalando/go-keyring** | No CGo, cross-platform, simple API | Uses /usr/bin/security binary | HIGH (official docs) |
| **common-fate/go-apple-security** | Go bindings for Security framework | CGo required, macOS-only | MEDIUM (limited docs) |
| **crypto/x509/internal/macos** | Go stdlib (as of Jan 2026) | Internal package (not public API) | MEDIUM (stdlib docs) |

**Recommendation: zalando/go-keyring**

**Why:**
- No CGo = simpler builds, cross-compilation friendly
- Cross-platform API (macOS, Linux, Windows) = future-proof
- Simple API: Set(service, user, password), Get(), Delete()
- Uses /usr/bin/security which is always present on macOS

**Trade-offs:**
- Spawns process for each operation (vs direct C calls)
- Performance: Not an issue for SyncDev's use case (low-frequency credential operations)

**Use cases in SyncDev:**
1. **Pairing codes**: Store device pairing codes (currently in-memory)
2. **Peer secrets**: Store SharedSecret for each paired peer (currently in config.json)
3. **Future auth**: API tokens, cloud sync credentials if added

**Implementation pattern:**

```go
// internal/keychain/service.go
package keychain

import "github.com/zalando/go-keyring"

const ServiceName = "com.syncdev.app"

type KeychainService struct{}

func (k *KeychainService) StorePeerSecret(peerID, secret string) error {
    return keyring.Set(ServiceName, "peer:"+peerID, secret)
}

func (k *KeychainService) GetPeerSecret(peerID string) (string, error) {
    return keyring.Get(ServiceName, "peer:"+peerID)
}

func (k *KeychainService) DeletePeerSecret(peerID string) error {
    return keyring.Delete(ServiceName, "peer:"+peerID)
}

func (k *KeychainService) StorePairingCode(code string) error {
    return keyring.Set(ServiceName, "pairing:current", code)
}

func (k *KeychainService) GetPairingCode() (string, error) {
    return keyring.Get(ServiceName, "pairing:current")
}
```

**Integration points:**
- `config.Config`: Remove `SharedSecret` field, retrieve from Keychain on demand
- `app.go`: Add KeychainService to App struct
- `sync/engine.go`: Use KeychainService for pairing operations
- Migration: On first run with Keychain, migrate existing secrets from config.json → Keychain, then delete from config

**Security considerations:**
- Keychain prompts for access on first use (expected UX)
- Code signing required for Keychain entitlements (already needed for app distribution)
- Secrets never in plaintext JSON after migration

**Sources:**
- [zalando/go-keyring GitHub](https://github.com/zalando/go-keyring) (HIGH confidence - official repo)
- [keybase/go-keychain GitHub](https://github.com/keybase/go-keychain) (HIGH confidence - official repo)

### 3. Progress Tracking Architecture

**Decision: Hierarchical progress aggregation with per-file and aggregate levels**

**Current state:**
- Single `TransferProgress` struct with TotalBytes, TransferredBytes, Speed, ETA
- Updated per transfer operation via `SetProgressCallback()`
- UI displays one progress bar

**Requirements:**
- Per-file progress (file name, size, transferred, status)
- Aggregate progress (total files, completed files, overall %, speed, ETA)
- Efficient updates (avoid UI thrashing with 100+ files syncing)

**Recommended pattern: Event aggregation with throttled emissions**

**Architecture:**

```
Transfer Workers (N parallel)
    ↓ (per-file updates, high frequency)
ProgressAggregator
    • Collects updates in memory
    • Computes aggregate metrics
    • Throttles emissions (max 10 Hz to UI)
    ↓ (aggregated updates, 100ms intervals)
SetProgressCallback()
    ↓ (runtime.EventsEmit)
Svelte stores
    ↓ (reactive subscriptions)
UI components (progress bars, file lists)
```

**Data structures:**

```go
// internal/sync/progress.go

type FileProgress struct {
    FilePath       string    `json:"filePath"`
    Size           int64     `json:"size"`
    Transferred    int64     `json:"transferred"`
    Status         string    `json:"status"` // "pending", "active", "complete", "error"
    Speed          float64   `json:"speed"`  // bytes/sec
    StartTime      time.Time `json:"-"`
    LastUpdateTime time.Time `json:"-"`
}

type AggregateProgress struct {
    TotalFiles      int              `json:"totalFiles"`
    CompletedFiles  int              `json:"completedFiles"`
    TotalBytes      int64            `json:"totalBytes"`
    TransferredBytes int64           `json:"transferredBytes"`
    Percentage      float64          `json:"percentage"`
    OverallSpeed    float64          `json:"overallSpeed"` // bytes/sec
    ETA             int64            `json:"eta"`          // seconds
    Files           []FileProgress   `json:"files"`        // Top N or all
}

type ProgressAggregator struct {
    files          map[string]*FileProgress
    mu             sync.RWMutex
    callback       func(*AggregateProgress)
    ticker         *time.Ticker
    stopChan       chan struct{}
}

func NewProgressAggregator(callback func(*AggregateProgress)) *ProgressAggregator {
    pa := &ProgressAggregator{
        files:    make(map[string]*FileProgress),
        callback: callback,
        ticker:   time.NewTicker(100 * time.Millisecond), // 10 Hz max
        stopChan: make(chan struct{}),
    }
    go pa.emitLoop()
    return pa
}

func (pa *ProgressAggregator) UpdateFile(filePath string, transferred, total int64, status string) {
    pa.mu.Lock()
    defer pa.mu.Unlock()

    fp, exists := pa.files[filePath]
    if !exists {
        fp = &FileProgress{
            FilePath:  filePath,
            Size:      total,
            StartTime: time.Now(),
            Status:    "pending",
        }
        pa.files[filePath] = fp
    }

    fp.Transferred = transferred
    fp.Status = status
    fp.LastUpdateTime = time.Now()

    // Calculate per-file speed
    elapsed := fp.LastUpdateTime.Sub(fp.StartTime).Seconds()
    if elapsed > 0 {
        fp.Speed = float64(fp.Transferred) / elapsed
    }
}

func (pa *ProgressAggregator) emitLoop() {
    for {
        select {
        case <-pa.ticker.C:
            pa.emitAggregate()
        case <-pa.stopChan:
            pa.ticker.Stop()
            return
        }
    }
}

func (pa *ProgressAggregator) emitAggregate() {
    pa.mu.RLock()
    defer pa.mu.RUnlock()

    agg := &AggregateProgress{
        TotalFiles: len(pa.files),
        Files:      make([]FileProgress, 0, len(pa.files)),
    }

    var totalTransferred int64
    var totalSpeed float64

    for _, fp := range pa.files {
        agg.TotalBytes += fp.Size
        totalTransferred += fp.Transferred
        totalSpeed += fp.Speed

        if fp.Status == "complete" {
            agg.CompletedFiles++
        }

        agg.Files = append(agg.Files, *fp)
    }

    agg.TransferredBytes = totalTransferred
    if agg.TotalBytes > 0 {
        agg.Percentage = float64(agg.TransferredBytes) / float64(agg.TotalBytes) * 100
    }
    agg.OverallSpeed = totalSpeed

    // Calculate ETA
    remaining := agg.TotalBytes - agg.TransferredBytes
    if agg.OverallSpeed > 0 {
        agg.ETA = int64(float64(remaining) / agg.OverallSpeed)
    }

    if pa.callback != nil {
        pa.callback(agg)
    }
}

func (pa *ProgressAggregator) Reset() {
    pa.mu.Lock()
    defer pa.mu.Unlock()
    pa.files = make(map[string]*FileProgress)
}

func (pa *ProgressAggregator) Stop() {
    close(pa.stopChan)
}
```

**Integration with existing Engine:**

```go
// In sync/engine.go

type Engine struct {
    // ... existing fields ...
    progressAggregator *ProgressAggregator
}

func NewEngine(cfg *config.Store) (*Engine, error) {
    // ... existing setup ...

    engine := &Engine{
        // ... existing fields ...
    }

    engine.progressAggregator = NewProgressAggregator(func(agg *AggregateProgress) {
        if engine.onProgress != nil {
            engine.onProgress(agg) // Callback to app.go → EventsEmit
        }
    })

    return engine, nil
}

// Transfer operations update per-file:
func (e *Engine) transferFile(filePath string, size int64, ...) error {
    e.progressAggregator.UpdateFile(filePath, 0, size, "active")

    // ... transfer loop ...
    for {
        n, err := // ... read/write ...
        transferred += n
        e.progressAggregator.UpdateFile(filePath, transferred, size, "active")
    }

    e.progressAggregator.UpdateFile(filePath, size, size, "complete")
    return nil
}
```

**UI components (Svelte):**

```svelte
<!-- lib/ProgressDetail.svelte -->
<script>
    import { progressStore } from '../stores/sync.js';

    $: aggregate = $progressStore;
    $: files = aggregate?.files || [];
    $: overallPct = aggregate?.percentage || 0;
</script>

<div class="progress-container">
    <div class="progress-header">
        <h3>Syncing {aggregate?.completedFiles || 0} / {aggregate?.totalFiles || 0} files</h3>
        <span class="progress-speed">{formatSpeed(aggregate?.overallSpeed)}</span>
        <span class="progress-eta">ETA: {formatDuration(aggregate?.eta)}</span>
    </div>

    <div class="progress-bar-container">
        <div class="progress-bar" style="width: {overallPct}%"></div>
    </div>

    <div class="file-list">
        {#each files as file}
            <div class="file-item" class:complete={file.status === 'complete'}>
                <div class="file-name">{file.filePath}</div>
                <div class="file-progress">
                    <progress value={file.transferred} max={file.size}></progress>
                    <span>{formatBytes(file.transferred)} / {formatBytes(file.size)}</span>
                </div>
            </div>
        {/each}
    </div>
</div>
```

**Performance considerations:**
- **Throttling**: 100ms (10 Hz) max emission rate prevents UI thrashing
- **Memory**: O(N) where N = number of files in current sync (bounded by typical sync size)
- **Concurrency**: RWMutex allows parallel workers to update without blocking
- **Event emission**: Single aggregate event per tick reduces Wails runtime overhead

**Scalability:**
- 100 files @ 10 Hz = 1,000 updates/sec → manageable
- 1,000 files @ 10 Hz = 10,000 updates/sec → still manageable (JSON serialization is bottleneck)
- If needed: Sample top N files for UI (e.g., active + recent 20), keep full map in memory

**Sources:**
- [Event-Driven Architecture Guide 2026](https://estuary.dev/blog/event-driven-architecture/) (MEDIUM confidence - general patterns)
- [Notification System Architecture](https://www.systemdesignhandbook.com/guides/design-a-notification-system/) (MEDIUM confidence - worker aggregation patterns)

## Data Flow Diagrams

### System Tray Status Updates

```
Engine state change (StatusIdle → StatusSyncing)
    ↓
engine.onStatusChange(status, action)
    ↓
app.go callback: runtime.EventsEmit("sync:status", ...)
    ↓ (to UI)
Svelte store update
    |
    ↓ (to system tray)
app.TrayManager.UpdateStatus(status) via IPC
    ↓
Systray updates icon/tooltip
```

### Keychain Secret Retrieval

```
Engine needs peer secret for connection
    ↓
engine.keychain.GetPeerSecret(peerID)
    ↓
zalando/go-keyring → /usr/bin/security get-generic-password
    ↓ (macOS Keychain prompt if first access)
Return secret
    ↓
Engine uses secret for HMAC authentication
```

### Progress Update Flow

```
Transfer.Write(chunk) completes
    ↓
progressAggregator.UpdateFile(path, transferred, total, "active")
    ↓ (updates in-memory map)
[100ms ticker fires]
    ↓
progressAggregator.emitAggregate()
    ↓ (computes totals, speeds, ETA)
engine.onProgress(aggregateProgress)
    ↓
app.go callback: runtime.EventsEmit("sync:progress", agg)
    ↓
Svelte progressStore updates
    ↓ (reactive subscription)
UI components re-render
```

## Patterns to Follow

### Pattern 1: Callback Registry with Event Emission

**What:** Central callback functions in Engine emit to Wails runtime
**When:** Any state change that UI needs to reflect
**Example:**

```go
// Engine registers callbacks during startup
engine.SetStatusCallback(func(status sync.SyncStatus, action string) {
    runtime.EventsEmit(ctx, "sync:status", map[string]interface{}{
        "status": status,
        "action": action,
    })
})

// UI subscribes
import { EventsOn } from '../../wailsjs/runtime';
EventsOn("sync:status", (data) => {
    statusStore.set(data);
});
```

**Why:** Decouples Engine from UI, allows multiple consumers (UI + system tray)

### Pattern 2: Throttled Aggregation

**What:** Collect high-frequency updates, emit at lower frequency
**When:** Many workers producing updates faster than UI can consume
**Example:** ProgressAggregator with 100ms ticker (see above)
**Why:** Prevents UI thrashing, reduces Wails runtime overhead, improves perceived performance

### Pattern 3: IPC-Based Service Isolation

**What:** Run conflicting libraries in separate process, communicate via IPC
**When:** Library has threading/event loop requirements incompatible with main app
**Example:** System tray in separate binary with Unix socket or HTTP IPC
**Why:** Isolates failure domain, allows using libraries that would otherwise conflict

### Pattern 4: Secure Credential Storage

**What:** Never store secrets in plaintext config files, use OS-provided keychain
**When:** Any credential, token, or secret that grants access
**Example:** SharedSecret in Keychain instead of config.json
**Why:** OS-level protection, user control over access, audit trail

## Anti-Patterns to Avoid

### Anti-Pattern 1: Polling for Progress

**What:** UI polls backend every N milliseconds for current progress
**Why bad:** Wasted CPU cycles, higher latency, scales poorly with multiple windows
**Instead:** Event-driven push from backend to UI via runtime.EventsEmit

**Incorrect:**
```javascript
setInterval(() => {
    const progress = await GetSyncProgress(); // Round-trip call every 100ms
    progressStore.set(progress);
}, 100);
```

**Correct:**
```javascript
EventsOn("sync:progress", (progress) => {
    progressStore.set(progress); // Push from backend when available
});
```

### Anti-Pattern 2: Per-Update Event Emission

**What:** Emit event for every byte transferred across all files
**Why bad:** Event serialization overhead, UI re-render thrashing, JSON marshaling cost
**Instead:** Aggregate in memory, emit at fixed intervals (100ms)

**Consequences:** With 10 files @ 10 MB/s each = 100,000 events/sec → UI freezes

### Anti-Pattern 3: CGo When Not Necessary

**What:** Using libraries requiring CGo when pure-Go alternatives exist
**Why bad:** Complex build setup, cross-compilation difficulties, larger binaries
**Instead:** Use pure-Go libraries like zalando/go-keyring over keybase/go-keychain

**When CGo is acceptable:** Direct hardware access, no pure-Go alternative, performance critical

### Anti-Pattern 4: Blocking Main Thread with System Tray

**What:** Running systray.Run() on main thread in Wails app
**Why bad:** Thread conflict between Wails runtime and systray event loop
**Instead:** Separate process or wait for Wails v3 native support

**Detection:** App hangs on startup, segfaults, "NSApplication not initialized" errors on macOS

### Anti-Pattern 5: Secrets in Config JSON

**What:** Storing SharedSecret, pairing codes, tokens in plaintext JSON
**Why bad:**
- Visible in backups, config management tools, logs
- No audit trail of access
- User loses fine-grained control
**Instead:** Use OS Keychain with go-keyring

## Scalability Considerations

| Concern | Current Scale | At 1,000 Files/Sync | At 10,000 Files/Sync | Mitigation |
|---------|--------------|---------------------|----------------------|------------|
| **Progress memory** | N/A (single progress) | ~100 KB (FileProgress * 1K) | ~1 MB | Acceptable, but could sample top N active files for UI |
| **Event emission rate** | ~1-10 Hz | ~10 Hz (throttled) | ~10 Hz (throttled) | Throttling prevents scaling issues |
| **UI re-renders** | ~10/sec | ~10/sec | ~10/sec | Svelte reactive updates optimized |
| **IPC overhead (tray)** | ~1 Hz (status updates) | ~1 Hz | ~1 Hz | Status updates infrequent, no scaling issue |
| **Keychain lookups** | ~1/connection | ~10/hr typical | ~100/hr heavy use | Cache in memory after first lookup, invalidate on unpair |

**Bottlenecks to watch:**
1. **JSON serialization** of large progress objects (1K+ files) → Sample files if needed
2. **Svelte re-render** of large file lists → Virtual scrolling for 100+ files
3. **IPC latency** for system tray → Use Unix domain socket (faster than TCP localhost)

## Build Order and Dependencies

**Recommended implementation sequence:**

### Phase 1: Enhanced Progress Tracking (Least Invasive)
**Why first:** Builds on existing callback architecture, no external dependencies, immediate UX improvement
**Effort:** 2-3 days
**Dependencies:** None
**Steps:**
1. Create `ProgressAggregator` in `internal/sync/progress.go`
2. Integrate with existing `Transfer` operations to report per-file
3. Update `SetProgressCallback` to emit `AggregateProgress` instead of `TransferProgress`
4. Update Svelte components to display per-file progress
5. Test with large file sets (100+ files)

**Risk:** Low - isolated change, backwards compatible callback signature

---

### Phase 2: Keychain Integration (Medium Complexity)
**Why second:** Isolated to credential management, no UI changes needed
**Effort:** 2-3 days
**Dependencies:** Phase 1 complete (for testing sync with Keychain credentials)
**Steps:**
1. Add `zalando/go-keyring` dependency (`go get`)
2. Create `internal/keychain/service.go` wrapper
3. Add `KeychainService` to `App` struct
4. Migrate `SharedSecret` from `config.Config` to Keychain
5. Update pairing flow to store secrets in Keychain
6. Add migration logic: on startup, if secrets in config, move to Keychain
7. Test pairing, unpairing, app restart

**Risk:** Medium - sensitive credential handling, requires manual testing of migration

---

### Phase 3A: System Tray (Wails v2 + IPC) OR Phase 3B: Wails v3 Migration
**Why last:** External dependency, architectural complexity, blocks least work

#### Phase 3A: System Tray with IPC (if staying on Wails v2)
**Effort:** 4-5 days
**Dependencies:** Phase 1 + 2 complete (for full feature set in tray)
**Steps:**
1. Create separate `cmd/systray` binary
2. Implement IPC server in main app (Unix socket listener)
3. Implement IPC client in systray binary
4. Add `TrayManager` to coordinate communication
5. Handle menu clicks: Show, Sync Now, Quit
6. Update tray icon/tooltip on status changes
7. Test process lifecycle (spawn, communication, cleanup)

**Risk:** Medium-High - process management, IPC reliability, platform-specific behaviors

#### Phase 3B: Wails v3 Migration (if Wails v3 stable by implementation time)
**Effort:** 3-4 days (migration) + 1-2 days (tray implementation)
**Dependencies:** Phase 1 + 2 complete, Wails v3 stable release
**Steps:**
1. Review migration guide at https://v3alpha.wails.io/migration/v2-to-v3/
2. Update `wails.json` and dependencies
3. Update API calls (EventsEmit, drag-and-drop attributes)
4. Test all existing functionality
5. Implement native system tray using Wails v3 API
6. Test system tray functionality

**Risk:** Medium - migration unknowns, but cleaner long-term architecture

**Decision criteria for 3A vs 3B:**
- If Wails v3 stable release < 2 months away → Wait and do 3B
- If Wails v3 stable release > 2 months away → Implement 3A, plan 3B later
- As of 2026-01-22: Wails v3 in alpha, "nearly ready" per roadmap discussions

---

### Phase 4: UI Polish (Parallel with Phase 3)
**Effort:** 2-3 days
**Dependencies:** Phase 1 complete (for progress components to polish)
**Steps:**
1. Create detailed progress view component
2. Implement virtual scrolling for large file lists (if needed)
3. Add native macOS styling (system fonts, colors, spacing)
4. Responsive layout for window resizing
5. Loading states and error displays

**Risk:** Low - UI-only changes, iterative refinement

## Component Architecture for Svelte (Native macOS Look)

**Current Svelte architecture:**
- Single-page app with tab navigation
- Component files: PeerList, FolderPairs, SyncStatus, Settings
- Stores: app.js (currentTab, modalState, pairingState)
- Inline SVG icons, custom CSS

**Recommendations for native look:**

### 1. CSS Variables for System Theming

```css
/* app.css */
:root {
    /* System colors (macOS adaptive) */
    --color-background: #1e1e1e;
    --color-surface: #2d2d2d;
    --color-surface-hover: #3d3d3d;
    --color-border: rgba(255, 255, 255, 0.1);
    --color-text-primary: rgba(255, 255, 255, 0.9);
    --color-text-secondary: rgba(255, 255, 255, 0.6);
    --color-accent: #007aff; /* macOS blue */

    /* Typography (San Francisco) */
    --font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    --font-size-small: 11px;
    --font-size-body: 13px;
    --font-size-heading: 16px;

    /* Spacing (8px grid) */
    --spacing-xs: 4px;
    --spacing-sm: 8px;
    --spacing-md: 16px;
    --spacing-lg: 24px;

    /* Animation */
    --transition-fast: 150ms cubic-bezier(0.4, 0, 0.2, 1);
}

/* Light mode (respects OS setting) */
@media (prefers-color-scheme: light) {
    :root {
        --color-background: #ffffff;
        --color-surface: #f5f5f5;
        --color-surface-hover: #e8e8e8;
        --color-border: rgba(0, 0, 0, 0.1);
        --color-text-primary: rgba(0, 0, 0, 0.9);
        --color-text-secondary: rgba(0, 0, 0, 0.6);
    }
}
```

### 2. Component Composition Pattern

```svelte
<!-- lib/ProgressPanel.svelte -->
<script>
    import ProgressBar from './ProgressBar.svelte';
    import FileList from './FileList.svelte';
    import { progressStore } from '../stores/sync.js';
</script>

<Panel title="Sync Progress">
    <ProgressBar
        value={$progressStore.transferredBytes}
        max={$progressStore.totalBytes}
    />
    <FileList files={$progressStore.files} />
</Panel>

<!-- lib/ProgressBar.svelte -->
<script>
    export let value = 0;
    export let max = 100;
    $: percentage = max > 0 ? (value / max) * 100 : 0;
</script>

<div class="progress-bar-container">
    <div class="progress-bar-fill" style="width: {percentage}%"></div>
</div>

<style>
    .progress-bar-container {
        height: 6px;
        background: var(--color-surface);
        border-radius: 3px;
        overflow: hidden;
    }
    .progress-bar-fill {
        height: 100%;
        background: var(--color-accent);
        transition: width var(--transition-fast);
    }
</style>
```

### 3. Store Organization

```javascript
// stores/sync.js
import { writable, derived } from 'svelte/store';
import { EventsOn } from '../../wailsjs/runtime';

export const progressStore = writable({
    totalFiles: 0,
    completedFiles: 0,
    totalBytes: 0,
    transferredBytes: 0,
    percentage: 0,
    overallSpeed: 0,
    eta: 0,
    files: []
});

export const syncStatusStore = writable({
    status: 'idle',
    action: ''
});

// Derived store for formatted display
export const progressDisplay = derived(progressStore, $progress => ({
    ...$progress,
    speedFormatted: formatSpeed($progress.overallSpeed),
    etaFormatted: formatDuration($progress.eta),
    bytesFormatted: `${formatBytes($progress.transferredBytes)} / ${formatBytes($progress.totalBytes)}`
}));

// Event subscriptions
EventsOn("sync:progress", (data) => {
    progressStore.set(data);
});

EventsOn("sync:status", (data) => {
    syncStatusStore.set(data);
});
```

**Sources:**
- [Svelte + Wails Best Practices](https://sophieau.com/article/building-with-wails/) (MEDIUM confidence - community article)
- [Svelte Component Patterns](https://render.com/blog/svelte-design-patterns) (MEDIUM confidence - general patterns)
- [SvelteKit with Wails](https://wails.io/docs/guides/sveltekit/) (HIGH confidence - official guide)

## Technical Debt and Future Considerations

### Known Limitations

1. **System Tray IPC Approach (Phase 3A)**
   - Limitation: Two processes increase complexity
   - Mitigation: Document process lifecycle, add health checks
   - Future: Migrate to Wails v3 native tray when stable

2. **Progress Aggregator Memory**
   - Limitation: O(N) memory for N files in sync
   - Mitigation: Acceptable for typical use (< 10K files)
   - Future: Implement sampling if users report 100K+ file syncs

3. **Keychain Performance**
   - Limitation: /usr/bin/security spawns process per operation
   - Mitigation: Cache credentials in memory after first lookup
   - Future: Consider keybase/go-keychain if performance becomes issue

### Future Enhancements

1. **Multi-Window Support** (if Wails v3 migration happens)
   - System tray could show multiple windows (main + progress detail)
   - Event subscriptions work per-window in v3

2. **Background Sync** (requires system tray)
   - Continue syncing with main window closed
   - Show notifications on completion

3. **Cloud Sync Credentials** (Keychain ready)
   - If cloud storage integration added, credentials already in Keychain
   - API tokens, OAuth refresh tokens stored securely

4. **Advanced Progress Filtering**
   - Filter file list by status (active, complete, error)
   - Sort by size, speed, name
   - Search file list

## Verification Checklist

Before implementation:
- [ ] Confirm Wails v3 stable release status (3A vs 3B decision)
- [ ] Test zalando/go-keyring on macOS (proof of concept)
- [ ] Prototype ProgressAggregator with mock transfers (verify throttling)
- [ ] Review existing Transfer code for integration points

After implementation:
- [ ] System tray shows/hides window correctly
- [ ] System tray updates reflect Engine status (idle, syncing)
- [ ] Keychain stores and retrieves secrets without errors
- [ ] macOS Keychain prompts on first access (expected UX)
- [ ] Progress UI updates smoothly during 100-file sync
- [ ] Per-file progress shows all files
- [ ] Aggregate progress shows correct totals, speed, ETA
- [ ] App restart loads secrets from Keychain
- [ ] Unpairing removes secrets from Keychain
- [ ] UI matches macOS native look (system fonts, colors)

## Open Questions

1. **Wails v3 timeline**: Stable release ETA? (Affects 3A vs 3B decision)
   - Status as of 2026-01-22: Alpha, "nearly ready", no fixed date
   - Recommendation: Check monthly, implement 3A if v3 stable > 2 months out

2. **Progress UI performance**: Test with 1,000+ files, does virtual scrolling become necessary?
   - Recommendation: Implement basic list first, add virtual scrolling if users report slowness

3. **Keychain migration**: How to handle users with existing paired peers?
   - Recommendation: Automatic migration on first launch with new version
   - Backup config.json before migration
   - Log migration success/failure

4. **System tray icons**: Sync status animations (spinning icon during sync)?
   - Recommendation: Start with static icons (idle, syncing, error)
   - Add animation in Phase 4 polish if time permits

## Sources

### Official Documentation (HIGH Confidence)
- [Wails v3 System Tray](https://v3alpha.wails.io/features/menus/systray/)
- [Wails v3 Migration Guide](https://v3alpha.wails.io/migration/v2-to-v3/)
- [zalando/go-keyring GitHub](https://github.com/zalando/go-keyring)
- [keybase/go-keychain GitHub](https://github.com/keybase/go-keychain)
- [getlantern/systray GitHub](https://github.com/getlantern/systray)
- [Wails SvelteKit Guide](https://wails.io/docs/guides/sveltekit/)

### Community Resources (MEDIUM Confidence)
- [Wails v2 System Tray Discussions](https://github.com/wailsapp/wails/discussions/4514)
- [Building with Wails and Svelte](https://sophieau.com/article/building-with-wails/)
- [Svelte Component Design Patterns](https://render.com/blog/svelte-design-patterns)
- [Event-Driven Architecture 2026](https://estuary.dev/blog/event-driven-architecture/)
- [Notification System Architecture](https://www.systemdesignhandbook.com/guides/design-a-notification-system/)

### WebSearch Findings (LOW-MEDIUM Confidence)
- Wails v3 alpha status and roadmap discussions
- System tray workaround approaches for Wails v2
- Progress tracking aggregation patterns
- macOS Keychain access methods

---

**Research conducted:** 2026-01-22
**Overall confidence:** MEDIUM-HIGH
- System tray: MEDIUM (Wails v3 in alpha, IPC approach proven but complex)
- Keychain: HIGH (zalando/go-keyring well-documented, proven approach)
- Progress tracking: HIGH (standard event aggregation pattern, clear implementation path)
