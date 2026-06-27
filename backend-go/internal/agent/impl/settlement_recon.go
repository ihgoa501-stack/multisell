// Package impl provides concrete agent implementations.
//
// SettlementReconAgent implements A8 Settlement Reconciliation business logic.
//   - settlement_import:    Import settlement reports from platforms
//   - reconciliation_check: Cross-reference settlement items vs orders and ledger
//   - discrepancy_resolve:  Suggest resolution actions for discrepancies
//   - cash_flow_watch:      Monitor platform account balances and forecast cash flow
package impl

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
	"github.com/lingmirror/backend-go/internal/domain/exchangerate"
	"github.com/lingmirror/backend-go/internal/domain/finance"
	"github.com/lingmirror/backend-go/internal/domain/order"
	"github.com/lingmirror/backend-go/internal/domain/settlement"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------- Required and optional context field names ----------

var (
	settlementImportRequiredFields   = []string{"platform"}
	reconciliationRequiredFields     = []string{}
	discrepancyResolveRequiredFields = []string{}
	cashFlowWatchRequiredFields      = []string{}
)

// ---------- SettlementReconAgent ----------

// AccountInfo represents a platform account's summary for cash flow monitoring.
type AccountInfo struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Currency string  `json:"currency"`
	Balance  float64 `json:"balance"`
	Status   string  `json:"status"`
}

// SettlementReconAgent implements A8 Settlement Reconciliation logic.
// It handles settlement report import, order-vs-settlement reconciliation,
// discrepancy resolution, and platform cash flow monitoring.
type SettlementReconAgent struct {
	db       *gorm.DB
	logger   *zap.Logger
	registry *toolregistry.ToolRegistry
}

// NewSettlementReconAgent creates a new SettlementReconAgent.
// The db handle is used for querying settlement records, orders, ledger entries,
// platform accounts, and exchange rates.
func NewSettlementReconAgent(db *gorm.DB, logger *zap.Logger) *SettlementReconAgent {
	a := &SettlementReconAgent{
		db:       db,
		logger:   logger,
		registry: toolregistry.NewToolRegistry(logger),
	}
	a.registerTools()
	return a
}

// agentResult wraps the tuple returned by agent decision handlers for transport
// through the tool registry's Call method (which returns interface{}).
type agentResult struct {
	Output     map[string]interface{}
	Confidence float64
	RiskLevel  string
}

// registerTools registers each decision point as a tool in the tool registry.
func (a *SettlementReconAgent) registerTools() {
	tools := []toolregistry.Tool{
		{
			Name:        "settlement_import",
			Version:     "1.0.0",
			Description: "Import a settlement report for a platform",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "Settlement import parameters",
				Properties: map[string]*toolregistry.Schema{
					"platform": {Type: "string", Description: "Platform name (e.g. shopify, amazon, etsy)"},
					"filename": {Type: "string", Description: "Report filename for duplicate checking"},
					"raw_data": {Type: "array", Description: "Inline data rows to simulate import"},
				},
				Required: []string{"platform"},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "Import result with records count, warnings, and recommendations",
			},
			RiskLevel:  toolregistry.RiskMedium,
			MaxDuration: 30 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				output, confidence, riskLevel, err := a.importSettlement(input)
				if err != nil {
					return nil, err
				}
				return &agentResult{Output: output, Confidence: confidence, RiskLevel: riskLevel}, nil
			},
		},
		{
			Name:        "reconciliation_check",
			Version:     "1.0.0",
			Description: "Cross-reference settlement items against orders and ledger entries",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "Reconciliation parameters",
				Properties: map[string]*toolregistry.Schema{
					"period_start": {Type: "string", Description: "Start date (YYYY-MM-DD)"},
					"period_end":   {Type: "string", Description: "End date (YYYY-MM-DD)"},
				},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "Reconciliation result with match rate, discrepancy list, and recommendations",
			},
			RiskLevel:  toolregistry.RiskLow,
			MaxDuration: 60 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				output, confidence, riskLevel, err := a.reconciliationCheck(input)
				if err != nil {
					return nil, err
				}
				return &agentResult{Output: output, Confidence: confidence, RiskLevel: riskLevel}, nil
			},
		},
		{
			Name:        "discrepancy_resolve",
			Version:     "1.0.0",
			Description: "Suggest a resolution action for a given discrepancy",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "Discrepancy resolution parameters",
				Properties: map[string]*toolregistry.Schema{
					"discrepancy_type": {Type: "string", Description: "One of amount_mismatch, missing_order, extra_settlement"},
					"item_id":          {Type: "integer", Description: "Specific settlement_item ID to resolve"},
				},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "Resolution action, reasoning, and confidence",
			},
			RiskLevel:  toolregistry.RiskMedium,
			MaxDuration: 15 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				output, confidence, riskLevel, err := a.discrepancyResolve(input)
				if err != nil {
					return nil, err
				}
				return &agentResult{Output: output, Confidence: confidence, RiskLevel: riskLevel}, nil
			},
		},
		{
			Name:        "cash_flow_watch",
			Version:     "1.0.0",
			Description: "Monitor platform account balances and forecast cash flow",
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "Cash flow overview with pending amounts, forecast, and currency risk assessment",
			},
			RiskLevel:  toolregistry.RiskLow,
			MaxDuration: 30 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				output, confidence, riskLevel, err := a.cashFlowWatch(input)
				if err != nil {
					return nil, err
				}
				return &agentResult{Output: output, Confidence: confidence, RiskLevel: riskLevel}, nil
			},
		},
	}

	for i := range tools {
		a.registry.Register(&tools[i])
	}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
