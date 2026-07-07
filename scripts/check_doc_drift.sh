#!/usr/bin/env bash
# Documentation Drift Checker
# Verifies that markdown file references in INDEX.md, AGENTS.md, CLAUDE.md exist.
# Usage: ./scripts/check_doc_drift.sh
# Exit 0 if all checks pass, 1 if any drift found.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exit_code=0
checked=0
missing=0

check_file() {
  local ref="$1"
  local src="$2"
  ((checked++)) || true

  # INDEX.md references can be relative to docs/ or repo root.
  # Try exact path first; if not found, try under docs/.
  local full_path="${ROOT_DIR}/${ref}"
  if [[ ! -f "$full_path" && ! -d "$full_path" ]]; then
    full_path="${ROOT_DIR}/docs/${ref}"
  fi

  if [[ ! -f "$full_path" && ! -d "$full_path" ]]; then
    echo "❌ MISSING: ${ref} (referenced in ${src})"
    ((missing++)) || true
    exit_code=1
  fi
}

# ── References in AGENTS.md and CLAUDE.md ─────────────────────────
echo "## 1. AGENTS.md / CLAUDE.md file references"
echo

for src in AGENTS.md CLAUDE.md; do
  src_path="${ROOT_DIR}/${src}"
  [[ -f "$src_path" ]] || { echo "⚠️  ${src} not found, skipping"; continue; }
  while IFS= read -r ref; do
    ref="${ref%)}"; ref="${ref%)}"; ref="${ref%,}"; ref="${ref%.}"
    ref="${ref%\'}"; ref="${ref%\"}"
    [[ -z "$ref" ]] && continue
    check_file "$ref" "$src"
  done < <(grep -oP 'docs/[^()#,\s)]+\.md' "$src_path" 2>/dev/null || true)
done

# ── References in docs/INDEX.md ──────────────────────────────────
echo "## 2. docs/INDEX.md file references"
echo

INDEX="${ROOT_DIR}/docs/INDEX.md"
if [[ -f "$INDEX" ]]; then
  while IFS= read -r ref; do
    ref="${ref%)}"; ref="${ref%)}"; ref="${ref%,}"; ref="${ref%.}"
    ref="${ref%\'}"; ref="${ref%\"}"
    [[ -z "$ref" ]] && continue
    [[ "$ref" =~ ^https?:// ]] && continue
    check_file "$ref" "docs/INDEX.md"
  done < <(grep -oP '\(([^)]+)\)' "$INDEX" | sed 's/^(//;s/)$//' 2>/dev/null || true)
fi

# ── API route consistency (skipped) ──────────────────────────────
echo "## 3. API route consistency (skipped — static scan produces false positives)"
echo
echo "⚠️  Skipped. Add a Go-based route linter in a future PR (see #306)."

# ── Summary ───────────────────────────────────────────────────────
echo
echo "---"
echo "References checked: ${checked}"
echo "Missing: ${missing}"
if [[ "$exit_code" -eq 0 ]]; then
  echo "✅ No documentation drift detected."
else
  echo "❌ Drift detected."
fi
exit "$exit_code"
