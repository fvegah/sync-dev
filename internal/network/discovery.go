package network

import (
	"SyncDev/internal/config"
	"SyncDev/internal/models"
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
)

// Discovery handles mDNS service discovery for finding peers on the local network
type Discovery struct {
	deviceID   string
	deviceName string
	port       int
	server     *mdns.Server
	peers      map[string]*models.Peer
	mu         sync.RWMutex
	onPeerFound func(*models.Peer)
	onPeerLost  func(string)
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewDiscovery creates a new Discovery instance
func NewDiscovery(deviceID, deviceName string, port int) *Discovery {
	ctx, cancel := context.WithCancel(context.Background())
	return &Discovery{
		deviceID:   deviceID,
		deviceName: deviceName,
		port:       port,
		peers:      make(map[string]*models.Peer),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// SetPeerFoundCallback sets the callback for when a peer is discovered
func (d *Discovery) SetPeerFoundCallback(cb func(*models.Peer)) {
	d.onPeerFound = cb
}

// SetPeerLostCallback sets the callback for when a peer is lost
func (d *Discovery) SetPeerLostCallback(cb func(string)) {
	d.onPeerLost = cb
}

// Start begins advertising this device and scanning for others
func (d *Discovery) Start() error {
	if err := d.startAdvertising(); err != nil {
		return fmt.Errorf("failed to start advertising: %w", err)
	}

	go d.scanLoop()
	return nil
}

// Stop stops discovery and advertising
func (d *Discovery) Stop() {
	d.cancel()
	if d.server != nil {
		d.server.Shutdown()
	}
}

// startAdvertising begins advertising this device via mDNS
func (d *Discovery) startAdvertising() error {
	// Get local IP
	localIP, err := getLocalIP()
	if err != nil {
		return fmt.Errorf("failed to get local IP: %w", err)
	}

	// Create TXT records with device info
	txtRecords := []string{
		fmt.Sprintf("device_id=%s", d.deviceID),
		fmt.Sprintf("device_name=%s", d.deviceName),
		fmt.Sprintf("version=%s", config.AppVersion),
	}

	// Create mDNS service
	service, err := mdns.NewMDNSService(
		d.deviceID,
		config.ServiceName,
		"",
		"",
		d.port,
		[]net.IP{localIP},
		txtRecords,
	)
	if err != nil {
		return fmt.Errorf("failed to create mDNS service: %w", err)
	}

	// Create and start server
	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return fmt.Errorf("failed to start mDNS server: %w", err)
	}

	d.server = server
	log.Printf("mDNS: Advertising as %s on port %d", d.deviceName, d.port)
	return nil
}

// scanLoop periodically scans for peers
func (d *Discovery) scanLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Do initial scan immediately
	d.scan()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.scan()
			d.checkStalePeers()
		}
	}
}

// scan performs a single mDNS scan
func (d *Discovery) scan() {
	entriesCh := make(chan *mdns.ServiceEntry, 10)
	go func() {
		for entry := range entriesCh {
			d.handleServiceEntry(entry)
		}
	}()

	params := &mdns.QueryParam{
		Service:             config.ServiceName,
		Domain:              "local",
		Timeout:             5 * time.Second,
		Entries:             entriesCh,
		WantUnicastResponse: true,
	}

	if err := mdns.Query(params); err != nil {
		log.Printf("mDNS: Query error: %v", err)
	}
	close(entriesCh)
}

// handleServiceEntry processes a discovered service
func (d *Discovery) handleServiceEntry(entry *mdns.ServiceEntry) {
	// Parse TXT records
	deviceID := ""
	deviceName := ""
	version := ""

	for _, txt := range entry.InfoFields {
		switch {
		case len(txt) > 10 && txt[:10] == "device_id=":
			deviceID = txt[10:]
		case len(txt) > 12 && txt[:12] == "device_name=":
			deviceName = txt[12:]
		case len(txt) > 8 && txt[:8] == "version=":
			version = txt[8:]
		}
	}

	// Skip if this is our own device
	if deviceID == d.deviceID {
		return
	}

	// Skip if missing required info
	if deviceID == "" || deviceName == "" {
		return
	}

	// Get IP address
	var host string
	if entry.AddrV4 != nil {
		host = entry.AddrV4.String()
	} else if entry.AddrV6 != nil {
		host = entry.AddrV6.String()
	} else {
		return
	}

	d.mu.Lock()
	peer, exists := d.peers[deviceID]
	if exists {
		// Update existing peer
		peer.Host = host
		peer.Port = entry.Port
		peer.Name = deviceName
		peer.Version = version
		peer.Status = models.PeerStatusOnline
		peer.LastSeen = time.Now()
	} else {
		// New peer discovered
		peer = &models.Peer{
			ID:       deviceID,
			Name:     deviceName,
			Host:     host,
			Port:     entry.Port,
			Version:  version,
			Status:   models.PeerStatusOnline,
			LastSeen: time.Now(),
			Paired:   false,
		}
		d.peers[deviceID] = peer
	}
	d.mu.Unlock()

	if !exists && d.onPeerFound != nil {
		log.Printf("mDNS: Discovered peer %s (%s) at %s:%d", deviceName, deviceID, host, entry.Port)
		d.onPeerFound(peer)
	}
}

// checkStalePeers marks peers as offline if not seen recently
func (d *Discovery) checkStalePeers() {
	d.mu.Lock()
	defer d.mu.Unlock()

	staleThreshold := time.Now().Add(-30 * time.Second)
	for id, peer := range d.peers {
		if peer.LastSeen.Before(staleThreshold) && peer.Status == models.PeerStatusOnline {
			peer.Status = models.PeerStatusOffline
			if d.onPeerLost != nil {
				log.Printf("mDNS: Peer %s (%s) went offline", peer.Name, id)
				go d.onPeerLost(id)
			}
		}
	}
}

// GetPeers returns all discovered peers
func (d *Discovery) GetPeers() []*models.Peer {
	d.mu.RLock()
	defer d.mu.RUnlock()

	peers := make([]*models.Peer, 0, len(d.peers))
	for _, p := range d.peers {
		peerCopy := *p
		peers = append(peers, &peerCopy)
	}
	return peers
}

// GetPeer returns a peer by ID
func (d *Discovery) GetPeer(id string) *models.Peer {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if p, ok := d.peers[id]; ok {
		peerCopy := *p
		return &peerCopy
	}
	return nil
}

// UpdatePeer updates a peer's information
func (d *Discovery) UpdatePeer(peer *models.Peer) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.peers[peer.ID] = peer
}

// getLocalIP returns the local IP address
func getLocalIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}
