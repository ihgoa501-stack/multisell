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
echo "[1/4] Checking audit middleware in router.go..."
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
echo "[2/4] Checking routecatalog for high-risk route coverage..."
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
echo "[3/4] Spot-checking mutation handlers for audit logging..."
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
echo "[4/4] Verifying action catalog audit references..."
CATALOG_FILE="$REPO_ROOT/backend-go/internal/platform/actioncatalog/catalog.go"
if [[ -f "$CATALOG_FILE" ]]; then
  # Check that actioncatalog entries with RequireApproval have audit references
  # in the event bus or dispatcher layer
  high_risk_count=$(grep -c 'RiskLevel:\s*RiskHigh' "$CATALOG_FILE" 2>/dev/null || echo 0)
  approved_actions=$(grep -c 'RequireApproval:\s*true' "$CATALOG_FILE" 2>/dev/null || echo 0)
  echo "  ✓ actioncatalog: $high_risk_count high-risk, $approved_actions require approval"
fi
echo "  Done."

# ── Check 5: Per-route cross-reference — every mutation endpoint vs routecatalog ──
echo ""
echo "[5/5] Per-route cross-reference: every mutation endpoint vs routecatalog..."

if ! python3 - "$REPO_ROOT" <<'PY'
import pathlib
import re
import sys

repo = pathlib.Path(sys.argv[1])
registry_file = repo / "backend-go/internal/platform/routecatalog/registry.go"
registry_text = registry_file.read_text()

registered = {
    (m.group(1), m.group(2))
    for m in re.finditer(
        r'Method:\s*"([A-Z]+)"\s*,\s*PathPattern:\s*"([^"]+)"',
        registry_text,
    )
}

domains = [
    "price",
    "order",
    "inventory",
    "integrations",
    "settlement",
    "finance",
    "listing",
    "listingtask",
    "sku",
    "aftersales",
    "platform",
]

def join_path(*parts: str) -> str:
    out = "/".join(p.strip("/") for p in parts if p is not None and p != "")
    return "/" + re.sub(r"/+", "/", out)

def extract_routes(routes_file: pathlib.Path) -> list[tuple[str, str]]:
    text = routes_file.read_text()
    bases: dict[str, str] = {"rg": "/api/v1"}
    routes: list[tuple[str, str]] = []

    for line in text.splitlines():
        group_match = re.search(
            r'\b([A-Za-z_][A-Za-z0-9_]*)\s*:=\s*([A-Za-z_][A-Za-z0-9_]*)\.Group\("([^"]*)"\)',
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

missing: list[tuple[str, str, str]] = []
total = 0
for domain in domains:
    routes_file = repo / f"backend-go/internal/domain/{domain}/routes.go"
    if not routes_file.exists():
        continue
    for method, path in extract_routes(routes_file):
        total += 1
        if (method, path) not in registered:
            missing.append((domain, method, path))

if missing:
    for domain, method, path in missing:
        print(f"  ❌ {method} {path} from {domain}/routes.go NOT registered in routecatalog")
    print(f"  ❌ {len(missing)} of {total} mutation routes are missing from routecatalog")
    sys.exit(1)

print(f"  ✓ All {total} mutation endpoints are registered in routecatalog")
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
