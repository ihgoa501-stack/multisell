package pipeline

import "time"

// DefaultEdges defines the standard pipeline chain edges that replace the
// inline event bus handlers previously in router.go.
//
// Current edges:
//  1. A5 stock_alert (stock_status=red)           -> G3 discount_risk_check
//  2. G3 discount_risk_check (action=block)        -> A6 profit_watch
//  3. A6 profit_watch (is_loss=true)               -> A2 listing_optimize
//  4. A6 profit_watch (below_threshold=true)       -> A2 listing_optimize
//  5. G0 system_health (anomaly_count > 3)         -> G1 dashboard_overview
var DefaultEdges = []PipelineEdge{
	{
		SourceTopic: "agent.decided.A5.stock_alert",
		Condition: Condition{
			Field:  "stock_status",
			Equals: "red",
		},
		TargetAgent: "G3",
		TargetDP:    "discount_risk_check",
		Timeout:     30 * time.Second,
		Priority:    1,
		MaxRetries:  1,
	},
	{
		SourceTopic: "agent.decided.G3.discount_risk_check",
		Condition: Condition{
			Field:  "action",
			Equals: "block",
		},
		TargetAgent: "A6",
		TargetDP:    "profit_watch",
		Timeout:     30 * time.Second,
		Priority:    1,
	},
	{
		SourceTopic: "agent.decided.A6.profit_watch",
		Condition: Condition{
			Field:      "is_loss",
			BoolEquals: boolPtr(true),
		},
		TargetAgent: "A2",
		TargetDP:    "listing_optimize",
		Timeout:     30 * time.Second,
		Priority:    1,
	},
	{
		SourceTopic: "agent.decided.A6.profit_watch",
		Condition: Condition{
			Field:      "below_threshold",
			BoolEquals: boolPtr(true),
		},
		TargetAgent: "A2",
		TargetDP:    "listing_optimize",
		Timeout:     30 * time.Second,
		Priority:    1,
	},
	{
		SourceTopic: "agent.decided.G0.system_health",
		Condition: Condition{
			Field: "anomaly_count",
			GT:    3,
		},
		TargetAgent: "G1",
		TargetDP:    "dashboard_overview",
		Timeout:     30 * time.Second,
		Priority:    1,
	},
}

func boolPtr(v bool) *bool {
	return &v
}
