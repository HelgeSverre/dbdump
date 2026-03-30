package database

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDumpFlushesBeforeReportingSize(t *testing.T) {
	resetMySQLDumpFeatures()

	restore := stubMySQLDump(t,
		func(ctx context.Context, name string, args ...string) *exec.Cmd {
			assertDefaultsFileArg(t, args)
			if len(args) == 0 {
				t.Fatal("expected mysqldump args")
			}

			script := "printf 'CREATE TABLE users(id int);\\n'"
			if containsArg(args, "--no-create-info") {
				script = "printf 'INSERT INTO users VALUES (1);\\n'"
			}

			return exec.CommandContext(ctx, "sh", "-c", script)
		},
	)
	defer restore()

	outputFile := filepath.Join(t.TempDir(), "dump.sql")
	dumper := NewDumper(&DumpOptions{
		Connection: &Connection{
			Host:     "127.0.0.1",
			Port:     3306,
			User:     "root",
			Password: `p@"ss`,
			Database: "testdb",
		},
		OutputFile: outputFile,
	})

	result, err := dumper.Dump()
	if err != nil {
		t.Fatalf("Dump returned error: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	if result.FileSize == 0 {
		t.Fatal("expected dump result to report a non-zero file size")
	}

	if result.FileSize != int64(len(data)) {
		t.Fatalf("reported file size mismatch: got %d want %d", result.FileSize, len(data))
	}

	if !bytes.Contains(data, []byte("CREATE TABLE users")) || !bytes.Contains(data, []byte("INSERT INTO users")) {
		t.Fatalf("unexpected dump contents: %q", string(data))
	}
}

func TestDumpRemovesPartialOutputOnFailure(t *testing.T) {
	resetMySQLDumpFeatures()

	call := 0
	restore := stubMySQLDump(t,
		func(ctx context.Context, name string, args ...string) *exec.Cmd {
			call++
			if call == 1 {
				return exec.CommandContext(ctx, "sh", "-c", "printf 'CREATE TABLE users(id int);\\n'")
			}

			return exec.CommandContext(ctx, "sh", "-c", "printf 'broken'; exit 1")
		},
	)
	defer restore()

	dir := t.TempDir()
	outputFile := filepath.Join(dir, "dump.sql")
	dumper := NewDumper(&DumpOptions{
		Connection: &Connection{Host: "127.0.0.1", Port: 3306, User: "root", Database: "testdb"},
		OutputFile: outputFile,
	})

	if _, err := dumper.Dump(); err == nil {
		t.Fatal("expected dump to fail")
	}

	if _, err := os.Stat(outputFile); !os.IsNotExist(err) {
		t.Fatalf("expected final output file to be absent, stat err=%v", err)
	}

	if matches, _ := filepath.Glob(filepath.Join(dir, "dump.sql.tmp-*")); len(matches) != 0 {
		t.Fatalf("expected temporary files to be cleaned up, found %v", matches)
	}
}

func TestDumpRejectsSymlinkOutputPath(t *testing.T) {
	resetMySQLDumpFeatures()

	restore := stubMySQLDump(t,
		func(ctx context.Context, name string, args ...string) *exec.Cmd {
			if containsArg(args, "--no-create-info") {
				return exec.CommandContext(ctx, "sh", "-c", "printf 'INSERT INTO users VALUES (1);\\n'")
			}

			return exec.CommandContext(ctx, "sh", "-c", "printf 'CREATE TABLE users(id int);\\n'")
		},
	)
	defer restore()

	dir := t.TempDir()
	protectedTarget := filepath.Join(dir, "protected.sql")
	if err := os.WriteFile(protectedTarget, []byte("keep me"), 0600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	outputPath := filepath.Join(dir, "dump.sql")
	if err := os.Symlink(protectedTarget, outputPath); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}

	dumper := NewDumper(&DumpOptions{
		Connection: &Connection{Host: "127.0.0.1", Port: 3306, User: "root", Database: "testdb"},
		OutputFile: outputPath,
	})

	if _, err := dumper.Dump(); err == nil {
		t.Fatal("expected dump to reject symlink output path")
	}
}

func TestDumpReplacesExistingOutputViaFallbackPath(t *testing.T) {
	resetMySQLDumpFeatures()

	restore := stubMySQLDump(t,
		func(ctx context.Context, name string, args ...string) *exec.Cmd {
			if containsArg(args, "--no-create-info") {
				return exec.CommandContext(ctx, "sh", "-c", "printf 'INSERT INTO users VALUES (1);\\n'")
			}

			return exec.CommandContext(ctx, "sh", "-c", "printf 'CREATE TABLE users(id int);\\n'")
		},
	)
	defer restore()

	dir := t.TempDir()
	outputPath := filepath.Join(dir, "dump.sql")
	if err := os.WriteFile(outputPath, []byte("previous"), 0600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	originalRename := renameFile
	renameAttempts := 0
	renameFile = func(oldPath, newPath string) error {
		if oldPath != outputPath && newPath == outputPath && renameAttempts == 0 {
			renameAttempts++
			return os.ErrExist
		}
		return originalRename(oldPath, newPath)
	}
	t.Cleanup(func() {
		renameFile = originalRename
	})

	dumper := NewDumper(&DumpOptions{
		Connection: &Connection{Host: "127.0.0.1", Port: 3306, User: "root", Database: "testdb"},
		OutputFile: outputPath,
	})

	result, err := dumper.Dump()
	if err != nil {
		t.Fatalf("Dump returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected dump result")
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !bytes.Contains(data, []byte("CREATE TABLE users")) || !bytes.Contains(data, []byte("INSERT INTO users")) {
		t.Fatalf("unexpected dump contents after fallback replace: %q", string(data))
	}
}

func stubMySQLDump(t *testing.T, run func(context.Context, string, ...string) *exec.Cmd) func() {
	t.Helper()

	oldExecCommand := execCommand
	oldExecCommandContext := execCommandContext

	resetMySQLDumpFeatures()
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "printf '%s\n' '--column-statistics' '--set-gtid-purged'")
	}
	execCommandContext = run

	return func() {
		execCommand = oldExecCommand
		execCommandContext = oldExecCommandContext
		resetMySQLDumpFeatures()
	}
}

func containsArg(args []string, needle string) bool {
	for _, arg := range args {
		if arg == needle {
			return true
		}
	}
	return false
}

func assertDefaultsFileArg(t *testing.T, args []string) {
	t.Helper()

	if len(args) == 0 || !strings.HasPrefix(args[0], "--defaults-extra-file=") {
		t.Fatalf("expected mysqldump args to start with --defaults-extra-file, got %v", args)
	}
}
