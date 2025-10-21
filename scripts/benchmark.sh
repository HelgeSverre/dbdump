#!/bin/bash
set -euo pipefail

# Benchmark script for dbdump performance testing
# Usage: ./scripts/benchmark.sh [database] [iterations]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BINARY="$PROJECT_ROOT/bin/dbdump"
RESULTS_DIR="$PROJECT_ROOT/benchmark-results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Configuration
DATABASE="${1:-crescat_dump}"
ITERATIONS="${2:-3}"
MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create results directory
mkdir -p "$RESULTS_DIR"
RESULTS_FILE="$RESULTS_DIR/benchmark_${DATABASE}_${TIMESTAMP}.json"
SUMMARY_FILE="$RESULTS_DIR/benchmark_${DATABASE}_${TIMESTAMP}_summary.txt"

echo -e "${BLUE}=== dbdump Performance Benchmark ===${NC}"
echo -e "Database: ${GREEN}$DATABASE${NC}"
echo -e "Iterations: ${GREEN}$ITERATIONS${NC}"
echo -e "Output: ${GREEN}$RESULTS_FILE${NC}"
echo ""

# Check if binary exists
if [ ! -f "$BINARY" ]; then
    echo -e "${RED}Error: Binary not found at $BINARY${NC}"
    echo "Run 'make build' first"
    exit 1
fi

# Check if database exists
if ! mysql -h "$MYSQL_HOST" -u "$MYSQL_USER" ${MYSQL_PASSWORD:+-p"$MYSQL_PASSWORD"} -e "USE $DATABASE" 2>/dev/null; then
    echo -e "${RED}Error: Database '$DATABASE' not found${NC}"
    exit 1
fi

