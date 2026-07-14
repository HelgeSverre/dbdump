package database

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/klauspost/compress/zstd"
)

var (
	execCommandContext = exec.CommandContext
	execCommand        = exec.Command
	createTempFile     = os.CreateTemp
	removeFile         = os.Remove
	renameFile         = os.Rename
	signalNotify       = signal.NotifyContext
)

type mysqlDumpFeatureSet struct {
	ColumnStatistics bool
	SetGTIDPurged    bool
}

var (
	mysqlDumpFeaturesOnce sync.Once
	mysqlDumpFeatures     mysqlDumpFeatureSet
	mysqlDumpFeaturesErr  error
)

// DumpOptions contains options for dumping the database
type DumpOptions struct {
	Connection    *Connection
	ExcludeTables []string
	OutputFile    string
	Compression   string
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
func (d *Dumper) Dump() (result *DumpResult, err error) {
	startTime := time.Now()

	// Trap interrupt/termination signals for the entire dump — including the
	// finalize/rename window — so the deferred temp-file cleanup always runs
	// instead of the process dying mid-write and orphaning a .tmp file.
	ctx, stop := signalNotify(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := validateDumpOutputPath(d.options.OutputFile); err != nil {
		return nil, err
	}

	defaultsFile, cleanupDefaults, err := d.createDefaultsExtraFile()
	if err != nil {
		return nil, err
	}
	defer cleanupDefaults()

	outFile, err := createTempFile(
		filepath.Dir(d.options.OutputFile),
		filepath.Base(d.options.OutputFile)+".tmp-*",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}

	tempPath := outFile.Name()
	closed := false
	defer func() {
		if !closed {
			if closeErr := outFile.Close(); closeErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to close output file: %v\n", closeErr)
			}
		}
		if err != nil {
			if removeErr := removeFile(tempPath); removeErr != nil && !os.IsNotExist(removeErr) {
				fmt.Fprintf(os.Stderr, "Warning: failed to remove temporary output file: %v\n", removeErr)
			}
		}
	}()

	if err = outFile.Chmod(0600); err != nil {
		return nil, fmt.Errorf("failed to set output file permissions: %w", err)
	}

	compressionFormat, err := ResolveCompressionFormat(d.options.OutputFile, d.options.Compression)
	if err != nil {
		return nil, err
	}

	compressedWriter, err := newCompressedWriter(outFile, compressionFormat)
	if err != nil {
		return nil, err
	}

	writer := bufio.NewWriterSize(compressedWriter, 256*1024)

	if err = d.dumpStructure(ctx, writer, defaultsFile); err != nil {
		return nil, fmt.Errorf("failed to dump structure: %w", err)
	}

	if err = d.dumpData(ctx, writer, defaultsFile); err != nil {
		return nil, fmt.Errorf("failed to dump data: %w", err)
	}

	if err = writer.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush output: %w", err)
	}

	if err = compressedWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize compressed output: %w", err)
	}

	if err = outFile.Sync(); err != nil {
		return nil, fmt.Errorf("failed to sync output file: %w", err)
	}

	fileInfo, err := outFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	if err = outFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close output file: %w", err)
	}
	closed = true

	if err = replaceOutputFile(tempPath, d.options.OutputFile); err != nil {
		return nil, fmt.Errorf("failed to move output file into place: %w", err)
	}

	result = &DumpResult{
		OutputFile:      d.options.OutputFile,
		Duration:        time.Since(startTime),
		ExcludedTables:  d.options.ExcludeTables,
		FileSize:        fileInfo.Size(),
		FileSizeDisplay: formatBytes(fileInfo.Size()),
	}

	return result, nil
}

func validateDumpOutputPath(path string) error {
	info, err := os.Lstat(path)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to write dump output through symlink: %s", path)
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect dump output path: %w", err)
	}

	return nil
}

func replaceOutputFile(tempPath, finalPath string) error {
	if err := renameFile(tempPath, finalPath); err == nil {
		return nil
	} else if !os.IsExist(err) {
		if _, statErr := os.Stat(finalPath); statErr != nil {
			return err
		}
	}

	backupPath := fmt.Sprintf("%s.bak-%d", finalPath, time.Now().UnixNano())
	if err := renameFile(finalPath, backupPath); err != nil {
		return err
	}

	if err := renameFile(tempPath, finalPath); err != nil {
		restoreErr := renameFile(backupPath, finalPath)
		if restoreErr != nil {
			return fmt.Errorf("%w (restore failed: %v)", err, restoreErr)
		}
		return err
	}

	if err := removeFile(backupPath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Warning: failed to remove backup output file: %v\n", err)
	}

	return nil
}

// Compression formats supported by dbdump.
const (
	CompressionAuto = "auto"
	CompressionNone = "none"
	CompressionGzip = "gzip"
	CompressionZstd = "zstd"
)

func ResolveCompressionFormat(outputPath, requested string) (string, error) {
	format := strings.TrimSpace(strings.ToLower(requested))
	if format == "" {
		format = CompressionAuto
	}

	if format == CompressionAuto {
		return inferCompressionFormat(outputPath), nil
	}

	switch format {
	case CompressionNone, CompressionGzip, CompressionZstd:
		return format, nil
	default:
		return "", fmt.Errorf("unsupported compression format %q (expected auto, none, gzip, or zstd)", requested)
	}
}

func CompressionExtension(format string) string {
	switch format {
	case CompressionGzip:
		return ".gz"
	case CompressionZstd:
		return ".zst"
	default:
		return ""
	}
}

