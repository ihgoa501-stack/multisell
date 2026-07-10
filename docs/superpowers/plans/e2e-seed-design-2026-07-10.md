# E2E Seed Design & Product Loop Path — 2026-07-10

## Part A: 5 Seed Scenarios

### Scenario 1: Profitable Listing ("Premium Wireless Earbuds")

**What it tests:** Full completeness, positive margin, ready to approve.

| Table | Key Data |
|---|---|
| `candidate_product` | title="Premium Wireless Earbuds", purchase_price=45.00 (CNY), target_sale_price=29.99 (USD), package_weight_kg=0.15, dimensions=8/4/3, hs_code=8518.30, supplier_id=1, all completeness fields filled |
| `completeness_check` | score=85, status="complete", missing_items=`[]` |
| `profit_summary` | purchase_cost=6.25, shipping_cost=8.50, platform_fee=4.50, total_cost=19.25, target_revenue=29.99, estimated_profit=10.74, profit_margin=35.80, status="profitable" |
| `listing_recommendation` | decision="list", confidence=0.95, risk_flags=`[]` |

**Expected API response** (`/v1/owner/suggestions` entry):
```json
{
  "id": 1, "product_id": 101, "product_title": "Premium Wireless Earbuds",
  "completeness_score": 85, "profit_margin": 35.80, "estimated_profit": 10.74,
  "decision": "list", "confidence": 0.95,
  "reason": "资料完整（85%），利润良好（35.80%），建议上架。",
  "risk_level": "low", "risk_flags": "[]"
}
```

**Expected frontend display:** Green "推荐上架" tag, green profit margin stat card ("高利润"), approve button enabled.

---

### Scenario 2: Loss-Making Listing ("Unknown Brand Charger")

**What it tests:** Negative margin detection, skip/flag behavior.

| Table | Key Data |
|---|---|
| `candidate_product` | title="Unknown Brand Charger", purchase_price=85.00 (CNY), target_sale_price=9.99 (USD), all fields present |
| `completeness_check` | score=75, status="incomplete", missing_items=`["main_image", "supplier_id"]` |
| `profit_summary` | purchase_cost=11.81, shipping_cost=3.50, platform_fee=1.50, total_cost=16.81, target_revenue=9.99, estimated_profit=-6.82, profit_margin=-68.27, status="unprofitable" |
| `listing_recommendation` | decision="skip", confidence=0.30, risk_flags=`["负利润"]` |

**Expected API response:**
```json
{
  "id": 2, "product_id": 102, "product_title": "Unknown Brand Charger",
  "completeness_score": 75, "profit_margin": -68.27, "estimated_profit": -6.82,
  "decision": "skip", "confidence": 0.30,
  "reason": "利润为负（unprofitable: -68.27%），成本（$16.81）高于目标售价（$9.99）。",
  "risk_level": "high", "risk_flags": "[\"负利润\"]"
}
```

**Expected frontend display:** Red "不建议上架" tag, negative number shown in red, approve button disabled/hidden, profit warning tooltip.

---

### Scenario 3: Missing Logistics Fee ("Leather Phone Case")

**What it tests:** Incomplete candidate (no package weight), missing shipping cost in profit calc.

| Table | Key Data |
|---|---|
| `candidate_product` | title="Leather Phone Case", purchase_price=20.00 (CNY), target_sale_price=15.99 (USD), **package_weight_kg=0** (not set), package dimensions empty, hs_code present |
| `completeness_check` | score=40, status="incomplete", missing_items=`["package_weight_kg", "package_length_cm", "package_width_cm", "package_height_cm"]` |
| `profit_summary` | purchase_cost=2.78, shipping_cost=0, platform_fee=2.40, total_cost=5.18, target_revenue=15.99, estimated_profit=10.81, profit_margin=67.60, status="profitable" — but shipping_cost=0 means unreliable |
| `listing_recommendation` | decision="skip" (completeness < 50), confidence=0.25, risk_flags=`["资料严重缺失"]` |

