# Phase 2: Menu Bar Integration - Research

**Researched:** 2026-01-22
**Domain:** macOS System Tray / Menu Bar Integration with Wails
**Confidence:** HIGH

## Summary

This research covers implementing a professional macOS menu bar application using Wails. The app needs to minimize to system tray on window close, provide a contextual menu (Sync Now, Pause, Open, Quit), and display dynamic icons based on sync state (idle, syncing, error).

Key findings:
- **Wails v3 is the recommended path.** v3.0.0-alpha.62 includes native systray support with template icons for macOS, is stable enough for production (API finalized, active daily releases), and eliminates the main thread conflict that blocks `fyne.io/systray` in v2.
- **Migration from v2 to v3 is manageable** (1-4 hours typical). The major changes are: procedural API vs declarative, separate application/window creation, events via hooks instead of callbacks, and `ShouldClose` replaced with `RegisterHook`.
- **Remotray is NOT recommended** due to POC status (last update July 2022), minimal documentation, and the availability of a better solution (Wails v3).
- **Dynamic icon states** are well-supported in v3 via `SetIcon()`, `SetTemplateIcon()`, and `SetDarkModeIcon()` methods that can be called at any time to change the tray icon.

**Primary recommendation:** Migrate to Wails v3.0.0-alpha.62+ and use the native systray API with `ActivationPolicyAccessory` for proper menu bar app behavior on macOS.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Wails v3 | v3.0.0-alpha.62 | Desktop framework with native systray | Official, native systray support, active development (daily releases) |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| wails/v3/pkg/application | (bundled) | Application, Window, SystemTray APIs | Always - core framework |
| wails/v3/pkg/events | (bundled) | Event hooks for window lifecycle | For hide-on-close behavior |
| wails/v3/pkg/icons | (bundled) | Default systray icons | Can use or replace with custom |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Wails v3 native | Remotray + Wails v2 | POC status, no docs, unmaintained since July 2022 |
| Wails v3 native | fyne.io/systray direct | Main thread conflict with Wails, causes hangs |
| Template icons | Light/dark mode icons | Template icons auto-adapt to macOS theme, simpler |

**Installation:**
```bash
# Install Wails v3 CLI
go install github.com/wailsapp/wails/v3/cmd/wails3@latest

# Update go.mod
go get github.com/wailsapp/wails/v3@v3.0.0-alpha.62
```

## Architecture Patterns

### Recommended Project Structure
```
├── main.go                    # Application setup, systray creation
├── app.go                     # App struct, business logic (mostly unchanged)
├── internal/
│   └── tray/
│       ├── tray.go            # Systray manager with state icons
│       └── icons.go           # Embedded icon assets
└── build/
    └── icons/
        ├── tray-idle.png      # 22x22 template icon (black + alpha)
        ├── tray-syncing.png   # 22x22 template icon
        └── tray-error.png     # 22x22 template icon
```

### Pattern 1: Accessory Application with Systray
**What:** macOS app that lives in menu bar without Dock icon
**When to use:** Always for menu bar utility apps
**Example:**
```go
// Source: https://pkg.go.dev/github.com/wailsapp/wails/v3/examples/systray-basic
package main

import (
    "runtime"
    "github.com/wailsapp/wails/v3/pkg/application"
    "github.com/wailsapp/wails/v3/pkg/events"
)

func main() {
    app := application.New(application.Options{
        Name:        "SyncDev",
        Description: "Folder Sync for Mac",
        Mac: application.MacOptions{
            // Key setting: app runs as menu bar accessory, no Dock icon
            ActivationPolicy: application.ActivationPolicyAccessory,
        },
    })

    // Create system tray
    systemTray := app.SystemTray.New()

    // Create hidden window attached to tray
    window := app.Window.NewWithOptions(application.WebviewWindowOptions{
        Width:         1024,
        Height:        700,
        Name:          "SyncDev",
        Hidden:        true,  // Start hidden
        AlwaysOnTop:   true,  // When shown via tray
    })

    // Hide instead of close
    window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
        window.Hide()
        e.Cancel()
    })

    // Template icon for macOS (auto light/dark)
    if runtime.GOOS == "darwin" {
        systemTray.SetTemplateIcon(trayIdleIcon)
    }

    // Attach window - click tray to toggle
    systemTray.AttachWindow(window).WindowOffset(5)

    app.Run()
}
```

