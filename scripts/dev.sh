#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PORT=${1:-8090}

cd "$ROOT_DIR"

./scripts/migrate.sh up

./scripts/build-server.sh

export PORT
exec ./tmp/kero-server
