package sync

import (
	"SyncDev/internal/config"
	"SyncDev/internal/models"
	"SyncDev/internal/network"
	"SyncDev/internal/secrets"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SyncStatus represents the current sync status
type SyncStatus string

const (
	StatusIdle     SyncStatus = "idle"
	StatusScanning SyncStatus = "scanning"
	StatusSyncing  SyncStatus = "syncing"
	StatusError    SyncStatus = "error"
)

// SyncEvent represents a sync activity event
type SyncEvent struct {
	Time        time.Time `json:"time"`
	Type        string    `json:"type"` // "push", "pull", "delete", "error"
	FolderPair  string    `json:"folderPair"`
	FilePath    string    `json:"filePath"`
	PeerName    string    `json:"peerName"`
	Description string    `json:"description"`
}

// Engine is the main sync engine
type Engine struct {
	config       *config.Store
	indexManager *IndexManager
	scanner      *Scanner
	server       *network.Server
	client       *network.Client
	discovery    *network.Discovery

	status        SyncStatus
	currentAction string
	progress      *models.TransferProgress
	recentEvents  []*SyncEvent

	connections   map[string]*network.PeerConnection
	fileReceivers map[string]*FileReceiver

	onStatusChange func(SyncStatus, string)
	onProgress     func(*models.TransferProgress)
	onEvent        func(*SyncEvent)
	onPeerChange   func()

	// Progress aggregation
	progressAggregator    *ProgressAggregator
	onAggregateProgress   func(*models.AggregateProgress)
	onSyncStart           func()
	onSyncEnd             func()

	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	schedulerStop chan struct{}
	schedulerMu   sync.Mutex
	schedulerRunning bool

	pairingCode   string
	pairingCodeMu sync.RWMutex
}

// NewEngine creates a new sync engine
func NewEngine(cfg *config.Store) (*Engine, error) {
	cfgData := cfg.Get()

	indexDir := filepath.Join(cfg.GetDataDir(), "indices")
	indexManager, err := NewIndexManager(indexDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create index manager: %w", err)
	}

	scanner := NewScanner(cfgData.GlobalExclusions)

	ctx, cancel := context.WithCancel(context.Background())

	engine := &Engine{
		config:        cfg,
		indexManager:  indexManager,
		scanner:       scanner,
		status:        StatusIdle,
		connections:   make(map[string]*network.PeerConnection),
		fileReceivers: make(map[string]*FileReceiver),
		recentEvents:  make([]*SyncEvent, 0),
		ctx:           ctx,
		cancel:        cancel,
		schedulerStop: make(chan struct{}),
	}

	// Create network components
	engine.server = network.NewServer(cfgData.Port)
	engine.server.SetHandler(engine)

	engine.client = network.NewClient(cfgData.DeviceID, cfgData.DeviceName)

	engine.discovery = network.NewDiscovery(cfgData.DeviceID, cfgData.DeviceName, cfgData.Port)
	engine.discovery.SetPeerFoundCallback(engine.handlePeerFound)
	engine.discovery.SetPeerLostCallback(engine.handlePeerLost)

	return engine, nil
}

// getSecretForPeer loads a peer's shared secret from keychain
func (e *Engine) getSecretForPeer(peerID string) string {
	secret, err := e.config.GetSecrets().GetSecret(peerID)
	if err != nil && err != secrets.ErrSecretNotFound {
		log.Printf("Warning: failed to load secret for peer %s: %v", peerID, err)
	}
	return secret
}

// Start starts the sync engine
func (e *Engine) Start() error {
	if err := e.server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	if err := e.discovery.Start(); err != nil {
		return fmt.Errorf("failed to start discovery: %w", err)
	}

	// Start scheduler for periodic syncs
	cfg := e.config.Get()
	if cfg.AutoSync {
		e.schedulerStop = make(chan struct{})
		go e.startScheduler()
	}

	log.Println("Sync engine started")
	return nil
}

// Stop stops the sync engine
func (e *Engine) Stop() {
	e.cancel()
	e.schedulerMu.Lock()
	if e.schedulerStop != nil && e.schedulerRunning {
		select {
		case e.schedulerStop <- struct{}{}:
		default:
		}
	}
	e.schedulerMu.Unlock()
	e.server.Stop()
	e.discovery.Stop()

	e.mu.Lock()
	for _, conn := range e.connections {
		conn.Close()
	}
	e.mu.Unlock()

	log.Println("Sync engine stopped")
}

// SetStatusCallback sets the status change callback
func (e *Engine) SetStatusCallback(cb func(SyncStatus, string)) {
	e.onStatusChange = cb
}

// SetProgressCallback sets the progress callback
func (e *Engine) SetProgressCallback(cb func(*models.TransferProgress)) {
	e.onProgress = cb
}

// SetEventCallback sets the event callback
func (e *Engine) SetEventCallback(cb func(*SyncEvent)) {
	e.onEvent = cb
}

// SetPeerChangeCallback sets the peer change callback
func (e *Engine) SetPeerChangeCallback(cb func()) {
	e.onPeerChange = cb
}

// SetAggregateProgressCallback sets up the progress aggregator with a callback
func (e *Engine) SetAggregateProgressCallback(cb func(*models.AggregateProgress)) {
	e.onAggregateProgress = cb
	e.progressAggregator = NewProgressAggregator(cb)
}

// SetSyncStartCallback sets the callback for sync start events
func (e *Engine) SetSyncStartCallback(cb func()) {
	e.onSyncStart = cb
}

// SetSyncEndCallback sets the callback for sync end events
func (e *Engine) SetSyncEndCallback(cb func()) {
	e.onSyncEnd = cb
}

// GetAggregateProgress returns the current aggregate progress
func (e *Engine) GetAggregateProgress() *models.AggregateProgress {
	if e.progressAggregator == nil {
		return nil
	}
	return e.progressAggregator.GetProgress()
}

// NotifySyncStart notifies that a sync session has started
func (e *Engine) NotifySyncStart(totalFiles int, totalBytes int64) {
	if e.progressAggregator != nil {
		e.progressAggregator.StartSync(totalFiles, totalBytes)
	}
	if e.onSyncStart != nil {
		e.onSyncStart()
	}
}

// NotifySyncEnd notifies that a sync session has ended
func (e *Engine) NotifySyncEnd() {
	if e.progressAggregator != nil {
		e.progressAggregator.EndSync()
	}
	if e.onSyncEnd != nil {
		e.onSyncEnd()
	}
}

// GetStatus returns the current sync status
func (e *Engine) GetStatus() (SyncStatus, string) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status, e.currentAction
}