// It looks up the tool in the registry and delegates execution.
//
// Supported decision points:
//   - "settlement_import"     -- import a settlement report for a platform
//   - "reconciliation_check"  -- cross-reference settlement items vs orders and ledger
//   - "discrepancy_resolve"   -- suggest resolution actions for discrepancies
//   - "cash_flow_watch"       -- monitor platform balances and forecast cash flow
//
// Returns: output map, confidence [0-1], riskLevel (low/medium/high), error.
func (a *SettlementReconAgent) Decide(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	result, err := a.registry.Call(context.Background(), decisionPoint, ctx)
	if err != nil {
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "low", nil
	}
	war := result.(*agentResult)
	return war.Output, war.Confidence, war.RiskLevel, nil
}

// ====================== Decision Point: settlement_import ======================

// importSettlement imports settlement report data for a given platform.
//
// Input (from ctx):
//   - platform (string, required) -- e.g. "shopify", "amazon", "etsy"
//   - filename (string, optional) -- report filename, checked against platform_settlement_batch
//   - raw_data ([]interface{}, optional) -- inline data rows to simulate import
//
// The agent queries platform_settlement_batch to detect duplicate filenames.
func (a *SettlementReconAgent) importSettlement(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	// Validate required fields.
	if missing := missingFields(ctx, settlementImportRequiredFields); len(missing) > 0 {
		return insufficientData("settlement_import", missing), 0.0, "low", nil
	}

	platform := safeString(ctx["platform"], "unknown")
	filename := safeString(ctx["filename"], "")

	// -- Check for duplicate import --
	duplicateWarning := false
	var existingImport string
	if filename != "" && a.db != nil {
		var batch settlement.PlatformSettlementBatch
		result := a.db.Where("platform_name = ? AND filename = ?", platform, filename).
			Order("created_at DESC").
			First(&batch)
		if result.Error == nil {
			duplicateWarning = true
			existingImport = fmt.Sprintf("%s (%s)", batch.Filename, batch.CreatedAt.Format("2006-01-02"))
		}
	}

	// -- Determine record count from raw_data --
	recordsImported := 0
	errorsList := make([]string, 0)

	if raw, ok := ctx["raw_data"]; ok {
		if arr, ok := raw.([]interface{}); ok {
			recordsImported = len(arr)
		}
	}

	if duplicateWarning {
		errorsList = append(errorsList, fmt.Sprintf("文件 '%s' 已在 %s 导入过", filename, existingImport))
	}

	// -- Build output --
	status := "completed"
	if len(errorsList) > 0 {
		status = "completed_with_warnings"
	}

	output = map[string]interface{}{
		"status":            status,
		"decision_point":    "settlement_import",
		"platform":          platform,
		"filename":          filename,
		"records_imported":  recordsImported,
		"errors":            errorsList,
		"duplicate_warning": duplicateWarning,
		"recommendation":    importRecommendation(recordsImported, duplicateWarning),
	}

	// Confidence: high when actual records imported and no warnings.
	switch {
	case recordsImported == 0 && duplicateWarning:
		confidence = 0.55
		riskLevel = "high"
	case recordsImported > 0 && !duplicateWarning:
		confidence = 0.90
		riskLevel = "low"
	case recordsImported > 0 && duplicateWarning:
		confidence = 0.65
		riskLevel = "medium"
	default:
		confidence = 0.75
		riskLevel = "medium"
	}

	return output, confidence, riskLevel, nil
}

