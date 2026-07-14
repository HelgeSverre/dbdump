package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/helgesverre/dbdump/internal/config"
	"github.com/helgesverre/dbdump/internal/database"
	"github.com/helgesverre/dbdump/internal/patterns"
	"github.com/helgesverre/dbdump/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type inspectionSession interface {
	Close() error
}

type tableInspector interface {
	GetAllTablesInfo() ([]database.TableInfo, error)
}

type dumpRunner interface {
	Dump() (*database.DumpResult, error)
}

var (
	openInspection = func(conn *database.Connection) (inspectionSession, tableInspector, error) {
		db, err := conn.Connect()
		if err != nil {
			return nil, nil, err
		}

		return db, database.NewInspector(db), nil
	}
	checkMySQLDump    = database.CheckMySQLDump
	newDumper         = func(opts *database.DumpOptions) dumpRunner { return database.NewDumper(opts) }
	runTableSelection = ui.RunInteractiveSelection
	isTerminal        = func(fd int) bool { return term.IsTerminal(fd) }
	nowTime           = time.Now
	printInfo         = ui.PrintInfo
	printSuccess      = ui.PrintSuccess
	printSummary      = ui.PrintSummary
	startSSHTunnel    = func(ctx context.Context, conn *database.Connection) (func() error, error) {
		return database.StartSSHTunnel(ctx, conn)
	}
)

type connectionFlags struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	Profile  string
	SSH      sshFlags
}

type sshFlags struct {
	Host      string
	Port      int
	User      string
	KeyFile   string
	LocalPort int
}

type dumpFlags struct {
	OutputFile      string
	ConfigFile      string
	ExcludeTables   []string
	ExcludePatterns []string
	AutoMode        bool
	DryRun          bool
	Compression     string
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	conn := &connectionFlags{}

	rootCmd := &cobra.Command{
		Use:   "dbdump",
		Short: "Intelligent MySQL database dumping tool",
		Long: `dbdump is a CLI tool for intelligent MySQL database dumping.
It excludes noisy table data while preserving structure, making database
dumps faster and more manageable for development environments.`,
	}

	rootCmd.PersistentFlags().StringVarP(&conn.Host, "host", "H", "127.0.0.1", "Database host")
	rootCmd.PersistentFlags().IntVarP(&conn.Port, "port", "P", 3306, "Database port")
	rootCmd.PersistentFlags().StringVarP(&conn.User, "user", "u", "", "Database user")
	rootCmd.PersistentFlags().StringVarP(&conn.Password, "password", "p", "", "Database password (or use DBDUMP_MYSQL_PWD/MYSQL_PWD env)")
	rootCmd.PersistentFlags().StringVarP(&conn.Database, "database", "d", "", "Database name")
	rootCmd.PersistentFlags().StringVar(&conn.Profile, "profile", "", "Load connection settings from a saved profile (see 'dbdump config')")
	rootCmd.PersistentFlags().StringVar(&conn.SSH.Host, "ssh-host", "", "SSH bastion host for tunneling to the database")
	rootCmd.PersistentFlags().IntVar(&conn.SSH.Port, "ssh-port", 22, "SSH bastion port")
	rootCmd.PersistentFlags().StringVar(&conn.SSH.User, "ssh-user", "", "SSH username (defaults to database user)")
	rootCmd.PersistentFlags().StringVar(&conn.SSH.KeyFile, "ssh-key", "", "SSH private key path")
	rootCmd.PersistentFlags().IntVar(&conn.SSH.LocalPort, "ssh-local-port", 0, "Local port for the SSH tunnel (0 picks a free port)")

	rootCmd.AddCommand(newDumpCmd(conn))
	rootCmd.AddCommand(newListCmd(conn))
	rootCmd.AddCommand(newConfigCmd(conn))

	return rootCmd
}

func newDumpCmd(conn *connectionFlags) *cobra.Command {
	opts := &dumpFlags{}

	dumpCmd := &cobra.Command{
		Use:   "dump",
		Short: "Dump database with intelligent exclusions",
		Long: `Dump a MySQL database, excluding data from noisy tables (like audit logs,
sessions, cache) while preserving their structure.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			connFlags, err := applyProfileFlags(*conn, cmd.Flags().Changed)
			if err != nil {
				return err
			}
			return runDump(connFlags, *opts)
		},
	}

	dumpCmd.Flags().StringVarP(&opts.OutputFile, "output", "o", "", "Output file (default: {database}_{timestamp}.sql)")
	dumpCmd.Flags().StringVarP(&opts.ConfigFile, "config", "c", "", "Config file path")
	dumpCmd.Flags().StringArrayVar(&opts.ExcludeTables, "exclude", []string{}, "Exclude specific table data (repeatable)")
	dumpCmd.Flags().StringArrayVar(&opts.ExcludePatterns, "exclude-pattern", []string{}, "Exclude tables matching pattern (repeatable)")
	dumpCmd.Flags().BoolVar(&opts.AutoMode, "auto", false, "Use smart defaults without interaction")
	dumpCmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show what would be dumped without dumping")
	dumpCmd.Flags().StringVar(&opts.Compression, "compress", "auto", "Compression format: auto, none, gzip, zstd")

	return dumpCmd
}

func newListCmd(conn *connectionFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all tables in the database",
		Long:  `List all tables in the database with their sizes and row counts.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			connFlags, err := applyProfileFlags(*conn, cmd.Flags().Changed)
			if err != nil {
				return err
			}
			return runList(connFlags)
		},
	}
}

