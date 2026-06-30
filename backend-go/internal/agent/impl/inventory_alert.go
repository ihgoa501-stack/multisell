// Package impl provides concrete agent implementations.
//
// InventoryAlertAgent implements A5 Inventory Alert business logic ported from
// backend/app/agent/agents/inventory_alert.py (Python FastAPI codebase).
//
// Design docs: docs/AI_AGENT_FEASIBLE_DEVELOPMENT_SPEC.md section 7.1.2
//   - Input: SKU code, sellable/locked/in-transit stock, multi-period sales,
//     lead time, MOQ, safety stock days
//   - Output: stock status (red/yellow/green), days of cover, suggested
//     replenish quantity, suggested logistics, risk reason, actions
//   - Insufficient data returns insufficient_data status
package impl

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
	"github.com/lingmirror/backend-go/internal/domain/inventory"
	"github.com/lingmirror/backend-go/internal/domain/sku"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	stockAlertRequiredFields = []string{
		"sku_code", "sellable_stock", "sales_7d",
		"lead_time_days", "safety_stock_days",
	}
	replenishRequiredFields = []string{
		"sku_code", "sellable_stock", "sales_30d",
		"lead_time_days", "moq",
	}
)

// InventoryAlertAgent implements A5 Inventory Alert logic.
//
// Decision points:
//   - "stock_alert" — three-tier stock status with replenish suggestion
//   - "replenishment_plan" — suggested reorder quantity calculation
//   - "logistics_choice" — logistics channel recommendation by urgency
//
// When a ToolRegistry is configured via SetToolRegistry, the agent delegates
// through it, gaining hook-based middleware, circuit-breaker protection, and
// LLM function-calling discoverability.
type InventoryAlertAgent struct {
	db       *gorm.DB
	logger   *zap.Logger
	registry *toolregistry.ToolRegistry
}

// NewInventoryAlertAgent creates a new InventoryAlertAgent.
// Call SetToolRegistry to enable ToolRegistry-based dispatch.
func NewInventoryAlertAgent(db *gorm.DB, logger *zap.Logger) *InventoryAlertAgent {
	return &InventoryAlertAgent{db: db, logger: logger}
}

// SetToolRegistry configures the ToolRegistry for this agent and registers its
// decision points as discoverable tools. After this call, Decide() will delegate
// through the ToolRegistry.
func (a *InventoryAlertAgent) SetToolRegistry(registry *toolregistry.ToolRegistry) {
	if registry == nil {
		return
	}
	a.registry = registry
	a.registerTools()
}

