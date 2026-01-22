# Technology Stack: SyncDev Improvements

**Project:** SyncDev macOS Enhancements
**Researched:** 2026-01-22
**Confidence:** HIGH for system tray and Keychain, MEDIUM for UI framework choices

## Current Stack (Baseline)

| Technology | Version | Purpose |
|------------|---------|---------|
| Wails | v2.11.0 | Desktop application framework |
| Go | 1.23 | Backend/business logic |
| Svelte | 3.49 | Frontend framework |
| Vite | 3.0.7 | Frontend build tool |

## Recommended Stack for Improvements

### System Tray Solution

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| **Wails v3** (recommended) | v3.0.0-alpha.55+ | Upgrade path with native systray | Native systray support, eliminates conflicts, future-proof. Alpha is stable enough for production use. |
| **Remotray** (v2 workaround) | v0.1.7 | System tray for Wails v2 | If staying on v2, solves fyne.io/systray conflict via separate process IPC. POC but functional on macOS. |

**Confidence:** HIGH
**Source:** [Wails v3 Changelog](https://v3alpha.wails.io/changelog/), [Wails Discussion #4514](https://github.com/wailsapp/wails/discussions/4514), [Remotray GitHub](https://github.com/Ironpark/remotray)

#### Decision Matrix: Systray Approach

| Criterion | Wails v3 | Remotray + Wails v2 | fyne.io/systray (current) |
|-----------|----------|---------------------|---------------------------|
| **Conflict-free** | Yes | Yes (separate process) | No (main thread conflict) |
| **Maintenance** | Active, daily releases | POC, last update July 2022 | N/A - incompatible |
| **macOS Support** | Full, with light/dark icons | Tested, functional | N/A - incompatible |
| **Implementation Effort** | Medium (migration required) | Low (drop-in for v2) | N/A - blocked |
| **Risk** | Low (alpha is stable, used in production) | Medium (POC status) | High (doesn't work) |

**Recommendation:** **Migrate to Wails v3** because:
1. Native systray support eliminates architectural conflicts
2. v3.0.0-alpha.55 (Jan 2, 2026) is stable enough for production use
3. Daily releases show active development momentum
4. Future-proof: v2 will not receive systray support
5. Template icon support for macOS light/dark mode
6. Avoids technical debt of workaround solutions

**If v3 migration is deferred:** Use Remotray as interim solution. While POC status is concerning, it's the only battle-tested workaround for Wails v2 on macOS.

### macOS Keychain Integration

| Library | Version | Purpose | Why |
|---------|---------|---------|-----|
| **zalando/go-keyring** (recommended) | v0.2.6 | Cross-platform keyring access | Statically linked (no cgo), cross-platform, actively maintained (Oct 2024 release). Uses `/usr/bin/security` on macOS. |
| keybase/go-keychain | v0.0.1 | macOS/iOS Keychain access | Latest release Feb 2025, but v0.0.1 suggests early stage. macOS-specific, requires macOS 10.9+. |
| 99designs/keyring | Latest | Advanced keychain features | More configuration options but heavier dependency. |

**Confidence:** HIGH
**Sources:** [zalando/go-keyring GitHub](https://github.com/zalando/go-keyring), [keybase/go-keychain GitHub](https://github.com/keybase/go-keychain)

**Recommendation:** **zalando/go-keyring** because:
1. No cgo required = easier cross-compilation and distribution
2. Cross-platform abstraction (macOS Keychain, Windows Credential Manager, Linux Secret Service)
3. Simple API: `Set()`, `Get()`, `Delete()`
4. Actively maintained with v0.2.6 release in Oct 2024
5. Statically linked binaries avoid deployment complexity
6. Future-proof if you need Windows/Linux support

**Migration Path:**
```go
// Install
go get github.com/zalando/go-keyring

// Replace plaintext JSON storage with:
import "github.com/zalando/go-keyring"

// Store secret
err := keyring.Set("SyncDev", "peer-auth-token", secret)

// Retrieve secret
secret, err := keyring.Get("SyncDev", "peer-auth-token")

// Delete secret
err := keyring.Delete("SyncDev", "peer-auth-token")
```

### Frontend Framework & UI Components

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| **Svelte** | 5.46.0+ | Frontend framework | Upgrade from 3.49 to leverage runes, performance improvements, better reactivity. Compatible with Wails. |
| **shadcn-svelte** | 1.1.1+ | Component library | Accessible, customizable components. Svelte 5 compatible. Copy-paste approach = no proprietary lock-in. |
| **Bits UI** | 2.15.4+ | Headless component primitives | Built on Melt UI, Svelte 5 native, handles accessibility and interaction logic. |
| **Tailwind CSS** | 4.x | Utility-first CSS | Industry standard, works seamlessly with shadcn-svelte and Bits UI. Fast development. |

**Confidence:** HIGH for libraries, MEDIUM for macOS-native appearance approach
**Sources:** [shadcn-svelte GitHub](https://github.com/huntabyte/shadcn-svelte), [Bits UI GitHub](https://github.com/huntabyte/bits-ui), [Svelte 5 Migration](https://www.shadcn-svelte.com/docs/migration/svelte-5)

**Recommendation:** **Svelte 5 + shadcn-svelte + Tailwind CSS** because:
1. Svelte 5 is production-ready (v5.46.0, Jan 2026) with modern features
2. shadcn-svelte actively maintained (175 contributors, 95 releases, v1.1.1 Jan 18, 2026)
3. Full Svelte 5 support with migration guides
4. Bits UI (dependency) handles accessibility and headless primitives
5. No proprietary lock-in: copy components into your codebase
6. Tailwind CSS provides rapid styling with design system constraints
7. Easy integration with Wails (SvelteKit guides available)

#### macOS-Native UI Approach

**Finding:** No off-the-shelf CSS framework provides true macOS-native appearance.

**Approaches Evaluated:**

| Approach | Pros | Cons | Verdict |
|----------|------|------|---------|
| **Photon Kit** | Cocoa style support | Unmaintained (2+ years), Electron-focused | Avoid - stale |
| **system.css** | Retro macOS look | System 6 aesthetic (1984-1991), not modern | Avoid - wrong era |
| **Custom CSS Variables** | Full control | High effort, must track macOS design changes | Viable for Phase 2 |
| **shadcn-svelte + Custom Styling** | Modern components + macOS theming | Requires CSS craftsmanship | **Recommended** |

**Recommended Strategy for Native macOS Appearance:**

1. **Phase 1: Use shadcn-svelte with subtle macOS theming**
   - Use Bits UI headless components for full styling control
   - Apply Tailwind utilities with custom macOS-inspired design tokens:
     - Font: SF Pro (system font on macOS)
     - Colors: Match macOS light/dark mode palettes
     - Border radius: macOS standard (8px for panels, 6px for buttons)
     - Shadows: Subtle, like macOS panels
     - Translucency effects using backdrop-blur

2. **Phase 2: Advanced macOS Integration (if needed)**
   - Implement CSS custom properties matching macOS design system
   - Use Wails runtime theme detection for light/dark mode
   - Consider window vibrancy effects via Wails macOS options

**Example macOS Design Tokens (Tailwind config):**
```js
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      fontFamily: {
        sans: ['-apple-system', 'BlinkMacSystemFont', 'SF Pro', 'system-ui'],
      },
      colors: {
        'macos-gray-1': '#F5F5F7',  // Light mode background
        'macos-gray-2': '#1D1D1F',  // Dark mode background
        'macos-blue': '#007AFF',     // macOS accent blue
      },
      borderRadius: {
        'macos-panel': '8px',
        'macos-button': '6px',
      },
      backdropBlur: {
        'macos': '20px',  // macOS translucency
      }
    }
  }
}
```

### Build & Development Tools

| Tool | Version | Purpose | Why |
|------|---------|---------|-----|
| Vite | 5.x | Frontend build tool | Upgrade from 3.0.7 for faster builds, better HMR, Svelte 5 optimizations |
| TypeScript | 5.x | Type safety | Recommended for Svelte 5 projects, better DX with Wails bindings |

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| System Tray (v2) | Remotray | getlantern/systray | Remotray uses fyne.io/systray fork which is more actively maintained |
| System Tray (v3) | Wails v3 native | Remotray | v3 eliminates need for workarounds, native support is cleaner |
| Keychain | zalando/go-keyring | keybase/go-keychain | keybase is macOS-only, still v0.0.1, cgo-dependent |
| Keychain | zalando/go-keyring | 99designs/keyring | go-keyring simpler API, statically linked, less config overhead |
| UI Library | shadcn-svelte | Flowbite-Svelte | shadcn has better Svelte 5 support, copy-paste approach, more active (1.1.1 Jan 2026) |
| UI Library | shadcn-svelte | Svelte Material UI | Material Design not macOS-native, shadcn more customizable |
| UI Library | shadcn-svelte | SvelteUI | shadcn more mature (8.2k stars vs lower), better Svelte 5 migration path |

## Installation

### Option A: Wails v3 Migration (Recommended)

```bash
# Install Wails v3
go install github.com/wailsapp/wails/v3/cmd/wails3@latest

# Create new v3 project or migrate existing
# Follow migration guide: https://v3alpha.wails.io/

# Update go.mod
go get github.com/wailsapp/wails/v3@latest

# Keychain integration
go get github.com/zalando/go-keyring@latest

# Frontend: Upgrade to Svelte 5
cd frontend
npm install svelte@latest
npx sv migrate svelte-5  # Official Svelte migration tool

# Install shadcn-svelte
npx shadcn-svelte@latest init

# Install supporting libraries
npm install -D tailwindcss@latest postcss autoprefixer
npm install bits-ui @melt-ui/svelte
```

### Option B: Wails v2 with Remotray (Interim)

```bash
# Keep Wails v2
# go get github.com/wailsapp/wails/v2@v2.11.0  # Already installed

# Add Remotray for systray
go get github.com/Ironpark/remotray@latest

# Keychain integration
go get github.com/zalando/go-keyring@latest

# Frontend: Upgrade to Svelte 5
cd frontend
npm install svelte@latest
npx sv migrate svelte-5

# Install shadcn-svelte
npx shadcn-svelte@latest init

# Install supporting libraries
npm install -D tailwindcss@latest postcss autoprefixer
npm install bits-ui @melt-ui/svelte
```

## Implementation Priorities

### Phase 1: Foundation (Recommended First)
1. **Keychain Integration** - Highest value, lowest risk
   - Replace plaintext JSON with zalando/go-keyring
   - Migrate existing secrets to Keychain
   - Update config loading logic
   - Test cross-device sync with Keychain storage

2. **UI Modernization** - High value, medium effort
   - Upgrade Svelte 3.49 → 5.46.0+
   - Install shadcn-svelte + Tailwind
   - Refactor existing components using Bits UI primitives
   - Apply macOS-inspired design tokens

### Phase 2: System Tray (Higher Risk, Recommend Research First)
1. **Evaluate Wails v3 Stability**
   - Test v3 alpha with SyncDev codebase
   - Verify compatibility with existing Go backend
   - Assess migration effort vs. Remotray workaround

2. **Implement System Tray**
   - If v3: Use native systray API
   - If v2: Implement Remotray with IPC
   - Add menu items: Show, Sync Now, Quit
   - Add tray icon with light/dark mode support

## Risk Assessment

| Component | Risk Level | Mitigation |
|-----------|-----------|------------|
| Wails v3 migration | Medium | Alpha stable enough for production, daily releases, test thoroughly before commit |
| Remotray (v2 path) | Medium-High | POC status, last update July 2022, limited to macOS, test extensively |
| Svelte 5 migration | Low | Production-ready, official migration tools, excellent documentation |
| shadcn-svelte | Low | Actively maintained (175 contributors), Svelte 5 ready, v1.1.1 Jan 2026 |
| zalando/go-keyring | Low | Stable v0.2.6 (Oct 2024), simple API, widely used |
| macOS-native UI | Medium | No framework provides native look, requires custom CSS craftsmanship |

## Open Questions

1. **Wails v3 Timeline:** When will v3 reach stable release?
   - **Current Answer:** No fixed date, "done when it's ready", but alpha stable enough for production
   - **Action:** Monitor releases, consider early adoption given daily updates

2. **Remotray Maintenance:** Is Remotray still maintained?
   - **Current Answer:** Last update July 2022, POC status
   - **Action:** Test thoroughly, consider contributing fixes if issues found

3. **Svelte 5 Breaking Changes:** What breaks in migration from 3.49 to 5.x?
   - **Current Answer:** Use `npx sv migrate svelte-5` for automated migration
   - **Action:** Review migration guide, test components individually

## Version Pinning Recommendations

**For Production Stability:**

```toml
# go.mod
github.com/wailsapp/wails/v3 v3.0.0-alpha.55  // Pin to tested alpha version
github.com/zalando/go-keyring v0.2.6
// github.com/Ironpark/remotray v0.1.7  // If using v2 path
```

```json
// frontend/package.json
{
  "dependencies": {
    "svelte": "^5.46.0",
    "bits-ui": "^2.15.4"
  },
  "devDependencies": {
    "shadcn-svelte": "^1.1.1",
    "tailwindcss": "^4.0.0",
    "vite": "^5.0.0"
  }
}
```

## Sources

### System Tray Research
- [Wails v2 SysTray Discussion #4514](https://github.com/wailsapp/wails/discussions/4514) - Confirmed v2 won't support systray
- [Wails Discussion #1438](https://github.com/wailsapp/wails/discussions/1438) - Community workaround discussions
- [Wails v3 Changelog](https://v3alpha.wails.io/changelog/) - Native systray features, alpha.55 release Jan 2, 2026
- [Remotray GitHub](https://github.com/Ironpark/remotray) - POC systray solution for Wails v2 via separate process
- [Wails v3 Release Discussion #4447](https://github.com/wailsapp/wails/discussions/4447) - "Done when ready" release philosophy

### Keychain Integration Research
- [keybase/go-keychain GitHub](https://github.com/keybase/go-keychain) - v0.0.1, Feb 2025, macOS-specific
- [zalando/go-keyring GitHub](https://github.com/zalando/go-keyring) - v0.2.6, Oct 2024, cross-platform, no cgo
- [99designs/keyring Package](https://pkg.go.dev/github.com/99designs/keyring) - Advanced features, heavier dependency

### UI Component Research
- [shadcn-svelte GitHub](https://github.com/huntabyte/shadcn-svelte) - v1.1.1, Jan 18, 2026, 8.2k stars, 175 contributors
- [shadcn-svelte Svelte 5 Migration](https://www.shadcn-svelte.com/docs/migration/svelte-5) - Official migration guide
- [Bits UI GitHub](https://github.com/huntabyte/bits-ui) - v2.15.4, Jan 5, 2026, Svelte 5 native
- [Melt UI GitHub](https://github.com/melt-ui/melt-ui) - v0.86.6, March 2025, headless components
- [Svelte 5 Release](https://svelte.dev/blog/svelte-5-is-alive) - Production-ready, v5.46.0 Jan 2026
- [Best Svelte UI Libraries 2025](https://componentlibraries.com/collection/best-svelte-ui-component-libraries-in-2025) - Ecosystem survey

### macOS Design Research
- [Photon Kit](https://photonkit.com/) - Electron UI kit with Cocoa style, unmaintained 2+ years
- [system.css](https://sakofchit.github.io/system.css/) - Retro macOS System 6 design
- [Tauri macOS Native UI Tips](https://dev.to/akr/8-tips-for-creating-a-native-look-and-feel-in-tauri-applications-3loe) - Theme integration patterns
- [macOS Big Sur Control Center CSS](https://github.com/StephenMcVicker/macos-bigsur-controlcenter-css) - CSS replication example

## Next Steps

1. **Roadmap Creation:** Use this stack research to structure implementation phases
2. **Spike: Wails v3 Migration** - Allocate 1-2 days to test v3 compatibility before committing
3. **Prototype: Keychain Integration** - Implement zalando/go-keyring in isolated branch, verify macOS Keychain behavior
4. **Design: macOS Theme Tokens** - Create Tailwind config with macOS design system variables
5. **Decision: Systray Approach** - After v3 spike, decide between migration or Remotray workaround
