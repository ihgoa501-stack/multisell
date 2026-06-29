package owner

import (
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides Owner cockpit aggregated data.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new Owner cockpit service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// RiskSummary returns aggregated risk metrics for the Owner cockpit.
func (s *Service) RiskSummary() (map[string]interface{}, error) {
	result := map[string]interface{}{
		"low_profit_products":   0,
		"missing_data_products": 0,
		"pending_approvals":     0,
		"sync_errors":           0,
		"total_candidates":      0,
		"total_recommendations": 0,
		"list_ready_products":   0,
	}

	var totalCandidates int64
	s.db.Raw("SELECT COUNT(*) FROM candidate_product").Scan(&totalCandidates)
	result["total_candidates"] = totalCandidates

	var missingCount int64
	s.db.Model(&struct{}{}).Table("completeness_check").
		Where("status = ?", "incomplete").
		Distinct("product_id").Count(&missingCount)
	result["missing_data_products"] = missingCount

	var lowProfit int64
	s.db.Raw(`
		SELECT COUNT(DISTINCT product_id) FROM profit_summary
		WHERE status IN ('marginal', 'unprofitable')
	`).Scan(&lowProfit)
	result["low_profit_products"] = lowProfit

	var pendingApp int64
	s.db.Model(&struct{}{}).Table("listing_task").
		Where("status = ?", "blocked").Count(&pendingApp)
	result["pending_approvals"] = pendingApp

	var syncErr int64
	s.db.Model(&struct{}{}).Table("mock_sync_status").
		Where("status = ?", "failed").Count(&syncErr)
	result["sync_errors"] = syncErr

	var totalRec int64
	s.db.Model(&struct{}{}).Table("listing_recommendation").Count(&totalRec)
	result["total_recommendations"] = totalRec

	var listReady int64
	s.db.Raw(`
		SELECT COUNT(DISTINCT product_id) FROM listing_recommendation
		WHERE decision = 'list'
	`).Scan(&listReady)
	result["list_ready_products"] = listReady

	return result, nil
}

// Suggestions returns the latest listing recommendations as Owner-facing suggestions.
func (s *Service) Suggestions(limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 20
	}
	type Rec struct {
		ID                  int64
		ProductID           int64
		Decision            string
		Confidence          float64
		Reason              string
		RiskFlags           string
		CreatedListingTaskID *int64
		CreatedAt           time.Time
	}
	var recs []Rec
	if err := s.db.Table("listing_recommendation").
		Order("id DESC").Limit(limit).Find(&recs).Error; err != nil {
		return nil, err
	}

	// Batch load product titles
	productIDs := make([]int64, 0, len(recs))
	for _, r := range recs {
		productIDs = append(productIDs, r.ProductID)
	}
	type ProdInfo struct {
		ID    int64
		Title string
	}
	var prodInfos []ProdInfo
	if len(productIDs) > 0 {
		s.db.Table("candidate_product").Select("id, title").Where("id IN ?", productIDs).Scan(&prodInfos)
	}
	titleMap := make(map[int64]string, len(prodInfos))
	for _, p := range prodInfos {
		titleMap[p.ID] = p.Title
	}

	var result []map[string]interface{}
	for _, r := range recs {
		title := titleMap[r.ProductID]

		riskLevel := "low"
		if r.Decision == "skip" {
			riskLevel = "high"
		} else if r.Confidence < 0.6 {
			riskLevel = "medium"
		}

		suggestion := "建议上架"
		if r.Decision == "cautious" {
			suggestion = "谨慎上架"
		} else if r.Decision == "skip" {
			suggestion = "不建议上架"
		}

		result = append(result, map[string]interface{}{
			"id":              r.ID,
			"product_id":      r.ProductID,
			"product_title":   title,
			"listing_task_id": r.CreatedListingTaskID,
			"agent_source":    "商品评估 Agent",
			"suggestion":      suggestion,
			"decision":        r.Decision,
			"reason":          r.Reason,
			"confidence":      r.Confidence,
			"risk_flags":      r.RiskFlags,
			"risk_level":      riskLevel,
			"created_at":      r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return result, nil
}

// PlatformSyncStatus returns sync status for Owner cockpit.
func (s *Service) PlatformSyncStatus() ([]map[string]interface{}, error) {
	type SyncRow struct {
		PlatformID    int64
		PlatformName  string
		SyncType      string
		Status        string
		RecordsSynced int
		ErrorMessage  string
		LastSyncAt    *time.Time
		IsMockData    bool
	}
	var rows []SyncRow
	s.db.Table("mock_sync_status").
		Where("id IN (SELECT DISTINCT ON (platform_id, sync_type) id FROM mock_sync_status ORDER BY platform_id, sync_type, id DESC)").
		Order("platform_id, sync_type").Find(&rows)

	platforms := make(map[int64]map[string]interface{})
	for _, r := range rows {
		if _, ok := platforms[r.PlatformID]; !ok {
			mode := "mock"
			if !r.IsMockData {
				mode = "sandbox"
			}
			lastSyncStr := ""
			if r.LastSyncAt != nil {
				lastSyncStr = r.LastSyncAt.Format("2006-01-02 15:04:05")
			}
			platforms[r.PlatformID] = map[string]interface{}{
				"platform_id":    r.PlatformID,
				"platform_name":  r.PlatformName,
				"mode":           mode,
				"orders_sync":    "-",
				"products_sync":  "-",
				"fees_sync":      "-",
				"settlements_sync": "-",
				"last_sync_time":  lastSyncStr,
			}
		}
		entry := platforms[r.PlatformID]
		switch r.SyncType {
		case "orders":
			entry["orders_sync"] = r.Status
		case "products":
			entry["products_sync"] = r.Status
		case "fees":
			entry["fees_sync"] = r.Status
		}
	}

	var result []map[string]interface{}
	for _, v := range platforms {
		result = append(result, v)
	}
	return result, nil
}
