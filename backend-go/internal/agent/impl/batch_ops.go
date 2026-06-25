package impl

import "fmt"

type BatchOpsAgent struct{}
func NewBatchOpsAgent() *BatchOpsAgent { return &BatchOpsAgent{} }

func (a *BatchOpsAgent) Decide(decisionPoint string, ctx map[string]interface{}) (map[string]interface{}, float64, string, error) {
	switch decisionPoint {
	case "batch_price_update": return a.batchPriceUpdate(ctx)
	case "batch_inventory_sync": return a.batchInventorySync(ctx)
	case "batch_listing_update": return a.batchListingUpdate(ctx)
	case "import_validation": return a.importValidation(ctx)
	}
	return insufficientData(decisionPoint, []string{"unknown"}), 0.5, "low", nil
}

func (a *BatchOpsAgent) batchPriceUpdate(ctx map[string]interface{}) (map[string]interface{}, float64, string, error) {
	skuCount := int(safeFloat(ctx["sku_count"]))
	if skuCount <= 0 { skuCount = 1 }

	out := map[string]interface{}{
		"action":    "batch_price_update",
		"sku_count": skuCount,
	}
	if skuCount > 50 {
		out["risk"] = "high"
		out["recommendation"] = fmt.Sprintf("批量更新 %d 个 SKU 价格，需人工审批", skuCount)
		return out, 0.70, "high", nil
	}
	if skuCount > 10 {
		out["recommendation"] = fmt.Sprintf("批量更新 %d 个 SKU 价格，建议审核后执行", skuCount)
		return out, 0.80, "medium", nil
	}
	out["recommendation"] = fmt.Sprintf("批量更新 %d 个 SKU 价格", skuCount)
	return out, 0.90, "low", nil
}

func (a *BatchOpsAgent) batchInventorySync(ctx map[string]interface{}) (map[string]interface{}, float64, string, error) {
	out := map[string]interface{}{
		"action":  "batch_inventory_sync",
		"status":  "ready",
		"recommendation": "库存同步任务已就绪，建议在低峰期执行",
	}
	return out, 0.85, "low", nil
}

func (a *BatchOpsAgent) batchListingUpdate(ctx map[string]interface{}) (map[string]interface{}, float64, string, error) {
	count := int(safeFloat(ctx["listing_count"]))
	if count <= 0 { count = 1 }
	out := map[string]interface{}{
		"action":       "batch_listing_update",
		"listing_count": count,
		"recommendation": fmt.Sprintf("批量更新 %d 个 Listing", count),
	}
	return out, 0.85, "medium", nil
}

func (a *BatchOpsAgent) importValidation(ctx map[string]interface{}) (map[string]interface{}, float64, string, error) {
	fields := []string{"file_type", "record_count"}
	if missing := missingFields(ctx, fields); len(missing) > 0 {
		return insufficientData("import_validation", missing), 0.3, "low", nil
	}
	out := map[string]interface{}{
		"action":     "import_validation",
		"file_type":  safeString(ctx["file_type"]),
		"valid":      true,
		"columns":    ctx["columns"],
		"recommendation": "文件格式验证通过，可导入",
	}
	return out, 0.90, "low", nil
}
