#!/bin/bash
# Install dependencies for GeoAgent Platform

set -e

echo "Installing Gnosis GeoAgent dependencies..."

# Detect OS
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "Detected macOS"
    
    # Check for Homebrew
    if ! command -v brew &> /dev/null; then
        echo "❌ Homebrew not found. Please install from https://brew.sh"
        exit 1
    fi  
    
    echo "Installing mlpack and armadillo..."
    brew install mlpack armadillo
    
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "Detected Linux"
    
    # Check for apt
    if command -v apt-get &> /dev/null; then
        echo "Installing mlpack and armadillo..."
        sudo apt-get update
        sudo apt-get install -y libmlpack-dev libarmadillo-dev g++ pkg-config
    else
        echo "❌ apt-get not found. Please install mlpack and armadillo manually."
        exit 1
    fi
else
    echo "❌ Unsupported OS: $OSTYPE"
    exit 1
fi

# Install Templ CLI
echo "Installing Templ CLI..."
go install github.com/a-h/templ/cmd/templ@latest

echo "✓ Dependencies installed successfully!"
echo ""
echo "Next steps:"
echo "  1. Configure your organization: configs/org.yaml"
echo "  2. Build the application: ./scripts/build.sh"
echo "  3. Run: ./bin/geoagent-* start --config configs/org.yaml"
