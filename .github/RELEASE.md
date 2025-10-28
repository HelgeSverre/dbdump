# Release Process

This document describes how to create a new release of dbdump.

## Quick Release

```bash
# 1. Update CHANGELOG.md with version changes
vim CHANGELOG.md

# 2. Commit changes
git add CHANGELOG.md
git commit -m "chore: prepare v1.1.0 release"
git push

# 3. Create and push tag
git tag -a v1.1.0 -m "Release v1.1.0"
git push origin v1.1.0
```

**That's it!** GitHub Actions will automatically:
- ✅ Run all security tests
- ✅ Run unit tests  
- ✅ Build binaries for 5 platforms
- ✅ Generate SHA256 checksums
- ✅ Create GitHub Release
- ✅ Upload release artifacts
- ✅ Verify binaries work on each platform

## Manual Release Trigger

If the automatic release fails or you need to rebuild:

1. Go to: https://github.com/HelgeSverre/dbdump/actions/workflows/release.yml
2. Click "Run workflow"
3. Enter version (e.g., `v1.1.0`)
4. Click "Run workflow"

## Release Artifacts

Each release includes:

### Binaries
- `dbdump-vX.X.X-darwin-amd64.tar.gz` - macOS Intel
- `dbdump-vX.X.X-darwin-arm64.tar.gz` - macOS Apple Silicon
- `dbdump-vX.X.X-linux-amd64.tar.gz` - Linux AMD64
- `dbdump-vX.X.X-linux-arm64.tar.gz` - Linux ARM64
- `dbdump-vX.X.X-windows-amd64.zip` - Windows AMD64

### Checksums
- `checksums.txt` - SHA256 checksums for all binaries

### Release Notes
- Automatically extracted from `CHANGELOG.md` for the version

## Continuous Integration

### On Every Push/PR
The `test.yml` workflow runs:
- Security verification
- Unit tests
- Integration tests (MySQL 5.7, 8.0, 8.4, MariaDB)
- Code quality checks (fmt, vet, golangci-lint)
- Build verification on Ubuntu, macOS, Windows

### On Release Tags
The `release.yml` workflow runs:
1. Tests (same as above)
2. Multi-platform binary builds
3. Archive creation (tar.gz/zip)
4. Checksum generation
5. GitHub Release creation
6. Binary verification on each OS

### PR-Specific Checks
The `pr-checks.yml` workflow adds:
- Security scanning (gosec)
- Secrets detection (TruffleHog)
- PR title validation
- Auto-labeling by size

## Version Numbering

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR.MINOR.PATCH** (e.g., `v1.2.3`)
  - **MAJOR**: Breaking changes
  - **MINOR**: New features (backward compatible)
  - **PATCH**: Bug fixes

Examples:
- `v1.0.0` - Initial release
- `v1.1.0` - Added new features (security improvements)
- `v1.1.1` - Bug fix release
- `v2.0.0` - Breaking changes (CLI flags changed)

## CHANGELOG Format

Update `CHANGELOG.md` before each release:

```markdown
## [1.2.0] - 2025-11-15

### Added
- New feature X
- Support for Y

### Fixed
- Bug Z
- Issue W

### Security
- CVE-XXXX fix

### Changed
- Behavior A now does B
```

The release workflow automatically extracts this section for release notes.

## Pre-Release Checklist

Before creating a release tag:

- [ ] All tests passing locally (`make test-all`)
- [ ] CHANGELOG.md updated with version changes
- [ ] Security fixes documented (if any)
- [ ] Breaking changes clearly documented
- [ ] Version number follows semver
- [ ] README examples still accurate

## Post-Release Tasks

After GitHub Release is created:

1. **Verify artifacts**
   - Download and test each platform binary
   - Verify checksums match
   
2. **Announcements** (optional)
   - Tweet/post about release
   - Update any external documentation
   
3. **Monitor**
   - Watch for GitHub issues
   - Check Actions tab for any failures

## Troubleshooting

### Release Workflow Failed

**Check the logs:**
```bash
# View in GitHub Actions UI
https://github.com/HelgeSverre/dbdump/actions
```

**Common issues:**
- Tests failing: Fix tests, commit, push new tag
- Build errors: Check Go version compatibility
- Permission errors: Verify GitHub token has `contents: write`

### Wrong Version Released

1. Delete the tag locally and remotely:
   ```bash
   git tag -d v1.1.0
   git push origin :refs/tags/v1.1.0
   ```

2. Delete the GitHub Release (in GitHub UI)

3. Fix issues and create tag again

### Binary Doesn't Work

The verification step should catch this, but if not:

1. Download the artifact from Actions
2. Test locally:
   ```bash
   ./dbdump-darwin-arm64 --help
   ```
3. Check for missing dependencies
4. Re-run release workflow if needed

## Architecture

### Workflow Files

- `.github/workflows/test.yml` - CI testing on push/PR
- `.github/workflows/release.yml` - Release builds and publishing
- `.github/workflows/pr-checks.yml` - Additional PR validation
- `.github/dependabot.yml` - Automated dependency updates

### Build Process

The release workflow uses `make build-all` which:
1. Sets version info via ldflags
2. Cross-compiles for 5 platforms
3. Outputs to `bin/` directory
4. Each binary is named: `dbdump-{os}-{arch}[.exe]`

### Platforms Supported

| OS | Architecture | Binary Name | Archive Format |
|----|--------------|-------------|----------------|
| macOS | AMD64 | dbdump-darwin-amd64 | .tar.gz |
| macOS | ARM64 | dbdump-darwin-arm64 | .tar.gz |
| Linux | AMD64 | dbdump-linux-amd64 | .tar.gz |
| Linux | ARM64 | dbdump-linux-arm64 | .tar.gz |
| Windows | AMD64 | dbdump-windows-amd64.exe | .zip |

## GitHub Actions Permissions

The release workflow requires:
- `contents: write` - To create releases and upload assets
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions

These are already configured in the workflow files.

## Example Release

Here's what happens when you push `v1.1.0`:

```
1. Tag pushed: v1.1.0
   ├─ Trigger: release.yml workflow
   │
2. Test Job (~2 min)
   ├─ Security tests ✓
   └─ Unit tests ✓
   │
3. Build Job (~3 min)
   ├─ Build macOS AMD64 ✓
   ├─ Build macOS ARM64 ✓
   ├─ Build Linux AMD64 ✓
   ├─ Build Linux ARM64 ✓
   ├─ Build Windows AMD64 ✓
   ├─ Create archives ✓
   ├─ Generate checksums ✓
   └─ Upload artifacts ✓
   │
4. Release Job (~1 min)
   ├─ Download artifacts ✓
   ├─ Extract CHANGELOG ✓
   ├─ Create GitHub Release ✓
   └─ Upload binaries ✓
   │
5. Verify Job (~2 min)
   ├─ Test on Ubuntu ✓
   ├─ Test on macOS ✓
   └─ Test on Windows ✓
   │
✅ Release v1.1.0 published!
   https://github.com/HelgeSverre/dbdump/releases/tag/v1.1.0
```

Total time: ~8 minutes

## Support

For issues with the release process:
1. Check GitHub Actions logs
2. Review this document
3. Open an issue on GitHub
