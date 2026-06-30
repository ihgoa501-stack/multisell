// Package impl provides concrete agent implementations.
//
// WarehouseCustomsAgent implements G2 Warehouse & Customs business logic ported
// from backend/app/agent/agents/warehouse_customs.py (Python FastAPI codebase).
//
// Design docs: docs/aiagent/final-integrated-solution.md section 5.2
//   - HS code suggestion (12 rule mapping table)
//   - Customs clearance document checklist
//   - Warehouse strategy recommendation based on sales volume and destination
//   - Tariff estimation
package impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

// ---------- Context field names ----------

var customsRequiredFields = []string{"product_name", "destination_country", "cargo_type"}

// ---------- HS Code database ----------

var hsCodeDB = map[string]string{
	"electronics_US": "8471.30.0100",
	"electronics_EU": "8471.30.00",
	"electronics_JP": "8471.30.000",
	"clothing_US":    "6204.62.3030",
	"clothing_EU":    "6204.62.00",
	"food_US":        "2008.19.9090",
	"food_EU":        "2008.19.00",
	"cosmetics_US":   "3304.99.0000",
	"cosmetics_EU":   "3304.99.00",
	"baby_US":        "3924.10.4000",
	"toys_US":        "9503.00.0073",
	"furniture_US":   "9403.60.8081",
}

// hsCodeKey builds a lookup key for the HS code database.
func hsCodeKey(cargo, country string) string {
	return strings.ToLower(cargo) + "_" + strings.ToUpper(country)
}

// decisionPointToTool maps the legacy decision point names used in the
// Decide interface to the tool names registered in the ToolRegistry.
// This enables the ToolRegistry-based dispatch path while keeping the
// existing public interface unchanged.
var decisionPointToTool = map[string]string{
	"customs_clearance": "customs_declare",
	"warehouse_advice":  "warehouse_routing",
}

// ---------- WarehouseCustomsAgent ----------

// WarehouseCustomsAgent implements G2 Warehouse & Customs logic.
//
// Decision points:
//   - "customs_clearance" — HS code lookup, tariff estimate, document checklist
//   - "warehouse_advice" — warehouse strategy recommendation
//
// When a ToolRegistry is configured via SetToolRegistry, the agent delegates
// through it, gaining hook-based middleware, circuit-breaker protection, and
// LLM function-calling discoverability.
type WarehouseCustomsAgent struct {
	registry *toolregistry.ToolRegistry
}

// NewWarehouseCustomsAgent creates a new WarehouseCustomsAgent.
// Call SetToolRegistry to enable ToolRegistry-based dispatch.
func NewWarehouseCustomsAgent() *WarehouseCustomsAgent {
	return &WarehouseCustomsAgent{}
}

// SetToolRegistry configures the ToolRegistry for this agent and registers its
// decision points as discoverable tools. After this call, Decide() will delegate
// through the ToolRegistry.
func (a *WarehouseCustomsAgent) SetToolRegistry(registry *toolregistry.ToolRegistry) {
	if registry == nil {
		return
	}
	a.registry = registry
	a.registerTools()
}

