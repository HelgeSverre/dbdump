# Database Dump Tool Specification

## Overview
Create a standalone Go CLI tool called `dbdump` for intelligent MySQL database dumping across multiple projects. The tool will exclude noisy table data while preserving structure, with support for interactive mode, config files, and CLI flags.

## Technical Decisions
- **Platform**: Go binary (compiled, no dependencies, fast and portable)
- **Configuration**: Config file + Interactive mode + Command flags
- **Connections**: Direct MySQL connection (standard connection flags)
- **Features**: Progress indicator during dumps

## Project Structure
```
dump-tool/
├── cmd/
│   └── dbdump/
│       └── main.go              # Entry point
├── internal/
│   ├── config/
│   │   ├── config.go            # Config loading/parsing
│   │   └── profiles.go          # Connection profile management
│   ├── database/
│   │   ├── connection.go        # MySQL connection handling
│   │   ├── inspector.go         # Table inspection/analysis
│   │   └── dumper.go            # mysqldump wrapper
│   ├── ui/
│   │   ├── interactive.go       # Interactive table selection
│   │   └── progress.go          # Progress indicators
│   └── patterns/
│       └── matcher.go           # Table pattern matching
├── configs/
│   ├── defaults.yaml            # Default exclude patterns
│   └── example.yaml             # Example project config
├── go.mod
├── go.sum
├── README.md
├── Makefile                     # Build commands
└── .gitignore
```

## Core Files to Create

### 1. `configs/defaults.yaml`
Default noisy table patterns (Laravel-focused):
```yaml
default_excludes:
  exact:
    - audits
    - sessions
    - cache
    - cache_locks
    - failed_jobs
    - telescope_entries
    - telescope_entries_tags
    - telescope_monitoring
    - pulse_entries
    - pulse_aggregates
  patterns:
    - "telescope_*"
    - "pulse_*"
    - "_cache"
```

### 2. `configs/example.yaml`
Example project-specific configuration:
```yaml
# project.yaml
name: "Project Name"

# Tables to exclude data from (structure will be preserved)
exclude:
  exact:
    - audits
    - activity_logs
    - custom_noisy_table
  patterns:
    - "temp_*"
    - "*_cache"

```

### 3. `cmd/dbdump/main.go`
CLI interface with commands:
- `dbdump dump` - Main dump command
- `dbdump list` - List tables in database
- `dbdump config` - Manage configurations

### 4. `internal/database/dumper.go`
Core dumping logic:
- Two-phase dump (structure first, then data excluding noisy tables)
- Progress tracking using table count and estimated sizes
- Streaming output to file

### 5. `internal/ui/interactive.go`
Interactive mode using bubbletea/survey library:
- Checkbox list of all tables
- Pre-select tables based on config/patterns
- Show estimated table sizes
- Confirm before dumping

## CLI Interface

### Basic Usage
```bash
# Interactive mode (default)
dbdump dump -h localhost -u root -p password -d mydb

# With explicit excludes
dbdump dump -h localhost -u root -d mydb --exclude audits --exclude-pattern "telescope_*"

# Using config file
dbdump dump -h localhost -u root -d mydb --config ./project.yaml

# Non-interactive with smart defaults
dbdump dump -h localhost -u root -d mydb --auto -o dump.sql
```

### Flags
```
Global:
  -h, --host        Database host (default: 127.0.0.1)
  -P, --port        Database port (default: 3306)
  -u, --user        Database user
  -p, --password    Database password (or use MYSQL_PWD env)
  -d, --database    Database name

Dump specific:
  -o, --output      Output file (default: {database}_{timestamp}.sql)
  -c, --config      Config file path
  --exclude         Exclude specific table data (repeatable)
  --exclude-pattern Exclude tables matching pattern (repeatable)
  --auto            Use smart defaults without interaction
  --no-progress     Disable progress indicator
  --dry-run         Show what would be dumped without dumping
```

## Config File Format (YAML)

```yaml
# project.yaml
name: "Project Name"

# Tables to exclude data from (structure will be preserved)
exclude:
  exact:
    - audits
    - activity_logs
    - custom_noisy_table
  patterns:
    - "temp_*"
    - "*_cache"

```

## Implementation Plan

1. **Project Setup**
   - Initialize Go module
   - Set up project structure
   - Add dependencies (cobra for CLI, bubbletea/survey for interactive, progressbar)

2. **Core Database Logic**
   - MySQL connection handling
   - Table listing and size estimation
   - mysqldump wrapper with proper argument building

3. **Configuration System**
   - YAML config parsing
   - Default patterns loading
   - Pattern matching logic (exact match, glob patterns)

4. **Interactive UI**
   - Table selection interface
   - Progress bar during dump
   - Summary output

