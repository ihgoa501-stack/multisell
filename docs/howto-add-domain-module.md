# How to 添加新领域模块

> 在凌镜后端添加一个新的 CRUD 领域模块，包含完整的路由、处理器、服务和数据模型。

---

## 前置条件

- 熟悉 Go 1.25 语法和 GORM
- 凌镜后端开发环境已就绪（详见 [入门教程](tutorial-getting-started.md)）
- 了解 [模块模式](reference-module-catalog.md#1-平台基础设施-internalplatform)

## 步骤

### 1. 创建模块目录

```bash
cd backend-go
mkdir -p internal/domain/yourmodule
```

### 2. 创建数据模型 (`model.go`)

```go
package yourmodule

import (
    "time"
    "gorm.io/gorm"
)

// YourModel 是数据库模型。
type YourModel struct {
    ID        int64          `gorm:"primaryKey" json:"id"`
    Name      string         `gorm:"type:varchar(255);not null" json:"name"`
    Status    string         `gorm:"type:varchar(50);default:active" json:"status"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// CreateRequest 和 CreateResponse 放在同一个文件里。
type CreateRequest struct {
    Name   string `json:"name" binding:"required,max=255"`
    Status string `json:"status"`
}

type UpdateRequest struct {
    Name   string `json:"name" binding:"omitempty,max=255"`
    Status string `json:"status"`
}

type Response struct {
    ID        int64     `json:"id"`
    Name      string    `json:"name"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
}
```

### 3. 创建服务层 (`service.go`)

```go
package yourmodule

import (
    "go.uber.org/zap"
    "gorm.io/gorm"
)

type Service struct {
    db  *gorm.DB
    log *zap.Logger
}

func NewService(db *gorm.DB, log *zap.Logger) *Service {
    return &Service{db: db, log: log}
}

func (s *Service) List(page, size int) ([]YourModel, int64, error) {
    var items []YourModel
    var total int64
    query := s.db.Model(&YourModel{})
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }
    if err := query.Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
        return nil, 0, err
    }
    return items, total, nil
}

func (s *Service) GetByID(id int64) (*YourModel, error) {
    var item YourModel
    if err := s.db.First(&item, id).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (s *Service) Create(req *CreateRequest) (*YourModel, error) {
    item := YourModel{
        Name:   req.Name,
        Status: req.Status,
    }
    if item.Status == "" {
        item.Status = "active"
    }
    if err := s.db.Create(&item).Error; err != nil {
        return nil, err
    }
    return &item, nil
}

func (s *Service) Update(id int64, req *UpdateRequest) (*YourModel, error) {
    item, err := s.GetByID(id)
    if err != nil {
        return nil, err
    }
    updates := map[string]interface{}{}
    if req.Name != "" {
        updates["name"] = req.Name
    }
    if req.Status != "" {
        updates["status"] = req.Status
    }
    if err := s.db.Model(item).Updates(updates).Error; err != nil {
        return nil, err
    }
    return item, nil
}

func (s *Service) Delete(id int64) error {
    return s.db.Delete(&YourModel{}, id).Error
}
```

### 4. 创建处理器 (`handler.go`)

```go
package yourmodule

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/lingmirror/backend-go/internal/common"
    "github.com/lingmirror/backend-go/internal/response"
    "go.uber.org/zap"
    "gorm.io/gorm"
)

type Handler struct {
    svc *Service
    log *zap.Logger
}

func NewHandler(db *gorm.DB, log *zap.Logger) *Handler {
    return &Handler{
        svc: NewService(db, log),
        log: log,
    }
}

func (h *Handler) List(c *gin.Context) {
    page, size := common.ParsePagination(c)
    items, total, err := h.svc.List(page, size)
    if err != nil {
        response.InternalError(c, err)
        return
    }
    response.Paginated(c, items, total, page, size)
}

func (h *Handler) Get(c *gin.Context) {
    id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
    item, err := h.svc.GetByID(id)
    if err != nil {
        response.Error(c, http.StatusNotFound, "not found")
        return
    }
    response.Success(c, item)
}

func (h *Handler) Create(c *gin.Context) {
    var req CreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }
    item, err := h.svc.Create(&req)
    if err != nil {
        response.InternalError(c, err)
        return
    }
    response.Success(c, item)
}

func (h *Handler) Update(c *gin.Context) {
    id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
    var req UpdateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }
    item, err := h.svc.Update(id, &req)
    if err != nil {
        response.InternalError(c, err)
        return
    }
    response.Success(c, item)
}

func (h *Handler) Delete(c *gin.Context) {
    id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
    if err := h.svc.Delete(id); err != nil {
        response.InternalError(c, err)
        return
    }
    response.Success(c, nil)
}
```

### 5. 创建路由注册 (`routes.go`)

```go
package yourmodule

import (
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    "gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
    h := NewHandler(db, logger)
    rg.GET("/your-module", h.List)
    rg.GET("/your-module/:id", h.Get)
    rg.POST("/your-module", h.Create)
    rg.PUT("/your-module/:id", h.Update)
    rg.DELETE("/your-module/:id", h.Delete)
}
```

### 6. 在路由汇总中注册

打开 `internal/httpx/router.go`，添加 import 和注册代码：

```go
import "github.com/lingmirror/backend-go/internal/domain/yourmodule"

// 在 RegisterRoutes 调用行附近添加：
yourmodule.RegisterRoutes(protected, db, logger)
```

### 7. 创建迁移文件

```sql
-- backend-go/migrations/000027_your_module.up.sql
CREATE TABLE IF NOT EXISTS your_models (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_your_models_deleted_at ON your_models(deleted_at);

-- backend-go/migrations/000027_your_module.down.sql
DROP TABLE IF EXISTS your_models;
```

### 8. 写测试

```go
package yourmodule

import (
    "testing"
    "github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_Create(t *testing.T) {
    db := dbtest.NewDB(t, &YourModel{})
    svc := NewService(db, dbtest.NewLogger(t))

    item, err := svc.Create(&CreateRequest{Name: "测试", Status: "active"})
    if err != nil {
        t.Fatalf("创建失败: %v", err)
    }
    if item.Name != "测试" {
        t.Errorf("Name = %q, 期望 %q", item.Name, "测试")
    }
}
```

## 验证

```bash
cd backend-go
go test ./internal/domain/yourmodule/
go vet ./internal/domain/yourmodule/
go build ./...
```

## 故障排查

| 问题 | 原因与解决 |
|------|-----------|
| 编译报 `undefined: YourModel` | 检查 model.go 中的结构体定义是否被其他文件正确导入 |
| 路由 404 | 确认 `routes.go` 的路径没重复 `/api/v1` 前缀（protected 组已有） |
| 数据库表不存在 | 运行迁移，或设置 `AutoMigrate` |
| 测试连 DB 失败 | 用 `dbtest.NewDB`（内存 SQLite），不依赖外部 PostgreSQL |

---

## 相关文档

- [参考 - 模块目录](reference-module-catalog.md)
- [参考 - API 快速参考](reference-api-quick.md)