func inferCompressionFormat(outputPath string) string {
	lower := strings.ToLower(outputPath)
	switch {
	case strings.HasSuffix(lower, ".sql.gz"), strings.HasSuffix(lower, ".gz"):
		return CompressionGzip
	case strings.HasSuffix(lower, ".sql.zst"), strings.HasSuffix(lower, ".zst"), strings.HasSuffix(lower, ".zstd"):
		return CompressionZstd
	default:
		return CompressionNone
	}
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

func newCompressedWriter(target io.Writer, format string) (io.WriteCloser, error) {
	switch format {
	case CompressionNone:
		return nopWriteCloser{Writer: target}, nil
	case CompressionGzip:
		return gzip.NewWriter(target), nil
	case CompressionZstd:
		writer, err := zstd.NewWriter(target)
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd writer: %w", err)
		}
		return writer, nil
	default:
		return nil, fmt.Errorf("unsupported compression format %q", format)
	}
}

// dumpStructure dumps the structure of all tables
func (d *Dumper) dumpStructure(ctx context.Context, writer io.Writer, defaultsFile string) error {
	args := d.buildMySQLDumpArgs(defaultsFile)
	args = append(args,
		"--no-data",
		"--triggers",
		"--events",
	)

	args, err := appendMySQLDumpFeatureArgs(args)
	if err != nil {
		return err
	}

	args = append(args, d.options.Connection.Database)

	cmd := execCommandContext(ctx, "mysqldump", args...)
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysqldump structure failed: %w", err)
	}

	return nil
}

// dumpData dumps data for non-excluded tables
func (d *Dumper) dumpData(ctx context.Context, writer io.Writer, defaultsFile string) error {
	args := d.buildMySQLDumpArgs(defaultsFile)
	args = append(args,
		"--no-create-info",
		"--skip-triggers",
		"--skip-routines",
		"--skip-events",
	)

	args, err := appendMySQLDumpFeatureArgs(args)
	if err != nil {
		return err
	}

	for _, table := range d.options.ExcludeTables {
		args = append(args, fmt.Sprintf("--ignore-table=%s.%s",
			d.options.Connection.Database, table))
	}

	args = append(args, d.options.Connection.Database)

	cmd := execCommandContext(ctx, "mysqldump", args...)
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysqldump data failed: %w", err)
	}

	return nil
}

// appendMySQLDumpFeatureArgs appends version-dependent compatibility flags shared
// by the structure and data passes, so a new flag only has to be added once.
func appendMySQLDumpFeatureArgs(args []string) ([]string, error) {
	features, err := getMySQLDumpFeatures()
	if err != nil {
		return nil, err
	}
	if features.SetGTIDPurged {
		args = append(args, "--set-gtid-purged=OFF")
	}
	if features.ColumnStatistics {
		args = append(args, "--column-statistics=0")
	}
	return args, nil
}

// buildMySQLDumpArgs builds common mysqldump arguments.
func (d *Dumper) buildMySQLDumpArgs(defaultsFile string) []string {
	args := []string{}
	if defaultsFile != "" {
		args = append(args, "--defaults-extra-file="+defaultsFile)
	}

	args = append(args,
		"-h", d.options.Connection.Host,
		"-P", fmt.Sprintf("%d", d.options.Connection.Port),
		"-u", d.options.Connection.User,
		"--single-transaction",
		"--quick",
		"--lock-tables=false",
		"--max-allowed-packet=1G",
		"--net-buffer-length=1M",
		"--skip-comments",
		"--hex-blob",
	)

	return args
}

func (d *Dumper) createDefaultsExtraFile() (string, func(), error) {
	if d.options.Connection.Password == "" {
		return "", func() {}, nil
	}

	file, err := createTempFile("", "dbdump-mysql-*.cnf")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create mysqldump defaults file: %w", err)
	}

	path := file.Name()
	cleanup := func() {
		if err := removeFile(path); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove mysqldump defaults file: %v\n", err)
		}
	}

	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		cleanup()
		return "", nil, fmt.Errorf("failed to secure mysqldump defaults file: %w", err)
	}

	content := fmt.Sprintf("[client]\npassword=%s\n", mysqlOptionValue(d.options.Connection.Password))
	if _, err := io.WriteString(file, content); err != nil {
		_ = file.Close()
		cleanup()
		return "", nil, fmt.Errorf("failed to write mysqldump defaults file: %w", err)
	}

	if err := file.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to close mysqldump defaults file: %w", err)
	}

	return path, cleanup, nil
}

func mysqlOptionValue(value string) string {
	escaped := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"\n", "\\n",
		"\r", "\\r",
	).Replace(value)
	return `"` + escaped + `"`
}

func getMySQLDumpFeatures() (mysqlDumpFeatureSet, error) {
	mysqlDumpFeaturesOnce.Do(func() {
		cmd := execCommand("mysqldump", "--help")
		output, err := cmd.CombinedOutput()
		if err != nil {
			mysqlDumpFeaturesErr = fmt.Errorf("failed to inspect mysqldump features: %w", err)
			if len(output) > 0 {
				mysqlDumpFeaturesErr = fmt.Errorf("%w: %s", mysqlDumpFeaturesErr, strings.TrimSpace(string(output)))
			}
			return
		}

		help := string(output)
		mysqlDumpFeatures = mysqlDumpFeatureSet{
			ColumnStatistics: strings.Contains(help, "column-statistics"),
			SetGTIDPurged:    strings.Contains(help, "set-gtid-purged"),
		}
	})

	return mysqlDumpFeatures, mysqlDumpFeaturesErr
}

func resetMySQLDumpFeatures() {
	mysqlDumpFeaturesOnce = sync.Once{}
	mysqlDumpFeatures = mysqlDumpFeatureSet{}
	mysqlDumpFeaturesErr = nil
}

// CheckMySQLDump verifies that mysqldump is available
func CheckMySQLDump() error {
	cmd := execCommand("mysqldump", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysqldump not found in PATH: %w", err)
	}
	return nil
}
