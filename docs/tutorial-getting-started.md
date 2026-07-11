# 入门教程：从零搭建到创建第一个商品

> 你将搭建一个完整的凌镜开发环境，创建第一个商品，看到它在仪表盘里出现。全程约 15 分钟。

---

## 你需要什么

- **Docker Desktop**（[下载](https://www.docker.com/products/docker-desktop/)）— 运行 PostgreSQL
- **Go 1.25+**（`go version` 确认）— 后端编译
- **Node.js 20+**（`node --version` 确认）— 前端运行
- **Git**（`git --version` 确认）— 代码管理
- 一个终端

---

## 第一步：启动基础设施（2 分钟）

```bash
# 从项目根目录启动 PostgreSQL
docker compose up -d db
```

你看到 `✔ Container multisell-db Started` 就对了。PostgreSQL 15 在 `localhost:5432` 运行，数据库 `multisell` 已自动创建。

验证：

```bash
docker compose ps
# 你应该看到 multisell-db 的状态是 "Up"
```

---

## 第二步：启动后端（3 分钟）

开一个新终端窗口：

```bash
cd backend-go
go run cmd/server/main.go
```

第一次启动会自动：
1. 连接 PostgreSQL
2. 运行数据库迁移（建表）
3. 注册所有 60+ 领域模块路由
4. 启动 Agent 调度器
5. Seed mock 演示数据

你看到这样就算成功了：

```
[INFO] Starting server on :8080
[INFO] Database migrated
[INFO] Mock data seeded
```

验证后端：

```bash
curl http://localhost:8080/api/health
# → {"status":"ok","version":"0.3.0.0"}
```

---

## 第三步：启动前端并看到商品（3 分钟）

再开一个终端：

```bash
cd frontend-next
npm install
npm run dev -- --hostname 127.0.0.1 --port 3000
```

等看到 `✓ Ready in XXXms`，打开浏览器访问 **http://localhost:3000**。

你应该看到凌镜登录页。登录（Mock seed 数据默认账号密码可在 `.env.example` 中找到或查看后端日志）：

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
# 复制返回的 token
```

登录后你会看到仪表盘——已经有一些 mock 商品数据了。点左侧「商品管理」可以看到商品列表。

---

## 第四步：创建一个商品（2 分钟）

通过 API 创建一个新商品：

```bash
curl -X POST http://localhost:8080/api/v1/products \
  -H "Authorization: Bearer <刚才得到的token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "无线蓝牙耳机 - 测试",
    "description": "高品质无线蓝牙耳机，支持降噪",
    "category_id": 1,
    "brand_id": 1,
    "status": "active"
  }'
```

成功返回：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": 1,
    "name": "无线蓝牙耳机 - 测试",
    "status": "active",
    ...
  }
}
```

现在回到浏览器，刷新商品管理页面——你创建的商品就在列表里。

---

## 下一步

你已经完成了一个完整的凌镜开发环：启动 DB → 启动后端 → 启动前端 → 创建商品。

继续探索：

| 如果你想... | 去看... |
|------------|---------|
| 了解项目各模块 | [模块目录](reference-module-catalog.md) |
| 添加一个新模块 | [How to 添加新领域模块](howto-add-domain-module.md) |
| 给商品添加 SKU 和价格 | API 参考的商品管理部分 |
| 配置一个真实平台 | [How to 平台集成](howto-platform-integrations.md) |
| 理解 Agent 如何协作 | [Agent Pipeline 解释](explanation-agent-pipeline.md) |
| 运行测试确保改动可靠 | [How to 运行测试](howto-test-and-verify.md) |

---

## 故障排查

| 问题 | 解决 |
|------|------|
| `docker compose up -d db` 报端口占用 | 关掉本地 PostgreSQL 服务或改 `docker-compose.yml` 端口映射 |
| 后端启动报数据库连接失败 | 检查 `DB_HOST` 是否为 `localhost`，确认 Docker PostgreSQL 在运行 |
| 前端页面白屏 | 检查浏览器控制台网络请求，确认 `NEXT_PUBLIC_API_URL` 指向 `http://localhost:8080/api` |
| curl 返回 401 | Token 过期了。重新登录获取新的 token。 |
| go run 报找不到依赖 | 运行 `go mod tidy` 后再重试 |

---

## 相关文档

- [参考 - 配置参考](reference-configuration.md) — 了解全部配置项
- [Owner 与 AI 统一部署测试手册](ops/OWNER_AND_AI_DEPLOYMENT_RUNBOOK.md) — 唯一服务器部署、恢复与验收入口
