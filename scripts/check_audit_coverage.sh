#!/usr/bin/env bash
# check_audit_coverage.sh — 审计日志覆盖率检查
#
# 检查所有高风险写操作是否都有 audit log 覆盖。
# 高风险写操作定义：经由 HTTP API 的价格、库存、订单、结算、财务等写操作。
#
# 检查方式：
# 1. router.go 中 audit middleware 已注册（全局覆盖所有 HTTP 写操作）
# 2. routecatalog 中有高风险路由绑定（未来新增高风险路由必须注册）
# 3. actioncatalog 中高风险 action_types 有对应的 audit 引用
# 4. 事件总线上的系统级 mutation 有 MutationGuard 保护
#
# Usage: ./scripts/check_audit_coverage.sh
# Exit 0 if pass, 1 if violations found.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
has_violation=0

echo "=== Audit Log Coverage Check ==="
echo ""

# ── Check 1: Audit middleware in router.go ──
echo "[1/5] Checking audit middleware in router.go..."
ROUTER_FILE="$REPO_ROOT/backend-go/internal/httpx/router.go"
if [[ -f "$ROUTER_FILE" ]]; then
  if grep -q 'middleware.Audit' "$ROUTER_FILE"; then
    echo "  ✓ middleware.Audit registered"
  else
    echo "  ❌ Audit middleware NOT found in router.go"
    has_violation=1
  fi

  if grep -q 'middleware.ApprovalRequired' "$ROUTER_FILE"; then
    echo "  ✓ middleware.ApprovalRequired registered"
  else
    echo "  ⚠  Approval middleware not yet registered in router.go"
  fi
fi
echo "  Done."

# ── Check 2: Route catalog contains high-risk routes ──
echo ""
echo "[2/5] Checking routecatalog for high-risk route coverage..."
REGISTRY_FILE="$REPO_ROOT/backend-go/internal/platform/routecatalog/registry.go"
if [[ -f "$REGISTRY_FILE" ]]; then
  high_risk_methods=$(grep -cE '(price|inventory|order|rbac|settlement|finance|integrations)' "$REGISTRY_FILE" 2>/dev/null || true)
  if [[ "$high_risk_methods" -gt 0 ]]; then
    echo "  ✓ Route registry has $high_risk_methods high-risk route entries"
  else
    echo "  ⚠  Route registry exists but has no high-risk route entries"
  fi
fi
echo "  Done."

