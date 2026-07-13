package sourcing1688

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestListPrivateCollectionBoxExposesImmutableFieldStatusesAndRelationshipCounts(t *testing.T) {
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &Sourcing1688TaskLink{}, &demandCaseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	title := "采集箱商品"
	product := Sourcing1688Product{OwnerID: 42, SourceURL: "https://detail.1688.com/offer/123.html", SourceOfferID: "123", Title: &title, Status: StatusUnverifiedLead, LifecycleStatus: LifecycleNeedsReview}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	statuses := map[string]string{"title": "observed", "price": "unknown", "moq": "unknown", "supplier": "observed", "images": "unknown", "sku": "parse_failed"}
	raw, _ := json.Marshal(map[string]any{"field_statuses": statuses})
	for index := 0; index < 2; index++ {
		snapshot := Sourcing1688Snapshot{SourcingProductID: product.ID, SourceURL: product.SourceURL, CollectedAt: time.Now().UTC(), CollectedBy: 42, Driver: "chrome_extension", ParserVersion: "v1", CaptureMode: CaptureModeExtensionClick, CollectionRequestID: "request-" + string(rune('a'+index)), RawPayload: raw, RawSHA256: "hash"}
		if err := db.Create(&snapshot).Error; err != nil {
			t.Fatal(err)
		}
		product.SnapshotID = &snapshot.ID
	}
	if err := db.Save(&product).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Sourcing1688TaskLink{SourcingProductID: product.ID, DemandCaseID: 9, ExperimentID: "exp-1", OwnerID: 42, Status: "linked"}).Error; err != nil {
		t.Fatal(err)
	}

	items, total, err := svc.ListPrivateCollectionBox(42, &common.Pagination{Page: 1, Size: 20}, &ListFilter{LifecycleStatus: LifecycleNeedsReview})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("items=%#v total=%d", items, total)
	}
	if items[0].ObservationCount != 2 || items[0].TaskLinkCount != 1 {
		t.Fatalf("counts=%#v", items[0])
	}
	if items[0].FieldStatuses["sku"] != "parse_failed" || items[0].FieldStatuses["price"] != "unknown" {
		t.Fatalf("field statuses=%#v", items[0].FieldStatuses)
	}
	if items[0].LatestPageKind != CollectionPageDetail || items[0].LatestObservedAt == nil {
		t.Fatalf("latest source=%#v", items[0])
	}

	otherTitle := "其他Owner商品"
	other := Sourcing1688Product{OwnerID: 99, SourceURL: "https://detail.1688.com/offer/999.html", SourceOfferID: "999", Title: &otherTitle, Status: StatusUnverifiedLead, LifecycleStatus: LifecycleNeedsReview}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	found, total, err := svc.ListPrivateCollectionBox(42, &common.Pagination{Page: 1, Size: 20}, &ListFilter{RecordID: &product.ID})
	if err != nil || total != 1 || len(found) != 1 || found[0].ID != product.ID {
		t.Fatalf("exact record items=%#v total=%d err=%v", found, total, err)
	}
	leaked, total, err := svc.ListPrivateCollectionBox(42, &common.Pagination{Page: 1, Size: 20}, &ListFilter{RecordID: &other.ID})
	if err != nil || total != 0 || len(leaked) != 0 {
		t.Fatalf("cross-owner exact record leaked items=%#v total=%d err=%v", leaked, total, err)
	}

	requestList := func(recordID string) (int, int64) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", int64(42))
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/sourcing-1688?record_id="+recordID, nil)
		NewHandler(svc).List(c)
		var body struct {
			Total int64 `json:"total"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		return w.Code, body.Total
	}
	if code, gotTotal := requestList(strconv.FormatInt(product.ID, 10)); code != http.StatusOK || gotTotal != 1 {
		t.Fatalf("record API code=%d total=%d", code, gotTotal)
	}
	if code, gotTotal := requestList(strconv.FormatInt(other.ID, 10)); code != http.StatusOK || gotTotal != 0 {
		t.Fatalf("cross-owner record API code=%d total=%d", code, gotTotal)
	}
	if code, _ := requestList("bad"); code != http.StatusBadRequest {
		t.Fatalf("invalid record API code=%d", code)
	}

	empty, total, err := svc.ListPrivateCollectionBox(42, &common.Pagination{Page: 1, Size: 20}, &ListFilter{LifecycleStatus: LifecycleArchived})
	if err != nil || total != 0 || len(empty) != 0 {
		t.Fatalf("archived filter items=%#v total=%d err=%v", empty, total, err)
	}
}
