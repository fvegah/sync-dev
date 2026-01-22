# Codebase Concerns

**Analysis Date:** 2026-01-22

## Tech Debt

**System Tray Integration Disabled:**
- Issue: System tray functionality is completely disabled on macOS due to conflicts between `fyne.io/systray` and Wails. Code is commented out.
- Files: `main.go` (lines 16, 35-45), `systray.go`
- Impact: Application cannot minimize to system tray or run in background. Users must use the main window interface. This reduces usability on macOS.
- Fix approach: Find a Wails-compatible system tray implementation or use Wails-native alternatives. Consider investigating `trayapp` or building custom native integration via Wails-Go bindings.

**Large Monolithic Sync Engine:**
- Issue: `internal/sync/engine.go` is 1,247 lines with multiple responsibilities: sync coordination, networking, pairing, file transfers, and state management all in one file.
- Files: `internal/sync/engine.go`
- Impact: Difficult to test, maintain, and extend. Changes to one concern affect others. High cognitive load when debugging.
- Fix approach: Split into smaller focused components: `PairingManager`, `SyncCoordinator`, `FileTransferHandler` in separate files with clear interfaces.

**Incomplete File Transfer Handling:**
- Issue: `internal/sync/transfer.go` encodes file chunks as base64 during transfer, doubling payload size. No streaming optimization or resume capability.
- Files: `internal/sync/transfer.go` (lines 66, 146)
- Impact: Network bandwidth usage is 33% higher than necessary. Large files take longer to transfer. No way to resume interrupted transfers.
- Fix approach: Use binary protocol for chunks instead of base64 encoding. Add offset tracking and resume capability to `FileReceiver`.

**Weak Pairing Code Implementation:**
- Issue: Pairing code in `internal/network/server.go` (line 287) generates a 6-digit code using only 3 bytes of randomness, limiting entropy. Code is cleared immediately after use but stored in memory.
- Files: `internal/network/server.go` (lines 283-289), `internal/sync/engine.go` (lines 1129-1136, 1145-1150)
- Impact: Only 1 million possible codes (000000-999999). Code can be brute-forced. No rate limiting on pairing attempts.
- Fix approach: Use 4+ bytes for codes (minimum 16.7M combinations). Add rate limiting to pairing handler. Implement code expiration (5-minute timeout).

## Known Bugs

**Race Condition in Scheduler Restart:**
- Symptoms: Scheduler may stop or miss sync intervals when auto-sync is toggled rapidly
- Files: `internal/sync/engine.go` (lines 281-306)
- Trigger: Repeatedly calling `UpdateAutoSync()` or `RestartScheduler()` within 100ms intervals
- Workaround: Wait a full sync interval before toggling auto-sync again
- Root cause: 100ms sleep (line 294) is arbitrary and not guaranteed to fully stop the goroutine; the stop channel may be recreated before the old one finishes.

**Folder Pair Sync Ordering Issue:**
- Symptoms: When a peer creates a folder pair and sends it to you, if you create a folder pair back simultaneously, conflicts can occur with ID collisions.
- Files: `internal/sync/engine.go` (lines 968-1010), `app.go` (lines 237-243)
- Trigger: Both peers create folder pairs with same ID or initiate sync simultaneously
- Root cause: No coordination mechanism for folder pair creation across peers; IDs are generated independently per device.

**Unvalidated Remote Path in Mirrored Folder Pairs:**
- Symptoms: A malicious peer can send a folder pair sync with any path, creating a folder pair that syncs to an arbitrary local directory
- Files: `internal/sync/engine.go` (lines 979-999)
- Trigger: When `handleFolderPairSync` receives a message from a paired peer
- Root cause: `payload.RemotePath` is used directly without validation; only paired peers can trigger this but could be exploited if pairing is compromised

## Security Considerations

