# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.4.2] - 2026-08-10

### Fixed
- **[TLS]** Explicit TLS modes now keep their documented semantics when certificate options are also present: `disabled` remains plaintext, `require` does not begin verifying the server, and ambiguous `preferred` plus custom TLS options is rejected before connecting. A standalone `--tls-server-name` now correctly enables identity verification instead of being ignored under `verify-ca` semantics.
- **[Docs]** Corrected non-interactive dry-run examples and website copy buttons so agents and scripts use `--auto --dry-run` consistently.
- **[Docs]** Fixed release-install commands to use the filenames actually contained in the archives, corrected Windows PATH setup, removed unsupported performance claims, and clarified that preserving foreign-key definitions does not guarantee logically complete retained data.
- **[Verification]** Removed stale release-test paths and hardened password process-list checks against vacuous passes.

### Changed
- **[Agents]** Expanded the deployed `/llms.txt` with a safe setup sequence, validation steps, operational guidance, and troubleshooting.

### Dependencies
- Bumped `github.com/klauspost/compress` from 1.19.1 to 1.19.2 (zstd correctness, race, and arm64 decoder fixes plus arm64 huff0 assembly).

## [1.4.1] - 2026-07-27

### Dependencies
- Bumped `github.com/klauspost/compress` from 1.19.0 to 1.19.1 (zstd SnappyConverter literal-copy validation, faster flate `bufio.Reader` decode path, regenerated arm64 zstd assembly)
- Bumped `actions/setup-go` from 6 to 7 in CI and release workflows

## [1.4.0] - 2026-07-14

### Added
- **[CLI]** Added a `version` subcommand and `--version` flag. Release builds stamp the version via ldflags; other builds autopopulate from Go's embedded build info (module version for `go install ...@vX.Y.Z`, VCS commit/time for local builds).
- **[Connectivity]** Added TLS/SSL support via `--tls-mode` (disabled/preferred/require/verify-ca/verify-identity), `--tls-ca`, `--tls-cert`, `--tls-key`, `--tls-skip-verify`, and `--tls-server-name`. Applies to both dbdump's inspection connection and the `mysqldump` subprocess (with a MariaDB `--ssl` fallback), and the settings persist in connection profiles. With no `--tls-*` flag, behavior is unchanged.
- **[Dump]** `--dry-run` now prints the full plan: each table's size and row count, whether its data will be dumped or only its structure preserved, totals, and the resolved output path/compression — instead of just the excluded-table list.
- **[Profiles]** Implemented connection profiles end to end: `dbdump config add <name>` saves the current connection flags, `dbdump config remove <name>` deletes a profile, and `--profile <name>` loads a saved profile for `dump`/`list`. Explicit flags override profile values; the password falls back to `DBDUMP_MYSQL_PWD`/`MYSQL_PWD` when unset. The profiles file is written with 0600 permissions.

### Fixed
- **[Connectivity]** Fixed an indefinite hang when the SSH tunnel process exited before becoming ready (bad key, DNS failure, `ExitOnForwardFailure`): tunnel shutdown no longer deadlocks and the underlying error is surfaced. Also fixed a data race on the ssh stderr buffer.
- **[CLI]** An external `SIGTERM`/`SIGINT` (or kill) during interactive table selection now cancels the dump instead of proceeding with a full one; a selection is honored only when explicitly confirmed with Enter.
- **[Exclusions]** A malformed exclude glob (e.g. `secrets[0-9`) is now rejected up front instead of silently un-excluding the table.
- **[Config]** Config files are decoded strictly: an unknown or misspelled key (e.g. `excludes:`) now errors instead of silently dropping the user's excludes.
- **[CLI]** The `--compress` value and `mysqldump` availability are validated before interactive selection, so invalid input fails fast without discarding the user's choices.
- **[CLI]** A runtime dump failure is no longer printed twice across stdout and stderr.
- **[Dump]** Interrupt/termination signals are trapped across the whole dump, including the finalize/rename window, so an interrupt no longer orphans `.tmp` files.
- **[Inspect]** `formatBytes` now reports petabyte-scale sizes correctly (previously 1024× too small).

### Changed
- **[Performance]** The `--compress gzip` path now uses `klauspost/compress`'s gzip (already a dependency) instead of the standard library — roughly 5× faster deflate for about 4% larger output; the stream is still standard gzip. `zstd` is unchanged and remains the fastest option. Profiling showed dbdump's own code is negligible CPU (it pipes `mysqldump` bytes); compression was the only real hot spot.
- **[Dev]** Consolidated Docker test infrastructure under `docker/` and added a re-runnable smoke test (`docker/smoke-test.sh`, `just smoke`) covering connection profiles across MySQL 5.7/8.0/8.4 + MariaDB and every TLS mode. `just install` now installs to `~/.local/bin` without sudo.

