# Spec: Short-Term Product Positioning — AI Copilot

## Objective

Converge LingMirror's short-term positioning from "AI autonomous operations" to "AI-assisted business decisions". This is a positioning and messaging change — the architecture already supports approval-based execution. We need to make the positioning explicit and visible across docs and UI copy.

### Acceptance Criteria

1. Product direction explicitly documented as **短期 Copilot，长期 Autopilot**
2. High-risk actions explicitly listed with default-approval policy: pricing, inventory, procurement, advertising, publishing
3. Owner dashboard and Agent page copy/status do not imply "fully automatic managed" (全自动托管)

## Tech Stack

Not applicable — this is a positioning and documentation change, not a code feature.

## Commands

No new commands. For verification:

```bash
# Verify docs built
grep -rn "Copilot\|Autopilot\|辅助经营\|全自动托管" docs/ --include="*.md"

# Verify frontend copy audit
grep -rn "自动运营\|自主运营\|autonomous\|Autopilot" frontend-next/src/app/ --include="*.tsx" --include="*.ts"
grep -rn "自动运营\|自主运营\|autonomous" backend-go/internal/ --include="*.go" | grep -v vendor | grep -v _test
```

## Project Structure

```
docs/
├── CURRENT_DIRECTION_AND_PRIORITIES.md  ← update positioning language
├── PRODUCT_VISION_AND_MVP.md            ← add Copilot/Autopilot framing
└── POSITIONING_2026-07-06.md            ← NEW: standalone positioning spec (this file)

frontend-next/src/app/(main)/
├── agents/page.tsx     ← audit squad/autonomy column labels for misleading auto-implication
├── agentos/page.tsx    ← audit AgentOS dashboard copy
├── owners/page.tsx     ← audit Owner dashboard copy
└── ai/page.tsx         ← audit AI command center copy
```

## Code Style — Positioning Statements

When adding positioning content to docs, use this consistent pattern:

```markdown
### Short-Term Positioning (current → Q4 2026)

**LingMirror is an AI-assisted business decision platform.**

- The system recommends; the Owner decides.
- Every actionable output includes what, why, risk level, and recommendation.
- High-risk actions (pricing, inventory, procurement, advertising, publishing) require explicit approval.
- The Owner always stays in control.
```

```markdown
### Long-Term Direction (2027+)

**LingMirror evolves toward AI-autonomous operations.**

- Agents may execute approved, bounded actions under supervision.
- Automatic execution applies only to low-risk, reversible actions.
- The Owner sets strategy, monitors exceptions, and audits outcomes.
```

## Testing Strategy

No code tests needed. Verification checklist:

- [ ] `docs/CURRENT_DIRECTION_AND_PRIORITIES.md` updated with Copilot/Autopilot language
- [ ] `docs/PRODUCT_VISION_AND_MVP.md` updated or cross-referenced
- [ ] `frontend-next/src/app/(main)/agents/page.tsx` — columns/squad labels no longer imply "autonomous = fully automatic"
- [ ] Owner dashboard copy reviewed for "全自动托管" implications
- [ ] No frontend page uses "自动运营" or "完全托管" as unqualified status text

## Boundaries

Always:
- Use "AI 辅助经营决策" (AI-assisted business decisions) for short-term positioning
- Use "Copilot" for the current product promise
- Use "Autopilot" only for the 2027+ direction
- Keep the `autonomous` squad/level suffix as an *internal architecture* label, not a *product promise* label

Ask first:
- Removing or renaming the `autonomous` autonomy level — it's used in Go backend code and action catalog
- Changing the `Squad: "autonomous"` field in agent registry — affects multiple consumers

Never:
- Display "全自动托管" or "完全自主运营" as a UI feature description
- Market the product as "fully automatic" in any customer-facing doc
- Remove the approval gate on high-risk actions

## Success Criteria

| # | Criterion | How to verify |
|---|-----------|--------------|
| 1 | `CURRENT_DIRECTION_AND_PRIORITIES.md` contains explicit "短期 Copilot，长期 Autopilot" section | Read the file |
| 2 | `PRODUCT_VISION_AND_MVP.md` references the Copilot/Autopilot direction | Read the file |
| 3 | High-risk action list (pricing/inventory/procurement/advertising/publishing) appears in a positioning doc with default-approval policy | Read the file |
| 4 | No frontend page copy describes agents as "全自动" or "自主运营" without caveat | grep across frontend |
| 5 | The Owner dashboard and Agent sidebar do not display "autonomous" in a way that reads as "fully automatic" to a non-technical user | Visual review |

## Open Questions

None — the user gave clear acceptance criteria. Proceeding.
