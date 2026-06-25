# 凌镜 LingMirror — 运维手册

> 版本：2.0
> 最后更新：2026-06-24
> 适用环境：当前 Go + Next 新栈

## 1. 环境概览

| 组件 | 技术栈 | 默认端口 | Docker 服务名 |
|---|---|---:|---|
| 数据库 | PostgreSQL 15 | 5432 | `db` |
| 后端 | Go / Gin / GORM | 8080 | `backend-go` |
| 前端 | Next.js / React / Ant Design | 3000 | `frontend-next` |

旧服务：

- `backend/` Python/FastAPI：reference-only
- `frontend/` Vue：reference-only
- `docker-compose.legacy.yml`：rollback/reference only

## 2. 启动与停止

启动全栈：

```bash
docker compose up -d
```

仅启动数据库：

```bash
docker compose up -d db
```

查看状态：

```bash
docker compose ps
```

停止：

```bash
docker compose down
```

停止并删除卷前必须确认这是可丢弃环境：

```bash
docker compose down -v
```

## 3. 本地开发启动

后端：

```bash
cd backend-go
go run cmd/server/main.go
```

前端：

```bash
cd frontend-next
npm run dev -- --hostname 127.0.0.1 --port 3000
```

## 4. 健康检查

```bash
curl http://localhost:8080/api/health
curl http://localhost:8080/api/v1/health
```

预期响应包含：

```json
{"status":"ok"}
```

## 5. 日志

Docker：

```bash
docker compose logs -f
docker compose logs -f backend-go
docker compose logs -f frontend-next
docker compose logs -f db
```

本地开发：

- 后端日志直接看 `go run` 终端。
- 前端日志直接看 `npm run dev` 终端。

## 6. 数据库

默认开发数据库：

| 变量 | 默认值 |
|---|---|
| `DB_HOST` | `localhost` / compose 中为 `db` |
| `DB_PORT` | `5432` |
| `DB_USER` | `postgres` |
| `DB_PASSWORD` | `postgres` |
| `DB_NAME` | `multisell` |

进入数据库：

```bash
docker compose exec db psql -U postgres -d multisell
```

备份：

```bash
docker compose exec db pg_dump -U postgres multisell > backup-$(date +%Y%m%d-%H%M%S).sql
```

恢复前请确认目标库可覆盖：

```bash
cat backup.sql | docker compose exec -T db psql -U postgres -d multisell
```

## 7. 迁移

当前 Go 新栈迁移文件在：

```text
backend-go/migrations/
```

关键文件：

- `000001_init_schema.up.sql`
- `000003_data_migration.up.sql`
- `validate.sql`
- `MIGRATION_RUNBOOK.md`

迁移执行方式以当前部署脚本/运维脚本为准。不要再使用旧 Alembic 流程管理新栈数据库。

验证迁移：

```bash
psql "$DATABASE_URL" -f backend-go/migrations/validate.sql
```

## 8. 验证命令

后端：

```bash
cd backend-go
go test ./...
go vet ./...
go build -o bin/server cmd/server/main.go
```

前端：

```bash
cd frontend-next
npm test
npm run build
npm run lint
```

当前已知：`npm run lint` 尚未通过，生产门禁不能把 lint 视为绿色。

## 9. 生产配置检查

上线前至少确认：

- `JWT_SECRET` 已设置且不是开发默认值。
- `DB_*` 指向正确数据库。
- `LLM_PROVIDER`、`LLM_API_KEY`、`LLM_MODEL` 按生产策略配置。
- `NEXT_PUBLIC_API_URL` 指向正确 API base，通常为 `/api` 或后端网关地址。
- Sentry release/source map 上传策略明确配置或显式关闭。

## 10. 回滚与旧栈

旧栈仅用于 reference / rollback：

```bash
docker compose -f docker-compose.legacy.yml up -d
```

除非明确执行回滚演练或安全修复，不要修改 `backend/` 和 `frontend/`。

## 11. 常见问题

### 前端请求 404

检查 API path 是否缺少 `/v1`。当前默认：

```text
NEXT_PUBLIC_API_URL=http://localhost:8080/api
```

因此调用应写：

```text
/v1/...
```

最终请求：

```text
/api/v1/...
```

### 前端 build 通过但 lint 失败

这是当前已知状态。修复 lint 后才能将前端质量门禁视为完全通过。

### 数据库连接失败

确认：

```bash
docker compose ps db
docker compose exec db pg_isready -U postgres
```

并检查 `DB_HOST` 在本地和 compose 环境中是否分别使用 `localhost` / `db`。
