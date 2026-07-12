package sourcing1688

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

func (h *Handler) ListSourcingCostVersions(c *gin.Context) {
	sourceID, ok := parseID(c)
	if !ok {
		return
	}
	ownerID, ok := h.requireSourceOwner(c, sourceID)
	if !ok {
		return
	}
	items, err := h.service.ListSourcingCostVersions(ownerID, sourceID)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *Handler) CreateSourcingCostVersion(c *gin.Context) {
	sourceID, ok := parseID(c)
	if !ok {
		return
	}
	ownerID, ok := h.requireSourceOwner(c, sourceID)
	if !ok {
		return
	}
	var in CreateSourcingCostVersionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.CreateSourcingCostVersion(ownerID, sourceID, &in)
	if err != nil {
		workflowError(c, err)
		return
	}
	response.Success(c, item)
}
