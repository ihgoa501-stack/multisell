#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-$ROOT_DIR/backend-go/migrations}"
DATABASE_URL="${DATABASE_URL:-}"

if [[ -z "$DATABASE_URL" ]]; then
  cat >&2 <<'EOF'
DATABASE_URL is required.

Example:
  DATABASE_URL="postgresql://postgres:postgres@localhost:5432/multisell_test?sslmode=disable" scripts/check_migrations.sh
EOF
  exit 2
fi

if ! command -v migrate >/dev/null 2>&1; then
  echo "migrate CLI is required. Install with:" >&2
  echo "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" >&2
  exit 2
fi

echo "==> Checking migrations up"
migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" up

echo "==> Checking latest migration down"
migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" down 1

echo "==> Checking latest migration up again"
migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" up 1

echo "==> Migration rollback smoke passed"