# Get database statistics
echo -e "${BLUE}Collecting database statistics...${NC}"
DB_STATS=$(mysql -h "$MYSQL_HOST" -u "$MYSQL_USER" ${MYSQL_PASSWORD:+-p"$MYSQL_PASSWORD"} -N -e "
SELECT
    COUNT(*) as table_count,
    ROUND(SUM(data_length + index_length) / 1024 / 1024, 2) as total_mb,
    SUM(table_rows) as total_rows
FROM information_schema.tables
WHERE table_schema = '$DATABASE'
" 2>/dev/null)

read TABLE_COUNT TOTAL_MB TOTAL_ROWS <<< "$DB_STATS"

echo -e "  Tables: ${GREEN}$TABLE_COUNT${NC}"
echo -e "  Size: ${GREEN}${TOTAL_MB} MB${NC}"
echo -e "  Rows: ${GREEN}$TOTAL_ROWS${NC}"
echo ""

# Initialize JSON results
cat > "$RESULTS_FILE" << EOF
{
  "timestamp": "$TIMESTAMP",
  "database": "$DATABASE",
  "iterations": $ITERATIONS,
  "database_stats": {
    "table_count": $TABLE_COUNT,
    "total_mb": $TOTAL_MB,
    "total_rows": $TOTAL_ROWS
  },
  "system_info": {
    "os": "$(uname -s)",
    "arch": "$(uname -m)",
    "go_version": "$(go version | awk '{print $3}')"
  },
  "runs": [
EOF

# Arrays to store timing data
declare -a REAL_TIMES
declare -a USER_TIMES
declare -a SYS_TIMES
declare -a THROUGHPUTS
declare -a FILE_SIZES

# Run benchmarks
echo -e "${BLUE}Running $ITERATIONS benchmark iterations...${NC}"
for i in $(seq 1 $ITERATIONS); do
    echo -e "${YELLOW}Iteration $i/$ITERATIONS${NC}"

    OUTPUT_FILE="/tmp/dbdump_benchmark_${DATABASE}_${i}.sql"
    TIME_FILE="/tmp/dbdump_time_${i}.txt"

    # Run with time measurement
    (/usr/bin/time -p "$BINARY" dump \
        -H "$MYSQL_HOST" \
        -u "$MYSQL_USER" \
        ${MYSQL_PASSWORD:+-p "$MYSQL_PASSWORD"} \
        -d "$DATABASE" \
        --auto \
        -o "$OUTPUT_FILE" \
        2>&1) 2> "$TIME_FILE" | grep -E "(Connected|Found|excluding|Starting|complete|Duration)" || true

    # Parse timing results
    REAL_TIME=$(grep "^real" "$TIME_FILE" | awk '{print $2}')
    USER_TIME=$(grep "^user" "$TIME_FILE" | awk '{print $2}')
    SYS_TIME=$(grep "^sys" "$TIME_FILE" | awk '{print $2}')

    # Get file size
    FILE_SIZE=$(stat -f%z "$OUTPUT_FILE" 2>/dev/null || stat -c%s "$OUTPUT_FILE" 2>/dev/null)
    FILE_SIZE_MB=$(echo "scale=2; $FILE_SIZE / 1024 / 1024" | bc)

    # Calculate throughput (MB/s)
    THROUGHPUT=$(echo "scale=2; $FILE_SIZE_MB / $REAL_TIME" | bc)

    # Store results
    REAL_TIMES+=($REAL_TIME)
    USER_TIMES+=($USER_TIME)
    SYS_TIMES+=($SYS_TIME)
    THROUGHPUTS+=($THROUGHPUT)
    FILE_SIZES+=($FILE_SIZE_MB)

    echo -e "  Real: ${GREEN}${REAL_TIME}s${NC}, User: ${GREEN}${USER_TIME}s${NC}, Sys: ${GREEN}${SYS_TIME}s${NC}"
    echo -e "  Output: ${GREEN}${FILE_SIZE_MB} MB${NC}, Throughput: ${GREEN}${THROUGHPUT} MB/s${NC}"

    # Add to JSON (add comma except for last iteration)
    COMMA=","
    [ $i -eq $ITERATIONS ] && COMMA=""

    cat >> "$RESULTS_FILE" << EOF
    {
      "iteration": $i,
      "real_time": $REAL_TIME,
      "user_time": $USER_TIME,
      "sys_time": $SYS_TIME,
      "file_size_mb": $FILE_SIZE_MB,
      "throughput_mbps": $THROUGHPUT
    }$COMMA
EOF

    # Cleanup output file
    rm -f "$OUTPUT_FILE" "$TIME_FILE"

    # Brief pause between runs
    [ $i -lt $ITERATIONS ] && sleep 2
done

# Close JSON
cat >> "$RESULTS_FILE" << EOF
  ]
}
EOF

echo ""
echo -e "${BLUE}=== Benchmark Results ===${NC}"

# Calculate statistics
function calculate_avg() {
    local sum=0
    local count=$#
    for val in "$@"; do
        sum=$(echo "$sum + $val" | bc)
    done
    echo "scale=2; $sum / $count" | bc
}

function calculate_median() {
    local sorted=($(printf '%s\n' "$@" | sort -n))
    local count=${#sorted[@]}
    local mid=$((count / 2))
    if [ $((count % 2)) -eq 0 ]; then
        echo "scale=2; (${sorted[$mid-1]} + ${sorted[$mid]}) / 2" | bc
    else
        echo "${sorted[$mid]}"
    fi
}

function calculate_min() {
    printf '%s\n' "$@" | sort -n | head -1
}

function calculate_max() {
    printf '%s\n' "$@" | sort -n | tail -1
}

AVG_REAL=$(calculate_avg "${REAL_TIMES[@]}")
MED_REAL=$(calculate_median "${REAL_TIMES[@]}")
MIN_REAL=$(calculate_min "${REAL_TIMES[@]}")
MAX_REAL=$(calculate_max "${REAL_TIMES[@]}")

AVG_THROUGHPUT=$(calculate_avg "${THROUGHPUTS[@]}")
MED_THROUGHPUT=$(calculate_median "${THROUGHPUTS[@]}")
MIN_THROUGHPUT=$(calculate_min "${THROUGHPUTS[@]}")
MAX_THROUGHPUT=$(calculate_max "${THROUGHPUTS[@]}")

AVG_FILE_SIZE=$(calculate_avg "${FILE_SIZES[@]}")

# Print summary
echo -e "${GREEN}Real Time (seconds):${NC}"
echo "  Average: $AVG_REAL"
echo "  Median:  $MED_REAL"
echo "  Min:     $MIN_REAL"
echo "  Max:     $MAX_REAL"
echo ""

echo -e "${GREEN}Throughput (MB/s):${NC}"
echo "  Average: $AVG_THROUGHPUT"
echo "  Median:  $MED_THROUGHPUT"
echo "  Min:     $MIN_THROUGHPUT"
echo "  Max:     $MAX_THROUGHPUT"
echo ""

echo -e "${GREEN}Output File Size:${NC}"
echo "  Average: $AVG_FILE_SIZE MB"
echo ""

# Save summary
cat > "$SUMMARY_FILE" << EOF
Benchmark Summary - $DATABASE
Generated: $TIMESTAMP
Iterations: $ITERATIONS

Database Statistics:
  Tables: $TABLE_COUNT
  Size: $TOTAL_MB MB
  Rows: $TOTAL_ROWS

Performance Results:
  Real Time (seconds):
    Average: $AVG_REAL
    Median:  $MED_REAL
    Min:     $MIN_REAL
    Max:     $MAX_REAL

  Throughput (MB/s):
    Average: $AVG_THROUGHPUT
    Median:  $MED_THROUGHPUT
    Min:     $MIN_THROUGHPUT
    Max:     $MAX_THROUGHPUT

  Output File Size:
    Average: $AVG_FILE_SIZE MB

Detailed Results: $RESULTS_FILE
EOF

echo -e "${BLUE}Results saved to:${NC}"
echo -e "  JSON:    ${GREEN}$RESULTS_FILE${NC}"
echo -e "  Summary: ${GREEN}$SUMMARY_FILE${NC}"
echo ""
echo -e "${GREEN}✓ Benchmark complete!${NC}"
