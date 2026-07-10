# Multi-Domain AgentOS: Self-Evolving & AI-Maintained Systems Design

> **Document Status**: Proposal Spec (Design Stage - Peer Reviewed & Updated)
> **Target Audience**: AI Coding Agents (Developers/Maintainers) & Human Owner (Approver)
> **Creation Date**: 2026-07-09
> **Objective**: Define the technical foundation and evolution roadmap of LingMirror (MultiSell) as a general-purpose, enterprise-grade AI AgentOS developed 100% by AI coding agents.

---

## 1. Executive Summary & Vision

LingMirror is an **Operating System for Enterprise Business AI Agents (AgentOS)**.
- **The Core Business Assumption**: The system is designed to plug in different industry suites (E-Commerce, Finance, Foreign Trade) without changing the core platform kernel.
- **The Development Assumption**: The system is developed, researched, and maintained **entirely by AI coding agents** (e.g., Google Antigravity) pair-programming with a non-technical Owner.

To ensure this model is robust, scalable, and completely safe, the system implements a **Four-Pillar Architecture**:
1. **AI-Native Context Synchronization (ARCS)**: Prevents AI developer memory loss across stateless chat sessions using compact, low-token manifests and handoff ledgers.
2. **Automated Staging Sandboxes**: Validates all AI commits via Playwright E2E and Go unit tests in fully isolated Docker Compose networks.
3. **Stateful Mock Driver Adapter Layer & Outbound Gates**: Permits safe, realistic end-to-end dry-runs without live credential leakage or external account flags, protected by both network-level and HTTP-level blockers.
4. **AI Edit Loop Prevention**: Monitors AI developers, detects ping-pong debugging loops, error stagnation, or random walk non-convergence, and enforces safe rollbacks to the Owner.

---

## 2. Pillar 1: AI-Native Context Synchronization (ARCS)

When AI agents maintain a codebase, they suffer from **Session Amnesia** and **Context Limits**. ARCS provides machine-readable metadata maps for instant alignment.

```
Incoming Agent ──> [Read .ai-manifest.json] ──> [Read .loop/dev-state.md] ──> [Query CodeGraph] ──> Complete Scope
```

### 2.1 The Root Index File: `.ai-manifest.json`
Located at `/`, this tells incoming agents the active tech stacks, safe boundaries, and index directories:
```json
{
  "manifest_version": "1.0.0",
  "project_name": "LingMirror (MultiSell)",
  "active_stack": {
    "backend": "Go 1.21 / Gin",
    "frontend": "Next.js 14 / App Router / AntD",
    "database": "PostgreSQL 15 / GORM v2",
    "event_bus": "In-memory Publisher-Subscriber (internal/platform/eventbus)"
  },
  "entrypoints": {
    "backend_main": "backend-go/cmd/server/main.go",
    "frontend_root": "frontend-next/src/app/"
  },
  "core_indexes": {
    "module_catalog": "docs/reference-module-catalog.md",
    "documentation_index": "docs/INDEX.md",
    "governance_directory": "docs/governance/",
    "session_state": ".loop/dev-state.md",
    "codegraph_db": ".codegraph/codegraph.db",
    "agent_instructions": "AGENTS.md",
    "claude_instructions": "CLAUDE.md"
  },
  "safe_boundaries": {
    "risk_manifest": "docs/governance/PLATFORM_CONSTITUTION.md",
    "high_risk_layers": ["price", "inventory", "order", "finance", "audit", "auth"]
  }
}
```

### 2.2 The Handoff Ledger: `.loop/dev-state.md`
Tracks only the **1-2 most recent active task slices** to prevent context window inflation, archiving older entries to `.loop/history.md`.
```markdown
# AI Developer Handoff Ledger

- **Current Goal**: "Implement Stripe Mock Adapter for Finance suite"
- **Active Task Slice**: "Verify payment billing ledger updates on mocked webhooks"
- **Completed in Branch**:
  - Added mock_transactions table to Go GORM models.
  - Implemented MockPaymentAdapter.Charge handler.
- **Verification Results**:
  - `go test ./internal/domain/payment`: PASS (12/12)
  - `npm run lint`: PASS (0 errors, 2 warnings)
  - `npm run build`: PASS
- **New System Delta**:
  - API added: `POST /api/v1/mock/payment/charge`
  - Event published: `mock.payment.charged`
- **Open Debt/Unresolved Warnings**:
  - Vite compilation warning in `payment-ledger.tsx:L42` regarding unhandled promise.
- **Next Step**: "Write E2E Playwright check for payment-ledger UI status display."
```

