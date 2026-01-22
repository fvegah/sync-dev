# Testing Patterns

**Analysis Date:** 2026-01-22

## Test Framework

**Status:** Not Implemented

**Current State:**
- No test files (`*_test.go`, `*.test.ts`, `*.spec.ts`) present in codebase
- No test framework configured (go test is available but not used)
- No JavaScript test runner (Jest, Vitest) configured
- No CI/CD pipeline for test execution

**Why Testing is Absent:**
This is a Wails-based desktop application (Go backend + Svelte frontend) in early stages. Testing infrastructure has not yet been established.

## Test Configuration Files

**Go Testing:**
- `go.mod` specifies Go 1.23 - standard `go test` command is available
- No configuration files like `go.test.yml` present

**JavaScript Testing:**
- `package.json` contains no test dependencies (Jest, Vitest, etc.)
- `jsconfig.json` configured for development but not testing
- `vite.config.js` is build-only configuration

**Run Commands (Proposed):**
```bash
# Go unit tests (no tests currently)
go test ./...

# JavaScript tests (no tests currently)
npm test

# Code coverage (Go)
go test ./... -cover
```

## Recommended Test Structure

**Go Test Organization:**

Based on codebase layout, tests should be co-located with source:
- `internal/config/config_test.go` - for config logic
- `internal/sync/engine_test.go` - for sync engine
- `internal/network/server_test.go` - for network layer
- `internal/network/client_test.go` - for client connections
- `internal/sync/scanner_test.go` - for file scanning

**JavaScript Test Organization:**

Store tests in `frontend/src/` alongside components:
- `frontend/src/stores/app.test.js` - store mutations
- `frontend/src/lib/PeerList.test.js` - component rendering
- `frontend/src/lib/FolderPairs.test.js` - folder operations

## Proposed Test Structure (Go)

Given the codebase patterns, tests should follow this structure:

```go
package config

import (
    "testing"
)

func TestNewStore(t *testing.T) {
    // Test store initialization
}

func TestConfigLoad(t *testing.T) {
    // Test loading persisted config
}

func TestConfigSave(t *testing.T) {
    // Test saving changes to disk
}
```

**Pattern Observations:**
- Use Go's standard `testing` package (no external framework)
- Table-driven tests would be appropriate for sync logic
- Use `t.Run()` for sub-tests

## Proposed Test Structure (JavaScript)

Given the Svelte + Wails integration, tests should be:

```javascript
// frontend/src/stores/app.test.js
import { describe, it, expect, beforeEach } from 'vitest';
import { peers, currentTab } from './app';

describe('App Stores', () => {
    beforeEach(() => {
        // Reset stores before each test
    });

    it('should initialize peers as empty array', () => {
        // Test store initialization
    });

    it('should update peers when set', () => {
        // Test reactive updates
    });
});
```

## Mocking Strategy

**Go Mocking:**

No mocking framework currently in use. For testing without mocking:
- Network tests: use `net.Listener` with test connections
- File system: use temporary directories created with `os.TempDir()`
- Config: use in-memory Config struct without persistence

Example pattern (proposed):
```go
func TestSyncEngine(t *testing.T) {
    // Create temporary config directory
    tmpDir := t.TempDir()

    // Initialize with temp config
    cfg, _ := config.NewStoreWithPath(tmpDir)

    // Create engine
    engine, _ := sync.NewEngine(cfg)

    // Test without real network/files
}
```

**JavaScript Mocking (Proposed):**

Use Vitest's mocking for Wails bindings:

```javascript
import { describe, it, expect, vi } from 'vitest';
import * as App from '../../wailsjs/go/main/App';

// Mock the Wails Go binding
vi.mock('../../wailsjs/go/main/App', () => ({
    GetPeers: vi.fn(async () => [
        { id: '1', name: 'Device1', paired: true }
    ])
}));
```

**What to Mock:**
- Wails Go bindings (App.GetPeers, App.AddFolderPair, etc.)
- EventsOn/EventsOff runtime events
- File system operations in config tests

**What NOT to Mock:**
- Core business logic (sync engine, file scanning)
- Store state mutations
- Simple utility functions (format bytes, parse time)

## Fixtures and Test Data

**Go Test Data:**

Create test fixtures in separate files:

```go
// internal/config/testdata.go
func NewTestConfig() *Config {
    return &Config{
        DeviceID: "test-device-123",
        DeviceName: "Test Device",
        Port: 52525,
        // ... other fields
    }
}

func NewTestPeer() *models.Peer {
    return &models.Peer{
        ID: "peer-123",
        Name: "Test Peer",
        Status: models.PeerStatusOnline,
    }
}
```

**JavaScript Test Data:**

Create fixture factories:

```javascript
// frontend/src/lib/__fixtures__/data.js
export const mockPeer = {
    id: 'peer-1',
    name: 'Test Device',
    paired: true,
    status: 'online'
};

export const mockFolderPair = {
    id: 'pair-1',
    peerId: 'peer-1',
    localPath: '/Users/test/Documents',
    remotePath: 'Documents',
    enabled: true
};
```

