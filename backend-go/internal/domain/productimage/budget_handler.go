package productimage

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

type BudgetHandler struct{ service *Service }

func NewBudgetHandler(s *Service) *BudgetHandler { return &BudgetHandler{service: s} }

func (h *BudgetHandler) CreatePolicy(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	var in BudgetPolicyInput
	if c.ShouldBindJSON(&in) != nil {
		problem(c, 422, "VALIDATION_ERROR", "invalid budget policy")
		return
	}
	out, err := h.service.CreateBudgetPolicy(c.Request.Context(), owner, in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.Result{Code: 0, Message: "created", Data: out})
}
func (h *BudgetHandler) ListPolicies(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	out, err := h.service.ListBudgetPolicies(c.Request.Context(), owner)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, out)
}
func (h *BudgetHandler) ListReservations(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	out, err := h.service.ListBudgetReservations(c.Request.Context(), owner)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, out)
}
func budgetReservationID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("reservation_id"), 10, 64)
	if err != nil || id <= 0 {
		problem(c, 400, "VALIDATION_ERROR", "invalid reservation id")
		return 0, false
	}
	return id, true
}
func (h *BudgetHandler) Cancel(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := budgetReservationID(c)
	if !ok {
		return
	}
	var in struct {
		Reason string `json:"reason" binding:"required"`
	}
	if c.ShouldBindJSON(&in) != nil {
		problem(c, 422, "VALIDATION_ERROR", "reason is required")
		return
	}
	out, err := h.service.ReleaseBudgetReservation(c.Request.Context(), owner, id, in.Reason)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, out)
}
func (h *BudgetHandler) Reconcile(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := budgetReservationID(c)
	if !ok {
		return
	}
	var in BudgetChargeInput
	if c.ShouldBindJSON(&in) != nil {
		problem(c, 422, "VALIDATION_ERROR", "invalid charge")
		return
	}
	out, err := h.service.ReconcileBudgetCharge(c.Request.Context(), owner, id, in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.Result{Code: 0, Message: "reconciled", Data: out})
}

func (h *BudgetHandler) ReconcileNoCharge(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	id, ok := budgetReservationID(c)
	if !ok {
		return
	}
	var in BudgetNoChargeInput
	if c.ShouldBindJSON(&in) != nil {
		problem(c, 422, "VALIDATION_ERROR", "invalid no-charge evidence")
		return
	}
	out, err := h.service.ReconcileBudgetNoCharge(c.Request.Context(), owner, id, in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.Result{Code: 0, Message: "reconciled", Data: out})
}
