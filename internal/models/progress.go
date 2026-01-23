package models

// AggregateProgress represents overall sync progress across all files
type AggregateProgress struct {
	Status           string         `json:"status"`           // "idle", "syncing", "complete"
	TotalFiles       int            `json:"totalFiles"`
	CompletedFiles   int            `json:"completedFiles"`
	TotalBytes       int64          `json:"totalBytes"`
	TransferredBytes int64          `json:"transferredBytes"`
	Percentage       float64        `json:"percentage"`
	BytesPerSecond   float64        `json:"bytesPerSecond"`   // smoothed speed
	ETA              int64          `json:"eta"`              // seconds remaining, -1 if unknown
	ActiveFiles      []FileProgress `json:"activeFiles"`      // max 10 files
}

// FileProgress represents progress for a single file
type FileProgress struct {
	Path        string  `json:"path"`
	Size        int64   `json:"size"`
	Transferred int64   `json:"transferred"`
	Percentage  float64 `json:"percentage"`
	Status      string  `json:"status"` // "active", "pending", "complete"
}
