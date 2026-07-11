package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/platform/eventbus"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
	"go.uber.org/zap"
)

type marketListDriver struct {
	items             []toolbridge.ListItemData
	seenURLs          []string
	seenOwnerID       int64
	seenCorrelationID string
}

func (d *marketListDriver) FetchListPage(ctx context.Context, url string) (*toolbridge.ListPageData, error) {
	d.seenURLs = append(d.seenURLs, url)
	d.seenOwnerID, _ = toolbridge.OwnerUserIDFromContext(ctx)
	d.seenCorrelationID = eventbus.CorrelationIDFromContext(ctx)
	return &toolbridge.ListPageData{
		PageURL:     url,
		CollectedAt: time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC),
		Items:       d.items,
		Driver:      "test-browser",
		RawData:     json.RawMessage(`{"source":"ozon-search"}`),
	}, nil
}

func TestCollectionAgentMarketDiscoverBuildsOzonSearchURLFromQuery(t *testing.T) {
	db := dbtest.NewDB(t, &candidate.CollectLead{}, &candidate.CollectionEvidence{})
	items := make([]toolbridge.ListItemData, 20)
	for i := range items {
		items[i] = toolbridge.ListItemData{Title: fmt.Sprintf("桌面收纳%d", i), PriceRange: "1299", DetailURL: fmt.Sprintf("https://www.ozon.ru/product/%d", i)}
	}
	driver := &marketListDriver{items: items}
	bridge := toolbridge.NewToolBridge([]toolbridge.DriverEntry{{Name: "market", Driver: driver, Weight: 1}}, time.Second, zap.NewNop())
	agent := NewCollectionAgent(bridge, candidate.NewService(db, zap.NewNop()), zap.NewNop())

	result, _, _, err := agent.Decide(context.Background(), "market_discover", map[string]interface{}{
		"queries":           []interface{}{"桌面 收纳"},
		"min_opportunities": float64(1),
		"_owner_user_id":    float64(42),
		"_correlation_id":   "request-test-42",
	})
	if err != nil {
		t.Fatalf("market_discover error = %v", err)
	}
	if len(driver.seenURLs) != 1 || driver.seenURLs[0] != "https://www.ozon.ru/search/?text=%E6%A1%8C%E9%9D%A2+%E6%94%B6%E7%BA%B3" {
		t.Fatalf("unexpected search URLs: %v", driver.seenURLs)
	}
	if got := result["minimum_required"]; got != 20 {
		t.Fatalf("minimum_required = %#v, want hard floor 20", got)
	}
	if driver.seenOwnerID != 42 || driver.seenCorrelationID != "request-test-42" {
		t.Fatalf("collection identity not propagated: owner=%d correlation=%q", driver.seenOwnerID, driver.seenCorrelationID)
	}
}

func (d *marketListDriver) FetchPage(context.Context, string) (*toolbridge.PageData, error) {
	return nil, fmt.Errorf("not used")
}

func TestCollectionAgentMarketDiscoverDoesNotCountExistingLeadAsNew(t *testing.T) {
	db := dbtest.NewDB(t, &candidate.CollectLead{}, &candidate.CollectionEvidence{})
	svc := candidate.NewService(db, zap.NewNop())
	lead := &candidate.CollectLead{Title: "已存在", PriceRange: "1000", DetailURL: "https://www.ozon.ru/product/1"}
	if err := svc.CreateCollectLead(lead); err != nil {
		t.Fatal(err)
	}
	driver := &marketListDriver{items: []toolbridge.ListItemData{{Title: "已存在", PriceRange: "1000", DetailURL: lead.DetailURL}}}
	bridge := toolbridge.NewToolBridge([]toolbridge.DriverEntry{{Name: "market", Driver: driver, Weight: 1}}, time.Second, zap.NewNop())
	agent := NewCollectionAgent(bridge, svc, zap.NewNop())

	result, _, _, err := agent.Decide(context.Background(), "market_discover", map[string]interface{}{
		"search_urls":    []interface{}{"https://www.ozon.ru/search/?text=storage"},
		"_owner_user_id": float64(42),
	})
	if err != nil {
		t.Fatalf("market_discover error = %v", err)
	}
	if got := result["collected"]; got != 0 {
		t.Fatalf("collected = %#v, want 0 new leads", got)
	}
	if got := result["duplicates"]; got != 1 {
		t.Fatalf("duplicates = %#v, want 1", got)
	}
}
func (d *marketListDriver) Health() (bool, time.Duration, error) { return true, 0, nil }
func (d *marketListDriver) Category() toolbridge.ToolCategory    { return toolbridge.ToolCategoryRead }
func (d *marketListDriver) Execute(map[string]interface{}) (*toolbridge.ToolResult, error) {
	return nil, fmt.Errorf("not used")
}

func TestCollectionAgentMarketDiscoverPersistsTwentyTraceableLeads(t *testing.T) {
	db := dbtest.NewDB(t, &candidate.CollectLead{}, &candidate.CollectionEvidence{})
	items := make([]toolbridge.ListItemData, 0, 21)
	for i := 1; i <= 20; i++ {
		items = append(items, toolbridge.ListItemData{
			Title:      fmt.Sprintf("Ozon 商品 %02d", i),
			PriceRange: fmt.Sprintf("%d", 1000+i),
			DetailURL:  fmt.Sprintf("https://www.ozon.ru/product/%d", i),
			ImageURL:   fmt.Sprintf("https://cdn.ozon.ru/%d.jpg", i),
		})
	}
	items = append(items, items[0]) // duplicate URL must not be persisted twice

	driver := &marketListDriver{items: items}
	bridge := toolbridge.NewToolBridge([]toolbridge.DriverEntry{{Name: "market", Driver: driver, Weight: 1}}, time.Second, zap.NewNop())
	agent := NewCollectionAgent(bridge, candidate.NewService(db, zap.NewNop()), zap.NewNop())

	result, _, _, err := agent.Decide(context.Background(), "market_discover", map[string]interface{}{
		"search_urls":       []interface{}{"https://www.ozon.ru/search/?text=storage"},
		"min_opportunities": float64(20),
		"_owner_user_id":    float64(42),
	})
	if err != nil {
		t.Fatalf("market_discover error = %v", err)
	}
	if got := result["collected"]; got != 20 {
		t.Fatalf("collected = %#v, want 20", got)
	}

	var leads []candidate.CollectLead
	if err := db.Order("id ASC").Find(&leads).Error; err != nil {
		t.Fatal(err)
	}
	if len(leads) != 20 {
		t.Fatalf("persisted leads = %d, want 20", len(leads))
	}
	if leads[0].CollectionDriver != "test-browser" || leads[0].EvidenceID == nil {
		t.Fatalf("lead provenance missing: %+v", leads[0])
	}
	var evidence []candidate.CollectionEvidence
	if err := db.Find(&evidence).Error; err != nil {
		t.Fatal(err)
	}
	if len(evidence) != 1 || len(evidence[0].RawPayload) == 0 || evidence[0].ParserVersion == "" {
		t.Fatalf("page evidence missing or duplicated: %+v", evidence)
	}
}
