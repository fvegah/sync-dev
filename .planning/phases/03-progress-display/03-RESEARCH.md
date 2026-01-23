# Phase 3: Progress Display - Research

**Researched:** 2026-01-22
**Domain:** Real-time progress display for file synchronization (Go backend + Svelte frontend)
**Confidence:** HIGH

## Summary

This phase adds comprehensive progress visibility during file synchronization, including global progress bars, per-file progress, transfer speed (MB/s), and estimated time remaining (ETA). The research examines the current codebase's progress infrastructure, throttling strategies to avoid UI freeze, and ETA calculation algorithms.

The current codebase already has foundational progress infrastructure in place:
- `models.TransferProgress` struct with `FileName`, `TotalBytes`, `TransferBytes`, `Percentage`, `BytesPerSecond`
- `engine.SetProgressCallback()` wired to `app.go` which emits `sync:progress` events via Wails
- `SyncStatus.svelte` already displays a single-file progress bar with percentage

The main gaps are:
1. No global/aggregate progress across all files in a sync operation
2. No ETA calculation (only current speed)
3. No throttling - callbacks fire on every chunk (1MB base64 encoded)
4. No per-file list showing multiple active/queued transfers
5. Progress only tracks single file at a time (not batch operations)

**Primary recommendation:** Implement a `ProgressAggregator` in Go that collects per-file progress, throttles emissions to 10-20 Hz, computes aggregate statistics (total %, overall speed, ETA), and emits a single structured event to the frontend.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Wails v3 | 3.0.0-alpha.62 | Event emission to frontend | Already in use; `app.Event.Emit()` for real-time updates |
| Svelte | 3.49.0 | Reactive UI components | Already in use; stores + reactivity perfect for progress |
| Go `time.Ticker` | stdlib | Throttled emissions | Simple, reliable, no external dependency |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/bep/debounce` | 1.2.1 | Debouncing (already indirect dep) | Alternative to manual throttling |
| Svelte writable stores | built-in | Frontend state management | Already in `stores/app.js` |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Manual time.Ticker | `bep/debounce` | Less control over final emission timing |
| Svelte stores | RxJS/observables | Overkill for this use case, adds complexity |
| Custom ETA algo | No library needed | ETA is simple enough to implement inline |

**No additional npm packages needed.** The existing Svelte setup is sufficient.

## Architecture Patterns

### Recommended Project Structure
```
internal/sync/
  engine.go          # Existing - add ProgressAggregator integration
  transfer.go        # Existing - already emits per-chunk progress
  progress.go        # NEW - ProgressAggregator implementation

frontend/src/
  stores/
    app.js           # Existing - add progressStore shape updates
    progress.js      # NEW (optional) - dedicated progress store
  lib/
    SyncStatus.svelte    # Modify - add aggregate progress display
    ProgressBar.svelte   # NEW - reusable progress bar component
    FileProgressList.svelte  # NEW - scrollable list of active files
```

### Pattern 1: Throttled Progress Aggregator (Backend)
**What:** Go struct that collects per-file progress, emits aggregated data at fixed intervals
**When to use:** Any high-frequency data that needs UI display (progress, metrics, logs)
**Example:**
```go
// Source: Codebase pattern from .planning/research/ARCHITECTURE.md
type AggregateProgress struct {
    TotalFiles       int                    `json:"totalFiles"`
    CompletedFiles   int                    `json:"completedFiles"`
    TotalBytes       int64                  `json:"totalBytes"`
    TransferredBytes int64                  `json:"transferredBytes"`
    Percentage       float64                `json:"percentage"`
    OverallSpeed     float64                `json:"overallSpeed"`     // bytes/sec
    ETA              int64                  `json:"eta"`              // seconds remaining
    Files            []FileProgress         `json:"files"`            // top N active
}

type FileProgress struct {
    FilePath    string  `json:"filePath"`
    Size        int64   `json:"size"`
    Transferred int64   `json:"transferred"`
    Speed       float64 `json:"speed"`
    Status      string  `json:"status"` // "active", "pending", "complete"
}

