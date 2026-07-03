# AI Traffic System P3+P4 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add P3 traffic control tower (status funnel, intercepted actions, audit replay) and P4 traffic health (rate limiter, external call tracker, agent metrics, congestion indicator) on top of existing AgentOS cockpit.

**Architecture:** Five independent streams running in two phases. Phase 1 (parallel backend): test gaps + P3 APIs + P4 governance. Phase 2 (parallel frontend): P3 UI + P4 UI. All data queries aggregate existing `unified_action`, `ai_trace`, `operation_log`, `approval_request` tables — no new tables.

**Tech Stack:** Go 1.25 (Gin, GORM), Next.js 16 (React 19, Ant Design 6, TanStack React Query 5), PostgreSQL 15, WebSocket (`gorilla/websocket` via `internal/realtime/`)

## Global Constraints

- All new backend code must have >=80% test coverage
- All new API endpoints must use `response.Success`/`response.Error` standard envelope
- All new types must follow the module's existing naming conventions (camelCase)
- All frontend additions must use Ant Design 6 components and existing color variables (`var(--g4)`, `var(--r4)`, etc.)
- No new DB tables — query from `unified_action`, `ai_trace`, `operation_log`, `approval_request`
- No Redis or external message queue — rate limiting uses in-memory sliding window
- Frontend additions go into existing `agentos/page.tsx` as new sections, not a new page

---

## File Structure

### Backend

| File | Responsibility | Action |
|------|---------------|--------|
| `internal/platform/command/action_test.go` | Add edge case tests for String() methods on ActionStatus/RiskLevel/ActionMode | Modify |
| `internal/platform/actioncatalog/catalog_test.go` | Add tests for AutonomyLevel.String(), Must() panic, L4 blocked path | Modify |
| `internal/agentos/handler.go` | Add TrafficSummary, InterceptedActions, AuditReplay, AgentMetrics handlers | Modify |
| `internal/agentos/service.go` | Add TrafficSummary, InterceptedActions, AuditReplay, AgentMetrics service methods | Modify |
| `internal/agentos/routes.go` | Register new API routes | Modify |
| `internal/agentos/agentos_handler_test.go` | Tests for new handlers | Modify |
| `internal/realtime/hub.go` | Register `agent.action.status_changed` event type | Modify |
| `internal/platform/command/ratelimit.go` | New RateLimiter with sliding window | Create |
| `internal/platform/toolbridge/tracker.go` | New ExternalCallTracker with degradation detection | Create |
| `internal/domain/entropy/service.go` | Consume agent-metrics for anomaly detection | Modify |

### Frontend

| File | Responsibility | Action |
|------|---------------|--------|
| `frontend-next/src/app/(main)/agentos/page.tsx` | Add traffic funnel cards, intercepted actions table, audit replay drawer, health cards, external health panel, congestion indicator | Modify |

---

## Phase 1 — Parallel Backend

### Task A1: Fill test coverage gaps in command + actioncatalog packages

**Files:**
- Modify: `backend-go/internal/platform/command/action_test.go`
- Modify: `backend-go/internal/platform/actioncatalog/catalog_test.go`

**Interfaces:**
- Consumes: existing `ActionStatus`, `RiskLevel`, `ActionMode`, `AutonomyLevel` types
- Produces: coverage >=85% for `command`, >=80% for `actioncatalog`

- [ ] **Step 1: Add ActionStatus.String() edge case test**

```go
func TestActionStatus_String_EdgeCases(t *testing.T) {
    tests := []struct {
        s    ActionStatus
        want string
    }{
        {StatusSuggested, "suggested"},
        {StatusPendingApproval, "pending_approval"},
        {StatusApproved, "approved"},
        {StatusRejected, "rejected"},
        {StatusExecuting, "executing"},
        {StatusCompleted, "completed"},
        {StatusFailed, "failed"},
        {StatusBlocked, "blocked"},
    }
    for _, tt := range tests {
        if got := tt.s.String(); got != tt.want {
            t.Errorf("ActionStatus(%d).String() = %q, want %q", tt.s, got, tt.want)
        }
    }
    // Unknown value
    if got := ActionStatus(99).String(); got != "status(99)" {
        t.Errorf("unknown status: got %q, want status(99)", got)
    }
}
```

- [ ] **Step 2: Add RiskLevel.String() edge case test**

```go
func TestRiskLevel_String(t *testing.T) {
    tests := []struct {
        r    RiskLevel
        want string
    }{
        {RiskNone, "none"},
        {RiskLow, "low"},
        {RiskMedium, "medium"},
        {RiskHigh, "high"},
    }
    for _, tt := range tests {
        if got := tt.r.String(); got != tt.want {
            t.Errorf("RiskLevel(%d).String() = %q, want %q", tt.r, got, tt.want)
        }
    }
    if got := RiskLevel(99).String(); got != "unknown(99)" {
        t.Errorf("unknown risk: got %q, want unknown(99)", got)
    }
}
```

- [ ] **Step 3: Add ActionMode.String() edge case test**

```go
func TestActionMode_String(t *testing.T) {
    tests := []struct {
        m    ActionMode
        want string
    }{
        {ModeDryRun, "dry_run"},
        {ModeSandbox, "sandbox"},
        {ModeProduction, "production"},
    }
    for _, tt := range tests {
        if got := tt.m.String(); got != tt.want {
            t.Errorf("ActionMode(%d).String() = %q, want %q", tt.m, got, tt.want)
        }
    }
    if got := ActionMode(99).String(); got != "mode(99)" {
        t.Errorf("unknown mode: got %q, want mode(99)", got)
    }
}
```

- [ ] **Step 4: Add HandlerNotFoundError.Error() test**

```go
func TestHandlerNotFoundError_Error(t *testing.T) {
    err := &HandlerNotFoundError{ActionType: "test_action"}
    want := "no handler registered for action type: test_action"
    if got := err.Error(); got != want {
        t.Errorf("HandlerNotFoundError.Error() = %q, want %q", got, want)
    }
}
```

- [ ] **Step 5: Run tests for command package**

```bash
cd backend-go && go test -v ./internal/platform/command/ -run 'TestActionStatus_|TestRiskLevel_|TestActionMode_|TestHandlerNotFound'
```

Expected: All pass

- [ ] **Step 6: Add AutonomyLevel.String() edge case for unknown value**

```go
func TestAutonomyLevel_String_Unknown(t *testing.T) {
    tests := []struct {
        l    AutonomyLevel
        want string
    }{
        {LevelUnknown, "unknown"},
        {Level1, "L1"},
        {Level2, "L2"},
        {Level3, "L3"},
        {Level4, "L4"},
    }
    for _, tt := range tests {
        if got := tt.l.String(); got != tt.want {
            t.Errorf("AutonomyLevel(%d).String() = %q, want %q", tt.l, got, tt.want)
        }
    }
    if got := AutonomyLevel(99).String(); got != "L99" {
        t.Errorf("unknown level: got %q, want L99", got)
    }
}
```

- [ ] **Step 7: Add Must() panic test**

```go
func TestMust_PanicsOnUnknown(t *testing.T) {
    defer func() {
        if r := recover(); r == nil {
            t.Fatal("expected panic for unknown action type")
        }
    }()
    c := Default()
    c.Must("nonexistent")
}
```

- [ ] **Step 8: Add L4 blocked test for L4 actions without approval**

