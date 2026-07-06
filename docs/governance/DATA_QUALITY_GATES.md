# Data Quality Gates

Last updated: 2026-07-06

Agents produce recommendations based on the data they have. If data is incomplete, the recommendation is unreliable. This document defines minimum completeness thresholds per data domain and the rules that govern agent behavior when thresholds are not met.

## 1. Completeness Thresholds

Each measurement is a percentage: `(populated fields / total expected fields for entity) * 100`.

### Product Data Completeness

| Field | Required | Weight | Notes |
|-------|----------|--------|-------|
| Title | Yes | 25% | Must be non-empty, not placeholder text |
| Description | Yes | 25% | Must be >= 50 characters of substantive text |
| Price | Yes | 20% | Must be positive number |
| Category | Yes | 15% | Must map to platform category tree leaf node |
| Images | Yes | 15% | At least 1 image URL accessible and not broken |

**Threshold: 80%** (all 5 fields must be present for full score; missing any single field drops below threshold).

**Measurement method:** Scan the product record. Each missing or invalid field contributes 0 for its weight. Sum weighted scores. Script: `scripts/data_quality/product_completeness.go`.

**Action when below threshold:**
- Agent MUST note in output: "Product data incomplete (score: X%). Fields missing: [list]."
- Agent MUST NOT publish, list, or sync this product to any platform.
- Agent MAY generate a "data repair ticket" for human completion.

### Cost Completeness

| Field | Required | Weight | Notes |
|-------|----------|--------|-------|
| Procurement cost | Yes | 40% | Per-unit landed cost from supplier |
| Logistics cost (per unit) | Yes | 30% | Estimated or actual shipping cost to warehouse |
| Platform fee rate | Yes | 20% | Commission + fixed fee per platform |
| Exchange rate | Yes | 10% | Current rate for cross-currency calculations |

**Threshold: 70%** (procurement cost + platform fee rate achieve 60% alone; at least one logistics cost element needed to reach 70%).

**Measurement method:** Scan the cost record linked to the SKU/candidate product. Sum weighted scores of populated fields.

**Action when below threshold:**
- Agent MUST NOT produce a profit estimate or profit-related recommendation.
- Agent MAY produce a sourcing optimization recommendation focusing only on procurement cost if that field is complete.
- Any output MUST include: "Cost data incomplete (score: X%). Profit estimates unavailable."

### Logistics Fee Completeness

| Field | Required | Weight | Notes |
|-------|----------|--------|-------|
| Weight | Yes | 30% | Actual or estimated package weight |
| Dimensions (LxWxH) | Yes | 30% | Dimensional weight calculation |
| Origin | Yes | 20% | Warehouse / fulfillment center |
| Destination | Yes | 20% | Target market / customer region |

**Threshold: 80%** (origin + destination give 40%; need at least one of weight or dimensions to cross threshold).

**Measurement method:** Scan the logistics record for the shipment or SKU template.

**Action when below threshold:**
- Agent MUST NOT compute shipping cost or include shipping cost in any recommendation.
- Agent MAY use a proxy estimate labeled as such (trust_level = STUB with explicit note).
- Any output MUST include: "Logistics data incomplete (score: X%). Shipping cost estimate unavailable."

### Order Settlement Completeness

| Field | Required | Weight | Notes |
|-------|----------|--------|-------|
| Order amount | Yes | 25% | Total order value in settlement currency |
| Fee breakdown | Yes | 25% | Platform fees, transaction fees, taxes itemized |
| Actual shipping cost | Yes | 20% | What the logistics provider actually charged |
| Actual procurement cost | Yes | 20% | What was actually paid to supplier |
| Net profit / loss | Yes | 10% | Calculated or provided |

**Threshold: 80%** (order amount + fee breakdown achieve 50%; need actual shipping or actual procurement to cross).

**Measurement method:** Scan the settlement record after order fulfillment.

**Action when below threshold:**
- Agent MUST NOT mark the settlement as "complete" or "reconciled."
- Platform MUST flag the settlement for manual review in the dashboard.
- Agent MAY generate a "settlement gap" notification for the Owner.

## 2. Agent Output Rules

These rules override any domain-specific behavior when data quality constraints are active:

1. **Agent MUST NOT state a strong conclusion when the supporting data is incomplete.** A strong conclusion is one that would lead an Owner or downstream system to act without further investigation. Examples: "This product is profitable," "Ship to this warehouse," "This supplier is cheaper."

2. **Agent MUST qualify every conclusion that depends on incomplete data with:**
   - The completeness score for the relevant domain.
   - Which fields are missing.
   - The impact on confidence: "This recommendation is based on X% complete data; missing [fields] means the actual cost may be higher/lower."

3. **Agent MUST declare data quality in its output metadata.** Every agent output SHOULD contain a `data_quality` section:

   ```json
   {
     "data_quality": {
       "product_completeness": { "score": 85, "threshold": 80, "passed": true },
       "cost_completeness": { "score": 60, "threshold": 70, "passed": false, "missing": ["logistics_cost", "exchange_rate"] }
     },
     "trust_level": "REAL_LLM",
     "recommendation_confidence": "LOW",
     "qualifier": "Cost data incomplete. Profit estimates are not available."
   }
   ```

4. **Missing data is not a bug.** Agents MUST NOT silently impute missing fields and proceed as if data is complete. Imputed values MUST be labeled as estimates (trust_level = DETERMINISTIC_RULE with `is_estimate=true`).

## 3. Data Quality Dashboard Requirements

The platform MUST surface data quality in a dashboard view accessible to the Owner. Minimum requirements:

1. **Per-domain completeness cards** showing current score / threshold / trend (last 7 days) for each of the four domains.

2. **Entity-level drill-down** listing specific products, costs, logistics items, or settlements that are below threshold, sorted by impact (highest-priority first).

3. **Trend line** of completeness scores over time (daily rollup) — allows the Owner to see if data quality is improving or degrading.

4. **Agent impact summary**: "In the last 24 hours, X agent recommendations were marked LOW confidence due to data incompleteness. Fixing [top 3 missing fields] would raise X recommendations to MEDIUM or HIGH confidence."

5. **Alert on regression**: if any completeness score drops by more than 10 points in a 24-hour period, the platform MUST generate a notification (via the notification module) with the change and likely cause.

## 4. Enforcement

| Layer | Enforcement |
|-------|------------|
| Agent Service | MUST check completeness before generating strong recommendations. Test coverage required for each check. |
| EventBus | Post-processing subscriber MAY validate completeness on output and log violation. |
| Dashboard | MUST surface all below-threshold entities. MUST NOT show profit/confidence metrics that rely on incomplete data. |
| CI/CD | New domain logic that adds data-quality-sensitive output MUST include completeness gate test. CI rejects if coverage drops below the existing baseline. |
