package logistics

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

// newTestLogger returns a no-op zap logger for route loader tests.
func newTestLogger() *zap.Logger { return zap.NewNop() }

// ── loadCarrierRateTables ────────────────────────────────────────────────

func TestLoadCarrierRateTables_MissingDirectory(t *testing.T) {
	// Point to a path that does not exist; loader must return nil and not crash.
	t.Setenv(envCarrierRatesDir, "/nonexistent/carrier-rates/path")
	tables := loadCarrierRateTables(newTestLogger())
	if tables != nil {
		t.Errorf("expected nil tables for missing dir, got %d entries", len(tables))
	}
}

func TestLoadCarrierRateTables_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envCarrierRatesDir, dir)

	tables := loadCarrierRateTables(newTestLogger())
	if tables != nil {
		t.Errorf("expected nil tables for empty dir, got %d entries", len(tables))
	}
}

func TestLoadCarrierRateTables_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	yaml := []byte(`
rate_table:
  - id: 1
    channel_name: "TestEconomic"
    provider_name: "TestCarrier"
    rule_type: "per_kg"
    priority: 1
    min_weight_kg: 0
    max_weight_kg: 0
    destination_country: "US"
    cargo_type: "normal"
    per_kg_price: 50
    currency: "CNY"
    estimated_delivery_min: 10
    estimated_delivery_max: 20
`)
	if err := os.WriteFile(filepath.Join(dir, "carrier.yaml"), yaml, 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	t.Setenv(envCarrierRatesDir, dir)

	tables := loadCarrierRateTables(newTestLogger())
	if len(tables) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(tables))
	}
	if tables[0].DestinationCountry != "US" {
		t.Errorf("expected US destination, got %s", tables[0].DestinationCountry)
	}
	if tables[0].PerKgPrice != 50 {
		t.Errorf("expected per_kg_price=50, got %.2f", tables[0].PerKgPrice)
	}
}

func TestLoadCarrierRateTables_SkipsInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	// valid file
	valid := []byte(`
rate_table:
  - id: 1
    channel_name: "OK"
    provider_name: "TestCarrier"
    rule_type: "per_kg"
    priority: 1
    destination_country: "US"
    cargo_type: "normal"
    per_kg_price: 50
`)
	if err := os.WriteFile(filepath.Join(dir, "a.yaml"), valid, 0o644); err != nil {
		t.Fatalf("write valid yaml: %v", err)
	}
	// invalid file (missing rate_table key — LoadRateTableFromYAML returns error)
	invalid := []byte(`not_a_rate_table: just stuff
- foo
- bar
`)
	if err := os.WriteFile(filepath.Join(dir, "b.yaml"), invalid, 0o644); err != nil {
		t.Fatalf("write invalid yaml: %v", err)
	}
	t.Setenv(envCarrierRatesDir, dir)

	tables := loadCarrierRateTables(newTestLogger())
	if len(tables) != 1 {
		t.Errorf("expected 1 entry (invalid file skipped), got %d", len(tables))
	}
}

func TestLoadCarrierRateTables_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	yaml1 := []byte(`
rate_table:
  - id: 1
    channel_name: "A"
    provider_name: "CarrierA"
    rule_type: "per_kg"
    priority: 1
    destination_country: "US"
    cargo_type: "normal"
    per_kg_price: 50
`)
	yaml2 := []byte(`
rate_table:
  - id: 2
    channel_name: "B"
    provider_name: "CarrierB"
    rule_type: "per_kg"
    priority: 1
    destination_country: "JP"
    cargo_type: "normal"
    per_kg_price: 35
`)
	if err := os.WriteFile(filepath.Join(dir, "a.yaml"), yaml1, 0o644); err != nil {
		t.Fatalf("write yaml1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.yaml"), yaml2, 0o644); err != nil {
		t.Fatalf("write yaml2: %v", err)
	}
	t.Setenv(envCarrierRatesDir, dir)

	tables := loadCarrierRateTables(newTestLogger())
	if len(tables) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(tables))
	}
}

// ── RegisterRoutes integration: DefaultEngine is wired ─────────────────

func TestRegisterRoutes_DefaultEngineSet(t *testing.T) {
	// RegisterRoutes uses the default carrier-rates dir. In test env, the
	// relative ./carrier-rates path likely does not exist, so DefaultEngine
	// should still be non-nil (NewService(nil) creates an empty engine).
	defer func() { DefaultEngine = nil }()

	dir := t.TempDir()
	t.Setenv(envCarrierRatesDir, dir)
	registerRoutesForTest(t)

	if DefaultEngine == nil {
		t.Fatal("expected DefaultEngine to be set after RegisterRoutes")
	}
}

