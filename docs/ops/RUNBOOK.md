# 凌镜 LingMirror — 运维手册

> **版本**：2.1
> **最后更新**：2026-07-03
> **适用环境**：当前 Go + Next 新栈
> **相关文档**：[突发事件响应](INCIDENT_RESPONSE.md) | [灾难恢复](DISASTER_RECOVERY.md)

---

## 1. 环境概览

| 组件 | 技术栈 | 默认端口 | Docker 服务名 |
|---|---|---:|---|
| 数据库 | PostgreSQL 15 | 5432 | `db` |
| 后端 | Go / Gin / GORM | 8080 | `backend` |
| 前端 | Next.js / React / Ant Design | 3000 | `frontend` |
| 反向代理 | Caddy | 443/80 | `caddy` |
| 监控 | Prometheus | 9090 | `prometheus` |
| 看板 | Grafana | 3001 | `grafana` |
| 告警 | Alertmanager | -- | `alertmanager` |

旧服务：

- `backend/` Python/FastAPI：reference-only
- `frontend/` Vue：reference-only
- `docker-compose.legacy.yml`：rollback/reference only

## 2. 启动与停止

| 操作 | 命令 |
|------|------|
| 启动全栈 | `docker compose up -d` |
| 启动带监控 | `docker compose -f docker-compose.yml -f docker-compose.monitoring.yml up -d` |
| 生产部署 | `docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build` |
| 仅数据库 | `docker compose up -d db` |
| 停止 | `docker compose down` |
| 停止并删除数据卷 | `docker compose down -v`（仅限可丢弃环境） |
| 查看状态 | `docker compose ps` |
| 重启单个服务 | `docker compose restart <服务名>` |
| 强制重建 | `docker compose up -d --force-recreate <服务名>` |

---

## 3. 快速诊断

```bash
# API 健康检查
curl -f http://localhost:8080/api/health

# 检查数据库连接
docker compose exec db pg_isready -U postgres

# 查看容器资源占用
docker stats --no-stream

# 查看磁盘
df -h /

# 查看 Docker 卷大小
docker system df -v
```

---

## 4. 日志

```bash
# 追踪所有日志
docker compose logs -f

# 按服务
docker compose logs -f backend
docker compose logs -f frontend
docker compose logs -f db

# 查看最后 N 行
docker compose logs --tail=100 backend

# 按关键词搜索
docker compose logs --tail=500 backend | grep -i "error\|panic\|fatal"

# 查看 Caddy 访问日志
docker compose logs caddy
```

## 5. 数据库操作

### 5.1 连接与查询

```bash
# 进入 psql 交互环境
docker compose exec db psql -U postgres -d multisell

# 直接执行 SQL
docker compose exec db psql -U postgres -d multisell -c "SELECT count(*) FROM users;"

# 查看当前连接数
docker compose exec db psql -U postgres -c "SELECT count(*) FROM pg_stat_activity;"

# 查看慢查询（运行超过 5 秒的）
docker compose exec db psql -U postgres -d multisell -c "
  SELECT pid, now() - pg_stat_activity.query_start AS duration, query, state
  FROM pg_stat_activity
  WHERE state != 'idle' AND now() - pg_stat_activity.query_start > interval '5 seconds'
  ORDER BY duration DESC;
"
```

### 5.2 备份与恢复

```bash
# 手动备份
docker compose exec db pg_dump -U postgres multisell > backup_$(date +%Y%m%d_%H%M%S).sql

# 使用备份服务（推荐，支持 S3 同步）
docker compose run --rm backup

# 恢复（注意：会覆盖现有数据）
cat backup_file.sql | docker compose exec -T db psql -U postgres multisell

# 查看备份目录
docker run --rm -v db_backups:/backups alpine ls -lh /backups
```

**独立脚本（非 Docker 环境）：**

