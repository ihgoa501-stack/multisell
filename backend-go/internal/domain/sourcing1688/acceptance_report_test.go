package sourcing1688

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
	"gorm.io/gorm"
)

func newAcceptanceTestService(t *testing.T) (*Service, *gorm.DB, int64) {
	t.Helper()
	db := dbtest.NewDB(t,
		&Sourcing1688Product{}, &Sourcing1688Snapshot{}, &SourcingChangeEvent{}, &DuplicateCandidate{},
		&ImageProcessingRecord{}, &CaptureAttempt{}, &demandCaseRow{}, &experimentRow{}, &gateRow{},
		&objectLinkRow{}, &platformRow{}, &productRow{}, &skuRow{}, &mediaRow{}, &costRow{}, &authoritativeSupplierRow{}, &authoritativeProductSupplierRow{}, &SourcingSKUMapping{},
		&listingRow{}, &draftRow{}, &approval.ApprovalRequest{}, &operationlog.OperationLog{}, &PublishAttempt{},
		&Sourcing1688TaskLink{}, &sourcingOpportunityRow{}, &sourcingOpportunityDecisionRow{}, &sourcingMarketDecisionRow{},
		&SourcingSample{}, &SourcingSampleEvent{}, &SourcingCostVersion{}, &SourcingCostLine{}, &SourcingComplianceEvidence{},
	)
	if err := db.Exec(`CREATE TRIGGER test_acceptance_apply_sourcing_approval
		AFTER UPDATE OF status ON approval_request
		WHEN NEW.request_type = 'sourcing_1688_draft' AND OLD.status = 'pending'
		BEGIN
			UPDATE sourcing_listing_draft SET approval_status = NEW.status,
				approval_rejection_reason = CASE WHEN NEW.status = 'rejected' THEN NEW.review_note ELSE '' END
			WHERE id = NEW.target_id AND approval_id = NEW.id;
			UPDATE sourcing_1688_product SET lifecycle_status = CASE WHEN NEW.status = 'approved' THEN 'approved_draft' ELSE 'editing' END,
				lifecycle_actor_id = NEW.reviewer_user_id,
				lifecycle_reason = CASE WHEN NEW.status = 'rejected' THEN NEW.review_note ELSE '' END
			WHERE id = (SELECT sourcing_product_id FROM sourcing_listing_draft WHERE id = NEW.target_id)
				AND lifecycle_status = 'pending_approval';
		END`).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, dbtest.NewLogger(t))
	now := time.Now().UTC().Truncate(time.Second)
	db.Create(&demandCaseRow{ID: 7, OwnerID: 42, Region: "RU", Consumer: "城市养猫家庭", NeedScenario: "日常互动", SalesChannel: "Ozon", TargetLocale: "ru-RU", Status: "experiment_ready", CreatedAt: now, UpdatedAt: now})
	db.Create(&experimentRow{ExperimentID: "EXP-REAL-1", OwnerID: 42, Status: "active", Stage: "product"})
	db.Create(&gateRow{ExperimentID: "EXP-REAL-1", Stage: "opportunity", Result: "pass"})
	db.Create(&objectLinkRow{ExperimentID: "EXP-REAL-1", ObjectType: "demand_case", ObjectID: "7"})
	db.Create(&platformRow{ID: 3, Name: "Ozon", Code: "ozon", Status: 1})

	page := toolbridge.PageData{
		SourceURL: "https://detail.1688.com/offer/987654321.html", Title: "真实猫玩具", PriceCNY: 12,
		MOQ: 2, Images: []string{"https://cbu01.alicdn.com/a.png"},
		SpecVariants: []toolbridge.SpecVariant{{Spec: "红色/标准", Price: 12, Stock: 100}},
		SupplierName: "真实供应商", Description: "页面说明", RawHTML: `<main data-offer="987654321">真实页面结构</main>`,
		CollectedAt: now, Driver: "plugin", ParserVersion: "lingmirror-extension@0.1.0", SupplierBusinessID: "supplier-real-987", CollectionRequestID: "req_acceptance-real-987",
	}
	raw, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	title, price, moq := page.Title, page.PriceCNY, page.MOQ
	images, _ := json.Marshal(page.Images)
	variants, _ := json.Marshal(page.SpecVariants)
	source, err := svc.Capture(&CaptureInput{DemandCaseID: 7, ExperimentID: "EXP-REAL-1", SourceURL: page.SourceURL, CollectedAt: now, CollectedBy: 42, Driver: page.Driver, ParserVersion: page.ParserVersion, RawPayload: raw, Title: &title, Price: &price, MOQ: &moq, SupplierName: page.SupplierName, SupplierBusinessID: page.SupplierBusinessID, Images: images, SkuVariants: variants, CaptureMode: CaptureModeControlledFetch, CollectionRequestID: page.CollectionRequestID})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	seedFrozenOpportunityAuthority(t, db, source.ID, 7, 42, "EXP-REAL-1", "Ozon")
	if _, err := svc.Review(source.ID, &ReviewInput{ReviewedBy: 42, Notes: "Owner 对照真实页面复核"}); err != nil {
		t.Fatalf("Review: %v", err)
	}
	processed, err := svc.ProcessImage(&ProcessImageInput{SourcingProductID: source.ID, SourceURL: page.Images[0], SourceBase64: testSourceImageBase64(), Width: 1200, Height: 1200, Format: "jpeg", Quality: 90, RightsEvidenceURI: "evidence://rights/real-1", RightsTruthStatus: "actual", RightsObservedAt: now, ChannelRuleURI: "evidence://ozon/images/current", ProcessedBy: 42})
	if err != nil {
		t.Fatalf("ProcessImage: %v", err)
	}
	in := completeConvertInput(now)
	in.Media[0].ProcessingRecordID = processed.RecordID
	in.Media[0].ProcessedURL = processed.ContentURL
	in.Media[0].ContentSHA256 = processed.ProcessedSHA256
	in.Media[0].Operations = processed.Operations
	in.Media[0].RightsEvidenceURI = processed.RightsEvidenceURI
	in.Media[0].RightsObservedAt = processed.RightsObservedAt
	in.Media[0].ChannelRuleURI = "evidence://ozon/images/current"
	in.Validation.Images[0].SourceURI = "sha256:" + processed.ProcessedSHA256
	_, err = svc.Convert(source.ID, in)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	var task Sourcing1688TaskLink
	if err := db.Where("sourcing_product_id = ? AND owner_id = ? AND is_primary = ?", source.ID, int64(42), true).First(&task).Error; err != nil {
		t.Fatal(err)
	}
	var mappings []SourcingSKUMapping
	if err := db.Where("sourcing_product_id = ? AND task_link_id = ?", source.ID, task.ID).Find(&mappings).Error; err != nil {
		t.Fatal(err)
	}
	for _, mapping := range mappings {
		lines := make([]CreateSourcingCostLineInput, 0, len(requiredCostTypes))
		for _, costType := range requiredCostTypes {
			lines = append(lines, CreateSourcingCostLineInput{CostType: costType, AmountMinor: 100, Currency: "RUB", NormalizedAmountMinor: 100, TruthStatus: "actual", SourceURI: "evidence://cost/" + costType, ObservedAt: now})
		}
		if _, err := svc.CreateSourcingCostVersion(42, source.ID, &CreateSourcingCostVersionInput{TaskLinkID: task.ID, SourceSnapshotID: *source.SnapshotID, SKUMappingID: mapping.ID, TargetCurrency: "RUB", Lines: lines}); err != nil {
			t.Fatalf("CreateSourcingCostVersion: %v", err)
		}
	}
	var converted Sourcing1688Product
	if err := db.First(&converted, source.ID).Error; err != nil || converted.ProductID == nil {
		t.Fatalf("converted source: %v", err)
	}
	for _, code := range []string{"brand_ip", "patent", "certification", "dangerous_goods", "material", "labeling_instructions"} {
		row, err := svc.CreateComplianceEvidence(source.ID, task.ID, &CreateComplianceEvidenceInput{OwnerID: 42, ProductID: *converted.ProductID, CountryCode: "RU", ChannelCode: "Ozon", RequirementCode: code, RequirementText: "已核验 " + code, EvidenceSource: "evidence://compliance/" + code, TruthStatus: "actual", Scope: "product", ObservedAt: now})
		if err != nil {
			t.Fatalf("CreateComplianceEvidence: %v", err)
		}
		if _, err := svc.ReviewComplianceEvidence(source.ID, task.ID, row.ID, &ReviewComplianceEvidenceInput{OwnerID: 42, Decision: ComplianceReviewApproved, Notes: "Owner 复核"}); err != nil {
			t.Fatalf("ReviewComplianceEvidence: %v", err)
		}
	}
	submitted, err := svc.SubmitDraftApproval(source.ID, &DraftApprovalSubmissionInput{RequesterID: 42, Reason: "批准内部草稿"})
	if err != nil {
		t.Fatalf("SubmitDraftApproval: %v", err)
	}
	if _, err := svc.DecideDraftApproval(source.ID, submitted.ApprovalID, &DraftApprovalDecisionInput{OwnerID: 42, Action: "approve", Note: "Owner 已复核当前内容"}); err != nil {
		t.Fatalf("DecideDraftApproval: %v", err)
	}
	return svc, db, source.ID
}

