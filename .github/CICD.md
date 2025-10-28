# CI/CD Documentation

This document describes the continuous integration and deployment workflows for dbdump.

---

## GitHub Actions Workflows

### 1. Test Workflow (`test.yml`)

**Triggers:**
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop`

**Jobs:**

#### Security Tests
- Runs `make verify-security`
- Verifies passwords are hidden from process lists
- Checks file permissions and code patterns
- **Runtime:** ~30 seconds

#### Unit Tests
- Runs Go unit tests with race detection
- Generates code coverage reports
- Uploads to Codecov
- **Runtime:** ~1 minute

#### Integration Tests
- Starts 4 MySQL service containers:
  - MySQL 5.7 (port 3307)
  - MySQL 8.0 (port 3308)
  - MySQL 8.4 (port 3309)
  - MariaDB 10.11 (port 3310)
- Generates sample data on all databases
- Runs full integration test suite (52 tests)
- **Runtime:** ~8-10 minutes

#### Code Quality
- Runs `gofmt`, `go vet`, and `golangci-lint`
- Checks code formatting and style
- Verifies `go.mod` and `go.sum` are tidy
- **Runtime:** ~2 minutes

#### Build Test
- Builds on Ubuntu, macOS, and Windows
- Verifies binary executes on each platform
- **Runtime:** ~2 minutes per platform

**Total Runtime:** ~15 minutes

---

### 2. Release Workflow (`release.yml`)

**Triggers:**
- Git tags matching `v*.*.*` (e.g., `v1.1.0`)
- Manual workflow dispatch with version input

**Jobs:**

#### Test
- Runs security and unit tests before building
- Ensures release is built from passing code

#### Build
- Builds binaries for all platforms:
  - macOS AMD64
  - macOS ARM64 (Apple Silicon)
  - Linux AMD64
  - Linux ARM64
  - Windows AMD64
- Generates SHA256 checksums
- Creates compressed archives (`.tar.gz` for Unix, `.zip` for Windows)
- Uploads as GitHub artifacts

#### Release
- Extracts changelog for the version from `CHANGELOG.md`
- Creates GitHub Release with binaries attached
- Includes checksums file

#### Verify
- Downloads release artifacts
- Tests binary execution on Ubuntu, macOS, and Windows
- Ensures binaries work correctly

**Total Runtime:** ~10 minutes

**Release Assets:**
```
dbdump-v1.1.0-darwin-amd64.tar.gz
dbdump-v1.1.0-darwin-arm64.tar.gz
dbdump-v1.1.0-linux-amd64.tar.gz
dbdump-v1.1.0-linux-arm64.tar.gz
dbdump-v1.1.0-windows-amd64.zip
checksums.txt
```

---

### 3. PR Checks Workflow (`pr-checks.yml`)

**Triggers:**
- Pull requests (opened, synchronized, reopened)

**Jobs:**

#### Security Check
- Runs `gosec` security scanner
- Uploads SARIF results to GitHub Security
- Identifies potential security vulnerabilities

#### Secrets Scan
- Runs TruffleHog to detect leaked secrets
- Scans commit diffs for credentials
- Prevents accidental secret commits

#### PR Validation
- Validates PR title follows conventional commits
- Accepted types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`

#### Size Label
- Automatically labels PR by size:
  - `size/xs`: 0-10 lines
  - `size/s`: 11-100 lines
  - `size/m`: 101-500 lines
  - `size/l`: 501-1000 lines
  - `size/xl`: >1000 lines

---

### 4. Dependabot (`dependabot.yml`)

**Schedule:** Weekly on Mondays at 9:00 AM

**Monitors:**
- Go module dependencies
- GitHub Actions versions
- Docker base images

**Configuration:**
- Max 10 open PRs for Go modules
- Max 5 open PRs for GitHub Actions
- Max 5 open PRs for Docker
- Auto-assigns to repository owner
- Labels: `dependencies`, `go`, `github-actions`, or `docker`

---

## How to Use

### Running Tests Locally

Before pushing, run:

```bash
# Quick check
make verify-security

# Full local test suite
make test-all
```

### Creating a Release

1. **Update version:**
   ```bash
   # Update CHANGELOG.md with version changes
   vim CHANGELOG.md
   
   # Commit changes
   git add CHANGELOG.md
   git commit -m "chore: prepare v1.1.0 release"
   git push
   ```

2. **Create and push tag:**
   ```bash
   git tag -a v1.1.0 -m "Release v1.1.0"
   git push origin v1.1.0
   ```

3. **GitHub Actions will:**
   - Run all tests
   - Build binaries for all platforms
   - Create GitHub Release
   - Upload binaries and checksums

4. **Verify release:**
   - Go to https://github.com/helgesverre/dbdump/releases
   - Verify binaries are attached
   - Test download and execution

### Manual Release Trigger

