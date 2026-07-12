package sourcing1688

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SourcingChangeEvent struct {
	ID                 int64           `gorm:"primaryKey" json:"id"`
	SourcingProductID  int64           `gorm:"not null;index" json:"sourcing_product_id"`
	PreviousSnapshotID *int64          `json:"previous_snapshot_id,omitempty"`
	CurrentSnapshotID  int64           `gorm:"not null;index" json:"current_snapshot_id"`
	ChangeType         string          `gorm:"size:40;not null" json:"change_type"`
	BeforeValue        json.RawMessage `gorm:"type:jsonb" json:"before_value,omitempty"`
	AfterValue         json.RawMessage `gorm:"type:jsonb" json:"after_value,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

func (SourcingChangeEvent) TableName() string { return "sourcing_1688_change_event" }

type DuplicateCandidate struct {
	ID               int64      `gorm:"primaryKey" json:"id"`
	SourceProductID  int64      `gorm:"not null;index" json:"source_product_id"`
	MatchedProductID int64      `gorm:"not null;index" json:"matched_product_id"`
	MatchType        string     `gorm:"size:32;not null" json:"match_type"`
	Fingerprint      string     `gorm:"size:64;not null" json:"fingerprint"`
	Status           string     `gorm:"size:20;not null" json:"status"`
	ReviewedBy       *int64     `json:"reviewed_by,omitempty"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

func (DuplicateCandidate) TableName() string { return "sourcing_1688_duplicate_candidate" }

type IdentityHistory struct {
	Snapshots  []Sourcing1688Snapshot `json:"snapshots"`
	Changes    []SourcingChangeEvent  `json:"changes"`
	Duplicates []DuplicateCandidate   `json:"duplicates"`
}

type ResolveDuplicateInput struct {
	ReviewedBy int64  `json:"reviewed_by"`
	Decision   string `json:"decision" binding:"required,oneof=same_product different_product"`
}

func productFingerprint(title string, variants json.RawMessage) (string, error) {
	var decoded any
	if len(variants) > 0 {
		if err := json.Unmarshal(variants, &decoded); err != nil {
			return "", fmt.Errorf("invalid variants for fingerprint: %w", err)
		}
	}
	decoded = canonicalIdentityValue(decoded)
	normalizedTitle := normalizeIdentityText(title)
	payload, err := json.Marshal(struct {
		Title    string `json:"title"`
		Variants any    `json:"variants"`
	}{normalizedTitle, decoded})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func normalizeIdentityText(value string) string {
	var out []rune
	spacePending := false
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			if spacePending && len(out) > 0 {
				out = append(out, ' ')
			}
			out = append(out, r)
			spacePending = false
		} else if len(out) > 0 {
			spacePending = true
		}
	}
	return strings.TrimSpace(string(out))
}

func canonicalIdentityValue(value any) any {
	switch typed := value.(type) {
	case string:
		return normalizeIdentityText(typed)
	case []any:
		items := make([]any, len(typed))
		for i := range typed {
			items[i] = canonicalIdentityValue(typed[i])
		}
		sort.Slice(items, func(i, j int) bool {
			left, _ := json.Marshal(items[i])
			right, _ := json.Marshal(items[j])
			return string(left) < string(right)
		})
		return items
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			result[normalizeIdentityText(key)] = canonicalIdentityValue(item)
		}
		return result
	default:
		return value
	}
}

func variantIdentity(variants *json.RawMessage) (string, error) {
	if variants == nil || len(*variants) == 0 {
		return "", nil
	}
	var decoded any
	if err := json.Unmarshal(*variants, &decoded); err != nil {
		return "", err
	}
	raw, err := json.Marshal(canonicalIdentityValue(decoded))
	return string(raw), err
}

func titleSimilarity(left, right string) float64 {
	a, b := []rune(normalizeIdentityText(left)), []rune(normalizeIdentityText(right))
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	if string(a) == string(b) {
		return 1
	}
	bigrams := func(input []rune) map[string]bool {
		result := map[string]bool{}
		if len(input) == 1 {
			result[string(input)] = true
			return result
		}
		for i := 0; i < len(input)-1; i++ {
			result[string(input[i:i+2])] = true
		}
		return result
	}
	leftSet, rightSet := bigrams(a), bigrams(b)
	intersection, union := 0, len(leftSet)
	for token := range rightSet {
		if leftSet[token] {
			intersection++
		} else {
			union++
		}
	}
	return float64(intersection) / float64(union)
}

