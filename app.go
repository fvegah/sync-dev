package main

import (
	"SyncDev/internal/config"
	"SyncDev/internal/models"
	"SyncDev/internal/sync"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx          context.Context
	configStore  *config.Store
	syncEngine   *sync.Engine
	pairingCode  string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize config store
	store, err := config.NewStore()
	if err != nil {
		log.Printf("Failed to initialize config store: %v", err)
		return
	}
	a.configStore = store

	// Initialize sync engine
	engine, err := sync.NewEngine(store)
	if err != nil {
		log.Printf("Failed to initialize sync engine: %v", err)
		return
	}
	a.syncEngine = engine

	// Set up callbacks
	engine.SetStatusCallback(func(status sync.SyncStatus, action string) {
		runtime.EventsEmit(ctx, "sync:status", map[string]interface{}{
			"status": status,
			"action": action,
		})
	})

	engine.SetProgressCallback(func(progress *models.TransferProgress) {
		runtime.EventsEmit(ctx, "sync:progress", progress)
	})

	engine.SetEventCallback(func(event *sync.SyncEvent) {
		runtime.EventsEmit(ctx, "sync:event", event)
	})

	engine.SetPeerChangeCallback(func() {
		runtime.EventsEmit(ctx, "peers:changed", nil)
	})

	// Start sync engine
	if err := engine.Start(); err != nil {
		log.Printf("Failed to start sync engine: %v", err)
	}

	log.Println("SyncDev started successfully")
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	if a.syncEngine != nil {
		a.syncEngine.Stop()
	}
}

// ============================================
// Configuration Methods
// ============================================

// GetConfig returns the current configuration
func (a *App) GetConfig() *config.Config {
	return a.configStore.Get()
}

// UpdateDeviceName updates the device name
func (a *App) UpdateDeviceName(name string) error {
	return a.configStore.Update(func(c *config.Config) {
		c.DeviceName = name
	})
}

// UpdateSyncInterval updates the sync interval
func (a *App) UpdateSyncInterval(mins int) error {
	if mins < 1 || mins > 60 {
		return fmt.Errorf("interval must be between 1 and 60 minutes")
	}
	return a.configStore.Update(func(c *config.Config) {
		c.SyncIntervalMins = mins
	})
}

// UpdateAutoSync updates the auto sync setting
func (a *App) UpdateAutoSync(enabled bool) error {
	return a.configStore.Update(func(c *config.Config) {
		c.AutoSync = enabled
	})
}

// UpdateGlobalExclusions updates the global exclusion patterns
func (a *App) UpdateGlobalExclusions(patterns []string) error {
	return a.configStore.Update(func(c *config.Config) {
		c.GlobalExclusions = patterns
	})
}

// ============================================
// Peer Methods
// ============================================

// GetPeers returns all discovered and paired peers
func (a *App) GetPeers() []*models.Peer {
	if a.syncEngine == nil {
		return []*models.Peer{}
	}
	return a.syncEngine.GetDiscoveredPeers()
}

// GeneratePairingCode generates a new pairing code
func (a *App) GeneratePairingCode() string {
	if a.syncEngine == nil {
		return ""
	}
	a.pairingCode = a.syncEngine.GeneratePairingCode()
	return a.pairingCode
}

// GetCurrentPairingCode returns the current pairing code
func (a *App) GetCurrentPairingCode() string {
	return a.pairingCode
}

// RequestPairing initiates pairing with a peer
func (a *App) RequestPairing(peerID, code string) error {
	if a.syncEngine == nil {
		return fmt.Errorf("sync engine not initialized")
	}
	return a.syncEngine.RequestPairing(peerID, code)
}

// AcceptPairing accepts a pairing request
func (a *App) AcceptPairing(peerID string) error {
	if a.syncEngine == nil {
		return fmt.Errorf("sync engine not initialized")
	}
	return a.syncEngine.AcceptPairing(peerID)
}

// RejectPairing rejects a pairing request
func (a *App) RejectPairing(peerID string) error {
	if a.syncEngine == nil {
		return fmt.Errorf("sync engine not initialized")
	}
	return a.syncEngine.RejectPairing(peerID)
}

// UnpairPeer removes pairing with a peer
func (a *App) UnpairPeer(peerID string) error {
	if a.syncEngine == nil {
		return fmt.Errorf("sync engine not initialized")
	}
	return a.syncEngine.UnpairPeer(peerID)
}

