package sync

import (
	"SyncDev/internal/models"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// IndexManager manages file indices for folder pairs
type IndexManager struct {
	indexDir string
	indices  map[string]*models.FileIndex
	mu       sync.RWMutex
}

// NewIndexManager creates a new IndexManager
func NewIndexManager(indexDir string) (*IndexManager, error) {
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create index directory: %w", err)
	}

	return &IndexManager{
		indexDir: indexDir,
		indices:  make(map[string]*models.FileIndex),
	}, nil
}

// LoadIndex loads an index from disk
func (im *IndexManager) LoadIndex(folderPairID string) (*models.FileIndex, error) {
	im.mu.Lock()
	defer im.mu.Unlock()

	// Check cache first
	if idx, ok := im.indices[folderPairID]; ok {
		return idx, nil
	}

	// Load from disk
	path := im.getIndexPath(folderPairID)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read index file: %w", err)
	}

	var index models.FileIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse index file: %w", err)
	}

	im.indices[folderPairID] = &index
	return &index, nil
}

// SaveIndex saves an index to disk
func (im *IndexManager) SaveIndex(folderPairID string, index *models.FileIndex) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	index.UpdatedAt = time.Now()
	im.indices[folderPairID] = index

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	path := im.getIndexPath(folderPairID)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}

// DeleteIndex deletes an index from disk and cache
func (im *IndexManager) DeleteIndex(folderPairID string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	delete(im.indices, folderPairID)
	path := im.getIndexPath(folderPairID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// getIndexPath returns the path for an index file
func (im *IndexManager) getIndexPath(folderPairID string) string {
	return filepath.Join(im.indexDir, folderPairID+".json")
}

// CompareIndices compares local and remote indices to determine sync actions
func CompareIndices(local, remote *models.FileIndex) []*models.SyncAction {
	var actions []*models.SyncAction

	// Build a set of all paths
	allPaths := make(map[string]bool)
	if local != nil {
		for path := range local.Files {
			allPaths[path] = true
		}
	}
	if remote != nil {
		for path := range remote.Files {
			allPaths[path] = true
		}
	}

	// Compare each path
	for path := range allPaths {
		var localFile, remoteFile *models.FileInfo
		if local != nil {
			localFile = local.Files[path]
		}
		if remote != nil {
			remoteFile = remote.Files[path]
		}

		action := compareFiles(path, localFile, remoteFile)
		if action != nil {
			actions = append(actions, action)
		}
	}

	return actions
}

// compareFiles compares two file infos and returns the appropriate action
func compareFiles(path string, local, remote *models.FileInfo) *models.SyncAction {
	// Only exists locally -> push to remote
	if local != nil && remote == nil {
		return &models.SyncAction{
			Action:    models.FileActionPush,
			LocalFile: local,
			Reason:    "File only exists locally",
		}
	}

	// Only exists remotely -> pull from remote
	if local == nil && remote != nil {
		return &models.SyncAction{
			Action:     models.FileActionPull,
			RemoteFile: remote,
			Reason:     "File only exists on remote",
		}
	}

	// Both exist - compare
	if local != nil && remote != nil {
		// Directories don't need content comparison
		if local.IsDir && remote.IsDir {
			return nil
		}

		// If one is a dir and one is a file, that's a conflict
		// For now, use last-write-wins
		if local.IsDir != remote.IsDir {
			if local.ModTime.After(remote.ModTime) {
				return &models.SyncAction{
					Action:     models.FileActionPush,
					LocalFile:  local,
					RemoteFile: remote,
					Reason:     "Type conflict, local is newer",
				}
			}
			return &models.SyncAction{
				Action:     models.FileActionPull,
				LocalFile:  local,
				RemoteFile: remote,
				Reason:     "Type conflict, remote is newer",
			}
		}

		// Compare by hash if available
		if local.Hash != "" && remote.Hash != "" {
			if local.Hash == remote.Hash {
				return nil // Files are identical
			}
		} else {
			// Fallback to size and modtime
			if local.Size == remote.Size && local.ModTime.Equal(remote.ModTime) {
				return nil // Probably identical
			}
		}

		// Files differ - use last-write-wins
		if local.ModTime.After(remote.ModTime) {
			return &models.SyncAction{
				Action:     models.FileActionPush,
				LocalFile:  local,
				RemoteFile: remote,
				Reason:     "Local file is newer",
			}
		} else if remote.ModTime.After(local.ModTime) {
			return &models.SyncAction{
				Action:     models.FileActionPull,
				LocalFile:  local,
				RemoteFile: remote,
				Reason:     "Remote file is newer",
			}
		}

		// Same modtime but different content - this shouldn't happen
		// but if it does, don't do anything to avoid data loss
		return nil
	}

	return nil
}

// MergeIndex merges updates into an existing index
func MergeIndex(existing *models.FileIndex, updates map[string]*models.FileInfo) *models.FileIndex {
	if existing == nil {
		existing = &models.FileIndex{
			Files: make(map[string]*models.FileInfo),
		}
	}

	for path, info := range updates {
		if info == nil {
			delete(existing.Files, path)
		} else {
			existing.Files[path] = info
		}
	}

	existing.UpdatedAt = time.Now()
	return existing
}
