#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <backup.dump>" >&2
    exit 1
fi

BACKUP_FILE=$1
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-postgres}
VERIFY_DB=${RESTORE_VERIFY_DB:-lingmirror_restore_verify_$$}
VERIFY_TABLES=${RESTORE_VERIFY_TABLES:-user operation_log schema_migrations}

case "$VERIFY_DB" in
    ""|*[!A-Za-z0-9_]*) echo "[ERROR] RESTORE_VERIFY_DB must be a safe PostgreSQL identifier." >&2; exit 1 ;;
esac
for table in $VERIFY_TABLES; do
    case "$table" in
        ""|*[!A-Za-z0-9_]*) echo "[ERROR] invalid verification table: $table" >&2; exit 1 ;;
    esac
done

for command in pg_restore psql; do
    command -v "$command" >/dev/null 2>&1 || { echo "[ERROR] $command not found." >&2; exit 1; }
done
[ -s "$BACKUP_FILE" ] || { echo "[ERROR] backup archive is missing or empty." >&2; exit 1; }
pg_restore --list "$BACKUP_FILE" >/dev/null

export PGPASSWORD=$DB_PASSWORD
drop_verify_db() {
    psql -X -v ON_ERROR_STOP=1 -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres \
      -c "DROP DATABASE IF EXISTS \"$VERIFY_DB\" WITH (FORCE);" >/dev/null 2>&1 || true
}
cleanup() {
    drop_verify_db
    unset PGPASSWORD
}
trap cleanup EXIT HUP INT TERM

drop_verify_db
psql -X -v ON_ERROR_STOP=1 -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres \
  -c "CREATE DATABASE \"$VERIFY_DB\";" >/dev/null

pg_restore -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$VERIFY_DB" \
  --no-owner --no-privileges --exit-on-error "$BACKUP_FILE" >/dev/null

TABLE_COUNT=$(psql -X -A -t -v ON_ERROR_STOP=1 -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$VERIFY_DB" \
  -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';")
[ "$TABLE_COUNT" -gt 0 ] || { echo "[ERROR] restored database has no public tables." >&2; exit 1; }

for table in $VERIFY_TABLES; do
    EXISTS=$(psql -X -A -t -v ON_ERROR_STOP=1 -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$VERIFY_DB" \
      -c "SELECT to_regclass('public.\"$table\"') IS NOT NULL;")
    [ "$EXISTS" = "t" ] || { echo "[ERROR] restored database is missing public.$table" >&2; exit 1; }
done

echo "backup restore verified in isolated database $VERIFY_DB ($TABLE_COUNT public tables)"