**Expected API response:**
```json
{
  "id": 3, "product_id": 103, "product_title": "Leather Phone Case",
  "completeness_score": 40, "profit_margin": 67.60, "estimated_profit": 10.81,
  "decision": "skip", "confidence": 0.25,
  "reason": "资料完整度过低（评分 <50），不建议上架。请先补充：package_weight_kg、package_length_cm、package_width_cm、package_height_cm",
  "risk_level": "high"
}
```

**Expected frontend display:** "资料不完整" tag, completeness score bar in red (<50), profit margin shown but with a "物流费缺失" warning, "补充资料" action button enabled.

---

### Scenario 4: Missing Platform/Category Fee ("Portable Bluetooth Speaker")

**What it tests:** Moderate completeness + only marginal profit when platform fee is missing.

| Table | Key Data |
|---|---|
| `candidate_product` | title="Portable Bluetooth Speaker", purchase_price=110.00 (CNY), target_sale_price=22.00 (USD), all fields present except supplier and hs_code |
| `completeness_check` | score=75, status="incomplete", missing_items=`["supplier_id", "hs_code"]` |
| `profit_summary` | purchase_cost=15.28, shipping_cost=4.00, platform_fee=0 (no rule matched), total_cost=19.28, target_revenue=22.00, estimated_profit=2.72, profit_margin=12.36, status="marginal" |
| `listing_recommendation` | decision="cautious" (profit_margin < 15, completeness < 80), confidence=0.60, risk_flags=`["利润偏低"]` |

**Expected API response:**
```json
{
  "id": 4, "product_id": 104, "product_title": "Portable Bluetooth Speaker",
  "completeness_score": 75, "profit_margin": 12.36, "estimated_profit": 2.72,
  "decision": "cautious", "confidence": 0.60,
  "reason": "条件适中：完整度 75%，利润率 12.36%。建议在补充资料或确认成本后决定。",
  "risk_flags": "[\"利润偏低\"]"
}
```

**Expected frontend display:** Yellow "谨慎上架" tag, profit margin with warning color, completeness score moderate.

---

### Scenario 5: Approval-to-Sandbox Pipeline ("Eco-Friendly Water Bottle")

**What it tests:** Full candidate → completeness → profit → recommendation → approval → listing task pipeline.

| Table | Key Data |
|---|---|
| `candidate_product` | title="Eco-Friendly Water Bottle", purchase_price=25.00 (CNY), target_sale_price=19.99 (USD), all completeness fields present, target_platform_id=1 |
| `completeness_check` | score=95, status="complete", missing_items=`[]` |
| `profit_summary` | purchase_cost=3.47, shipping_cost=4.00, platform_fee=3.00, total_cost=10.47, target_revenue=19.99, estimated_profit=9.52, profit_margin=47.62, status="profitable" |
| `listing_recommendation` | decision="list", confidence=0.95, risk_flags=`[]`, feedback_status="pending", created_listing_task_id=1 |
| `listing_task` | id=1, product_id=105, status="blocked", platform_id=1, approval_id=1 |
| `approval_request` | id=1, product_id=105, status="pending", request_type="listing_task", entity_type="listing_task", entity_id=1 |
| `platform_fee_rule` | platform_id=1, fee_type="commission", fee_rate_pct=15.00 |

**Expected API response:**
```json
{
  "id": 5, "product_id": 105, "product_title": "Eco-Friendly Water Bottle",
  "completeness_score": 95, "profit_margin": 47.62, "estimated_profit": 9.52,
  "decision": "list", "confidence": 0.95,
  "reason": "资料完整（95%），利润良好（47.62%），建议上架。",
  "risk_level": "low", "risk_flags": "[]",
  "listing_task_id": 1, "task_status": "blocked",
  "approval_id": 1, "approval_status": "pending"
}
```

**Expected frontend display:** Green "推荐上架" tag. Listing task shows as "已阻断" in listing-tasks page. Approval request shows "待审批" with approve/reject buttons. After approve → task shows "待审批" status.

---

## Part B: demo_seed.go Changes

Safe to add. Observations:
- `demo_seed.go` currently seeds only A5 stock_alert demo data (1 brand, 1 category, 1 product, 1 SKU, 1 inventory)
- Adding new candidate products, completeness checks, profit summaries, recommendations, and listing tasks does not conflict — these are separate tables
- The `e2e_seed.sh` just calls `go run scripts/demo_seed.go` — no changes needed there
- All inserts use `FirstOrCreate` / idempotent patterns matching existing code style