func newConfigCmd(conn *connectionFlags) *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and manage dbdump configuration",
		Long:  `Inspect and manage dbdump configuration and saved connection profiles.`,
	}

	configCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List saved connection profiles",
		RunE:  runConfigList,
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "add <name>",
		Short: "Save the current connection flags as a named profile",
		Long: `Save the current connection flags (--host/--port/--user/--password/--database)
as a named profile. The profiles file is written with 0600 permissions; treat it
like any other credential store.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigAdd(args[0], *conn)
		},
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "remove <name>",
		Short: "Delete a saved connection profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigRemove(args[0])
		},
	})

	return configCmd
}

func runDump(connFlags connectionFlags, opts dumpFlags) error {
	connFlags = connFlags.withResolvedPassword()
	if err := connFlags.validate(); err != nil {
		return err
	}

	// Validate the compression flag before doing any work so a bad value fails
	// fast instead of after connecting and running interactive selection.
	if _, err := database.ResolveCompressionFormat(opts.OutputFile, opts.Compression); err != nil {
		return err
	}

	conn := connFlags.toConnection()
	stopTunnel, err := startSSHTunnel(context.Background(), conn)
	if err != nil {
		return err
	}
	defer closeSSH(stopTunnel)

	session, inspector, err := openInspection(conn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer closeDB(session)

	printSuccess("Connected to database")

	tablesInfo, err := inspector.GetAllTablesInfo()
	if err != nil {
		return fmt.Errorf("failed to get table information: %w", err)
	}

	printInfo(fmt.Sprintf("Found %d tables", len(tablesInfo)))

	// A real dump needs mysqldump; check it before asking the user to make
	// selections so a missing binary doesn't waste their interactive input.
	if !opts.DryRun {
		if err := checkMySQLDump(); err != nil {
			return fmt.Errorf("mysqldump is required but not found in PATH: %w", err)
		}
	}

	excludeConfig, err := buildExcludeConfig(opts)
	if err != nil {
		return err
	}

	matcher, err := patterns.NewMatcher(excludeConfig)
	if err != nil {
		return err
	}
	tableNames := make([]string, len(tablesInfo))
	for i, info := range tablesInfo {
		tableNames[i] = info.Name
	}
	preSelected := matcher.FilterTables(tableNames)

	finalExcludes, err := resolveExcludes(tablesInfo, preSelected, opts.AutoMode)
	if err != nil {
		if errors.Is(err, ui.ErrSelectionCancelled) {
			printInfo("Dump cancelled")
			return nil
		}
		return err
	}

	outputPath, err := resolveOutputPath(connFlags.Database, opts.OutputFile, opts.Compression)
	if err != nil {
		return err
	}

	if opts.DryRun {
		return writeDryRunPlan(os.Stdout, tablesInfo, finalExcludes, outputPath, opts.Compression)
	}

	printInfo(fmt.Sprintf("Starting dump to %s", outputPath))

	dumper := newDumper(&database.DumpOptions{
		Connection:    conn,
		ExcludeTables: finalExcludes,
		OutputFile:    outputPath,
		Compression:   opts.Compression,
	})

	result, err := dumper.Dump()
	if err != nil {
		return err
	}

	printSummary(result.OutputFile, len(result.ExcludedTables), result.Duration, result.FileSizeDisplay)
	return nil
}

func runList(connFlags connectionFlags) error {
	connFlags = connFlags.withResolvedPassword()
	if err := connFlags.validate(); err != nil {
		return err
	}

	conn := connFlags.toConnection()
	stopTunnel, err := startSSHTunnel(context.Background(), conn)
	if err != nil {
		return err
	}
	defer closeSSH(stopTunnel)

	session, inspector, err := openInspection(conn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer closeDB(session)

	tablesInfo, err := inspector.GetAllTablesInfo()
	if err != nil {
		return fmt.Errorf("failed to get table information: %w", err)
	}

	fmt.Printf("\nTables in database '%s':\n\n", connFlags.Database)
	fmt.Printf("%-40s %12s %15s\n", "Table Name", "Size", "Rows")
	fmt.Println(strings.Repeat("-", 70))

	for _, info := range tablesInfo {
		fmt.Printf("%-40s %12s %15d\n", info.Name, info.SizeDisplay, info.RowCount)
	}

	fmt.Printf("\nTotal: %d tables\n", len(tablesInfo))
	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	profiles, err := config.LoadProfiles()
	if err != nil {
		return fmt.Errorf("failed to load profiles: %w", err)
	}

	if len(profiles.Profiles) == 0 {
		fmt.Println("No saved profiles found")
		return nil
	}

	fmt.Println("\nSaved connection profiles:")
	for _, profile := range profiles.Profiles {
		fmt.Printf("  %s\n", profile.Name)
		fmt.Printf("    Host: %s:%d\n", profile.Host, profile.Port)
		fmt.Printf("    User: %s\n", profile.User)
		if profile.Database != "" {
			fmt.Printf("    Database: %s\n", profile.Database)
		}
		if profile.Password != "" {
			fmt.Println("    Password: (saved)")
		}
		fmt.Println()
	}

	return nil
}

func runConfigAdd(name string, connFlags connectionFlags) error {
	profiles, err := config.LoadProfiles()
	if err != nil {
		return fmt.Errorf("failed to load profiles: %w", err)
	}

	profiles.Upsert(config.ConnectionProfile{
		Name:     name,
		Host:     connFlags.Host,
		Port:     connFlags.Port,
		User:     connFlags.User,
		Password: connFlags.Password,
		Database: connFlags.Database,
	})

	if err := config.SaveProfiles(profiles); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	printSuccess(fmt.Sprintf("Saved connection profile %q", name))
	return nil
}

func runConfigRemove(name string) error {
	profiles, err := config.LoadProfiles()
	if err != nil {
		return fmt.Errorf("failed to load profiles: %w", err)
	}

	if !profiles.Remove(name) {
		return fmt.Errorf("connection profile %q not found", name)
	}

	if err := config.SaveProfiles(profiles); err != nil {
		return fmt.Errorf("failed to save profiles: %w", err)
	}

	printSuccess(fmt.Sprintf("Removed connection profile %q", name))
	return nil
}

// applyProfileFlags merges a named profile into the connection flags. Explicit
// command-line flags win over profile values (changed reports whether a flag was
// set on the command line), and only non-empty profile fields are applied. The
// password still falls back to the environment via withResolvedPassword when
// neither a flag nor the profile provides one.
func applyProfileFlags(conn connectionFlags, changed func(string) bool) (connectionFlags, error) {
	if conn.Profile == "" {
		return conn, nil
	}

	profiles, err := config.LoadProfiles()
	if err != nil {
		return conn, fmt.Errorf("failed to load profiles: %w", err)
	}

	profile, ok := profiles.Find(conn.Profile)
	if !ok {
		return conn, fmt.Errorf("connection profile %q not found", conn.Profile)
	}

	if profile.Host != "" && !changed("host") {
		conn.Host = profile.Host
	}
	if profile.Port != 0 && !changed("port") {
		conn.Port = profile.Port
	}
	if profile.User != "" && !changed("user") {
		conn.User = profile.User
	}
	if profile.Password != "" && !changed("password") {
		conn.Password = profile.Password
	}
	if profile.Database != "" && !changed("database") {
		conn.Database = profile.Database
	}

	return conn, nil
}

func (c connectionFlags) withResolvedPassword() connectionFlags {
	if c.Password != "" {
		return c
	}

	c.Password = os.Getenv("DBDUMP_MYSQL_PWD")
	if c.Password == "" {
		c.Password = os.Getenv("MYSQL_PWD")
	}

	return c
}

func (c connectionFlags) validate() error {
	if c.User == "" {
		return errors.New("database user is required (use -u or --user)")
	}
	if c.Database == "" {
		return errors.New("database name is required (use -d or --database)")
	}
	return nil
}

func (c connectionFlags) toConnection() *database.Connection {
	sshConfig := database.SSHConfig{
		Host:      c.SSH.Host,
		Port:      c.SSH.Port,
		User:      c.SSH.User,
		KeyFile:   c.SSH.KeyFile,
		LocalPort: c.SSH.LocalPort,
	}
	if sshConfig.User == "" {
		sshConfig.User = c.User
	}

	return &database.Connection{
		Host:     c.Host,
		Port:     c.Port,
		User:     c.User,
		Password: c.Password,
		Database: c.Database,
		SSH:      sshConfig,
	}
}

func buildExcludeConfig(opts dumpFlags) (config.ExcludeConfig, error) {
	var excludeConfig config.ExcludeConfig

	defaults, err := config.LoadDefaults()
	if err != nil {
		return excludeConfig, fmt.Errorf("failed to load defaults: %w", err)
	}

	excludeConfig = defaults.DefaultExcludes

	globalConfig, err := config.LoadGlobalConfig()
	if err != nil {
		return excludeConfig, fmt.Errorf("failed to load global config: %w", err)
	}
	if globalConfig != nil {
		excludeConfig = config.MergeExcludes(defaults, globalConfig)
	}

	if opts.ConfigFile != "" {
		projectConfig, err := config.LoadConfig(opts.ConfigFile)
		if err != nil {
			return excludeConfig, fmt.Errorf("failed to load config file: %w", err)
		}

		tempDefaults := &config.DefaultConfig{DefaultExcludes: excludeConfig}
		excludeConfig = config.MergeExcludes(tempDefaults, projectConfig)
	}

	if len(opts.ExcludeTables) > 0 {
		excludeConfig.Exact = append(excludeConfig.Exact, opts.ExcludeTables...)
		excludeConfig.Exact = config.UniqueStrings(excludeConfig.Exact)
	}
	if len(opts.ExcludePatterns) > 0 {
		excludeConfig.Patterns = append(excludeConfig.Patterns, opts.ExcludePatterns...)
		excludeConfig.Patterns = config.UniqueStrings(excludeConfig.Patterns)
	}

	return excludeConfig, nil
}

func resolveExcludes(tables []database.TableInfo, preSelected []string, autoMode bool) ([]string, error) {
	if autoMode {
		ui.PrintInfo(fmt.Sprintf("Auto mode: excluding %d tables based on patterns", len(preSelected)))
		return preSelected, nil
	}

	if len(tables) == 0 {
		printInfo("No tables found; skipping interactive selection")
		return nil, nil
	}

	if !isTerminal(int(os.Stdin.Fd())) || !isTerminal(int(os.Stdout.Fd())) {
		return nil, fmt.Errorf("interactive mode requires a TTY; rerun with --auto")
	}

	selected, err := runTableSelection(tables, preSelected)
	if err != nil {
		return nil, fmt.Errorf("interactive selection failed: %w", err)
	}

	return selected, nil
}

// writeDryRunPlan renders the full dump plan without writing any data: every table
// with its size/row count and whether its data will be dumped or only its
// structure preserved, plus totals and the resolved output path and compression.
func writeDryRunPlan(w io.Writer, tables []database.TableInfo, excludes []string, outputPath, compression string) error {
	format, err := database.ResolveCompressionFormat(outputPath, compression)
	if err != nil {
		return err
	}

	excluded := make(map[string]struct{}, len(excludes))
	for _, name := range excludes {
		excluded[name] = struct{}{}
	}

	lines := []string{
		"\nDry run — no data will be written.\n",
		fmt.Sprintf("\nOutput: %s  (compression: %s)\n\n", outputPath, format),
		fmt.Sprintf("%-40s %12s %14s  %s\n", "TABLE", "SIZE", "ROWS", "MODE"),
	}

	var totalSize, skippedData int64
	excludedCount := 0
	for _, table := range tables {
		mode := "data + structure"
		if _, ok := excluded[table.Name]; ok {
			mode = "structure only"
			excludedCount++
			skippedData += table.DataSize
		}
		lines = append(lines, fmt.Sprintf("%-40s %12s %14d  %s\n", table.Name, table.SizeDisplay, table.RowCount, mode))
		totalSize += table.TotalSize
	}

	lines = append(lines,
		"\nSummary:\n",
		fmt.Sprintf("  %d tables, %d excluded from data (structure preserved)\n", len(tables), excludedCount),
		fmt.Sprintf("  Total size across all tables: %s\n", database.FormatBytes(totalSize)),
	)
	if excludedCount > 0 {
		lines = append(lines, fmt.Sprintf("  Data skipped for excluded tables: ~%s\n", database.FormatBytes(skippedData)))
	}

	_, err = io.WriteString(w, strings.Join(lines, ""))
	return err
}

func resolveOutputPath(databaseName, configuredPath, compression string) (string, error) {
	compressionFormat, err := database.ResolveCompressionFormat(configuredPath, compression)
	if err != nil {
		return "", err
	}

	outputPath := configuredPath
	if outputPath == "" {
		timestamp := nowTime().Format("20060102_150405")
		outputPath = fmt.Sprintf("%s_%s.sql%s", databaseName, timestamp, database.CompressionExtension(compressionFormat))
	}

	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return absPath, nil
}

func closeDB(db inspectionSession) {
	if err := db.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to close database connection: %v\n", err)
	}
}

func closeSSH(stop func() error) {
	if stop == nil {
		return
	}
	if err := stop(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to stop SSH tunnel: %v\n", err)
	}
}
