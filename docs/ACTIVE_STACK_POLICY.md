# LingMirror Active Stack Policy

> Effective date: 2026-06-23

## Decision

LingMirror development is now focused on the new stack:

- Backend: `backend-go/` (Go / Gin / GORM / PostgreSQL)
- Frontend: `frontend-next/` (Next.js / React / TypeScript / Ant Design)
- Default compose entry: `docker-compose.yml`

The old stack is paused:

- Legacy backend: `backend/` (Python / FastAPI)
- Legacy frontend: `frontend/` (Vue 3)
- Legacy compose entry: `docker-compose.legacy.yml`

## Development Rules

1. New product work must be implemented in `backend-go/` and `frontend-next/`.
2. Do not add features to `backend/` or `frontend/`.
3. Do not fix legacy bugs unless they block data migration, parity analysis, or emergency rollback.
4. If old behavior is needed, read legacy code as a reference and port the behavior into the new stack.
5. New API contracts should use `/api/v1/*`.
6. The old `/api/*` FastAPI contract is reference-only unless explicitly needed for compatibility.
7. Production cutover still requires green migration, E2E, parity, and rollback rehearsal gates.

## Allowed Legacy Changes

Legacy directories may be changed only for:

- data export or migration scripts;
- parity reports and compatibility notes;
- security fixes required before rollback use;
- documentation marking behavior that must be ported.

Any other legacy change should be rejected during review.

## Default Commands

```bash
# New stack
docker compose up -d

# Legacy stack, only for rollback/reference
docker compose -f docker-compose.legacy.yml up -d
```
