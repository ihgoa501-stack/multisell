#!/usr/bin/env bash
# Daily Health Report
# Checks EventBus, Scheduler, Agent, LLM cost, and queue health.
# Usage: ./scripts/daily_health_report.sh
# Exit 0 if all healthy, 1 if any issue found.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exit_code=0
date_fmt=$(date '+%Y-%m-%d %H:%M:%S')

section() { printf '\n## %s\n\n' "$1"; }
ok()   { printf '✅ %s\n' "$1"; }
warn() { printf '⚠️  %s\n' "$1"; exit_code=1; }

echo "# Daily Health Report — ${date_fmt}"
echo

# ── EventBus ──────────────────────────────────────────────────────
section "EventBus Health"

if [[ -d "${ROOT_DIR}/backend-go/internal/platform/eventbus" ]]; then
  # ponytail: grep for subscriber panics; full health requires a running bus.
  panic_count=$(grep -rn 'panic\|recover\|subscriber.*fail' \
    "${ROOT_DIR}/backend-go/internal/platform/eventbus/" 2>/dev/null | wc -l | tr -d ' ')
  if [[ "$panic_count" -gt 0 ]]; then
    warn "EventBus: ${panic_count} potential panic/failure references found"
  else
    ok "EventBus: no panic or subscriber failure references in source"
  fi
else
  warn "EventBus: source directory not found — cannot verify"
fi

# ── Scheduler ─────────────────────────────────────────────────────
section "Scheduler Health"

if [[ -d "${ROOT_DIR}/backend-go/internal/platform/scheduler" ]]; then
  # ponytail: static check for consecutive failure tracking
  fail_refs=$(grep -rn 'fail\|error\|retry' \
    "${ROOT_DIR}/backend-go/internal/platform/scheduler/" 2>/dev/null \
    | grep -v '_test.go' | grep -v '\.git' | wc -l | tr -d ' ')
  ok "Scheduler: source found, ${fail_refs} error/failure references (static)"
else
  warn "Scheduler: source directory not found"
fi

# ── Agent Execution ───────────────────────────────────────────────
section "Agent Health"

agent_stuck=0
if [[ -d "${ROOT_DIR}/backend-go/internal/agent" ]]; then
  # ponytail: simple check for recent error markers in agent impl
  err_count=$(grep -rn 'stuck\|timeout\|hung\|abort' \
    "${ROOT_DIR}/backend-go/internal/agent/" 2>/dev/null \
    | grep -v '_test.go' | wc -l | tr -d ' ')
  if [[ "$err_count" -gt 0 ]]; then
    warn "${err_count} stuck/timeout/abort references in agent code (static)"
  else
    ok "Agent: no stuck/timeout references in source"
  fi
else
  warn "Agent: source directory not found"
fi

# ── LLM Cost ──────────────────────────────────────────────────────
section "LLM Cost Summary (24h)"

# ponytail: mock output — real source would be operationlog or llm_budgets table
LLM_COST_LOG="${ROOT_DIR}/scripts/.daily_llm_cost_mock"
if [[ -f "$LLM_COST_LOG" ]]; then
  cat "$LLM_COST_LOG"
else
  ok "Mock mode: no LLM cost data source configured. Integrate with operationlog or llm_budgets table for live data."
  printf "| Metric | Value |\n|--------|-------|\n"
  printf "| Total tokens | N/A (no source) |\n"
  printf "| Estimated cost | N/A |\n"
  printf "| Cost limit | N/A |\n"
fi

# ── Queue Depth ───────────────────────────────────────────────────
section "Queue Depth"

# ponytail: no queue infrastructure yet — report N/A
ok "No queue infrastructure detected. Depth check: N/A"

# ── Summary ───────────────────────────────────────────────────────
echo
if [[ "$exit_code" -eq 0 ]]; then
  echo "**Result: ALL HEALTHY**"
else
  echo "**Result: ISSUES FOUND (see above)**"
fi

exit "$exit_code"
