package completeness

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/domain/profit"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides completeness check business logic.
type Service struct {
	db        *gorm.DB
	logger    *zap.Logger
	profitSvc *profit.Service // optional; nil means economic estimates skipped
}

// NewService creates a new completeness service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// dimensions defines all checkable dimensions with labels and weights.
var dimensions = []struct {
	Key    string
	Label  string
	Weight float64
}{
	{"title", "商品标题", 15},
	{"description", "商品描述", 10},
	{"main_image", "主图", 10},
	{"images", "多图", 5},
	{"category", "类目", 10},
	{"brand", "品牌", 5},
	{"spec", "规格参数（颜色/尺寸/重量）", 5},
	{"purchase_price", "采购成本", 10},
	{"package_info", "包装信息（重量/尺寸）", 10},
	{"hs_code", "HS编码", 10},
	{"target_sale_price", "目标售价", 5},
	{"origin_country", "原产地", 5},
}

// Check performs a completeness check on a candidate product.
func (s *Service) Check(productID int64, triggeredBy string) (*CheckResult, error) {
	var prod candidate.CandidateProduct
	if err := s.db.First(&prod, productID).Error; err != nil {
		return nil, err
	}

	var dims []CompletenessDimension
	var missing []string
	totalScore := 0.0
	totalWeight := 0.0

	for _, d := range dimensions {
		dim := s.checkDimension(&prod, d.Key, d.Label)
		dim.Weight = d.Weight
		dims = append(dims, dim)
		totalScore += dim.Score * d.Weight / 100.0
		totalWeight += d.Weight
		if !dim.Complete {
			missing = append(missing, d.Label)
		}
	}

	score := 0.0
	if totalWeight > 0 {
		score = totalScore / totalWeight * 100.0
	}
	if score > 100 {
		score = 100
	}

	status := "complete"
	if score < 80 {
		status = "incomplete"
	}

	// Store the check
	breakdownJSON, _ := json.Marshal(dims)
	missingJSON, _ := json.Marshal(missing)

	check := CompletenessCheck{
		ProductID:      productID,
		Score:          score,
		MissingItems:   string(missingJSON),
		ScoreBreakdown: string(breakdownJSON),
		Status:         status,
		TriggeredBy:    triggeredBy,
	}
	if triggeredBy == "" {
		check.TriggeredBy = "system"
	}
	if err := s.db.Create(&check).Error; err != nil {
		return nil, fmt.Errorf("save completeness check: %w", err)
	}

	return &CheckResult{
		ProductID:    productID,
		Score:        score,
		Status:       status,
		Dimensions:   dims,
		MissingItems: missing,
		Breakdown:    string(breakdownJSON),
	}, nil
}

func (s *Service) checkDimension(prod *candidate.CandidateProduct, key, label string) CompletenessDimension {
	switch key {
	case "title":
		v := strings.TrimSpace(prod.Title)
		return s.makeDim(label, v != "" && len(v) >= 10, v, "标题为空或过短（至少10个字符）")
	case "description":
		v := strings.TrimSpace(prod.Description)
		return s.makeDim(label, v != "" && len(v) >= 20, v, "描述为空或过短（至少20个字符）")
	case "main_image":
		return s.makeDim(label, strings.TrimSpace(prod.MainImage) != "", prod.MainImage, "主图未设置")
	case "images":
		hasImages := len(prod.Images) > 2 && string(prod.Images) != "null"
		return s.makeDim(label, hasImages, string(prod.Images), "未设置多图或图片列表为空")
	case "category":
		return s.makeDim(label, prod.CategoryID != nil && *prod.CategoryID > 0, fmt.Sprintf("%v", prod.CategoryID), "类目未设置")
	case "brand":
		return s.makeDim(label, prod.BrandID != nil && *prod.BrandID > 0, fmt.Sprintf("%v", prod.BrandID), "品牌未设置")
	case "spec":
		hasSpec := len(prod.SpecJSON) > 2 && string(prod.SpecJSON) != "null"
		return s.makeDim(label, hasSpec, string(prod.SpecJSON), "规格参数未设置")
	case "purchase_price":
		return s.makeDim(label, prod.PurchasePrice > 0, fmt.Sprintf("%.2f", prod.PurchasePrice), "采购成本未设置或为0")
	case "package_info":
		hasWeight := prod.PackageWeightKg > 0
		hasDims := prod.PackageLengthCm > 0 && prod.PackageWidthCm > 0 && prod.PackageHeightCm > 0
		complete := hasWeight && hasDims
		if !complete {
			reason := "包装信息不完整"
			if !hasWeight {
				reason += "（缺重量）"
			}
			if !hasDims {
				reason += "（缺尺寸）"
			}
			return CompletenessDimension{Dimension: key, Label: label, Score: 0, Complete: false, Reason: reason}
		}
		return CompletenessDimension{Dimension: key, Label: label, Score: 100, Complete: true, Reason: ""}
	case "hs_code":
		return s.makeDim(label, strings.TrimSpace(prod.HSCode) != "", prod.HSCode, "HS编码未设置")
	case "target_sale_price":
		return s.makeDim(label, prod.TargetSalePrice > 0, fmt.Sprintf("%.2f", prod.TargetSalePrice), "目标售价未设置或为0")
	case "origin_country":
		v := strings.TrimSpace(prod.OriginCountry)
		if v == "" {
			v = "CN" // default is always set
		}
		return s.makeDim(label, v != "", v, "原产地未设置")
	default:
		return CompletenessDimension{Dimension: key, Label: label, Score: 0, Complete: false, Reason: "未知检查项"}
	}
}

