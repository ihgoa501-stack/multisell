# AgentOS Cognitive and Domain Suites (Phase 2, 3, 4) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Concurrently implement the E-Commerce Stateful Mock adapters (Phase 2), Python Cognitive Microservice (Phase 3), and Finance/Foreign Trade domain verticals (Phase 4).

**Architecture:**
- **Subagent A (Phase 2)**: Focuses on Go backend mock Ozon/Shopee integration (`internal/domain/integrations`), GORM mock tables, and seeder locks.
- **Subagent B (Phase 3)**: Focuses on Python AIOS brain (`python-agentos/`), sqlite-vec database setup, and FastAPI endpoints.
- **Subagent C (Phase 4)**: Focuses on Go backend domain packages for Finance (`internal/domain/finance`) and Trade (`internal/domain/foreigntrade`), and BankAdapter / RFQAdapter interfaces.

**Tech Stack:** Go 1.21 / GORM v2 / Python 3.10 / FastAPI / sqlite-vec / Next.js 14

## Global Constraints
- Keep code files highly focused.
- All outbound requests must traverse `FailSafeRoundTripper` in Go, or equivalent blockers in Python.
- Database operations must be transactional.

---

### Task 1: Phase 2 - E-Commerce Stateful Mocking & Storefront Adapters

**Files:**
- Create: `backend-go/internal/domain/integrations/mock_storefront.go`
- Create: `backend-go/internal/domain/integrations/mock_storefront_test.go`
- Modify: `backend-go/internal/domain/integrations/routes.go`

- [ ] **Step 1: Write GORM mock schemas and seeder logic**
  Implement `mock_listings` database table and GORM model inside `backend-go/internal/domain/integrations/mock_storefront.go`.
  Write transactional seeding handler using advisory locks:
  ```go
  func (s *MockService) SeedStorefront(db *gorm.DB) error {
      return db.Transaction(func(tx *gorm.DB) error {
          if err := tx.Exec("SELECT pg_advisory_xact_lock(777888)").Error; err != nil {
              return err
          }
          // Seeding records
          return tx.FirstOrCreate(&MockListing{SKU: "SKU-TEST-123", PriceCents: 1999}).Error
      })
  }
  ```

- [ ] **Step 2: Implement Shopee and Ozon Mock Adapter Interfaces**
  Provide mock implementations of `PlatformAdapter` interface returning simulated SKU details, stock changes, and shipping rates based on GORM mock tables.

- [ ] **Step 3: Write tests verifying stateful updates**
  Add unit tests in `mock_storefront_test.go` verifying that publishing a product changes the mock table listing status from `suggested` to `live`, and updates stock levels correctly.

- [ ] **Step 4: Verify tests pass**
  Run: `cd backend-go && go test -v ./internal/domain/integrations`
  Expected: PASS

- [ ] **Step 5: Commit**
  Run: `git add backend-go/internal/domain/integrations` && `git commit -m "feat(integrations): add stateful mock storefront adapters"`

---

### Task 2: Phase 3 - Python Cognitive Microservice (GEPA & DSPy)

**Files:**
- Create: `python-agentos/main.py`
- Create: `python-agentos/memory.py`
- Create: `python-agentos/requirements.txt`
- Create: `python-agentos/test_memory.py`

- [ ] **Step 1: Write python dependency requirements**
  Create `python-agentos/requirements.txt`:
  ```txt
  fastapi==0.110.0
  uvicorn==0.28.0
  sqlite-vec==0.2.0
  pydantic==2.6.4
  ```

- [ ] **Step 2: Implement sqlite-vec memory database**
  Create `python-agentos/memory.py` implementing:
  - Working memory, Episodic memory, Semantic memory tables.
  - SQLite extension loading for `sqlite_vec`.
  - Methods to save and query episodic traces based on text embedding similarity.

- [ ] **Step 3: Implement cognitive brain endpoints**
  Create `python-agentos/main.py` launching a FastAPI server exposing `/api/v1/brain/decide` and `/api/v1/brain/reflect`. Endpoint processes context from Go backend and queries episodic memory.

- [ ] **Step 4: Run unit tests**
  Create `python-agentos/test_memory.py` asserting that saving a successful trace and searching for it returns the exact matching context. Run using `python -m unittest python-agentos/test_memory.py`.
  Expected: PASS

- [ ] **Step 5: Commit**
  Run: `git add python-agentos/` && `git commit -m "feat(cognitive): add Python cognitive brain microservice with sqlite-vec"`

---

### Task 3: Phase 4 - Finance & Foreign Trade Domains (Go Backend)

**Files:**
- Create: `backend-go/internal/domain/finance/model.go`
- Create: `backend-go/internal/domain/finance/service.go`
- Create: `backend-go/internal/domain/foreigntrade/model.go`
- Create: `backend-go/internal/domain/foreigntrade/service.go`
- Create: `backend-go/internal/domain/finance/finance_test.go`

- [ ] **Step 1: Define BankAdapter & RFQAdapter Contracts**
  Define implicit Go interfaces:
  `BankAdapter`: `ExecuteTransfer(ctx, req) (*TransferResponse, error)`
  `RFQAdapter`: `SubmitQuotation(ctx, rfqID, quote) (*QuotationResult, error)`

- [ ] **Step 2: Create GORM database schemas**
  Create financial ledger models (`LedgerEntry` with `PriceCents` int64 fields) and RFQ quotation models (`RFQRecord`, `Quotation`).

- [ ] **Step 3: Implement Service handlers**
  Implement transaction-safe money transfers and quotation generation in service structs. Inject these into domain routers.

- [ ] **Step 4: Verify test suite**
  Add unit tests in `finance_test.go` asserting ledger accounting balance validation (credits equal debits) and RFQ quotations formatting.
  Run: `cd backend-go && go test -v ./internal/domain/finance`
  Expected: PASS

- [ ] **Step 5: Commit**
  Run: `git add backend-go/internal/domain/finance backend-go/internal/domain/foreigntrade` && `git commit -m "feat(domains): implement finance and foreign trade domain suites"`
