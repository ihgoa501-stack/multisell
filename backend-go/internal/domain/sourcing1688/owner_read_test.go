package sourcing1688

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func newOwnerReadDB(t *testing.T) *Service {
	t.Helper()
	db := dbtest.NewDB(t,
		&Sourcing1688Product{}, &Sourcing1688Snapshot{}, &draftRow{}, &demandCaseRow{},
		&experimentRow{}, &listingRow{}, &productRow{}, &skuRow{}, &mediaRow{}, &costRow{},
	)
	return NewService(db, dbtest.NewLogger(t))
}

func seedOwnerReadFixture(t *testing.T, svc *Service) int64 {
	t.Helper()
	tx := svc.db
	now := time.Date(2026, 7, 12, 8, 0, 0, 0, time.UTC)
	title, price := "Owner reviewed offer", 12.5
	demand := demandCaseRow{ID: 7, OwnerID: 42, SalesChannel: "test-channel", Status: "experiment_ready"}
	experiment := experimentRow{ExperimentID: "exp-owner", OwnerID: 42, Status: "active", Stage: "supply"}
	source := Sourcing1688Product{ID: 8, SourceURL: "https://detail.1688.com/offer/8.html?access_token=secret#private", Title: &title, Price: &price, MOQ: 2, SupplierName: "SECRET SUPPLIER", DemandCaseID: ptrInt64(7), ExperimentID: ptrString("exp-owner"), SnapshotID: ptrInt64(9), LifecycleStatus: "editing", RawData: rawJSON(`{"secret":"raw-data"}`)}
	snapshot := Sourcing1688Snapshot{ID: 9, SourcingProductID: 8, SourceURL: source.SourceURL, CollectedAt: now, CollectedBy: 42, Driver: "controlled", ParserVersion: "v1", RawPayload: json.RawMessage(`{"secret":"raw-payload"}`), RawSHA256: strings.Repeat("a", 64), ObservedTitle: &title, ObservedPrice: &price, ObservedMOQ: 2, ObservedSupplier: "Supplier"}
	product := productRow{ID: 10, Name: "Internal draft", Description: "SECRET DESCRIPTION", Status: 0}
	listing := listingRow{ID: 11, ProductID: 10, PlatformID: 3, PlatformSKU: "SECRET-PLATFORM-SKU", Status: "draft", PublishedData: json.RawMessage(`{"secret":"published-data"}`)}
	draft := draftRow{ID: 12, SourcingProductID: 8, SnapshotID: 9, ProductID: 10, ListingID: 11, DemandCaseID: 7, ExperimentID: "exp-owner", CreatedBy: 42, ApprovalStatus: "editing"}
	rows := []interface{}{&demand, &experiment, &source, &snapshot, &product, &listing, &draft,
		&mediaRow{ID: 13, ProductID: 10, SourceSnapshotID: 9, ProcessedURL: "internal://processed/13?token=secret", MediaRole: "main", RightsStatus: "verified", RightsEvidenceURI: "internal://rights/13?token=secret", CreatedAt: now},
		&mediaRow{ID: 14, ProductID: 10, SourceSnapshotID: 999, ProcessedURL: "internal://wrong-snapshot", MediaRole: "detail"},
		&costRow{ID: 15, ProductID: 10, ExperimentID: "exp-owner", CostType: "purchase", Amount: 12.5, Currency: "CNY", TruthStatus: "quoted", SourceURI: "internal://cost/15?token=secret", ObservedAt: now},
		&costRow{ID: 16, ProductID: 10, ExperimentID: "other-experiment", CostType: "shipping", Amount: 999, Currency: "CNY", TruthStatus: "estimated", SourceURI: "internal://wrong-cost", ObservedAt: now},
	}
	for _, row := range rows {
		if err := tx.Create(row).Error; err != nil {
			t.Fatalf("seed %T: %v", row, err)
		}
	}
	return source.ID
}

