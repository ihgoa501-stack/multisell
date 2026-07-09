# AgentOS Sandbox and Safety Guards Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish the safety runtime infrastructure for AI-driven development on the Multi-Domain AgentOS, including ARCS context synchronization files, a fail-safe outbound HTTP transport blocker, a Docker-compose dynamic sandbox running Playwright E2E, and the observational AI edit loop detector.

**Architecture:**
- The system core metadata is synchronized through standard JSON and Markdown ledger indices at root.
- All HTTP requests to external APIs are routed through a mutex-secured, private-subnet-aware HTTP RoundTripper, supplemented by Docker network-level isolation (`internal: true`).
- AI edit cycles are recorded, normalized (stripping paths/lines/pointers), and checked for ping-pong, stagnation, or alternating oscillation loops using SHA-256 state hashes.

**Tech Stack:** Go (GORM/Gin), Node (Next.js/Playwright), Docker Compose, PostgreSQL.

## Global Constraints
- Target workspace directory: `/Users/lc/multisell`
- Backend source: `backend-go`
- Frontend source: `frontend-next`
- Single file length constraint: All newly created files should be modular and kept under **300 lines** where possible.
- Hardcoded secrets check: No hardcoded API keys, JWT secrets, or database passwords. All secrets are injected dynamically via environment variables in Compose.
- GORM schema alignment: All GORM models must align explicitly with PostgreSQL migration types to prevent drift warnings.

---

## Tasks

### Task 1: AI-Native Context Synchronization (ARCS) Setup & Linter

**Files:**
- Create: `.ai-manifest.json`
- Create: `.loop/dev-state.md`
- Create: `scripts/check_module_catalog.sh`

**Interfaces:**
- Produces: Metadata index and pre-commit check verifying that APIs and GORM schemas are cataloged.

- [ ] **Step 1: Create the root AI manifest file**
  Create `.ai-manifest.json` at the root directory:
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

- [ ] **Step 2: Create the dynamic session state ledger**
  Create `.loop/dev-state.md` with the current starting state:
  ```markdown
  # AI Developer Handoff Ledger

  - **Current Goal**: "Implement AgentOS Sandbox and Safety Guards"
  - **Active Task Slice**: "Task 1: ARCS Setup & Linter"
  - **Completed in Branch**:
    - [x] Initialized `.ai-manifest.json`
  - **Verification Results**:
    - Build: N/A
  - **New System Delta**:
    - Meta: Added ARCS manifest mapping
  - **Open Debt/Unresolved Warnings**:
    - None
  - **Next Step**: "Implement Fail-Safe Outbound Network Gates in Go"
  ```

- [ ] **Step 3: Create the Documentation Sync Linter (DSL) script**
  Create `scripts/check_module_catalog.sh` to parse API endpoints in Gin handler registrations and verify they are documented in `docs/reference-module-catalog.md`:
  ```bash
  #!/usr/bin/env bash
  set -euo pipefail

  CATALOG="docs/reference-module-catalog.md"
  if [ ! -f "$CATALOG" ]; then
    echo "Error: Catalog file $CATALOG not found"
    exit 1
  fi

  echo "Verifying API endpoints in docs/reference-module-catalog.md..."
  # Extract endpoints defined in backend-go/internal/domain (finding patterns like r.GET("/path") or r.POST("/path"))
  grep -r -h -o -E '"/api/v1/[a-zA-Z0-9_\-/:]+"' backend-go/internal/ | tr -d '"' | sort -u | while read -r endpoint; do
    if ! grep -q "$endpoint" "$CATALOG"; then
      echo "Error: API endpoint $endpoint is not documented in $CATALOG!"
      exit 1
    fi
  done

  echo "Verification complete. All Gin endpoints are documented."
  ```

- [ ] **Step 4: Verify the linter fails on missing documentation**
  Temporarily add a mock route registration containing `r.GET("/api/v1/mock/unregistered-test")` in `backend-go/internal/domain/sourcing/routes.go` and run the script:
  Run: `bash scripts/check_module_catalog.sh`
  Expected: Exit code 1 with "Error: API endpoint /api/v1/mock/unregistered-test is not documented in docs/reference-module-catalog.md!"
  Then revert the mock route addition.