- `scripts/backup.sh` — pg_dump 备份，支持本地保留天数 + S3 上传
- `scripts/restore.sh <backup_file>` — 从 custom-format dump 恢复，含完整性校验和危险确认
- `scripts/backup.env.example` — 环境变量模板

**定时备份（crontab 示例）：**

```bash
# 每天凌晨 3 点备份，保留 7 天
0 3 * * * cd /path/to/project && ./scripts/backup.sh >> /var/log/db_backup.log 2>&1
```

### 5.3 迁移

```bash
# 应用所有未执行的迁移
docker compose run --rm migrate

# 回退一个版本
docker compose run --rm migrate down 1

# 回退到指定版本
docker compose run --rm migrate goto <version>

# 查看当前版本
docker compose exec db psql -U postgres -d multisell -c "SELECT version, dirty FROM schema_migrations;"
```

迁移文件：`backend-go/migrations/`，每个版本有 `.up.sql` + `.down.sql`。

---

## 6. 回退流程

> 发版前必须先验证回退流程可行。详见 `scripts/rollback.sh`。

### 6.1 Git Tag 策略

每次发版必须在 `main` 分支打 annotated tag，格式为 `v<major>.<minor>.<patch>`，例如：

```bash
# 发版时打 tag（与 VERSION 文件保持同步）
git tag -a v0.4.2.0 -m "Release v0.4.2.0"
git push origin v0.4.2.0
```

现有 tag：`v0.3.0.0`、`v0.3.1`、`v0.4.0.0`、`v0.4.1.0`。

```bash
# 查看所有 tag
git tag -l | sort -V

# 查看两个版本间的变更
git log --oneline v0.4.0.0..HEAD
```

### 6.2 回退步骤

```bash
# Step 1: 确认当前版本回退目标
git tag -l | sort -V
git log --oneline v0.4.1.0..HEAD   # 查看当前发版的变更

# Step 2: 执行回退脚本（不包含数据库回滚）
./scripts/rollback.sh

# 或包含数据库回滚（如果当前发版包含迁移）
./scripts/rollback.sh --revert-migration

# Step 3: 验证回退结果
# 3a) 确认代码回退到目标版本
git log --oneline -1

# 3b) 确认健康检查通过
sleep 5 && curl -sf http://localhost:8080/api/health && echo "  OK" || echo "  FAIL"

# 3c) 确认数据库迁移版本（如果执行了回滚）
docker compose exec db psql -U postgres -d multisell -c \
  "SELECT version, dirty FROM schema_migrations;"

# 3d) 确认前端可访问
curl -sf -o /dev/null -w "%{http_code}" http://localhost/

# Step 4: 如果回退失败需要恢复，重新部署
./scripts/deploy.sh
```

### 6.3 发版前回退验证清单

每次发版前必须完整执行一次回退验证：

- [ ] 当前 HEAD 对应一个 tag（`git tag -l --points-at HEAD`）
- [ ] 回退脚本可用（`./scripts/rollback.sh` 以 dry-run 确认参数正确）
- [ ] 目标回退版本的健康检查可预期通过（API + 前端）
- [ ] 如果本次包含数据库迁移，已测试 `--revert-migration` 路径
- [ ] 回退后重新部署可恢复（`./scripts/deploy.sh`）

### 6.4 数据库迁移回退原则

- 每个迁移文件必须有对应的 `down.sql`，否则拒绝合并
- `down.sql` 必须能回退到上一个 Schema 版本而不丢数据
- 如果回退迁移会丢数据，在 `down.sql` 注释中写明风险
- 迁移回退命令：`docker compose -f docker-compose.yml -f docker-compose.prod.yml run --rm migrate down 1`

---

## 7. 监控

| 组件 | 访问地址 | 说明 |
|------|---------|------|
| Prometheus | `http://localhost:9090` | 指标查询，仅本地访问 |
| Grafana | `http://localhost:3001` | 看板，管理员密码由 `GRAFANA_ADMIN_PASSWORD` 设置 |
| Alertmanager | 内部 | 告警路由，输出到 Slack |

