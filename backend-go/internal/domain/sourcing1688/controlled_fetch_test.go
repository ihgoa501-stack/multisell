package sourcing1688

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
)

type controlledFetchStub struct {
	page    *toolbridge.PageData
	err     error
	called  int
	ownerID int64
}

func (s *controlledFetchStub) FetchPage(ctx context.Context, _ string) (*toolbridge.PageData, error) {
	s.called++
	s.ownerID, _ = toolbridge.OwnerUserIDFromContext(ctx)
	return s.page, s.err
}

func controlledFetchDB(t *testing.T) (*Service, *ControlledFetchHandler) {
	t.Helper()
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &Sourcing1688TaskLink{}, &SourcingChangeEvent{}, &DuplicateCandidate{}, &CaptureAttempt{}, &demandCaseRow{}, &experimentRow{}, &gateRow{}, &objectLinkRow{}, &sourcingMarketDecisionRow{}, &sourcingOpportunityRow{}, &sourcingOpportunityDecisionRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	return svc, nil
}

func seedControlledFetchGate(svc *Service) {
	svc.db.Create(&demandCaseRow{ID: 7, OwnerID: 42, SalesChannel: "approved-channel", TargetLocale: "en-US", Status: "experiment_ready"})
	title := "private lead"
	demandCaseID, experimentID := int64(7), "EXP-1"
	svc.db.Create(&Sourcing1688Product{ID: 60, OwnerID: 42, SourceURL: "https://detail.1688.com/offer/123.html", SourceOfferID: "123", Title: &title, DemandCaseID: &demandCaseID, ExperimentID: &experimentID, Status: StatusPendingReview, LifecycleStatus: LifecyclePendingReview})
	svc.db.Create(&sourcingMarketDecisionRow{ID: 61, DemandCaseID: 7, OwnerID: 42, Decision: "selected"})
	svc.db.Create(&sourcingOpportunityRow{ID: 62, OwnerID: 42, DemandCaseID: 7, MarketDecisionID: 61, Version: 1, Title: "approved opportunity", TargetChannel: "approved-channel", Status: "approved", ContentHash: strings.Repeat("a", 64)})
	svc.db.Create(&sourcingOpportunityDecisionRow{ID: 63, OpportunityID: 62, OwnerID: 42, Version: 1, Decision: "approved", ContentHash: strings.Repeat("a", 64)})
	svc.db.Create(&experimentRow{ExperimentID: "EXP-1", OwnerID: 42, Status: "active", Stage: "product"})
	svc.db.Create(&gateRow{ExperimentID: "EXP-1", Stage: "opportunity", Result: "pass"})
	svc.db.Create(&objectLinkRow{ExperimentID: "EXP-1", ObjectType: "demand_case", ObjectID: "7"})
	opportunityID, decisionID := int64(62), int64(63)
	svc.db.Create(&Sourcing1688TaskLink{SourcingProductID: 60, DemandCaseID: 7, ExperimentID: "EXP-1", OwnerID: 42, ProductOpportunityID: &opportunityID, OpportunityDecisionID: &decisionID, AuthorityKind: "product_opportunity", Status: "active_workflow", IsPrimary: true})
}

func performControlledFetch(t *testing.T, h *ControlledFetchHandler, ownerID int64) *httptest.ResponseRecorder {
	t.Helper()
	body := bytes.NewBufferString(`{"demand_case_id":7,"experiment_id":"EXP-1","source_url":"https://detail.1688.com/offer/123.html"}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sourcing-1688/fetch", body)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", ownerID)
	h.Fetch(c)
	return w
}

func TestControlledFetchChecksGateBeforeExternalCall(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _ := controlledFetchDB(t)
	stub := &controlledFetchStub{}
	w := performControlledFetch(t, NewControlledFetchHandler(svc, stub), 42)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if stub.called != 0 {
		t.Fatalf("external collector called before gate: %d", stub.called)
	}
}

func TestControlledFetchRejectsUnsafeURLBeforeExternalCall(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _ := controlledFetchDB(t)
	seedControlledFetchGate(svc)
	stub := &controlledFetchStub{}
	body := bytes.NewBufferString(`{"demand_case_id":7,"experiment_id":"EXP-1","source_url":"https://detail.1688.com:8443/offer/123.html"}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sourcing-1688/fetch", body)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", int64(42))
	NewControlledFetchHandler(svc, stub).Fetch(c)
	if w.Code != http.StatusBadRequest || stub.called != 0 {
		t.Fatalf("status=%d external_calls=%d body=%s", w.Code, stub.called, w.Body.String())
	}
}

