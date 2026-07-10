package profit

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

type EvidenceHandler struct {
	db *gorm.DB
}

func NewEvidenceHandler(db *gorm.DB) *EvidenceHandler {
	return &EvidenceHandler{db: db}
}

// GetEvidenceCard GET /api/v1/profit/evidence-card/:productId
func (h *EvidenceHandler) GetEvidenceCard(c *gin.Context) {
	idStr := c.Param("productId")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid productId")
		return
	}

	// Read candidate product (raw scan to avoid importing candidate package)
	var prod candidateProductReader
	row := h.db.Table("candidate_product").
		Select("title, target_sale_price, target_currency, purchase_price, "+
			"purchase_currency, package_weight_kg, package_length_cm, "+
			"package_width_cm, package_height_cm, origin_country, "+
			"destination_country, hs_code, source_url").
		Where("id = ?", id).Row()
	if err := row.Scan(
		&prod.Title, &prod.TargetSalePrice, &prod.TargetCurrency,
		&prod.PurchasePrice, &prod.PurchaseCurrency,
		&prod.PackageWeightKg, &prod.PackageLengthCm,
		&prod.PackageWidthCm, &prod.PackageHeightCm,
		&prod.OriginCountry, &prod.DestinationCountry,
		&prod.HSCode, &prod.SourceURL,
	); err != nil {
		response.Error(c, http.StatusNotFound, "商品不存在")
		return
	}

	// Read platform commission rate (default 15% for Ozon)
	// ponytail: hardcoded default, but try to read from platform table
	commissionRate := 0.15
	var platformRows []struct {
		CommissionRate float64
	}
	if err := h.db.Table("platforms").
		Select("commission_rate").
		Where("id = (SELECT target_platform_id FROM candidate_product WHERE id = ?)", id).
		Scan(&platformRows).Error; err == nil && len(platformRows) > 0 {
		commissionRate = platformRows[0].CommissionRate / 100.0
	}

	// Estimate international shipping from weight
	// ponytail: simple weight-based estimate
	shippingCost := 0.0
	if prod.PackageWeightKg > 0 {
		// Rough: $8 base + $4/kg for US
		shippingCost = 8.0 + prod.PackageWeightKg*4.0
	}

	card := BuildEvidenceCard(&prod, commissionRate, shippingCost)
	card.ProductID = id

	response.Success(c, card)
}