### 2.3 The Single Source of Truth: `docs/reference-module-catalog.md`
A catalog mapping all APIs, EventBus topics, and frontend routes.
- **Linter Enforced (DSL)**: A pre-commit hook runs AST checks to compare Gin/Next.js routes and GORM models against the catalog. If they are out of sync, the build fails.

---

## 3. Pillar 2: Automated Staging Sandbox & Test Gatekeeper

To prevent AI coding bugs from entering production, the Gatekeeper spins up an isolated Docker sandbox on every PR.

### 3.1 Docker Staging Stack (`docker-compose.sandbox.yml`)
To avoid dynamic YAML preprocessing, service names are static. Sandboxes are isolated using Docker Compose's built-in namespace flag: `docker compose -p sandbox-pr-${SANDBOX_ID} -f docker-compose.sandbox.yml up`.

* **Zero Host Port Exposure**: All communication occurs inside the Docker internal network to prevent port collisions on concurrent host runs.
* **Shared Compiler Caches**: Mounts host directories in read-only/concurrent mode to speed up container build times:
  - `- ~/.go-pkg-cache:/go/pkg/mod` (Go modules)
  - `- ~/.npm-cache:/root/.npm` (NPM packages)
* **Isolated Workspaces**: The host runs each sandbox in an isolated Git worktree: `/tmp/sandboxes/pr-${SANDBOX_ID}`.

```yaml
version: '3.8'

networks:
  sandbox_internal:
    internal: true # Strict firewall cutting off internet access for database & backend
  traefik_public:
    external: true

services:
  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: multisell_sandbox
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    networks:
      - sandbox_internal
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d multisell_sandbox"]
      interval: 3s
      timeout: 3s
      retries: 5

  migrate:
    image: migrate/migrate:v4.18.1
    depends_on:
      db:
        condition: service_healthy
    volumes:
      - ./backend-go/migrations:/migrations:ro
    networks:
      - sandbox_internal
    command:
      - -path
      - /migrations
      - -database
      - postgresql://postgres:${DB_PASSWORD}@db:5432/multisell_sandbox?sslmode=disable
      - up

  backend:
    image: golang:1.25
    working_dir: /app
    volumes:
      - ./backend-go:/app
      - ~/.go-pkg-cache:/go/pkg/mod:ro # Read-only Go module cache
    environment:
      DB_HOST: db
      DB_PORT: "5432"
      DB_USER: postgres
      DB_PASSWORD: ${DB_PASSWORD}
      DB_NAME: multisell_sandbox
      JWT_SECRET: ${JWT_SECRET}
      SERVER_PORT: "8080"
    networks:
      - sandbox_internal
    depends_on:
      db:
        condition: service_healthy
      migrate:
        condition: service_completed_successfully
    command: >
      sh -c "go mod download && go run cmd/server/main.go"

  seed:
    image: golang:1.25
    working_dir: /work
    volumes:
      - .:/work
      - ~/.go-pkg-cache:/go/pkg/mod:ro
    environment:
      DB_HOST: db
      DB_PORT: "5432"
      DB_USER: postgres
      DB_PASSWORD: ${DB_PASSWORD}
      DB_NAME: multisell_sandbox
    networks:
      - sandbox_internal
    depends_on:
      migrate:
        condition: service_completed_successfully
    command: ["bash", "scripts/e2e_seed.sh"]

  frontend:
    image: node:22
    working_dir: /app
    volumes:
      - ./frontend-next:/app
      - ~/.npm-cache:/root/.npm:ro # Read-only NPM cache
    environment:
      NEXT_PUBLIC_API_URL: http://backend:8080/api
    networks:
      - sandbox_internal
      - traefik_public
    depends_on:
      - backend
    command: >
      sh -c "npm ci && npm run dev -- --hostname 0.0.0.0 --port 3000"
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=traefik_public"
      - "traefik.http.routers.frontend-pr-${SANDBOX_ID}.rule=Host(`pr-${SANDBOX_ID}.staging.lingmirror.com`)"
      - "traefik.http.routers.frontend-pr-${SANDBOX_ID}.service=frontend-pr-${SANDBOX_ID}"
      - "traefik.http.services.frontend-pr-${SANDBOX_ID}.loadbalancer.server.port=3000"

  e2e:
    image: mcr.microsoft.com/playwright:v1.55.0-jammy
    working_dir: /work/frontend-next/e2e
    volumes:
      - .:/work
    environment:
      E2E_BASE_URL: http://frontend:3000
      E2E_API_BASE: http://backend:8080
      E2E_SKIP_WEB_SERVER: "1"
    networks:
      - sandbox_internal
    depends_on:
      frontend:
        condition: service_started
      backend:
        condition: service_started
      seed:
        condition: service_completed_successfully
    command: >
      bash -lc "npm ci &&
                for i in $$(seq 1 90); do
                  if curl -fsS http://frontend:3000/login >/dev/null && curl -fsS http://backend:8080/api/health >/dev/null; then
                    break
                  fi
                  sleep 1
                done &&
                npm run e2e"
```

