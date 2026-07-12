package sourcing1688

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

type privateFailureOwnerAccount struct {
	ID     int64  `gorm:"column:id;primaryKey"`
	Role   string `gorm:"column:role"`
	Status int    `gorm:"column:status"`
}

func (privateFailureOwnerAccount) TableName() string { return "user" }

func TestPrivateCaptureFailureRecordsSafeParseFailuresWithoutExperiment(t *testing.T) {
	db := dbtest.NewDB(t, &PrivateCaptureFailure{}, &PrivateCollectionRequest{})
	svc := NewService(db, dbtest.NewLogger(t))
	now := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		requestID string
		code      string
		want      string
		sourceURL string
		wantURL   string
	}{
		{"collect_bad_url", PrivateFailureInvalidSourceURL, "商品链接无法识别，请确认当前页面是1688商品详情页", "https://detail.1688.com/device-token-secret", "invalid://redacted"},
		{"collect_no_title", PrivateFailureTitleParseFailed, "未能读取商品标题，请刷新页面后重试", "https://detail.1688.com/offer/123.html?token=secret", "https://detail.1688.com/offer/123.html"},
		{"collect_bad_sku", PrivateFailureSKUParseFailed, "未能完整读取SKU信息，可稍后重试或仅保存已读取字段", "https://detail.1688.com/offer/123.html?token=secret", "https://detail.1688.com/offer/123.html"},
	}
	for _, tt := range tests {
		got, replay, err := svc.RecordPrivateCaptureFailure(&PrivateCaptureFailureInput{
			OwnerID: 42, RequestID: tt.requestID, SourceURL: tt.sourceURL,
			ErrorCode: tt.code, SchemaVersion: "sourcing1688.private.v1", ExtensionVersion: "0.2.0",
			ParserVersion: "1688-detail-v1", OccurredAt: now,
		})
		if err != nil || replay {
			t.Fatalf("RecordPrivateCaptureFailure(%s) = %#v replay=%v err=%v", tt.code, got, replay, err)
		}
		if got.OwnerID != 42 || got.SafeMessage != tt.want || got.SourceURL != tt.wantURL {
			t.Fatalf("failure = %#v", got)
		}
	}

	items, err := svc.ListPrivateCaptureFailures(42)
	if err != nil || len(items) != 3 {
		t.Fatalf("ListPrivateCaptureFailures(owner) = %#v err=%v", items, err)
	}
	other, err := svc.ListPrivateCaptureFailures(99)
	if err != nil || len(other) != 0 {
		t.Fatalf("ListPrivateCaptureFailures(other) = %#v err=%v", other, err)
	}
	var terminalReceipts int64
	if err := db.Model(&PrivateCollectionRequest{}).Where("owner_id = ?", 42).Count(&terminalReceipts).Error; err != nil || terminalReceipts != 2 {
		t.Fatalf("only identity/title failures may create terminal receipts, count=%d err=%v", terminalReceipts, err)
	}
	var skuReceipt int64
	if err := db.Model(&PrivateCollectionRequest{}).Where("owner_id = ? AND request_id = ?", 42, "collect_bad_sku").Count(&skuReceipt).Error; err != nil || skuReceipt != 0 {
		t.Fatalf("sku parser warning must not block later collection, count=%d err=%v", skuReceipt, err)
	}
}

