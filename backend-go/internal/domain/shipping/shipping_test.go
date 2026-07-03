package shipping

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testDBCounter int64

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	testDBCounter++
	dsn := "file:shipping_test_" + itoa(testDBCounter) + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&ShippingProvider{}, &ShippingChannel{}, &ShippingZone{}, &ShippingQuoteRule{}, &ShippingBillBatch{}, &ShippingBillItem{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func statusPtr(s int16) *int16 { return &s }

// ---------- Provider Tests ----------

func TestProvider_CRUD(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	// Create
	p, err := svc.CreateProvider(&CreateProviderInput{
		Name: "极兔速递", Code: "JITU", Contact: "张三", Phone: "13800138000",
	})
	if err != nil {
		t.Fatalf("CreateProvider failed: %v", err)
	}
	if p.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	// Get
	got, err := svc.GetProvider(p.ID)
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}
	if got.Name != "极兔速递" {
		t.Errorf("expected 极兔速递, got %s", got.Name)
	}

	// Update
	newName := "极兔国际"
	updated, err := svc.UpdateProvider(p.ID, &UpdateProviderInput{Name: &newName})
	if err != nil {
		t.Fatalf("UpdateProvider failed: %v", err)
	}
	if updated.Name != "极兔国际" {
		t.Errorf("expected 极兔国际, got %s", updated.Name)
	}

	// List
	items, total, err := svc.ListProviders(&common.Pagination{Page: 1, Size: 10}, "")
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	// Delete
	if err := svc.DeleteProvider(p.ID); err != nil {
		t.Fatalf("DeleteProvider failed: %v", err)
	}
	_, err = svc.GetProvider(p.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestProvider_List_Search(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	svc.CreateProvider(&CreateProviderInput{Name: "极兔速递", Code: "JITU"})
	svc.CreateProvider(&CreateProviderInput{Name: "圆通速递", Code: "YTO"})
	svc.CreateProvider(&CreateProviderInput{Name: "顺丰国际", Code: "SF"})

	// Search by name
	_, total, err := svc.ListProviders(&common.Pagination{Page: 1, Size: 10}, "极兔")
	if err != nil {
		t.Fatalf("ListProviders search failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 result, got %d", total)
	}

	// Search by code
	_, total, err = svc.ListProviders(&common.Pagination{Page: 1, Size: 10}, "SF")
	if err != nil {
		t.Fatalf("ListProviders search by code failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 result for SF, got %d", total)
	}
}

func TestProvider_Delete_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	err := svc.DeleteProvider(99999)
	if err == nil {
		t.Fatal("expected error for not found delete")
	}
}

// ---------- Channel Tests ----------

func TestChannel_FullCRUD(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	prov, _ := svc.CreateProvider(&CreateProviderInput{Name: "承运商", Code: "CARRIER"})

	// Create
	ch, err := svc.CreateChannel(&CreateChannelInput{
		ProviderID: prov.ID, Name: "经济快递", Code: "ECO",
		CargoTypes: json.RawMessage(`["normal"]`),
	})
	if err != nil {
		t.Fatalf("CreateChannel failed: %v", err)
	}

	// Get
	got, err := svc.GetChannel(ch.ID)
	if err != nil {
		t.Fatalf("GetChannel failed: %v", err)
	}
	if got.Name != "经济快递" {
		t.Errorf("expected 经济快递, got %s", got.Name)
	}
	if got.VolumetricDivisor != 6000 {
		t.Errorf("expected default volumetric divisor 6000, got %d", got.VolumetricDivisor)
	}
	if got.Currency != "CNY" {
		t.Errorf("expected default currency CNY, got %s", got.Currency)
	}

	// Update
	newCurrency := "USD"
	updated, err := svc.UpdateChannel(ch.ID, &UpdateChannelInput{Currency: &newCurrency})
	if err != nil {
		t.Fatalf("UpdateChannel failed: %v", err)
	}
	if updated.Currency != "USD" {
		t.Errorf("expected USD, got %s", updated.Currency)
	}

	// Delete
	if err := svc.DeleteChannel(ch.ID); err != nil {
		t.Fatalf("DeleteChannel failed: %v", err)
	}
}

func TestChannel_List_FilterByProvider(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	p1, _ := svc.CreateProvider(&CreateProviderInput{Name: "承运商A", Code: "A"})
	p2, _ := svc.CreateProvider(&CreateProviderInput{Name: "承运商B", Code: "B"})

	svc.CreateChannel(&CreateChannelInput{ProviderID: p1.ID, Name: "A标准"})
	svc.CreateChannel(&CreateChannelInput{ProviderID: p2.ID, Name: "B标准"})

	items, total, err := svc.ListChannels(&common.Pagination{Page: 1, Size: 10}, &p1.ID, "")
	if err != nil {
		t.Fatalf("ListChannels filter failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 channel for provider A, got %d", total)
	}
	if len(items) != 1 || items[0].Name != "A标准" {
		t.Errorf("expected A标准, got %s", items[0].Name)
	}
}

// ---------- Zone Tests ----------

func TestZone_CRUD(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	prov, _ := svc.CreateProvider(&CreateProviderInput{Name: "承运商", Code: "C"})
	ch, _ := svc.CreateChannel(&CreateChannelInput{ProviderID: prov.ID, Name: "标准快递"})

	z, err := svc.CreateZone(&CreateZoneInput{
		ChannelID: ch.ID, CountryCode: "RU",
	})
	if err != nil {
		t.Fatalf("CreateZone failed: %v", err)
	}
	if z.ID == 0 {
		t.Fatal("expected non-zero zone ID")
	}

	// List by channel
	items, total, err := svc.ListZones(&common.Pagination{Page: 1, Size: 10}, &ch.ID)
	if err != nil {
		t.Fatalf("ListZones failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 zone, got %d", total)
	}
	if len(items) != 1 || items[0].CountryCode != "RU" {
		t.Errorf("expected RU, got %s", items[0].CountryCode)
	}

	// Delete
	if err := svc.DeleteZone(z.ID); err != nil {
		t.Fatalf("DeleteZone failed: %v", err)
	}
	items, total, _ = svc.ListZones(&common.Pagination{Page: 1, Size: 10}, &ch.ID)
	if total != 0 {
		t.Errorf("expected 0 zones after delete, got %d", total)
	}
}

// ---------- Quote Rule Tests ----------

func TestQuoteRule_CRUD(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	prov, _ := svc.CreateProvider(&CreateProviderInput{Name: "承运商", Code: "C"})
	ch, _ := svc.CreateChannel(&CreateChannelInput{ProviderID: prov.ID, Name: "标准快递"})
	zone, _ := svc.CreateZone(&CreateZoneInput{ChannelID: ch.ID, CountryCode: "RU"})

	fixedFee := 20.0
	perKg := 8.0
	minCharge := 25.0

	r, err := svc.CreateRule(&CreateQuoteRuleInput{
		ChannelID: ch.ID, ZoneID: &zone.ID,
		RuleType:     "fixed_plus_per_kg",
		FixedFee:     &fixedFee,
		PerKgPrice:   &perKg,
		MinimumCharge: &minCharge,
		Status:       statusPtr(1),
	})
	if err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}
	if r.ID == 0 {
		t.Fatal("expected non-zero rule ID")
	}

	// List by channel
	_, total, err := svc.ListRules(&common.Pagination{Page: 1, Size: 10}, &ch.ID)
	if err != nil {
		t.Fatalf("ListRules failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 rule, got %d", total)
	}

	// Delete
	if err := svc.DeleteRule(r.ID); err != nil {
		t.Fatalf("DeleteRule failed: %v", err)
	}
}

