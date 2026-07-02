# First Kilometer P1: 选品调研 Agent + 供应商发现 Agent

Date: 2026-07-02
Author: AI Agent
Status: Implemented
Risk Level: Medium (business logic, new decision points, no money/price/inventory touch)

## 1. Goal

Enable the user to say "我想调研家居类目，找几个适合上架的产品" and get back:
1. Recommended sub-categories / keywords and WHY
2. Risks and uncertainties
3. 1688 search page URLs to collect from
4. Clear warnings that these are hypotheses, not business conclusions

## 2. Architecture

### Design Decision

Reuse **A1 ProductScoutAgent**, adding two new decision points:
- `product_research` — research hypothesis generator
- `supplier_discovery` — 1688 collection plan generator

Route table updated so chat keywords route to these decision points.

### Why A1 ProductScoutAgent?

A1 already owns product-related decisions (`product_scout`, `market_analysis`). The new decision points are tightly coupled to category/market research, which is A1's domain. Adding them to A1 avoids creating a duplicate agent.

### Flow

```
User: "我想调研家居类目"
  → POST /api/v1/ai/chat {"message": "我想调研家居类目"}
  → routeChat() matches "调研" → A1, "product_research"
  → ProductScoutAgent.Decide("product_research", {category, target_market, target_platform})
  → returns research directions with confidence, risks, data_needed

User: "给我看供应商页面"
  → POST /api/v1/ai/chat {"message": "给我看 1688 供应商页面"}
  → routeChat() matches "供应商" → A1, "supplier_discovery"
  → ProductScoutAgent.Decide("supplier_discovery", {keywords or directions})
  → returns 1688 search URLs + collection instructions
```

### Risk & Safety

- Both decision points are `RiskLow` — read-only analysis, no mutations
- Both outputs include mandatory `warnings` stating these are hypotheses
- No CandidateProduct created, no prices changed, no inventory modified
- All outputs have explicit `confidence` scores (≤ 0.65 for research, ≤ 0.60 for discovery)

## 3. Data Structures

### product_research output

```json
{
  "status": "research_ready" | "insufficient_data",
  "category": "家居",
  "target_market": "RU",
  "target_platform": "Ozon",
  "recommended_directions": [
    {
      "name": "厨房收纳小件",
      "why": "...",
      "target_price_band": "$5-$20",
      "risk_notes": ["..."],
      "keywords": ["kitchen storage", ...],
      "data_needed": ["1688 采购价", ...],
      "confidence": 0.65
    }
  ],
  "data_needed": ["1688 采购价", "重量", ...],
  "warnings": ["这是调研假设，不是确定经营结论"]
}
```

### supplier_discovery output

```json
{
  "status": "collection_plan_ready" | "needs_keywords",
  "source_platform": "1688",
  "search_keywords": ["厨房收纳", "免打孔置物架"],
  "suggested_pages": [
    {
      "type": "search",
      "url": "https://s.1688.com/...",
      "reason": "搜索「厨房收纳」候选商品"
    }
  ],
  "supplier_filter_rules": ["优先看有成交记录的店铺", ...],
  "collection_instructions": ["由用户手动打开页面", ...],
  "warnings": ["这是调研假设，不是确定经营结论"]
}
```

## 4. Implementation Notes

- `productResearch` uses rule-based category→direction heuristics (static map)
- `supplierDiscovery` generates 1688 search URLs with URL encoding
- Direction map covers `家居` and `宠物用品` with sub-directions; all other categories get a generic fallback
- `ponytail:` comments mark deliberate simplifications (static maps, naive URL encoding)

## 5. Route Table Changes

Added two entries:
- "调研" keywords → A1, `product_research`
- "供应商/1688" keywords → A1, `supplier_discovery`

Existing "选品" keyword still routes to `product_scout`. The new entries are additive.

## 6. Acceptance Criteria

1. ✅ Input "我想调研家居类目" generates research directions
2. ✅ supplier_discovery generates 1688 search URLs from keywords
3. ✅ Outputs include warnings and confidence scores
4. ✅ No price, inventory, order, or listing mutations
5. ✅ Cannot create CandidateProduct or other database records
6. ✅ All tests pass (except pre-existing page_data test)
