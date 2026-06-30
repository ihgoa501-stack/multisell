package tools

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

// DiscountTools returns tool definitions for the discount risk domain (G3).
func DiscountTools() []toolregistry.Tool {
	return []toolregistry.Tool{
		{
			Name:        "discount.check",
			Version:     "1.0.0",
			Description: "多折扣叠加模拟和利润风险分析——输入售价、成本价、折扣列表，输出折扣后价格、利润率和拦截/警告/允许决策",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "折扣检查参数",
				Properties: map[string]*toolregistry.Schema{
					"price":          {Type: "number", Description: "商品售价(原价)"},
					"cost":           {Type: "number", Description: "成本价"},
					"discounts":      {Type: "array", Description: "折扣列表，每项含 type(pct/fixed/bxgy)和value"},
					"min_margin_pct": {Type: "number", Description: "最低利润率百分比(默认10)"},
					"platform":       {Type: "string", Description: "平台标识(ozon/shopee等)，用于价格地板检查"},
				},
				Required: []string{"price", "cost"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "折扣分析结果，包含折扣后价格、利润率、决策建议",
			},
			Handler:          discountCheckHandler,
			RiskLevel:        toolregistry.RiskMedium,
			MaxDuration:      5 * time.Second,
			SensitiveData:    true,
		},
		{
			Name:             "discount.risk_check",
			Version:          "1.0.0",
			Description:      "discount.check 别名——折扣风险检查，用于折扣风控定时扫描",
			Handler:          discountCheckHandler,
			RiskLevel:        toolregistry.RiskMedium,
			MaxDuration:      5 * time.Second,
			SensitiveData:    true,
		},
		{
			Name:        "discount.validate",
			Version:     "1.0.0",
			Description: "单一促销验证——检查单个促销活动是否满足价格地板和利润要求",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "促销验证参数",
				Properties: map[string]*toolregistry.Schema{
					"price":    {Type: "number", Description: "促销价"},
					"cost":     {Type: "number", Description: "成本价"},
					"original": {Type: "number", Description: "原价"},
				},
				Required: []string{"price", "cost"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "验证结果，包含利润率和通过/拒绝状态",
			},
			Handler:          discountValidateHandler,
			RiskLevel:        toolregistry.RiskMedium,
			MaxDuration:      3 * time.Second,
			SensitiveData:    true,
		},
	}
}

// discountCheckHandler performs multi-discount stacking simulation and margin risk analysis.
func discountCheckHandler(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	price := safeFloat(input["price"], 0)
	cost := safeFloat(input["cost"], 0)
	minMargin := safeFloat(input["min_margin_pct"], 10.0)

	if price <= 0 || cost <= 0 {
		return nil, fmt.Errorf("discount.check: price and cost must be positive")
	}

	// Calculate effective discount rate from stacked discounts.
	discounts := safeDiscounts(input["discounts"])
	finalPrice := price
	for _, d := range discounts {
		switch d.t {
		case "pct":
			finalPrice = finalPrice * (1 - d.v/100)
		case "fixed":
			finalPrice -= d.v
		case "bxgy":
			// BXGY: buy b, get g free → effective pct = g/(b+g)
			if d.buy > 0 {
				finalPrice = finalPrice * (1 - float64(d.free)/float64(d.buy+d.free))
			}
		}
	}
	if finalPrice < 0 {
		finalPrice = 0
	}

	effectiveRate := (1 - finalPrice/price) * 100
	margin := finalPrice - cost
	marginPct := (margin / cost) * 100

	// Build decision.
	status := "allow"
	action := "allow"
	riskLevel := "low"
	confidence := 0.95
	alerts := []string{}
	var message string

	switch {
	case marginPct < minMargin:
		status = "block"
		action = "block"
		riskLevel = "critical"
		confidence = 0.99
		message = fmt.Sprintf("折扣后利润率 %.1f%% 低于最低要求 %.0f%%，建议拦截该折扣", marginPct, minMargin)
		alerts = append(alerts, message)
	case marginPct < minMargin*1.5:
		status = "warn"
		action = "warn"
		riskLevel = "medium"
		confidence = 0.85
		message = fmt.Sprintf("折扣后利润率 %.1f%% 接近风险阈值(%.0f%%)，建议人工复核", marginPct, minMargin)
		alerts = append(alerts, message)
	default:
		message = fmt.Sprintf("折扣后利润率 %.1f%%，在安全范围内，允许上架", marginPct)
	}

	return map[string]interface{}{
		"status":        status,
		"action":        action,
		"risk_level":    riskLevel,
		"confidence":    confidence,
		"message":       message,
		"alerts":        alerts,
		"original_price": price,
		"final_price":   finalPrice,
		"discount_rate": math.Round(effectiveRate*100) / 100,
		"margin":        math.Round(margin*100) / 100,
		"margin_pct":    math.Round(marginPct*100) / 100,
	}, nil
}

// discountValidateHandler validates a single promotion price against cost.
func discountValidateHandler(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	price := safeFloat(input["price"], 0)
	cost := safeFloat(input["cost"], 0)

	if price <= 0 || cost <= 0 {
		return nil, fmt.Errorf("discount.validate: price and cost must be positive")
	}

	margin := price - cost
	marginPct := (margin / cost) * 100

	valid := marginPct >= 0
	riskLevel := "low"
	if marginPct < 5 {
		riskLevel = "medium"
	}
	if marginPct < 0 {
		riskLevel = "critical"
	}

	return map[string]interface{}{
		"valid":      valid,
		"risk_level": riskLevel,
		"margin":     math.Round(margin*100) / 100,
		"margin_pct": math.Round(marginPct*100) / 100,
		"message":    fmt.Sprintf("促销后利润率 %.1f%%，%s", marginPct, map[bool]string{true: "通过", false: "拒绝"}[valid]),
	}, nil
}

// --- Discount helpers ---

type discount struct {
	t    string // "pct" | "fixed" | "bxgy"
	v    float64
	buy  int
	free int
}

// safeDiscounts parses the discounts input into a slice of discount structs.
// Accepts nil, an array of maps, or a single float (treated as pct discount).
func safeDiscounts(v interface{}) []discount {
	if v == nil {
		return nil
	}
	switch raw := v.(type) {
	case []interface{}:
		var ds []discount
		for _, item := range raw {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			d := discount{
				t:    safeString(m["type"], "pct"),
				v:    safeFloat(m["value"], 0),
				buy:  int(safeFloat(m["buy_qty"], 0)),
				free: int(safeFloat(m["free_qty"], 0)),
			}
			ds = append(ds, d)
		}
		return ds
	case float64:
		return []discount{{t: "pct", v: raw * 100}}
	}
	return nil
}
