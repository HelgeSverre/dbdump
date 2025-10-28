package database

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// DumpOptions contains options for dumping the database
type DumpOptions struct {
	Connection    *Connection
	ExcludeTables []string
	OutputFile    string
	ShowProgress  bool
	DryRun        bool
}

// Dumper handles database dumping operations
type Dumper struct {
	options *DumpOptions
}

// NewDumper creates a new Dumper
func NewDumper(options *DumpOptions) *Dumper {
	return &Dumper{options: options}
}

// DumpResult contains the result of a dump operation
type DumpResult struct {
	OutputFile      string
	Duration        time.Duration
	ExcludedTables  []string
	FileSize        int64
	FileSizeDisplay string
}

// Dump performs the database dump
func (d *Dumper) Dump() (*DumpResult, error) {
	startTime := time.Now()

	if d.options.DryRun {
		return d.dryRun()
	}

	// Create output file with restrictive permissions (owner read/write only)
	outFile, err := os.OpenFile(d.options.OutputFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if err := outFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close output file: %v\n", err)
		}
	}()

	// Use 256KB buffer for optimal write performance
	writer := bufio.NewWriterSize(outFile, 256*1024)
	defer func() {
		if err := writer.Flush(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to flush writer: %v\n", err)
		}
	}()

	// Phase 1: Dump structure for all tables
	if err := d.dumpStructure(writer); err != nil {
		return nil, fmt.Errorf("failed to dump structure: %w", err)
	}

	// Phase 2: Dump data for non-excluded tables
	if err := d.dumpData(writer); err != nil {
		return nil, fmt.Errorf("failed to dump data: %w", err)
	}

	// Get file size
	fileInfo, err := outFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	result := &DumpResult{
		OutputFile:      d.options.OutputFile,
		Duration:        time.Since(startTime),
		ExcludedTables:  d.options.ExcludeTables,
		FileSize:        fileInfo.Size(),
		FileSizeDisplay: formatBytes(fileInfo.Size()),
	}

	return result, nil
}

// dumpStructure dumps the structure of all tables
func (d *Dumper) dumpStructure(writer io.Writer) error {
	// Create context that cancels on Ctrl+C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	args := d.buildMySQLDumpArgs()
	args = append(args,
		"--no-data",
		"--triggers",            // Explicitly include triggers
		"--events",              // Include scheduled events
		"--set-gtid-purged=OFF", // Cross-version compatibility
		"--column-statistics=0", // Avoid MySQL 8.0 warnings/errors
		// Note: --routines disabled due to MySQL 5.7 compatibility issues with INFORMATION_SCHEMA.LIBRARIES
	)
	args = append(args, d.options.Connection.Database)

	cmd := exec.CommandContext(ctx, "mysqldump", args...)
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr

	// Set MYSQL_PWD environment variable for secure password passing
	if d.options.Connection.Password != "" {
		cmd.Env = append(os.Environ(), "MYSQL_PWD="+d.options.Connection.Password)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysqldump structure failed: %w", err)
	}

	return nil
}

// dumpData dumps data for non-excluded tables
func (d *Dumper) dumpData(writer io.Writer) error {
	// Create context that cancels on Ctrl+C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	args := d.buildMySQLDumpArgs()
	args = append(args,
		"--no-create-info",
		"--skip-triggers",       // Prevent duplicate triggers
		"--skip-routines",       // Prevent duplicate routines
		"--skip-events",         // Prevent duplicate events
		"--set-gtid-purged=OFF", // Cross-version compatibility
		"--column-statistics=0", // Avoid MySQL 8.0 warnings/errors
	)

	// Add ignore-table flags for excluded tables
	for _, table := range d.options.ExcludeTables {
		args = append(args, fmt.Sprintf("--ignore-table=%s.%s",
			d.options.Connection.Database, table))
	}

	args = append(args, d.options.Connection.Database)

	cmd := exec.CommandContext(ctx, "mysqldump", args...)
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr

	// Set MYSQL_PWD environment variable for secure password passing
	if d.options.Connection.Password != "" {
		cmd.Env = append(os.Environ(), "MYSQL_PWD="+d.options.Connection.Password)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysqldump data failed: %w", err)
	}

	return nil
}

// buildMySQLDumpArgs builds common mysqldump arguments
// Note: Password is NOT included here - it's passed via MYSQL_PWD environment variable
func (d *Dumper) buildMySQLDumpArgs() []string {
	args := []string{
		"-h", d.options.Connection.Host,
		"-P", fmt.Sprintf("%d", d.options.Connection.Port),
		"-u", d.options.Connection.User,
	}

	// Add common flags
	args = append(args,
		"--single-transaction",
		"--quick",
		"--lock-tables=false",
	)

	// Add performance optimization flags
	args = append(args,
		"--max-allowed-packet=1G",
		"--net-buffer-length=1M",
		"--skip-comments",
		"--hex-blob", // Handle binary columns safely
	)

	return args
}

// dryRun performs a dry run showing what would be dumped
func (d *Dumper) dryRun() (*DumpResult, error) {
	result := &DumpResult{
		OutputFile:     d.options.OutputFile,
		Duration:       0,
		ExcludedTables: d.options.ExcludeTables,
		FileSize:       0,
	}

	return result, nil
}

// CheckMySQLDump verifies that mysqldump is available
func CheckMySQLDump() error {
	cmd := exec.Command("mysqldump", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysqldump not found in PATH: %w", err)
	}
	return nil
}
