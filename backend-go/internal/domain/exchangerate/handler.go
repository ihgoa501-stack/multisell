package exchangerate

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

// Handler handles exchange-rate HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new exchange-rate handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List GET /exchange-rates
func (h *Handler) List(c *gin.Context) {
	p := common.ParsePagination(c)
	f := &ListFilter{
		FromCurrency:  c.Query("from_currency"),
		ToCurrency:    c.Query("to_currency"),
		EffectiveDate: c.Query("effective_date"),
	}
	items, total, err := h.service.List(&p, f)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// Create POST /exchange-rates
func (h *Handler) Create(c *gin.Context) {
	var in CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	er, err := h.service.Create(&in)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, er)
}

// UpdateByPair PUT /exchange-rates/:from_currency/:to_currency
func (h *Handler) UpdateByPair(c *gin.Context) {
	from := c.Param("from_currency")
	to := c.Param("to_currency")
	if len(from) != 3 || len(to) != 3 {
		response.Error(c, http.StatusBadRequest, "from_currency and to_currency must be 3-letter codes")
		return
	}
	var in UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	er, err := h.service.UpdateByPair(from, to, &in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, er)
}

// Delete DELETE /exchange-rates/:id
func (h *Handler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.service.Delete(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// GetLatest GET /exchange-rates/:from_currency/:to_currency/latest
func (h *Handler) GetLatest(c *gin.Context) {
	from := c.Param("from_currency")
	to := c.Param("to_currency")
	er, err := h.service.GetLatest(from, to)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "no rate found for "+from+"/"+to)
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, er)
}
