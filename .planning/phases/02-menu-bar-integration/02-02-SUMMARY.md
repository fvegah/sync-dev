---
phase: 02-menu-bar-integration
plan: 02
subsystem: ui
tags: [systray, wails-v3, macos, menu-bar, template-icons]

# Dependency graph
requires:
  - phase: 02-menu-bar-integration plan 01
    provides: Wails v3 migration with application.App and WebviewWindow
provides:
  - System tray icon in macOS menu bar
  - Context menu with Sync Now, Pause/Resume, Open, Quit
  - Click-to-toggle window visibility
  - No Dock icon (ActivationPolicyAccessory)
  - Dynamic icon states (idle, syncing, error)
affects: [02-menu-bar-integration plan 03, progress-display, sync-engine-tray-integration]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Wails v3 SystemTray.New() for native tray"
    - "//go:embed for tray icons"
    - "Template icons (22x22 black+transparent)"

key-files:
  created:
    - internal/tray/tray.go
    - internal/tray/icons.go
    - internal/tray/icons/tray-idle.png
    - internal/tray/icons/tray-syncing.png
    - internal/tray/icons/tray-error.png
    - internal/tray/icons/generate.go
  modified:
    - main.go
    - app.go

key-decisions:
  - "Used app.SystemTray.New() API (Wails v3 manager pattern)"
  - "Created custom Go program to generate icons (ImageMagick not available)"
  - "Template icons for automatic dark/light mode adaptation"

patterns-established:
  - "SyncActions interface for tray-to-app communication"
  - "Manager pattern for tray lifecycle"
  - "Window starts hidden for menu bar app"

# Metrics
duration: 4min 20s
completed: 2026-01-22
---

# Phase 2 Plan 02: System Tray Implementation Summary

**Native macOS system tray with context menu using Wails v3 API - app runs as menu bar accessory without Dock icon**

## Performance

- **Duration:** 4 min 20s
- **Started:** 2026-01-23T02:08:17Z
- **Completed:** 2026-01-23T02:12:37Z
- **Tasks:** 6
- **Files modified:** 8

## Accomplishments

- System tray icon appears in macOS menu bar
- Context menu with Sync Now, Pause/Resume, Open SyncDev, Quit
- App runs without Dock icon (ActivationPolicyAccessory)
- Click tray icon toggles window visibility
- Icon state management for idle/syncing/error states

## Task Commits

Each task was committed atomically:

1. **Task 1: Create tray icon assets** - `0abefc7` (feat)
2. **Task 2: Create internal/tray/icons.go** - `14e6fb8` (feat)
3. **Task 3: Create internal/tray/tray.go** - `465fa70` (feat)
4. **Task 4: Update app.go for tray integration** - `0a1505b` (feat)
5. **Task 5: Update main.go to use tray manager** - `40877ac` (feat)
6. **Task 6: Verify systray functionality** - no commit (verification only)

## Files Created/Modified

- `internal/tray/icons/generate.go` - Go program to generate 22x22 template icons
- `internal/tray/icons/tray-idle.png` - Idle state icon (circular arrows)
- `internal/tray/icons/tray-syncing.png` - Syncing state icon (opposing arrows)
- `internal/tray/icons/tray-error.png` - Error state icon (warning triangle)
- `internal/tray/icons.go` - Embedded icon assets with //go:embed
- `internal/tray/tray.go` - Manager struct, NewManager, SetState, context menu
- `app.go` - Added tray field, paused field, IsPaused/Pause/Resume methods
- `main.go` - ActivationPolicyAccessory, tray.NewManager(), Hidden window

## Decisions Made

- **Wails v3 API:** Used `app.SystemTray.New()` (manager pattern) instead of deprecated `app.NewSystemTray()`
- **Icon generation:** Created custom Go program `generate.go` since ImageMagick was not available
- **Template icons:** 22x22 black+transparent PNGs for automatic macOS light/dark mode adaptation
- **SyncActions interface:** Clean abstraction for tray menu callbacks to app methods

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed Wails v3 SystemTray API**
- **Found during:** Task 3 (Create internal/tray/tray.go)
- **Issue:** Plan used `app.NewSystemTray()` but Wails v3 API is `app.SystemTray.New()`
- **Fix:** Updated to use correct v3 manager pattern
- **Files modified:** internal/tray/tray.go
- **Verification:** Package compiles successfully
- **Committed in:** 465fa70 (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** API correction necessary for compilation. No scope creep.

## Issues Encountered

- ImageMagick (`convert`) not available on system - created Go-based icon generator instead

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- System tray fully functional with static icons
- Ready for Plan 02-03: Dynamic state icons and sync engine integration
- Tray Manager exposes SetState() for icon updates based on sync status

---
*Phase: 02-menu-bar-integration*
*Completed: 2026-01-22*