// ============================================
// Folder Pair Methods
// ============================================

// GetFolderPairs returns all folder pairs
func (a *App) GetFolderPairs() []*models.FolderPair {
	cfg := a.configStore.Get()
	return cfg.FolderPairs
}

// AddFolderPair adds a new folder pair
func (a *App) AddFolderPair(peerID, localPath, remotePath string) (*models.FolderPair, error) {
	// Validate local path exists
	info, err := os.Stat(localPath)
	if err != nil {
		return nil, fmt.Errorf("local path does not exist: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("local path is not a directory")
	}

	// Validate peer exists and is paired
	cfg := a.configStore.Get()
	peer := cfg.GetPeer(peerID)
	if peer == nil || !peer.Paired {
		return nil, fmt.Errorf("peer not found or not paired")
	}

	pair := &models.FolderPair{
		ID:         uuid.New().String(),
		PeerID:     peerID,
		LocalPath:  localPath,
		RemotePath: remotePath,
		Enabled:    true,
		Exclusions: []string{},
	}

	if err := a.configStore.Update(func(c *config.Config) {
		c.AddFolderPair(pair)
	}); err != nil {
		return nil, err
	}

	return pair, nil
}

// UpdateFolderPair updates a folder pair
func (a *App) UpdateFolderPair(id string, enabled bool, exclusions []string) error {
	return a.configStore.Update(func(c *config.Config) {
		if fp := c.GetFolderPair(id); fp != nil {
			fp.Enabled = enabled
			fp.Exclusions = exclusions
		}
	})
}

// RemoveFolderPair removes a folder pair
func (a *App) RemoveFolderPair(id string) error {
	return a.configStore.Update(func(c *config.Config) {
		c.RemoveFolderPair(id)
	})
}

// ============================================
// Sync Methods
// ============================================

// SyncNow triggers an immediate sync for all folder pairs
func (a *App) SyncNow() {
	if a.syncEngine == nil {
		return
	}
	go a.syncEngine.SyncAllPairs()
}

// SyncFolderPair syncs a specific folder pair
func (a *App) SyncFolderPair(folderPairID string) error {
	if a.syncEngine == nil {
		return fmt.Errorf("sync engine not initialized")
	}
	return a.syncEngine.SyncFolderPair(folderPairID)
}

// GetSyncStatus returns the current sync status
func (a *App) GetSyncStatus() map[string]interface{} {
	if a.syncEngine == nil {
		return map[string]interface{}{
			"status": sync.StatusIdle,
			"action": "",
		}
	}
	status, action := a.syncEngine.GetStatus()
	return map[string]interface{}{
		"status": status,
		"action": action,
	}
}

// GetSyncProgress returns the current transfer progress
func (a *App) GetSyncProgress() *models.TransferProgress {
	if a.syncEngine == nil {
		return nil
	}
	return a.syncEngine.GetProgress()
}

// GetRecentEvents returns recent sync events
func (a *App) GetRecentEvents() []*sync.SyncEvent {
	if a.syncEngine == nil {
		return []*sync.SyncEvent{}
	}
	return a.syncEngine.GetRecentEvents()
}

// ============================================
// Utility Methods
// ============================================

// SelectFolder opens a folder picker dialog
func (a *App) SelectFolder() (string, error) {
	homeDir, _ := os.UserHomeDir()
	result, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select Folder to Sync",
		DefaultDirectory: homeDir,
	})
	if err != nil {
		return "", err
	}
	return result, nil
}

// GetHomeDirectory returns the user's home directory
func (a *App) GetHomeDirectory() string {
	home, _ := os.UserHomeDir()
	return home
}

// PathExists checks if a path exists
func (a *App) PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// FormatBytes formats bytes as human-readable string
func (a *App) FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration formats a duration as human-readable string
func (a *App) FormatDuration(seconds int64) string {
	d := time.Duration(seconds) * time.Second
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

// GetAppVersion returns the application version
func (a *App) GetAppVersion() string {
	return config.AppVersion
}

// GetDataDirectory returns the app data directory
func (a *App) GetDataDirectory() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".syncdev")
}

// OpenDataDirectory opens the data directory in Finder
func (a *App) OpenDataDirectory() {
	dataDir := a.GetDataDirectory()
	runtime.BrowserOpenURL(a.ctx, "file://"+dataDir)
}
