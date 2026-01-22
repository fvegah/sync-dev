package sync

import (
	"SyncDev/internal/config"
	"SyncDev/internal/models"
	"SyncDev/internal/network"
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

	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	schedulerStop chan struct{}

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
		go e.startScheduler()
	}

	log.Println("Sync engine started")
	return nil
}

// Stop stops the sync engine
func (e *Engine) Stop() {
	e.cancel()
	close(e.schedulerStop)
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
				dp.SharedSecret = cp.SharedSecret
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
	cfg := e.config.Get()
	interval := time.Duration(cfg.SyncIntervalMins) * time.Minute
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-e.schedulerStop:
			return
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.SyncAllPairs()
		}
	}
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

	peer := cfg.GetPeer(fp.PeerID)
	if peer == nil {
		return fmt.Errorf("peer not found: %s", fp.PeerID)
	}

	if peer.Status != models.PeerStatusOnline {
		return fmt.Errorf("peer is offline: %s", peer.Name)
	}

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
	conn.SharedSecret = peer.SharedSecret
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
		conn.SharedSecret = peer.SharedSecret
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
		peer.SharedSecret = existingPeer.SharedSecret
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

	// Save to config
	e.config.Update(func(c *config.Config) {
		peer := c.GetPeer(payload.DeviceID)
		if peer == nil {
			peer = &models.Peer{
				ID:   payload.DeviceID,
				Name: payload.DeviceName,
			}
			c.AddPeer(peer)
		}
		peer.SharedSecret = secret
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

		// Save to config
		e.config.Update(func(c *config.Config) {
			peer := c.GetPeer(conn.PeerID)
			if peer == nil {
				peer = &models.Peer{
					ID:   conn.PeerID,
					Name: conn.PeerName,
				}
				c.AddPeer(peer)
			}
			peer.SharedSecret = payload.SharedSecret
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

	// Execute actions
	for _, action := range actions {
		switch action.Action {
		case models.FileActionPush:
			e.pushFile(conn, fp, action.LocalFile)
		case models.FileActionPull:
			e.pullFile(conn, fp, action.RemoteFile)
		}
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
		if e.onProgress != nil {
			e.onProgress(p)
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
			if e.onProgress != nil {
				e.onProgress(p)
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

	// Save pairing
	e.config.Update(func(c *config.Config) {
		peer := c.GetPeer(peerID)
		if peer == nil {
			peer = &models.Peer{
				ID:   peerID,
				Name: conn.PeerName,
			}
			c.AddPeer(peer)
		}
		peer.SharedSecret = secret
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
	e.config.Update(func(c *config.Config) {
		if peer := c.GetPeer(peerID); peer != nil {
			peer.Paired = false
			peer.SharedSecret = ""
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
