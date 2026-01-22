# Domain Pitfalls: Wails System Tray, Keychain, and Native UI

**Domain:** Wails desktop application enhancements (system tray, macOS Keychain, native UI, progress)
**Researched:** 2026-01-22
**Confidence:** MEDIUM (verified with official sources where possible, LOW for Wails v3 specifics due to alpha status)

## Critical Pitfalls

Mistakes that cause rewrites, crashes, or major architectural issues.

### Pitfall 1: System Tray Library Conflicts (fyne.io/systray + Wails)

**What goes wrong:** Using `fyne.io/systray` with Wails causes runtime conflicts on macOS, leading to crashes or non-functional system tray. The existing codebase already has this disabled.

**Why it happens:**
- Fyne's systray package manages its own event loop and window lifecycle
- Wails v2 also manages window lifecycle and event dispatch
- Both attempt to hook into macOS AppKit simultaneously, creating conflicts

**Consequences:**
- Application crashes on startup or when tray is accessed
- Memory leaks from competing goroutines
- Unpredictable behavior when window show/hide is triggered from tray

**Prevention:**
- **For Wails v2:** Avoid third-party system tray libraries entirely
- **For Wails v3:** Use native Wails system tray API (v3.0.0-alpha.22+ has built-in support)
- If upgrading to v3, completely remove `fyne.io/systray` dependency from go.mod
- Test tray functionality after every Wails version upgrade

**Detection:**
- App crashes when calling `systray.Run()`
- Error logs mentioning "multiple event loops" or "AppKit thread conflict"
- System tray icon appears but menu items don't respond
- Memory grows continuously without user activity (goroutine leak)

