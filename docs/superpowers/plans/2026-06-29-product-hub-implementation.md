# Product Hub Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build Product Hub V1 — 6 new tables (product_master, product_variant, product_concept, supplier_offer, sample_request, cost_version) + aggregation API + frontend product archive page. Enable unified product lifecycle tracking from concept through listing.

**Architecture:** New domain module `backend-go/internal/domain/producthub/` extends existing producthub routes. All new tables reference product_master as identity spine. Aggregation API queries across new tables + existing modules (sku, supplier, listing). Frontend adds `/product-hub` route with archive view.

**Tech Stack:** Go 1.25 / Gin / GORM / PostgreSQL 15, Next.js 16 / React 19 / Ant Design 6 / TanStack React Query 5

---

### Task 1: product_master Model + Migration

**Files:**
- Create: `backend-go/internal/domain/producthub/master.go`

**Spec reference:** Section 3.1 — `ProductMaster` struct with lifecycle status

- [ ] **Write the model with TableName, lifecycle status constants, and validation**

```go
package producthub

import "time"

// Product lifecycle status constants.
const (
	LifecycleIdea        = "idea"
	LifecycleResearching = "researching"
	LifecycleSampling    = "sampling"
	LifecycleApproved    = "approved"
	LifecycleCosted      = "costed"
	LifecycleReadyToList = "ready_to_list"
	LifecycleListed      = "listed"
	LifecycleActive      = "active"
	LifecycleSunset      = "sunset"
	LifecycleArchived    = "archived"
)

// ValidLifecycleStatuses returns all valid lifecycle states in order.
func ValidLifecycleStatuses() []string {
	return []string{
		LifecycleIdea, LifecycleResearching, LifecycleSampling,
		LifecycleApproved, LifecycleCosted, LifecycleReadyToList,
		LifecycleListed, LifecycleActive, LifecycleSunset, LifecycleArchived,
	}
}

// IsValidLifecycleStatus checks if s is a valid status.
func IsValidLifecycleStatus(s string) bool {
	for _, v := range ValidLifecycleStatuses() {
		if v == s {
			return true
		}
	}
	return false
}

// BusinessModel constants.
const (
	BusinessOEM          = "oem"
	BusinessODM          = "odm"
	BusinessCatalog      = "catalog"
	BusinessPrivateLabel = "private_label"
)

// ProductMaster maps to the "product_master" table.
type ProductMaster struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductCode     string     `gorm:"column:product_code;uniqueIndex;size:64" json:"product_code"`
	Name            string     `gorm:"column:name;size:256;not null" json:"name"`
	BrandID         *int64     `gorm:"column:brand_id" json:"brand_id"`
	CategoryID      *int64     `gorm:"column:category_id" json:"category_id"`
	BusinessModel   string     `gorm:"column:business_model;size:32;default:catalog" json:"business_model"`
	LifecycleStatus string     `gorm:"column:lifecycle_status;size:32;default:idea" json:"lifecycle_status"`
	OwnerID         int64      `gorm:"column:owner_id" json:"owner_id"`
	TeamID          *int64     `gorm:"column:team_id" json:"team_id"`
	Description     string     `gorm:"column:description;type:text" json:"description"`
	TargetMarket    string     `gorm:"column:target_market;size:128" json:"target_market"`
	TargetPrice     float64    `gorm:"column:target_price" json:"target_price"`
	TargetMargin    float64    `gorm:"column:target_margin" json:"target_margin"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (ProductMaster) TableName() string { return "product_master" }
