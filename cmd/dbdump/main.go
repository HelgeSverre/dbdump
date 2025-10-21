package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/helgesverre/dbdump/internal/config"
	"github.com/helgesverre/dbdump/internal/database"
	"github.com/helgesverre/dbdump/internal/patterns"
	"github.com/helgesverre/dbdump/internal/ui"
	"github.com/spf13/cobra"
)

var (
	// Connection flags
	host     string
	port     int
	user     string
	password string
	dbName   string

	// Dump flags
	outputFile     string
	configFile     string
	excludeTables  []string
	excludePattern []string
	autoMode       bool
	noProgress     bool
	dryRun         bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "dbdump",
	Short: "Intelligent MySQL database dumping tool",
	Long: `dbdump is a CLI tool for intelligent MySQL database dumping.
It excludes noisy table data while preserving structure, making database
dumps faster and more manageable for development environments.`,
}

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump database with intelligent exclusions",
	Long: `Dump a MySQL database, excluding data from noisy tables (like audit logs,
sessions, cache) while preserving their structure.`,
	RunE: runDump,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tables in the database",
	Long:  `List all tables in the database with their sizes and row counts.`,
	RunE:  runList,
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration and profiles",
	Long:  `Manage dbdump configuration and connection profiles.`,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved connection profiles",
	RunE:  runConfigList,
}

func init() {
	// Global flags for database connection
	rootCmd.PersistentFlags().StringVarP(&host, "host", "H", "127.0.0.1", "Database host")
	rootCmd.PersistentFlags().IntVarP(&port, "port", "P", 3306, "Database port")
	rootCmd.PersistentFlags().StringVarP(&user, "user", "u", "", "Database user")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "Database password (or use MYSQL_PWD env)")
	rootCmd.PersistentFlags().StringVarP(&dbName, "database", "d", "", "Database name")

	// Dump command flags
	dumpCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: {database}_{timestamp}.sql)")
	dumpCmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file path")
	dumpCmd.Flags().StringArrayVar(&excludeTables, "exclude", []string{}, "Exclude specific table data (repeatable)")
	dumpCmd.Flags().StringArrayVar(&excludePattern, "exclude-pattern", []string{}, "Exclude tables matching pattern (repeatable)")
	dumpCmd.Flags().BoolVar(&autoMode, "auto", false, "Use smart defaults without interaction")
	dumpCmd.Flags().BoolVar(&noProgress, "no-progress", false, "Disable progress indicator")
	dumpCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be dumped without dumping")

	// Add commands
	rootCmd.AddCommand(dumpCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configListCmd)
}