// importRecommendation returns a Chinese-language recommendation string.
func importRecommendation(recordsImported int, duplicate bool) string {
	if duplicate {
		return "检测到重复导入，建议核实后确认是否覆盖或跳过"
	}
	if recordsImported > 0 {
		return fmt.Sprintf("成功导入 %d 条结算记录，建议执行对账检查", recordsImported)
	}
	return "未提供结算数据，请上传文件或传入 raw_data"
}

// ====================== Decision Point: reconciliation_check ======================

// reconciliationCheck cross-references settlement items against orders and ledger entries.
//
// Input (from ctx):
//   - period_start (string, optional) -- YYYY-MM-DD, default 90 days ago
//   - period_end   (string, optional) -- YYYY-MM-DD, default now
//
// Queries:
//   - settlement + settlement_item for recent records
//   - sales_order by order_no for amount comparison
//   - finance_ledger_entry for the same period
//
// Categorizes discrepancies as:
//   - amount_mismatch:    settlement_item.amount != order.pay_amount
//   - missing_order:      settlement_item has order_no but no matching order exists
//   - extra_settlement:   settlement_item has no order_no reference
func (a *SettlementReconAgent) reconciliationCheck(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	// Determine period.
	periodStart := parseTimeField(ctx, "period_start", time.Now().AddDate(0, -3, 0))
	periodEnd := parseTimeField(ctx, "period_end", time.Now())

	// -- Query recent settlements --
	var settlements []settlement.Settlement
	if a.db != nil {
		if err := a.db.Where("created_at BETWEEN ? AND ?", periodStart, periodEnd).
			Order("created_at ASC").
			Find(&settlements).Error; err != nil {
			a.logger.Warn("settlement query failed in reconciliation_check",
				zap.Error(err))
		}
	}

	// -- Query settlement items for the period --
	var items []settlement.SettlementItem
	if a.db != nil {
		if err := a.db.Where("created_at BETWEEN ? AND ?", periodStart, periodEnd).
			Find(&items).Error; err != nil {
			a.logger.Warn("settlement_item query failed in reconciliation_check",
				zap.Error(err))
		}
	}

	// -- Query finance_ledger_entry for the same period --
	var ledgerEntries []finance.FinanceLedgerEntry
	if a.db != nil {
		if err := a.db.Where("created_at BETWEEN ? AND ?", periodStart, periodEnd).
			Find(&ledgerEntries).Error; err != nil {
			a.logger.Warn("finance_ledger_entry query failed in reconciliation_check",
				zap.Error(err))
		}
	}

	// -- Cross-reference: settlement item vs order --
	type Discrepancy struct {
		ItemID      int64   `json:"item_id"`
		OrderNo     string  `json:"order_no"`
		Type        string  `json:"type"`
		SettledAmt  float64 `json:"settled_amount"`
		ExpectedAmt float64 `json:"expected_amount"`
		Diff        float64 `json:"difference"`
		Description string  `json:"description"`
	}

	discrepancies := make([]Discrepancy, 0)
	totalChecked := 0

	for _, item := range items {
		totalChecked++
		disc := Discrepancy{
			ItemID:     item.ID,
			OrderNo:    item.OrderNo,
			SettledAmt: item.Amount,
		}

		if item.OrderNo == "" {
			// Settlement item without order reference -- flag as extra_settlement.
			disc.Type = "extra_settlement"
			disc.Description = fmt.Sprintf("结算条目 #%d 无关联订单号，可能为平台杂费或调整项", item.ID)
			disc.ExpectedAmt = 0
			disc.Diff = item.Amount
			discrepancies = append(discrepancies, disc)
			continue
		}

		// Look up the order by order_no.
		if a.db == nil {
			continue
		}
		var ord order.Order
		result := a.db.Where("order_no = ?", item.OrderNo).First(&ord)
		if result.Error != nil {
			disc.Type = "missing_order"
			disc.Description = fmt.Sprintf("订单 %s 在系统中不存在，结算金额 ¥%.2f", item.OrderNo, item.Amount)
			disc.ExpectedAmt = 0
			disc.Diff = item.Amount
			discrepancies = append(discrepancies, disc)
			continue
		}

		// Compare settlement amount vs order pay_amount.
		diff := round2(math.Abs(item.Amount - ord.PayAmount))
		if diff > 0.01 {
			disc.Type = "amount_mismatch"
			disc.ExpectedAmt = ord.PayAmount
			disc.Diff = diff
			disc.Description = fmt.Sprintf(
				"订单 %s 结算金额 ¥%.2f 与订单实付金额 ¥%.2f 不一致，差异 ¥%.2f",
				item.OrderNo, item.Amount, ord.PayAmount, diff,
			)
			discrepancies = append(discrepancies, disc)
		}
	}

	// -- Compare settlement totals with ledger totals --
	totalSettledRevenue := 0.0
	for _, s := range settlements {
		totalSettledRevenue += s.TotalRevenue
	}
	totalLedgerRevenue := 0.0
	for _, le := range ledgerEntries {
		if le.EntryType == "revenue" {
			totalLedgerRevenue += le.Amount
		}
	}

	ledgerDiff := round2(math.Abs(totalSettledRevenue - totalLedgerRevenue))
	hasLedgerDiscrepancy := ledgerDiff > 1.0

	// -- Compute match rate --
	discrepancyCount := len(discrepancies)
	matchRate := 1.0
	if totalChecked > 0 {
		matchRate = round2(float64(totalChecked-discrepancyCount) / float64(totalChecked))
	}

	// -- Build summary --
	countByType := map[string]int{
		"amount_mismatch":  0,
		"missing_order":    0,
		"extra_settlement": 0,
	}
	for _, d := range discrepancies {
		countByType[d.Type]++
	}

	discrepancyItems := make([]map[string]interface{}, 0, len(discrepancies))
	for _, d := range discrepancies {
		discrepancyItems = append(discrepancyItems, map[string]interface{}{
			"item_id":         d.ItemID,
			"order_no":        d.OrderNo,
			"type":            d.Type,
			"settled_amount":  d.SettledAmt,
			"expected_amount": d.ExpectedAmt,
			"difference":      d.Diff,
			"description":     d.Description,
		})
	}

	summary := map[string]interface{}{
		"total_checked":          totalChecked,
		"discrepancy_count":      discrepancyCount,
		"match_rate":             matchRate,
		"amount_mismatch_count":  countByType["amount_mismatch"],
		"missing_order_count":    countByType["missing_order"],
		"extra_settlement_count": countByType["extra_settlement"],
		"settlement_total":       round2(totalSettledRevenue),
		"ledger_total":           round2(totalLedgerRevenue),
		"ledger_difference":      ledgerDiff,
		"has_ledger_discrepancy": hasLedgerDiscrepancy,
	}

	// -- Status and confidence --
	riskLevel = "low"
	confidence = 0.85

	switch {
	case discrepancyCount == 0 && totalChecked == 0:
		confidence = 0.55
		riskLevel = "low"
		output = map[string]interface{}{
			"status":         "no_data",
			"decision_point": "reconciliation_check",
			"period_start":   periodStart.Format("2006-01-02"),
			"period_end":     periodEnd.Format("2006-01-02"),
			"summary":        "所选时间段内无结算记录，请扩大查询范围",
			"confidence":     confidence,
			"risk_level":     riskLevel,
		}
		return output, confidence, riskLevel, nil

	case discrepancyCount == 0 && totalChecked > 0:
		confidence = 0.92
		riskLevel = "low"

	case matchRate < 0.8:
		confidence = 0.70
		riskLevel = "high"

	case matchRate < 0.95:
		confidence = 0.80
		riskLevel = "medium"
	}

	if hasLedgerDiscrepancy {
		if matchRate >= 0.95 {
			confidence = round2(confidence - 0.05)
		}
	}

	// -- Build recommendation --
	recommendation := reconciliationRecommendation(matchRate, countByType, hasLedgerDiscrepancy)

	output = map[string]interface{}{
		"status":            "completed",
		"decision_point":    "reconciliation_check",
		"period_start":      periodStart.Format("2006-01-02"),
		"period_end":        periodEnd.Format("2006-01-02"),
		"total_checked":     totalChecked,
		"match_rate":        matchRate,
		"discrepancies":     discrepancyItems,
		"discrepancy_count": discrepancyCount,
		"summary":           summary,
		"recommendation":    recommendation,
		"confidence":        confidence,
	}

	return output, confidence, riskLevel, nil
}

