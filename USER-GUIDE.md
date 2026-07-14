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

The release assets are versioned. Download the matching archive from the releases page, for example `dbdump-v1.2.0-darwin-arm64.tar.gz`.

```bash
curl -LO https://github.com/helgesverre/dbdump/releases/download/vX.Y.Z/dbdump-vX.Y.Z-darwin-arm64.tar.gz
tar -xzf dbdump-vX.Y.Z-darwin-arm64.tar.gz
chmod +x dbdump-vX.Y.Z-darwin-arm64
sudo mv dbdump-vX.Y.Z-darwin-arm64 /usr/local/bin/dbdump
```

### From Source

```bash
git clone https://github.com/helgesverre/dbdump.git
cd dbdump
just install
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

# Stream-compressed dump
dbdump dump -u root -p yourpassword -d mydatabase --compress gzip
```

For security, you should use environment variables for passwords:

```bash
# Recommended: Use dbdump-specific variable
export DBDUMP_MYSQL_PWD=yourpassword
dbdump dump -H localhost -u root -d mydatabase

# Alternative: Standard MySQL variable (fallback)
export MYSQL_PWD=yourpassword
dbdump dump -H localhost -u root -d mydatabase
```

---

## Configuration

dbdump uses a flexible, layered configuration system that allows you to customize which tables to exclude at multiple levels.

### Configuration Priority

Configurations are merged in this order:

1. **Built-in defaults** (embedded in the binary)
2. **Global user config** (`~/.dbdump.yaml`)
3. **Project-specific config** (specified with `--config` flag)
4. **CLI flags** (`--exclude`, `--exclude-pattern`)

Later layers can add exclusions, but they do not remove earlier ones.

### Default Excludes

dbdump comes with sensible defaults for common Laravel and PHP framework tables:

**Exact table names:**
- `activity_log`
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
- `*_cache` - Any table ending with "_cache"

These defaults are always loaded first and can be supplemented by your custom configs.

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
dbdump dump -H localhost -u root -d mydb --config ./myproject.yaml
```

**When to use:**
- Project-specific large tables
- Tables unique to your application
- Different requirements per database

### CLI Flags

Override everything at runtime with flags:

```bash
# Exclude specific tables
dbdump dump -H localhost -u root -d mydb --exclude users --exclude orders

# Exclude by pattern
dbdump dump -H localhost -u root -d mydb --exclude-pattern "temp_*" --exclude-pattern "old_*"

# Combine both
dbdump dump -H localhost -u root -d mydb \
  --exclude sessions \
  --exclude cache \
  --exclude-pattern "log_*"

# Use built-in SSH tunneling
dbdump dump -H 127.0.0.1 -P 3306 -u root -d mydb \
  --ssh-host bastion.example.com \
  --ssh-user deploy
```

**When to use:**
- One-off dumps with special requirements
- Testing different exclude patterns
- Quick overrides without editing config files

### Compression

The `dump` command supports streaming compression:

- `--compress auto` infers the format from the output path
- `--compress none` writes plain SQL
- `--compress gzip` writes gzip-compressed SQL
- `--compress zstd` writes zstd-compressed SQL

If you do not pass `--output`, the generated filename picks the right extension automatically.

### Built-In SSH Tunneling

You can ask `dbdump` to create and tear down a local SSH tunnel for the run:

```bash
dbdump dump -H 127.0.0.1 -P 3306 -u root -d mydb \
  --ssh-host bastion.example.com \
  --ssh-user deploy \
  --ssh-key ~/.ssh/id_ed25519
```

The database host and port flags still describe the MySQL endpoint reachable from the SSH server. The tool forwards that endpoint to a temporary localhost port and uses it for both metadata inspection and `mysqldump`.

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
| `--password` | `-p` | Database password | $DBDUMP_MYSQL_PWD or $MYSQL_PWD | No |
| `--database` | `-d` | Database name | - | Yes |

**Dump Flags:**

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--output` | `-o` | Output file path | `{database}_{timestamp}.sql` |
| `--config` | `-c` | Project config file | - |
| `--exclude` | - | Exclude specific table (repeatable) | - |
| `--exclude-pattern` | - | Exclude pattern (repeatable) | - |
| `--auto` | - | Use defaults without interaction | false |
| `--dry-run` | - | Show what would be excluded without dumping | false |

