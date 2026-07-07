#!/usr/bin/env bash
# Documentation Drift Checker
# Verifies that markdown file references in INDEX.md, AGENTS.md, CLAUDE.md
# exist, and that API route docs match actual Go router registrations.
# Enhances the doc-links CI job logic into a reusable standalone script.
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
  if [[ ! -f "${ROOT_DIR}/${ref}" && ! -d "${ROOT_DIR}/${ref}" ]]; then
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
  # Extract docs/*.md references
  while IFS= read -r ref; do
    # strip trailing punctuation, parens, quotes
    ref="${ref%)}"
    ref="${ref%)}"
    ref="${ref%,}"
    ref="${ref%.}"
    ref="${ref%\'}"
    ref="${ref%\"}"
    [[ -z "$ref" ]] && continue
    check_file "$ref" "$src"
  done < <(grep -oP 'docs/[^()#,\s)]+\.md' "$src_path" 2>/dev/null || true)
  # Extract docs/governance/*.md references
  while IFS= read -r ref; do
    ref="${ref%)}"
    ref="${ref%)}"
    ref="${ref%,}"
    ref="${ref%.}"
    [[ -z "$ref" ]] && continue
    check_file "$ref" "$src"
  done < <(grep -oP 'docs/governance/[^()#,\s)]+\.md' "$src_path" 2>/dev/null || true)
done

# ── References in docs/INDEX.md ──────────────────────────────────
echo "## 2. docs/INDEX.md file references"
echo

INDEX="${ROOT_DIR}/docs/INDEX.md"
if [[ -f "$INDEX" ]]; then
  while IFS= read -r ref; do
    ref="${ref%)}"
    ref="${ref%)}"
    ref="${ref%,}"
    ref="${ref%.}"
    ref="${ref%\'}"
    ref="${ref%\"}"
    [[ -z "$ref" ]] && continue
    # skip URLs
    [[ "$ref" =~ ^https?:// ]] && continue
    check_file "$ref" "docs/INDEX.md"
  done < <(grep -oP '\(([^)]+)\)' "$INDEX" | sed 's/^(//;s/)$//' 2>/dev/null || true)
fi

# ── API route consistency ────────────────────────────────────────
echo "## 3. API route consistency (docs vs Go router)"
echo

ROUTER="${ROOT_DIR}/backend-go/internal/httpx/router.go"
DOCS_API="${ROOT_DIR}/docs/api-inventory.md"

if [[ -f "$ROUTER" && -f "$DOCS_API" ]]; then
  # Extract registered routes from Go (method + path)
  # ponytail: naive extraction of Gin route registrations
  go_routes=$(grep -oP '\.(GET|POST|PUT|DELETE|PATCH)\([^)]*"/api/v[0-9]+/[^")]+' "$ROUTER" \
    | sed 's/\.\(GET\|POST\|PUT\|DELETE\|PATCH\)("\/api\/v[0-9]\+//' \
    | sort -u)
  # Extract documented routes from api-inventory.md
  doc_routes=$(grep -oP '/api/v[0-9]+/[a-zA-Z0-9_/-]+' "$DOCS_API" | sort -u)
  mismatch=0
  while IFS= read -r route; do
    [[ -z "$route" ]] && continue
    if ! echo "$doc_routes" | grep -qF "$route"; then
      echo "⚠️  Route in Go but NOT documented: /api/v1${route}"
      ((mismatch++)) || true
    fi
  done <<< "$go_routes"
  while IFS= read -r route; do
    [[ -z "$route" ]] && continue
    if ! echo "$go_routes" | grep -qF "$route"; then
      echo "⚠️  Route documented but NOT in Go: ${route}"
      ((mismatch++)) || true
    fi
  done <<< "$doc_routes"
  if [[ "$mismatch" -eq 0 ]]; then
    echo "✅ All API routes consistent between Go router and api-inventory.md"
  else
    exit_code=1
  fi
else
  echo "⚠️  Router or api-inventory.md not found — skipping route check"
fi

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
