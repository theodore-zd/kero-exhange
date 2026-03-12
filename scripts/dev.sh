#!/bin/bash

set -euo pipefail

# Default port
PORT=${1:-8090}

cd backend
go build ./cmd/server -o ./temp/server
cd ..

sh ./scripts/migrate.sh up

cd backend
PORT=$PORT ./server