// reconciliationRecommendation returns a Chinese-language recommendation.
func reconciliationRecommendation(matchRate float64, countByType map[string]int, ledgerDisc bool) string {
	if countByType["amount_mismatch"]+countByType["missing_order"]+countByType["extra_settlement"] == 0 {
		return "对账完成，所有结算条目与订单一致，无需处理"
	}

	parts := make([]string, 0)
	if countByType["amount_mismatch"] > 0 {
		parts = append(parts, fmt.Sprintf("有 %d 笔金额不符，建议核实后调整", countByType["amount_mismatch"]))
	}
	if countByType["missing_order"] > 0 {
		parts = append(parts, fmt.Sprintf("有 %d 条结算记录缺少对应订单，需调查是否为平台补录", countByType["missing_order"]))
	}
	if countByType["extra_settlement"] > 0 {
		parts = append(parts, fmt.Sprintf("有 %d 条无订单关联项，可能为平台费用或退款", countByType["extra_settlement"]))
	}
	if ledgerDisc {
		parts = append(parts, "结算总额与账本收入不一致，建议核查账务记录")
	}

	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "；"
		}
		result += p
	}
	return result
}

// ====================== Decision Point: discrepancy_resolve ======================

// discrepancyResolve suggests a resolution action for a given discrepancy.
//
// Input (from ctx):
//   - discrepancy_type (string, optional) -- one of "amount_mismatch", "missing_order",
//     "extra_settlement". If omitted, the agent auto-detects from recent unreconciled items.
//   - item_id (int64, optional) -- specific settlement_item to resolve
//
// Returns suggested action, reasoning, and confidence.
func (a *SettlementReconAgent) discrepancyResolve(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	discType := safeString(ctx["discrepancy_type"], "")
	itemID := safeFloat(ctx["item_id"], 0)

	// If no specific type given, auto-detect from recent unreconciled items.
	if discType == "" && a.db != nil {
		var recentItems []settlement.SettlementItem
		query := a.db.Where("reconciliation_status = ?", "pending").
			Order("created_at DESC").
			Limit(5).
			Find(&recentItems)

		if query.Error == nil && len(recentItems) > 0 {
			// Count by pattern.
			missingOrders := 0
			hasAmountMismatch := false
			for _, item := range recentItems {
				if item.OrderNo == "" {
					continue
				}
				var ord order.Order
				if err := a.db.Where("order_no = ?", item.OrderNo).First(&ord).Error; err != nil {
					missingOrders++
				} else if math.Abs(item.Amount-ord.PayAmount) > 0.01 {
					hasAmountMismatch = true
				}
			}

			switch {
			case hasAmountMismatch:
				discType = "amount_mismatch"
			case missingOrders > len(recentItems)/2:
				discType = "missing_order"
			default:
				discType = "extra_settlement"
			}
		} else {
			discType = "extra_settlement"
		}
		confidence = 0.75
		riskLevel = "medium"
	}

	if discType == "" {
		discType = "extra_settlement"
	}

	// -- Map discrepancy type to action --
	type Resolution struct {
		ResolveAction string
		Reasoning     string
	}

	resolutions := map[string]Resolution{
		"amount_mismatch": {
			ResolveAction: "review_and_adjust",
			Reasoning:     "结算金额与订单实付不符，建议人工核查平台结算报告与订单明细，确认差异原因后进行账务调整",
		},
		"missing_order": {
			ResolveAction: "investigate_settlement",
			Reasoning:     "结算记录引用了系统中不存在的订单号，可能是平台补录数据或历史订单未同步，需在平台后台核实并补充订单信息",
		},
		"extra_settlement": {
			ResolveAction: "claim_or_write_off",
			Reasoning:     "无关联订单的结算条目，可能为平台费用、退款、或调整项，需核实后确认是申请退款还是直接核销",
		},
	}

	res, ok := resolutions[discType]
	if !ok {
		return map[string]interface{}{
			"status":           "unknown",
			"decision_point":   "discrepancy_resolve",
			"discrepancy_type": discType,
			"error":            fmt.Sprintf("不支持的差异类型: %s", discType),
		}, 0.0, "low", nil
	}

	output = map[string]interface{}{
		"status":           "completed",
		"decision_point":   "discrepancy_resolve",
		"discrepancy_type": discType,
		"item_id":          int64(itemID),
		"resolve_action":   res.ResolveAction,
		"reasoning":        res.Reasoning,
	}

	// Confidence based on data available.
	switch discType {
	case "amount_mismatch":
		if itemID > 0 {
			confidence = 0.85
			riskLevel = "medium"
		} else {
			confidence = 0.70
			riskLevel = "high"
		}
	case "missing_order":
		if itemID > 0 {
			confidence = 0.80
			riskLevel = "high"
		} else {
			confidence = 0.65
			riskLevel = "high"
		}
	case "extra_settlement":
		confidence = 0.75
		riskLevel = "medium"
	default:
		confidence = 0.60
		riskLevel = "medium"
	}

	return output, confidence, riskLevel, nil
}

