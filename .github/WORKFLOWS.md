# GitHub Actions Workflows

## Overview

This project has comprehensive CI/CD automation via GitHub Actions.

## Workflows

### 1. Tests (`test.yml`)

**Triggers:** Push to `main`/`develop`, Pull Requests

**Jobs:**
- **Security Tests** (~30s) - Password hiding verification, code pattern checks
- **Unit Tests** (~1m) - Go tests with race detection, coverage reporting
- **Integration Tests** (~10m) - Testing against MySQL 5.7, 8.0, 8.4, MariaDB
- **Code Quality** (~2m) - gofmt, go vet, golangci-lint, go.mod tidy check
- **Build Test** (~2m) - Parallel builds on Ubuntu, macOS, Windows

**Total Runtime:** ~15 minutes

### 2. Release (`release.yml`)

**Triggers:** 
- Git tags matching `v*.*.*` (e.g., `v1.1.0`)
- Manual workflow dispatch

**Jobs:**
1. **Test** - Run security and unit tests
2. **Build** - Build binaries for all platforms (macOS AMD64/ARM64, Linux AMD64/ARM64, Windows AMD64)
3. **Release** - Create GitHub Release with binaries and checksums
4. **Verify** - Test each binary on its target platform

**Artifacts:**
- Platform-specific binaries (tar.gz for Unix, zip for Windows)
- SHA256 checksums
- Auto-generated release notes from CHANGELOG.md

**Total Runtime:** ~10 minutes

### 3. PR Checks (`pr-checks.yml`)

**Triggers:** Pull Requests

**Additional validation:**
- Security scanning with gosec (SARIF upload)
- Secrets detection with TruffleHog
- Semantic PR title validation
- Auto-labeling by PR size

## Quick Commands

### Create a Release

```bash
git tag -a v1.1.0 -m "Release v1.1.0"
git push origin v1.1.0
```

### View Workflow Runs

```bash
# Visit:
https://github.com/HelgeSverre/dbdump/actions
```

### Manual Release Trigger

1. Go to Actions → Release workflow
2. Click "Run workflow"
3. Enter version (e.g., `v1.1.0`)
4. Click "Run workflow"

## Status Badges

Add to README.md:

```markdown
[![Tests](https://github.com/helgesverre/dbdump/actions/workflows/test.yml/badge.svg)](https://github.com/helgesverre/dbdump/actions/workflows/test.yml)
[![Release](https://github.com/helgesverre/dbdump/actions/workflows/release.yml/badge.svg)](https://github.com/helgesverre/dbdump/actions/workflows/release.yml)
```

## Integration Test Details

### Service Containers

- MySQL 5.7 (port 3307)
- MySQL 8.0 (port 3308)  
- MySQL 8.4 (port 3309)
- MariaDB 10.11 (port 3310)

### Test Data

Small dataset (~10MB) generated on each database with:
- Users, Products, Orders, Order Items tables
- Triggers and stored procedures
- "Noisy" tables for exclusion testing

### Test Coverage

**52 total tests** (13 per database):
- Security (password hiding, file permissions)
- Data integrity (triggers, procedures, restoration)
- Exclusion logic (structure preserved, data excluded)
- CLI features (list, dry-run, custom output)

## Automated Updates

### Dependabot

Configured in `.github/dependabot.yml`:
- Go module updates (weekly)
- GitHub Actions updates (weekly)
- Docker image updates (weekly)

Updates automatically create PRs with:
- Auto-assignment
- Auto-labels
- Version bump details

## Troubleshooting

### Workflow Fails

1. **Check logs:** Click on failed job in Actions tab
2. **Reproduce locally:**
   ```bash
   make test-all  # Run all tests
   make build-all # Test multi-platform builds
   ```

### Release Issues

**Tag doesn't trigger release:**
- Ensure tag format is `v*.*.*` (e.g., `v1.1.0`)
- Check workflow file triggers

**Build fails:**
- Verify Go version compatibility (1.24.6)
- Check for platform-specific code issues

**Checksums mismatch:**
- Re-run the workflow
- Check for non-deterministic build issues

### Permission Errors

Workflows need:
- `contents: write` for releases (already configured)
- `GITHUB_TOKEN` (automatically provided)

## Workflow Maintenance

### Updating Go Version

Edit all workflow files:
```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.24'  # Update this
```

### Adding New Platforms

Edit `release.yml` build job:
```yaml
# Add new platform build
GOOS=freebsd GOARCH=amd64 go build ...
```

### Changing Test Databases

Edit `test.yml` services section to add/remove MySQL versions.

## Performance

### Caching

Go modules are automatically cached by `actions/setup-go@v5` with `cache: true`.

**Typical cache hit:** Saves ~30% of build time

### Parallel Execution

- Test jobs run concurrently
- Build test runs on 3 platforms simultaneously
- Release builds run sequentially (safer)

## Cost (GitHub Actions Free Tier)

**Free tier:** 2,000 minutes/month

**Estimated usage:**
- 50 commits × 15 min = 750 min/month
- 20 PRs × 20 min = 400 min/month  
- 4 releases × 10 min = 40 min/month
- **Total:** ~1,200 min/month

**Well within free tier for open source projects.**

## Security

### Scanning

- **gosec** - Static security analysis (SARIF upload to GitHub Security)
- **TruffleHog** - Secrets detection in commits
- **Custom tests** - Runtime password hiding verification

### Results Location

Security scan results appear in:
- GitHub Security → Code scanning alerts
- PR checks and annotations

## Documentation

- **RELEASE.md** - Complete release process guide
- **TESTING_GUIDE.md** - Testing documentation
- **.github/CICD.md** - Detailed CI/CD documentation

## Support

For workflow issues:
1. Check GitHub Actions logs
2. Review workflow YAML files
3. Test locally with Makefile targets
4. Open issue if needed
