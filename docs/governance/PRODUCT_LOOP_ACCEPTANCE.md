# Product Loop Acceptance

**Version:** 1.0
**Last updated:** 2026-07-06
**Owner:** QA Architect
**Governance reference:** PLATFORM_CONSTITUTION.md, KERNEL_CONTRACTS.md, OWNER_FIRST_PROTOCOL.md

---

## 1. Purpose

Verify that the complete product lifecycle end-to-end operates correctly across all domain modules, Agent decision points, approval gates, and audit trails. The acceptance covers the flow from raw candidate product discovery through completeness validation, cost-profit calculation, listing recommendation, Owner approval, listing task execution, and result review.

## 2. Flow Overview

```
candidate product → completeness check → cost/logistics/platform fee/profit calculation
→ listing recommendation → Owner approval → listing task creation → result review
```

### Modules Involved

| Step | Module(s) | AI Agent(s) |
|------|-----------|-------------|
| Candidate sourcing | `domain/candidate/` | A1 (Discovery), A8 (Sourcing) |
| Completeness validation | `domain/completeness/` | A7 (Review) |
| Cost/Profit calculation | `domain/cost/`, `domain/landedcost/`, `domain/logistics/`, `domain/platformfee/`, `domain/exchangerate/`, `domain/finance/`, `domain/decision/` | A2 (Listing Optimize), A9 (Pricing) |
| Listing recommendation | `domain/decision/`, `domain/approval/` | A2, G1 (Dashboard Overview) |
| Owner approval | `domain/approval/`, `domain/actionpolicy/` | N/A (Owner UI) |
| Listing task creation | `domain/listingtask/` | Command dispatch |
| Result review | `domain/listing/`, `domain/integrations/` | G1, A7 |

## 3. Step-by-Step Acceptance

### 3.1 Candidate Sourcing

| Field | Detail |
|-------|--------|
| Test data needed | 3 candidate products from different sources: manual import (CSV), platform sync (Shopee/Ozon mock), Agent-sourced (A1 research output). Each with incomplete fields intentionally missing to test downstream completeness check. |
| Operation steps | 1. Trigger A1 candidate discovery for a target niche (e.g., "wireless earbuds, Thailand market"). 2. Manually create a candidate via POST `/api/v1/candidate`. 3. Import a CSV with 2 rows via the batch import endpoint. |
| API/UI/DB verification points | API: `POST /api/v1/candidate` returns 201 with candidate ID. GET list returns all candidates with correct status. DB: `candidate_product` table contains rows with status=`pending` or `draft`. Agent trace shows source attribution. |
| Expected results | All 3 candidates appear in the candidate workbench with status `pending`. Agent A1 trace shows confidence score and research summary. CSV import reports 2 rows imported, 0 failures. |
| Audit evidence | `operation_log` contains `create` entry for each candidate with actor and source info. |
| Failure impact | Candidates not appearing in workbench blocks the entire pipeline. Missing source attribution makes downstream decisions untrustable. |

**PASS criteria:** All candidates visible in workbench. Agent trace links to each candidate. CSV import reports exact row count.

---

### 3.2 Completeness Check

| Field | Detail |
|-------|--------|
| Test data needed | 3 candidates from step 3.1. Define completeness rules: title (required), description (required), category (required), images (required for platform publish, optional for internal), dimensions (required for logistics), HS code (required for customs). |
| Operation steps | 1. A7 Agent runs completeness check on all pending candidates. 2. Check that a candidate missing required fields gets status `incomplete` with a `missing_requirements` JSON listing each gap. 3. Owner fills gaps via UI (title, description, category). 4. Agent re-checks completeness after fill. |
| API/UI/DB verification points | API: `GET /api/v1/completeness/{candidate_id}` returns completeness report with scoring. `PUT /api/v1/candidate/{id}` updates fields. DB: `completeness_score` table (or equivalent) has per-candidate scores. Agent trace shows completeness evaluation. |
| Expected results | Complete candidate → status `ready_for_cost_analysis`. Incomplete candidate → status `incomplete` with actionable requirements list. After Owner fills gaps and re-checks → status advances to `ready_for_cost_analysis`. |
| Audit evidence | `operation_log` records the completeness check trigger, each field fill by Owner, and the re-check. |
| Failure impact | Missing completeness gate allows bad products into cost calculation, wasting compute and generating misleading recommendations. Overly strict gates block valid products. |