// ---------- Bill Batch Tests ----------

func TestBillBatch_CRUD(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	prov, _ := svc.CreateProvider(&CreateProviderInput{Name: "承运商", Code: "C"})

	b, err := svc.CreateBillBatch(&CreateBillBatchInput{
		ProviderID: &prov.ID, SourceFilename: "运费单_202606.csv", CreatedBy: "admin",
	})
	if err != nil {
		t.Fatalf("CreateBillBatch failed: %v", err)
	}
	if b.ID == 0 {
		t.Fatal("expected non-zero batch ID")
	}
	if b.Status != "imported" {
		t.Errorf("expected default status imported, got %s", b.Status)
	}
	if b.Currency != "CNY" {
		t.Errorf("expected default currency CNY, got %s", b.Currency)
	}

	// List
	items, total, err := svc.ListBillBatches(&common.Pagination{Page: 1, Size: 10}, nil)
	if err != nil {
		t.Fatalf("ListBillBatches failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 batch, got %d", total)
	}

	_ = items
	batch, billItems, err := svc.GetBillBatch(b.ID)
	if err != nil {
		t.Fatalf("GetBillBatch failed: %v", err)
	}
	if batch.ID != b.ID {
		t.Errorf("expected batch ID %d, got %d", b.ID, batch.ID)
	}
	if len(billItems) != 0 {
		t.Errorf("expected 0 bill items, got %d", len(billItems))
	}

	// Delete
	if err := svc.DeleteBillBatch(b.ID); err != nil {
		t.Fatalf("DeleteBillBatch failed: %v", err)
	}
}

