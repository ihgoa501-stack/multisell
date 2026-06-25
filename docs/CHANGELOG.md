# Changelog

## [0.3.0] - 2026-06-25

### Added
- Frontend: purchase and support pages (4 files, 309 lines)
- Frontend: SSE streaming and WebSocket realtime client for AI chat
- AIOS: batch work including finance, WebSocket, inventory, purchase, support modules
- Design: LingMirror design system tokens, theme, and layout applied project-wide
- Design: Agent Hub sidebar and ToolPanel with three-panel interactive layout
- Design: agent workspace dashboard with chat input
- Design: full three-panel interactive implementation — AI chat, products table, ToolPanel
- Test: 158 focused test functions across 6 domain modules
- Docs: design consultation session summary for cross-agent reference

### Fixed
- API: added missing /v1 prefix to 5 frontend API paths
- EventBus: replaced inline goroutine spawning with priority queue for backpressure control
- Design: hardcoded colors replaced with design system tokens in DetailDrawer and EmptyState

### Refactored
- Design: login page restyled — removed purple/blur AI slop, aligned with new tokens
- Design: restyled all remaining pages (settings, agents, agentos, actions, reports, search, products, PageContainer, CrudListPage)
- Code: split logistics_ops.go (1217 lines) and aftersales_mgmt.go (1058 lines) into smaller files

### Changed
- Dependencies: updated package-lock.json after dependency install
- Infra: merged design consultation worktree branch

### Documentation
- PROJECT_STATUS.md: updated with 2026-06-25 parallel fix results
- PROJECT_STATUS.md: corrected lint status (1 error remains)
