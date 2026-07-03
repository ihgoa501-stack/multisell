package shipping

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/logistics"
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
		q = q.Where("LOWER(name) LIKE LOWER(?) OR LOWER(code) LIKE LOWER(?)", like, like)
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
	// After Model(&p).Updates, GORM refreshes p in-place so a second First() is unnecessary.
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
		q = q.Where("LOWER(name) LIKE LOWER(?) OR LOWER(code) LIKE LOWER(?)", like, like)
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
	// After Model(&ch).Updates, GORM refreshes ch in-place so a second First() is unnecessary.
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

// ---- Phase 1: Fulfillment Intelligence OS ----

// QuoteUnified computes shipping costs using the unified logistics.RateEngine.
// All quote calculation goes through a single code path:
// DB models -> logistics.RateTableEntry -> logistics.RateEngine.
func (s *Service) QuoteUnified(req *QuoteRequest) (*QuoteResponse, error) {
	qty := req.Quantity
	if qty <= 0 {
		qty = 1
	}
	var actualWeight, lengthCm, widthCm, heightCm float64
	cargoType := req.CargoType
	if cargoType == "" {
		cargoType = "normal"
	}
	if req.Mode == "sku" && req.SkuID != nil {
		var sku struct {
			ProductID   int64
			SkuWeightKg *float64 `gorm:"column:sku_weight_kg"`
			SkuLengthCm *float64 `gorm:"column:sku_length_cm"`
			SkuWidthCm  *float64 `gorm:"column:sku_width_cm"`
			SkuHeightCm *float64 `gorm:"column:sku_height_cm"`
			Weight      *float64 `gorm:"column:weight"`
		}
		if err := s.db.Table("sku").Select("product_id, sku_weight_kg, sku_length_cm, sku_width_cm, sku_height_cm, weight").
			Where("id = ?", *req.SkuID).Scan(&sku).Error; err != nil {
			return nil, err
		}
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
		if lengthCm == 0 || widthCm == 0 || heightCm == 0 {
			var prod struct {
				PackageLengthCm *float64 `gorm:"column:package_length_cm"`
				PackageWidthCm  *float64 `gorm:"column:package_width_cm"`
				PackageHeightCm *float64 `gorm:"column:package_height_cm"`
				PackageWeightKg *float64 `gorm:"column:package_weight_kg"`
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

	var tables []logistics.RateTableEntry
	var channels []ShippingChannel
	if err := s.db.Where("status = 1").Order("sort_order ASC, id ASC").Find(&channels).Error; err != nil {
		return nil, err
	}
	for _, ch := range channels {
		var zone ShippingZone
		if err := s.db.Where("channel_id = ? AND country_code = ? AND status = 1", ch.ID, req.DestinationCountry).First(&zone).Error; err != nil {
			continue
		}
		var rules []ShippingQuoteRule
		q := s.db.Where("channel_id = ? AND status = 1", ch.ID)
		q = q.Where("(zone_id = ? OR zone_id IS NULL)", zone.ID)
		if err := q.Order("priority ASC, id ASC").Find(&rules).Error; err != nil || len(rules) == 0 {
			continue
		}
		providerName := ""
		var prov ShippingProvider
		if err := s.db.Select("name").Where("id = ?", ch.ProviderID).First(&prov).Error; err == nil {
			providerName = prov.Name
		}
		for _, rule := range rules {
			entry := ToRateTableEntry(&ch, &rule, &zone)
			entry.ProviderName = providerName
			tables = append(tables, entry)
		}
	}
	if len(tables) == 0 {
		return &QuoteResponse{Results: []QuoteResult{}}, nil
	}

	engine := logistics.NewRateEngine(tables)
	cargo := logistics.Cargo{
		ActualWeightKg: totalActualWeight,
		LengthCm:       lengthCm,
		WidthCm:        widthCm,
		HeightCm:       heightCm,
	}
	resp, err := engine.CalculateRate(cargo, req.DestinationCountry, cargoType)
	if err != nil {
		return nil, err
	}
	results := make([]QuoteResult, 0, len(resp.Results))
	for _, r := range resp.Results {
		qr := ToQuoteResult(r)
		for _, ch := range channels {
			if ch.Name == r.ChannelName {
				qr.ChannelID = ch.ID
				break
			}
		}
		results = append(results, qr)
	}
	return &QuoteResponse{Results: results}, nil
}

// CreateSnapshot creates an immutable shipping snapshot for an order.
func (s *Service) CreateSnapshot(in *CreateSnapshotInput) (*SalesOrderShippingSnapshot, error) {
	snap := SalesOrderShippingSnapshot{
		OrderID: in.OrderID, SkuID: in.SkuID, Quantity: in.Quantity,
		DestinationCountry: in.DestinationCountry, PostalCode: in.PostalCode, CargoType: in.CargoType,
		PackageSource: in.PackageSource,
		PackageLengthCm: in.PackageLengthCm, PackageWidthCm: in.PackageWidthCm, PackageHeightCm: in.PackageHeightCm,
		PackageWeightKg: in.PackageWeightKg,
		ProviderID: in.ProviderID, ProviderName: in.ProviderName, ChannelID: in.ChannelID, ChannelName: in.ChannelName,
		Currency: in.Currency,
		ActualWeightKg: in.ActualWeightKg, VolumetricWeightKg: in.VolumetricWeightKg,
		ChargeableWeightKg: in.ChargeableWeightKg,
		BaseShippingFee: in.BaseShippingFee, SurchargeFee: in.SurchargeFee, FuelSurchargeFee: in.FuelSurchargeFee,
		TotalShippingFee: in.TotalShippingFee, CalculationDetail: in.CalculationDetail,
		RuleVersionID: in.RuleVersionID, RuleVersion: in.RuleVersion,
		QuotedBy: in.QuotedBy, SourceTrigger: in.SourceTrigger,
	}
	if snap.Quantity <= 0 {
		snap.Quantity = 1
	}
	if snap.Currency == "" {
		snap.Currency = "CNY"
	}
	if snap.CargoType == "" {
		snap.CargoType = "normal"
	}
	if snap.SourceTrigger == "" {
		snap.SourceTrigger = "manual"
	}
	if snap.QuotedBy == "" {
		snap.QuotedBy = "system"
	}
	if err := s.db.Create(&snap).Error; err != nil {
		return nil, err
	}
	return &snap, nil
}

// GetSnapshotByOrderID retrieves the snapshot for an order.
func (s *Service) GetSnapshotByOrderID(orderID int64) (*SalesOrderShippingSnapshot, error) {
	var snap SalesOrderShippingSnapshot
	if err := s.db.Where("order_id = ?", orderID).First(&snap).Error; err != nil {
		return nil, err
	}
	return &snap, nil
}

// ReconcileBillBatch matches bill items to order shipping snapshots and computes variances.
func (s *Service) ReconcileBillBatch(batchID int64) (*BillReconciliationResult, error) {
	var items []ShippingBillItem
	if err := s.db.Where("batch_id = ?", batchID).Find(&items).Error; err != nil {
		return nil, err
	}
	result := &BillReconciliationResult{TotalItems: len(items), Currency: "CNY"}
	for i, item := range items {
		matched := false
		if item.OrderNo != "" {
			var orderID int64
			if _, err := fmt.Sscanf(item.OrderNo, "%d", &orderID); err == nil && orderID > 0 {
				var snap SalesOrderShippingSnapshot
				if err := s.db.Where("order_id = ?", orderID).First(&snap).Error; err == nil {
					matchedID := snap.ID
					fee := snap.TotalShippingFee
					variance := 0.0
					if item.TotalActualFee != nil {
						variance = *item.TotalActualFee - snap.TotalShippingFee
					}
					variancePct := 0.0
					if snap.TotalShippingFee > 0 {
						variancePct = (variance / snap.TotalShippingFee) * 100
					}
					anomalyType := ""
					if variancePct > 5 {
						anomalyType = "overcharge"
					} else if variancePct < -5 {
						anomalyType = "undercharge"
					}
					items[i].MatchedOrderID = &snap.OrderID
					items[i].MatchedSnapshotID = &matchedID
					items[i].SnapshotShippingFee = &fee
					items[i].VarianceAmount = &variance
					items[i].VariancePct = &variancePct
					items[i].AnomalyType = anomalyType
					items[i].ReconciliationStatus = "matched"
					if variancePct > 5 || variancePct < -5 {
						result.AnomalousItems++
					}
					result.MatchedItems++
					result.TotalVariance += variance
					s.db.Save(&items[i])
					matched = true
				}
			}
		}
		if !matched {
			items[i].ReconciliationStatus = "unmatched_order"
			result.UnmatchedItems++
			s.db.Save(&items[i])
		}
	}
	var batch ShippingBillBatch
	if err := s.db.First(&batch, batchID).Error; err == nil {
		batch.RowCount = result.TotalItems
		batch.MatchedCount = result.MatchedItems
		batch.MismatchCount = result.AnomalousItems
		batch.UnmatchedCount = result.UnmatchedItems
		if result.UnmatchedItems == 0 && result.AnomalousItems == 0 {
			batch.Status = "reconciled"
		} else if result.AnomalousItems > 0 {
			batch.Status = "has_anomalies"
		} else {
			batch.Status = "partial"
		}
		s.db.Save(&batch)
	}
	return result, nil
}

// GetActiveRulesAtTime returns rules effective at a given time.
func (s *Service) GetActiveRulesAtTime(channelID *int64, at time.Time) ([]ShippingQuoteRule, error) {
	var rules []ShippingQuoteRule
	q := s.db.Where("status = 1").
		Where("(effective_start_time IS NULL OR effective_start_time <= ?)", at).
		Where("(effective_end_time IS NULL OR effective_end_time >= ?)", at)
	if channelID != nil {
		q = q.Where("channel_id = ?", *channelID)
	}
	if err := q.Order("priority ASC, rule_version DESC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// ListRuleVersions lists all versions for a given channel's rules.
func (s *Service) ListRuleVersions(channelID int64) ([]ShippingQuoteRule, int64, error) {
	var rules []ShippingQuoteRule
	var total int64
	q := s.db.Model(&ShippingQuoteRule{}).Where("channel_id = ?", channelID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("rule_version DESC, created_at DESC").Find(&rules).Error; err != nil {
		return nil, 0, err
	}
	return rules, total, nil
}

// ReviewBillItem updates the review status and notes on a bill item.
func (s *Service) ReviewBillItem(id int64, reviewStatus string, note string, resolvedBy string) error {
	updates := map[string]interface{}{
		"review_status": reviewStatus,
		"note":          note,
		"resolved_by":   resolvedBy,
	}
	if reviewStatus == "resolved" || reviewStatus == "confirmed" {
		now := time.Now()
		updates["resolved_at"] = &now
	}
	return s.db.Model(&ShippingBillItem{}).Where("id = ?", id).Updates(updates).Error
}

// ListBillAnomalies returns bill items with anomalies for a batch.
func (s *Service) ListBillAnomalies(batchID int64) ([]ShippingBillItem, error) {
	var items []ShippingBillItem
	if err := s.db.Where("batch_id = ? AND anomaly_type != ''", batchID).
		Order("variance_pct DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