func TestControlledFetchBindsOwnerAndReusesCapture(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _ := controlledFetchDB(t)
	seedControlledFetchGate(svc)
	rawResponse := json.RawMessage(`{ "source_url":"https://detail.1688.com/offer/123.html", "title":"actual structured title", "price_cny":12.5, "moq":2, "images":["https://cbu01.alicdn.com/a.jpg"], "supplier_name":"supplier", "supplier_business_id":"supplier-42", "raw_data":{"offer_id":"123"}, "raw_html":"<main>actual page</main>", "parser_version":"extension-1.2.3" }`)
	stub := &controlledFetchStub{page: &toolbridge.PageData{SourceURL: "https://detail.1688.com/offer/123.html", Title: "actual structured title", PriceCNY: 12.5, MOQ: 2, Images: []string{"https://cbu01.alicdn.com/a.jpg"}, SupplierName: "supplier", SupplierBusinessID: "supplier-42", CollectedAt: time.Now().UTC(), Driver: "plugin", ParserVersion: "extension-1.2.3", CollectionRequestID: "req_controlled-fetch-123", RawHTML: "<main>actual page</main>", RawData: json.RawMessage(`{"offer_id":"123"}`), RawResponse: rawResponse}}
	w := performControlledFetch(t, NewControlledFetchHandler(svc, stub), 42)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if stub.ownerID != 42 {
		t.Fatalf("collector owner=%d", stub.ownerID)
	}
	var product Sourcing1688Product
	if err := svc.db.First(&product).Error; err != nil {
		t.Fatal(err)
	}
	if product.SnapshotID == nil || product.Status != StatusPendingReview {
		t.Fatalf("product=%#v", product)
	}
	var snapshot Sourcing1688Snapshot
	if err := svc.db.First(&snapshot, *product.SnapshotID).Error; err != nil {
		t.Fatal(err)
	}
	if snapshot.Driver != "plugin" || snapshot.ParserVersion != "extension-1.2.3" || snapshot.CaptureMode != CaptureModeControlledFetch || snapshot.CollectionRequestID != "req_controlled-fetch-123" {
		t.Fatalf("snapshot=%#v", snapshot)
	}
	var captured toolbridge.PageData
	if err := json.Unmarshal(snapshot.RawPayload, &captured); err != nil || captured.Title != "actual structured title" {
		t.Fatalf("raw evidence=%s err=%v", snapshot.RawPayload, err)
	}
	if !bytes.Equal(snapshot.RawPayload, rawResponse) {
		t.Fatalf("exact extension data bytes changed: got=%q want=%q", snapshot.RawPayload, rawResponse)
	}
}

func TestControlledFetchFailureWritesAttempt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _ := controlledFetchDB(t)
	seedControlledFetchGate(svc)
	stub := &controlledFetchStub{err: errors.New("1688 login required")}
	w := performControlledFetch(t, NewControlledFetchHandler(svc, stub), 42)
	if w.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "完成登录") {
		t.Fatalf("response is not actionable: %s", w.Body.String())
	}
	var attempt CaptureAttempt
	if err := svc.db.First(&attempt).Error; err != nil {
		t.Fatal(err)
	}
	if attempt.ErrorCode != "login_required" || attempt.AttemptedBy != 42 {
		t.Fatalf("attempt=%#v", attempt)
	}
}

func TestControlledFetchRejectsMismatchedCollectorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _ := controlledFetchDB(t)
	seedControlledFetchGate(svc)
	stub := &controlledFetchStub{page: &toolbridge.PageData{SourceURL: "https://detail.1688.com/offer/999.html", Title: "wrong offer", PriceCNY: 12, MOQ: 1, SupplierName: "supplier", SupplierBusinessID: "supplier-42", CollectedAt: time.Now().UTC(), Driver: "owner-browser", ParserVersion: "1.0.0"}}
	w := performControlledFetch(t, NewControlledFetchHandler(svc, stub), 42)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var products int64
	svc.db.Model(&Sourcing1688Product{}).Count(&products)
	if products != 1 {
		t.Fatalf("mismatched response changed product count to %d", products)
	}
}

func TestControlledFetchUnavailableWritesAttempt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _ := controlledFetchDB(t)
	seedControlledFetchGate(svc)
	w := performControlledFetch(t, NewControlledFetchHandler(svc, nil), 42)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var attempt CaptureAttempt
	if err := svc.db.First(&attempt).Error; err != nil || attempt.ErrorCode != "collector_unavailable" {
		t.Fatalf("attempt=%#v err=%v", attempt, err)
	}
}