**Tables to seed (all new, no schema changes):**
- `candidate_product` x 5 records
- `completeness_check` x 5 records
- `profit_summary` x 5 records
- `listing_recommendation` x 5 records
- `listing_task` x 1 record (scenario 5)
- `approval_request` x 1 record (scenario 5)
- `platform_fee_rule` x 1 record (scenario 5 — commission rate for platform 1)

**Seed function refactoring:** Add a `seedProductLoopData(db)` function alongside the existing code, called from `seed()`. Does not touch existing A5 data.

---

## Part C: Playwright E2E Path Design

### What exists now

1. **`main-chain.spec.ts`** — Tests the AI Agent flow: login → dashboard → AI command center → agent run (A5 stock_alert) → trace replay → action review → approve → execute → status. This is the AGENT LOOP, NOT the Product Loop.

2. **`owner-approval.spec.ts`** — Tests the Owner cockpit page via API MOCKING. The mock returns static data. This test does NOT call the real backend product loop API.

### What's missing

The spec defines the Product Loop as:

```
Candidate product → completeness check → cost/logistics/platform fee/profit calculation → listing recommendation → Owner approval → controlled listing task → result review
```

**No existing E2E test covers this real backend flow.**

### Minimum Playwright test to add

A new file: `frontend-next/e2e/tests/product-loop.spec.ts`

**Test 1: Profitable candidate → Owner sees recommendation → Approve → Listing task created.**

1. Login via UI
2. Navigate to `/candidates` → verify scenario 1 (Premium Wireless Earbuds) appears with "可上架"
3. Click "Evaluate" or trigger evaluation via API call
4. Navigate to `/owner` → verify recommendation table shows scenario 1 with green "推荐上架" tag
5. Click "批准" → confirm modal → verify success indicator
6. Navigate to `/listing-tasks` → verify a listing task exists with status "待审批"

**Test 2: Loss-making candidate shows as "不建议上架".**

1. Navigate to `/candidates`
2. Verify scenario 2 (Unknown Brand Charger) shows red "不建议上架" decision

**Test 3: Candidate with missing data shows incomplete state.**

1. Navigate to `/candidates`
2. Verify scenario 3 (Leather Phone Case) shows "资料不完整" tag
3. Verify completeness score is low (< 50)

### Command to run

```bash
cd frontend-next/e2e && npx playwright test tests/product-loop.spec.ts
# or run all:
cd frontend-next/e2e && npx playwright test
```

### Prerequisites

1. Backend running on :8080
2. DB migrated and seeded (`scripts/e2e_seed.sh`)
3. Frontend dev server on :3000
4. Test user exists (same `e2e@lingmirror.test` / `e2e-password-123` pattern)

### Key learning from existing tests

- `main-chain.spec.ts` uses real login + real API calls — pattern to replicate
- `owner-approval.spec.ts` uses fake JWT + route interception — NOT suitable for real backend E2E
- The product loop test should follow `main-chain.spec.ts` pattern: real login, real API calls, verify frontend with real data

### Risk: Owner page acceptance criteria

The existing `owner-approval.spec.ts` has specific stat card assertions that won't match real seed data (it expects `pending_approvals: 1`, `low_profit_products: 3`, etc.). The new `product-loop.spec.ts` should not duplicate those assertions. The mocked test stays as a fast CI check; the real-data test is the business acceptance gate.

---

## Implementation Summary

| Item | File | Action |
|---|---|---|
| Seed scenarios 1-5 data | `backend-go/scripts/demo_seed.go` | ADD product loop data in new `seedProductLoopData()` function |
| E2E seed script | `scripts/e2e_seed.sh` | NO CHANGE (already calls demo_seed.go) |
| Product Loop E2E test | `frontend-next/e2e/tests/product-loop.spec.ts` | NEW FILE (design only, not implementing) |
| Design document | `docs/superpowers/plans/e2e-seed-design-2026-07-10.md` | THIS FILE |

## Skip List

- No schema changes needed — all tables already exist
- No new dependencies
- No frontend page modifications
- No handler/route changes
- No modification to existing E2E tests — only add new ones
