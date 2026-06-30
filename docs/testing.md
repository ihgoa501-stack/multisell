# Testing Guide

## Unit Tests

Unit tests use an in-memory SQLite database provided by `internal/dbtest`.
Each test gets an isolated database instance, safe for `t.Parallel()`.

```go
import "github.com/lingmirror/backend-go/internal/dbtest"

func TestSomething(t *testing.T) {
    db := dbtest.NewDB(t, &MyModel{})
    svc := NewService(db, logger)
    // ...
}
```

## API Integration Tests

Integration tests exercise the full HTTP stack:

```
HTTP request -> Gin routing -> middleware (CORS, Auth) -> handler -> service -> database
```

They use `internal/integrationtest.NewTestServer` which:

- Starts an `httptest.Server` with a full Gin engine
- Connects an in-memory SQLite database (via `dbtest`)
- Applies CORS, RequestID, and Auth middleware
- Registers auth routes (login/register) so tests can obtain JWT tokens
- Calls the domain's `RegisterRoutes` function on the JWT-protected route group
- Automatically migrates the `auth.User` table for authentication

### Writing Integration Tests

```go
import (
    "encoding/json"
    "net/http"
    "testing"

    "github.com/lingmirror/backend-go/internal/integrationtest"
    "github.com/lingmirror/backend-go/internal/response"
)

func TestMyRoutes(t *testing.T) {
    // Create test server, passing domain models for auto-migration
    ts := integrationtest.NewTestServer(t, domain.RegisterRoutes, &domain.Model{})
    defer ts.Close()

    // Get JWT token for authenticated requests
    token := ts.Login(t)

    // Test unauthenticated access (401)
    resp := ts.Get(t, "/api/v1/domain/path", "")
    defer resp.Body.Close()
    // resp.StatusCode should be 401

    // Test authenticated requests
    resp = ts.Post(t, "/api/v1/domain/path", body, token)
    defer resp.Body.Close()

    // Parse standard response envelope
    var result response.Result
    json.NewDecoder(resp.Body).Decode(&result)
    // result.Code == 0 for success
}
```

### Test Server Helper Methods

| Method | Description |
|--------|-------------|
| `Get(t, path, token)` | HTTP GET with optional Bearer token |
| `Post(t, path, body, token)` | HTTP POST with JSON body |
| `Put(t, path, body, token)` | HTTP PUT with JSON body |
| `Delete(t, path, token)` | HTTP DELETE with optional Bearer token |
| `Login(t)` | Register a test user and return JWT access token |

### For Modules with RBAC

Modules gated by `RequirePermission` (like finance in production) need the middleware applied inside the registration closure. However, in integration tests we recommend registering routes directly on the protected group to focus on testing the domain logic rather than RBAC middleware (which has its own tests in the RBAC package):

```go
ts := integrationtest.NewTestServer(t,
    func(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
        RegisterRoutes(rg, db, logger)
    },
    &domain.Model{},
)
```

### Running Tests

```bash
# All tests
cd backend-go && go test ./...

# Single module
cd backend-go && go test ./internal/domain/order/...

# Integration tests only (by naming convention)
cd backend-go && go test ./... -run "Test.*Routes"

# With verbose output
cd backend-go && go test ./internal/domain/order/... -v -run TestOrderRoutes
```

### Test File Naming Convention

| File | Content |
|------|---------|
| `<module>_test.go` | Unit tests (service layer, isolated) |
| `routes_test.go` | Integration tests (HTTP -> DB, full stack) |

### CI

Integration tests run the same way as unit tests -- no special CI setup required because `dbtest` uses in-memory SQLite (not PostgreSQL). For PostgreSQL-specific tests, see the `postgres:15-alpine` service in CI configuration.

## 闭环集成测试

可信经营闭环的集成测试位于 `internal/integrationtest/closed_loop_test.go`。

### 测试覆盖的路径

**Happy Path (`TestTrustedClosedLoop_HappyPath`)**

完整的可信经营闭环 HTTP 流程：

1. **注册用户 → 获取 JWT token**
2. **创建候选商品** (`POST /api/v1/candidates`)
3. **全链路评估** (`POST /api/v1/loop/evaluate/:id`)
4. **Owner 采纳建议** (`POST /api/v1/owner/suggestions/:id/feedback {action:"adopt"}`)
5. **审批通过** (`PUT /api/v1/approval/:id/review {action:"approve"}`)
6. **设置任务状态为 approved + 关联 approval_id**
7. **执行上架** (`POST /api/v1/listing-task/:id/execute`)
8. 验证 listing task 状态为 `completed`
9. 验证 recommendation feedback_status 为 `executed`
10. 验证 operation_log 审计记录存在

**失败路径 (`TestTrustedClosedLoop_Unauthenticated`)**

验证所有闭环 API 在没有 JWT 时返回 401：

- `GET /api/v1/candidates`
- `POST /api/v1/candidates`
- `POST /api/v1/loop/evaluate/1`
- `GET /api/v1/owner/suggestions`
- `POST /api/v1/owner/suggestions/1/feedback`
- `GET /api/v1/approval`
- `PUT /api/v1/approval/1/review`
- `GET /api/v1/listing-tasks`
- `POST /api/v1/listing-task/1/execute`

**执行门禁失败路径**

- `TestTrustedClosedLoop_BlockedTaskCannotExecute` — blocked 状态拒绝执行
- `TestTrustedClosedLoop_NoApprovalRejected` — 缺少 approval_id 拒绝执行

### 运行闭环测试

```bash
# 所有集成测试
cd backend-go && go test ./internal/integrationtest/... -v

# 仅闭环测试
cd backend-go && go test ./internal/integrationtest/... -v -run "TestTrustedClosedLoop"

# 全量测试
cd backend-go && go test ./...
```

### 测试架构

闭环集成测试使用 `internal/integrationtest.NewTestServer` 创建隔离的 HTTP 测试服务器：

- **数据库**：in-memory SQLite（`dbtest.NewDB`），每个测试实例独立
- **中间件**：CORS + RequestID + Auth（JWT）
- **路由**：手动注册 listingtask、loop、approval、owner、candidate 路由
- **依赖注入**：创建完整的服务链（`approval.Service` → `loop.Service` → `listingtask.Service`）

### 路由注册模式

由于 listingtask 路由注册函数需要多个依赖参数，集成测试中使用闭包包装：

```go
func registerClosedLoopRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
    opsvc := operationlog.NewService(db, logger)
    apprSvc := approval.NewService(db, logger, opsvc)
    loopSvc := loop.NewService(db, logger, nil, false)

    listingtask.RegisterRoutes(rg, db, logger, nil, false, apprSvc, opsvc, nil, loopSvc)
    approval.RegisterRoutes(rg, db, logger, opsvc)
    owner.RegisterRoutes(rg, db, logger)
    candidate.RegisterRoutes(rg, db, logger)
    loop.RegisterRoutes(rg, db, logger, nil, false)
}
```