**PASS criteria:** Completeness engine correctly identifies missing fields. Missing fields are actionable (human-readable). Re-check after fill succeeds. Edge case: zero required fields → passes.

---

### 3.3 Cost / Logistics / Platform Fee / Profit Calculation

| Field | Detail |
|-------|--------|
| Test data needed | 1 fully complete candidate with known dimensions/weight (e.g., 30x20x10cm, 500g), known product cost ($8.00), target platform (Shopee Thailand), payment gateway (credit card). 1 candidate with missing dimensions to test logistics fallback. 1 candidate with zero-margin scenario. |
| Operation steps | 1. Trigger A9 pricing analysis on `ready_for_cost_analysis` candidates. 2. Verify platform fee engine: commission rate (Shopee Thailand = 4.28% + $0.30), transaction fee (2%), service fee (1%). 3. Verify logistics rate engine: quote from 3 carriers, select cheapest. 4. Verify landed cost: product cost + shipping + customs + insurance. 5. Verify exchange rate conversion (CNY → THB). 6. Verify profit calculation: revenue - product_cost - shipping - platform_fee - payment_fee - other_fee. |
| API/UI/DB verification points | API: `GET /api/v1/decision/{sku_id}` returns estimated costs and profit. `GET /api/v1/logistics/quote` returns rates. DB: `pre_listing_decision` row has non-zero estimated_revenue, estimated_product_cost, estimated_shipping_cost, estimated_platform_fee, estimated_payment_fee, estimated_profit, profit_margin. |
| Expected results | Full candidate: profit margin ~15-25% (depends on pricing strategy). Missing-dimensions candidate: logistics quote shows "dimensions required" error, profit calculation blocked. Zero-margin candidate: recommendation = "not_recommended", reasoning explains unprofitability. |
| Audit evidence | Decision trace in `pre_listing_decision` with all cost breakdown fields populated. Agent trace shows each calculation step. |
| Failure impact | Wrong cost estimates lead to bad pricing decisions. Unprofitable products listed at a loss without detection. Over-estimated profits cause inventory cashflow issues. |

**PASS criteria:** All 7 cost components populated. Profit margin calculation matches manual verification. Missing required data correctly blocks calculation. Zero-margin product correctly flagged. Exchange rate conversion uses latest rate.

**Test Matrix: Cost Calculation Scenarios**

| Scenario | Product Cost | Shipping | Platform Fee | Revenue | Expected Profit | Expected Recommendation |
|----------|-------------|----------|--------------|---------|-----------------|----------------------|
| Profitable | $8.00 | $3.50 | $2.10 | $18.00 | $4.40 (24.4%) | list |
| Low margin | $12.00 | $4.00 | $2.50 | $20.00 | $1.50 (7.5%) | review (borderline) |
| Negative | $15.00 | $5.00 | $3.00 | $20.00 | -$3.00 (-15%) | not_recommended |
| Zero revenue | $5.00 | $2.00 | $0 | $0.00 | -$7.00 | not_recommended |
| Missing data | $8.00 | N/A | N/A | $18.00 | N/A | incomplete_data |

---

### 3.4 Listing Recommendation

| Field | Detail |
|-------|--------|
| Test data needed | Minimum 5 decision records from step 3.3 with various recommendations (list, review, not_recommended, incomplete_data). |
| Operation steps | 1. Agent A2 reads decision records with recommendation=recommended. 2. Agent generates listing recommendation with target sale price, target profit margin, destination country, and platform selection. 3. Recommendation includes confidence score and reasoning. 4. Owner views decision recommendations in dashboard. |
| API/UI/DB verification points | API: `GET /api/v1/decision?status=pending` returns all pending decisions. `GET /api/v1/decision/summary` returns aggregated statistics. DB: `pre_listing_decision` rows have recommendation populated and status=`pending`. |
| Expected results | Recommended products show actionable proposal: "List on Shopee Thailand at $18.00 with 24.4% margin". Agent includes confidence score and key assumptions. Not-recommended products show clear reason. |
| Audit evidence | Agent trace shows recommendation generation with reasoning. Decision records linked to underlying cost breakdown. |
| Failure impact | Misleading recommendations cause wrong business decisions. Missing confidence scores reduce trustworthiness. |

