#!/usr/bin/env bash

set -euo pipefail

# Security verification script
# Tests that passwords are NOT visible in process lists

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PASSED=0
FAILED=0

start_hanging_endpoint() {
    ENDPOINT_PORT_FILE="$(mktemp)"
    python3 - "$ENDPOINT_PORT_FILE" <<'PY' &
import socket
import sys
import time

s = socket.socket()
s.bind(("127.0.0.1", 0))
s.listen(1)
with open(sys.argv[1], "w", encoding="utf-8") as port_file:
    port_file.write(str(s.getsockname()[1]))
conn, _ = s.accept()
time.sleep(10)
conn.close()
s.close()
PY
    ENDPOINT_PID=$!
    for _ in {1..20}; do
        if [ -s "$ENDPOINT_PORT_FILE" ]; then
            ENDPOINT_PORT="$(cat "$ENDPOINT_PORT_FILE")"
            return 0
        fi
        kill -0 "$ENDPOINT_PID" 2>/dev/null || return 1
        sleep 0.05
    done
    return 1
}

stop_hanging_endpoint() {
    kill "${ENDPOINT_PID:-}" 2>/dev/null || true
    wait "${ENDPOINT_PID:-}" 2>/dev/null || true
    rm -f "${ENDPOINT_PORT_FILE:-}"
    ENDPOINT_PID=""
    ENDPOINT_PORT=""
    ENDPOINT_PORT_FILE=""
}

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
    echo -e "${RED}Error: Binary not found. Run 'just build' first.${NC}"
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

start_hanging_endpoint
"$PROJECT_ROOT/bin/dbdump" dump -H 127.0.0.1 -P "$ENDPOINT_PORT" -u root -d testdb --auto -o /tmp/test_security_dump1.sql &>/dev/null &
DUMP_PID=$!

sleep 0.3

if ! kill -0 "$DUMP_PID" 2>/dev/null; then
    log_fail "dbdump exited before its command line could be inspected"
elif ps -p "$DUMP_PID" -o command= | grep -Fq "$TEST_PASSWORD"; then
    log_fail "Password '$TEST_PASSWORD' is visible in process list!"
    ps -p "$DUMP_PID" -o command=
else
    log_pass "Password NOT visible in process list (DBDUMP_MYSQL_PWD)"
fi

# Cleanup
kill $DUMP_PID 2>/dev/null || true
wait $DUMP_PID 2>/dev/null || true
stop_hanging_endpoint
unset DBDUMP_MYSQL_PWD
rm -f /tmp/test_security_dump1.sql

##
## Test 2: Password NOT in process list when using MYSQL_PWD
##
log_test "Test 2: Password hidden when using MYSQL_PWD"

export MYSQL_PWD="$TEST_PASSWORD"

start_hanging_endpoint
"$PROJECT_ROOT/bin/dbdump" dump -H 127.0.0.1 -P "$ENDPOINT_PORT" -u root -d testdb --auto -o /tmp/test_security_dump2.sql &>/dev/null &
DUMP_PID=$!

sleep 0.3

if ! kill -0 "$DUMP_PID" 2>/dev/null; then
    log_fail "dbdump exited before its command line could be inspected"
elif ps -p "$DUMP_PID" -o command= | grep -Fq "$TEST_PASSWORD"; then
    log_fail "Password '$TEST_PASSWORD' is visible in process list!"
    ps -p "$DUMP_PID" -o command=
else
    log_pass "Password NOT visible in process list (MYSQL_PWD)"
fi

kill $DUMP_PID 2>/dev/null || true
wait $DUMP_PID 2>/dev/null || true
stop_hanging_endpoint
unset MYSQL_PWD
rm -f /tmp/test_security_dump2.sql

##
## Test 3: Password NOT in mysqldump child process
##
log_test "Test 3: Password hidden in mysqldump subprocess"

# This test requires a real database, so we'll use Docker if available
if command -v docker &>/dev/null && docker compose -f docker/docker-compose.yml ps 2>/dev/null | grep -q "mysql80.*Up\|mysql80.*running"; then
    log_info "MySQL 8.0 container is running, testing with real connection..."
    
    export DBDUMP_MYSQL_PWD="testpass123"
    
    # Start dump in background
    "$PROJECT_ROOT/bin/dbdump" dump -H 127.0.0.1 -P 3308 -u root -d testdb --auto -o /tmp/test_security_dump3.sql &
    DUMP_PID=$!
    
    # Wait a bit for mysqldump to start
    sleep 2
    
    # Check all processes including mysqldump
    # shellcheck disable=SC2009 # We need full command lines, not only matching PIDs.
    if ps -axo command= | grep -E '[m]ysqldump|[d]bdump' | grep -q "testpass123"; then
        log_fail "Password visible in mysqldump or dbdump process!"
        # shellcheck disable=SC2009 # Print the exact command lines on failure.
        ps -axo command= | grep -E '[m]ysqldump|[d]bdump'
    else
        log_pass "Password NOT visible in dbdump or mysqldump processes"
    fi
    
    # Positive test: dump succeeds with password sourced via env and handed off securely
    log_test "Test 3b: mysqldump authentication handoff works"
    # We can't easily check child process env, but we can verify the dump works
    wait "$DUMP_PID"
    
    if [ -f /tmp/test_security_dump3.sql ] && [ -s /tmp/test_security_dump3.sql ]; then
        log_pass "Dump completed successfully with secure mysqldump auth handoff"
    else
        log_fail "Dump failed (mysqldump auth handoff may be broken)"
    fi
    
    unset DBDUMP_MYSQL_PWD
    rm -f /tmp/test_security_dump3.sql
else
    log_info "Docker not running, skipping mysqldump subprocess test"
    log_info "Run 'docker compose -f docker/docker-compose.yml up -d' to enable this test"
fi

##
## Test 4: File permissions are restrictive (0600)
##
log_test "Test 4: Dump files created with restrictive permissions"

if command -v docker &>/dev/null && docker compose -f docker/docker-compose.yml ps 2>/dev/null | grep -q "mysql80.*Up\|mysql80.*running"; then
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
        
        echo "  Permissions: $PERMS"
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
    log_fail "Found '-p' password flag in dumper.go (should use defaults-extra-file)"
else
    log_pass "No '-p' password flag found in dumper.go (correct)"
fi

##
## Test 6: defaults-extra-file is used in subprocess
##
log_test "Test 6: Code uses defaults-extra-file for mysqldump subprocess"

if grep -q 'defaults-extra-file' "$PROJECT_ROOT/internal/database/dumper.go"; then
    log_pass "Code uses temporary defaults-extra-file auth"
else
    log_fail "Code does not use defaults-extra-file (password may not be passed securely)"
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
    echo "All executed checks passed; Docker-dependent checks are reported above when skipped."
    echo ""
    exit 0
else
    echo -e "${RED}✗ Some security tests failed!${NC}"
    echo ""
    echo "Please review the failures above and fix them before release."
    echo ""
    exit 1
fi
