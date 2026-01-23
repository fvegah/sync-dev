package sync

import (
	"SyncDev/internal/models"
	"sync"
	"time"
)

const (
	// Throttle interval for emissions (~15 Hz)
	emitInterval = 66 * time.Millisecond

	// Exponential smoothing alpha (0.1 = smooth, 0.9 = responsive)
	smoothingAlpha = 0.1

	// Minimum progress percentage to calculate ETA (avoid division by small numbers)
	minProgressForETA = 0.05

	// Maximum active files to track
	maxActiveFiles = 10
)

// ProgressAggregator collects per-file progress and emits throttled aggregate updates
type ProgressAggregator struct {
	mu sync.RWMutex

	// Sync state
	status      string
	totalFiles  int
	totalBytes  int64
	startTime   time.Time

	// Progress tracking
	fileProgress   map[string]*fileState
	completedFiles int
	completedBytes int64

	// Speed smoothing
	smoothedSpeed   float64
	lastBytesUpdate int64
	lastUpdateTime  time.Time

	// Throttling
	lastEmitTime time.Time
	emitCallback func(*models.AggregateProgress)

	// Pending emission flag
	pendingEmit bool
	emitTimer   *time.Timer
}

// fileState tracks progress for a single file
type fileState struct {
	path        string
	size        int64
	transferred int64
	status      string // "active", "pending", "complete"
}

// NewProgressAggregator creates a new progress aggregator
func NewProgressAggregator(callback func(*models.AggregateProgress)) *ProgressAggregator {
	return &ProgressAggregator{
		status:       "idle",
		fileProgress: make(map[string]*fileState),
		emitCallback: callback,
	}
}

// StartSync initializes a new sync session
func (p *ProgressAggregator) StartSync(totalFiles int, totalBytes int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.status = "syncing"
	p.totalFiles = totalFiles
	p.totalBytes = totalBytes
	p.startTime = time.Now()
	p.lastUpdateTime = time.Now()

	p.fileProgress = make(map[string]*fileState)
	p.completedFiles = 0
	p.completedBytes = 0
	p.smoothedSpeed = 0
	p.lastBytesUpdate = 0
	p.pendingEmit = false

	p.emit()
}

// UpdateFile updates progress for a specific file
func (p *ProgressAggregator) UpdateFile(path string, size int64, transferred int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Get or create file state
	fs, exists := p.fileProgress[path]
	if !exists {
		fs = &fileState{
			path:   path,
			size:   size,
			status: "active",
		}
		p.fileProgress[path] = fs
	}

	// Update transferred bytes
	oldTransferred := fs.transferred
	fs.transferred = transferred
	fs.status = "active"

	// Update smoothed speed
	byteDelta := transferred - oldTransferred
	if byteDelta > 0 {
		p.updateSpeed(byteDelta)
	}

	// Check if file is complete
	if transferred >= size && size > 0 {
		fs.status = "complete"
	}

	// Throttled emit
	p.scheduleEmit()
}

// CompleteFile marks a file as complete and updates counters
func (p *ProgressAggregator) CompleteFile(path string, size int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	fs, exists := p.fileProgress[path]
	if exists {
		fs.status = "complete"
		fs.transferred = fs.size
	} else {
		p.fileProgress[path] = &fileState{
			path:        path,
			size:        size,
			transferred: size,
			status:      "complete",
		}
	}

	p.completedFiles++
	p.completedBytes += size

	// Force emit on completion
	p.emit()
}

// EndSync finalizes the sync session
func (p *ProgressAggregator) EndSync() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.status = "complete"

	// Cancel any pending emit timer
	if p.emitTimer != nil {
		p.emitTimer.Stop()
		p.emitTimer = nil
	}

	p.emit()
}

// GetProgress returns the current aggregate progress
func (p *ProgressAggregator) GetProgress() *models.AggregateProgress {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.buildProgress()
}

