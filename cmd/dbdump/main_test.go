package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/helgesverre/dbdump/internal/config"
	"github.com/helgesverre/dbdump/internal/database"
	"github.com/helgesverre/dbdump/internal/ui"
)

type fakeSession struct {
	closed bool
}

func (s *fakeSession) Close() error {
	s.closed = true
	return nil
}

type fakeInspector struct {
	tables []database.TableInfo
	err    error
}

func (i fakeInspector) GetAllTablesInfo() ([]database.TableInfo, error) {
	return i.tables, i.err
}

type fakeDumper struct {
	result *database.DumpResult
	err    error
}

func (d fakeDumper) Dump() (*database.DumpResult, error) {
	return d.result, d.err
}

func TestNewRootCmdIncludesExpectedCommands(t *testing.T) {
	cmd := newRootCmd()

	commandNames := make([]string, 0, len(cmd.Commands()))
	for _, subcommand := range cmd.Commands() {
		commandNames = append(commandNames, subcommand.Name())
	}
	slices.Sort(commandNames)

	want := []string{"config", "dump", "list"}
	if !reflect.DeepEqual(commandNames, want) {
		t.Fatalf("unexpected commands: got %v want %v", commandNames, want)
	}

	for _, flagName := range []string{"host", "port", "user", "password", "database"} {
		if cmd.PersistentFlags().Lookup(flagName) == nil {
			t.Fatalf("expected persistent flag %q to be registered", flagName)
		}
	}

	for _, flagName := range []string{"ssh-host", "ssh-port", "ssh-user", "ssh-key", "ssh-local-port"} {
		if cmd.PersistentFlags().Lookup(flagName) == nil {
			t.Fatalf("expected SSH flag %q to be registered", flagName)
		}
	}

	if cmd.Flags().Lookup("compress") != nil {
		t.Fatal("compress flag should be scoped to the dump subcommand")
	}
}

func TestConnectionValidateRequiresUserAndDatabase(t *testing.T) {
	if err := (connectionFlags{}).validate(); err == nil || !strings.Contains(err.Error(), "database user is required") {
		t.Fatalf("expected missing-user error, got %v", err)
	}

	if err := (connectionFlags{User: "root"}).validate(); err == nil || !strings.Contains(err.Error(), "database name is required") {
		t.Fatalf("expected missing-database error, got %v", err)
	}

	if err := (connectionFlags{User: "root", Database: "testdb"}).validate(); err != nil {
		t.Fatalf("expected validation success, got %v", err)
	}
}

func TestEffectiveConnectionPrefersExplicitPassword(t *testing.T) {
	t.Setenv("DBDUMP_MYSQL_PWD", "from-dbdump-env")
	t.Setenv("MYSQL_PWD", "from-mysql-env")

	conn := connectionFlags{
		Host:     "127.0.0.1",
		Port:     3306,
		User:     "root",
		Password: "from-flag",
		Database: "testdb",
	}

	connection := conn.withResolvedPassword().toConnection()

	if connection.Password != "from-flag" {
		t.Fatalf("expected explicit password to win, got %q", connection.Password)
	}
}

func TestEffectiveConnectionPrefersDBDumpEnvOverMySQLEnv(t *testing.T) {
	t.Setenv("DBDUMP_MYSQL_PWD", "from-dbdump-env")
	t.Setenv("MYSQL_PWD", "from-mysql-env")

	conn := connectionFlags{
		Host:     "127.0.0.1",
		Port:     3306,
		User:     "root",
		Database: "testdb",
	}

	connection := conn.withResolvedPassword().toConnection()

	if connection.Password != "from-dbdump-env" {
		t.Fatalf("expected DBDUMP_MYSQL_PWD to win, got %q", connection.Password)
	}
}

