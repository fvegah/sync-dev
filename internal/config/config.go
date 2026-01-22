package config

import (
	"SyncDev/internal/models"
	"time"
)

const (
	DefaultPort         = 52525
	DefaultSyncInterval = 5 * time.Minute
	ServiceName         = "_syncdev._tcp"
	AppVersion          = "1.0.0"
)

// Config represents the application configuration
type Config struct {
	DeviceID          string              `json:"deviceId"`
	DeviceName        string              `json:"deviceName"`
	Port              int                 `json:"port"`
	SyncIntervalMins  int                 `json:"syncIntervalMins"`
	GlobalExclusions  []string            `json:"globalExclusions"`
	Peers             []*models.Peer      `json:"peers"`
	FolderPairs       []*models.FolderPair `json:"folderPairs"`
	AutoSync          bool                `json:"autoSync"`
	StartOnLogin      bool                `json:"startOnLogin"`
	ShowNotifications bool                `json:"showNotifications"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Port:             DefaultPort,
		SyncIntervalMins: 5,
		GlobalExclusions: []string{
			".DS_Store",
			".git",
			".svn",
			"node_modules",
			"*.tmp",
			"*.swp",
			"*~",
			".Trash",
			"Thumbs.db",
		},
		Peers:             []*models.Peer{},
		FolderPairs:       []*models.FolderPair{},
		AutoSync:          true,
		ShowNotifications: true,
	}
}

// GetPeer returns a peer by ID
func (c *Config) GetPeer(id string) *models.Peer {
	for _, p := range c.Peers {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// GetPairedPeers returns all paired peers
func (c *Config) GetPairedPeers() []*models.Peer {
	var paired []*models.Peer
	for _, p := range c.Peers {
		if p.Paired {
			paired = append(paired, p)
		}
	}
	return paired
}

// AddPeer adds or updates a peer
func (c *Config) AddPeer(peer *models.Peer) {
	for i, p := range c.Peers {
		if p.ID == peer.ID {
			c.Peers[i] = peer
			return
		}
	}
	c.Peers = append(c.Peers, peer)
}

// RemovePeer removes a peer by ID
func (c *Config) RemovePeer(id string) {
	for i, p := range c.Peers {
		if p.ID == id {
			c.Peers = append(c.Peers[:i], c.Peers[i+1:]...)
			return
		}
	}
}

// GetFolderPair returns a folder pair by ID
func (c *Config) GetFolderPair(id string) *models.FolderPair {
	for _, fp := range c.FolderPairs {
		if fp.ID == id {
			return fp
		}
	}
	return nil
}

// GetFolderPairsForPeer returns all folder pairs for a specific peer
func (c *Config) GetFolderPairsForPeer(peerID string) []*models.FolderPair {
	var pairs []*models.FolderPair
	for _, fp := range c.FolderPairs {
		if fp.PeerID == peerID {
			pairs = append(pairs, fp)
		}
	}
	return pairs
}

// AddFolderPair adds or updates a folder pair
func (c *Config) AddFolderPair(pair *models.FolderPair) {
	for i, fp := range c.FolderPairs {
		if fp.ID == pair.ID {
			c.FolderPairs[i] = pair
			return
		}
	}
	c.FolderPairs = append(c.FolderPairs, pair)
}

// RemoveFolderPair removes a folder pair by ID
func (c *Config) RemoveFolderPair(id string) {
	for i, fp := range c.FolderPairs {
		if fp.ID == id {
			c.FolderPairs = append(c.FolderPairs[:i], c.FolderPairs[i+1:]...)
			return
		}
	}
}
