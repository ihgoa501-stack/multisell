package tools

import "github.com/lingmirror/backend-go/internal/aios/toolregistry"

// AllTools returns every tool definition across all domains.
// This is the single entry point for registering all tools with the ToolRegistry.
func AllTools() []toolregistry.Tool {
	all := make([]toolregistry.Tool, 0)
	all = append(all, InventoryTools()...)
	all = append(all, PurchaseTools()...)
	all = append(all, FinanceTools()...)
	all = append(all, OrderTools()...)
	all = append(all, ListingTools()...)
	all = append(all, CustomerServiceTools()...)
	all = append(all, ShippingTools()...)
	all = append(all, SupplierTools()...)
	all = append(all, PlatformTools()...)
	all = append(all, ProductScoutTools()...)
	all = append(all, AdAdviceTools()...)
	all = append(all, DashboardTools()...)
	all = append(all, SourcingTools()...)
	return all
}

func init() {
	reg := toolregistry.DefaultRegistry
	if reg == nil {
		return
	}
	for _, t := range AllTools() {
		reg.Register(&t)
	}
}
