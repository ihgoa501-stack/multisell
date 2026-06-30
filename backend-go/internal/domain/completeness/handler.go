package completeness

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles completeness check HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new completeness handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Check POST /completeness/check/:productId
func (h *Handler) Check(c *gin.Context) {
	productIDStr := c.Param("productId")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid productId")
		return
	}

	var in CheckInput
	c.ShouldBindJSON(&in) // optional body

	result, err := h.service.Check(productID, in.TriggeredBy)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// ListChecks GET /completeness/checks
func (h *Handler) ListChecks(c *gin.Context) {
	p := common.ParsePagination(c)
	status := c.Query("status")
	items, total, err := h.service.ListChecks(p.Page, p.Size, status)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}
