// Package actioncatalog provides the canonical registry of all action types
// with their risk, approval, and execution constraints. Every action an Agent
// or system proposes must be registered here before it can execute in
// production mode.
//
// Unknown action types fail closed — DispatchSafe rejects them in production.
package actioncatalog

import (
	"errors"
	"fmt"
)

// Risk level constants mirroring command.RiskLevel values to avoid import
// cycles. 0=none, 1=low, 2=medium, 3=high.
const (
	RiskNone   = 0
	RiskLow    = 1
	RiskMedium = 2
	RiskHigh   = 3
)

// AutonomyLevel represents the execution autonomy level of an action.
// Level3+ actions require approval in production. Level4 actions are
// blocked from autonomous execution by default.
type AutonomyLevel int

const (
	LevelUnknown AutonomyLevel = 0
	Level1       AutonomyLevel = 1 // read/alert — no approval needed
	Level2       AutonomyLevel = 2 // draft/suggestion — no approval needed
	Level3       AutonomyLevel = 3 // production mutation requiring approval
	Level4       AutonomyLevel = 4 // autonomous execution (disabled by default)
)

func (l AutonomyLevel) String() string {
	switch l {
	case LevelUnknown:
		return "unknown"
	case Level1:
		return "L1"
	case Level2:
		return "L2"
	case Level3:
		return "L3"
	case Level4:
		return "L4"
	default:
		return fmt.Sprintf("L%d", l)
	}
}

// Entry defines one action type's catalog specification.
type Entry struct {
	ActionType      string // e.g. "stock_alert", "price_update"
	Name            string // human-readable name
	Description     string // what this action does
	RiskLevel       int    // 1=low, 2=medium, 3=high (mirrors command.RiskLevel)
	Level           AutonomyLevel
	RequireApproval bool
	TargetTypes     []string
	HandlerRequired bool
	AutonomousBlocked bool
}

// Catalog is the canonical registry of all action types.
type Catalog struct {
	entries map[string]Entry
}

// New creates a Catalog from a list of entries. Panics on duplicate action types.
func New(entries []Entry) *Catalog {
	c := &Catalog{entries: make(map[string]Entry, len(entries))}
	for _, e := range entries {
		if _, ok := c.entries[e.ActionType]; ok {
			panic(fmt.Sprintf("actioncatalog: duplicate action type %q", e.ActionType))
		}
		c.entries[e.ActionType] = e
	}
	return c
}

// Lookup returns the entry for an action type. Returns false if unknown.
func (c *Catalog) Lookup(actionType string) (Entry, bool) {
	e, ok := c.entries[actionType]
	return e, ok
}

// Must panics if the action type is not in the catalog.
func (c *Catalog) Must(actionType string) Entry {
	e, ok := c.Lookup(actionType)
	if !ok {
		panic(fmt.Sprintf("actioncatalog: unknown action type %q", actionType))
	}
	return e
}

// ValidateProduction checks whether an action can execute in production mode.
// Returns nil if allowed, or an error describing why it is blocked.
func (c *Catalog) ValidateProduction(actionType string, riskLevel int, hasApproval bool) error {
	spec, ok := c.Lookup(actionType)
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownAction, actionType)
	}
	if spec.AutonomousBlocked {
		return fmt.Errorf("%w: action %q is Level 4 and blocked from unsupervised execution", ErrAutonomousBlocked, actionType)
	}
	if spec.RequireApproval && !hasApproval {
		return fmt.Errorf("%w: action %q requires approval for production execution", ErrApprovalRequired, actionType)
	}
	return nil
}

// HasAction returns true if the action type is registered in the catalog.
func (c *Catalog) HasAction(actionType string) bool {
	_, ok := c.entries[actionType]
	return ok
}

// Default returns the canonical action catalog for the system.
func Default() *Catalog {
	return New(DefaultEntries())
}