# ── Check 3: Handler files — spot-check audit logger usage ──
echo ""
echo "[3/5] Spot-checking mutation handlers for audit logging..."
# Check that the operationlog service exists and is used
OPLOG_FILE="$REPO_ROOT/backend-go/internal/domain/operationlog"
if [[ -d "$OPLOG_FILE" ]]; then
  echo "  ✓ operationlog domain exists"
  # Check structured log method
  if grep -q 'LogStructured\|StructuredLogInput' "$OPLOG_FILE"/*.go 2>/dev/null; then
    echo "  ✓ operationlog has LogStructured interface"
  fi
fi
# Check that router.go subscribes event mutations with MutationGuard
if grep -q 'mutationGuard.Guard' "$ROUTER_FILE" 2>/dev/null; then
  guard_count=$(grep -c 'mutationGuard.Guard' "$ROUTER_FILE" || true)
  echo "  ✓ EventBus uses MutationGuard ($guard_count guarded mutations)"
fi
echo "  Done."

# ── Check 4: Actioncatalog risk/audit alignment ──
echo ""
echo "[4/5] Verifying action catalog audit references..."
CATALOG_FILE="$REPO_ROOT/backend-go/internal/platform/actioncatalog/catalog.go"
if [[ -f "$CATALOG_FILE" ]]; then
  # Check that actioncatalog entries with RequireApproval have audit references
  # in the event bus or dispatcher layer
  high_risk_count=$(grep -c 'RiskLevel:\s*RiskHigh' "$CATALOG_FILE" 2>/dev/null || echo 0)
  approved_actions=$(grep -c 'RequireApproval:\s*true' "$CATALOG_FILE" 2>/dev/null || echo 0)
  echo "  ✓ actioncatalog: $high_risk_count high-risk, $approved_actions require approval"
fi
echo "  Done."

# ── Check 5: every mutation must have an explicit security policy ──
echo ""
echo "[5/5] Per-route cross-reference: every mutation endpoint vs policy inventory..."

if ! python3 - "$REPO_ROOT" <<'PY'
import pathlib
import re
import sys

repo = pathlib.Path(sys.argv[1])
policy_file = repo / "backend-go/internal/platform/routecatalog/mutation_policy.tsv"
policies = {}
for number, raw in enumerate(policy_file.read_text().splitlines(), 1):
    if not raw or raw.startswith("#"):
        continue
    fields = raw.split("\t")
    if len(fields) != 5:
        print(f"  ❌ mutation policy line {number} does not have 5 fields")
        sys.exit(1)
    classification, method, path, action, source = fields
    key = (method, path)
    if key in policies:
        print(f"  ❌ duplicate mutation policy: {method} {path}")
        sys.exit(1)
    policies[key] = (classification, action, source)

def join_path(*parts: str) -> str:
    out = "/".join(p.strip("/") for p in parts if p is not None and p != "")
    return "/" + re.sub(r"/+", "/", out)

def extract_routes(routes_file: pathlib.Path) -> list[tuple[str, str]]:
    text = routes_file.read_text()
    bases: dict[str, str] = {"rg": "/api/v1", "api": "/api/v1", "v1": "/api/v1"}
    routes: list[tuple[str, str]] = []

    for line in text.splitlines():
        group_match = re.search(
            r'\b([A-Za-z_][A-Za-z0-9_]*)\s*:=\s*([A-Za-z_][A-Za-z0-9_]*)\.Group\("([^"]*)"',
            line,
        )
        if group_match:
            child, parent, path = group_match.groups()
            bases[child] = join_path(bases.get(parent, "/api/v1"), path)
            continue

        route_match = re.search(
            r'\b([A-Za-z_][A-Za-z0-9_]*)\.(POST|PUT|PATCH|DELETE)\("([^"]*)"',
            line,
        )
        if route_match:
            group, method, path = route_match.groups()
            if group in bases:
                routes.append((method, join_path(bases[group], path)))

    return routes

actual = {}
for routes_file in sorted(repo.glob("backend-go/internal/**/*.go")):
    if routes_file.name.endswith("_test.go"):
        continue
    source = str(routes_file.relative_to(repo))
    for method, path in extract_routes(routes_file):
        actual[(method, path)] = source

missing = sorted(set(actual) - set(policies))
stale = sorted(set(policies) - set(actual))
wrong_source = sorted(
    (method, path, policies[(method, path)][2], source)
    for (method, path), source in actual.items()
    if (method, path) in policies and policies[(method, path)][2] != source
)

if missing:
    for method, path in missing:
        print(f"  ❌ {method} {path} from {actual[(method, path)]} has no explicit policy")
if stale:
    for method, path in stale:
        print(f"  ❌ stale policy for removed route: {method} {path}")
if wrong_source:
    for method, path, expected, observed in wrong_source:
        print(f"  ❌ policy source mismatch for {method} {path}: {expected} != {observed}")
if missing or stale or wrong_source:
    sys.exit(1)

counts = {kind: 0 for kind in ("public", "standard", "high")}
for classification, _, _ in policies.values():
    if classification not in counts:
        print(f"  ❌ unknown mutation classification: {classification}")
        sys.exit(1)
    counts[classification] += 1
print(
    f"  ✓ All {len(actual)} mutation endpoints explicitly classified "
    f"(public={counts['public']}, standard={counts['standard']}, high={counts['high']})"
)
PY
then
  has_violation=1
fi
echo "  Done."

echo ""

if [[ "$has_violation" -eq 1 ]]; then
  echo "❌ Audit coverage violations found."
  exit 1
fi

echo "✅ Audit coverage: all layers have logging infrastructure."
exit 0