### Removed
- **[Docs]** Removed `SECURITY.md` and all references to it, and removed the stale, version-labelled "Future Roadmap" section.
- **[Internal]** Removed dead code (unused inspector/connection/matcher helpers and the unreachable dry-run path) and de-duplicated the `mysqldump` compatibility-flag handling; no user-facing behavior change.

### Dependencies
- Bumped `golang.org/x/term` from 0.44.0 to 0.45.0 (and indirect `golang.org/x/sys` to 0.47.0)

## [1.3.2] - 2026-07-06

### Dependencies
- Bumped `github.com/klauspost/compress` from 1.18.6 to 1.19.0 (includes the v1.18.7 security fix for an s2.NewDict out-of-bounds read)

## [1.3.1] - 2026-06-23

### Dependencies
- Bumped `golang.org/x/term` from 0.42.0 to 0.44.0 (and indirect `golang.org/x/sys` to 0.46.0)
- Bumped `actions/checkout` from 6 to 7 in CI workflows
- Bumped `codecov/codecov-action` from 6 to 7 in CI workflows

## [1.3.0] - 2026-05-04

### Added
- **[Dump]** Added streaming dump compression with `--compress auto|none|gzip|zstd`, including extension-based inference and compressed default output names.
- **[Connectivity]** Added optional SSH tunnel support via `--ssh-host`, `--ssh-port`, `--ssh-user`, `--ssh-key`, and `--ssh-local-port` for both inspection and dumping.

### Changed
- **[Docs]** Updated the roadmap, changelog links, README, guides, and website docs to the current `v1.2.x` release line and the new dump features.
- **[Verification]** Live end-to-end checks now cover gzip and zstd dump creation/restoration plus SSH-tunneled dumps against Docker-backed services.

### Dependencies
- Bumped `github.com/go-sql-driver/mysql` from 1.9.3 to 1.10.0
- Bumped `github.com/klauspost/compress` from 1.18.5 to 1.18.6
- Bumped CI to Go 1.26 and refreshed indirect Go module graph

## [1.2.0] - 2026-03-31

### Added
- **[CLI]** Added configurable seams so `dbdump dump`, `list`, and `config list` can be unit tested and wired through CI.
- **[Integration]** Added explicit verification for events, event restoration, and commented-on routine handling in the matrix, bringing the total to 14 assertions per database.

### Changed
- **[Dump]** Switched the `mysqldump` helper to create a temporary `defaults-extra-file` and report feature detection failures instead of silently degrading.
- **[Docs]** Synced the README, USER-GUIDE, TESTING_GUIDE, release docs, and workflows with current behavior, TLS limitations, and test counts.
- **[Tests]** Added CLI-wide coverage and expanded integration assertions, raising total tooling coverage to 63.5%.

### Fixed
- **[Docs]** Updated the sample data generator guidance to describe triggers/events and omit routines intentionally.
- **[Release]** Hardened changelog, version metadata, and doc instructions so future releases reference the correct tag (now `v1.2.0`).

## [1.1.1] - 2026-03-02

### Dependencies
- Bumped actions/upload-artifact from 6 to 7
- Bumped actions/download-artifact from 7 to 8
- Bumped github.com/schollz/progressbar/v3 from 3.18.0 to 3.19.0

## [1.1.0] - 2025-12-16

### Added
- **[Website]** New minimal, beginner-friendly documentation website
  - Clean single-page design with sticky navigation
  - Copy-to-clipboard buttons on all code blocks
  - Platform selector tabs for installation instructions
  - Full user guide with collapsible FAQ/troubleshooting sections
  - Responsive design with dark mode support
  - Monda font and Remix stock-line icon branding
- **[Install]** Added `go install` command as installation option
  - `go install github.com/helgesverre/dbdump/cmd/dbdump@latest`

### Changed
- **[Build]** Replaced Makefile with justfile as the sole build system
  - All `make` commands are now `just` commands
  - Updated all documentation to reference `just`
  - GitHub Actions workflows use direct commands (no build tool dependency)
- **[Build]** Updated to `docker compose` (removed hyphenated `docker-compose`)
- **[CI]** Upgraded to Go 1.24 for bubbletea v1.3.10 compatibility

### Fixed
- **[Benchmark]** Fixed benchmark script locale issues
  - Added `LC_ALL=C` to ensure consistent decimal separators
  - Fixed time output capture in benchmark script
- **[Benchmark]** Replaced internal database name with `example_db` in benchmarks

### Dependencies
- Bumped actions/upload-artifact from 5 to 6
- Bumped actions/download-artifact from 6 to 7
- Bumped github.com/spf13/cobra from 1.10.1 to 1.10.2

## [1.0.1] - 2025-10-28

