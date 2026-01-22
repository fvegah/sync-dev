# Feature Landscape: File Sync App UX

**Domain:** Desktop file synchronization app (macOS)
**Researched:** 2026-01-22
**Focus:** Progress display, system tray, native macOS UI

## Table Stakes

Features users expect. Missing = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **System tray icon with status** | Every sync app (Dropbox, Google Drive, Syncthing) has menu bar presence. Users expect "always running" apps to live in the menu bar, not Dock | Low | macOS calls this "menu bar extra" not system tray |
| **Icon state variations** | Visual at-a-glance sync status without opening app. Dropbox uses different icons for idle/syncing/error | Low-Med | Use template images (monochrome) that auto-adapt to light/dark mode. 35% opacity = disabled state |
| **Global progress indicator** | Users need to know "how much is left?" for entire sync operation. All modern sync apps show overall percentage | Med | Display % complete with total files/bytes remaining |
| **Transfer speed display** | Critical for managing expectations. Users judge performance by MB/s metric | Low | Update every 1-2 seconds. Use exponential smoothing algorithm for stability |
| **ETA calculation** | Users need time estimate to decide "can I close laptop now?" Standard in rsync, Dropbox, all transfer tools | Med | Wait 10-15s for steady state before showing ETA. Mark as "calculating..." initially to avoid wild fluctuations |
| **Per-file progress** | For large files, users want to see individual file progress to know system isn't frozen | Med | Show filename, size, % complete, speed for current file |
| **Pause/Resume sync** | Users need control over bandwidth usage and timing. Table stakes since Dropbox early days | Med | Persist pause state across app restarts |
| **Native macOS look** | macOS users expect apps to follow platform conventions. Non-native apps feel "janky" and untrustworthy | High | Follow HIG for typography (13pt body), spacing, SF Symbols, translucent sidebars |
| **Basic menu bar menu** | Minimum: Open app, Quit. Users expect right-click on menu bar icon for quick actions | Low | Standard NSMenu with SF Symbol icons for items |
| **File list of active transfers** | Users want to see what's transferring NOW, not just overall progress. WinSCP, rsync --progress pattern | Med | Show currently syncing files with individual progress |
| **Full-height sidebar** | macOS Big Sur+ pattern. Sidebars span entire window height with translucent background | Med | Use SF Symbols with app accent color. Inset style for rows |
| **Light/Dark mode support** | Expected since Mojave (2018). Apps that ignore dark mode look broken | Low | Use template images for menu bar. System colors in UI |

## Differentiators

Features that set product apart. Not expected, but valued.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **LAN-only speed indicator** | Show peer-to-peer LAN speeds (often 100+ MB/s) to highlight "faster than cloud sync" advantage | Low | Emphasize speed difference vs cloud solutions. Marketing win |
| **Real-time file queue visualization** | Live updating list showing "waiting" vs "transferring" vs "completed" with smart grouping | Med-High | WinSCP has this. Shows completed in gray, in-progress highlighted, waiting dimmed |
| **Bandwidth throttling UI** | Let users set max speed to avoid saturating network. Resilio has this | Med | Slider in preferences: "Max upload/download speed" |
| **Conflict resolution preview** | Show file conflicts BEFORE sync with side-by-side diff preview | High | Git-like conflict handling. Huge UX win but complex |
| **Per-folder-pair status** | Show sync status per folder pair, not just global. Lets users see "Work synced, Photos pending" | Med | Sidebar list with status badges per folder pair |
| **Transfer history log** | Searchable log of what synced when. Useful for debugging "where did my file go?" | Med | Simple list view with filters. Export as CSV |
| **Smart notifications** | Only notify on errors or completion, not every file. macOS native notifications | Low-Med | User configurable: errors only, completions, or all activity |
| **Menubar icon badge count** | Show number of pending files on menu bar icon (like Mail app unread count) | Low | Subtle way to convey "12 files waiting" without opening app |
| **Estimated data remaining** | Show "450 MB of 2.1 GB remaining" in addition to % | Low | Helps users gauge if they have enough bandwidth/time |
| **Network activity graph** | Mini sparkline graph showing transfer rate over last 30s-1min | Med | Visual indicator of network health. Fun but not critical |
| **Drag-and-drop folder pairing** | Drag folder onto menu bar icon to add as sync pair | Med | Finder integration. Very Mac-like interaction |
| **Quick Actions in menu** | Menu bar menu with "Sync Now", "Pause All", "Open Folder" shortcuts | Low | Reduces clicks. Power user feature |