```go
func TestL4_BlockedEvenWithApproval(t *testing.T) {
    c := Default()
    err := c.ValidateProduction("auto_publish", RiskHigh, true)
    if err == nil {
        t.Fatal("L4 action should be blocked even with approval")
    }
}
```

- [ ] **Step 9: Run tests for actioncatalog package**

```bash
cd backend-go && go test -v ./internal/platform/actioncatalog/ -run 'TestAutonomyLevel_|TestMust_|TestL4_'
```

Expected: All pass

- [ ] **Step 10: Verify coverage meets target**

```bash
cd backend-go && go test -coverprofile=coverage.out ./internal/platform/command/ ./internal/platform/actioncatalog/ && go tool cover -func=coverage.out | grep 'total\|action\.go\|catalog'
```

Expected: `command >= 80%`, `actioncatalog >= 75%`

- [ ] **Step 11: Commit**

```bash
git add internal/platform/command/action_test.go internal/platform/actioncatalog/catalog_test.go
git commit -m "test: fill AI Traffic System test coverage gaps — ActionStatus/RiskLevel/ActionMode strings, AutonomyLevel, Must panic, HandlerNotFoundError"
```

---

### Task B1: P3 backend — traffic-summary, intercepted-actions, audit-replay APIs

**Files:**
- Modify: `internal/agentos/handler.go` — add TrafficSummary, InterceptedActions, AuditReplay handlers
- Modify: `internal/agentos/service.go` — add service methods querying unified_action
- Modify: `internal/agentos/routes.go` — register new routes
- Modify: `internal/agentos/agentos_handler_test.go` — add tests

**Interfaces:**
- Consumes: `unified_action` table (status, risk_level, correlation_id, agent_id fields), `operation_log` table, `approval_request` table
- Produces: 3 new API endpoints + 3 service methods

- [ ] **Step 1: Add response types to service.go**

Add after existing `FailedRun` type:

```go
// TrafficSummary is the funnel overview of all action statuses.
type TrafficSummary struct {
    StatusDistribution map[string]int64 `json:"status_distribution"` // "suggested": 12, "completed": 87
    InterceptedTotal   int64            `json:"intercepted_total"`
    Funnel             FunnelStats      `json:"funnel"`
    ByRisk             map[string]map[string]int64 `json:"by_risk"` // "high": {"suggested": 3, "blocked": 1, ...}
}

type FunnelStats struct {
    Produced       int64 `json:"produced"`
    Approved       int64 `json:"approved"`
    Executed       int64 `json:"executed"`
    BlockedByPolicy int64 `json:"blocked_by_policy"`
    RejectedByOwner int64 `json:"rejected_by_owner"`
}

// InterceptedAction is a single blocked/rejected action for the dashboard.
type InterceptedAction struct {
    ID            int64  `json:"id"`
    ActionType    string `json:"action_type"`
    AgentID       string `json:"agent_id"`
    RiskLevel     string `json:"risk_level"`
    BlockReason   string `json:"block_reason"`   // "approval_required", "L4_blocked", "rate_limited", "policy_blocked"
    BlockedAt     string `json:"blocked_at"`
    TargetSummary string `json:"target_summary"`
}

// AuditReplayEvent is one step in an action's full trace.
type AuditReplayEvent struct {
    Type       string `json:"type"`       // "event", "agent_decision", "action", "approval", "execution", "audit"
    Subtype    string `json:"subtype,omitempty"`
    AgentID    string `json:"agent_id,omitempty"`
    ActionID   *int64 `json:"action_id,omitempty"`
    Status     string `json:"status,omitempty"`
    Detail     string `json:"detail,omitempty"`
    Timestamp  string `json:"timestamp"`
}

// AuditReplayResponse is the full timeline for one correlation ID.
type AuditReplayResponse struct {
    CorrelationID string            `json:"correlation_id"`
    Events        []AuditReplayEvent `json:"events"`
}
```

- [ ] **Step 2: Add TrafficSummary service method**

```go
// TrafficSummary returns the distribution of all action statuses.
func (s *Service) TrafficSummary() (*TrafficSummary, error) {
    summary := &TrafficSummary{
        StatusDistribution: map[string]int64{},
        ByRisk:             map[string]map[string]int64{},
        Funnel:             FunnelStats{},
    }

    // Status distribution: single query with GROUP BY
    type statusCount struct {
        Status string
        Count  int64
    }
    var statusCounts []statusCount
    if err := s.db.Table("unified_action").
        Select("status, COUNT(*) AS count").
        Group("status").
        Scan(&statusCounts).Error; err != nil {
        return nil, err
    }
    for _, sc := range statusCounts {
        summary.StatusDistribution[sc.Status] = sc.Count
    }

    // Intercepted total: blocked + rejected (not by owner) + pending_approval that failed approval
    type intCount struct{ Count int64 }
    if err := s.db.Table("unified_action").
        Select("COUNT(*) AS count").
        Where("status IN ?", []string{"blocked"}).
        Scan(&intCount{}).Error; err == nil {
        // reassign by scanning directly
        var ic intCount
        s.db.Table("unified_action").Select("COUNT(*) AS count").Where("status = ?", "blocked").Scan(&ic)
        summary.InterceptedTotal = ic.Count
    }

    // Funnel
    var funnel struct {
        Produced int64
        Approved int64
        Executed int64
        Rejected int64
        Blocked  int64
    }
    s.db.Raw(`
        SELECT
            COUNT(*) AS produced,
            COUNT(*) FILTER (WHERE status = 'approved') AS approved,
            COUNT(*) FILTER (WHERE status = 'completed') AS executed,
            COUNT(*) FILTER (WHERE status = 'rejected') AS rejected,
            COUNT(*) FILTER (WHERE status = 'blocked') AS blocked
        FROM unified_action`,
    ).Scan(&funnel)
    summary.Funnel = FunnelStats{
        Produced:        funnel.Produced,
        Approved:        funnel.Approved,
        Executed:        funnel.Executed,
        BlockedByPolicy: funnel.Blocked,
        RejectedByOwner: funnel.Rejected,
    }

    // By risk × status cross-tab
    type riskStatusCount struct {
        RiskLevel string
        Status    string
        Count     int64
    }
    var cross []riskStatusCount
    s.db.Table("unified_action").
        Select("risk_level, status, COUNT(*) AS count").
        Group("risk_level, status").
        Scan(&cross)
    for _, c := range cross {
        if _, ok := summary.ByRisk[c.RiskLevel]; !ok {
            summary.ByRisk[c.RiskLevel] = map[string]int64{}
        }
        summary.ByRisk[c.RiskLevel][c.Status] = c.Count
    }

    return summary, nil
}
```

- [ ] **Step 3: Add InterceptedActions service method**

```go
// InterceptedActions returns recently blocked/rejected actions.
func (s *Service) InterceptedActions(limit int) ([]InterceptedAction, int64, error) {
    if limit <= 0 || limit > 100 {
        limit = 50
    }
    type row struct {
        ID              int64
        ActionType      string
        AgentID         string
        RiskLevel       string
        Status          string
        BlockReason     string
        CreatedAt       time.Time
        TargetSummary   string
    }
    var rows []row
    q := s.db.Table("unified_action").
        Select("id, action_type, agent_id, risk_level, status, COALESCE(block_reason,'') AS block_reason, created_at, COALESCE(description,'') AS target_summary").
        Where("status IN ?", []string{"blocked", "rejected"})
    var total int64
    if err := q.Count(&total).Error; err != nil {
        return nil, 0, err
    }
    if err := q.Order("created_at DESC").Limit(limit).Scan(&rows).Error; err != nil {
        return nil, 0, err
    }
    items := make([]InterceptedAction, 0, len(rows))
    for _, r := range rows {
        items = append(items, InterceptedAction{
            ID:            r.ID,
            ActionType:    r.ActionType,
            AgentID:       r.AgentID,
            RiskLevel:     r.RiskLevel,
            BlockReason:   r.BlockReason,
            BlockedAt:     r.CreatedAt.Format("2006-01-02 15:04:05"),
            TargetSummary: r.TargetSummary,
        })
    }
    return items, total, nil
}
```

