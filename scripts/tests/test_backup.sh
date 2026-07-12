#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT HUP INT TERM
mkdir -p "$TMP/bin" "$TMP/backups"

cat > "$TMP/bin/pg_dump" <<'SH'
#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--file" ]; then out=$2; shift 2; else shift; fi
done
[ -n "$out" ] || exit 2
printf 'valid-custom-archive\n' > "$out"
SH

cat > "$TMP/bin/pg_restore" <<'SH'
#!/bin/sh
[ "${FAKE_RESTORE_FAIL:-false}" != "true" ]
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

chmod +x "$TMP/bin/pg_dump" "$TMP/bin/pg_restore" "$TMP/bin/aws"

PATH="$TMP/bin:$PATH" BACKUP_DIR="$TMP/backups" BACKUP_REQUIRE_OFFSITE=false \
  "$ROOT/scripts/backup.sh" >/dev/null
[ "$(find "$TMP/backups" -name '*.dump' | wc -l | tr -d ' ')" = "1" ]
[ "$(find "$TMP/backups" -name '*.dump.sha256' | wc -l | tr -d ' ')" = "1" ]

if PATH="$TMP/bin:$PATH" BACKUP_DIR="$TMP/backups" BACKUP_REQUIRE_OFFSITE=true \
  BACKUP_S3_BUCKET= "$ROOT/scripts/backup.sh" >/dev/null 2>&1; then
  echo "backup unexpectedly succeeded without required offsite bucket" >&2
  exit 1
fi

: > "$TMP/aws.log"
PATH="$TMP/bin:$PATH" FAKE_AWS_LOG="$TMP/aws.log" BACKUP_DIR="$TMP/backups" \
  BACKUP_REQUIRE_OFFSITE=true BACKUP_S3_BUCKET=test-bucket \
  "$ROOT/scripts/backup.sh" >/dev/null
[ "$(wc -l < "$TMP/aws.log" | tr -d ' ')" = "4" ]

: > "$TMP/aws.log"
PATH="$TMP/bin:$PATH" FAKE_AWS_LOG="$TMP/aws.log" BACKUP_DIR="$TMP/backups" \
  BACKUP_REQUIRE_OFFSITE=true BACKUP_REQUIRE_IMMUTABLE_OFFSITE=true BACKUP_S3_BUCKET=test-bucket \
  "$ROOT/scripts/backup.sh" >/dev/null
[ "$(wc -l < "$TMP/aws.log" | tr -d ' ')" = "6" ]

if PATH="$TMP/bin:$PATH" FAKE_AWS_LOG="$TMP/aws.log" FAKE_VERSIONING=Suspended \
  BACKUP_DIR="$TMP/backups" BACKUP_REQUIRE_OFFSITE=true \
  BACKUP_REQUIRE_IMMUTABLE_OFFSITE=true BACKUP_S3_BUCKET=test-bucket \
  "$ROOT/scripts/backup.sh" >/dev/null 2>&1; then
  echo "backup unexpectedly accepted suspended S3 versioning" >&2
  exit 1
fi

if PATH="$TMP/bin:$PATH" FAKE_AWS_LOG="$TMP/aws.log" FAKE_RETAIN_UNTIL=None \
  BACKUP_DIR="$TMP/backups" BACKUP_REQUIRE_OFFSITE=true \
  BACKUP_REQUIRE_IMMUTABLE_OFFSITE=true BACKUP_S3_BUCKET=test-bucket \
  "$ROOT/scripts/backup.sh" >/dev/null 2>&1; then
  echo "backup unexpectedly accepted an object without retention" >&2
  exit 1
fi

if PATH="$TMP/bin:$PATH" FAKE_RESTORE_FAIL=true BACKUP_DIR="$TMP/backups" \
  BACKUP_REQUIRE_OFFSITE=false "$ROOT/scripts/backup.sh" >/dev/null 2>&1; then
  echo "backup unexpectedly accepted an invalid archive" >&2
  exit 1
fi

echo "backup script contract verified"