func TestBuildExcludeConfigMergesAllLayers(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	globalPath := filepath.Join(homeDir, ".dbdump.yaml")
	if err := os.WriteFile(globalPath, []byte("exclude:\n  exact:\n    - global_table\n  patterns:\n    - \"global_*\"\n"), 0600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	projectPath := filepath.Join(t.TempDir(), "project.yaml")
	if err := os.WriteFile(projectPath, []byte("exclude:\n  exact:\n    - project_table\n  patterns:\n    - \"project_*\"\n"), 0600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	got, err := buildExcludeConfig(dumpFlags{
		ConfigFile:      projectPath,
		ExcludeTables:   []string{"cli_table"},
		ExcludePatterns: []string{"cli_*"},
	})
	if err != nil {
		t.Fatalf("buildExcludeConfig returned error: %v", err)
	}

	for _, exact := range []string{"audits", "global_table", "project_table", "cli_table"} {
		if !slices.Contains(got.Exact, exact) {
			t.Fatalf("expected exact excludes to contain %q, got %v", exact, got.Exact)
		}
	}

	for _, pattern := range []string{"*_cache", "global_*", "project_*", "cli_*"} {
		if !slices.Contains(got.Patterns, pattern) {
			t.Fatalf("expected pattern excludes to contain %q, got %v", pattern, got.Patterns)
		}
	}
}

func TestResolveExcludesAutoModeReturnsPreselected(t *testing.T) {
	stubPrintInfo(t, func(string) {})

	got, err := resolveExcludes(nil, []string{"audits", "sessions"}, true)
	if err != nil {
		t.Fatalf("resolveExcludes returned error: %v", err)
	}

	want := []string{"audits", "sessions"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected excludes: got %v want %v", got, want)
	}
}

func TestResolveExcludesRequiresTTYInInteractiveMode(t *testing.T) {
	stubTerminalCheck(t, func(int) bool { return false })

	_, err := resolveExcludes([]database.TableInfo{{Name: "users"}}, nil, false)
	if err == nil || !strings.Contains(err.Error(), "interactive mode requires a TTY") {
		t.Fatalf("expected TTY error, got %v", err)
	}
}

func TestResolveExcludesUsesInteractiveSelection(t *testing.T) {
	stubTerminalCheck(t, func(int) bool { return true })
	stubTableSelection(t, func(tables []database.TableInfo, preSelected []string) ([]string, error) {
		return []string{"sessions"}, nil
	})

	got, err := resolveExcludes([]database.TableInfo{{Name: "users"}}, []string{"audits"}, false)
	if err != nil {
		t.Fatalf("resolveExcludes returned error: %v", err)
	}

	want := []string{"sessions"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected excludes: got %v want %v", got, want)
	}
}

func TestRunDumpReturnsNilOnCancelledSelection(t *testing.T) {
	session := &fakeSession{}
	stubOpenInspection(t, func(conn *database.Connection) (inspectionSession, tableInspector, error) {
		return session, fakeInspector{tables: []database.TableInfo{{Name: "users"}}}, nil
	})
	stubTerminalCheck(t, func(int) bool { return true })
	stubTableSelection(t, func(tables []database.TableInfo, preSelected []string) ([]string, error) {
		return nil, ui.ErrSelectionCancelled
	})
	stubPrintInfo(t, func(string) {})
	stubPrintSuccess(t, func(string) {})

	err := runDump(connectionFlags{Host: "127.0.0.1", Port: 3306, User: "root", Database: "testdb"}, dumpFlags{})
	if err != nil {
		t.Fatalf("expected cancellation to be treated as nil error, got %v", err)
	}

	if !session.closed {
		t.Fatal("expected session to be closed")
	}
}

func TestResolveOutputPathUsesDefaultTimestamp(t *testing.T) {
	stubNowTime(t, func() time.Time {
		return time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	})

	got, err := resolveOutputPath("testdb", "", "auto")
	if err != nil {
		t.Fatalf("resolveOutputPath returned error: %v", err)
	}

	if filepath.Base(got) != "testdb_20260102_030405.sql" {
		t.Fatalf("unexpected default output path: %s", got)
	}
}

func TestResolveOutputPathUsesCompressionExtension(t *testing.T) {
	stubNowTime(t, func() time.Time {
		return time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	})

	got, err := resolveOutputPath("testdb", "", "gzip")
	if err != nil {
		t.Fatalf("resolveOutputPath returned error: %v", err)
	}

	if filepath.Base(got) != "testdb_20260102_030405.sql.gz" {
		t.Fatalf("unexpected compressed default output path: %s", got)
	}
}

func TestApplyProfileFlagsFillsOnlyUnsetFields(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	writeProfiles(t, "profiles:\n  - name: prod\n    host: db.example.com\n    port: 4406\n    user: readonly\n    password: s3cret\n    database: mydb\n")

	// host was set explicitly on the command line, everything else comes from the profile.
	changed := func(flag string) bool { return flag == "host" }
	conn := connectionFlags{Profile: "prod", Host: "cli-host"}

	merged, err := applyProfileFlags(conn, changed)
	if err != nil {
		t.Fatalf("applyProfileFlags returned error: %v", err)
	}

	if merged.Host != "cli-host" {
		t.Fatalf("expected explicit host to be kept, got %q", merged.Host)
	}
	if merged.Port != 4406 || merged.User != "readonly" || merged.Password != "s3cret" || merged.Database != "mydb" {
		t.Fatalf("expected unset fields to come from profile, got %#v", merged)
	}
}

func TestApplyProfileFlagsUnknownProfileErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := applyProfileFlags(connectionFlags{Profile: "nope"}, func(string) bool { return false })
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected profile-not-found error, got %v", err)
	}
}

