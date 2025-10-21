package database

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
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

	// Create output file
	outFile, err := os.Create(d.options.OutputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	writer := bufio.NewWriter(outFile)
	defer writer.Flush()

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
	args := d.buildMySQLDumpArgs()
	args = append(args, "--no-data")
	args = append(args, d.options.Connection.Database)

	cmd := exec.Command("mysqldump", args...)
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysqldump structure failed: %w", err)
	}

	return nil
}

// dumpData dumps data for non-excluded tables
func (d *Dumper) dumpData(writer io.Writer) error {
	args := d.buildMySQLDumpArgs()
	args = append(args, "--no-create-info")

	// Add ignore-table flags for excluded tables
	for _, table := range d.options.ExcludeTables {
		args = append(args, fmt.Sprintf("--ignore-table=%s.%s",
			d.options.Connection.Database, table))
	}

	args = append(args, d.options.Connection.Database)

	cmd := exec.Command("mysqldump", args...)
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysqldump data failed: %w", err)
	}

	return nil
}

// buildMySQLDumpArgs builds common mysqldump arguments
func (d *Dumper) buildMySQLDumpArgs() []string {
	args := []string{
		"-h", d.options.Connection.Host,
		"-P", fmt.Sprintf("%d", d.options.Connection.Port),
		"-u", d.options.Connection.User,
	}

	// Add password if provided
	if d.options.Connection.Password != "" {
		args = append(args, fmt.Sprintf("-p%s", d.options.Connection.Password))
	}

	// Add common flags
	args = append(args,
		"--single-transaction",
		"--quick",
		"--lock-tables=false",
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