## Anti-Features

Features to explicitly NOT build. Common mistakes in this domain.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Show every file in main window** | Causes UI to freeze/lag with thousands of files. Dropbox sync queue complaints about this | Show max 50-100 files with "View all in log" link |
| **Auto-sync without pause option** | Users hate losing control. "My sync is eating bandwidth" complaints are common | Always provide pause/resume. Respect user's time/bandwidth |
| **Cloud storage integration** | Scope creep. Trying to compete with Dropbox/Drive on cloud features dilutes LAN-sync focus | Stay focused on LAN P2P. Cloud is different product |
| **File versioning/backup** | Complex feature that requires storage strategy, UI for browsing versions, retention policies | Out of scope for v1. Mark as "future consideration" |
| **Real-time collaboration (OT/CRDT)** | Extremely complex. Google Docs-level engineering for marginal benefit in file sync | Sync files, not edits. Let apps handle collaboration |
| **Sync scheduling** | Adds complexity. Users want "always on" sync or manual trigger, rarely "sync at 3pm daily" | Provide auto-sync interval setting (every 5min, 30min, etc) but not cron-like scheduling |
| **Mobile app** | Different UX paradigm, battery concerns, cellular data. Dilutes macOS focus | Desktop-only for now. Mobile is separate product decision |
| **Encryption configuration UI** | Exposing crypto settings (cipher, key length) confuses users and creates support burden | Use secure defaults (AES-256, TLS 1.3). No config needed |
| **Detailed network stats** | Packet loss, latency graphs, TCP window size. Overwhelming for general users | Show simple speed/ETA. Advanced stats in hidden debug panel if needed |
| **File permission sync** | Cross-platform permission mapping is nightmare. Unix vs NTFS vs APFS permissions don't align | Sync content only. Preserve local permissions |
| **Background auto-update without notice** | macOS users expect to control when apps update. Silent updates feel invasive | Use Sparkle framework with user-initiated update checks |
| **Non-standard UI controls** | Custom scrollbars, dropdowns, checkboxes break muscle memory and accessibility | Use native AppKit/SwiftUI controls exclusively |

## Feature Dependencies

```
Menu Bar Icon
  ├─> Icon State Variations (requires status tracking)
  ├─> Menu Bar Menu (requires menu bar icon to exist)
  └─> Quick Actions (requires menu bar menu)

Progress Display
  ├─> Global Progress (foundation)
  ├─> Per-file Progress (requires file tracking)
  ├─> Transfer Speed (requires bytes/time tracking)
  └─> ETA Calculation (requires speed + remaining bytes)

File List UI
  ├─> Active Transfers List (foundation)
  ├─> Queue Visualization (requires transfer queue data structure)
  └─> Transfer History (requires persistence of completed transfers)

Native macOS UI
  ├─> Light/Dark Mode (required for "native" feel)
  ├─> SF Symbols (visual consistency with system)
  ├─> Full-height Sidebar (Big Sur+ pattern)
  └─> Native Controls (accessibility + platform integration)

Advanced Features (post-MVP)
  ├─> Conflict Resolution (requires diff engine)
  ├─> Bandwidth Throttling (requires rate limiting in sync engine)
  └─> Per-folder-pair Status (requires multi-folder state management)
```

## MVP Recommendation

For this milestone (UX improvements), prioritize:

### Phase 1: Menu Bar Integration (Table Stakes)
1. Menu bar icon with template image
2. Icon state variations (idle, syncing, error)
3. Basic menu (Open, Sync Now, Pause, Quit)
4. Light/dark mode support

**Rationale:** Addresses TRAY-01, TRAY-02, TRAY-03. Without menu bar, app feels incomplete.

### Phase 2: Progress Display (Table Stakes)
1. Global progress bar with % complete
2. Transfer speed display (MB/s)
3. ETA calculation with smoothing
4. Per-file progress for current transfer

**Rationale:** Addresses PROG-01, PROG-02, PROG-03. Users need visibility into what's happening.

### Phase 3: File List (Table Stakes)
1. Active transfers list (max 50-100 items)
2. Show file name, size, % complete
3. Group by status (transferring, waiting, completed)

**Rationale:** Addresses PROG-04. Completes the "progress visibility" feature set.

### Phase 4: Native macOS Polish (Table Stakes)
1. Full-height sidebar with SF Symbols
2. Native AppKit controls throughout
3. macOS typography (13pt SF Pro body)
4. Translucent backgrounds

**Rationale:** Addresses UI-01, UI-02. Makes app feel professional and trustworthy.

### Defer to Post-MVP

**Differentiators** (nice-to-have, not critical):
- Conflict resolution preview (complex, needs separate research)
- Transfer history log (useful but not urgent)
- Bandwidth throttling UI (niche use case)
- Network activity graph (eye candy)
- Drag-and-drop pairing (polish feature)

**Advanced Table Stakes** (functional but can wait):
- Per-folder-pair status (requires architecture changes)
- Smart notifications (polish on top of working sync)

**Rationale:** Focus on core UX gaps first. Differentiators add value but don't fix current pain points.

## Implementation Notes

### Menu Bar Icon States

Based on Dropbox, Syncthing patterns:

- **Idle**: Static icon, white/black template (auto-tints)
- **Syncing**: Animated icon (2-3 frame animation, 0.5s interval) OR spinner overlay
- **Error**: Red badge or X overlay (use SF Symbol "xmark.circle.fill")
- **Paused**: Use 35% opacity or pause symbol overlay

### Progress Display Best Practices

From rsync, pv, alive-progress research:

1. **Display components**: % complete, transferred/total bytes, speed (MB/s), ETA
2. **Update frequency**: Every 1-2 seconds (not faster, causes flicker)
3. **ETA stability**: Use exponential smoothing algorithm. Wait 10-15s before showing ETA
4. **Large numbers**: Format as "1.2 GB" not "1,234,567,890 bytes"
5. **Speed calculation**: Average over 5-10s window, not instantaneous

### Native macOS Checklist

From Apple HIG, Big Sur design research:

- [ ] Use SF Pro font family (system default)
- [ ] 13pt for body text, follow type scale
- [ ] Use SF Symbols for icons (vector, scales automatically)
- [ ] Template images for menu bar (auto-adapts light/dark)
- [ ] Translucent sidebar if using sidebar layout
- [ ] Native NSProgressIndicator for progress bars
- [ ] System colors (NSColor.labelColor, .secondaryLabelColor, etc.)
- [ ] Respect reduced transparency accessibility setting
- [ ] Menu bar icon max height: 22pt (working area), circular items: 16x16pt recommended
- [ ] Full-height sidebar: interrupts toolbar, spans to window bottom

### File List Display Patterns

From WinSCP, Dropbox research:

- **Completed items**: Gray text + checkmark icon
- **In-progress**: Highlighted row + progress bar + speed
- **Waiting**: Normal text + queue icon
- **Failed**: Red text + error icon
- **Limit display**: Max 100 items visible, paginate or "show more" for larger lists
- **Auto-scroll**: Keep current transfer visible, don't jump when user is browsing

## Complexity Assessment

