#!/usr/bin/env bash
#
# End-to-end smoke test for dbdump against real database servers in Docker.
#
#   docker/smoke-test.sh [profiles|tls|all]   (default: all)
#
# Part A (profiles): brings up MySQL 5.7 / 8.0 / 8.4 and MariaDB (plaintext),
#   saves a connection profile for each, and dumps via --profile. Exercises the
#   profile feature and surfaces server-version compatibility.
# Part B (tls): brings up TLS-enforcing MySQL 8.0 and MariaDB, and verifies each
#   TLS mode (require / verify-ca / verify-identity / mutual TLS) connects and
#   dumps, and that a plaintext connection is correctly rejected.
#
# Env:
#   KEEP=1   leave containers running after the run (default: tear down)
#
# Portable across macOS (bash 3.2) and Linux; needs docker, go, openssl, mysqldump.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(dirname "$SCRIPT_DIR")"
MODE="${1:-all}"

COMPOSE_PLAIN="docker compose -f $SCRIPT_DIR/docker-compose.yml"
COMPOSE_TLS="docker compose -f $SCRIPT_DIR/docker-compose.tls.yml"
CERTS="$SCRIPT_DIR/certs"
PASS_PW="testpass123"

RED=$'\033[0;31m'; GREEN=$'\033[0;32m'; YELLOW=$'\033[1;33m'; BLUE=$'\033[0;34m'; NC=$'\033[0m'
PASSED=0; FAILED=0
pass() { echo "${GREEN}[PASS]${NC} $1"; PASSED=$((PASSED + 1)); }
fail() { echo "${RED}[FAIL]${NC} $1"; FAILED=$((FAILED + 1)); }
info() { echo "${BLUE}[INFO]${NC} $1"; }
head() { echo; echo "${YELLOW}=== $1 ===${NC}"; }

WORK="$(mktemp -d)"
PROFILE_HOME="$WORK/home"   # isolate profiles.yaml from the real ~/.config
mkdir -p "$PROFILE_HOME"
BIN="$WORK/dbdump"

# dbd runs dbdump with an isolated HOME so 'config add' writes to a throwaway
# profiles.yaml. HOME is scoped to this call only, so go/docker keep the real one.
dbd() { HOME="$PROFILE_HOME" "$BIN" "$@"; }

cleanup() {
  if [ "${KEEP:-0}" != "1" ]; then
    info "Tearing down containers..."
    $COMPOSE_PLAIN down -v >/dev/null 2>&1
    $COMPOSE_TLS down -v >/dev/null 2>&1
  else
    info "KEEP=1 set; leaving containers up."
  fi
  rm -rf "$WORK"
}
trap cleanup EXIT

info "Building dbdump..."
(cd "$REPO" && go build -o "$BIN" ./cmd/dbdump) || { echo "build failed"; exit 1; }

# seed <container> — create a small table with data over the container socket
# (exempt from require_secure_transport, so it works for TLS servers too).
seed() {
  docker exec "$1" mysql -uroot -p"$PASS_PW" testdb \
    -e "DROP TABLE IF EXISTS widgets; CREATE TABLE widgets (id INT PRIMARY KEY, name VARCHAR(50)); INSERT INTO widgets VALUES (1,'alpha'),(2,'beta');" \
    >/dev/null 2>&1
}

# assert_dump <file> <label> — dump file exists and has the schema + data
assert_dump() {
  if [ -s "$1" ] && grep -q "CREATE TABLE" "$1" && grep -q "INSERT INTO" "$1"; then
    pass "$2: dump contains schema + data"
  else
    fail "$2: dump missing/empty or lacking schema+data"
  fi
}

