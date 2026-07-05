> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# CandidateProduct Completeness Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade CandidateProduct completeness engine from 2 states to 4 states, surface missing_fields and suggestions in list + detail views, add completeness_status filter.

**Architecture:** Upgrade `computeCompleteness()` function in candidate/completeness.go with 4-state logic. Enhance List/Get/Update service methods. Update frontend candidates page to show completeness status + missing fields + suggestions.

**Tech Stack:** Go 1.25 + Gin/GORM, Next.js 16 + React 19 + Ant Design 6 + TanStack React Query 5

## Global Constraints

- Do not modify completeness domain module (`domain/completeness/`)
- Do not modify any migration file
- Do not modify order/inventory/listing/shipping domains
- No auto-publish, no auto-buy, no auto-price-change
- go test ./... and go vet ./... must pass
- Frontend npm run build must pass
- Follow project patterns: PageContainer/SectionCard for frontend pages
- API prefix: /api/v1, all endpoints JWT-protected

---

### Task 1: Upgrade completeness.go — 4-state engine with missing_fields

**Files:**
- Modify: `backend-go/internal/domain/candidate/completeness.go`

**Interfaces:**
- Produces: `func computeCompleteness(p *CandidateProduct) (status string, missingFields []string)` — returns 4 states and list of missing field keys

- [ ] **Step 1: Upgrade the completeness function**

Replace the existing `computeCompleteness` with the 4-state version:

```go
package candidate

// computeCompleteness evaluates a CandidateProduct's field completeness.
// Returns the overall status and a list of missing field names.
//
// Status progression:
//
//	incomplete     — missing core fields (title, purchase_price, main_image)
//	needs_review   — has core fields, missing supplier or package info
//	research_ready — has core + supplier + package, can proceed to profit check
//	listing_ready  — all 11 fields present
func computeCompleteness(p *CandidateProduct) (status string, missingFields []string) {
	// Check all 11 fields
	checks := []struct {
		key    string
		exists bool
	}{
		{"title", p.Title != ""},
		{"purchase_price", p.PurchasePrice > 0},
		{"main_image", p.MainImage != ""},
		{"supplier_id", p.SupplierID != nil && *p.SupplierID > 0},
		{"package_weight_kg", p.PackageWeightKg > 0},
		{"package_length_cm", p.PackageLengthCm > 0},
		{"package_width_cm", p.PackageWidthCm > 0},
		{"package_height_cm", p.PackageHeightCm > 0},
		{"hs_code", p.HSCode != ""},
		{"target_sale_price", p.TargetSalePrice > 0},
		{"origin_country", p.OriginCountry != ""},
	}

	for _, c := range checks {
		if !c.exists {
			missingFields = append(missingFields, c.key)
		}
	}

	// 3-tier classification
	hasCore := p.Title != "" && p.PurchasePrice > 0 && p.MainImage != ""
	hasSupplier := p.SupplierID != nil && *p.SupplierID > 0
	hasPackage := p.PackageWeightKg > 0 || (p.PackageLengthCm > 0 && p.PackageWidthCm > 0 && p.PackageHeightCm > 0)
	hasAll := len(missingFields) == 0

	switch {
	case !hasCore:
		return "incomplete", missingFields
	case !hasSupplier || !hasPackage:
		return "needs_review", missingFields
	case hasAll:
		return "listing_ready", nil
	default:
		return "research_ready", missingFields
	}
}
```

- [ ] **Step 2: Verify file compiles**

```bash
cd backend-go && go vet ./internal/domain/candidate/
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add backend-go/internal/domain/candidate/completeness.go
git commit -m "feat: upgrade computeCompleteness to 4 states (incomplete/needs_review/research_ready/listing_ready)"
```

### Task 2: Add CandidateDetail response + ListCandidateFilter types

**Files:**
- Modify: `backend-go/internal/domain/candidate/model.go`

**Interfaces:**
- Produces: `CandidateDetail` struct with `missing_fields`; `ListCandidateFilter` struct for list params

