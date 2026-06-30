# How to 配置与部署

> 将凌镜部署到生产环境。当前方案基于 Docker Compose + Nginx。

---

## 前置条件

- 一台 Linux 服务器（Ubuntu 24.04 已测试）
- Docker 和 Docker Compose v2
- PostgreSQL 15（Docker 部署）
- 域名（可选，配置 HTTPS 需要）
- Sentry DSN（可选，错误追踪）

## 步骤

### 1. 生产环境配置

复制环境变量模板并修改：

```bash
cp deploy/.env.example .env
```

必须修改的项：

```ini
# 生产环境密钥
JWT_SECRET=<生成一个强随机字符串>

# 数据库
DB_PASSWORD=<强密码>

# 请使用 LLM 提供商密钥
LLM_API_KEY=<your-openai-or-anthropic-key>

# 生产环境限制跨域来源
CORS_ALLOWED_ORIGINS=https://yourdomain.com

# 生产环境日志格式用 JSON（方便日志聚合）
LOG_FORMAT=json
LOG_LEVEL=info
```

### 2. 数据库

```bash
# 启动 PostgreSQL
docker compose -f deploy/docker-compose.yml up -d db

# 运行迁移（后端启动时自动执行）
docker compose -f deploy/docker-compose.yml up -d backend
```

### 3. 构建和启动服务

```bash
# 构建并启动全部服务
docker compose -f deploy/docker-compose.yml up -d --build

# 查看日志
docker compose -f deploy/docker-compose.yml logs -f backend
```

### 4. Nginx 配置

`deploy/nginx.conf` 已包含反向代理配置：

```nginx
server {
    listen 80;
    server_name yourdomain.com;

    location /api/ {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /ws {
        proxy_pass http://backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    location / {
        proxy_pass http://frontend:3000;
    }
}
```

HTTPS 建议用 Caddy（`Caddyfile.prod` 已提供）：

```caddy
yourdomain.com {
    reverse_proxy /api/* backend:8080
    reverse_proxy /ws backend:8080
    reverse_proxy /* frontend:3000
}
```

### 5. 健康检查

```bash
curl https://yourdomain.com/api/health
# → {"status":"ok","version":"0.3.0.0"}
```

### 6. 监控（可选）

```bash
# 启动 Prometheus + Grafana
docker compose -f docker-compose.monitoring.yml up -d
```

Grafana 默认地址: `http://localhost:3001`，预置仪表盘在 `deploy/grafana/dashboards/`。

Sentry 配置：设置 `SENTRY_DSN` 环境变量即可自动集成。

## 验证清单

- [ ] `/api/health` 返回 `{"status":"ok"}`
- [ ] 前端页面可访问，能登录
- [ ] WebSocket 连接正常（`/ws`）
- [ ] Agent 定时任务日志无错误
- [ ] 平台集成连接测试通过
- [ ] 数据库迁移已执行（表结构正确）

## 备份与恢复

### 备份

```bash
# 每日备份脚本（已提供 scripts/backup.sh）
bash scripts/backup.sh

# 手动备份
docker exec -t multisell-db pg_dump -U postgres multisell > backup_$(date +%Y%m%d).sql
```

### 恢复

```bash
cat backup_20260630.sql | docker exec -i multisell-db psql -U postgres multisell
```

## 故障排查

| 问题 | 解决 |
|------|------|
| 后端容器崩溃 | `docker logs multisell-backend` 查看错误。检查数据库连接配置。 |
| WebSocket 连不上 | Nginx 确认配置了 Upgrade/Connection 头。|
| 前端白屏 | 检查 `NEXT_PUBLIC_API_URL` 是否指向正确的后端地址。 |
| 权限被拒 | 检查 `JWT_SECRET` 是否匹配。检查 RBAC 角色配置。 |
| 慢查询 | PostgreSQL 中 `EXPLAIN ANALYZE` 排查，确认 GORM 索引已创建。 |

---

## 相关文档

- [参考 - 配置参考](reference-configuration.md)
- [生产服务器信息](../../production-server-info.md)
- [部署脚本](../../scripts/deploy.sh)