**Sources:**
- [Wails Issue #2437](https://github.com/wailsapp/wails/issues/2437) - Community discussing system tray conflicts
- [Wails Issue #2821](https://github.com/wailsapp/wails/issues/2821) - How to use system tray
- Current codebase: `main.go` line 35 documents this exact issue

### Pitfall 2: Progress Callbacks Blocking UI Thread

**What goes wrong:** Calling progress callbacks synchronously on every file chunk (e.g., every 64KB) overwhelms the UI, causing stuttering, freezing, or dropped updates.

**Why it happens:**
- File transfers generate hundreds/thousands of progress events per second
- Each callback may trigger:
  - Wails runtime event emission (IPC to frontend)
  - Svelte reactivity updates
  - DOM reflows for progress bars
- Synchronous callbacks block the transfer goroutine, slowing actual sync

**Consequences:**
- UI becomes unresponsive during large file transfers
- Progress bar updates stutter or freeze
- Transfer performance degrades significantly (50-90% slower)
- Memory consumption spikes from queued UI updates

**Prevention:**
- **Throttle progress callbacks** to 10-20 updates per second maximum
- Use time-based throttling: only emit if >50ms since last update
- Debounce final 100% completion to ensure it always fires
- Example pattern:
```go
type ThrottledProgress struct {
    lastUpdate time.Time
    minInterval time.Duration
    callback func(*models.TransferProgress)
}

func (tp *ThrottledProgress) Update(p *models.TransferProgress) {
    now := time.Now()
    // Always send 100% completion
    if p.Progress == 100 || now.Sub(tp.lastUpdate) >= tp.minInterval {
        tp.callback(p)
        tp.lastUpdate = now
    }
}
```

**Detection:**
- UI freezes during file transfers
- Progress bar shows < 10 fps update rate
- Transfer speed much slower than network capacity
- CPU usage spikes on UI thread during transfers

**Sources:**
- Current codebase: `internal/sync/engine.go` lines 773-775, 881-883 show synchronous callback pattern
- [JavaScript Debounce vs. Throttle](https://www.syncfusion.com/blogs/post/javascript-debounce-vs-throttle) - General pattern applies to Go callbacks
- [The art of Smooth UX](https://dev.to/abhirupa/the-art-of-smooth-ux-debouncing-and-throttling-for-a-more-performant-ui-m0h) - UI performance patterns

### Pitfall 3: Keychain Access Without Code Signing

**What goes wrong:** Unsigned apps can access Keychain with user permission prompts, but macOS repeatedly prompts on every access, making the app unusable.

**Why it happens:**
- App Sandbox relies on code signature to verify app identity
- Without signature, macOS treats each launch as a "new" app
- Keychain ACLs (Access Control Lists) cannot bind to unsigned apps
- User grants permission per-session, not persistently

**Consequences:**
- User sees "SyncDev wants to access Keychain" on every app launch
- Credentials must be re-entered frequently
- Cannot use AccessGroup for app extension sharing
- App may be rejected from distribution or flagged by Gatekeeper

**Prevention:**
- **Phase 1: Development** - Accept repeated prompts during dev
- **Phase 2: Before Keychain Integration** - Implement code signing
  - Obtain Apple Developer certificate
  - Sign with `codesign -s "Developer ID Application: Your Name"`
  - Include entitlements for Keychain access
- **Phase 3: Distribution** - Notarize the app
- Use proper bundle identifier and team ID in AccessGroup
- Test signed vs unsigned to verify permission persistence

**Detection:**
- Keychain prompt appears on every app launch
- Error: `errSecInteractionNotAllowed` when trying headless access
- Users report "too many permission dialogs"
- Credentials don't persist between launches

**Sources:**
- [macOS Code Signing at 20](https://mjtsai.com/blog/2026/01/20/mac-code-signing-at-20/) - Recent (Jan 2026) discussion of code signing challenges
- [macOS distribution guide](https://gist.github.com/rsms/929c9c2fec231f0cf843a1a746a416f5) - Code signing and notarization
- [go-keychain API](https://github.com/keybase/go-keychain) - Documents `errSecInteractionNotAllowed` error

### Pitfall 4: Goroutine Leaks in Event Handlers

**What goes wrong:** System tray menu click handlers spawn goroutines that never terminate, causing memory leaks that grow with each menu interaction.

**Why it happens:**
- Menu click handlers are long-running `select` loops waiting on channels
- If window is destroyed or tray is rebuilt, old goroutines keep running
- No context cancellation or cleanup mechanism in place
- Example from current code: `systray.go` lines 45-62 has infinite `select` loop

**Consequences:**
- Memory usage grows 1-10 MB per menu rebuild
- Can leak 50,000+ goroutines over long sessions (documented in similar apps)
- Eventually causes OOM crashes
- Goroutine scheduler overhead degrades performance

**Prevention:**
- Use context cancellation for all long-running goroutines
- Store goroutine references and clean up on window close
- Pattern for tray handlers:
```go
func (s *SystemTray) onReady() {
    s.ctx, s.cancel = context.WithCancel(context.Background())

    go func() {
        for {
            select {
            case <-s.mShow.ClickedCh:
                // handle
            case <-s.ctx.Done():
                return // Clean exit
            }
        }
    }()
}

func (s *SystemTray) Cleanup() {
    if s.cancel != nil {
        s.cancel()
    }
}
```
- Call `Cleanup()` before destroying tray or on app quit

**Detection:**
- Memory usage grows continuously even when app is idle
- `runtime.NumGoroutine()` increases over time without bound
- Use `pprof` to identify leaked goroutines: `go tool pprof http://localhost:6060/debug/pprof/goroutine`
- Stack traces show many `select` statements in system tray code

**Sources:**
- [Wails v3.0.0-alpha.38 Changelog](https://v3alpha.wails.io/changelog/) - Fixed goroutine leaks in system tray
- [Wails Issue #2772](https://github.com/wailsapp/wails/issues/2772) - Memory leak report on macOS
- [Finding and Fixing 50,000 Goroutine Leak](https://skoredin.pro/blog/golang/goroutine-leak-debugging) - Real-world goroutine debugging

## Moderate Pitfalls

Mistakes that cause delays, technical debt, or require significant rework.

### Pitfall 5: Wrong Menu Bar Icon Resolution

**What goes wrong:** Menu bar icon appears blurry on Retina displays, or is wrong size (too large/small) in the menu bar.

**Why it happens:**
- macOS menu bar expects specific sizes: 22×22 pt (points, not pixels)
- On Retina (2×), this requires 44×44 px version
- Single-resolution PNGs scale poorly
- Wrong template mode causes icon to not adapt to dark/light mode

**Consequences:**
- Icon looks unprofessional (blurry or pixelated)
- Doesn't invert in dark mode
- May be truncated if too large

**Prevention:**
- Provide both 1× (22×22 px) and 2× (44×44 px) versions, or use SVG
- Use monochrome template images that macOS can tint
- Name files with `@2x` suffix: `icon.png` and `icon@2x.png`
- OR embed template PNG with proper alpha channel
- Test on both Retina and non-Retina displays
- Test in both light and dark modes

**Current Issue:**
- `systray.go` line 73-95 embeds a 22×22 PNG
- Needs verification: Is it a template image? Does it have 2× version?

**Sources:**
- [Designing macOS menu bar extras](https://bjango.com/articles/designingmenubarextras/) - Authoritative guide on menu bar icon specs
- [ResolutionMenu](https://github.com/robbertkl/ResolutionMenu) - Example of proper HiDPI icon handling

### Pitfall 6: Wails v3 API Breaking Changes

**What goes wrong:** Code examples in Wails v3 documentation have type mismatches, causing compilation errors.

**Why it happens:**
- Wails v3 is in alpha (currently alpha.38+ as of Nov 2025)
- Documentation lags behind API changes
- System tray callback signatures changed: now require `*application.Context` parameter

**Consequences:**
- Copy-paste from docs doesn't compile
- Error: `Cannot use 'func()' as the type func(*Context)`
- Developer frustration and time wasted debugging

**Prevention:**
- Always check Wails version in go.mod
- When upgrading Wails, read full changelog
- If using v3 alpha, expect documentation errors
- Cross-reference docs with working examples in `/examples` directory
- Update all `OnClick` callbacks to include context:
```go
// Wrong (from old docs)
menu.Add("Quit").OnClick(func() { app.Quit() })

// Correct (v3 alpha)
menu.Add("Quit").OnClick(func(ctx *application.Context) { app.Quit() })
```

**Detection:**
- Compilation error mentioning function signature mismatch
- Error references `*application.Context` or `*Context`

**Sources:**
- [Wails Issue #4137](https://github.com/wailsapp/wails/issues/4137) - Documented API mismatch in system tray example
- [Wails v3 Changelog](https://v3alpha.wails.io/changelog/) - Alpha status and recent changes

### Pitfall 7: Non-Native Font Rendering on macOS

**What goes wrong:** Web-based UI in Wails uses generic sans-serif fonts instead of San Francisco, making the app feel "un-Mac-like."

**Why it happens:**
- Default CSS doesn't specify system fonts
- Wails renders web content in WebView, which defaults to web fonts
- San Francisco font cannot be embedded via `@font-face` (licensing)
- Must use CSS system font keywords instead

**Consequences:**
- App looks foreign on macOS (Helvetica Neue or Arial instead of San Francisco)
- Users perceive app as less polished or not "native"
- Inconsistent with macOS Human Interface Guidelines

**Prevention:**
- Use system font stack in CSS:
```css
body {
    font-family: -apple-system, BlinkMacSystemFont,
                 "Segoe UI", "Roboto", "Oxygen",
                 "Ubuntu", "Helvetica Neue", Arial, sans-serif;
}
```
- `-apple-system` maps to San Francisco on macOS/iOS
- `BlinkMacSystemFont` ensures Chromium-based WebView uses it
- Test on actual macOS device (not just in browser)

**Svelte-Specific:**
- Add to global CSS in `app.css` or root layout
- Don't hardcode font in component styles

**Sources:**
- [System Font Stack](https://systemfontstack.com/) - Authoritative cross-platform font stack
- [Leveraging System Fonts on the Web](https://blog.jim-nielsen.com/2020/system-fonts-on-the-web/) - Best practices
- [CSS System Fonts](https://thevalleyofcode.com/css-system-fonts/) - Browser support details

### Pitfall 8: Vibrancy/Transparency Window Issues

**What goes wrong:** Enabling window vibrancy (blur-behind effect) on macOS causes white backgrounds, shadow rendering issues, or vibrancy loss when window loses focus.

**Why it happens:**
- macOS vibrancy requires specific window configuration
- Electron (and potentially Wails) has ongoing bugs with vibrancy + transparency
- Vibrancy requires continuous background sampling and re-blurring, which is expensive
- Improper configuration can overload WindowServer process

**Consequences:**
- Window shows white instead of blurred background
- Vibrancy disappears when switching apps
- Window shadows render incorrectly in Mission Control
- High CPU usage (WindowServer process)
- Performance degradation

**Prevention:**
- **For Wails v2:** Check if vibrancy is supported and stable in your version
- **For Wails v3:** Verify vibrancy API status (may be improved)
- Test vibrancy on actual macOS hardware, not VMs
- Measure WindowServer CPU usage during testing
- Consider vibrancy as "nice to have," not required feature
- Provide fallback: solid background with slight transparency
- If implementing:
  - Test with both active and inactive windows
  - Test in Mission Control and Split View
  - Test on different macOS versions (Monterey, Ventura, Sonoma, Sequoia)

**Detection:**
- Window background is white instead of blurred
- Vibrancy effect disappears after alt-tabbing
- High CPU usage even when app is idle
- Users report "window looks broken"

**Sources:**
- [Electron Issue #31862](https://github.com/electron/electron/issues/31862) - Vibrancy + transparency broken on macOS
- [Electron Issue #46164](https://github.com/electron/electron/issues/46164) - Vibrancy lost in inactive windows (2025)
- [Understanding WindowServer](https://andreafortuna.org/2025/10/05/macos-windowserver) - Performance impact of transparency

## Minor Pitfalls

Mistakes that cause annoyance but are fixable without major refactoring.

### Pitfall 9: Keychain API Non-Idiomatic Go

**What goes wrong:** The `go-keychain` library API feels "un-Go-like," causing confusion for Go developers.

**Why it happens:**
- Library intentionally mirrors macOS/iOS Keychain C API
- Uses patterns like `item.SetAccessGroup()` instead of options struct
- Error handling doesn't follow Go conventions exactly

**Consequences:**
- Code feels foreign to Go developers
- Learning curve higher than expected
- May lead to incorrect usage patterns

**Prevention:**
- Read library README carefully - it explicitly warns about this
- Wrap library in your own idiomatic interface if needed:
```go
type KeychainStore struct {
    serviceName string
}

func (ks *KeychainStore) Save(account, password string) error {
    item := keychain.NewItem()
    item.SetSecClass(keychain.SecClassGenericPassword)
    item.SetService(ks.serviceName)
    item.SetAccount(account)
    item.SetData([]byte(password))
    return keychain.AddItem(item)
}
```
- This provides Go-friendly interface while using library underneath

**Sources:**
- [go-keychain README](https://github.com/keybase/go-keychain) - Documents non-idiomatic design decision

### Pitfall 10: Hidden Window Behavior on Windows (Wails v3 Regression)

**What goes wrong:** In Wails v3 alpha 22+, setting `Hidden: true` on Windows results in a non-hidden, non-interactive blank window.

**Why it happens:**
- Regression introduced in Wails v3.0.0-alpha.22
- Affects Windows platform specifically
- Attempting to hide window immediately after creation also fails

**Consequences:**
- Can't create background-only apps with system tray
- Blank window confuses users
- Breaking change from previous alpha versions

**Prevention:**
- If using Wails v3 alpha 22+, test `Hidden: true` immediately
- Check Wails GitHub issues for regression status
- Workaround: Use alpha 21 until fixed, or avoid hidden windows
- Monitor Wails changelog for fix

**Detection:**
- Blank window appears despite `Hidden: true`
- Window is non-interactive (can't click or focus)

**Sources:**
- [Wails Issue #4498](https://github.com/wailsapp/wails/issues/4498) - Hidden window regression on Windows (Aug 2025)

### Pitfall 11: System Tray Not Showing After Taskbar Restart

**What goes wrong:** System tray icon disappears if Windows Explorer crashes or user restarts taskbar, and never returns.

**Why it happens:**
- Windows sends notification when taskbar restarts, but app must re-register tray
- Not handled automatically by some tray libraries

**Consequences:**
- User loses access to tray menu after Explorer crash
- Must fully restart app to restore tray icon

**Prevention:**
- **Wails v3 solution:** Fixed in recent alphas - tray registration recovers automatically
- Listen for `TaskbarCreated` Windows message if using custom implementation
- Wails v3 improvements:
  - Reuses resolved icons on re-registration
  - Sets `NOTIFYICON_VERSION_4` for better tooltip recovery
  - Enables `NIF_SHOWTIP` so tooltips work after Explorer restart

**Detection:**
- Restart Windows Explorer (`taskkill /f /im explorer.exe && start explorer.exe`)
- Check if tray icon reappears

**Sources:**
- [Wails v3 Changelog](https://v3alpha.wails.io/changelog/) - System tray improvements

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation | Priority |
|-------------|---------------|------------|----------|
| System Tray Implementation | Library conflicts (Pitfall #1) | Use Wails v3 native API, remove fyne.io/systray | CRITICAL |
| System Tray Implementation | Goroutine leaks (Pitfall #4) | Implement context cancellation | CRITICAL |
| System Tray Implementation | Icon resolution (Pitfall #5) | Provide @2x assets, test on Retina | MEDIUM |
| Progress UI | Callback flooding (Pitfall #2) | Throttle to 10-20 Hz | CRITICAL |
| Progress UI | UI thread blocking | Use non-blocking callback dispatch | HIGH |
| Keychain Integration | Unsigned app prompts (Pitfall #3) | Implement code signing before Keychain | CRITICAL |
| Keychain Integration | Non-idiomatic API (Pitfall #9) | Wrap in Go-friendly interface | LOW |
| Native UI Styling | Non-native fonts (Pitfall #7) | Use system font stack | MEDIUM |
| Native UI Styling | Vibrancy issues (Pitfall #8) | Defer or use fallback | LOW |
| Wails v3 Migration | API breaking changes (Pitfall #6) | Reference changelog, verify examples | HIGH |
| Windows Support | Hidden window regression (Pitfall #10) | Test on Windows, monitor issues | MEDIUM |

## Recommended Phase Ordering

Based on dependency analysis of pitfalls:

1. **Phase 1: Code Signing Setup**
   - Must happen BEFORE Keychain integration (Pitfall #3)
   - Affects distribution strategy
   - One-time setup cost, enables future features

2. **Phase 2: System Tray (Wails v3 Native)**
   - Migrate from Wails v2 to v3 (or investigate v2 native options)
   - Remove fyne.io/systray completely (Pitfall #1)
   - Implement context cancellation from start (Pitfall #4)
   - Fix icon resolution (Pitfall #5)

3. **Phase 3: Progress UI Optimization**
   - Implement throttling before adding more UI features (Pitfall #2)
   - Independent of other features, can parallelize

4. **Phase 4: Keychain Integration**
   - Depends on Phase 1 (code signing)
   - Wrap API for ergonomics (Pitfall #9)

5. **Phase 5: Native UI Polish**
   - System font stack (Pitfall #7)
   - Vibrancy (optional, Pitfall #8)
   - Last because it's polish, not functionality

## Cross-Cutting Concerns

Issues that affect multiple phases:

### Memory Management
- Goroutine leaks (Pitfall #4): Affects system tray, progress callbacks, any event handlers
- Solution: Establish context cancellation pattern early, apply everywhere

### Testing Strategy
- Test on actual macOS hardware, not just VMs
- Test both Retina and non-Retina displays
- Test with unsigned and signed builds
- Test system tray recovery (restart Explorer/Dock)

### Wails Version Strategy
- **Current:** Wails v2 (stable but limited system tray support)
- **Target:** Wails v3 (alpha, but has native system tray)
- **Risk:** v3 is alpha, has known issues (Pitfall #6, #10)
- **Decision point:** Evaluate v3 stability before committing to migration

## Sources Summary

### HIGH Confidence (Official/Verified)
- Wails official changelog and issues
- macOS Human Interface Guidelines
- Apple Developer documentation
- Official library READMEs (go-keychain)

### MEDIUM Confidence (Community/Recent)
- Recent blog posts (2025-2026) on code signing
- Electron vibrancy issues (applies to Wails via shared WebView challenges)
- Bjango menu bar design guide (industry standard)

### LOW Confidence (Needs Validation)
- Specific Wails v3 API stability (alpha state)
- Performance characteristics (throttling thresholds)
- Vibrancy support in current Wails versions

## Open Questions for Phase-Specific Research

- [ ] Wails v3 stability timeline - when will it reach beta/stable?
- [ ] Does Wails v3 system tray API handle all Pitfall #4 concerns internally?
- [ ] What's the actual performance of progress callbacks at 1 Hz vs 10 Hz vs 100 Hz?
- [ ] Can we use `zalando/go-keyring` instead of `keybase/go-keychain` for better API?
- [ ] Does Wails support window vibrancy at all, or only Electron?
