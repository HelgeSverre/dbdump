# dbdump

A fast, intelligent MySQL database dumping tool that excludes noisy table data while preserving structure.

## Why dbdump?

When dumping production databases for development, you often don't need millions of audit log entries, session data, or cache records. These tables can make dumps take hours and consume gigabytes of space.

**dbdump solves this by:**

- Excluding data from noisy tables (audits, sessions, cache, etc.)
- Always preserving table structure (no broken foreign keys)
- Reducing dump time from hours to minutes
- Making development database refreshes practical

## Features

- **Smart Defaults**: Pre-configured patterns for common Laravel tables (audits, telescope, pulse, sessions, cache)
- **Interactive Mode**: Visual table selection with sizes and row counts
- **Config Files**: Save project-specific exclusion patterns
- **Fast**: Skip gigabytes of unnecessary data
- **Safe**: Always preserves table structure
- **Portable**: Single binary, no dependencies

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/helgesverre/dbdump.git
cd dbdump

# Build the binary
make build

# Install to /usr/local/bin
make install
```

### Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/helgesverre/dbdump/releases).

### Requirements

- Go 1.21+ (for building from source)
- MySQL client tools (`mysqldump` must be in your PATH)

## Quick Start

### Interactive Mode (Default)

```bash
dbdump dump -h localhost -u root -p password -d mydb
```

This will:

1. Connect to your database
2. Show all tables with sizes
3. Pre-select noisy tables based on patterns
4. Let you customize the selection
5. Dump structure for all tables, data for selected tables

### Auto Mode (Non-Interactive)

```bash
dbdump dump -h localhost -u root -d mydb --auto
```

Uses smart defaults without interaction.

### With Config File

```bash
dbdump dump -h localhost -u root -d mydb --config ./project.yaml
```

## Usage

### Basic Commands

```bash
# Dump database (interactive)
dbdump dump -h localhost -u root -p password -d mydb

# List tables with sizes
dbdump list -h localhost -u root -d mydb

# Dry run (see what would be excluded)
dbdump dump -h localhost -u root -d mydb --dry-run

# Dump with custom output file
dbdump dump -h localhost -u root -d mydb -o backup.sql
```

### Connection Options

```bash
-H, --host        Database host (default: 127.0.0.1)
-P, --port        Database port (default: 3306)
-u, --user        Database user
-p, --password    Database password (or use MYSQL_PWD env)
-d, --database    Database name
```

### Dump Options

```bash
-o, --output           Output file (default: {database}_{timestamp}.sql)
-c, --config           Config file path
    --exclude          Exclude specific table data (repeatable)
    --exclude-pattern  Exclude tables matching pattern (repeatable)
    --auto             Use smart defaults without interaction
    --no-progress      Disable progress indicator
    --dry-run          Show what would be dumped without dumping
```

### Examples

```bash
# Use environment variable for password
export MYSQL_PWD=secret
dbdump dump -h prod.example.com -u readonly -d myapp_prod

# Exclude specific tables
dbdump dump -h localhost -u root -d mydb \
  --exclude audits \
  --exclude activity_logs \
  --exclude-pattern "temp_*"

# With project config
dbdump dump -h localhost -u root -d mydb --config ./myproject.yaml

# Auto mode with custom output
dbdump dump -h localhost -u root -d mydb --auto -o daily-backup.sql
```

## Configuration

dbdump supports multiple configuration layers that merge together:

1. **Built-in defaults** (always applied)
2. **Global user config** (`~/.dbdump.yaml`) - optional, applies to all dumps
3. **Project config** (via `--config` flag) - optional, project-specific
4. **CLI flags** (highest priority)

For a comprehensive guide, see [USER-GUIDE.md](USER-GUIDE.md).

### Global User Config

Create `~/.dbdump.yaml` for settings that apply to all your dumps:

```yaml
name: "My Global Config"

exclude:
  exact:
    - activity_logs
    - user_sessions
  patterns:
    - "temp_*"
    - "*_backup"
```

### Project Config File

Create a `project.yaml` file in your project:

```yaml
name: "My Project"

exclude:
  exact:
    - audits
    - activity_logs
    - custom_noisy_table
  patterns:
    - "temp_*"
    - "*_cache"
    - "old_*"
```

Use it with:

```bash
dbdump dump -h localhost -u root -d mydb --config ./project.yaml
```

### Default Exclusions

dbdump includes smart defaults for common Laravel tables:

**Exact matches:**

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

**Patterns:**

- `telescope_*`
- `pulse_*`
- `_cache`

These defaults are always applied and can be extended with project configs or CLI flags.

## How It Works

dbdump uses a two-phase approach:

1. **Phase 1: Structure Dump**
   - Dumps complete schema for ALL tables
   - Ensures foreign keys and relationships are preserved
   - Uses `mysqldump --no-data`

2. **Phase 2: Data Dump**
   - Dumps data for all tables EXCEPT excluded ones
   - Uses `mysqldump --no-create-info --ignore-table=...`

Result: A complete database dump with empty noisy tables.

## Real-World Example

**Before (standard mysqldump):**

- Database: 15GB total
- Audits table: 12GB (10M rows)
- Actual data needed: 3GB
- Dump time: 3-4 hours
- Transfer time: 2+ hours

**After (using dbdump):**

- Excludes: audits, telescope_entries, sessions
- Output: 3.2GB (structure for all, data for non-noisy)
- Dump time: 15-20 minutes
- Transfer time: 30 minutes

**Time saved: 4-5 hours per database refresh**

## Documentation

- **[USER-GUIDE.md](USER-GUIDE.md)** - Comprehensive user guide with detailed configuration, examples, and troubleshooting
- **[BENCHMARKING.md](BENCHMARKING.md)** - Performance testing guide
- **[VERIFIED_PERFORMANCE.md](VERIFIED_PERFORMANCE.md)** - Real-world benchmark results
- **[CHANGELOG.md](CHANGELOG.md)** - Version history

## Development

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Format code
make fmt
```

### Project Structure

```
dump-tool/
├── cmd/dbdump/          # CLI entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── database/        # Database operations
│   ├── patterns/        # Pattern matching
│   └── ui/              # Interactive UI and progress
├── configs/             # Default configurations
└── Makefile             # Build commands
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details

## Author

[Helge Sverre](https://github.com/helgesverre)

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI
- Interactive UI using [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- Progress bars by [progressbar](https://github.com/schollz/progressbar)
