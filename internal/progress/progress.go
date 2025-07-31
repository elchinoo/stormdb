// Package progress provides utilities for displaying progress during data seeding operations
package progress

import (
	"fmt"
	"strings"
	"time"
)

// Tracker represents a progress tracking instance
type Tracker struct {
	title      string
	total      int
	current    int
	startTime  time.Time
	width      int
	showETA    bool
	lastUpdate time.Time
}

// NewTracker creates a new progress tracker
func NewTracker(title string, total int) *Tracker {
	return &Tracker{
		title:     title,
		total:     total,
		current:   0,
		startTime: time.Now(),
		width:     50,
		showETA:   true,
	}
}

// SetWidth sets the width of the progress bar
func (p *Tracker) SetWidth(width int) *Tracker {
	p.width = width
	return p
}

// SetShowETA enables or disables ETA display
func (p *Tracker) SetShowETA(show bool) *Tracker {
	p.showETA = show
	return p
}

// Update updates the progress and displays the bar if enough time has passed
func (p *Tracker) Update(current int) {
	p.current = current

	// Only update display every 100ms to avoid overwhelming the terminal
	now := time.Now()
	if now.Sub(p.lastUpdate) >= 100*time.Millisecond || current == p.total {
		p.Display()
		p.lastUpdate = now
	}
}

// Increment increments the progress by 1
func (p *Tracker) Increment() {
	p.Update(p.current + 1)
}

// Add increments the progress by the specified amount
func (p *Tracker) Add(amount int) {
	p.Update(p.current + amount)
}

// Display shows the current progress bar
func (p *Tracker) Display() {
	if p.total <= 0 {
		return
	}

	percentage := float64(p.current) / float64(p.total) * 100
	filled := int(float64(p.width) * float64(p.current) / float64(p.total))

	// Build progress bar
	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)

	// Calculate elapsed time and ETA
	elapsed := time.Since(p.startTime)
	var eta string
	if p.showETA && p.current > 0 && p.current < p.total {
		totalTime := elapsed * time.Duration(p.total) / time.Duration(p.current)
		remaining := totalTime - elapsed
		eta = fmt.Sprintf(" ETA: %s", formatDuration(remaining))
	}

	// Rate calculation
	rate := float64(p.current) / elapsed.Seconds()
	rateStr := ""
	if rate >= 1 {
		rateStr = fmt.Sprintf(" (%.0f/s)", rate)
	} else if rate > 0 {
		rateStr = fmt.Sprintf(" (%.1f/s)", rate)
	}

	// Print progress line (use \r to overwrite the same line)
	fmt.Printf("\r%s: [%s] %d/%d (%.1f%%)%s%s",
		p.title, bar, p.current, p.total, percentage, rateStr, eta)

	// Print newline when complete
	if p.current >= p.total {
		totalTime := time.Since(p.startTime)
		fmt.Printf(" ✅ Completed in %s\n", formatDuration(totalTime))
	}
}

// Finish completes the progress and prints a newline
func (p *Tracker) Finish() {
	if p.current < p.total {
		p.current = p.total
	}
	p.Display()
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Nanoseconds())/1e6)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// BatchTracker tracks progress for operations with batches
type BatchTracker struct {
	*Tracker
	batchSize int
	batches   int
}

// NewBatchTracker creates a tracker for batch operations
func NewBatchTracker(title string, totalItems, batchSize int) *BatchTracker {
	batches := (totalItems + batchSize - 1) / batchSize // Ceiling division
	return &BatchTracker{
		Tracker:   NewTracker(title, totalItems),
		batchSize: batchSize,
		batches:   batches,
	}
}

// UpdateBatch updates progress after completing a batch
func (b *BatchTracker) UpdateBatch(completedBatches int) {
	completed := completedBatches * b.batchSize
	if completed > b.total {
		completed = b.total
	}
	b.Update(completed)
}

// IncrementBatch increments the completed batch count
func (b *BatchTracker) IncrementBatch() {
	completed := ((b.current / b.batchSize) + 1) * b.batchSize
	if completed > b.total {
		completed = b.total
	}
	b.Update(completed)
}

// GetBatchInfo returns current batch information
func (b *BatchTracker) GetBatchInfo() (currentBatch, totalBatches int) {
	currentBatch = (b.current + b.batchSize - 1) / b.batchSize
	return currentBatch, b.batches
}