func TestApplyProfileFlagsWithoutProfileIsNoop(t *testing.T) {
	conn := connectionFlags{Host: "127.0.0.1", User: "root", Database: "testdb"}
	got, err := applyProfileFlags(conn, func(string) bool { return false })
	if err != nil {
		t.Fatalf("applyProfileFlags returned error: %v", err)
	}
	if got != conn {
		t.Fatalf("expected connection flags unchanged, got %#v", got)
	}
}

func TestRunConfigAddSavesProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	stubPrintSuccess(t, func(string) {})

	err := runConfigAdd("prod", connectionFlags{
		Host: "db.example.com", Port: 3306, User: "readonly", Password: "s3cret", Database: "mydb",
	})
	if err != nil {
		t.Fatalf("runConfigAdd returned error: %v", err)
	}

	profiles, err := config.LoadProfiles()
	if err != nil {
		t.Fatalf("LoadProfiles returned error: %v", err)
	}
	got, ok := profiles.Find("prod")
	if !ok {
		t.Fatal("expected profile to be saved")
	}
	if got.Password != "s3cret" || got.Database != "mydb" {
		t.Fatalf("unexpected saved profile: %#v", got)
	}
}

func TestRunConfigRemoveDeletesProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	stubPrintSuccess(t, func(string) {})

	if err := runConfigAdd("prod", connectionFlags{User: "root", Database: "mydb"}); err != nil {
		t.Fatalf("runConfigAdd returned error: %v", err)
	}
	if err := runConfigRemove("prod"); err != nil {
		t.Fatalf("runConfigRemove returned error: %v", err)
	}

	profiles, err := config.LoadProfiles()
	if err != nil {
		t.Fatalf("LoadProfiles returned error: %v", err)
	}
	if _, ok := profiles.Find("prod"); ok {
		t.Fatal("expected profile to be removed")
	}

	if err := runConfigRemove("prod"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not-found error removing a missing profile, got %v", err)
	}
}