// registerTools registers the agent's decision points as tools in the ToolRegistry.
func (a *WarehouseCustomsAgent) registerTools() {
	if a.registry == nil {
		return
	}

	a.registry.Register(&toolregistry.Tool{
		Name:        "customs_declare",
		Version:     "1.0.0",
		Description: "报关单校验——HS编码建议、关税估算、必要单据清单",
		Squad:       "fulfillment",
		Parameters: &toolregistry.Schema{
			Type:        "object",
			Description: "报关校验参数",
			Properties: map[string]*toolregistry.Schema{
				"product_name":        {Type: "string", Description: "商品名称"},
				"destination_country": {Type: "string", Description: "目的国代码（如 US/EU/JP）"},
				"cargo_type":          {Type: "string", Description: "货物类型（如 electronics/clothing/food）"},
				"declared_value":      {Type: "number", Description: "申报价值（美元，可选）"},
				"weight_kg":           {Type: "number", Description: "重量（千克，可选）"},
			},
			Required: []string{"product_name", "destination_country", "cargo_type"},
		},
		Returns: &toolregistry.Schema{
			Type:        "object",
			Description: "报关结果，包含HS编码、关税估算、必要单据清单和置信度",
		},
		RequiredPermissions: []string{"fulfillment:read:customs"},
		RiskLevel:           toolregistry.RiskHigh,
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			output, confidence, riskLevel, _ := a.clearance(input)
			return map[string]interface{}{
				"output":     output,
				"confidence": confidence,
				"risk_level": riskLevel,
			}, nil
		},
	})

	a.registry.Register(&toolregistry.Tool{
		Name:        "warehouse_routing",
		Version:     "1.0.0",
		Description: "仓库路由——根据销量和目的地推荐仓库发货策略（FBA/海外仓/国内直发）",
		Squad:       "fulfillment",
		Parameters: &toolregistry.Schema{
			Type:        "object",
			Description: "仓库路由参数",
			Properties: map[string]*toolregistry.Schema{
				"destination_country":  {Type: "string", Description: "目的国代码（如 US/DE/FR/IT/ES/UK）"},
				"monthly_sales_volume": {Type: "number", Description: "月销量"},
			},
			Required: []string{"destination_country", "monthly_sales_volume"},
		},
		Returns: &toolregistry.Schema{
			Type:        "object",
			Description: "仓库路由建议，包含发货策略和详细说明",
		},
		RequiredPermissions: []string{"fulfillment:read:warehouse_routing"},
		RiskLevel:           toolregistry.RiskLow,
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			output, confidence, riskLevel, _ := a.warehouse(input)
			return map[string]interface{}{
				"output":     output,
				"confidence": confidence,
				"risk_level": riskLevel,
			}, nil
		},
	})
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// When a ToolRegistry is configured, the agent delegates via registry.Call(),
// which applies hooks, circuit breakers, and schema validation.
// Without a ToolRegistry, the agent falls back to direct internal dispatch.
//
// Supported decision points:
//   - "customs_clearance"
//   - "warehouse_advice"
func (a *WarehouseCustomsAgent) Decide(ctx context.Context, decisionPoint string, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if a.registry != nil {
		return a.decideViaRegistry(ctx, decisionPoint, params)
	}
	return a.decideDirect(decisionPoint, params)
}

// decideViaRegistry delegates the decision through the ToolRegistry's hook chain,
// mapping the decision point to its corresponding registered tool name.
func (a *WarehouseCustomsAgent) decideViaRegistry(callCtx context.Context, decisionPoint string, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	toolName, ok := decisionPointToTool[decisionPoint]
	if !ok {
		return a.decideDirect(decisionPoint, params)
	}

	result, callErr := a.registry.Call(callCtx, toolName, params)
	if callErr != nil {
		// Tool not found or hook rejected; fall through to the direct dispatch table.
		return a.decideDirect(decisionPoint, params)
	}

	// Unwrap the tool output envelope.
	env, ok := result.(map[string]interface{})
	if !ok {
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          "unexpected tool output format",
		}, 0.0, "low", nil
	}

	if out, ok := env["output"].(map[string]interface{}); ok {
		output = out
	}
	if conf, ok := env["confidence"].(float64); ok {
		confidence = conf
	}
	if risk, ok := env["risk_level"].(string); ok {
		riskLevel = risk
	}
	return output, confidence, riskLevel, nil
}

// decideDirect is the legacy direct-dispatch path (no ToolRegistry configured).
func (a *WarehouseCustomsAgent) decideDirect(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "customs_clearance":
		return a.clearance(ctx)
	case "warehouse_advice":
		return a.warehouse(ctx)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}

// ---------- Decision point: customs_clearance ----------

