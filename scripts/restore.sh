#!/usr/bin/env bash
# =============================================================================
# 凌镜 LingMirror — 数据库恢复脚本
# 从 pg_dump custom 格式备份文件恢复 multisell 数据库。
# 使用: bash scripts/restore.sh <backup_file>
# =============================================================================
set -euo pipefail

# ---- 颜色辅助 ----------------------------------------------------------------
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ---- 参数检查 ----------------------------------------------------------------
if [ $# -lt 1 ]; then
    echo "Usage: $0 <backup_file>"
    echo ""
    echo "Arguments:"
    echo "  backup_file  路径到 .dump 或历史 .sql.gz 备份文件"
    echo ""
    echo "Environment variables (defaults in parentheses):"
    echo "  DB_HOST      (localhost)"
    echo "  DB_PORT      (5432)"
    echo "  DB_USER      (postgres)"
    echo "  DB_PASSWORD  (postgres)"
    echo "  DB_NAME      (multisell)"
    exit 1
fi

BACKUP_FILE="$1"

# ---- 默认值 ----------------------------------------------------------------
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_NAME="${DB_NAME:-multisell}"

case "$DB_NAME" in
    ""|*[!A-Za-z0-9_]*) echo "[ERROR] DB_NAME must contain only letters, digits, and underscore."; exit 1 ;;
esac

# ---- 前置检查 ----------------------------------------------------------------
command -v pg_restore >/dev/null 2>&1 || { echo "[ERROR] pg_restore not found. Install postgresql client tools."; exit 1; }
command -v psql       >/dev/null 2>&1 || { echo "[ERROR] psql not found. Install postgresql client tools."; exit 1; }

if [ ! -f "$BACKUP_FILE" ]; then
    echo "[ERROR] Backup file not found: ${BACKUP_FILE}"
    exit 1
fi

CHECKSUM_FILE="${BACKUP_FILE}.sha256"
if [ -f "$CHECKSUM_FILE" ]; then
    echo "[INFO] Verifying SHA-256 checksum..."
    if command -v sha256sum >/dev/null 2>&1; then
        (cd "$(dirname "$BACKUP_FILE")" && sha256sum -c "$(basename "$CHECKSUM_FILE")")
    elif command -v shasum >/dev/null 2>&1; then
        (cd "$(dirname "$BACKUP_FILE")" && shasum -a 256 -c "$(basename "$CHECKSUM_FILE")")
    else
        echo "[ERROR] sha256sum or shasum is required to verify the checksum."
        exit 1
    fi
fi

# ---- 验证备份文件完整性 --------------------------------------------------------
echo "[INFO] Verifying backup file integrity..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

case "$BACKUP_FILE" in
    *.gz)
        command -v gunzip >/dev/null 2>&1 || { echo "[ERROR] gunzip not found."; exit 1; }
        if ! gunzip -c "$BACKUP_FILE" > "${TMP_DIR}/backup.dump" 2>/dev/null; then
            echo "[ERROR] Failed to decompress backup file. File may be corrupt."
            exit 1
        fi
        RESTORE_FILE="${TMP_DIR}/backup.dump"
        ;;
    *) RESTORE_FILE="$BACKUP_FILE" ;;
esac

if ! pg_restore --list "$RESTORE_FILE" >/dev/null 2>&1; then
    echo "[ERROR] Backup file validation failed: pg_restore --list returned an error."
    echo "        The file may be corrupt or not a valid pg_dump custom format."
    exit 1
fi

BACKUP_SIZE=$(stat -f%z "$BACKUP_FILE" 2>/dev/null || stat -c%s "$BACKUP_FILE" 2>/dev/null)
echo "[OK] Backup archive is valid (${BACKUP_SIZE} bytes)."

# ---- 危险确认 ----------------------------------------------------------------
echo ""
echo -e "${RED}WARNING: This will DROP and recreate the '${DB_NAME}' database.${NC}"
echo -e "${RED}All data in '${DB_NAME}' will be permanently lost.${NC}"
echo ""
read -rp "Type 'yes' to continue: " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo "[INFO] Restore cancelled."
    exit 0
fi

# ---- 恢复 --------------------------------------------------------------------
export PGPASSWORD="$DB_PASSWORD"

echo ""
echo "[INFO] Dropping existing connections to ${DB_NAME}..."

# Terminate existing connections (skip if db doesn't exist yet)
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "
    SELECT pg_terminate_backend(pg_stat_activity.pid)
    FROM pg_stat_activity
    WHERE pg_stat_activity.datname = '${DB_NAME}'
      AND pid <> pg_backend_pid();
" 2>/dev/null || true

echo "[INFO] Dropping database ${DB_NAME}..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "DROP DATABASE IF EXISTS \"${DB_NAME}\";" 2>/dev/null

echo "[INFO] Creating database ${DB_NAME}..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "CREATE DATABASE \"${DB_NAME}\";" 2>/dev/null

echo "[INFO] Starting restore from: ${BACKUP_FILE}"
echo ""

if pg_restore \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    --format=custom \
    --verbose \
    --clean \
    --if-exists \
    "$RESTORE_FILE" 2>/dev/null; then
    echo ""
    echo -e "${YELLOW}[OK] Database '${DB_NAME}' restored successfully from:${NC}"
    echo "     ${BACKUP_FILE}"
else
    EXIT_CODE=$?
    echo ""
    echo "[ERROR] Restore failed with exit code ${EXIT_CODE}."
    unset PGPASSWORD
    exit $EXIT_CODE
fi

unset PGPASSWORD
exit 0
