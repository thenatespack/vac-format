#!/bin/bash
set -e

# Output directory
BIN_DIR="bin"
mkdir -p $BIN_DIR

# Define platforms and architectures
OS_LIST=("darwin" "linux" "windows")
ARCH_LIST=("amd64" "arm64")

# Name of your app
APP_NAME="vac-format"

# Loop over all OS/ARCH combinations
for OS in "${OS_LIST[@]}"; do
  for ARCH in "${ARCH_LIST[@]}"; do
    OUTPUT="$BIN_DIR/$APP_NAME-$OS-$ARCH"
    
    # Add .exe for Windows
    if [ "$OS" == "windows" ]; then
      OUTPUT="$OUTPUT.exe"
    fi

    echo "Building $OUTPUT ..."
    GOOS=$OS GOARCH=$ARCH go build -o $OUTPUT ./cmd/vac-format

    # Make macOS/Linux executables executable
    if [ "$OS" != "windows" ]; then
      chmod +x $OUTPUT
    fi
  done
done

echo "All builds complete! Executables are in $BIN_DIR/"

