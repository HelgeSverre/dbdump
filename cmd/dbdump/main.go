package main

import (
	"context"
	"errors"
	"fmt"
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
	printError        = ui.PrintError
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
	rootCmd.PersistentFlags().StringVar(&conn.SSH.Host, "ssh-host", "", "SSH bastion host for tunneling to the database")
	rootCmd.PersistentFlags().IntVar(&conn.SSH.Port, "ssh-port", 22, "SSH bastion port")
	rootCmd.PersistentFlags().StringVar(&conn.SSH.User, "ssh-user", "", "SSH username (defaults to database user)")
	rootCmd.PersistentFlags().StringVar(&conn.SSH.KeyFile, "ssh-key", "", "SSH private key path")
	rootCmd.PersistentFlags().IntVar(&conn.SSH.LocalPort, "ssh-local-port", 0, "Local port for the SSH tunnel (0 picks a free port)")

	rootCmd.AddCommand(newDumpCmd(conn))
	rootCmd.AddCommand(newListCmd(conn))
	rootCmd.AddCommand(newConfigCmd())

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
			return runDump(*conn, *opts)
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
			return runList(*conn)
		},
	}
}

func newConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect dbdump configuration",
		Long:  `Inspect dbdump configuration and saved connection profiles.`,
	}

	configCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List saved connection profiles",
		RunE:  runConfigList,
	})

	return configCmd
}

func runDump(connFlags connectionFlags, opts dumpFlags) error {
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

	printSuccess("Connected to database")

	tablesInfo, err := inspector.GetAllTablesInfo()
	if err != nil {
		return fmt.Errorf("failed to get table information: %w", err)
	}

	printInfo(fmt.Sprintf("Found %d tables", len(tablesInfo)))

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
		fmt.Println("\nDry run - would exclude the following tables:")
		for _, table := range finalExcludes {
			fmt.Printf("  - %s\n", table)
		}
		fmt.Printf("\nWould create dump file: %s\n", outputPath)
		return nil
	}

	if err := checkMySQLDump(); err != nil {
		return fmt.Errorf("mysqldump is required but not found in PATH: %w", err)
	}

	printInfo(fmt.Sprintf("Starting dump to %s", outputPath))

	dumper := newDumper(&database.DumpOptions{
		Connection:    conn,
		ExcludeTables: finalExcludes,
		OutputFile:    outputPath,
		DryRun:        opts.DryRun,
		Compression:   opts.Compression,
	})

	result, err := dumper.Dump()
	if err != nil {
		printError(err)
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
		fmt.Println()
	}

	return nil
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
		return fmt.Errorf("database user is required (use -u or --user)")
	}
	if c.Database == "" {
		return fmt.Errorf("database name is required (use -d or --database)")
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

func closeDB(db interface{ Close() error }) {
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
