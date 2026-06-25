// Package knowledge provides the Knowledge Engine — a semantic query layer that
// translates agent questions into structured data lookups across business
// domains (inventory, order, settlement, SKU, supplier, shipping, platform,
// aftersales).
//
// This is v1 (keyword+regex, no vector store).
package knowledge

import "time"

// KnowledgeQuery is a natural-language query to the knowledge engine.
type KnowledgeQuery struct {
	AgentID  string                 `json:"agent_id"`
	Question string                 `json:"question"`          // "SKU 123的库存状态怎么样"
	Context  map[string]interface{} `json:"context,omitempty"`  // optional params: sku_id, order_id, etc.
	MaxAge   time.Duration          `json:"max_age"`            // maximum acceptable age of the data
}

// KnowledgeResponse is the engine's synthesized answer.
type KnowledgeResponse struct {
	Answer      string              `json:"answer"`
	Confidence  float64             `json:"confidence"`
	DataSources []DataSource        `json:"data_sources"`  // which sources were used
	Freshness   map[string]time.Time `json:"freshness"`    // per-source last sync time
	Inferences  []string            `json:"inferences"`    // engine reasoning steps
}

// DataSource describes a single data source used in the response.
type DataSource struct {
	Type      string    `json:"type"`       // "inventory" | "order" | "supplier" | "settlement"
	ID        string    `json:"id"`         // source identifier
	Table     string    `json:"table"`      // DB table name
	LastSync  time.Time `json:"last_sync"`  // when data was last refreshed
	Freshness string    `json:"freshness"`  // "real-time" | "t+1" | "stale"
}
