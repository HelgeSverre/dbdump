# dbdump

<p align="center">
     <img src="./art/header.png" alt="Header Image" />
</p>


[![Tests](https://github.com/helgesverre/dbdump/actions/workflows/test.yml/badge.svg)](https://github.com/helgesverre/dbdump/actions/workflows/test.yml)
[![Release](https://github.com/helgesverre/dbdump/actions/workflows/release.yml/badge.svg)](https://github.com/helgesverre/dbdump/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/helgesverre/dbdump)](https://goreportcard.com/report/github.com/helgesverre/dbdump)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![dbdump](https://img.shields.io/badge/dbdump-0F172A?style=flat&logo=data:image/svg+xml;base64,PHN2ZyB4bWxucz0naHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmcnIHZpZXdCb3g9JzAgMCAyNCAyNCcgZmlsbD0nd2hpdGUnPjxwYXRoIGQ9J004LjAwNDg4IDUuMDAyODFIMTEuMDA0OVYxNC4wMDI4SDguMDA0ODhWMTcuMDAyOEg2LjAwNDg4VjE0LjAwMjhIMy4wMDQ4OFY1LjAwMjgxSDYuMDA0ODhWMi4wMDI4MUg4LjAwNDg4VjUuMDAyODFaTTUuMDA0ODggNy4wMDI4MVYxMi4wMDI4SDkuMDA0ODhWNy4wMDI4MUg1LjAwNDg4Wk0xOC4wMDQ5IDEwLjAwMjhIMjEuMDA0OVYxOS4wMDI4SDE4LjAwNDlWMjIuMDAyOEgxNi4wMDQ5VjE5LjAwMjhIMTMuMDA0OVYxMC4wMDI4SDE2LjAwNDlWNy4wMDI4MUgxOC4wMDQ5VjEwLjAwMjhaTTE1LjAwNDkgMTIuMDAyOFYxNy4wMDI4SDE5LjAwNDlWMTIuMDAyOEgxNS4wMDQ5Wic+PC9wYXRoPjwvc3ZnPgo=)](https://dbdump-five.vercel.app/)

Fast MySQL dump tool that excludes noisy data while preserving complete database structure.

## Why dbdump?

When dumping production databases for development, you often don't need millions of audit log entries, session data, or
cache records. These tables can make dumps take hours and consume gigabytes of space.

**dbdump solves this by:**

- Excluding data from noisy tables (audits, sessions, cache, etc.)
- Always preserving table structure (no broken foreign keys)
- Reducing dump time from hours to minutes
- Making development database refreshes practical

## Installation

### Using Homebrew

```bash
brew install helgesverre/tap/dbdump
```

Upgrade later with `brew upgrade helgesverre/tap/dbdump`. Available for macOS and Linux (Intel and Apple Silicon / ARM64).

### Using Go

If you have Go 1.26+ installed:

```bash
go install github.com/helgesverre/dbdump/cmd/dbdump@latest
```

### Pre-built Binaries

Download the matching versioned asset from the [releases page](https://github.com/helgesverre/dbdump/releases). Asset names include the tag, for example `dbdump-v1.2.0-darwin-arm64.tar.gz`.

#### macOS (Apple Silicon)

```bash
curl -LO https://github.com/helgesverre/dbdump/releases/download/vX.Y.Z/dbdump-vX.Y.Z-darwin-arm64.tar.gz
tar -xzf dbdump-vX.Y.Z-darwin-arm64.tar.gz
chmod +x dbdump-vX.Y.Z-darwin-arm64
sudo mv dbdump-vX.Y.Z-darwin-arm64 /usr/local/bin/dbdump
```

#### macOS (Intel)

```bash
curl -LO https://github.com/helgesverre/dbdump/releases/download/vX.Y.Z/dbdump-vX.Y.Z-darwin-amd64.tar.gz
tar -xzf dbdump-vX.Y.Z-darwin-amd64.tar.gz
chmod +x dbdump-vX.Y.Z-darwin-amd64
sudo mv dbdump-vX.Y.Z-darwin-amd64 /usr/local/bin/dbdump
```

#### Linux (AMD64)

```bash
curl -LO https://github.com/helgesverre/dbdump/releases/download/vX.Y.Z/dbdump-vX.Y.Z-linux-amd64.tar.gz
tar -xzf dbdump-vX.Y.Z-linux-amd64.tar.gz
chmod +x dbdump-vX.Y.Z-linux-amd64
sudo mv dbdump-vX.Y.Z-linux-amd64 /usr/local/bin/dbdump
```

#### Linux (ARM64)

```bash
curl -LO https://github.com/helgesverre/dbdump/releases/download/vX.Y.Z/dbdump-vX.Y.Z-linux-arm64.tar.gz
tar -xzf dbdump-vX.Y.Z-linux-arm64.tar.gz
chmod +x dbdump-vX.Y.Z-linux-arm64
sudo mv dbdump-vX.Y.Z-linux-arm64 /usr/local/bin/dbdump
```

#### Windows (AMD64)

Download the versioned Windows asset from the [releases page](https://github.com/helgesverre/dbdump/releases), extract it, and add the executable to your PATH.

#### Verify Installation

```bash
dbdump --help
```

### Requirements

- **MySQL client tools** - `mysqldump` must be in your PATH (comes with MySQL client)
  - macOS: `brew install mysql-client`
  - Ubuntu/Debian: `sudo apt-get install mysql-client`
  - CentOS/RHEL: `sudo yum install mysql`

### From Source (Developers)

```bash
git clone https://github.com/helgesverre/dbdump.git
cd dbdump
just install
```

Requires Go 1.26+ and [just](https://github.com/casey/just).

## Quick Start

### Interactive Mode (Default)

```bash
# Recommended: Use environment variable for password
export DBDUMP_MYSQL_PWD=yourpassword
dbdump dump -H localhost -u root -d mydb

# Or provide password as flag (less secure)
dbdump dump -H localhost -u root -p password -d mydb
```

This will:

1. Connect to your database
2. Show all tables with sizes
3. Pre-select noisy tables based on patterns
4. Let you customize the selection
5. Dump structure for all tables, data for selected tables

### Auto Mode (Non-Interactive)

```bash
dbdump dump -H localhost -u root -d mydb --auto
```

Uses smart defaults without interaction.

### With Config File

```bash
dbdump dump -H localhost -u root -d mydb --config ./project.yaml
```

## Usage

### Basic Commands

```bash
# Set password securely via environment variable
export DBDUMP_MYSQL_PWD=yourpassword

# Dump database (interactive)
dbdump dump -H localhost -u root -d mydb

# List tables with sizes
dbdump list -H localhost -u root -d mydb

# Dry run (see what would be excluded)
dbdump dump -H localhost -u root -d mydb --dry-run

# Dump with custom output file
dbdump dump -H localhost -u root -d mydb -o backup.sql

# Dump with streaming compression
dbdump dump -H localhost -u root -d mydb --auto --compress gzip

# Dump through an SSH bastion
dbdump dump -H 127.0.0.1 -P 3306 -u root -d mydb \
  --ssh-host bastion.example.com \
  --ssh-user deploy
```

### Connection Options

```bash
-H, --host        Database host (default: 127.0.0.1)
-P, --port        Database port (default: 3306)
-u, --user        Database user
-p, --password    Database password (or use DBDUMP_MYSQL_PWD/MYSQL_PWD env)
-d, --database    Database name
    --profile     Load connection settings from a saved profile (see `dbdump config`)
```

### TLS/SSL Options

```bash
    --tls-mode          TLS mode: disabled, preferred, require, verify-ca, verify-identity
    --tls-ca            Path to the TLS CA certificate (PEM) used to verify the server
    --tls-cert          Path to the client certificate (PEM) for mutual TLS
    --tls-key           Path to the client private key (PEM) for mutual TLS
    --tls-skip-verify   Encrypt but skip server certificate verification (insecure)
    --tls-server-name   Override the hostname verified against the server certificate
```

### Dump Options

```bash
-o, --output           Output file (default: {database}_{timestamp}.sql)
-c, --config           Config file path
    --exclude          Exclude specific table data (repeatable)
    --exclude-pattern  Exclude tables matching pattern (repeatable)
    --auto             Use smart defaults without interaction
    --dry-run          Show what would be dumped without dumping
    --compress         Compression format: auto, none, gzip, zstd
    --ssh-host         SSH bastion host for tunneling to the database
    --ssh-port         SSH bastion port (default: 22)
    --ssh-user         SSH username (defaults to database user)
    --ssh-key          SSH private key path
    --ssh-local-port   Local port for the SSH tunnel (default: auto)
```

### Compression

`dbdump` can stream output as plain SQL, gzip, or zstd. `--compress auto` infers the format from the output filename, while explicit formats override filename inference.

If you omit `--output`, compressed dumps use matching default names such as `mydb_20260331_120000.sql.gz` or `mydb_20260331_120000.sql.zst`.

### SSH Tunneling

You can still create a manual tunnel yourself, but `dbdump` can now manage it directly:

```bash
dbdump dump -H 127.0.0.1 -P 3306 -u root -d mydb \
  --ssh-host bastion.example.com \
  --ssh-user deploy \
  --ssh-key ~/.ssh/id_ed25519 \
  --auto
```

When SSH tunneling is enabled, `-H/--host` and `-P/--port` still describe the database endpoint as seen from the SSH server.

### TLS/SSL

`dbdump` can connect over TLS, applied to both its own inspection connection and the `mysqldump` subprocess:

```bash
# Encrypt and verify the server against a CA
dbdump dump -H db.example.com -u readonly -d myapp \
  --tls-mode verify-identity --tls-ca /etc/ssl/certs/ca.pem --auto

# Mutual TLS (client certificate)
dbdump dump -H db.example.com -u readonly -d myapp \
  --tls-mode verify-ca --tls-ca ca.pem --tls-cert client.pem --tls-key client-key.pem --auto

# Encrypt only, skip verification (dev / self-signed)
dbdump dump -H localhost -u root -d mydb --tls-mode require --tls-skip-verify --auto
```

Modes mirror MySQL's `ssl-mode`: `disabled`, `preferred` (opportunistic), `require` (encrypt, no verification), `verify-ca` (verify the chain), and `verify-identity` (verify the chain and hostname). Behind an SSH tunnel the host becomes `127.0.0.1`, so use `--tls-server-name` to pin the real certificate name. With no `--tls-*` flag, connections behave exactly as before.

### Examples

```bash
# Use environment variable for password
export DBDUMP_MYSQL_PWD=secret
dbdump dump -H prod.example.com -u readonly -d myapp_prod

# Exclude specific tables
dbdump dump -H localhost -u root -d mydb \
  --exclude audits \
  --exclude activity_logs \
  --exclude-pattern "temp_*"

# With project config
dbdump dump -H localhost -u root -d mydb --config ./myproject.yaml

# Auto mode with custom output
dbdump dump -H localhost -u root -d mydb --auto -o daily-backup.sql

# Auto mode with gzip output
dbdump dump -H localhost -u root -d mydb --auto --compress gzip

# Remote dump through SSH
dbdump dump -H 127.0.0.1 -P 3306 -u root -d mydb \
  --ssh-host bastion.example.com \
  --ssh-user deploy \
  --auto
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
dbdump dump -H localhost -u root -d mydb --config ./project.yaml
```

### Default Exclusions

dbdump includes smart defaults for common Laravel tables:

**Exact matches:**

- activity_log
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
- `*_cache`

These defaults are always applied. Later layers can add more excludes, but they do not remove the built-ins.

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

> **Note:** Performance improvements vary based on database structure, server resources, and excluded table sizes.
> Typical improvements range from 5-20% faster than equivalent mysqldump commands.

## Documentation

### User Documentation

- **[USER-GUIDE.md](USER-GUIDE.md)** - Comprehensive user guide with detailed configuration, examples, and
  troubleshooting
- **[CHANGELOG.md](CHANGELOG.md)** - Version history

### Developer Documentation

- **[TESTING_GUIDE.md](TESTING_GUIDE.md)** - Complete testing documentation
- **[BENCHMARKING.md](BENCHMARKING.md)** - Performance testing guide

## Development

### Building

```bash
# Build for current platform
just build

# Build for all platforms
just build-all

# Run tests
just test

# Format code
just fmt
```

### Testing

#### Integration Tests

```bash
# Run the full integration test suite (starts Docker Compose locally)
./test/integration-test.sh

# Cleanup
docker compose down -v
```

See [test/README.md](test/README.md) for detailed testing documentation.

#### Manual Testing

```bash
# Test security (password not in process list)
export DBDUMP_MYSQL_PWD=testpass123
./bin/dbdump dump -H 127.0.0.1 -P 3308 -u root -d testdb --auto
ps aux | grep dbdump  # Should NOT show password

# Verify file permissions (should be 0600)
ls -la testdb_*.sql

# Test data integrity (triggers and events are included; routines are intentionally omitted)
grep -i "CREATE TRIGGER" testdb_*.sql
grep -i "CREATE EVENT" testdb_*.sql
```

### Project Structure

```
dump-tool/
├── cmd/dbdump/          # CLI entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── database/        # Database operations
│   ├── patterns/        # Pattern matching
│   └── ui/              # Interactive UI
├── configs/             # Example configurations
└── justfile             # Build commands
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

### Branding

- Font: [Monda](https://fonts.google.com/specimen/Monda)
- Icon: [Remix - Stock Line](https://remixicon.com/icon/stock-line)
- Colors:
    - Dark: `#0F172A`
    - Icon: `#F9FAFB`
    - Text: `#FFFFFF`
