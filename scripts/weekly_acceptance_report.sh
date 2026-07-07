#!/usr/bin/env bash
# Weekly Acceptance Report
# Scans decision log, known issues, and git log for acceptance posture.
# Usage: ./scripts/weekly_acceptance_report.sh
# Exits 0 (report may note issues but this is informational).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
report_date=$(date '+%Y-%m-%d')
# ponytail: macOS (BSD) date works with -d on Linux too via the portable format
if date -d '7 days ago' >/dev/null 2>&1; then
  week_ago=$(date -d '7 days ago' '+%Y-%m-%d')
else
  week_ago=$(date -j -v-7d '+%Y-%m-%d' 2>/dev/null || echo 'unknown')
fi

section() { printf '\n## %s\n\n' "$1"; }

echo "# Weekly Acceptance Report — ${report_date}"
echo "Period: ${week_ago} → ${report_date}"
echo

# ── Decision Log Expiring Decisions ───────────────────────────────
section "Expiring Owner Decisions"

DECISION_LOG="${ROOT_DIR}/docs/governance/OWNER_DECISION_LOG.md"
if [[ -f "$DECISION_LOG" ]]; then
  echo "| Decision | Status | Expires | Owner |"
  echo "|----------|--------|---------|-------|"
  # ponytail: simple pattern match for decision entries with dates
  while IFS='|' read -r _ _ decision status expires owner _; do
    decision=$(echo "$decision" | xargs)
    status=$(echo "$status" | xargs)
    expires=$(echo "$expires" | xargs)
    owner=$(echo "$owner" | xargs)
    [[ -z "$decision" ]] && continue
    echo "| ${decision} | ${status} | ${expires} | ${owner} |"
  done < <(grep -i 'expir\|deadline\|due' "$DECISION_LOG" 2>/dev/null || echo "")
else
  echo "_No OWNER_DECISION_LOG.md found. Skipping._"
fi

# ── Overdue Known Issues ──────────────────────────────────────────
section "Overdue / Expiring Known Issues"

KNOWN_ISSUES="${ROOT_DIR}/docs/KNOWN_ISSUES.md"
overdue_count=0
if [[ -f "$KNOWN_ISSUES" ]]; then
  while IFS='|' read -r _ id status owner opened target _; do
    id=$(echo "$id" | xargs)
    status=$(echo "$status" | xargs)
    owner=$(echo "$owner" | xargs)
    opened=$(echo "$opened" | xargs)
    target=$(echo "$target" | xargs)
    [[ -z "$id" || "$id" == "ID" ]] && continue
    [[ "$status" != "OPEN" && "$status" != "MITIGATED" ]] && continue
    if [[ "$target" != "TBD" && -n "$target" ]]; then
      if date -d "$target" >/dev/null 2>&1; then
        target_ts=$(date -d "$target" +%s)
        now_ts=$(date +%s)
        if [[ "$target_ts" -lt "$now_ts" ]]; then
          echo "- **${id}** (${status}, owner: ${owner}, target: ${target}) — OVERDUE"
          ((overdue_count++)) || true
        fi
      fi
    fi
  done < <(grep '^|' "$KNOWN_ISSUES")
  [[ "$overdue_count" -eq 0 ]] && echo "_No overdue issues found._"
else
  echo "_No KNOWN_ISSUES.md found._"
fi

# ── Recent Git Log ────────────────────────────────────────────────
section "Acceptance-Related Commits (7 days)"

cd "$ROOT_DIR"
acceptance_commits=$(git log --oneline --since="$week_ago" --grep="accept\|verify\|gate\|govern\|review\|check" 2>/dev/null || echo "")
if [[ -n "$acceptance_commits" ]]; then
  echo '```'
  echo "$acceptance_commits"
  echo '```'
else
  echo "_No acceptance-related commits in the last 7 days._"
fi

# ── Summary ───────────────────────────────────────────────────────
echo
echo "---"
echo "Report generated: ${report_date}"
echo "Overdue issues: ${overdue_count}"
echo "This report is informational. Exit code is always 0."
exit 0