### Pattern 2: Context Menu with Actions
**What:** Right-click menu on tray icon with app actions
**When to use:** Always - required for TRAY-02
**Example:**
```go
// Source: https://pkg.go.dev/github.com/wailsapp/wails/v3/examples/systray-menu
func setupTrayMenu(app *application.App, systray *application.SystemTray, appInstance *App) {
    menu := app.NewMenu()

    menu.Add("Sync Now").OnClick(func(ctx *application.Context) {
        appInstance.SyncNow()
    })

    menu.Add("Pause Sync").OnClick(func(ctx *application.Context) {
        item := ctx.ClickedMenuItem()
        if appInstance.IsPaused() {
            appInstance.Resume()
            item.SetLabel("Pause Sync")
        } else {
            appInstance.Pause()
            item.SetLabel("Resume Sync")
        }
    })

    menu.AddSeparator()

    menu.Add("Open SyncDev").OnClick(func(ctx *application.Context) {
        window.Show()
        window.Focus()
    })

    menu.AddSeparator()

    menu.Add("Quit").OnClick(func(ctx *application.Context) {
        app.Quit()
    })

    systray.SetMenu(menu)
}
```

### Pattern 3: Dynamic Icon States
**What:** Change tray icon based on sync status
**When to use:** Required for TRAY-03
**Example:**
```go
// Source: Wails v3 systray API
package tray

import (
    _ "embed"
    "github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed icons/tray-idle.png
var IconIdle []byte

//go:embed icons/tray-syncing.png
var IconSyncing []byte

//go:embed icons/tray-error.png
var IconError []byte

type Manager struct {
    systray *application.SystemTray
}

type State int

const (
    StateIdle State = iota
    StateSyncing
    StateError
)

func (m *Manager) SetState(state State) {
    var icon []byte
    switch state {
    case StateIdle:
        icon = IconIdle
    case StateSyncing:
        icon = IconSyncing
    case StateError:
        icon = IconError
    }
    m.systray.SetTemplateIcon(icon)
}
```

### Pattern 4: State Communication via Events
**What:** Sync engine notifies tray of state changes
**When to use:** Decouple sync logic from UI
**Example:**
```go
// Source: Application architecture pattern
// In sync/engine.go - emit events on state change
func (e *Engine) emitStateChange(state SyncStatus) {
    e.app.Events.Emit(&application.WailsEvent{
        Name: "sync:state-changed",
        Data: map[string]interface{}{
            "state": state,
        },
    })
}

// In tray/tray.go - listen for state changes
func (m *Manager) Start(app *application.App) {
    app.Events.On("sync:state-changed", func(event *application.WailsEvent) {
        data := event.Data.(map[string]interface{})
        state := data["state"].(string)
        switch state {
        case "idle":
            m.SetState(StateIdle)
        case "syncing":
            m.SetState(StateSyncing)
        case "error":
            m.SetState(StateError)
        }
    })
}
```

### Anti-Patterns to Avoid
- **Keeping v2 and adding Remotray:** Technical debt, POC library, no maintenance
- **Using `fyne.io/systray` directly with Wails:** Main thread conflict causes app hangs
- **Full-color tray icons on macOS:** Won't adapt to light/dark mode, looks unprofessional
- **Polling for state in tray:** Use events/callbacks, not timers
- **Blocking in menu click handlers:** Run sync operations in goroutines

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| System tray | Custom objc bindings | Wails v3 SystemTray API | Cross-platform, tested, maintained |
| Hide on close | Custom window delegate | `RegisterHook(events.Common.WindowClosing)` | v3 provides the hook |
| Template icons | Manual light/dark handling | `SetTemplateIcon()` | macOS handles color adaptation |
| Menu state | Manual enable/disable logic | Menu item callbacks with `SetLabel()`, `SetEnabled()` | Built into API |

**Key insight:** Wails v3 has solved all the systray problems that made v2 require workarounds. Use the built-in APIs.

## Common Pitfalls

### Pitfall 1: Using HideWindowOnClose (Removed in v3)
**What goes wrong:** Code compiles but flag is ignored, window closes
**Why it happens:** v2 had `HideWindowOnClose`, v3 removed it
**How to avoid:** Use `RegisterHook(events.Common.WindowClosing)` with `e.Cancel()`
**Warning signs:** Window closes instead of hiding when clicking X

### Pitfall 2: Forgetting ActivationPolicyAccessory
**What goes wrong:** App shows in Dock even though it's a tray app
**Why it happens:** Default policy is `ActivationPolicyRegular`
**How to avoid:** Set `Mac.ActivationPolicy: application.ActivationPolicyAccessory`
**Warning signs:** Dock icon appears, app doesn't feel like menu bar utility

