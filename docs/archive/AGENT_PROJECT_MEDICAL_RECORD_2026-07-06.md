# Agent Project Medical Record Archive — 2026-07-06

This document preserves the `Project Medical Record` section that previously lived in `AGENTS.md`.

It is a historical snapshot, not current product direction, not current technical priority, and not an instruction to start with any listed item. For current guidance, use:

- `docs/governance/`
- `docs/CURRENT_DIRECTION_AND_PRIORITIES.md`
- `docs/PROJECT_STATUS.md`
- `docs/reference-module-catalog.md`

## Original Snapshot

> Last updated: 2026-07-06. Read this before any work. It prevents repeating mistakes.
> For the latest verification status, run: `cd backend-go && go test ./...`

### What Works (verified this session)

- `go build ./...` — passes
- `go vet ./...` — passes
- `go test ./...` — 96 packages green, 11 pkgs no-test (107 total), 0 failures
- Frontend: `npm run dev` — starts on port 3001 (but dev server can exit unexpectedly)
- Login: admin / admin123456 (user table seeded, RBAC roles linked)
- All 30+ frontend pages render (product hub, categories, brands, SKU, inventory, orders, agents, AI command center, etc.)
- Seed data in DB: 5 categories, 3 brands, 2 platforms (Ozon + Shopee), product + SKU + inventory

### Known Issues (unfixed)

| Priority | Issue | Location |
|----------|-------|----------|
| P0 | Agent output is stub (fake data, not real LLM) | `orchestrator.go:172` — `synthesizeOutput()` |
| P1 | MoA aggregation is structured but still deterministic, not LLM-synthesized | `moa.go` — `synthesize()` returns structured findings/conflicts/recommendation |
| P1 | Owner dashboard /owner is Mock | `frontend-next/src/app/(main)/owner/` |
| P2 | Only 3 platform adapters (Ozon + Shopee + Shopify), still thinly tested | `domain/integrations/` |
| P2 | Frontend dev server has no watchdog — exits silently | `npm run dev` process |
| P3 | No real CI trigger yet (doc-links job added but not tested) | `.github/workflows/ci.yml` |

### What Was Fixed (2026-07-06)

- 工程可信度恢复: `go build ./...` / `go vet ./...` / `go test ./...` 全绿
  - 修复 `internal/common/types.go` 中 `UserIDFromCtx` 重复定义（删除第二个副本）
  - 清理 `internal/ai/` 6 个文件共 19 处 merge conflict（handler.go, service.go, orchestrator.go, routes.go, model.go, ai_test.go）
  - 冲突来自 merge commit `964e0624`（合并远程 main v0.4.0），保留 HEAD 版本
- Merge conflicts in `routes.go` + `router.go` (HEAD won over worktree-wf)
- Duplicate `UserIDFromCtx` in `types.go` (kept first, deleted second)
- AuthGuard SSR crash (`useState` reading localStorage during server render → `useEffect` + `mounted`)
- RBAC endpoint 404 (frontend called `/v1/rbac/current/permissions` but route was unregistered)
- Inventory + product-hub 403 (operator users not linked to `ops` RBAC role → migration `000064`)
- Supplier test failure (handler read `c.Query()` but route used path param → fixed to `c.Param()`)
- Owner test failure (test CREATE TABLE missing `requester_user_id` + `reviewer_user_id` columns)
- Doc dead links removed from CLAUDE.md, INDEX.md, KERNEL_CONTRACTS.md

### Project Rules (Do Not Violate)

1. Doc sync is mandatory. Changing module names, API paths, or package layout requires updating AGENTS.md, CLAUDE.md, and docs/INDEX.md. CI `doc-links` job rejects stale references.
2. Do not touch `.kilo/worktrees/` — managed by external tooling.
3. Do not rewrite history. No `git rebase` on shared branches (main, feat/*, codex/*).
4. Test before commit. Minimum: `go build ./...` + `go vet ./...` + `go test ./...` for touched packages.
5. No unrequested refactors. Match existing patterns. Drive-by style changes are rejected.
6. Old-stack docs (superpowers/plans/ etc.) are marked deprecated — do not treat as executable instructions.
7. Keep frontend API path format consistent: `/api/v1/*` prefix, apiClient with `/v1/*` paths.

### Cron Jobs

| Name | Schedule | What it does |
|------|----------|-------------|
| 文档链接审计 | Mon 9:00 | Checks AGENTS.md/CLAUDE.md/INDEX.md for dead links |
| 依赖安全检查 | Mon 10:00 | go mod verify + npm audit |
| 每周健康检查 | Mon 9:00 | Full test suite + git status + service check |

- `docs/features/` — feature specs; use `TEMPLATE.md`.