// ====================== Decision Point: cash_flow_watch ======================

// cashFlowWatch monitors platform account balances and forecasts cash flow.
//
// Queries:
//   - finance_account for platform-type accounts
//   - settlement for pending/unsettled totals
//   - settlement history for 30-day cash flow forecast
//   - exchange_rate for multi-currency risk assessment
//
// Returns: total_pending, total_settled, forecast_next_30d, currency_risk, etc.
func (a *SettlementReconAgent) cashFlowWatch(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	// -- Platform accounts --
	platformAccounts := make([]AccountInfo, 0)
	pendingSettlements := 0
	var pendingAmount, settledAmount float64

	if a.db != nil {
		var accounts []finance.FinanceAccount
		if err := a.db.Where("account_type = ? AND status = ?", "platform", "active").
			Find(&accounts).Error; err != nil {
			a.logger.Warn("finance_account query failed in cash_flow_watch", zap.Error(err))
		} else {
			for _, acct := range accounts {
				platformAccounts = append(platformAccounts, AccountInfo{
					ID:       acct.ID,
					Name:     acct.Name,
					Currency: acct.Currency,
					Balance:  acct.Balance,
					Status:   acct.Status,
				})
			}
		}

		// -- Pending/unsettled settlement totals --
		type SettleSum struct {
			TotalAmount float64
		}
		var pendingSum SettleSum
		if err := a.db.Model(&settlement.Settlement{}).
			Select("COALESCE(SUM(total_net), 0) as total_amount").
			Where("status IN ?", []string{"pending", "unsettled"}).
			Scan(&pendingSum).Error; err == nil {
			pendingAmount = pendingSum.TotalAmount
		}

		// Count pending records.
		var pendingCount int64
		if err := a.db.Model(&settlement.Settlement{}).
			Where("status IN ?", []string{"pending", "unsettled"}).
			Count(&pendingCount).Error; err == nil {
			pendingSettlements = int(pendingCount)
		}

		// -- Settled total (last 90 days) --
		type SettleRow struct {
			TotalAmount float64
		}
		var settledSum SettleRow
		threeMonthsAgo := time.Now().AddDate(0, -3, 0)
		if err := a.db.Model(&settlement.Settlement{}).
			Select("COALESCE(SUM(total_net), 0) as total_amount").
			Where("status = ? AND created_at > ?", "settled", threeMonthsAgo).
			Scan(&settledSum).Error; err == nil {
			settledAmount = settledSum.TotalAmount
		}
	}

	// -- Forecast: estimate next 30 days from recent 90-day settled average --
	forecast := round2(settledAmount / 90 * 30)

	// -- Multi-currency risk assessment --
	currencyRisk := assessCurrencyRisk(a, platformAccounts)

	// -- Build platform balance summary --
	totalPlatformBalance := 0.0
	currencySet := make(map[string]bool)
	platformDetails := make([]map[string]interface{}, 0, len(platformAccounts))
	for _, acct := range platformAccounts {
		totalPlatformBalance += acct.Balance
		currencySet[acct.Currency] = true
		platformDetails = append(platformDetails, map[string]interface{}{
			"id":       acct.ID,
			"name":     acct.Name,
			"currency": acct.Currency,
			"balance":  acct.Balance,
			"status":   acct.Status,
		})
	}

	// -- Status and confidence --
	switch {
	case pendingAmount > 100000:
		confidence = 0.88
		riskLevel = "medium"
	case settledAmount > 0:
		confidence = 0.85
		riskLevel = "low"
	default:
		confidence = 0.55
		riskLevel = "medium"
	}

	if len(currencySet) > 1 {
		confidence = round2(confidence - 0.05)
	}

	output = map[string]interface{}{
		"status":                  "completed",
		"decision_point":          "cash_flow_watch",
		"platform_accounts":       platformDetails,
		"total_platform_balance":  round2(totalPlatformBalance),
		"pending_settlements":     pendingSettlements,
		"total_pending":           round2(pendingAmount),
		"total_settled_90d":       round2(settledAmount),
		"forecast_next_30d":       forecast,
		"currency_risk":           currencyRisk,
		"currencies_in_use":       currencyKeys(currencySet),
		"recommendation":          cashFlowRecommendation(pendingAmount, forecast, currencyRisk),
		"confidence":              confidence,
	}

	return output, confidence, riskLevel, nil
}