---

## 4. Pillar 3: Stateful Mock Driver Layer & Outbound Gates

AI agents must run stateful E2E tests. Stateless mocks fail here. We use a local **Mock Database Store** combined with a **Hard Outbound Traffic Gate** to prevent live API calls.

### 4.1 Schema Mappings & Numerical Precision (GORM)
String columns use explicit GORM tags to match migration schemas and prevent drift warnings. Financial values use integer cents (`int64`) or GORM Decimal types to avoid float rounding issues.

```go
package mock

import (
	"time"
)

// MockListing represents product listings on storefronts (Ozon/Shopee)
type MockListing struct {
	ID                int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	PlatformID        int64     `gorm:"column:platform_id;not null" json:"platform_id"`
	AccountID         int64     `gorm:"column:account_id;not null" json:"account_id"`
	ProductID         int64     `gorm:"column:product_id;not null" json:"product_id"`
	PlatformProductID string    `gorm:"column:platform_product_id;type:varchar(100);uniqueIndex;not null" json:"platform_product_id"`
	Status            string    `gorm:"column:status;type:varchar(20);default:'live';not null" json:"status"` // live, suspended
	PriceCents        int64     `gorm:"column:price_cents;not null" json:"price_cents"`                       // Money stored as cents
	Stock             int       `gorm:"column:stock;default:0" json:"stock"`
	CreatedAt         time.Time `gorm:"column:created_at"`
}

// MockShipment represents logistics transactions
type MockShipment struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID        int64     `gorm:"column:order_id;not null;uniqueIndex" json:"order_id"`
	TrackingNumber string    `gorm:"column:tracking_number;type:varchar(100);uniqueIndex;not null" json:"tracking_number"`
	LabelURL       string    `gorm:"column:label_url;type:varchar(512)" json:"label_url"`
	Status         string    `gorm:"column:status;type:varchar(20);default:'pending';not null" json:"status"` // pending, picked_up, delivered
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}
```

### 4.2 Seed Concurrency with Transactional Advisory Locks
Seeding demo data uses transactional advisory locks to prevent race conditions during parallel test triggers:

```go
package mock

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func (s *Service) SeedMockData() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Acquire PostgreSQL transactional advisory lock to prevent parallel seeder execution
		if err := tx.Exec("SELECT pg_advisory_xact_lock(123456)").Error; err != nil {
			return fmt.Errorf("failed to acquire advisory lock: %w", err)
		}

		var count int64
		if err := tx.Model(&MockListing{}).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil // Already seeded
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

### 4.3 Fail-Safe Outbound Network Gate (`FailSafeRoundTripper`)
To prevent outgoing API calls from custom client instantiations, adapters must fetch clients from a central constructor in `httpx` containing the thread-safe `FailSafeRoundTripper`.

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

	// Dynamic IP check to automatically allow local subnets / private loops
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

## 5. Pillar 4: AI Development Loop Monitor & Prevention

To prevent AI developers from getting stuck in infinite debugging/editing loops, the **AIOS Observability** module tracks code states, compiler outputs, and test runner exceptions.

### 5.1 Code State Space
$$\text{State}_t = \langle \text{GitDiffHash}_t, \text{ErrorSignatureHash}_t, \text{ErrorCount}_t \rangle$$

* `GitDiffHash`: SHA-256 hash of the git diff from a *fixed baseline* established at the start of the session. If diff returns to baseline (e.g. reset), it hashes to empty string hash.
* `ErrorSignatureHash`: Hashed normalized output of compiler AND test failures. Normalization applies regex replacements in sequence to strip dynamic timestamps, absolute paths, container IDs, and pointer addresses.

### 5.2 The Production-Grade Go Implementation (`observability/loop_detector.go`)

```go
package observability

