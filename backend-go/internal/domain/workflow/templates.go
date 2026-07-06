package workflow

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// ── Standard workflow templates ────────────────────────────────────────

// ProductListingWorkflow returns the standard product listing template.
// Chain: Event → GenerateSuggestion → Approval → CreatePublishTask → ReviewResult
func ProductListingWorkflow() WorkflowDef {
	return WorkflowDef{
		Name:        "Product Listing Workflow",
		Description: "Standard product listing: await product ready event, generate suggestion via A2, wait for approval, publish, then review.",
		Steps: `[{"name":"receive_event","type":"event","wait_for_event":"product.ready","timeout_seconds":86400},{"name":"generate_suggestion","type":"agent","agent_id":"A2","decision_point":"listing_suggestion","timeout_seconds":300},{"name":"approval","type":"event","wait_for_event":"approval.response","timeout_seconds":86400},{"name":"publish","type":"command","command":"listing_publish","timeout_seconds":60},{"name":"review_result","type":"agent","agent_id":"A2","decision_point":"listing_review","timeout_seconds":300}]`,
	}
}

// OrderProfitWorkflow returns the standard order profit template.
// Chain: Event → CalculateProfit → Condition(profit<0 → Exception) → Approval
func OrderProfitWorkflow() WorkflowDef {
	return WorkflowDef{
		Name:        "Order Profit Workflow",
		Description: "Order profit calculation: receive order, calculate profit via A6, handle exceptions if profit negative, then final approval.",
		Steps: `[{"name":"receive_order","type":"event","wait_for_event":"order.placed","timeout_seconds":86400},{"name":"calculate_profit","type":"agent","agent_id":"A6","decision_point":"profit_calculation","timeout_seconds":300},{"name":"exception_handler","type":"event","wait_for_event":"exception.resolved","condition":"$steps.calculate_profit.status == \"completed\"","timeout_seconds":3600},{"name":"approval","type":"event","wait_for_event":"approval.response","timeout_seconds":86400}]`,
	}
}

// SeedTemplates creates default workflow templates if they don't exist.
func (e *Engine) SeedTemplates(ctx context.Context) error {
	templates := []WorkflowDef{
		ProductListingWorkflow(),
		OrderProfitWorkflow(),
	}

	for _, t := range templates {
		var count int64
		e.db.WithContext(ctx).Model(&WorkflowDef{}).Where("name = ?", t.Name).Count(&count)
		if count > 0 {
			continue
		}
		if err := e.db.WithContext(ctx).Create(&t).Error; err != nil {
			return fmt.Errorf("seed template %q: %w", t.Name, err)
		}
		e.logger.Info("seeded workflow template", zap.String("name", t.Name))
	}
	return nil
}
