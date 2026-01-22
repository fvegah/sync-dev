# Phase 1: Keychain Security - Research

**Researched:** 2026-01-22
**Domain:** macOS Keychain integration via go-keyring
**Confidence:** HIGH

## Summary

This research covers migrating SharedSecret storage from plaintext config.json to macOS Keychain using the zalando/go-keyring library. The library provides a simple, CGo-free interface that delegates to `/usr/bin/security` on macOS, making it ideal for this use case.

Key findings:
- go-keyring v0.2.6 is stable, well-maintained, and used by 463+ packages
- The macOS implementation uses `/usr/bin/security` which avoids CGo and works on all macOS versions
- When the `security` command creates a keychain item, it automatically grants itself access without prompts
- Testing is well-supported via `keyring.MockInit()` for in-memory mocking
- Migration is straightforward: read from config, store in keychain, remove from config

**Primary recommendation:** Use go-keyring with service name `com.syncdev.secrets` and peer ID as the account name. Items created by the security binary are automatically trusted, so no code signing is required for keychain access during development.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| zalando/go-keyring | v0.2.6 | Cross-platform keyring interface | No CGo, uses /usr/bin/security on macOS, MIT license, 463+ dependents |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| errors (stdlib) | - | Error wrapping | Wrap keyring errors with context |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| go-keyring | keybase/go-keychain | Requires CGo, more control but harder distribution |
| go-keyring | 99designs/keyring | More backends but more complex API |

**Installation:**
```bash
go get github.com/zalando/go-keyring@v0.2.6
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── secrets/
│   └── keychain.go      # Keychain wrapper with migration logic
├── config/
│   ├── config.go        # Existing - Config struct (unchanged)
│   └── store.go         # Existing - Add migration call on load
└── models/
    └── peer.go          # Existing - SharedSecret stays in struct for runtime
```

### Pattern 1: Secrets Manager Interface
**What:** Abstract keychain operations behind an interface for testability
**When to use:** Always - enables mocking in tests
**Example:**
```go
// Source: go-keyring API pattern
package secrets

import "github.com/zalando/go-keyring"

const (
    ServiceName = "com.syncdev.secrets"
)

type Manager interface {
    GetSecret(peerID string) (string, error)
    SetSecret(peerID, secret string) error
    DeleteSecret(peerID string) error
}

type KeychainManager struct{}

func (k *KeychainManager) GetSecret(peerID string) (string, error) {
    secret, err := keyring.Get(ServiceName, peerID)
    if err == keyring.ErrNotFound {
        return "", ErrSecretNotFound
    }
    return secret, err
}

func (k *KeychainManager) SetSecret(peerID, secret string) error {
    return keyring.Set(ServiceName, peerID, secret)
}

func (k *KeychainManager) DeleteSecret(peerID string) error {
    err := keyring.Delete(ServiceName, peerID)
    if err == keyring.ErrNotFound {
        return nil // Already deleted, not an error
    }
    return err
}
```

### Pattern 2: Migration on First Load
**What:** Migrate plaintext secrets to keychain during config load
**When to use:** When loading config with existing plaintext secrets
**Example:**
```go
// Source: Common migration pattern
func (s *Store) migrateSecretsToKeychain(mgr secrets.Manager) error {
    migrated := false
    for _, peer := range s.config.Peers {
        if peer.SharedSecret != "" && peer.Paired {
            // Store in keychain
            if err := mgr.SetSecret(peer.ID, peer.SharedSecret); err != nil {
                return fmt.Errorf("failed to migrate secret for peer %s: %w", peer.ID, err)
            }
            // Clear from config struct (will be saved without secret)
            peer.SharedSecret = ""
            migrated = true
        }
    }
    if migrated {
        return s.saveUnlocked() // Save config without secrets
    }
    return nil
}
```

### Pattern 3: Lazy Secret Loading
**What:** Load secrets from keychain only when needed for network operations
**When to use:** When establishing connections to peers
**Example:**
```go
// Source: Application pattern
func (h *Handler) getSharedSecretForPeer(peerID string) (string, error) {
    // First check if already loaded in connection
    if conn := h.server.GetConnection(peerID); conn != nil && conn.SharedSecret != "" {
        return conn.SharedSecret, nil
    }
    // Load from keychain
    return h.secrets.GetSecret(peerID)
}
```

