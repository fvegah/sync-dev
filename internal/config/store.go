package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

// Store manages configuration persistence
type Store struct {
	configPath string
	config     *Config
	mu         sync.RWMutex
}

// NewStore creates a new configuration store
func NewStore() (*Store, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".syncdev")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.json")
	store := &Store{
		configPath: configPath,
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

// load reads the configuration from disk
func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.configPath)
	if os.IsNotExist(err) {
		// Create default config
		s.config = DefaultConfig()
		s.config.DeviceID = uuid.New().String()
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "Mac"
		}
		s.config.DeviceName = hostname
		return s.saveUnlocked()
	}
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	s.config = &cfg
	return nil
}

// saveUnlocked saves the configuration without locking (caller must hold lock)
func (s *Store) saveUnlocked() error {
	data, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(s.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Save persists the configuration to disk
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveUnlocked()
}

// Get returns a copy of the current configuration
func (s *Store) Get() *Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Return a copy to prevent concurrent modification
	configCopy := *s.config
	return &configCopy
}

// Update allows updating the configuration with a callback
func (s *Store) Update(fn func(*Config)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(s.config)
	return s.saveUnlocked()
}

// GetConfigPath returns the path to the config file
func (s *Store) GetConfigPath() string {
	return s.configPath
}

// GetDataDir returns the data directory path
func (s *Store) GetDataDir() string {
	return filepath.Dir(s.configPath)
}

// GetIndexPath returns the path for storing file indices
func (s *Store) GetIndexPath(folderPairID string) string {
	return filepath.Join(s.GetDataDir(), "indices", folderPairID+".json")
}

// EnsureIndexDir ensures the index directory exists
func (s *Store) EnsureIndexDir() error {
	indexDir := filepath.Join(s.GetDataDir(), "indices")
	return os.MkdirAll(indexDir, 0755)
}