| Feature Category | Implementation Effort | Risk Level |
|------------------|----------------------|------------|
| Menu bar icon + menu | 2-3 days | Low - well-documented patterns |
| Icon state animations | 1-2 days | Low - simple frame switching |
| Global progress display | 2-3 days | Low - straightforward calculation |
| Per-file progress | 3-5 days | Medium - requires sync engine integration |
| Transfer speed + ETA | 2-4 days | Medium - smoothing algorithm needed |
| File list UI | 5-7 days | Medium - data structure + UI binding |
| Native macOS styling | 5-10 days | Medium-High - touches all UI components |
| Full-height sidebar | 3-5 days | Medium - layout restructuring |
| Light/dark mode | 2-3 days | Low-Medium - mostly color updates |

**Total estimate for MVP (Phases 1-4):** 4-5 weeks for single developer

## Sources

### File Sync Progress UI
- [Dropbox Sync & Storage Dashboard](https://help.dropbox.com/sync/sync-storage-dashboard) - Sync status overview patterns
- [CLI UX Best Practices: Progress Displays](https://evilmartians.com/chronicles/cli-ux-best-practices-3-patterns-for-improving-progress-displays) - Spinner, X/Y, progress patterns
- [Alive Progress Library](https://github.com/rsalmei/alive-progress) - ETA smoothing algorithms
- [Monitoring Rsync Progress](https://thelinuxcode.com/measure-and-show-progress-of-a-rsync-copy-linux/) - Transfer rate and ETA display

### macOS Human Interface Guidelines
- [Apple HIG: Designing for macOS](https://developer.apple.com/design/human-interface-guidelines/designing-for-macos) - Official guidelines
- [Apple HIG: SF Symbols](https://developer.apple.com/design/human-interface-guidelines/sf-symbols) - Icon system documentation
- [Adopt the New Look of macOS - WWDC20](https://developer.apple.com/videos/play/wwdc2020/10104/) - Big Sur design patterns

### Menu Bar Apps
- [Designing macOS Menu Bar Extras](https://bjango.com/articles/designingmenubarextras/) - Dimensions, template images, best practices
- [Tutorial: Add Menu Bar Extra to macOS App](https://8thlight.com/insights/tutorial-add-a-menu-bar-extra-to-a-macos-app) - Implementation guide
- [Dropbox Menu Bar Icon](https://help.dropbox.com/installs/system-tray-menu-bar) - Reference implementation

### System Tray Patterns
- [SyncTrayzor](https://github.com/canton7/SyncTrayzor) - Syncthing Windows tray with progress window, alerts
- [Syncthing Tray](https://github.com/Martchus/syncthingtray) - Linux/Windows tray showing detailed status
- [WinSCP Background Operations Queue](https://winscp.net/eng/docs/ui_queue) - File list patterns for transfers

### macOS Big Sur Design
- [App Design on Big Sur](https://www.git-tower.com/blog/app-design-on-big-sur) - Full-height sidebars, SF Symbols, translucency
- [The Design of macOS Big Sur](https://medium.com/futureproofd/the-design-of-macos-big-sur-fe9db098b651) - Visual design overview
- [macOS Big Sur: MacStories Review](https://www.macstories.net/stories/macos-big-sur-the-macstories-review/7/) - Sidebar patterns

### File Sync Market Research
- [Top File Sync Tools 2025](https://www.cotocus.com/blog/top-10-file-sync-tools-in-2025-features-pros-cons-comparison/) - Differentiator features
- [Best File Sync Software](https://www.softwarepursuits.com/blog/best-file-sync-software) - Market landscape
- [Data Synchronization Patterns](https://hasanenko.medium.com/data-synchronization-patterns-c222bd749f99) - Technical patterns

### UI/UX Best Practices
- [Mobile Design Anti-Patterns](https://www.sitepoint.com/examples-mobile-design-anti-patterns/) - Common mistakes
- [10 Most Common UI Design Mistakes](https://www.mindinventory.com/blog/ui-design-mistakes/) - Visual hierarchy, clutter, complexity
- [Progress Window: WinSCP](https://winscp.net/eng/docs/ui_progress) - File transfer progress patterns