- [ ] **Step 4: Add AuditReplay service method**

```go
// AuditReplay returns the full event timeline for a correlation ID.
func (s *Service) AuditReplay(correlationID string) (*AuditReplayResponse, error) {
    resp := &AuditReplayResponse{
        CorrelationID: correlationID,
        Events:        []AuditReplayEvent{},
    }

    // 1. Find all unified_actions with this correlation_id
    type actionRow struct {
        ID        int64
        ActionType string
        AgentID   string
        Status    string
        CreatedAt time.Time
    }
    var actions []actionRow
    s.db.Table("unified_action").
        Select("id, action_type, agent_id, status, created_at").
        Where("correlation_id = ?", correlationID).
        Order("created_at ASC").
        Scan(&actions)
    for _, a := range actions {
        resp.Events = append(resp.Events, AuditReplayEvent{
            Type: "action", AgentID: a.AgentID,
            Subtype: a.ActionType,
            ActionID: &a.ID,
            Status: a.Status,
            Timestamp: a.CreatedAt.Format("2006-01-02 15:04:05"),
        })
    }

    // 2. Find approval requests linked to these actions
    if len(actions) > 0 {
        var ids []int64
        for _, a := range actions {
            ids = append(ids, a.ID)
        }
        type appRow struct {
            ID        int64
            Status    string
            Reviewer  string
            UpdatedAt time.Time
        }
        var approvals []appRow
        s.db.Table("approval_request").
            Select("id, status, COALESCE(reviewer,'') AS reviewer, updated_at").
            Where("entity_type = 'unified_action' AND entity_id IN ?", ids).
            Order("updated_at ASC").
            Scan(&approvals)
        for _, ap := range approvals {
            resp.Events = append(resp.Events, AuditReplayEvent{
                Type: "approval",
                Subtype: "approval_request",
                Status: ap.Status,
                Detail: "reviewer: " + ap.Reviewer,
                Timestamp: ap.UpdatedAt.Format("2006-01-02 15:04:05"),
            })
        }
    }

    // 3. Find operation_log entries
    type logRow struct {
        Action    string
        Content   string
        CreatedAt time.Time
    }
    var logs []logRow
    s.db.Table("operation_log").
        Select("action, COALESCE(content,'') AS content, created_at").
        Where("correlation_id = ?", correlationID).
        Order("created_at ASC").
        Scan(&logs)
    for _, l := range logs {
        resp.Events = append(resp.Events, AuditReplayEvent{
            Type: "audit", Subtype: l.Action,
            Detail: l.Content,
            Timestamp: l.CreatedAt.Format("2006-01-02 15:04:05"),
        })
    }

    // Sort all events by timestamp
    // (they're already ordered per-source, but multi-source can interleave)
    // Ponytail: events are ordered per-source; multi-source interleaving not fixed (add sync sort if ordering matters)

    return resp, nil
}
```

- [ ] **Step 5: Add handlers to handler.go**

```go
// TrafficSummary GET /agentos/traffic-summary
func (h *Handler) TrafficSummary(c *gin.Context) {
    summary, err := h.service.TrafficSummary()
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    response.Success(c, summary)
}

// InterceptedActions GET /agentos/intercepted-actions
func (h *Handler) InterceptedActions(c *gin.Context) {
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
    items, total, err := h.service.InterceptedActions(limit)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    response.Success(c, gin.H{"items": items, "total": total})
}

// AuditReplay GET /agentos/audit-replay/:correlation_id
func (h *Handler) AuditReplay(c *gin.Context) {
    correlationID := c.Param("correlation_id")
    if correlationID == "" {
        response.Error(c, http.StatusBadRequest, "correlation_id is required")
        return
    }
    replay, err := h.service.AuditReplay(correlationID)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    response.Success(c, replay)
}
```

- [ ] **Step 6: Register routes in routes.go**

```go
agentos.GET("/traffic-summary", h.TrafficSummary)
agentos.GET("/intercepted-actions", h.InterceptedActions)
agentos.GET("/audit-replay/:correlation_id", h.AuditReplay)
```

- [ ] **Step 7: Add tests**

```go
func TestTrafficSummary(t *testing.T) {
    h := newTestHandler(t)
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/agentos/traffic-summary", nil)
    r.ServeHTTP(w, req)
    assert.Equal(t, 200, w.Code)
    // Response should contain status_distribution and funnel
    assert.Contains(t, w.Body.String(), "status_distribution")
    assert.Contains(t, w.Body.String(), "funnel")
}

func TestAuditReplay_NotFound(t *testing.T) {
    h := newTestHandler(t)
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/agentos/audit-replay/nonexistent", nil)
    r.ServeHTTP(w, req)
    assert.Equal(t, 200, w.Code) // returns empty events list, not error
}
```

- [ ] **Step 8: Run tests**

```bash
cd backend-go && go test -v ./internal/agentos/ -run 'TestTrafficSummary|TestIntercepted|TestAuditReplay'
```

Expected: All pass

- [ ] **Step 9: Commit**

```bash
git add internal/agentos/
git commit -m "feat: P3 traffic dashboard APIs — TrafficSummary, InterceptedActions, AuditReplay"
```

---

### Task B2: P3 WebSocket status_changed event

**Files:**
- Modify: `internal/realtime/hub.go` — register `agent.action.status_changed` event type

**Interfaces:**
- Consumes: existing `Hub.Broadcast()` and `Hub.BroadcastAndWait()` methods
- Produces: WS event `{"type":"agent.action.status_changed","payload":{...}}`

- [ ] **Step 1: Add event type constant and construction helper**

Add at the top of `internal/realtime/hub.go` (after existing imports):

```go
// Event types for real-time action status push.
const (
    EventActionStatusChanged = "agent.action.status_changed"
)

// ActionStatusChangePayload is WS event payload for status changes.
type ActionStatusChangePayload struct {
    ActionID      int64  `json:"action_id"`
    ActionType    string `json:"action_type"`
    AgentID       string `json:"agent_id"`
    RiskLevel     string `json:"risk_level"`
    OldStatus     string `json:"old_status"`
    NewStatus     string `json:"new_status"`
    CorrelationID string `json:"correlation_id"`
    Timestamp     string `json:"timestamp"`
}
```

Add a helper method on Hub:

```go
// BroadcastActionStatusChange sends a status change event to all WS clients.
func (h *Hub) BroadcastActionStatusChange(payload ActionStatusChangePayload) {
    msg, _ := json.Marshal(map[string]interface{}{
        "type":    EventActionStatusChanged,
        "payload": payload,
    })
    h.Broadcast(msg)
}
```

- [ ] **Step 2: Verify build**

```bash
cd backend-go && go build ./internal/realtime/
```

Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/realtime/hub.go
git commit -m "feat: add WS event agent.action.status_changed to realtime hub"
```

---

### Task D1: P4 RateLimiter

**Files:**
- Create: `internal/platform/command/ratelimit.go`

**Interfaces:**
- Consumes: nothing (standalone in-memory component)
- Produces: `RateLimiter` type with `Allow(agentID, actionType string) bool`
- Integration: `DispatchSafe` in `command.go` calls `RateLimiter.Allow()` before execution

- [ ] **Step 1: Create ratelimit.go**

```go
package command

