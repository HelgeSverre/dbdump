# Benchmarking Guide

This document explains how to run performance benchmarks for dbdump.

## Quick Start

```bash
# Run default benchmark (crescat_dump, 3 iterations)
make bench

# Quick single-iteration benchmark
make bench-quick

# Benchmark specific database with custom iterations
make bench DB=kassalapp ITER=5

# Run benchmarks on all test databases
make bench-all
```

## Benchmark Results

All benchmark results are stored in the `benchmark-results/` directory with:
- JSON format for programmatic analysis
- Text summary for human reading
- Timestamp-based filenames for tracking over time

## Understanding the Output

### Timing Metrics

- **Real Time**: Wall-clock time (total duration)
- **User Time**: CPU time spent in user space
- **Sys Time**: CPU time spent in kernel/system calls

### Performance Metrics

- **Throughput (MB/s)**: Output file size divided by real time
  - Higher is better
  - Typical range: 100-150 MB/s for complex Laravel databases

- **File Size**: Size of the generated SQL dump
  - Should be consistent across runs
  - Variations indicate non-deterministic behavior

### Statistics Reported

For each metric, the benchmark calculates:
- **Average**: Mean of all iterations
- **Median**: Middle value (less sensitive to outliers)
- **Min**: Best (fastest) result
- **Max**: Worst (slowest) result

## Benchmark Workflow

### 1. Establish Baseline

Before making any changes:

```bash
# Create baseline with 5 iterations for statistical confidence
make bench DB=crescat_dump ITER=5

# Note the median throughput from the output
# Example: Median throughput: 115.23 MB/s
```

### 2. Make Changes

Edit code, apply optimizations, etc.

### 3. Rebuild and Test

```bash
# Rebuild with changes
make build

# Run benchmark again
make bench DB=crescat_dump ITER=5

# Compare median throughput to baseline
# Calculate improvement: ((new - old) / old) * 100
```

### 4. Compare Results

```bash
# View latest results
ls -lt benchmark-results/ | head -5

# Compare two specific runs
diff benchmark-results/benchmark_crescat_dump_20251021_120000_summary.txt \
     benchmark-results/benchmark_crescat_dump_20251021_130000_summary.txt
```

## Performance Targets

### Expected Performance

Based on real-world testing:

| Database Size | Table Count | Expected Throughput | Expected Duration |
|--------------|-------------|---------------------|-------------------|
| < 1 GB | < 50 | 150-200 MB/s | < 10s |
| 1-5 GB | 50-300 | 100-150 MB/s | 20-50s |
| > 5 GB | > 300 | 80-120 MB/s | > 50s |

### Regression Detection

A performance regression is indicated if:
- Median throughput drops > 10%
- Real time increases > 15%
- Memory usage increases significantly (check with profiling)

## Advanced Benchmarking

### Memory Profiling

```bash
# Build with profiling
go build -o bin/dbdump ./cmd/dbdump

# Run with memory profiling
GODEBUG=gctrace=1 ./bin/dbdump dump -H 127.0.0.1 -u root -d crescat_dump --auto

# Analyze memory profile (if instrumented)
go tool pprof bin/dbdump mem.prof
```

### CPU Profiling

```bash
# Add CPU profiling to main.go temporarily
import "runtime/pprof"

# In main():
f, _ := os.Create("cpu.prof")
pprof.StartCPUProfile(f)
defer pprof.StopCPUProfile()

# Rebuild and run
make build
./bin/dbdump dump -H 127.0.0.1 -u root -d crescat_dump --auto

# Analyze
go tool pprof bin/dbdump cpu.prof
```

### Detailed mysqldump Timing

To see where time is spent in mysqldump:

```bash
# Run with verbose mysqldump
mysqldump --verbose \
  -h 127.0.0.1 -u root \
  --max-allowed-packet=1G \
  --net-buffer-length=1M \
  --skip-comments \
  --no-data crescat_dump \
  > /dev/null 2>&1

# Time each phase separately
time mysqldump --no-data crescat_dump > structure.sql
time mysqldump --no-create-info crescat_dump > data.sql
```

## Environment Variables

Control benchmark behavior with environment variables:

```bash
# Use different MySQL host
MYSQL_HOST=prod.example.com make bench

# Use different credentials
MYSQL_USER=readonly MYSQL_PASSWORD=secret make bench

# Combine with custom database
MYSQL_HOST=remote.db make bench DB=production ITER=3
```

## Continuous Integration

### Automated Regression Testing

Add to your CI pipeline:

```yaml
# .github/workflows/benchmark.yml
name: Performance Benchmark

on: [pull_request]

jobs:
  benchmark:
    runs-on: ubuntu-latest
    services:
      mysql:
        image: mysql:8.0
        env:
          MYSQL_ROOT_PASSWORD: root
          MYSQL_DATABASE: test
        options: >-
          --health-cmd="mysqladmin ping"
          --health-interval=10s
          --health-timeout=5s
          --health-retries=3

    steps:
      - uses: actions/checkout@v2

      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.23

      - name: Load test database
        run: |
          mysql -h 127.0.0.1 -u root -proot test < test/fixtures/benchmark.sql

      - name: Run benchmark
        run: make bench DB=test ITER=5

      - name: Upload results
        uses: actions/upload-artifact@v2
        with:
          name: benchmark-results
          path: benchmark-results/
```

## Interpreting Results

### Good Performance Indicators

✅ **Consistent throughput** across iterations (< 5% variation)
✅ **User time >> System time** (4:1 or higher ratio)
✅ **Linear scaling** with database size
✅ **Median close to average** (indicates few outliers)

### Performance Issues

⚠️ **High variation** (> 10% between min/max)
- Indicates environmental factors (CPU throttling, disk I/O)
- Run more iterations for stable median

⚠️ **Low throughput** (< 80 MB/s for typical databases)
- Check MySQL server performance
- Verify network latency (if remote)
- Profile for bottlenecks

⚠️ **High system time** (> 25% of user time)
- Excessive syscalls or I/O operations
- May need larger buffer sizes
- Check disk performance

## Benchmark Database Setup

### Creating Test Databases

```bash
# Create test database with realistic data
mysql -u root << EOF
CREATE DATABASE IF NOT EXISTS benchmark_small;
USE benchmark_small;

CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255),
    email VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

# Insert test data
INSERT INTO users (name, email)
SELECT
    CONCAT('User ', n),
    CONCAT('user', n, '@example.com')
FROM (
    SELECT a.N + b.N * 10 + c.N * 100 + d.N * 1000 AS n
    FROM
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) a,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) b,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) c,
        (SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) d
) numbers
LIMIT 100000;
EOF
```

### Benchmark Database Sizes

Maintain test databases of various sizes:

- **Small** (< 100 MB): Quick feedback during development
- **Medium** (1-5 GB): Representative of typical use cases
- **Large** (> 5 GB): Stress testing and performance limits

## Optimization History

Track performance improvements over time:

| Date | Version | Optimization | Baseline | After | Improvement |
|------|---------|--------------|----------|-------|-------------|
| 2025-10-21 | v1.0.0 | Initial release | - | 115 MB/s | - |
| 2025-10-21 | v1.1.0 | Buffer size 256KB | 115 MB/s | 125 MB/s | +8.7% |
| 2025-10-21 | v1.1.0 | mysqldump flags | 125 MB/s | 135 MB/s | +8.0% |
| - | - | **Total improvement** | 115 MB/s | 135 MB/s | **+17.4%** |

## Troubleshooting

### Benchmark Script Fails

```bash
# Check script permissions
ls -la scripts/benchmark.sh
# Should show: -rwxr-xr-x

# Make executable if needed
chmod +x scripts/benchmark.sh

# Verify dependencies
which bc  # Required for calculations
which mysql  # Required for database stats
```

### Inconsistent Results

If results vary wildly between runs:

1. **Close other applications** to reduce CPU contention
2. **Disable CPU throttling** (varies by OS)
3. **Use SSD** if possible for consistent I/O
4. **Run more iterations** (5-10) to get stable median
5. **Check MySQL server load** with `SHOW PROCESSLIST`

### Database Not Found

```bash
# List available databases
mysql -u root -e "SHOW DATABASES"

# Verify database exists
mysql -u root -e "USE crescat_dump; SELECT COUNT(*) FROM information_schema.tables"
```

## Best Practices

1. **Always compare median values** (less sensitive to outliers)
2. **Run at least 3 iterations** (5-10 for official benchmarks)
3. **Document system state** (load average, other processes)
4. **Use consistent hardware** for comparisons
5. **Archive baseline results** before making changes
6. **Test on multiple databases** to verify improvements are general

## Example Benchmark Session

```bash
# 1. Create baseline before optimization
make bench DB=crescat_dump ITER=5
# Note median: 115.23 MB/s

# 2. Make optimization (increase buffer size)
# Edit internal/database/dumper.go

# 3. Rebuild and test
make build
make bench DB=crescat_dump ITER=5
# Note median: 125.45 MB/s

# 4. Calculate improvement
# (125.45 - 115.23) / 115.23 * 100 = 8.87% improvement

# 5. Test on other databases to verify
make bench DB=crescat_dump_2 ITER=3
make bench DB=kassalapp ITER=3

# 6. Document in OPTIMIZATION_HISTORY.md
```

## Further Reading

- [Go Profiling](https://go.dev/blog/pprof)
- [MySQL Performance Tuning](https://dev.mysql.com/doc/refman/8.0/en/optimization.html)
- [Benchmarking Best Practices](https://go.dev/blog/benchmarks)
