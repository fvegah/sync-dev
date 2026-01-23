---
phase: 02-menu-bar-integration
plan: 01
subsystem: framework
tags: [wails, wails-v3, desktop, webview, macos, migration]

# Dependency graph
requires:
  - phase: 01-keychain-security
    provides: Keychain secrets manager and engine integration
provides:
  - Wails v3 application framework
  - Hide-on-close window behavior
  - Event emission via app.Event.Emit
  - Dialog API via app.Dialog
  - Browser API via app.Browser
affects: [02-02-PLAN (systray), 02-03-PLAN (dynamic icons), native-macos-ui]

# Tech tracking
tech-stack:
  added: [github.com/wailsapp/wails/v3 v3.0.0-alpha.62]
  patterns: [app.Window.NewWithOptions, RegisterHook for lifecycle, app.Event.Emit for events]

key-files:
  created: []
  modified: [main.go, app.go, go.mod, go.sum]

key-decisions:
  - "Use application.AssetFileServerFS() for embedded assets instead of direct FS field"
  - "Use application.NewService() for binding app methods to frontend"
  - "Use RegisterHook with events.Common.WindowClosing for hide-on-close"
  - "Use app.Event.Emit(name, data) instead of CustomEvent struct"

patterns-established:
  - "Window lifecycle: app.Window.NewWithOptions() returns *WebviewWindow"
  - "Event emission: app.Event.Emit(eventName, data) for frontend communication"
  - "Dialog access: app.Dialog.OpenFile() for file/folder selection"
  - "Service binding: app.RegisterService(application.NewService(instance))"

# Metrics
duration: 25min
completed: 2026-01-22
---

# Phase 2 Plan 1: Wails v3 Migration Summary

**Migrated from Wails v2.11.0 to v3.0.0-alpha.62 with hide-on-close window behavior for systray support**

## Performance

- **Duration:** 25 min
- **Started:** 2026-01-22T15:30:00Z
- **Completed:** 2026-01-22T15:55:00Z
- **Tasks:** 5 (4 required commits, 1 verification)
- **Files modified:** 4 (main.go, app.go, go.mod, go.sum)

## Accomplishments

- Migrated application from Wails v2 to v3 using new procedural API
- Implemented hide-on-close behavior via RegisterHook and WindowClosing event
- Updated all event emission to use v3 app.Event.Emit pattern
- Removed incompatible fyne.io/systray dependency
- All existing tests pass, build succeeds

## Task Commits

Each task was committed atomically:

1. **Task 1: Update go.mod for Wails v3** - `964d9db` (chore)
2. **Task 2: Rewrite main.go for Wails v3 API** - `2286318` (feat)
3. **Task 3: Update app.go for Wails v3 runtime** - `8b08e18` (refactor)
4. **Task 4: Delete obsolete systray.go** - No commit needed (file was untracked)
5. **Task 5: Verify full application works** - Verification only, no commit

## Files Created/Modified

- `go.mod` - Updated to Wails v3.0.0-alpha.62, removed v2 and fyne.io/systray
- `go.sum` - Updated transitive dependencies
- `main.go` - Rewrote for v3 API: application.New(), Window.NewWithOptions(), RegisterHook
- `app.go` - Updated for v3: removed context, added app/window refs, updated event/dialog/browser APIs

## Decisions Made

1. **AssetFileServerFS vs direct FS** - Wails v3 AssetOptions.Handler expects http.Handler, not embed.FS directly. Used `application.AssetFileServerFS(assets)` wrapper.

2. **Service binding** - v3 uses `application.NewService[T](instance)` instead of interface binding. Called via `app.RegisterService()`.

3. **Event emission pattern** - v3 uses `app.Event.Emit(name string, data ...any)` not struct-based CustomEvent. Simpler API.

4. **Keep ActivationPolicyRegular** - Deferred ActivationPolicyAccessory to Plan 02-02 when systray is added. App runs as normal window until then.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Wails v3 API differences from plan**
- **Found during:** Task 2 and 3 (main.go and app.go rewrites)
- **Issue:** Plan examples used outdated/incorrect v3 API patterns. Actual v3.0.0-alpha.62 API differs:
  - `application.Options.Assets.FS` doesn't exist - need `Handler: AssetFileServerFS()`
  - `app.NewWebviewWindowWithOptions()` doesn't exist - use `app.Window.NewWithOptions()`
  - `app.Bind()` doesn't exist - use `app.RegisterService(application.NewService())`
  - `app.Event.Emit()` takes `(name, data)` not `*CustomEvent`
- **Fix:** Consulted go doc and Wails v3 examples to determine correct API signatures
- **Files modified:** main.go, app.go
- **Verification:** Build succeeds, all tests pass
- **Committed in:** 2286318, 8b08e18

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** API changes required consulting documentation. Core functionality unchanged.

## Issues Encountered

- **Linker warnings about macOS version** - Object files built for macOS 26.0 being linked against 11.0. These are warnings only, not errors. Build succeeds. May need to set deployment target in future.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Wails v3 foundation complete
- Window hides on close (key for systray behavior)
- Ready for Plan 02-02: Add system tray with native Wails v3 API
- All frontend bindings preserved and working

---
*Phase: 02-menu-bar-integration*
*Plan: 01*
*Completed: 2026-01-22*
