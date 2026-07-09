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
		{
			ActionType:        "system.trustscore.auto_upgrade",
			Name:              "信任分自动升级",
			Description:       "EventBus scheduler.tick.trustscore 触发的信任分重算和Agent自主化等级自动升级。",
			RiskLevel:         RiskMedium,
			Level:             Level3,
			RequireApproval:   false,
			TargetTypes:       []string{"agent"},
			HandlerRequired:   false,
			AutonomousBlocked: false,
		},
		{
			ActionType:        "system.entropy.run_defenses",
			Name:              "熵防御运行",
			Description:       "EventBus scheduler.tick.entropy 触发的异常检测和Agent健康检查。",
			RiskLevel:         RiskMedium,
			Level:             Level3,
			RequireApproval:   false,
			TargetTypes:       []string{"agent"},
			HandlerRequired:   false,
			AutonomousBlocked: false,
		},
		{
			ActionType:        "system.approval.agent_decision_auto_create",
			Name:              "Agent决策自动创建审批",
			Description:       "EventBus agent.decided.* 触发的低置信度Agent决策自动生成审批请求。",
			RiskLevel:         RiskMedium,
			Level:             Level3,
			RequireApproval:   false,
			TargetTypes:       []string{"approval", "ai_trace"},
			HandlerRequired:   false,
			AutonomousBlocked: false,
		},
		{
			ActionType:        "system.integrations.ozon_order_sync",
			Name:              "Ozon订单同步",
			Description:       "EventBus scheduler.tick.ozon_sync 触发的定时从 Ozon 平台同步订单数据到本地数据库。",
			RiskLevel:         RiskLow,
			Level:             Level3,
			RequireApproval:   false,
			TargetTypes:       []string{"order", "integration"},
			HandlerRequired:   false,
			AutonomousBlocked: false,
		},
		{
			ActionType:        "system.supplychain.create_recommend_flow",
			Name:              "选品推荐创建供应链流",
			Description:       "EventBus sourcing.recommend 触发的创建供应链流水记录并触发物流报价。",
			RiskLevel:         RiskMedium,
			Level:             Level3,
			RequireApproval:   false,
			TargetTypes:       []string{"supplychain", "logistics"},
			HandlerRequired:   false,
			AutonomousBlocked: false,
		},
		{
			ActionType:        "system.supplychain.aftersale_return",
			Name:              "售后退货逆向物流",
			Description:       "EventBus supplychain.aftersale.returned 触发的创建逆向物流流水并记录退货跟踪数据。",
			RiskLevel:         RiskMedium,
			Level:             Level3,
			RequireApproval:   false,
			TargetTypes:       []string{"supplychain", "aftersales"},
			HandlerRequired:   false,
			AutonomousBlocked: false,
		},
		{
			ActionType:        "system.supplychain.stock_critical",
			Name:              "库存红色预警",
			Description:       "EventBus supplychain.stock.critical 触发的创建供应链流水并触发补货审批。",
			RiskLevel:         RiskHigh,
			Level:             Level3,
			RequireApproval:   false,
			TargetTypes:       []string{"supplychain", "inventory"},
			HandlerRequired:   false,
			AutonomousBlocked: false,
		},
		{
			ActionType:        "system.listingtask.approval_approved",
			Name:              "上架任务审批通过推进",
			Description:       "EventBus approval.approved.listing_task 推进 listing_task 到已批准状态。",
			RiskLevel:         RiskHigh,
			Level:             Level3,
			RequireApproval:   false,
			TargetTypes:       []string{"listing"},
			HandlerRequired:   false,
			AutonomousBlocked: false,
		},
		{
			ActionType:        "system.agentos.sla_escalation",
			Name:              "SLA过期升级",
			Description:       "EventBus scheduler.tick.agentos 触发的超时待审批动作自动升级处理。",
			RiskLevel:         RiskMedium,
			Level:             Level3,
			RequireApproval:   false,
			TargetTypes:       []string{"approval"},
			HandlerRequired:   false,
			AutonomousBlocked: false,
		},
		{
			ActionType:        "system.metabolism.excrete",
			Name:              "代谢排泄评分",
			Description:       "EventBus scheduler.tick.M1 触发的定时评分并清除不合格实体。",
			RiskLevel:         RiskMedium,
			Level:             Level3,
			RequireApproval:   false,
			TargetTypes:       []string{"entity"},
			HandlerRequired:   false,
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

		// ── Additional production mutations (L3) for P1 coverage ───────────
		{
			ActionType:       "order_cancel",
			Name:             "取消订单",
			Description:      "取消客户订单。高风险，必须审批。",
			RiskLevel:        RiskHigh,
			Level:            Level3,
			RequireApproval:  true,
			TargetTypes:      []string{"order"},
			HandlerRequired:  false,
		},
		{
			ActionType:       "refund_issue",
			Name:             "发起退款",
			Description:      "向客户发起退款。高风险，必须审批。",
			RiskLevel:        RiskHigh,
			Level:            Level3,
			RequireApproval:  true,
			TargetTypes:      []string{"order"},
			HandlerRequired:  false,
		},
		{
			ActionType:       "sync_inventory",
			Name:             "同步库存",
			Description:      "将库存同步到外部平台。高风险，必须审批。",
			RiskLevel:        RiskHigh,
			Level:            Level3,
			RequireApproval:  true,
			TargetTypes:      []string{"sku", "inventory"},
			HandlerRequired:  false,
		},
		{
			ActionType:       "credential_change",
			Name:             "修改凭证",
			Description:      "修改平台API凭证或密钥。高风险，必须审批。",
			RiskLevel:        RiskHigh,
			Level:            Level3,
			RequireApproval:  true,
			TargetTypes:      []string{"credential"},
			HandlerRequired:  false,
		},
		{
			ActionType:       "permission_change",
			Name:             "修改权限",
			Description:      "修改用户或Agent权限。高风险，必须审批。",
			RiskLevel:        RiskHigh,
			Level:            Level3,
			RequireApproval:  true,
			TargetTypes:      []string{"permission", "rbac"},
			HandlerRequired:  false,
		},
		{
			ActionType:       "destructive_data_change",
			Name:             "破坏性数据修改",
			Description:      "删除或批量修改关键业务数据。高风险，必须审批。",
			RiskLevel:        RiskHigh,
			Level:            Level3,
			RequireApproval:  true,
			TargetTypes:      []string{"*"},
			HandlerRequired:  false,
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
