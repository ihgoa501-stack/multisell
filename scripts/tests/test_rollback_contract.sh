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
build_line=$(grep -n 'build backend frontend' "$SCRIPT" | head -n 1 | cut -d: -f1)
stop_line=$(grep -n 'stop backend' "$SCRIPT" | head -n 1 | cut -d: -f1)
down_line=$(grep -n 'run --rm migrate down 1' "$SCRIPT" | head -n 1 | cut -d: -f1)
[ -n "$backup_line" ] && [ -n "$switch_line" ] && [ -n "$build_line" ] && [ -n "$stop_line" ] && [ -n "$down_line" ] || fail "rollback safety steps missing"
[ "$backup_line" -lt "$switch_line" ] || fail "backup must use the current trusted release before checkout"
[ "$build_line" -lt "$down_line" ] || fail "target images must build before database mutation"
[ "$stop_line" -lt "$down_line" ] || fail "backend writes must stop before migration rollback"
[ "$backup_line" -lt "$down_line" ] || fail "backup must complete before migration rollback"

echo "rollback safety contract verified"