**PASS criteria:** Each decision has recommendation, reasoning, and confidence score. Dashboard aggregates show correct counts by recommendation type. Recommended products have actionable data (price, platform, margin).

---

### 3.5 Owner Approval

| Field | Detail |
|-------|--------|
| Test data needed | 1 recommended product (approve expected). 1 borderline product with `review` status (reject expected). 1 product with `not_recommended` status (verify Owner cannot force-publish without override). |
| Operation steps | 1. Owner opens decision workbench in UI. 2. Owner reviews recommended product: sees cost breakdown, profit projection, and Agent recommendation. 3. Owner clicks "Approve" on recommended product. 4. System creates approval request with status=pending, then Owner (or designated reviewer) approves it. 5. Owner reviews borderline product, clicks "Reject" with reason. 6. Owner attempts to approve `not_recommended` product. |
| API/UI/DB verification points | API: `POST /api/v1/decision/{id}/approve` returns 200. `POST /api/v1/approval` creates request. `PUT /api/v1/approval/{id}/review` with action=approve transitions status. DB: `approval_request` row with status=`approved`, old/new values, reviewer. `pre_listing_decision.status` transitions to `approved` or `rejected`. |
| Expected results | Approved product: approval_request status `approved`, decision status `approved`, listing task auto-created (see step 3.6). Rejected product: status `rejected`, decision retains reasoning. Not-recommended override attempt: system prompts with risk warning confirmation or blocks with policy. |
| Audit evidence | `operation_log` records approve/reject action with Owner identity, timestamp, and reason. Approval request has all required fields. |
| Failure impact | Approval without audit makes unauthorized listings possible. Rejected product still appearing in listing queue. Not-recommended products publishable without warning. |

**PASS criteria:** Approval creates audit trail. Rejection updates decision status. Override blocked or requires explicit risk acceptance. Approval expiry (if set) enforced.

---

### 3.6 Listing Task Creation

| Field | Detail |
|-------|--------|
| Test data needed | Approved decision from step 3.5. |
| Operation steps | 1. After approval, verify system creates a `listing_task` record automatically (via EventBus subscriber or command dispatch). 2. Task is created with status=`blocked` (pre-flight checks). 3. Pre-flight checks run: platform credentials valid, product listing not duplicate, required images exist. 4. If checks pass, task transitions to `pending`. 5. Task executor runs: calls platform adapter (dry-run first, then sandbox, then production based on mode). |
| API/UI/DB verification points | API: `GET /api/v1/listingtask` returns the created task. `GET /api/v1/listingtask/{id}` returns status and decision snapshot. DB: `listing_task` table has row with correct product_id, platform_id, target_sale_price, target_profit_margin, destination_country, approval_id. `listing_task_item` created for each platform. |
| Expected results | Task created within seconds of approval. Status transitions: blocked → pending → executing → completed. Task includes full decision snapshot for traceability. |
| Audit evidence | EventBus event `listing.task.created` published with task payload. `operation_log` records task creation. Listing task has approval_id linking back to approval_request. |
| Failure impact | Task not created blocks product from market. Task created with wrong price causes financial loss. Missing decision snapshot loses audit trail for what was approved. |

**PASS criteria:** Task auto-created on approval. Snapshot preserved. Status transitions correct. Pre-flight checks run before execution.

---

### 3.7 Result Review

| Field | Detail |
|-------|--------|
| Test data needed | Completed listing task from step 3.6 (success). 1 failed listing task (platform rejection). 1 partially completed task (multi-platform, one succeeds one fails). |
| Operation steps | 1. View listing task that completed successfully in dashboard. 2. View listing task that failed: see error message and retry count. 3. For partial multi-platform task: see per-item status breakdown. 4. Attempt retry on failed task. 5. Review product listing on target platform via integration status. |
| API/UI/DB verification points | API: `GET /api/v1/listingtask?status=completed` returns success results. `GET /api/v1/listing/{product_id}` returns platform listing status. DB: `listing_task_item` rows show per-platform status and result JSON. |
| Expected results | Completed task shows link to live product listing on platform. Failed task shows actionable error. Retry increases retry_count and re-executes. |
| Audit evidence | Listing task item stores execution result JSON. Platform adapter call logged with external platform response. |
| Failure impact | Failed tasks silent → Owner unaware product not live. Success but wrong result not detected → financial loss. |