启动监控栈：
```bash
docker compose -f docker-compose.yml -f docker-compose.monitoring.yml up -d
```

---

## 8. 生产配置检查清单

上线/部署前确认：

- [ ] `JWT_SECRET` 已设置且非默认值 `dev-secret-change-in-production`
- [ ] `DB_PASSWORD` 已设置为强密码
- [ ] `LLM_API_KEY` / `LLM_PROVIDER` / `LLM_MODEL` 按生产策略配置
- [ ] `NEXT_PUBLIC_API_URL` 正确指向 API base（通常为 `/api`）
- [ ] `DOMAIN` 和 `ACME_EMAIL` 已设置，Caddy 自动 TLS 可工作
- [ ] `GRAFANA_ADMIN_PASSWORD` 已设置
- [ ] 定时备份 crontab 已配置
- [ ] 服务器防火墙只开放 80/443

---

## 9. 验证命令

```bash
# 后端
cd backend-go && go test ./... && go vet ./... && go build -o /dev/null cmd/server/main.go

# 前端
cd frontend-next && npm test && npm run build

# 注意：npm run lint 已知未通过，不阻断部署但需尽快修复
```

---

## 10. 常见问题速查

| 问题 | 可能原因 | 快速解决 |
|------|---------|---------|
| 前端请求 404 | API path 缺 `/v1` | 检查 `NEXT_PUBLIC_API_URL`，请求写 `/v1/...` |
| 数据库连接失败 | DB_HOST 用错 | 本地用 `localhost`，Docker 内用 `db` |
| 后端 crash 循环 | OOM / 端口冲突 | `docker compose logs backend` 查原因 |
| AI 功能超时 | LLM provider 不可达 | 检查 `LLM_API_KEY` 和 provider 状态 |
| 后端 build 通过但 lint 失败 | 已知问题 | 不阻断部署，但需尽快修复 |
| Webhook 不回调 | Caddy 配置 / 签名过期 | 检查 `docker compose logs caddy` |
| 前端 build 失败 | 依赖版本冲突 | `cd frontend-next && npm ci` 重新安装 |

## 10. 告警通知渠道配置

系统通过配置项 `llm.daily_budget_usd` 和 Webhook 通知渠道发送告警。

### 10.1 LLM 预算控制

| 配置项 | 环境变量 | 说明 | 默认值 |
|--------|---------|------|--------|
| `llm.daily_budget_usd` | `LLM_DAILY_BUDGET_USD` | 每日 LLM 调用预算上限 (USD) | `0` (不限制) |

超限行为：
- 调用 `Allow()` 返回 `ActionBlock`，拒绝 LLM 调用
- 同时记录 `WARN` 级别日志
- 可在 Grafana 仪表盘查看 `llm_cost_logs` 表的日消耗趋势

### 10.2 Webhook 告警

系统内置了 `WebhookSender` (`backend-go/internal/platform/alert/`)：

```go
sender := alert.NewWebhookSender([]string{"https://hooks.example.com/alert"})
sender.SendHealthAlert(ctx, "backend", "service unreachable")
```

| 字段 | 说明 | 示例 |
|------|------|------|
| `title` | 告警标题 | `[backend] Health check failed` |
| `message` | 详情描述 | `service unreachable` |
| `level` | 级别 | `critical` / `error` / `warning` / `info` |
| `timestamp` | ISO 8601 时间 | `2026-07-06T12:00:00Z` |
| `source` | 服务名 | `backend` |

支持的接收端：
- **飞书机器人**：使用飞书 Webhook URL
- **Slack Webhook**：使用 Slack Incoming Webhook URL
- **自定义**：任意接受 POST JSON 的 HTTP 端点

配置方式（`config.yaml`）：
```yaml
alert:
  webhook_urls:
    - "https://hooks.example.com/alert"
```
