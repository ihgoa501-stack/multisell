package support

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles customer support HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new support handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseID(c *gin.Context) (int64, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

// ---------- Conversation handlers ----------

// ListConversations GET /support/conversations
func (h *Handler) ListConversations(c *gin.Context) {
	p := common.ParsePagination(c)
	var filter ConversationFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	items, total, err := h.service.ListConversations(&p, &filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetConversation GET /support/conversations/:id
func (h *Handler) GetConversation(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	conv, err := h.service.GetConversation(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "conversation not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, conv)
}

// CreateConversation POST /support/conversations
func (h *Handler) CreateConversation(c *gin.Context) {
	var in CreateConversationInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	conv, err := h.service.CreateConversation(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, conv)
}

// UpdateConversation PUT /support/conversations/:id
func (h *Handler) UpdateConversation(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdateConversationInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	conv, err := h.service.UpdateConversation(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "conversation not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, conv)
}

// DeleteConversation DELETE /support/conversations/:id
func (h *Handler) DeleteConversation(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteConversation(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "conversation not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// SendReply POST /support/conversations/:id/reply
func (h *Handler) SendReply(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in SendReplyInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	msg, err := h.service.SendReply(id, in.Content, in.IsAuto)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "conversation not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, msg)
}

// CloseConversation POST /support/conversations/:id/close
func (h *Handler) CloseConversation(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.CloseConversation(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "conversation not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// GetMessages GET /support/conversations/:id/messages
func (h *Handler) GetMessages(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	msgs, err := h.service.GetMessages(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, msgs)
}

// ---------- Template handlers ----------

// ListTemplates GET /support/templates
func (h *Handler) ListTemplates(c *gin.Context) {
	category := c.Query("category")
	platform := c.Query("platform")
	items, err := h.service.ListTemplates(category, platform)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}

// GetTemplate GET /support/templates/:id
func (h *Handler) GetTemplate(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	t, err := h.service.GetTemplate(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "template not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, t)
}

// CreateTemplate POST /support/templates
func (h *Handler) CreateTemplate(c *gin.Context) {
	var in CreateTemplateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	t, err := h.service.CreateTemplate(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, t)
}

// UpdateTemplate PUT /support/templates/:id
func (h *Handler) UpdateTemplate(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdateTemplateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	t, err := h.service.UpdateTemplate(id, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "template not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, t)
}

// DeleteTemplate DELETE /support/templates/:id
func (h *Handler) DeleteTemplate(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteTemplate(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "template not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ---------- Blacklist handlers ----------

// AddBlacklist POST /support/blacklist
func (h *Handler) AddBlacklist(c *gin.Context) {
	var in CreateBlacklistInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	entry, err := h.service.AddBlacklist(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, entry)
}

// CheckBlacklist GET /support/blacklist/check?email=xxx
func (h *Handler) CheckBlacklist(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		response.Error(c, http.StatusBadRequest, "email is required")
		return
	}
	blocked := h.service.CheckBlacklist(email)
	response.Success(c, gin.H{"email": email, "blocked": blocked})
}

// ListBlacklist GET /support/blacklist
func (h *Handler) ListBlacklist(c *gin.Context) {
	items, err := h.service.ListBlacklist()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, items)
}

// DeleteBlacklist DELETE /support/blacklist/:id
func (h *Handler) DeleteBlacklist(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteBlacklist(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "blacklist entry not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}
