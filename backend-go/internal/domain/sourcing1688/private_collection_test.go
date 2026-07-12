package sourcing1688

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func privateRaw(t *testing.T, offerID, title string, extra map[string]any) json.RawMessage {
	t.Helper()
	value := map[string]any{"schema_version": "sourcing1688.private.v1", "offer_id_url": offerID, "offer_id_page": offerID,
		"source_url": "https://detail.1688.com/offer/" + offerID + ".html", "title": title, "price_model": "unknown",
		"field_statuses": map[string]string{"title": "observed", "price": "unknown", "moq": "unknown", "supplier": "unknown", "images": "unknown", "sku": "no_sku"}}
	for key, item := range extra {
		value[key] = item
	}
	if _, hasPrice := value["price_1688"]; hasPrice {
		if _, explicitModel := extra["price_model"]; !explicitModel {
			value["price_model"] = "fixed"
		}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestPrivateCollectionHTTPReturnsSavedRecord(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &PrivateCollectionRequest{}, &PrivateCaptureFailure{}, &demandCaseRow{})
	handler := NewHandler(NewService(db, dbtest.NewLogger(t)))
	body := []byte(`{
		"schema_version":"sourcing1688.private.v1",
		"page_offer_id":"456",
		"price_model":"unknown",
		"request_id":"collect_http_001",
		"source_url":"https://detail.1688.com/offer/456.html",
		"observed_at":"2026-07-12T10:00:00Z",
		"parser_version":"1688-detail-v1",
		"extension_version":"0.2.0",
		"raw_payload":{"schema_version":"sourcing1688.private.v1","offer_id_url":"456","offer_id_page":"456","source_url":"https://detail.1688.com/offer/456.html","title":"HTTP采集商品","price_model":"unknown","field_statuses":{"title":"observed","price":"unknown","moq":"unknown","supplier":"unknown","images":"unknown","sku":"no_sku"}},
		"title":"HTTP采集商品",
		"field_statuses":{"title":"observed","price":"unknown","moq":"unknown","supplier":"unknown","images":"unknown","sku":"no_sku"}
	}`)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sourcing-1688/private-collections", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", int64(42))
	handler.CollectPrivate(c)

	if w.Code != http.StatusOK {
		t.Fatalf("CollectPrivate HTTP status=%d body=%s", w.Code, w.Body.String())
	}
	var response struct {
		Code int `json:"code"`
		Data struct {
			Status    string `json:"status"`
			RecordID  int64  `json:"record_id"`
			RequestID string `json:"request_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != 0 || response.Data.Status != "saved" || response.Data.RecordID <= 0 || response.Data.RequestID != "collect_http_001" {
		t.Fatalf("CollectPrivate HTTP response=%s", w.Body.String())
	}
	otherOwnerBody := bytes.ReplaceAll(body, []byte("collect_http_001"), []byte("collect_http_other_owner"))
	otherOwnerBody = bytes.ReplaceAll(otherOwnerBody, []byte("HTTP采集商品"), []byte("其他Owner私有标题"))
	other := httptest.NewRecorder()
	otherContext, _ := gin.CreateTestContext(other)
	otherContext.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sourcing-1688/private-collections", bytes.NewReader(otherOwnerBody))
	otherContext.Request.Header.Set("Content-Type", "application/json")
	otherContext.Set("user_id", int64(43))
	handler.CollectPrivate(otherContext)
	if other.Code != http.StatusOK {
		t.Fatalf("other Owner setup status=%d body=%s", other.Code, other.Body.String())
	}

	duplicateBody := bytes.ReplaceAll(body, []byte("collect_http_001"), []byte("collect_http_002"))
	callDuplicate := func() *httptest.ResponseRecorder {
		out := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(out)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sourcing-1688/private-collections", bytes.NewReader(duplicateBody))
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Set("user_id", int64(42))
		handler.CollectPrivate(ctx)
		return out
	}
	for attempt := 0; attempt < 2; attempt++ { // same request_id replay returns the same safe choice contract
		conflict := callDuplicate()
		if conflict.Code != http.StatusConflict {
			t.Fatalf("duplicate attempt %d status=%d body=%s", attempt, conflict.Code, conflict.Body.String())
		}
		var envelope map[string]any
		if err := json.Unmarshal(conflict.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		data, _ := envelope["data"].(map[string]any)
		existing, _ := data["existing"].(map[string]any)
		if existing["title"] != "HTTP采集商品" {
			t.Fatalf("duplicate summary crossed Owner boundary: %s", conflict.Body.String())
		}
		for _, key := range []string{"title", "price", "moq", "supplier_name", "sku_count", "image_count", "observed_at"} {
			if _, ok := existing[key]; !ok {
				t.Fatalf("duplicate safe summary missing %q: %s", key, conflict.Body.String())
			}
		}
		for _, forbidden := range []string{"raw_payload", "raw_html", "credentials", "request_envelope_sha256"} {
			if strings.Contains(conflict.Body.String(), forbidden) {
				t.Fatalf("duplicate response leaked %q: %s", forbidden, conflict.Body.String())
			}
		}
	}
}

func TestPrivateCollectionSavesUnverifiedLeadWithoutExperiment(t *testing.T) {
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &PrivateCollectionRequest{}, &PrivateCaptureFailure{}, &demandCaseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	now := time.Now().UTC().Truncate(time.Second)
	title := "1688页面原始标题"
	price := 12.5
	moq := 2

	result, err := svc.CollectPrivate(&PrivateCollectInput{
		OwnerID:       42,
		SchemaVersion: "sourcing1688.private.v1", PageOfferID: "123456", PriceModel: "fixed", FieldStatuses: json.RawMessage(`{"title":"observed","price":"observed","moq":"observed","supplier":"observed","images":"unknown","sku":"no_sku"}`),
		RequestID:        "collect_owner_42_001",
		SourceURL:        "https://detail.1688.com/offer/123456.html?spm=tracking",
		ObservedAt:       now,
		ParserVersion:    "1688-detail-v1",
		ExtensionVersion: "0.2.0",
		RawPayload:       privateRaw(t, "123456", title, map[string]any{"price_1688": price, "min_order_qty": moq, "supplier_name": "测试供应商", "field_statuses": map[string]string{"title": "observed", "price": "observed", "moq": "observed", "supplier": "observed", "images": "unknown", "sku": "no_sku"}}),
		Title:            &title,
		Price:            &price,
		MOQ:              &moq,
		SupplierName:     "测试供应商",
	})
	if err != nil {
		t.Fatalf("CollectPrivate() error = %v", err)
	}
	if result.Product.OwnerID != 42 || result.Product.DemandCaseID != nil || result.Product.ExperimentID != nil {
		t.Fatalf("private product ownership/workflow = %#v", result.Product)
	}
	if result.Product.Status != StatusUnverifiedLead || result.Product.LifecycleStatus != LifecycleUnverifiedLead {
		t.Fatalf("private product status = %q/%q", result.Product.Status, result.Product.LifecycleStatus)
	}
	if result.Product.SourceOfferID != "123456" || result.Product.SourceURL != "https://detail.1688.com/offer/123456.html" {
		t.Fatalf("private product identity = %#v", result.Product)
	}
	if result.Snapshot.CollectionRequestID != "collect_owner_42_001" || result.Snapshot.CollectedBy != 42 {
		t.Fatalf("private snapshot = %#v", result.Snapshot)
	}

	items, total, err := svc.ListOwned(42, &common.Pagination{Page: 1, Size: 20}, &ListFilter{})
	if err != nil || total != 1 || len(items) != 1 || items[0].ID != result.Product.ID {
		t.Fatalf("ListOwned() = %#v total=%d err=%v", items, total, err)
	}
	otherItems, otherTotal, err := svc.ListOwned(99, &common.Pagination{Page: 1, Size: 20}, &ListFilter{})
	if err != nil || otherTotal != 0 || len(otherItems) != 0 {
		t.Fatalf("other owner ListOwned() = %#v total=%d err=%v", otherItems, otherTotal, err)
	}
}

func TestPrivateCollectionRejectsFullDOMOrCredentialLikeRawFields(t *testing.T) {
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &PrivateCollectionRequest{}, &PrivateCaptureFailure{})
	svc := NewService(db, dbtest.NewLogger(t))
	now := time.Now().UTC()
	title := "安全字段测试"
	for _, forbidden := range []string{"raw_html", "cookie_value", "authorization_header", "account_secret"} {
		raw := map[string]any{
			"schema_version": "sourcing1688.private.v1", "offer_id_url": "9911", "offer_id_page": "9911",
			"source_url": "https://detail.1688.com/offer/9911.html", "title": title, "price_model": "unknown",
			"field_statuses": map[string]string{"title": "observed", "price": "unknown", "moq": "unknown", "supplier": "unknown", "images": "unknown", "sku": "no_sku"},
			forbidden:        "must-not-be-stored",
		}
		encoded, _ := json.Marshal(raw)
		_, err := svc.CollectPrivate(&PrivateCollectInput{OwnerID: 42, SchemaVersion: "sourcing1688.private.v1", PageOfferID: "9911", PriceModel: "unknown", RequestID: "collect_forbidden_" + forbidden, SourceURL: "https://detail.1688.com/offer/9911.html", ObservedAt: now, ParserVersion: "v1", ExtensionVersion: "0.2.0", RawPayload: encoded, Title: &title, FieldStatuses: json.RawMessage(`{"title":"observed","price":"unknown","moq":"unknown","supplier":"unknown","images":"unknown","sku":"no_sku"}`)})
		if !errors.Is(err, ErrInvalidWorkflow) {
			t.Fatalf("forbidden raw field %q err=%v", forbidden, err)
		}
	}
}

func TestPrivateCollectionRequestStatusIsDurableAndOwnerIsolated(t *testing.T) {
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &PrivateCollectionRequest{}, &PrivateCaptureFailure{}, &demandCaseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	title := "可对账商品"
	result, err := svc.CollectPrivate(&PrivateCollectInput{
		OwnerID: 42, SchemaVersion: "sourcing1688.private.v1", PageOfferID: "654", PriceModel: "unknown",
		RequestID: "collect_status_saved", SourceURL: "https://detail.1688.com/offer/654.html",
		ObservedAt: time.Now().UTC(), ParserVersion: "v1", ExtensionVersion: "0.2.0", Title: &title,
		RawPayload:    privateRaw(t, "654", title, nil),
		FieldStatuses: json.RawMessage(`{"title":"observed","price":"unknown","moq":"unknown","supplier":"unknown","images":"unknown","sku":"no_sku"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	status, err := svc.GetPrivateCollectionRequest(42, "collect_status_saved")
	if err != nil || status.Status != PrivateRequestSaved || status.RecordID != result.Product.ID || status.SnapshotID != result.Snapshot.ID || !status.IdempotentReplay {
		t.Fatalf("saved status=%#v err=%v", status, err)
	}
	if _, err := svc.GetPrivateCollectionRequest(99, "collect_status_saved"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("other Owner should not see request: %v", err)
	}
	if _, _, err := svc.RecordPrivateCaptureFailure(&PrivateCaptureFailureInput{
		OwnerID: 42, RequestID: "collect_status_saved", SourceURL: "https://detail.1688.com/offer/654.html",
		ErrorCode: PrivateFailureNetworkError, SchemaVersion: "sourcing1688.private.v1", ExtensionVersion: "0.2.0",
		ParserVersion: "v1", OccurredAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	stillSaved, err := svc.GetPrivateCollectionRequest(42, "collect_status_saved")
	if err != nil || stillSaved.Status != PrivateRequestSaved {
		t.Fatalf("late failure downgraded saved request: %#v err=%v", stillSaved, err)
	}

	for _, receipt := range []PrivateCollectionRequest{
		{OwnerID: 42, RequestID: "collect_status_receiving", Status: PrivateRequestReceiving},
		{OwnerID: 42, RequestID: "collect_status_reconcile", Status: PrivateRequestReconcileRequired, FailureCode: "persistence_outcome_unknown", SafeMessage: "服务器未能确认采集是否保存，请勿重复采集并继续对账"},
	} {
		if err := db.Create(&receipt).Error; err != nil {
			t.Fatal(err)
		}
	}
	receiving, err := svc.GetPrivateCollectionRequest(42, "collect_status_receiving")
	if err != nil || receiving.Status != PrivateRequestReceiving || receiving.RecordID != 0 || receiving.SnapshotID != 0 {
		t.Fatalf("receiving=%#v err=%v", receiving, err)
	}
	reconcile, err := svc.GetPrivateCollectionRequest(42, "collect_status_reconcile")
	if err != nil || reconcile.Status != PrivateRequestReconcileRequired || reconcile.FailureCode != "persistence_outcome_unknown" || reconcile.SafeMessage == "" {
		t.Fatalf("reconcile=%#v err=%v", reconcile, err)
	}
}

func TestPrivateCollectionFailureCreatesNotSavedReceiptWithoutPageLeak(t *testing.T) {
	db := dbtest.NewDB(t, &PrivateCaptureFailure{}, &PrivateCollectionRequest{})
	svc := NewService(db, dbtest.NewLogger(t))
	_, _, err := svc.RecordPrivateCaptureFailure(&PrivateCaptureFailureInput{
		OwnerID: 42, RequestID: "collect_status_failed", SourceURL: "https://detail.1688.com/offer/987.html?token=secret",
		ErrorCode: PrivateFailureInvalidPayload, SchemaVersion: "sourcing1688.private.v1", ExtensionVersion: "0.2.0",
		ParserVersion: "v1", OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	status, err := svc.GetPrivateCollectionRequest(42, "collect_status_failed")
	if err != nil || status.Status != PrivateRequestNotSaved || status.FailureCode != PrivateFailureInvalidPayload || status.SafeMessage == "" {
		t.Fatalf("not_saved=%#v err=%v", status, err)
	}
	encoded, _ := json.Marshal(status)
	if bytes.Contains(encoded, []byte("token=secret")) || bytes.Contains(encoded, []byte("raw_payload")) {
		t.Fatalf("request status leaked page data: %s", encoded)
	}
}

func TestPrivateCollectionRetryIsIdempotentAndRecaptureKeepsOneProduct(t *testing.T) {
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &PrivateCollectionRequest{}, &PrivateCaptureFailure{}, &demandCaseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	title := "同一个商品"
	price10 := 10.0
	base := PrivateCollectInput{
		SchemaVersion: "sourcing1688.private.v1", PageOfferID: "888", PriceModel: "fixed", FieldStatuses: json.RawMessage(`{"title":"observed","price":"observed","moq":"unknown","supplier":"unknown","images":"unknown","sku":"no_sku"}`),
		OwnerID: 42, RequestID: "collect_retry_001",
		SourceURL: "https://detail.1688.com/offer/888.html", ObservedAt: time.Now().UTC(),
		ParserVersion: "v1", ExtensionVersion: "0.2.0", Title: &title, Price: &price10,
		RawPayload: privateRaw(t, "888", title, map[string]any{"price_1688": 10, "field_statuses": map[string]string{"title": "observed", "price": "observed", "moq": "unknown", "supplier": "unknown", "images": "unknown", "sku": "no_sku"}}),
	}
	first, err := svc.CollectPrivate(&base)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := svc.CollectPrivate(&base)
	if err != nil || retry.Product.ID != first.Product.ID || retry.Snapshot.ID != first.Snapshot.ID || !retry.IdempotentReplay {
		t.Fatalf("idempotent retry = %#v err=%v; first=%#v", retry, err, first)
	}
	tamperedReplay := base
	tamperedReplay.ObservedAt = base.ObservedAt.Add(time.Second)
	if _, err := svc.CollectPrivate(&tamperedReplay); !errors.Is(err, ErrInvalidWorkflow) {
		t.Fatalf("same request with changed envelope err=%v", err)
	}

	changed := base
	changed.RequestID = "collect_retry_002"
	changed.ObservedAt = base.ObservedAt.Add(time.Minute)
	price9 := 9.0
	changed.Price = &price9
	changed.RawPayload = privateRaw(t, "888", title, map[string]any{"price_1688": 9, "field_statuses": map[string]string{"title": "observed", "price": "observed", "moq": "unknown", "supplier": "unknown", "images": "unknown", "sku": "no_sku"}})
	if _, err := svc.CollectPrivate(&changed); err == nil {
		t.Fatal("duplicate offer was silently saved without Owner observation intent")
	} else {
		var duplicate *DuplicatePrivateCollectionError
		if !errors.As(err, &duplicate) || duplicate.RecordID != first.Product.ID || duplicate.SnapshotID != first.Snapshot.ID {
			t.Fatalf("duplicate choice error = %#v err=%v", duplicate, err)
		}
		if duplicate.Existing.Title == nil || *duplicate.Existing.Title != title || duplicate.Existing.Price == nil || *duplicate.Existing.Price != price10 || !duplicate.Existing.ObservedAt.Equal(base.ObservedAt) {
			t.Fatalf("duplicate safe summary = %#v", duplicate.Existing)
		}
	}
	if _, err := svc.CollectPrivate(&changed); err == nil {
		t.Fatal("same duplicate request replay was not idempotently rejected")
	} else {
		var duplicate *DuplicatePrivateCollectionError
		if !errors.As(err, &duplicate) || duplicate.RecordID != first.Product.ID || duplicate.SnapshotID != first.Snapshot.ID {
			t.Fatalf("duplicate replay = %#v err=%v", duplicate, err)
		}
	}
	requestState, stateErr := svc.GetPrivateCollectionRequest(42, changed.RequestID)
	if stateErr != nil || requestState.Status != PrivateRequestNotSaved || requestState.FailureCode != "duplicate_requires_choice" {
		t.Fatalf("duplicate request must be durably confirmed not_saved, got state=%#v err=%v", requestState, stateErr)
	}
	changed.RequestID = "collect_retry_003"
	changed.ObservationIntent = ObservationIntentNew
	second, err := svc.CollectPrivate(&changed)
	if err != nil || second.Product.ID != first.Product.ID || second.Snapshot.ID == first.Snapshot.ID || !second.NewObservation {
		t.Fatalf("changed recapture = %#v err=%v; first=%#v", second, err, first)
	}

	var productCount, snapshotCount int64
	db.Model(&Sourcing1688Product{}).Count(&productCount)
	db.Model(&Sourcing1688Snapshot{}).Count(&snapshotCount)
	if productCount != 1 || snapshotCount != 2 {
		t.Fatalf("counts product=%d snapshot=%d", productCount, snapshotCount)
	}
}

func TestOwnerEditsPrivateWorkcopyWithoutMutatingSnapshot(t *testing.T) {
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &PrivateCollectionRequest{}, &PrivateCaptureFailure{}, &demandCaseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	rawTitle := "页面原始标题"
	collected, err := svc.CollectPrivate(&PrivateCollectInput{OwnerID: 42, RequestID: "collect_workcopy_1", SourceURL: "https://detail.1688.com/offer/777.html",
		SchemaVersion: "sourcing1688.private.v1", PageOfferID: "777", PriceModel: "unknown", FieldStatuses: json.RawMessage(`{"title":"observed","price":"unknown","moq":"unknown","supplier":"unknown","images":"unknown","sku":"no_sku"}`),
		ObservedAt: time.Now().UTC(), ParserVersion: "v1", ExtensionVersion: "0.2.0", RawPayload: privateRaw(t, "777", rawTitle, nil), Title: &rawTitle})
	if err != nil {
		t.Fatal(err)
	}
	price := 8.8
	moq := 3
	supplier := "Owner整理的供应商名称"
	updated, err := svc.UpdatePrivateWorkcopy(collected.Product.ID, &PrivateWorkcopyInput{OwnerID: 42, ExpectedUpdatedAt: collected.Product.UpdatedAt,
		Title: "Owner整理后的标题", Price: &price, MOQ: &moq, SupplierName: &supplier, Notes: "待确认材质"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title == nil || *updated.Title != "Owner整理后的标题" || updated.ReviewNotes != "待确认材质" || updated.LifecycleStatus != LifecycleNeedsReview {
		t.Fatalf("updated=%#v", updated)
	}
	var snapshot Sourcing1688Snapshot
	if err := db.First(&snapshot, collected.Snapshot.ID).Error; err != nil {
		t.Fatal(err)
	}
	if snapshot.ObservedTitle == nil || *snapshot.ObservedTitle != rawTitle || !bytes.Contains(snapshot.RawPayload, []byte(`"title":"页面原始标题"`)) {
		t.Fatalf("snapshot mutated=%#v", snapshot)
	}
	if _, err := svc.UpdatePrivateWorkcopy(collected.Product.ID, &PrivateWorkcopyInput{OwnerID: 42, ExpectedUpdatedAt: collected.Product.UpdatedAt, Title: "过期覆盖"}); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("stale edit err=%v", err)
	}
	if _, err := svc.UpdatePrivateWorkcopy(collected.Product.ID, &PrivateWorkcopyInput{OwnerID: 99, ExpectedUpdatedAt: updated.UpdatedAt, Title: "越权"}); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("other owner edit err=%v", err)
	}
}

func TestOwnerArchivesAndRestoresOnlyUnlinkedPrivateBookmark(t *testing.T) {
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &Sourcing1688TaskLink{}, &draftRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	caseID, experimentID := int64(7), "EXP-LINKED"
	private := Sourcing1688Product{OwnerID: 42, SourceURL: "https://detail.1688.com/offer/7001.html", SourceOfferID: "7001", Status: StatusUnverifiedLead, LifecycleStatus: LifecycleUnverifiedLead}
	linked := Sourcing1688Product{OwnerID: 42, SourceURL: "https://detail.1688.com/offer/7002.html", SourceOfferID: "7002", Status: StatusPendingReview, LifecycleStatus: LifecyclePendingReview, DemandCaseID: &caseID, ExperimentID: &experimentID}
	if err := db.Create(&private).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&linked).Error; err != nil {
		t.Fatal(err)
	}

	archived, err := svc.SetPrivateArchive(private.ID, 42, true)
	if err != nil || archived.LifecycleStatus != LifecycleArchived || archived.Status != LifecycleArchived {
		t.Fatalf("archive=%#v err=%v", archived, err)
	}
	if _, err := svc.SetPrivateArchive(private.ID, 99, false); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("other Owner restore err=%v", err)
	}
	restored, err := svc.SetPrivateArchive(private.ID, 42, false)
	if err != nil || restored.LifecycleStatus != LifecycleNeedsReview {
		t.Fatalf("restore=%#v err=%v", restored, err)
	}
	if _, err := svc.SetPrivateArchive(linked.ID, 42, true); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("linked source archive err=%v", err)
	}
}

func TestOwnerCanLinkPrivateCollectionToApprovedTaskLater(t *testing.T) {
	db := dbtest.NewDB(t,
		&Sourcing1688Product{}, &Sourcing1688Snapshot{}, &PrivateCollectionRequest{}, &PrivateCaptureFailure{}, &Sourcing1688TaskLink{},
		&demandCaseRow{}, &experimentRow{}, &gateRow{}, &objectLinkRow{},
		&sourcingMarketDecisionRow{}, &sourcingOpportunityRow{}, &sourcingOpportunityDecisionRow{},
	)
	svc := NewService(db, dbtest.NewLogger(t))
	title := "先收藏后关联"
	collected, err := svc.CollectPrivate(&PrivateCollectInput{
		SchemaVersion: "sourcing1688.private.v1", PageOfferID: "999", PriceModel: "unknown", FieldStatuses: json.RawMessage(`{"title":"observed","price":"unknown","moq":"unknown","supplier":"unknown","images":"unknown","sku":"no_sku"}`),
		OwnerID: 42, RequestID: "collect_link_001", SourceURL: "https://detail.1688.com/offer/999.html",
		ObservedAt: time.Now().UTC(), ParserVersion: "v1", ExtensionVersion: "0.2.0",
		RawPayload: privateRaw(t, "999", title, nil), Title: &title,
	})
	if err != nil {
		t.Fatal(err)
	}
	db.Create(&demandCaseRow{ID: 7, OwnerID: 42, TargetLocale: "en-US", SalesChannel: "marketplace", Status: "experiment_ready"})
	db.Create(&sourcingMarketDecisionRow{ID: 70, DemandCaseID: 7, OwnerID: 42, Decision: "selected"})
	db.Create(&sourcingOpportunityRow{ID: 71, OwnerID: 42, DemandCaseID: 7, MarketDecisionID: 70, Version: 1, TargetChannel: "marketplace", Status: "approved", ContentHash: strings.Repeat("a", 64)})
	db.Create(&sourcingOpportunityDecisionRow{ID: 72, OpportunityID: 71, OwnerID: 42, Version: 1, Decision: "approved", ContentHash: strings.Repeat("a", 64)})
	db.Create(&experimentRow{ExperimentID: "EXP-LINK-1", OwnerID: 42, Status: "active", Stage: "product"})
	db.Create(&gateRow{ExperimentID: "EXP-LINK-1", Stage: "opportunity", Result: "pass"})
	db.Create(&objectLinkRow{ExperimentID: "EXP-LINK-1", ObjectType: "demand_case", ObjectID: "7"})

	linked, err := svc.LinkPrivateToTask(collected.Product.ID, &LinkPrivateTaskInput{
		OwnerID: 42, DemandCaseID: 7, ExperimentID: "EXP-LINK-1", ProductOpportunityID: 71,
	})
	if err != nil {
		t.Fatalf("LinkPrivateToTask() error = %v", err)
	}
	if linked.Product.DemandCaseID == nil || *linked.Product.DemandCaseID != 7 ||
		linked.Product.ExperimentID == nil || *linked.Product.ExperimentID != "EXP-LINK-1" ||
		linked.Product.Status != StatusPendingReview || linked.Product.LifecycleStatus != LifecyclePendingReview {
		t.Fatalf("linked product = %#v", linked.Product)
	}
	if linked.Link.SourcingProductID != collected.Product.ID || linked.Link.ExperimentID != "EXP-LINK-1" {
		t.Fatalf("task link = %#v", linked.Link)
	}
	if !linked.Link.IsPrimary || linked.Link.Status != "active_workflow" {
		t.Fatalf("primary task link = %#v", linked.Link)
	}
	db.Create(&demandCaseRow{ID: 8, OwnerID: 42, TargetLocale: "de-DE", SalesChannel: "marketplace", Status: "experiment_ready"})
	db.Create(&sourcingMarketDecisionRow{ID: 80, DemandCaseID: 8, OwnerID: 42, Decision: "selected"})
	db.Create(&sourcingOpportunityRow{ID: 81, OwnerID: 42, DemandCaseID: 8, MarketDecisionID: 80, Version: 1, TargetChannel: "marketplace", Status: "approved", ContentHash: strings.Repeat("b", 64)})
	db.Create(&sourcingOpportunityDecisionRow{ID: 82, OpportunityID: 81, OwnerID: 42, Version: 1, Decision: "approved", ContentHash: strings.Repeat("b", 64)})
	db.Create(&experimentRow{ExperimentID: "EXP-LINK-2", OwnerID: 42, Status: "active", Stage: "supply"})
	db.Create(&gateRow{ExperimentID: "EXP-LINK-2", Stage: "opportunity", Result: "pass"})
	db.Create(&objectLinkRow{ExperimentID: "EXP-LINK-2", ObjectType: "demand_case", ObjectID: "8"})
	second, err := svc.LinkPrivateToTask(collected.Product.ID, &LinkPrivateTaskInput{OwnerID: 42, DemandCaseID: 8, ExperimentID: "EXP-LINK-2", ProductOpportunityID: 81})
	if err != nil {
		t.Fatal(err)
	}
	if second.Link.IsPrimary || second.Link.Status != "linked" {
		t.Fatalf("secondary task link = %#v", second.Link)
	}
	if second.Product.ExperimentID == nil || *second.Product.ExperimentID != "EXP-LINK-1" {
		t.Fatalf("second link overwrote primary workflow: %#v", second.Product)
	}
	links, err := svc.ListPrivateTaskLinks(collected.Product.ID, 42)
	if err != nil || len(links) != 2 || !links[0].IsPrimary || links[1].ExperimentID != "EXP-LINK-2" {
		t.Fatalf("task links = %#v err=%v", links, err)
	}
	if _, err := svc.ListPrivateTaskLinks(collected.Product.ID, 99); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("other owner list links error=%v", err)
	}
	if _, err := svc.LinkPrivateToTask(collected.Product.ID, &LinkPrivateTaskInput{OwnerID: 99, DemandCaseID: 7, ExperimentID: "EXP-LINK-1", ProductOpportunityID: 71}); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("other owner link error = %v", err)
	}
	tasks, err := svc.ListEligibleTasks(42)
	if err != nil || len(tasks) != 2 || tasks[0].Label == "" {
		t.Fatalf("ListEligibleTasks() = %#v err=%v", tasks, err)
	}
	db.Create(&sourcingMarketDecisionRow{ID: 83, DemandCaseID: 8, OwnerID: 42, Decision: "paused"})
	links, err = svc.ListPrivateTaskLinks(collected.Product.ID, 42)
	if err != nil || links[1].CurrentStatus != "blocked" || links[1].CurrentBlocker == "" {
		t.Fatalf("revoked opportunity link must be visibly blocked, links=%#v err=%v", links, err)
	}
}

func TestLegacyExperimentPassCannotAuthorizeSourcingWithoutApprovedOpportunity(t *testing.T) {
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688TaskLink{}, &demandCaseRow{}, &experimentRow{}, &gateRow{}, &objectLinkRow{}, &sourcingMarketDecisionRow{}, &sourcingOpportunityRow{}, &sourcingOpportunityDecisionRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	title := "legacy trace only"
	product := Sourcing1688Product{OwnerID: 42, SourceURL: "https://detail.1688.com/offer/555.html", SourceOfferID: "555", Title: &title, Status: StatusUnverifiedLead, LifecycleStatus: LifecycleUnverifiedLead}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	db.Create(&demandCaseRow{ID: 55, OwnerID: 42, SalesChannel: "marketplace", Status: "experiment_ready"})
	db.Create(&experimentRow{ExperimentID: "EXP-LEGACY", OwnerID: 42, Status: "active", Stage: "product"})
	db.Create(&gateRow{ExperimentID: "EXP-LEGACY", Stage: "opportunity", Result: "pass"})
	db.Create(&objectLinkRow{ExperimentID: "EXP-LEGACY", ObjectType: "demand_case", ObjectID: "55"})
	_, err := svc.LinkPrivateToTask(product.ID, &LinkPrivateTaskInput{OwnerID: 42, DemandCaseID: 55, ExperimentID: "EXP-LEGACY", ProductOpportunityID: 999})
	if !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("legacy experiment unexpectedly authorized sourcing: %v", err)
	}
	legacy := Sourcing1688TaskLink{SourcingProductID: product.ID, DemandCaseID: 55, ExperimentID: "EXP-LEGACY", OwnerID: 42, AuthorityKind: "legacy_experiment", Status: "linked", IsPrimary: true}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	links, err := svc.ListPrivateTaskLinks(product.ID, 42)
	if err != nil || len(links) != 1 || links[0].CurrentStatus != "blocked" || links[0].CurrentBlocker == "" {
		t.Fatalf("legacy link must remain trace-only and visibly blocked, links=%#v err=%v", links, err)
	}
}

func TestPrivateRecaptureDoesNotReplaceGovernedWorkflowSnapshot(t *testing.T) {
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &PrivateCollectionRequest{}, &PrivateCaptureFailure{}, &demandCaseRow{})
	caseID := int64(7)
	experimentID := "EXP-GOVERNED"
	oldTitle := "已进入受控流程的标题"
	product := Sourcing1688Product{
		OwnerID: 42, SourceURL: "https://detail.1688.com/offer/321.html", SourceOfferID: "321",
		Title: &oldTitle, DemandCaseID: &caseID, ExperimentID: &experimentID,
		Status: StatusPendingReview, LifecycleStatus: LifecyclePendingReview,
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	governed := Sourcing1688Snapshot{
		SourcingProductID: product.ID, SourceURL: product.SourceURL, CollectedAt: time.Now().UTC().Add(-time.Hour),
		CollectedBy: 42, Driver: "plugin", ParserVersion: "controlled-v1", CaptureMode: CaptureModeControlledFetch,
		CollectionRequestID: "req_governed", RawPayload: json.RawMessage(`{"governed":true}`), RawSHA256: "governed-hash",
	}
	if err := db.Create(&governed).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&product).Update("snapshot_id", governed.ID).Error; err != nil {
		t.Fatal(err)
	}

	newTitle := "私人再次浏览时看到的新标题"
	result, err := NewService(db, dbtest.NewLogger(t)).CollectPrivate(&PrivateCollectInput{
		SchemaVersion: "sourcing1688.private.v1", PageOfferID: "321", PriceModel: "unknown", ObservationIntent: ObservationIntentNew, FieldStatuses: json.RawMessage(`{"title":"observed","price":"unknown","moq":"unknown","supplier":"unknown","images":"unknown","sku":"no_sku"}`),
		OwnerID: 42, RequestID: "collect_governed_recapture", SourceURL: product.SourceURL,
		ObservedAt: time.Now().UTC(), ParserVersion: "page-v2", ExtensionVersion: "0.2.0",
		RawPayload: privateRaw(t, "321", newTitle, map[string]any{"governed": false, "changed": true}), Title: &newTitle,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Product.SnapshotID == nil || *result.Product.SnapshotID != governed.ID || result.Product.Title == nil || *result.Product.Title != oldTitle || result.Product.Status != StatusPendingReview {
		t.Fatalf("governed product was mutated by private recapture: %#v", result.Product)
	}
	if result.Snapshot.ID == governed.ID || result.Snapshot.CaptureMode != CaptureModeExtensionClick {
		t.Fatalf("private observation not recorded separately: %#v", result.Snapshot)
	}
}
