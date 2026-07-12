# LingMirror Backend (Active)

This is the active LingMirror backend.

- Stack: Go / Gin / GORM / PostgreSQL
- Entry: `cmd/server/main.go`
- API prefix: `/api/v1`
- Liveness check: `/api/health`
- Readiness check (database + EventBus + Scheduler): `/api/ready`
- Migrations: `migrations/`

## Development

```bash
go run cmd/server/main.go
```

The default local config is `configs/config.yaml`. Environment variables can override database and server settings:

- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `JWT_SECRET`
- `SERVER_PORT`

## Verification

```bash
go test ./...
go vet ./...
go build -o bin/server cmd/server/main.go
```

## Legacy Boundary

The old Python backend in `../backend/` is paused. Use it only as a behavior reference for parity, migration, or emergency rollback support.
