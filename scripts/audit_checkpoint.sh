#!/bin/sh
set -eu

: "${AUDIT_CHECKPOINT_KEY:?AUDIT_CHECKPOINT_KEY is required}"

db_psql() {
  if [ -n "${DATABASE_URL:-}" ]; then
    psql "$DATABASE_URL" "$@"
  else
    : "${DB_HOST:?DB_HOST is required when DATABASE_URL is unset}"
    : "${DB_USER:?DB_USER is required when DATABASE_URL is unset}"
    : "${DB_NAME:?DB_NAME is required when DATABASE_URL is unset}"
    PGPASSWORD=${DB_PASSWORD:-} psql -h "$DB_HOST" -p "${DB_PORT:-5432}" -U "$DB_USER" -d "$DB_NAME" "$@"
  fi
}

CHECKPOINT_DIR=${AUDIT_CHECKPOINT_DIR:-./backups/audit-checkpoints}
REQUIRE_OFFSITE=${AUDIT_CHECKPOINT_REQUIRE_OFFSITE:-false}
REQUIRE_IMMUTABLE_OFFSITE=${AUDIT_CHECKPOINT_REQUIRE_IMMUTABLE_OFFSITE:-false}
S3_URI=${AUDIT_CHECKPOINT_S3_URI:-}

case "$REQUIRE_OFFSITE" in true|false) ;; *) echo "AUDIT_CHECKPOINT_REQUIRE_OFFSITE must be true or false" >&2; exit 1 ;; esac
case "$REQUIRE_IMMUTABLE_OFFSITE" in true|false) ;; *) echo "AUDIT_CHECKPOINT_REQUIRE_IMMUTABLE_OFFSITE must be true or false" >&2; exit 1 ;; esac
if [ "$REQUIRE_IMMUTABLE_OFFSITE" = "true" ] && [ -z "$S3_URI" ]; then
  echo "AUDIT_CHECKPOINT_S3_URI is required when immutable offsite checkpoint is mandatory" >&2
  exit 1
fi

mkdir -p "$CHECKPOINT_DIR"
chmod 700 "$CHECKPOINT_DIR"

status=$(db_psql -X -v ON_ERROR_STOP=1 -At -F '|' -c "SELECT * FROM audit_operation_chain_status();")
old_ifs=$IFS
IFS='|'
set -- $status
IFS=$old_ifs
total=$1 self_hash_bad=$2 roots=$3 missing_predecessors=$4 fork_points=$5 tips=$6 reachable=$7
if [ "$self_hash_bad" != "0" ] || [ "$missing_predecessors" != "0" ] || \
   [ "$fork_points" != "0" ] || [ "$reachable" != "$total" ] || \
   { [ "$total" != "0" ] && { [ "$roots" != "1" ] || [ "$tips" != "1" ]; }; }; then
  echo "audit checkpoint refused: invalid hash chain (total=$total self_hash_bad=$self_hash_bad roots=$roots missing_predecessors=$missing_predecessors forks=$fork_points tips=$tips reachable=$reachable)" >&2
  exit 1
fi

head=$(db_psql -X -v ON_ERROR_STOP=1 -At -F '|' -c "SELECT COALESCE(id,0), COALESCE(record_hash,'') FROM operation_log candidate WHERE NOT EXISTS (SELECT 1 FROM operation_log child WHERE child.previous_hash = candidate.record_hash) UNION ALL SELECT 0, '' WHERE NOT EXISTS (SELECT 1 FROM operation_log) LIMIT 1;")
last_id=${head%%|*}
last_hash=${head#*|}
created_at=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
payload=$(printf '{"created_at":"%s","last_id":%s,"last_hash":"%s"}' "$created_at" "$last_id" "$last_hash")
hmac=$(printf '%s' "$payload" | openssl dgst -sha256 -hmac "$AUDIT_CHECKPOINT_KEY" -hex | awk '{print $NF}')

stamp=$(date -u '+%Y%m%dT%H%M%SZ')
file="$CHECKPOINT_DIR/audit-checkpoint-${stamp}-${last_id}.json"
tmp="$file.tmp"
umask 077
printf '{"checkpoint":%s,"hmac_sha256":"%s"}\n' "$payload" "$hmac" > "$tmp"
mv "$tmp" "$file"

if [ -n "$S3_URI" ]; then
  command -v aws >/dev/null 2>&1 || { echo "aws CLI is required for offsite checkpoint" >&2; exit 1; }
  case "$S3_URI" in s3://*/*) ;; *) echo "AUDIT_CHECKPOINT_S3_URI must be s3://bucket/prefix" >&2; exit 1 ;; esac
  s3_path=${S3_URI#s3://}
  bucket=${s3_path%%/*}
  prefix=${s3_path#*/}
  remote_key="${prefix%/}/$(basename "$file")"
  if [ "$REQUIRE_IMMUTABLE_OFFSITE" = "true" ]; then
    versioning=$(aws s3api get-bucket-versioning --bucket "$bucket" --query Status --output text)
    [ "$versioning" = "Enabled" ] || { echo "audit checkpoint bucket versioning must be Enabled" >&2; exit 1; }
    retention_mode=$(aws s3api get-object-lock-configuration --bucket "$bucket" \
      --query ObjectLockConfiguration.Rule.DefaultRetention.Mode --output text)
    case "$retention_mode" in GOVERNANCE|COMPLIANCE) ;; *)
      echo "audit checkpoint bucket requires default Object Lock retention" >&2; exit 1 ;;
    esac
  fi
  aws s3 cp "$file" "s3://${bucket}/${remote_key}" --only-show-errors --sse AES256
  if [ "$REQUIRE_IMMUTABLE_OFFSITE" = "true" ]; then
    lock_metadata=$(aws s3api head-object --bucket "$bucket" --key "$remote_key" \
      --query '[ObjectLockMode,ObjectLockRetainUntilDate]' --output text)
    lock_mode=${lock_metadata%%[[:space:]]*}
    retain_until=${lock_metadata#*[[:space:]]}
    case "$lock_mode" in GOVERNANCE|COMPLIANCE) ;; *) echo "checkpoint object has no Object Lock mode" >&2; exit 1 ;; esac
    case "$retain_until" in ""|None|null) echo "checkpoint object has no retain-until date" >&2; exit 1 ;; esac
  else
    aws s3api head-object --bucket "$bucket" --key "$remote_key" >/dev/null
  fi
elif [ "$REQUIRE_OFFSITE" = "true" ]; then
  echo "AUDIT_CHECKPOINT_S3_URI is required when offsite checkpoint is mandatory" >&2
  exit 1
fi

printf '%s\n' "$file"
