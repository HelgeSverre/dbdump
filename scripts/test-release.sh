#!/bin/bash
set -euo pipefail

# Test release build locally
# Simulates what GitHub Actions does

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "========================================="
echo "Local Release Build Test"
echo "========================================="
echo ""

VERSION="${1:-v1.0.0-test}"
echo "📦 Building release for version: $VERSION"
echo ""

# Clean previous builds
echo "🧹 Cleaning previous builds..."
rm -rf bin/
mkdir -p bin/

# Build all platforms
echo "🔨 Building binaries for all platforms..."
just build-all
echo ""

# Generate checksums
echo "🔐 Generating checksums..."
cd bin
if command -v sha256sum >/dev/null 2>&1; then
    sha256sum dbdump-* > checksums.txt
else
    shasum -a 256 dbdump-* > checksums.txt
fi
echo "✓ Checksums generated:"
cat checksums.txt
echo ""

# Create archives
echo "📦 Creating release archives..."

# macOS AMD64
tar -czf "dbdump-${VERSION}-darwin-amd64.tar.gz" dbdump-darwin-amd64
echo "✓ Created dbdump-${VERSION}-darwin-amd64.tar.gz"

# macOS ARM64
tar -czf "dbdump-${VERSION}-darwin-arm64.tar.gz" dbdump-darwin-arm64
echo "✓ Created dbdump-${VERSION}-darwin-arm64.tar.gz"

# Linux AMD64
tar -czf "dbdump-${VERSION}-linux-amd64.tar.gz" dbdump-linux-amd64
echo "✓ Created dbdump-${VERSION}-linux-amd64.tar.gz"

# Linux ARM64
tar -czf "dbdump-${VERSION}-linux-arm64.tar.gz" dbdump-linux-arm64
echo "✓ Created dbdump-${VERSION}-linux-arm64.tar.gz"

# Windows AMD64
zip -q "dbdump-${VERSION}-windows-amd64.zip" dbdump-windows-amd64.exe
echo "✓ Created dbdump-${VERSION}-windows-amd64.zip"

cd ..
echo ""

# List release artifacts
echo "📋 Release artifacts:"
ls -lh bin/dbdump-"${VERSION}"-*
echo ""

# Test binary execution
echo "🧪 Testing binary execution..."
if ./bin/dbdump-darwin-arm64 --help &> /dev/null; then
    echo "✓ macOS ARM64 binary works"
else
    echo "⚠️  macOS ARM64 binary failed (might be expected on different architecture)"
fi

if ./bin/dbdump-darwin-amd64 --help &> /dev/null; then
    echo "✓ macOS AMD64 binary works"
else
    echo "⚠️  macOS AMD64 binary failed (might be expected on different architecture)"
fi

# Check file types
echo ""
echo "📝 Binary file types:"
file bin/dbdump-darwin-amd64
file bin/dbdump-darwin-arm64
file bin/dbdump-linux-amd64
file bin/dbdump-linux-arm64
file bin/dbdump-windows-amd64.exe
echo ""

# Calculate total size
echo "💾 Archive sizes:"
du -sh bin/dbdump-"${VERSION}"-* | sort -h
echo ""

echo "Total release size: $(du -ch bin/dbdump-"${VERSION}"-* | tail -1 | awk '{print $1}') (compressed)"
echo ""

echo "========================================="
echo "✅ Release build test complete!"
echo "========================================="
echo ""
echo "Release artifacts ready in: bin/"
echo ""
echo "To test extraction:"
echo "  cd /tmp"
echo "  tar -xzf ${PROJECT_ROOT}/bin/dbdump-${VERSION}-darwin-arm64.tar.gz"
echo "  ./dbdump-darwin-arm64 --help"
echo ""
echo "To create actual release:"
echo "  git tag -a ${VERSION} -m \"Release ${VERSION}\""
echo "  git push origin ${VERSION}"
echo ""