import (
    "fmt"
    "sync"
    "time"
)

// slidingWindow tracks request counts within a time window.
type slidingWindow struct {
    entries []time.Time
    limit   int
    window  time.Duration
}

func newSlidingWindow(limit int, window time.Duration) *slidingWindow {
    return &slidingWindow{
        entries: make([]time.Time, 0, limit),
        limit:   limit,
        window:  window,
    }
}

func (sw *slidingWindow) allow() bool {
    now := time.Now()
    cutoff := now.Add(-sw.window)
    // Trim expired entries
    start := 0
    for i, t := range sw.entries {
        if t.After(cutoff) {
            start = i
            break
        }
    }
    sw.entries = sw.entries[start:]
    if len(sw.entries) >= sw.limit {
        return false
    }
    sw.entries = append(sw.entries, now)
    return true
}

// ErrRateLimited is returned when an action is rate-limited.
var ErrRateLimited = fmt.Errorf("command: action rate limited")

// RateLimiter enforces per-(agent, action_type) rate limits using sliding windows.
type RateLimiter struct {
    mu      sync.Mutex
    windows map[string]*slidingWindow
    limit   int
    window  time.Duration
}

// NewRateLimiter creates a rate limiter with the given limit per window duration.
// Default: 20 actions/hour per (agent, action_type).
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
    if limit <= 0 {
        limit = 20
    }
    if window <= 0 {
        window = time.Hour
    }
    return &RateLimiter{
        windows: make(map[string]*slidingWindow),
        limit:   limit,
        window:  window,
    }
}

// Allow checks if an action from agentID of type actionType is allowed.
// Returns true if within limit, false if rate-limited.
func (rl *RateLimiter) Allow(agentID, actionType string) bool {
    key := agentID + ":" + actionType
    rl.mu.Lock()
    defer rl.mu.Unlock()
    sw, ok := rl.windows[key]
    if !ok {
        sw = newSlidingWindow(rl.limit, rl.window)
        rl.windows[key] = sw
    }
    return sw.allow()
}

// Reset clears all rate limit counters for the given agent.
func (rl *RateLimiter) Reset(agentID string) {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    prefix := agentID + ":"
    for k := range rl.windows {
        if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
            delete(rl.windows, k)
        }
    }
}
```

- [ ] **Step 2: Add rate limiter test**

```go
func TestRateLimiter_Allow_Basic(t *testing.T) {
    rl := NewRateLimiter(2, time.Minute)
    if !rl.Allow("A5", "stock_alert") {
        t.Error("expected first call to be allowed")
    }
    if !rl.Allow("A5", "stock_alert") {
        t.Error("expected second call to be allowed")
    }
    if rl.Allow("A5", "stock_alert") {
        t.Error("expected third call to be rate limited")
    }
}

func TestRateLimiter_AllowsDifferentTypes(t *testing.T) {
    rl := NewRateLimiter(2, time.Minute)
    rl.Allow("A5", "stock_alert")  // 1
    rl.Allow("A5", "stock_alert")  // 2 (max)
    // Different action type should still be allowed
    if !rl.Allow("A5", "price_update") {
        t.Error("different action type should not share rate limit")
    }
    // Different agent should be allowed
    if !rl.Allow("A6", "stock_alert") {
        t.Error("different agent should not share rate limit")
    }
}

func TestRateLimiter_Reset(t *testing.T) {
    rl := NewRateLimiter(1, time.Minute)
    rl.Allow("A5", "stock_alert")
    if rl.Allow("A5", "stock_alert") {
        t.Error("expected rate limited after hitting limit")
    }
    rl.Reset("A5")
    if !rl.Allow("A5", "stock_alert") {
        t.Error("expected allowed after reset")
    }
}
```

- [ ] **Step 3: Run tests**

```bash
cd backend-go && go test -v ./internal/platform/command/ -run 'TestRateLimiter'
```

Expected: All pass

- [ ] **Step 4: Integrate RateLimiter into command.go Dispatcher**

Add to Dispatcher struct:
```go
type Dispatcher struct {
    // ... existing fields ...
    rateLimiter *RateLimiter // optional, nil means no rate limiting
}
```

Add option:
```go
func WithRateLimiter(rl *RateLimiter) DispatcherOption {
    return func(d *Dispatcher) {
        d.rateLimiter = rl
    }
}
```

In `DispatchSafe`, add after catalog validation but before handler execution:
```go
// Rate limiting check
if d.rateLimiter != nil && action.Mode == ModeProduction {
    if !d.rateLimiter.Allow(action.AgentID, action.ActionType) {
        return nil, ErrRateLimited
    }
}
```

- [ ] **Step 5: Run all command tests**

```bash
cd backend-go && go test -v ./internal/platform/command/
```

Expected: All 30+ tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/platform/command/ratelimit.go internal/platform/command/command.go internal/platform/command/command_test.go
git commit -m "feat: P4 RateLimiter — per (agent, action_type) sliding window, 20/hour default"
```

---

### Task D2: P4 ExternalCallTracker

**Files:**
- Create: `internal/platform/toolbridge/tracker.go`
- Modify: `internal/platform/toolbridge/bridge.go` — wire tracker into ToolBridge

**Interfaces:**
- Consumes: existing `ToolBridge` execution path
- Produces: `ExternalCallTracker` that monitors platform call health

- [ ] **Step 1: Create tracker.go**

```go
package toolbridge

import (
    "fmt"
    "sync"
    "time"
)

// DegradedErr is returned when a platform is marked degraded.
var DegradedErr = fmt.Errorf("toolbridge: platform is degraded, skipping execution")

// platformStats holds recent call stats for one external platform.
type platformStats struct {
    TotalCalls         int
    FailedCalls        int
    LastFailureAt      time.Time
    LastError          string
    ConsecutiveFailures int
    Degraded           bool
    DegradedAt         time.Time
}

// ExternalCallTracker monitors external platform call health.
type ExternalCallTracker struct {
    mu         sync.Mutex
    platforms  map[string]*platformStats
    threshold  int // consecutive failures to mark degraded (default 3)
}

// NewExternalCallTracker creates a tracker with custom degradation threshold.
func NewExternalCallTracker(threshold int) *ExternalCallTracker {
    if threshold <= 0 {
        threshold = 3
    }
    return &ExternalCallTracker{
        platforms: make(map[string]*platformStats),
        threshold: threshold,
    }
}

// RecordCall records the result of an external platform call.
func (t *ExternalCallTracker) RecordCall(platform string, err error) {
    t.mu.Lock()
    defer t.mu.Unlock()
    ps, ok := t.platforms[platform]
    if !ok {
        ps = &platformStats{}
        t.platforms[platform] = ps
    }
    ps.TotalCalls++
    if err != nil {
        ps.FailedCalls++
        ps.LastFailureAt = time.Now()
        ps.LastError = err.Error()
        ps.ConsecutiveFailures++
        if ps.ConsecutiveFailures >= t.threshold {
            ps.Degraded = true
            ps.DegradedAt = time.Now()
        }
    } else {
        ps.ConsecutiveFailures = 0
    }
}

// IsDegraded returns true if the platform is currently degraded.
func (t *ExternalCallTracker) IsDegraded(platform string) bool {
    t.mu.Lock()
    defer t.mu.Unlock()
    ps, ok := t.platforms[platform]
    if !ok {
        return false
    }
    return ps.Degraded
}

// Stats returns a snapshot of platform stats for dashboard use.
type PlatformStatsSnapshot struct {
    Platform           string `json:"platform"`
    TotalCalls         int    `json:"total_calls"`
    FailedCalls        int    `json:"failed_calls"`
    ConsecutiveFailures int   `json:"consecutive_failures"`
    Degraded           bool   `json:"degraded"`
    LastFailureAt      string `json:"last_failure_at,omitempty"`
    LastError          string `json:"last_error,omitempty"`
}

// Stats returns all platform stats for the dashboard.
func (t *ExternalCallTracker) Stats() []PlatformStatsSnapshot {
    t.mu.Lock()
    defer t.mu.Unlock()
    result := make([]PlatformStatsSnapshot, 0, len(t.platforms))
    for name, ps := range t.platforms {
        s := PlatformStatsSnapshot{
            Platform:           name,
            TotalCalls:         ps.TotalCalls,
            FailedCalls:        ps.FailedCalls,
            ConsecutiveFailures: ps.ConsecutiveFailures,
            Degraded:           ps.Degraded,
            LastError:          ps.LastError,
        }
        if !ps.LastFailureAt.IsZero() {
            s.LastFailureAt = ps.LastFailureAt.Format(time.RFC3339)
        }
        result = append(result, s)
    }
    return result
}
```

