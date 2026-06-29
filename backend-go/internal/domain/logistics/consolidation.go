package logistics

import (
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// ConsolidationConfig — 集单议价规则配置
// ---------------------------------------------------------------------------

// ConsolidationConfig 集单议价规则配置
type ConsolidationConfig struct {
	ID               int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SourceCountry    string     `gorm:"column:source_country" json:"source_country"`                                // 发货国
	Destination      string     `gorm:"column:destination" json:"destination"`                                     // 目的国
	Category         string     `gorm:"column:category" json:"category"`                                           // 品类(可选)
	MinTotalWeightKg float64    `gorm:"column:min_total_weight_kg" json:"min_total_weight_kg"`                     // 最低聚合重量
	NegotiatedRate   float64    `gorm:"column:negotiated_rate" json:"negotiated_rate"`                             // 议定价(CNY/kg)
	EffectiveFrom    time.Time  `gorm:"column:effective_from" json:"effective_from"`                               // 生效时间
	EffectiveTo      *time.Time `gorm:"column:effective_to" json:"effective_to,omitempty"`                         // 失效时间(可选)
	CreatedAt        time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (ConsolidationConfig) TableName() string { return "consolidation_config" }

// ---------------------------------------------------------------------------
// ConsolidationItem — 单次聚合中的一个发货项
// ---------------------------------------------------------------------------

// ConsolidationItem 单次聚合中的一个发货项
type ConsolidationItem struct {
	SellerID int64   // 卖家ID
	SkuID    int64   // SKU ID
	WeightKg float64 // 重量(kg)
	VolumeM3 float64 // 体积(m³)
}

// ---------------------------------------------------------------------------
// ConsolidationResult — 聚合议价结果
// ---------------------------------------------------------------------------

// ConsolidationResult 聚合议价结果
type ConsolidationResult struct {
	TotalWeightKg  float64
	MatchedConfig  *ConsolidationConfig
	NegotiatedRate float64 // 议价费率(CNY/kg)
	OriginalRate   float64 // 原始费率(无聚合)
	Savings        float64 // 节省金额
	SavingsPct     float64 // 节省百分比
}

// ---------------------------------------------------------------------------
// ConsolidationService — 集单议价服务
// ---------------------------------------------------------------------------

// DefaultStandardRate 是默认无集单时的标准运费(CNY/kg)。
// 当无法从费率表获取原始费率时,以此值作为基准计算节省金额。
const DefaultStandardRate = 100.0

// ConsolidationService 集单议价服务
type ConsolidationService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewConsolidationService 创建集单议价服务。
func NewConsolidationService(db *gorm.DB, logger *zap.Logger) *ConsolidationService {
	return &ConsolidationService{
		db:     db,
		logger: logger,
	}
}

// Evaluate 聚合 items 总重 → 查找匹配 ConsolidationConfig → 计算节省。
//
// 匹配规则:
//   - destination 必须匹配
//   - items 总重 >= min_total_weight_kg
//   - effective_from <= 当前时间
//   - effective_to 为空或 >= 当前时间
//   - 多个匹配时取 NegotiatedRate 最低的规则(最优费率优先)
func (s *ConsolidationService) Evaluate(items []ConsolidationItem, destination string) (*ConsolidationResult, error) {
	// 计算总重
	var totalWeightKg float64
	for _, item := range items {
		totalWeightKg += item.WeightKg
	}

	// 查询匹配的配置
	now := time.Now()
	var configs []ConsolidationConfig
	err := s.db.Where(
		"destination = ? AND min_total_weight_kg <= ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to >= ?)",
		destination, totalWeightKg, now, now,
	).Order("negotiated_rate ASC").Find(&configs).Error
	if err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		return nil, errors.New("consolidation: no matching consolidation config for the given parameters")
	}

	// 取 NegotiatedRate 最低的配置 (第一条,因 ORDER BY negotiated_rate ASC)
	matched := configs[0]

	negotiatedRate := matched.NegotiatedRate
	originalRate := DefaultStandardRate

	// 节省金额 = (原始费率 - 议价费率) * 总重
	savings := (originalRate - negotiatedRate) * totalWeightKg
	var savingsPct float64
	if originalRate > 0 && totalWeightKg > 0 {
		savingsPct = (savings / (originalRate * totalWeightKg)) * 100
	}

	s.logger.Info("consolidation evaluated",
		zap.String("destination", destination),
		zap.Float64("total_weight_kg", totalWeightKg),
		zap.Float64("negotiated_rate", negotiatedRate),
		zap.Float64("savings", savings),
	)

	return &ConsolidationResult{
		TotalWeightKg:  totalWeightKg,
		MatchedConfig:  &matched,
		NegotiatedRate: negotiatedRate,
		OriginalRate:   originalRate,
		Savings:        savings,
		SavingsPct:     savingsPct,
	}, nil
}
