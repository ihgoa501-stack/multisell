package sourcing1688

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
)

const (
	CollectionPageList       = "list_lead"
	CollectionPageDetail     = "detail_observation"
	CollectionPageControlled = "controlled_fetch"
)

var collectionQualityFields = []string{"title", "price", "moq", "supplier", "images", "sku", "attributes"}

type CollectionFieldSource struct {
	PageKind   string    `json:"page_kind"`
	SnapshotID int64     `json:"snapshot_id"`
	ObservedAt time.Time `json:"observed_at"`
	Parser     string    `json:"parser"`
}

type CollectionFieldObservation struct {
	Field        string                `json:"field"`
	Status       string                `json:"status"`
	ValueSummary any                   `json:"value_summary,omitempty"`
	Source       CollectionFieldSource `json:"source"`
}

type CollectionQualityObservation struct {
	SnapshotID int64                                 `json:"snapshot_id"`
	PageKind   string                                `json:"page_kind"`
	ObservedAt time.Time                             `json:"observed_at"`
	Parser     string                                `json:"parser"`
	Fields     map[string]CollectionFieldObservation `json:"fields"`
}

type CollectionFieldConflict struct {
	Field   string                       `json:"field"`
	Values  []CollectionFieldObservation `json:"values"`
	Message string                       `json:"message"`
}