func (s *Service) makeDim(label string, complete bool, val string, failReason string) CompletenessDimension {
	if complete {
		return CompletenessDimension{Label: label, Score: 100, Complete: true, Reason: ""}
	}
	return CompletenessDimension{Label: label, Score: 0, Complete: false, Reason: failReason}
}

// CheckEnhanced computes an enhanced completeness report including economic
// estimates from the profit service (cost, logistics, platform fee, margin).
func (s *Service) CheckEnhanced(productID int64, triggeredBy string) (*CompletenessReport, error) {
	var prod candidate.CandidateProduct
	if err := s.db.First(&prod, productID).Error; err != nil {
		return nil, err
	}

	// 1. Base info completeness (also persists a CompletenessCheck record).
	base, err := s.Check(productID, triggeredBy)
	if err != nil {
		return nil, err
	}
	baseScore := base.Score / 100.0

	// 2. Cost score: purchase price + target sale price
	costComplete := prod.PurchasePrice > 0 && prod.TargetSalePrice > 0
	costScore := 0.0
	if costComplete {
		costScore = 1.0
	}

	// 3. Logistics score: package weight + dimensions
	logComplete := prod.PackageWeightKg > 0 &&
		prod.PackageLengthCm > 0 && prod.PackageWidthCm > 0 && prod.PackageHeightCm > 0
	logisticsScore := 0.0
	if logComplete {
		logisticsScore = 1.0
	}

	// 4. Platform fee score: target platform set
	platformFeeScore := 0.0
	if prod.TargetPlatformID != nil && *prod.TargetPlatformID > 0 {
		platformFeeScore = 1.0
	}

	// 5. Profit score: average of economic subscores
	profitScore := (costScore + logisticsScore + platformFeeScore) / 3.0

	// 6. Overall: base 40%, cost 20%, logistics 15%, platform fee 10%, profit 15%
	overall := baseScore*0.40 + costScore*0.20 + logisticsScore*0.15 + platformFeeScore*0.10 + profitScore*0.15
	overall = math.Round(overall*100) / 100

	// 7. Collect economic missing fields
	var econMissing []string
	if prod.PurchasePrice <= 0 {
		econMissing = append(econMissing, "采购成本")
	}
	if prod.TargetSalePrice <= 0 {
		econMissing = append(econMissing, "目标售价")
	}
	if prod.PackageWeightKg <= 0 {
		econMissing = append(econMissing, "包装重量")
	}
	if prod.PackageLengthCm <= 0 || prod.PackageWidthCm <= 0 || prod.PackageHeightCm <= 0 {
		econMissing = append(econMissing, "包装尺寸")
	}
	if prod.TargetPlatformID == nil || *prod.TargetPlatformID <= 0 {
		econMissing = append(econMissing, "目标平台")
	}

	// Merge with base missing fields, dedup by label.
	seen := make(map[string]bool, len(base.MissingItems)+len(econMissing))
	allMissing := make([]string, 0, len(base.MissingItems)+len(econMissing))
	for _, m := range base.MissingItems {
		if !seen[m] {
			seen[m] = true
			allMissing = append(allMissing, m)
		}
	}
	for _, m := range econMissing {
		if !seen[m] {
			seen[m] = true
			allMissing = append(allMissing, m)
		}
	}

	// 8. Economic estimates via profit service (best-effort, nil ptrs on failure).
	var estProfit, estMargin, estLogistics, estPlatformFee *float64
	if s.profitSvc != nil {
		pr, err := s.profitSvc.Calculate(productID, triggeredBy)
		if err != nil {
			s.logger.Warn("CheckEnhanced: profit.Calculate failed, estimates omitted",
				zap.Int64("product_id", productID), zap.Error(err))
		} else {
			estProfit = &pr.EstimatedProfit
			estMargin = &pr.ProfitMargin
			estLogistics = &pr.ShippingCost
			estPlatformFee = &pr.PlatformFee
		}
	}

	return &CompletenessReport{
		CandidateID:          productID,
		BaseInfoScore:        math.Round(baseScore*100) / 100,
		CostScore:            costScore,
		LogisticsScore:       logisticsScore,
		PlatformFeeScore:     platformFeeScore,
		ProfitScore:          math.Round(profitScore*100) / 100,
		OverallScore:         overall,
		MissingFields:        allMissing,
		EstimatedProfit:      estProfit,
		EstimatedMargin:      estMargin,
		EstimatedLogistics:   estLogistics,
		EstimatedPlatformFee: estPlatformFee,
	}, nil
}

// ListChecks returns paginated completeness checks with filter by status.
func (s *Service) ListChecks(page, size int, status string) ([]CompletenessCheck, int64, error) {
	var items []CompletenessCheck
	var total int64
	q := s.db.Model(&CompletenessCheck{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