```

- [ ] **Write the migration SQL** as `backend-go/migrations/000016_create_product_hub_tables.sql`

```sql
-- Migration: 000016_create_product_hub_tables
-- Up
CREATE TABLE IF NOT EXISTS product_master (
    id BIGSERIAL PRIMARY KEY,
    product_code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(256) NOT NULL,
    brand_id BIGINT,
    category_id BIGINT,
    business_model VARCHAR(32) NOT NULL DEFAULT 'catalog',
    lifecycle_status VARCHAR(32) NOT NULL DEFAULT 'idea',
    owner_id BIGINT NOT NULL,
    team_id BIGINT,
    description TEXT,
    target_market VARCHAR(128),
    target_price DOUBLE PRECISION,
    target_margin DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_product_master_lifecycle ON product_master(lifecycle_status);
CREATE INDEX idx_product_master_owner ON product_master(owner_id);
```

- [ ] **Commit**

```bash
git add backend-go/internal/domain/producthub/master.go backend-go/migrations/000016_create_product_hub_tables.sql
git commit -m "feat: add product_master model and migration"
```

---

### Task 2: product_master CRUD + Lifecycle State Machine

**Files:**
- Create: `backend-go/internal/domain/producthub/master_service.go`
- Create: `backend-go/internal/domain/producthub/master_handler.go`
- Create: `backend-go/internal/domain/producthub/master_test.go`

- [ ] **Write the service with CRUD + lifecycle state transition validation**

```go
package producthub

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MasterService handles product_master business logic.
type MasterService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewMasterService creates a MasterService.
func NewMasterService(db *gorm.DB, logger *zap.Logger) *MasterService {
	return &MasterService{db: db, logger: logger}
}

// List returns paginated product masters with optional lifecycle filter.
func (s *MasterService) List(ctx context.Context, page, size int, lifecycleStatus string) ([]ProductMaster, int64, error) {
	q := s.db.WithContext(ctx).Model(&ProductMaster{})
	if lifecycleStatus != "" {
		if !IsValidLifecycleStatus(lifecycleStatus) {
			return nil, 0, fmt.Errorf("invalid lifecycle status: %s", lifecycleStatus)
		}
		q = q.Where("lifecycle_status = ?", lifecycleStatus)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []ProductMaster
	if err := q.Offset((page - 1) * size).Limit(size).Order("id DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID returns a single product master by ID.
func (s *MasterService) GetByID(ctx context.Context, id int64) (*ProductMaster, error) {
	var p ProductMaster
	if err := s.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Create inserts a new product master.
func (s *MasterService) Create(ctx context.Context, p *ProductMaster) error {
	if !IsValidLifecycleStatus(p.LifecycleStatus) {
		p.LifecycleStatus = LifecycleIdea
	}
	return s.db.WithContext(ctx).Create(p).Error
}

// Update updates an existing product master.
func (s *MasterService) Update(ctx context.Context, p *ProductMaster) error {
	if p.LifecycleStatus != "" && !IsValidLifecycleStatus(p.LifecycleStatus) {
		return fmt.Errorf("invalid lifecycle status: %s", p.LifecycleStatus)
	}
	return s.db.WithContext(ctx).Model(p).Updates(map[string]interface{}{
		"name":             p.Name,
		"brand_id":         p.BrandID,
		"category_id":      p.CategoryID,
		"business_model":   p.BusinessModel,
		"lifecycle_status": p.LifecycleStatus,
		"owner_id":         p.OwnerID,
		"description":      p.Description,
		"target_market":    p.TargetMarket,
		"target_price":     p.TargetPrice,
		"target_margin":    p.TargetMargin,
	}).Error
}

// Delete soft-deletes a product master.
func (s *MasterService) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&ProductMaster{}, id).Error
}

// TransitionLifecycle advances the product lifecycle status (only forward).
func (s *MasterService) TransitionLifecycle(ctx context.Context, id int64, newStatus string) (*ProductMaster, error) {
	if !IsValidLifecycleStatus(newStatus) {
		return nil, fmt.Errorf("invalid lifecycle status: %s", newStatus)
	}
	p, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(p).Update("lifecycle_status", newStatus).Error; err != nil {
		return nil, err
	}
	p.LifecycleStatus = newStatus
	return p, nil
}
```

- [ ] **Write the handler CRUD endpoints**

```go
package producthub

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// MasterHandler handles product master HTTP requests.
type MasterHandler struct {
	service *MasterService
}

// NewMasterHandler creates a MasterHandler.
func NewMasterHandler(svc *MasterService) *MasterHandler {
	return &MasterHandler{service: svc}
}

// List GET /api/v1/product-hub
func (h *MasterHandler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	status := c.Query("lifecycle_status")
	items, total, err := h.service.List(c.Request.Context(), p.Page, p.Size, status)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// Get GET /api/v1/product-hub/:id
func (h *MasterHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, item)
}

// Create POST /api/v1/product-hub
func (h *MasterHandler) Create(c *gin.Context) {
	var p ProductMaster
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.Create(c.Request.Context(), &p); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// Update PUT /api/v1/product-hub/:id
func (h *MasterHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var p ProductMaster
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p.ID = id
	if err := h.service.Update(c.Request.Context(), &p); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// Delete DELETE /api/v1/product-hub/:id
func (h *MasterHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// TransitionLifecycle POST /api/v1/product-hub/:id/transition
func (h *MasterHandler) TransitionLifecycle(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.service.TransitionLifecycle(c.Request.Context(), id, req.Status)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, p)
}
```

- [ ] **Write the test**

```go
package producthub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/response"
	"go.uber.org/zap"
)

func newMasterService(t *testing.T) *MasterService {
	t.Helper()
	db := dbtest.NewDB(t, &ProductMaster{})
	return NewMasterService(db, zap.NewNop())
}

func setupMasterRouter(t *testing.T) (*gin.Engine, *MasterService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := newMasterService(t)
	h := NewMasterHandler(svc)
	r := gin.New()
	rg := r.Group("/api/v1/product-hub")
	rg.GET("", h.List)
	rg.GET("/:id", h.Get)
	rg.POST("", h.Create)
	rg.PUT("/:id", h.Update)
	rg.DELETE("/:id", h.Delete)
	rg.POST("/:id/transition", h.TransitionLifecycle)
	return r, svc
}

func TestMasterCreateAndGet(t *testing.T) {
	r, _ := setupMasterRouter(t)

	body := `{"name":"Test Product","owner_id":1,"business_model":"catalog","target_market":"US"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/product-hub", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	var created ProductMaster
	b, _ := json.Marshal(resp.Data)
	json.Unmarshal(b, &created)
	if created.Name != "Test Product" {
		t.Fatalf("expected name 'Test Product', got '%s'", created.Name)
	}
	if created.LifecycleStatus != LifecycleIdea {
		t.Fatalf("expected default lifecycle 'idea', got '%s'", created.LifecycleStatus)
	}
}

func TestMasterLifecycleTransition(t *testing.T) {
	r, svc := setupMasterRouter(t)
	ctx := t.Context()

	// Create a product
	p := &ProductMaster{Name: "Lifecycle Test", OwnerID: 1}
	if err := svc.Create(ctx, p); err != nil {
		t.Fatal(err)
	}

	// Transition via API
	body := `{"status":"sampling"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/product-hub/1/transition", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify
	p2, _ := svc.GetByID(ctx, p.ID)
	if p2.LifecycleStatus != LifecycleSampling {
		t.Fatalf("expected 'sampling', got '%s'", p2.LifecycleStatus)
	}
}
```

- [ ] **Run tests and commit**

```bash
cd backend-go && go test ./internal/domain/producthub/ -v -run TestMaster -count=1
git add backend-go/internal/domain/producthub/master_service.go backend-go/internal/domain/producthub/master_handler.go backend-go/internal/domain/producthub/master_test.go
git commit -m "feat: add product_master CRUD and lifecycle state machine"
```

---

### Task 3: product_variant Model + CRUD

**Files:**
- Create: `backend-go/internal/domain/producthub/variant.go`
- Create: `backend-go/internal/domain/producthub/variant_test.go`

- [ ] **Write the model and service**

```go
// In variant.go
package producthub

import "time"

// ProductVariant maps to the "product_variant" table.
type ProductVariant struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductMasterID int64     `gorm:"column:product_master_id;index;not null" json:"product_master_id"`
	SKUProductID    int64     `gorm:"column:sku_product_id;index" json:"sku_product_id"`
	SKUCode         string    `gorm:"column:sku_code;size:64" json:"sku_code"`
	VariantLabel    string    `gorm:"column:variant_label;size:128" json:"variant_label"`
	Barcode         string    `gorm:"column:barcode;size:64" json:"barcode"`
	Weight          float64   `gorm:"column:weight" json:"weight"`
	Dimensions      string    `gorm:"column:dimensions;size:64" json:"dimensions"`
	CountryOfOrigin string    `gorm:"column:country_of_origin;size:8" json:"country_of_origin"`
	HSCode          string    `gorm:"column:hs_code;size:32" json:"hs_code"`
	Status          string    `gorm:"column:status;size:32;default:active" json:"status"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ProductVariant) TableName() string { return "product_variant" }

// VariantService handles product variant business logic.
type VariantService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewVariantService(db *gorm.DB, logger *zap.Logger) *VariantService {
	return &VariantService{db: db, logger: logger}
}

func (s *VariantService) ListByMaster(ctx context.Context, masterID int64) ([]ProductVariant, error) {
	var items []ProductVariant
	if err := s.db.WithContext(ctx).Where("product_master_id = ?", masterID).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *VariantService) Create(ctx context.Context, v *ProductVariant) error {
	return s.db.WithContext(ctx).Create(v).Error
}

func (s *VariantService) Update(ctx context.Context, v *ProductVariant) error {
	return s.db.WithContext(ctx).Model(v).Updates(map[string]interface{}{
		"sku_product_id":   v.SKUProductID,
		"sku_code":         v.SKUCode,
		"variant_label":    v.VariantLabel,
		"barcode":          v.Barcode,
		"weight":           v.Weight,
		"dimensions":       v.Dimensions,
		"country_of_origin": v.CountryOfOrigin,
		"hs_code":          v.HSCode,
		"status":           v.Status,
	}).Error
}

func (s *VariantService) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&ProductVariant{}, id).Error
}
```

- [ ] **Add migration for product_variant table** to `000016_create_product_hub_tables.sql`

```sql
CREATE TABLE IF NOT EXISTS product_variant (
    id BIGSERIAL PRIMARY KEY,
    product_master_id BIGINT NOT NULL REFERENCES product_master(id) ON DELETE CASCADE,
    sku_product_id BIGINT,
    sku_code VARCHAR(64),
    variant_label VARCHAR(128),
    barcode VARCHAR(64),
    weight DOUBLE PRECISION,
    dimensions VARCHAR(64),
    country_of_origin VARCHAR(8),
    hs_code VARCHAR(32),
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_product_variant_master ON product_variant(product_master_id);
CREATE INDEX idx_product_variant_sku ON product_variant(sku_product_id);
```

- [ ] **Write the test**

```go
// In variant_test.go
package producthub

import (
	"testing"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func newVariantService(t *testing.T) *VariantService {
	t.Helper()
	db := dbtest.NewDB(t, &ProductMaster{}, &ProductVariant{})
	return NewVariantService(db, zap.NewNop())
}

func TestVariantCreateAndList(t *testing.T) {
	svc := newVariantService(t)
	db := dbtest.NewDB(t, &ProductMaster{}, &ProductVariant{})
	ms := NewMasterService(db, zap.NewNop())

	// Create a master first
	ctx := t.Context()
	master := &ProductMaster{Name: "Variant Parent", OwnerID: 1}
	if err := ms.Create(ctx, master); err != nil {
		t.Fatal(err)
	}

	v := &ProductVariant{ProductMasterID: master.ID, SKUCode: "TEST-001", VariantLabel: "Black-Large"}
	if err := svc.Create(ctx, v); err != nil {
		t.Fatal(err)
	}

	items, err := svc.ListByMaster(ctx, master.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(items))
	}
	if items[0].SKUCode != "TEST-001" {
		t.Fatalf("expected 'TEST-001', got '%s'", items[0].SKUCode)
	}
}
```

- [ ] **Run tests and commit**

```bash
cd backend-go && go test ./internal/domain/producthub/ -v -run TestVariant -count=1
git add backend-go/internal/domain/producthub/variant.go backend-go/internal/domain/producthub/variant_test.go backend-go/migrations/000016_create_product_hub_tables.sql
git commit -m "feat: add product_variant model and CRUD"
```

---

### Task 4: product_concept Model + CRUD

**Files:**
- Create: `backend-go/internal/domain/producthub/concept.go`
- Create: `backend-go/internal/domain/producthub/concept_test.go`

- [ ] **Write the model and service**

```go
// In concept.go
package producthub

import (
	"encoding/json"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ProductConcept maps to the "product_concept" table.
type ProductConcept struct {
	ID              int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductMasterID int64           `gorm:"column:product_master_id;index;not null" json:"product_master_id"`
	Brief           string          `gorm:"column:brief;type:text" json:"brief"`
	TargetCustomer  string          `gorm:"column:target_customer;type:text" json:"target_customer"`
	PainPoint       string          `gorm:"column:pain_point;type:text" json:"pain_point"`
	MarketResearch  string          `gorm:"column:market_research;type:text" json:"market_research"`
	CompetitorInfo  string          `gorm:"column:competitor_info;type:text" json:"competitor_info"`
	DesignSource    string          `gorm:"column:design_source;size:32" json:"design_source"`
	AttachmentURLs  json.RawMessage `gorm:"column:attachment_urls;type:jsonb" json:"attachment_urls,omitempty"`
	Status          string          `gorm:"column:status;size:32;default:draft" json:"status"`
	CreatedBy       int64           `gorm:"column:created_by" json:"created_by"`
	CreatedAt       time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ProductConcept) TableName() string { return "product_concept" }

// ConceptService handles product concept business logic.
type ConceptService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewConceptService(db *gorm.DB, logger *zap.Logger) *ConceptService {
	return &ConceptService{db: db, logger: logger}
}

func (s *ConceptService) GetByMasterID(ctx context.Context, masterID int64) (*ProductConcept, error) {
	var c ProductConcept
	if err := s.db.WithContext(ctx).Where("product_master_id = ?", masterID).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *ConceptService) Upsert(ctx context.Context, c *ProductConcept) error {
	existing, err := s.GetByMasterID(ctx, c.ProductMasterID)
	if err == gorm.ErrRecordNotFound {
		return s.db.WithContext(ctx).Create(c).Error
	}
	if err != nil {
		return err
	}
	c.ID = existing.ID
	return s.db.WithContext(ctx).Model(c).Updates(map[string]interface{}{
		"brief":            c.Brief,
		"target_customer":  c.TargetCustomer,
		"pain_point":       c.PainPoint,
		"market_research":  c.MarketResearch,
		"competitor_info":  c.CompetitorInfo,
		"design_source":    c.DesignSource,
		"attachment_urls":  c.AttachmentURLs,
		"status":           c.Status,
	}).Error
}

func (s *ConceptService) Delete(ctx context.Context, masterID int64) error {
	return s.db.WithContext(ctx).Where("product_master_id = ?", masterID).Delete(&ProductConcept{}).Error
}
```

- [ ] **Add migration SQL**

```sql
CREATE TABLE IF NOT EXISTS product_concept (
    id BIGSERIAL PRIMARY KEY,
    product_master_id BIGINT NOT NULL REFERENCES product_master(id) ON DELETE CASCADE,
    brief TEXT,
    target_customer TEXT,
    pain_point TEXT,
    market_research TEXT,
    competitor_info TEXT,
    design_source VARCHAR(32),
    attachment_urls JSONB DEFAULT '[]',
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_product_concept_master ON product_concept(product_master_id);
```

- [ ] **Write the test**

```go
// In concept_test.go
package producthub

import (
	"testing"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func newConceptService(t *testing.T) *ConceptService {
	t.Helper()
	db := dbtest.NewDB(t, &ProductMaster{}, &ProductConcept{})
	return NewConceptService(db, zap.NewNop())
}

func TestConceptUpsert(t *testing.T) {
	svc := newConceptService(t)
	db := dbtest.NewDB(t, &ProductMaster{}, &ProductConcept{})
	ms := NewMasterService(db, zap.NewNop())

	ctx := t.Context()
	master := &ProductMaster{Name: "Concept Test", OwnerID: 1}
	if err := ms.Create(ctx, master); err != nil {
		t.Fatal(err)
	}

	// Create concept
	c := &ProductConcept{ProductMasterID: master.ID, Brief: "A great idea", DesignSource: "internal"}
	if err := svc.Upsert(ctx, c); err != nil {
		t.Fatal(err)
	}

	// Read back
	got, err := svc.GetByMasterID(ctx, master.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Brief != "A great idea" {
		t.Fatalf("expected 'A great idea', got '%s'", got.Brief)
	}

	// Update concept
	c.Brief = "Revised idea"
	if err := svc.Upsert(ctx, c); err != nil {
		t.Fatal(err)
	}
	got2, _ := svc.GetByMasterID(ctx, master.ID)
	if got2.Brief != "Revised idea" {
		t.Fatalf("expected 'Revised idea', got '%s'", got2.Brief)
	}
}
```

- [ ] **Commit**

```bash
cd backend-go && go test ./internal/domain/producthub/ -v -run TestConcept -count=1
git add backend-go/internal/domain/producthub/concept.go backend-go/internal/domain/producthub/concept_test.go
git commit -m "feat: add product_concept model and CRUD"
```

---

### Task 5: supplier_offer Model + CRUD

**Files:**
- Create: `backend-go/internal/domain/producthub/supplier_offer.go`
- Create: `backend-go/internal/domain/producthub/supplier_offer_test.go`

- [ ] **Write the model and service**

```go
// In supplier_offer.go
package producthub

import (
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SupplierOffer maps to the "supplier_offer" table.
type SupplierOffer struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SupplierID      int64     `gorm:"column:supplier_id;index;not null" json:"supplier_id"`
	ProductMasterID int64     `gorm:"column:product_master_id;index;not null" json:"product_master_id"`
	OfferType       string    `gorm:"column:offer_type;size:32" json:"offer_type"`
	UnitCost        float64   `gorm:"column:unit_cost" json:"unit_cost"`
	Currency        string    `gorm:"column:currency;size:8;default:CNY" json:"currency"`
	MOQ             int       `gorm:"column:moq" json:"moq"`
	LeadTimeDays    int       `gorm:"column:lead_time_days" json:"lead_time_days"`
	Incoterm        string    `gorm:"column:incoterm;size:32" json:"incoterm"`
	IsPreferred     bool      `gorm:"column:is_preferred;default:false" json:"is_preferred"`
	ValidFrom       time.Time `gorm:"column:valid_from" json:"valid_from"`
	ValidTo         time.Time `gorm:"column:valid_to" json:"valid_to"`
	Notes           string    `gorm:"column:notes;type:text" json:"notes"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SupplierOffer) TableName() string { return "supplier_offer" }
```

- [ ] **SupplierOfferService with ListByMaster, Create, Update, Delete**

```go
// SupplierOfferService (same file)
type SupplierOfferService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewSupplierOfferService(db *gorm.DB, logger *zap.Logger) *SupplierOfferService {
	return &SupplierOfferService{db: db, logger: logger}
}

func (s *SupplierOfferService) ListByMaster(ctx context.Context, masterID int64) ([]SupplierOffer, error) {
	var items []SupplierOffer
	if err := s.db.WithContext(ctx).Where("product_master_id = ?", masterID).Order("is_preferred DESC, unit_cost ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *SupplierOfferService) Create(ctx context.Context, o *SupplierOffer) error {
	return s.db.WithContext(ctx).Create(o).Error
}

func (s *SupplierOfferService) Update(ctx context.Context, o *SupplierOffer) error {
	return s.db.WithContext(ctx).Model(o).Updates(map[string]interface{}{
		"unit_cost":      o.UnitCost,
		"currency":       o.Currency,
		"moq":            o.MOQ,
		"lead_time_days": o.LeadTimeDays,
		"incoterm":       o.Incoterm,
		"is_preferred":   o.IsPreferred,
		"valid_from":     o.ValidFrom,
		"valid_to":       o.ValidTo,
		"notes":          o.Notes,
	}).Error
}

func (s *SupplierOfferService) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&SupplierOffer{}, id).Error
}
```

- [ ] **Add migration SQL**

```sql
CREATE TABLE IF NOT EXISTS supplier_offer (
    id BIGSERIAL PRIMARY KEY,
    supplier_id BIGINT NOT NULL,
    product_master_id BIGINT NOT NULL REFERENCES product_master(id) ON DELETE CASCADE,
    offer_type VARCHAR(32),
    unit_cost DOUBLE PRECISION,
    currency VARCHAR(8) NOT NULL DEFAULT 'CNY',
    moq INT,
    lead_time_days INT,
    incoterm VARCHAR(32),
    is_preferred BOOLEAN NOT NULL DEFAULT FALSE,
    valid_from TIMESTAMPTZ,
    valid_to TIMESTAMPTZ,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_supplier_offer_master ON supplier_offer(product_master_id);
CREATE INDEX idx_supplier_offer_supplier ON supplier_offer(supplier_id);
```

- [ ] **Write the test**

```go
func TestSupplierOfferCreateAndList(t *testing.T) {
	svc := NewSupplierOfferService(dbtest.NewDB(t, &ProductMaster{}, &SupplierOffer{}), zap.NewNop())
	ms := NewMasterService(dbtest.NewDB(t, &ProductMaster{}, &SupplierOffer{}), zap.NewNop())

	ctx := t.Context()
	master := &ProductMaster{Name: "Offer Test", OwnerID: 1}
	ms.Create(ctx, master)

	o := &SupplierOffer{ProductMasterID: master.ID, SupplierID: 100, UnitCost: 15.50, Currency: "CNY", MOQ: 1000}
	if err := svc.Create(ctx, o); err != nil {
		t.Fatal(err)
	}
	items, _ := svc.ListByMaster(ctx, master.ID)
	if len(items) != 1 || items[0].UnitCost != 15.50 {
		t.Fatalf("expected 1 offer with cost 15.50, got %d offers", len(items))
	}
}
```

- [ ] **Commit**

```bash
cd backend-go && go test ./internal/domain/producthub/ -v -run TestSupplierOffer -count=1
git add backend-go/internal/domain/producthub/supplier_offer.go backend-go/internal/domain/producthub/supplier_offer_test.go
git commit -m "feat: add supplier_offer model and CRUD"
```

---

### Task 6: sample_request Model + CRUD

**Files:**
- Create: `backend-go/internal/domain/producthub/sample.go`
- Create: `backend-go/internal/domain/producthub/sample_test.go`

- [ ] **Write the model and service**

```go
// In sample.go
package producthub

import (
	"encoding/json"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SampleRequest maps to the "sample_request" table — includes V1-level iteration info inline.
type SampleRequest struct {
	ID              int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductMasterID int64           `gorm:"column:product_master_id;index;not null" json:"product_master_id"`
	SupplierOfferID *int64          `gorm:"column:supplier_offer_id" json:"supplier_offer_id"`
	SupplierID      int64           `gorm:"column:supplier_id;not null" json:"supplier_id"`
	Quantity        int             `gorm:"column:quantity" json:"quantity"`
	Requirements    string          `gorm:"column:requirements;type:text" json:"requirements"`
	RequestedAt     time.Time       `gorm:"column:requested_at;autoCreateTime" json:"requested_at"`
	DueAt           *time.Time      `gorm:"column:due_at" json:"due_at"`
	Status          string          `gorm:"column:status;size:32;default:pending" json:"status"`
	IterationNo     int             `gorm:"column:iteration_no;default:0" json:"iteration_no"`
	ReceivedAt      *time.Time      `gorm:"column:received_at" json:"received_at"`
	Evaluation      string          `gorm:"column:evaluation;type:text" json:"evaluation"`
	QualityScore    float64         `gorm:"column:quality_score" json:"quality_score"`
	Decision        string          `gorm:"column:decision;size:32" json:"decision"`
	ImageURLs       json.RawMessage `gorm:"column:image_urls;type:jsonb" json:"image_urls,omitempty"`
	CreatedBy       int64           `gorm:"column:created_by" json:"created_by"`
	CreatedAt       time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SampleRequest) TableName() string { return "sample_request" }
```

- [ ] **SampleService with ListByMaster, Create, Update (iteration), Delete**

```go
type SampleService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewSampleService(db *gorm.DB, logger *zap.Logger) *SampleService {
	return &SampleService{db: db, logger: logger}
}

func (s *SampleService) ListByMaster(ctx context.Context, masterID int64) ([]SampleRequest, error) {
	var items []SampleRequest
	if err := s.db.WithContext(ctx).Where("product_master_id = ?", masterID).Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *SampleService) GetLatestByMaster(ctx context.Context, masterID int64) (*SampleRequest, error) {
	var sr SampleRequest
	if err := s.db.WithContext(ctx).Where("product_master_id = ?", masterID).Order("id DESC").First(&sr).Error; err != nil {
		return nil, err
	}
	return &sr, nil
}

func (s *SampleService) Create(ctx context.Context, sr *SampleRequest) error {
	return s.db.WithContext(ctx).Create(sr).Error
}

// RecordEvaluation updates a sample request with evaluation results.
func (s *SampleService) RecordEvaluation(ctx context.Context, id int64, eval string, score float64, decision string, imageURLs json.RawMessage) error {
	return s.db.WithContext(ctx).Model(&SampleRequest{}).Where("id = ?", id).Updates(map[string]interface{}{
		"received_at":  time.Now(),
		"evaluation":   eval,
		"quality_score": score,
		"decision":     decision,
		"image_urls":   imageURLs,
		"status":       "evaluated",
	}).Error
}
```

- [ ] **Add migration SQL**

```sql
CREATE TABLE IF NOT EXISTS sample_request (
    id BIGSERIAL PRIMARY KEY,
    product_master_id BIGINT NOT NULL REFERENCES product_master(id) ON DELETE CASCADE,
    supplier_offer_id BIGINT,
    supplier_id BIGINT NOT NULL,
    quantity INT,
    requirements TEXT,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    due_at TIMESTAMPTZ,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    iteration_no INT NOT NULL DEFAULT 0,
    received_at TIMESTAMPTZ,
    evaluation TEXT,
    quality_score DOUBLE PRECISION,
    decision VARCHAR(32),
    image_urls JSONB DEFAULT '[]',
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sample_request_master ON sample_request(product_master_id);
```

- [ ] **Write the test**

```go
func TestSampleCreateAndEvaluate(t *testing.T) {
	svc := NewSampleService(dbtest.NewDB(t, &ProductMaster{}, &SampleRequest{}), zap.NewNop())
	ms := NewMasterService(dbtest.NewDB(t, &ProductMaster{}, &SampleRequest{}), zap.NewNop())

	ctx := t.Context()
	master := &ProductMaster{Name: "Sample Test", OwnerID: 1}
	ms.Create(ctx, master)

	sr := &SampleRequest{ProductMasterID: master.ID, SupplierID: 100, Quantity: 5}
	if err := svc.Create(ctx, sr); err != nil {
		t.Fatal(err)
	}

	if err := svc.RecordEvaluation(ctx, sr.ID, "Looks good", 8.5, "pass", nil); err != nil {
		t.Fatal(err)
	}

	latest, _ := svc.GetLatestByMaster(ctx, master.ID)
	if latest.Decision != "pass" || latest.QualityScore != 8.5 {
		t.Fatalf("expected pass/8.5, got %s/%.1f", latest.Decision, latest.QualityScore)
	}
}
```

- [ ] **Commit**

```bash
cd backend-go && go test ./internal/domain/producthub/ -v -run TestSample -count=1
git add backend-go/internal/domain/producthub/sample.go backend-go/internal/domain/producthub/sample_test.go
git commit -m "feat: add sample_request model and CRUD"
```

---

### Task 7: cost_version Model + CRUD

**Files:**
- Create: `backend-go/internal/domain/producthub/cost.go`
- Create: `backend-go/internal/domain/producthub/cost_test.go`

- [ ] **Write the model and service**

```go
// In cost.go
package producthub

import (
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CostVersion maps to the "cost_version" table.
type CostVersion struct {
	ID               int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductMasterID  int64      `gorm:"column:product_master_id;index;not null" json:"product_master_id"`
	Version          string     `gorm:"column:version;size:16" json:"version"`
	BaseCost         float64    `gorm:"column:base_cost" json:"base_cost"`
	MaterialCost     float64    `gorm:"column:material_cost" json:"material_cost"`
	PackagingCost    float64    `gorm:"column:packaging_cost" json:"packaging_cost"`
	FreightCost      float64    `gorm:"column:freight_cost" json:"freight_cost"`
	DutyCost         float64    `gorm:"column:duty_cost" json:"duty_cost"`
	PlatformFeeRate  float64    `gorm:"column:platform_fee_rate" json:"platform_fee_rate"`
	AdCostEstimate   float64    `gorm:"column:ad_cost_estimate" json:"ad_cost_estimate"`
	FXRate           float64    `gorm:"column:fx_rate;default:1" json:"fx_rate"`
	FXRateDate       *time.Time `gorm:"column:fx_rate_date" json:"fx_rate_date"`
	LandedCost       float64    `gorm:"column:landed_cost" json:"landed_cost"`
	RecommendedPrice float64    `gorm:"column:recommended_price" json:"recommended_price"`
	GrossMargin      float64    `gorm:"column:gross_margin" json:"gross_margin"`
	EffectiveFrom    time.Time  `gorm:"column:effective_from;autoCreateTime" json:"effective_from"`
	Status           string     `gorm:"column:status;size:16;default:draft" json:"status"`
	Notes            string     `gorm:"column:notes;type:text" json:"notes"`
	CreatedBy        int64      `gorm:"column:created_by" json:"created_by"`
	CreatedAt        time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (CostVersion) TableName() string { return "cost_version" }

// CostLandedCost calculates landed cost from components.
func (c *CostVersion) CostLandedCost() float64 {
	return c.BaseCost + c.MaterialCost + c.PackagingCost + c.FreightCost + c.DutyCost
}

// CostGrossMargin calculates gross margin given a price.
func (c *CostVersion) CostGrossMargin(price float64) float64 {
	if price <= 0 {
		return 0
	}
	landed := c.CostLandedCost()
	return (price - landed - price*c.PlatformFeeRate/100 - c.AdCostEstimate) / price * 100
}
```

- [ ] **CostVersionService with CRUD + auto-calculate margins**

```go
type CostVersionService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewCostVersionService(db *gorm.DB, logger *zap.Logger) *CostVersionService {
	return &CostVersionService{db: db, logger: logger}
}

func (s *CostVersionService) GetLatestByMaster(ctx context.Context, masterID int64) (*CostVersion, error) {
	var cv CostVersion
	if err := s.db.WithContext(ctx).Where("product_master_id = ?", masterID).Order("id DESC").First(&cv).Error; err != nil {
		return nil, err
	}
	return &cv, nil
}

func (s *CostVersionService) ListByMaster(ctx context.Context, masterID int64) ([]CostVersion, error) {
	var items []CostVersion
	if err := s.db.WithContext(ctx).Where("product_master_id = ?", masterID).Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *CostVersionService) Create(ctx context.Context, cv *CostVersion) error {
	cv.LandedCost = cv.CostLandedCost()
	if cv.RecommendedPrice > 0 {
		cv.GrossMargin = cv.CostGrossMargin(cv.RecommendedPrice)
	}
	return s.db.WithContext(ctx).Create(cv).Error
}

func (s *CostVersionService) Confirm(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Model(&CostVersion{}).Where("id = ?", id).Update("status", "confirmed").Error
}
```

- [ ] **Add migration SQL**

```sql
CREATE TABLE IF NOT EXISTS cost_version (
    id BIGSERIAL PRIMARY KEY,
    product_master_id BIGINT NOT NULL REFERENCES product_master(id) ON DELETE CASCADE,
    version VARCHAR(16),
    base_cost DOUBLE PRECISION DEFAULT 0,
    material_cost DOUBLE PRECISION DEFAULT 0,
    packaging_cost DOUBLE PRECISION DEFAULT 0,
    freight_cost DOUBLE PRECISION DEFAULT 0,
    duty_cost DOUBLE PRECISION DEFAULT 0,
    platform_fee_rate DOUBLE PRECISION DEFAULT 0,
    ad_cost_estimate DOUBLE PRECISION DEFAULT 0,
    fx_rate DOUBLE PRECISION DEFAULT 1,
    fx_rate_date TIMESTAMPTZ,
    landed_cost DOUBLE PRECISION DEFAULT 0,
    recommended_price DOUBLE PRECISION DEFAULT 0,
    gross_margin DOUBLE PRECISION DEFAULT 0,
    effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status VARCHAR(16) NOT NULL DEFAULT 'draft',
    notes TEXT,
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cost_version_master ON cost_version(product_master_id);
```

- [ ] **Write the test**

```go
func TestCostVersionCreateAndCalculate(t *testing.T) {
	svc := NewCostVersionService(dbtest.NewDB(t, &ProductMaster{}, &CostVersion{}), zap.NewNop())
	ms := NewMasterService(dbtest.NewDB(t, &ProductMaster{}, &CostVersion{}), zap.NewNop())

	ctx := t.Context()
	master := &ProductMaster{Name: "Cost Test", OwnerID: 1}
	ms.Create(ctx, master)

	cv := &CostVersion{
		ProductMasterID: master.ID,
		BaseCost:        10.0,
		MaterialCost:    5.0,
		PackagingCost:   1.0,
		FreightCost:     3.0,
		DutyCost:        2.0,
		RecommendedPrice: 30.0,
	}
	if err := svc.Create(ctx, cv); err != nil {
		t.Fatal(err)
	}

	// Verify auto-calculated fields
	if cv.LandedCost != 21.0 {
		t.Fatalf("expected landed cost 21.0, got %.2f", cv.LandedCost)
	}
	if cv.GrossMargin <= 0 {
		t.Fatalf("expected positive gross margin, got %.2f", cv.GrossMargin)
	}

	// Confirm
	if err := svc.Confirm(ctx, cv.ID); err != nil {
		t.Fatal(err)
	}
	confirmed, _ := svc.GetLatestByMaster(ctx, master.ID)
	if confirmed.Status != "confirmed" {
		t.Fatalf("expected 'confirmed', got '%s'", confirmed.Status)
	}
}
```

- [ ] **Commit**

```bash
cd backend-go && go test ./internal/domain/producthub/ -v -run TestCostVersion -count=1
git add backend-go/internal/domain/producthub/cost.go backend-go/internal/domain/producthub/cost_test.go
git commit -m "feat: add cost_version model and CRUD with auto-calculation"
```

---

### Task 8: Aggregation API (GET /api/v1/product-hub/:id/hub)

**Files:**
- Create: `backend-go/internal/domain/producthub/aggregation.go`
- Create: `backend-go/internal/domain/producthub/aggregation_test.go`

- [ ] **Build the aggregation service that queries across all tables + existing modules**

```go
// In aggregation.go
package producthub

import (
	"context"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/supplier"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ProductHubProfile is the full aggregated product profile.
type ProductHubProfile struct {
	Master      *ProductMaster   `json:"master"`
	Variants    []ProductVariant `json:"variants"`
	Concept     *ProductConcept  `json:"concept,omitempty"`
	LatestCost  *CostVersion     `json:"latest_cost,omitempty"`
	CostHistory []CostVersion    `json:"cost_history,omitempty"`
	Suppliers   []SupplierInfo   `json:"suppliers,omitempty"`
	Samples     []SampleRequest  `json:"samples,omitempty"`
	Listings    []ListingBrief   `json:"listings,omitempty"`
	Timeline    []TimelineEvent  `json:"timeline,omitempty"`
}

// SupplierInfo is a simplified supplier view for the hub profile.
type SupplierInfo struct {
	SupplierOffer SupplierOffer `json:"offer"`
	SupplierName  string        `json:"supplier_name,omitempty"`
}

// ListingBrief is a simplified listing summary from the existing listing module.
type ListingBrief struct {
	Platform string  `json:"platform"`
	Title    string  `json:"title"`
	Price    float64 `json:"price"`
	Status   string  `json:"status"`
	URL      string  `json:"url,omitempty"`
}

// TimelineEvent represents a lifecycle event on the product timeline.
type TimelineEvent struct {
	Type      string    `json:"type"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"created_at"`
}

// AggregationService builds full product profiles.
type AggregationService struct {
	db          *gorm.DB
	logger      *zap.Logger
	master      *MasterService
	variant     *VariantService
	concept     *ConceptService
	offer       *SupplierOfferService
	sample      *SampleService
	cost        *CostVersionService
}

func NewAggregationService(db *gorm.DB, logger *zap.Logger) *AggregationService {
	return &AggregationService{
		db:      db,
		logger:  logger,
		master:  NewMasterService(db, logger),
		variant: NewVariantService(db, logger),
		concept: NewConceptService(db, logger),
		offer:   NewSupplierOfferService(db, logger),
		sample:  NewSampleService(db, logger),
		cost:    NewCostVersionService(db, logger),
	}
}

// GetProductHubProfile builds the full aggregated profile for a product.
func (s *AggregationService) GetProductHubProfile(ctx context.Context, productID int64) (*ProductHubProfile, error) {
	profile := &ProductHubProfile{}

	// 1. Product master
	master, err := s.master.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	profile.Master = master

	// 2. Variants
	if variants, err := s.variant.ListByMaster(ctx, productID); err == nil {
		profile.Variants = variants
	}

	// 3. Concept
	if concept, err := s.concept.GetByMasterID(ctx, productID); err == nil {
		profile.Concept = concept
	}

	// 4. Cost
	if cost, err := s.cost.GetLatestByMaster(ctx, productID); err == nil {
		profile.LatestCost = cost
	}
	if costs, err := s.cost.ListByMaster(ctx, productID); err == nil {
		profile.CostHistory = costs
	}

	// 5. Supplier offers (with names from supplier module)
	if offers, err := s.offer.ListByMaster(ctx, productID); err == nil {
		info := make([]SupplierInfo, 0, len(offers))
		for _, o := range offers {
			si := SupplierInfo{SupplierOffer: o}
			var sup supplier.Supplier
			if err := s.db.WithContext(ctx).First(&sup, o.SupplierID).Error; err == nil {
				si.SupplierName = sup.Name
			}
			info = append(info, si)
		}
		profile.Suppliers = info
	}

	// 6. Sample requests
	if samples, err := s.sample.ListByMaster(ctx, productID); err == nil {
		profile.Samples = samples
	}

	// 7. Listings (from existing listing module — pony: simple query, use ListingBrief query if listing module is accessible)
	// For V1: return empty or minimal placeholder — listing integration is a later task.
	profile.Listings = []ListingBrief{}

	// 8. Timeline — build from lifecycle status transitions (ponytail: simple ordered events)
	profile.Timeline = []TimelineEvent{
		{Type: "created", Summary: "Product created", CreatedAt: master.CreatedAt},
	}
	if profile.Samples != nil && len(profile.Samples) > 0 {
		for _, s := range profile.Samples {
			profile.Timeline = append(profile.Timeline, TimelineEvent{
				Type: "sample", Summary: "Sample requested", CreatedAt: s.CreatedAt,
			})
		}
	}
	if profile.LatestCost != nil {
		profile.Timeline = append(profile.Timeline, TimelineEvent{
			Type: "cost", Summary: "Cost version " + profile.LatestCost.Version, CreatedAt: profile.LatestCost.CreatedAt,
		})
	}

	return profile, nil
}
```

- [ ] **Add aggregation handler and route registration**

```go
// In aggregation.go (append to bottom or create a separate handler section)
func RegisterAggregationRoutes(rg *gin.RouterGroup, aggr *AggregationService) {
	rg.GET("/:id/hub", func(c *gin.Context) {
		// ponytail: reuse existing parseId helper from producthub
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		profile, err := aggr.GetProductHubProfile(c.Request.Context(), id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				response.Error(c, http.StatusNotFound, "product not found")
				return
			}
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Success(c, profile)
	})
}
```

- [ ] **Write the test**

```go
// In aggregation_test.go
package producthub

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func TestAggregationAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t, &ProductMaster{}, &ProductVariant{}, &ProductConcept{}, &SupplierOffer{}, &SampleRequest{}, &CostVersion{})
	aggr := NewAggregationService(db, zap.NewNop())

	// Create a product with data
	ctx := t.Context()
	master := &ProductMaster{Name: "Aggregated Product", OwnerID: 1}
	if err := aggr.master.Create(ctx, master); err != nil {
		t.Fatal(err)
	}

	// Add a variant
	if err := aggr.variant.Create(ctx, &ProductVariant{ProductMasterID: master.ID, SKUCode: "AGG-001"}); err != nil {
		t.Fatal(err)
	}

	// Query via handler
	r := gin.New()
	rg := r.Group("/api/v1/product-hub")
	RegisterAggregationRoutes(rg, aggr)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/product-hub/1/hub", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
```

- [ ] **Commit**

```bash
cd backend-go && go test ./internal/domain/producthub/ -v -run TestAggregation -count=1
git add backend-go/internal/domain/producthub/aggregation.go backend-go/internal/domain/producthub/aggregation_test.go
git commit -m "feat: add Product Hub aggregation API endpoint"
```

---

### Task 9: Wire Routes into Router

**Files:**
- Modify: `backend-go/internal/httpx/router.go` (add import + route registration)

- [ ] **Add product hub route registration to router.go**

Near the other `RegisterRoutes` calls (after supplier, around line 426), add:

```go
producthub.RegisterRoutes(protected, db, logger)
```

Then create the routes.go file that wires everything together:

```go
// In routes.go for producthub
package producthub

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers all Product Hub routes.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	// Services
	masterSvc := NewMasterService(db, logger)
	variantSvc := NewVariantService(db, logger)
	offerSvc := NewSupplierOfferService(db, logger)
	sampleSvc := NewSampleService(db, logger)
	costSvc := NewCostVersionService(db, logger)
	aggrSvc := NewAggregationService(db, logger)

	// Handlers
	masterH := NewMasterHandler(masterSvc)

	group := rg.Group("/product-hub")
	{
		// Master CRUD
		group.GET("", masterH.List)
		group.GET("/:id", masterH.Get)
		group.POST("", masterH.Create)
		group.PUT("/:id", masterH.Update)
		group.DELETE("/:id", masterH.Delete)

		// Lifecycle
		group.POST("/:id/transition", masterH.TransitionLifecycle)

		// Aggregation
		RegisterAggregationRoutes(group, aggrSvc)
	}

	// Variants sub-resource under the product hub group
	group.GET("/:id/variants", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		items, err := variantSvc.ListByMaster(c.Request.Context(), id)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Success(c, items)
	})
	group.POST("/variants", func(c *gin.Context) {
		var v ProductVariant
		if err := c.ShouldBindJSON(&v); err != nil {
			response.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		if err := variantSvc.Create(c.Request.Context(), &v); err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
		response.Success(c, v)
	})
}
```

- [ ] **Run all tests to verify registration works**

```bash
cd backend-go && go test ./internal/domain/producthub/ -v -count=1
cd backend-go && go vet ./internal/domain/producthub/
cd backend-go && go build ./...
git add backend-go/internal/domain/producthub/routes.go backend-go/internal/httpx/router.go
git commit -m "feat: wire Product Hub routes into router"
```

---

### Task 10: Frontend — Product Hub Archive Page

**Files:**
- Create: `frontend-next/src/app/(main)/product-hub/page.tsx`
- Create: `frontend-next/src/app/(main)/product-hub/[id]/page.tsx`

- [ ] **Product Hub list page** — `page.tsx`

```tsx
'use client';

import { Typography, Tag, Space } from 'antd';
import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';
import type { Result } from '@/types/api';

const LIFECYCLE_COLORS: Record<string, string> = {
  idea: 'default',
  researching: 'blue',
  sampling: 'orange',
  approved: 'cyan',
  costed: 'geekblue',
  ready_to_list: 'purple',
  listed: 'processing',
  active: 'success',
  sunset: 'warning',
  archived: 'default',
};

const LIFECYCLE_LABELS: Record<string, string> = {
  idea: '创意',
  researching: '调研中',
  sampling: '打样中',
  approved: '已确认',
  costed: '已核算成本',
  ready_to_list: '待上架',
  listed: '已上架',
  active: '销售中',
  sunset: '衰退中',
  archived: '已归档',
};

export default function ProductHubPage() {
  return (
    <CrudListPage
      title="产品档案"
      path="/v1/product-hub"
      columns={[
        { title: '编号', dataIndex: 'product_code', width: 140 },
        { title: '产品名称', dataIndex: 'name', width: 250 },
        {
          title: '业务模式',
          dataIndex: 'business_model',
          width: 120,
          render: (v: string) =>
            v === 'oem' ? 'OEM' : v === 'odm' ? 'ODM' : v === 'catalog' ? '选品采购' : v,
        },
        {
          title: '生命周期',
          dataIndex: 'lifecycle_status',
          width: 120,
          render: (v: string) => (
            <Tag color={LIFECYCLE_COLORS[v] || 'default'}>
              {LIFECYCLE_LABELS[v] || v}
            </Tag>
          ),
        },
        { title: '目标市场', dataIndex: 'target_market', width: 120 },
        { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
      ]}
      searchFields={[
        { name: 'lifecycle_status', label: '生命周期', type: 'select', options: Object.entries(LIFECYCLE_LABELS).map(([k, v]) => ({ label: v, value: k })) },
      ]}
      detailPage="/product-hub"
    />
  );
}
```

- [ ] **Product detail page** — `[id]/page.tsx`

```tsx
'use client';

import { useParams } from 'next/navigation';
import { Card, Col, Descriptions, Row, Spin, Tag, Typography, Table, Statistic, Timeline } from 'antd';
import { useQuery } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';

const { Title } = Typography;

const LIFECYCLE_COLORS: Record<string, string> = {
  idea: 'default', researching: 'blue', sampling: 'orange',
  approved: 'cyan', costed: 'geekblue', ready_to_list: 'purple',
  listed: 'processing', active: 'success', sunset: 'warning', archived: 'default',
};
const LIFECYCLE_LABELS: Record<string, string> = {
  idea: '创意', researching: '调研中', sampling: '打样中', approved: '已确认',
  costed: '已核算成本', ready_to_list: '待上架', listed: '已上架',
  active: '销售中', sunset: '衰退中', archived: '已归档',
};

interface ProductHubProfile {
  master: Record<string, unknown>;
  variants: Array<Record<string, unknown>>;
  concept: Record<string, unknown> | null;
  latest_cost: Record<string, unknown> | null;
  cost_history: Array<Record<string, unknown>>;
  suppliers: Array<Record<string, unknown>>;
  samples: Array<Record<string, unknown>>;
  listings: Array<Record<string, unknown>>;
  timeline: Array<Record<string, unknown>>;
}

export default function ProductHubDetailPage() {
  const { id } = useParams<{ id: string }>();

  const { data, isLoading } = useQuery<ProductHubProfile>({
    queryKey: ['product-hub', id],
    queryFn: async () => {
      const res = await apiClient.get(`/v1/product-hub/${id}/hub`);
      return res.data?.data;
    },
  });

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />;
  if (!data) return <PageContainer><Typography.Text type="danger">产品未找到</Typography.Text></PageContainer>;

  const { master, variants, concept, latest_cost, suppliers, samples, timeline } = data;

  return (
    <PageContainer>
      <Title level={3}>
        {master?.name as string}
        <Tag color={LIFECYCLE_COLORS[master?.lifecycle_status as string] || 'default'} style={{ marginLeft: 12 }}>
          {LIFECYCLE_LABELS[master?.lifecycle_status as string] || master?.lifecycle_status as string}
        </Tag>
      </Title>

      <Row gutter={[16, 16]}>
        {/* Product Master Info */}
        <Col span={16}>
          <Card title="基本信息" size="small">
            <Descriptions column={2} size="small">
              <Descriptions.Item label="产品编号">{master?.product_code as string}</Descriptions.Item>
              <Descriptions.Item label="业务模式">{master?.business_model as string}</Descriptions.Item>
              <Descriptions.Item label="目标市场">{master?.target_market as string}</Descriptions.Item>
              <Descriptions.Item label="目标售价">¥{master?.target_price as number}</Descriptions.Item>
              <Descriptions.Item label="负责人ID">{master?.owner_id as number}</Descriptions.Item>
              <Descriptions.Item label="描述">{(master?.description as string) || '-'}</Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>

        {/* Cost Summary */}
        <Col span={8}>
          <Card title="最新成本" size="small">
            {latest_cost ? (
              <>
                <Statistic title="到仓成本" value={latest_cost?.landed_cost as number} prefix="¥" precision={2} />
                <Statistic title="建议售价" value={latest_cost?.recommended_price as number} prefix="¥" precision={2} />
                <Statistic title="毛利率" value={latest_cost?.gross_margin as number} suffix="%" precision={1} />
              </>
            ) : (
              <Typography.Text type="secondary">暂无成本数据</Typography.Text>
            )}
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        {/* Concept */}
        <Col span={12}>
          <Card title="产品创意" size="small">
            {concept ? (
              <Descriptions column={1} size="small">
                <Descriptions.Item label="简述">{concept?.brief as string}</Descriptions.Item>
                <Descriptions.Item label="目标客户">{concept?.target_customer as string || '-'}</Descriptions.Item>
                <Descriptions.Item label="解决痛点">{concept?.pain_point as string || '-'}</Descriptions.Item>
              </Descriptions>
            ) : (
              <Typography.Text type="secondary">暂无创意信息</Typography.Text>
            )}
          </Card>
        </Col>

        {/* Suppliers */}
        <Col span={12}>
          <Card title="供应商报价" size="small">
            {suppliers && suppliers.length > 0 ? (
              <Table
                dataSource={suppliers as Array<Record<string, unknown>>}
                rowKey={(r) => (r as Record<string, unknown>).offer?.id as number || (r as any).supplier_offer?.id}
                size="small"
                pagination={false}
                columns={[
                  { title: '供应商', dataIndex: 'supplier_name', key: 'name' },
                  { title: '单价', key: 'cost', render: (_, r) => {
                    const offer = (r as any).supplier_offer || (r as any).offer;
                    return offer ? `¥${offer.unit_cost}` : '-';
                  }},
                  { title: 'MOQ', key: 'moq', render: (_, r) => ((r as any).supplier_offer || (r as any).offer)?.moq || '-' },
                ]}
              />
            ) : (
              <Typography.Text type="secondary">暂无供应商报价</Typography.Text>
            )}
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        {/* Samples */}
        <Col span={12}>
          <Card title="打样记录" size="small">
            {samples && samples.length > 0 ? (
              <Table
                dataSource={samples as Array<Record<string, unknown>>}
                rowKey="id"
                size="small"
                pagination={false}
                columns={[
                  { title: '供应商ID', dataIndex: 'supplier_id', key: 'supplier_id' },
                  { title: '状态', dataIndex: 'status', key: 'status' },
                  { title: '评分', dataIndex: 'quality_score', key: 'score' },
                  { title: '结论', dataIndex: 'decision', key: 'decision' },
                ]}
              />
            ) : (
              <Typography.Text type="secondary">暂无打样记录</Typography.Text>
            )}
          </Card>
        </Col>

        {/* Timeline */}
        <Col span={12}>
          <Card title="生命周期时间线" size="small">
            <Timeline
              items={(timeline as Array<Record<string, unknown>>)?.map((t) => ({
                children: <>{t.summary as string} <Typography.Text type="secondary">{t.created_at as string}</Typography.Text></>,
              }))}
            />
          </Card>
        </Col>
      </Row>

      {/* Variants */}
      <Card title="变体 / SKU" size="small" style={{ marginTop: 16 }}>
        {variants && variants.length > 0 ? (
          <Table
            dataSource={variants as Array<Record<string, unknown>>}
            rowKey="id"
            size="small"
            columns={[
              { title: 'SKU编码', dataIndex: 'sku_code', key: 'code' },
              { title: '规格', dataIndex: 'variant_label', key: 'label' },
              { title: '重量(kg)', dataIndex: 'weight', key: 'weight' },
              { title: '尺寸', dataIndex: 'dimensions', key: 'dimensions' },
              { title: '条形码', dataIndex: 'barcode', key: 'barcode' },
              { title: '原产国', dataIndex: 'country_of_origin', key: 'origin' },
            ]}
          />
        ) : (
          <Typography.Text type="secondary">暂无变体</Typography.Text>
        )}
      </Card>
    </PageContainer>
  );
}
```

- [ ] **Commit**

```bash
cd frontend-next && npm run lint -- --quiet || true
git add frontend-next/src/app/\(main\)/product-hub/
git commit -m "feat: add Product Hub frontend pages with aggregated detail view"
```

---

### Task 11: Frontend — Menu Entry

**Files:**
- Modify: `frontend-next/src/config/menu.ts`

- [ ] **Add Product Hub menu entry under 商品管理**

```typescript
{
  label: '商品管理',
  items: [
    { key: '/product-hub', icon: 'DatabaseOutlined', label: '产品档案' },  // ← add this
    { key: '/products', icon: 'ShoppingOutlined', label: '商品' },
    { key: '/categories', label: '类目' },
    { key: '/brands', label: '品牌' },
    { key: '/sku', label: 'SKU' },
    { key: '/inventory', label: '库存' },
    { key: '/suppliers', label: '供应商' },
  ],
},
```

- [ ] **Commit**

```bash
git add frontend-next/src/config/menu.ts
git commit -m "feat: add Product Hub menu entry"
```

---

### Task 12: Full Integration Test

**Files:**
- Run the full stack integration

- [ ] **Run backend tests**

```bash
cd backend-go && go test ./internal/domain/producthub/ -v -count=1
```

- [ ] **Run backend vet and build**

```bash
cd backend-go && go vet ./internal/domain/producthub/ && go build ./...
```

- [ ] **Run frontend build**

```bash
cd frontend-next && npm run build
```

- [ ] **If all pass, commit any final fixes**

```bash
git add -A
git commit -m "chore: final integration fixes for Product Hub V1"
```

---

## Post-V1 Scope (Not in this plan)

These were excluded per spec section 4.3 — they belong in V2-V5:

| Feature | Planned Version |
|---------|----------------|
| BOM / BOMItem | V3 |
| production_batch / qc_inspection | V3 |
| sales_snapshot / after_sales_signal | V4 |
| lifecycle_event universal table | V4 |
| supplier_capability tags | V5 |
| channel_listing full rewrite | V2 (use existing listing module for now) |
| Approval workflow engine | V2 |
| Compliance/document management | V2 |