**Shared Secret Exposure:**
- Risk: Shared secrets are stored in plaintext in `~/.syncdev/config.json`. If machine is compromised, all peer secrets are exposed.
- Files: `internal/config/store.go`, `internal/models/peer.go`
- Current mitigation: File is only readable by user (0644 on file, but parent directory is 0755)
- Recommendations: (1) Encrypt secrets at rest using OS keychain (keychain on macOS, credential manager on Windows). (2) Add filesystem permission warnings on startup if config.json has wrong permissions. (3) Never log shared secrets.

**HMAC Verification Optional:**
- Risk: HMAC signature is only verified if both `pc.SharedSecret` is not empty AND `msg.HMAC` is not empty. Unpaired peers could send forged messages that pass verification checks.
- Files: `internal/network/server.go` (lines 220-224)
- Current mitigation: Unpaired peers won't have shared secrets, so verification is skipped. But there's no check that *paired* messages always have HMAC.
- Recommendations: (1) Enforce HMAC for all paired connections. (2) Log warnings when HMAC is missing on paired connections. (3) Consider denying message processing without HMAC for paired peers.

**No Authentication Before Pairing Request:**
- Risk: Any peer on the network can initiate pairing and receive the active pairing code in logs/error messages
- Files: `internal/sync/engine.go` (lines 560-632)
- Current mitigation: Pairing code must match and is only valid for one device at a time
- Recommendations: (1) Rate-limit pairing attempts per source IP. (2) Add user confirmation dialog for pairing requests (not just code validation). (3) Log all pairing attempts, even failed ones. (4) Add cooldown period after 3 failed attempts.

**File Deletion Not Reversible:**
- Risk: When a peer requests file deletion via `MsgTypeDeleteFile`, it's executed immediately without backup or recovery option.
- Files: `internal/sync/engine.go` (lines 935-966)
- Current mitigation: None. Deletion is permanent.
- Recommendations: (1) Add soft-delete with trash/backup directory before actual deletion. (2) Add confirmation dialog for delete operations. (3) Maintain deletion audit log. (4) Add undo window (e.g., 30 seconds).

**No Encryption in Transit:**
- Risk: All file data and metadata is transmitted unencrypted over the network. Packet sniffing can reveal file contents and structure.
- Files: `internal/network/server.go`, `internal/network/client.go`, `internal/sync/transfer.go`
- Current mitigation: HMAC provides authenticity but not confidentiality. Shared secret is known to both peers so doesn't add encryption.
- Recommendations: (1) Use TLS for all connections after initial pairing. (2) Implement per-message encryption with shared secret. (3) Use authenticated encryption (AES-GCM, not manual HMAC).

## Performance Bottlenecks

**Hash Calculation on Every Scan:**
- Problem: Every file scan recalculates SHA256 hash for all files, even unchanged ones. Large folders (1000+ files) take seconds.
- Files: `internal/sync/scanner.go` (lines 80-87, 129-142)
- Cause: No caching of hashes between scans. Each `ScanDirectory()` call hashes everything again.
- Improvement path: (1) Cache hashes from previous scan index. (2) Only recalculate if `ModTime` changed. (3) Implement `QuickScan()` mode without hashing for initial discovery. (4) Use incremental hashing with inode tracking.

**Full Index Comparison on Every Sync:**
- Problem: `CompareIndices()` creates a map of all files every sync cycle, even if folder hasn't changed.
- Files: `internal/sync/index.go` (lines 100-134)
- Cause: No delta tracking. Previous sync state is loaded but full comparison is always done.
- Improvement path: (1) Track last sync hash. (2) Only scan folders where mtime > lastSyncTime. (3) Build a bloom filter of file hashes for quick change detection.

**Blocking File Transfer Progress:**
- Problem: `SendFile()` calls progress callback synchronously, blocking on UI updates during large file transfers.
- Files: `internal/sync/transfer.go` (lines 82-96)
- Cause: Progress callback is invoked for every chunk, and if UI is slow, blocks the transfer loop.
- Improvement path: (1) Use non-blocking channel to send progress updates. (2) Add rate limiting (update UI max 10x per second). (3) Defer progress updates to separate goroutine.

