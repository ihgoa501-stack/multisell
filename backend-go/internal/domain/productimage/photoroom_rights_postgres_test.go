package productimage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// This test intentionally requires a disposable, fully-migrated PostgreSQL
// database. The caller owns database creation and deletion; snapshots are
// immutable by design and therefore this test does not pretend it can clean
// one out of a shared database safely.
func TestPostgresPhotoroomExecuteRevoke100WayMutualExclusion(t *testing.T) {
	dsn := os.Getenv("PRODUCTIMAGE_POSTGRES_DESTRUCTIVE_TEST_DSN")
	if dsn == "" {
		t.Skip("set PRODUCTIMAGE_POSTGRES_DESTRUCTIVE_TEST_DSN to a disposable migrated PostgreSQL database")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(120)

	remote := &photoroomCapabilityImageService{newFakeImageService()}
	svc := NewService(db, dbtest.NewLogger(t), remote, strings.Repeat("k", 32))
	owner := time.Now().UnixNano() & 0x3fffffffffffffff
	_, in, grant := seedPhotoroomInput(t, svc, owner)
	in.IdempotencyKey = fmt.Sprintf("photoroom-race-task-%d", owner)
	task, err := svc.CreateTask(t.Context(), owner, in)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err = svc.CreateBudgetPolicy(t.Context(), owner, BudgetPolicyInput{Currency: "USD", PeriodStart: now.Add(-time.Hour), PeriodEnd: now.Add(time.Hour), TotalAmount: "1", IdempotencyKey: fmt.Sprintf("race-budget-%d", owner)}); err != nil {
		t.Fatal(err)
	}
	approval, err := svc.ApproveExecution(t.Context(), owner, task.ID, ApprovalInput{Processor: photoroomProcessor, MaxCost: "0", Currency: "USD", ExpectedVersion: task.Version})
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	var revokeSuccess atomic.Int64
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			if i%2 == 0 {
				_, _ = svc.Execute(t.Context(), owner, task.ID, fmt.Sprintf("execute-%d-%d", owner, i))
				return
			}
			if _, err := svc.RevokeRightsGrant(t.Context(), owner, grant.ID, RevokeRightsInput{ExpectedVersion: 1, IdempotencyKey: fmt.Sprintf("revoke-%d-%d", owner, i), Reason: "concurrent owner revocation"}); err == nil {
				revokeSuccess.Add(1)
			}
		}(i)
	}
	close(start)
	wg.Wait()
	if revokeSuccess.Load() != 1 {
		t.Fatalf("exactly one revocation must commit, got %d", revokeSuccess.Load())
	}
	if remote.authorizedExecuteCalls < 0 || remote.authorizedExecuteCalls > 1 {
		t.Fatalf("provider enqueue calls=%d, want 0 or 1", remote.authorizedExecuteCalls)
	}
	var snapshots []ExecutionRightsSnapshot
	if err := db.Where("owner_id=? AND approval_id=?", owner, approval.ID).Find(&snapshots).Error; err != nil {
		t.Fatal(err)
	}
	var storedApproval ExecutionApproval
	if err := db.First(&storedApproval, approval.ID).Error; err != nil {
		t.Fatal(err)
	}
	if remote.authorizedExecuteCalls == 0 {
		if len(snapshots) != 0 || storedApproval.ConsumedAt != nil {
			t.Fatalf("revoke-first must make zero snapshot/claim: snapshots=%d consumed=%v", len(snapshots), storedApproval.ConsumedAt)
		}
		return
	}
	if len(snapshots) != 1 || storedApproval.ConsumedAt == nil {
		t.Fatalf("execute-first must atomically freeze one snapshot and claim approval: snapshots=%d consumed=%v", len(snapshots), storedApproval.ConsumedAt)
	}
	snapshot := snapshots[0]
	if snapshot.GrantID != grant.ID || snapshot.GrantVersion != 1 || snapshot.EvidenceSHA != grant.EvidenceSHA || snapshot.GrantRequestHash != grant.RequestHash {
		t.Fatalf("wrong frozen grant evidence: %+v", snapshot)
	}
	parts := strings.Split(remote.lastExecutionToken, ".")
	if len(parts) != 3 {
		t.Fatal("missing signed execution token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatal(err)
	}
	if int64(claims["execution_rights_snapshot_id"].(float64)) != snapshot.ID || int64(claims["rights_grant_id"].(float64)) != grant.ID || int64(claims["rights_grant_version"].(float64)) != 1 || claims["rights_evidence_sha256"] != grant.EvidenceSHA {
		t.Fatalf("provider authorization is not bound to frozen rights: %+v", claims)
	}
}
