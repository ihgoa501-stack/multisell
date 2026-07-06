package costcontrol

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ActionResult describes what the budget system decided for a request.
type ActionResult string

const (
	ActionAllow    ActionResult = "allow"
	ActionDowngrade ActionResult = "downgrade"
	ActionBlock    ActionResult = "block"
)

// Result is returned by Allow.
type Result struct {
	Action    ActionResult `json:"action"`
	Reason    string       `json:"reason"`
	Cheapest  string       `json:"cheapest,omitempty"` // set when downgraded
	DailySpent float64     `json:"daily_spent_usd"`
	BurstSpent float64     `json:"burst_spent_usd"`
}

// Controller manages LLM spend budgets across global and burst windows.
type Controller struct {
	db       *gorm.DB
	logger   *zap.Logger

	mu          sync.RWMutex
	dailyCapUSD float64               // 0 = unlimited
	burstWindow time.Duration
	burstMult   float64               // burst allowed as multiple of expected rate

	burstDetector *BurstDetector

	// cached daily spend (refreshed on each Allow call)
	todayDate string
	todayCost float64
}

// NewController creates a budget controller.
//
//   dailyCapUSD: maximum allowed spend per day (0 = unlimited).
//   burstWindow: sliding window for burst detection (e.g. 5 minutes).
//   burstMult:   burst threshold as multiple of expected hourly rate from daily cap.
//                burst allowed = (dailyCapUSD / 24) * (burstWindow / 1h) * burstMult
func NewController(db *gorm.DB, logger *zap.Logger, dailyCapUSD float64, burstWindow time.Duration, burstMult float64) *Controller {
	return &Controller{
		db:            db,
		logger:        logger,
		dailyCapUSD:   dailyCapUSD,
		burstWindow:   burstWindow,
		burstMult:     burstMult,
		burstDetector: NewBurstDetector(burstWindow),
	}
}

// Allow checks whether an LLM call should proceed, be downgraded, or blocked.
//
// Returns:
//   - ActionAllow: call the LLM normally.
//   - ActionDowngrade + Cheapest "claude-haiku-4": call but force the cheapest model.
//   - ActionBlock: reject the call.
func (c *Controller) Allow(ctx context.Context, req AllowInput) (*Result, error) {
	if c.dailyCapUSD <= 0 {
		// Unlimited mode — allow everything.
		return &Result{Action: ActionAllow, Reason: "unlimited"}, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Refresh daily spend from DB (lazy).
	today := time.Now().UTC().Format("2006-01-02")
	if today != c.todayDate {
		s, err := DailySpend(c.db)
		if err != nil {
			c.logger.Warn("costcontrol: failed to read daily spend", zap.Error(err))
		} else {
			c.todayCost = s.TotalCost
		}
		c.todayDate = today
	}

	// 1. Daily cap check.
	remaining := c.dailyCapUSD - c.todayCost
	if remaining <= 0 {
		c.logger.Warn("costcontrol: daily budget exceeded",
			zap.Float64("daily_spent", c.todayCost),
			zap.Float64("daily_cap", c.dailyCapUSD),
			zap.String("agent_id", req.AgentID),
		)
		return &Result{
			Action:     ActionBlock,
			Reason:     fmt.Sprintf("daily budget exhausted: $%.4f / $%.4f", c.todayCost, c.dailyCapUSD),
			DailySpent: c.todayCost,
		}, nil
	}

	// 2. Burst detection.
	burstAllowance := c.dailyCapUSD / 24.0 * c.burstWindow.Hours() * c.burstMult
	burstSpent := c.burstDetector.SpendLast()
	if burstSpent > burstAllowance {
		// Burst — force to cheapest model.
		return &Result{
			Action:     ActionDowngrade,
			Reason:     fmt.Sprintf("burst detected: $%.4f in %.0fm exceeds $%.4f allowance", burstSpent, c.burstWindow.Minutes(), burstAllowance),
			Cheapest:   "claude-haiku-4",
			DailySpent: c.todayCost,
			BurstSpent: burstSpent,
		}, nil
	}

	return &Result{
		Action:     ActionAllow,
		Reason:     fmt.Sprintf("$%.4f / $%.4f remaining", remaining, c.dailyCapUSD),
		DailySpent: c.todayCost,
		BurstSpent: burstSpent,
	}, nil
}

// AllowInput captures the LLM request metadata for budget decisions.
type AllowInput struct {
	AgentID string
	Model   string
	Tokens  int
}

// Record logs a completed LLM call's cost to the DB and burst detector.
func (c *Controller) Record(ctx context.Context, r RecordInput) error {
	// Always record in burst detector (no-op if unlimited, still tracks rate).
	c.burstDetector.Record(r.CostUSD)

	if c.dailyCapUSD <= 0 {
		// Unlimited mode — skip DB write.
		return nil
	}

	// Round cost to nearest cent for readable logs and to limit table size.
	cost := math.Round(r.CostUSD*100) / 100
	if cost <= 0 {
		return nil
	}

	log := CostLog{
		UserID:      r.UserID,
		AgentID:     r.AgentID,
		Model:       r.Model,
		TokensIn:    r.TokensIn,
		TokensOut:   r.TokensOut,
		CostUSD:     cost,
		RequestHash: r.RequestHash,
		Cached:      r.Cached,
	}
	if err := c.db.Create(&log).Error; err != nil {
		c.logger.Warn("costcontrol: failed to write cost log", zap.Error(err))
		return err
	}
	return nil
}

// RecordInput captures the LLM response cost data for logging.
type RecordInput struct {
	UserID      int64
	AgentID     string
	Model       string
	TokensIn    int
	TokensOut   int
	CostUSD     float64
	RequestHash string
	Cached      bool
}
