#!/usr/bin/env bash
# Known Issue Deadline Checker
# Parses docs/KNOWN_ISSUES.md, finds OPEN/MITIGATED issues past target_fix_date.
# Usage: ./scripts/check_known_issues.sh
# Exit 0 if no overdue, 1 if any overdue.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KNOWN_ISSUES="${ROOT_DIR}/docs/KNOWN_ISSUES.md"
exit_code=0

# ponytail: date parsing compatible with both macOS (BSD) and GNU date
current_epoch=$(date +%s)

parse_date_epoch() {
  local d="$1"
  local epoch
  # Try GNU date (-d)
  epoch=$(date -d "$d" +%s 2>/dev/null) && echo "$epoch" && return 0
  # Try BSD date (-j -f "%Y-%m-%d")
  epoch=$(date -j -f "%Y-%m-%d" "$d" +%s 2>/dev/null) && echo "$epoch" && return 0
  echo "0"
}

if [[ ! -f "$KNOWN_ISSUES" ]]; then
  echo "⚠️  docs/KNOWN_ISSUES.md not found — skipping check"
  exit 0
fi

echo "# Known Issue Deadline Check — $(date '+%Y-%m-%d %H:%M:%S')"
echo
echo "Overdue issues (target_fix_date past today):"
echo

overdue_count=0

# Parse markdown table rows — format: | ID | Status | Owner | Opened | Target Fix | Impact | Evidence |
while IFS='|' read -r _ id status owner opened target impact evidence; do
  # Trim whitespace
  id=$(echo "$id" | xargs 2>/dev/null || echo "")
  status=$(echo "$status" | xargs 2>/dev/null || echo "")
  owner=$(echo "$owner" | xargs 2>/dev/null || echo "")
  target=$(echo "$target" | xargs 2>/dev/null || echo "")

  # Skip header and separator rows
  [[ -z "$id" ]] && continue
  [[ "$id" == "----" || "$id" == "ID" ]] && continue

  # Only OPEN or MITIGATED
  [[ "$status" != "OPEN" && "$status" != "MITIGATED" ]] && continue

  # Skip if target is TBD or empty
  [[ "$target" == "TBD" || -z "$target" ]] && continue

  target_epoch=$(parse_date_epoch "$target")
  if [[ "$target_epoch" -gt 0 && "$target_epoch" -lt "$current_epoch" ]]; then
    days_overdue=$(( (current_epoch - target_epoch) / 86400 ))
    echo "- **${id}** | Status: ${status} | Owner: ${owner} | Target: ${target} | Days overdue: ${days_overdue}"
    ((overdue_count++)) || true
  fi
done < <(grep '^|' "$KNOWN_ISSUES" || true)

if [[ "$overdue_count" -eq 0 ]]; then
  echo "_No overdue issues found._"
  echo
  echo "✅ All issues within deadline."
else
  echo
  echo "❌ ${overdue_count} issue(s) overdue."
  exit_code=1
fi

exit "$exit_code"
