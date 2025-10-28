# Testing dbdump

This directory contains testing tools and scripts for dbdump.

## Quick Start

```bash
# Start test databases
docker compose up -d

# Generate sample data (medium size)
./test/generate-sample-data.sh medium 127.0.0.1 3308 testdb

# Run integration tests
./test/integration-test.sh

# Stop test databases
docker compose down
```

---

## Test Environment

### Docker Compose Setup

The `docker-compose.yml` provides isolated test databases:

| Database | Version | Port | Container Name |
|----------|---------|------|----------------|
| MySQL    | 5.7     | 3307 | dbdump-mysql57 |
| MySQL    | 8.0     | 3308 | dbdump-mysql80 |
| MySQL    | 8.4     | 3309 | dbdump-mysql84 |
| MariaDB  | 10.11   | 3310 | dbdump-mariadb |

**Credentials:**
- Username: `root`
- Password: `testpass123`
- Database: `testdb`

### Starting the Environment

```bash
# Start all databases
docker compose up -d

# Start specific database
docker compose up -d mysql80

# View logs
docker compose logs -f mysql80

# Stop all
docker compose down

# Stop and remove volumes (clean slate)
docker compose down -v
```

---

## Sample Data Generation

### Usage

```bash
./test/generate-sample-data.sh [size] [host] [port] [database]
```

### Size Presets

| Size    | Users  | Orders | Audits  | Approx Size |
|---------|--------|--------|---------|-------------|
| small   | 1K     | 10K    | 5K      | ~10 MB      |
| medium  | 10K    | 100K   | 50K     | ~100 MB     |
| large   | 100K   | 1M     | 500K    | ~1 GB       |
| xlarge  | 1M     | 10M    | 5M      | ~10 GB      |

### What Gets Created

**Tables with data:**
- `users` - User accounts
- `products` - Product catalog
- `orders` - Order records
- `order_items` - Order line items

**Noisy tables (for exclusion testing):**
- `audits` - Audit log entries (should be excluded)
- `sessions` - Session data (should be excluded)
- `cache` - Cache entries (should be excluded)
- `telescope_entries` - Laravel Telescope data (should be excluded)

**Database features:**
- Stored procedure: `get_user_orders`
- Trigger: `after_order_insert` (creates audit log on order insert)
- Foreign keys between tables
- Indexes on common columns

### Examples

```bash
# Generate small dataset on MySQL 8.0
./test/generate-sample-data.sh small 127.0.0.1 3308 testdb

# Generate large dataset on MariaDB
./test/generate-sample-data.sh large 127.0.0.1 3310 testdb

# Generate medium dataset using environment variable
export MYSQL_ROOT_PASSWORD=testpass123
./test/generate-sample-data.sh medium
```

---

## Integration Tests

### Running Tests

```bash
# Run full test suite on all databases
./test/integration-test.sh

# Prerequisites:
# - docker-compose must be running
# - dbdump must be built (make build)
```

### What Gets Tested

#### Security Tests
- ✓ Password not visible in process list
- ✓ Dump files created with 0600 permissions
- ✓ Special characters in passwords handled correctly

#### Data Integrity Tests
- ✓ Triggers included in structure dump
- ✓ Stored procedures included in dump
- ✓ No duplicate triggers/procedures
- ✓ Dumps can be restored successfully

#### Exclusion Logic Tests
- ✓ Excluded table structure preserved
- ✓ Excluded table data not included
- ✓ Non-excluded table data included

#### CLI Feature Tests
- ✓ List command works
- ✓ Dry run mode works
- ✓ Custom output file naming

### Test Output

```
========================================
dbdump Integration Test Suite
========================================

[INFO] Building dbdump...
[INFO] Starting Docker Compose...
[INFO] Waiting for mysql80 to be ready on port 3308...
[INFO] mysql80 is ready!

Testing against: mysql80 (port 3308)
----------------------------------------
[INFO] Testing security features on mysql80...
  Testing: Password not in process list... ✓
  Testing: Dump file permissions 0600... ✓
  Testing: Special characters in password... ✓

========================================
Test Results
========================================
Tests Run:    52
Tests Passed: 52
Tests Failed: 0
========================================

[INFO] All tests passed! ✓
```

---

## Manual Testing

### Test Security Features

```bash
# Start database
docker-compose up -d mysql80

# Generate test data
./test/generate-sample-data.sh small 127.0.0.1 3308 testdb

# Test with environment variable
export DBDUMP_MYSQL_PWD=testpass123
./bin/dbdump dump -H 127.0.0.1 -P 3308 -u root -d testdb --auto

# Verify password not in process list
ps aux | grep dbdump  # Should NOT show password

# Verify file permissions
ls -la testdb_*.sql  # Should show -rw------- (600)
```

### Test Data Integrity