### Fixed
- **[CI/CD]** Fixed CI test failures with Docker Compose and error handling
  - Updated `docker-compose` to `docker compose` for newer Docker CLI
  - Fixed 7 errcheck violations (unchecked error returns for Close/Flush)
  - Made integration tests CI-aware to avoid port conflicts
  - Fixed portable file permission checks using `ls -l` instead of platform-specific `stat`

### Documentation
- Updated all documentation for accuracy and consistency
- Fixed Go version requirement from 1.21+ to 1.23+
- Corrected date inconsistencies throughout documentation
- Emphasized DBDUMP_MYSQL_PWD as preferred environment variable

## [1.0.0] - 2025-10-28

### Security Fixes (CRITICAL)

- **[SECURITY]** Fixed password exposure in process lists
  - Changed from `-p<password>` command-line argument to `MYSQL_PWD` environment variable
  - Passwords no longer visible in `ps aux` or process monitoring tools
  - Prevents credential leakage to other users on the system
- **[SECURITY]** Dump files now created with restrictive permissions (0600)
  - Only file owner can read/write dump files
  - Prevents unauthorized access to database dumps
  - Protects sensitive data from exposure
- **[SECURITY]** Safe DSN construction using mysql.Config
  - Proper escaping of special characters in passwords/usernames
  - Prevents potential injection issues
  - Added connection timeouts (5s connect, 30s read/write)

### Compatibility Fixes

- **[FIX]** MySQL 5.7 compatibility with newer mysqldump clients
  - Removed `--routines` flag to avoid INFORMATION_SCHEMA.LIBRARIES error
  - Ensures dumps work when using MySQL 8.0+ client against MySQL 5.7 servers
  - Added platform flag (`linux/amd64`) to Docker Compose for Apple Silicon compatibility
- **[FIX]** macOS bash 3.2 compatibility for test scripts
  - Replaced bash 4.0+ associative arrays with case statements
  - Test suite now works on macOS default bash installation
- **[FIX]** Added proper mysqldump flags for triggers and events
  - Structure phase explicitly includes: `--triggers`, `--events`
  - Data phase explicitly skips them to prevent duplicates
  - Added `--set-gtid-purged=OFF` and `--column-statistics=0` for cross-version compatibility
  - Added `--hex-blob` for safe binary column handling
- **[FIX]** Ensures triggers and events are properly preserved in dumps
- **[FIX]** Prevents duplicate trigger definitions when restoring dumps

### Bug Fixes

- **[FIX]** List command header separator now displays correctly (was showing null bytes)
- **[FIX]** Removed redundant database connection in dump command
- **[FIX]** Added context cancellation support for clean Ctrl+C handling
  - mysqldump processes now terminate cleanly on interrupt
  - Prevents orphaned mysqldump processes

### Added

- **Environment variable support:** `DBDUMP_MYSQL_PWD` as preferred alternative to `MYSQL_PWD`
  - Avoids polluting standard MySQL environment
  - Falls back to `MYSQL_PWD` if not set
- **Docker Compose test environment** (`docker-compose.yml`)
  - MySQL 5.7, 8.0, 8.4 and MariaDB 10.11 for testing
  - Isolated test databases on different ports
  - Easy integration testing
- **Sample data generation script** (`test/generate-sample-data.sh`)
  - Configurable data sizes (small, medium, large, xlarge)
  - Generates realistic test data with foreign keys, triggers, procedures
  - Includes "noisy" tables for exclusion testing
- **Integration test suite** (`test/integration-test.sh`)
  - Automated testing across all MySQL versions
  - Security verification (password hiding, file permissions)
  - Data integrity tests (triggers, procedures, restoration)
  - Exclusion logic verification
- **Comprehensive documentation**
- Comprehensive benchmark suite (`scripts/benchmark.sh`)
  - Automated performance testing with statistical analysis
  - JSON output for programmatic analysis
  - Support for multiple iterations with avg/median/min/max calculations
- Benchmark documentation (`BENCHMARKING.md`)
  - Complete guide to running and interpreting benchmarks
  - CI/CD integration examples
  - Performance troubleshooting guide
- New Makefile targets:
  - `make bench` - Run benchmark with configurable database and iterations
  - `make bench-quick` - Single-iteration quick test
  - `make bench-all` - Test all available databases
  - `make bench-compare` - Compare before/after results

### Changed
- **PERFORMANCE:** Increased buffer size from 4KB to 256KB for improved write performance
  - Reduces system call overhead by ~95% (950k → 15k syscalls per 3.8 GB dump)
  - Results in ~11% reduction in system time
- **PERFORMANCE:** Added mysqldump optimization flags
  - `--max-allowed-packet=1G` - Handle large rows without splitting
  - `--net-buffer-length=1M` - Larger network buffer for better batching
  - `--skip-comments` - Reduce output size
  - Combined impact: ~6-8% throughput improvement

