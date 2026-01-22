package models

import (
	"time"
)

// FileInfo represents metadata about a file in the sync index
type FileInfo struct {
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	ModTime    time.Time `json:"modTime"`
	Hash       string    `json:"hash"`
	IsDir      bool      `json:"isDir"`
	Permission uint32    `json:"permission"`
}

// FileAction represents the type of action to take during sync
type FileAction string

const (
	FileActionPush   FileAction = "push"
	FileActionPull   FileAction = "pull"
	FileActionDelete FileAction = "delete"
	FileActionSkip   FileAction = "skip"
)

// SyncAction represents a single file operation to perform
type SyncAction struct {
	Action     FileAction `json:"action"`
	LocalFile  *FileInfo  `json:"localFile,omitempty"`
	RemoteFile *FileInfo  `json:"remoteFile,omitempty"`
	Reason     string     `json:"reason"`
}

// FileIndex represents the index of files in a synced folder
type FileIndex struct {
	FolderPath string               `json:"folderPath"`
	Files      map[string]*FileInfo `json:"files"`
	UpdatedAt  time.Time            `json:"updatedAt"`
}

// TransferProgress represents the progress of a file transfer
type TransferProgress struct {
	FileName       string  `json:"fileName"`
	TotalBytes     int64   `json:"totalBytes"`
	TransferBytes  int64   `json:"transferBytes"`
	Percentage     float64 `json:"percentage"`
	BytesPerSecond int64   `json:"bytesPerSecond"`
}

// FolderPair represents a pair of folders to sync between local and remote
type FolderPair struct {
	ID           string   `json:"id"`
	PeerID       string   `json:"peerId"`
	LocalPath    string   `json:"localPath"`
	RemotePath   string   `json:"remotePath"`
	Enabled      bool     `json:"enabled"`
	Exclusions   []string `json:"exclusions"`
	LastSyncTime time.Time `json:"lastSyncTime,omitempty"`
}
