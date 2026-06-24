# 凌镜 LingMirror — 运维手册

> 版本: 1.0
> 最后更新: 2026-06-24
> 适用环境: 开发 / 生产

---

## 目录

1. [环境概览](#1-环境概览)
2. [启动与停止](#2-启动与停止)
3. [查看日志](#3-查看日志)
4. [数据库备份](#4-数据库备份)
5. [数据库恢复](#5-数据库恢复)
6. [数据库迁移](#6-数据库迁移)
7. [健康检查](#7-健康检查)
8. [重置环境](#8-重置环境)
9. [容器管理](#9-容器管理)
10. [直接访问数据库](#10-直接访问数据库)
11. [生产部署](#11-生产部署)
12. [常见问题](#12-常见问题)

---

## 1. 环境概览

| 组件 | 技术栈 | 端口 | Docker 服务名 |
|------|--------|------|---------------|
| 数据库 | PostgreSQL 15 (Alpine) | 5432 | `db` |
| 后端 (Python) | FastAPI + SQLAlchemy async | 8000 | `backend` |
| 后端 (Go)  | Go + Chi + sqlx | 8080 | `backend-go`（独立部署） |
| 前端 | Vue 3 + Naive UI + Vite | 3000 | `frontend` |

数据库默认凭据：

| 变量 | 值 |
|------|-----|
| 数据库名 | `product_management` |
| 用户 | `postgres` |
| 密码 | `postgres` |

> **注意**: 备份脚本中使用的默认数据库名为 `multisell`，可通过 `DB_NAME` 环境变量覆写。实际容器内数据库名为 `product_management`（参见 `docker-compose.yml`）。

---

## 2. 启动与停止

### 启动所有服务

```bash
docker compose up -d
```

等待数据库健康检查通过后，后端和前端会自动启动。

### 启动特定服务

```bash
docker compose up -d db
docker compose up -d backend
docker compose up -d frontend
```

### 停止所有服务

```bash
docker compose down
```

### 停止并删除数据卷（见[重置环境](#8-重置环境)）

### 查看运行状态

```bash
docker compose ps
```

---

## 3. 查看日志

### 查看所有服务日志

```bash
docker compose logs -f
```

### 查看后端日志

```bash
docker compose logs -f backend
```

### 查看前端日志

```bash
docker compose logs -f frontend
```

### 查看数据库日志

```bash
docker compose logs -f db
```

### 查看最近 N 行

```bash
docker compose logs --tail=100 -f backend
```

---

## 4. 数据库备份

### 基本用法

```bash
bash scripts/backup.sh
```

备份文件将生成在 `./backups/` 目录，格式为 `multisell_YYYY-MM-DD_HHMMSS.sql.gz`。

### 使用环境变量

```bash
DB_HOST=db.example.com \
DB_PORT=5432 \
DB_USER=myuser \
DB_PASSWORD=mypass \
DB_NAME=product_management \
BACKUP_DIR=/data/backups \
RETENTION_DAYS=30 \
bash scripts/backup.sh
```

### 启用 S3 同步

```bash
BACKUP_S3_BUCKET=my-bucket \
bash scripts/backup.sh
```

需要预先配置好 AWS CLI 凭据（`aws configure` 或环境变量 `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY`）。

### 自动清理

脚本默认保留 7 天的备份。超过 `RETENTION_DAYS` 的旧备份会被自动删除。

### 定时备份（crontab）

```bash
# 每天凌晨 3 点备份，保留 30 天
0 3 * * * cd /path/to/project && DB_NAME=product_management RETENTION_DAYS=30 bash scripts/backup.sh >> /var/log/multisell-backup.log 2>&1
```

---

## 5. 数据库恢复

### 基本用法

```bash
bash scripts/restore.sh ./backups/multisell_2026-06-24_030000.sql.gz
```

### 恢复流程

1. 脚本会先用 `pg_restore --list` 验证备份文件完整性。
2. 如果文件有效，会显示一段**红色警告**，提示你将删除并重建数据库。
3. 输入 `yes` 确认后开始恢复。
4. 恢复中会自动终止现有数据库连接。

### 指定目标数据库

```bash
DB_NAME=product_management bash scripts/restore.sh ./backups/multisell_2026-06-24_030000.sql.gz
```

---

## 6. 数据库迁移

### Python 后端

迁移自动在容器启动时运行（通过 `alembic upgrade head`）。如需手动执行：

```bash
# 进入后端容器
docker compose exec backend alembic upgrade head

# 查看当前迁移版本
docker compose exec backend alembic current --verbose

# 回滚一步
docker compose exec backend alembic downgrade -1
```

### 从宿主机执行（需要本地虚拟环境）

```bash
cd backend
.venv/bin/alembic upgrade heads
.venv/bin/alembic current --verbose
```

> **注意**: Alembic 目前有多个 head，始终使用 `upgrade heads`（复数形式）。

### Go 后端迁移

> Go 后端尚在迁移中。当就绪后，使用：

```bash
cd backend-go
make migrate-up
```

---

## 7. 健康检查

### API 健康端点

```bash
curl http://localhost:8000/api/health
```

成功响应示例：

```json
{"status": "ok"}
```

### Go 后端健康端点（将来）

```bash
curl http://localhost:8080/api/health
```

### 数据库健康检查

```bash
docker compose exec db pg_isready -U postgres
```

生产环境中，Docker Compose 会自动执行健康检查（每 30 秒），容器状态可通过 `docker compose ps` 查看。

---

## 8. 重置环境

### 完整重置（删除所有数据）

```bash
docker compose down -v && docker compose up -d
```

> **注意**: `-v` 标志会删除所有数据卷（包括 PostgreSQL 数据和上传文件）。**不可逆操作**，确保已备份。

### 仅重启服务（保留数据）

```bash
docker compose down && docker compose up -d
```

### 重建镜像并重启

```bash
docker compose down && docker compose build --no-cache && docker compose up -d
```

---

## 9. 容器管理

### 查看容器状态

```bash
docker compose ps
```

### 查看资源占用

```bash
docker stats
```

### 进入容器 Shell

```bash
# 进入后端容器
docker compose exec backend sh

# 进入数据库容器
docker compose exec db sh

# 进入前端容器
docker compose exec frontend sh
```

### 导出容器日志到文件

```bash
docker compose logs backend > backend.log
```

---

## 10. 直接访问数据库

### 通过 Docker

```bash
docker compose exec db psql -U postgres -d product_management
```

### 通过宿主机 psql

```bash
psql -h localhost -p 5432 -U postgres -d product_management
```

### 常用 SQL 查询

```sql
-- 查看所有表
\dt

-- 查看表结构
\d+ table_name

-- 查看连接数
SELECT count(*) FROM pg_stat_activity WHERE datname = 'product_management';

-- 终止所有连接（恢复前使用）
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = 'product_management' AND pid <> pg_backend_pid();
```

---

## 11. 生产部署

### 首次部署

```bash
# 构建并启动
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build

# 执行数据初始化
docker compose run --rm backend python seed.py
```

### 更新部署

```bash
# 拉取最新代码
git pull

# 重新构建并启动
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build

# 如果需要后端 Go 服务（将来）
cd backend-go && make build && ./bin/server
```

### 生产环境变量

参见 `docker-compose.prod.yml`。必须通过 `.env` 文件或宿主环境变量设置以下密钥：

| 变量 | 说明 |
|------|------|
| `ENCRYPTION_KEY` | 32 字符十六进制加密密钥 |
| `LLM_API_KEY` | AI 模型 API 密钥 |
| `LLM_API_URL` | AI 模型 API 地址 |
| `SECRET_KEY` | JWT / Session 密钥 |

### 备份策略建议

- 每天凌晨执行一次备份（crontab），保留 30 天。
- 启用 S3 同步将备份异地存储。
- 部署前执行完整备份。
- 定期演练恢复流程，确保备份可用。

---

## 12. 常见问题

### 端口冲突

**问题**: 启动时提示 `port is already allocated`。

**排查**:

```bash
# 检查端口占用
lsof -i :5432    # PostgreSQL
lsof -i :8000    # 后端
lsof -i :3000    # 前端
lsof -i :8080    # Go 后端（将来）
```

**解决**:

```bash
# 停止占用端口的进程
kill <PID>

# 或修改 docker-compose.yml 中的端口映射，例如将宿主机端口改为 5433
# ports:
#   - "5433:5432"
```

如果本地安装了 PostgreSQL（非 Docker），它可能占用了 5432 端口。停止本地 PostgreSQL 后再启动 Docker：

```bash
# macOS
brew services stop postgresql

# Linux (systemd)
sudo systemctl stop postgresql
```

### 数据库连接失败

**问题**: 后端日志中提示 `could not connect to server` 或 `connection refused`。

**排查**:

```bash
# 检查数据库容器是否运行
docker compose ps db

# 检查数据库健康状态
docker compose exec db pg_isready -U postgres

# 检查数据库日志
docker compose logs db --tail=50
```

**解决**:

- 等待数据库启动完成（首次启动需要几秒钟初始化）。
- 确认 `DATABASE_URL` 中的主机名为 `db`（Docker 内部服务名），而非 `localhost`。
- 确认检查 `POSTGRES_DB`、`POSTGRES_USER`、`POSTGRES_PASSWORD` 是否与连接字符串匹配。

### 迁移失败

**问题**: `alembic upgrade head` 报错，容器启动失败。

**排查**:

```bash
# 查看迁移报错详情
docker compose logs backend --tail=100 | grep -i error

# 进入数据库查看 alembic_version 表
docker compose exec db psql -U postgres -d product_management -c "SELECT * FROM alembic_version;"
```

**解决**:

```bash
# 方法 1：手动指定迁移版本回退
docker compose run --rm backend alembic downgrade -1

# 方法 2：如果因为迁移失败导致后端容器无法启动，先启动数据库
docker compose up -d db

# 然后单独运行迁移
docker compose run --rm backend alembic upgrade head

# 方法 3：在紧急情况下，手动修正 alembic_version 表
docker compose exec db psql -U postgres -d product_management \
  -c "UPDATE alembic_version SET version_num = '<上一个版本号>';"
```

> **注意**: 当前项目有多个 Alembic head。如果出现多个 head 错误，使用 `alembic upgrade heads` 升级所有 head，或手动合并。

### 镜像构建失败

**问题**: `docker compose up -d --build` 时构建失败。

**排查**:

```bash
# 单独构建某个服务查看详细错误
docker compose build --no-cache backend

# 或查看构建日志
docker compose up --build backend 2>&1 | tee build.log
```

**常见原因**:

- **pip 安装失败**: 网络问题或依赖冲突。检查 `requirements.txt`，尝试更换 pip 镜像源。
- **Node 构建失败**: 检查 Node 版本兼容性，查看 `npm run build` 的错误输出。
- **磁盘空间不足**: `df -h` 检查磁盘，`docker system prune` 清理无用数据。

```bash
# 清理 Docker 构建缓存（不影响运行容器）
docker builder prune

# 清理所有未使用的 Docker 资源
docker system prune -a
```

### 备份恢复失败

**问题**: 恢复脚本报错，备份文件无效。

**原因**:

- 备份文件被损坏或使用了不同版本的 pg_dump/pg_restore。
- PostgreSQL 主版本不匹配。pg_dump 兼容高版本到低版本，但不保证低版本到高版本。
- 备份格式错误。确保备份是 `pg_dump --format=custom` 格式。

**解决**:

```bash
# 检查 pg_dump 和 pg_restore 版本
pg_dump --version
pg_restore --version

# 版本应与目标数据库的 PostgreSQL 版本匹配（>= 15）
docker compose exec db psql -U postgres -c "SELECT version();"
```

### 磁盘空间不足

**问题**: Docker 数据卷或日志占满磁盘。

**排查**:

```bash
# 查看各数据卷大小
docker system df -v

# 查看 Docker 目录整体占用
du -sh /var/lib/docker/
```

**解决**:

```bash
# 清理已停止的容器
docker container prune

# 清理未使用的镜像
docker image prune

# 全面清理（慎用，会删除所有未使用的资源）
docker system prune -a --volumes
```

---

## 附录 A: 快速参考卡片

| 操作 | 命令 |
|------|------|
| 启动 | `docker compose up -d` |
| 停止 | `docker compose down` |
| 查看状态 | `docker compose ps` |
| 查看日志 | `docker compose logs -f backend` |
| 备份 | `bash scripts/backup.sh` |
| 恢复 | `bash scripts/restore.sh <file>` |
| 迁移 | `docker compose exec backend alembic upgrade head` |
| 健康检查 | `curl http://localhost:8000/api/health` |
| 重置 | `docker compose down -v && docker compose up -d` |
| 生产部署 | `docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build` |
| 进入数据库 | `docker compose exec db psql -U postgres -d product_management` |
