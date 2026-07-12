package sourcing1688

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

var preciseCurrencyPattern = regexp.MustCompile(`^[A-Z]{3,8}$`)

// SourcingCostVersion is an immutable, Owner-scoped cost calculation for one
// exact task, opportunity approval, source observation and canonical SKU.
// Money is persisted as integer minor units; float values are never authority.
type SourcingCostVersion struct {
	ID                    int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OwnerID               int64     `gorm:"column:owner_id;not null" json:"owner_id"`
	SourcingProductID     int64     `gorm:"column:sourcing_product_id;not null" json:"sourcing_product_id"`
	TaskLinkID            int64     `gorm:"column:task_link_id;not null" json:"task_link_id"`
	ProductOpportunityID  int64     `gorm:"column:product_opportunity_id;not null" json:"product_opportunity_id"`
	OpportunityDecisionID int64     `gorm:"column:opportunity_decision_id;not null" json:"opportunity_decision_id"`
	SourceSnapshotID      int64     `gorm:"column:source_snapshot_id;not null" json:"source_snapshot_id"`
	SKUMappingID          int64     `gorm:"column:sku_mapping_id;not null" json:"sku_mapping_id"`
	Version               int64     `gorm:"column:version;not null" json:"version"`
	TargetCurrency        string    `gorm:"column:target_currency;not null" json:"target_currency"`
	TotalMinor            int64     `gorm:"column:total_minor;not null" json:"total_minor"`
	ContentHash           string    `gorm:"column:content_hash;not null" json:"content_hash"`
	CreatedBy             int64     `gorm:"column:created_by;not null" json:"created_by"`
	CreatedAt             time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (SourcingCostVersion) TableName() string { return "sourcing_cost_version" }

type SourcingCostLine struct {
	ID                     int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CostVersionID          int64      `gorm:"column:cost_version_id;not null" json:"cost_version_id"`
	CostType               string     `gorm:"column:cost_type;not null" json:"cost_type"`
	AmountMinor            int64      `gorm:"column:amount_minor;not null" json:"amount_minor"`
	Currency               string     `gorm:"column:currency;not null" json:"currency"`
	NormalizedAmountMinor  int64      `gorm:"column:normalized_amount_minor;not null" json:"normalized_amount_minor"`
	ExchangeRateDecimal    *string    `gorm:"column:exchange_rate_decimal" json:"exchange_rate_decimal,omitempty"`
	ExchangeRateSourceURI  *string    `gorm:"column:exchange_rate_source_uri" json:"exchange_rate_source_uri,omitempty"`
	ExchangeRateObservedAt *time.Time `gorm:"column:exchange_rate_observed_at" json:"exchange_rate_observed_at,omitempty"`
	TruthStatus            string     `gorm:"column:truth_status;not null" json:"truth_status"`
	SourceURI              string     `gorm:"column:source_uri;not null" json:"source_uri"`
	ObservedAt             time.Time  `gorm:"column:observed_at;not null" json:"observed_at"`
	CreatedAt              time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (SourcingCostLine) TableName() string { return "sourcing_cost_line" }

type CreateSourcingCostLineInput struct {
	CostType               string     `json:"cost_type" binding:"required"`
	AmountMinor            int64      `json:"amount_minor"`
	Currency               string     `json:"currency" binding:"required"`
	NormalizedAmountMinor  int64      `json:"normalized_amount_minor"`
	ExchangeRateDecimal    *string    `json:"exchange_rate_decimal"`
	ExchangeRateSourceURI  *string    `json:"exchange_rate_source_uri"`
	ExchangeRateObservedAt *time.Time `json:"exchange_rate_observed_at"`
	TruthStatus            string     `json:"truth_status" binding:"required"`
	SourceURI              string     `json:"source_uri" binding:"required"`
	ObservedAt             time.Time  `json:"observed_at" binding:"required"`
}

type CreateSourcingCostVersionInput struct {
	TaskLinkID       int64                         `json:"task_link_id" binding:"required"`
	SourceSnapshotID int64                         `json:"source_snapshot_id" binding:"required"`
	SKUMappingID     int64                         `json:"sku_mapping_id" binding:"required"`
	TargetCurrency   string                        `json:"target_currency" binding:"required"`
	Lines            []CreateSourcingCostLineInput `json:"lines" binding:"required"`
}

type SourcingCostVersionDetail struct {
	Version SourcingCostVersion `json:"version"`
	Lines   []SourcingCostLine  `json:"lines"`
}

func validatePreciseCostInput(in *CreateSourcingCostVersionInput) ([]CreateSourcingCostLineInput, int64, error) {
	if in == nil || in.TaskLinkID <= 0 || in.SourceSnapshotID <= 0 || in.SKUMappingID <= 0 {
		return nil, 0, ErrInvalidWorkflow
	}
	target := strings.ToUpper(strings.TrimSpace(in.TargetCurrency))
	if !preciseCurrencyPattern.MatchString(target) || len(in.Lines) != len(requiredCostTypes) {
		return nil, 0, fmt.Errorf("%w: target currency and complete 10-line cost set are required", ErrInvalidWorkflow)
	}
	allowed, seen := map[string]bool{}, map[string]bool{}
	for _, typ := range requiredCostTypes {
		allowed[typ] = true
	}
	lines := append([]CreateSourcingCostLineInput(nil), in.Lines...)
	var total int64
	for i := range lines {
		line := &lines[i]
		line.CostType = strings.TrimSpace(line.CostType)
		line.Currency = strings.ToUpper(strings.TrimSpace(line.Currency))
		line.TruthStatus = strings.ToLower(strings.TrimSpace(line.TruthStatus))
		line.SourceURI = strings.TrimSpace(line.SourceURI)
		if !allowed[line.CostType] || seen[line.CostType] || line.AmountMinor < 0 || line.NormalizedAmountMinor < 0 || !preciseCurrencyPattern.MatchString(line.Currency) || (line.TruthStatus != "actual" && line.TruthStatus != "quoted" && line.TruthStatus != "estimated") || line.SourceURI == "" || line.ObservedAt.IsZero() || line.ObservedAt.After(time.Now().Add(5*time.Minute)) {
			return nil, 0, fmt.Errorf("%w: invalid cost evidence line %s", ErrInvalidWorkflow, line.CostType)
		}
		seen[line.CostType] = true
		if line.Currency == target {
			if line.AmountMinor != line.NormalizedAmountMinor || line.ExchangeRateDecimal != nil || line.ExchangeRateSourceURI != nil || line.ExchangeRateObservedAt != nil {
				return nil, 0, fmt.Errorf("%w: same-currency line must preserve exact minor units without exchange rate", ErrInvalidWorkflow)
			}
		} else if err := validateExactConversion(line); err != nil {
			return nil, 0, err
		}
		if line.NormalizedAmountMinor > int64(^uint64(0)>>1)-total {
			return nil, 0, fmt.Errorf("%w: total cost overflow", ErrInvalidWorkflow)
		}
		total += line.NormalizedAmountMinor
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].CostType < lines[j].CostType })
	in.TargetCurrency = target
	return lines, total, nil
}

