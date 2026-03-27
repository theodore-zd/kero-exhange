#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATIONS_DIR="$ROOT_DIR/migrations"
POSTGRES_ENV="$ROOT_DIR/postgres.env"

if ! command -v goose >/dev/null 2>&1; then
  echo "goose is not installed. Install it first:"
  echo "  go install github.com/pressly/goose/v3/cmd/goose@latest"
  exit 1
fi

if [[ -f "$POSTGRES_ENV" ]]; then
  set -a
  source "$POSTGRES_ENV"
  set +a
fi

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "DATABASE_URL is not set."
  echo "Set it in your environment or in $POSTGRES_ENV."
  exit 1
fi

export GOOSE_DRIVER=postgres
export GOOSE_DBSTRING="$DATABASE_URL"

command="${1:-up}"
shift || true

exec goose -dir "$MIGRATIONS_DIR" "$command"
