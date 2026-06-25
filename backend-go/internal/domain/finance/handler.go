package finance

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles finance HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new finance handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func parseOptionalInt64(c *gin.Context, key string) *int64 {
	v := c.Query(key)
	if v == "" {
		return nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

// ---------- Accounts ----------

// ListAccounts GET /finance/accounts
func (h *Handler) ListAccounts(c *gin.Context) {
	p := common.ParsePagination(c)
	f := &AccountListFilter{
		Search:      c.Query("search"),
		AccountType: c.Query("account_type"),
		Status:      c.Query("status"),
	}
	items, total, err := h.service.ListAccounts(&p, f)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetAccount GET /finance/accounts/:id
func (h *Handler) GetAccount(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	a, err := h.service.GetAccount(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "account not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, a)
}

// CreateAccount POST /finance/accounts
func (h *Handler) CreateAccount(c *gin.Context) {
	var in CreateAccountInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	a, err := h.service.CreateAccount(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, a)
}

// UpdateAccount PUT /finance/accounts/:id
func (h *Handler) UpdateAccount(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdateAccountInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	a, err := h.service.UpdateAccount(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "account not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, a)
}

// DeleteAccount DELETE /finance/accounts/:id
func (h *Handler) DeleteAccount(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteAccount(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "account not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ---------- Transactions ----------

// ListTransactions GET /finance/transactions
func (h *Handler) ListTransactions(c *gin.Context) {
	p := common.ParsePagination(c)
	f := &TransactionListFilter{
		AccountID:       parseOptionalInt64(c, "account_id"),
		TransactionType: c.Query("transaction_type"),
		OrderID:         parseOptionalInt64(c, "order_id"),
	}
	items, total, err := h.service.ListTransactions(&p, f)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// CreateTransaction POST /finance/transactions
func (h *Handler) CreateTransaction(c *gin.Context) {
	var in CreateTransactionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	t, err := h.service.CreateTransaction(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, t)
}

// ---------- Ledger ----------

// ListLedger GET /finance/ledger
func (h *Handler) ListLedger(c *gin.Context) {
	p := common.ParsePagination(c)
	f := &LedgerListFilter{
		OrderID:   parseOptionalInt64(c, "order_id"),
		EntryType: c.Query("entry_type"),
		CostLayer: c.Query("cost_layer"),
	}
	items, total, err := h.service.ListLedgerEntries(&p, f)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// ---------- Summary ----------

// Summary GET /finance/summary
func (h *Handler) Summary(c *gin.Context) {
	sum, err := h.service.Summary()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, sum)
}

// ---------- Profit Summary / Order Ledger ----------

func parseOrderID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("order_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid order_id")
		return 0, false
	}
	return id, true
}

// ProfitSummary GET /finance/profit-summary
func (h *Handler) ProfitSummary(c *gin.Context) {
	f := &ProfitSummaryFilter{
		From:       c.Query("from"),
		To:         c.Query("to"),
		PlatformID: parseOptionalInt64(c, "platform_id"),
	}
	sum, err := h.service.ProfitSummary(f)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, sum)
}

// RebuildOrderLedger POST /finance/orders/:order_id/ledger/rebuild
func (h *Handler) RebuildOrderLedger(c *gin.Context) {
	orderID, ok := parseOrderID(c)
	if !ok {
		return
	}
	entries, err := h.service.RebuildOrderLedger(c.Request.Context(), orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "order not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"order_id": orderID, "entries": entries, "count": len(entries)})
}

// ListOrderLedger GET /finance/orders/:order_id/ledger
func (h *Handler) ListOrderLedger(c *gin.Context) {
	orderID, ok := parseOrderID(c)
	if !ok {
		return
	}
	items, err := h.service.ListOrderLedger(orderID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"order_id": orderID, "entries": items})
}

// OrderProfit GET /finance/orders/:order_id/profit
func (h *Handler) OrderProfit(c *gin.Context) {
	orderID, ok := parseOrderID(c)
	if !ok {
		return
	}
	p, err := h.service.OrderProfit(orderID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}


// ---------- Profit Calculation ----------

// CalculateProfit POST /finance/profit/calculate
func (h *Handler) CalculateProfit(c *gin.Context) {
	var in CalculateProfitInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.service.CalculateOrderProfit(c.Request.Context(), in.OrderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "order not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

// BatchCalculateProfit POST /finance/profit/batch-calculate
func (h *Handler) BatchCalculateProfit(c *gin.Context) {
	var in BatchCalculateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	since, err := time.Parse("2006-01-02", in.Since)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid since date (use YYYY-MM-DD)")
		return
	}
	until, err := time.Parse("2006-01-02", in.Until)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid until date (use YYYY-MM-DD)")
		return
	}
	count, err := h.service.BatchCalculate(c.Request.Context(), since, until)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"processed": count})
}

// GetProfitSummary GET /finance/profit/summary
func (h *Handler) GetProfitSummary(c *gin.Context) {
	sinceStr := c.Query("since")
	untilStr := c.Query("until")
	since, err := time.Parse("2006-01-02", sinceStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid since date (use YYYY-MM-DD)")
		return
	}
	until, err := time.Parse("2006-01-02", untilStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid until date (use YYYY-MM-DD)")
		return
	}
	sum, err := h.service.GetProfitSummary(c.Request.Context(), since, until)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, sum)
}

// GetSKUProfitRanking GET /finance/profit/ranking
func (h *Handler) GetSKUProfitRanking(c *gin.Context) {
	sinceStr := c.Query("since")
	since, err := time.Parse("2006-01-02", sinceStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid since date (use YYYY-MM-DD)")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	results, err := h.service.GetSKUProfitRanking(c.Request.Context(), since, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, results)
}

// Mock POST /finance/mock
// Mock POST /finance/mock
func (h *Handler) Mock(c *gin.Context) {
	var in MockDataInput
	_ = c.ShouldBindJSON(&in)
	count := in.Count
	if count <= 0 {
		count = 10
	}
	entries, err := h.service.Mock(count)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"count": len(entries), "entries": entries})
}