func TestAcceptanceReportBlocksLegacyExperimentOnlyAuthority(t *testing.T) {
	svc, db, sourceID := newAcceptanceTestService(t)
	if err := db.Model(&Sourcing1688TaskLink{}).Where("sourcing_product_id = ?", sourceID).Updates(map[string]any{
		"authority_kind": "legacy_experiment", "product_opportunity_id": nil, "opportunity_decision_id": nil,
	}).Error; err != nil {
		t.Fatal(err)
	}
	report, err := svc.BuildAcceptanceReport(context.Background(), sourceID, 42)
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready || report.Items[0].Status != AcceptanceBlocked || !strings.Contains(strings.Join(report.Items[0].Blockers, " "), "legacy experiment trace cannot authorize") {
		t.Fatalf("legacy authority must block report: ready=%v market=%#v", report.Ready, report.Items[0])
	}
}

// This synthetic fixture verifies deterministic report logic only. It is not
// external_observed evidence and must never be reported as a real 1688 run.
func TestAcceptanceReportReadyOnlyForCompletePersistedFixture(t *testing.T) {
	svc, _, sourceID := newAcceptanceTestService(t)
	report, err := svc.BuildAcceptanceReport(context.Background(), sourceID, 42)
	if err != nil {
		t.Fatalf("BuildAcceptanceReport: %v", err)
	}
	if !report.Ready || report.Status != AcceptancePassed || len(report.Items) != 15 {
		t.Fatalf("report ready=%v status=%s items=%d: %#v", report.Ready, report.Status, len(report.Items), report.Items)
	}
	for _, item := range report.Items {
		if item.Status != AcceptancePassed {
			t.Fatalf("item %d %s = %s, blockers=%v", item.Number, item.Code, item.Status, item.Blockers)
		}
	}
}