**Connection Pooling Missing:**
- Problem: New TCP connections are created per sync request. Connections are reused but cleanup on error is inconsistent.
- Files: `internal/sync/engine.go` (lines 432-457), `internal/network/client.go`
- Cause: Connections map is not cleaned up on peer offline/error. Stale connections accumulate.
- Improvement path: (1) Implement connection pooling with per-peer max connections. (2) Add connection health checks (ping/pong). (3) Auto-close idle connections after 5 minutes.

## Fragile Areas

**Folder Pair State Inconsistency:**
- Files: `app.go` (lines 202-247), `internal/sync/engine.go` (lines 968-1010)
- Why fragile: Config and UI state can diverge. If folder pair sync fails after local addition, peer won't have it. If peer sends a folder pair but connection drops, local state may be partially created.
- Safe modification: (1) Add transactional config updates. (2) Implement folder pair sync acknowledgment. (3) Add version numbers to folder pairs to detect divergence.
- Test coverage: No tests for concurrent folder pair creation and sync scenarios.

**Discovery Peer Status Tracking:**
- Files: `internal/network/discovery.go`, `internal/sync/engine.go` (lines 196-213)
- Why fragile: Peer status from discovery and config can diverge. mDNS can report peer as online but it might be unreachable.
- Safe modification: (1) Verify peer is truly reachable before attempting sync. (2) Add peer health checking. (3) Implement exponential backoff for unreachable peers.
- Test coverage: No tests for offline peer handling or discovery races.

**Index File Corruption Risk:**
- Files: `internal/sync/index.go` (lines 32-59, 61-80)
- Why fragile: Index files are JSON on disk. If app crashes during `SaveIndex()`, file can be partially written and unreadable on restart.
- Safe modification: (1) Use write-then-rename pattern for atomic writes. (2) Add JSON validation on load. (3) Implement index backup/recovery.
- Test coverage: No tests for corrupted index recovery.

**Message Parsing Without Schema Validation:**
- Files: `internal/network/protocol.go`, `internal/sync/engine.go` (lines 560-575 and similar)
- Why fragile: Payloads are unmarshaled with `json.Unmarshal()` but no schema validation. Unknown fields are silently ignored. Version mismatches cause silent failures.
- Safe modification: (1) Add protocol version to all messages. (2) Validate required fields after unmarshal. (3) Log rejected messages with details.
- Test coverage: No tests for malformed or unexpected message handling.

## Scaling Limits

**Single-Threaded Sync Engine Loop:**
- Current capacity: Can sync ~10 folder pairs per minute before blocking
- Limit: If any single sync hangs, all other syncs are blocked
- Files: `internal/sync/engine.go` (lines 332-357, 360-430)
- Scaling path: (1) Use goroutine pool for parallel folder pair syncs. (2) Add timeout for sync operations. (3) Implement backpressure for slow peers.

**In-Memory Event History:**
- Current capacity: Fixed 100 events (line 231-233)
- Limit: With high-frequency syncs, recent events are immediately discarded
- Files: `internal/sync/engine.go` (lines 227-239)
- Scaling path: (1) Implement rotating file-based event log. (2) Add event persistence with retention policy. (3) Expose query API for event history.

**All Connections Stored in Single Map:**
- Current capacity: Map grows unbounded per peer session
- Limit: Memory leak if connections aren't properly cleaned up
- Files: `internal/sync/engine.go` (lines 50, 433-456)
- Scaling path: (1) Implement connection cleanup on error. (2) Add per-peer connection limits. (3) Monitor and log connection map size.

**Index Files Grow Indefinitely:**
- Current capacity: Index stores all files ever synced
- Limit: Large folders (100K+ files) create multi-MB JSON files
- Files: `internal/sync/index.go`
- Scaling path: (1) Compress index files. (2) Implement incremental index snapshots. (3) Archive old indices.

## Dependencies at Risk

**github.com/hashicorp/mdns (mDNS Discovery):**
- Risk: No recent activity on repo. May not support future Go versions.
- Impact: Network peer discovery breaks if library is abandoned during Go version migration.
- Migration plan: Evaluate `bonjour-go` or `avahi` bindings; consider native mDNS implementation if critical.