func writeProfiles(t *testing.T, contents string) {
	t.Helper()
	path, err := config.GetProfilesPath()
	if err != nil {
		t.Fatalf("GetProfilesPath returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func TestRunConfigListPrintsProfiles(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	configDir := filepath.Join(homeDir, ".config", "dbdump")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	profilesPath := filepath.Join(configDir, "profiles.yaml")
	if err := os.WriteFile(profilesPath, []byte("profiles:\n  - name: prod\n    host: db.example.com\n    port: 3306\n    user: readonly\n    database: mydb\n"), 0600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	output := captureStdout(t, func() {
		if err := runConfigList(nil, nil); err != nil {
			t.Fatalf("runConfigList returned error: %v", err)
		}
	})

	for _, fragment := range []string{"Saved connection profiles:", "prod", "db.example.com:3306", "Database: mydb"} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected output to contain %q, got %q", fragment, output)
		}
	}
}

func TestRunListPrintsTables(t *testing.T) {
	session := &fakeSession{}
	stubOpenInspection(t, func(conn *database.Connection) (inspectionSession, tableInspector, error) {
		return session, fakeInspector{tables: []database.TableInfo{
			{Name: "users", SizeDisplay: "1.0 KB", RowCount: 10},
			{Name: "audits", SizeDisplay: "2.0 KB", RowCount: 20},
		}}, nil
	})

	output := captureStdout(t, func() {
		if err := runList(connectionFlags{Host: "127.0.0.1", Port: 3306, User: "root", Database: "testdb"}); err != nil {
			t.Fatalf("runList returned error: %v", err)
		}
	})

	if !session.closed {
		t.Fatal("expected session to be closed")
	}

	for _, fragment := range []string{"Tables in database 'testdb':", "users", "audits", "Total: 2 tables"} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected output to contain %q, got %q", fragment, output)
		}
	}
}

func TestRunDumpDryRunPrintsPlan(t *testing.T) {
	session := &fakeSession{}
	stubOpenInspection(t, func(conn *database.Connection) (inspectionSession, tableInspector, error) {
		return session, fakeInspector{tables: []database.TableInfo{
			{Name: "users"},
			{Name: "audits"},
		}}, nil
	})
	stubPrintInfo(t, func(string) {})
	stubPrintSuccess(t, func(string) {})
	stubStartSSHTunnel(t, func(ctx context.Context, conn *database.Connection) (func() error, error) {
		return nil, nil
	})
	stubNowTime(t, func() time.Time {
		return time.Date(2026, time.March, 30, 12, 0, 0, 0, time.UTC)
	})
	stubNewDumper(t, func(opts *database.DumpOptions) dumpRunner {
		t.Fatal("newDumper should not be called during dry-run mode")
		return nil
	})
	stubCheckMySQLDump(t, func() error {
		t.Fatal("checkMySQLDump should not be called during dry-run mode")
		return nil
	})

	output := captureStdout(t, func() {
		err := runDump(connectionFlags{Host: "127.0.0.1", Port: 3306, User: "root", Database: "testdb"}, dumpFlags{AutoMode: true, DryRun: true, Compression: "gzip"})
		if err != nil {
			t.Fatalf("runDump returned error: %v", err)
		}
	})

	if !session.closed {
		t.Fatal("expected session to be closed")
	}

	if !strings.Contains(output, "Dry run - would exclude the following tables:") || !strings.Contains(output, "audits") {
		t.Fatalf("unexpected dry-run output: %q", output)
	}
	if !strings.Contains(output, "testdb_20260330_120000.sql.gz") {
		t.Fatalf("expected default output path in dry-run output, got %q", output)
	}
}