// clearance provides HS code suggestion, tariff estimation, and required
// document checklist based on product, destination, and cargo type.
//
// Required context fields: product_name, destination_country, cargo_type
// Optional context fields: declared_value, weight_kg
func (a *WarehouseCustomsAgent) clearance(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	if missing := missingFields(ctx, customsRequiredFields); len(missing) > 0 {
		return insufficientData("customs_clearance", missing), 0.0, "low", nil
	}

	name := safeString(ctx["product_name"], "")
	country := strings.ToUpper(strings.TrimSpace(safeString(ctx["destination_country"], "")))
	cargo := strings.ToLower(strings.TrimSpace(safeString(ctx["cargo_type"], "")))
	value := safeFloat(ctx["declared_value"], 0)

	// Lookup HS code.
	hsCode, ok := hsCodeDB[hsCodeKey(cargo, country)]
	if !ok {
		hsCode = "需要人工归类"
	}

	// Tariff estimation.
	dutyFree := value < 800 && country == "US"
	var estimatedDuty float64
	if dutyFree {
		estimatedDuty = 0
	} else if value > 0 {
		estimatedDuty = round2(value * 0.05)
	}

	// Document checklist.
	docs := []string{"Commercial Invoice", "Packing List"}
	switch cargo {
	case "electronics", "baby", "toys":
		docs = append(docs, "Safety Certification (FCC/CE/CPC)")
	case "food":
		if country == "US" {
			docs = append(docs, "FDA Prior Notice")
		} else {
			docs = append(docs, "Health Certificate")
		}
	}
	if value > 2500 {
		docs = append(docs, "Customs Bond / Formal Entry")
	}

	// Confidence: lower if HS code needs manual classification.
	conf := 0.85
	if hsCode == "需要人工归类" {
		conf = 0.60
	}

	output = map[string]interface{}{
		"product":               name,
		"destination":           country,
		"hs_code":               hsCode,
		"required_documents":    docs,
		"estimated_duty":        estimatedDuty,
		"duty_free":             dutyFree,
		"value_threshold_note":  thresholdNote(value),
		"confidence":            conf,
	}
	return output, conf, "low", nil
}

// thresholdNote returns a human-readable note about the declared value threshold.
func thresholdNote(value float64) string {
	if value > 2500 {
		return "> $2500 需正式报关"
	}
	return "≤ $2500 可简易报关"
}

// ---------- Decision point: warehouse_advice ----------

// warehouse recommends a warehouse strategy based on destination and sales volume.
//
// Required context fields: destination_country, monthly_sales_volume
func (a *WarehouseCustomsAgent) warehouse(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	country := strings.ToUpper(strings.TrimSpace(safeString(ctx["destination_country"], "")))
	salesVolume := safeFloat(ctx["monthly_sales_volume"], 0)

	var strategy, note string

	switch {
	case country == "US" && salesVolume > 500:
		strategy = "FBA + 海外仓双轨"
		note = "建议同时使用 FBA（Prime 流量）和第三方海外仓（降低成本风险）"
	case country == "US" && salesVolume > 100:
		strategy = "FBA 优先"
		note = "建议以 FBA 为主，搭配少量自发货测试市场"
	case country == "US":
		strategy = "国内直发"
		note = "销量较小，建议国内直发或使用 Amazon 自配送"
	case country == "DE" || country == "FR" || country == "IT" || country == "ES" || country == "UK":
		strategy = "欧洲海外仓"
		note = "建议使用欧洲海外仓（FBA 欧洲或第三方），注意 VAT 注册要求"
	default:
		strategy = "国内直发"
		note = "非主流市场建议国内直发，根据订单增长再考虑海外仓"
	}

	output = map[string]interface{}{
		"strategy":                  strategy,
		"note":                      note,
		"destination":               country,
		"estimated_monthly_volume":  int(salesVolume),
		"confidence":                0.85,
	}
	return output, 0.85, "low", nil
}