```bash
# Dump database
export DBDUMP_MYSQL_PWD=testpass123
./bin/dbdump dump -H 127.0.0.1 -P 3308 -u root -d testdb -o test_dump.sql

# Check for triggers
grep -i "CREATE TRIGGER" test_dump.sql

# Check for procedures
grep -i "CREATE PROCEDURE" test_dump.sql

# Count trigger occurrences (should be 1)
grep -c "after_order_insert" test_dump.sql

# Restore to new database
mysql -h 127.0.0.1 -P 3308 -u root -ptestpass123 -e "CREATE DATABASE test_restore;"
mysql -h 127.0.0.1 -P 3308 -u root -ptestpass123 test_restore < test_dump.sql

# Verify restoration
mysql -h 127.0.0.1 -P 3308 -u root -ptestpass123 test_restore -e "SHOW TABLES;"
mysql -h 127.0.0.1 -P 3308 -u root -ptestpass123 test_restore -e "SHOW TRIGGERS;"
mysql -h 127.0.0.1 -P 3308 -u root -ptestpass123 test_restore -e "SHOW PROCEDURE STATUS WHERE Db = 'test_restore';"

# Cleanup
mysql -h 127.0.0.1 -P 3308 -u root -ptestpass123 -e "DROP DATABASE test_restore;"
```

### Test Exclusion Logic

```bash
# Dump with auto mode (excludes noisy tables)
export DBDUMP_MYSQL_PWD=testpass123
./bin/dbdump dump -H 127.0.0.1 -P 3308 -u root -d testdb --auto -o auto_dump.sql

# Verify audits table structure exists
grep "CREATE TABLE.*audits" auto_dump.sql

# Verify audits data NOT included
grep "INSERT INTO.*audits" auto_dump.sql  # Should return nothing

# Verify users data IS included
grep "INSERT INTO.*users" auto_dump.sql  # Should find inserts

# List all tables
./bin/dbdump list -H 127.0.0.1 -P 3308 -u root -d testdb
```

### Test Across Versions

```bash
# Test MySQL 5.7
./bin/dbdump dump -H 127.0.0.1 -P 3307 -u root -d testdb --auto

# Test MySQL 8.0
./bin/dbdump dump -H 127.0.0.1 -P 3308 -u root -d testdb --auto

# Test MySQL 8.4
./bin/dbdump dump -H 127.0.0.1 -P 3309 -u root -d testdb --auto

# Test MariaDB
./bin/dbdump dump -H 127.0.0.1 -P 3310 -u root -d testdb --auto
```

---

## Benchmarking

### Using Sample Data

```bash
# Generate large dataset for benchmarking
./test/generate-sample-data.sh large 127.0.0.1 3308 testdb

# Run benchmark
time ./bin/dbdump dump -H 127.0.0.1 -P 3308 -u root -d testdb --auto

# Compare with standard mysqldump
time mysqldump -h 127.0.0.1 -P 3308 -u root -ptestpass123 testdb > standard_dump.sql
```

### Using Make Targets

```bash
# Run benchmarks (if Make targets are set up)
make bench DB=testdb ITER=5
```

---

## Troubleshooting

### Database Won't Start

```bash
# Check logs
docker compose logs mysql80

# Restart specific container
docker compose restart mysql80

# Clean start
docker compose down -v
docker compose up -d
```

### Connection Refused

```bash
# Wait for database to be ready
while ! mysql -h 127.0.0.1 -P 3308 -u root -ptestpass123 -e "SELECT 1" &>/dev/null; do
    echo "Waiting for MySQL..."
    sleep 2
done
echo "MySQL is ready!"
```

### Sample Data Generation Fails

```bash
# Check database is accessible
mysql -h 127.0.0.1 -P 3308 -u root -ptestpass123 -e "SELECT 1"

# Verify database exists
mysql -h 127.0.0.1 -P 3308 -u root -ptestpass123 -e "SHOW DATABASES;"

# Create database if missing
mysql -h 127.0.0.1 -P 3308 -u root -ptestpass123 -e "CREATE DATABASE IF NOT EXISTS testdb;"
```

### Integration Tests Fail

```bash
# Rebuild dbdump
make clean build

# Regenerate test data
./test/generate-sample-data.sh small 127.0.0.1 3308 testdb

# Run tests with verbose output
bash -x ./test/integration-test.sh
```

---

## Cleanup

```bash
# Stop containers
docker compose down

# Remove volumes (deletes all data)
docker compose down -v

# Remove test dump files
rm -f *.sql test_*.sql /tmp/test_*.sql
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      
      - name: Build
        run: make build
      
      - name: Start test databases
        run: docker compose up -d

      - name: Wait for databases
        run: sleep 30

      - name: Generate test data
        run: ./test/generate-sample-data.sh small 127.0.0.1 3308 testdb

      - name: Run integration tests
        run: ./test/integration-test.sh

      - name: Cleanup
        if: always()
        run: docker compose down -v
```

---

## Contributing Tests

When adding new features, please:

1. Add test cases to `integration-test.sh`
2. Update this README with new test scenarios
3. Ensure tests pass on all database versions
4. Document any new test fixtures or requirements

---

For more information, see the main [README.md](../README.md) and [SECURITY.md](../SECURITY.md).