// registerTools registers the agent's decision points as tools in the ToolRegistry.
func (a *InventoryAlertAgent) registerTools() {
	if a.registry == nil {
		return
	}

	a.registry.Register(&toolregistry.Tool{
		Name:        "stock_alert",
		Version:     "1.0.0",
		Description: "库存预警——基于可售天数与提前期/安全库存对比，产出红/黄/绿三级库存状态及补货建议",
		Squad:       "logistics",
		Parameters: &toolregistry.Schema{
			Type:        "object",
			Description: "库存预警参数",
			Properties: map[string]*toolregistry.Schema{
				"sku_code":          {Type: "string", Description: "SKU编码"},
				"sellable_stock":    {Type: "number", Description: "可售库存数量"},
				"locked_stock":      {Type: "number", Description: "锁定库存数量"},
				"in_transit_stock":  {Type: "number", Description: "在途库存数量"},
				"sales_7d":          {Type: "number", Description: "近7天销量"},
				"sales_14d":         {Type: "number", Description: "近14天销量"},
				"sales_30d":         {Type: "number", Description: "近30天销量"},
				"lead_time_days":    {Type: "number", Description: "采购提前期（天）"},
				"safety_stock_days": {Type: "number", Description: "安全库存天数"},
				"moq":               {Type: "number", Description: "最小起订量"},
			},
			Required: []string{"sku_code", "sellable_stock", "sales_7d", "lead_time_days", "safety_stock_days"},
		},
		Returns: &toolregistry.Schema{
			Type:        "object",
			Description: "库存预警结果，包含状态（red/yellow/green）、可售天数、风险原因和补货建议",
		},
		RequiredPermissions: []string{"inventory:read:alert"},
		RiskLevel:           toolregistry.RiskLow,
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			output, confidence, riskLevel, _ := a.checkStockAlert("stock_alert", input)
			return map[string]interface{}{
				"output":     output,
				"confidence": confidence,
				"risk_level": riskLevel,
			}, nil
		},
	})

	a.registry.Register(&toolregistry.Tool{
		Name:        "replenishment_plan",
		Version:     "1.0.0",
		Description: "补货计划——根据销量、提前期和MOQ计算建议补货量，标注紧急程度",
		Squad:       "logistics",
		Parameters: &toolregistry.Schema{
			Type:        "object",
			Description: "补货计划参数",
			Properties: map[string]*toolregistry.Schema{
				"sku_code":          {Type: "string", Description: "SKU编码"},
				"sellable_stock":    {Type: "number", Description: "可售库存数量"},
				"in_transit_stock":  {Type: "number", Description: "在途库存数量"},
				"sales_7d":          {Type: "number", Description: "近7天销量"},
				"sales_14d":         {Type: "number", Description: "近14天销量"},
				"sales_30d":         {Type: "number", Description: "近30天销量"},
				"lead_time_days":    {Type: "number", Description: "采购提前期（天）"},
				"safety_stock_days": {Type: "number", Description: "安全库存天数"},
				"moq":               {Type: "number", Description: "最小起订量"},
			},
			Required: []string{"sku_code", "sellable_stock", "sales_30d", "lead_time_days", "moq"},
		},
		Returns: &toolregistry.Schema{
			Type:        "object",
			Description: "补货计算结果，包含目标库存、建议补货量和紧急程度",
		},
		RequiredPermissions: []string{"inventory:write:replenish"},
		RiskLevel:           toolregistry.RiskMedium,
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			output, confidence, riskLevel, _ := a.calculateReplenishment("replenishment_plan", input)
			return map[string]interface{}{
				"output":     output,
				"confidence": confidence,
				"risk_level": riskLevel,
			}, nil
		},
	})

	a.registry.Register(&toolregistry.Tool{
		Name:        "logistics_choice",
		Version:     "1.0.0",
		Description: "物流渠道推荐——根据库存紧急程度（红/黄/绿）和目的地推荐最优物流方案",
		Squad:       "logistics",
		Parameters: &toolregistry.Schema{
			Type:        "object",
			Description: "物流推荐参数",
			Properties: map[string]*toolregistry.Schema{
				"stock_status":   {Type: "string", Description: "库存状态（red/yellow/green）"},
				"sellable_days":  {Type: "number", Description: "可售天数"},
				"lead_time_days": {Type: "number", Description: "采购提前期（天）"},
				"cargo_value":    {Type: "number", Description: "货值"},
				"weight_kg":      {Type: "number", Description: "重量（kg）"},
				"destination":    {Type: "string", Description: "目的地（US/EU，默认US）"},
				"is_peak_season": {Type: "boolean", Description: "是否为旺季"},
			},
		},
		Returns: &toolregistry.Schema{
			Type:        "object",
			Description: "物流推荐结果，包含推荐方式、备选方案列表和置信度",
		},
		RequiredPermissions: []string{"logistics:read:route"},
		RiskLevel:           toolregistry.RiskLow,
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			output, confidence, riskLevel, _ := a.recommendLogistics("logistics_choice", input)
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
//   - "stock_alert"
//   - "replenishment_plan"
//   - "logistics_choice"
func (a *InventoryAlertAgent) Decide(ctx context.Context, decisionPoint string, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	a.fillSkuFromDB(params)

	if a.registry != nil {
		return a.decideViaRegistry(ctx, decisionPoint, params)
	}
	return a.decideDirect(decisionPoint, params)
}

// decideViaRegistry delegates the decision through the ToolRegistry's hook chain.
func (a *InventoryAlertAgent) decideViaRegistry(callCtx context.Context, decisionPoint string, params map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	result, callErr := a.registry.Call(callCtx, decisionPoint, params)
	if callErr != nil {
		// Tool not found or hook rejected; fall through to direct dispatch.
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
func (a *InventoryAlertAgent) decideDirect(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "stock_alert":
		return a.checkStockAlert(decisionPoint, ctx)
	case "replenishment_plan":
		return a.calculateReplenishment(decisionPoint, ctx)
	case "logistics_choice":
		return a.recommendLogistics(decisionPoint, ctx)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}

// fillSkuFromDB looks up the SKU and inventory records by code and fills
// missing context fields (mirrors Python AgentDataService.fill_sku_context).
func (a *InventoryAlertAgent) fillSkuFromDB(ctx map[string]interface{}) {
	if a.db == nil {
		return
	}
	skuCode := safeString(ctx["sku_code"])
	if skuCode == "" {
		return
	}
	var s sku.Sku
	if err := a.db.Where("code = ?", skuCode).First(&s).Error; err != nil {
		a.logger.Debug("sku db lookup failed",
			zap.String("sku_code", skuCode), zap.Error(err))
		return
	}
	if _, ok := ctx["sellable_stock"]; !ok {
		ctx["sellable_stock"] = float64(s.Stock)
	}
	if _, ok := ctx["locked_stock"]; !ok {
		ctx["locked_stock"] = float64(s.LockStock)
	}
	if _, ok := ctx["weight_kg"]; !ok {
		if w, exact := s.Weight.Float64(); exact {
			ctx["weight_kg"] = w
		}
	}
	var inv inventory.Inventory
	if err := a.db.Where("sku_id = ?", s.ID).First(&inv).Error; err != nil {
		a.logger.Debug("inventory db lookup failed",
			zap.Int64("sku_id", s.ID), zap.Error(err))
		return
	}
	if _, ok := ctx["sellable_stock"]; !ok {
		ctx["sellable_stock"] = float64(inv.Quantity)
	}
	if _, ok := ctx["safety_stock_days"]; !ok {
		ctx["safety_stock_days"] = float64(inv.SafetyStock)
	}
}

// checkStockAlert is the core three-tier stock alert logic.
func (a *InventoryAlertAgent) checkStockAlert(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	backfillLegacyFields(ctx)
	if missing := missingFields(ctx, stockAlertRequiredFields); len(missing) > 0 {
		return insufficientData(decisionPoint, missing), 0.0, "low", nil
	}
	skuCode := safeString(ctx["sku_code"])
	sellable := safeFloat(ctx["sellable_stock"])
	locked := safeFloat(ctx["locked_stock"])
	inTransit := safeFloat(ctx["in_transit_stock"])
	validStock := sellable + inTransit
	sales7d := safeFloat(ctx["sales_7d"])
	sales14d := safeFloat(ctx["sales_14d"])
	sales30d := safeFloat(ctx["sales_30d"])
	dailySales, salesSource := estimateDailySales(sales7d, sales14d, sales30d)
	leadTime := safeFloat(ctx["lead_time_days"], 30)
	safetyDays := safeFloat(ctx["safety_stock_days"], 14)
	moq := safeFloat(ctx["moq"])
	sellableDays := 999.0
	if dailySales > 0 {
		sellableDays = round1(validStock / dailySales)
	}
	redThreshold := leadTime * 0.5
	yellowThreshold := leadTime + safetyDays
	var stockStatus string
	var riskReason string
	var suggestedActions []string
	if sellableDays <= redThreshold {
		stockStatus = "red"; confidence = 0.95; riskLevel = "high"
		riskReason = fmt.Sprintf("可售天数(%.1f天)不足提前期(%.0f天)的一半，存在断货风险", sellableDays, leadTime)
		suggestedActions = []string{"紧急补货", "暂停广告投放", "通知采购部门", "考虑提价控量"}
	} else if sellableDays <= yellowThreshold {
		stockStatus = "yellow"; confidence = 0.88; riskLevel = "medium"
		riskReason = fmt.Sprintf("可售天数(%.1f天)小于提前期+安全库存天数(%.0f+%.0f=%.0f天)，建议尽快补货", sellableDays, leadTime, safetyDays, yellowThreshold)
		suggestedActions = []string{"安排补货", "关注在途到货时间", "适当降低广告预算"}
	} else {
		stockStatus = "green"; confidence = 0.85; riskLevel = "low"
		riskReason = "库存充足，暂无风险"
		suggestedActions = []string{"常规监控"}
	}
	suggestedReplenishQty := calcReplenishQty(dailySales, sellableDays, int(yellowThreshold), int(sellable), int(inTransit), int(moq))
	suggestedLogistics := pickLogistics(stockStatus, sellableDays, int(leadTime))
	output = map[string]interface{}{
		"stock_status": stockStatus, "decision_point": decisionPoint,
		"sku_code": skuCode, "sellable_days": sellableDays,
		"sellable_stock": sellable, "locked_stock": locked, "in_transit_stock": inTransit,
		"daily_sales_used": round1(dailySales), "daily_sales_source": salesSource,
		"lead_time_days": leadTime, "safety_stock_days": safetyDays, "moq": moq,
		"suggested_replenish_qty": suggestedReplenishQty, "suggested_logistics": suggestedLogistics,
		"risk_reason": riskReason, "suggested_actions": suggestedActions, "confidence": confidence,
	}
	return output, confidence, riskLevel, nil
}

// calculateReplenishment computes the suggested reorder quantity for a SKU.
func (a *InventoryAlertAgent) calculateReplenishment(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	backfillLegacyFields(ctx)
	if missing := missingFields(ctx, replenishRequiredFields); len(missing) > 0 {
		return insufficientData(decisionPoint, missing), 0.0, "low", nil
	}
	skuCode := safeString(ctx["sku_code"])
	sellable := safeFloat(ctx["sellable_stock"])
	inTransit := safeFloat(ctx["in_transit_stock"])
	leadTime := safeFloat(ctx["lead_time_days"], 30)
	moq := safeFloat(ctx["moq"], 100)
	safetyDaysInput := safeFloat(ctx["safety_stock_days"], 14)
	sales7d := safeFloat(ctx["sales_7d"])
	sales14d := safeFloat(ctx["sales_14d"])
	sales30d := safeFloat(ctx["sales_30d"])
	dailySales, salesSource := estimateDailySales(sales7d, sales14d, sales30d)
	if dailySales <= 0 {
		return insufficientData(decisionPoint, []string{"sales_7d/sales_30d"}), 0.0, "low", nil
	}
	targetDays := int(leadTime + safetyDaysInput)
	targetStock := int(dailySales * float64(targetDays))
	available := int(sellable + inTransit)
	replenishQty := targetStock - available
	if replenishQty < int(moq) {
		replenishQty = int(moq)
	}
	riskReason := ""
	urgency := "normal"
	leadSales := int(dailySales * leadTime)
	if replenishQty > 0 && available < leadSales {
		urgency = "urgent"
		riskReason = fmt.Sprintf("当前可用库存(%d)小于提前期(%.0f天)预估销量(%d)", available, leadTime, leadSales)
	}
	if urgency == "normal" {
		confidence = 0.90; riskLevel = "low"
	} else {
		confidence = 0.85; riskLevel = "medium"
	}
	output = map[string]interface{}{
		"sku_code": skuCode, "decision_point": decisionPoint,
		"sellable_stock": sellable, "in_transit_stock": inTransit, "available_stock": available,
		"daily_sales_used": round1(dailySales), "daily_sales_source": salesSource,
		"lead_time_days": leadTime, "safety_stock_days": safetyDaysInput,
		"target_stock": targetStock, "suggested_replenish_qty": replenishQty,
		"moq": moq, "urgency": urgency, "risk_reason": riskReason, "confidence": confidence,
	}
	return output, confidence, riskLevel, nil
}

// recommendLogistics generates logistics channel recommendations.
func (a *InventoryAlertAgent) recommendLogistics(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	stockStatus := safeString(ctx["stock_status"], "green")
	sellableDays := safeFloat(ctx["sellable_days"], 999)
	leadTime := safeFloat(ctx["lead_time_days"], 20)
	cargoValue := safeFloat(ctx["cargo_value"])
	weightKg := safeFloat(ctx["weight_kg"])
	destination := safeString(ctx["destination"], "US")
	isPeakSeason := false
	if v, ok := ctx["is_peak_season"]; ok {
		if b, ok := v.(bool); ok {
			isPeakSeason = b
		}
	}
	var options []map[string]interface{}
	suggestedLogistics := "海运"
	if stockStatus == "red" || sellableDays < leadTime*0.5 {
		suggestedLogistics = "空运/国际快递"
		if cargoValue > 50 {
			options = append(options, map[string]interface{}{
				"method": "air_express", "name": "空运/国际快递 (DHL/UPS)",
				"estimated_days": "3-7", "cost_estimate": "高", "suitability": "recommended",
			})
		}
		options = append(options, map[string]interface{}{
			"method": "air_freight", "name": "空运 + 海运快船并行",
			"estimated_days": "7-15", "cost_estimate": "中高", "suitability": "alternative",
		})
		confidence = 0.90; riskLevel = "medium"
	} else if stockStatus == "yellow" {
		suggestedLogistics = "快船"
		options = append(options, map[string]interface{}{
			"method": "express_sea", "name": "快船 (美森/以星)",
			"estimated_days": "10-15", "cost_estimate": "中", "suitability": "recommended",
		})
		options = append(options, map[string]interface{}{
			"method": "air_freight", "name": "空运 (备选)",
			"estimated_days": "7-12", "cost_estimate": "高", "suitability": "alternative",
		})
		confidence = 0.88; riskLevel = "medium"
	} else {
		seaDays := "15-20"
		if !strings.EqualFold(destination, "US") {
			seaDays = "25-35"
		}
		options = append(options, map[string]interface{}{
			"method": "sea_freight", "name": "海运",
			"estimated_days": seaDays, "cost_estimate": "低", "suitability": "recommended",
		})
		if strings.EqualFold(destination, "EU") {
			options = append(options, map[string]interface{}{
				"method": "rail", "name": "中欧班列",
				"estimated_days": "15-20", "cost_estimate": "中低", "suitability": "alternative",
			})
		}
		confidence = 0.85; riskLevel = "low"
	}
	if isPeakSeason && len(options) > 0 {
		options = append(options, map[string]interface{}{
			"method": "advance_buffer", "name": "旺季建议提前2周备货",
			"estimated_days": "提前2周", "cost_estimate": "—", "suitability": "warning",
		})
		if confidence-0.05 > 0.80 {
			confidence -= 0.05
		} else {
			confidence = 0.80
		}
	}
	output = map[string]interface{}{
		"suggested_logistics": suggestedLogistics, "decision_point": decisionPoint,
		"destination": destination, "stock_status": stockStatus,
		"sellable_days": sellableDays, "cargo_value": cargoValue, "weight_kg": weightKg,
		"options": options, "confidence": confidence,
	}
	return output, confidence, riskLevel, nil
}

// estimateDailySales estimates daily sales from multi-period sales data.
func estimateDailySales(sales7d, sales14d, sales30d float64) (float64, string) {
	if sales7d > 0 {
		return sales7d / 7, "7d"
	}
	if sales14d > 0 {
		return sales14d / 14, "14d"
	}
	if sales30d > 0 {
		return sales30d / 30, "30d"
	}
	return 0, "none"
}

// calcReplenishQty calculates the suggested replenish quantity.
func calcReplenishQty(dailySales float64, _ float64, targetDays int, currentStock, inTransit, moq int) int {
	targetStock := int(dailySales * float64(targetDays))
	available := currentStock + inTransit
	qty := targetStock - available
	if qty < 0 {
		qty = 0
	}
	if moq > 0 && qty < moq && qty > 0 {
		qty = moq
	}
	return qty
}

// pickLogistics selects a logistics channel name based on stock urgency.
func pickLogistics(stockStatus string, sellableDays float64, leadTime int) string {
	if stockStatus == "red" || sellableDays < float64(leadTime)*0.5 {
		return "空运/国际快递"
	}
	if stockStatus == "yellow" {
		return "快船"
	}
	return "海运"
}

// backfillLegacyFields maps old/legacy field names to current ones.
func backfillLegacyFields(ctx map[string]interface{}) {
	if _, ok := ctx["sellable_stock"]; !ok {
		if v, ok := ctx["quantity"]; ok {
			ctx["sellable_stock"] = v
		} else if v, ok := ctx["current_stock"]; ok {
			ctx["sellable_stock"] = v
		}
	}
	if _, ok := ctx["in_transit_stock"]; !ok {
		if v, ok := ctx["in_transit"]; ok {
			ctx["in_transit_stock"] = v
		}
	}
	if _, ok := ctx["moq"]; !ok {
		if v, ok := ctx["min_moq"]; ok {
			ctx["moq"] = v
		}
	}
	if _, ok := ctx["sales_7d"]; !ok {
		if v, ok := ctx["daily_sales"]; ok {
			if f := safeFloat(v); f > 0 {
				ctx["sales_7d"] = f * 7
			}
		}
	}
}

// round1 rounds a float64 to 1 decimal place.
func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