// assessCurrencyRisk checks exchange rate data to evaluate multi-currency risk.
// It queries the latest exchange rates for platform account currencies to CNY.
// Low risk: only CNY. Medium: rates stable. High: rates unavailable or volatile.
func assessCurrencyRisk(a *SettlementReconAgent, accounts []AccountInfo) string {
	// Collect unique currencies excluding CNY.
	currencies := make(map[string]bool)
	for _, acct := range accounts {
		c := acct.Currency
		if c == "" {
			c = "CNY"
		}
		if c != "CNY" {
			currencies[c] = true
		}
	}

	if len(currencies) == 0 {
		return "low"
	}

	if a.db == nil {
		return "medium"
	}

	// Try to fetch latest exchange rates for foreign currencies to CNY.
	for from := range currencies {
		var rate exchangerate.ExchangeRate
		result := a.db.Where("from_currency = ? AND to_currency = ?", from, "CNY").
			Order("effective_date DESC").
			First(&rate)
		if result.Error != nil {
			a.logger.Warn("no exchange rate found for currency pair",
				zap.String("from", from), zap.String("to", "CNY"))
			return "medium"
		}
	}

	return "low"
}

// cashFlowRecommendation returns a Chinese-language recommendation.
func cashFlowRecommendation(pendingAmount, forecast float64, currencyRisk string) string {
	parts := make([]string, 0)

	if pendingAmount > 100000 {
		parts = append(parts, fmt.Sprintf("待结算金额 ¥%.2f 较高，建议关注回款进度", pendingAmount))
	} else if pendingAmount > 0 {
		parts = append(parts, fmt.Sprintf("当前待结算金额 ¥%.2f，在正常范围内", pendingAmount))
	} else {
		parts = append(parts, "当前无待结算记录")
	}

	if forecast > 0 {
		parts = append(parts, fmt.Sprintf("预计未来30天结算流入 ¥%.2f", forecast))
	}

	switch currencyRisk {
	case "high":
		parts = append(parts, "多币种汇率波动风险较高，建议考虑锁汇或加速结汇")
	case "medium":
		parts = append(parts, "存在多币种结算，建议监控汇率变动")
	}

	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "；"
		}
		result += p
	}
	return result
}

// ====================== Helpers ======================

// parseTimeField extracts a time.Time from ctx by key in YYYY-MM-DD format.
// Returns fallback if the key is missing or unparseable.
func parseTimeField(ctx map[string]interface{}, key string, fallback time.Time) time.Time {
	v, ok := ctx[key]
	if !ok || v == nil {
		return fallback
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return fallback
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fallback
	}
	return t
}

// currencyKeys returns the keys of a map[string]bool as a string slice.
func currencyKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
