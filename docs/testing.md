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

## Testing Execution Gates

The listingtask module enforces pre-execution gates. Test coverage includes:

### State Machine Gate
```go
func TestService_StateMachine_InvalidTransitions(t *testing.T) {
    svc := NewService(db, logger, nil, false, nil)
    task, _ := svc.Create(&CreateTaskInput{ProductID: 1, PlatformID: 10})

    // blocked -> completed is invalid
    _, err := svc.Update(task.ID, &UpdateTaskInput{Status: strPtr("completed")})
    if err == nil { t.Fatal("should be rejected") }
}
```

### Approval Gate
```go
// ExecuteTask without approval returns error
_, err := svc.ExecuteTask(task.ID)
if err == nil { t.Fatal("should require approval") }

// Create an approved approval first
db.Create(&approval.ApprovalRequest{ProductID: 1, RequestType: "publish", Status: "approved", ExpiresAt: &future})

// Now execution succeeds
executed, err := svc.ExecuteTask(task.ID)
if err != nil { t.Fatalf("ExecuteTask: %v", err) }
```

### Idempotency Gate
```go
// completed task cannot execute again
_, err = svc.ExecuteTask(task.ID)
if err == nil { t.Fatal("idempotency guard should block duplicate execution") }
```

### Agent Feedback Tests
```go
// Submit accepted/rejected feedback
updated, err := svc.SubmitFeedback(taskID, "accepted", "good suggestion", "user-1")

// Invalid status rejected
_, err = svc.SubmitFeedback(taskID, "invalid", "", "user-1")
if err == nil { t.Fatal("expected error") }
```

Run:
```bash
cd backend-go && go test ./internal/domain/listingtask/... -v -run "TestService_StateMachine|TestService_SubmitFeedback|TestService_Task_Execute"
```
