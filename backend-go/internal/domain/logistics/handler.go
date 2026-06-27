package logistics

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for logistics.
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// GetQuotesRequest is the request body for POST /logistics/quote.
type GetQuotesRequest struct {
	WeightKg     float64  `json:"weight_kg" binding:"required"`
	LengthCm     *float64 `json:"length_cm,omitempty"`
	WidthCm      *float64 `json:"width_cm,omitempty"`
	HeightCm     *float64 `json:"height_cm,omitempty"`
	Destinations []string `json:"destinations" binding:"required"`
	CargoType    string   `json:"cargo_type,omitempty"`
}

// quoteItem represents a single quote line in the response.
type quoteItem struct {
	Channel          string  `json:"channel"`
	Provider         string  `json:"provider"`
	ChargeableWeight float64 `json:"chargeable_weight_kg"`
	BaseFee          float64 `json:"base_fee"`
	Surcharge        float64 `json:"surcharge_fee"`
	FuelSurcharge    float64 `json:"fuel_surcharge_fee"`
	TotalFee         float64 `json:"total_fee"`
	Currency         string  `json:"currency"`
	ETADaysMin       int     `json:"eta_days_min"`
	ETADaysMax       int     `json:"eta_days_max"`
}

// GetQuotes handles POST /logistics/quote.
func (h *Handler) GetQuotes(c *gin.Context) {
	var req GetQuotesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	cargo := Cargo{
		ActualWeightKg: req.WeightKg,
	}
	if req.LengthCm != nil {
		cargo.LengthCm = *req.LengthCm
	}
	if req.WidthCm != nil {
		cargo.WidthCm = *req.WidthCm
	}
	if req.HeightCm != nil {
		cargo.HeightCm = *req.HeightCm
	}

	cargoType := req.CargoType

	var items []quoteItem
	for _, dest := range req.Destinations {
		resp, err := h.svc.GetQuote(cargo, dest, cargoType)
		if err != nil {
			h.logger.Warn("logistics: quote failed",
				zap.String("destination", dest),
				zap.Error(err))
			continue
		}
		for _, r := range resp.Results {
			items = append(items, quoteItem{
				Channel:          r.ChannelName,
				Provider:         r.ProviderName,
				ChargeableWeight: r.ChargeableWeightKg,
				BaseFee:          r.BaseShippingFee,
				Surcharge:        r.SurchargeFee,
				FuelSurcharge:    r.FuelSurchargeFee,
				TotalFee:         r.TotalShippingFee,
				Currency:         r.Currency,
				ETADaysMin:       r.EstimatedDeliveryMin,
				ETADaysMax:       r.EstimatedDeliveryMax,
			})
		}
	}

	if items == nil {
		items = make([]quoteItem, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "ok",
		"data":    items,
	})
}