// ---------- Quote Calculation Tests ----------

func TestQuote_Manual_FixedPlusPerKg(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	prov, _ := svc.CreateProvider(&CreateProviderInput{Name: "极兔", Code: "JITU"})
	ch, _ := svc.CreateChannel(&CreateChannelInput{
		ProviderID: prov.ID, Name: "经济空运", Code: "ECO-AIR",
		VolumetricDivisor: intPtr(6000), CargoTypes: json.RawMessage(`["normal"]`),
	})
	zone, _ := svc.CreateZone(&CreateZoneInput{ChannelID: ch.ID, CountryCode: "RU"})

	fixedFee := 10.0
	perKg := 5.0
	svc.CreateRule(&CreateQuoteRuleInput{
		ChannelID: ch.ID, ZoneID: &zone.ID,
		RuleType:   "fixed_plus_per_kg",
		FixedFee:   &fixedFee,
		PerKgPrice: &perKg,
		Status:     statusPtr(1),
	})

	resp, err := svc.Quote(&QuoteRequest{
		Mode:               "manual",
		DestinationCountry: "RU",
		ManualWeightKg:     floatPtr(2.0),
		ManualLengthCM:     floatPtr(30),
		ManualWidthCM:      floatPtr(20),
		ManualHeightCM:     floatPtr(15),
		CargoType:          "normal",
	})
	if err != nil {
		t.Fatalf("Quote failed: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	// Weight=2.0 => 2.0*5 + 10 = 20
	r := resp.Results[0]
	if r.ChannelName != "经济空运" {
		t.Errorf("expected 经济空运, got %s", r.ChannelName)
	}
	if r.BaseShippingFee != 20.0 {
		t.Errorf("expected base fee 20.0, got %.2f", r.BaseShippingFee)
	}
	if r.TotalShippingFee != 20.0 {
		t.Errorf("expected total fee 20.0, got %.2f", r.TotalShippingFee)
	}
	if r.ChargeableWeightKg != 2.0 {
		t.Errorf("expected chargeable weight 2.0, got %.4f", r.ChargeableWeightKg)
	}
}

func TestQuote_Manual_WithVolumetricWeight(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	prov, _ := svc.CreateProvider(&CreateProviderInput{Name: "极兔", Code: "JITU"})
	ch, _ := svc.CreateChannel(&CreateChannelInput{
		ProviderID: prov.ID, Name: "经济空运",
		VolumetricDivisor: intPtr(6000), CargoTypes: json.RawMessage(`["normal"]`),
	})
	zone, _ := svc.CreateZone(&CreateZoneInput{ChannelID: ch.ID, CountryCode: "RU"})

	fixedFee := 10.0
	perKg := 5.0
	svc.CreateRule(&CreateQuoteRuleInput{
		ChannelID: ch.ID, ZoneID: &zone.ID,
		RuleType:   "fixed_plus_per_kg",
		FixedFee:   &fixedFee,
		PerKgPrice: &perKg,
		Status:     statusPtr(1),
	})

	// Light weight but large volume: 40x40x40cm = 64000cm3, /6000 = 10.67kg volumetric
	// Chargeable = max(2.0, 10.67) = 10.67
	resp, err := svc.Quote(&QuoteRequest{
		Mode:               "manual",
		DestinationCountry: "RU",
		ManualWeightKg:     floatPtr(2.0),
		ManualLengthCM:     floatPtr(40),
		ManualWidthCM:      floatPtr(40),
		ManualHeightCM:     floatPtr(40),
	})
	if err != nil {
		t.Fatalf("Quote failed: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	r := resp.Results[0]
	// Volumetric = 40*40*40/6000 = 10.6667, chargeable = max(2, 10.667) = 10.667
	if r.VolumetricWeightKg < 10.6 || r.VolumetricWeightKg > 10.7 {
		t.Errorf("expected volumetric weight ~10.67, got %.4f", r.VolumetricWeightKg)
	}
		if r.ChargeableWeightKg != 10.7 {
			t.Errorf("expected chargeable weight 10.7, got %.4f", r.ChargeableWeightKg)
	}
		// base = 10 + 10.7*5 = 63.5 (rounded to 0.1kg increments)
		if r.BaseShippingFee != 63.5 {
			t.Errorf("expected base fee 63.5, got %.2f", r.BaseShippingFee)
	}
}

func TestQuote_Manual_FirstWeightPlusIncrement(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	prov, _ := svc.CreateProvider(&CreateProviderInput{Name: "承运商", Code: "C"})
	ch, _ := svc.CreateChannel(&CreateChannelInput{
		ProviderID: prov.ID, Name: "标准快递",
		VolumetricDivisor: intPtr(6000),
	})
	zone, _ := svc.CreateZone(&CreateZoneInput{ChannelID: ch.ID, CountryCode: "RU"})

	firstKg := 0.5
	firstPrice := 15.0
	addKg := 0.5
	addPrice := 8.0
	svc.CreateRule(&CreateQuoteRuleInput{
		ChannelID: ch.ID, ZoneID: &zone.ID,
		RuleType:        "first_weight_plus_increment",
		FirstKg:        &firstKg,
		FirstPrice:     &firstPrice,
		AdditionalKg:   &addKg,
		AdditionalPrice: &addPrice,
		RoundingIncrement: floatPtr(0.1),
		Status:         statusPtr(1),
	})

	// 1.2kg: first 0.5kg = 15, remaining 0.7kg / 0.5 = 1.4 => ceil=2 increments => 2*8=16 => total=31
	resp, err := svc.Quote(&QuoteRequest{
		Mode:               "manual",
		DestinationCountry: "RU",
		ManualWeightKg:     floatPtr(1.2),
		ManualLengthCM:     floatPtr(10),
		ManualWidthCM:      floatPtr(10),
		ManualHeightCM:     floatPtr(10),
	})
	if err != nil {
		t.Fatalf("Quote failed: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	r := resp.Results[0]
	if r.BaseShippingFee != 31.0 {
		t.Errorf("expected base fee 31.0 (15+2*8), got %.2f", r.BaseShippingFee)
	}

	// 0.3kg (under first): total should be first price = 15
	resp2, _ := svc.Quote(&QuoteRequest{
		Mode:               "manual",
		DestinationCountry: "RU",
		ManualWeightKg:     floatPtr(0.3),
		ManualLengthCM:     floatPtr(10),
		ManualWidthCM:      floatPtr(10),
		ManualHeightCM:     floatPtr(10),
	})
	if resp2.Results[0].BaseShippingFee != 15.0 {
		t.Errorf("expected base fee 15.0 for under-first weight, got %.2f", resp2.Results[0].BaseShippingFee)
	}
}

func TestQuote_Manual_TieredWeight(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	prov, _ := svc.CreateProvider(&CreateProviderInput{Name: "承运商", Code: "C"})
	ch, _ := svc.CreateChannel(&CreateChannelInput{
		ProviderID: prov.ID, Name: "经济运输",
		VolumetricDivisor: intPtr(6000),
	})
	zone, _ := svc.CreateZone(&CreateZoneInput{ChannelID: ch.ID, CountryCode: "RU"})

	tierConfig := json.RawMessage(`[
		{"min_kg": 0, "max_kg": 1, "price": 15},
		{"min_kg": 1, "max_kg": 2, "price": 25},
		{"min_kg": 2, "price": 35}
	]`)
	svc.CreateRule(&CreateQuoteRuleInput{
		ChannelID:  ch.ID, ZoneID: &zone.ID,
		RuleType:   "tiered_weight",
		TierConfig: tierConfig,
		Status:     statusPtr(1),
	})

	// 0.5kg -> tier 1 (0-1): 15
	resp, _ := svc.Quote(&QuoteRequest{
		Mode: "manual", DestinationCountry: "RU",
		ManualWeightKg: floatPtr(0.5), ManualLengthCM: floatPtr(10),
		ManualWidthCM: floatPtr(10), ManualHeightCM: floatPtr(10),
	})
	if resp.Results[0].BaseShippingFee != 15.0 {
		t.Errorf("expected 15.0 for tier 1, got %.2f", resp.Results[0].BaseShippingFee)
	}

	// 1.5kg -> tier 2 (1-2): 25
	resp2, _ := svc.Quote(&QuoteRequest{
		Mode: "manual", DestinationCountry: "RU",
		ManualWeightKg: floatPtr(1.5), ManualLengthCM: floatPtr(10),
		ManualWidthCM: floatPtr(10), ManualHeightCM: floatPtr(10),
	})
	if resp2.Results[0].BaseShippingFee != 25.0 {
		t.Errorf("expected 25.0 for tier 2, got %.2f", resp2.Results[0].BaseShippingFee)
	}

	// 5kg -> tier 3 (2+): 35
	resp3, _ := svc.Quote(&QuoteRequest{
		Mode: "manual", DestinationCountry: "RU",
		ManualWeightKg: floatPtr(5.0), ManualLengthCM: floatPtr(10),
		ManualWidthCM: floatPtr(10), ManualHeightCM: floatPtr(10),
	})
	if resp3.Results[0].BaseShippingFee != 35.0 {
		t.Errorf("expected 35.0 for tier 3, got %.2f", resp3.Results[0].BaseShippingFee)
	}
}

func TestQuote_Manual_MinimumCharge(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	prov, _ := svc.CreateProvider(&CreateProviderInput{Name: "承运商", Code: "C"})
	ch, _ := svc.CreateChannel(&CreateChannelInput{
		ProviderID: prov.ID, Name: "标准快递",
		VolumetricDivisor: intPtr(6000),
	})
	zone, _ := svc.CreateZone(&CreateZoneInput{ChannelID: ch.ID, CountryCode: "RU"})

	fixedFee := 5.0
	perKg := 3.0
	minCharge := 30.0
	svc.CreateRule(&CreateQuoteRuleInput{
		ChannelID: ch.ID, ZoneID: &zone.ID,
		RuleType:      "fixed_plus_per_kg",
		FixedFee:      &fixedFee,
		PerKgPrice:    &perKg,
		MinimumCharge: &minCharge,
		Status:        statusPtr(1),
	})

	// 1kg: 5 + 1*3 = 8, minimum 30 => total 30
	resp, _ := svc.Quote(&QuoteRequest{
		Mode: "manual", DestinationCountry: "RU",
		ManualWeightKg: floatPtr(1.0), ManualLengthCM: floatPtr(10),
		ManualWidthCM: floatPtr(10), ManualHeightCM: floatPtr(10),
	})
	if resp.Results[0].TotalShippingFee != 30.0 {
		t.Errorf("expected total 30.0 (minimum charge), got %.2f", resp.Results[0].TotalShippingFee)
	}
}

func TestQuote_Manual_Surcharges(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	prov, _ := svc.CreateProvider(&CreateProviderInput{Name: "承运商", Code: "C"})
	ch, _ := svc.CreateChannel(&CreateChannelInput{
		ProviderID: prov.ID, Name: "标准快递",
		VolumetricDivisor: intPtr(6000),
	})
	zone, _ := svc.CreateZone(&CreateZoneInput{ChannelID: ch.ID, CountryCode: "RU"})

	fixedFee := 10.0
	perKg := 5.0
	surcharge := 3.0
	fuelPct := 10.0
	svc.CreateRule(&CreateQuoteRuleInput{
		ChannelID: ch.ID, ZoneID: &zone.ID,
		RuleType:       "fixed_plus_per_kg",
		FixedFee:       &fixedFee,
		PerKgPrice:     &perKg,
		SurchargeFixed: &surcharge,
		FuelSurchargePct: &fuelPct,
		Status:         statusPtr(1),
	})

	// 2kg: base=10+2*5=20, surcharge=3, fuel=(20+3)*0.1=2.3, total=25.3
	resp, _ := svc.Quote(&QuoteRequest{
		Mode: "manual", DestinationCountry: "RU",
		ManualWeightKg: floatPtr(2.0), ManualLengthCM: floatPtr(10),
		ManualWidthCM: floatPtr(10), ManualHeightCM: floatPtr(10),
	})
	r := resp.Results[0]
	if r.BaseShippingFee != 20.0 {
		t.Errorf("expected base fee 20.0, got %.2f", r.BaseShippingFee)
	}
	if r.SurchargeFee != 3.0 {
		t.Errorf("expected surcharge 3.0, got %.2f", r.SurchargeFee)
	}
	if r.FuelSurchargeFee != 2.3 {
		t.Errorf("expected fuel surcharge 2.3, got %.2f", r.FuelSurchargeFee)
	}
	if r.TotalShippingFee != 25.3 {
		t.Errorf("expected total 25.3, got %.2f", r.TotalShippingFee)
	}
}

func TestQuote_Manual_MultipleChannels_Sorted(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	prov, _ := svc.CreateProvider(&CreateProviderInput{Name: "承运商", Code: "C"})

	// Create 2 channels serving the same country
	ch1, _ := svc.CreateChannel(&CreateChannelInput{
		ProviderID: prov.ID, Name: "经济空运",
		VolumetricDivisor: intPtr(6000),
	})
	ch2, _ := svc.CreateChannel(&CreateChannelInput{
		ProviderID: prov.ID, Name: "极速空运",
		VolumetricDivisor: intPtr(5000),
	})

	zone1, _ := svc.CreateZone(&CreateZoneInput{ChannelID: ch1.ID, CountryCode: "RU"})
	zone2, _ := svc.CreateZone(&CreateZoneInput{ChannelID: ch2.ID, CountryCode: "RU"})

	econFixed := 10.0
	econPerKg := 5.0  // 2kg => base=20, total=20
	svc.CreateRule(&CreateQuoteRuleInput{
		ChannelID: ch1.ID, ZoneID: &zone1.ID,
		RuleType: "fixed_plus_per_kg", FixedFee: &econFixed,
		PerKgPrice: &econPerKg, Status: statusPtr(1),
	})

	fastFixed := 20.0
	fastPerKg := 8.0  // 2kg => base=36, total=36
	svc.CreateRule(&CreateQuoteRuleInput{
		ChannelID: ch2.ID, ZoneID: &zone2.ID,
		RuleType: "fixed_plus_per_kg", FixedFee: &fastFixed,
		PerKgPrice: &fastPerKg, Status: statusPtr(1),
	})

	resp, _ := svc.Quote(&QuoteRequest{
		Mode: "manual", DestinationCountry: "RU",
		ManualWeightKg: floatPtr(2.0), ManualLengthCM: floatPtr(10),
		ManualWidthCM: floatPtr(10), ManualHeightCM: floatPtr(10),
	})
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}
	// Results should be sorted by total fee ascending
	if resp.Results[0].TotalShippingFee > resp.Results[1].TotalShippingFee {
		t.Errorf("results not sorted by fee: first %.2f > second %.2f",
			resp.Results[0].TotalShippingFee, resp.Results[1].TotalShippingFee)
	}
	if resp.Results[0].ChannelName != "经济空运" {
		t.Errorf("expected cheapest first (经济空运), got %s", resp.Results[0].ChannelName)
	}
}

func TestQuote_NoEligibleChannel(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	// No providers/channels configured
	resp, err := svc.Quote(&QuoteRequest{
		Mode: "manual", DestinationCountry: "RU",
		ManualWeightKg: floatPtr(1.0), ManualLengthCM: floatPtr(10),
		ManualWidthCM: floatPtr(10), ManualHeightCM: floatPtr(10),
	})
	if err != nil {
		t.Fatalf("Quote with empty config should succeed (zero results): %v", err)
	}
	if len(resp.Results) != 0 {
		t.Errorf("expected 0 results with no channels, got %d", len(resp.Results))
	}
}

func TestQuote_ChannelNotServingCountry(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	prov, _ := svc.CreateProvider(&CreateProviderInput{Name: "承运商", Code: "C"})
	ch, _ := svc.CreateChannel(&CreateChannelInput{
		ProviderID: prov.ID, Name: "渠道",
		VolumetricDivisor: intPtr(6000),
	})
	// Zone for KZ, not RU
	svc.CreateZone(&CreateZoneInput{ChannelID: ch.ID, CountryCode: "KZ"})
	fixedFee := 10.0
	perKg := 5.0
	svc.CreateRule(&CreateQuoteRuleInput{
		ChannelID: ch.ID,
		RuleType: "fixed_plus_per_kg", FixedFee: &fixedFee,
		PerKgPrice: &perKg, Status: statusPtr(1),
	})

	resp, _ := svc.Quote(&QuoteRequest{
		Mode: "manual", DestinationCountry: "RU",
		ManualWeightKg: floatPtr(1.0), ManualLengthCM: floatPtr(10),
		ManualWidthCM: floatPtr(10), ManualHeightCM: floatPtr(10),
	})
	if len(resp.Results) != 0 {
		t.Errorf("expected 0 results for non-served country, got %d", len(resp.Results))
	}
}

func TestQuote_MultipleQuantity(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	prov, _ := svc.CreateProvider(&CreateProviderInput{Name: "承运商", Code: "C"})
	ch, _ := svc.CreateChannel(&CreateChannelInput{
		ProviderID: prov.ID, Name: "标准渠道",
		VolumetricDivisor: intPtr(6000),
	})
	zone, _ := svc.CreateZone(&CreateZoneInput{ChannelID: ch.ID, CountryCode: "RU"})

	fixedFee := 10.0
	perKg := 5.0
	svc.CreateRule(&CreateQuoteRuleInput{
		ChannelID: ch.ID, ZoneID: &zone.ID,
		RuleType: "fixed_plus_per_kg", FixedFee: &fixedFee,
		PerKgPrice: &perKg, Status: statusPtr(1),
	})

	// 3 units, each 2kg, 10x10x10cm
	// total actual weight = 6kg
	// base = 10 + 6*5 = 40
	resp, _ := svc.Quote(&QuoteRequest{
		Mode: "manual", Quantity: 3, DestinationCountry: "RU",
		ManualWeightKg: floatPtr(2.0), ManualLengthCM: floatPtr(10),
		ManualWidthCM: floatPtr(10), ManualHeightCM: floatPtr(10),
	})
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].ActualWeightKg != 6.0 {
		t.Errorf("expected actual weight 6.0 (3*2), got %.4f", resp.Results[0].ActualWeightKg)
	}
	if resp.Results[0].BaseShippingFee != 40.0 {
		t.Errorf("expected base fee 40.0, got %.2f", resp.Results[0].BaseShippingFee)
	}
}

func TestQuote_ZeroQuantityDefaultsToOne(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	prov, _ := svc.CreateProvider(&CreateProviderInput{Name: "承运商", Code: "C"})
	ch, _ := svc.CreateChannel(&CreateChannelInput{
		ProviderID: prov.ID, Name: "渠道",
		VolumetricDivisor: intPtr(6000),
	})
	zone, _ := svc.CreateZone(&CreateZoneInput{ChannelID: ch.ID, CountryCode: "RU"})

	fixed := 5.0
	perKg := 3.0
	svc.CreateRule(&CreateQuoteRuleInput{
		ChannelID: ch.ID, ZoneID: &zone.ID,
		RuleType: "fixed_plus_per_kg", FixedFee: &fixed,
		PerKgPrice: &perKg, Status: statusPtr(1),
	})

	// Quantity=0 should default to 1
	resp, _ := svc.Quote(&QuoteRequest{
		Mode: "manual", Quantity: 0, DestinationCountry: "RU",
		ManualWeightKg: floatPtr(1.0), ManualLengthCM: floatPtr(10),
		ManualWidthCM: floatPtr(10), ManualHeightCM: floatPtr(10),
	})
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].ActualWeightKg != 1.0 {
		t.Errorf("expected actual weight 1.0 (default 1 qty), got %.4f", resp.Results[0].ActualWeightKg)
	}
	// base = 5 + 1*3 = 8
	if resp.Results[0].BaseShippingFee != 8.0 {
		t.Errorf("expected base fee 8.0, got %.2f", resp.Results[0].BaseShippingFee)
	}
}

func TestProvider_DefaultStatusOnCreate(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	p, err := svc.CreateProvider(&CreateProviderInput{Name: "测试承运商", Code: "TEST"})
	if err != nil {
		t.Fatalf("CreateProvider failed: %v", err)
	}
	if p.Status != 1 {
		t.Errorf("expected default status 1, got %d", p.Status)
	}

	// Verify list returns it
	items, total, _ := svc.ListProviders(&common.Pagination{Page: 1, Size: 10}, "")
	if total != 1 || len(items) != 1 {
		t.Errorf("expected 1 provider, got %d, %d", total, len(items))
	}
}

func intPtr(v int) *int       { return &v }
func floatPtr(v float64) *float64 { return &v }

// ---------- Carrier API Handler Tests ----------

func TestListCarriers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	h := &Handler{service: nil} // ListCarriers doesn't touch the service

	h.ListCarriers(c)

	var resp struct {
		Code int            `json:"code"`
		Data []CarrierInfo  `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 carrier, got %d", len(resp.Data))
	}
	if resp.Data[0].Code != "mock_carrier" {
		t.Errorf("expected mock_carrier, got %s", resp.Data[0].Code)
	}
	if resp.Data[0].Status != "sandbox" {
		t.Errorf("expected status sandbox, got %s", resp.Data[0].Status)
	}
}

func TestCarrierQuote_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{{Key: "code", Value: "nonexistent_carrier"}}
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h := &Handler{service: nil}
	h.CarrierQuote(c)

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCarrierQuote_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{{Key: "code", Value: "mock_carrier"}}
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{
		"origin_country": "CN", "destination_country": "RU",
		"weight_kg": 1.0
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h := &Handler{service: nil}
	h.CarrierQuote(c)

	var resp struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if resp.Data == nil {
		t.Fatal("expected data, got nil")
	}
	fee, ok := resp.Data["total_fee"].(float64)
	if !ok || fee != 15.0 {
		t.Errorf("expected total_fee 15.0, got %v", resp.Data["total_fee"])
	}
}

func TestCarrierQuote_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{{Key: "code", Value: "mock_carrier"}}
	// Invalid JSON body
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{invalid}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h := &Handler{service: nil}
	h.CarrierQuote(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON, got %d", w.Code)
	}
}
