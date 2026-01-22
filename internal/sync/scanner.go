package sync

import (
	"SyncDev/internal/models"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gobwas/glob"
)

// Scanner scans directories and builds file indices
type Scanner struct {
	exclusions []glob.Glob
}

// NewScanner creates a new Scanner with the given exclusion patterns
func NewScanner(exclusionPatterns []string) *Scanner {
	var globs []glob.Glob
	for _, pattern := range exclusionPatterns {
		if g, err := glob.Compile(pattern); err == nil {
			globs = append(globs, g)
		}
	}
	return &Scanner{
		exclusions: globs,
	}
}

// ScanDirectory scans a directory and returns a file index
func (s *Scanner) ScanDirectory(rootPath string) (*models.FileIndex, error) {
	index := &models.FileIndex{
		FolderPath: rootPath,
		Files:      make(map[string]*models.FileInfo),
		UpdatedAt:  time.Now(),
	}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files we can't access
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return nil
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Check exclusions
		if s.isExcluded(relPath, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		fileInfo := &models.FileInfo{
			Path:       relPath,
			Size:       info.Size(),
			ModTime:    info.ModTime(),
			IsDir:      info.IsDir(),
			Permission: uint32(info.Mode().Perm()),
		}

		// Calculate hash for files (not directories)
		if !info.IsDir() {
			hash, err := s.calculateHash(path)
			if err != nil {
				// Skip files we can't hash
				return nil
			}
			fileInfo.Hash = hash
		}

		index.Files[relPath] = fileInfo
		return nil
	})

	if err != nil {
		return nil, err
	}

	return index, nil
}

// isExcluded checks if a path matches any exclusion pattern
func (s *Scanner) isExcluded(path string, isDir bool) bool {
	// Normalize path separators
	path = filepath.ToSlash(path)
	name := filepath.Base(path)

	for _, g := range s.exclusions {
		// Check both full path and basename
		if g.Match(path) || g.Match(name) {
			return true
		}
	}

	// Also check for hidden files on macOS (starting with .)
	if strings.HasPrefix(name, ".") {
		// Allow .gitignore and similar files to be synced if not explicitly excluded
		// But exclude .DS_Store, .git, etc.
		hiddenExclusions := []string{".DS_Store", ".git", ".svn", ".hg", ".Spotlight-V100", ".Trashes", ".fseventsd"}
		for _, he := range hiddenExclusions {
			if name == he {
				return true
			}
		}
	}

	return false
}

// calculateHash calculates the SHA256 hash of a file
func (s *Scanner) calculateHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// QuickScan performs a quick scan using only file metadata (no hashing)
func (s *Scanner) QuickScan(rootPath string) (*models.FileIndex, error) {
	index := &models.FileIndex{
		FolderPath: rootPath,
		Files:      make(map[string]*models.FileInfo),
		UpdatedAt:  time.Now(),
	}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return nil
		}

		if relPath == "." {
			return nil
		}

		if s.isExcluded(relPath, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		index.Files[relPath] = &models.FileInfo{
			Path:       relPath,
			Size:       info.Size(),
			ModTime:    info.ModTime(),
			IsDir:      info.IsDir(),
			Permission: uint32(info.Mode().Perm()),
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return index, nil
}

// HashFile calculates the hash for a single file
func (s *Scanner) HashFile(path string) (string, error) {
	return s.calculateHash(path)
}

// GetFileInfo gets the FileInfo for a single file
func (s *Scanner) GetFileInfo(rootPath, relPath string) (*models.FileInfo, error) {
	fullPath := filepath.Join(rootPath, relPath)
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	fileInfo := &models.FileInfo{
		Path:       relPath,
		Size:       info.Size(),
		ModTime:    info.ModTime(),
		IsDir:      info.IsDir(),
		Permission: uint32(info.Mode().Perm()),
	}

	if !info.IsDir() {
		hash, err := s.calculateHash(fullPath)
		if err != nil {
			return nil, err
		}
		fileInfo.Hash = hash
	}

	return fileInfo, nil
}