- [ ] **Step 5: Run the linter to verify successful pass**
  Run: `bash scripts/check_module_catalog.sh`
  Expected: Exit code 0, outputting "Verification complete. All Gin endpoints are documented."

- [ ] **Step 6: Commit**
  Run:
  ```bash
  git add .ai-manifest.json .loop/dev-state.md scripts/check_module_catalog.sh
  git commit -m "feat(arcs): add ARCS metadata and documentation sync linter"
  ```

---

### Task 2: Fail-Safe Outbound Network Gate & HTTP Client Interceptor

**Files:**
- Create: `backend-go/internal/httpx/failsafe_transport.go`
- Modify: `backend-go/internal/httpx/client.go`
- Create: `backend-go/internal/httpx/failsafe_transport_test.go`

**Interfaces:**
- Produces: `httpx.NewClient(env string) *http.Client` which embeds the thread-safe `FailSafeRoundTripper`.

- [ ] **Step 1: Write the failing unit test for network gates**
  Create `backend-go/internal/httpx/failsafe_transport_test.go`:
  ```go
  package httpx

  import (
  	"net/http"
  	"testing"
  )

  func TestFailSafeRoundTripper_Blocked(t *testing.T) {
  	transport := NewFailSafeRoundTripper(nil, "development")
  	client := &http.Client{Transport: transport}

  	_, err := client.Get("https://api.ozon.ru/v1/products")
  	if err == nil {
  		t.Fatal("Expected outbound request to api.ozon.ru to be blocked in development mode, but it succeeded")
  	}
  	if !strings.Contains(err.Error(), "blocked outbound request") {
  		t.Fatalf("Expected blocked outbound request error, got: %v", err)
  	}
  }

  func TestFailSafeRoundTripper_AllowedPrivate(t *testing.T) {
  	transport := NewFailSafeRoundTripper(nil, "development")
  	client := &http.Client{Transport: transport}

  	// Loopback/Private subnet addresses should pass the interceptor
  	_, err := client.Get("http://127.0.0.1:8080/api/health")
  	// The request should fail due to dial connection refused (no server listening), but NOT due to block gate
  	if err != nil && strings.Contains(err.Error(), "blocked outbound request") {
  		t.Fatalf("Expected loopback to bypass gate, but it was blocked: %v", err)
  	}
  }
  ```

- [ ] **Step 2: Run the test to verify compilation failure**
  Run: `cd backend-go && go test -v ./internal/httpx`
  Expected: Fails to compile due to missing `NewFailSafeRoundTripper`.

- [ ] **Step 3: Implement `FailSafeRoundTripper`**
  Create `backend-go/internal/httpx/failsafe_transport.go`:
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

- [ ] **Step 4: Integrate the transport into the central Client constructor**
  Verify if `backend-go/internal/httpx/client.go` exists. If not, create it. Otherwise, modify it to export `NewClient` wrapping the transport:
  ```go
  package httpx

  import (
  	"net/http"
  	"time"
  )

  // NewClient creates a standard HTTP Client pre-loaded with outbound safety barriers.
  func NewClient(env string, timeout time.Duration) *http.Client {
  	transport := NewFailSafeRoundTripper(http.DefaultTransport, env)
  	return &http.Client{
  		Transport: transport,
  		Timeout:   timeout,
  	}
  }
  ```

- [ ] **Step 5: Run tests and verify success**
  Run: `cd backend-go && go test -v ./internal/httpx`
  Expected: PASS

- [ ] **Step 6: Commit**
  Run:
  ```bash
  git add backend-go/internal/httpx/failsafe_transport.go backend-go/internal/httpx/failsafe_transport_test.go backend-go/internal/httpx/client.go
  git commit -m "feat(safety): implement FailSafeRoundTripper outbound gate"
  ```

---

### Task 3: Loop Detector Implementation in Go

