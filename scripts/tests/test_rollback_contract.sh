#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
SCRIPT=$ROOT/scripts/rollback.sh

fail() {
  echo "rollback contract failed: $*" >&2
  exit 1
}

grep -q ': "${DOMAIN:?set the production domain}"' "$SCRIPT" || fail "DOMAIN must be required"
grep -q -- '--target' "$SCRIPT" || fail "an explicit rollback target must be required"
grep -q 'git switch --detach' "$SCRIPT" || fail "rollback must preserve branch history"
if grep -Eq 'git reset --hard|git checkout --force' "$SCRIPT"; then
  fail "destructive worktree reset is forbidden"
fi
if grep -Eq 'localhost:8080|127\.0\.0\.1:8080' "$SCRIPT"; then
  fail "production backend port must not be published for health checks"
fi
grep -q 'https://${DOMAIN}/api/health' "$SCRIPT" || fail "TLS liveness check missing"
grep -q 'https://${DOMAIN}/api/ready' "$SCRIPT" || fail "TLS readiness check missing"
grep -q './scripts/verify_prod_compose.sh' "$SCRIPT" || fail "target Compose boundary check missing"
grep -q 'exit 1' "$SCRIPT" || fail "verification failures must be fatal"

backup_line=$(grep -n 'run --rm backup' "$SCRIPT" | head -n 1 | cut -d: -f1)
switch_line=$(grep -n 'git switch --detach' "$SCRIPT" | head -n 1 | cut -d: -f1)
build_line=$(grep -n 'build backend frontend image-service image-service-migrate' "$SCRIPT" | head -n 1 | cut -d: -f1)
stop_line=$(grep -n 'stop backend image-service' "$SCRIPT" | head -n 1 | cut -d: -f1)
image_migrate_line=$(grep -n 'run --rm image-service-migrate' "$SCRIPT" | head -n 1 | cut -d: -f1)
[ -n "$backup_line" ] && [ -n "$switch_line" ] && [ -n "$build_line" ] && [ -n "$stop_line" ] && [ -n "$image_migrate_line" ] || fail "rollback safety steps missing"
[ "$backup_line" -lt "$switch_line" ] || fail "backup must use the current trusted release before checkout"
[ "$build_line" -lt "$image_migrate_line" ] || fail "target images must build before image-service migration check"
[ "$stop_line" -lt "$image_migrate_line" ] || fail "application writes must stop before image-service migration check"
[ "$backup_line" -lt "$image_migrate_line" ] || fail "backup must complete before image-service migration check"

grep -q '应用回滚必须保留数据库 migration 152' "$SCRIPT" || fail "migration 152 preservation must be explicit"
grep -q 'Owner 批准并使用异地不可变备份整库恢复' "$SCRIPT" || fail "database restore approval path missing"
if grep -Eq 'run --rm migrate (down|goto)|migrate down|migrate goto' "$SCRIPT"; then
  fail "generic main database down/revert is forbidden"
fi
grep -q 'up -d --no-deps image-service backend frontend caddy' "$SCRIPT" || fail "image-service rollback startup missing"

if reject_output=$("$SCRIPT" --revert-migration 2>&1); then
  fail "deprecated migration revert flag must fail closed"
else
  reject_status=$?
fi
[ "$reject_status" -eq 2 ] || fail "deprecated migration revert flag must exit 2"
printf '%s' "$reject_output" | grep -q 'migration 152' || fail "revert rejection must explain migration 152"

echo "rollback safety contract verified"
