package toolbridge

import (
	"testing"
)

// ---------------------------------------------------------------------------
// ToolCall — read-only and suggestion tools can be created without execution.
// ---------------------------------------------------------------------------

func TestToolCall_Creation_NoExecution(t *testing.T) {
	call := ToolCall{
		ToolName: "search_product",
		Version:  "1.0.0",
		Category: ToolCategoryRead,
		Mode:     ModeDryRun,
		Input:    map[string]interface{}{"query": "widget"},
	}
	if call.ToolName != "search_product" {
		t.Errorf("expected search_product, got %s", call.ToolName)
	}
}

func TestToolCall_Suggestion_Creation(t *testing.T) {
	call := ToolCall{
		ToolName: "analyze_price_trend",
		Version:  "1.0.0",
		Category: ToolCategorySuggestion,
		Mode:     ModeDryRun,
		Input:    map[string]interface{}{"sku_code": "SKU001"},
	}
	if call.Category != ToolCategorySuggestion {
		t.Errorf("expected suggestion category, got %v", call.Category)
	}
}

// ---------------------------------------------------------------------------
// ToolCall.Validate — mutation/production calls require approval.
// ---------------------------------------------------------------------------

func TestToolCall_Validate_ReadOnly_AlwaysAllowed(t *testing.T) {
	call := ToolCall{
		ToolName: "inspect_product",
		Category: ToolCategoryRead,
		Mode:     ModeProduction,
	}
	if err := call.Validate(); err != nil {
		t.Errorf("read-only tool should be allowed in production, got: %v", err)
	}
}

func TestToolCall_Validate_Suggestion_AlwaysAllowed(t *testing.T) {
	call := ToolCall{
		ToolName: "recommend_price",
		Category: ToolCategorySuggestion,
		Mode:     ModeProduction,
	}
	if err := call.Validate(); err != nil {
		t.Errorf("suggestion tool should be allowed in production, got: %v", err)
	}
}

func TestToolCall_Validate_Mutation_DryRun_Allowed(t *testing.T) {
	call := ToolCall{
		ToolName: "publish_listing",
		Category: ToolCategoryMutation,
		Mode:     ModeDryRun,
	}
	if err := call.Validate(); err != nil {
		t.Errorf("mutation tool should be allowed in dry_run mode, got: %v", err)
	}
}

func TestToolCall_Validate_Mutation_Sandbox_Allowed(t *testing.T) {
	call := ToolCall{
		ToolName: "sync_inventory",
		Category: ToolCategoryMutation,
		Mode:     ModeSandbox,
	}
	if err := call.Validate(); err != nil {
		t.Errorf("mutation tool should be allowed in sandbox mode, got: %v", err)
	}
}

func TestToolCall_Validate_Mutation_Production_RequiresApproval(t *testing.T) {
	call := ToolCall{
		ToolName:   "publish_listing",
		Category:   ToolCategoryMutation,
		Mode:       ModeProduction,
		ApprovalID: nil,
	}
	err := call.Validate()
	if err != ErrMutationRequiresApproval {
		t.Errorf("expected ErrMutationRequiresApproval, got %v", err)
	}
}

func TestToolCall_Validate_Mutation_Production_WithApproval_Allowed(t *testing.T) {
	approvalID := int64(42)
	call := ToolCall{
		ToolName:   "publish_listing",
		Category:   ToolCategoryMutation,
		Mode:       ModeProduction,
		ApprovalID: &approvalID,
	}
	if err := call.Validate(); err != nil {
		t.Errorf("mutation with approval should be allowed, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ToolCategory risk levels.
// ---------------------------------------------------------------------------

func TestToolCategoryRiskLevel(t *testing.T) {
	if ToolCategoryRead.RiskLevel() != 1 {
		t.Errorf("read risk level: expected 1, got %d", ToolCategoryRead.RiskLevel())
	}
	if ToolCategorySuggestion.RiskLevel() != 1 {
		t.Errorf("suggestion risk level: expected 1, got %d", ToolCategorySuggestion.RiskLevel())
	}
	if ToolCategoryMutation.RiskLevel() != 3 {
		t.Errorf("mutation risk level: expected 3, got %d", ToolCategoryMutation.RiskLevel())
	}
}

// ---------------------------------------------------------------------------
// Mode string representations.
// ---------------------------------------------------------------------------

func TestExecutionMode_String(t *testing.T) {
	tests := []struct {
		mode ExecutionMode
		want string
	}{
		{ModeDryRun, "dry_run"},
		{ModeSandbox, "sandbox"},
		{ModeProduction, "production"},
	}
	for _, tc := range tests {
		if got := tc.mode.String(); got != tc.want {
			t.Errorf("ExecutionMode(%d).String() = %q, want %q", tc.mode, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// ToolCategory string representations.
// ---------------------------------------------------------------------------

func TestToolCategory_String(t *testing.T) {
	tests := []struct {
		cat  ToolCategory
		want string
	}{
		{ToolCategoryRead, "read"},
		{ToolCategorySuggestion, "suggestion"},
		{ToolCategoryMutation, "mutation"},
	}
	for _, tc := range tests {
		if got := tc.cat.String(); got != tc.want {
			t.Errorf("ToolCategory(%d).String() = %q, want %q", tc.cat, got, tc.want)
		}
	}
}
