#!/bin/sh
# =============================================================================
# 凌镜 LingMirror — 数据库备份脚本
# 用 pg_dump 备份 multisell 数据库，支持可选 S3 同步和自动清理旧备份。
# =============================================================================
set -eu

# ---- 默认值 ----------------------------------------------------------------
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_NAME="${DB_NAME:-multisell}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
BACKUP_S3_BUCKET="${BACKUP_S3_BUCKET:-}"
BACKUP_REQUIRE_OFFSITE="${BACKUP_REQUIRE_OFFSITE:-false}"
BACKUP_REQUIRE_IMMUTABLE_OFFSITE="${BACKUP_REQUIRE_IMMUTABLE_OFFSITE:-false}"
BACKUP_S3_SSE="${BACKUP_S3_SSE:-AES256}"

# ---- 前置检查 ----------------------------------------------------------------
command -v pg_dump >/dev/null 2>&1 || { echo "[ERROR] pg_dump not found. Install postgresql client tools."; exit 1; }
command -v pg_restore >/dev/null 2>&1 || { echo "[ERROR] pg_restore not found. Install postgresql client tools."; exit 1; }

case "$BACKUP_REQUIRE_OFFSITE" in
  true|false) ;;
  *) echo "[ERROR] BACKUP_REQUIRE_OFFSITE must be true or false."; exit 1 ;;
esac
case "$BACKUP_REQUIRE_IMMUTABLE_OFFSITE" in
  true|false) ;;
  *) echo "[ERROR] BACKUP_REQUIRE_IMMUTABLE_OFFSITE must be true or false."; exit 1 ;;
esac
if [ "$BACKUP_REQUIRE_OFFSITE" = "true" ] && [ -z "$BACKUP_S3_BUCKET" ]; then
    echo "[ERROR] Offsite backup is required but BACKUP_S3_BUCKET is empty."
    exit 1
fi
if [ "$BACKUP_REQUIRE_IMMUTABLE_OFFSITE" = "true" ] && [ -z "$BACKUP_S3_BUCKET" ]; then
    echo "[ERROR] Immutable offsite backup is required but BACKUP_S3_BUCKET is empty."
    exit 1
fi
if [ -n "$BACKUP_S3_BUCKET" ] && ! command -v aws >/dev/null 2>&1; then
    echo "[ERROR] BACKUP_S3_BUCKET is set but aws CLI is unavailable."
    exit 1
fi

# ---- 创建备份目录 ------------------------------------------------------------
mkdir -p "$BACKUP_DIR"

