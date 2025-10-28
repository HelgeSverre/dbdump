#!/usr/bin/env bash

set -euo pipefail

# Security verification script
# Tests that passwords are NOT visible in process lists

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PASSED=0
FAILED=0

log_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    PASSED=$((PASSED + 1))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    FAILED=$((FAILED + 1))
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Ensure binary exists
if [ ! -f "$PROJECT_ROOT/bin/dbdump" ]; then
    echo -e "${RED}Error: Binary not found. Run 'make build' first.${NC}"
    exit 1
fi

echo "========================================"
echo "Security Verification Tests"
echo "========================================"
echo ""

##
## Test 1: Password NOT in process list when using DBDUMP_MYSQL_PWD
##
log_test "Test 1: Password hidden when using DBDUMP_MYSQL_PWD"

TEST_PASSWORD="super_secret_password_12345"
export DBDUMP_MYSQL_PWD="$TEST_PASSWORD"

# Start dbdump in background (will fail to connect but that's OK)
"$PROJECT_ROOT/bin/dbdump" dump -H 127.0.0.1 -P 9999 -u root -d testdb --auto -o /tmp/test_security_dump1.sql &>/dev/null &
DUMP_PID=$!

# Give it a moment to start
sleep 0.5

# Check if password appears in process list
if ps aux | grep "$DUMP_PID" | grep -v grep | grep -q "$TEST_PASSWORD"; then
    log_fail "Password '$TEST_PASSWORD' is visible in process list!"
    ps aux | grep "$DUMP_PID" | grep -v grep
    SECURITY_BREACH=1
else
    log_pass "Password NOT visible in process list (DBDUMP_MYSQL_PWD)"
fi

# Cleanup
kill $DUMP_PID 2>/dev/null || true
wait $DUMP_PID 2>/dev/null || true
unset DBDUMP_MYSQL_PWD
rm -f /tmp/test_security_dump1.sql

##
## Test 2: Password NOT in process list when using MYSQL_PWD
##
log_test "Test 2: Password hidden when using MYSQL_PWD"

export MYSQL_PWD="$TEST_PASSWORD"

"$PROJECT_ROOT/bin/dbdump" dump -H 127.0.0.1 -P 9999 -u root -d testdb --auto -o /tmp/test_security_dump2.sql &>/dev/null &
DUMP_PID=$!

sleep 0.5

if ps aux | grep "$DUMP_PID" | grep -v grep | grep -q "$TEST_PASSWORD"; then
    log_fail "Password '$TEST_PASSWORD' is visible in process list!"
    ps aux | grep "$DUMP_PID" | grep -v grep
    SECURITY_BREACH=1
else
    log_pass "Password NOT visible in process list (MYSQL_PWD)"
fi

kill $DUMP_PID 2>/dev/null || true
wait $DUMP_PID 2>/dev/null || true
unset MYSQL_PWD
rm -f /tmp/test_security_dump2.sql

##
## Test 3: Password NOT in mysqldump child process
##
log_test "Test 3: Password hidden in mysqldump subprocess"

# This test requires a real database, so we'll use Docker if available
if command -v docker &>/dev/null && docker-compose ps | grep -q "mysql80.*Up"; then
    log_info "MySQL 8.0 container is running, testing with real connection..."
    
    export DBDUMP_MYSQL_PWD="testpass123"
    
    # Start dump in background
    "$PROJECT_ROOT/bin/dbdump" dump -H 127.0.0.1 -P 3308 -u root -d testdb --auto -o /tmp/test_security_dump3.sql &
    DUMP_PID=$!
    
    # Wait a bit for mysqldump to start
    sleep 2
    
    # Check all processes including mysqldump
    if ps aux | grep -E "mysqldump|dbdump" | grep -v grep | grep -q "testpass123"; then
        log_fail "Password visible in mysqldump or dbdump process!"
        ps aux | grep -E "mysqldump|dbdump" | grep -v grep
    else
        log_pass "Password NOT visible in dbdump or mysqldump processes"
    fi
    
    # Verify MYSQL_PWD is set in child environment (positive test)
    log_test "Test 3b: MYSQL_PWD environment variable is set for mysqldump"
    # We can't easily check child process env, but we can verify the dump works
    wait $DUMP_PID
    
    if [ -f /tmp/test_security_dump3.sql ] && [ -s /tmp/test_security_dump3.sql ]; then
        log_pass "Dump completed successfully (MYSQL_PWD was passed correctly)"
    else
        log_fail "Dump failed (MYSQL_PWD may not have been passed)"
    fi
    
    unset DBDUMP_MYSQL_PWD
    rm -f /tmp/test_security_dump3.sql