// GetProgress returns the current transfer progress
func (e *Engine) GetProgress() *models.TransferProgress {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.progress
}

// GetRecentEvents returns recent sync events
func (e *Engine) GetRecentEvents() []*SyncEvent {
	e.mu.RLock()
	defer e.mu.RUnlock()
	events := make([]*SyncEvent, len(e.recentEvents))
	copy(events, e.recentEvents)
	return events
}

// GetDiscoveredPeers returns all discovered peers
func (e *Engine) GetDiscoveredPeers() []*models.Peer {
	discovered := e.discovery.GetPeers()
	cfg := e.config.Get()

	// Merge with paired peers from config
	for _, dp := range discovered {
		for _, cp := range cfg.Peers {
			if dp.ID == cp.ID {
				dp.Paired = cp.Paired
				// Note: SharedSecret loaded from keychain when connection is established
				break
			}
		}
	}

	return discovered
}

// setStatus updates the current status
func (e *Engine) setStatus(status SyncStatus, action string) {
	e.mu.Lock()
	e.status = status
	e.currentAction = action
	e.mu.Unlock()

	if e.onStatusChange != nil {
		e.onStatusChange(status, action)
	}
}

// addEvent adds a sync event
func (e *Engine) addEvent(event *SyncEvent) {
	e.mu.Lock()
	e.recentEvents = append([]*SyncEvent{event}, e.recentEvents...)
	if len(e.recentEvents) > 100 {
		e.recentEvents = e.recentEvents[:100]
	}
	e.mu.Unlock()

	if e.onEvent != nil {
		e.onEvent(event)
	}
}

// startScheduler starts the periodic sync scheduler
func (e *Engine) startScheduler() {
	e.schedulerMu.Lock()
	if e.schedulerRunning {
		e.schedulerMu.Unlock()
		return
	}
	e.schedulerRunning = true
	e.schedulerMu.Unlock()

	cfg := e.config.Get()
	interval := time.Duration(cfg.SyncIntervalMins) * time.Minute
	ticker := time.NewTicker(interval)
	defer func() {
		ticker.Stop()
		e.schedulerMu.Lock()
		e.schedulerRunning = false
		e.schedulerMu.Unlock()
	}()

	log.Printf("Scheduler started with interval: %d minutes", cfg.SyncIntervalMins)

	for {
		select {
		case <-e.schedulerStop:
			log.Println("Scheduler stopped via stop channel")
			return
		case <-e.ctx.Done():
			log.Println("Scheduler stopped via context cancellation")
			return
		case <-ticker.C:
			cfg := e.config.Get()
			if cfg.AutoSync {
				e.SyncAllPairs()
			}
		}
	}
}

// RestartScheduler restarts the scheduler with current config settings
func (e *Engine) RestartScheduler() {
	// Stop existing scheduler
	e.schedulerMu.Lock()
	wasRunning := e.schedulerRunning
	e.schedulerMu.Unlock()

	if wasRunning {
		// Signal stop and wait a bit
		select {
		case e.schedulerStop <- struct{}{}:
		default:
		}
		// Give it time to stop
		time.Sleep(100 * time.Millisecond)
	}

	// Recreate the stop channel
	e.schedulerStop = make(chan struct{})

	// Start new scheduler if auto-sync is enabled
	cfg := e.config.Get()
	if cfg.AutoSync {
		go e.startScheduler()
	}
	log.Printf("Scheduler restarted: autoSync=%v, interval=%d min", cfg.AutoSync, cfg.SyncIntervalMins)
}

// UpdateAutoSync updates auto-sync setting and restarts scheduler
func (e *Engine) UpdateAutoSync(enabled bool) {
	if enabled {
		e.schedulerMu.Lock()
		running := e.schedulerRunning
		e.schedulerMu.Unlock()
		if !running {
			e.schedulerStop = make(chan struct{})
			go e.startScheduler()
		}
	} else {
		e.schedulerMu.Lock()
		running := e.schedulerRunning
		e.schedulerMu.Unlock()
		if running {
			select {
			case e.schedulerStop <- struct{}{}:
			default:
			}
		}
	}
	log.Printf("Auto-sync updated: %v", enabled)
}