# ---- 备份 --------------------------------------------------------------------
TIMESTAMP=$(date +%Y-%m-%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/${DB_NAME}_${TIMESTAMP}.dump"
PARTIAL_FILE="${BACKUP_FILE}.partial"
CHECKSUM_FILE="${BACKUP_FILE}.sha256"

export PGPASSWORD="$DB_PASSWORD"
cleanup() {
    rm -f "$PARTIAL_FILE"
    unset PGPASSWORD
}
trap cleanup EXIT HUP INT TERM

echo "[INFO] Starting backup: ${DB_NAME}@${DB_HOST}:${DB_PORT} -> ${BACKUP_FILE}"

pg_dump \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    --format=custom \
    --file "$PARTIAL_FILE"

# ---- 验证备份文件 ------------------------------------------------------------
if [ ! -s "$PARTIAL_FILE" ]; then
    echo "[ERROR] Backup file is empty or was not created."
    exit 1
fi
if ! pg_restore --list "$PARTIAL_FILE" >/dev/null; then
    echo "[ERROR] Backup archive failed pg_restore --list validation."
    exit 1
fi
mv "$PARTIAL_FILE" "$BACKUP_FILE"

if command -v sha256sum >/dev/null 2>&1; then
    (cd "$BACKUP_DIR" && sha256sum "$(basename "$BACKUP_FILE")") > "$CHECKSUM_FILE"
elif command -v shasum >/dev/null 2>&1; then
    (cd "$BACKUP_DIR" && shasum -a 256 "$(basename "$BACKUP_FILE")") > "$CHECKSUM_FILE"
else
    echo "[ERROR] sha256sum or shasum is required."
    exit 1
fi

BACKUP_SIZE=$(stat -f%z "$BACKUP_FILE" 2>/dev/null || stat -c%s "$BACKUP_FILE" 2>/dev/null)
BACKUP_SIZE_HUMAN=$(numfmt --to=iec "$BACKUP_SIZE" 2>/dev/null || echo "${BACKUP_SIZE} bytes")

echo "[OK] Backup created successfully:"
echo "    Path: ${BACKUP_FILE}"
echo "    Size: ${BACKUP_SIZE_HUMAN}"

# ---- S3 上传（可选） ----------------------------------------------------------
if [ -n "$BACKUP_S3_BUCKET" ]; then
    if [ "$BACKUP_REQUIRE_IMMUTABLE_OFFSITE" = "true" ]; then
        versioning=$(aws s3api get-bucket-versioning --bucket "$BACKUP_S3_BUCKET" --query Status --output text)
        [ "$versioning" = "Enabled" ] || {
            echo "[ERROR] S3 bucket versioning must be Enabled for immutable backups."
            exit 1
        }
        retention_mode=$(aws s3api get-object-lock-configuration --bucket "$BACKUP_S3_BUCKET" \
            --query ObjectLockConfiguration.Rule.DefaultRetention.Mode --output text)
        case "$retention_mode" in
          GOVERNANCE|COMPLIANCE) ;;
          *) echo "[ERROR] S3 bucket must have a default Object Lock retention policy."; exit 1 ;;
        esac
    fi
    S3_KEY="backups/${DB_NAME}_${TIMESTAMP}.dump"
    echo "[INFO] Uploading encrypted offsite backup to s3://${BACKUP_S3_BUCKET}/${S3_KEY} ..."
    aws s3 cp "$BACKUP_FILE" "s3://${BACKUP_S3_BUCKET}/${S3_KEY}" --sse "$BACKUP_S3_SSE"
    aws s3 cp "$CHECKSUM_FILE" "s3://${BACKUP_S3_BUCKET}/${S3_KEY}.sha256" --sse "$BACKUP_S3_SSE"
    for remote_key in "$S3_KEY" "${S3_KEY}.sha256"; do
        if [ "$BACKUP_REQUIRE_IMMUTABLE_OFFSITE" = "true" ]; then
            lock_metadata=$(aws s3api head-object --bucket "$BACKUP_S3_BUCKET" --key "$remote_key" \
                --query '[ObjectLockMode,ObjectLockRetainUntilDate]' --output text)
            lock_mode=${lock_metadata%%[[:space:]]*}
            retain_until=${lock_metadata#*[[:space:]]}
            case "$lock_mode" in GOVERNANCE|COMPLIANCE) ;; *)
                echo "[ERROR] Uploaded object $remote_key has no active Object Lock mode."
                exit 1
            esac
            case "$retain_until" in ""|None|null)
                echo "[ERROR] Uploaded object $remote_key has no Object Lock retain-until date."
                exit 1 ;;
            esac
        else
            aws s3api head-object --bucket "$BACKUP_S3_BUCKET" --key "$remote_key" >/dev/null
        fi
    done
    echo "[OK] Encrypted S3 upload and remote existence checks complete."
fi

# Retention runs only after every required offsite upload and remote
# verification succeeds. A failed upload must preserve older local recovery
# points for the operator.
echo "[INFO] Cleaning backups older than ${RETENTION_DAYS} days in ${BACKUP_DIR}..."
find "$BACKUP_DIR" \( -name "${DB_NAME}_*.dump" -o -name "${DB_NAME}_*.dump.sha256" \) -type f -mtime "+${RETENTION_DAYS}" -print -delete

exit 0