If you need to re-release or create a release without a tag:

1. Go to Actions → Release workflow
2. Click "Run workflow"
3. Enter version (e.g., `v1.1.0`)
4. Click "Run workflow"

---

## Workflow Status Badges

Add to README.md:

```markdown
[![Tests](https://github.com/helgesverre/dbdump/actions/workflows/test.yml/badge.svg)](https://github.com/helgesverre/dbdump/actions/workflows/test.yml)
[![Release](https://github.com/helgesverre/dbdump/actions/workflows/release.yml/badge.svg)](https://github.com/helgesverre/dbdump/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/helgesverre/dbdump/branch/main/graph/badge.svg)](https://codecov.io/gh/helgesverre/dbdump)
```

---

## Service Containers

Integration tests use GitHub Actions service containers for databases:

```yaml
services:
  mysql80:
    image: mysql:8.0
    env:
      MYSQL_ROOT_PASSWORD: testpass123
      MYSQL_DATABASE: testdb
    ports:
      - 3308:3306
    options: >-
      --health-cmd="mysqladmin ping"
      --health-interval=10s
      --health-timeout=5s
      --health-retries=10
```

This provides isolated database instances for testing without Docker Compose.

---

## Secrets and Variables

### Repository Secrets

No secrets required for current workflows.

**Optional (if using Codecov):**
- `CODECOV_TOKEN` - For uploading coverage reports

### Repository Variables

None currently required.

---

## Troubleshooting

### Tests Fail in CI But Pass Locally

**Common causes:**
1. **Different Go version** - Check workflow uses correct version
2. **Missing dependencies** - Ensure all deps are in `go.mod`
3. **Platform differences** - Test on Linux if using macOS/Windows locally
4. **Race conditions** - Tests with `-race` flag may catch issues

**Fix:**
```bash
# Run with same flags as CI
go test -v -race ./...
```

### Integration Tests Timeout

**Cause:** Databases not ready

**Fix:** Increase health check retries:
```yaml
options: >-
  --health-retries=20  # Increase from 10
```

### Release Build Fails

**Cause:** Usually version tagging issues

**Fix:**
```bash
# Ensure tag is properly annotated
git tag -a v1.1.0 -m "Release v1.1.0"

# Push tag
git push origin v1.1.0

# Verify tag exists
git tag -l
```

### Binary Doesn't Work on Target Platform

**Cause:** Cross-compilation issues or missing CGO

**Fix:** dbdump should not require CGO, but verify:
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build
```

---

## Adding New Workflows

### Creating a New Workflow

1. Create file in `.github/workflows/`
2. Define triggers and jobs
3. Test with `act` (local GitHub Actions runner) if possible
4. Create PR to test on real GitHub Actions

### Best Practices

- ✅ Use latest action versions (`@v4`, `@v5`)
- ✅ Pin Go version to match development
- ✅ Cache Go modules for speed
- ✅ Use `if: always()` for artifact uploads
- ✅ Add descriptive job names
- ✅ Fail fast when appropriate
- ✅ Use service containers for databases
- ✅ Generate artifacts for debugging

---

## Security Considerations

### SARIF Upload

Security scan results are uploaded to GitHub Security tab:

```yaml
- name: Upload SARIF file
  uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: gosec.sarif
```

View results at: `https://github.com/USERNAME/REPO/security/code-scanning`

### Secrets Scanning

TruffleHog scans for:
- API keys
- Passwords
- Private keys
- Tokens
- Credentials

Configured to scan PR diffs only for performance.

---

## Performance Optimization

### Caching

Go modules are cached automatically:

```yaml
- uses: actions/setup-go@v5
  with:
    go-version: '1.24'
    cache: true  # Enables automatic caching
```

### Parallel Jobs

Jobs run in parallel when possible:
- Security, unit tests, integration tests, code quality run concurrently
- Build test runs on 3 platforms simultaneously

### Service Container Health Checks

Health checks prevent tests from running before databases are ready:

```yaml
options: >-
  --health-cmd="mysqladmin ping"
  --health-interval=10s
```

---

## Cost Considerations

### GitHub Actions Minutes

**Free tier:** 2,000 minutes/month for public repos

**Estimated usage:**
- Per commit: ~15 minutes (test workflow)
- Per PR: ~20 minutes (test + PR checks)
- Per release: ~10 minutes

**Optimization:**
- Use caching to reduce build time
- Run expensive tests only on main/PR
- Use `if:` conditions to skip unnecessary jobs

---

## Further Reading

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [golangci-lint Documentation](https://golangci-lint.run/)
- [Dependabot Configuration](https://docs.github.com/en/code-security/dependabot)
- [SARIF Format](https://docs.github.com/en/code-security/code-scanning/integrating-with-code-scanning/sarif-support-for-code-scanning)

---

**Last Updated:** 2025-10-28