**Location:**
- Go: Place in same package with `_test.go` suffix or dedicated testdata files
- JavaScript: Create `__fixtures__/` directory in test hierarchy

## Coverage

**Current Status:** No coverage tracking

**Recommended Target:**
- Core business logic (sync engine, file scanning): 80%+
- Configuration/storage: 90%+
- Network protocol: 70%+ (some platform-specific behavior hard to test)
- Frontend components: 60%+ (focus on event handling, store interaction)

**View Coverage (Proposed):**

```bash
# Go coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out

# JavaScript coverage (once Vitest is set up)
npm test -- --coverage
```

## Unit Test Examples (Proposed)

**Go Example - Config Store:**

```go
func TestStoreUpdate(t *testing.T) {
    tmpDir := t.TempDir()
    store, err := config.NewStoreWithPath(tmpDir)
    if err != nil {
        t.Fatalf("Failed to create store: %v", err)
    }

    // Update device name
    err = store.Update(func(c *config.Config) {
        c.DeviceName = "New Device Name"
    })
    if err != nil {
        t.Fatalf("Update failed: %v", err)
    }

    // Verify change persisted
    cfg := store.Get()
    if cfg.DeviceName != "New Device Name" {
        t.Errorf("Expected 'New Device Name', got '%s'", cfg.DeviceName)
    }
}
```

**JavaScript Example - Store Mutation:**

```javascript
import { describe, it, expect } from 'vitest';
import { peers } from '../stores/app';
import { get } from 'svelte/store';

describe('Peers Store', () => {
    it('should update peer list', () => {
        const testPeers = [
            { id: '1', name: 'Device1' },
            { id: '2', name: 'Device2' }
        ];

        peers.set(testPeers);

        expect(get(peers)).toEqual(testPeers);
    });
});
```

## Async Testing

**Go Pattern (Proposed):**

```go
func TestAsyncSync(t *testing.T) {
    engine := createTestEngine(t)

    // Channel to wait for async completion
    done := make(chan bool, 1)

    engine.SetStatusCallback(func(status sync.SyncStatus, action string) {
        if status == sync.StatusIdle {
            done <- true
        }
    })

    // Trigger sync
    go engine.SyncAllPairs()

    // Wait for completion (with timeout)
    select {
    case <-done:
        // Success
    case <-time.After(5 * time.Second):
        t.Fatal("Timeout waiting for sync to complete")
    }
}
```

**JavaScript Pattern (Proposed - Vitest):**

```javascript
import { describe, it, expect, vi } from 'vitest';

describe('Async Operations', () => {
    it('should load peers asynchronously', async () => {
        const result = await GetPeers();
        expect(Array.isArray(result)).toBe(true);
    });

    it('should handle peer loading errors', async () => {
        vi.mocked(GetPeers).mockRejectedValueOnce(
            new Error('Network error')
        );

        await expect(GetPeers()).rejects.toThrow('Network error');
    });
});
```

## Error Testing

**Go Pattern (Proposed):**

```go
func TestErrorHandling(t *testing.T) {
    t.Run("missing directory", func(t *testing.T) {
        scanner := sync.NewScanner([]string{})
        _, err := scanner.ScanDirectory("/nonexistent/path")

        if err == nil {
            t.Error("Expected error for nonexistent directory")
        }
    })

    t.Run("invalid pairing code", func(t *testing.T) {
        engine := createTestEngine(t)
        err := engine.RequestPairing("peer-1", "invalid")

        if err == nil {
            t.Error("Expected error for invalid pairing code")
        }
    })
}
```

**JavaScript Pattern (Proposed):**

```javascript
describe('Error Handling', () => {
    it('should show error alert on failed add', async () => {
        const alertSpy = vi.spyOn(window, 'alert');
        vi.mocked(AddFolderPair).mockRejectedValueOnce(
            new Error('Invalid path')
        );

        // Trigger operation that calls AddFolderPair
        // Component should display error

        expect(alertSpy).toHaveBeenCalledWith(
            expect.stringContaining('Failed to add folder pair')
        );
    });
});
```

## Integration Points to Test

**High Priority:**

1. **Config Persistence** (`internal/config/store.go`):
   - Save and load configuration from disk
   - Handle missing config file (first-run scenario)
   - Concurrent access safety

2. **Sync Engine State** (`internal/sync/engine.go`):
   - Start/stop lifecycle
   - Callback invocation
   - Status transitions

3. **File Scanning** (`internal/sync/scanner.go`):
   - Correctly scan directories
   - Apply exclusion patterns
   - Handle permission errors gracefully

4. **Network Communication** (`internal/network/`):
   - Send/receive messages between peers
   - HMAC validation
   - Connection lifecycle

5. **Frontend-Backend Integration**:
   - Wails event emission/subscription
   - Go function calls from JavaScript
   - State synchronization between Go and frontend stores

---

*Testing analysis: 2026-01-22*