type ProgressAggregator struct {
    files       map[string]*FileProgress
    mu          sync.RWMutex
    callback    func(*AggregateProgress)
    ticker      *time.Ticker
    stopChan    chan struct{}
    lastEmit    time.Time
    minInterval time.Duration // 50-100ms for 10-20 Hz

    // ETA smoothing
    speedHistory []float64
    smoothingFactor float64 // 0.1-0.15 recommended
}
```

### Pattern 2: Exponential Smoothing for ETA
**What:** Smooth speed measurements to prevent ETA from jumping wildly
**When to use:** Any estimated time calculation based on variable-rate progress
**Example:**
```go
// Source: https://reversed.top/2022-12-31/benchmarking-io-eta-algorithms/
// Exponential Moving Average with alpha = 0.1 (slow learning, stable)
func (pa *ProgressAggregator) smoothedSpeed(instantSpeed float64) float64 {
    const alpha = 0.1
    if pa.lastSmoothedSpeed == 0 {
        pa.lastSmoothedSpeed = instantSpeed
        return instantSpeed
    }
    pa.lastSmoothedSpeed = alpha*instantSpeed + (1-alpha)*pa.lastSmoothedSpeed
    return pa.lastSmoothedSpeed
}

func (pa *ProgressAggregator) calculateETA(remaining int64, smoothedSpeed float64) int64 {
    if smoothedSpeed <= 0 {
        return -1 // unknown
    }
    return int64(float64(remaining) / smoothedSpeed)
}
```

### Pattern 3: Wails Event Emission (Already Established)
**What:** Backend emits structured events, frontend subscribes
**When to use:** All backend-to-frontend real-time updates
**Example:**
```go
// Source: Current app.go pattern
engine.SetProgressCallback(func(progress *AggregateProgress) {
    a.app.Event.Emit("sync:progress", progress)
})

// Frontend subscription (already in SyncStatus.svelte)
EventsOn('sync:progress', (data) => {
    transferProgress.set(data);
});
```

### Pattern 4: Reactive Progress Store (Frontend)
**What:** Svelte store that receives progress events and provides computed values
**When to use:** Complex derived state from event data
**Example:**
```javascript
// Source: Svelte reactivity pattern
// In stores/app.js or dedicated progress.js
import { writable, derived } from 'svelte/store';

export const progressData = writable(null);

// Derived computations
export const progressPercentage = derived(progressData, $p => $p?.percentage ?? 0);
export const activeFiles = derived(progressData, $p =>
    ($p?.files ?? []).filter(f => f.status === 'active').slice(0, 10)
);
export const formattedETA = derived(progressData, $p => {
    if (!$p?.eta || $p.eta < 0) return '--:--';
    const mins = Math.floor($p.eta / 60);
    const secs = $p.eta % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
});
```

### Anti-Patterns to Avoid
- **Synchronous callback on every chunk:** Causes UI freeze (documented in PITFALLS.md Pitfall #2)
- **Polling from frontend:** Wails event-driven model is push-based; don't poll
- **Unbounded file list:** Limit to top N (10-20) active files; too many causes DOM thrashing
- **Raw speed display:** Always smooth speed; instant measurements are noisy

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Time-based throttling | Custom timer goroutine | `time.Ticker` + select | Proper cleanup, no goroutine leaks |
| Speed smoothing | Rolling average array | Exponential moving average | O(1) memory, better responsiveness |
| Progress bar styling | Custom CSS animations | Native CSS `transition: width 0.3s` | Browser-optimized, GPU accelerated |
| Bytes formatting | Custom division logic | Keep existing `FormatBytes()` in app.go | Already implemented correctly |
| Debouncing | Manual timers | `github.com/bep/debounce` (already dep) | Battle-tested, correct edge cases |

**Key insight:** The complexity is in the aggregation logic and throttling, not in individual operations. Focus on the ProgressAggregator architecture rather than micro-optimizations.

## Common Pitfalls

### Pitfall 1: Progress Callbacks Blocking UI Thread
**What goes wrong:** Calling progress callbacks synchronously on every file chunk (1MB in this codebase) overwhelms the UI, causing stuttering, freezing, or dropped updates.
**Why it happens:**
- Current `transfer.go` calls `progressCb()` after every 1MB chunk
- 100MB file = 100 callbacks in rapid succession
- Each callback triggers Wails IPC, Svelte reactivity, DOM updates
**How to avoid:**
- Throttle to 50-100ms intervals (10-20 Hz)
- Batch multiple file updates into single aggregate emission
- Use `time.Ticker` in `ProgressAggregator`, not per-callback
**Warning signs:**
- UI freezes during large file transfers
- Progress bar stutters instead of smooth animation
- CPU spikes on UI thread during transfers

### Pitfall 2: ETA Jumping Wildly
**What goes wrong:** ETA shows "2 minutes" then "30 seconds" then "5 minutes" in rapid succession
**Why it happens:**
- Using instant speed (bytes this chunk / time for this chunk)
- Network speed varies, especially over WiFi
- Small files complete quickly, causing speed spikes
**How to avoid:**
- Exponential smoothing with alpha 0.1 (slow learning)
- Don't show ETA until at least 5% complete (insufficient data)
- Show "calculating..." for first few seconds
**Warning signs:** Users report "ETA is useless, keeps changing"

### Pitfall 3: Memory Leak in Progress Map
**What goes wrong:** `ProgressAggregator.files` map grows unbounded over long sessions
**Why it happens:** Files are added but never removed after completion
**How to avoid:**
- Remove completed files after emitting final 100% status
- Or use TTL: remove files with `status=complete` after 5 seconds
- Clear entire map at start of each sync operation
**Warning signs:** Memory usage grows during multi-hour sync sessions

### Pitfall 4: Base64 Overhead Affects Speed Calculation
**What goes wrong:** Reported speed is ~33% higher than actual network throughput
**Why it happens:**
- Current `transfer.go` uses base64 encoding (lines 66-68)
- Base64 inflates data by ~33%
- Speed is calculated on decoded bytes, but network transfer is encoded
**How to avoid:**
- Calculate speed on actual network bytes transferred (pre-decode size)
- Or note that displayed speed is "effective throughput" not raw
- Long-term: migrate away from base64 (Phase 5+ optimization)
**Warning signs:** Users report "speed says 10 MB/s but I only have 7 MB/s connection"

### Pitfall 5: Frontend Store Not Resetting Between Syncs
**What goes wrong:** Old progress data persists, showing stale file list
**Why it happens:** Store only receives updates, never explicit reset
**How to avoid:**
- Backend emits "sync:start" event with fresh aggregate (empty files list)
- Frontend clears store on sync start
- Add `status` field to aggregate: "starting", "syncing", "complete"
**Warning signs:** File list shows completed files from previous sync

## Code Examples

Verified patterns from codebase and research:

### Current TransferProgress Model (Extend This)
```go
// Source: internal/models/file.go lines 42-49
// Current model - single file only
type TransferProgress struct {
    FileName       string  `json:"fileName"`
    TotalBytes     int64   `json:"totalBytes"`
    TransferBytes  int64   `json:"transferBytes"`
    Percentage     float64 `json:"percentage"`
    BytesPerSecond int64   `json:"bytesPerSecond"`
}

