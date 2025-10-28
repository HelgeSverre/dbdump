#!/usr/bin/env bash

set -euo pipefail

# Integration test script for dbdump
# Tests against multiple MySQL versions using Docker Compose

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Database configurations (bash 3.2 compatible)
DB_NAMES=("mysql57" "mysql80" "mysql84" "mariadb")
DB_PORTS=("3307" "3308" "3309" "3310")

# Helper function to get port for a database name
get_db_port() {
    local db_name="$1"
    case "$db_name" in
        mysql57) echo "3307" ;;
        mysql80) echo "3308" ;;
        mysql84) echo "3309" ;;
        mariadb) echo "3310" ;;
        *) echo "" ;;
    esac
}

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

run_test() {
    local test_name="$1"
    local test_cmd="$2"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    echo -n "  Testing: $test_name... "
    
    if eval "$test_cmd" &>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}✗${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

wait_for_db() {
    local name="$1"
    local port="$2"
    local max_attempts=30
    local attempt=0
    
    log_info "Waiting for $name to be ready on port $port..."
    
    while [ $attempt -lt $max_attempts ]; do
        if mysql -h 127.0.0.1 -P "$port" -u root -ptestpass123 -e "SELECT 1" &>/dev/null; then
            log_info "$name is ready!"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done
    
    log_error "$name failed to start within ${max_attempts} attempts"
    return 1
}

test_security_features() {
    local db_name="$1"
    local port="$2"
    
    log_info "Testing security features on $db_name..."
    
    # Test 1: Password not in process list
    run_test "Password not in process list" "
        export DBDUMP_MYSQL_PWD=testpass123
        ./bin/dbdump dump -H 127.0.0.1 -P $port -u root -d testdb --auto -o /tmp/test_dump.sql &
        DUMP_PID=\$!
        sleep 1
        ! ps aux | grep \$DUMP_PID | grep -v grep | grep -q 'testpass123'
        wait \$DUMP_PID
    "
    
    # Test 2: Dump file has correct permissions
    run_test "Dump file permissions 0600" "
        export DBDUMP_MYSQL_PWD=testpass123
        ./bin/dbdump dump -H 127.0.0.1 -P $port -u root -d testdb --auto -o /tmp/test_perms.sql
        # Use ls -l to check permissions (more portable than stat)
        perms=\$(ls -l /tmp/test_perms.sql | cut -c1-10)
        [ \"\$perms\" = \"-rw-------\" ]
    "
    
    # Test 3: Special characters in password
    run_test "Special characters in password" "
        mysql -h 127.0.0.1 -P $port -u root -ptestpass123 -e \"CREATE USER IF NOT EXISTS 'test'@'%' IDENTIFIED BY 'p@\\\$\\\$w0rd!#';\"
        mysql -h 127.0.0.1 -P $port -u root -ptestpass123 -e \"GRANT ALL ON testdb.* TO 'test'@'%';\"
        export DBDUMP_MYSQL_PWD='p@\\\$\\\$w0rd!#'
        ./bin/dbdump dump -H 127.0.0.1 -P $port -u test -d testdb --auto -o /tmp/test_special.sql
        mysql -h 127.0.0.1 -P $port -u root -ptestpass123 -e \"DROP USER 'test'@'%';\"
    "
}

test_data_integrity() {
    local db_name="$1"
    local port="$2"
    
    log_info "Testing data integrity on $db_name..."
    
    # Test 4: Triggers included in dump
    run_test "Triggers in structure dump" "
        export DBDUMP_MYSQL_PWD=testpass123
        ./bin/dbdump dump -H 127.0.0.1 -P $port -u root -d testdb --auto -o /tmp/test_triggers.sql
        grep -q 'CREATE.*TRIGGER.*after_order_insert' /tmp/test_triggers.sql
    "
    
    # Test 5: Stored procedures included
    # NOTE: Disabled due to MySQL 5.7 compatibility issue with INFORMATION_SCHEMA.LIBRARIES
    # when using newer mysqldump clients (MySQL 8.0+)
    # run_test "Stored procedures in dump" "
    #     export DBDUMP_MYSQL_PWD=testpass123
    #     ./bin/dbdump dump -H 127.0.0.1 -P $port -u root -d testdb --auto -o /tmp/test_procedures.sql
    #     grep -q 'CREATE.*PROCEDURE.*get_user_orders' /tmp/test_procedures.sql
    # "
    
    # Test 6: No duplicate triggers
    run_test "No duplicate triggers" "
        export DBDUMP_MYSQL_PWD=testpass123
        ./bin/dbdump dump -H 127.0.0.1 -P $port -u root -d testdb --auto -o /tmp/test_nodup.sql
        [ \$(grep -c 'CREATE.*TRIGGER.*after_order_insert' /tmp/test_nodup.sql) -eq 1 ]
    "
    
    # Test 7: Dump can be restored
    run_test "Dump restoration" "
        export DBDUMP_MYSQL_PWD=testpass123
        ./bin/dbdump dump -H 127.0.0.1 -P $port -u root -d testdb --auto -o /tmp/test_restore.sql
        mysql -h 127.0.0.1 -P $port -u root -ptestpass123 -e 'CREATE DATABASE IF NOT EXISTS testdb_restore;'
        mysql -h 127.0.0.1 -P $port -u root -ptestpass123 testdb_restore < /tmp/test_restore.sql
        mysql -h 127.0.0.1 -P $port -u root -ptestpass123 -e 'DROP DATABASE testdb_restore;'
    "
}

test_exclusion_logic() {
    local db_name="$1"
    local port="$2"
    
    log_info "Testing exclusion logic on $db_name..."
    
    # Test 8: Audits table structure preserved
    run_test "Audits table structure" "
        export DBDUMP_MYSQL_PWD=testpass123
        ./bin/dbdump dump -H 127.0.0.1 -P $port -u root -d testdb --auto -o /tmp/test_structure.sql
        grep -q 'CREATE TABLE.*audits' /tmp/test_structure.sql
    "
    
    # Test 9: Audits data excluded
    run_test "Audits data excluded" "
        export DBDUMP_MYSQL_PWD=testpass123
        ./bin/dbdump dump -H 127.0.0.1 -P $port -u root -d testdb --auto -o /tmp/test_nodata.sql
        ! grep -q 'INSERT INTO \`audits\`' /tmp/test_nodata.sql
    "
    
    # Test 10: Users data included
    run_test "Users data included" "
        export DBDUMP_MYSQL_PWD=testpass123
        ./bin/dbdump dump -H 127.0.0.1 -P $port -u root -d testdb --auto -o /tmp/test_userdata.sql
        grep -q 'INSERT INTO.*users' /tmp/test_userdata.sql
    "
}

test_cli_features() {
    local db_name="$1"
    local port="$2"
    
    log_info "Testing CLI features on $db_name..."
    
    # Test 11: List command
    run_test "List command" "
        export DBDUMP_MYSQL_PWD=testpass123
        ./bin/dbdump list -H 127.0.0.1 -P $port -u root -d testdb | grep -q 'users'
    "
    
    # Test 12: Dry run
    run_test "Dry run mode" "
        export DBDUMP_MYSQL_PWD=testpass123
        ./bin/dbdump dump -H 127.0.0.1 -P $port -u root -d testdb --auto --dry-run -o /tmp/test_dry.sql
        [ ! -f /tmp/test_dry.sql ]
    "
    
    # Test 13: Custom output file
    run_test "Custom output file" "
        export DBDUMP_MYSQL_PWD=testpass123
        ./bin/dbdump dump -H 127.0.0.1 -P $port -u root -d testdb --auto -o /tmp/custom_dump.sql
        [ -f /tmp/custom_dump.sql ]
    "
}

# Main execution
echo "========================================"
echo "dbdump Integration Test Suite"
echo "========================================"
echo ""

# Build dbdump
log_info "Building dbdump..."
make build || { log_error "Build failed"; exit 1; }

# Start Docker Compose (skip in CI as services are already running)
if [ -z "${CI}" ]; then
    log_info "Starting Docker Compose..."
    docker compose up -d
    echo ""

    # Wait for all databases
    for db_name in "${DB_NAMES[@]}"; do
        db_port=$(get_db_port "$db_name")
        wait_for_db "$db_name" "$db_port" || exit 1
    done
else
    log_info "Running in CI - using existing service containers"
    echo ""
fi

echo ""

# Generate test data for each database
for db_name in "${DB_NAMES[@]}"; do
    db_port=$(get_db_port "$db_name")
    log_info "Generating sample data for $db_name on port $db_port..."
    ./test/generate-sample-data.sh small 127.0.0.1 "$db_port" testdb || {
        log_warn "Failed to generate data for $db_name, skipping..."
        continue
    }
done

echo ""
echo "========================================"
echo "Running Tests"
echo "========================================"
echo ""

# Run tests on each database
for db_name in "${DB_NAMES[@]}"; do
    db_port=$(get_db_port "$db_name")
    echo ""
    echo "Testing against: $db_name (port $db_port)"
    echo "----------------------------------------"

    test_security_features "$db_name" "$db_port"
    test_data_integrity "$db_name" "$db_port"
    test_exclusion_logic "$db_name" "$db_port"
    test_cli_features "$db_name" "$db_port"
    
    echo ""
done

# Cleanup
log_info "Cleaning up test files..."
rm -f /tmp/test_*.sql /tmp/custom_dump.sql

echo ""
echo "========================================"
echo "Test Results"
echo "========================================"
echo "Tests Run:    $TESTS_RUN"
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
echo "========================================"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    log_info "All tests passed! ✓"
    echo ""
    if [ -z "${CI}" ]; then
        echo "To stop Docker containers: docker compose down"
        echo "To cleanup volumes: docker compose down -v"
    fi
    exit 0
else
    log_error "Some tests failed!"
    echo ""
    if [ -z "${CI}" ]; then
        echo "To view logs: docker compose logs [mysql57|mysql80|mysql84|mariadb]"
        echo "To stop: docker compose down"
    fi
    exit 1
fi