func TestRunDumpSuccessInvokesDumper(t *testing.T) {
	session := &fakeSession{}
	stubOpenInspection(t, func(conn *database.Connection) (inspectionSession, tableInspector, error) {
		return session, fakeInspector{tables: []database.TableInfo{
			{Name: "users"},
			{Name: "audits"},
		}}, nil
	})
	stubPrintInfo(t, func(string) {})
	stubPrintSuccess(t, func(string) {})
	stubCheckMySQLDump(t, func() error { return nil })
	stubStartSSHTunnel(t, func(ctx context.Context, conn *database.Connection) (func() error, error) {
		return nil, nil
	})

	var gotOptions *database.DumpOptions
	stubNewDumper(t, func(opts *database.DumpOptions) dumpRunner {
		gotOptions = opts
		return fakeDumper{
			result: &database.DumpResult{
				OutputFile:      opts.OutputFile,
				ExcludedTables:  opts.ExcludeTables,
				Duration:        time.Second,
				FileSizeDisplay: "1.0 KB",
			},
		}
	})

	var summaryOutput string
	stubPrintSummary(t, func(outputFile string, excludedCount int, duration time.Duration, size string) {
		summaryOutput = outputFile
	})

	outputPath := filepath.Join(t.TempDir(), "backup.sql")
	err := runDump(
		connectionFlags{Host: "127.0.0.1", Port: 3306, User: "root", Database: "testdb"},
		dumpFlags{AutoMode: true, OutputFile: outputPath, Compression: "zstd"},
	)
	if err != nil {
		t.Fatalf("runDump returned error: %v", err)
	}

	if !session.closed {
		t.Fatal("expected session to be closed")
	}

	if gotOptions == nil {
		t.Fatal("expected dumper options to be captured")
	}
	if gotOptions.OutputFile != outputPath {
		t.Fatalf("unexpected output file: got %q want %q", gotOptions.OutputFile, outputPath)
	}
	if gotOptions.Compression != "zstd" {
		t.Fatalf("unexpected compression: got %q want %q", gotOptions.Compression, "zstd")
	}
	if !slices.Contains(gotOptions.ExcludeTables, "audits") {
		t.Fatalf("expected audits to be excluded, got %v", gotOptions.ExcludeTables)
	}
	if summaryOutput != outputPath {
		t.Fatalf("unexpected summary output path: got %q want %q", summaryOutput, outputPath)
	}
}

func TestRunDumpReturnsFeatureCheckError(t *testing.T) {
	session := &fakeSession{}
	stubOpenInspection(t, func(conn *database.Connection) (inspectionSession, tableInspector, error) {
		return session, fakeInspector{tables: []database.TableInfo{{Name: "users"}}}, nil
	})
	stubPrintInfo(t, func(string) {})
	stubPrintSuccess(t, func(string) {})
	stubStartSSHTunnel(t, func(ctx context.Context, conn *database.Connection) (func() error, error) {
		return nil, nil
	})
	stubCheckMySQLDump(t, func() error {
		return errors.New("missing binary")
	})

	err := runDump(connectionFlags{Host: "127.0.0.1", Port: 3306, User: "root", Database: "testdb"}, dumpFlags{AutoMode: true})
	if err == nil || !strings.Contains(err.Error(), "mysqldump is required but not found in PATH") {
		t.Fatalf("expected mysqldump error, got %v", err)
	}
}

func TestRunDumpRejectsInvalidCompressionBeforeConnecting(t *testing.T) {
	stubOpenInspection(t, func(*database.Connection) (inspectionSession, tableInspector, error) {
		t.Fatal("should not connect when the compression flag is invalid")
		return nil, nil, nil
	})

	err := runDump(
		connectionFlags{Host: "127.0.0.1", Port: 3306, User: "root", Database: "testdb"},
		dumpFlags{AutoMode: true, Compression: "brotli"},
	)
	if err == nil || !strings.Contains(err.Error(), "unsupported compression") {
		t.Fatalf("expected unsupported compression error, got %v", err)
	}
}

func TestRunDumpChecksMySQLDumpBeforeSelection(t *testing.T) {
	session := &fakeSession{}
	stubOpenInspection(t, func(conn *database.Connection) (inspectionSession, tableInspector, error) {
		return session, fakeInspector{tables: []database.TableInfo{{Name: "users"}}}, nil
	})
	stubPrintInfo(t, func(string) {})
	stubPrintSuccess(t, func(string) {})
	stubStartSSHTunnel(t, func(ctx context.Context, conn *database.Connection) (func() error, error) {
		return nil, nil
	})
	stubTerminalCheck(t, func(int) bool { return true })
	stubTableSelection(t, func([]database.TableInfo, []string) ([]string, error) {
		t.Fatal("interactive selection must not run when mysqldump is missing")
		return nil, nil
	})
	stubCheckMySQLDump(t, func() error { return errors.New("missing binary") })

	err := runDump(connectionFlags{Host: "127.0.0.1", Port: 3306, User: "root", Database: "testdb"}, dumpFlags{})
	if err == nil || !strings.Contains(err.Error(), "mysqldump is required but not found in PATH") {
		t.Fatalf("expected mysqldump error, got %v", err)
	}
	if !session.closed {
		t.Fatal("expected session to be closed")
	}
}

