package logistics

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles logistics HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new logistics handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetQuotes handles POST /logistics/quote.
// Accepts RateQuoteRequest JSON body, returns RateQuoteResponse.
func (h *Handler) GetQuotes(c *gin.Context) {
	var req RateQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if req.WeightKG <= 0 {
		response.Error(c, http.StatusBadRequest, "weight_kg must be positive")
		return
	}
	if req.DestinationCountry == "" {
		response.Error(c, http.StatusBadRequest, "destination_country is required")
		return
	}

	quotes, err := h.service.GetQuotes(&req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, quotes)
}

// ListCarriers handles GET /logistics/carriers.
// Returns a list of configured carriers with their channel counts.
func (h *Handler) ListCarriers(c *gin.Context) {
	carriers := h.service.ListCarriers()
	response.Success(c, carriers)
}