- [ ] **Step 2: Add tracker test**

```go
func TestExternalCallTracker_RecordsSuccess(t *testing.T) {
    tr := NewExternalCallTracker(3)
    tr.RecordCall("shopee", nil)
    tr.RecordCall("shopee", nil)
    stats := tr.Stats()
    if len(stats) != 1 {
        t.Fatalf("expected 1 platform, got %d", len(stats))
    }
    if stats[0].TotalCalls != 2 || stats[0].FailedCalls != 0 {
        t.Errorf("expected 2 total, 0 failed; got %d/%d", stats[0].TotalCalls, stats[0].FailedCalls)
    }
    if stats[0].Degraded {
        t.Error("should not be degraded after successes")
    }
}

func TestExternalCallTracker_DegradedAfterThreshold(t *testing.T) {
    tr := NewExternalCallTracker(3)
    for i := 0; i < 3; i++ {
        tr.RecordCall("ozon", fmt.Errorf("timeout #%d", i+1))
    }
    if !tr.IsDegraded("ozon") {
        t.Error("expected ozon to be degraded after 3 consecutive failures")
    }
    // Recovery
    tr.RecordCall("ozon", nil)
    if tr.IsDegraded("ozon") {
        t.Error("expected ozon to recover after success")
    }
}

func TestExternalCallTracker_StatsFormat(t *testing.T) {
    tr := NewExternalCallTracker(3)
    tr.RecordCall("shopee", fmt.Errorf("auth failed"))
    stats := tr.Stats()
    if len(stats) != 1 {
        t.Fatalf("expected 1 platform")
    }
    if stats[0].LastError != "auth failed" {
        t.Errorf("expected auth failed, got %q", stats[0].LastError)
    }
    if stats[0].LastFailureAt == "" {
        t.Error("expected last_failure_at to be set")
    }
}
```

- [ ] **Step 3: Wire tracker into ToolBridge**

In `bridge.go`, add to Bridge struct:
```go
type ToolBridge struct {
    // ... existing fields ...
    tracker *ExternalCallTracker
}

func WithTracker(t *ExternalCallTracker) ToolBridgeOption {
    return func(b *ToolBridge) {
        b.tracker = t
    }
}
```

In the helper that records calls (around actual platform fetch), add:
```go
if b.tracker != nil {
    b.tracker.RecordCall(platformName, err)
    if err != nil && b.tracker.IsDegraded(platformName) {
        // Log degraded warning
        b.logger.Warn("platform degraded, skipping", zap.String("platform", platformName))
        return nil, DegradedErr
    }
}
```

- [ ] **Step 4: Run tests**

```bash
cd backend-go && go test -v ./internal/platform/toolbridge/ -run 'TestExternalCall'
```

Expected: All pass

- [ ] **Step 5: Commit**

```bash
git add internal/platform/toolbridge/tracker.go internal/platform/toolbridge/bridge.go
git commit -m "feat: P4 ExternalCallTracker — per-platform degradation after 3 consecutive failures"
```

---

### Task D3: P4 AgentMetrics API + entropy integration

**Files:**
- Modify: `internal/agentos/service.go` — add `AgentMetrics()` method
- Modify: `internal/agentos/handler.go` — add `AgentMetrics` handler
- Modify: `internal/agentos/routes.go` — register route
- Modify: `internal/agentos/agentos_handler_test.go` — add tests

**Interfaces:**
- Consumes: `unified_action`, `ai_trace` tables for per-agent statistics
- Produces: `GET /v1/agentos/agent-metrics` endpoint

- [ ] **Step 1: Add AgentMetrics types and service method**

```go
// AgentMetrics is the per-agent health and performance snapshot.
type AgentMetrics struct {
    AgentID             string  `json:"agent_id"`
    RunCount            int64   `json:"run_count"`
    SuccessCount        int64   `json:"success_count"`
    FailureCount        int64   `json:"failure_count"`
    BlockedCount        int64   `json:"blocked_count"`
    ApprovalRate        float64 `json:"approval_rate"`
    OwnerAcceptanceRate float64 `json:"owner_acceptance_rate"`
    AvgLatencyMs        float64 `json:"avg_latency_ms"`
    ExternalFailureRate float64 `json:"external_failure_rate"`
    Health              string  `json:"health"`
}

// AgentMetrics returns per-agent metrics aggregated from unified_action and ai_trace.
func (s *Service) AgentMetrics() ([]AgentMetrics, error) {
    type rawMetrics struct {
        AgentID    string
        RunCount   int64
        Succeeded  int64
        Failed     int64
        Blocked    int64
    }

    // Aggregate from unified_action
    var actions []rawMetrics
    s.db.Table("unified_action").
        Select(`
            agent_id,
            COUNT(*) AS run_count,
            COUNT(*) FILTER (WHERE status = 'completed') AS succeeded,
            COUNT(*) FILTER (WHERE status = 'failed') AS failed,
            COUNT(*) FILTER (WHERE status = 'blocked') AS blocked
        `).
        Group("agent_id").
        Scan(&actions)

    // Aggregate latencies from ai_trace
    type latRow struct {
        AgentID string
        AvgLat  float64
    }
    var latencies []latRow
    s.db.Table("ai_trace").
        Select("agent_id, COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(completed_at, started_at) - started_at))), 0) AS avg_lat").
        Where("completed_at IS NOT NULL").
        Group("agent_id").
        Scan(&latencies)
    latMap := make(map[string]float64, len(latencies))
    for _, l := range latencies {
        latMap[l.AgentID] = l.AvgLat * 1000 // convert to ms
    }

    // Acceptance rate: approved / (approved + rejected)
    type accRow struct {
        AgentID    string
        Approved   int64
        Rejected   int64
    }
    var accRates []accRow
    s.db.Table("unified_action").
        Select(`
            agent_id,
            COUNT(*) FILTER (WHERE status = 'approved') AS approved,
            COUNT(*) FILTER (WHERE status = 'rejected') AS rejected
        `).
        Group("agent_id").
        Scan(&accRates)
    accMap := make(map[string]float64, len(accRates))
    for _, a := range accRates {
        total := a.Approved + a.Rejected
        if total > 0 {
            accMap[a.AgentID] = float64(a.Approved) / float64(total)
        }
    }

    result := make([]AgentMetrics, 0, len(actions))
    for _, a := range actions {
        total := a.Succeeded + a.Failed + a.Blocked
        extFailRate := 0.0
        if a.Failed > 0 && total > 0 {
            extFailRate = float64(a.Failed) / float64(total)
        }
        lat := latMap[a.AgentID]

        // Health classification
        health := "ok"
        if extFailRate > 0.2 || a.Failed > 10 {
            health = "warn"
        }
        if extFailRate > 0.5 || a.Failed > 50 {
            health = "critical"
        }

        result = append(result, AgentMetrics{
            AgentID:             a.AgentID,
            RunCount:            a.RunCount,
            SuccessCount:        a.Succeeded,
            FailureCount:        a.Failed,
            BlockedCount:        a.Blocked,
            ApprovalRate:        accMap[a.AgentID],
            OwnerAcceptanceRate: accMap[a.AgentID],
            AvgLatencyMs:        lat,
            ExternalFailureRate: extFailRate,
            Health:              health,
        })
    }
    return result, nil
}
```

