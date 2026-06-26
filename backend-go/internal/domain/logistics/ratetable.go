package logistics

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// rateTableYAML is the top-level YAML document structure.
type rateTableYAML struct {
	Table []RateTableEntry `yaml:"rate_table"`
}

// LoadRateTableFromYAML parses a YAML rate table into []RateTableEntry.
// The YAML must contain a top-level "rate_table" key whose value is an array
// of rate table entries. Returns an error on parse failure or if the table is empty.
func LoadRateTableFromYAML(data []byte) ([]RateTableEntry, error) {
	var doc rateTableYAML
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("logistics: parse rate table YAML: %w", err)
	}
	if len(doc.Table) == 0 {
		return nil, fmt.Errorf("logistics: rate table is empty")
	}
	return doc.Table, nil
}

// SampleRateTableYAML is example rate table data covering two carriers
// (Yanwen and Cainiao), two destinations (RU and KZ), and two pricing modes
// (first_additional and per_kg). It is suitable for testing and reference.
const SampleRateTableYAML = `
rate_table:
  # ── Yanwen (燕文) ──────────────────────────────────────
  - id: 1
    channel_name: "燕文经济线"
    provider_name: "Yanwen"
    rule_type: "first_additional"
    priority: 1
    min_weight_kg: 0
    max_weight_kg: 2
    destination_country: "RU"
    cargo_type: "normal"
    first_kg: 1
    first_price: 60
    additional_kg: 0.5
    additional_price: 30
    minimum_charge: 60
    fuel_surcharge_pct: 5
    surcharge_fixed: 2
    currency: "CNY"
    estimated_delivery_min: 15
    estimated_delivery_max: 25

  - id: 2
    channel_name: "燕文标准线"
    provider_name: "Yanwen"
    rule_type: "per_kg"
    priority: 2
    min_weight_kg: 0
    max_weight_kg: 30
    destination_country: "KZ"
    cargo_type: "normal"
    per_kg_price: 65
    minimum_charge: 65
    fuel_surcharge_pct: 5
    surcharge_fixed: 0
    currency: "CNY"
    estimated_delivery_min: 12
    estimated_delivery_max: 20

  # ── Cainiao (菜鸟) ─────────────────────────────────────
  - id: 3
    channel_name: "菜鸟超级经济"
    provider_name: "Cainiao"
    rule_type: "per_kg"
    priority: 1
    min_weight_kg: 0
    max_weight_kg: 10
    destination_country: "RU"
    cargo_type: "normal"
    per_kg_price: 80
    fuel_surcharge_pct: 0
    surcharge_fixed: 5
    currency: "CNY"
    estimated_delivery_min: 20
    estimated_delivery_max: 35

  - id: 4
    channel_name: "菜鸟标准线"
    provider_name: "Cainiao"
    rule_type: "first_additional"
    priority: 1
    min_weight_kg: 0
    max_weight_kg: 5
    destination_country: "KZ"
    cargo_type: "normal"
    first_kg: 1
    first_price: 70
    additional_kg: 1
    additional_price: 50
    fuel_surcharge_pct: 10
    surcharge_fixed: 0
    currency: "CNY"
    estimated_delivery_min: 18
    estimated_delivery_max: 28

  # ── Yanwen tiered pricing (higher weight bracket) ──────
  - id: 5
    channel_name: "燕文经济线"
    provider_name: "Yanwen"
    rule_type: "tiered"
    priority: 3
    min_weight_kg: 2
    max_weight_kg: 10
    destination_country: "RU"
    cargo_type: "normal"
    tiers:
      - { min: 2, max: 3, price: 140 }
      - { min: 3, max: 5, price: 210 }
      - { min: 5, max: 0, price: 340 }
    minimum_charge: 0
    fuel_surcharge_pct: 5
    surcharge_fixed: 0
    currency: "CNY"
    estimated_delivery_min: 12
    estimated_delivery_max: 20
`