func TestRunDumpStartsSSHTunnel(t *testing.T) {
	session := &fakeSession{}
	stubOpenInspection(t, func(conn *database.Connection) (inspectionSession, tableInspector, error) {
		if conn.Host != "127.0.0.1" || conn.Port != 4406 {
			t.Fatalf("expected tunneled connection, got %s:%d", conn.Host, conn.Port)
		}
		return session, fakeInspector{tables: []database.TableInfo{{Name: "users"}}}, nil
	})
	stubPrintInfo(t, func(string) {})
	stubPrintSuccess(t, func(string) {})
	stubCheckMySQLDump(t, func() error { return nil })
	stubNewDumper(t, func(opts *database.DumpOptions) dumpRunner {
		return fakeDumper{result: &database.DumpResult{OutputFile: opts.OutputFile, FileSizeDisplay: "1 B"}}
	})

	started := false
	stopped := false
	stubStartSSHTunnel(t, func(ctx context.Context, conn *database.Connection) (func() error, error) {
		started = true
		conn.Host = "127.0.0.1"
		conn.Port = 4406
		return func() error {
			stopped = true
			return nil
		}, nil
	})

	err := runDump(connectionFlags{
		Host: "db.internal", Port: 3306, User: "root", Database: "testdb",
		SSH: sshFlags{Host: "bastion.example.com", Port: 22, User: "deploy"},
	}, dumpFlags{AutoMode: true})
	if err != nil {
		t.Fatalf("runDump returned error: %v", err)
	}

	if !started || !stopped {
		t.Fatalf("expected SSH tunnel lifecycle to run, started=%v stopped=%v", started, stopped)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe returned error: %v", err)
	}

	os.Stdout = writer
	t.Cleanup(func() {
		os.Stdout = originalStdout
	})

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}

	return string(output)
}

func stubOpenInspection(t *testing.T, fn func(*database.Connection) (inspectionSession, tableInspector, error)) {
	t.Helper()
	original := openInspection
	openInspection = fn
	t.Cleanup(func() {
		openInspection = original
	})
}

func stubNewDumper(t *testing.T, fn func(*database.DumpOptions) dumpRunner) {
	t.Helper()
	original := newDumper
	newDumper = fn
	t.Cleanup(func() {
		newDumper = original
	})
}

func stubCheckMySQLDump(t *testing.T, fn func() error) {
	t.Helper()
	original := checkMySQLDump
	checkMySQLDump = fn
	t.Cleanup(func() {
		checkMySQLDump = original
	})
}

func stubStartSSHTunnel(t *testing.T, fn func(context.Context, *database.Connection) (func() error, error)) {
	t.Helper()
	original := startSSHTunnel
	startSSHTunnel = fn
	t.Cleanup(func() {
		startSSHTunnel = original
	})
}

func stubTableSelection(t *testing.T, fn func([]database.TableInfo, []string) ([]string, error)) {
	t.Helper()
	original := runTableSelection
	runTableSelection = fn
	t.Cleanup(func() {
		runTableSelection = original
	})
}

func stubTerminalCheck(t *testing.T, fn func(int) bool) {
	t.Helper()
	original := isTerminal
	isTerminal = fn
	t.Cleanup(func() {
		isTerminal = original
	})
}

func stubNowTime(t *testing.T, fn func() time.Time) {
	t.Helper()
	original := nowTime
	nowTime = fn
	t.Cleanup(func() {
		nowTime = original
	})
}

func stubPrintInfo(t *testing.T, fn func(string)) {
	t.Helper()
	original := printInfo
	printInfo = fn
	t.Cleanup(func() {
		printInfo = original
	})
}

func stubPrintSuccess(t *testing.T, fn func(string)) {
	t.Helper()
	original := printSuccess
	printSuccess = fn
	t.Cleanup(func() {
		printSuccess = original
	})
}

func stubPrintSummary(t *testing.T, fn func(string, int, time.Duration, string)) {
	t.Helper()
	original := printSummary
	printSummary = fn
	t.Cleanup(func() {
		printSummary = original
	})
}