**Files:**
- Create: `backend-go/internal/aios/observability/loop_detector.go`
- Create: `backend-go/internal/aios/observability/loop_detector_test.go`

**Interfaces:**
- Produces: `observability.NewLoopDetector(cfg Config) *LoopDetector`
- Produces: `(ld *LoopDetector) RecordStep(gitDiff, errOutput, errorCount, stepCost) (LoopType, bool)`

- [ ] **Step 1: Write loop detector unit tests**
  Create `backend-go/internal/aios/observability/loop_detector_test.go` checking all four loop conditions (Ping-pong, Same-Error stagnation, Error oscillation, Cost limit):
  ```go
  package observability

  import (
  	"testing"
  )

  func TestLoopDetector_PingPong(t *testing.T) {
  	ld := NewLoopDetector(Config{PingPongWindow: 3, MaxCostUSD: 1.00})

  	// Step 1: Baseline edit
  	_, blocked := ld.RecordStep("diffA", "errorA", 1, 0.01)
  	if blocked {
  		t.Fatal("Unexpected block on first step")
  	}

  	// Step 2: Second code state
  	_, blocked = ld.RecordStep("diffB", "errorB", 1, 0.01)
  	if blocked {
  		t.Fatal("Unexpected block on second step")
  	}

  	// Step 3: Revert to first code state (ping-pong oscillation)
  	loopType, blocked := ld.RecordStep("diffA", "errorA", 1, 0.01)
  	if !blocked || loopType != LoopPingPong {
  		t.Fatalf("Expected PingPong loop detection, got loopType=%s blocked=%t", loopType, blocked)
  	}
  }

  func TestLoopDetector_SameErrorStagnation(t *testing.T) {
  	ld := NewLoopDetector(Config{StagnationWindow: 3, MaxCostUSD: 1.00})

  	// Step 1: Error output containing dynamic pointer and path
  	_, blocked := ld.RecordStep("diffA", "error at /Users/lc/multisell/main.go:12: struct 0xc0001", 1, 0.01)
  	if blocked {
  		t.Fatal("Unexpected block")
  	}

  	// Step 2: Altered diff, but identical normalized error (line number shifted, pointer changed)
  	_, blocked = ld.RecordStep("diffB", "error at /Users/lc/multisell/main.go:15: struct 0xc0002", 1, 0.01)
  	if blocked {
  		t.Fatal("Unexpected block")
  	}

  	// Step 3: Altered diff again, identical normalized error
  	loopType, blocked := ld.RecordStep("diffC", "error at /Users/lc/multisell/main.go:18: struct 0xc0003", 1, 0.01)
  	if !blocked || loopType != LoopErrorStagnation {
  		t.Fatalf("Expected Same-Error Stagnation detection, got loopType=%s blocked=%t", loopType, blocked)
  	}
  }
  ```

- [ ] **Step 2: Run tests to verify compilation failure**
  Run: `cd backend-go && go test -v ./internal/aios/observability`
  Expected: Fails to compile due to missing package `observability` or missing structs.