### Performance Notes

Performance improvements vary based on database structure, server resources, and excluded table sizes:
- **Typical improvement:** 5-10% faster than equivalent mysqldump commands
- **Best case:** Up to 20% improvement with optimal conditions
- **Throughput:** 100-135 MB/s depending on system and database characteristics
- **Memory:** Constant 30-50 MB usage (streaming architecture)

See `BENCHMARKING.md` for detailed benchmark analysis and environmental factors.

### Documentation
- Added `BENCHMARKING.md` - Comprehensive benchmarking guide

## [0.9.0] - 2025-10-21

### Added
- Initial public release of dbdump
- Two-phase dumping strategy (structure + selective data)
- Smart defaults for Laravel applications
  - Pre-configured patterns for audits, telescope, pulse, sessions, cache
- Interactive table selection mode using Bubble Tea TUI
- Auto mode for non-interactive usage
- Config file support (YAML)
- Connection profile management
- Pattern matching (exact + glob patterns)
- Dry-run mode
- Progress tracking and reporting
- CLI commands:
  - `dbdump dump` - Dump database with intelligent exclusions
  - `dbdump list` - List tables with sizes and row counts
  - `dbdump config list` - Show saved connection profiles

### Performance
- Streaming architecture with minimal memory footprint (30-50 MB)
- Throughput: 100-150 MB/s (typical for complex databases)
- Handles databases with millions of rows
- Tested on Laravel production databases up to 10+ GB

### Documentation
- Comprehensive README with examples
- Architecture documentation
- Contributing guidelines
- Build and installation instructions

### Platform Support
- macOS (Intel + Apple Silicon)
- Linux (AMD64 + ARM64)
- Windows (AMD64)

## [0.1.0] - 2025-10-15

### Added
- Initial prototype
- Basic dumping functionality
- Pattern matching for table exclusion

---

## Versioning Strategy

- **MAJOR** (X.0.0): Breaking changes to CLI, configuration, or behavior
- **MINOR** (0.X.0): New features, backward-compatible enhancements
- **PATCH** (0.0.X): Bug fixes, performance improvements, documentation

## Upgrade Notes

### v0.9.0 → v1.0.0

**Security improvements:** v1.0.0 includes critical security fixes. All users should upgrade.

**No breaking changes to CLI or configuration.**

**Recommendations:**
- Use `DBDUMP_MYSQL_PWD` environment variable instead of command-line `-p` flag
- Use `make bench` to validate performance in your environment

## Migration Guides

### From Standard mysqldump

If currently using:
```bash
mysqldump -h host -u user -p password database > dump.sql
```

Replace with:
```bash
dbdump dump -H host -u user -p password -d database --auto
```

**Benefits:**
- 50-60% smaller dumps (excludes noisy tables)
- 30-40% faster (optimized flags + streaming)
- Preserved structure (no broken foreign keys)

### From TablePlus/SequelPro Export

dbdump is faster and more scriptable:
- CLI-friendly for automation
- Configurable exclusions
- Progress tracking
- Dry-run mode for verification

## Support

- **Issues:** [GitHub Issues](https://github.com/helgesverre/dbdump/issues)
- **Discussions:** [GitHub Discussions](https://github.com/helgesverre/dbdump/discussions)
- **Documentation:** [README.md](README.md)

## Contributors

- Helge Sverre (@helgesverre) - Original author
- Claude Code (AI Assistant) - Performance optimizations, documentation, benchmark suite

## License

MIT License - see [LICENSE](LICENSE) file for details.

---

<!-- Version comparison links -->
[Unreleased]: https://github.com/helgesverre/dbdump/compare/v1.4.1...HEAD
[1.4.1]: https://github.com/helgesverre/dbdump/compare/v1.4.0...v1.4.1
[1.4.0]: https://github.com/helgesverre/dbdump/compare/v1.3.2...v1.4.0
[1.3.2]: https://github.com/helgesverre/dbdump/compare/v1.3.1...v1.3.2
[1.3.1]: https://github.com/helgesverre/dbdump/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/helgesverre/dbdump/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/helgesverre/dbdump/compare/v1.1.1...v1.2.0
[1.1.1]: https://github.com/helgesverre/dbdump/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/helgesverre/dbdump/compare/v1.0.1...v1.1.0
[1.0.1]: https://github.com/helgesverre/dbdump/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/helgesverre/dbdump/compare/v0.9.0...v1.0.0
[0.9.0]: https://github.com/helgesverre/dbdump/compare/v0.1.0...v0.9.0
[0.1.0]: https://github.com/helgesverre/dbdump/releases/tag/v0.1.0
