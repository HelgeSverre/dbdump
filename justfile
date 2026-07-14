# dbdump - MySQL/MariaDB Database Dump Tool
# https://just.systems/man/en/

# Variables
binary_name := "dbdump"
build_dir := "bin"
version := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
commit := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
build_time := `date -u '+%Y-%m-%d_%H:%M:%S'`
ldflags := "-ldflags \"-X main.Version=" + version + " -X main.Commit=" + commit + " -X main.BuildTime=" + build_time + "\""

# Show available commands by default (first recipe is default)
help:
    @just --list

# === Build ===

[group('build')]
[doc('Build the binary for current platform')]
build:
    @echo "Building {{binary_name}}..."
    @mkdir -p {{build_dir}}
    go build {{ldflags}} -o {{build_dir}}/{{binary_name}} ./cmd/dbdump
    @echo "Binary built: {{build_dir}}/{{binary_name}}"

[group('build')]
[doc('Build binaries for all platforms')]
build-all:
    @echo "Building for multiple platforms..."
    @mkdir -p {{build_dir}}
    # macOS AMD64
    GOOS=darwin GOARCH=amd64 go build {{ldflags}} -o {{build_dir}}/{{binary_name}}-darwin-amd64 ./cmd/dbdump
    # macOS ARM64 (Apple Silicon)
    GOOS=darwin GOARCH=arm64 go build {{ldflags}} -o {{build_dir}}/{{binary_name}}-darwin-arm64 ./cmd/dbdump
    # Linux AMD64
    GOOS=linux GOARCH=amd64 go build {{ldflags}} -o {{build_dir}}/{{binary_name}}-linux-amd64 ./cmd/dbdump
    # Linux ARM64
    GOOS=linux GOARCH=arm64 go build {{ldflags}} -o {{build_dir}}/{{binary_name}}-linux-arm64 ./cmd/dbdump
    # Windows AMD64
    GOOS=windows GOARCH=amd64 go build {{ldflags}} -o {{build_dir}}/{{binary_name}}-windows-amd64.exe ./cmd/dbdump
    @echo "Build complete! Binaries in {{build_dir}}/"
    @ls -lh {{build_dir}}/

[group('build')]
[doc('Install the binary to ~/.local/bin (no sudo)')]
install dest="~/.local/bin": build
    mkdir -p {{dest}}
    install -m 0755 {{build_dir}}/{{binary_name}} {{dest}}/{{binary_name}}
    @echo "Installed {{binary_name}} to {{dest}}/{{binary_name}}"
    @echo "Ensure {{dest}} is on your PATH."

[group('build')]
[doc('Uninstall the binary from ~/.local/bin')]
uninstall dest="~/.local/bin":
    rm -f {{dest}}/{{binary_name}}
    @echo "Removed {{dest}}/{{binary_name}}"

[group('build')]
[doc('Remove build artifacts')]
clean:
    @echo "Cleaning..."
    rm -rf {{build_dir}}
    go clean
    @echo "Clean complete!"

# === Testing ===

[group('test')]
[doc('Run unit tests')]
test:
    go test -v ./...

[group('test')]
[doc('Start Docker Compose test databases')]
test-docker-up:
    @echo "Starting test databases..."
    docker compose up -d
    @echo "Waiting for databases to be ready (30s)..."
    @sleep 30
    @echo "Databases ready!"

[group('test')]
[doc('Stop Docker Compose test databases')]
test-docker-down:
    @echo "Stopping test databases..."
    docker compose down

[group('test')]
[doc('Stop Docker and remove all data volumes')]
test-docker-clean:
    @echo "Stopping and cleaning test databases..."
    docker compose down -v
    @echo "All test data removed!"

[group('test')]
[doc('Generate small test dataset (~10MB) on MySQL 8.0')]
test-data-small:
    @echo "Generating small test dataset..."
    @./test/generate-sample-data.sh small 127.0.0.1 3308 testdb

[group('test')]
[doc('Generate medium test dataset (~100MB) on MySQL 8.0')]
test-data-medium:
    @echo "Generating medium test dataset..."
    @./test/generate-sample-data.sh medium 127.0.0.1 3308 testdb

[group('test')]
[doc('Generate large test dataset (~1GB) on MySQL 8.0')]
test-data-large:
    @echo "Generating large test dataset..."
    @./test/generate-sample-data.sh large 127.0.0.1 3308 testdb