- [ ] **Step 1: Add response types to model.go**

```go
// CandidateDetail enriches CandidateProduct with computed completeness info.
type CandidateDetail struct {
	CandidateProduct
	MissingFields []string `json:"missing_fields"`
}

// ListCandidateFilter holds optional filters for listing candidate products.
type ListCandidateFilter struct {
	Status             string
	Search             string
	CompletenessStatus string
}
```

- [ ] **Step 2: Verify file compiles**

```bash
cd backend-go && go vet ./internal/domain/candidate/
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add backend-go/internal/domain/candidate/model.go
git commit -m "feat: add CandidateDetail and ListCandidateFilter types"
```

### Task 3: Enhance Service — List filter, Update recalc, Get with missing_fields

**Files:**
- Modify: `backend-go/internal/domain/candidate/service.go`

**Interfaces:**
- Consumes: `computeCompleteness(p)`, `ListCandidateFilter`, `CandidateDetail`
- Produces: enhanced `List()` with completeness_status filter, `GetByID()` returning `CandidateDetail`, `Update()` recalculating completeness

- [ ] **Step 1: Refactor List() to accept ListCandidateFilter and support completeness_status filter**

```go
// List returns paginated candidate products with optional filters.
func (s *Service) List(p *common.Pagination, filter *ListCandidateFilter) ([]CandidateProduct, int64, error) {
	var items []CandidateProduct
	var total int64
	q := s.db.Model(&CandidateProduct{})
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.CompletenessStatus != "" {
		q = q.Where("completeness_status = ?", filter.CompletenessStatus)
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		q = q.Where("title ILIKE ? OR description ILIKE ?", like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
```

- [ ] **Step 2: Add GetDetail() that returns CandidateDetail with missing_fields**

```go
// GetDetail returns a candidate product with computed missing fields.
func (s *Service) GetDetail(id int64) (*CandidateDetail, error) {
	var c CandidateProduct
	if err := s.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	_, missing := computeCompleteness(&c)
	return &CandidateDetail{
		CandidateProduct: c,
		MissingFields:    missing,
	}, nil
}
```

- [ ] **Step 3: Enhance Update() to recalculate completeness_status after update**

```go
// Update patches a candidate product by id, recalculating completeness_status.
func (s *Service) Update(id int64, in *UpdateCandidateInput) (*CandidateProduct, error) {
	var c CandidateProduct
	if err := s.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	// ... (existing field mapping stays unchanged) ...

	// After applying updates, recalculate completeness
	// Re-read the updated record
	if len(updates) == 0 {
		return &c, nil
	}
	if err := s.db.Model(&c).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	// Recalculate completeness based on current state
	status, _ := computeCompleteness(&c)
	if status != c.CompletenessStatus {
		s.db.Model(&c).Update("completeness_status", status)
		c.CompletenessStatus = status
	}
	return &c, nil
}
```

- [ ] **Step 4: Verify file compiles**

```bash
cd backend-go && go vet ./internal/domain/candidate/
```

Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add backend-go/internal/domain/candidate/service.go
git commit -m "feat: add completeness_status filter to List, GetDetail with missing_fields, Update recalculates completeness"
```

### Task 4: Enhance Handler — expose missing_fields and filter params

**Files:**
- Modify: `backend-go/internal/domain/candidate/handler.go`

- [ ] **Step 1: Update List handler to parse completeness_status and use ListCandidateFilter**

```go
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	filter := &ListCandidateFilter{}
	filter.Status = c.Query("status")
	filter.Search = c.Query("search")
	filter.CompletenessStatus = c.Query("completeness_status")
	items, total, err := h.service.List(&p, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}
