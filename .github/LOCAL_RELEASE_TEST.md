# Testing Release Builds Locally

Before pushing a release tag to GitHub, you can test the entire release build process locally.

## Quick Test

```bash
# Test the release build process
make test-release VERSION=v1.1.0-test
```

This simulates what GitHub Actions will do:
1. ✅ Builds binaries for all 5 platforms
2. ✅ Generates SHA256 checksums
3. ✅ Creates compressed archives (tar.gz/zip)
4. ✅ Tests binary execution
5. ✅ Shows file types and sizes

## Manual Testing

### 1. Build All Platforms

```bash
make build-all
```

Creates binaries in `bin/`:
- `dbdump-darwin-amd64` (macOS Intel)
- `dbdump-darwin-arm64` (macOS Apple Silicon)
- `dbdump-linux-amd64` (Linux AMD64)
- `dbdump-linux-arm64` (Linux ARM64)
- `dbdump-windows-amd64.exe` (Windows)

### 2. Test a Binary

```bash
# Test the binary for your platform
./bin/dbdump-darwin-arm64 --help

# Check file type
file bin/dbdump-darwin-arm64
```

### 3. Create Checksums

```bash
cd bin
sha256sum dbdump-* > checksums.txt
cat checksums.txt
```

### 4. Create Archives

```bash
# macOS ARM64
tar -czf dbdump-v1.1.0-darwin-arm64.tar.gz dbdump-darwin-arm64

# Linux AMD64
tar -czf dbdump-v1.1.0-linux-amd64.tar.gz dbdump-linux-amd64

# Windows
zip dbdump-v1.1.0-windows-amd64.zip dbdump-windows-amd64.exe
```

### 5. Test Archive Extraction

```bash
cd /tmp
tar -xzf ~/code/dump-tool/bin/dbdump-v1.1.0-darwin-arm64.tar.gz
./dbdump-darwin-arm64 --help
```

## What Gets Built

### Binaries

| Platform | Architecture | File | Size |
|----------|--------------|------|------|
| macOS | Intel (x64) | dbdump-darwin-amd64 | ~9.7 MB |
| macOS | Apple Silicon (ARM64) | dbdump-darwin-arm64 | ~9.2 MB |
| Linux | AMD64 | dbdump-linux-amd64 | ~9.7 MB |
| Linux | ARM64 | dbdump-linux-arm64 | ~9.2 MB |
| Windows | AMD64 | dbdump-windows-amd64.exe | ~10 MB |

### Archives (Compressed)

| Archive | Format | Size |
|---------|--------|------|
| dbdump-vX.X.X-darwin-amd64.tar.gz | tar+gzip | ~5.2 MB |
| dbdump-vX.X.X-darwin-arm64.tar.gz | tar+gzip | ~4.9 MB |
| dbdump-vX.X.X-linux-amd64.tar.gz | tar+gzip | ~5.3 MB |
| dbdump-vX.X.X-linux-arm64.tar.gz | tar+gzip | ~5.0 MB |
| dbdump-vX.X.X-windows-amd64.zip | zip | ~5.5 MB |

**Total:** ~26 MB compressed (~48 MB uncompressed)

## Verification Checklist

Before pushing a release tag:

- [ ] All binaries build successfully
- [ ] Native binary executes and shows help
- [ ] File types are correct (Mach-O, ELF, PE32+)
- [ ] Archives are created properly
- [ ] Checksums file is generated
- [ ] CHANGELOG.md is updated for version
- [ ] No uncommitted changes

## Compare with GitHub Actions

The local test script (`scripts/test-release.sh`) mimics what happens in `.github/workflows/release.yml`:

| Step | Local Script | GitHub Actions |
|------|--------------|----------------|
| Clean build | ✅ `rm -rf bin/` | ✅ Fresh runner |
| Build all | ✅ `make build-all` | ✅ Same |
| Checksums | ✅ `sha256sum` | ✅ Same |
| Archives | ✅ `tar -czf`, `zip` | ✅ Same |
| Test binary | ✅ `--help` | ✅ Same on each OS |
| Upload | ❌ Local only | ✅ GitHub Release |

## Troubleshooting

### Binary Doesn't Execute

**On macOS:**
```bash
# If you get "cannot be opened because the developer cannot be verified"
xattr -d com.apple.quarantine bin/dbdump-darwin-arm64
```

**Wrong architecture:**
```bash
# Check your architecture
uname -m
# arm64 = Apple Silicon
# x86_64 = Intel

# Run the correct binary
./bin/dbdump-darwin-arm64  # For Apple Silicon
./bin/dbdump-darwin-amd64  # For Intel
```

### Build Fails

```bash
# Check Go version
go version
# Should be 1.24.6 or compatible

# Clean and rebuild
make clean
make build-all
```

### Archives Are Too Large

This is normal! Go binaries include:
- Runtime
- All dependencies
- Debug symbols (not stripped)

Compressed size (~5 MB each) is expected.

## Clean Up

```bash
# Remove test artifacts
rm -rf bin/dbdump-v*.tar.gz bin/dbdump-v*.zip
rm -f bin/checksums.txt

# Or clean everything
make clean
```

## Next Steps

After successful local testing:

```bash
# 1. Update CHANGELOG.md
vim CHANGELOG.md

# 2. Commit and push
git add CHANGELOG.md
git commit -m "chore: prepare v1.1.0 release"
git push

# 3. Create and push tag
git tag -a v1.1.0 -m "Release v1.1.0"
git push origin v1.1.0

# 4. Watch GitHub Actions
# Visit: https://github.com/HelgeSverre/dbdump/actions
```

## Example Output

```
=========================================
Local Release Build Test
=========================================

📦 Building release for version: v1.1.0-test

🧹 Cleaning previous builds...
🔨 Building binaries for all platforms...
✓ All 5 binaries built successfully

🔐 Generating checksums...
✓ Checksums generated

📦 Creating release archives...
✓ Created 5 release archives

🧪 Testing binary execution...
✓ macOS ARM64 binary works
✓ macOS AMD64 binary works

📝 Binary file types:
✓ All file types correct

💾 Archive sizes:
Total release size: ~25.9M (compressed)

=========================================
✅ Release build test complete!
=========================================
```

## See Also

- [RELEASE.md](RELEASE.md) - Complete release process
- [.github/workflows/release.yml](.github/workflows/release.yml) - GitHub Actions workflow
- [Makefile](Makefile) - Build targets