### Pitfall 3: Using Full-Color Icons as Template
**What goes wrong:** Icon invisible or wrong color in light/dark mode
**Why it happens:** Template icons use alpha channel only, color is ignored
**How to avoid:** Create black + transparent icons, let macOS handle color
**Warning signs:** Icon looks fine in dark mode, invisible in light mode

### Pitfall 4: Blocking Menu Click Handlers
**What goes wrong:** Menu becomes unresponsive during long operations
**Why it happens:** Sync operations block the main thread
**How to avoid:** Wrap operations in goroutines: `go app.SyncNow()`
**Warning signs:** Clicking "Sync Now" freezes the tray menu briefly

### Pitfall 5: Icon Size Not 22x22
**What goes wrong:** Icon looks blurry or disproportionate
**Why it happens:** macOS menu bar expects 22pt height icons
**How to avoid:** Create 22x22 @1x and 44x44 @2x PNG icons
**Warning signs:** Icon looks smaller/larger than other menu bar icons

### Pitfall 6: Not Handling App Quit Properly
**What goes wrong:** Background resources not cleaned up
**Why it happens:** Tray apps don't have obvious quit path
**How to avoid:** Add "Quit" menu item that calls `app.Quit()`, ensure cleanup in shutdown
**Warning signs:** Orphan processes after quitting

## Code Examples

Verified patterns from official sources:

### Complete Wails v3 Systray Setup
```go
// Source: Wails v3 examples + documentation
package main

import (
    _ "embed"
    "log"
    "runtime"

    "github.com/wailsapp/wails/v3/pkg/application"
    "github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/icons/tray-idle.png
var trayIconIdle []byte

//go:embed build/icons/tray-syncing.png
var trayIconSyncing []byte

//go:embed build/icons/tray-error.png
var trayIconError []byte

func main() {
    appInstance := NewApp()

    app := application.New(application.Options{
        Name:        "SyncDev",
        Description: "Folder Sync for Mac",
        Assets:      application.AssetOptions{
            FS: assets,
        },
        Mac: application.MacOptions{
            ActivationPolicy: application.ActivationPolicyAccessory,
        },
    })

    // Create main window (hidden by default)
    window := app.Window.NewWithOptions(application.WebviewWindowOptions{
        Title:       "SyncDev",
        Width:       1024,
        Height:      700,
        MinWidth:    800,
        MinHeight:   600,
        Hidden:      true,
        AlwaysOnTop: true,
    })

    // Hide instead of close
    window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
        window.Hide()
        e.Cancel()
    })

    // Create system tray
    systemTray := app.SystemTray.New()
    systemTray.SetTooltip("SyncDev - Folder Sync")

    // Set template icon (macOS)
    if runtime.GOOS == "darwin" {
        systemTray.SetTemplateIcon(trayIconIdle)
    }

    // Create context menu
    menu := app.NewMenu()
    menu.Add("Sync Now").OnClick(func(ctx *application.Context) {
        go appInstance.SyncNow()
    })

    pauseItem := menu.Add("Pause Sync")
    pauseItem.OnClick(func(ctx *application.Context) {
        if appInstance.IsPaused() {
            appInstance.Resume()
            pauseItem.SetLabel("Pause Sync")
        } else {
            appInstance.Pause()
            pauseItem.SetLabel("Resume Sync")
        }
    })

    menu.AddSeparator()
    menu.Add("Open SyncDev").OnClick(func(ctx *application.Context) {
        window.Show()
        window.Focus()
    })
    menu.AddSeparator()
    menu.Add("Quit").OnClick(func(ctx *application.Context) {
        app.Quit()
    })

    systemTray.SetMenu(menu)
    systemTray.AttachWindow(window).WindowOffset(5)

    // Listen for sync state changes to update icon
    app.Events.On("sync:status", func(event *application.WailsEvent) {
        data := event.Data.(map[string]interface{})
        status := data["status"].(string)
        switch status {
        case "idle":
            systemTray.SetTemplateIcon(trayIconIdle)
        case "syncing":
            systemTray.SetTemplateIcon(trayIconSyncing)
        case "error":
            systemTray.SetTemplateIcon(trayIconError)
        }
    })

    // Bind app methods
    app.Bind(appInstance)

    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Icon Specifications for macOS Menu Bar
```
Template Icon Requirements:
- Format: PNG with alpha transparency
- Size: 22x22 pixels (@1x), 44x44 pixels (@2x Retina)
- Colors: Black (#000000) + transparent only
- macOS will invert to white in dark mode automatically

State Icons:
1. tray-idle.png - Two circular arrows (sync symbol), solid black
2. tray-syncing.png - Circular arrows with motion lines, or animated
3. tray-error.png - Circular arrows with exclamation mark, or red tint

Note: For animation, swap icons on a timer (e.g., 500ms)
Opacity can indicate secondary states (35% opacity = disabled)
```

### Migration Checklist: v2 to v3
```go
// v2 Pattern (OLD)
err := wails.Run(&options.App{
    OnBeforeClose: func(ctx context.Context) bool {
        runtime.WindowHide(ctx)
        return true  // prevent close
    },
})

// v3 Pattern (NEW)
window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
    window.Hide()
    e.Cancel()
})
```

```go
// v2 Pattern (OLD)
runtime.EventsEmit(ctx, "sync:status", data)

