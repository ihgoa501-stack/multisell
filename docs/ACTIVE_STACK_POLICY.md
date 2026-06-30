# LingMirror Active Stack Policy

> Effective date: 2026-06-23
> Last reviewed: 2026-06-30
> Change: Legacy stack deleted per Owner direction

## Decision

LingMirror has completed the full-site migration to the new stack. Active development, verification, and documentation should use:

- Backend: `backend-go/` (Go / Gin / GORM / PostgreSQL)
- Frontend: `frontend-next/` (Next.js / React / TypeScript / Ant Design)
- Default compose entry: `docker-compose.yml`

The old stack has been deleted (2026-06-30):

- ~~Legacy backend: `backend/` (Python / FastAPI)~~ **Deleted**
- ~~Legacy frontend: `frontend/` (Vue 3)~~ **Deleted**
- ~~Legacy compose entry: `docker-compose.legacy.yml`~~ **Deleted**

Historical code is preserved in git history under these paths for reference.

## Development Rules

1. New product work must be implemented in `backend-go/` and `frontend-next/`.
2. API contracts use `/api/v1/*`.
3. All legacy behavior references should look up git history (`git show HEAD~1:backend/...`).

## Default Commands

```bash
docker compose up -d
```
