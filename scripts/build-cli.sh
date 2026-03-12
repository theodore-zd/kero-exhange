#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT_DIR="$ROOT_DIR/tmp"
BINARY_NAME="kero"

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

VERSION=${VERSION:-"dev"}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS="-s -w -X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME"

log_info "Building CLI..."
log_info "  Version:    $VERSION"
log_info "  Build Time: $BUILD_TIME"

mkdir -p "$OUTPUT_DIR"

cd "$ROOT_DIR"

if go build -ldflags "$LDFLAGS" -o "$OUTPUT_DIR/$BINARY_NAME" ./cmd/cli; then
    log_success "CLI built successfully"
    log_info "Binary: $OUTPUT_DIR/$BINARY_NAME"
    echo ""
    log_info "Usage:"
    echo "  $OUTPUT_DIR/$BINARY_NAME --help"
    echo "  $OUTPUT_DIR/$BINARY_NAME currency create --code BTC --name Bitcoin"
    echo "  $OUTPUT_DIR/$BINARY_NAME balance issue --wallet <uuid> --currency BTC --amount 100"
else
    log_error "Failed to build CLI"
    exit 1
fi
