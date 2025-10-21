# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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

### Performance Improvements
- **17% faster** dump times on average (tested on 3.8-4.3 GB databases)
- **20% higher** throughput (103-104 MB/s → 123-125 MB/s)
- **11% reduction** in system time (more efficient I/O)
- **Zero** memory increase (maintains ~30-50 MB constant usage)

Detailed benchmark results:
- crescat_dump_3 (4.3 GB): 41.48s → 34.34s (-17.2%, +20% throughput)
- crescat_dump_2 (3.8 GB): 37.02s → 30.80s (-16.8%, +19% throughput)

See `OPTIMIZATION_RESULTS.md` for complete analysis.

### Documentation
- Added `OPTIMIZATION_RESULTS.md` - Detailed performance test results
- Added `BENCHMARKING.md` - Comprehensive benchmarking guide

## [1.0.0] - 2025-10-21

### Added
- Initial release of dbdump
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

### v1.0.0 → v1.1.0 (Unreleased)

**No breaking changes.** Performance optimizations are transparent to users.

**New features:**
- Benchmark suite for performance testing
- Enhanced Makefile with benchmark targets

**Recommendations:**
- Update to benefit from 15-20% faster dumps
- Use `make bench` to validate performance in your environment
- Review `BENCHMARKING.md` for best practices

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

## Future Roadmap

### v1.2.0 (Planned)
- Parallel table dumping (2-3x speedup)
- Streaming compression support (gzip/zstd)
- SSH tunnel support
- Cloud storage integration (S3, GCS)

### v1.3.0 (Planned)
- Data anonymization/masking
- Incremental dumps
- Binary format support
- Import helper functionality

See `ROADMAP.md` (coming soon) for detailed plans.

## Support

- **Issues:** [GitHub Issues](https://github.com/helgesverre/dbdump/issues)
- **Discussions:** [GitHub Discussions](https://github.com/helgesverre/dbdump/discussions)
- **Documentation:** [README.md](README.md)

## Contributors

- Helge Sverre (@helgesverre) - Original author
- Claude Code (AI Assistant) - Performance optimizations, documentation, benchmark suite

## License

MIT License - see [LICENSE](LICENSE) file for details.