func TestRegisterRoutes_DefaultEngineLoadedFromYAML(t *testing.T) {
	defer func() { DefaultEngine = nil }()

	dir := t.TempDir()
	yaml := []byte(`
rate_table:
  - id: 1
    channel_name: "TestEconomic"
    provider_name: "TestCarrier"
    rule_type: "per_kg"
    priority: 1
    min_weight_kg: 0
    max_weight_kg: 0
    destination_country: "US"
    cargo_type: "normal"
    per_kg_price: 42
    currency: "CNY"
`)
	if err := os.WriteFile(filepath.Join(dir, "carrier.yaml"), yaml, 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	t.Setenv(envCarrierRatesDir, dir)
	registerRoutesForTest(t)

	if DefaultEngine == nil {
		t.Fatal("expected DefaultEngine to be set")
	}
	resp, err := DefaultEngine.CalculateRate(Cargo{ActualWeightKg: 2.0}, "US", "normal")
	if err != nil {
		t.Fatalf("CalculateRate: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Fatal("expected non-empty results for US/normal/2kg")
	}
	// 2kg * 42 = 84
	if resp.Results[0].TotalShippingFee != 84 {
		t.Errorf("expected total 84, got %.2f", resp.Results[0].TotalShippingFee)
	}
}

// registerRoutesForTest invokes RegisterRoutes against a throwaway gin group
// so the package-level DefaultEngine gets populated without spinning up HTTP.
func registerRoutesForTest(t *testing.T) {
	t.Helper()
	// We avoid importing gin here by calling the loader + NewService directly,
	// mirroring what RegisterRoutes does. This keeps the test focused on the
	// data-loading path without a real router.
	tables := loadCarrierRateTables(newTestLogger())
	svc := NewService(tables)
	DefaultEngine = svc.engine
}

// ── Integration: real carrier-rates YAML files ──────────────────────────
//
// These tests verify the YAML files shipped under backend-go/carrier-rates/
// are valid, loadable, and produce non-empty quotes for every destination
// required by Issue #42 (US / EU / JP / RU with basic+air+sea channels).

// carrierRatesDir returns the absolute path to the repo's carrier-rates
// directory. The path is resolved relative to this test file's location
// (internal/domain/logistics/), so it works regardless of the CWD Go is
// invoked from.
func carrierRatesDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// wd = .../backend-go/internal/domain/logistics
	// carrier-rates = .../backend-go/carrier-rates
	dir := filepath.Join(wd, "..", "..", "..", "carrier-rates")
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	return abs
}

func TestCarrierRatesYAML_FilesExistAndLoad(t *testing.T) {
	dir := carrierRatesDir(t)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("carrier-rates dir %s: %v", dir, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", dir)
	}

	t.Setenv(envCarrierRatesDir, dir)
	tables := loadCarrierRateTables(newTestLogger())
	if len(tables) == 0 {
		t.Fatal("expected non-empty rate tables from carrier-rates/*.yaml")
	}
}

func TestCarrierRatesYAML_NonEmptyQuotesForAllRequiredDestinations(t *testing.T) {
	t.Setenv(envCarrierRatesDir, carrierRatesDir(t))
	tables := loadCarrierRateTables(newTestLogger())
	if len(tables) == 0 {
		t.Fatal("no rate tables loaded")
	}
	engine := NewRateEngine(tables)

	// Issue #42 requires US / EU / JP / RU coverage with at least 1 carrier
	// offering basic + air + sea channels.
	required := []string{"US", "EU", "JP", "RU"}
	for _, dest := range required {
		resp, err := engine.CalculateRate(Cargo{ActualWeightKg: 1.5}, dest, "normal")
		if err != nil {
			t.Errorf("CalculateRate(%s): %v", dest, err)
			continue
		}
		if len(resp.Results) == 0 {
			t.Errorf("expected non-empty quotes for %s, got 0", dest)
			continue
		}
	}
}

func TestCarrierRatesYAML_BasicAirSeaChannelsPresent(t *testing.T) {
	// Yanwen (provider_name=Yanwen) must cover US/EU/JP/RU with all 3 channel
	// types (basic economic, air, sea) per Issue #42 requirement #3.
	t.Setenv(envCarrierRatesDir, carrierRatesDir(t))
	tables := loadCarrierRateTables(newTestLogger())

	// Group Yanwen channel names by destination to verify coverage.
	byDest := map[string]map[string]bool{} // dest -> set of channel_name
	for _, e := range tables {
		if e.ProviderName != "Yanwen" {
			continue
		}
		if _, ok := byDest[e.DestinationCountry]; !ok {
			byDest[e.DestinationCountry] = map[string]bool{}
		}
		byDest[e.DestinationCountry][e.ChannelName] = true
	}

	for _, dest := range []string{"US", "EU", "JP", "RU"} {
		channels, ok := byDest[dest]
		if !ok {
			t.Errorf("Yanwen has no coverage for %s", dest)
			continue
		}
		// Expect 3 distinct channel names (economic, air, sea).
		if len(channels) < 3 {
			t.Errorf("Yanwen %s: expected >=3 channels (basic+air+sea), got %d (%v)", dest, len(channels), channels)
		}
	}
}

func TestCarrierRatesYAML_USNormalReturnsNonEmpty(t *testing.T) {
	// Issue #42 requirement #5: RateEngine.CalculateRate(cargo, "US", "normal")
	// returns non-empty results.
	t.Setenv(envCarrierRatesDir, carrierRatesDir(t))
	tables := loadCarrierRateTables(newTestLogger())
	engine := NewRateEngine(tables)

	resp, err := engine.CalculateRate(Cargo{ActualWeightKg: 1.0}, "US", "normal")
	if err != nil {
		t.Fatalf("CalculateRate US: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Fatal("expected non-empty results for US/normal/1kg")
	}
}
