#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# LingMirror / MultiSell -- End-to-End Smoke Test
#
# Walks through the core candidate-product pipeline:
#   health → register → login → seed → list → completeness → profit → loop → mock → approval
#
# Usage:
#   bash backend-go/scripts/smoke_test.sh
#   BASE_URL=http://localhost:8080 bash backend-go/scripts/smoke_test.sh
#   bash backend-go/scripts/smoke_test.sh --help
#
# Environment variables:
#   BASE_URL       server base URL (default: http://localhost:8080)
#   SMOKE_USER     test login username (default: smoke_test_user)
#   SMOKE_PASS     test login password (default: smoke_test_pass123)
#   SMOKE_DISPLAY  test user display name (default: Smoke Test User)
# ---------------------------------------------------------------------------
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
SMOKE_USER="${SMOKE_USER:-smoke_test_user}"
SMOKE_PASS="${SMOKE_PASS:-smoke_test_pass123}"
SMOKE_DISPLAY="${SMOKE_DISPLAY:-Smoke Test User}"
API="$BASE_URL/api/v1"

# Colors
RED='\033[0;31m'
GRN='\033[0;32m'
CYN='\033[0;36m'
NC='\033[0m' # No Color

# Counters
TOTAL=0
PASSED=0

# ---------------------------------------------------------------------------
# Help
# ---------------------------------------------------------------------------
if [ "${1:-}" = "--help" ] || [ "${1:-}" = "-h" ]; then
  cat <<'HELP'
LingMirror / MultiSell -- End-to-End Smoke Test

Walks through the core candidate-product pipeline:
  1. GET  /api/health              liveness check
  2. POST /api/v1/auth/register    create test user
  3. POST /api/v1/auth/login       get JWT token
  4. POST /api/v1/candidates/seed  seed candidate products (if empty)
  5. GET  /api/v1/candidates       list candidates, extract first ID
  6. POST /api/v1/completeness/check/:id  12-dimension completeness engine
  7. GET  /api/v1/profit/summary/:id      profit calculation
  8. POST /api/v1/loop/evaluate/:id       full pipeline evaluation
  9. POST /api/v1/mock/seed        seed mock orders (no-op if done)
  10. GET /api/v1/approval         list approvals

Exit 0 on success, non-zero on any failure.