**Examples:**

```bash
# Interactive mode (default)
dbdump dump -H localhost -u root -d mydb

# Auto mode with custom output
dbdump dump -H localhost -u root -d mydb --auto -o production-backup.sql

# Remote database
dbdump dump -H db.example.com -P 3306 -u dbuser -d mydb

# With project config
dbdump dump -H localhost -u root -d mydb --config ./project-config.yaml

# Dry run to preview
dbdump dump -H localhost -u root -d mydb --dry-run

# Additional CLI excludes
dbdump dump -H localhost -u root -d mydb --exclude users --exclude-pattern "test_*"
```

### list Command

Display all tables in the database with their sizes and row counts.

**Basic Syntax:**

```bash
dbdump list [flags]
```

**Example:**

```bash
dbdump list -H localhost -u root -d mydb
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

Show all saved connection profiles stored in `~/.config/dbdump/profiles.yaml`.

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
    Password: (saved)
```

### config add Command

Save the current connection flags as a named profile:

```bash
dbdump config add production -H db.example.com -P 3306 -u produser -p secret -d maindb
```

The profiles file (`~/.config/dbdump/profiles.yaml`) is written with `0600`
permissions. A stored password is optional; omit `-p` to resolve it from
`DBDUMP_MYSQL_PWD`/`MYSQL_PWD` at dump time instead.

### config remove Command

Delete a saved profile:

```bash
dbdump config remove production
```

### Using a profile

Pass `--profile <name>` to `dump` or `list` to load a saved profile. Any explicit
connection flag overrides the corresponding profile value:

```bash
dbdump dump --profile production --auto
dbdump list --profile staging
dbdump dump --profile production -d otherdb   # override just the database
```

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
dbdump dump -H localhost -u root -d mydb
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
dbdump dump -H localhost -u root -d laravel_db --config laravel-config.yaml --auto
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
dbdump dump -H localhost -u root -d mydb \
  --exclude user_activity \
  --exclude api_logs \
  --exclude-pattern "temp_*" \
  -o quick-backup.sql
```

### Example 5: Inspecting Before Dumping

Check your database first:

```bash
# See all tables and sizes
dbdump list -H localhost -u root -d mydb

# Do a dry run
dbdump dump -H localhost -u root -d mydb --dry-run

# If happy, do the real dump
dbdump dump -H localhost -u root -d mydb
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
| `q` or `Ctrl+C` | Cancel and exit |

### Tips

- **Pre-selected tables** are marked based on your configuration (defaults + global + project)
- You can toggle any table on or off
- Tables shown in the selector will have their DATA excluded (structure is always kept)
- Review the sizes to identify large tables you might want to exclude

---

## Connection Profiles

`dbdump config list` reads saved profiles from `~/.config/dbdump/profiles.yaml`.
Create profiles with `dbdump config add <name>` (saving the current connection
flags), delete them with `dbdump config remove <name>`, and select one for a dump
or listing with `--profile <name>`. Explicit connection flags override profile
values, and the password falls back to `DBDUMP_MYSQL_PWD`/`MYSQL_PWD` when neither
a flag nor the profile provides one. The profiles file is written with `0600`
permissions.

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
dbdump list -H localhost -u root -d mydb

# Dry run with pattern
dbdump dump -H localhost -u root -d mydb --exclude-pattern "your_pattern_*" --dry-run
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
dbdump dump -H localhost -u root -d mydb --exclude-pattern "*"
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
dbdump dump -H localhost -u root -d mydb --dry-run
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
- Always use environment variables (`DBDUMP_MYSQL_PWD`) for passwords instead of CLI flags
- Dump files are created with restrictive permissions (0600) for security

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