// DefaultEntries returns the full list of default catalog entries.
func DefaultEntries() []Entry {
	return []Entry{
		// ── Read / Alert (L1) ───────────────────────────────────────────
		{
			ActionType:      "stock_alert",
			Name:            "库存预警",
			Description:     "创建库存预警通知，不改库存。",
			RiskLevel:       RiskLow,
			Level:           Level1,
			RequireApproval: false,
			TargetTypes:     []string{"sku"},
			HandlerRequired: true,
		},
		{
			ActionType:      "system_health",
			Name:            "系统健康检查",
			Description:     "只读系统健康检查。",
			RiskLevel:       RiskLow,
			Level:           Level1,
			RequireApproval: false,
			TargetTypes:     []string{"system"},
			HandlerRequired: false,
		},
		{
			ActionType:      "dashboard_overview",
			Name:            "仪表盘概览",
			Description:     "只读仪表盘聚合数据。",
			RiskLevel:       RiskLow,
			Level:           Level1,
			RequireApproval: false,
			TargetTypes:     []string{"dashboard"},
			HandlerRequired: false,
		},
		{
			ActionType:      "auto_reply",
			Name:            "客服自动回复",
			Description:     "生成客服自动回复内容。",
			RiskLevel:       RiskLow,
			Level:           Level1,
			RequireApproval: false,
			TargetTypes:     []string{"order", "message"},
			HandlerRequired: false,
		},
		{
			ActionType:      "profit_watch",
			Name:            "利润监控",
			Description:     "SKU 利润分析监控。",
			RiskLevel:       RiskLow,
			Level:           Level1,
			RequireApproval: false,
			TargetTypes:     []string{"sku"},
			HandlerRequired: false,
		},

		// ── Draft / Suggestion (L2) ──────────────────────────────────────
		{
			ActionType:      "listing_optimize",
			Name:            "Listing 优化",
			Description:     "创建 listing draft 但不发布到外部平台。",
			RiskLevel:       RiskMedium,
			Level:           Level2,
			RequireApproval: false,
			TargetTypes:     []string{"listing", "sku"},
			HandlerRequired: true,
		},
		{
			ActionType:      "compliance_check",
			Name:            "合规检测",
			Description:     "标记不合规商品但不阻断刊登。命中高风险时升级至 L3。",
			RiskLevel:       RiskMedium,
			Level:           Level2,
			RequireApproval: false,
			TargetTypes:     []string{"product", "sku"},
			HandlerRequired: true,
		},
		{
			ActionType:      "replenish",
			Name:            "补货建议",
			Description:     "创建补货建议/采购请求。",
			RiskLevel:       RiskMedium,
			Level:           Level2,
			RequireApproval: false,
			TargetTypes:     []string{"sku"},
			HandlerRequired: true,
		},
		{
			ActionType:      "discount_risk_check",
			Name:            "折扣风控检查",
			Description:     "分析折扣风险但不修改价格。",
			RiskLevel:       RiskMedium,
			Level:           Level2,
			RequireApproval: false,
			TargetTypes:     []string{"price_rule"},
			HandlerRequired: false,
		},

		// ── Production Mutation (L3) — requires approval ────────────────
		{
			ActionType:       "price_update",
			Name:             "调价",
			Description:      "修改商品价格。高风险，必须审批。",
			RiskLevel:        RiskHigh,
			Level:            Level3,
			RequireApproval:  true,
			TargetTypes:      []string{"sku"},
			HandlerRequired:  true,
			AutonomousBlocked: false,
		},
		{
			ActionType:       "price_review",
			Name:             "价格审查（遗留）",
			Description:      "审查并修改价格。与 price_update 相同语义，生产必须审批。",
			RiskLevel:        RiskHigh,
			Level:            Level3,
			RequireApproval:  true,
			TargetTypes:      []string{"sku"},
			HandlerRequired:  true,
			AutonomousBlocked: false,
		},
		{
			ActionType:       "listing_publish",
			Name:             "发布 Listing",
			Description:      "将 listing draft 发布到外部平台。高风险，必须审批。",
			RiskLevel:        RiskHigh,
			Level:            Level3,
			RequireApproval:  true,
			TargetTypes:      []string{"listing"},
			HandlerRequired:  false,
			AutonomousBlocked: false,
		},
		{
			ActionType:       "inventory_change",
			Name:             "库存调整",
			Description:      "修改库存数量。高风险，必须审批。",
			RiskLevel:        RiskHigh,
			Level:            Level3,
			RequireApproval:  true,
			TargetTypes:      []string{"sku", "inventory"},
			HandlerRequired:  false,
			AutonomousBlocked: false,
		},

		// ── System Actions (L3) — event-bus triggered business mutations ──
		// These are registered system actions that execute as explicit,
		// documented, audited side effects of EventBus event handlers.
		// They bypass the agent-initiated action approval path because they
		// are deterministic system-internal state transitions, but they are
		// subject to the mutation guard audit layer and the EventBus
		// idempotency claim/state model.
		// Any addition here requires an update to the EventBus mutation
		// audit or the mutation guard in internal/platform/eventbus/guard.go.
		{
			ActionType:       "system.inventory.receive",
			Name:             "系统库存接收",
			Description:      "EventBus supplychain.order.received 触发的库存自动增加。采购入库确认后，系统自动增加对应 SKU 的库存数量。",
			RiskLevel:        RiskHigh,
			Level:            Level3,
			RequireApproval:  false, // system-internal, deterministic transition
			TargetTypes:      []string{"inventory", "sku"},
			HandlerRequired:  false,
			AutonomousBlocked: false,
		},
		{
			ActionType:       "system.inventory.aftersale_restock",
			Name:             "系统售后入库",
			Description:      "EventBus supplychain.aftersale.completed 触发的售后入库库存增加。退货完成后，系统自动增加对应 SKU 的库存数量。",
			RiskLevel:        RiskHigh,
			Level:            Level3,
			RequireApproval:  false,
			TargetTypes:      []string{"inventory", "sku"},
			HandlerRequired:  false,
			AutonomousBlocked: false,
		},

		// ── Autonomous Execution (L4) — blocked by default ─────────────
		{
			ActionType:        "auto_publish",
			Name:              "自动发布",
			Description:       "自动发布 listing 到外部平台。当前阶段强制阻止。",
			RiskLevel:         RiskHigh,
			Level:             Level4,
			RequireApproval:   true,
			TargetTypes:       []string{"listing"},
			HandlerRequired:   false,
			AutonomousBlocked: true,
		},
	}
}

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

var (
	// ErrUnknownAction is returned when an action type is not in the catalog.
	ErrUnknownAction = errors.New("actioncatalog: unknown action type")

	// ErrAutonomousBlocked is returned when a Level 4 action is
	// attempted without explicit ownership approval.
	ErrAutonomousBlocked = errors.New("actioncatalog: action is L4 blocked from unsupervised execution")

	// ErrApprovalRequired is returned when a production action requires
	// approval but none was provided.
	ErrApprovalRequired = errors.New("actioncatalog: action requires approval")
)