func runDump(cmd *cobra.Command, args []string) error {
	// Check mysqldump availability
	if err := database.CheckMySQLDump(); err != nil {
		return fmt.Errorf("mysqldump is required but not found in PATH")
	}

	// Get password from environment if not provided
	if password == "" {
		password = os.Getenv("MYSQL_PWD")
	}

	// Validate required flags
	if user == "" {
		return fmt.Errorf("database user is required (use -u or --user)")
	}
	if dbName == "" {
		return fmt.Errorf("database name is required (use -d or --database)")
	}

	// Create connection
	conn := &database.Connection{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: dbName,
	}

	// Test connection
	if err := conn.TestConnection(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	ui.PrintSuccess("Connected to database")

	// Connect to database for inspection
	db, err := conn.Connect()
	if err != nil {
		return err
	}
	defer db.Close()

	// Get table information
	inspector := database.NewInspector(db)
	tablesInfo, err := inspector.GetAllTablesInfo()
	if err != nil {
		return fmt.Errorf("failed to get table information: %w", err)
	}

	ui.PrintInfo(fmt.Sprintf("Found %d tables", len(tablesInfo)))

	// Build exclude list
	excludeConfig, err := buildExcludeConfig()
	if err != nil {
		return err
	}

	// Match tables against patterns
	matcher := patterns.NewMatcher(excludeConfig)
	tableNames := make([]string, len(tablesInfo))
	for i, info := range tablesInfo {
		tableNames[i] = info.Name
	}
	preSelected := matcher.FilterTables(tableNames)

	var finalExcludes []string

	if autoMode {
		// Auto mode: use pattern-matched excludes
		finalExcludes = preSelected
		ui.PrintInfo(fmt.Sprintf("Auto mode: excluding %d tables based on patterns", len(finalExcludes)))
	} else {
		// Interactive mode
		selected, err := ui.RunInteractiveSelection(tablesInfo, preSelected)
		if err != nil {
			return fmt.Errorf("interactive selection failed: %w", err)
		}
		finalExcludes = selected
	}

	// Generate output filename if not provided
	if outputFile == "" {
		timestamp := time.Now().Format("20060102_150405")
		outputFile = fmt.Sprintf("%s_%s.sql", dbName, timestamp)
	}

	// Make output path absolute
	outputFile, err = filepath.Abs(outputFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if dryRun {
		fmt.Println("\nDry run - would exclude the following tables:")
		for _, table := range finalExcludes {
			fmt.Printf("  - %s\n", table)
		}
		fmt.Printf("\nWould create dump file: %s\n", outputFile)
		return nil
	}

	// Perform the dump
	ui.PrintInfo(fmt.Sprintf("Starting dump to %s", outputFile))

	dumper := database.NewDumper(&database.DumpOptions{
		Connection:    conn,
		ExcludeTables: finalExcludes,
		OutputFile:    outputFile,
		ShowProgress:  !noProgress,
		DryRun:        dryRun,
	})

	result, err := dumper.Dump()
	if err != nil {
		ui.PrintError(err)
		return err
	}

	// Print summary
	ui.PrintSummary(result.OutputFile, len(result.ExcludedTables), result.Duration, result.FileSizeDisplay)

	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	// Get password from environment if not provided
	if password == "" {
		password = os.Getenv("MYSQL_PWD")
	}

	// Validate required flags
	if user == "" {
		return fmt.Errorf("database user is required (use -u or --user)")
	}
	if dbName == "" {
		return fmt.Errorf("database name is required (use -d or --database)")
	}

	// Create connection
	conn := &database.Connection{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: dbName,
	}

	// Connect to database
	db, err := conn.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Get table information
	inspector := database.NewInspector(db)
	tablesInfo, err := inspector.GetAllTablesInfo()
	if err != nil {
		return fmt.Errorf("failed to get table information: %w", err)
	}

	// Print table information
	fmt.Printf("\nTables in database '%s':\n\n", dbName)
	fmt.Printf("%-40s %12s %15s\n", "Table Name", "Size", "Rows")
	fmt.Println(string(make([]byte, 70)))

	for _, info := range tablesInfo {
		fmt.Printf("%-40s %12s %15d\n", info.Name, info.SizeDisplay, info.RowCount)
	}

	fmt.Printf("\nTotal: %d tables\n\n", len(tablesInfo))

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

	fmt.Println("\nSaved connection profiles:\n")
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

func buildExcludeConfig() (config.ExcludeConfig, error) {
	var excludeConfig config.ExcludeConfig

	// Load defaults
	defaults, err := config.LoadDefaults()
	if err != nil {
		return excludeConfig, fmt.Errorf("failed to load defaults: %w", err)
	}

	// Start with defaults
	excludeConfig = defaults.DefaultExcludes

	// Load global config if it exists
	globalConfig, err := config.LoadGlobalConfig()
	if err != nil {
		return excludeConfig, fmt.Errorf("failed to load global config: %w", err)
	}
	if globalConfig != nil {
		excludeConfig = config.MergeExcludes(defaults, globalConfig)
	}

	// Load project config if provided (overrides global)
	if configFile != "" {
		projectConfig, err := config.LoadConfig(configFile)
		if err != nil {
			return excludeConfig, fmt.Errorf("failed to load config file: %w", err)
		}
		// Create a temporary defaults structure with the current merged config
		tempDefaults := &config.DefaultConfig{
			DefaultExcludes: excludeConfig,
		}
		excludeConfig = config.MergeExcludes(tempDefaults, projectConfig)
	}

	// Add CLI-specified excludes
	if len(excludeTables) > 0 {
		excludeConfig.Exact = append(excludeConfig.Exact, excludeTables...)
	}
	if len(excludePattern) > 0 {
		excludeConfig.Patterns = append(excludeConfig.Patterns, excludePattern...)
	}

	return excludeConfig, nil
}