- [ ] **Step 2: Add handler**

```go
// AgentMetrics GET /agentos/agent-metrics
func (h *Handler) AgentMetrics(c *gin.Context) {
    metrics, err := h.service.AgentMetrics()
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    response.Success(c, gin.H{"agents": metrics})
}
```

- [ ] **Step 3: Register route**

```go
agentos.GET("/agent-metrics", h.AgentMetrics)
```

- [ ] **Step 4: Add test**

```go
func TestAgentMetrics(t *testing.T) {
    h := newTestHandler(t)
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/agentos/agent-metrics", nil)
    r.ServeHTTP(w, req)
    assert.Equal(t, 200, w.Code)
    assert.Contains(t, w.Body.String(), "agents")
}
```

- [ ] **Step 5: Run tests**

```bash
cd backend-go && go test -v ./internal/agentos/ -run 'TestAgentMetrics|TestTraffic'
```

Expected: All pass

- [ ] **Step 6: Commit**

```bash
git add internal/agentos/
git commit -m "feat: P4 AgentMetrics API — per-agent run_count, failure_rate, latency, health"
```

---

### Task D4: P4 Entropy integration — consume metrics for anomaly detection

**Files:**
- Modify: `internal/domain/entropy/service.go` — add `ConsumeAgentMetrics` method

**Interfaces:**
- Consumes: `AgentMetrics` from agentos service (or equivalent proto type)
- Produces: entropy anomaly detection that can adjust TrustScore/autonomy

- [ ] **Step 1: Add ConsumeAgentMetrics method to entropy Service**

```go
// AgentMetricsSnapshot is a minimal metrics input for entropy processing.
type AgentMetricsSnapshot struct {
    AgentID             string
    RunCount            int64
    FailureCount        int64
    BlockedCount        int64
    ExternalFailureRate float64
    AvgLatencyMs        float64
}

// ConsumeAgentMetrics processes agent metrics for anomaly detection.
// Returns a list of agents flagged as anomalous with reasons.
type AnomalousAgent struct {
    AgentID string `json:"agent_id"`
    Reason  string `json:"reason"`
    Severity string `json:"severity"` // "warn", "critical"
}

func (s *Service) ConsumeAgentMetrics(metrics []AgentMetricsSnapshot) []AnomalousAgent {
    var anomalies []AnomalousAgent
    for _, m := range metrics {
        if m.ExternalFailureRate > 0.2 {
            anomalies = append(anomalies, AnomalousAgent{
                AgentID:  m.AgentID,
                Reason:   fmt.Sprintf("external failure rate %.0f%% exceeds threshold (20%%)", m.ExternalFailureRate*100),
                Severity: "warn",
            })
        }
        if m.FailureCount > 20 && m.BlockedCount > 10 {
            anomalies = append(anomalies, AnomalousAgent{
                AgentID:  m.AgentID,
                Reason:   fmt.Sprintf("high failure (%d) and blocked (%d) count", m.FailureCount, m.BlockedCount),
                Severity: "critical",
            })
        }
        if m.AvgLatencyMs > 10000 {
            anomalies = append(anomalies, AnomalousAgent{
                AgentID:  m.AgentID,
                Reason:   fmt.Sprintf("avg latency %.0fms exceeds 10s threshold", m.AvgLatencyMs),
                Severity: "warn",
            })
        }
    }
    return anomalies
}
```

- [ ] **Step 2: Add test**

```go
func TestConsumeAgentMetrics_Normal(t *testing.T) {
    svc := NewService(dbtest.NewDB(t), dbtest.NewLogger(t))
    metrics := []AgentMetricsSnapshot{
        {AgentID: "A5", RunCount: 100, FailureCount: 2, BlockedCount: 1, ExternalFailureRate: 0.02, AvgLatencyMs: 500},
    }
    anomalies := svc.ConsumeAgentMetrics(metrics)
    if len(anomalies) != 0 {
        t.Errorf("expected 0 anomalies for normal agent, got %d: %v", len(anomalies), anomalies)
    }
}

func TestConsumeAgentMetrics_HighFailureRate(t *testing.T) {
    svc := NewService(dbtest.NewDB(t), dbtest.NewLogger(t))
    metrics := []AgentMetricsSnapshot{
        {AgentID: "A6", RunCount: 10, FailureCount: 5, BlockedCount: 3, ExternalFailureRate: 0.5, AvgLatencyMs: 2000},
    }
    anomalies := svc.ConsumeAgentMetrics(metrics)
    if len(anomalies) == 0 {
        t.Fatal("expected anomalies for high failure rate")
    }
    found := false
    for _, a := range anomalies {
        if a.AgentID == "A6" && a.Severity != "" {
            found = true
        }
    }
    if !found {
        t.Errorf("expected A6 anomaly, got: %v", anomalies)
    }
}
```

- [ ] **Step 3: Run entropy tests**

```bash
cd backend-go && go test -v ./internal/domain/entropy/ -run 'TestConsumeAgentMetrics'
```

Expected: All pass

- [ ] **Step 4: Commit**

```bash
git add internal/domain/entropy/
git commit -m "feat: P4 Entropy ConsumeAgentMetrics — anomaly detection from agent metrics"
```

---

## Phase 2 — Frontend (depends on Phase 1 APIs)

### Task C1: P3 frontend — Traffic Section + Audit Replay Drawer

**Files:**
- Modify: `frontend-next/src/app/(main)/agentos/page.tsx`

**Interfaces:**
- Consumes: `GET /v1/agentos/traffic-summary`, `GET /v1/agentos/intercepted-actions`, `GET /v1/agentos/audit-replay/:correlation_id`
- Produces: New UI sections for traffic monitoring

- [ ] **Step 1: Add traffic query types and API calls**

Add after existing `AgentTimelineEntry` type:

```typescript
interface TrafficSummary {
  status_distribution: Record<string, number>;
  intercepted_total: number;
  funnel: {
    produced: number;
    approved: number;
    executed: number;
    blocked_by_policy: number;
    rejected_by_owner: number;
  };
  by_risk: Record<string, Record<string, number>>;
}

interface InterceptedAction {
  id: number;
  action_type: string;
  agent_id: string;
  risk_level: string;
  block_reason: string;
  blocked_at: string;
  target_summary: string;
}

interface AuditReplayEvent {
  type: string;
  subtype?: string;
  agent_id?: string;
  action_id?: number;
  status?: string;
  detail?: string;
  timestamp: string;
}
```