**PASS criteria:** Status visible for all task items. Error messages actionable. Retry mechanism works and is bounded (max retries enforced). Multi-platform tasks report per-platform status.

---

## 4. End-to-End Test Matrix

| Test Case | Candidate | Completeness | Cost Calc | Recommendation | Approval | Task | Expected Final Status |
|-----------|-----------|-------------|-----------|----------------|----------|------|---------------------|
| Happy path: full product | All fields complete | Complete (score >= 80%) | Profitable | list | Approved | Created, Completed | product live on platform |
| Missing images | No images | Incomplete (score < 80%) | Blocked | N/A | N/A | Not created | candidate incomplete |
| Unprofitable product | All fields complete | Complete | Negative profit | not_recommended | Override | Owner forced through | override warning shown |
| Platform credential failure | All fields complete | Complete | Profitable | list | Approved | Pre-flight fail | task blocked |
| Multi-platform listing | All fields complete | Complete | Profitable (x2) | list (x2) | Approved | 2 items | 1 success, 1 fail |
| Owner reject | All fields complete | Complete | Profitable | list | Rejected | Not created | decision rejected |
| Retry after failure | All fields complete | Complete | Profitable | list | Approved | Fail → Retry → Success | completed |

---

## 5. Failure Impact Analysis

| Failure Point | Detection | Impact | Mitigation |
|--------------|-----------|--------|------------|
| Candidate import broken | No candidates in workbench | Pipeline halts at step 1 | Manual CSV import via SQL as fallback |
| Completeness check broken | All candidates incomplete or all pass | Over-filtering or under-filtering | Disable gate temporarily, manual review |
| Cost engine wrong values | Profit margins unrealistic | Wrong pricing decisions | Manual price audit before publish |
| Approval system down | Cannot approve/reject decisions | Pipeline stalls at step 5 | Owner can set `auto_approve` policy temporarily |
| Platform adapter fails | Listing task stays blocked | Product not published | Retry with timeout, notify Owner |
| Result not visible | Task completed but no feedback | Owner unaware of live products | Dashboard polling with status indicator |

---

## 6. Acceptance Gate Sign-off

| Section | QA Verdict (PASS/FAIL) | Notes |
|---------|------------------------|-------|
| Candidate Sourcing | | |
| Completeness Check | | |
| Cost/Profit Calculation | | |
| Listing Recommendation | | |
| Owner Approval | | |
| Listing Task Creation | | |
| Result Review | | |
| **Overall** | | |

Date: ______________  Sign-off: ______________

---

## 7. References

- PLATFORM_CONSTITUTION.md — risk levels, approval rules, audit requirements
- KERNEL_CONTRACTS.md — EventBus guarantees, Command Dispatch, Approval Contract, Audit Contract
- OWNER_FIRST_PROTOCOL.md — Owner risk acceptance, delivery report format
- ACCEPTANCE_MATRIX.md — cross-module acceptance traceability
- KNOWN_ISSUES.md — known defects affecting this loop
- AGENT_TRUST_MARKERS.md — trust scoring for Agent recommendations
- `backend-go/internal/domain/candidate/model.go` — candidate_product schema
- `backend-go/internal/domain/completeness/` — completeness scoring engine
- `backend-go/internal/domain/decision/model.go` — pre_listing_decision schema
- `backend-go/internal/domain/listingtask/model.go` — listing_task schema
- `backend-go/internal/domain/approval/model.go` — approval_request schema
- `backend-go/internal/domain/actionpolicy/model.go` — policy evaluation for override rules
- `backend-go/internal/domain/integrations/` — PlatformAdapter interface
- `docs/statemachine.md` — state machine validation framework
