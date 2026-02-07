#!/bin/bash

# GeoAgent Platform - Enhanced Build Script
# Builds the application with embedded assets and cross-platform support

set -e

echo "================================================"
echo "Building Gnosis GeoAgent Platform"
echo "================================================"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
VERSION=${VERSION:-"1.0.0"}
BUILD_DIR="bin"
PLATFORMS=${PLATFORMS:-"linux/amd64 darwin/amd64 darwin/arm64"}

# Check for required tools
echo -e "${YELLOW}Checking dependencies...${NC}"

# Check for Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go not found. Please install Go 1.21+${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Go found: $(go version)${NC}"

# Check for Templ CLI
if ! command -v templ &> /dev/null; then
    echo -e "${YELLOW}⚠ Templ CLI not found. Installing...${NC}"
    go install github.com/a-h/templ/cmd/templ@latest
fi
echo -e "${GREEN}✓ Templ CLI ready${NC}"

# Check for mlpack (optional, warn if not found)
if ! pkg-config --exists mlpack 2>/dev/null; then
    echo -e "${YELLOW}⚠ mlpack not found. CGo bridge will be skipped.${NC}"
    echo -e "${YELLOW}  Install with: brew install mlpack armadillo (macOS) or apt-get install libmlpack-dev (Linux)${NC}"
    SKIP_MLPACK=1
else
    echo -e "${GREEN}✓ mlpack found${NC}"
fi

# Clean previous builds
echo -e "${YELLOW}Cleaning previous builds...${NC}"
rm -rf ${BUILD_DIR}
mkdir -p ${BUILD_DIR}

# Generate Templ templates
echo -e "${YELLOW}Generating Templ templates...${NC}"
templ generate
echo -e "${GREEN}✓ Templates generated${NC}"

# Compile MLPack C++ wrapper (if mlpack is available)
if [ -z "$SKIP_MLPACK" ]; then
    echo -e "${YELLOW}Compiling MLPack C++ wrapper...${NC}"
    
    if [ -f "internal/ml/mlpack/anomaly_detector.cpp" ]; then
        cd internal/ml/mlpack
        
        # Compile C++ to object file
        g++ -std=c++17 -c anomaly_detector.cpp -o anomaly_detector.o \
            $(pkg-config --cflags mlpack armadillo) -fPIC
        
        # Create static library
        ar rcs libanomalydetector.a anomaly_detector.o
        
        cd ../../..
        echo -e "${GREEN}✓ MLPack wrapper compiled${NC}"
        
        # Set CGo flags
        export CGO_ENABLED=1
        export CGO_LDFLAGS="-L$(pwd)/internal/ml/mlpack -lanomalydetector $(pkg-config --libs mlpack armadillo)"
        export CGO_CXXFLAGS="$(pkg-config --cflags mlpack armadillo)"
    else
        echo -e "${YELLOW}⚠ MLPack source not found, skipping compilation${NC}"
        SKIP_MLPACK=1
    fi
else
    echo -e "${YELLOW}Skipping MLPack compilation${NC}"
fi

# Build for specified platforms
echo -e "${YELLOW}Building binaries...${NC}"

for platform in $PLATFORMS; do
    GOOS=${platform%/*}
    GOARCH=${platform#*/}
    output_name="geoagent-${GOOS}-${GOARCH}"
    
    if [ "$GOOS" == "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    echo -e "${YELLOW}Building for $GOOS/$GOARCH...${NC}"
    
    # Build flags
    BUILD_FLAGS="-ldflags=-w -s -X main.Version=${VERSION}"
    
    if [ -z "$SKIP_MLPACK" ] && [ "$GOOS" == "$(go env GOOS)" ] && [ "$GOARCH" == "$(go env GOARCH)" ]; then
        # Build with CGo for current platform only
        CGO_ENABLED=1 GOOS=$GOOS GOARCH=$GOARCH go build \
            $BUILD_FLAGS \
            -o ${BUILD_DIR}/${output_name} \
            ./cmd/geoagent
    else
        # Build without CGo for cross-compilation
        CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build \
            $BUILD_FLAGS \
            -tags "nocgo" \
            -o ${BUILD_DIR}/${output_name} \
            ./cmd/geoagent
    fi
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Built ${output_name}${NC}"
        
        # Show binary size
        size=$(du -h ${BUILD_DIR}/${output_name} | cut -f1)
        echo -e "${GREEN}  Size: ${size}${NC}"
    else
        echo -e "${RED}❌ Failed to build for $GOOS/$GOARCH${NC}"
        exit 1
    fi
done

echo -e "${GREEN}================================================${NC}"
echo -e "${GREEN}Build completed successfully!${NC}"
echo -e "${GREEN}================================================${NC}"
echo ""
echo "Binaries available in ${BUILD_DIR}/"
ls -lh ${BUILD_DIR}/

echo ""
echo "To run the application:"
echo "  ./${BUILD_DIR}/geoagent-$(go env GOOS)-$(go env GOARCH) --config configs/org.example.yaml"