[group('test')]
[doc('Generate test data on ALL database versions')]
test-data-all: test-docker-up
    @echo "Generating test data on all databases..."
    @echo "[1/4] MySQL 5.7..."
    @./test/generate-sample-data.sh small 127.0.0.1 3307 testdb
    @echo "[2/4] MySQL 8.0..."
    @./test/generate-sample-data.sh small 127.0.0.1 3308 testdb
    @echo "[3/4] MySQL 8.4..."
    @./test/generate-sample-data.sh small 127.0.0.1 3309 testdb
    @echo "[4/4] MariaDB..."
    @./test/generate-sample-data.sh small 127.0.0.1 3310 testdb
    @echo "Test data generated on all databases!"

[group('test')]
[doc('Quick integration test (small data, MySQL 8.0 only)')]
test-integration-quick: build
    @echo "Running quick integration test..."
    @TEST_QUICK=1 ./test/integration-test.sh

[group('test')]
[doc('Full integration test (all databases)')]
test-integration: build test-docker-up test-data-all
    @echo "Running full integration test suite..."
    @./test/integration-test.sh

[group('test')]
[doc('Run integration tests then cleanup')]
test-integration-clean: test-integration
    @just test-docker-clean

[group('test')]
[doc('Verify security fixes (password hiding, file perms)')]
verify-security: build
    @echo "Running security verification..."
    @./test/verify-security.sh

[group('test')]
[doc('Run security-specific tests only')]
test-security: build verify-security
    @echo "Security tests complete!"

[group('test')]
[doc('Run all tests (unit + integration)')]
test-all: test test-integration

# === Code Quality ===

[group('quality')]
[doc('Format code')]
fmt:
    go fmt ./...

[group('quality')]
[doc('Run go vet')]
vet:
    go vet ./...

[group('quality')]
[doc('Tidy go.mod')]
tidy:
    go mod tidy

[group('quality')]
[doc('Run all quality checks (fmt + vet + test)')]
check: fmt vet test
    @echo "Quality checks complete!"

# === Benchmarking ===

[group('bench')]
[doc('Run benchmark (default: testdb, 3 iterations)')]
bench db="testdb" iter="3": build
    @./scripts/benchmark.sh {{db}} {{iter}}

[group('bench')]
[doc('Quick benchmark (1 iteration)')]
bench-quick db="testdb": build
    @./scripts/benchmark.sh {{db}} 1

[group('bench')]
[doc('Benchmark the local test databases')]
bench-all: build
    @echo "Running benchmarks on all local test databases..."
    @MYSQL_PORT=3307 ./scripts/benchmark.sh testdb 3
    @MYSQL_PORT=3308 ./scripts/benchmark.sh testdb 3
    @MYSQL_PORT=3309 ./scripts/benchmark.sh testdb 3
    @MYSQL_PORT=3310 ./scripts/benchmark.sh testdb 3

[group('bench')]
[doc('Compare benchmarks before and after changes')]
bench-compare:
    @echo "Run this to create baseline: just bench testdb 5"
    @echo "Then make your changes and run: just bench testdb 5"
    @echo "Results will be in benchmark-results/ directory"

# === Development ===

[group('dev')]
[doc('Build and run the binary')]
run: build
    ./{{build_dir}}/{{binary_name}}

[group('dev')]
[doc('Run without building (using go run)')]
dev *args:
    go run ./cmd/dbdump {{args}}

# === Examples ===

[group('examples')]
[doc('Example: list tables')]
example-list: build
    ./{{build_dir}}/{{binary_name}} list -H localhost -u root -d mydb

[group('examples')]
[doc('Example: dump database')]
example-dump: build
    ./{{build_dir}}/{{binary_name}} dump -H localhost -u root -d mydb

[group('examples')]
[doc('Example: dry run')]
example-dry-run: build
    ./{{build_dir}}/{{binary_name}} dump -H localhost -u root -d mydb --dry-run

# === Release ===

[group('release')]
[doc('Test release build locally (simulates GitHub Actions)')]
test-release version=version:
    @./scripts/test-release.sh {{version}}

# === Workflow Shortcuts ===

[group('workflow')]
[doc('Quick dev cycle: format, vet, and test')]
quick: fmt vet test
    @echo "Quick check complete!"

[group('workflow')]
[doc('Full quality suite before PR')]
pr: check test-integration-quick
    @echo "Ready for PR!"

# === Website ===

[group('website')]
[doc('Deploy website to production')]
deploy-website:
    cd website && vc --prod
