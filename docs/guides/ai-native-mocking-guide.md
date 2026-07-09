# Stateful Mocking & Outbound Safety Gates Guide

> **Purpose**: Guide AI coding agents on how to construct stateful mock adapters, enforce numerical financial precision, use transactional seeder locks, and implement fail-safe outbound network gates.

---

## 1. Stateful Mocks vs. Stateless Mocks
When testing complex business flows (e.g., *Publish Product -> Receive Order -> Request Shipping Label -> Capture Payment -> Settle Ledgers*), stateless mocks (which return hardcoded static data) fail because they cannot track progression of states.

We implement **Stateful Mocking**:
- Mocks read/write to dedicated mock tables (`mock_listings`, `mock_shipments`, etc.).
- Mode is controlled via `ExecutionMode` context (DryRun / Sandbox / Production).
- AI developers can simulate full multi-step loops safely without connecting to live APIs.

---

## 2. GORM Mock Schema Design

### 2.1 String Column Lengths & Schema Drift Protection
String fields in GORM models must be explicitly tagged to match migration limits (e.g. `type:varchar(50)`). This prevents the startup Schema Drift Detector (`schemadrift.DriftDetector`) from generating false warnings when matching GORM `text` reflection against PostgreSQL `character varying`.

### 2.2 Financial Numerical Precision
**Never use `float64` for money, price, or commission variables.** Floats introduce binary rounding errors. Always store currency values as **integer cents** (using `int64` suffixing `_cents` in GORM models, or GORM-compatible Decimal structs like `shopspring/decimal.Decimal`).

```go
package mock

import (
	"time"
)

type MockListing struct {
	ID                int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	PlatformID        int64     `gorm:"column:platform_id;not null" json:"platform_id"`
	AccountID         int64     `gorm:"column:account_id;not null" json:"account_id"`
	ProductID         int64     `gorm:"column:product_id;not null" json:"product_id"`
	PlatformProductID string    `gorm:"column:platform_product_id;type:varchar(100);uniqueIndex;not null" json:"platform_product_id"`
	Status            string    `gorm:"column:status;type:varchar(20);default:'live';not null" json:"status"`
	PriceCents        int64     `gorm:"column:price_cents;not null" json:"price_cents"` // Money stored as cents
	Stock             int       `gorm:"column:stock;default:0" json:"stock"`
	CreatedAt         time.Time `gorm:"column:created_at"`
}
```

---

## 3. Seeding Concurrency & Transaction Safety
When multiple sandbox runs trigger seeding (`POST /api/v1/mock/seed`) concurrently, a TOCTOU (Time-of-check to time-of-use) race condition can cause double seeding.

To solve this, seeding must occur in a single **GORM Transaction block** secured by a **PostgreSQL Transactional Advisory Lock**:

```go
func (s *Service) SeedMockData() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Acquire PostgreSQL transactional advisory lock to block concurrent seeding execution
		if err := tx.Exec("SELECT pg_advisory_xact_lock(123456)").Error; err != nil {
			return fmt.Errorf("failed to acquire advisory lock: %w", err)
		}

		var count int64
		if err := tx.Model(&MockListing{}).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil // Already seeded, skip
		}

		// Perform bulk insert
		mockListings := []MockListing{
			{PlatformID: 1, AccountID: 1, ProductID: 101, PlatformProductID: "mock-ozon-101", PriceCents: 1999, Stock: 50, CreatedAt: time.Now()},
			{PlatformID: 2, AccountID: 2, ProductID: 102, PlatformProductID: "mock-shopee-102", PriceCents: 2450, Stock: 30, CreatedAt: time.Now()},
		}

		return tx.Create(&mockListings).Error
	})
}
```

---

## 4. Fail-Safe Outbound Network Interceptor (`FailSafeRoundTripper`)

To prevent API connection leaks during testing/development, AI developers **must never** instantiate `&http.Client{}` directly in business adapters. They must call `httpx.NewClient()` which automatically binds a thread-safe `FailSafeRoundTripper`.

### Implementation: `backend-go/internal/httpx/failsafe_transport.go`

```go
package httpx

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
)

var ErrOutboundBlocked = errors.New("outbound network traffic blocked by FailSafeRoundTripper")

type FailSafeRoundTripper struct {
	BaseTransport   http.RoundTripper
	Environment     string
	AllowedHosts    map[string]bool
	AllowedSuffixes []string
	mu              sync.RWMutex
}

func NewFailSafeRoundTripper(base http.RoundTripper, env string) *FailSafeRoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &FailSafeRoundTripper{
		BaseTransport: base,
		Environment:   strings.ToLower(env),
		AllowedHosts: map[string]bool{
			"localhost": true,
			"127.0.0.1": true,
			"::1":       true,
		},
	}
}

func (rt *FailSafeRoundTripper) SetAllowedHosts(hosts []string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.AllowedHosts = map[string]bool{
		"localhost": true,
		"127.0.0.1": true,
		"::1":       true,
	}
	var suffixes []string
	for _, h := range hosts {
		h = strings.ToLower(strings.TrimSpace(h))
		if strings.HasPrefix(h, "*.") {
			suffixes = append(suffixes, h[1:])
		} else {
			rt.AllowedHosts[h] = true
		}
	}
	rt.AllowedSuffixes = suffixes
}

func (rt *FailSafeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.Environment == "production" {
		return rt.BaseTransport.RoundTrip(req)
	}

	host := strings.ToLower(req.URL.Hostname())

	rt.mu.RLock()
	allowed := rt.AllowedHosts[host]
	suffixes := rt.AllowedSuffixes
	rt.mu.RUnlock()

	if !allowed {
		for _, suffix := range suffixes {
			if strings.HasSuffix(host, suffix) {
				allowed = true
				break
			}
		}
	}

	// Unconditionally allow loopback and private subnets (local microservices, docker DNS names)
	if !allowed {
		if ip := net.ParseIP(host); ip != nil {
			if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
				allowed = true
			}
		}
	}

	if !allowed {
		return nil, fmt.Errorf("%w: blocked outbound request to %q (env=%s)",
			ErrOutboundBlocked, host, rt.Environment)
	}

	return rt.BaseTransport.RoundTrip(req)
}
```

---

## 5. Interface Segregation Principle (ISP) Compliance
Do not pack all vertical domains into one interface. Implement clean, segregated contracts in Go for different industries:

* **`PlatformAdapter` (E-Commerce)**:
  `Publish(ctx, input) (*PublishResult, error)`
  `SyncInventory(ctx, platformProductID, sku, qty) error`
  `FetchOrders(ctx, since) ([]Order, error)`
* **`BankAdapter` (Finance)**:
  `ExecuteTransfer(ctx, req *TransferRequest) (*TransferResponse, error)`
  `FetchBankStatement(ctx, period) ([]BankTransaction, error)`
* **`RFQAdapter` (Foreign Trade)**:
  `FetchRFQs(ctx, filters) ([]RFQRecord, error)`
  `SubmitQuotation(ctx, rfqID, quote) (*QuotationResult, error)`
