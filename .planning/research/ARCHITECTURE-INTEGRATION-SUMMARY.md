# Research Summary: SyncDev System Tray, Keychain, and Progress Enhancements

**Domain:** Desktop app enhancement (P2P file sync)
**Researched:** 2026-01-22
**Overall confidence:** MEDIUM-HIGH

## Executive Summary

This research addresses architectural integration of three enhancement capabilities into SyncDev's existing Wails-based architecture: system tray integration, macOS Keychain access, and enhanced progress tracking. The current three-tier event-driven architecture (UI → App → SyncEngine) provides a solid foundation for these enhancements.

**System tray integration** faces a critical architectural decision: Wails v2 has no native support and requires a workaround (separate process with IPC), while Wails v3 alpha provides native support but requires migration. The timing of Wails v3's stable release (currently "nearly ready" but no fixed date) determines the optimal implementation path.

**Keychain integration** has a clear winner: zalando/go-keyring provides CGo-free, cross-platform credential storage using macOS's native /usr/bin/security command. This enables secure storage of pairing codes and peer secrets currently stored in plaintext JSON.

**Progress tracking** can be enhanced through hierarchical aggregation building on the existing callback architecture. A ProgressAggregator component collects per-file updates from transfer workers and emits throttled aggregate updates to the UI, preventing event storms while providing granular visibility.

All three enhancements integrate naturally into SyncDev's existing event-driven architecture with minimal refactoring. The main architectural risk is system tray implementation complexity, mitigated by following proven IPC patterns or waiting for Wails v3 stable.

## Key Findings

**Stack:** zalando/go-keyring (no CGo), getlantern/systray with IPC (Wails v2) or native Wails v3 tray, existing Svelte/Wails architecture preserved

**Architecture:** Hierarchical progress aggregation with throttled emission (10 Hz), event-driven callbacks to UI and system tray, OS Keychain for credential storage replacing config.json secrets

**Critical pitfall:** System tray libraries conflict with Wails v2 event loop on main thread, requiring separate process IPC or Wails v3 migration

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Enhanced Progress Tracking (2-3 days)
**Rationale:** Lowest risk, highest immediate UX value, builds on existing architecture
- Addresses: Per-file progress visibility, aggregate metrics (speed, ETA), smoother UI updates
- Avoids: External dependencies, migration risks, new process management
- Implementation: Create ProgressAggregator in internal/sync/, update callbacks, enhance Svelte components
- Dependencies: None (self-contained enhancement)
- Risk: Low - isolated change within existing callback pattern

### Phase 2: Keychain Integration (2-3 days)
**Rationale:** Isolated to credential management, clear implementation path, no UI changes
- Addresses: Secure credential storage, removes plaintext secrets from config.json
- Avoids: CGo complexity (using zalando/go-keyring instead of keybase/go-keychain)
- Implementation: Add go-keyring dependency, create KeychainService wrapper, migrate config secrets
- Dependencies: Phase 1 complete for testing sync with Keychain credentials
- Risk: Medium - requires migration logic and manual testing, but clear rollback path

### Phase 3A: System Tray with IPC (4-5 days) — IF Wails v3 stable > 2 months away
**Rationale:** Proven workaround, stays on stable Wails v2, unblocks feature
- Addresses: Background presence, quick access menu, status visibility
- Avoids: Alpha software risk, migration complexity
- Implementation: Separate systray binary, Unix socket IPC, TrayManager coordinator
- Dependencies: Phase 1+2 complete for full feature set in tray
- Risk: Medium-High - process management complexity, IPC reliability, platform-specific behaviors

### Phase 3B: Wails v3 Migration + Native Tray (4-6 days) — IF Wails v3 stable < 2 months away
**Rationale:** Cleaner architecture, native support, long-term maintainability
- Addresses: System tray natively, multi-window support, improved APIs
- Avoids: IPC complexity, two-process architecture
- Implementation: Follow migration guide, update API calls, implement native tray
- Dependencies: Phase 1+2 complete, Wails v3 stable release
- Risk: Medium - migration unknowns, but official guide exists, cleaner long-term

