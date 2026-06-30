package consolidation

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// Handler handles HTTP requests for consolidation.
type Handler struct {
	svc *ConsolidationService
}

// NewHandler creates a new consolidation handler.
func NewHandler(svc *ConsolidationService) *Handler {
	return &Handler{svc: svc}
}

// ---------------------------------------------------------------------------
// POST /consolidation/groups
// ---------------------------------------------------------------------------

// CreateGroup creates a new consolidation group.
func (h *Handler) CreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	group, err := h.svc.CreateGroup(req.Destination, req.TimeWindowH)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.Success(c, group)
}

// ---------------------------------------------------------------------------
// GET /consolidation/groups
// ---------------------------------------------------------------------------

// ListGroups returns all consolidation groups.
func (h *Handler) ListGroups(c *gin.Context) {
	groups, err := h.svc.ListGroups()
	if err != nil {
		response.InternalError(c, err)
		return
	}

	if groups == nil {
		groups = []ConsolidationGroup{}
	}

	response.Success(c, groups)
}

// ---------------------------------------------------------------------------
// GET /consolidation/groups/:groupId
// ---------------------------------------------------------------------------

// GetGroup returns a single consolidation group.
func (h *Handler) GetGroup(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("groupId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid group_id")
		return
	}

	group, err := h.svc.GetGroup(groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "consolidation group not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.Success(c, group)
}

// ---------------------------------------------------------------------------
// POST /consolidation/groups/:groupId/items
// ---------------------------------------------------------------------------

// AddItem adds an item to a consolidation group.
func (h *Handler) AddItem(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("groupId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid group_id")
		return
	}

	var req AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	item, err := h.svc.AddItem(groupID, req.SkuID, req.WeightKg, req.VolumeM3, req.Destination)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.Success(c, item)
}

// ---------------------------------------------------------------------------
// GET /consolidation/groups/:groupId/items
// ---------------------------------------------------------------------------

// GetGroupItems returns all items for a consolidation group.
func (h *Handler) GetGroupItems(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("groupId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid group_id")
		return
	}

	items, err := h.svc.GetItemsByGroup(groupID)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	if items == nil {
		items = []ConsolidationItem{}
	}

	response.Success(c, items)
}

// ---------------------------------------------------------------------------
// DELETE /consolidation/groups/:groupId/items/:itemId
// ---------------------------------------------------------------------------

// RemoveItem removes an item from a consolidation group.
func (h *Handler) RemoveItem(c *gin.Context) {
	itemID, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid item_id")
		return
	}

	if err := h.svc.RemoveItem(itemID); err != nil {
		if err.Error() == "consolidation: item already removed" {
			response.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		response.InternalError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "item removed"})
}

// ---------------------------------------------------------------------------
// POST /consolidation/groups/:groupId/negotiate
// ---------------------------------------------------------------------------

// NegotiateGroup triggers rate negotiation (discount calculation) for a group.
func (h *Handler) NegotiateGroup(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("groupId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid group_id")
		return
	}

	group, err := h.svc.Negotiate(groupID)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	result := NegotiateResult{
		Group:         *group,
		DiscountRate:  group.DiscountRate,
		TotalWeightKg: group.TotalWeightKg,
		DiscountLabel: DiscountLabel(group.TotalWeightKg),
	}

	response.Success(c, result)
}
