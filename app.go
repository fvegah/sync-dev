package main

import (
	"SyncDev/internal/config"
	"SyncDev/internal/models"
	"SyncDev/internal/sync"
	"SyncDev/internal/tray"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// App struct
type App struct {
	app         *application.App
	window      *application.WebviewWindow
	tray        *tray.Manager
	configStore *config.Store
	syncEngine  *sync.Engine
	pairingCode string
	paused      bool
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(app *application.App, window *application.WebviewWindow, trayManager *tray.Manager) {
	a.app = app
	a.window = window
	a.tray = trayManager

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

	// Set up callbacks - emit events via app.Event.Emit
	engine.SetStatusCallback(func(status sync.SyncStatus, action string) {
		a.app.Event.Emit("sync:status", map[string]interface{}{
			"status": status,
			"action": action,
		})

		// Update tray icon based on status
		if a.tray != nil {
			switch status {
			case sync.StatusIdle:
				a.tray.SetState(tray.StateIdle)
			case sync.StatusSyncing, sync.StatusScanning:
				a.tray.SetState(tray.StateSyncing)
			case sync.StatusError:
				a.tray.SetState(tray.StateError)
			default:
				a.tray.SetState(tray.StateIdle)
			}
		}
	})

	// Legacy per-file progress callback (for backward compatibility)
	engine.SetProgressCallback(func(progress *models.TransferProgress) {
		a.app.Event.Emit("sync:file-progress", progress)
	})

	// Aggregate progress callback (throttled, for main UI)
	engine.SetAggregateProgressCallback(func(progress *models.AggregateProgress) {
		a.app.Event.Emit("sync:progress", progress)
	})

	// Sync lifecycle events
	engine.SetSyncStartCallback(func() {
		a.app.Event.Emit("sync:start", nil)
	})
	engine.SetSyncEndCallback(func() {
		a.app.Event.Emit("sync:end", nil)
	})

	engine.SetEventCallback(func(event *sync.SyncEvent) {
		a.app.Event.Emit("sync:event", event)

		// Set error state on failure events
		if a.tray != nil && event.Type == "error" {
			a.tray.SetState(tray.StateError)
		}
	})

	engine.SetPeerChangeCallback(func() {
		a.app.Event.Emit("peers:changed", nil)
	})

	// Start sync engine
	if err := engine.Start(); err != nil {
		log.Printf("Failed to start sync engine: %v", err)
	}

	log.Println("SyncDev started successfully")
}

// shutdown is called when the app is closing
func (a *App) shutdown() {
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
	err := a.configStore.Update(func(c *config.Config) {
		c.SyncIntervalMins = mins
	})
	if err == nil && a.syncEngine != nil {
		a.syncEngine.RestartScheduler()
	}
	return err
}

// UpdateAutoSync updates the auto sync setting
func (a *App) UpdateAutoSync(enabled bool) error {
	err := a.configStore.Update(func(c *config.Config) {
		c.AutoSync = enabled
	})
	if err == nil && a.syncEngine != nil {
		a.syncEngine.UpdateAutoSync(enabled)
	}
	return err
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

	// Send folder pair configuration to the peer
	if a.syncEngine != nil {
		go func() {
			if err := a.syncEngine.SendFolderPairSync(peerID, pair, "add"); err != nil {
				log.Printf("Failed to sync folder pair to peer: %v", err)
			} else {
				log.Printf("Folder pair synced to peer %s", peerID)
			}
		}()
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

// GetSyncProgress returns the current transfer progress (legacy, per-file)
func (a *App) GetSyncProgress() *models.TransferProgress {
	if a.syncEngine == nil {
		return nil
	}
	return a.syncEngine.GetProgress()
}

// GetAggregateProgress returns the aggregate sync progress (throttled)
func (a *App) GetAggregateProgress() *models.AggregateProgress {
	if a.syncEngine == nil {
		return nil
	}
	return a.syncEngine.GetAggregateProgress()
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
	result, err := a.app.Dialog.OpenFile().
		SetTitle("Select Folder to Sync").
		SetDirectory(homeDir).
		CanChooseDirectories(true).
		CanChooseFiles(false).
		PromptForSingleSelection()
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
	a.app.Browser.OpenFile(dataDir)
}

// MinimizeToTray hides the window to system tray
func (a *App) MinimizeToTray() {
	a.window.Hide()
}

// ShowWindow shows and focuses the main window
func (a *App) ShowWindow() {
	a.window.Show()
	a.window.Focus()
}

// QuitApp quits the application
func (a *App) QuitApp() {
	a.app.Quit()
}

// ============================================
// Tray SyncActions Interface
// ============================================

// IsPaused returns whether sync is paused
func (a *App) IsPaused() bool {
	return a.paused
}

// Pause pauses automatic sync
func (a *App) Pause() {
	a.paused = true
	if a.syncEngine != nil {
		a.syncEngine.UpdateAutoSync(false)
	}
}

// Resume resumes automatic sync
func (a *App) Resume() {
	a.paused = false
	if a.syncEngine != nil {
		a.syncEngine.UpdateAutoSync(a.configStore.Get().AutoSync)
	}
}

// RefreshPeers triggers a peer discovery refresh
func (a *App) RefreshPeers() error {
	if a.syncEngine == nil {
		return fmt.Errorf("sync engine not initialized")
	}
	// The discovery is continuously running, just refresh UI
	a.app.Event.Emit("peers:changed", nil)
	return nil
}

// AnalyzeFolderPair returns a preview of what would be synced
func (a *App) AnalyzeFolderPair(folderPairID string) (*sync.SyncPreview, error) {
	if a.syncEngine == nil {
		return nil, fmt.Errorf("sync engine not initialized")
	}
	return a.syncEngine.AnalyzeFolderPair(folderPairID)
}