### Phase 4: UI Polish (2-3 days, parallel with Phase 3)
**Rationale:** Iterative refinement, can proceed independently
- Addresses: Native macOS styling, virtual scrolling for large file lists, responsive layout
- Avoids: Performance issues with 1000+ file displays
- Implementation: CSS variables for system theming, component composition, virtual scrolling if needed
- Dependencies: Phase 1 complete for progress components to exist
- Risk: Low - UI-only changes, iterative improvements

## Phase Ordering Rationale

**Why progress tracking first:**
- Zero external dependencies
- Builds directly on existing callback architecture
- Immediate UX improvement users will notice
- Validates throttling and aggregation approach before system tray uses it

**Why Keychain second:**
- Isolated feature with clear scope
- Enables testing of credential flow with enhanced progress tracking
- Prepares for future features (cloud sync, API tokens)
- No UI changes required, reducing scope

**Why system tray last:**
- Highest architectural complexity and external dependencies
- Benefits from progress tracking and Keychain being complete (full feature set in tray)
- Wails v3 stable release timing affects implementation path (3A vs 3B)
- Can be delivered independently without blocking other work

**Why UI polish in parallel:**
- No dependencies on system tray implementation
- Can refine as progress components are built
- Allows iteration based on user feedback during earlier phases

## Research Flags for Phases

**Phase 1 (Progress Tracking):** Standard patterns, unlikely to need research
- Event aggregation pattern well-understood
- Throttling approach proven in notification systems
- Integration points clear from existing callbacks

**Phase 2 (Keychain):** Standard patterns, minimal research needed
- Library choice clear (zalando/go-keyring)
- API straightforward (Set/Get/Delete)
- May need research: Migration UX for existing users with paired peers

**Phase 3A (System Tray IPC):** Likely needs implementation-specific research
- IPC protocol design (Unix socket vs HTTP localhost)
- Process lifecycle management (spawn, monitor, restart)
- Platform-specific behaviors (macOS app bundle structure)
- Error handling and recovery

**Phase 3B (Wails v3 Migration):** Deeper research needed during implementation
- Breaking changes in migration guide may reveal edge cases
- Event system changes (Off/OffAll scoping differences)
- Multi-window implications for future features
- Testing across all platforms (macOS, Linux if planned)

**Phase 4 (UI Polish):** Minimal research, mostly implementation
- Virtual scrolling library choice if needed
- macOS Human Interface Guidelines for native styling
- Performance profiling if large file lists cause issues

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Progress Tracking | HIGH | Standard event aggregation pattern, clear integration points in existing Engine callbacks |
| Keychain | HIGH | zalando/go-keyring well-documented, proven approach, simple API, CGo-free |
| System Tray (v2 IPC) | MEDIUM | Community-proven workaround, but process management complexity and platform-specific behaviors |
| System Tray (v3 native) | MEDIUM | Official support documented, but Wails v3 in alpha, migration unknowns exist |
| Svelte Architecture | HIGH | Existing component structure sound, event-driven stores work well with Wails |
| Build Order | HIGH | Clear dependency chain, phases isolated, risks identified |

## Gaps to Address

### During Planning
1. **Wails v3 release timeline:** Check current status monthly, affects Phase 3A vs 3B decision
   - As of 2026-01-22: Alpha, "nearly ready", applications running in production, no fixed release date
   - Decision point: If stable release > 2 months, implement 3A; if < 2 months, wait for 3B

2. **Existing user migration:** Design UX for migrating paired peers to Keychain
   - Options: Automatic on startup, manual migration with prompt, background migration
   - Need to validate: Error handling if Keychain access denied

3. **Progress UI scalability:** Test threshold for virtual scrolling necessity
   - Hypothesis: 100-500 files manageable with simple list
   - Research if users report > 1000 file syncs common

### During Implementation
1. **IPC protocol details** (if Phase 3A): Unix socket vs HTTP localhost
   - Need: Message format, authentication, error recovery
   - Research: Platform differences (macOS vs Linux)

2. **System tray icon states:** Static vs animated during sync
   - Defer: Start with static icons, add animation in polish phase if time permits

3. **Keychain entitlements:** Code signing requirements for production build
   - Validate: App Store vs direct distribution signing differences

