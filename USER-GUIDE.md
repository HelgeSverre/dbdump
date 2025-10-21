# dbdump User Guide

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
  - [Configuration Priority](#configuration-priority)
  - [Default Excludes](#default-excludes)
  - [Global User Config](#global-user-config)
  - [Project-Specific Config](#project-specific-config)
  - [CLI Flags](#cli-flags)
- [Commands](#commands)
  - [dump](#dump-command)
  - [list](#list-command)
  - [config list](#config-list-command)
- [Usage Examples](#usage-examples)
- [Interactive Mode](#interactive-mode)
- [Connection Profiles](#connection-profiles)
- [Troubleshooting](#troubleshooting)
- [FAQ](#faq)

---

## Overview

**dbdump** is an intelligent MySQL database dumping tool designed specifically for development environments. It solves a common problem: production database dumps are often massive and time-consuming because they contain huge amounts of data that developers don't need (audit logs, session data, cache tables, etc.).

### Key Benefits

- **Faster dumps**: Reduce dump time from hours to minutes (verified: 3-4 hours → 15-20 minutes)
- **Smaller files**: Dramatically reduce file sizes (verified: 15GB → 3.2GB)
- **Safer approach**: Always preserves ALL table structures, preventing foreign key errors
- **Smart defaults**: Pre-configured to exclude common "noisy" tables from Laravel and other frameworks
- **User-friendly**: Interactive table selection with visual feedback
- **Zero dependencies**: Single binary (requires only `mysqldump` which comes with MySQL)

### How It Works

dbdump uses a two-phase dump strategy:

1. **Phase 1 - Structure**: Dumps the schema (CREATE TABLE statements) for ALL tables
2. **Phase 2 - Data**: Dumps data only for tables you want to keep

This ensures your database structure remains intact (no broken foreign keys!) while excluding unwanted data.

---

## Installation

### macOS/Linux

```bash
# Download the latest release
curl -LO https://github.com/helgesverre/dbdump/releases/latest/download/dbdump-darwin-arm64

# Make it executable
chmod +x dbdump-darwin-arm64

# Move to your PATH
sudo mv dbdump-darwin-arm64 /usr/local/bin/dbdump
```

### From Source

```bash
git clone https://github.com/helgesverre/dbdump.git
cd dbdump
make install
```

### Verify Installation

```bash
dbdump --help
```

You should also verify that `mysqldump` is available:

```bash
mysqldump --version
```

---

## Quick Start

The fastest way to get started:

```bash
# Basic usage with interactive mode
dbdump dump -u root -p yourpassword -d mydatabase

# Auto mode (uses smart defaults, no interaction)
dbdump dump -u root -p yourpassword -d mydatabase --auto

# Specify output file
dbdump dump -u root -p yourpassword -d mydatabase -o backup.sql
```

For security, you can also use the `MYSQL_PWD` environment variable:

```bash
export MYSQL_PWD=yourpassword
dbdump dump -u root -d mydatabase
```

---

## Configuration

dbdump uses a flexible, layered configuration system that allows you to customize which tables to exclude at multiple levels.

### Configuration Priority

Configurations are merged in this order (later overrides earlier):

1. **Built-in defaults** (embedded in the binary)
2. **Global user config** (`~/.dbdump.yaml`)
3. **Project-specific config** (specified with `--config` flag)
4. **CLI flags** (`--exclude`, `--exclude-pattern`)

### Default Excludes

dbdump comes with sensible defaults for common Laravel and PHP framework tables:

**Exact table names:**
- `audits`
- `sessions`
- `cache`
- `cache_locks`
- `failed_jobs`
- `telescope_entries`
- `telescope_entries_tags`
- `telescope_monitoring`
- `pulse_entries`
- `pulse_aggregates`

**Pattern matches:**
- `telescope_*` - All Laravel Telescope tables
- `pulse_*` - All Laravel Pulse tables
- `_cache` - Any table ending with "_cache"

These defaults are always loaded first and can be supplemented (not replaced) by your custom configs.

### Global User Config

You can create a global config file that applies to all your dumps:

**Location:** `~/.dbdump.yaml`

This file is optional and will be automatically loaded if it exists.

**Example:**

```yaml
name: "My Global Config"
exclude:
  exact:
    - activity_logs
    - user_sessions
    - notifications
  patterns:
    - "temp_*"
    - "*_backup"
    - "old_*"
```

**When to use:**
- Tables you always want to exclude across all projects
- Organization-wide standards
- Personal preferences

### Project-Specific Config

Create a config file for each project to customize excludes for that specific database:

**Location:** Anywhere (specify with `--config` flag)

**Example:**

```yaml
name: "E-commerce Site"
exclude:
  exact:
    - cart_abandonments
    - email_tracking
    - search_logs
  patterns:
    - "analytics_*"
    - "tracking_*"
```

**Usage:**

```bash
dbdump dump -u root -d mydb --config ./myproject.yaml
```

**When to use:**
- Project-specific large tables
- Tables unique to your application
- Different requirements per database

### CLI Flags

Override everything at runtime with flags:

```bash
# Exclude specific tables
dbdump dump -u root -d mydb --exclude users --exclude orders

# Exclude by pattern
dbdump dump -u root -d mydb --exclude-pattern "temp_*" --exclude-pattern "old_*"

# Combine both
dbdump dump -u root -d mydb \
  --exclude sessions \
  --exclude cache \
  --exclude-pattern "log_*"
```

**When to use:**
- One-off dumps with special requirements
- Testing different exclude patterns
- Quick overrides without editing config files

---

## Commands

### dump Command

Dump a MySQL database with intelligent table exclusions.

**Basic Syntax:**

```bash
dbdump dump [flags]
```

**Connection Flags:**

| Flag | Short | Description | Default | Required |
|------|-------|-------------|---------|----------|
| `--host` | `-H` | Database host | 127.0.0.1 | No |
| `--port` | `-P` | Database port | 3306 | No |
| `--user` | `-u` | Database user | - | Yes |
| `--password` | `-p` | Database password | $MYSQL_PWD | No |
| `--database` | `-d` | Database name | - | Yes |

**Dump Flags:**

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--output` | `-o` | Output file path | `{database}_{timestamp}.sql` |
| `--config` | `-c` | Project config file | - |
| `--exclude` | - | Exclude specific table (repeatable) | - |
| `--exclude-pattern` | - | Exclude pattern (repeatable) | - |
| `--auto` | - | Use defaults without interaction | false |
| `--no-progress` | - | Disable progress bar | false |
| `--dry-run` | - | Show what would be excluded without dumping | false |

**Examples:**

```bash
# Interactive mode (default)
dbdump dump -u root -d mydb

# Auto mode with custom output
dbdump dump -u root -d mydb --auto -o production-backup.sql

# Remote database
dbdump dump -H db.example.com -P 3306 -u dbuser -d mydb

# With project config
dbdump dump -u root -d mydb --config ./project-config.yaml

# Dry run to preview
dbdump dump -u root -d mydb --dry-run

# Additional CLI excludes
dbdump dump -u root -d mydb --exclude users --exclude-pattern "test_*"
```

### list Command

Display all tables in the database with their sizes and row counts.

**Basic Syntax:**

```bash
dbdump list [flags]
```

**Example:**

```bash
dbdump list -u root -d mydb
```

**Output:**

```
Tables in database 'mydb':

Table Name                               Size         Rows
----------------------------------------------------------------------
users                                   15.2 MB       12500
orders                                 125.3 MB       85000
products                                 8.1 MB        2500
sessions                               450.0 MB      150000
telescope_entries                      1.2 GB        500000

Total: 5 tables
```

**Use cases:**
- Identify large tables before dumping
- See which tables are taking up space
- Decide what to exclude

### config list Command

Show all saved connection profiles.

**Basic Syntax:**

```bash
dbdump config list
```

**Example Output:**

```
Saved connection profiles:

  production
    Host: db.example.com:3306
    User: produser
    Database: maindb

  staging
    Host: staging.example.com:3306
    User: stageuser
    Database: stagedb
```

**Note:** Connection profile saving is not yet implemented but reserved for future use.

---

## Usage Examples

### Example 1: First Time Setup

Create a global config for your personal preferences:

```bash
# Create your global config
cat > ~/.dbdump.yaml << EOF
name: "My Global Defaults"
exclude:
  exact:
    - activity_logs
    - user_sessions
  patterns:
    - "temp_*"
    - "*_old"
EOF

# Now all your dumps will use this config
dbdump dump -u root -d mydb
```

### Example 2: Laravel Application

For a typical Laravel application:

```bash
# Create project config
cat > laravel-config.yaml << EOF
name: "Laravel App"
exclude:
  exact:
    - jobs
    - failed_jobs
    - password_resets
    - personal_access_tokens
  patterns:
    - "cache_*"
    - "session_*"
EOF

# Use it
dbdump dump -u root -d laravel_db --config laravel-config.yaml --auto
```

### Example 3: Production to Development

Dump a production database for local development:

```bash
# On production server
dbdump dump \
  -H localhost \
  -u produser \
  -d production_db \
  --auto \
  -o prod-for-dev.sql

# Transfer file
scp prod-for-dev.sql dev@localhost:~/

# On development machine
mysql -u root mydb < prod-for-dev.sql
```

### Example 4: Quick One-Off Dump

Need to exclude something specific just this once:

```bash
dbdump dump -u root -d mydb \
  --exclude user_activity \
  --exclude api_logs \
  --exclude-pattern "temp_*" \
  -o quick-backup.sql
```

### Example 5: Inspecting Before Dumping

Check your database first:

```bash
# See all tables and sizes
dbdump list -u root -d mydb

# Do a dry run
dbdump dump -u root -d mydb --dry-run

# If happy, do the real dump
dbdump dump -u root -d mydb
```

---

## Interactive Mode

When you run `dbdump dump` without the `--auto` flag, you'll enter interactive mode.

### What You'll See

1. **Connection confirmation**: "Connected to database ✓"
2. **Table count**: "Found 47 tables"
3. **Interactive selector**: Visual list of all tables with:
   - Table names
   - Sizes (MB/GB)
   - Row counts
   - Pre-selected status (based on your configs)

### Keyboard Controls

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Move cursor up/down |
| `Space` | Toggle selection |
| `Enter` | Confirm and proceed with dump |
| `Ctrl+C` | Cancel and exit |

### Tips

- **Pre-selected tables** are marked based on your configuration (defaults + global + project)
- You can toggle any table on or off
- Tables shown in the selector will have their DATA excluded (structure is always kept)
- Review the sizes to identify large tables you might want to exclude

---

## Connection Profiles

**Note:** Connection profile management is planned for a future release.

The profiles system will allow you to save connection details:

```bash
# Save a profile (future feature)
dbdump profile save production \
  --host db.example.com \
  --port 3306 \
  --user produser \
  --database maindb

# Use it (future feature)
dbdump dump --profile production
```

Profiles will be stored in `~/.config/dbdump/profiles.yaml`.

---

## Troubleshooting

### "mysqldump is required but not found in PATH"

**Solution:** Install MySQL client tools:

```bash
# macOS
brew install mysql-client

# Ubuntu/Debian
sudo apt-get install mysql-client

# CentOS/RHEL
sudo yum install mysql
```

### "failed to connect to database"

**Check:**
1. Database is running: `mysql -u root -p -e "SELECT 1"`
2. Credentials are correct
3. Host/port are correct
4. User has necessary permissions

**Permissions needed:**

```sql
GRANT SELECT, SHOW VIEW, TRIGGER, LOCK TABLES ON database_name.* TO 'user'@'host';
```

### "failed to load config file"

**Check:**
1. File exists: `ls -la ~/.dbdump.yaml`
2. YAML syntax is valid
3. File is readable: `chmod 644 ~/.dbdump.yaml`

**Validate YAML:**

```bash
# Install yq if needed
brew install yq

# Validate
yq eval ~/.dbdump.yaml
```

### Dump is still too large

**Options:**
1. Use `dbdump list` to identify large tables
2. Add more excludes to your config
3. Use `--dry-run` to preview what will be excluded
4. Use interactive mode to manually deselect tables

### Pattern not matching

Pattern matching uses glob syntax:
- `*` matches any characters
- `?` matches a single character
- Use quotes: `--exclude-pattern "temp_*"`

**Test your pattern:**

```bash
# List tables first
dbdump list -u root -d mydb

# Dry run with pattern
dbdump dump -u root -d mydb --exclude-pattern "your_pattern_*" --dry-run
```

---

## FAQ

### Does dbdump delete my data?

No! dbdump never modifies your source database. It only creates a dump file.

### Will excluding tables break foreign keys?

No! The two-phase approach ensures ALL table structures are preserved in the dump. Only the DATA is excluded for specified tables.

### Can I use this in production?

dbdump is designed for creating development dumps from production databases. It's safe to run on production (read-only operations), but consider:
- Running during low-traffic periods
- Impact on database performance
- Using a read replica if available

### How do I restore a dbdump file?

Just like any MySQL dump:

```bash
mysql -u root -p database_name < dump_file.sql
```

### Can I exclude all data and dump only structure?

Yes! Use interactive mode and deselect all tables, or configure patterns to match everything:

```bash
dbdump dump -u root -d mydb --exclude-pattern "*"
```

### What's the difference between exact and patterns?

- **Exact**: Must match the table name exactly (fast, uses hash map)
- **Patterns**: Uses glob-style wildcards (*, ?) to match multiple tables

### Can I include only specific tables?

dbdump focuses on exclusion, but you can achieve this by:
1. Excluding everything: `--exclude-pattern "*"`
2. Using interactive mode to manually select only what you want

### How do I update my global config?

Just edit the file:

```bash
nano ~/.dbdump.yaml
```

Changes take effect on the next dump.

### Can I see what will be excluded before dumping?

Yes! Use the `--dry-run` flag:

```bash
dbdump dump -u root -d mydb --dry-run
```

This shows exactly what would be excluded without creating a dump.

### Why is my dump still slow?

dbdump optimizes what's dumped, but can't make `mysqldump` itself faster. To improve speed:
- Exclude more tables
- Use `--auto` to skip interactive mode
- Ensure good network connectivity to database
- Consider database performance

### Is dbdump safe for sensitive data?

dbdump doesn't modify or send data anywhere—it's just a wrapper around `mysqldump`. However:
- Dump files contain database contents (encrypt if needed)
- Be careful with connection profiles (passwords stored in plain text)
- Use environment variables for passwords instead of CLI flags

### How do I contribute or report bugs?

- GitHub: https://github.com/helgesverre/dbdump
- Issues: https://github.com/helgesverre/dbdump/issues
- Pull requests welcome!

---

## Additional Resources

- **README.md**: Project overview and quick reference
- **BENCHMARKING.md**: How to run performance tests
- **VERIFIED_PERFORMANCE.md**: Real-world benchmark results
- **CHANGELOG.md**: Version history

---

**Happy dumping!** 🚀
