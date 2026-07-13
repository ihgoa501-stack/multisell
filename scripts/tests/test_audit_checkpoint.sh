#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT HUP INT TERM
mkdir -p "$TMP/bin" "$TMP/checkpoints"

cat > "$TMP/bin/psql" <<'SH'
#!/bin/sh
case "$*" in
  *"audit_operation_chain_status"*) printf '%s\n' "${FAKE_CHAIN_STATUS:-7|0|1|0|0|1|7}" ;;
  *) printf '7|abc123\n' ;;
esac
SH

cat > "$TMP/bin/aws" <<'SH'
#!/bin/sh
printf '%s\n' "$*" >> "${FAKE_AWS_LOG:?}"
case "$*" in
  *"get-bucket-versioning"*) printf '%s\n' "${FAKE_VERSIONING:-Enabled}" ;;
  *"get-object-lock-configuration"*) printf '%s\n' "${FAKE_RETENTION_MODE:-COMPLIANCE}" ;;
  *"head-object"*"ObjectLockMode"*) printf '%s\t%s\n' "${FAKE_OBJECT_LOCK_MODE:-COMPLIANCE}" "${FAKE_RETAIN_UNTIL:-2099-01-01T00:00:00Z}" ;;
esac
SH

chmod +x "$TMP/bin/psql" "$TMP/bin/aws"

PATH="$TMP/bin:$PATH" DATABASE_URL=postgresql://test AUDIT_CHECKPOINT_KEY=test-key \
  AUDIT_CHECKPOINT_DIR="$TMP/checkpoints" AUDIT_CHECKPOINT_REQUIRE_OFFSITE=false \
  "$ROOT/scripts/audit_checkpoint.sh" >/dev/null
[ "$(find "$TMP/checkpoints" -name '*.json' | wc -l | tr -d ' ')" = "1" ]
grep -Eq '"last_id":7.*"last_hash":"abc123".*"hmac_sha256":"[0-9a-f]{64}"' "$TMP/checkpoints"/*.json

if PATH="$TMP/bin:$PATH" FAKE_CHAIN_STATUS='7|0|1|0|1|2|7' \
  DATABASE_URL=postgresql://test AUDIT_CHECKPOINT_KEY=test-key \
  AUDIT_CHECKPOINT_DIR="$TMP/checkpoints" AUDIT_CHECKPOINT_REQUIRE_OFFSITE=false \
  "$ROOT/scripts/audit_checkpoint.sh" >/dev/null 2>&1; then
  echo "checkpoint unexpectedly accepted a forked audit chain" >&2
  exit 1
fi

if PATH="$TMP/bin:$PATH" DATABASE_URL=postgresql://test AUDIT_CHECKPOINT_KEY=test-key \
  AUDIT_CHECKPOINT_DIR="$TMP/checkpoints" AUDIT_CHECKPOINT_REQUIRE_OFFSITE=true \
  AUDIT_CHECKPOINT_S3_URI= "$ROOT/scripts/audit_checkpoint.sh" >/dev/null 2>&1; then
  echo "checkpoint unexpectedly succeeded without mandatory offsite destination" >&2
  exit 1
fi

: > "$TMP/aws.log"
PATH="$TMP/bin:$PATH" FAKE_AWS_LOG="$TMP/aws.log" DATABASE_URL=postgresql://test \
  AUDIT_CHECKPOINT_KEY=test-key AUDIT_CHECKPOINT_DIR="$TMP/checkpoints" \
  AUDIT_CHECKPOINT_REQUIRE_OFFSITE=true AUDIT_CHECKPOINT_REQUIRE_IMMUTABLE_OFFSITE=true \
  AUDIT_CHECKPOINT_S3_URI=s3://audit-bucket/checkpoints \
  "$ROOT/scripts/audit_checkpoint.sh" >/dev/null
[ "$(wc -l < "$TMP/aws.log" | tr -d ' ')" = "4" ]

if PATH="$TMP/bin:$PATH" FAKE_AWS_LOG="$TMP/aws.log" FAKE_VERSIONING=Suspended \
  DATABASE_URL=postgresql://test AUDIT_CHECKPOINT_KEY=test-key \
  AUDIT_CHECKPOINT_DIR="$TMP/checkpoints" AUDIT_CHECKPOINT_REQUIRE_OFFSITE=true \
  AUDIT_CHECKPOINT_REQUIRE_IMMUTABLE_OFFSITE=true \
  AUDIT_CHECKPOINT_S3_URI=s3://audit-bucket/checkpoints \
  "$ROOT/scripts/audit_checkpoint.sh" >/dev/null 2>&1; then
  echo "checkpoint unexpectedly accepted suspended bucket versioning" >&2
  exit 1
fi

echo "audit checkpoint script contract verified"