### Future Considerations
1. **Background sync:** Once system tray implemented, enable sync with window closed
2. **Multi-window support:** Wails v3 enables progress detail window separate from main
3. **Cloud sync credentials:** Keychain ready for OAuth tokens if cloud features added
4. **Advanced progress filtering:** Sort, search, filter file list if users request

## Critical Success Factors

**For Phase 1 (Progress Tracking):**
- [ ] Throttling prevents UI thrashing (< 20ms render time with 100 files)
- [ ] Per-file progress visible and accurate
- [ ] Aggregate metrics (speed, ETA) computed correctly
- [ ] Memory usage acceptable (< 10 MB for 1000 files)

**For Phase 2 (Keychain):**
- [ ] Secrets successfully stored and retrieved from macOS Keychain
- [ ] Migration from config.json completes without data loss
- [ ] Keychain prompt appears as expected (first access only)
- [ ] Unpairing removes secrets from Keychain

**For Phase 3A (System Tray IPC):**
- [ ] Systray process spawns reliably on app startup
- [ ] IPC communication stable (no dropped messages)
- [ ] Menu items trigger correct actions (Show, Sync, Quit)
- [ ] Process cleanup on app exit (no orphans)
- [ ] Icon/tooltip updates reflect current status

**For Phase 3B (Wails v3 Migration):**
- [ ] All existing features work after migration
- [ ] Native system tray integrates cleanly
- [ ] No performance regressions
- [ ] Build process remains simple

**For Phase 4 (UI Polish):**
- [ ] App matches macOS native look (fonts, colors, spacing)
- [ ] Smooth animations and transitions
- [ ] Responsive to window resizing
- [ ] Large file lists remain performant

## Recommendation

**Proceed with 4-phase roadmap:**

1. **Phase 1 (Progress Tracking)** — Immediate start, 2-3 days
   - Highest confidence, lowest risk, immediate value
   - Establishes patterns for Phase 3 system tray updates

2. **Phase 2 (Keychain)** — After Phase 1, 2-3 days
   - Clear path, proven library, security improvement
   - Prepares credential infrastructure for future features

3. **Phase 3A or 3B (System Tray)** — Decision based on Wails v3 timing
   - Check Wails v3 stable status at Phase 2 completion
   - If v3 stable < 2 months away: Implement Phase 3B (migration + native tray)
   - If v3 stable > 2 months away: Implement Phase 3A (IPC workaround)
   - Plan 3A→3B migration later if implementing 3A

4. **Phase 4 (UI Polish)** — Parallel with Phase 3, 2-3 days
   - Iterative refinement, can adjust scope based on feedback
   - Low risk, high user satisfaction impact

**Total estimated effort:** 10-14 days of focused development

**Key decision points:**
- After Phase 1: Validate progress aggregation performance with large file sets
- Before Phase 3: Check Wails v3 stable release status (3A vs 3B)
- During Phase 3: Monitor IPC reliability (3A) or migration issues (3B)
- During Phase 4: User feedback on native styling and performance

**Rollback plans:**
- Phase 1: Revert to single TransferProgress callback
- Phase 2: Revert to config.json secrets (keep backup during migration)
- Phase 3A: Remove IPC, disable system tray feature
- Phase 3B: Revert to Wails v2 if migration issues (reason to test alpha first)

## Next Steps

1. **Validate research findings:**
   - Create proof-of-concept for ProgressAggregator (1 hour)
   - Test zalando/go-keyring on macOS (1 hour)
   - Check current Wails v3 alpha stability (review recent releases)

2. **Create detailed phase plans:**
   - Break each phase into tasks with acceptance criteria
   - Identify integration points in existing code
   - Prepare test scenarios for each phase

3. **Set up development environment:**
   - Ensure Wails v2 build works
   - Add go-keyring dependency
   - Review existing callback implementations

4. **Monitor Wails v3:**
   - Subscribe to Wails release notifications
   - Review migration guide for updates
   - Test v3 alpha if considering 3B path

---

**Research complete.** Architectural patterns identified, implementation paths clear, risks documented, phase structure recommended. Ready for roadmap creation and task breakdown.

See `/Users/felipe/dev/fvegah/sync-dev/.planning/research/ARCHITECTURE.md` for detailed technical architecture, component boundaries, data flows, and implementation patterns.
