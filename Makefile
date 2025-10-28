.PHONY: build install clean test fmt vet lint tidy build-all help run dev bench bench-quick bench-all bench-compare \
	test-integration test-integration-quick test-security test-docker-up test-docker-down test-docker-clean \
	test-data-small test-data-medium test-data-large verify-security

# Binary name
BINARY_NAME=dbdump

# Build directory
BUILD_DIR=bin

# Version info
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# LDFLAGS for version info
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Default target
all: help

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary for current platform
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/dbdump
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

install: build ## Install the binary to /usr/local/bin
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Installation complete!"

clean: ## Remove build artifacts
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	go clean
	@echo "Clean complete!"

test: ## Run tests
	go test -v ./...

bench: build ## Run benchmark (default: crescat_dump, 3 iterations)
	@./scripts/benchmark.sh $(DB) $(ITER)

bench-quick: build ## Quick benchmark (1 iteration)
	@./scripts/benchmark.sh $(DB) 1

bench-all: build ## Benchmark all available databases
	@echo "Running benchmarks on all available databases..."
	@./scripts/benchmark.sh crescat_dump 3
	@./scripts/benchmark.sh crescat_dump_2 3
	@./scripts/benchmark.sh crescat_dump_3 3

bench-compare: ## Compare benchmarks before and after changes
	@echo "Run this to create baseline: make bench DB=crescat_dump ITER=5"
	@echo "Then make your changes and run: make bench DB=crescat_dump ITER=5"
	@echo "Results will be in benchmark-results/ directory"

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

tidy: ## Tidy go.mod
	go mod tidy

build-all: ## Build binaries for all platforms
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)

	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/dbdump

	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/dbdump

	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/dbdump

	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/dbdump

	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/dbdump

	@echo "Build complete! Binaries in $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)/

run: build ## Build and run the binary
	./$(BUILD_DIR)/$(BINARY_NAME)

dev: ## Run without building (using go run)
	go run ./cmd/dbdump $(ARGS)

# Example usage targets
example-list: build ## Example: list tables
	./$(BUILD_DIR)/$(BINARY_NAME) list -h localhost -u root -d mydb

example-dump: build ## Example: dump database
	./$(BUILD_DIR)/$(BINARY_NAME) dump -h localhost -u root -d mydb

example-dry-run: build ## Example: dry run
	./$(BUILD_DIR)/$(BINARY_NAME) dump -h localhost -u root -d mydb --dry-run

## Integration Testing Targets

test-docker-up: ## Start Docker Compose test databases
	@echo "Starting test databases..."
	docker-compose up -d
	@echo "Waiting for databases to be ready (30s)..."
	@sleep 30
	@echo "Databases ready!"

test-docker-down: ## Stop Docker Compose test databases
	@echo "Stopping test databases..."
	docker-compose down

test-docker-clean: ## Stop Docker and remove all data volumes
	@echo "Stopping and cleaning test databases..."
	docker-compose down -v
	@echo "All test data removed!"

test-data-small: ## Generate small test dataset (~10MB) on MySQL 8.0
	@echo "Generating small test dataset..."
	@./test/generate-sample-data.sh small 127.0.0.1 3308 testdb

test-data-medium: ## Generate medium test dataset (~100MB) on MySQL 8.0
	@echo "Generating medium test dataset..."
	@./test/generate-sample-data.sh medium 127.0.0.1 3308 testdb

test-data-large: ## Generate large test dataset (~1GB) on MySQL 8.0
	@echo "Generating large test dataset..."
	@./test/generate-sample-data.sh large 127.0.0.1 3308 testdb

test-data-all: test-docker-up ## Generate test data on ALL database versions
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

verify-security: build ## Verify security fixes (password hiding, file perms)
	@echo "Running security verification..."
	@./test/verify-security.sh

test-integration-quick: build test-docker-up test-data-small ## Quick integration test (small data, MySQL 8.0 only)
	@echo "Running quick integration test..."
	@export TEST_QUICK=1 && ./test/integration-test.sh
	@$(MAKE) test-docker-down

test-integration: build test-docker-up test-data-all ## Full integration test (all databases)
	@echo "Running full integration test suite..."
	@./test/integration-test.sh

test-integration-clean: test-integration ## Run integration tests then cleanup
	@$(MAKE) test-docker-clean

test-security: build verify-security ## Run security-specific tests only
	@echo "Security tests complete!"

test-all: test test-integration ## Run all tests (unit + integration)

test-release: ## Test release build locally (simulates GitHub Actions)
	@./scripts/test-release.sh $(VERSION)
