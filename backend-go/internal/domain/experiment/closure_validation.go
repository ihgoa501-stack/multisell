package experiment

import (
	"strings"
	"time"
)

// These pure predicates are shared by the profit gate and XiaoQ's read-only
// Owner view so both paths apply exactly the same trust boundary.
func isTrustedSettlement(status, sourceType string, importedAt *time.Time) bool {
	return (status == "reconciled" || status == "closed") &&
		(sourceType == "platform_import" || sourceType == "api_sync") &&
		importedAt != nil
}

func isFullyReconciledSettlement(itemCount, unmatchedCount, matchedOrderCount int64) bool {
	return itemCount > 0 && unmatchedCount == 0 && matchedOrderCount > 0
}

func isFinalProfitForOrder(profitOrderID, expectedOrderID int64, status, missingCosts string) bool {
	return profitOrderID == expectedOrderID && status == "final" && strings.TrimSpace(missingCosts) == ""
}