func TestPrivateCaptureFailureHTTPReportAndOwnerListAreIsolated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t, &PrivateCaptureFailure{}, &PrivateCollectionRequest{}, &privateFailureOwnerAccount{})
	if err := db.Create(&privateFailureOwnerAccount{ID: 42, Role: "owner", Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&privateFailureOwnerAccount{ID: 99, Role: "user", Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	h := NewHandler(NewService(db, dbtest.NewLogger(t)))
	body := []byte(`{"request_id":"collect_http_failure","source_url":"https://detail.1688.com/offer/123.html","error_code":"sku_parse_failed","schema_version":"sourcing1688.private.v1","extension_version":"0.2.0","parser_version":"v1","occurred_at":"2026-07-12T10:00:00Z"}`)

	postRecorder := httptest.NewRecorder()
	postContext, _ := gin.CreateTestContext(postRecorder)
	postContext.Request = httptest.NewRequest(http.MethodPost, "/api/v1/extension/sourcing-1688/private-collections/failures", bytes.NewReader(body))
	postContext.Request.Header.Set("Content-Type", "application/json")
	postContext.Set("user_id", int64(42))
	h.RecordPrivateCaptureFailure(postContext)
	if postRecorder.Code != http.StatusOK || !bytes.Contains(postRecorder.Body.Bytes(), []byte(`"status":"recorded"`)) {
		t.Fatalf("POST failure status=%d body=%s", postRecorder.Code, postRecorder.Body.String())
	}

	list := func(ownerID int64) (int, string) {
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/sourcing-1688/private-collections/failures", nil)
		context.Set("user_id", ownerID)
		h.ListPrivateCaptureFailures(context)
		return recorder.Code, recorder.Body.String()
	}
	if status, got := list(42); status != http.StatusOK || !bytes.Contains([]byte(got), []byte("collect_http_failure")) {
		t.Fatalf("owner list=%s", got)
	}
	if status, got := list(99); status != http.StatusForbidden || bytes.Contains([]byte(got), []byte("collect_http_failure")) {
		t.Fatalf("non-owner status=%d leaked list=%s", status, got)
	}
}

func TestPrivateCaptureFailureIsIdempotentPerOwnerRequestAndCode(t *testing.T) {
	db := dbtest.NewDB(t, &PrivateCaptureFailure{}, &PrivateCollectionRequest{})
	svc := NewService(db, dbtest.NewLogger(t))
	in := &PrivateCaptureFailureInput{
		OwnerID: 42, RequestID: "collect_retry_failure", SourceURL: "https://detail.1688.com/offer/123.html",
		ErrorCode: PrivateFailureSKUParseFailed, SchemaVersion: "sourcing1688.private.v1",
		ExtensionVersion: "0.2.0", ParserVersion: "v1", OccurredAt: time.Now().UTC(),
	}
	first, replay, err := svc.RecordPrivateCaptureFailure(in)
	if err != nil || replay {
		t.Fatalf("first = %#v replay=%v err=%v", first, replay, err)
	}
	second, replay, err := svc.RecordPrivateCaptureFailure(in)
	if err != nil || !replay || second.ID != first.ID {
		t.Fatalf("second = %#v replay=%v err=%v", second, replay, err)
	}
	otherOwner := *in
	otherOwner.OwnerID = 99
	third, replay, err := svc.RecordPrivateCaptureFailure(&otherOwner)
	if err != nil || replay || third.ID == first.ID {
		t.Fatalf("other owner = %#v replay=%v err=%v", third, replay, err)
	}
}

func TestCollectPrivateValidationFailureIsBestEffortAudited(t *testing.T) {
	db := dbtest.NewDB(t, &PrivateCaptureFailure{}, &PrivateCollectionRequest{}, &Sourcing1688Product{}, &Sourcing1688Snapshot{})
	svc := NewService(db, dbtest.NewLogger(t))
	title := ""
	in := &PrivateCollectInput{
		OwnerID: 42, RequestID: "collect_missing_title", SourceURL: "https://detail.1688.com/offer/123.html",
		ObservedAt: time.Now().UTC(), SchemaVersion: "sourcing1688.private.v1", ExtensionVersion: "0.2.0",
		ParserVersion: "v1", PageOfferID: "123", Title: &title,
		RawPayload:    json.RawMessage(`{"schema_version":"sourcing1688.private.v1"}`),
		FieldStatuses: json.RawMessage(`{"title":"parse_failed","price":"unknown","moq":"unknown","supplier":"unknown","sku":"no_sku"}`),
	}
	if _, err := svc.CollectPrivate(in); err == nil {
		t.Fatal("CollectPrivate() expected validation error")
	}
	validTitle := "页面标题"
	invalidURL := *in
	invalidURL.RequestID = "collect_invalid_url"
	invalidURL.SourceURL = "https://example.invalid/path/device-secret"
	invalidURL.Title = &validTitle
	if _, err := svc.CollectPrivate(&invalidURL); err == nil {
		t.Fatal("CollectPrivate() expected invalid URL error")
	}
	items, err := svc.ListPrivateCaptureFailures(42)
	if err != nil || len(items) != 2 {
		t.Fatalf("audited failures = %#v err=%v", items, err)
	}
	codes := map[string]bool{}
	for _, item := range items {
		codes[item.ErrorCode] = true
	}
	if !codes[PrivateFailureTitleParseFailed] || !codes[PrivateFailureInvalidSourceURL] {
		t.Fatalf("audited failure codes = %#v", codes)
	}
}
