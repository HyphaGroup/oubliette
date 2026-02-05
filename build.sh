#!/bin/bash
set -e

echo "🗝️  Building Oubliette"
echo ""

# Parse arguments
BUILD_MODE="${1:-local}"

case "$BUILD_MODE" in
  local)
    # Build container images for agent workloads
    echo "🐳 Building container images..."
    
    # Build base image first (required by other types)
    echo "   Building oubliette-base:latest..."
    docker build -f containers/base/Dockerfile -t oubliette-base:latest .
    
    # Build dev image (extends base)
    echo "   Building oubliette-dev:latest..."
    docker build -f containers/dev/Dockerfile -t oubliette-dev:latest .
    
    # Create default oubliette:latest as alias for dev
    docker tag oubliette-dev:latest oubliette:latest
    echo "   Tagged oubliette:latest -> oubliette-dev:latest"

    # Check if Apple Container is available and sync the images
    if command -v container &> /dev/null; then
        echo ""
        echo "🍎 Syncing images to Apple Container..."
        for img in oubliette-base oubliette-dev oubliette; do
            container image rm ${img}:latest 2>/dev/null || true
            docker save ${img}:latest -o /tmp/oubliette-build.tar
            container image load -i /tmp/oubliette-build.tar
            rm -f /tmp/oubliette-build.tar
            echo "   ✅ Synced ${img}:latest"
        done
    fi

    # Sync plugins to template (if they exist)
    echo ""
    echo "🔌 Syncing plugins to template..."
    mkdir -p template/.claude/plugins/marketplaces
    if [ -d "plugins/every-marketplace" ]; then
        cp -r plugins/every-marketplace template/.claude/plugins/marketplaces/
        echo "   ✅ Synced every-marketplace plugin"
    fi

    # Build Go binaries
    echo ""
    echo "⚙️  Building Oubliette server..."
    mkdir -p bin
    go build -ldflags "-X main.Version=dev" -o bin/oubliette ./cmd/server

    echo ""
    echo "✅ Build complete!"
    echo ""
    echo "To start the server:"
    echo "  ./bin/oubliette"
    echo ""
    echo "Make sure to configure config/ first:"
    echo "  cp config/credentials.json.example config/credentials.json"
    echo "  # Edit config/credentials.json with your API keys"
    ;;

  docker)
    # Build everything for Docker deployment
    echo "🐳 Building container images..."
    
    # Build base image first
    echo "   Building oubliette-base:latest..."
    docker build -f containers/base/Dockerfile -t oubliette-base:latest .
    
    # Build dev image
    echo "   Building oubliette-dev:latest..."
    docker build -f containers/dev/Dockerfile -t oubliette-dev:latest .
    
    # Tag dev as default
    docker tag oubliette-dev:latest oubliette:latest

    echo ""
    echo "🚀 Building oubliette-server:latest image..."
    docker build -f Dockerfile.server -t oubliette-server:latest .

    echo ""
    echo "✅ Docker build complete!"
    echo ""
    echo "To deploy:"
    echo "  docker-compose up -d"
    echo ""
    echo "Make sure .env is configured first:"
    echo "  cp .env.example .env"
    echo "  # Edit .env with your API keys"
    ;;

  release)
    # Build release binaries for all platforms
    VERSION="${2:-}"
    if [ -z "$VERSION" ]; then
        echo "Error: Version required for release build"
        echo "Usage: $0 release <version>"
        echo "Example: $0 release v1.0.0"
        exit 1
    fi

    echo "📦 Building release binaries for $VERSION"
    echo ""
    mkdir -p bin

    PLATFORMS=(
        "darwin/arm64"
        "darwin/amd64"
        "linux/arm64"
        "linux/amd64"
    )

    for PLATFORM in "${PLATFORMS[@]}"; do
        OS="${PLATFORM%/*}"
        ARCH="${PLATFORM#*/}"
        OUTPUT="bin/oubliette-${OS}-${ARCH}"
        
        echo "   Building $OUTPUT..."
        GOOS=$OS GOARCH=$ARCH go build -ldflags "-X main.Version=$VERSION" -o "$OUTPUT" ./cmd/server
    done

    # Generate checksums
    echo ""
    echo "   Generating checksums..."
    cd bin
    shasum -a 256 oubliette-* > checksums.txt
    cd ..

    echo ""
    echo "✅ Release build complete!"
    echo ""
    echo "Artifacts in bin/:"
    ls -la bin/oubliette-* bin/checksums.txt
    ;;

  *)
    echo "Usage: $0 [local|docker|release]"
    echo ""
    echo "  local          Build for local development (default)"
    echo "  docker         Build Docker images for deployment"
    echo "  release <ver>  Build release binaries for all platforms"
    exit 1
    ;;
esac
