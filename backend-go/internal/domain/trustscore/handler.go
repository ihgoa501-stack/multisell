package trustscore

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

type Handler struct {
	service *Service
	ug      *Upgrader
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
		ug:      NewUpgraderFromSvc(service),
	}
}

func (h *Handler) List(c *gin.Context) {
	scores, err := h.service.List()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, scores)
}

func (h *Handler) GetByAgent(c *gin.Context) {
	agentID := c.Param("agent_id")
	if agentID == "" {
		response.Error(c, http.StatusBadRequest, "agent_id required")
		return
	}
	score, err := h.service.GetByAgent(agentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if score == nil {
		response.Error(c, http.StatusNotFound, "trust score not found")
		return
	}
	response.Success(c, score)
}

func (h *Handler) Recalculate(c *gin.Context) {
	if err := h.service.Recalculate(); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"status": "ok"})
}

func (h *Handler) Eligible(c *gin.Context) {
	scores, err := h.service.GetEligibleForUpgrade()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, scores)
}

func (h *Handler) UpdateLevel(c *gin.Context) {
	agentID := c.Param("agent_id")
	if agentID == "" {
		response.Error(c, http.StatusBadRequest, "agent_id required")
		return
	}
	var in struct {
		Level string `json:"level" binding:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.UpdateAutonomyLevel(agentID, in.Level); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"status": "ok"})
}

func (h *Handler) AutoUpgrade(c *gin.Context) {
	results, err := h.ug.AutoUpgrade()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, results)
}

func (h *Handler) Summary(c *gin.Context) {
	items, err := h.ug.GetAutonomySummary()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}
