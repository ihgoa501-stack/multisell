package completeness

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides completeness check business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
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
