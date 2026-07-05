> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# 一人 Agent 公司 MVP — Day 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build product completeness check engine + 20 seed candidate products + register routes.

**Architecture:** Two new domain modules (`candidate` for product CRUD, `completeness` for data quality scoring) under `backend-go/internal/domain/`. All new code goes in `backend-go/` only. DB migration for new tables.

**Tech Stack:** Go 1.25, GORM, Gin, PostgreSQL 15 (migration), SQLite (test via dbtest).

**Context:** The user answered: Ozon (mock platform), Chinese (console language), Lead Agent defines the 20 products.

## Global Constraints

- All new code goes in `backend-go/` — never touch `backend/` (legacy Python) or `frontend-next/` (Day 1 is backend only)
- Follow existing domain module pattern: model.go → service.go → handler.go → routes.go (see platformfee, approval for reference)
- Use `dbtest.NewDB(t, &MyModel{})` for test DB isolation
- Use `response.Success`, `response.Error`, `response.Paginated` for HTTP responses
- Use `common.ParsePagination(c)` for pagination
- Routes register via `RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger)`
- Router wire in `internal/httpx/router.go` under the `protected` group
- All API paths prefixed with `/api/v1` (handled by parent router group)
- Use `go.uber.org/zap` for logging
- Migration files go in `migrations/` with sequential numbering (next: 000032)
- No AutoMigrate — production schema comes from migrations

---

## File Map

### New files to create

| # | Path | Purpose |
|---|------|---------|
| 1 | `migrations/000032_candidate_product.up.sql` | CREATE TABLE candidate_product |
| 2 | `migrations/000032_candidate_product.down.sql` | DROP TABLE candidate_product |
| 3 | `migrations/000033_completeness_check.up.sql` | CREATE TABLE completeness_check |
| 4 | `migrations/000033_completeness_check.down.sql` | DROP TABLE completeness_check |
| 5 | `internal/domain/candidate/model.go` | CandidateProduct GORM model + input types |
| 6 | `internal/domain/candidate/service.go` | CRUD business logic |
| 7 | `internal/domain/candidate/handler.go` | HTTP request handlers |
| 8 | `internal/domain/candidate/routes.go` | Route registration |
| 9 | `internal/domain/candidate/candidate_test.go` | Full CRUD tests |
| 10 | `internal/domain/completeness/model.go` | CompletenessCheck model + dimension definitions |
| 11 | `internal/domain/completeness/service.go` | Check logic + scoring |
| 12 | `internal/domain/completeness/handler.go` | HTTP handlers |
| 13 | `internal/domain/completeness/routes.go` | Route registration |
| 14 | `internal/domain/completeness/completeness_test.go` | Check logic tests |
| 15 | `cmd/seed/main.go` | Seed script for 20 candidate products |

### Existing files to modify

| # | Path | Change |
|---|------|--------|
| 16 | `internal/httpx/router.go` | Add imports + registration for candidate + completeness |

---

## Task 1: Database Migration — candidate_product table

**Files:**
- Create: `migrations/000032_candidate_product.up.sql`
- Create: `migrations/000032_candidate_product.down.sql`

**Interfaces:**
- Consumes: Nothing
- Produces: Table `candidate_product` with columns matching CandidateProduct model

- [ ] **Step 1: Create up migration**

```sql
-- migrations/000032_candidate_product.up.sql

CREATE TABLE IF NOT EXISTS candidate_product (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    main_image TEXT,
    images JSONB DEFAULT '[]'::jsonb,
    category_id BIGINT,
    brand_id BIGINT,
    specs JSONB DEFAULT '{}'::jsonb,
    purchase_price DOUBLE PRECISION DEFAULT 0,
    purchase_currency VARCHAR(3) DEFAULT 'CNY',
    supplier_id BIGINT,
    package_weight_kg DOUBLE PRECISION DEFAULT 0,
    package_length_cm DOUBLE PRECISION DEFAULT 0,
    package_width_cm DOUBLE PRECISION DEFAULT 0,
    package_height_cm DOUBLE PRECISION DEFAULT 0,
    target_sale_price DOUBLE PRECISION DEFAULT 0,
    target_currency VARCHAR(3) DEFAULT 'USD',
    destination_country VARCHAR(10) DEFAULT '',
    hs_code VARCHAR(20) DEFAULT '',
    status VARCHAR(20) DEFAULT 'draft',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_candidate_product_status ON candidate_product(status);
```

- [ ] **Step 2: Create down migration**

```sql
-- migrations/000032_candidate_product.down.sql

DROP TABLE IF EXISTS candidate_product;
```

- [ ] **Step 3: Stage and test (optional — skip if no Docker running)**

Run: `docker compose exec -T db psql -U postgres -d multisell -f migrations/000032_candidate_product.up.sql`

Expected: `CREATE TABLE`

---

## Task 2: Database Migration — completeness_check table

**Files:**
- Create: `migrations/000033_completeness_check.up.sql`
- Create: `migrations/000033_completeness_check.down.sql`

**Interfaces:**
- Consumes: `candidate_product.id` (foreign key reference)
- Produces: Table `completeness_check`

- [ ] **Step 1: Create up migration**

```sql
-- migrations/000033_completeness_check.up.sql

CREATE TABLE IF NOT EXISTS completeness_check (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES candidate_product(id) ON DELETE CASCADE,
    total_score DOUBLE PRECISION DEFAULT 0,
    max_score DOUBLE PRECISION DEFAULT 100,
    percentage DOUBLE PRECISION DEFAULT 0,
    missing_items JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_completeness_check_product ON completeness_check(product_id);
```

- [ ] **Step 2: Create down migration**

```sql
-- migrations/000033_completeness_check.down.sql

DROP TABLE IF EXISTS completeness_check;
```

---

## Task 3: CandidateProduct model

**Files:**
- Create: `internal/domain/candidate/model.go`

**Interfaces:**
- Consumes: Nothing (standalone model)
- Produces: `CandidateProduct` struct, `CreateCandidateInput`, `UpdateCandidateInput`, `CandidateListResponse`

- [ ] **Step 1: Write the model file**