5. **CLI Commands**
   - `dump` command with all flags
   - `list` command to preview tables
   - `config` command for config management

6. **Build & Distribution**
   - Makefile for cross-compilation (macOS, Linux, Windows)
   - Installation instructions
   - Binary releases

## Key Dependencies
```go
require (
    github.com/spf13/cobra       // CLI framework
    github.com/charmbracelet/bubbletea // TUI framework
    github.com/schollz/progressbar/v3 // Progress bars
    gopkg.in/yaml.v3             // YAML parsing
    github.com/go-sql-driver/mysql // MySQL driver
)
```

## Dumping Strategy

### Two-Phase Approach
1. **Phase 1: Structure Dump**
   - Dump complete schema for ALL tables (including excluded ones)
   - Uses `mysqldump --no-data database_name`

2. **Phase 2: Data Dump**
   - Dump data for all tables EXCEPT excluded ones
   - Uses `mysqldump --no-create-info --ignore-table=db.table1 --ignore-table=db.table2 ...`

### Implementation Details
```go
// Pseudocode for dumper.go
func DumpDatabase(conn *Connection, excludes []string, output string) error {
    // Phase 1: Structure
    cmd1 := exec.Command("mysqldump",
        "-h", conn.Host,
        "-P", conn.Port,
        "-u", conn.User,
        "-p" + conn.Password,
        "--no-data",
        conn.Database,
    )

    // Phase 2: Data (with ignores)
    ignoreTables := []string{}
    for _, table := range excludes {
        ignoreTables = append(ignoreTables, fmt.Sprintf("--ignore-table=%s.%s", conn.Database, table))
    }

    cmd2 := exec.Command("mysqldump",
        "-h", conn.Host,
        "-P", conn.Port,
        "-u", conn.User,
        "-p" + conn.Password,
        "--no-create-info",
        ignoreTables...,
        conn.Database,
    )

    // Stream to output file with progress tracking
}
```

## Example Workflow

```bash
# 1. Connect to production database
$ dbdump dump -h prod.example.com -u readonly -d myapp_production

# Interactive mode shows:
# ┌─────────────────────────────────────────┐
# │ Select tables to EXCLUDE data from:     │
# │ ☑ audits (1.2GB)                        │
# │ ☑ telescope_entries (450MB)             │
# │ ☑ sessions (120MB)                      │
# │ ☐ users (2MB)                           │
# │ ☐ events (850MB)                        │
# └─────────────────────────────────────────┘

# 2. Dump executes
# Dumping structure...  ████████████ 100%
# Dumping data...       ████████░░░░ 75%
#
# ✓ Dump complete: myapp_production_20250121_143022.sql (1.5GB)
# ✓ Excluded 3 tables (data only, structure preserved)
# ✓ Duration: 2m 34s
```

## Advanced Features (Future Enhancements)

### Phase 2 (Optional)
- **SSH Tunneling**: Support SSH tunnel for secure remote access
- **Compression**: Automatic gzip compression
- **Import Helper**: Not just dump, but also import to local database
- **Data Anonymization**: Hash/anonymize sensitive fields during dump
- **Saved Profiles**: Save connection configurations for reuse
- **Progress Estimation**: More accurate progress based on actual data sizes
- **Parallel Dumping**: Dump multiple tables in parallel for speed
- **Custom mysqldump args**: Pass-through for additional mysqldump flags

## Benefits
- **Fast**: Skip gigabytes of audit/log data
- **Safe**: Always preserves table structure
- **Portable**: Single binary, works anywhere
- **Flexible**: Interactive, config, or CLI flags
- **Reusable**: One tool for all projects
- **Visual**: Progress indicators and summaries

## Use Case
This tool solves the problem of dumping production databases to local development environments where audit/log tables contain millions of rows that aren't needed for development but take hours to transfer. By excluding their data (but preserving structure), dump times can be reduced from hours to minutes while maintaining a fully functional database schema.

## Installation (After Build)

```bash
# Build for current platform
make build

# Install to /usr/local/bin
make install

# Build for all platforms
make build-all

# Usage
dbdump dump -h prod.example.com -u user -d database_name
```

## Example Real-World Scenario

**Before (using TablePlus or standard mysqldump):**
- Database: 15GB total
- Audits table: 12GB (10M rows)
- Actual data: 3GB
- Dump time: 3-4 hours
- Transfer time: 2+ hours

**After (using dbdump):**
- Dump excludes: audits, telescope_entries, sessions
- Output: 3.2GB (structure for all, data for non-noisy)
- Dump time: 15-20 minutes
- Transfer time: 30 minutes

**Time saved: 4-5 hours per database refresh**


----

Misc info for reference

github username: helgesverre
github repo: github.com/helgesverre/dbdump