**github.com/gobwas/glob (Path Pattern Matching):**
- Risk: Not frequently updated; potential performance issues with complex patterns.
- Impact: Exclusion patterns could be slow with large folder scans.
- Migration plan: Migrate to `filepath.Match()` or `github.com/bmatcuk/doublestar` if needed.

## Missing Critical Features

**No Conflict Resolution Strategy:**
- Problem: When both peers modify the same file simultaneously, last-write-wins is applied silently without user notification.
- Blocks: (1) User cannot recover older versions. (2) Collaborative editing scenarios don't work. (3) Data loss is possible without warning.
- Priority: **High** - Users expect conflict notifications.

**No Sync Scheduling Pause/Resume:**
- Problem: Users cannot temporarily pause sync during important work without disabling it entirely.
- Blocks: (1) Batch operations are disrupted by sync. (2) Network bandwidth cannot be temporarily reserved.
- Priority: **Medium** - Quality of life feature.

**No Selective Sync (Branch Sync):**
- Problem: All enabled folder pairs sync every interval. Cannot sync only one pair or exclude subdirectories.
- Blocks: (1) Large folder pairs cannot be partially synced. (2) Testing individual pairs is difficult.
- Priority: **Medium** - Needed for large deployments.

**No Bandwidth Throttling:**
- Problem: Large file transfers use unlimited bandwidth, affecting other network users.
- Blocks: (1) Home networks become unusable during sync. (2) Mobile hotspots can be exhausted.
- Priority: **Medium** - Important for consumer use.

**No Sync Version/History:**
- Problem: No way to see when files were synced or which version peers have.
- Blocks: (1) Debugging sync issues is difficult. (2) Users don't know if sync succeeded.
- Priority: **Low** - Observability feature.

## Test Coverage Gaps

**Untested Pairing Flow:**
- What's not tested: (1) Concurrent pairing requests. (2) Pairing code expiration. (3) Code mismatch scenarios. (4) Pairing rejection and recovery.
- Files: `internal/sync/engine.go` (lines 559-632, 1045-1089)
- Risk: Pairing bugs could lock users out or leak into production.
- Priority: **High**

**Untested Network Failures:**
- What's not tested: (1) Connection drops mid-transfer. (2) Peer goes offline during sync. (3) Network timeout handling. (4) Reconnection and resume logic.
- Files: `internal/network/server.go`, `internal/sync/engine.go` (lines 459-481)
- Risk: Sync gets stuck or fails silently in real-world network conditions.
- Priority: **High**

**Untested File Transfer Edge Cases:**
- What's not tested: (1) Symlinks. (2) Special characters in filenames. (3) Very large files (>1GB). (4) Permission preservation. (5) Sparse files.
- Files: `internal/sync/transfer.go`, `internal/sync/scanner.go`
- Risk: File corruption or silent failures on certain file types.
- Priority: **High**

**Untested Concurrent Sync Scenarios:**
- What's not tested: (1) Multiple folder pairs syncing simultaneously. (2) Same file being modified on both peers. (3) Rapid enable/disable of folder pairs.
- Files: `internal/sync/engine.go` (lines 332-357)
- Risk: Race conditions and data loss in production under load.
- Priority: **High**

**Untested Configuration Persistence:**
- What's not tested: (1) Config corruption recovery. (2) Migration from old to new config format. (3) Permission issues on config directory. (4) Disk full scenarios.
- Files: `internal/config/store.go`
- Risk: App fails to start or loses configuration on edge cases.
- Priority: **Medium**

**Untested Frontend/Backend Integration:**
- What's not tested: (1) Event delivery reliability. (2) Long-running operations. (3) Progress update latency. (4) Error message propagation.
- Files: `app.go` (callbacks), `frontend/src/lib/`
- Risk: UI becomes out of sync with backend state.
- Priority: **Medium**

---

*Concerns audit: 2026-01-22*