### Anti-Patterns to Avoid
- **Storing secrets in memory longer than needed:** Load from keychain when establishing connection, don't cache in Config struct
- **Ignoring migration errors:** If migration fails, log and continue but don't silently lose secrets
- **Using generic service names:** Use reverse-domain naming (`com.syncdev.secrets`) to avoid conflicts

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Keychain access | Direct Security.framework calls | go-keyring | Cross-platform, no CGo, handles encoding |
| Base64 encoding of secrets | Custom encoding | go-keyring handles it | Library already base64-encodes with prefix to handle multi-line/special chars |
| Keychain error handling | Custom error types | keyring.ErrNotFound, ErrSetDataTooBig | Standard errors, well-tested |
| Mock keychain for tests | Custom mock | keyring.MockInit() | Built-in, thread-safe, handles all operations |

**Key insight:** go-keyring handles edge cases like base64 encoding (to prevent macOS from hex-encoding certain content) and size limits (~3000 bytes for service+user+password on macOS).

## Common Pitfalls

### Pitfall 1: Assuming Prompts Will Appear
**What goes wrong:** Developers expect keychain prompts like other apps but they don't appear
**Why it happens:** When `/usr/bin/security` creates an item, it automatically adds itself to the item's ACL, granting prompt-free access
**How to avoid:** Understand that go-keyring uses the security binary which is self-trusting for items it creates
**Warning signs:** Testing for prompt handling that never triggers

### Pitfall 2: Data Size Limits
**What goes wrong:** `ErrSetDataTooBig` when storing large secrets
**Why it happens:** macOS limits service+username+password to ~3000 bytes
**How to avoid:** SharedSecrets are base64-encoded 32-byte values (44 chars), well within limits. But validate if adding metadata.
**Warning signs:** Storing JSON blobs or additional data with secrets

### Pitfall 3: Forgetting Migration Cleanup
**What goes wrong:** Secrets remain in plaintext config.json after migration
**Why it happens:** Migration stores to keychain but forgets to clear and save config
**How to avoid:** Always clear `peer.SharedSecret = ""` and call `Save()` after successful keychain storage
**Warning signs:** Secrets appearing in both locations

### Pitfall 4: ErrNotFound vs Other Errors
**What goes wrong:** Treating all errors as "not found"
**Why it happens:** Not checking specific error type
**How to avoid:** Explicitly check `err == keyring.ErrNotFound` before assuming secret doesn't exist
**Warning signs:** Overwriting existing secrets on error, losing data

### Pitfall 5: Testing Without MockInit
**What goes wrong:** Tests interact with real keychain, cause prompts, or fail in CI
**Why it happens:** Forgetting to call `keyring.MockInit()` in test setup
**How to avoid:** Call `keyring.MockInit()` in `TestMain` or test setup functions
**Warning signs:** Tests pass locally but fail in CI, or tests leave keychain entries

## Code Examples

Verified patterns from official sources:

### Basic Get/Set/Delete
```go
// Source: https://pkg.go.dev/github.com/zalando/go-keyring
import "github.com/zalando/go-keyring"

service := "com.syncdev.secrets"
user := "peer-uuid-here"
password := "base64-encoded-shared-secret"

// Store
err := keyring.Set(service, user, password)
if err != nil {
    if err == keyring.ErrSetDataTooBig {
        // Handle size limit exceeded
    }
    return err
}

// Retrieve
secret, err := keyring.Get(service, user)
if err != nil {
    if err == keyring.ErrNotFound {
        // Secret doesn't exist
        return "", nil
    }
    return "", err
}

// Delete
err = keyring.Delete(service, user)
// Note: Delete also returns ErrNotFound if item doesn't exist
```

### Testing with Mock
```go
// Source: https://pkg.go.dev/github.com/zalando/go-keyring
func TestSecretStorage(t *testing.T) {
    keyring.MockInit() // Replace OS keyring with in-memory store

    err := keyring.Set("com.syncdev.secrets", "peer-123", "secret-value")
    if err != nil {
        t.Fatal(err)
    }

    secret, err := keyring.Get("com.syncdev.secrets", "peer-123")
    if err != nil {
        t.Fatal(err)
    }

    if secret != "secret-value" {
        t.Errorf("expected 'secret-value', got '%s'", secret)
    }
}
```

