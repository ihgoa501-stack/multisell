package metabolism

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"go.uber.org/zap"
)

// Handler handles metabolism HTTP requests.
type Handler struct {
	service *MetabolismService
	logger  *zap.Logger
}

// NewHandler creates a new metabolism handler.
func NewHandler(service *MetabolismService) *Handler {
	return &Handler{service: service, logger: service.logger}
}

// ListLogs GET /api/v1/metabolism
func (h *Handler) ListLogs(c *gin.Context) {
	p := common.ParsePagination(c)
	logs, total, err := h.service.ListLogs(p.Page, p.Size)
	if err != nil {
		h.logger.Error("metabolism: list logs error", zap.Error(err))
		response.InternalError(c, err)
		return
	}
	response.Paginated(c, logs, total, p.Page, p.Size)
}

// GetLog GET /api/v1/metabolism/:id
func (h *Handler) GetLog(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	log, err := h.service.GetLog(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "log not found")
		return
	}
	response.Success(c, log)
}

// DryRun POST /api/v1/metabolism/dry-run
func (h *Handler) DryRun(c *gin.Context) {
	if err := h.service.Execute(true); err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "dry-run completed"})
}
