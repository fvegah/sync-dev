package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"SyncDev/internal/models"

	"github.com/zalando/go-keyring"
)

func TestMain(m *testing.M) {
	// Use mock keyring for all tests
	keyring.MockInit()
	m.Run()
}

func TestMigrateSecretsToKeychain(t *testing.T) {
	// Create temp dir for config
	tmpDir := t.TempDir()

	// Create a config file with plaintext secrets
	configPath := filepath.Join(tmpDir, "config.json")
	config := &Config{
		DeviceID:   "test-device",
		DeviceName: "Test",
		Port:       52525,
		Peers: []*models.Peer{
			{
				ID:           "peer-123",
				Name:         "Test Peer",
				SharedSecret: "secret-value",
				Paired:       true,
			},
		},
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load store (should trigger migration)
	store, err := NewStoreWithPath(configPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Verify secret is in keychain
	secret, err := store.GetSecrets().GetSecret("peer-123")
	if err != nil {
		t.Fatalf("Failed to get secret from keychain: %v", err)
	}
	if secret != "secret-value" {
		t.Errorf("Expected 'secret-value', got '%s'", secret)
	}

	// Verify secret cleared from config
	cfg := store.Get()
	if cfg.Peers[0].SharedSecret != "" {
		t.Error("SharedSecret should be empty in config after migration")
	}

	// Verify config file no longer contains secret
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	if strings.Contains(string(data), "secret-value") {
		t.Error("Config file still contains plaintext secret")
	}
}

func TestMigrateSecretsSkipsUnpairedPeers(t *testing.T) {
	// Create temp dir for config
	tmpDir := t.TempDir()

	// Create a config file with an unpaired peer that has a secret
	configPath := filepath.Join(tmpDir, "config.json")
	config := &Config{
		DeviceID:   "test-device-2",
		DeviceName: "Test 2",
		Port:       52525,
		Peers: []*models.Peer{
			{
				ID:           "unpaired-peer",
				Name:         "Unpaired Peer",
				SharedSecret: "should-not-migrate",
				Paired:       false,
			},
		},
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load store
	store, err := NewStoreWithPath(configPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Verify secret is NOT in keychain (unpaired peers should not be migrated)
	_, err = store.GetSecrets().GetSecret("unpaired-peer")
	if err == nil {
		t.Error("Expected error getting secret for unpaired peer, got nil")
	}

	// Secret should still be in config since unpaired
	cfg := store.Get()
	if cfg.Peers[0].SharedSecret != "should-not-migrate" {
		t.Errorf("Expected secret to remain for unpaired peer, got '%s'", cfg.Peers[0].SharedSecret)
	}
}

func TestMigrateSecretsHandlesEmptySecrets(t *testing.T) {
	// Create temp dir for config
	tmpDir := t.TempDir()

	// Create a config file with a paired peer but no secret
	configPath := filepath.Join(tmpDir, "config.json")
	config := &Config{
		DeviceID:   "test-device-3",
		DeviceName: "Test 3",
		Port:       52525,
		Peers: []*models.Peer{
			{
				ID:           "peer-no-secret",
				Name:         "Peer Without Secret",
				SharedSecret: "",
				Paired:       true,
			},
		},
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load store - should not fail
	_, err = NewStoreWithPath(configPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
}

func TestNewStoreWithPathLoadsExistingConfig(t *testing.T) {
	// Create temp dir for config
	tmpDir := t.TempDir()

	// Create a config file
	configPath := filepath.Join(tmpDir, "config.json")
	config := &Config{
		DeviceID:   "existing-device",
		DeviceName: "Existing Device",
		Port:       12345,
		Peers:      []*models.Peer{},
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load store
	store, err := NewStoreWithPath(configPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Verify config loaded correctly
	cfg := store.Get()
	if cfg.DeviceID != "existing-device" {
		t.Errorf("Expected DeviceID 'existing-device', got '%s'", cfg.DeviceID)
	}
	if cfg.Port != 12345 {
		t.Errorf("Expected Port 12345, got %d", cfg.Port)
	}
}