func validateExactConversion(line *CreateSourcingCostLineInput) error {
	if line.ExchangeRateDecimal == nil || line.ExchangeRateSourceURI == nil || line.ExchangeRateObservedAt == nil {
		return fmt.Errorf("%w: cross-currency line requires exchange-rate evidence", ErrInvalidWorkflow)
	}
	rateText := strings.TrimSpace(*line.ExchangeRateDecimal)
	rate, ok := new(big.Rat).SetString(rateText)
	if !ok || rate.Sign() <= 0 || strings.TrimSpace(*line.ExchangeRateSourceURI) == "" || line.ExchangeRateObservedAt.IsZero() || line.ExchangeRateObservedAt.After(time.Now().Add(5*time.Minute)) {
		return fmt.Errorf("%w: invalid exchange-rate evidence", ErrInvalidWorkflow)
	}
	exact := new(big.Rat).Mul(new(big.Rat).SetInt64(line.AmountMinor), rate)
	rounded, ok := roundPositiveRatHalfUp(exact)
	if !ok || rounded != line.NormalizedAmountMinor {
		return fmt.Errorf("%w: normalized minor amount does not exactly match decimal exchange rate", ErrInvalidWorkflow)
	}
	*line.ExchangeRateDecimal, *line.ExchangeRateSourceURI = rateText, strings.TrimSpace(*line.ExchangeRateSourceURI)
	return nil
}

// roundPositiveRatHalfUp defines the sole normalization rule at the smallest
// target-currency unit, avoiding database/float-specific rounding behavior.
func roundPositiveRatHalfUp(value *big.Rat) (int64, bool) {
	if value == nil || value.Sign() < 0 {
		return 0, false
	}
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(value.Num(), value.Denom(), remainder)
	if new(big.Int).Lsh(remainder, 1).Cmp(value.Denom()) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if !quotient.IsInt64() {
		return 0, false
	}
	return quotient.Int64(), true
}