// v3 Pattern (NEW)
app.Events.Emit(&application.WailsEvent{
    Name: "sync:status",
    Data: data,
})
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `fyne.io/systray` + Wails v2 | Wails v3 native systray | Wails v3 alpha (2025) | No more main thread conflicts |
| `HideWindowOnClose` flag | `RegisterHook` + `Cancel()` | Wails v3 | More flexible, consistent |
| `OnBeforeClose` callback | Event hooks system | Wails v3 | Unified event handling |
| Context-based runtime | Object-based APIs | Wails v3 | `window.Hide()` not `runtime.WindowHide(ctx)` |
| Remotray IPC workaround | Native systray | Wails v3 | Simpler, no external process |

**Deprecated/outdated:**
- **Remotray:** No longer needed with Wails v3, POC status, unmaintained
- **`fyne.io/systray` with Wails:** Never worked properly, main thread conflict
- **`HideWindowOnClose`:** Removed in v3, use hooks instead
- **Wails v2 for tray apps:** v2 will not receive systray support

## Open Questions

Things that couldn't be fully resolved:

1. **Animation for syncing icon**
   - What we know: Can swap icons dynamically with `SetTemplateIcon()`
   - What's unclear: Whether there's a built-in animation API or if manual timer-based swapping is needed
   - Recommendation: Start with static syncing icon, add timer-based animation later if desired

2. **Wails v3 final release timeline**
   - What we know: Alpha is stable, API finalized, used in production
   - What's unclear: When beta/stable will be released ("done when ready")
   - Recommendation: Proceed with alpha.62+, daily releases ensure rapid bug fixes

3. **Icon caching on state change**
   - What we know: `SetTemplateIcon()` accepts byte slice each time
   - What's unclear: Whether Wails caches icons internally or re-processes each call
   - Recommendation: Pre-load icon bytes at startup (as shown in examples), no performance concern

## Sources

### Primary (HIGH confidence)
- [Wails v3 Systray Documentation](https://v3alpha.wails.io/features/menus/systray/) - Official systray API
- [Wails v3 systray-basic example](https://pkg.go.dev/github.com/wailsapp/wails/v3/examples/systray-basic) - Complete working example
- [Wails v3 systray-menu example](https://pkg.go.dev/github.com/wailsapp/wails/v3/examples/systray-menu) - Menu with actions
- [Wails v3 What's New](https://v3alpha.wails.io/whats-new/) - Feature overview
- [Wails v3 Migration Guide](https://v3alpha.wails.io/migration/v2-to-v3/) - v2 to v3 changes
- [Wails v3 Releases](https://github.com/wailsapp/wails/releases) - v3.0.0-alpha.62 current

### Secondary (MEDIUM confidence)
- [Wails GitHub Discussion #4514](https://github.com/wailsapp/wails/discussions/4514) - Confirmed v2 won't support systray
- [Wails v3 Changes Summary](https://wimaha.github.io/wails-v3-alpha/development/changes/) - Detailed API changes
- [Bjango Menu Bar Extras Design](https://bjango.com/articles/designingmenubarextras/) - macOS icon specifications

### Tertiary (LOW confidence)
- [Remotray GitHub](https://github.com/Ironpark/remotray) - POC, minimal docs, not recommended

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Wails v3 is documented, has working examples
- Architecture: HIGH - Patterns from official examples
- Pitfalls: MEDIUM - Based on v3 docs and community issues, not production experience
- Migration: HIGH - Official migration guide available
- Icon specs: HIGH - Apple HIG and Bjango design guide

**Research date:** 2026-01-22
**Valid until:** 2026-04-22 (Wails v3 in active development, monitor changelog)
