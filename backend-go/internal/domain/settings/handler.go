package settings

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles settings HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new settings handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// LLMConfig is the LLM settings payload returned to the frontend.
type LLMConfig struct {
	Model             string  `json:"model"`
	APIKeyPlaceholder string  `json:"api_key"`
	Temperature       float64 `json:"temperature"`
	MaxTokens         int     `json:"max_tokens"`
}

type UpdateLLMInput struct {
	Model       *string  `json:"model"`
	APIKey      *string  `json:"api_key"`
	Temperature *float64 `json:"temperature"`
	MaxTokens   *int     `json:"max_tokens"`
}

// GetLLM GET /settings/llm
func (h *Handler) GetLLM(c *gin.Context) {
	cfg := h.service.GetLLMConfig()
	response.Success(c, cfg)
}

// UpdateLLM PUT /settings/llm
func (h *Handler) UpdateLLM(c *gin.Context) {
	var in UpdateLLMInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	cfg := h.service.UpdateLLMConfig(&in)
	response.Success(c, cfg)
}