func (s *Service) CreateSourcingCostVersion(ownerID, sourceID int64, in *CreateSourcingCostVersionInput) (*SourcingCostVersionDetail, error) {
	if ownerID <= 0 || sourceID <= 0 {
		return nil, ErrInvalidWorkflow
	}
	lines, total, err := validatePreciseCostInput(in)
	if err != nil {
		return nil, err
	}
	var out SourcingCostVersionDetail
	err = s.db.Transaction(func(tx *gorm.DB) error {
		link, err := requireTaskSourcingAuthority(tx, sourceID, ownerID, in.TaskLinkID)
		if err != nil {
			return err
		}
		if link.ProductOpportunityID == nil || link.OpportunityDecisionID == nil {
			return fmt.Errorf("%w: frozen opportunity authority missing", ErrWorkflowGate)
		}
		var snapshot Sourcing1688Snapshot
		if err := tx.Where("id = ? AND sourcing_product_id = ?", in.SourceSnapshotID, sourceID).First(&snapshot).Error; err != nil {
			return fmt.Errorf("%w: source snapshot mismatch", ErrWorkflowGate)
		}
		var mapping SourcingSKUMapping
		if err := tx.Where("id = ? AND owner_id = ? AND sourcing_product_id = ? AND task_link_id = ? AND snapshot_id = ? AND product_opportunity_id = ?", in.SKUMappingID, ownerID, sourceID, link.ID, in.SourceSnapshotID, *link.ProductOpportunityID).First(&mapping).Error; err != nil {
			return fmt.Errorf("%w: canonical SKU authority mismatch", ErrWorkflowGate)
		}
		var latest int64
		if err := tx.Model(&SourcingCostVersion{}).Where("owner_id = ? AND task_link_id = ? AND sku_mapping_id = ?", ownerID, link.ID, mapping.ID).Select("COALESCE(MAX(version), 0)").Scan(&latest).Error; err != nil {
			return err
		}
		row := SourcingCostVersion{OwnerID: ownerID, SourcingProductID: sourceID, TaskLinkID: link.ID, ProductOpportunityID: *link.ProductOpportunityID, OpportunityDecisionID: *link.OpportunityDecisionID, SourceSnapshotID: in.SourceSnapshotID, SKUMappingID: mapping.ID, Version: latest + 1, TargetCurrency: in.TargetCurrency, TotalMinor: total, CreatedBy: ownerID}
		row.ContentHash = preciseCostHash(row, lines)
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		persisted := make([]SourcingCostLine, 0, len(lines))
		for _, line := range lines {
			persisted = append(persisted, SourcingCostLine{CostVersionID: row.ID, CostType: line.CostType, AmountMinor: line.AmountMinor, Currency: line.Currency, NormalizedAmountMinor: line.NormalizedAmountMinor, ExchangeRateDecimal: line.ExchangeRateDecimal, ExchangeRateSourceURI: line.ExchangeRateSourceURI, ExchangeRateObservedAt: line.ExchangeRateObservedAt, TruthStatus: line.TruthStatus, SourceURI: line.SourceURI, ObservedAt: line.ObservedAt})
		}
		if err := tx.Create(&persisted).Error; err != nil {
			return err
		}
		out = SourcingCostVersionDetail{Version: row, Lines: persisted}
		return nil
	})
	return &out, err
}

func preciseCostHash(row SourcingCostVersion, lines []CreateSourcingCostLineInput) string {
	payload, _ := json.Marshal(struct {
		OwnerID, SourceID, TaskID, OpportunityID, DecisionID, SnapshotID, SKUID, Version int64
		Currency                                                                         string
		Total                                                                            int64
		Lines                                                                            []CreateSourcingCostLineInput
	}{row.OwnerID, row.SourcingProductID, row.TaskLinkID, row.ProductOpportunityID, row.OpportunityDecisionID, row.SourceSnapshotID, row.SKUMappingID, row.Version, row.TargetCurrency, row.TotalMinor, lines})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func (s *Service) ListSourcingCostVersions(ownerID, sourceID int64) ([]SourcingCostVersionDetail, error) {
	if err := s.RequireSourceOwner(sourceID, ownerID); err != nil {
		return nil, err
	}
	var versions []SourcingCostVersion
	if err := s.db.Where("owner_id = ? AND sourcing_product_id = ?", ownerID, sourceID).Order("id DESC").Find(&versions).Error; err != nil {
		return nil, err
	}
	out := make([]SourcingCostVersionDetail, 0, len(versions))
	for _, version := range versions {
		var lines []SourcingCostLine
		if err := s.db.Where("cost_version_id = ?", version.ID).Order("cost_type").Find(&lines).Error; err != nil {
			return nil, err
		}
		out = append(out, SourcingCostVersionDetail{Version: version, Lines: lines})
	}
	return out, nil
}
