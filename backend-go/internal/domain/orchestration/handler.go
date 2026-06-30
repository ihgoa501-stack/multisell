package orchestration

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

type Handler struct {
	orch *PipelineOrchestrator
}

func NewHandler(orch *PipelineOrchestrator) *Handler {
	return &Handler{orch: orch}
}

// GetPipelineStatus returns the current pipeline status for a product.
func (h *Handler) GetPipelineStatus(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || productID == 0 {
		response.Error(c, http.StatusBadRequest, "invalid product id")
		return
	}

	steps, err := h.orch.GetPipelineStatus(c.Request.Context(), productID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, steps)
}

// StartPipeline begins the lifecycle pipeline for a product.
func (h *Handler) StartPipeline(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || productID == 0 {
		response.Error(c, http.StatusBadRequest, "invalid product id")
		return
	}

	if err := h.orch.StartPipeline(c.Request.Context(), productID); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"status": "started", "product_id": productID})
}

// RetryStep retries a specific failed pipeline step.
func (h *Handler) RetryStep(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || productID == 0 {
		response.Error(c, http.StatusBadRequest, "invalid product id")
		return
	}

	step := c.Param("step")
	if step == "" {
		response.Error(c, http.StatusBadRequest, "step required")
		return
	}

	if err := h.orch.RetryStep(c.Request.Context(), productID, step); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"status": "retrying", "product_id": productID, "step": step})
}

// ListConfigs returns all orchestration configs.
func (h *Handler) ListConfigs(c *gin.Context) {
	configs, err := h.orch.ListConfigs(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, configs)
}

// CreateConfig creates a new orchestration config.
func (h *Handler) CreateConfig(c *gin.Context) {
	var cfg OrchestrationConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.orch.CreateConfig(c.Request.Context(), &cfg); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, cfg)
}
