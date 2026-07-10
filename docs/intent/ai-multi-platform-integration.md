# Intent: AI-Powered Multi-Platform Integration Layer

**Date:** 2026-07-09
**Status:** Draft intent, pending spec

## The Problem

LingMirror has Ozon working but all other platforms (Amazon, Shopify, Shopee, Lazada) are stubs. Profit data is mock/estimated. The user said the system "can't be used" — zero real data, no write-back, UX unfinished.

## The Insight

Traditional approach (hand-write 5× full adapters) doesn't scale for a single-developer team. Each platform has unique API shapes, fee structures, category trees, and event formats. The maintenance burden grows linearly with platforms.

## The Solution

Replace hand-written field-mapping code with an AI-powered transformation layer:

```
Layer 1: Platform communication — HTTP + auth + rate limiting (hand-written, ~150 lines each)
Layer 2: AI transformation       — LLM maps fields, recognizes events, maps categories, parses settlements
Layer 3: Profit truth engine     — Real settlement data → per-SKU profit
```

**Key shift:** From "write 600 lines of Go per platform" to "write 150 lines of Go + 50 lines of prompt per platform."

## Constraints

- One-person team; must pick one platform to break first
- AI cost must be controlled (cache + batch + deterministic fallback)
- Ozon must stay working during and after the transition
- Every AI-mapped value must pass schema validation (prevent hallucinations on money amounts)
- 80%+ of mapping is deterministic (simple field extraction); AI only for the complex 20%

## Out of Scope (Phase 1)

- Prism 商品图引擎
- Deep fulfillment intelligence
- Full write-back (publishing) to non-Ozon platforms
- AI 客服 / Support Mate
- Frontend UX polish beyond data display

## Next Step

Write a formal spec, then plan, then implement.