```go
// internal/domain/candidate/model.go
package candidate

import (
	"encoding/json"
	"time"
)

// CandidateProduct maps to the "candidate_product" table.
type CandidateProduct struct {
	ID                 int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Title              string          `gorm:"column:title;size:500;not null" json:"title"`
	Description        string          `gorm:"column:description;type:text" json:"description,omitempty"`
	MainImage          string          `gorm:"column:main_image" json:"main_image,omitempty"`
	Images             json.RawMessage `gorm:"column:images;type:jsonb" json:"images,omitempty"`
	CategoryID         *int64          `gorm:"column:category_id" json:"category_id,omitempty"`
	BrandID            *int64          `gorm:"column:brand_id" json:"brand_id,omitempty"`
	Specs              json.RawMessage `gorm:"column:specs;type:jsonb" json:"specs,omitempty"`
	PurchasePrice      float64         `gorm:"column:purchase_price" json:"purchase_price"`
	PurchaseCurrency   string          `gorm:"column:purchase_currency;size:3;default:CNY" json:"purchase_currency"`
	SupplierID         *int64          `gorm:"column:supplier_id" json:"supplier_id,omitempty"`
	PackageWeightKG    float64         `gorm:"column:package_weight_kg" json:"package_weight_kg"`
	PackageLengthCM    float64         `gorm:"column:package_length_cm" json:"package_length_cm"`
	PackageWidthCM     float64         `gorm:"column:package_width_cm" json:"package_width_cm"`
	PackageHeightCM    float64         `gorm:"column:package_height_cm" json:"package_height_cm"`
	TargetSalePrice    float64         `gorm:"column:target_sale_price" json:"target_sale_price"`
	TargetCurrency     string          `gorm:"column:target_currency;size:3;default:USD" json:"target_currency"`
	DestinationCountry string          `gorm:"column:destination_country;size:10" json:"destination_country"`
	HSCode             string          `gorm:"column:hs_code;size:20" json:"hs_code,omitempty"`
	Status             string          `gorm:"column:status;size:20;default:draft" json:"status"`
	CreatedAt          time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (CandidateProduct) TableName() string { return "candidate_product" }

// --- Input / DTO types ---

type CreateCandidateInput struct {
	Title              string          `json:"title" binding:"required"`
	Description        string          `json:"description"`
	MainImage          string          `json:"main_image"`
	Images             json.RawMessage `json:"images"`
	CategoryID         *int64          `json:"category_id"`
	BrandID            *int64          `json:"brand_id"`
	Specs              json.RawMessage `json:"specs"`
	PurchasePrice      float64         `json:"purchase_price"`
	PurchaseCurrency   string          `json:"purchase_currency"`
	SupplierID         *int64          `json:"supplier_id"`
	PackageWeightKG    float64         `json:"package_weight_kg"`
	PackageLengthCM    float64         `json:"package_length_cm"`
	PackageWidthCM     float64         `json:"package_width_cm"`
	PackageHeightCM    float64         `json:"package_height_cm"`
	TargetSalePrice    float64         `json:"target_sale_price"`
	TargetCurrency     string          `json:"target_currency"`
	DestinationCountry string          `json:"destination_country"`
	HSCode             string          `json:"hs_code"`
	Status             string          `json:"status"`
}

type UpdateCandidateInput struct {
	Title              *string         `json:"title"`
	Description        *string         `json:"description"`
	MainImage          *string         `json:"main_image"`
	Images             json.RawMessage `json:"images"`
	CategoryID         *int64          `json:"category_id"`
	BrandID            *int64          `json:"brand_id"`
	Specs              json.RawMessage `json:"specs"`
	PurchasePrice      *float64        `json:"purchase_price"`
	PurchaseCurrency   *string         `json:"purchase_currency"`
	SupplierID         *int64          `json:"supplier_id"`
	PackageWeightKG    *float64        `json:"package_weight_kg"`
	PackageLengthCM    *float64        `json:"package_length_cm"`
	PackageWidthCM     *float64        `json:"package_width_cm"`
	PackageHeightCM    *float64        `json:"package_height_cm"`
	TargetSalePrice    *float64        `json:"target_sale_price"`
	TargetCurrency     *string         `json:"target_currency"`
	DestinationCountry *string         `json:"destination_country"`
	HSCode             *string         `json:"hs_code"`
	Status             *string         `json:"status"`
}

type CandidateListResponse struct {
	Items []CandidateProduct `json:"items"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Size  int                `json:"size"`
}
```

---

## Task 4: CandidateProduct service

**Files:**
- Create: `internal/domain/candidate/service.go`

**Interfaces:**
- Consumes: `gorm.DB`, `*zap.Logger`
- Produces: `NewService(db, logger)`, methods: `List`, `Create`, `GetByID`, `Update`, `Delete`

- [ ] **Step 1: Write the service**

```go
// internal/domain/candidate/service.go
package candidate

