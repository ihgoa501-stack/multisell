#!/usr/bin/env bash
# =============================================================================
# 凌镜 LingMirror — 数据库备份脚本
# 用 pg_dump 备份 multisell 数据库，支持可选 S3 同步和自动清理旧备份。
# =============================================================================
set -euo pipefail

# ---- 默认值 ----------------------------------------------------------------
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_NAME="${DB_NAME:-multisell}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
BACKUP_S3_BUCKET="${BACKUP_S3_BUCKET:-}"

# ---- 前置检查 ----------------------------------------------------------------
command -v pg_dump >/dev/null 2>&1 || { echo "[ERROR] pg_dump not found. Install postgresql client tools."; exit 1; }
command -v gzip    >/dev/null 2>&1 || { echo "[ERROR] gzip not found."; exit 1; }

# ---- 创建备份目录 ------------------------------------------------------------
mkdir -p "$BACKUP_DIR"

# ---- 备份 --------------------------------------------------------------------
TIMESTAMP=$(date +%Y-%m-%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/${DB_NAME}_${TIMESTAMP}.sql.gz"

export PGPASSWORD="$DB_PASSWORD"

echo "[INFO] Starting backup: ${DB_NAME}@${DB_HOST}:${DB_PORT} -> ${BACKUP_FILE}"

pg_dump \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    --format=custom \
    --verbose \
    2>/dev/null \
  | gzip > "$BACKUP_FILE"

# ---- 验证备份文件 ------------------------------------------------------------
if [ ! -s "$BACKUP_FILE" ]; then
    echo "[ERROR] Backup file is empty or was not created."
    rm -f "$BACKUP_FILE"
    exit 1
fi

BACKUP_SIZE=$(stat -f%z "$BACKUP_FILE" 2>/dev/null || stat -c%s "$BACKUP_FILE" 2>/dev/null)
BACKUP_SIZE_HUMAN=$(numfmt --to=iec "$BACKUP_SIZE" 2>/dev/null || echo "${BACKUP_SIZE} bytes")

echo "[OK] Backup created successfully:"
echo "    Path: ${BACKUP_FILE}"
echo "    Size: ${BACKUP_SIZE_HUMAN}"

# ---- 清理旧备份 --------------------------------------------------------------
echo "[INFO] Cleaning backups older than ${RETENTION_DAYS} days in ${BACKUP_DIR}..."
find "$BACKUP_DIR" -name "${DB_NAME}_*.sql.gz" -type f -mtime "+${RETENTION_DAYS}" -print -delete

# ---- S3 上传（可选） ----------------------------------------------------------
if [ -n "$BACKUP_S3_BUCKET" ]; then
    if command -v aws >/dev/null 2>&1; then
        S3_KEY="backups/${DB_NAME}_${TIMESTAMP}.sql.gz"
        echo "[INFO] Uploading to s3://${BACKUP_S3_BUCKET}/${S3_KEY} ..."
        aws s3 cp "$BACKUP_FILE" "s3://${BACKUP_S3_BUCKET}/${S3_KEY}"
        echo "[OK] S3 upload complete."
    else
        echo "[WARN] BACKUP_S3_BUCKET is set but 'aws' CLI not found. Skipping S3 upload."
    fi
fi

unset PGPASSWORD
exit 0
