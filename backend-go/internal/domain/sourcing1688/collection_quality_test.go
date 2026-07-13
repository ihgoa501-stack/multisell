package sourcing1688

import (
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
)

func qualityRaw(t *testing.T, driver, title string, price float64, statuses map[string]string) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"driver": driver, "title": title, "price_1688": price, "price_model": "fixed",
		"min_order_qty": 1, "supplier_name": "供应商", "supplier_business_id": "shop-1",
		"images": []string{"https://cbu01.alicdn.com/a.jpg"}, "spec_variants": []map[string]any{{"name": "红色"}},
		"attributes": map[string]string{"材质": "棉"}, "field_statuses": statuses,
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func qualitySnapshot(productID, ownerID, id int64, at time.Time, parser string, raw json.RawMessage) Sourcing1688Snapshot {
	return Sourcing1688Snapshot{ID: id, SourcingProductID: productID, SourceURL: "https://detail.1688.com/offer/123.html", CollectedAt: at, CollectedBy: ownerID, Driver: "chrome_extension", ParserVersion: parser, CaptureMode: CaptureModeExtensionClick, CollectionRequestID: "collect-quality", RawPayload: raw, RawSHA256: "hash"}
}

func TestCollectionQualityListThenDetailDerivesBestFieldsAndConflicts(t *testing.T) {
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &demandCaseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	title := "列表标题"
	product := Sourcing1688Product{OwnerID: 42, SourceURL: "https://detail.1688.com/offer/123.html", SourceOfferID: "123", Title: &title, LifecycleStatus: LifecycleUnverifiedLead}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 7, 13, 1, 0, 0, 0, time.UTC)
	listStatuses := map[string]string{"title": "observed", "price": "observed", "moq": "unknown", "supplier": "unknown", "images": "observed", "sku": "unknown", "attributes": "unknown"}
	detailStatuses := map[string]string{"title": "observed", "price": "observed", "moq": "observed", "supplier": "observed", "images": "observed", "sku": "observed", "attributes": "observed"}
	list := qualitySnapshot(product.ID, 42, 0, base, "1688-list-visible-v1", qualityRaw(t, "chrome_extension_list_visible", "列表标题", 10, listStatuses))
	detail := qualitySnapshot(product.ID, 42, 0, base.Add(time.Hour), "lingmirror-extension@0.2.0", qualityRaw(t, "plugin", "详情标题", 12, detailStatuses))
	if err := db.Create(&list).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&detail).Error; err != nil {
		t.Fatal(err)
	}

	quality, err := svc.GetCollectionQuality(product.ID, 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(quality.Observations) != 2 || quality.LatestListObservation == nil || quality.LatestDetailObservation == nil {
		t.Fatalf("observations=%#v", quality)
	}
	if quality.BestFields["price"].Source.PageKind != CollectionPageDetail {
		t.Fatalf("best price=%#v", quality.BestFields["price"])
	}
	if len(quality.Missing) != 0 {
		t.Fatalf("missing=%v", quality.Missing)
	}
	if len(quality.Conflicts) < 2 {
		t.Fatalf("conflicts=%#v", quality.Conflicts)
	}
	if quality.RecaptureAction.Kind != "retry_detail_collection" {
		t.Fatalf("action=%#v", quality.RecaptureAction)
	}
}

func TestCollectionQualityIsOwnerIsolated(t *testing.T) {
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &demandCaseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	title := "私有商品"
	product := Sourcing1688Product{OwnerID: 42, SourceURL: "https://detail.1688.com/offer/9.html", SourceOfferID: "9", Title: &title}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetCollectionQuality(product.ID, 99); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("expected owner gate, got %v", err)
	}
}

func TestCollectionQualityHTTPIsOwnerIsolatedAndNeverReturnsRawPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &demandCaseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	title := "HTTP质量商品"
	product := Sourcing1688Product{OwnerID: 42, SourceURL: "https://detail.1688.com/offer/19.html", SourceOfferID: "19", Title: &title}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	statuses := map[string]string{"title": "observed", "price": "observed", "moq": "observed", "supplier": "observed", "images": "observed", "sku": "observed", "attributes": "observed"}
	snapshot := qualitySnapshot(product.ID, 42, 0, time.Now().UTC(), "lingmirror-extension@0.2.0", qualityRaw(t, "plugin", title, 10, statuses))
	if err := db.Create(&snapshot).Error; err != nil {
		t.Fatal(err)
	}
	h := NewHandler(svc)

	request := func(ownerID int64) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/sourcing-1688/%d/collection-quality", product.ID), nil)
		context.Params = gin.Params{{Key: "id", Value: fmt.Sprint(product.ID)}}
		context.Set("user_id", ownerID)
		h.CollectionQuality(context)
		return recorder
	}
	owner := request(42)
	if owner.Code != http.StatusOK {
		t.Fatalf("owner status=%d body=%s", owner.Code, owner.Body.String())
	}
	if strings.Contains(owner.Body.String(), "raw_payload") || strings.Contains(owner.Body.String(), "cbu01.alicdn.com") {
		t.Fatalf("unsafe payload leaked: %s", owner.Body.String())
	}
	other := request(99)
	if other.Code != http.StatusConflict {
		t.Fatalf("other owner status=%d body=%s", other.Code, other.Body.String())
	}
}

func TestCollectionQualitySeesRecaptureAfterGovernedSnapshotPointerFreezes(t *testing.T) {
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &demandCaseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	title, experiment := "治理商品", "EXP-Q"
	product := Sourcing1688Product{OwnerID: 42, SourceURL: "https://detail.1688.com/offer/77.html", SourceOfferID: "77", Title: &title, ExperimentID: &experiment, LifecycleStatus: LifecycleNeedsReview}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 7, 13, 2, 0, 0, 0, time.UTC)
	statuses := map[string]string{"title": "observed", "price": "observed", "moq": "observed", "supplier": "observed", "images": "observed", "sku": "observed", "attributes": "observed"}
	old := qualitySnapshot(product.ID, 42, 0, at, "lingmirror-extension@0.2.0", qualityRaw(t, "plugin", title, 10, statuses))
	if err := db.Create(&old).Error; err != nil {
		t.Fatal(err)
	}
	product.SnapshotID = &old.ID
	if err := db.Save(&product).Error; err != nil {
		t.Fatal(err)
	}
	newer := qualitySnapshot(product.ID, 42, 0, at.Add(time.Hour), "lingmirror-extension@0.2.1", qualityRaw(t, "plugin", title, 8, statuses))
	if err := db.Create(&newer).Error; err != nil {
		t.Fatal(err)
	}

	quality, err := svc.GetCollectionQuality(product.ID, 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(quality.Observations) != 2 || quality.LatestDetailObservation == nil || quality.LatestDetailObservation.SnapshotID != newer.ID {
		t.Fatalf("new observation hidden: %#v", quality)
	}
	if product.SnapshotID == nil || *product.SnapshotID != old.ID {
		t.Fatalf("test changed governed pointer")
	}
}