### Testing Error Conditions
```go
// Source: https://pkg.go.dev/github.com/zalando/go-keyring
func TestSecretNotFound(t *testing.T) {
    keyring.MockInit()

    _, err := keyring.Get("com.syncdev.secrets", "nonexistent")
    if err != keyring.ErrNotFound {
        t.Errorf("expected ErrNotFound, got %v", err)
    }
}

func TestMockWithError(t *testing.T) {
    // Force all operations to return an error
    keyring.MockInitWithError(errors.New("keychain unavailable"))

    err := keyring.Set("service", "user", "pass")
    if err == nil {
        t.Error("expected error")
    }
}
```

### Complete Migration Function
```go
// Source: Application pattern combining go-keyring docs
func MigrateSecretsToKeychain(config *Config, serviceName string) error {
    migrated := false

    for _, peer := range config.Peers {
        if peer.SharedSecret == "" || !peer.Paired {
            continue
        }

        // Check if already in keychain
        existing, err := keyring.Get(serviceName, peer.ID)
        if err == nil && existing != "" {
            // Already migrated, just clear from config
            peer.SharedSecret = ""
            migrated = true
            continue
        }

        // Store in keychain
        if err := keyring.Set(serviceName, peer.ID, peer.SharedSecret); err != nil {
            return fmt.Errorf("migrate secret for peer %s: %w", peer.ID, err)
        }

        // Clear from config
        peer.SharedSecret = ""
        migrated = true
        log.Printf("Migrated secret for peer %s to keychain", peer.Name)
    }

    return nil // Caller should save config if migrated
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Plaintext in config files | System keychain | Always recommended | Secrets not in plain files |
| CGo keychain bindings | /usr/bin/security via go-keyring | go-keyring available | Simpler build, no Xcode needed |
| Per-app keychain | Default login keychain | macOS default | Better user experience |

**Deprecated/outdated:**
- Direct CGo calls to Security.framework: Use go-keyring instead for simpler distribution
- Custom encryption of config files: System keychain is more secure and OS-managed

## Open Questions

Things that couldn't be fully resolved:

1. **Behavior if keychain is locked**
   - What we know: Login keychain unlocks on user login, stays unlocked during session
   - What's unclear: Behavior if user manually locks keychain mid-session
   - Recommendation: Catch errors on Get/Set, log warning, retry on next sync cycle

2. **Keychain backup/restore across machines**
   - What we know: Keychain syncs via iCloud Keychain if enabled
   - What's unclear: Whether third-party app items sync (likely not for generic passwords)
   - Recommendation: Document that secrets are device-specific, re-pairing needed on new device

3. **Cleanup on peer removal**
   - What we know: keyring.Delete() removes individual entries
   - What's unclear: Whether to clean up immediately or lazily
   - Recommendation: Delete from keychain when peer is explicitly removed by user

## Sources

### Primary (HIGH confidence)
- [zalando/go-keyring GitHub](https://github.com/zalando/go-keyring) - README, implementation details
- [go-keyring pkg.go.dev](https://pkg.go.dev/github.com/zalando/go-keyring) - Full API documentation, error types, examples
- [Scripting OS X - Keychain](https://scriptingosx.com/2021/04/get-password-from-keychain-in-shell-scripts/) - Security command behavior, ACL details

### Secondary (MEDIUM confidence)
- [Apple Developer Forums](https://developer.apple.com/forums/thread/116579) - Keychain access control behavior
- [Apple Support - Keychain Access](https://support.apple.com/guide/keychain-access/if-youre-asked-for-access-to-your-keychain-kyca1243/mac) - User-facing keychain behavior
- [GitHub Gist - macOS Keychain CLI](https://gist.github.com/tamakiii/9c3eadc493597ed819b9ff96cbcf61d4) - Command-line keychain usage

### Tertiary (LOW confidence)
- General web search results about macOS code signing - Not directly verified with Apple docs

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - go-keyring well-documented, 463+ dependents, recent release
- Architecture: HIGH - Common patterns verified against library documentation
- Pitfalls: MEDIUM - Based on documentation and general keychain knowledge, not production experience with this specific lib
- Migration: HIGH - Standard pattern, well understood

**Research date:** 2026-01-22
**Valid until:** 2026-04-22 (go-keyring is stable, unlikely to change significantly)
