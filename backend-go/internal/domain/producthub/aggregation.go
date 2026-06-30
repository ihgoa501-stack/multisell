package producthub

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ProductHubProfile is the full aggregated product profile.
type ProductHubProfile struct {
	Master      *ProductMaster   `json:"master"`
	Variants    []ProductVariant `json:"variants"`
	Concept     *ProductConcept  `json:"concept,omitempty"`
	LatestCost  *CostVersion     `json:"latest_cost,omitempty"`
	CostHistory []CostVersion    `json:"cost_history,omitempty"`
	Suppliers   []SupplierInfo   `json:"suppliers,omitempty"`
	Samples     []SampleRequest  `json:"samples,omitempty"`
	Timeline    []TimelineEvent  `json:"timeline,omitempty"`
}

// SupplierInfo is a simplified supplier view for the hub profile.
type SupplierInfo struct {
	SupplierOffer SupplierOffer `json:"supplier_offer"`
	SupplierName  string        `json:"supplier_name,omitempty"`
}

// TimelineEvent represents a lifecycle event on the product timeline.
type TimelineEvent struct {
	Type      string    `json:"type"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"created_at"`
}

// AggregationService builds full product profiles.
type AggregationService struct {
	db      *gorm.DB
	logger  *zap.Logger
	master  *MasterService
	variant *VariantService
	concept *ConceptService
	offer   *SupplierOfferService
	sample  *SampleService
	cost    *CostVersionService
}

// NewAggregationService creates an AggregationService.
func NewAggregationService(db *gorm.DB, logger *zap.Logger) *AggregationService {
	return &AggregationService{
		db:      db,
		logger:  logger,
		master:  NewMasterService(db, logger),
		variant: NewVariantService(db, logger),
		concept: NewConceptService(db, logger),
		offer:   NewSupplierOfferService(db, logger),
		sample:  NewSampleService(db, logger),
		cost:    NewCostVersionService(db, logger),
	}
}

// GetProductHubProfile builds the full aggregated profile for a product.
func (s *AggregationService) GetProductHubProfile(ctx context.Context, productID int64) (*ProductHubProfile, error) {
	profile := &ProductHubProfile{}

	// 1. Product master
	master, err := s.master.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	profile.Master = master

	// 2. Variants
	if variants, err := s.variant.ListByMaster(ctx, productID); err == nil {
		profile.Variants = variants
	}

	// 3. Concept
	if concept, err := s.concept.GetByMasterID(ctx, productID); err == nil {
		profile.Concept = concept
	}

	// 4. Cost
	if cost, err := s.cost.GetLatestByMaster(ctx, productID); err == nil {
		profile.LatestCost = cost
	}
	if costs, err := s.cost.ListByMaster(ctx, productID); err == nil {
		profile.CostHistory = costs
	}

	// 5. Supplier offers with names
	if offers, err := s.offer.ListByMaster(ctx, productID); err == nil {
		info := make([]SupplierInfo, 0, len(offers))
		for _, o := range offers {
			si := SupplierInfo{SupplierOffer: o}
			// Try to fetch supplier name from the supplier module.
			type supplierBrief struct {
				Name string `gorm:"column:name"`
			}
			var sb supplierBrief
			if err := s.db.WithContext(ctx).Table("supplier").Select("name").Where("id = ?", o.SupplierID).Scan(&sb).Error; err == nil {
				si.SupplierName = sb.Name
			}
			info = append(info, si)
		}
		profile.Suppliers = info
	}

	// 6. Sample requests
	if samples, err := s.sample.ListByMaster(ctx, productID); err == nil {
		profile.Samples = samples
	}

	// 7. Timeline — build from lifecycle status and key events.
	events := []TimelineEvent{
		{Type: "created", Summary: "Product created", CreatedAt: master.CreatedAt},
	}
	if profile.Samples != nil {
		for _, s := range profile.Samples {
			events = append(events, TimelineEvent{
				Type: "sample", Summary: "Sample requested", CreatedAt: s.CreatedAt,
			})
		}
	}
	if profile.LatestCost != nil {
		events = append(events, TimelineEvent{
			Type: "cost", Summary: "Cost version " + profile.LatestCost.Version, CreatedAt: profile.LatestCost.CreatedAt,
		})
	}
	profile.Timeline = events

	return profile, nil
}

// HubHandler handles the product hub profile endpoint.
type HubHandler struct {
	service *AggregationService
}

// NewHubHandler creates a HubHandler.
func NewHubHandler(svc *AggregationService) *HubHandler {
	return &HubHandler{service: svc}
}

// GetHub GET /api/v1/product-hub/:id/hub
func (h *HubHandler) GetHub(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	profile, err := h.service.GetProductHubProfile(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "product not found")
			return
		}
		response.InternalError(c, err)
		return
	}
	response.Success(c, profile)
}
