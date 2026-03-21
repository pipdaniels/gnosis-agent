#!/bin/bash

# GNOSISAGENT Platform - Mac-Optimized Build Script
set -e

echo "================================================"
echo "Building Gnosis GNOSISAGENT Platform"
echo "================================================"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
VERSION=${VERSION:-"1.0.0"}
BUILD_DIR="bin"

# --- OS DETECTION LOGIC ---
CURRENT_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
if [ "$CURRENT_OS" == "darwin" ]; then
    echo -e "${GREEN}🍎 macOS detected. Limiting build to Darwin targets.${NC}"
    # Target only Intel (amd64) for Mac
    PLATFORMS="darwin/amd64"
else
    # Default for Linux or other systems
    PLATFORMS=${PLATFORMS:-"linux/amd64"}
fi

# Check for required tools
echo -e "${YELLOW}Checking dependencies...${NC}"

if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go not found.${NC}"; exit 1
fi

if ! command -v templ &> /dev/null; then
    echo -e "${YELLOW}⚠ Templ CLI not found. Installing...${NC}"
    go install github.com/a-h/templ/cmd/templ@latest
fi

# Check for mlpack
if ! pkg-config --exists mlpack 2>/dev/null; then
    echo -e "${YELLOW}⚠ mlpack not found. CGo bridge will be skipped.${NC}"
    SKIP_MLPACK=1
else
    echo -e "${GREEN}✓ mlpack found${NC}"
fi

# Clean and Prep
rm -rf ${BUILD_DIR}
mkdir -p ${BUILD_DIR}

echo -e "${YELLOW}Generating Templ templates...${NC}"
templ generate

# Compile MLPack C++ wrapper
if [ -z "$SKIP_MLPACK" ]; then
    echo -e "${YELLOW}Compiling MLPack C++ wrapper...${NC}"
    if [ -f "internal/ml/mlpack/anomaly_detector.cpp" ]; then
        cd internal/ml/mlpack
        g++ -std=c++17 -c anomaly_detector.cpp -o anomaly_detector.o $(pkg-config --cflags mlpack armadillo) -fPIC
        ar rcs libanomalydetector.a anomaly_detector.o
        cd ../../..
        
        export CGO_ENABLED=1
        export CGO_LDFLAGS="-L$(pwd)/internal/ml/mlpack -lanomalydetector $(pkg-config --libs mlpack armadillo)"
        export CGO_CXXFLAGS="$(pkg-config --cflags mlpack armadillo)"
    else
        SKIP_MLPACK=1
    fi
fi

# Build Loop
echo -e "${YELLOW}Building binaries...${NC}"
for platform in $PLATFORMS; do
    GOOS=${platform%/*}
    GOARCH=${platform#*/}
    output_name="GNOSISAGENT-${GOOS}-${GOARCH}"
    
    echo -e "${YELLOW}Building for $GOOS/$GOARCH...${NC}"
    
    # Correctly quoted linker flags to avoid "flag provided but not defined"
    LD_FLAGS="-s -w -X main.Version=${VERSION}"
    
    # We only use CGO if we aren't cross-compiling
    # (e.g., Building darwin/amd64 on a darwin/amd64 Mac)
    if [ -z "$SKIP_MLPACK" ] && [ "$GOOS" == "$(go env GOOS)" ]; then
        CGO_ENABLED=1 GOOS=$GOOS GOARCH=$GOARCH go build \
            -ldflags="$LD_FLAGS" \
            -o ${BUILD_DIR}/${output_name} \
            .
    else
        # Cross-compiling Darwin/amd64 on Darwin/arm64 (or vice versa) 
        # usually requires CGO_ENABLED=0 unless you have a cross-compiler.
        CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build \
            -ldflags="$LD_FLAGS" \
            -tags "nocgo" \
            -o ${BUILD_DIR}/${output_name} \
            .
    fi
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Built ${output_name} ($(du -h ${BUILD_DIR}/${output_name} | cut -f1))${NC}"
    fi
done

echo -e "${GREEN}Build completed successfully!${NC}"