package sourcing1688

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestPrivateCollectionSavesOwnerUnverifiedLead(t *testing.T) {
	db := dbtest.NewDB(t, &Sourcing1688Product{}, &Sourcing1688Snapshot{}, &PrivateCollectionRequest{}, &PrivateCaptureFailure{}, &demandCaseRow{})
	svc := NewService(db, dbtest.NewLogger(t))
	title := "1688测试商品"
	statuses := json.RawMessage(`{"title":"observed","price":"unknown","moq":"unknown","supplier":"unknown","images":"unknown","sku":"no_sku"}`)
	raw := json.RawMessage(`{"schema_version":"sourcing1688.private.v1","offer_id_url":"123456","offer_id_page":"123456","source_url":"https://detail.1688.com/offer/123456.html","title":"1688测试商品","price_model":"unknown","field_statuses":{"title":"observed","price":"unknown","moq":"unknown","supplier":"unknown","images":"unknown","sku":"no_sku"}}`)
	result, err := svc.CollectPrivate(&PrivateCollectInput{OwnerID: 42, SchemaVersion: "sourcing1688.private.v1", PageOfferID: "123456", PriceModel: "unknown", RequestID: "collect_test_001", SourceURL: "https://detail.1688.com/offer/123456.html", ObservedAt: time.Now().UTC(), ParserVersion: "1688-detail-v1", ExtensionVersion: "0.2.0", RawPayload: raw, Title: &title, FieldStatuses: statuses})
	if err != nil {
		t.Fatal(err)
	}
	if result.Product.OwnerID != 42 || result.Product.Status != StatusUnverifiedLead || result.Snapshot.CollectionRequestID != "collect_test_001" {
		t.Fatalf("unexpected result: %+v", result)
	}
	replay, err := svc.CollectPrivate(&PrivateCollectInput{OwnerID: 42, SchemaVersion: "sourcing1688.private.v1", PageOfferID: "123456", PriceModel: "unknown", RequestID: "collect_test_001", SourceURL: "https://detail.1688.com/offer/123456.html", ObservedAt: result.Snapshot.CollectedAt, ParserVersion: "1688-detail-v1", ExtensionVersion: "0.2.0", RawPayload: raw, Title: &title, FieldStatuses: statuses})
	if err != nil || !replay.IdempotentReplay || replay.Product.ID != result.Product.ID {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
}