// NEW: Aggregate model to add
type AggregateProgress struct {
    Status           string         `json:"status"` // "idle", "syncing", "complete"
    TotalFiles       int            `json:"totalFiles"`
    CompletedFiles   int            `json:"completedFiles"`
    TotalBytes       int64          `json:"totalBytes"`
    TransferredBytes int64          `json:"transferredBytes"`
    Percentage       float64        `json:"percentage"`
    BytesPerSecond   int64          `json:"bytesPerSecond"`
    ETA              int64          `json:"eta"` // seconds, -1 if unknown
    ActiveFiles      []FileProgress `json:"activeFiles"` // max 10
}

type FileProgress struct {
    Path        string  `json:"path"`
    Size        int64   `json:"size"`
    Transferred int64   `json:"transferred"`
    Percentage  float64 `json:"percentage"`
    Status      string  `json:"status"` // "active", "pending", "complete"
}
```

### Current Event Emission Pattern (Preserve This)
```go
// Source: app.go lines 78-80
engine.SetProgressCallback(func(progress *models.TransferProgress) {
    a.app.Event.Emit("sync:progress", progress)
})

// Change to accept AggregateProgress
engine.SetProgressCallback(func(progress *models.AggregateProgress) {
    a.app.Event.Emit("sync:progress", progress)
})
```

### Current Frontend Subscription (Extend This)
```svelte
<!-- Source: frontend/src/lib/SyncStatus.svelte lines 27-35 -->
<script>
    EventsOn('sync:progress', (data) => {
        transferProgress.set(data);
    });
</script>

