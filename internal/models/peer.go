package models

import (
	"time"
)

// PeerStatus represents the current status of a peer
type PeerStatus string

const (
	PeerStatusOnline   PeerStatus = "online"
	PeerStatusOffline  PeerStatus = "offline"
	PeerStatusSyncing  PeerStatus = "syncing"
	PeerStatusPairing  PeerStatus = "pairing"
)

// Peer represents a remote device that can sync with this instance
type Peer struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Host         string     `json:"host"`
	Port         int        `json:"port"`
	Version      string     `json:"version"`
	Status       PeerStatus `json:"status"`
	SharedSecret string     `json:"sharedSecret,omitempty"`
	Paired       bool       `json:"paired"`
	LastSeen     time.Time  `json:"lastSeen"`
	LastSyncTime time.Time  `json:"lastSyncTime,omitempty"`
}

// PairingRequest represents a request to pair two devices
type PairingRequest struct {
	FromPeerID   string `json:"fromPeerId"`
	FromPeerName string `json:"fromPeerName"`
	Code         string `json:"code"`
	Timestamp    int64  `json:"timestamp"`
}

// PairingResponse represents the response to a pairing request
type PairingResponse struct {
	Accepted     bool   `json:"accepted"`
	SharedSecret string `json:"sharedSecret,omitempty"`
	Error        string `json:"error,omitempty"`
}