func TestAcceptanceReportDoesNotTreatModuleOrDraftAsRealAcceptance(t *testing.T) {
	svc, db, sourceID := newAcceptanceTestService(t)
	var source Sourcing1688Product
	if err := db.First(&source, sourceID).Error; err != nil {
		t.Fatal(err)
	}
	var snapshot Sourcing1688Snapshot
	if err := db.First(&snapshot, *source.SnapshotID).Error; err != nil {
		t.Fatal(err)
	}
	var page toolbridge.PageData
	if err := json.Unmarshal(snapshot.RawPayload, &page); err != nil {
		t.Fatal(err)
	}
	page.RawHTML = ""
	raw, _ := json.Marshal(page)
	sum := sha256ForTest(raw)
	if err := db.Model(&Sourcing1688Snapshot{}).Where("id = ?", snapshot.ID).Updates(map[string]any{"raw_payload": raw, "raw_sha256": sum}).Error; err != nil {
		t.Fatal(err)
	}

	report, err := svc.BuildAcceptanceReport(context.Background(), sourceID, 42)
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready || report.Items[1].Status != AcceptanceBlocked || report.Items[14].Status != AcceptanceUnknown {
		t.Fatalf("raw evidence must block real acceptance: ready=%v raw=%s real=%s", report.Ready, report.Items[1].Status, report.Items[14].Status)
	}
}