import (
	"fmt"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides candidate product business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new candidate service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns paginated candidate products.
func (s *Service) List(p *common.Pagination) ([]CandidateProduct, int64, error) {
	q := s.db.Model(&CandidateProduct{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count candidate products: %w", err)
	}
	var items []CandidateProduct
	if err := q.Order("created_at DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list candidate products: %w", err)
	}
	return items, total, nil
}

// Create inserts a new candidate product.
func (s *Service) Create(in *CreateCandidateInput) (*CandidateProduct, error) {
	p := &CandidateProduct{
		Title:              in.Title,
		Description:        in.Description,
		MainImage:          in.MainImage,
		Images:             in.Images,
		CategoryID:         in.CategoryID,
		BrandID:            in.BrandID,
		Specs:              in.Specs,
		PurchasePrice:      in.PurchasePrice,
		PurchaseCurrency:   in.PurchaseCurrency,
		SupplierID:         in.SupplierID,
		PackageWeightKG:    in.PackageWeightKG,
		PackageLengthCM:    in.PackageLengthCM,
		PackageWidthCM:     in.PackageWidthCM,
		PackageHeightCM:    in.PackageHeightCM,
		TargetSalePrice:    in.TargetSalePrice,
		TargetCurrency:     in.TargetCurrency,
		DestinationCountry: in.DestinationCountry,
		HSCode:             in.HSCode,
		Status:             in.Status,
	}
	if p.Status == "" {
		p.Status = "draft"
	}
	if p.PurchaseCurrency == "" {
		p.PurchaseCurrency = "CNY"
	}
	if p.TargetCurrency == "" {
		p.TargetCurrency = "USD"
	}
	if err := s.db.Create(p).Error; err != nil {
		return nil, fmt.Errorf("create candidate product: %w", err)
	}
	return p, nil
}

// GetByID returns a single candidate product.
func (s *Service) GetByID(id int64) (*CandidateProduct, error) {
	var p CandidateProduct
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Update applies partial updates to a candidate product.
func (s *Service) Update(id int64, in *UpdateCandidateInput) (*CandidateProduct, error) {
	updates := map[string]interface{}{}
	if in.Title != nil {
		updates["title"] = *in.Title
	}
	if in.Description != nil {
		updates["description"] = *in.Description
	}
	if in.MainImage != nil {
		updates["main_image"] = *in.MainImage
	}
	if in.Images != nil {
		updates["images"] = in.Images
	}
	if in.CategoryID != nil {
		updates["category_id"] = *in.CategoryID
	}
	if in.BrandID != nil {
		updates["brand_id"] = *in.BrandID
	}
	if in.Specs != nil {
		updates["specs"] = in.Specs
	}
	if in.PurchasePrice != nil {
		updates["purchase_price"] = *in.PurchasePrice
	}
	if in.PurchaseCurrency != nil {
		updates["purchase_currency"] = *in.PurchaseCurrency
	}
	if in.SupplierID != nil {
		updates["supplier_id"] = *in.SupplierID
	}
	if in.PackageWeightKG != nil {
		updates["package_weight_kg"] = *in.PackageWeightKG
	}
	if in.PackageLengthCM != nil {
		updates["package_length_cm"] = *in.PackageLengthCM
	}
	if in.PackageWidthCM != nil {
		updates["package_width_cm"] = *in.PackageWidthCM
	}
	if in.PackageHeightCM != nil {
		updates["package_height_cm"] = *in.PackageHeightCM
	}
	if in.TargetSalePrice != nil {
		updates["target_sale_price"] = *in.TargetSalePrice
	}
	if in.TargetCurrency != nil {
		updates["target_currency"] = *in.TargetCurrency
	}
	if in.DestinationCountry != nil {
		updates["destination_country"] = *in.DestinationCountry
	}
	if in.HSCode != nil {
		updates["hs_code"] = *in.HSCode
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if err := s.db.Model(&CandidateProduct{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("update candidate product: %w", err)
	}
	return s.GetByID(id)
}

// Delete removes a candidate product.
func (s *Service) Delete(id int64) error {
	return s.db.Delete(&CandidateProduct{}, id).Error
}
```

---

## Task 5: CandidateProduct handler + routes

**Files:**
- Create: `internal/domain/candidate/handler.go`
- Create: `internal/domain/candidate/routes.go`

**Interfaces:**
- Consumes: `Service`
- Produces: HTTP handlers for GET/POST/PUT/DELETE on `/api/v1/candidate`

- [ ] **Step 1: Write handler.go**

```go
// internal/domain/candidate/handler.go
package candidate

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles candidate product HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new candidate handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseID(c *gin.Context) (int64, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return 0, false
	}
	return id, true
}

// List GET /candidate
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	items, total, err := h.service.List(p)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// Create POST /candidate
func (h *Handler) Create(c *gin.Context) {
	var in CreateCandidateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	product, err := h.service.Create(&in)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, product)
}

// Get GET /candidate/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	product, err := h.service.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "候选商品不存在")
			return
		}
		response.InternalError(c, err)
		return
	}
	response.Success(c, product)
}

// Update PUT /candidate/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdateCandidateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	product, err := h.service.Update(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "候选商品不存在")
			return
		}
		response.InternalError(c, err)
		return
	}
	response.Success(c, product)
}

// Delete DELETE /candidate/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(id); err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}
```

- [ ] **Step 2: Write routes.go**

```go
// internal/domain/candidate/routes.go
package candidate

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers candidate product routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	g := rg.Group("/candidate")
	{
		g.GET("", h.List)
		g.POST("", h.Create)
		g.GET("/:id", h.Get)
		g.PUT("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
	}
}
```

---

## Task 6: CandidateProduct tests

**Files:**
- Create: `internal/domain/candidate/candidate_test.go`

**Interfaces:**
- Tests: List, Create, GetByID, Update, Delete using `dbtest.NewDB`

- [ ] **Step 1: Write the full test file**

```go
// internal/domain/candidate/candidate_test.go
package candidate

