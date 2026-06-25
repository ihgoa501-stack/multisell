package finance

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers finance routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	group := rg.Group("/finance")
	{
		// Static routes first to avoid conflict with /:id
		group.GET("/summary", h.Summary)
		group.GET("/profit-summary", h.ProfitSummary)
		group.GET("/ledger", h.ListLedger)
		group.POST("/mock", h.Mock)

		// Order-scoped ledger / profit (static sub-paths before any :id)
		group.GET("/orders/:order_id/ledger", h.ListOrderLedger)
		group.GET("/orders/:order_id/profit", h.OrderProfit)
		group.POST("/orders/:order_id/ledger/rebuild", h.RebuildOrderLedger)

		// Accounts
		group.GET("/accounts", h.ListAccounts)
		group.POST("/accounts", h.CreateAccount)
		group.GET("/accounts/:id", h.GetAccount)
		group.PUT("/accounts/:id", h.UpdateAccount)
		group.DELETE("/accounts/:id", h.DeleteAccount)

		// Transactions
		group.GET("/transactions", h.ListTransactions)
		group.POST("/transactions", h.CreateTransaction)
	}
}