// SyncAllPairs syncs all folder pairs with all peers
func (e *Engine) SyncAllPairs() {
	cfg := e.config.Get()

	for _, fp := range cfg.FolderPairs {
		if !fp.Enabled {
			continue
		}

		peer := cfg.GetPeer(fp.PeerID)
		if peer == nil || !peer.Paired {
			continue
		}

		if err := e.SyncFolderPair(fp.ID); err != nil {
			log.Printf("Failed to sync folder pair %s: %v", fp.ID, err)
			e.addEvent(&SyncEvent{
				Time:        time.Now(),
				Type:        "error",
				FolderPair:  fp.ID,
				PeerName:    peer.Name,
				Description: fmt.Sprintf("Sync failed: %v", err),
			})
		}
	}
}

// SyncFolderPair syncs a specific folder pair
func (e *Engine) SyncFolderPair(folderPairID string) error {
	cfg := e.config.Get()
	fp := cfg.GetFolderPair(folderPairID)
	if fp == nil {
		return fmt.Errorf("folder pair not found: %s", folderPairID)
	}

	// Get peer from config (for pairing info)
	peer := cfg.GetPeer(fp.PeerID)
	if peer == nil {
		return fmt.Errorf("peer not found: %s", fp.PeerID)
	}

	// Check if peer is online via discovery (more up-to-date status)
	discoveredPeer := e.discovery.GetPeer(fp.PeerID)
	if discoveredPeer == nil || discoveredPeer.Status != models.PeerStatusOnline {
		return fmt.Errorf("peer is offline: %s", peer.Name)
	}

	// Update peer info from discovery
	peer.Host = discoveredPeer.Host
	peer.Port = discoveredPeer.Port
	peer.Status = discoveredPeer.Status

	e.setStatus(StatusScanning, fmt.Sprintf("Scanning %s", fp.LocalPath))

	// Scan local directory
	localIndex, err := e.scanner.ScanDirectory(fp.LocalPath)
	if err != nil {
		e.setStatus(StatusError, err.Error())
		return fmt.Errorf("failed to scan local directory: %w", err)
	}

	// Connect to peer
	conn, err := e.getOrCreateConnection(peer)
	if err != nil {
		e.setStatus(StatusError, err.Error())
		return fmt.Errorf("failed to connect to peer: %w", err)
	}

	e.setStatus(StatusSyncing, fmt.Sprintf("Syncing with %s", peer.Name))

	// Send sync request
	if err := e.client.SendSyncRequest(conn, fp.ID, fp.LocalPath, fp.RemotePath); err != nil {
		return fmt.Errorf("failed to send sync request: %w", err)
	}

	// Send our index
	indexPayload := &network.IndexExchangePayload{
		FolderPairID: fp.ID,
		Index:        localIndex.Files,
	}
	if err := e.client.SendIndexExchange(conn, indexPayload); err != nil {
		return fmt.Errorf("failed to send index: %w", err)
	}

	// Save local index
	if err := e.indexManager.SaveIndex(fp.ID, localIndex); err != nil {
		log.Printf("Failed to save index: %v", err)
	}

	// Update last sync time
	e.config.Update(func(c *config.Config) {
		if cfp := c.GetFolderPair(fp.ID); cfp != nil {
			cfp.LastSyncTime = time.Now()
		}
	})

	e.setStatus(StatusIdle, "")
	return nil
}

// getOrCreateConnection gets an existing connection or creates a new one
func (e *Engine) getOrCreateConnection(peer *models.Peer) (*network.PeerConnection, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if conn, ok := e.connections[peer.ID]; ok {
		return conn, nil
	}

	conn, err := e.client.Connect(peer.Host, peer.Port)
	if err != nil {
		return nil, err
	}

	conn.PeerID = peer.ID
	conn.PeerName = peer.Name
	conn.SharedSecret = e.getSecretForPeer(peer.ID)
	conn.Paired = peer.Paired

	e.connections[peer.ID] = conn

	// Start read loop in background
	go e.readConnectionLoop(conn)

	return conn, nil
}

// readConnectionLoop reads messages from an outgoing connection
func (e *Engine) readConnectionLoop(conn *network.PeerConnection) {
	defer func() {
		e.mu.Lock()
		delete(e.connections, conn.PeerID)
		e.mu.Unlock()
		conn.Close()
	}()

	for {
		select {
		case <-e.ctx.Done():
			return
		default:
			msg, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Connection read error: %v", err)
				return
			}
			e.HandleMessage(conn, msg)
		}
	}
}

