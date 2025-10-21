package ui

import (
	"fmt"
	"time"

	"github.com/schollz/progressbar/v3"
)

// ProgressTracker tracks progress during dump operations
type ProgressTracker struct {
	bar *progressbar.ProgressBar
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(description string, max int64) *ProgressTracker {
	bar := progressbar.NewOptions64(
		max,
		progressbar.OptionSetDescription(description),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)

	return &ProgressTracker{bar: bar}
}

// NewSimpleProgress creates a simple progress bar without byte display
func NewSimpleProgress(description string, max int) *ProgressTracker {
	bar := progressbar.NewOptions(
		max,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)

	return &ProgressTracker{bar: bar}
}

// Add increments the progress bar
func (p *ProgressTracker) Add(n int) error {
	return p.bar.Add(n)
}

// Add64 increments the progress bar with int64
func (p *ProgressTracker) Add64(n int64) error {
	return p.bar.Add64(n)
}

// Finish completes the progress bar
func (p *ProgressTracker) Finish() error {
	return p.bar.Finish()
}

// Clear clears the progress bar from the screen
func (p *ProgressTracker) Clear() error {
	return p.bar.Clear()
}

// PrintSummary prints a summary after the dump
func PrintSummary(outputFile string, excludedCount int, duration time.Duration, fileSize string) {
	fmt.Println()
	fmt.Printf("✓ Dump complete: %s (%s)\n", outputFile, fileSize)
	if excludedCount > 0 {
		fmt.Printf("✓ Excluded %d table(s) (data only, structure preserved)\n", excludedCount)
	}
	fmt.Printf("✓ Duration: %s\n", duration.Round(time.Second))
	fmt.Println()
}

// PrintError prints an error message
func PrintError(err error) {
	fmt.Printf("\n✗ Error: %s\n\n", err)
}

// PrintInfo prints an informational message
func PrintInfo(message string) {
	fmt.Printf("ℹ %s\n", message)
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	fmt.Printf("✓ %s\n", message)
}
