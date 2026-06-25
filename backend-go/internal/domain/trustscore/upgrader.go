package trustscore

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Upgrader manages conditional autonomy upgrades based on trust scores.
type Upgrader struct {
	db      *gorm.DB
	logger  *zap.Logger
	svc     *Service
}

// NewUpgrader creates a new autonomy upgrader from raw deps.

// NewUpgraderFromSvc creates an upgrader from an existing Service.
func NewUpgraderFromSvc(svc *Service) *Upgrader {
	return newUpgrader(svc.db, svc.logger)
}

// NewUpgrader creates a new autonomy upgrader from raw deps.
func NewUpgrader(db *gorm.DB, logger *zap.Logger) *Upgrader {
	return newUpgrader(db, logger)
}

func newUpgrader(db *gorm.DB, logger *zap.Logger) *Upgrader {
	return &Upgrader{
		db:     db,
		logger: logger,
		svc:    NewService(db, logger),
	}
}

// UpgradeEligible agents to their target autonomy level.
// Returns a list of agents that were upgraded.
func (u *Upgrader) UpgradeEligible() ([]UpgradeResult, error) {
	eligible, err := u.svc.GetEligibleForUpgrade()
	if err != nil {
		return nil, err
	}

	var results []UpgradeResult
	for _, score := range eligible {
		if score.TargetLevel == "" || score.TargetLevel == score.AutonomyLevel {
			continue
		}

		// Perform upgrade
		if err := u.svc.UpdateAutonomyLevel(score.AgentID, score.TargetLevel); err != nil {
			u.logger.Warn("upgrade failed", zap.String("agent", score.AgentID), zap.Error(err))
			continue
		}

		result := UpgradeResult{
			AgentID:      score.AgentID,
			FromLevel:    score.AutonomyLevel,
			ToLevel:      score.TargetLevel,
			TrustScore:   score.TrustScore,
		}
		results = append(results, result)

		u.logger.Info("agent autonomy upgraded",
			zap.String("agent", score.AgentID),
			zap.String("from", score.AutonomyLevel),
			zap.String("to", score.TargetLevel),
			zap.Float64("trust_score", score.TrustScore),
		)
	}
	return results, nil
}

// UpgradeResult describes one autonomy upgrade.
type UpgradeResult struct {
	AgentID    string  `json:"agent_id"`
	FromLevel  string  `json:"from_level"`
	ToLevel    string  `json:"to_level"`
	TrustScore float64 `json:"trust_score"`
}

func (r UpgradeResult) String() string {
	return fmt.Sprintf("%s: %s → %s (trust=%.2f)", r.AgentID, r.FromLevel, r.ToLevel, r.TrustScore)
}

// AutoUpgrade tick runs on every action complete: recalculate trust scores,
// then auto-upgrade any eligible agents.
func (u *Upgrader) AutoUpgrade() ([]UpgradeResult, error) {
	// Step 1: recalculate all trust scores from action history
	if err := u.svc.Recalculate(); err != nil {
		return nil, fmt.Errorf("recalculate: %w", err)
	}
	// Step 2: upgrade eligible agents
	return u.UpgradeEligible()
}

// GetAutonomySummary returns the current state of all agents with their autonomy info.
func (u *Upgrader) GetAutonomySummary() ([]AutonomySummaryItem, error) {
	scores, err := u.svc.List()
	if err != nil {
		return nil, err
	}
	registry := getRegistry() // from model.go
	items := make([]AutonomySummaryItem, 0, len(scores))
	for _, s := range scores {
		item := AutonomySummaryItem{
			AgentID:        s.AgentID,
			AgentName:      s.AgentName,
			SquadID:        s.SquadID,
			CurrentLevel:   s.AutonomyLevel,
			TargetLevel:    s.TargetLevel,
			TrustScore:     s.TrustScore,
			AdoptionRate:   s.AdoptionRate,
			ExecSuccess:    s.ExecutionSuccess,
			AvgConfidence:  s.AvgConfidence,
			TotalActions:   s.TotalActions,
			NextThreshold:  nextThreshold(s.AutonomyLevel),
		}
		if spec, ok := registry[s.AgentID]; ok {
			item.Description = spec.Description
			item.Capabilities = spec.DecisionPoints
		}
		items = append(items, item)
	}
	return items, nil
}

// AutonomySummaryItem is a display-friendly autonomy summary.
type AutonomySummaryItem struct {
	AgentID       string   `json:"agent_id"`
	AgentName     string   `json:"agent_name"`
	SquadID       string   `json:"squad_id"`
	Description   string   `json:"description,omitempty"`
	Capabilities  []string `json:"capabilities,omitempty"`
	CurrentLevel  string   `json:"current_level"`
	TargetLevel   string   `json:"target_level"`
	TrustScore    float64  `json:"trust_score"`
	AdoptionRate  float64  `json:"adoption_rate"`
	ExecSuccess   float64  `json:"exec_success"`
	AvgConfidence float64  `json:"avg_confidence"`
	TotalActions  int      `json:"total_actions"`
	NextThreshold float64  `json:"next_threshold"`
}

func nextThreshold(current string) float64 {
	levels := []string{"advisory", "guided", "supervised", "autonomous"}
	found := false
	for _, l := range levels {
		if found {
			return AutonomyThresholds[l]
		}
		if l == current {
			found = true
		}
	}
	return 0
}

// getRegistry returns a minimal agent spec map for display.
func getRegistry() map[string]struct {
	Description   string
	DecisionPoints []string
} {
	return map[string]struct {
		Description   string
		DecisionPoints []string
	}{
		"A1": {"选品探路：市场机会扫描、新品推荐", []string{"product_scout", "market_analysis"}},
		"A2": {"Listing 优化：标题/描述/关键词", []string{"listing_optimize", "keyword_research"}},
		"A3": {"广告建议：ACOS 分析、投放优化", []string{"acos_analysis", "ad_optimization"}},
		"A4": {"客服自动化：意图分类、自动回复", []string{"auto_reply", "intent_classify"}},
		"A5": {"库存预警：缺货/补货/物流切换", []string{"stock_alert", "replenishment_plan"}},
		"A6": {"利润看护：SKU 利润率、成本优化", []string{"profit_check", "cost_optimization"}},
		"A7": {"合规守门：商品合规、认证查询", []string{"compliance_check", "certification_lookup"}},
		"G1": {"驾驶舱聚合：全局指标、趋势", []string{"dashboard_overview"}},
		"G2": {"仓储报关：仓库路由、报关单校验", []string{"warehouse_routing", "customs_declare"}},
		"G3": {"折扣风控：促销折扣风险、价格底线", []string{"discount_check", "promotion_validation"}},
	}
}