Environment:
  BASE_URL       server base URL (default: http://localhost:8080)
  SMOKE_USER     test username   (default: smoke_test_user)
  SMOKE_PASS     test password   (default: smoke_test_pass123)
  SMOKE_DISPLAY  display name    (default: Smoke Test User)
HELP
  exit 0
fi

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

# Check for JSON parsing tools
HAS_JQ=false
command -v jq >/dev/null 2>&1 && HAS_JQ=true
HAS_PY3=false
command -v python3 >/dev/null 2>&1 && HAS_PY3=true

# json_extract <expr> <json> -- extracts a value from JSON
# Uses jq, python3, or grep fallback.
json_extract() {
  local expr="$1" json="$2"
  if $HAS_JQ; then
    jq -r "$expr" <<< "$json"
  elif $HAS_PY3; then
    python3 -c "
import sys, json
d = json.loads(sys.argv[1])
expr = sys.argv[2]
# Support simple dot-path expressions like .data.access_token or .data[0].id
parts = expr.strip('.').split('.')
val = d
for p in parts:
    if '[' in p and p.endswith(']'):
        key = p[:p.index('[')]
        idx = int(p[p.index('[')+1:p.index(']')])
        val = val.get(key, [{}])[idx] if key else val[idx]
    else:
        val = val.get(p, '')
print(val if val is not None else '')
" "$json" "$expr" 2>/dev/null || echo ""
  else
    # grep fallback: extract quoted string value for the last key in the path
    local key
    key=$(echo "$expr" | sed 's/^\.//;s/\[0\]//' | tr '.' '\n' | tail -1)
    grep -o "\"$key\":\"[^\"]*\"" <<< "$json" | head -1 | cut -d'"' -f4
  fi
}

# json_code <json> -- returns response envelope code
json_code() {
  if $HAS_JQ; then
    jq -r '.code' <<< "$1" 2>/dev/null || echo "-1"
  elif $HAS_PY3; then
    python3 -c "import sys,json; print(json.loads(sys.argv[1]).get('code',-1))" "$1" 2>/dev/null || echo "-1"
  else
    grep -o '"code":[0-9]*' <<< "$1" | head -1 | grep -o '[0-9]*$' || echo "-1"
  fi
}

# json_first_id <json> -- extracts the first id from a data array
json_first_id() {
  if $HAS_JQ; then
    jq -r '.data[0].id // empty' <<< "$1" 2>/dev/null
  elif $HAS_PY3; then
    python3 -c "import sys,json; d=json.loads(sys.argv[1]); print(d['data'][0]['id'])" "$1" 2>/dev/null || echo ""
  else
    grep -o '"data":\[{"id":[0-9]*' <<< "$1" | head -1 | grep -o '[0-9]*$' || echo ""
  fi
}

# json_total <json> -- extracts total count from paginated response
json_total() {
  if $HAS_JQ; then
    jq -r '.total // empty' <<< "$1" 2>/dev/null
  elif $HAS_PY3; then
    python3 -c "import sys,json; print(json.loads(sys.argv[1]).get('total',0))" "$1" 2>/dev/null || echo "0"
  else
    grep -o '"total":[0-9]*' <<< "$1" | head -1 | grep -o '[0-9]*$' || echo "0"
  fi
}

# step <num> <desc> [curl_args...] -- run a curl step with status/validation
step() {
  local num="$1" desc="$2"
  shift 2
  TOTAL=$((TOTAL + 1))

  echo ""
  echo -e "${CYN}[Step $num]${NC} $desc"

  local resp_file
  resp_file=$(mktemp)

  # Construct display version of the command (mask auth header)
  local first=true
  local in_body=false body_file
  body_file=$(mktemp)
  for arg in "$@"; do
    if $first; then
      first=false
      continue
    fi
    if [ "$arg" = "-d" ]; then
      in_body=true
    elif $in_body; then
      echo "$arg" > "$body_file"
      echo -n " -d '${arg:0:80}"
      [ "${#arg}" -gt 80 ] && echo "...'" || echo "'"
      in_body=false
    elif [ "$arg" = "-H" ]; then
      echo -n " -H 'Header'"
    else
      echo -n " $arg"
    fi
  done
  if [ -s "$body_file" ]; then
    echo -n "  Body: "
    head -c 200 "$body_file"
    echo ""
  fi
  rm -f "$body_file"
  echo "" 2>/dev/null || true

  # Execute curl
  local http_code
  http_code=$(curl -s -o "$resp_file" -w '%{http_code}' "$@" 2>/dev/null) || {
    echo -e "  ${RED}CURL FAILED${NC}"
    rm -f "$resp_file"
    exit 1
  }

  http_code="${http_code//[[:space:]]/}"
  local body
  body=$(cat "$resp_file")
  rm -f "$resp_file"

  local code
  code=$(json_code "$body" 2>/dev/null || echo "-1")

  echo "  HTTP $http_code | code=$code"

  # Accept 2xx with code=0, or specific non-2xx that mean "already done"
  if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
    if [ "$code" = "0" ] || [ "$code" = "" ] || [ -z "$code" ]; then
      PASSED=$((PASSED + 1))
      echo -e "  ${GRN}PASS${NC}"
    else
      echo -e "  ${RED}FAIL (response code=$code)${NC}"
      echo "  Response: $body"
      exit 1
    fi
  elif [ "$http_code" -eq 409 ]; then
    # 409 = duplicate user / already exists — acceptable
    PASSED=$((PASSED + 1))
    echo -e "  ${GRN}PASS (already exists)${NC}"
  else
    # Check if body mentions "already" for idempotent operations
    if echo "$body" | grep -qi "already\|already seeded"; then
      PASSED=$((PASSED + 1))
      echo -e "  ${GRN}PASS (already done)${NC}"
    else
      echo -e "  ${RED}FAIL (HTTP $http_code)${NC}"
      echo "  Response: $body"
      exit 1
    fi
  fi
}

# ---------------------------------------------------------------------------
# Pre-flight check: wait for server
# ---------------------------------------------------------------------------
echo ""
echo "╔══════════════════════════════════════════════╗"
echo "║   LingMirror E2E Smoke Test                  ║"
echo "╚══════════════════════════════════════════════╝"
echo "  Target:  $BASE_URL"
echo "  User:    $SMOKE_USER"
if $HAS_JQ; then echo "  JSON:    jq"; elif $HAS_PY3; then echo "  JSON:    python3 (jq not found)"; else echo "  JSON:    grep fallback"; fi
echo ""

echo "Waiting for server..."
for i in $(seq 1 30); do
  if curl -sf "$BASE_URL/api/health" >/dev/null 2>&1; then
    echo "  Server ready after ${i}s"
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo -e "  ${RED}Server not reachable after 30s${NC}"
    exit 1
  fi
  sleep 1
done

# ── Step 0: Health check ───────────────────────────────────────────────
step 0 "Health check (GET $BASE_URL/api/health)" \
  -X GET "$BASE_URL/api/health"

# ── Step 1: Register test user ──────────────────────────────────────────
REG_BODY=$(cat <<END
{"username":"$SMOKE_USER","password":"$SMOKE_PASS","display_name":"$SMOKE_DISPLAY","email":"$SMOKE_USER@smoke.test"}
END
)
step 1 "Register test user (POST $API/auth/register)" \
  -X POST "$API/auth/register" \
  -H "Content-Type: application/json" \
  -d "$REG_BODY"

# ── Step 2: Login, extract token ────────────────────────────────────────
LOGIN_BODY=$(cat <<END
{"username":"$SMOKE_USER","password":"$SMOKE_PASS"}
END
)
echo ""
echo -e "${CYN}[Step 2]${NC} Login (POST $API/auth/login)"
echo "  Body: $LOGIN_BODY"

LOGIN_RESP=$(curl -s -X POST "$API/auth/login" \
  -H "Content-Type: application/json" \
  -d "$LOGIN_BODY")
LOGIN_CODE=$(json_code "$LOGIN_RESP" 2>/dev/null || echo "-1")
LOGIN_HTTP=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$API/auth/login" \
  -H "Content-Type: application/json" \
  -d "$LOGIN_BODY")
LOGIN_HTTP="${LOGIN_HTTP//[[:space:]]/}"

echo "  HTTP $LOGIN_HTTP | code=$LOGIN_CODE"

if [ "$LOGIN_HTTP" -ge 200 ] && [ "$LOGIN_HTTP" -lt 300 ] && [ "$LOGIN_CODE" = "0" ]; then
  PASSED=$((PASSED + 1))
  echo -e "  ${GRN}PASS${NC}"
else
  echo -e "  ${RED}FAIL (login returned HTTP $LOGIN_HTTP, code $LOGIN_CODE)${NC}"
  echo "  Response: $LOGIN_RESP"
  exit 1
fi

TOKEN=$(json_extract ".data.access_token" "$LOGIN_RESP" 2>/dev/null || echo "")
if [ -z "$TOKEN" ]; then
  echo -e "  ${RED}FAIL: could not extract access_token${NC}"
  echo "  Response: $LOGIN_RESP"
  exit 1
fi
echo -e "  ${GRN}  Token: ${#TOKEN} chars${NC}"

AUTH=("Authorization: Bearer $TOKEN")

# ── Step 3: Seed candidates ────────────────────────────────────────────
step 3 "Seed candidate products (POST $API/candidates/seed)" \
  -X POST "$API/candidates/seed" \
  -H "${AUTH[@]}" \
  -H "Content-Type: application/json"

# ── Step 4: List candidates ────────────────────────────────────────────
step 4 "List candidates (GET $API/candidates)" \
  -X GET "$API/candidates" \
  -H "${AUTH[@]}"

CAND_RESP=$(curl -s -X GET "$API/candidates" -H "${AUTH[@]}")
FIRST_ID=$(json_first_id "$CAND_RESP")
CAND_TOTAL=$(json_total "$CAND_RESP" || echo "0")

if [ -z "$FIRST_ID" ] || [ "$FIRST_ID" = "null" ]; then
  echo -e "  ${RED}FAIL: no candidate products found${NC}"
  exit 1
fi
echo -e "  ${GRN}  First candidate ID: $FIRST_ID${NC}"
if [ -n "$CAND_TOTAL" ] && [ "$CAND_TOTAL" != "null" ] && [ "$CAND_TOTAL" != "0" ]; then
  echo -e "  ${GRN}  Total candidates: $CAND_TOTAL${NC}"
fi

# ── Step 5: Completeness check ──────────────────────────────────────────
COMPL_BODY='{"triggered_by":"smoke_test"}'
step 5 "Check completeness for candidate $FIRST_ID (POST $API/completeness/check/$FIRST_ID)" \
  -X POST "$API/completeness/check/$FIRST_ID" \
  -H "${AUTH[@]}" \
  -H "Content-Type: application/json" \
  -d "$COMPL_BODY"

COMPL_RESP=$(curl -s -X POST "$API/completeness/check/$FIRST_ID" \
  -H "${AUTH[@]}" \
  -H "Content-Type: application/json" \
  -d "$COMPL_BODY")
COMPL_SCORE=$(json_extract ".data.score" "$COMPL_RESP" 2>/dev/null || echo "0")
echo -e "  ${GRN}  Score: $COMPL_SCORE${NC}"

# ── Step 6: Profit summary ──────────────────────────────────────────────
step 6 "Get profit summary for candidate $FIRST_ID (GET $API/profit/summary/$FIRST_ID)" \
  -X GET "$API/profit/summary/$FIRST_ID" \
  -H "${AUTH[@]}"

# ── Step 7: Loop evaluate ──────────────────────────────────────────────
step 7 "Evaluate loop for candidate $FIRST_ID (POST $API/loop/evaluate/$FIRST_ID)" \
  -X POST "$API/loop/evaluate/$FIRST_ID" \
  -H "${AUTH[@]}" \
  -H "Content-Type: application/json" \
  -d "$COMPL_BODY"

# ── Step 8: Mock seed ───────────────────────────────────────────────────
step 8 "Seed mock orders (POST $API/mock/seed)" \
  -X POST "$API/mock/seed" \
  -H "${AUTH[@]}" \
  -H "Content-Type: application/json"

# ── Step 9: List approvals ──────────────────────────────────────────────
step 9 "List approvals (GET $API/approval)" \
  -X GET "$API/approval" \
  -H "${AUTH[@]}"

# ── Summary ─────────────────────────────────────────────────────────────
echo ""
echo "╔══════════════════════════════════════════════╗"
echo -e "║  ${GRN}All $PASSED/$TOTAL smoke tests passed!${NC}          ║"
echo "╚══════════════════════════════════════════════╝"
echo ""