func jsonValue(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func optionalFloatValue(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func (s *Service) recordIdentityAndChanges(tx *gorm.DB, p *Sourcing1688Product, previous *Sourcing1688Snapshot, current *Sourcing1688Snapshot) error {
	fingerprint := current.ProductFingerprint
	if fingerprint == "" || p.SourceProductFingerprint != fingerprint {
		return fmt.Errorf("%w: source and immutable snapshot fingerprints must match before identity recording", ErrInvalidWorkflow)
	}

	if previous != nil {
		changes := map[string][2]any{}
		if fmt.Sprint(optionalFloatValue(previous.ObservedPrice)) != fmt.Sprint(optionalFloatValue(current.ObservedPrice)) {
			changes["price"] = [2]any{optionalFloatValue(previous.ObservedPrice), optionalFloatValue(current.ObservedPrice)}
		}
		if previous.ObservedMOQ != current.ObservedMOQ {
			changes["moq"] = [2]any{previous.ObservedMOQ, current.ObservedMOQ}
		}
		if previous.ObservedSupplier != current.ObservedSupplier {
			changes["supplier"] = [2]any{previous.ObservedSupplier, current.ObservedSupplier}
		}
		if previous.ObservedSupplierBusinessID != current.ObservedSupplierBusinessID {
			changes["supplier_business_id"] = [2]any{previous.ObservedSupplierBusinessID, current.ObservedSupplierBusinessID}
		}
		if previous.SourceURL != current.SourceURL {
			changes["source_url"] = [2]any{previous.SourceURL, current.SourceURL}
		}
		if previous.ProductFingerprint != current.ProductFingerprint {
			changes["product_identity"] = [2]any{previous.ProductFingerprint, current.ProductFingerprint}
		}
		keys := make([]string, 0, len(changes))
		for key := range changes {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			values := changes[key]
			event := SourcingChangeEvent{SourcingProductID: p.ID, PreviousSnapshotID: &previous.ID, CurrentSnapshotID: current.ID, ChangeType: key, BeforeValue: jsonValue(values[0]), AfterValue: jsonValue(values[1])}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&event).Error; err != nil {
				return err
			}
		}
	}

	var matches []Sourcing1688Product
	if p.DemandCaseID == nil {
		return fmt.Errorf("%w: source demand case is required for duplicate isolation", ErrWorkflowGate)
	}
	var ownerID int64
	if err := tx.Model(&demandCaseRow{}).Select("owner_id").Where("id = ?", *p.DemandCaseID).Scan(&ownerID).Error; err != nil {
		return err
	}
	if err := tx.Table("sourcing_1688_product AS match").
		Select("match.*").
		Joins("JOIN demand_case match_case ON match_case.id = match.demand_case_id").
		Where("match.id <> ? AND match_case.owner_id = ?", p.ID, ownerID).
		Order("match.id").Scan(&matches).Error; err != nil {
		return err
	}
	currentVariants, err := variantIdentity(p.SkuVariants)
	if err != nil {
		return err
	}
	currentTitle := ""
	if p.Title != nil {
		currentTitle = *p.Title
	}
	for _, match := range matches {
		matchType := ""
		if match.SourceProductFingerprint == fingerprint {
			matchType = "content_fingerprint"
		} else {
			matchedVariants, variantErr := variantIdentity(match.SkuVariants)
			if variantErr != nil {
				return variantErr
			}
			matchedTitle := ""
			if match.Title != nil {
				matchedTitle = *match.Title
			}
			similarity := titleSimilarity(currentTitle, matchedTitle)
			if currentVariants != "" && currentVariants == matchedVariants && similarity >= 0.80 {
				matchType = "variant_title_similarity"
			} else if p.SupplierBusinessID != "" && p.SupplierBusinessID == match.SupplierBusinessID && similarity >= 0.70 {
				matchType = "supplier_title_similarity"
			}
		}
		if matchType == "" {
			continue
		}
		candidate := DuplicateCandidate{SourceProductID: p.ID, MatchedProductID: match.ID, MatchType: matchType, Fingerprint: fingerprint, Status: "pending_review"}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&candidate).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) GetIdentityHistory(id int64) (*IdentityHistory, error) {
	var history IdentityHistory
	if err := s.db.Where("sourcing_product_id = ?", id).Order("id DESC").Find(&history.Snapshots).Error; err != nil {
		return nil, err
	}
	if len(history.Snapshots) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if err := s.db.Where("sourcing_product_id = ?", id).Order("id DESC").Find(&history.Changes).Error; err != nil {
		return nil, err
	}
	if err := s.db.Where("source_product_id = ? OR matched_product_id = ?", id, id).Order("id DESC").Find(&history.Duplicates).Error; err != nil {
		return nil, err
	}
	return &history, nil
}

func (s *Service) ResolveDuplicate(candidateID int64, in *ResolveDuplicateInput) (*DuplicateCandidate, error) {
	if in.ReviewedBy <= 0 || (in.Decision != "same_product" && in.Decision != "different_product") {
		return nil, ErrInvalidWorkflow
	}
	var candidate DuplicateCandidate
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&candidate, candidateID).Error; err != nil {
			return err
		}
		if candidate.Status != "pending_review" {
			return fmt.Errorf("%w: duplicate candidate already reviewed", ErrWorkflowGate)
		}
		var source Sourcing1688Product
		if err := tx.First(&source, candidate.SourceProductID).Error; err != nil {
			return err
		}
		var dc demandCaseRow
		if source.DemandCaseID == nil || tx.First(&dc, *source.DemandCaseID).Error != nil || dc.OwnerID != in.ReviewedBy {
			return fmt.Errorf("%w: duplicate decision requires workflow Owner", ErrWorkflowGate)
		}
		var matched Sourcing1688Product
		var matchedDC demandCaseRow
		if err := tx.First(&matched, candidate.MatchedProductID).Error; err != nil {
			return err
		}
		if matched.DemandCaseID == nil || tx.First(&matchedDC, *matched.DemandCaseID).Error != nil || matchedDC.OwnerID != in.ReviewedBy {
			return fmt.Errorf("%w: duplicate candidates must belong to the same workflow Owner", ErrWorkflowGate)
		}
		now := time.Now().UTC()
		if err := tx.Model(&candidate).Updates(map[string]any{"status": in.Decision, "reviewed_by": in.ReviewedBy, "reviewed_at": now}).Error; err != nil {
			return err
		}
		candidate.Status, candidate.ReviewedBy, candidate.ReviewedAt = in.Decision, &in.ReviewedBy, &now
		return nil
	})
	return &candidate, err
}