func TestAcceptanceReportRejectsSpoofedManualCaptureProvenance(t *testing.T) {
	svc, db, sourceID := newAcceptanceTestService(t)
	var source Sourcing1688Product
	if err := db.First(&source, sourceID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&Sourcing1688Snapshot{}).Where("id = ?", *source.SnapshotID).Update("capture_mode", CaptureModeManualImport).Error; err != nil {
		t.Fatal(err)
	}
	report, err := svc.BuildAcceptanceReport(context.Background(), sourceID, 42)
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready || report.Items[1].Status != AcceptanceBlocked || report.Items[14].Status != AcceptanceUnknown {
		t.Fatalf("manual capture spoof must not pass: ready=%v raw=%s real=%s", report.Ready, report.Items[1].Status, report.Items[14].Status)
	}
}

func TestAcceptanceReportRejectsContentChangedAfterDraftApproval(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*gorm.DB, int64) error
	}{
		{"product", func(db *gorm.DB, productID int64) error {
			return db.Model(&productRow{}).Where("id = ?", productID).Update("name", "tampered product").Error
		}},
		{"listing", func(db *gorm.DB, productID int64) error {
			return db.Model(&listingRow{}).Where("product_id = ?", productID).Update("platform_sku", "tampered-listing").Error
		}},
		{"sku", func(db *gorm.DB, productID int64) error {
			return db.Model(&skuRow{}).Where("product_id = ?", productID).Update("price", 999).Error
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, db, sourceID := newAcceptanceTestService(t)
			var source Sourcing1688Product
			if err := db.First(&source, sourceID).Error; err != nil {
				t.Fatal(err)
			}
			if source.ProductID == nil {
				t.Fatal("missing product")
			}
			if err := tc.mutate(db, *source.ProductID); err != nil {
				t.Fatal(err)
			}
			report, err := svc.BuildAcceptanceReport(context.Background(), sourceID, 42)
			if err != nil {
				t.Fatal(err)
			}
			if report.Ready || report.Items[11].Status != AcceptanceBlocked {
				t.Fatalf("tampered %s must invalidate approval: ready=%v lifecycle=%s", tc.name, report.Ready, report.Items[11].Status)
			}
		})
	}
}

func TestAcceptanceReportIsOwnerScoped(t *testing.T) {
	svc, _, sourceID := newAcceptanceTestService(t)
	if _, err := svc.BuildAcceptanceReport(context.Background(), sourceID, 99); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("other owner error = %v", err)
	}
}

func TestAcceptanceReportHTTPIsReadOnlyAndOwnerScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _, sourceID := newAcceptanceTestService(t)
	h := NewHandler(svc)

	router := gin.New()
	router.GET("/:id/acceptance-report", func(c *gin.Context) {
		c.Set("user_id", int64(42))
		h.AcceptanceReport(c)
	})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%d/acceptance-report", sourceID), nil)
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"ready":true`) {
		t.Fatalf("Owner report code=%d body=%s", recorder.Code, recorder.Body.String())
	}

	otherRouter := gin.New()
	otherRouter.GET("/:id/acceptance-report", func(c *gin.Context) {
		c.Set("user_id", int64(99))
		h.AcceptanceReport(c)
	})
	recorder = httptest.NewRecorder()
	otherRouter.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%d/acceptance-report", sourceID), nil))
	if recorder.Code != http.StatusConflict {
		t.Fatalf("other Owner code=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAcceptanceReportRealDriverUsesExplicitAllowlist(t *testing.T) {
	for _, allowed := range []string{"plugin"} {
		if !realDriver(allowed) {
			t.Fatalf("expected allowed driver %q", allowed)
		}
	}
	for _, denied := range []string{"manual", "owner-browser", "custom", "mock-plugin", "pluginish", "plugin@1.2.3", "playwright-v2", "api1688_prod"} {
		if realDriver(denied) {
			t.Fatalf("expected denied driver %q", denied)
		}
	}
}

func TestCaptureModeCannotBeSpoofedThroughJSON(t *testing.T) {
	var input CaptureInput
	if err := json.Unmarshal([]byte(`{"capture_mode":"controlled_fetch"}`), &input); err != nil {
		t.Fatal(err)
	}
	if input.CaptureMode != "" {
		t.Fatalf("request JSON set trusted capture mode: %q", input.CaptureMode)
	}
}

func sha256ForTest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
