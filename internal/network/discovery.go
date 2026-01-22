package network

import (
	"SyncDev/internal/config"
	"SyncDev/internal/models"
	"context"
	"fmt"
	"log"
	"net"
	"strings"
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
	// Get all local IPs
	localIPs := getAllLocalIPs()
	if len(localIPs) == 0 {
		return fmt.Errorf("no local IPs found")
	}

	log.Printf("mDNS: Found %d local IPs: %v", len(localIPs), localIPs)

	// Create TXT records with device info
	txtRecords := []string{
		fmt.Sprintf("device_id=%s", d.deviceID),
		fmt.Sprintf("device_name=%s", d.deviceName),
		fmt.Sprintf("version=%s", config.AppVersion),
	}

	log.Printf("mDNS: Creating service with ID=%s, Name=%s, Port=%d",
		d.deviceID, d.deviceName, d.port)
	log.Printf("mDNS: TXT records: %v", txtRecords)

	// Create mDNS service with all local IPs
	service, err := mdns.NewMDNSService(
		d.deviceID,
		config.ServiceName,
		"",
		"",
		d.port,
		localIPs,
		txtRecords,
	)
	if err != nil {
		return fmt.Errorf("failed to create mDNS service: %w", err)
	}

	// Try without interface binding first (let the library choose)
	serverConfig := &mdns.Config{Zone: service}

	// Try to get the active interface, but don't fail if we can't
	iface, _ := getActiveInterface()
	if iface != nil {
		log.Printf("mDNS: Preferred interface: %s", iface.Name)
		// Note: We're NOT binding to interface to allow broader advertising
	}

	server, err := mdns.NewServer(serverConfig)
	if err != nil {
		return fmt.Errorf("failed to start mDNS server: %w", err)
	}

	d.server = server
	log.Printf("mDNS: Advertising as %s on port %d (IPs: %v)", d.deviceName, d.port, localIPs)
	return nil
}

// getAllLocalIPs returns all non-loopback local IPs
func getAllLocalIPs() []net.IP {
	var ips []net.IP

	interfaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Only include IPv4 addresses
			if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
				ips = append(ips, ip)
				log.Printf("mDNS: Found IP %s on interface %s", ip.String(), iface.Name)
			}
		}
	}

	return ips
}

// scanLoop periodically scans for peers
func (d *Discovery) scanLoop() {
	// Do aggressive initial scans (3 scans in first 10 seconds)
	for i := 0; i < 3; i++ {
		select {
		case <-d.ctx.Done():
			return
		default:
			log.Printf("mDNS: Initial scan %d/3", i+1)
			d.scan()
			time.Sleep(3 * time.Second)
		}
	}

	// Then switch to regular interval
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

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

// scan performs a single mDNS scan on all available interfaces
func (d *Discovery) scan() {
	// Get all multicast interfaces
	interfaces := getAllMulticastInterfaces()
	if len(interfaces) == 0 {
		log.Printf("mDNS: No multicast interfaces found, trying without interface binding")
		d.scanOnInterface(nil)
		return
	}

	log.Printf("mDNS: Scanning on %d interfaces", len(interfaces))

	// Scan on each interface
	for _, iface := range interfaces {
		log.Printf("mDNS: Scanning on interface %s", iface.Name)
		d.scanOnInterface(iface)
	}

	// Also try without interface binding as fallback
	d.scanOnInterface(nil)
}

// scanOnInterface performs a scan on a specific interface
func (d *Discovery) scanOnInterface(iface *net.Interface) {
	entriesCh := make(chan *mdns.ServiceEntry, 10)
	go func() {
		for entry := range entriesCh {
			d.handleServiceEntry(entry)
		}
	}()

	ifaceName := "default"
	if iface != nil {
		ifaceName = iface.Name
	}

	params := &mdns.QueryParam{
		Service:             config.ServiceName,
		Domain:              "local",
		Timeout:             3 * time.Second,
		Entries:             entriesCh,
		WantUnicastResponse: false, // Use multicast
		Interface:           iface,
	}

	if err := mdns.Query(params); err != nil {
		log.Printf("mDNS: Query error on %s: %v", ifaceName, err)
	}
	close(entriesCh)
}

// getActiveInterface returns the active network interface (en0 for WiFi/Ethernet)
func getActiveInterface() (*net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	// Prefer en0 (primary interface on macOS)
	for _, iface := range interfaces {
		if iface.Name == "en0" && iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagMulticast != 0 {
			return &iface, nil
		}
	}

	// Try en1 (secondary interface, common on some Macs)
	for _, iface := range interfaces {
		if iface.Name == "en1" && iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagMulticast != 0 {
			return &iface, nil
		}
	}

	// Fallback to any active interface with multicast support
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 &&
			iface.Flags&net.FlagMulticast != 0 &&
			iface.Flags&net.FlagLoopback == 0 {
			return &iface, nil
		}
	}

	return nil, nil // Let the library choose
}

// getAllMulticastInterfaces returns all interfaces that support multicast
func getAllMulticastInterfaces() []*net.Interface {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	var result []*net.Interface
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 &&
			iface.Flags&net.FlagMulticast != 0 &&
			iface.Flags&net.FlagLoopback == 0 {
			ifaceCopy := iface
			result = append(result, &ifaceCopy)
		}
	}
	return result
}

// handleServiceEntry processes a discovered service
func (d *Discovery) handleServiceEntry(entry *mdns.ServiceEntry) {
	log.Printf("mDNS: Processing entry - Name: %s, Host: %s, Port: %d, AddrV4: %v, InfoFields: %v, Info: %s",
		entry.Name, entry.Host, entry.Port, entry.AddrV4, entry.InfoFields, entry.Info)

	// Parse TXT records
	deviceID := ""
	deviceName := ""
	version := ""

	// Try InfoFields first
	for _, txt := range entry.InfoFields {
		if strings.HasPrefix(txt, "device_id=") {
			deviceID = strings.TrimPrefix(txt, "device_id=")
		} else if strings.HasPrefix(txt, "device_name=") {
			deviceName = strings.TrimPrefix(txt, "device_name=")
		} else if strings.HasPrefix(txt, "version=") {
			version = strings.TrimPrefix(txt, "version=")
		}
	}

	// Fallback: parse Info field if InfoFields is empty
	if deviceID == "" && entry.Info != "" {
		for _, part := range strings.Split(entry.Info, ",") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "device_id=") {
				deviceID = strings.TrimPrefix(part, "device_id=")
			} else if strings.HasPrefix(part, "device_name=") {
				deviceName = strings.TrimPrefix(part, "device_name=")
			} else if strings.HasPrefix(part, "version=") {
				version = strings.TrimPrefix(part, "version=")
			}
		}
	}

	// If still no deviceID, try using the entry Name (which is the instance name)
	if deviceID == "" && entry.Name != "" {
		// The instance name might be the device ID
		deviceID = entry.Name
		if deviceName == "" {
			deviceName = entry.Host
		}
	}

	log.Printf("mDNS: Parsed - deviceID: %s, deviceName: %s, version: %s", deviceID, deviceName, version)

	// Skip if this is our own device
	if deviceID == d.deviceID {
		log.Printf("mDNS: Skipping own device %s", deviceID)
		return
	}

	// Skip if missing required info
	if deviceID == "" {
		log.Printf("mDNS: Skipping entry with no deviceID")
		return
	}

	// Use hostname as device name if not provided
	if deviceName == "" {
		deviceName = entry.Host
		if deviceName == "" {
			deviceName = "Unknown Mac"
		}
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