#############################################
# Part A: profiles across MySQL versions
#############################################
run_profiles() {
  head "Part A — profiles across MySQL versions (plaintext)"
  info "Starting databases (amd64 emulation can be slow the first time)..."
  if ! $COMPOSE_PLAIN up -d --wait --wait-timeout 240; then
    fail "plaintext databases did not become healthy"
    return
  fi

  # name container port
  for target in \
    "mysql57 dbdump-mysql57 3307" \
    "mysql80 dbdump-mysql80 3308" \
    "mysql84 dbdump-mysql84 3309" \
    "mariadb dbdump-mariadb 3310"; do
    set -- $target
    local name="$1" container="$2" port="$3"
    info "Target: $name (127.0.0.1:$port)"
    seed "$container"

    dbd config add "$name" -H 127.0.0.1 -P "$port" -u root -p "$PASS_PW" -d testdb >/dev/null 2>&1

    if dbd --profile "$name" list >/dev/null 2>&1; then
      pass "$name: connect + list via --profile"
    else
      fail "$name: --profile list failed"
      continue
    fi

    local out="$WORK/$name.sql"
    if dbd --profile "$name" dump --auto -o "$out" >/dev/null 2>&1; then
      assert_dump "$out" "$name"
    else
      fail "$name: dump via --profile failed (server/mysqldump compatibility)"
    fi
  done
}

#############################################
# Part B: TLS
#############################################
run_tls() {
  head "Part B — TLS-enforced servers"
  if [ ! -f "$CERTS/ca.pem" ]; then
    info "Generating certs..."
    bash "$CERTS/gen-certs.sh" >/dev/null
  fi
  info "Starting TLS databases..."
  if ! $COMPOSE_TLS up -d --wait --wait-timeout 240; then
    fail "TLS databases did not become healthy"
    return
  fi

  for target in \
    "mysql80-tls dbdump-mysql80-tls 3320" \
    "mariadb-tls dbdump-mariadb-tls 3321"; do
    set -- $target
    local name="$1" container="$2" port="$3"
    info "Target: $name (127.0.0.1:$port)"
    seed "$container"

    local base="-H 127.0.0.1 -P $port -u root -p $PASS_PW -d testdb"

    # Negative: plaintext must be rejected by require_secure_transport.
    if dbd $base --tls-mode disabled list >/dev/null 2>&1; then
      fail "$name: plaintext connection unexpectedly succeeded (TLS not enforced?)"
    else
      pass "$name: plaintext connection correctly rejected"
    fi

    # require (encrypt, no verification)
    if dbd $base --tls-mode require list >/dev/null 2>&1; then
      pass "$name: --tls-mode require connects"
    else
      fail "$name: --tls-mode require failed"
    fi

    # verify-ca (verify chain against our CA)
    if dbd $base --tls-mode verify-ca --tls-ca "$CERTS/ca.pem" list >/dev/null 2>&1; then
      pass "$name: --tls-mode verify-ca connects"
    else
      fail "$name: --tls-mode verify-ca failed"
    fi

    # verify-identity (chain + hostname; server SAN covers 127.0.0.1)
    if dbd $base --tls-mode verify-identity --tls-ca "$CERTS/ca.pem" list >/dev/null 2>&1; then
      pass "$name: --tls-mode verify-identity connects"
    else
      fail "$name: --tls-mode verify-identity failed"
    fi

    # mutual TLS
    if dbd $base --tls-mode verify-ca --tls-ca "$CERTS/ca.pem" \
        --tls-cert "$CERTS/client-cert.pem" --tls-key "$CERTS/client-key.pem" list >/dev/null 2>&1; then
      pass "$name: mutual TLS connects"
    else
      fail "$name: mutual TLS failed"
    fi

    # A real dump over TLS.
    local out="$WORK/$name-tls.sql"
    if dbd $base --tls-mode verify-ca --tls-ca "$CERTS/ca.pem" dump --auto -o "$out" >/dev/null 2>&1; then
      assert_dump "$out" "$name (TLS dump)"
    else
      fail "$name: dump over TLS failed"
    fi
  done
}

case "$MODE" in
  profiles) run_profiles ;;
  tls)      run_tls ;;
  all)      run_profiles; run_tls ;;
  *) echo "usage: $0 [profiles|tls|all]"; exit 2 ;;
esac

head "Results"
echo "Passed: ${GREEN}${PASSED}${NC}   Failed: ${RED}${FAILED}${NC}"
[ "$FAILED" -eq 0 ]