import (
	"encoding/json"
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestCandidateCRUD(t *testing.T) {
	db := dbtest.NewDB(t, &CandidateProduct{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger)

	t.Run("Create", func(t *testing.T) {
		in := &CreateCandidateInput{
			Title:              "Test Product",
			Description:        "A test product description",
			MainImage:          "https://example.com/image.jpg",
			Images:             json.RawMessage(`["https://example.com/img1.jpg"]`),
			Specs:              json.RawMessage(`{"color":"red","weight":"100g"}`),
			PurchasePrice:      50.0,
			PurchaseCurrency:   "CNY",
			PackageWeightKG:    0.5,
			PackageLengthCM:    20,
			PackageWidthCM:     15,
			PackageHeightCM:    10,
			TargetSalePrice:    25.0,
			TargetCurrency:     "USD",
			DestinationCountry: "RU",
			HSCode:             "8504.40",
			Status:             "draft",
		}
		p, err := svc.Create(in)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if p.Title != "Test Product" {
			t.Errorf("got title=%q, want %q", p.Title, "Test Product")
		}
		if p.PurchasePrice != 50.0 {
			t.Errorf("got purchase_price=%f, want 50.0", p.PurchasePrice)
		}
		if p.Status != "draft" {
			t.Errorf("got status=%q, want draft", p.Status)
		}
		if p.ID == 0 {
			t.Error("expected non-zero ID")
		}
	})

	t.Run("GetByID_not_found", func(t *testing.T) {
		_, err := svc.GetByID(999)
		if err == nil {
			t.Fatal("expected error for non-existent product")
		}
	})

	t.Run("List_empty", func(t *testing.T) {
		// List uses Create in same DB but we need to be careful with test isolation.
		// dbtest.NewDB creates a fresh SQLite per call, so this test has its own DB.
		// To test with data, we must create within the same t.Run or share setup.
	})
}

func TestCandidateList(t *testing.T) {
	db := dbtest.NewDB(t, &CandidateProduct{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger)

	// Create 3 products
	for i := 0; i < 3; i++ {
		_, err := svc.Create(&CreateCandidateInput{
			Title:              "Product " + string(rune('A'+i)),
			DestinationCountry: "RU",
		})
		if err != nil {
			t.Fatalf("setup create failed: %v", err)
		}
	}

	t.Run("List_all", func(t *testing.T) {
		p := &common.Pagination{Page: 1, Size: 10}
		items, total, err := svc.List(p)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 3 {
			t.Errorf("got total=%d, want 3", total)
		}
		if len(items) != 3 {
			t.Errorf("got %d items, want 3", len(items))
		}
	})

	t.Run("List_paginated", func(t *testing.T) {
		p := &common.Pagination{Page: 1, Size: 2}
		items, total, err := svc.List(p)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 3 {
			t.Errorf("got total=%d, want 3", total)
		}
		if len(items) != 2 {
			t.Errorf("got %d items, want 2", len(items))
		}
	})
}

func TestCandidateUpdate(t *testing.T) {
	db := dbtest.NewDB(t, &CandidateProduct{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger)

	p, err := svc.Create(&CreateCandidateInput{
		Title:              "Original",
		PurchasePrice:      30.0,
		DestinationCountry: "RU",
	})
	if err != nil {
		t.Fatalf("setup create failed: %v", err)
	}

	newPrice := 45.0
	newTitle := "Updated"
	updated, err := svc.Update(p.ID, &UpdateCandidateInput{
		Title:         &newTitle,
		PurchasePrice: &newPrice,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Title != "Updated" {
		t.Errorf("got title=%q, want Updated", updated.Title)
	}
	if updated.PurchasePrice != 45.0 {
		t.Errorf("got purchase_price=%f, want 45.0", updated.PurchasePrice)
	}
}

func TestCandidateDelete(t *testing.T) {
	db := dbtest.NewDB(t, &CandidateProduct{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger)

	p, err := svc.Create(&CreateCandidateInput{
		Title:              "To Delete",
		DestinationCountry: "RU",
	})
	if err != nil {
		t.Fatalf("setup create failed: %v", err)
	}

	if err := svc.Delete(p.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = svc.GetByID(p.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
```

---

## Task 7: CompletenessCheck model

**Files:**
- Create: `internal/domain/completeness/model.go`

**Interfaces:**
- Consumes: Nothing
- Produces: `CompletenessCheck` struct, `CheckItem` (dimension definition), `CheckResult` (per-dimension result), `MissingItem` struct

- [ ] **Step 1: Write the model file**

```go
// internal/domain/completeness/model.go
package completeness

import (
	"encoding/json"
	"time"
)

// CompletenessCheck maps to the "completeness_check" table.
type CompletenessCheck struct {
	ID           int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID    int64           `gorm:"column:product_id;not null;index" json:"product_id"`
	TotalScore   float64         `gorm:"column:total_score" json:"total_score"`
	MaxScore     float64         `gorm:"column:max_score;default:100" json:"max_score"`
	Percentage   float64         `gorm:"column:percentage" json:"percentage"`
	MissingItems json.RawMessage `gorm:"column:missing_items;type:jsonb" json:"missing_items,omitempty"`
	CreatedAt    time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (CompletenessCheck) TableName() string { return "completeness_check" }

// MissingItem describes one specific thing that's missing.
type MissingItem struct {
	Field    string `json:"field"`
	Label    string `json:"label"`        // Chinese label for Owner display
	Severity string `json:"severity"`     // required / recommended
	Score    float64 `json:"score"`        // points deducted
}

// CheckDimension defines one completeness dimension.
type CheckDimension struct {
	Field    string
	Label    string
	MaxScore float64
	Severity string // required or recommended
}

// DefaultDimensions returns the standard list of completeness dimensions.
func DefaultDimensions() []CheckDimension {
	return []CheckDimension{
		{Field: "title", Label: "商品标题", MaxScore: 10, Severity: "required"},
		{Field: "description", Label: "商品描述", MaxScore: 5, Severity: "recommended"},
		{Field: "main_image", Label: "主图", MaxScore: 10, Severity: "required"},
		{Field: "category_id", Label: "商品类目", MaxScore: 10, Severity: "required"},
		{Field: "brand_id", Label: "品牌信息", MaxScore: 5, Severity: "recommended"},
		{Field: "specs", Label: "规格参数", MaxScore: 5, Severity: "recommended"},
		{Field: "purchase_price", Label: "采购成本", MaxScore: 10, Severity: "required"},
		{Field: "package_weight_kg", Label: "包装重量", MaxScore: 10, Severity: "required"},
		{Field: "dimensions", Label: "包装尺寸", MaxScore: 10, Severity: "required"},
		{Field: "target_sale_price", Label: "目标售价", MaxScore: 10, Severity: "required"},
		{Field: "destination_country", Label: "目标国家", MaxScore: 10, Severity: "required"},
		{Field: "hs_code", Label: "HS编码", MaxScore: 5, Severity: "recommended"},
	}
}

// MaxPossibleScore returns the sum of all dimension max scores.
func MaxPossibleScore() float64 {
	total := 0.0
	for _, d := range DefaultDimensions() {
		total += d.MaxScore
	}
	return total
}

// --- HTTP request / response ---

type CheckRequest struct {
	ProductID int64 `json:"product_id" binding:"required"`
}

type CheckResponse struct {
	CompletenessCheck
	ItemResults []CheckItemResult `json:"item_results"`
}

type CheckItemResult struct {
	Field   string  `json:"field"`
	Label   string  `json:"label"`
	Score   float64 `json:"score"`
	MaxScore float64 `json:"max_score"`
	Passed  bool    `json:"passed"`
	Severity string `json:"severity"`
	Message  string `json:"message,omitempty"` // Chinese explanation
}

type ListResponse struct {
	Items []CompletenessCheck `json:"items"`
	Total int64               `json:"total"`
	Page  int                 `json:"page"`
	Size  int                 `json:"size"`
}
```

---

## Task 8: CompletenessCheck service

**Files:**
- Create: `internal/domain/completeness/service.go`

**Interfaces:**
- Consumes: `*gorm.DB`, `*zap.Logger`, `candidate.Service` (for fetching product)
- Produces: `Check(productID)` → `CheckResponse`, `ListChecks(p)` → paginated checks, `GetLatestCheck(productID)` → latest check
- Produces: `CheckItemResult` for each dimension

- [ ] **Step 1: Write the service**

```go
// internal/domain/completeness/service.go
package completeness

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides completeness checking business logic.
type Service struct {
	db          *gorm.DB
	logger      *zap.Logger
	candidateSvc *candidate.Service
}

// NewService creates a new completeness service.
func NewService(db *gorm.DB, logger *zap.Logger, candidateSvc *candidate.Service) *Service {
	return &Service{
		db:          db,
		logger:      logger,
		candidateSvc: candidateSvc,
	}
}

// Check evaluates the completeness of a candidate product.
func (s *Service) Check(productID int64) (*CheckResponse, error) {
	prod, err := s.candidateSvc.GetByID(productID)
	if err != nil {
		return nil, fmt.Errorf("get product for completeness check: %w", err)
	}

	dimensions := DefaultDimensions()
	results := make([]CheckItemResult, 0, len(dimensions))
	missing := make([]MissingItem, 0)
	totalScore := 0.0
	maxScore := MaxPossibleScore()

	for _, d := range dimensions {
		score, passed, message := s.evaluateDimension(prod, d)
		totalScore += score
		results = append(results, CheckItemResult{
			Field:    d.Field,
			Label:    d.Label,
			Score:    score,
			MaxScore: d.MaxScore,
			Passed:   passed,
			Severity: d.Severity,
			Message:  message,
		})
		if !passed {
			missing = append(missing, MissingItem{
				Field:    d.Field,
				Label:    d.Label,
				Severity: d.Severity,
				Score:    score,
			})
		}
	}

	percentage := math.Round(totalScore / maxScore * 100)
	missingJSON, _ := json.Marshal(missing)

	check := &CompletenessCheck{
		ProductID:    productID,
		TotalScore:   totalScore,
		MaxScore:     maxScore,
		Percentage:   percentage,
		MissingItems: missingJSON,
	}

	if err := s.db.Create(check).Error; err != nil {
		return nil, fmt.Errorf("save completeness check: %w", err)
	}

	return &CheckResponse{
		CompletenessCheck: *check,
		ItemResults:       results,
	}, nil
}

// evaluateDimension checks one dimension and returns score, passed, Chinese message.
func (s *Service) evaluateDimension(prod *candidate.CandidateProduct, d CheckDimension) (float64, bool, string) {
	switch d.Field {
	case "title":
		if prod.Title == "" {
			return 0, false, "缺少商品标题"
		}
		return d.MaxScore, true, ""
	case "description":
		if prod.Description == "" {
			return 0, false, "缺少商品描述，建议补充"
		}
		return d.MaxScore, true, ""
	case "main_image":
		if prod.MainImage == "" {
			return 0, false, "缺少主图"
		}
		return d.MaxScore, true, ""
	case "category_id":
		if prod.CategoryID == nil || *prod.CategoryID == 0 {
			return 0, false, "缺少商品类目"
		}
		return d.MaxScore, true, ""
	case "brand_id":
		if prod.BrandID == nil || *prod.BrandID == 0 {
			return 0, false, "缺少品牌信息，建议补充"
		}
		return d.MaxScore, true, ""
	case "specs":
		if len(prod.Specs) <= 2 { // empty JSONB is "{}" (2 bytes)
			return 0, false, "缺少规格参数，建议补充"
		}
		return d.MaxScore, true, ""
	case "purchase_price":
		if prod.PurchasePrice <= 0 {
			return 0, false, "缺少采购成本"
		}
		return d.MaxScore, true, ""
	case "package_weight_kg":
		if prod.PackageWeightKG <= 0 {
			return 0, false, "缺少包装重量"
		}
		return d.MaxScore, true, ""
	case "dimensions":
		if prod.PackageLengthCM <= 0 || prod.PackageWidthCM <= 0 || prod.PackageHeightCM <= 0 {
			return 0, false, "缺少包装尺寸（长/宽/高）"
		}
		return d.MaxScore, true, ""
	case "target_sale_price":
		if prod.TargetSalePrice <= 0 {
			return 0, false, "缺少目标售价"
		}
		return d.MaxScore, true, ""
	case "destination_country":
		if prod.DestinationCountry == "" {
			return 0, false, "缺少目标国家"
		}
		return d.MaxScore, true, ""
	case "hs_code":
		if prod.HSCode == "" {
			return 0, false, "缺少HS编码，建议补充"
		}
		return d.MaxScore, true, ""
	default:
		return 0, false, "未知检查项"
	}
}

// ListChecks returns paginated completeness checks (latest first).
func (s *Service) ListChecks(p *common.Pagination) ([]CompletenessCheck, int64, error) {
	q := s.db.Model(&CompletenessCheck{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []CompletenessCheck
	if err := q.Order("created_at DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetLatestCheck returns the most recent completeness check for a product.
func (s *Service) GetLatestCheck(productID int64) (*CompletenessCheck, error) {
	var c CompletenessCheck
	if err := s.db.Where("product_id = ?", productID).Order("created_at DESC").First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// GetChecksByProduct returns all checks for a product (paginated, newest first).
func (s *Service) GetChecksByProduct(productID int64, p *common.Pagination) ([]CompletenessCheck, int64, error) {
	q := s.db.Model(&CompletenessCheck{}).Where("product_id = ?", productID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []CompletenessCheck
	if err := q.Order("created_at DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
```

---

## Task 9: CompletenessCheck handler + routes

**Files:**
- Create: `internal/domain/completeness/handler.go`
- Create: `internal/domain/completeness/routes.go`

**Interfaces:**
- Consumes: `Service`
- Produces: HTTP handlers for POST check, GET checks, GET product checks

- [ ] **Step 1: Write handler.go**

```go
// internal/domain/completeness/handler.go
package completeness

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles completeness check HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new completeness handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseID(c *gin.Context) (int64, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return 0, false
	}
	return id, true
}

// CheckProduct POST /completeness/check/:productId
func (h *Handler) CheckProduct(c *gin.Context) {
	productID, ok := parseID(c)
	if !ok {
		return
	}
	result, err := h.service.Check(productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "商品不存在")
			return
		}
		response.InternalError(c, err)
		return
	}
	response.Success(c, result)
}

// ListChecks GET /completeness
func (h *Handler) ListChecks(c *gin.Context) {
	p := common.ParsePagination(c)
	items, total, err := h.service.ListChecks(p)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetProductCheck GET /completeness/product/:productId
func (h *Handler) GetProductCheck(c *gin.Context) {
	productID, ok := parseID(c)
	if !ok {
		return
	}
	check, err := h.service.GetLatestCheck(productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "该商品暂无完整度检查")
			return
		}
		response.InternalError(c, err)
		return
	}
	response.Success(c, check)
}
```

- [ ] **Step 2: Write routes.go**

```go
// internal/domain/completeness/routes.go
package completeness

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers completeness check routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	candidateSvc := candidate.NewService(db, logger)
	svc := NewService(db, logger, candidateSvc)
	h := NewHandler(svc)

	g := rg.Group("/completeness")
	{
		g.GET("", h.ListChecks)
		g.POST("/check/:productId", h.CheckProduct)
		g.GET("/product/:productId", h.GetProductCheck)
	}
}
```

---

## Task 10: CompletenessCheck tests

**Files:**
- Create: `internal/domain/completeness/completeness_test.go`

**Interfaces:**
- Tests: `Check` logic with various product data, `ListChecks`, `GetLatestCheck`

- [ ] **Step 1: Write the test file**

```go
// internal/domain/completeness/completeness_test.go
package completeness

import (
	"encoding/json"
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func newService(t *testing.T) *Service {
	t.Helper()
	db := dbtest.NewDB(t, &candidate.CandidateProduct{}, &CompletenessCheck{})
	logger := dbtest.NewLogger(t)
	candidateSvc := candidate.NewService(db, logger)
	return NewService(db, logger, candidateSvc)
}

func TestCompletenessCheck_FullProduct(t *testing.T) {
	svc := newService(t)

	prod, err := svc.candidateSvc.Create(&candidate.CreateCandidateInput{
		Title:              "Complete Product",
		Description:        "Full description here",
		MainImage:          "https://example.com/img.jpg",
		Images:             json.RawMessage(`["https://example.com/img1.jpg"]`),
		Specs:              json.RawMessage(`{"color":"red"}`),
		PurchasePrice:      50.0,
		PackageWeightKG:    0.5,
		PackageLengthCM:    20,
		PackageWidthCM:     15,
		PackageHeightCM:    10,
		TargetSalePrice:    25.0,
		DestinationCountry: "RU",
		HSCode:             "8504.40",
	})
	if err != nil {
		t.Fatalf("setup create failed: %v", err)
	}

	result, err := svc.Check(prod.ID)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if result.Percentage < 90 {
		t.Errorf("got percentage=%.0f%%, want >= 90%% for complete product", result.Percentage)
	}
	if len(result.MissingItems) > 0 {
		t.Errorf("got %d missing items, want 0 for complete product: %+v", len(result.MissingItems), result.MissingItems)
	}
}

func TestCompletenessCheck_MinimalProduct(t *testing.T) {
	svc := newService(t)

	prod, err := svc.candidateSvc.Create(&candidate.CreateCandidateInput{
		Title:              "Minimal",
		DestinationCountry: "RU",
	})
	if err != nil {
		t.Fatalf("setup create failed: %v", err)
	}

	result, err := svc.Check(prod.ID)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if result.Percentage >= 50 {
		t.Errorf("got percentage=%.0f%%, want < 50%% for minimal product", result.Percentage)
	}
	if len(result.MissingItems) == 0 {
		t.Fatal("expected missing items for minimal product, got 0")
	}
}

func TestCompletenessCheck_MissingBrandAndSKU(t *testing.T) {
	svc := newService(t)

	prod, err := svc.candidateSvc.Create(&candidate.CreateCandidateInput{
		Title:              "Mid Product",
		Description:        "Has desc",
		MainImage:          "https://example.com/img.jpg",
		PurchasePrice:      30.0,
		PackageWeightKG:    0.3,
		PackageLengthCM:    10,
		PackageWidthCM:     10,
		PackageHeightCM:    10,
		TargetSalePrice:    15.0,
		DestinationCountry: "RU",
	})
	if err != nil {
		t.Fatalf("setup create failed: %v", err)
	}

	result, err := svc.Check(prod.ID)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	// Should be missing: category_id, brand_id, specs, hs_code, images
	// Relevant field names in MissingItems
	found := false
	for _, m := range result.MissingItems {
		if m.Field == "category_id" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected missing category_id, not found")
	}
}

func TestCompletenessListChecks(t *testing.T) {
	svc := newService(t)

	prod, err := svc.candidateSvc.Create(&candidate.CreateCandidateInput{
		Title:              "Test",
		DestinationCountry: "RU",
	})
	if err != nil {
		t.Fatalf("setup create failed: %v", err)
	}

	// Run check twice
	_, err = svc.Check(prod.ID)
	if err != nil {
		t.Fatalf("Check 1 failed: %v", err)
	}
	_, err = svc.Check(prod.ID)
	if err != nil {
		t.Fatalf("Check 2 failed: %v", err)
	}

	t.Run("List_all", func(t *testing.T) {
		p := &common.Pagination{Page: 1, Size: 10}
		items, total, err := svc.ListChecks(p)
		if err != nil {
			t.Fatalf("ListChecks failed: %v", err)
		}
		if total != 2 {
			t.Errorf("got total=%d, want 2", total)
		}
		if len(items) != 2 {
			t.Errorf("got %d items, want 2", len(items))
		}
	})

	t.Run("GetLatestCheck", func(t *testing.T) {
		check, err := svc.GetLatestCheck(prod.ID)
		if err != nil {
			t.Fatalf("GetLatestCheck failed: %v", err)
		}
		if check.ProductID != prod.ID {
			t.Errorf("got product_id=%d, want %d", check.ProductID, prod.ID)
		}
	})
}
```

---

## Task 11: Router registration

**Files:**
- Modify: `internal/httpx/router.go`

**Interfaces:**
- Consumes: imports from `candidate` and `completeness` packages
- Produces: Routes registered under the `protected` group

- [ ] **Step 1: Add imports**

Insert after line 17 (or near `"github.com/lingmirror/backend-go/internal/domain/brand"`):

```go
"github.com/lingmirror/backend-go/internal/domain/candidate"
"github.com/lingmirror/backend-go/internal/domain/completeness"
```

- [ ] **Step 2: Add route registration**

Insert after `supplier.RegisterRoutes(...)` (around line 447):

```go
// Candidate products + completeness check (Day 1 — 一人Agent公司MVP)
candidate.RegisterRoutes(protected, db, logger)
completeness.RegisterRoutes(protected, db, logger)
```

---

## Task 12: 20 seed products

**Files:**
- Create: `cmd/seed/main.go`

**Interfaces:**
- Consumes: `db` connection
- Produces: 20 candidate products inserted into the database

- [ ] **Step 1: Write the seed script**

```go
// cmd/seed/main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/domain/completeness"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfg := config.Load()
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.Name,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	zapLogger, _ := zap.NewProduction()
	svc := candidate.NewService(db, zapLogger)

	// --- Seed data: 10 complete, 5 medium, 5 minimal ---
	products := []candidate.CreateCandidateInput{
		// ===== Full products (score > 80%) =====
		{
			Title: "智能蓝牙耳机 TWS 降噪版", Description: "高品质无线蓝牙耳机，支持主动降噪、IPX5防水、30小时续航。适合跑步、通勤和日常使用。",
			MainImage: "https://img.example.com/tws-earphones.jpg",
			Images:    json.RawMessage(`["https://img.example.com/tws-1.jpg","https://img.example.com/tws-2.jpg"]`),
			Specs:     json.RawMessage(`{"颜色":"黑色","连接方式":"蓝牙5.3","续航":"30小时","防水等级":"IPX5"}`),
			PurchasePrice: 45.0, PurchaseCurrency: "CNY",
			PackageWeightKG: 0.12, PackageLengthCM: 10, PackageWidthCM: 8, PackageHeightCM: 4,
			TargetSalePrice: 19.99, TargetCurrency: "USD", DestinationCountry: "RU",
			HSCode: "8517.62",
		},
		{
			Title: "便携式充电宝 20000mAh", Description: "大容量移动电源，支持22.5W快充，双USB输出，LED电量显示。",
			MainImage: "https://img.example.com/powerbank.jpg",
			Images:    json.RawMessage(`["https://img.example.com/pb-1.jpg"]`),
			Specs:     json.RawMessage(`{"容量":"20000mAh","快充":"22.5W","接口":"USB-C+USB-A"}`),
			PurchasePrice: 55.0, PurchaseCurrency: "CNY",
			PackageWeightKG: 0.35, PackageLengthCM: 15, PackageWidthCM: 8, PackageHeightCM: 3,
			TargetSalePrice: 24.99, TargetCurrency: "USD", DestinationCountry: "RU",
			HSCode: "8507.60",
		},
		{
			Title: "男士运动手表 多功能防水", Description: "多功能运动手表，支持计步、心率监测、睡眠追踪、50米防水。",
			MainImage: "https://img.example.com/sports-watch.jpg",
			Images:    json.RawMessage(`["https://img.example.com/sw-1.jpg","https://img.example.com/sw-2.jpg","https://img.example.com/sw-3.jpg"]`),
			Specs:     json.RawMessage(`{"表盘":"45mm","防水":"50米","续航":"14天","传感器":"心率/计步/血氧"}`),
			PurchasePrice: 68.0, PurchaseCurrency: "CNY",
			PackageWeightKG: 0.08, PackageLengthCM: 12, PackageWidthCM: 10, PackageHeightCM: 6,
			TargetSalePrice: 35.99, TargetCurrency: "USD", DestinationCountry: "RU",
			HSCode: "9102.12",
		},
		{
			Title: "LED台灯 护眼充电式", Description: "可充电LED护眼台灯，三档亮度调节，USB充电，适合学生和办公室使用。",
			MainImage: "https://img.example.com/led-lamp.jpg",
			Images:    json.RawMessage(`["https://img.example.com/lamp-1.jpg","https://img.example.com/lamp-2.jpg"]`),
			Specs:     json.RawMessage(`{"功率":"5W","色温":"3000-6500K","调节":"三档","供电":"USB充电"}`),
			PurchasePrice: 30.0, PurchaseCurrency: "CNY",
			PackageWeightKG: 0.25, PackageLengthCM: 20, PackageWidthCM: 15, PackageHeightCM: 8,
			TargetSalePrice: 14.99, TargetCurrency: "USD", DestinationCountry: "RU",
			HSCode: "9405.20",
		},
		{
			Title: "瑜伽垫 加厚防滑 NBR材质", Description: "加厚防滑瑜伽垫，NBR环保材质，附带绑带和背包。适合瑜伽、普拉提和健身。",
			MainImage: "https://img.example.com/yoga-mat.jpg",
			Images:    json.RawMessage(`["https://img.example.com/ym-1.jpg","https://img.example.com/ym-2.jpg"]`),
			Specs:     json.RawMessage(`{"尺寸":"183x61cm","厚度":"10mm","材质":"NBR","颜色":"紫色"}`),
			PurchasePrice: 35.0, PurchaseCurrency: "CNY",
			PackageWeightKG: 0.8, PackageLengthCM: 65, PackageWidthCM: 18, PackageHeightCM: 18,
			TargetSalePrice: 16.99, TargetCurrency: "USD", DestinationCountry: "RU",
			HSCode: "9506.91",
		},
		{
			Title: "无线鼠标 静音双模", Description: "2.4G+蓝牙双模静音鼠标，适用于笔记本、平板和手机。续航6个月。",
			MainImage: "https://img.example.com/mouse.jpg",
			Images:    json.RawMessage(`["https://img.example.com/mouse-1.jpg","https://img.example.com/mouse-2.jpg"]`),
			Specs:     json.RawMessage(`{"连接":"2.4G+蓝牙5.0","按键":"静音6键","DPI":"800-2400","供电":"AA电池"}`),
			PurchasePrice: 22.0, PurchaseCurrency: "CNY",
			PackageWeightKG: 0.06, PackageLengthCM: 12, PackageWidthCM: 8, PackageHeightCM: 3,
			TargetSalePrice: 11.99, TargetCurrency: "USD", DestinationCountry: "RU",
			HSCode: "8471.60",
		},
		{
			Title: "不锈钢保温杯 500ml", Description: "双层不锈钢真空保温杯，12小时保温8小时保冷。食品级304不锈钢。",
			MainImage: "https://img.example.com/bottle.jpg",
			Images:    json.RawMessage(`["https://img.example.com/bt-1.jpg"]`),
			Specs:     json.RawMessage(`{"容量":"500ml","材质":"304不锈钢","保温":"12小时","颜色":"白色/黑色"}`),
			PurchasePrice: 20.0, PurchaseCurrency: "CNY",
			PackageWeightKG: 0.3, PackageLengthCM: 25, PackageWidthCM: 8, PackageHeightCM: 8,
			TargetSalePrice: 12.99, TargetCurrency: "USD", DestinationCountry: "RU",
			HSCode: "9617.00",
		},
		{
			Title: "手机支架 桌面可调节", Description: "可调节手机支架，兼容4-10英寸手机和平板。铝合金底座，防滑硅胶垫。",
			MainImage: "https://img.example.com/phone-stand.jpg",
			Images:    json.RawMessage(`["https://img.example.com/ps-1.jpg","https://img.example.com/ps-2.jpg"]`),
			Specs:     json.RawMessage(`{"材质":"铝合金+硅胶","兼容":"4-10英寸","调节":"角度+高度","颜色":"银色"}`),
			PurchasePrice: 15.0, PurchaseCurrency: "CNY",
			PackageWeightKG: 0.2, PackageLengthCM: 15, PackageWidthCM: 12, PackageHeightCM: 5,
			TargetSalePrice: 8.99, TargetCurrency: "USD", DestinationCountry: "RU",
			HSCode: "8302.50",
		},
		{
			Title: "儿童益智积木 100粒", Description: "大颗粒积木套装，适合3岁以上儿童。ABS环保塑料，圆角设计安全无毒。",
			MainImage: "https://img.example.com/blocks.jpg",
			Images:    json.RawMessage(`["https://img.example.com/bl-1.jpg","https://img.example.com/bl-2.jpg"]`),
			Specs:     json.RawMessage(`{"颗粒数":"100","适用年龄":"3岁以上","材质":"ABS","认证":"CE/EN71"}`),
			PurchasePrice: 28.0, PurchaseCurrency: "CNY",
			PackageWeightKG: 0.5, PackageLengthCM: 25, PackageWidthCM: 20, PackageHeightCM: 8,
			TargetSalePrice: 13.99, TargetCurrency: "USD", DestinationCountry: "RU",
			HSCode: "9503.00",
		},
		{
			Title: "USB-C 扩展坞 7合1", Description: "7合1 USB-C扩展坞，支持HDMI 4K输出、USB 3.0、SD卡读取、PD快充。",
			MainImage: "https://img.example.com/usb-hub.jpg",
			Images:    json.RawMessage(`["https://img.example.com/uh-1.jpg","https://img.example.com/uh-2.jpg","https://img.example.com/uh-3.jpg"]`),
			Specs:     json.RawMessage(`{"接口":"HDMI+USB3.0x3+SD+PD","HDMI":"4K@30Hz","材质":"铝合金","兼容":"MacBook/Windows"}`),
			PurchasePrice: 42.0, PurchaseCurrency: "CNY",
			PackageWeightKG: 0.08, PackageLengthCM: 12, PackageWidthCM: 5, PackageHeightCM: 2,
			TargetSalePrice: 22.99, TargetCurrency: "USD", DestinationCountry: "RU",
			HSCode: "8471.80",
		},
		// ===== Medium products (50-80%) =====
		{
			Title: "女士太阳镜 UV400防护", Description: "时尚太阳镜，UV400防护，轻量TR90材质镜框。",
			MainImage: "https://img.example.com/sunglasses.jpg",
			Images:    json.RawMessage(`["https://img.example.com/sg-1.jpg"]`),
			Specs:     json.RawMessage(`{"镜框":"TR90","防护":"UV400","颜色":"黑色"}`),
			PurchasePrice: 18.0, PurchaseCurrency: "CNY",
			PackageWeightKG: 0.03, PackageLengthCM: 16, PackageWidthCM: 6, PackageHeightCM: 4,
			TargetSalePrice: 9.99, TargetCurrency: "USD", DestinationCountry: "RU",
			// Missing: brand_id, hs_code
		},
		{
			Title: "家用电子秤 精准测脂",
			// Missing: description, specs
			MainImage: "https://img.example.com/scale.jpg",
			Specs:     json.RawMessage(`{"最大称重":"180kg"}`),
			PurchasePrice: 25.0, PurchaseCurrency: "CNY",
			// Missing: package dimensions, hs_code
			PackageWeightKG: 0.4,
			TargetSalePrice: 11.99, TargetCurrency: "USD", DestinationCountry: "RU",
		},
		{
			Title: "厨房计时器 磁性", Description: "磁性背贴厨房计时器，最大计时99分59秒。",
			// Missing: main_image, images, specs, hs_code
			MainImage: "",
			PurchasePrice: 8.0, PurchaseCurrency: "CNY",
			PackageWeightKG: 0.05, PackageLengthCM: 6, PackageWidthCM: 6, PackageHeightCM: 2,
			TargetSalePrice: 4.99, TargetCurrency: "USD", DestinationCountry: "RU",
		},
		{
			Title: "车载手机支架 出风口式",
			// Missing: main_image, description, specs, brand_id
			MainImage: "https://img.example.com/car-mount.jpg",
			PurchasePrice: 12.0, PurchaseCurrency: "CNY",
			PackageWeightKG: 0.08, PackageLengthCM: 10, PackageWidthCM: 8, PackageHeightCM: 5,
			TargetSalePrice: 5.99, TargetCurrency: "USD", DestinationCountry: "RU",
			HSCode: "8302.50",
		},
		{
			Title: "运动发带 吸汗速干",
			// Missing: main_image, description, specs, category_id, brand_id, hs_code
			MainImage: "",
			PurchasePrice: 3.0, PurchaseCurrency: "CNY",
			PackageWeightKG: 0.02, PackageLengthCM: 12, PackageWidthCM: 8, PackageHeightCM: 1,
			TargetSalePrice: 1.99, TargetCurrency: "USD", DestinationCountry: "RU",
		},
		// ===== Minimal products (< 50%) =====
		{
			Title: "DIY串珠手链材料包",
			// Missing: most required fields
			Description: "含200颗彩色珠子+弹力线",
			DestinationCountry: "RU",
		},
		{
			Title: "冰箱贴 创意冰箱装饰",
			// Only title + destination
			DestinationCountry: "RU",
		},
		{
			Title: "钥匙扣 个性定制",
			// Only title
		},
		{
			Title: "手机壳 透明防摔",
			// Only title
		},
		{
			Title: "书签 金属镂空",
			// Only title
		},
	}

	created := 0
	for _, p := range products {
		// Set defaults for empty required fields
		if p.Status == "" {
			p.Status = "draft"
		}
		if p.PurchaseCurrency == "" {
			p.PurchaseCurrency = "CNY"
		}
		if p.TargetCurrency == "" {
			p.TargetCurrency = "USD"
		}
		if p.DestinationCountry == "" {
			p.DestinationCountry = "RU"
		}

		prod, err := svc.Create(&p)
		if err != nil {
			log.Printf("WARN: failed to create product %q: %v", p.Title, err)
			continue
		}
		created++
		fmt.Printf("  ✓ [%d] %s\n", prod.ID, prod.Title)
	}

	fmt.Printf("Created %d candidate products (out of %d)\n", created, len(products))

	// Run completeness check on all created products
	checkSvc := completeness.NewService(db, zapLogger, svc)
	for i := 1; i <= created; i++ {
		result, err := checkSvc.Check(int64(i))
		if err != nil {
			log.Printf("WARN: completeness check failed for product %d: %v", i, err)
			continue
		}
		fmt.Printf("  Check #%d: score=%.0f%% missing=%d\n", i, result.Percentage, len(result.MissingItems))
	}
	fmt.Println("Seed complete.")
	os.Exit(0)
}
```

---

## Task 13: Compile check + test run

**No new files.** Run full verification.

- [ ] **Step 1: Compile check**

Run: `cd backend-go && go build ./...`
Expected: builds without errors

- [ ] **Step 2: Run candidate tests**

Run: `cd backend-go && go test -v ./internal/domain/candidate/`
Expected: All tests pass

- [ ] **Step 3: Run completeness tests**

Run: `cd backend-go && go test -v ./internal/domain/completeness/`
Expected: All tests pass

- [ ] **Step 4: Full test sweep**

Run: `cd backend-go && go test ./...`
Expected: All tests pass (baseline: 38 packages)

- [ ] **Step 5: Vet check**

Run: `cd backend-go && go vet ./...`
Expected: No output (pass)