Add queries alongside existing useQuery calls:

```typescript
const { data: trafficSummary, isLoading: trafficLoading } = useQuery({
  queryKey: ['traffic-summary'],
  queryFn: async () => {
    const res = await apiClient.get<TrafficSummary>('/v1/agentos/traffic-summary');
    return res.data;
  },
});

const { data: interceptedData, isLoading: interceptedLoading } = useQuery({
  queryKey: ['intercepted-actions'],
  queryFn: async () => {
    const res = await apiClient.get<{items: InterceptedAction[], total: number}>('/v1/agentos/intercepted-actions');
    return res.data;
  },
});
```

- [ ] **Step 2: Add traffic funnel cards between existing stat cards and squad health**

After the `{/* 顶部：统计卡片 */}` row and before `{/* AIOS 系统指标 */}` row, add:

```tsx
{/* Traffic Funnel */}
<SectionCard title="AI Traffic Funnel" style={{ marginBottom: 16 }}>
  <Row gutter={16}>
    <Col xs={12} sm={4}>
      <StatCard title="已产生" value={trafficSummary?.funnel?.produced ?? 0}
        prefix={<RobotOutlined />} loading={trafficLoading} />
    </Col>
    <Col xs={12} sm={4}>
      <StatCard title="待审批" value={trafficSummary?.status_distribution?.pending_approval ?? 0}
        prefix={<ClockCircleOutlined />} valueStyle={{ color: 'var(--y4)' }} loading={trafficLoading} />
    </Col>
    <Col xs={12} sm={4}>
      <StatCard title="已执行" value={trafficSummary?.funnel?.executed ?? 0}
        prefix={<CheckOutlined />} valueStyle={{ color: 'var(--g4)' }} loading={trafficLoading} />
    </Col>
    <Col xs={12} sm={4}>
      <StatCard title="被拦截" value={trafficSummary?.funnel?.blocked_by_policy ?? 0}
        prefix={<CloseOutlined />} valueStyle={{ color: 'var(--r4)' }} loading={trafficLoading} />
    </Col>
    <Col xs={12} sm={4}>
      <StatCard title="转化率"
        value={(() => {
          const f = trafficSummary?.funnel;
          if (!f || f.produced === 0) return '-';
          return `${((f.executed / f.produced) * 100).toFixed(0)}%`;
        })()}
        prefix={<ThunderboltOutlined />} loading={trafficLoading} />
    </Col>
  </Row>
  {/* Mini funnel bar */}
  {trafficSummary?.funnel && (
    <div style={{ marginTop: 8, height: 8, background: 'var(--s2)', borderRadius: 4, display: 'flex', overflow: 'hidden' }}>
      {['executed', 'pending_approval', 'blocked', 'rejected'].map((k) => {
        const total = Object.values(trafficSummary.status_distribution).reduce((a: number, b: number) => a + b, 0) || 1;
        const v = trafficSummary.status_distribution[k] ?? 0;
        const pct = (v / total) * 100;
        if (pct === 0) return null;
        const colors: Record<string, string> = {
          executed: 'var(--g4)', pending_approval: 'var(--y4)',
          blocked: 'var(--r4)', rejected: 'var(--r3)',
        };
        return <div key={k} style={{ width: `${pct}%`, background: colors[k] ?? 'var(--i4)', height: '100%' }} title={`${k}: ${v}`} />;
      })}
    </div>
  )}
</SectionCard>
```

- [ ] **Step 3: Add intercepted actions table under squad health**

After the squad health section and before the work queue section:

```tsx
{/* Blocked/Intercepted Actions */}
<SectionCard title="被拦截动作" style={{ marginBottom: 16 }}>
  <Table
    rowKey="id"
    loading={interceptedLoading}
    dataSource={interceptedData?.items ?? []}
    size="small"
    pagination={false}
    columns={[
      { title: '类型', dataIndex: 'action_type', width: 140 },
      { title: 'Agent', dataIndex: 'agent_id', width: 100 },
      { title: '风险', dataIndex: 'risk_level', width: 80,
        render: (v: string) => <Tag color={riskColor(v)}>{v}</Tag> },
      { title: '拦截原因', dataIndex: 'block_reason', width: 160,
        render: (v: string) => {
          const reasons: Record<string, string> = {
            approval_required: '缺少审批',
            L4_blocked: 'L4 自主执行阻止',
            rate_limited: '频率限制',
            policy_blocked: '策略拦截',
          };
          return <Tag color="red">{reasons[v] ?? v}</Tag>;
        }
      },
      { title: '时间', dataIndex: 'blocked_at', width: 150 },
      { title: '目标', dataIndex: 'target_summary', ellipsis: true },
    ]}
  />
</SectionCard>
```

- [ ] **Step 4: Add audit replay drawer (modal opened from work item context)**

Add an `auditReplayCorrelationId` state and a query for replay data:

```typescript
const [auditReplayCorrelationId, setAuditReplayCorrelationId] = useState<string | null>(null);
const { data: auditReplayData, isLoading: replayLoading } = useQuery({
  queryKey: ['audit-replay', auditReplayCorrelationId],
  queryFn: async () => {
    const res = await apiClient.get<AuditReplayResponse>(`/v1/agentos/audit-replay/${auditReplayCorrelationId}`);
    return res.data;
  },
  enabled: !!auditReplayCorrelationId,
});
```

Add a "审计回放" button to the work item table's action column (before "批准"):
```tsx
<Button
  size="small"
  icon={<BranchesOutlined />}
  onClick={(e) => {
    e.stopPropagation();
    // Fetch trace_id from work item or use correlation_id
    setAuditReplayCorrelationId(record.trace_id ?? record.id);
  }}
>
  回放
</Button>
```

Add the audit replay drawer:
```tsx
<Drawer
  title={`审计回放: ${auditReplayCorrelationId ?? ''}`}
  open={!!auditReplayCorrelationId}
  onClose={() => setAuditReplayCorrelationId(null)}
  width={640}
  loading={replayLoading}
>
  {auditReplayData?.events?.length ? (
    <Timeline
      items={auditReplayData.events.map((evt) => ({
        color: evt.type === 'action' ? 'blue' : evt.type === 'approval' ? 'orange' : evt.type === 'audit' ? 'green' : 'gray',
        children: (
          <div>
            <div><Text strong>{evt.type}</Text> {evt.subtype && `— ${evt.subtype}`}</div>
            {evt.agent_id && <div><Text type="secondary">Agent: {evt.agent_id}</Text></div>}
            {evt.status && <Tag color={statusColor(evt.status)}>{evt.status}</Tag>}
            {evt.detail && <div><Text type="secondary">{evt.detail}</Text></div>}
            <div><Text type="secondary" style={{ fontSize: '0.75rem' }}>{evt.timestamp}</Text></div>
          </div>
        ),
      }))}
    />
  ) : (
    <Empty description="无审计记录" />
  )}
</Drawer>
```

- [ ] **Step 5: Add required imports**

Make sure `BranchesOutlined`, `Timeline`, `Drawer` are imported from antd:

```typescript
import { ..., Timeline, Drawer } from 'antd';
import { ..., BranchesOutlined } from '@ant-design/icons';
```

- [ ] **Step 6: Build & lint**

