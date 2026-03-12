#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$ROOT_DIR/"
POSTGRES_ENV="$ROOT_DIR/postgres.env"

if ! command -v goose >/dev/null 2>&1; then
  echo "goose is not installed. Install it first:"
  echo "  go install github.com/pressly/goose/v3/cmd/goose@latest"
  exit 1
fi

if [[ -f "$POSTGRES_ENV" ]]; then
  # shellcheck disable=SC1090
  source "$POSTGRES_ENV"
fi

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "DATABASE_URL is not set."
  echo "Set it in your environment or in $POSTGRES_ENV."
  exit 1
fi

command="${1:-up}"
shift || true

exec goose -dir "$BACKEND_DIR/migrations" postgres "$DATABASE_URL" "$command" "$@"