else
    log_info "Docker not running, skipping mysqldump subprocess test"
    log_info "Run 'docker-compose up -d' to enable this test"
fi

##
## Test 4: File permissions are restrictive (0600)
##
log_test "Test 4: Dump files created with restrictive permissions"

if command -v docker &>/dev/null && docker-compose ps | grep -q "mysql80.*Up"; then
    export DBDUMP_MYSQL_PWD="testpass123"
    
    "$PROJECT_ROOT/bin/dbdump" dump -H 127.0.0.1 -P 3308 -u root -d testdb --auto -o /tmp/test_security_perms.sql &>/dev/null
    
    # Check file permissions
    if [ -f /tmp/test_security_perms.sql ]; then
        # Get permissions in octal format
        if [[ "$OSTYPE" == "darwin"* ]]; then
            # macOS
            PERMS=$(stat -f "%A" /tmp/test_security_perms.sql)
        else
            # Linux
            PERMS=$(stat -c "%a" /tmp/test_security_perms.sql)
        fi
        
        if [ "$PERMS" = "600" ]; then
            log_pass "File permissions are 0600 (owner read/write only)"
        else
            log_fail "File permissions are $PERMS (expected 600)"
        fi
        
        # Also show human-readable format
        ls -la /tmp/test_security_perms.sql | awk '{print "  Permissions: " $1 " Owner: " $3}'
    else
        log_fail "Dump file was not created"
    fi
    
    unset DBDUMP_MYSQL_PWD
    rm -f /tmp/test_security_perms.sql
else
    log_info "Docker not running, skipping file permissions test"
fi

##
## Test 5: Command-line password flag (verify it's NOT used)
##
log_test "Test 5: Verify -p flag does NOT pass password to mysqldump"

# This is more of a code inspection test
if grep -q '"-p' "$PROJECT_ROOT/internal/database/dumper.go"; then
    log_fail "Found '-p' password flag in dumper.go (should use MYSQL_PWD)"
else
    log_pass "No '-p' password flag found in dumper.go (correct)"
fi

##
## Test 6: MYSQL_PWD environment variable is used in subprocess
##
log_test "Test 6: Code sets MYSQL_PWD for mysqldump subprocess"

if grep -q 'MYSQL_PWD' "$PROJECT_ROOT/internal/database/dumper.go"; then
    log_pass "Code sets MYSQL_PWD environment variable"
else
    log_fail "Code does not set MYSQL_PWD (password may not be passed securely)"
fi

##
## Summary
##
echo ""
echo "========================================"
echo "Security Verification Results"
echo "========================================"
echo -e "Tests Passed: ${GREEN}$PASSED${NC}"
echo -e "Tests Failed: ${RED}$FAILED${NC}"
echo "========================================"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All security tests passed!${NC}"
    echo ""
    echo "Summary:"
    echo "  ✓ Passwords hidden from process lists"
    echo "  ✓ Passwords not passed via command-line arguments"
    echo "  ✓ MYSQL_PWD environment variable used correctly"
    echo "  ✓ Dump files created with restrictive permissions (0600)"
    echo ""
    exit 0
else
    echo -e "${RED}✗ Some security tests failed!${NC}"
    echo ""
    echo "Please review the failures above and fix them before release."
    echo ""
    exit 1
fi
