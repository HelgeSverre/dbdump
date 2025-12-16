# Testing Guide for dbdump

This guide explains how to run all tests for dbdump, including security verification and integration tests.

---

## Quick Start

### Option 1: Full Test Suite (Recommended)

```bash
# Runs everything: builds binary, starts Docker, generates data, runs all tests
just test-all
```

### Option 2: Quick Integration Test

```bash
# Faster test with small dataset on MySQL 8.0 only
just test-integration-quick
```

### Option 3: Security Tests Only

```bash
# Just verify security fixes (password hiding, file perms)
just verify-security
```

---

## Just Targets

### Testing Targets

| Command | Description | Time | Use Case |
|---------|-------------|------|----------|
| `just verify-security` | Verify password hiding & file perms | ~5s | Security verification |
| `just test-security` | Same as verify-security | ~5s | Alias |
| `just test-integration-quick` | Quick test (1 DB, small data) | ~2m | Fast feedback |
| `just test-integration` | Full test (4 DBs, all tests) | ~10m | Before release |
| `just test-integration-clean` | Full test + cleanup | ~10m | CI/CD |
| `just test-all` | Unit + integration tests | ~10m | Complete verification |

### Docker Management

| Command | Description |
|---------|-------------|
| `just test-docker-up` | Start all test databases |
| `just test-docker-down` | Stop databases (keep data) |
| `just test-docker-clean` | Stop databases + remove data |

### Data Generation

| Command | Description | Size | Rows |
|---------|-------------|------|------|
| `just test-data-small` | Small dataset | ~10MB | ~15K total |
| `just test-data-medium` | Medium dataset | ~100MB | ~160K total |
| `just test-data-large` | Large dataset | ~1GB | ~1.6M total |
| `just test-data-all` | Generate on all 4 databases | 4x small | - |

---

## Security Verification Details

The `verify-security.sh` script tests:

### Test 1: Password Hidden with DBDUMP_MYSQL_PWD

```bash
export DBDUMP_MYSQL_PWD="secret"
./bin/dbdump dump ... &
ps aux | grep dbdump  # Should NOT show "secret"
```

**Verifies:** Password not in process arguments

### Test 2: Password Hidden with MYSQL_PWD

```bash
export MYSQL_PWD="secret"
./bin/dbdump dump ... &
ps aux | grep dbdump  # Should NOT show "secret"
```

**Verifies:** Backward compatibility with standard MySQL env var

### Test 3: Password Hidden in mysqldump Subprocess

**Requires:** Docker running with test database

```bash
# Starts actual dump, checks both dbdump AND mysqldump processes
ps aux | grep -E "dbdump|mysqldump"  # Should NOT show password
```

**Verifies:** Password passed via MYSQL_PWD env, not `-p` flag

### Test 4: File Permissions

**Requires:** Docker running with test database

```bash
./bin/dbdump dump ... -o test.sql
stat test.sql  # Should show 0600 (-rw-------)
```

**Verifies:** Dump files created with owner-only read/write

### Test 5: Code Inspection

```bash
grep '"-p' internal/database/dumper.go
# Should return nothing (no -p flag usage)
```

**Verifies:** Code doesn't use insecure `-p<password>` pattern

### Test 6: MYSQL_PWD Usage

```bash
grep 'MYSQL_PWD' internal/database/dumper.go
# Should find environment variable being set
```

**Verifies:** Code sets MYSQL_PWD for subprocess

---

## Running Tests Manually

### 1. Security Tests (No Docker Required)

```bash
# Build binary
just build

# Run security verification
./test/verify-security.sh
```

**Expected Output:**
```
========================================
Security Verification Tests
========================================

[PASS] Password NOT visible in process list (DBDUMP_MYSQL_PWD)
[PASS] Password NOT visible in process list (MYSQL_PWD)
[INFO] Docker not running, skipping mysqldump subprocess test
[INFO] Docker not running, skipping file permissions test
[PASS] No '-p' password flag found in dumper.go (correct)
[PASS] Code sets MYSQL_PWD environment variable

========================================
Security Verification Results
========================================
Tests Passed: 4
Tests Failed: 0
========================================

✓ All security tests passed!
```

### 2. Integration Tests (Requires Docker)

```bash
# 1. Start databases
just test-docker-up

# 2. Generate test data
just test-data-small

# 3. Run integration tests
./test/integration-test.sh

# 4. Cleanup
just test-docker-down
```

**OR use one command:**

```bash
just test-integration-quick
```

---

## Interpreting Test Results

### Security Test Results

#### ✅ All Pass (Expected)
```
Tests Passed: 6
Tests Failed: 0
✓ All security tests passed!
```

**Action:** Ready to proceed

#### ❌ Password Visible in Process List
```
[FAIL] Password 'super_secret_password_12345' is visible in process list!
./bin/dbdump dump -h 127.0.0.1 -P 9999 -u root -p super_secret_password_12345
```

**Cause:** Password passed via `-p` flag instead of environment variable

**Fix:** Check `internal/database/dumper.go` - ensure MYSQL_PWD env is used

#### ❌ File Permissions Wrong
```
[FAIL] File permissions are 644 (expected 600)
```

**Cause:** File created with os.Create() instead of os.OpenFile(..., 0600)

**Fix:** Check `internal/database/dumper.go` Dump() function

### Integration Test Results