import (
	"crypto/sha256"
	"regexp"
	"strings"
	"sync"
)

type LoopType string

const (
	LoopNone            LoopType = "None"
	LoopPingPong        LoopType = "PingPong"        // Recurrence of code states (oscillation)
	LoopErrorStagnation LoopType = "ErrorStagnation"  // Edits made but error signature remains identical
	LoopErrorOscillate  LoopType = "ErrorOscillate"   // Alternating between two different errors
	LoopCostLimit       LoopType = "CostLimitExceeded" // Total API spending exceeds threshold
)

type CodeState struct {
	GitDiffHash        [32]byte
	ErrorSignatureHash [32]byte
	ErrorCount         int
}

type Config struct {
	MaxCostUSD         float64
	PingPongWindow     int
	StagnationWindow   int
	OscillationWindow  int
}

type LoopDetector struct {
	mu             sync.Mutex
	cfg            Config
	history        []CodeState
	cumulativeCost float64

	pathRegex      *regexp.Regexp
	ptrRegex       *regexp.Regexp
	lineRegex      *regexp.Regexp
	timestampRegex *regexp.Regexp
	timeRegex      *regexp.Regexp
	uuidRegex      *regexp.Regexp
	sandboxRegex   *regexp.Regexp
}

func NewLoopDetector(cfg Config) *LoopDetector {
	if cfg.PingPongWindow <= 0 {
		cfg.PingPongWindow = 6
	}
	if cfg.StagnationWindow <= 0 {
		cfg.StagnationWindow = 3
	}
	if cfg.OscillationWindow <= 0 {
		cfg.OscillationWindow = 4
	}
	if cfg.MaxCostUSD <= 0 {
		cfg.MaxCostUSD = 2.00
	}

	return &LoopDetector{
		cfg:            cfg,
		history:        make([]CodeState, 0),
		cumulativeCost: 0.0,
		pathRegex:      regexp.MustCompile(`/[a-zA-Z0-9_\-\.\+]+(?:/[a-zA-Z0-9_\-\.\+]+)+`),
		ptrRegex:       regexp.MustCompile(`\b0x[0-9a-fA-F]+\b`),
		lineRegex:      regexp.MustCompile(`(?::\d+)+`),
		timestampRegex: regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?\b`),
		timeRegex:      regexp.MustCompile(`\b\d{2}:\d{2}:\d{2}(?:\.\d+)?\b`),
		uuidRegex:      regexp.MustCompile(`\b[0-9a-fA-F]{8}(?:-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}\b`),
		sandboxRegex:   regexp.MustCompile(`\b(?:db|backend|frontend|migrate|seed|e2e)-[a-zA-Z0-9_\-]+\b`),
	}
}

func (ld *LoopDetector) NormalizeError(errStr string) string {
	if errStr == "" {
		return ""
	}
	s := strings.ReplaceAll(errStr, "\\", "/")
	s = ld.timestampRegex.ReplaceAllString(s, "<timestamp>")
	s = ld.timeRegex.ReplaceAllString(s, "<time>")
	s = ld.uuidRegex.ReplaceAllString(s, "<uuid>")
	s = ld.sandboxRegex.ReplaceAllString(s, "<sandbox-service>")
	s = ld.ptrRegex.ReplaceAllString(s, "<ptr>")
	s = ld.pathRegex.ReplaceAllString(s, "<path>")
	s = ld.lineRegex.ReplaceAllString(s, ":<line>")
	return s
}

func (ld *LoopDetector) RecordStep(gitDiff string, errOutput string, errorCount int, stepCost float64) (LoopType, bool) {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	ld.cumulativeCost += stepCost
	if ld.cumulativeCost > ld.cfg.MaxCostUSD {
		return LoopCostLimit, true
	}

	diffHash := sha256.Sum256([]byte(gitDiff))
	normalizedErr := ld.NormalizeError(errOutput)
	errHash := sha256.Sum256([]byte(normalizedErr))

	state := CodeState{
		GitDiffHash:        diffHash,
		ErrorSignatureHash: errHash,
		ErrorCount:         errorCount,
	}

	ld.history = append(ld.history, state)
	t := len(ld.history) - 1

	// Avoid memory leak when slicing arrays in Go.
	maxWindow := ld.cfg.PingPongWindow
	if ld.cfg.OscillationWindow > maxWindow {
		maxWindow = ld.cfg.OscillationWindow
	}
	if len(ld.history) > maxWindow*3 {
		newHistory := make([]CodeState, maxWindow*2)
		copy(newHistory, ld.history[len(ld.history)-maxWindow*2:])
		ld.history = newHistory
		t = len(ld.history) - 1
	}

	if t < 1 {
		return LoopNone, false
	}

	// 1. Ping-Pong Loop Check (diff oscillation)
	startIdx := t - ld.cfg.PingPongWindow
	if startIdx < 0 {
		startIdx = 0
	}
	for i := startIdx; i < t; i++ {
		// Only check if we are actually editing code (diff is not empty)
		if ld.history[t].GitDiffHash == ld.history[i].GitDiffHash && ld.history[t].GitDiffHash != [32]byte{} {
			return LoopPingPong, true
		}
	}

	// 2. Error Stagnation Check
	if len(ld.history) >= ld.cfg.StagnationWindow {
		stagnant := true
		firstErrHash := ld.history[t].ErrorSignatureHash
		firstErrCount := ld.history[t].ErrorCount

		if firstErrCount > 0 { // Do not trigger on success states
			for i := t - ld.cfg.StagnationWindow + 1; i <= t; i++ {
				if ld.history[i].ErrorSignatureHash != firstErrHash {
					stagnant = false
					break
				}
				if i > t-ld.cfg.StagnationWindow+1 {
					if ld.history[i].GitDiffHash == ld.history[i-1].GitDiffHash {
						stagnant = false
						break
					}
				}
			}
			if stagnant {
				return LoopErrorStagnation, true
			}
		}
	}

	// 3. Error Oscillation Check (Ping-ponging between two distinct errors)
	if len(ld.history) >= 4 {
		currErr := ld.history[t].ErrorSignatureHash
		prevErr := ld.history[t-1].ErrorSignatureHash
		twoBackErr := ld.history[t-2].ErrorSignatureHash
		threeBackErr := ld.history[t-3].ErrorSignatureHash

		if currErr == twoBackErr && prevErr == threeBackErr && currErr != prevErr {
			if ld.history[t].GitDiffHash != ld.history[t-1].GitDiffHash &&
				ld.history[t-1].GitDiffHash != ld.history[t-2].GitDiffHash {
				if ld.history[t].ErrorCount > 0 && ld.history[t-1].ErrorCount > 0 {
					return LoopErrorOscillate, true
				}
			}
		}
	}

	return LoopNone, false
}
```

---

## 6. Multi-Domain Pluggable Mapping (Interface Segregation)

To prevent Interface Segregation Principle (ISP) violations, we split the platform adapters into distinct interfaces:

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

---

## 7. Strategic Multi-Phase Evolution Roadmap

```
Phase P0: Trusted Safety Gate (Current)
   └── Secure EventBus, Action Gate, & RBAC (Go Base)
Phase P1: Staging Sandbox & Safety Guards
   └── Deploy Docker Sandbox pipeline (internal networking, shared caches)
   └── Write unit tests verifying loop_detector.go behavior & Fail-Safe triggers
Phase P2: Double Business Loop Integration (E-Commerce)
   └── Run stateful mock listing publishes & transactional seeder runs
Phase P3: Multi-Domain Framework Extension
   └── Implement Finance (BankAdapter) & Trade (RFQAdapter) suites
```
