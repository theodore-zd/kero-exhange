#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT_DIR="${ROOT_DIR}/bin"

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

VERSION="${VERSION:-dev}"

log_info "Building all binaries..."
log_info "  Version: $VERSION"

mkdir -p "$OUTPUT_DIR"

"${ROOT_DIR}/scripts/build-server.sh"
echo ""
"${ROOT_DIR}/scripts/build-cli.sh"

echo ""
log_success "All binaries built in ${OUTPUT_DIR}/"
ls -la "${OUTPUT_DIR}/"
