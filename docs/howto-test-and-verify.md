# How to 运行测试与验证

> 凌镜后端和前端测试的最佳实践，确保改动不会破坏已有功能。

---

## 前置条件

- Docker Desktop 已安装（用于启动 PostgreSQL）
- Go 1.25+、Node.js 20+
- 已安装项目依赖

## 步骤

### 1. 后端全部测试

```bash
cd backend-go
docker compose up -d db   # 启动 PostgreSQL（部分集成测试需要）
go test ./...             # 运行全部测试
```

**预期输出:** `ok   github.com/lingmirror/backend-go/...  (X.XXXs)`

单个包测试：

```bash
go test -v ./internal/domain/order/
go test -v -run TestCreate ./internal/domain/order/   # 按测试名称过滤
```

### 2. 后端静态分析

```bash
go vet ./...              # 检查常见错误
go build -o /dev/null ./cmd/server/main.go  # 确认编译通过
```

### 3. 测试数据库

大部分领域模块使用 `dbtest` 工具，无需真 PostgreSQL：

```go
import "github.com/lingmirror/backend-go/internal/dbtest"

func TestX(t *testing.T) {
    db := dbtest.NewDB(t, &MyModel{})  // 内存 SQLite，每个测试隔离
    svc := NewService(db, dbtest.NewLogger(t))
    // ... 测试逻辑
}
```

特点：
- 每个测试独立数据库（不同目录下的临时文件）
- 自动迁移你传入的模型
- 支持 `t.Parallel()`
- 也提供 `StringPtr` / `IntPtr` / `FloatPtr` 辅助函数

### 4. 前端测试

```bash
cd frontend-next
npm test                 # Vitest 单元测试
npm run build            # 生产构建（也跑 lint 和类型检查）
```

### 5. E2E 测试

```bash
cd frontend-next/e2e
npx playwright test      # Playwright 端到端测试
```

需要后端和前端都已启动。E2E 测试访问真实浏览器环境。

### 6. 冒烟测试

```bash
# 验证 API 是否在线
curl http://localhost:8080/api/health

# 验证登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}'

# 验证前端
open http://localhost:3000
```

## CI 验证

项目使用 GitHub Actions / pre-commit hooks。提交前确保：

```bash
cd backend-go && go test ./... && go vet ./... && go build ./...
cd frontend-next && npm run build
```

## 测试覆盖策略

| 层 | 测试方式 | 示例 |
|----|---------|------|
| Service | `dbtest.NewDB` + 直接调 service 方法 | `TestService_Create` |
| Handler | httptest + mock service | `TestHandler_List` |
| Integration | 需要 Docker PostgreSQL | 订单同步、平台集成 |
| E2E | Playwright | 用户登录→创建商品→发布 |

## 故障排查

| 问题 | 原因与解决 |
|------|-----------|
| `go test ./...` 超时 | Docker PostgreSQL 未启动。运行 `docker compose up -d db`。 |
| 前端 build 报错 | 检查 TypeScript 类型。`npm run lint`（已知有遗留 lint 问题）。 |
| 测试连 5432 被拒 | 确认 `docker ps` 中 postgres 容器在运行。 |
| E2E 找不到页面 | 启动前后端后再运行 E2E：`npm run dev -- --hostname 127.0.0.1 --port 3000` |

---

## 相关文档

- [参考 - API 快速参考](reference-api-quick.md)
- [How to 配置与部署](howto-deploy.md)
