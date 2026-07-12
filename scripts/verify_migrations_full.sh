#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
MIGRATIONS_DIR=${MIGRATIONS_DIR:-$ROOT/backend-go/migrations}
DATABASE_URL=${DATABASE_URL:-}

[ -n "$DATABASE_URL" ] || { echo "[ERROR] DATABASE_URL is required." >&2; exit 2; }
command -v migrate >/dev/null 2>&1 || { echo "[ERROR] migrate CLI is required." >&2; exit 2; }
command -v psql >/dev/null 2>&1 || { echo "[ERROR] psql is required." >&2; exit 2; }

DB_NAME=$(psql "$DATABASE_URL" -X -A -t -v ON_ERROR_STOP=1 -c 'SELECT current_database();')
case "$DB_NAME" in
  lingmirror_migration_verify*) ;;
  *) echo "[ERROR] refusing destructive migration verification on database: $DB_NAME" >&2; exit 1 ;;
esac

TABLE_COUNT=$(psql "$DATABASE_URL" -X -A -t -v ON_ERROR_STOP=1 \
  -c "SELECT count(*) FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog','information_schema');")
[ "$TABLE_COUNT" = "0" ] || { echo "[ERROR] verification database is not empty." >&2; exit 1; }

DATABASE_URL=$DATABASE_URL MIGRATIONS_DIR=$MIGRATIONS_DIR "$ROOT/scripts/check_migrations.sh"
migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" down -all

REMAINING=$(psql "$DATABASE_URL" -X -A -t -v ON_ERROR_STOP=1 \
  -c "SELECT count(*) FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog','information_schema') AND table_name <> 'schema_migrations';")
[ "$REMAINING" = "0" ] || { echo "[ERROR] full down left $REMAINING tables behind." >&2; exit 1; }

migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" up
VERSION=$(migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" version 2>&1 | awk '{print $1}')
[ -n "$VERSION" ] || { echo "[ERROR] could not read final migration version." >&2; exit 1; }
echo "full migration lifecycle verified on $DB_NAME at version $VERSION"
