#!/usr/bin/env bash
# Static Codebase Health Report
# Reports project hygiene metrics that can be gathered without a running server.
# For runtime health (EventBus, Scheduler, LLM budget, queue depth) see
# the production monitoring dashboards or run backend-go's health endpoint.
# Usage: ./scripts/daily_health_report.sh
# Exit 0 if all checks pass, 1 if any issue found.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exit_code=0
date_fmt=$(date '+%Y-%m-%d %H:%M:%S')

section() { printf '\n## %s\n\n' "$1"; }
ok()   { printf '✅ %s\n' "$1"; }
warn() { printf '⚠️  %s\n' "$1"; exit_code=1; }
info() { printf 'ℹ️  %s\n' "$1"; }

echo "# Static Codebase Health Report — ${date_fmt}"
echo

# ── Git Update Status ─────────────────────────────────────────────
section "Repository Freshness"

cd "$ROOT_DIR"
git fetch --quiet 2>/dev/null || true
behind=$(git rev-list --count HEAD..@{u} 2>/dev/null || echo "unknown")
ahead=$(git rev-list --count @{u}..HEAD 2>/dev/null || echo "unknown")
info "Branch: $(git branch --show-current 2>/dev/null || echo 'N/A')"
info "Behind remote: ${behind}"
info "Ahead of remote: ${ahead}"
if [ "$behind" != "0" ] && [ "$behind" != "unknown" ]; then
  warn "Local branch is ${behind} commit(s) behind remote — consider merging latest main"
fi

# ── Stale Branches ────────────────────────────────────────────────
section "Stale Local Branches"

# ponytail: only check branches merged into main; full stale-branch audit needs remote access.
stale=$(git branch --merged main 2>/dev/null | grep -v '^\*' | grep -v 'main$' | head -10)
if [ -n "$stale" ]; then
  info "Branches merged into main (candidates for cleanup):"
  echo "$stale" | while read -r b; do printf '  - %s\n' "$b"; done
else
  ok "No stale local branches detected"
fi

# ── Build Check ───────────────────────────────────────────────────
section "Backend Build (dry-run)"

if [ -d "${ROOT_DIR}/backend-go" ]; then
  if (cd "${ROOT_DIR}/backend-go" && go build -o /dev/null ./cmd/server/...) 2>/dev/null; then
    ok "backend-go compiles"
  else
    warn "backend-go does NOT compile — run \`go build ./...\` to see errors"
  fi
else
  warn "backend-go directory not found"
fi

# ── Gov Docs: KNOWN_ISSUES Deadline Check ────────────────────────
section "KNOWN_ISSUES Deadlines"

KI="${ROOT_DIR}/docs/KNOWN_ISSUES.md"
if [ -f "$KI" ]; then
  today=$(date '+%Y-%m-%d')
  expired=0
  # ponytail: naive line-by-line check for dates before today
  while IFS= read -r line; do
    deadline=$(echo "$line" | sed -n 's/.*|[[:space:]]*\([0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}\)[[:space:]]*|.*/\1/p')
    [ -z "$deadline" ] && continue
    if [[ "$deadline" < "$today" ]]; then
      echo "⚠️  Past deadline: ${line}"
      expired=$((expired + 1))
    fi
  done < "$KI"
  if [ "$expired" -gt 0 ]; then
    warn "${expired} known issue(s) past their deadline in KNOWN_ISSUES.md"
  else
    ok "All known issue deadlines are in the future (or today)"
  fi
else
  info "KNOWN_ISSUES.md not found — skipping deadline check"
fi

# ── Migration Safety Check ────────────────────────────────────────
section "Migration Integrity"

MIGRATIONS="${ROOT_DIR}/backend-go/migrations"
if [ -d "$MIGRATIONS" ]; then
  # Check every up.sql has a matching down.sql
  mismatch=0
  for up in "$MIGRATIONS"/*.up.sql; do
    base=$(basename "$up" .up.sql)
    down="${MIGRATIONS}/${base}.down.sql"
    if [ ! -f "$down" ]; then
      echo "⚠️  Missing down.sql for: ${base}"
      mismatch=$((mismatch + 1))
    fi
  done
  for down in "$MIGRATIONS"/*.down.sql; do
    base=$(basename "$down" .down.sql)
    up="${MIGRATIONS}/${base}.up.sql"
    if [ ! -f "$up" ]; then
      echo "⚠️  Orphaned down.sql (no matching up.sql): ${base}"
      mismatch=$((mismatch + 1))
    fi
  done
  if [ "$mismatch" -gt 0 ]; then
    warn "${mismatch} migration integrity issue(s)"
  else
    ok "All migrations have matching up/down pairs"
  fi

  # Check for duplicate version numbers
  dups=$(find "$MIGRATIONS" -name '*.up.sql' -exec basename {} \; | sed 's/_.*//' | sort | uniq -d)
  if [ -n "$dups" ]; then
    echo "⚠️  Duplicate migration version(s):"
    echo "$dups" | while read -r v; do printf '  - %s\n' "$v"; done
    warn "Duplicate migration versions found"
  else
    ok "No duplicate migration versions"
  fi
else
  info "Migrations directory not found — skipping"
fi

# ── Summary ───────────────────────────────────────────────────────
echo
if [ "$exit_code" -eq 0 ]; then
  echo "**Result: ALL CHECKS PASSED**"
else
  echo "**Result: ISSUES FOUND (see above)**"
fi

exit "$exit_code"
