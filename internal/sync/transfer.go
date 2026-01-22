package sync

import (
	"SyncDev/internal/models"
	"SyncDev/internal/network"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// TransferManager handles file transfers between peers
type TransferManager struct {
	rootPath string
	scanner  *Scanner
}

// NewTransferManager creates a new TransferManager
func NewTransferManager(rootPath string, scanner *Scanner) *TransferManager {
	return &TransferManager{
		rootPath: rootPath,
		scanner:  scanner,
	}
}

// SendFile sends a file to a peer in chunks
func (tm *TransferManager) SendFile(conn *network.PeerConnection, folderPairID, relPath string, progressCb func(*models.TransferProgress)) error {
	fullPath := filepath.Join(tm.rootPath, relPath)

	file, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	totalSize := info.Size()
	var transferred int64
	startTime := time.Now()

	buffer := make([]byte, network.ChunkSize)
	offset := int64(0)

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read file: %w", err)
		}

		if n == 0 {
			break
		}

		isLast := n < network.ChunkSize || err == io.EOF

		chunk := &network.FileChunkPayload{
			FolderPairID: folderPairID,
			FilePath:     relPath,
			Offset:       offset,
			Data:         base64Encode(buffer[:n]),
			IsLast:       isLast,
		}

		msg, err := network.NewMessage(network.MsgTypeFileChunk, chunk)
		if err != nil {
			return err
		}

		if err := conn.WriteMessage(msg); err != nil {
			return fmt.Errorf("failed to send chunk: %w", err)
		}

		transferred += int64(n)
		offset += int64(n)

		if progressCb != nil {
			elapsed := time.Since(startTime).Seconds()
			var bytesPerSec int64
			if elapsed > 0 {
				bytesPerSec = int64(float64(transferred) / elapsed)
			}

			progressCb(&models.TransferProgress{
				FileName:       relPath,
				TotalBytes:     totalSize,
				TransferBytes:  transferred,
				Percentage:     float64(transferred) / float64(totalSize) * 100,
				BytesPerSecond: bytesPerSec,
			})
		}

		if isLast {
			break
		}
	}

	return nil
}

// ReceiveFile receives file chunks and writes them to disk
type FileReceiver struct {
	rootPath     string
	tempPath     string
	file         *os.File
	expectedSize int64
	received     int64
	progressCb   func(*models.TransferProgress)
	startTime    time.Time
	filePath     string
}

// NewFileReceiver creates a new FileReceiver
func NewFileReceiver(rootPath, relPath string, expectedSize int64, progressCb func(*models.TransferProgress)) (*FileReceiver, error) {
	fullPath := filepath.Join(rootPath, relPath)
	tempPath := fullPath + ".syncdev.tmp"

	// Create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	file, err := os.Create(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	return &FileReceiver{
		rootPath:     rootPath,
		tempPath:     tempPath,
		file:         file,
		expectedSize: expectedSize,
		progressCb:   progressCb,
		startTime:    time.Now(),
		filePath:     relPath,
	}, nil
}

// WriteChunk writes a chunk of data to the file
func (fr *FileReceiver) WriteChunk(data []byte, offset int64) error {
	decoded, err := base64Decode(data)
	if err != nil {
		return fmt.Errorf("failed to decode chunk: %w", err)
	}

	if _, err := fr.file.WriteAt(decoded, offset); err != nil {
		return fmt.Errorf("failed to write chunk: %w", err)
	}

	fr.received += int64(len(decoded))

	if fr.progressCb != nil {
		elapsed := time.Since(fr.startTime).Seconds()
		var bytesPerSec int64
		if elapsed > 0 {
			bytesPerSec = int64(float64(fr.received) / elapsed)
		}

		fr.progressCb(&models.TransferProgress{
			FileName:       fr.filePath,
			TotalBytes:     fr.expectedSize,
			TransferBytes:  fr.received,
			Percentage:     float64(fr.received) / float64(fr.expectedSize) * 100,
			BytesPerSecond: bytesPerSec,
		})
	}

	return nil
}

// Finalize completes the file transfer
func (fr *FileReceiver) Finalize() error {
	if err := fr.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	if err := fr.file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	// Rename temp file to final path
	finalPath := filepath.Join(fr.rootPath, fr.filePath)
	if err := os.Rename(fr.tempPath, finalPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Abort cancels the file transfer and cleans up
func (fr *FileReceiver) Abort() {
	fr.file.Close()
	os.Remove(fr.tempPath)
}

// CopyFile copies a file locally (for local sync operations)
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Preserve file permissions
	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Preserve modification time
	if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return err
	}

	return nil
}

// DeleteFile deletes a file or directory
func DeleteFile(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil // Already deleted
	}
	if err != nil {
		return err
	}

	if info.IsDir() {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}

// CreateDirectory creates a directory with the specified permissions
func CreateDirectory(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// Helper functions for base64 encoding/decoding
func base64Encode(data []byte) []byte {
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(encoded, data)
	return encoded
}

func base64Decode(data []byte) ([]byte, error) {
	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	n, err := base64.StdEncoding.Decode(decoded, data)
	if err != nil {
		return nil, err
	}
	return decoded[:n], nil
}