```bash
cd frontend-next && npm run build 2>&1 | tail -20
```

Expected: Build succeeds

- [ ] **Step 7: Commit**

```bash
git add frontend-next/src/app/(main)/agentos/page.tsx
git commit -m "feat: P3 traffic section — funnel cards, intercepted actions table, audit replay drawer"
```

---

### Task E1: P4 frontend — Agent health cards + external health panel + congestion indicator

**Files:**
- Modify: `frontend-next/src/app/(main)/agentos/page.tsx`

**Interfaces:**
- Consumes: `GET /v1/agentos/agent-metrics`, `GET /v1/agentos/traffic-summary`
- Produces: Health cards, external platform panel, congestion alert

- [ ] **Step 1: Add query for agent-metrics**

```typescript
const { data: agentMetricsData, isLoading: metricsLoading } = useQuery({
  queryKey: ['agent-metrics'],
  queryFn: async () => {
    const res = await apiClient.get<{agents: AgentMetricsEntry[]}>('/v1/agentos/agent-metrics');
    return res.data?.agents ?? [];
  },
});

// Types
interface AgentMetricsEntry {
  agent_id: string;
  run_count: number;
  success_count: number;
  failure_count: number;
  blocked_count: number;
  approval_rate: number;
  owner_acceptance_rate: number;
  avg_latency_ms: number;
  external_failure_rate: number;
  health: string;
}

interface ExternalPlatformHealth {
  platform: string;
  total_calls: number;
  failed_calls: number;
  consecutive_failures: number;
  degraded: boolean;
  last_failure_at: string;
  last_error: string;
}
```

- [ ] **Step 2: Add Agent health cards section**

After the Squad health map section and before the work queue section:

```tsx
{/* Agent Health Cards */}
<SectionCard title="Agent 健康" style={{ marginBottom: 16 }}>
  <Spin spinning={metricsLoading}>
    {agentMetricsData && agentMetricsData.length > 0 ? (
      <Row gutter={[12, 12]}>
        {agentMetricsData.map((m) => (
          <Col xs={24} sm={12} lg={8} key={m.agent_id}>
            <div style={{
              background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8,
              borderLeft: `4px solid ${m.health === 'ok' ? 'var(--g4)' : m.health === 'warn' ? 'var(--y4)' : 'var(--r4)'}`,
              padding: 12,
            }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                <Text strong>{m.agent_id}</Text>
                <Tag color={healthColor(m.health)}>{m.health}</Tag>
              </div>
              <Space direction="vertical" size={4} style={{ width: '100%' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Text type="secondary">运行</Text>
                  <Text>{m.run_count}</Text>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Text type="secondary">成功/失败/拦截</Text>
                  <Text>{m.success_count}/{m.failure_count}/{m.blocked_count}</Text>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Text type="secondary">采纳率</Text>
                  <Text>{(m.owner_acceptance_rate * 100).toFixed(0)}%</Text>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Text type="secondary">平均延迟</Text>
                  <Text>{(m.avg_latency_ms / 1000).toFixed(1)}s</Text>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Text type="secondary">外部失败率</Text>
                  <Tag color={m.external_failure_rate > 0.2 ? 'red' : 'green'}>
                    {(m.external_failure_rate * 100).toFixed(0)}%
                  </Tag>
                </div>
              </Space>
            </div>
          </Col>
        ))}
      </Row>
    ) : !metricsLoading ? (
      <Empty description="暂无 Agent 指标数据" />
    ) : null}
  </Spin>
</SectionCard>
```

- [ ] **Step 3: Add congestion indicator in the top bar area**

After the refresh button in the header:

```tsx
{/* Congestion alert banner */}
{(() => {
  const blocked = trafficSummary?.funnel?.blocked_by_policy ?? 0;
  const unhealthy = (agentMetricsData ?? []).filter(m => m.health !== 'ok').length;
  if (blocked > 0 || unhealthy > 0) {
    return (
      <Alert
        type="warning"
        showIcon
        style={{ marginBottom: 12 }}
        message={
          <Space>
            {blocked > 0 && <span>🚫 {blocked} 个动作被拦截</span>}
            {unhealthy > 0 && <span>⚠️ {unhealthy} 个 Agent 异常</span>}
          </Space>
        }
      />
    );
  }
  return null;
})()}
```

- [ ] **Step 4: Add query for external platform health**

```typescript
const { data: externalHealthData, isLoading: extHealthLoading } = useQuery({
  queryKey: ['external-health'],
  queryFn: async () => {
    const res = await apiClient.get<ExternalPlatformHealth[]>('/v1/agentos/external-health');
    return res.data ?? [];
  },
});
```

Add after the Agent Health Cards section:

```tsx
{/* External Platform Health */}
<SectionCard title="外部平台健康" style={{ marginBottom: 16 }}>
  <Table
    rowKey="platform"
    dataSource={externalHealthData ?? []}
    size="small"
    pagination={false}
    columns={[
      { title: '平台', dataIndex: 'platform', width: 120 },
      { title: '调用总数', dataIndex: 'total_calls', width: 100 },
      { title: '失败数', dataIndex: 'failed_calls', width: 80 },
      { title: '连续失败', dataIndex: 'consecutive_failures', width: 100 },
      {
        title: '状态', dataIndex: 'degraded', width: 100,
        render: (v: boolean) => v
          ? <Tag color="red">降级</Tag>
          : <Tag color="green">正常</Tag>
      },
      { title: '最后失败', dataIndex: 'last_failure_at', width: 160 },
      { title: '错误', dataIndex: 'last_error', ellipsis: true },
    ]}
  />
</SectionCard>
```

Also add external health API endpoint to handler.go + routes.go:

```go
// ExternalHealth GET /agentos/external-health
func (h *Handler) ExternalHealth(c *gin.Context) {
    response.Success(c, h.service.ExternalHealth())
}
```

```go
agentos.GET("/external-health", h.ExternalHealth)
```

And in service.go, add ExternalHealth method that returns from toolbridge tracker (placeholder if tracker not configured):

```go
func (s *Service) ExternalHealth() []interface{} {
    // Returns external platform health from ToolBridge tracker if available.
    // Defaults to empty list if tracker not configured.
    return []interface{}{}
}
```

- [ ] **Step 6: Build**

```bash
cd frontend-next && npm run build 2>&1 | tail -20
```

Expected: Build succeeds

- [ ] **Step 7: Commit**

```bash
git add frontend-next/src/app/(main)/agentos/page.tsx
git commit -m "feat: P4 health views — Agent health cards, congestion alert"
```

---

## Verification

- [ ] **Run all backend tests**
```bash
cd backend-go && go test ./internal/platform/command/ ./internal/platform/actioncatalog/ ./internal/platform/toolbridge/ ./internal/agentos/ ./internal/domain/entropy/ ./internal/realtime/
```

- [ ] **Run all frontend checks**
```bash
cd frontend-next && npm run build
```

- [ ] **Manual verification**
  1. `GET /v1/agentos/traffic-summary` returns status distribution and funnel
  2. `GET /v1/agentos/intercepted-actions` returns blocked/rejected actions
  3. `GET /v1/agentos/audit-replay/:correlation_id` returns event timeline
  4. `GET /v1/agentos/agent-metrics` returns per-agent metrics
  5. WebSocket broadcasts `agent.action.status_changed`
  6. Frontend shows traffic funnel cards, intercepted table, health cards
  7. Frontend audit replay drawer shows timeline
  8. Frontend congestion alert shows when blocked > 0