type CollectionRecaptureAction struct {
	Kind   string `json:"kind"`
	URL    string `json:"url,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type CollectionQuality struct {
	SourcingProductID           int64                                 `json:"sourcing_product_id"`
	SourceURL                   string                                `json:"source_url"`
	Observations                []CollectionQualityObservation        `json:"observations"`
	LatestListObservation       *CollectionQualityObservation         `json:"latest_list_observation,omitempty"`
	LatestDetailObservation     *CollectionQualityObservation         `json:"latest_detail_observation,omitempty"`
	LatestControlledObservation *CollectionQualityObservation         `json:"latest_controlled_observation,omitempty"`
	BestFields                  map[string]CollectionFieldObservation `json:"best_fields"`
	Conflicts                   []CollectionFieldConflict             `json:"conflicts"`
	Missing                     []string                              `json:"missing"`
	RecaptureAction             CollectionRecaptureAction             `json:"recapture_action"`
}

type collectionRawPayload struct {
	Driver             string            `json:"driver"`
	Title              string            `json:"title"`
	Price              float64           `json:"price_1688"`
	PriceModel         string            `json:"price_model"`
	PriceMin           *float64          `json:"price_min"`
	PriceMax           *float64          `json:"price_max"`
	PriceTiers         []json.RawMessage `json:"price_tiers"`
	MOQ                int               `json:"min_order_qty"`
	SupplierName       string            `json:"supplier_name"`
	SupplierBusinessID string            `json:"supplier_business_id"`
	Images             []json.RawMessage `json:"images"`
	Variants           []json.RawMessage `json:"spec_variants"`
	Attributes         map[string]any    `json:"attributes"`
	FieldStatuses      map[string]string `json:"field_statuses"`
}

func collectionPageKind(snapshot Sourcing1688Snapshot, raw collectionRawPayload) string {
	if snapshot.CaptureMode == CaptureModeControlledFetch {
		return CollectionPageControlled
	}
	if strings.Contains(strings.ToLower(raw.Driver), "list") || strings.Contains(strings.ToLower(snapshot.ParserVersion), "list-visible") {
		return CollectionPageList
	}
	return CollectionPageDetail
}

func collectionValueSummary(field string, raw collectionRawPayload, snapshot Sourcing1688Snapshot) any {
	switch field {
	case "title":
		if raw.Title != "" {
			return raw.Title
		}
		if snapshot.ObservedTitle != nil {
			return *snapshot.ObservedTitle
		}
	case "price":
		value := raw.Price
		if value == 0 && snapshot.ObservedPrice != nil {
			value = *snapshot.ObservedPrice
		}
		return map[string]any{"value": value, "model": raw.PriceModel, "min": raw.PriceMin, "max": raw.PriceMax, "tier_count": len(raw.PriceTiers)}
	case "moq":
		if raw.MOQ > 0 {
			return raw.MOQ
		}
		if snapshot.ObservedMOQ > 0 {
			return snapshot.ObservedMOQ
		}
	case "supplier":
		name, businessID := raw.SupplierName, raw.SupplierBusinessID
		if name == "" {
			name = snapshot.ObservedSupplier
		}
		if businessID == "" {
			businessID = snapshot.ObservedSupplierBusinessID
		}
		return map[string]any{"name": name, "business_id": businessID}
	case "images":
		return map[string]any{"count": len(raw.Images)}
	case "sku":
		return map[string]any{"count": len(raw.Variants)}
	case "attributes":
		return map[string]any{"count": len(raw.Attributes)}
	}
	return nil
}

func deriveCollectionObservation(snapshot Sourcing1688Snapshot) CollectionQualityObservation {
	var raw collectionRawPayload
	_ = json.Unmarshal(snapshot.RawPayload, &raw)
	pageKind := collectionPageKind(snapshot, raw)
	source := CollectionFieldSource{PageKind: pageKind, SnapshotID: snapshot.ID, ObservedAt: snapshot.CollectedAt, Parser: snapshot.ParserVersion}
	fields := make(map[string]CollectionFieldObservation, len(collectionQualityFields))
	for _, field := range collectionQualityFields {
		status := raw.FieldStatuses[field]
		if status == "" {
			switch field {
			case "title":
				if raw.Title != "" || snapshot.ObservedTitle != nil {
					status = "observed"
				}
			case "price":
				if raw.Price > 0 || snapshot.ObservedPrice != nil {
					status = "observed"
				}
			case "moq":
				if raw.MOQ > 0 || snapshot.ObservedMOQ > 0 {
					status = "observed"
				}
			case "supplier":
				if raw.SupplierName != "" || raw.SupplierBusinessID != "" || snapshot.ObservedSupplier != "" {
					status = "observed"
				}
			case "images":
				if len(raw.Images) > 0 {
					status = "observed"
				}
			case "sku":
				if len(raw.Variants) > 0 {
					status = "observed"
				}
			case "attributes":
				if len(raw.Attributes) > 0 {
					status = "observed"
				}
			}
			if status == "" {
				status = "unknown"
			}
		}
		fields[field] = CollectionFieldObservation{Field: field, Status: status, ValueSummary: collectionValueSummary(field, raw, snapshot), Source: source}
	}
	return CollectionQualityObservation{SnapshotID: snapshot.ID, PageKind: pageKind, ObservedAt: snapshot.CollectedAt, Parser: snapshot.ParserVersion, Fields: fields}
}

func collectionSourceRank(kind string) int {
	switch kind {
	case CollectionPageControlled:
		return 3
	case CollectionPageDetail:
		return 2
	default:
		return 1
	}
}

func collectionFieldUsable(field, status string) bool {
	return status == "observed" || (field == "sku" && status == "no_sku")
}

func collectionSummaryKey(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

// GetCollectionQuality derives an Owner-facing quality view entirely from
// immutable snapshots. It deliberately does not trust or update the product's
// current snapshot pointer, so observations made after governance remain visible.
func (s *Service) GetCollectionQuality(productID, ownerID int64) (*CollectionQuality, error) {
	if productID <= 0 || ownerID <= 0 {
		return nil, ErrInvalidWorkflow
	}
	if err := s.RequireSourceOwner(productID, ownerID); err != nil {
		return nil, err
	}
	var product Sourcing1688Product
	if err := s.db.Where("id = ?", productID).First(&product).Error; err != nil {
		return nil, err
	}
	var snapshots []Sourcing1688Snapshot
	if err := s.db.Where("sourcing_product_id = ?", productID).Order("collected_at ASC, id ASC").Find(&snapshots).Error; err != nil {
		return nil, err
	}

	quality := &CollectionQuality{SourcingProductID: productID, SourceURL: product.SourceURL, Observations: make([]CollectionQualityObservation, 0, len(snapshots)), BestFields: map[string]CollectionFieldObservation{}, Conflicts: []CollectionFieldConflict{}, Missing: []string{}}
	values := map[string]map[string]CollectionFieldObservation{}
	for _, snapshot := range snapshots {
		observation := deriveCollectionObservation(snapshot)
		quality.Observations = append(quality.Observations, observation)
		copyObservation := observation
		switch observation.PageKind {
		case CollectionPageList:
			quality.LatestListObservation = &copyObservation
		case CollectionPageDetail:
			quality.LatestDetailObservation = &copyObservation
		case CollectionPageControlled:
			quality.LatestControlledObservation = &copyObservation
		}
		for _, field := range collectionQualityFields {
			candidate := observation.Fields[field]
			if !collectionFieldUsable(field, candidate.Status) {
				continue
			}
			current, exists := quality.BestFields[field]
			if !exists || collectionSourceRank(candidate.Source.PageKind) > collectionSourceRank(current.Source.PageKind) || (collectionSourceRank(candidate.Source.PageKind) == collectionSourceRank(current.Source.PageKind) && candidate.Source.ObservedAt.After(current.Source.ObservedAt)) {
				quality.BestFields[field] = candidate
			}
			if values[field] == nil {
				values[field] = map[string]CollectionFieldObservation{}
			}
			values[field][collectionSummaryKey(candidate.ValueSummary)] = candidate
		}
	}
	for _, field := range collectionQualityFields {
		if _, ok := quality.BestFields[field]; !ok {
			quality.Missing = append(quality.Missing, field)
		}
		if len(values[field]) > 1 {
			items := make([]CollectionFieldObservation, 0, len(values[field]))
			for _, item := range values[field] {
				items = append(items, item)
			}
			sort.Slice(items, func(i, j int) bool {
				if items[i].Source.ObservedAt.Equal(items[j].Source.ObservedAt) {
					return items[i].Source.SnapshotID < items[j].Source.SnapshotID
				}
				return items[i].Source.ObservedAt.Before(items[j].Source.ObservedAt)
			})
			// Map summaries with only empty/default values do not create noise.
			if !reflect.DeepEqual(items[0].ValueSummary, nil) {
				quality.Conflicts = append(quality.Conflicts, CollectionFieldConflict{Field: field, Values: items, Message: fmt.Sprintf("%s在不同观察中存在不同值，请由Owner核对", field)})
			}
		}
	}
	if quality.LatestDetailObservation == nil {
		quality.RecaptureAction = CollectionRecaptureAction{Kind: "open_detail_page", URL: product.SourceURL, Reason: "尚无详情页观察，请打开1688详情页补采"}
	} else if len(quality.Missing) > 0 || len(quality.Conflicts) > 0 {
		quality.RecaptureAction = CollectionRecaptureAction{Kind: "retry_detail_collection", URL: product.SourceURL, Reason: "详情字段仍有缺失或冲突，请重新打开页面核对并补采"}
	} else {
		quality.RecaptureAction = CollectionRecaptureAction{Kind: "none"}
	}
	return quality, nil
}
