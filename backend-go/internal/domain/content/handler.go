package content

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles content HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new content handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GenerateContent POST /content/generate
func (h *Handler) GenerateContent(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.service.Generate(&req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// ValidateContent POST /content/validate
func (h *Handler) ValidateContent(c *gin.Context) {
	var req ValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	// Build a GeneratedContent from the validate request for the validator.
	content := &GeneratedContent{
		Title:       req.Title,
		Description: req.Description,
	}
	review, err := h.service.Validate(content, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, review)
}