// Reset resets the aggregator to idle state
func (p *ProgressAggregator) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Cancel any pending emit timer
	if p.emitTimer != nil {
		p.emitTimer.Stop()
		p.emitTimer = nil
	}

	p.status = "idle"
	p.totalFiles = 0
	p.totalBytes = 0
	p.fileProgress = make(map[string]*fileState)
	p.completedFiles = 0
	p.completedBytes = 0
	p.smoothedSpeed = 0
	p.lastBytesUpdate = 0
	p.pendingEmit = false
}

// updateSpeed applies exponential smoothing to speed calculation
func (p *ProgressAggregator) updateSpeed(bytesDelta int64) {
	now := time.Now()
	elapsed := now.Sub(p.lastUpdateTime).Seconds()

	if elapsed > 0 {
		instantSpeed := float64(bytesDelta) / elapsed
		if p.smoothedSpeed == 0 {
			// First update - use instant speed
			p.smoothedSpeed = instantSpeed
		} else {
			// Exponential smoothing: new = alpha * instant + (1-alpha) * old
			p.smoothedSpeed = smoothingAlpha*instantSpeed + (1-smoothingAlpha)*p.smoothedSpeed
		}
	}

	p.lastBytesUpdate += bytesDelta
	p.lastUpdateTime = now
}

// scheduleEmit schedules a throttled emit
func (p *ProgressAggregator) scheduleEmit() {
	now := time.Now()
	timeSinceLastEmit := now.Sub(p.lastEmitTime)

	// If enough time has passed, emit immediately
	if timeSinceLastEmit >= emitInterval {
		p.emit()
		return
	}

	// Otherwise, schedule a delayed emit if not already pending
	if !p.pendingEmit {
		p.pendingEmit = true
		delay := emitInterval - timeSinceLastEmit

		p.emitTimer = time.AfterFunc(delay, func() {
			p.mu.Lock()
			defer p.mu.Unlock()

			p.pendingEmit = false
			p.emit()
		})
	}
}

// emit sends the current progress to the callback (must hold lock)
func (p *ProgressAggregator) emit() {
	if p.emitCallback == nil {
		return
	}

	p.lastEmitTime = time.Now()
	progress := p.buildProgress()

	// Call callback outside the lock to prevent deadlocks
	go p.emitCallback(progress)
}

// buildProgress constructs the AggregateProgress struct (must hold read lock)
func (p *ProgressAggregator) buildProgress() *models.AggregateProgress {
	// Calculate total transferred bytes
	var totalTransferred int64
	activeFiles := make([]models.FileProgress, 0, maxActiveFiles)
	activeCount := 0

	for _, fs := range p.fileProgress {
		totalTransferred += fs.transferred

		// Add to active files list (up to max)
		if activeCount < maxActiveFiles && fs.status == "active" {
			percentage := float64(0)
			if fs.size > 0 {
				percentage = float64(fs.transferred) / float64(fs.size) * 100
			}
			activeFiles = append(activeFiles, models.FileProgress{
				Path:        fs.path,
				Size:        fs.size,
				Transferred: fs.transferred,
				Percentage:  percentage,
				Status:      fs.status,
			})
			activeCount++
		}
	}

	// Add completed bytes
	totalTransferred += p.completedBytes

	// Calculate overall percentage
	var percentage float64
	if p.totalBytes > 0 {
		percentage = float64(totalTransferred) / float64(p.totalBytes) * 100
	}

	// Calculate ETA
	var eta int64 = -1
	if p.smoothedSpeed > 0 && percentage >= minProgressForETA*100 {
		remainingBytes := p.totalBytes - totalTransferred
		if remainingBytes > 0 {
			eta = int64(float64(remainingBytes) / p.smoothedSpeed)
		} else {
			eta = 0
		}
	}

	return &models.AggregateProgress{
		Status:           p.status,
		TotalFiles:       p.totalFiles,
		CompletedFiles:   p.completedFiles,
		TotalBytes:       p.totalBytes,
		TransferredBytes: totalTransferred,
		Percentage:       percentage,
		BytesPerSecond:   p.smoothedSpeed,
		ETA:              eta,
		ActiveFiles:      activeFiles,
	}
}