```

- [ ] **Step 2: Update Get handler to use GetDetail**

```go
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, err := h.service.GetDetail(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "candidate product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, item)
}
```

- [ ] **Step 3: Verify file compiles**

```bash
cd backend-go && go vet ./internal/domain/candidate/
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add backend-go/internal/domain/candidate/handler.go
git commit -m "feat: connect handler to GetDetail + ListCandidateFilter with completeness_status"
```

### Task 5: Add tests for 4-state completeness + completeness_status filter

**Files:**
- Modify: `backend-go/internal/domain/candidate/candidate_test.go`

- [ ] **Step 1: Add TestComputeCompleteness_NeedsReview test**

```go
func TestComputeCompleteness_NeedsReview(t *testing.T) {
	t.Parallel()
	sid := int64(1)
	p := &CandidateProduct{
		Title:           "Has Core Fields",
		PurchasePrice:   100.0,
		MainImage:       "https://example.com/img.jpg",
		// Missing: supplier_id, package info
	}
	status, missing := computeCompleteness(p)
	if status != "needs_review" {
		t.Fatalf("expected needs_review, got %s", status)
	}
	checked := make(map[string]bool)
	for _, f := range missing {
		checked[f] = true
	}
	if !checked["supplier_id"] {
		t.Fatal("expected supplier_id in missing fields")
	}
}
```

- [ ] **Step 2: Add TestComputeCompleteness_ResearchReady test**

```go
func TestComputeCompleteness_ResearchReady(t *testing.T) {
	t.Parallel()
	sid := int64(1)
	p := &CandidateProduct{
		Title:           "Research Ready",
		PurchasePrice:   100.0,
		MainImage:       "https://example.com/img.jpg",
		SupplierID:      &sid,
		PackageWeightKg: 0.5,
		// Missing: HSCode, target_sale_price, origin_country
	}
	status, missing := computeCompleteness(p)
	if status != "research_ready" {
		t.Fatalf("expected research_ready, got %s", status)
	}
	if len(missing) == 0 {
		t.Fatal("expected missing fields")
	}
}
```

- [ ] **Step 3: Add TestComputeCompleteness_ListingReady test**

```go
func TestComputeCompleteness_ListingReady(t *testing.T) {
	t.Parallel()
	sid := int64(1)
	p := &CandidateProduct{
		Title:           "Listing Ready",
		PurchasePrice:   100.0,
		MainImage:       "https://example.com/img.jpg",
		SupplierID:      &sid,
		PackageWeightKg: 0.5,
		PackageLengthCm: 10,
		PackageWidthCm:  8,
		PackageHeightCm: 6,
		HSCode:          "847130",
		TargetSalePrice: 25.99,
		OriginCountry:   "CN",
	}
	status, missing := computeCompleteness(p)
	if status != "listing_ready" {
		t.Fatalf("expected listing_ready, got %s", status)
	}
	if len(missing) != 0 {
		t.Fatalf("expected no missing fields, got %v", missing)
	}
}
```

- [ ] **Step 4: Add TestComputeCompleteness_ZeroPrice test**

```go
func TestComputeCompleteness_ZeroPrice(t *testing.T) {
	t.Parallel()
	p := &CandidateProduct{
		Title:     "Zero Price",
		MainImage: "https://example.com/img.jpg",
		// PurchasePrice is 0 (default) — incomplete
	}
	status, missing := computeCompleteness(p)
	if status != "incomplete" {
		t.Fatalf("expected incomplete, got %s", status)
	}
	found := false
	for _, f := range missing {
		if f == "purchase_price" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected purchase_price in missing fields")
	}
}
```

- [ ] **Step 5: Add TestComputeCompleteness_AllEmpty test** — record created with no fields at all

```go
func TestComputeCompleteness_AllEmpty(t *testing.T) {
	t.Parallel()
	p := &CandidateProduct{}
	status, missing := computeCompleteness(p)
	if status != "incomplete" {
		t.Fatalf("expected incomplete, got %s", status)
	}
	if len(missing) == 0 {
		t.Fatal("expected missing fields for empty product")
	}
}
```

- [ ] **Step 6: Add TestService_List_FilterByCompletenessStatus**

```go
func TestService_List_FilterByCompletenessStatus(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create several products with different completeness
	// Insert directly to bypass Create's auto-calculation
	now := time.Now()
	records := []CandidateProduct{
		{Title: "A", PurchasePrice: 50, MainImage: "img.jpg", CompletenessStatus: "research_ready", Status: "draft", CreatedBy: "tester", CreatedAt: now, UpdatedAt: now},
		{Title: "B", PurchasePrice: 0, CompletenessStatus: "incomplete", Status: "draft", CreatedBy: "tester", CreatedAt: now, UpdatedAt: now},
		{Title: "C", PurchasePrice: 50, MainImage: "img.jpg", CompletenessStatus: "research_ready", Status: "draft", CreatedBy: "tester", CreatedAt: now, UpdatedAt: now},
	}
	for _, r := range records {
		if err := db.Create(&r).Error; err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	p := common.Pagination{Page: 1, Size: 10}
	items, total, err := svc.List(&p, &ListCandidateFilter{CompletenessStatus: "research_ready"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 research_ready, got %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}
```

- [ ] **Step 7: Add TestService_Update_RecalculatesCompleteness**

```go
func TestService_Update_RecalculatesCompleteness(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create a minimal product — should be "incomplete"
	price := 50.0
	c, err := svc.Create(&CreateCandidateInput{
		Title:         "To Update",
		PurchasePrice: &price,
		CreatedBy:     "tester",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.CompletenessStatus != "incomplete" {
		t.Fatalf("expected incomplete, got %s", c.CompletenessStatus)
	}

	// Add main_image — should still be incomplete (no title? Wait, title exists)
	// Actually, existing: title + price + no main_image → incomplete because no main_image
	img := "https://example.com/img.jpg"
	updated, err := svc.Update(c.ID, &UpdateCandidateInput{MainImage: &img})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.CompletenessStatus != "needs_review" {
		t.Fatalf("expected needs_review after adding main_image, got %s", updated.CompletenessStatus)
	}
}
```

- [ ] **Step 8: Add TestService_GetDetail_ReturnsMissingFields**

```go
func TestService_GetDetail_ReturnsMissingFields(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	price := 50.0
	c, err := svc.Create(&CreateCandidateInput{
		Title:         "Detail Test",
		PurchasePrice: &price,
		CreatedBy:     "tester",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	detail, err := svc.GetDetail(c.ID)
	if err != nil {
		t.Fatalf("GetDetail: %v", err)
	}
	if len(detail.MissingFields) == 0 {
		t.Fatal("expected missing fields for incomplete product")
	}
	if detail.ID != c.ID {
		t.Fatal("ID mismatch")
	}
}
```

- [ ] **Step 9: Run all tests**

```bash
cd backend-go && go test ./internal/domain/candidate/ -v
```

Expected: all tests pass (existing + new)

- [ ] **Step 10: Commit**

```bash
git add backend-go/internal/domain/candidate/candidate_test.go
git commit -m "test: add tests for 4-state completeness, missing_fields, completeness_status filter"
```

### Task 6: Frontend — completeness display + filter + detail suggestions

**Files:**
- Modify: `frontend-next/src/app/(main)/candidates/page.tsx`

**Interfaces:**
- Consumes: `GET /v1/candidates?completeness_status=` and `GET /v1/candidates/:id` (returns `missing_fields`)

- [ ] **Step 1: Rebuild page using PageContainer with completeness_status column + filter + enhanced detail**

Replace the entire page.tsx content:

```tsx
'use client';

import { useState, useEffect } from 'react';
import { Button, Card, message, Space, Table, Tag, Typography, Select } from 'antd';
import { DatabaseOutlined, PlayCircleOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client';
import dayjs from 'dayjs';
import PageContainer from '@/components/ui/PageContainer';

interface CandidateProduct {
  id: number;
  title: string;
  description: string;
  main_image: string;
  purchase_price: number;
  purchase_currency: string;
  package_weight_kg: number;
  target_sale_price: number;
  target_currency: string;
  target_platform_id: number | null;
  destination_country: string;
  hs_code: string;
  origin_country: string;
  status: string;
  completeness_status: string;
  is_seed_data: boolean;
  created_by: string;
  created_at: string;
}

interface CandidateDetail extends CandidateProduct {
  missing_fields: string[];
}

const completenessColorMap: Record<string, string> = {
  incomplete: 'default',
  needs_review: 'warning',
  research_ready: 'processing',
  listing_ready: 'success',
};

const completenessLabelMap: Record<string, string> = {
  incomplete: '不完整',
  needs_review: '待补充',
  research_ready: '可调研',
  listing_ready: '可上架',
};

const completenessHintMap: Record<string, string> = {
  incomplete: '缺少核心信息（标题、采购价、主图），补充后才能继续',
  needs_review: '已有关键信息，补充供应商和包装信息后可进入调研',
  research_ready: '信息基本完整，可以执行利润分析和选品调研',
  listing_ready: '所有信息完整，可以准备上架草稿',
};

const fieldLabelMap: Record<string, string> = {
  title: '商品标题',
  purchase_price: '采购成本',
  main_image: '主图',
  supplier_id: '供应商',
  package_weight_kg: '包装重量',
  package_length_cm: '包装长度',
  package_width_cm: '包装宽度',
  package_height_cm: '包装高度',
  hs_code: 'HS编码',
  target_sale_price: '目标售价',
  origin_country: '原产地',
};

const statusColorMap: Record<string, string> = {
  draft: 'default',
  in_review: 'processing',
  approved: 'success',
  rejected: 'error',
};

const statusLabelMap: Record<string, string> = {
  draft: '草稿',
  in_review: '审核中',
  approved: '已通过',
  rejected: '已拒绝',
};

const platformLabelMap: Record<string, string> = {
  '1': 'Ozon',
  '2': 'Shopee',
  '3': 'Lazada',
};

export default function CandidatesPage() {
  const router = useRouter();
  const [data, setData] = useState<CandidateProduct[]>([]);
  const [loading, setLoading] = useState(false);
  const [seeding, setSeeding] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [completenessFilter, setCompletenessFilter] = useState<string>('');
  const [detail, setDetail] = useState<CandidateDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  const fetchCandidates = async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = { page: String(page), size: String(pageSize) };
      if (completenessFilter) params.completeness_status = completenessFilter;
      const res = (await apiClient.get('/v1/candidates', params)) as unknown as {
        data: CandidateProduct[];
        total: number;
      };
      setData(res.data || []);
      setTotal(res.total || 0);
    } catch {
      message.error('加载候选商品列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCandidates();
  }, [page, pageSize, completenessFilter]);

  const handleSeed = async () => {
    setSeeding(true);
    try {
      const res = await apiClient.post('/v1/candidates/seed');
      message.success('种子数据生成成功');
      await fetchCandidates();
    } catch {
      message.error('种子数据请求失败');
    } finally {
      setSeeding(false);
    }
  };

  const handleDetail = async (id: number) => {
    setDetailLoading(true);
    setDetail(null);
    try {
      const res = (await apiClient.get(`/v1/candidates/${id}`)) as unknown as {
        data: CandidateDetail;
      };
      setDetail(res.data || null);
    } catch {
      message.error('获取详情失败');
    } finally {
      setDetailLoading(false);
    }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    {
      title: '标题',
      dataIndex: 'title',
      ellipsis: true,
    },
    {
      title: '完整度',
      dataIndex: 'completeness_status',
      width: 100,
      render: (s: string) =>
        s ? (
          <Tag color={completenessColorMap[s] || 'default'}>
            {completenessLabelMap[s] || s}
          </Tag>
        ) : (
          <Tag>未检查</Tag>
        ),
    },
    {
      title: '采购价',
      dataIndex: 'purchase_price',
      width: 100,
      render: (price: number) => (price != null ? `¥${price.toFixed(2)}` : '-'),
    },
    {
      title: '目标售价',
      dataIndex: 'target_sale_price',
      width: 110,
      render: (price: number) => (price != null ? `$${price.toFixed(2)}` : '-'),
    },
    {
      title: '目标平台',
      dataIndex: 'target_platform_id',
      width: 100,
      render: (id: number) => (id ? platformLabelMap[String(id)] || `平台 #${id}` : '-'),
    },
    {
      title: '目的国',
      dataIndex: 'destination_country',
      width: 80,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (s: string) => (
        <Tag color={statusColorMap[s] || 'default'}>{statusLabelMap[s] || s}</Tag>
      ),
    },
    {
      title: '操作',
      width: 100,
      render: (_: unknown, record: CandidateProduct) => (
        <Button
          type="link"
          size="small"
          onClick={() => handleDetail(record.id)}
        >
          详情
        </Button>
      ),
    },
  ];

  return (
    <PageContainer
      title="候选商品"
      subtitle="采集回来的候选商品，查看完整度状态并执行下一步操作"
      loading={loading}
    >
      {/* Toolbar */}
      <Card
        size="small"
        style={{ marginBottom: 'var(--space-lg)' }}
        styles={{ body: { padding: '12px 20px', display: 'flex', alignItems: 'center', gap: 12 } }}
      >
        <Button type="primary" icon={<DatabaseOutlined />} onClick={handleSeed} loading={seeding}>
          生成种子数据
        </Button>
        <div style={{ flex: 1 }} />
        <Select
          allowClear
          placeholder="按完整度筛选"
          style={{ width: 160 }}
          value={completenessFilter || undefined}
          onChange={(val) => {
            setCompletenessFilter(val || '');
            setPage(1);
          }}
          options={[
            { value: 'incomplete', label: '不完整' },
            { value: 'needs_review', label: '待补充' },
            { value: 'research_ready', label: '可调研' },
            { value: 'listing_ready', label: '可上架' },
          ]}
        />
      </Card>

      {/* Table */}
      <Card size="small" styles={{ body: { padding: 0 } }}>
        <Table<CandidateProduct>
          rowKey="id"
          columns={columns}
          dataSource={data}
          loading={loading}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
          scroll={{ x: 850 }}
        />
      </Card>

      {/* Detail Drawer/Modal */}
      {detail && (
        <Card
          size="small"
          title={
            <Space>
              <span>候选商品 #{detail.id}</span>
              <Tag color={completenessColorMap[detail.completeness_status] || 'default'}>
                {completenessLabelMap[detail.completeness_status] || detail.completeness_status}
              </Tag>
            </Space>
          }
          style={{ marginTop: 'var(--space-lg)' }}
          extra={
            <Button size="small" onClick={() => setDetail(null)}>
              关闭
            </Button>
          }
        >
          <Typography.Title level={5} style={{ marginTop: 0 }}>
            {detail.title}
          </Typography.Title>

          <div style={{ marginBottom: 'var(--space-lg)', color: 'var(--t2)', lineHeight: 1.8 }}>
            <div>采购价：¥{detail.purchase_price?.toFixed(2)}</div>
            <div>目标售价：${detail.target_sale_price?.toFixed(2)} {detail.target_currency}</div>
            <div>
              目标平台：
              {detail.target_platform_id
                ? platformLabelMap[String(detail.target_platform_id)] || `平台 #${detail.target_platform_id}`
                : '-'}{' '}
              → {detail.destination_country || '-'}
            </div>
            <div>包装：{detail.package_weight_kg?.toFixed(2)}kg</div>
            <div>来源：{detail.created_by || '-'}</div>
            <div>
              创建时间：
              {detail.created_at ? dayjs(detail.created_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
            </div>
          </div>

          {/* Missing Fields Section */}
          {detail.missing_fields && detail.missing_fields.length > 0 ? (
            <Card
              size="small"
              title="缺失字段"
              type="inner"
              style={{ marginBottom: 'var(--space-md)' }}
            >
              <Space wrap>
                {detail.missing_fields.map((f: string) => (
                  <Tag key={f} color="error">
                    {fieldLabelMap[f] || f}
                  </Tag>
                ))}
              </Space>
            </Card>
          ) : (
            <Card
              size="small"
              type="inner"
              style={{ marginBottom: 'var(--space-md)' }}
            >
              <Typography.Text type="success">所有字段已完整</Typography.Text>
            </Card>
          )}

          {/* Suggestion */}
          <Card size="small" type="inner" style={{ marginBottom: 'var(--space-md)' }}>
            <Typography.Text>
              💡 <strong>下一步建议：</strong>
              {completenessHintMap[detail.completeness_status] || '信息完整，可进行后续操作'}
            </Typography.Text>
          </Card>

          <Button
            type="primary"
            icon={<ThunderboltOutlined />}
            onClick={() => {
              setDetail(null);
              // Navigate to evaluation
            }}
          >
            执行完整度+利润评估
          </Button>
        </Card>
      )}
    </PageContainer>
  );
}
```

- [ ] **Step 2: Build frontend**

```bash
cd frontend-next && npm run build 2>&1 | tail -20
```

Expected: build passes (exported successfully)

- [ ] **Step 3: Commit**

```bash
git add frontend-next/src/app/\(main\)/candidates/page.tsx
git commit -m "feat: update candidates page with completeness status, filter, and missing fields display"
```

### Task 7: Verify full test suite + lint

**Files:**
- None (verification only)

- [ ] **Step 1: Run backend tests**

```bash
cd backend-go && go test ./... 2>&1
```

Expected: all tests pass

- [ ] **Step 2: Run backend vet**

```bash
cd backend-go && go vet ./... 2>&1
```

Expected: clean

- [ ] **Step 3: Run frontend build**

```bash
cd frontend-next && npm run build 2>&1 | tail -30
```

Expected: build passes

- [ ] **Step 4: Run frontend lint (note any pre-existing issues)**

```bash
cd frontend-next && npm run lint 2>&1 | tail -20
```

- [ ] **Step 5: Final commit if any fixes**

```bash
git add -A
git commit -m "chore: fix verification issues"
```

### Task 8: PR creation

**Files:**
- None (git operation)

- [ ] **Step 1: Create PR**

```bash
gh pr create --draft \
  --title "feat: CandidateProduct completeness engine (4 states) + workbench联动" \
  --body "## Business Goal

解决第一公里的核心矛盾：采集回来的数据能否稳定、完整、可判断地变成可继续经营决策的 CandidateProduct。

## What You Can Do Now

| 功能 | 说明 |
|------|------|
| 完整度状态 | 每个候选商品显示「不完整/待补充/可调研/可上架」标签 |
| 缺失字段 | 详情查看缺失哪些字段 + 中文名 |
| 下一步建议 | 根据当前状态推荐下一步操作 |
| 按状态筛选 | 列表可按完整度阶段筛选 |

## 变更文件

| File | Change |
|------|--------|
| completeness.go | 2→4 状态引擎升级 |
| model.go | CandidateDetail + ListCandidateFilter 类型 |
| service.go | List加completeness_status筛选, Update重算, GetDetail |
| handler.go | 接入新参数和GetDetail |
| candidate_test.go | 新增 8 个测试用例 |
| candidates/page.tsx | 完整度列+筛选+缺失字段+详情建议+PageContainer重构 |

## 不变范围

- 不修改 completeness 域模块
- 不修改 migration
- 不修改 order/inventory/listing/shipping
- 无自动采购/上架/改价行为

## 验证结果

- go test ./... ✅
- go vet ./... ✅
- npm run build ✅

## Risk Level

Low-Medium. Read-only completeness tagging + display updates. No price/inventory/order changes."
```
