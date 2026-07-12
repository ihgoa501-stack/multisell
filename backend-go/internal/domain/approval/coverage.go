package approval

var requestTypeActions = map[string]map[string]struct{}{
	"publish":                 {"auto_publish": {}, "listing_optimize": {}},
	"price_change":            {"price_update": {}},
	"listing_task":            {"listing_optimize": {}},
	"list_generation":         {"listing_optimize": {}, "auto_publish": {}},
	"order_update":            {"order_cancel": {}},
	"order_cancel":            {"order_cancel": {}},
	"refund":                  {"refund_issue": {}},
	"sync_inventory":          {"sync_inventory": {}},
	"credential":              {"credential_change": {}},
	"credential_change":       {"credential_change": {}},
	"permission":              {"permission_change": {}},
	"permission_change":       {"permission_change": {}},
	"finance":                 {"destructive_data_change": {}},
	"settlement":              {"destructive_data_change": {}},
	"content_update":          {"listing_optimize": {}},
	"delist":                  {"listing_optimize": {}},
	"agent_action":            {"agent_approve": {}},
	"destructive_data_change": {"destructive_data_change": {}},
	"sourcing_1688_publish":   {"auto_publish": {}},
}

// RequestTypeCoversAction is the canonical, fail-closed bridge from a human
// approval request type to an executable action catalog type.
func RequestTypeCoversAction(requestType, actionType string) bool {
	actions, ok := requestTypeActions[requestType]
	if !ok {
		return false
	}
	_, ok = actions[actionType]
	return ok
}