// HandleMessage handles incoming protocol messages
func (e *Engine) HandleMessage(conn *network.PeerConnection, msg *network.Message) {
	switch msg.Type {
	case network.MsgTypePairingReq:
		e.handlePairingRequest(conn, msg)
	case network.MsgTypePairingResp:
		e.handlePairingResponse(conn, msg)
	case network.MsgTypeSyncRequest:
		e.handleSyncRequest(conn, msg)
	case network.MsgTypeSyncResponse:
		e.handleSyncResponse(conn, msg)
	case network.MsgTypeIndexExchange:
		e.handleIndexExchange(conn, msg)
	case network.MsgTypeFileRequest:
		e.handleFileRequest(conn, msg)
	case network.MsgTypeFileChunk:
		e.handleFileChunk(conn, msg)
	case network.MsgTypeFileComplete:
		e.handleFileComplete(conn, msg)
	case network.MsgTypeDeleteFile:
		e.handleDeleteFile(conn, msg)
	case network.MsgTypePing:
		e.client.SendPong(conn)
	case network.MsgTypeFolderPairSync:
		e.handleFolderPairSync(conn, msg)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// OnConnect is called when a peer connects
func (e *Engine) OnConnect(conn *network.PeerConnection) {
	log.Printf("Peer connected: %s (%s)", conn.PeerName, conn.PeerID)

	// Check if peer is already paired
	cfg := e.config.Get()
	if peer := cfg.GetPeer(conn.PeerID); peer != nil && peer.Paired {
		conn.SharedSecret = e.getSecretForPeer(conn.PeerID)
		conn.Paired = true
	}

	if e.onPeerChange != nil {
		e.onPeerChange()
	}
}

// OnDisconnect is called when a peer disconnects
func (e *Engine) OnDisconnect(conn *network.PeerConnection) {
	log.Printf("Peer disconnected: %s (%s)", conn.PeerName, conn.PeerID)

	if e.onPeerChange != nil {
		e.onPeerChange()
	}
}

// handlePeerFound is called when a peer is discovered via mDNS
func (e *Engine) handlePeerFound(peer *models.Peer) {
	// Check if already paired
	cfg := e.config.Get()
	if existingPeer := cfg.GetPeer(peer.ID); existingPeer != nil {
		peer.Paired = existingPeer.Paired
		// Don't copy SharedSecret from existingPeer - it's in keychain
		// Secret will be loaded when connection is established
	}

	if e.onPeerChange != nil {
		e.onPeerChange()
	}
}

// handlePeerLost is called when a peer goes offline
func (e *Engine) handlePeerLost(peerID string) {
	if e.onPeerChange != nil {
		e.onPeerChange()
	}
}

// handlePairingRequest handles an incoming pairing request
func (e *Engine) handlePairingRequest(conn *network.PeerConnection, msg *network.Message) {
	var payload network.PairingRequestPayload
	if err := msg.ParsePayload(&payload); err != nil {
		log.Printf("Failed to parse pairing request: %v", err)
		return
	}

	log.Printf("Received pairing request from %s with code %s", payload.DeviceName, payload.Code)

	// Verify the pairing code
	currentCode := e.GetPairingCode()
	if currentCode == "" {
		log.Printf("No pairing code generated, rejecting request")
		e.client.SendPairingResponse(conn, false, "", "No pairing code active on this device")
		return
	}

	if payload.Code != currentCode {
		log.Printf("Pairing code mismatch: expected %s, got %s", currentCode, payload.Code)
		e.client.SendPairingResponse(conn, false, "", "Invalid pairing code")
		return
	}

	// Code matches - accept pairing
	log.Printf("Pairing code verified, accepting pairing from %s", payload.DeviceName)

	// Generate shared secret
	secret, err := network.GenerateSharedSecret()
	if err != nil {
		log.Printf("Failed to generate shared secret: %v", err)
		e.client.SendPairingResponse(conn, false, "", "Internal error")
		return
	}

	// Update connection
	conn.SharedSecret = secret
	conn.Paired = true
	conn.PeerID = payload.DeviceID
	conn.PeerName = payload.DeviceName

	// Store secret in keychain
	if err := e.config.GetSecrets().SetSecret(payload.DeviceID, secret); err != nil {
		log.Printf("Error: failed to store secret for peer %s: %v", payload.DeviceID, err)
	}

	// Save to config (without secret - it's in keychain)
	e.config.Update(func(c *config.Config) {
		peer := c.GetPeer(payload.DeviceID)
		if peer == nil {
			peer = &models.Peer{
				ID:   payload.DeviceID,
				Name: payload.DeviceName,
			}
			c.AddPeer(peer)
		}
		// Don't set peer.SharedSecret - it's stored in keychain
		peer.Paired = true
		peer.Host = conn.Conn.RemoteAddr().String()
	})

	// Store connection
	e.mu.Lock()
	e.connections[payload.DeviceID] = conn
	e.mu.Unlock()

	// Send acceptance response with shared secret
	e.client.SendPairingResponse(conn, true, secret, "")

	// Clear the pairing code after successful pairing
	e.ClearPairingCode()

	// Notify UI
	if e.onPeerChange != nil {
		e.onPeerChange()
	}

	log.Printf("Pairing completed with %s", payload.DeviceName)
}

// handlePairingResponse handles a pairing response
func (e *Engine) handlePairingResponse(conn *network.PeerConnection, msg *network.Message) {
	var payload network.PairingResponsePayload
	if err := msg.ParsePayload(&payload); err != nil {
		log.Printf("Failed to parse pairing response: %v", err)
		return
	}

	if payload.Accepted {
		conn.SharedSecret = payload.SharedSecret
		conn.Paired = true

		// Store secret in keychain
		if err := e.config.GetSecrets().SetSecret(conn.PeerID, payload.SharedSecret); err != nil {
			log.Printf("Error: failed to store secret for peer %s: %v", conn.PeerID, err)
		}

		// Save to config (without secret - it's in keychain)
		e.config.Update(func(c *config.Config) {
			peer := c.GetPeer(conn.PeerID)
			if peer == nil {
				peer = &models.Peer{
					ID:   conn.PeerID,
					Name: conn.PeerName,
				}
				c.AddPeer(peer)
			}
			// Don't set peer.SharedSecret - it's stored in keychain
			peer.Paired = true
		})

		log.Printf("Pairing accepted by %s", conn.PeerName)
	} else {
		log.Printf("Pairing rejected by %s: %s", conn.PeerName, payload.Error)
	}

	if e.onPeerChange != nil {
		e.onPeerChange()
	}
}

// handleSyncRequest handles a sync request from a peer
func (e *Engine) handleSyncRequest(conn *network.PeerConnection, msg *network.Message) {
	var payload network.SyncRequestPayload
	if err := msg.ParsePayload(&payload); err != nil {
		log.Printf("Failed to parse sync request: %v", err)
		return
	}

	cfg := e.config.Get()
	fp := cfg.GetFolderPair(payload.FolderPairID)

	// Send response
	accepted := fp != nil && fp.Enabled
	respPayload := &network.SyncResponsePayload{
		FolderPairID: payload.FolderPairID,
		Accepted:     accepted,
	}
	if !accepted {
		respPayload.Error = "Folder pair not found or disabled"
	}

	respMsg, _ := network.NewMessage(network.MsgTypeSyncResponse, respPayload)
	conn.WriteMessage(respMsg)
}

// handleSyncResponse handles a sync response
func (e *Engine) handleSyncResponse(conn *network.PeerConnection, msg *network.Message) {
	var payload network.SyncResponsePayload
	if err := msg.ParsePayload(&payload); err != nil {
		log.Printf("Failed to parse sync response: %v", err)
		return
	}

	if !payload.Accepted {
		log.Printf("Sync rejected for folder pair %s: %s", payload.FolderPairID, payload.Error)
	}
}

// handleIndexExchange handles an incoming file index
func (e *Engine) handleIndexExchange(conn *network.PeerConnection, msg *network.Message) {
	var payload network.IndexExchangePayload
	if err := msg.ParsePayload(&payload); err != nil {
		log.Printf("Failed to parse index exchange: %v", err)
		return
	}

	cfg := e.config.Get()
	fp := cfg.GetFolderPair(payload.FolderPairID)
	if fp == nil {
		log.Printf("Folder pair not found: %s", payload.FolderPairID)
		return
	}

	// Scan our local directory
	localIndex, err := e.scanner.ScanDirectory(fp.LocalPath)
	if err != nil {
		log.Printf("Failed to scan local directory: %v", err)
		return
	}

	// Build remote index
	remoteIndex := &models.FileIndex{
		FolderPath: fp.RemotePath,
		Files:      payload.Index,
	}

	// Compare indices
	actions := CompareIndices(localIndex, remoteIndex)

	// Calculate total files and bytes for sync
	totalFiles := 0
	var totalBytes int64
	for _, action := range actions {
		if action.Action == models.FileActionPush && action.LocalFile != nil && !action.LocalFile.IsDir {
			totalFiles++
			totalBytes += action.LocalFile.Size
		} else if action.Action == models.FileActionPull && action.RemoteFile != nil && !action.RemoteFile.IsDir {
			totalFiles++
			totalBytes += action.RemoteFile.Size
		}
	}

	// Notify sync start
	if totalFiles > 0 {
		e.NotifySyncStart(totalFiles, totalBytes)
	}

	// Execute actions
	for _, action := range actions {
		switch action.Action {
		case models.FileActionPush:
			e.pushFile(conn, fp, action.LocalFile)
		case models.FileActionPull:
			e.pullFile(conn, fp, action.RemoteFile)
		}
	}

	// Notify sync end
	if totalFiles > 0 {
		e.NotifySyncEnd()
	}

	// Send our index back
	indexPayload := &network.IndexExchangePayload{
		FolderPairID: fp.ID,
		Index:        localIndex.Files,
	}
	e.client.SendIndexExchange(conn, indexPayload)

	// Save updated index
	e.indexManager.SaveIndex(fp.ID, localIndex)
}

// pushFile sends a file to the peer
func (e *Engine) pushFile(conn *network.PeerConnection, fp *models.FolderPair, fileInfo *models.FileInfo) {
	if fileInfo.IsDir {
		// Just notify about directory
		return
	}

	tm := NewTransferManager(fp.LocalPath, e.scanner)

	progressCb := func(p *models.TransferProgress) {
		e.mu.Lock()
		e.progress = p
		e.mu.Unlock()

		// Legacy callback for backward compatibility
		if e.onProgress != nil {
			e.onProgress(p)
		}

		// Feed progress to aggregator
		if e.progressAggregator != nil {
			e.progressAggregator.UpdateFile(p.FileName, p.TotalBytes, p.TransferBytes)
		}
	}

	if err := tm.SendFile(conn, fp.ID, fileInfo.Path, progressCb); err != nil {
		log.Printf("Failed to push file %s: %v", fileInfo.Path, err)
		e.addEvent(&SyncEvent{
			Time:        time.Now(),
			Type:        "error",
			FolderPair:  fp.ID,
			FilePath:    fileInfo.Path,
			PeerName:    conn.PeerName,
			Description: fmt.Sprintf("Push failed: %v", err),
		})
		return
	}

	// Mark file complete in aggregator
	if e.progressAggregator != nil {
		e.progressAggregator.CompleteFile(fileInfo.Path, fileInfo.Size)
	}

	e.addEvent(&SyncEvent{
		Time:        time.Now(),
		Type:        "push",
		FolderPair:  fp.ID,
		FilePath:    fileInfo.Path,
		PeerName:    conn.PeerName,
		Description: "File sent",
	})
}

// pullFile requests a file from the peer
func (e *Engine) pullFile(conn *network.PeerConnection, fp *models.FolderPair, fileInfo *models.FileInfo) {
	if fileInfo.IsDir {
		// Create directory locally
		dirPath := filepath.Join(fp.LocalPath, fileInfo.Path)
		os.MkdirAll(dirPath, os.FileMode(fileInfo.Permission))
		return
	}

	if err := e.client.SendFileRequest(conn, fp.ID, fileInfo.Path, 0); err != nil {
		log.Printf("Failed to request file %s: %v", fileInfo.Path, err)
	}
}

// handleFileRequest handles a file request from a peer
func (e *Engine) handleFileRequest(conn *network.PeerConnection, msg *network.Message) {
	var payload network.FileRequestPayload
	if err := msg.ParsePayload(&payload); err != nil {
		return
	}

	cfg := e.config.Get()
	fp := cfg.GetFolderPair(payload.FolderPairID)
	if fp == nil {
		return
	}

	fileInfo, err := e.scanner.GetFileInfo(fp.LocalPath, payload.FilePath)
	if err != nil {
		respPayload := &network.FileResponsePayload{
			FolderPairID: payload.FolderPairID,
			FilePath:     payload.FilePath,
			Error:        err.Error(),
		}
		respMsg, _ := network.NewMessage(network.MsgTypeFileResponse, respPayload)
		conn.WriteMessage(respMsg)
		return
	}

	// Send file response
	respPayload := &network.FileResponsePayload{
		FolderPairID: payload.FolderPairID,
		FilePath:     payload.FilePath,
		Size:         fileInfo.Size,
		Hash:         fileInfo.Hash,
	}
	respMsg, _ := network.NewMessage(network.MsgTypeFileResponse, respPayload)
	conn.WriteMessage(respMsg)

	// Send file chunks
	tm := NewTransferManager(fp.LocalPath, e.scanner)
	tm.SendFile(conn, fp.ID, payload.FilePath, nil)
}

// handleFileChunk handles an incoming file chunk
func (e *Engine) handleFileChunk(conn *network.PeerConnection, msg *network.Message) {
	var payload network.FileChunkPayload
	if err := msg.ParsePayload(&payload); err != nil {
		return
	}

	key := fmt.Sprintf("%s:%s", payload.FolderPairID, payload.FilePath)

	e.mu.Lock()
	receiver, exists := e.fileReceivers[key]
	e.mu.Unlock()

	if !exists {
		// Create new receiver
		cfg := e.config.Get()
		fp := cfg.GetFolderPair(payload.FolderPairID)
		if fp == nil {
			return
		}

		var err error
		receiver, err = NewFileReceiver(fp.LocalPath, payload.FilePath, 0, func(p *models.TransferProgress) {
			e.mu.Lock()
			e.progress = p
			e.mu.Unlock()

			// Legacy callback for backward compatibility
			if e.onProgress != nil {
				e.onProgress(p)
			}

			// Feed progress to aggregator
			if e.progressAggregator != nil {
				e.progressAggregator.UpdateFile(p.FileName, p.TotalBytes, p.TransferBytes)
			}
		})
		if err != nil {
			log.Printf("Failed to create file receiver: %v", err)
			return
		}

		e.mu.Lock()
		e.fileReceivers[key] = receiver
		e.mu.Unlock()
	}

	if err := receiver.WriteChunk(payload.Data, payload.Offset); err != nil {
		log.Printf("Failed to write chunk: %v", err)
		receiver.Abort()
		e.mu.Lock()
		delete(e.fileReceivers, key)
		e.mu.Unlock()
		return
	}

	if payload.IsLast {
		// Mark file complete in aggregator before finalize
		if e.progressAggregator != nil {
			// Get file size from receiver progress
			if e.progress != nil {
				e.progressAggregator.CompleteFile(payload.FilePath, e.progress.TotalBytes)
			}
		}

		if err := receiver.Finalize(); err != nil {
			log.Printf("Failed to finalize file: %v", err)
		}
		e.mu.Lock()
		delete(e.fileReceivers, key)
		e.mu.Unlock()

		e.addEvent(&SyncEvent{
			Time:        time.Now(),
			Type:        "pull",
			FolderPair:  payload.FolderPairID,
			FilePath:    payload.FilePath,
			PeerName:    conn.PeerName,
			Description: "File received",
		})
	}
}

// handleFileComplete handles a file complete notification
func (e *Engine) handleFileComplete(conn *network.PeerConnection, msg *network.Message) {
	var payload network.FileCompletePayload
	if err := msg.ParsePayload(&payload); err != nil {
		return
	}

	if !payload.Success {
		log.Printf("File transfer failed for %s: %s", payload.FilePath, payload.Error)
	}
}

// handleDeleteFile handles a delete file request
func (e *Engine) handleDeleteFile(conn *network.PeerConnection, msg *network.Message) {
	var payload network.DeleteFilePayload
	if err := msg.ParsePayload(&payload); err != nil {
		return
	}

	cfg := e.config.Get()
	fp := cfg.GetFolderPair(payload.FolderPairID)
	if fp == nil {
		return
	}

	fullPath := filepath.Join(fp.LocalPath, payload.FilePath)
	if err := DeleteFile(fullPath); err != nil {
		log.Printf("Failed to delete file %s: %v", payload.FilePath, err)
		return
	}

	// Send ack
	ackMsg, _ := network.NewMessage(network.MsgTypeDeleteAck, payload)
	conn.WriteMessage(ackMsg)

	e.addEvent(&SyncEvent{
		Time:        time.Now(),
		Type:        "delete",
		FolderPair:  fp.ID,
		FilePath:    payload.FilePath,
		PeerName:    conn.PeerName,
		Description: "File deleted",
	})
}

// handleFolderPairSync handles receiving a folder pair configuration from a peer
func (e *Engine) handleFolderPairSync(conn *network.PeerConnection, msg *network.Message) {
	var payload network.FolderPairSyncPayload
	if err := msg.ParsePayload(&payload); err != nil {
		log.Printf("Failed to parse folder pair sync: %v", err)
		return
	}

	log.Printf("Received folder pair sync from %s: action=%s, local=%s, remote=%s",
		conn.PeerName, payload.Action, payload.LocalPath, payload.RemotePath)

	if payload.Action == "add" {
		// Create the mirrored folder pair (swap local and remote paths)
		e.config.Update(func(c *config.Config) {
			// Check if we already have this folder pair
			existing := c.GetFolderPair(payload.FolderPairID)
			if existing != nil {
				log.Printf("Folder pair %s already exists, skipping", payload.FolderPairID)
				return
			}

			// Create mirrored folder pair - swap the paths!
			fp := &models.FolderPair{
				ID:         payload.FolderPairID,
				PeerID:     conn.PeerID,
				LocalPath:  payload.RemotePath, // Their remote is our local
				RemotePath: payload.LocalPath,  // Their local is our remote
				Enabled:    true,
			}
			c.AddFolderPair(fp)
			log.Printf("Created mirrored folder pair: local=%s, remote=%s", fp.LocalPath, fp.RemotePath)
		})
	} else if payload.Action == "remove" {
		e.config.Update(func(c *config.Config) {
			c.RemoveFolderPair(payload.FolderPairID)
			log.Printf("Removed folder pair %s", payload.FolderPairID)
		})
	}

	// Notify UI about the change
	if e.onPeerChange != nil {
		e.onPeerChange()
	}
}

// SendFolderPairSync sends a folder pair configuration to a peer
func (e *Engine) SendFolderPairSync(peerID string, fp *models.FolderPair, action string) error {
	peer := e.discovery.GetPeer(peerID)
	if peer == nil {
		// Try from config
		cfg := e.config.Get()
		peer = cfg.GetPeer(peerID)
	}
	if peer == nil {
		return fmt.Errorf("peer not found: %s", peerID)
	}

	conn, err := e.getOrCreateConnection(peer)
	if err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}

	payload := &network.FolderPairSyncPayload{
		FolderPairID: fp.ID,
		LocalPath:    fp.LocalPath,
		RemotePath:   fp.RemotePath,
		Action:       action,
	}

	msg, err := network.NewMessage(network.MsgTypeFolderPairSync, payload)
	if err != nil {
		return err
	}

	return conn.WriteMessage(msg)
}

// RequestPairing initiates pairing with a peer
func (e *Engine) RequestPairing(peerID, code string) error {
	peer := e.discovery.GetPeer(peerID)
	if peer == nil {
		return fmt.Errorf("peer not found")
	}

	conn, err := e.getOrCreateConnection(peer)
	if err != nil {
		return err
	}

	return e.client.SendPairingRequest(conn, code)
}

// AcceptPairing accepts a pairing request
func (e *Engine) AcceptPairing(peerID string) error {
	conn := e.server.GetConnection(peerID)
	if conn == nil {
		return fmt.Errorf("connection not found")
	}

	secret, err := network.GenerateSharedSecret()
	if err != nil {
		return err
	}

	// Store secret in keychain
	if err := e.config.GetSecrets().SetSecret(peerID, secret); err != nil {
		log.Printf("Error: failed to store secret for peer %s: %v", peerID, err)
	}

	// Save pairing to config (without secret - it's in keychain)
	e.config.Update(func(c *config.Config) {
		peer := c.GetPeer(peerID)
		if peer == nil {
			peer = &models.Peer{
				ID:   peerID,
				Name: conn.PeerName,
			}
			c.AddPeer(peer)
		}
		// Don't set peer.SharedSecret - it's stored in keychain
		peer.Paired = true
	})

	conn.SharedSecret = secret
	conn.Paired = true

	return e.client.SendPairingResponse(conn, true, secret, "")
}

// RejectPairing rejects a pairing request
func (e *Engine) RejectPairing(peerID string) error {
	conn := e.server.GetConnection(peerID)
	if conn == nil {
		return fmt.Errorf("connection not found")
	}

	return e.client.SendPairingResponse(conn, false, "", "Pairing rejected by user")
}

// UnpairPeer removes pairing with a peer
func (e *Engine) UnpairPeer(peerID string) error {
	// Delete secret from keychain
	if err := e.config.GetSecrets().DeleteSecret(peerID); err != nil {
		log.Printf("Warning: failed to delete secret for peer %s: %v", peerID, err)
	}

	e.config.Update(func(c *config.Config) {
		if peer := c.GetPeer(peerID); peer != nil {
			peer.Paired = false
			// Don't clear peer.SharedSecret - it's already in keychain (or deleted)
		}
		// Remove folder pairs for this peer
		for i := len(c.FolderPairs) - 1; i >= 0; i-- {
			if c.FolderPairs[i].PeerID == peerID {
				c.FolderPairs = append(c.FolderPairs[:i], c.FolderPairs[i+1:]...)
			}
		}
	})

	// Close connection if exists
	e.mu.Lock()
	if conn, ok := e.connections[peerID]; ok {
		conn.Close()
		delete(e.connections, peerID)
	}
	e.mu.Unlock()

	return nil
}

// GeneratePairingCode generates a new pairing code and stores it
func (e *Engine) GeneratePairingCode() string {
	code := network.GeneratePairingCode()
	e.pairingCodeMu.Lock()
	e.pairingCode = code
	e.pairingCodeMu.Unlock()
	log.Printf("Generated pairing code: %s", code)
	return code
}

// GetPairingCode returns the current pairing code
func (e *Engine) GetPairingCode() string {
	e.pairingCodeMu.RLock()
	defer e.pairingCodeMu.RUnlock()
	return e.pairingCode
}

// ClearPairingCode clears the current pairing code
func (e *Engine) ClearPairingCode() {
	e.pairingCodeMu.Lock()
	e.pairingCode = ""
	e.pairingCodeMu.Unlock()
}

// SyncPreview represents a preview of sync changes
type SyncPreview struct {
	FolderPairID string              `json:"folderPairId"`
	PeerName     string              `json:"peerName"`
	LocalPath    string              `json:"localPath"`
	RemotePath   string              `json:"remotePath"`
	ToPush       []*models.FileInfo  `json:"toPush"`
	ToPull       []*models.FileInfo  `json:"toPull"`
	ToDelete     []*models.FileInfo  `json:"toDelete"`
	PushCount    int                 `json:"pushCount"`
	PullCount    int                 `json:"pullCount"`
	DeleteCount  int                 `json:"deleteCount"`
	PushSize     int64               `json:"pushSize"`
	PullSize     int64               `json:"pullSize"`
	Error        string              `json:"error,omitempty"`
}

// AnalyzeFolderPair analyzes a folder pair and returns what would be synced
func (e *Engine) AnalyzeFolderPair(folderPairID string) (*SyncPreview, error) {
	cfg := e.config.Get()
	fp := cfg.GetFolderPair(folderPairID)
	if fp == nil {
		return nil, fmt.Errorf("folder pair not found: %s", folderPairID)
	}

	peer := cfg.GetPeer(fp.PeerID)
	if peer == nil {
		return nil, fmt.Errorf("peer not found: %s", fp.PeerID)
	}

	preview := &SyncPreview{
		FolderPairID: fp.ID,
		PeerName:     peer.Name,
		LocalPath:    fp.LocalPath,
		RemotePath:   fp.RemotePath,
		ToPush:       make([]*models.FileInfo, 0),
		ToPull:       make([]*models.FileInfo, 0),
		ToDelete:     make([]*models.FileInfo, 0),
	}

	// Check if peer is online
	discoveredPeer := e.discovery.GetPeer(fp.PeerID)
	if discoveredPeer == nil || discoveredPeer.Status != models.PeerStatusOnline {
		preview.Error = "Peer is offline"
		return preview, nil
	}

	// Scan local directory
	localIndex, err := e.scanner.ScanDirectory(fp.LocalPath)
	if err != nil {
		preview.Error = fmt.Sprintf("Failed to scan local directory: %v", err)
		return preview, nil
	}

	// Try to load last known remote index
	remoteIndex, err := e.indexManager.LoadIndex(fp.ID + "_remote")
	if err != nil {
		// No previous remote index, everything is new
		for _, f := range localIndex.Files {
			if !f.IsDir {
				preview.ToPush = append(preview.ToPush, f)
				preview.PushSize += f.Size
			}
		}
		preview.PushCount = len(preview.ToPush)
		return preview, nil
	}

	// Compare indices
	actions := CompareIndices(localIndex, remoteIndex)

	for _, action := range actions {
		switch action.Action {
		case models.FileActionPush:
			if action.LocalFile != nil && !action.LocalFile.IsDir {
				preview.ToPush = append(preview.ToPush, action.LocalFile)
				preview.PushSize += action.LocalFile.Size
			}
		case models.FileActionPull:
			if action.RemoteFile != nil && !action.RemoteFile.IsDir {
				preview.ToPull = append(preview.ToPull, action.RemoteFile)
				preview.PullSize += action.RemoteFile.Size
			}
		case models.FileActionDelete:
			if action.LocalFile != nil {
				preview.ToDelete = append(preview.ToDelete, action.LocalFile)
			}
		}
	}

	preview.PushCount = len(preview.ToPush)
	preview.PullCount = len(preview.ToPull)
	preview.DeleteCount = len(preview.ToDelete)

	return preview, nil
}