- [ ] **Step 3: Implement `loop_detector.go`**
  Create `backend-go/internal/aios/observability/loop_detector.go`. Use the exact code from the peer-reviewed spec:
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
  	LoopPingPong        LoopType = "PingPong"
  	LoopErrorStagnation LoopType = "ErrorStagnation"
  	LoopErrorOscillate  LoopType = "ErrorOscillate"
  	LoopCostLimit       LoopType = "CostLimitExceeded"
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

  	startIdx := t - ld.cfg.PingPongWindow
  	if startIdx < 0 {
  		startIdx = 0
  	}
  	for i := startIdx; i < t; i++ {
  		if ld.history[t].GitDiffHash == ld.history[i].GitDiffHash && ld.history[t].GitDiffHash != [32]byte{} {
  			return LoopPingPong, true
  		}
  	}

  	if len(ld.history) >= ld.cfg.StagnationWindow {
  		stagnant := true
  		firstErrHash := ld.history[t].ErrorSignatureHash
  		firstErrCount := ld.history[t].ErrorCount

  		if firstErrCount > 0 {
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

- [ ] **Step 4: Run loop detector tests and verify success**
  Run: `cd backend-go && go test -v ./internal/aios/observability`
  Expected: PASS

- [ ] **Step 5: Commit**
  Run:
  ```bash
  git add backend-go/internal/aios/observability/loop_detector.go backend-go/internal/aios/observability/loop_detector_test.go
  git commit -m "feat(observability): add loop_detector and verification tests"
  ```

---

### Task 4: Dynamic Sandbox Configuration Setup

**Files:**
- Create: `docker-compose.sandbox.yml`
- Create: `scripts/run_sandbox.sh`

**Interfaces:**
- Produces: `scripts/run_sandbox.sh <sandbox-id>` which spawns a git worktree, starts compose with internal isolation, runs e2e tests, and collects results.

- [ ] **Step 1: Write the sandbox Docker Compose configuration**
  Create `docker-compose.sandbox.yml` exactly matching the peer-reviewed design (Section 3.1 of spec) at root:
  ```yaml
  version: '3.8'

  networks:
    sandbox_internal:
      internal: true
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
        - ~/.go-pkg-cache:/go/pkg/mod:ro
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
        - ~/.npm-cache:/root/.npm:ro
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

- [ ] **Step 2: Create the sandbox execution wrapper script**
  Create `scripts/run_sandbox.sh` to run the sandbox setup, tests, and cleanup:
  ```bash
  #!/usr/bin/env bash
  set -euo pipefail

  SANDBOX_ID="${1:-}"
  if [ -z "$SANDBOX_ID" ]; then
    echo "Error: Please specify a sandbox ID"
    echo "Usage: $0 <sandbox-id>"
    exit 1
  fi

  export SANDBOX_ID
  export DB_PASSWORD=$(openssl rand -hex 16)
  export JWT_SECRET=$(openssl rand -hex 32)
  WORKTREE_DIR="/tmp/sandboxes/pr-${SANDBOX_ID}"

  echo "Allocating worktree in $WORKTREE_DIR..."
  rm -rf "$WORKTREE_DIR"
  git worktree add "$WORKTREE_DIR" HEAD

  cd "$WORKTREE_DIR"

  echo "Spinning up isolated Docker Compose namespace sandbox..."
  # Clean up any residual containers with the same namespace
  docker compose -p "sandbox-pr-${SANDBOX_ID}" -f docker-compose.sandbox.yml down -v --remove-orphans || true

  # Run the e2e container and wait for exit status
  set +e
  docker compose -p "sandbox-pr-${SANDBOX_ID}" -f docker-compose.sandbox.yml up --abort-on-container-exit --exit-code-from e2e e2e
  EXIT_CODE=$?
  set -e

  echo "Copying Playwright test reports out of container network..."
  docker cp "$(docker compose -p "sandbox-pr-${SANDBOX_ID}" -f docker-compose.sandbox.yml ps -q e2e)":/work/frontend-next/e2e/playwright-report /tmp/reports/pr-${SANDBOX_ID} || true

  echo "Tearing down sandbox containers..."
  docker compose -p "sandbox-pr-${SANDBOX_ID}" -f docker-compose.sandbox.yml down -v

  echo "Cleaning up worktree..."
  cd /Users/lc/multisell
  git worktree remove "$WORKTREE_DIR" --force

  echo "Sandbox execution finished with exit code: $EXIT_CODE"
  exit $EXIT_CODE
  ```

- [ ] **Step 3: Run the script to verify validation failure**
  Run: `bash scripts/run_sandbox.sh 999`
  Expected: Spawns sandbox, executes seed, fails on Playwright (due to no test specification or empty tests depending on current branch state), and exits cleanly with cleanup.

- [ ] **Step 4: Commit**
  Run:
  ```bash
  git add docker-compose.sandbox.yml scripts/run_sandbox.sh
  git commit -m "feat(sandbox): add docker-compose sandbox and execution wrapper"
  ```