<!-- Current single-file display (line 254-266) -->
{#if $transferProgress && ($syncStatus.status === 'syncing' || progressPercentage > 0)}
    <div class="progress-section">
        <div class="progress-info">
            <span class="file-name">{$transferProgress.fileName}</span>
            <span class="progress-stats">{Math.round(progressPercentage)}%</span>
        </div>
        <div class="progress-bar">
            <div class="progress-fill" style="width: {progressPercentage}%"></div>
        </div>
    </div>
{/if}
```

### Throttled Emission Pattern
```go
// Source: Pattern from .planning/research/PITFALLS.md lines 66-79
type ThrottledEmitter struct {
    lastEmit    time.Time
    minInterval time.Duration
    callback    func(interface{})
}

func (te *ThrottledEmitter) MaybeEmit(data interface{}, forceIfComplete bool) {
    now := time.Now()
    if forceIfComplete || now.Sub(te.lastEmit) >= te.minInterval {
        te.callback(data)
        te.lastEmit = now
    }
}
```

### Exponential Smoothing for Speed
```go
// Source: https://reversed.top/2022-12-31/benchmarking-io-eta-algorithms/
// Alpha = 0.1 provides good stability for file transfers
func exponentialSmooth(current, previous float64, alpha float64) float64 {
    if previous == 0 {
        return current
    }
    return alpha*current + (1-alpha)*previous
}
```

### Progress Bar CSS (Smooth Transitions)
```css
/* Source: Current SyncStatus.svelte lines 597-599 */
.progress-fill {
    height: 100%;
    background: linear-gradient(90deg, #3b82f6, #60a5fa);
    border-radius: 3px;
    transition: width 0.3s ease; /* Smooth animation */
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Per-chunk callbacks | Throttled aggregation | Best practice | 10x fewer events, smooth UI |
| Linear ETA | Exponential smoothing | Always recommended | Stable, accurate estimates |
| Polling progress | Event-driven push | Wails architecture | Lower overhead, real-time |

**Deprecated/outdated:**
- Polling `GetSyncProgress()` from frontend: Use event subscription instead
- Calculating speed from single interval: Use smoothed average

## Integration Points

### Existing Callback Wiring
The engine already has these callbacks in place (from `engine.go`):
```go
type Engine struct {
    // ...
    onProgress     func(*models.TransferProgress)  // Change to AggregateProgress
    // ...
}

func (e *Engine) SetProgressCallback(cb func(*models.TransferProgress)) {
    e.onProgress = cb
}
```

### Existing Progress Emission Points
In `engine.go`, progress callbacks are invoked at:
- Line 790-796: `pushFile()` - sends file to peer
- Line 898-904: `handleFileChunk()` - receives file chunk

These need to feed into `ProgressAggregator` instead of directly to callback.

### Frontend Store Structure
Current store (from `stores/app.js`):
```javascript
export const transferProgress = writable(null);
```
This will receive `AggregateProgress` instead of `TransferProgress`.

## Open Questions

Things that couldn't be fully resolved:

1. **Multiple concurrent transfers**
   - What we know: Current engine processes files sequentially in `handleIndexExchange()`
   - What's unclear: Will future phases add parallel transfers?
   - Recommendation: Design `ProgressAggregator` to handle multiple active files, even if currently only one is active

2. **File list length limit**
   - What we know: Need to limit for UI performance
   - What's unclear: Optimal number (10? 20? 50?)
   - Recommendation: Start with 10 active files, measure performance, adjust

3. **Progress persistence across app restart**
   - What we know: Current implementation doesn't persist
   - What's unclear: Is this needed for Phase 3?
   - Recommendation: Defer to Phase 5+; focus on in-memory progress for now

4. **Global vs per-folder-pair progress**
   - What we know: User might have multiple folder pairs
   - What's unclear: Should we show aggregate across all, or per-pair?
   - Recommendation: Start with global aggregate; add per-pair view if requested

## Sources

### Primary (HIGH confidence)
- Current codebase analysis: `internal/sync/engine.go`, `internal/sync/transfer.go`, `internal/models/file.go`
- Current codebase analysis: `frontend/src/lib/SyncStatus.svelte`, `frontend/src/stores/app.js`
- Current codebase analysis: `app.go` event emission pattern
- `.planning/research/ARCHITECTURE.md` - Pre-existing progress aggregator design
- `.planning/research/PITFALLS.md` - Documented callback flooding pitfall

### Secondary (MEDIUM confidence)
- [Wails v3 Events System](https://v3alpha.wails.io/reference/events/) - Event emission API
- [Benchmarking I/O ETA algorithms](https://reversed.top/2022-12-31/benchmarking-io-eta-algorithms/) - Exponential smoothing recommendations (alpha = 0.1)
- [Go by Example: Tickers](https://gobyexample.com/tickers) - Throttling pattern

### Tertiary (LOW confidence)
- [Svelte Progress Bar Components](https://flowbite-svelte.com/docs/components/progress) - Community patterns (not needed, native CSS sufficient)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - No new dependencies needed, all patterns verified in codebase
- Architecture: HIGH - ProgressAggregator pattern already designed in ARCHITECTURE.md
- Pitfalls: HIGH - Callback flooding documented with code examples
- ETA algorithm: MEDIUM - External source (benchmarking blog), well-reasoned but not official

**Research date:** 2026-01-22
**Valid until:** 60 days (stable patterns, no external library version concerns)