func TestReadOwnerViewReturnsOnlyConsistentSafeOwnerData(t *testing.T) {
	svc := newOwnerReadDB(t)
	sourceID := seedOwnerReadFixture(t, svc)

	got, err := svc.ReadOwnerView(context.Background(), sourceID, 42)
	if err != nil {
		t.Fatalf("ReadOwnerView: %v", err)
	}
	if got.Source.ID != sourceID || got.Snapshot.ID != 9 || got.Draft == nil || got.Draft.ID != 12 {
		t.Fatalf("missing controlled chain: %#v", got)
	}
	if len(got.Media) != 1 || got.Media[0].ID != 13 || len(got.Costs) != 1 || got.Costs[0].ID != 15 {
		t.Fatalf("unscoped media/cost leaked: media=%#v costs=%#v", got.Media, got.Costs)
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, forbidden := range []string{"raw_payload", "raw_data", "published_data", "raw-payload", "published-data", "wrong-snapshot", "wrong-cost", "secret", "SECRET SUPPLIER", "SECRET DESCRIPTION", "SECRET-PLATFORM-SKU", "internal://"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("owner view leaked %q: %s", forbidden, text)
		}
	}
	if got.Source.SourceReference != "detail.1688.com/offer/8.html" || got.Snapshot.SourceReference != "detail.1688.com/offer/8.html" {
		t.Fatalf("source URL not redacted: source=%q snapshot=%q", got.Source.SourceReference, got.Snapshot.SourceReference)
	}
	if got.Media[0].TruthStatus != "unknown" || got.Media[0].ObservedAt != nil {
		t.Fatalf("media evidence was upgraded without backing columns: %#v", got.Media[0])
	}
	if len(got.Limitations) == 0 {
		t.Fatal("owner view must state limitations")
	}
}

func TestReadOwnerViewUsesRepeatableReadReadOnlyTransaction(t *testing.T) {
	opts := ownerReadTxOptions()
	if !opts.ReadOnly || opts.Isolation.String() != "Repeatable Read" {
		t.Fatalf("transaction options = %#v", opts)
	}
	svc := newOwnerReadDB(t)
	if _, err := svc.ReadOwnerView(context.Background(), seedOwnerReadFixture(t, svc), 42); err != nil {
		t.Fatalf("configured transaction unsupported by test database: %v", err)
	}
}

func TestReadOwnerViewRejectsWrongDraftCreatorListingOrLifecycle(t *testing.T) {
	tests := []struct {
		name   string
		update func(*Service, int64) error
	}{
		{"draft creator", func(s *Service, id int64) error {
			return s.db.Model(&draftRow{}).Where("sourcing_product_id = ?", id).Update("created_by", 99).Error
		}},
		{"published listing", func(s *Service, _ int64) error {
			return s.db.Model(&listingRow{}).Where("id = ?", 11).Update("status", "published").Error
		}},
		{"editing pending approval", func(s *Service, id int64) error {
			return s.db.Model(&draftRow{}).Where("sourcing_product_id = ?", id).Update("approval_status", "pending").Error
		}},
		{"pending lifecycle without pending approval", func(s *Service, id int64) error {
			if err := s.db.Model(&Sourcing1688Product{}).Where("id = ?", id).Update("lifecycle_status", LifecyclePendingApproval).Error; err != nil {
				return err
			}
			return s.db.Model(&draftRow{}).Where("sourcing_product_id = ?", id).Update("approval_status", "editing").Error
		}},
		{"approved lifecycle without approval", func(s *Service, id int64) error {
			if err := s.db.Model(&Sourcing1688Product{}).Where("id = ?", id).Update("lifecycle_status", LifecycleApprovedDraft).Error; err != nil {
				return err
			}
			return s.db.Model(&draftRow{}).Where("sourcing_product_id = ?", id).Update("approval_status", "pending").Error
		}},
		{"draft in pre-draft lifecycle", func(s *Service, id int64) error {
			return s.db.Model(&Sourcing1688Product{}).Where("id = ?", id).Update("lifecycle_status", LifecyclePendingReview).Error
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newOwnerReadDB(t)
			id := seedOwnerReadFixture(t, svc)
			if err := tt.update(svc, id); err != nil {
				t.Fatal(err)
			}
			if _, err := svc.ReadOwnerView(context.Background(), id, 42); !errors.Is(err, ErrWorkflowGate) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestReadOwnerViewDoesNotCrossOwnerBoundary(t *testing.T) {
	svc := newOwnerReadDB(t)
	sourceID := seedOwnerReadFixture(t, svc)
	if _, err := svc.ReadOwnerView(context.Background(), sourceID, 99); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("cross-owner error = %v, want workflow gate", err)
	}
}

func TestReadOwnerViewRejectsInconsistentDraftBinding(t *testing.T) {
	svc := newOwnerReadDB(t)
	sourceID := seedOwnerReadFixture(t, svc)
	if err := svc.db.Model(&draftRow{}).Where("sourcing_product_id = ?", sourceID).Update("snapshot_id", 999).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReadOwnerView(context.Background(), sourceID, 42); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("inconsistent draft error = %v, want workflow gate", err)
	}
}

func ptrInt64(v int64) *int64           { return &v }
func ptrString(v string) *string        { return &v }
func rawJSON(v string) *json.RawMessage { r := json.RawMessage(v); return &r }
