package shipping

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides shipping business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new shipping service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// ---------- ShippingProvider ----------

// ListProviders returns all active providers (optionally filtered by status).
func (s *Service) ListProviders(c *common.Pagination, search string) ([]ShippingProvider, int64, error) {
	var items []ShippingProvider
	var total int64
	q := s.db.Model(&ShippingProvider{})
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("name ILIKE ? OR code ILIKE ?", like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id ASC").Offset(c.Offset()).Limit(c.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) GetProvider(id int64) (*ShippingProvider, error) {
	var p ShippingProvider
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) CreateProvider(in *CreateProviderInput) (*ShippingProvider, error) {
	p := ShippingProvider{Name: in.Name, Code: in.Code, Contact: in.Contact, Phone: in.Phone, Remark: in.Remark}
	if in.Status != nil {
		p.Status = *in.Status
	} else {
		p.Status = 1
	}
	if err := s.db.Create(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) UpdateProvider(id int64, in *UpdateProviderInput) (*ShippingProvider, error) {
	var p ShippingProvider
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Name != nil {
		updates["name"] = *in.Name
	}
	if in.Code != nil {
		updates["code"] = *in.Code
	}
	if in.Contact != nil {
		updates["contact"] = *in.Contact
	}
	if in.Phone != nil {
		updates["phone"] = *in.Phone
	}
	if in.Remark != nil {
		updates["remark"] = *in.Remark
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if len(updates) == 0 {
		return &p, nil
	}
	if err := s.db.Model(&p).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) DeleteProvider(id int64) error {
	res := s.db.Delete(&ShippingProvider{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ---------- ShippingChannel ----------

func (s *Service) ListChannels(c *common.Pagination, providerID *int64, search string) ([]ShippingChannel, int64, error) {
	var items []ShippingChannel
	var total int64
	q := s.db.Model(&ShippingChannel{})
	if providerID != nil {
		q = q.Where("provider_id = ?", *providerID)
	}
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("name ILIKE ? OR code ILIKE ?", like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("sort_order ASC, id ASC").Offset(c.Offset()).Limit(c.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) GetChannel(id int64) (*ShippingChannel, error) {
	var ch ShippingChannel
	if err := s.db.First(&ch, id).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

func (s *Service) CreateChannel(in *CreateChannelInput) (*ShippingChannel, error) {
	ch := ShippingChannel{
		ProviderID:           in.ProviderID,
		Name:                 in.Name,
		Code:                 in.Code,
		CargoTypes:           in.CargoTypes,
		EstimatedDeliveryMin: in.EstimatedDeliveryMin,
		EstimatedDeliveryMax: in.EstimatedDeliveryMax,
		Currency:             in.Currency,
	}
	if in.VolumetricDivisor != nil {
		ch.VolumetricDivisor = *in.VolumetricDivisor
	} else {
		ch.VolumetricDivisor = 6000
	}
	if in.SortOrder != nil {
		ch.SortOrder = *in.SortOrder
	}
	if in.Status != nil {
		ch.Status = *in.Status
	} else {
		ch.Status = 1
	}
	if ch.Currency == "" {
		ch.Currency = "CNY"
	}
	if err := s.db.Create(&ch).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

func (s *Service) UpdateChannel(id int64, in *UpdateChannelInput) (*ShippingChannel, error) {
	var ch ShippingChannel
	if err := s.db.First(&ch, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Name != nil {
		updates["name"] = *in.Name
	}
	if in.Code != nil {
		updates["code"] = *in.Code
	}
	if in.VolumetricDivisor != nil {
		updates["volumetric_divisor"] = *in.VolumetricDivisor
	}
	if in.CargoTypes != nil {
		updates["cargo_types"] = *in.CargoTypes
	}
	if in.EstimatedDeliveryMin != nil {
		updates["estimated_delivery_min"] = *in.EstimatedDeliveryMin
	}
	if in.EstimatedDeliveryMax != nil {
		updates["estimated_delivery_max"] = *in.EstimatedDeliveryMax
	}
	if in.Currency != nil {
		updates["currency"] = *in.Currency
	}
	if in.SortOrder != nil {
		updates["sort_order"] = *in.SortOrder
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if len(updates) == 0 {
		return &ch, nil
	}
	if err := s.db.Model(&ch).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&ch, id).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

func (s *Service) DeleteChannel(id int64) error {
	res := s.db.Delete(&ShippingChannel{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ---------- ShippingZone ----------

func (s *Service) ListZones(c *common.Pagination, channelID *int64) ([]ShippingZone, int64, error) {
	var items []ShippingZone
	var total int64
	q := s.db.Model(&ShippingZone{})
	if channelID != nil {
		q = q.Where("channel_id = ?", *channelID)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id ASC").Offset(c.Offset()).Limit(c.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) CreateZone(in *CreateZoneInput) (*ShippingZone, error) {
	z := ShippingZone{
		ChannelID:      in.ChannelID,
		CountryCode:    in.CountryCode,
		PostalCodeFrom: in.PostalCodeFrom,
		PostalCodeTo:   in.PostalCodeTo,
	}
	if in.Status != nil {
		z.Status = *in.Status
	} else {
		z.Status = 1
	}
	if err := s.db.Create(&z).Error; err != nil {
		return nil, err
	}
	return &z, nil
}

func (s *Service) DeleteZone(id int64) error {
	res := s.db.Delete(&ShippingZone{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ---------- ShippingQuoteRule ----------

func (s *Service) ListRules(c *common.Pagination, channelID *int64) ([]ShippingQuoteRule, int64, error) {
	var items []ShippingQuoteRule
	var total int64
	q := s.db.Model(&ShippingQuoteRule{})
	if channelID != nil {
		q = q.Where("channel_id = ?", *channelID)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("priority ASC, id ASC").Offset(c.Offset()).Limit(c.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) CreateRule(in *CreateQuoteRuleInput) (*ShippingQuoteRule, error) {
	r := ShippingQuoteRule{
		ChannelID:          in.ChannelID,
		ZoneID:             in.ZoneID,
		RuleType:           in.RuleType,
		Priority:           0,
		MinWeightKg:        in.MinWeightKg,
		MaxWeightKg:        in.MaxWeightKg,
		FirstKg:            in.FirstKg,
		FirstPrice:         in.FirstPrice,
		AdditionalKg:       in.AdditionalKg,
		AdditionalPrice:    in.AdditionalPrice,
		FixedFee:           in.FixedFee,
		PerKgPrice:        in.PerKgPrice,
		MinimumCharge:      in.MinimumCharge,
		TierConfig:         in.TierConfig,
		SurchargeFixed:    in.SurchargeFixed,
		FuelSurchargePct:  in.FuelSurchargePct,
		RoundingIncrement: in.RoundingIncrement,
		Remark:             in.Remark,
	}
	if in.Priority != nil {
		r.Priority = *in.Priority
	}
	if in.Status != nil {
		r.Status = *in.Status
	} else {
		r.Status = 1
	}
	if err := s.db.Create(&r).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Service) DeleteRule(id int64) error {
	res := s.db.Delete(&ShippingQuoteRule{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ---------- ShippingBillBatch / Item ----------

func (s *Service) ListBillBatches(c *common.Pagination, providerID *int64) ([]ShippingBillBatch, int64, error) {
	var items []ShippingBillBatch
	var total int64
	q := s.db.Model(&ShippingBillBatch{})
	if providerID != nil {
		q = q.Where("provider_id = ?", *providerID)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset(c.Offset()).Limit(c.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) GetBillBatch(id int64) (*ShippingBillBatch, []ShippingBillItem, error) {
	var b ShippingBillBatch
	if err := s.db.First(&b, id).Error; err != nil {
		return nil, nil, err
	}
	var items []ShippingBillItem
	s.db.Where("batch_id = ?", id).Order("row_number ASC").Find(&items)
	return &b, items, nil
}

func (s *Service) CreateBillBatch(in *CreateBillBatchInput) (*ShippingBillBatch, error) {
	b := ShippingBillBatch{
		ProviderID:     in.ProviderID,
		SourceFilename: in.SourceFilename,
		Currency:       in.Currency,
		CreatedBy:      in.CreatedBy,
		Status:         "imported",
	}
	if b.Currency == "" {
		b.Currency = "CNY"
	}
	if err := s.db.Create(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// DeleteBillBatch removes a bill batch by id.
func (s *Service) DeleteBillBatch(id int64) error {
	res := s.db.Delete(&ShippingBillBatch{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Service) ListBillItems(c *common.Pagination, batchID int64) ([]ShippingBillItem, int64, error) {
	var items []ShippingBillItem
	var total int64
	q := s.db.Model(&ShippingBillItem{}).Where("batch_id = ?", batchID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("row_number ASC").Offset(c.Offset()).Limit(c.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ---------- Quote calculation ----------

// Quote computes shipping cost across all eligible channels for the given request.
func (s *Service) Quote(req *QuoteRequest) (*QuoteResponse, error) {
	qty := req.Quantity
	if qty <= 0 {
		qty = 1
	}

	// Determine package dimensions & actual weight
	var actualWeight, lengthCm, widthCm, heightCm float64
	cargoType := req.CargoType
	if cargoType == "" {
		cargoType = "normal"
	}

	if req.Mode == "sku" && req.SkuID != nil {
		// Load SKU package info
		var sku struct {
			ProductID      int64   `gorm:"column:product_id"`
			SkuWeightKg    *float64 `gorm:"column:sku_weight_kg"`
			SkuLengthCm    *float64 `gorm:"column:sku_length_cm"`
			SkuWidthCm     *float64 `gorm:"column:sku_width_cm"`
			SkuHeightCm    *float64 `gorm:"column:sku_height_cm"`
			Weight         *float64 `gorm:"column:weight"`
		}
		if err := s.db.Table("sku").Select("product_id, sku_weight_kg, sku_length_cm, sku_width_cm, sku_height_cm, weight").
			Where("id = ?", *req.SkuID).Scan(&sku).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, gorm.ErrRecordNotFound
			}
			return nil, err
		}
		if sku.ProductID == 0 {
			return nil, gorm.ErrRecordNotFound
		}
		// Prefer SKU-level packaging; fall back to product-level
		if sku.SkuWeightKg != nil && *sku.SkuWeightKg > 0 {
			actualWeight = *sku.SkuWeightKg
		} else if sku.Weight != nil {
			actualWeight = *sku.Weight
		}
		if sku.SkuLengthCm != nil {
			lengthCm = *sku.SkuLengthCm
		}
		if sku.SkuWidthCm != nil {
			widthCm = *sku.SkuWidthCm
		}
		if sku.SkuHeightCm != nil {
			heightCm = *sku.SkuHeightCm
		}
		// Fall back to product packaging if SKU packaging incomplete
		if lengthCm == 0 || widthCm == 0 || heightCm == 0 {
			var prod struct {
				PackageLengthCm  *float64 `gorm:"column:package_length_cm"`
				PackageWidthCm   *float64 `gorm:"column:package_width_cm"`
				PackageHeightCm  *float64 `gorm:"column:package_height_cm"`
				PackageWeightKg  *float64 `gorm:"column:package_weight_kg"`
			}
			s.db.Table("product").Select("package_length_cm, package_width_cm, package_height_cm, package_weight_kg").
				Where("id = ?", sku.ProductID).Scan(&prod)
			if lengthCm == 0 && prod.PackageLengthCm != nil {
				lengthCm = *prod.PackageLengthCm
			}
			if widthCm == 0 && prod.PackageWidthCm != nil {
				widthCm = *prod.PackageWidthCm
			}
			if heightCm == 0 && prod.PackageHeightCm != nil {
				heightCm = *prod.PackageHeightCm
			}
			if actualWeight == 0 && prod.PackageWeightKg != nil {
				actualWeight = *prod.PackageWeightKg
			}
		}
	} else {
		// Manual mode
		if req.ManualWeightKg != nil {
			actualWeight = *req.ManualWeightKg
		}
		if req.ManualLengthCM != nil {
			lengthCm = *req.ManualLengthCM
		}
		if req.ManualWidthCM != nil {
			widthCm = *req.ManualWidthCM
		}
		if req.ManualHeightCM != nil {
			heightCm = *req.ManualHeightCM
		}
	}

	totalActualWeight := actualWeight * float64(qty)
	baseVolume := lengthCm * widthCm * heightCm * float64(qty)

	// Find eligible channels: active channels with zones covering destination country
	var channels []ShippingChannel
	if err := s.db.Where("status = 1").Order("sort_order ASC, id ASC").Find(&channels).Error; err != nil {
		return nil, err
	}

	var results []QuoteResult
	for _, ch := range channels {
		// Find zone for this channel + country
		var zone ShippingZone
		zoneErr := s.db.Where("channel_id = ? AND country_code = ? AND status = 1", ch.ID, req.DestinationCountry).First(&zone).Error
		if zoneErr != nil {
			if errors.Is(zoneErr, gorm.ErrRecordNotFound) {
				continue // channel doesn't serve this country
			}
			continue
		}

		// Find the highest-priority active rule (zone-specific first, then global)
		var rule ShippingQuoteRule
		ruleErr := s.db.Where("channel_id = ? AND zone_id = ? AND status = 1", ch.ID, zone.ID).
			Order("priority ASC, id ASC").First(&rule).Error
		if ruleErr != nil {
			ruleErr = s.db.Where("channel_id = ? AND zone_id IS NULL AND status = 1", ch.ID).
				Order("priority ASC, id ASC").First(&rule).Error
		}
		if ruleErr != nil {
			continue
		}

		// Volumetric weight
		volumetricWeight := 0.0
		if ch.VolumetricDivisor > 0 && baseVolume > 0 {
			volumetricWeight = baseVolume / float64(ch.VolumetricDivisor)
		}

		// Chargeable weight
		chargeable := math.Max(totalActualWeight, volumetricWeight)

		// Rounding increment
		roundInc := 0.1
		if rule.RoundingIncrement != nil && *rule.RoundingIncrement > 0 {
			roundInc = *rule.RoundingIncrement
		}
		rounded := math.Ceil(chargeable/roundInc) * roundInc

		// Base fee
		baseFee := applyRule(&rule, rounded)

		// Minimum charge
		if rule.MinimumCharge != nil && *rule.MinimumCharge > 0 && baseFee < *rule.MinimumCharge {
			baseFee = *rule.MinimumCharge
		}

		// Surcharges
		surcharge := 0.0
		if rule.SurchargeFixed != nil {
			surcharge = *rule.SurchargeFixed
		}
		fuelPct := 0.0
		if rule.FuelSurchargePct != nil {
			fuelPct = *rule.FuelSurchargePct
		}
		fuelFee := (baseFee + surcharge) * (fuelPct / 100.0)

		total := baseFee + surcharge + fuelFee

		// Provider name
		providerName := ""
		var prov ShippingProvider
		if err := s.db.Select("name").Where("id = ?", ch.ProviderID).First(&prov).Error; err == nil {
			providerName = prov.Name
		}

		results = append(results, QuoteResult{
			ChannelID:          ch.ID,
			ChannelName:         ch.Name,
			ProviderName:        providerName,
			ActualWeightKg:     roundTo(totalActualWeight, 4),
			VolumetricWeightKg:  roundTo(volumetricWeight, 4),
			ChargeableWeightKg: roundTo(rounded, 4),
			BaseShippingFee:    roundTo(baseFee, 2),
			SurchargeFee:       roundTo(surcharge, 2),
			FuelSurchargeFee:   roundTo(fuelFee, 2),
			TotalShippingFee:   roundTo(total, 2),
			Currency:           ch.Currency,
			CalculationDetail:  buildDetail(&rule, rounded, baseFee, surcharge, fuelFee),
		})
	}

	// Sort by total shipping fee ascending (cheapest first)
	sortResults(results)

	return &QuoteResponse{Results: results}, nil
}

// applyRule computes the base shipping fee for a rule given chargeable weight.
func applyRule(rule *ShippingQuoteRule, chargeableWeight float64) float64 {
	switch rule.RuleType {
	case "fixed_plus_per_kg":
		fixed := 0.0
		if rule.FixedFee != nil {
			fixed = *rule.FixedFee
		}
		perKg := 0.0
		if rule.PerKgPrice != nil {
			perKg = *rule.PerKgPrice
		}
		return fixed + (chargeableWeight * perKg)

	case "first_weight_plus_increment":
		firstKg := 0.0
		if rule.FirstKg != nil {
			firstKg = *rule.FirstKg
		}
		firstPrice := 0.0
		if rule.FirstPrice != nil {
			firstPrice = *rule.FirstPrice
		}
		addKg := 0.1
		if rule.AdditionalKg != nil && *rule.AdditionalKg > 0 {
			addKg = *rule.AdditionalKg
		}
		addPrice := 0.0
		if rule.AdditionalPrice != nil {
			addPrice = *rule.AdditionalPrice
		}
		if chargeableWeight <= firstKg {
			return firstPrice
		}
		rawUnits := (chargeableWeight - firstKg) / addKg
		additionalUnits := math.Ceil(rawUnits - 1e-10)
		return firstPrice + (additionalUnits * addPrice)

	case "tiered_weight":
		if len(rule.TierConfig) == 0 {
			return 0
		}
		var tiers []struct {
			MinKg  *float64 `json:"min_kg"`
			MaxKg  *float64 `json:"max_kg"`
			Price  *float64 `json:"price"`
		}
		if err := json.Unmarshal(rule.TierConfig, &tiers); err != nil {
			return 0
		}
		for _, tier := range tiers {
			minKg := 0.0
			if tier.MinKg != nil {
				minKg = *tier.MinKg
			}
			price := 0.0
			if tier.Price != nil {
				price = *tier.Price
			}
			if tier.MaxKg == nil {
				if chargeableWeight >= minKg-1e-10 {
					return price
				}
			} else {
				maxKg := *tier.MaxKg
				if (minKg-1e-10) <= chargeableWeight && chargeableWeight < (maxKg+1e-10) {
					return price
				}
			}
		}
		if len(tiers) > 0 {
			if tiers[len(tiers)-1].Price != nil {
				return *tiers[len(tiers)-1].Price
			}
		}
		return 0
	}
	return 0
}

func buildDetail(rule *ShippingQuoteRule, chargeable, baseFee, surcharge, fuelFee float64) string {
	parts := []string{}
	switch rule.RuleType {
	case "fixed_plus_per_kg":
		fixed := 0.0
		if rule.FixedFee != nil {
			fixed = *rule.FixedFee
		}
		perKg := 0.0
		if rule.PerKgPrice != nil {
			perKg = *rule.PerKgPrice
		}
		parts = append(parts, fmt.Sprintf("fixed %.1f + %.2fkg x %.1f = %.1f", fixed, chargeable, perKg, baseFee))
	case "first_weight_plus_increment":
		firstKg := 0.0
		if rule.FirstKg != nil {
			firstKg = *rule.FirstKg
		}
		firstPrice := 0.0
		if rule.FirstPrice != nil {
			firstPrice = *rule.FirstPrice
		}
		addKg := 0.1
		if rule.AdditionalKg != nil && *rule.AdditionalKg > 0 {
			addKg = *rule.AdditionalKg
		}
		addPrice := 0.0
		if rule.AdditionalPrice != nil {
			addPrice = *rule.AdditionalPrice
		}
		if chargeable <= firstKg {
			parts = append(parts, fmt.Sprintf("first %.1fkg = %.1f", firstKg, firstPrice))
		} else {
			rawUnits := (chargeable - firstKg) / addKg
			addUnits := math.Ceil(rawUnits - 1e-10)
			parts = append(parts, fmt.Sprintf("first %.1fkg=%.1f + %d units x %.1f = %.1f", firstKg, firstPrice, int(addUnits), addPrice, baseFee))
		}
	case "tiered_weight":
		parts = append(parts, fmt.Sprintf("tiered: %.2fkg -> %.1f", chargeable, baseFee))
	}
	if surcharge > 0 {
		parts = append(parts, fmt.Sprintf("surcharge %.1f", surcharge))
	}
	if fuelFee > 0 {
		parts = append(parts, fmt.Sprintf("fuel %.1f", fuelFee))
	}
	return strings.Join(parts, " + ")
}

func roundTo(v float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(v*pow) / pow
}

func sortResults(results []QuoteResult) {
	// Simple insertion sort by TotalShippingFee ascending
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].TotalShippingFee < results[j-1].TotalShippingFee; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
}
