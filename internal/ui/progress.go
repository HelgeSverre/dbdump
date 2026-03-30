package ui

import (
	"fmt"
	"time"
)

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