#### ✅ All Pass (Expected)
```
========================================
Test Results
========================================
Tests Run:    52
Tests Passed: 52
Tests Failed: 0
========================================

[INFO] All tests passed! ✓
```

#### ❌ Some Tests Failed
```
Testing: Triggers in structure dump... ✗
```

**Cause:** mysqldump flags missing (`--triggers`, `--routines`, `--events`)

**Fix:** Check `internal/database/dumper.go` dumpStructure() and dumpData()

---

## Continuous Integration

### GitHub Actions Example

```yaml
name: Tests

on: [push, pull_request]

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      - name: Security Tests
        run: just verify-security

  integration:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      - name: Integration Tests
        run: just test-integration-clean
```

---

## Troubleshooting

### Docker Won't Start

```bash
# Check Docker is running
docker ps

# If not, start Docker Desktop / OrbStack / etc.

# Verify Docker Compose is available
docker compose version
```

### Database Connection Refused

```bash
# Check if containers are running
docker compose ps

# Check if ports are accessible
nc -zv 127.0.0.1 3308

# View container logs
docker compose logs mysql80

# Restart containers
just test-docker-clean
just test-docker-up
```

### Tests Hang or Timeout

```bash
# Kill all related processes
pkill -f dbdump
pkill -f mysqldump

# Clean Docker
just test-docker-clean

# Start fresh
just test-integration-quick
```

### Permission Denied Errors

```bash
# Make scripts executable
chmod +x test/*.sh

# Or run via bash
bash test/verify-security.sh
```

### MySQL Version Compatibility

Different MySQL versions may have different default behaviors:

- **MySQL 5.7:** No `--column-statistics` flag
- **MySQL 8.0+:** Requires `--column-statistics=0` or will error
- **MariaDB:** Different authentication plugins

Our code handles these via conditional flags.

---

## What Gets Tested

### Security Tests (6 tests)

1. ✓ DBDUMP_MYSQL_PWD hides password in `ps aux`
2. ✓ MYSQL_PWD hides password in `ps aux`
3. ✓ mysqldump subprocess doesn't show password
4. ✓ Dump files created with 0600 permissions
5. ✓ Code doesn't use `-p<password>` pattern
6. ✓ Code sets MYSQL_PWD environment variable

### Integration Tests (12 tests × 4 databases = 48 tests)

**Per Database:**
1. ✓ Password not in process list
2. ✓ Dump file permissions 0600
3. ✓ Special characters in password work
4. ✓ Triggers in structure dump
5. ✓ No duplicate triggers
6. ✓ Dump can be restored
7. ✓ Audits table structure preserved
8. ✓ Audits data excluded
9. ✓ Users data included
10. ✓ List command works
11. ✓ Dry run mode works
12. ✓ Custom output file works

> **Note:** Stored procedures test is currently disabled due to MySQL 5.7 compatibility issues with newer mysqldump clients.

**Databases Tested:**
- MySQL 5.7 (port 3307)
- MySQL 8.0 (port 3308)
- MySQL 8.4 (port 3309)
- MariaDB 10.11 (port 3310)

---

## Performance Benchmarking

While not automated in the test suite, you can benchmark manually:

```bash
# Generate large dataset
just test-docker-up
just test-data-large

# Time dbdump
time ./bin/dbdump dump -H 127.0.0.1 -P 3308 -u root -d testdb --auto

# Compare with standard mysqldump
export MYSQL_PWD=testpass123
time mysqldump -h 127.0.0.1 -P 3308 -u root testdb > standard_dump.sql

# Compare file sizes
ls -lh testdb_*.sql standard_dump.sql
```

---

## Before Release Checklist

Run these commands before creating a release:

```bash
# 1. Security verification
just verify-security

# 2. Full integration tests
just test-integration

# 3. Build all platforms
just build-all

# 4. Manual smoke test
export DBDUMP_MYSQL_PWD=testpass123
./bin/dbdump dump -H 127.0.0.1 -P 3308 -u root -d testdb --auto

# 5. Verify dump restores
mysql -h 127.0.0.1 -P 3308 -u root -ptestpass123 -e "CREATE DATABASE test_restore;"
mysql -h 127.0.0.1 -P 3308 -u root -ptestpass123 test_restore < testdb_*.sql
mysql -h 127.0.0.1 -P 3308 -u root -ptestpass123 test_restore -e "SHOW TABLES;"

# 6. Cleanup
just test-docker-clean
```

---

## Test Data Schema

The generated test data includes:

**Core Tables (data included in dumps):**
- `users` - User accounts with names, emails
- `products` - Product catalog with prices
- `orders` - Order records with totals, statuses
- `order_items` - Order line items (FK to orders, products)

**Noisy Tables (excluded by default):**
- `audits` - Audit log entries
- `sessions` - User session data
- `cache` - Cache entries
- `telescope_entries` - Laravel Telescope monitoring data

**Database Objects:**
- Stored Procedure: `get_user_orders(userId INT)`
- Trigger: `after_order_insert` (creates audit on order insert)
- Foreign Keys: Between orders→users, order_items→orders/products
- Indexes: On common query columns

---

## Further Reading

- [test/README.md](test/README.md) - Detailed testing documentation
- [SECURITY.md](SECURITY.md) - Security best practices

---

**Last Updated:** 2025-12-16
**Maintainer:** Helge Sverre
