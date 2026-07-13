package sourcing1688

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type SKUWorkspaceTarget struct {
	SalesChannel         string  `json:"sales_channel"`
	TargetLocale         string  `json:"target_locale"`
	ProductOpportunityID *int64  `json:"product_opportunity_id,omitempty"`
	PlatformIDs          []int64 `json:"platform_ids"`
}

type SKUWorkspaceDimension struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
	Source string   `json:"source"`
}

type SKUWorkspaceDuplicate struct {
	Key     string `json:"key"`
	Indexes []int  `json:"indexes"`
}

type SKUWorkspaceMissingCombinations struct {
	Status       string              `json:"status"`
	Combinations []map[string]string `json:"combinations,omitempty"`
	Reason       string              `json:"reason,omitempty"`
}

type SKUWorkspaceCombination struct {
	Key         string              `json:"key"`
	SupplierSKU string              `json:"supplier_sku"`
	Spec        string              `json:"spec"`
	Values      map[string]string   `json:"values"`
	QuotedPrice *float64            `json:"quoted_price,omitempty"`
	StockStatus string              `json:"stock_status"`
	QuotedStock *int                `json:"quoted_stock,omitempty"`
	Issues      []string            `json:"issues"`
	Duplicate   bool                `json:"duplicate"`
	Mapping     *SourcingSKUMapping `json:"mapping,omitempty"`
}

type SKUWorkspace struct {
	SourceID              int64                           `json:"source_id"`
	TaskLinkID            int64                           `json:"task_link_id"`
	SnapshotID            int64                           `json:"snapshot_id,omitempty"`
	ObservedAt            *time.Time                      `json:"observed_at,omitempty"`
	Target                SKUWorkspaceTarget              `json:"target"`
	Dimensions            []SKUWorkspaceDimension         `json:"dimensions"`
	Combinations          []SKUWorkspaceCombination       `json:"combinations"`
	DuplicateCombinations []SKUWorkspaceDuplicate         `json:"duplicate_combinations"`
	MissingPrice          []string                        `json:"missing_price"`
	MissingStock          []string                        `json:"missing_stock"`
	MissingCombinations   SKUWorkspaceMissingCombinations `json:"missing_combinations"`
	CanonicalMappings     []SourcingSKUMapping            `json:"canonical_mappings"`
	Status                string                          `json:"status"`
	Blockers              []string                        `json:"blockers"`
}

type skuWorkspaceDimensionPayload struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type skuWorkspaceVariantPayload struct {
	SupplierSKU string            `json:"supplier_sku"`
	SKUID       string            `json:"sku_id"`
	Spec        string            `json:"spec"`
	Price       *float64          `json:"price"`
	Stock       *int              `json:"stock"`
	Values      map[string]string `json:"values"`
}

type skuWorkspacePayload struct {
	Variants   []skuWorkspaceVariantPayload   `json:"spec_variants"`
	Dimensions []skuWorkspaceDimensionPayload `json:"sku_dimensions"`
}

func cleanSKUValue(value string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(value), " "))
}

func skuCombinationKey(values map[string]string, spec string, dimensions []SKUWorkspaceDimension) string {
	parts := make([]string, 0, len(dimensions))
	for _, dimension := range dimensions {
		parts = append(parts, cleanSKUValue(values[dimension.Name]))
	}
	if len(parts) > 0 {
		return strings.Join(parts, "\x1f")
	}
	return strings.ToLower(cleanSKUValue(spec))
}

func deriveSKUDimensions(payload skuWorkspacePayload) []SKUWorkspaceDimension {
	if len(payload.Dimensions) > 0 {
		out := make([]SKUWorkspaceDimension, 0, len(payload.Dimensions))
		for _, dimension := range payload.Dimensions {
			name := cleanSKUValue(dimension.Name)
			seen, values := map[string]bool{}, []string{}
			for _, value := range dimension.Values {
				value = cleanSKUValue(value)
				if value != "" && !seen[value] {
					seen[value] = true
					values = append(values, value)
				}
			}
			if name != "" && len(values) > 0 {
				out = append(out, SKUWorkspaceDimension{Name: name, Values: values, Source: "declared"})
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	maxParts := 0
	for _, variant := range payload.Variants {
		if count := len(strings.Split(variant.Spec, "/")); count > maxParts {
			maxParts = count
		}
	}
	if maxParts == 0 {
		return []SKUWorkspaceDimension{}
	}
	values := make([][]string, maxParts)
	seen := make([]map[string]bool, maxParts)
	for i := range seen {
		seen[i] = map[string]bool{}
	}
	for _, variant := range payload.Variants {
		parts := strings.Split(variant.Spec, "/")
		for i, part := range parts {
			part = cleanSKUValue(part)
			if part != "" && !seen[i][part] {
				seen[i][part] = true
				values[i] = append(values[i], part)
			}
		}
	}
	out := make([]SKUWorkspaceDimension, 0, maxParts)
	for i := 0; i < maxParts; i++ {
		out = append(out, SKUWorkspaceDimension{Name: fmt.Sprintf("维度%d", i+1), Values: values[i], Source: "derived"})
	}
	return out
}

func variantValues(variant skuWorkspaceVariantPayload, dimensions []SKUWorkspaceDimension) map[string]string {
	values := map[string]string{}
	for key, value := range variant.Values {
		values[cleanSKUValue(key)] = cleanSKUValue(value)
	}
	if len(values) == 0 {
		parts := strings.Split(variant.Spec, "/")
		for i, dimension := range dimensions {
			if i < len(parts) {
				values[dimension.Name] = cleanSKUValue(parts[i])
			}
		}
	}
	return values
}

func missingSKUCartesian(dimensions []SKUWorkspaceDimension, observed map[string]bool) SKUWorkspaceMissingCombinations {
	if len(dimensions) == 0 {
		return SKUWorkspaceMissingCombinations{Status: "unknown", Reason: "详情页没有声明可核验的SKU维度"}
	}
	total := 1
	for _, dimension := range dimensions {
		if dimension.Source != "declared" || len(dimension.Values) == 0 {
			return SKUWorkspaceMissingCombinations{Status: "unknown", Reason: "SKU维度来自规格文本推导，不能可靠断言缺失组合"}
		}
		total *= len(dimension.Values)
		if total > 500 {
			return SKUWorkspaceMissingCombinations{Status: "unknown", Reason: "声明的SKU笛卡尔积过大，未自动计算"}
		}
	}
	missing := []map[string]string{}
	var walk func(int, map[string]string)
	walk = func(index int, current map[string]string) {
		if index == len(dimensions) {
			if !observed[skuCombinationKey(current, "", dimensions)] {
				copyValues := map[string]string{}
				for key, value := range current {
					copyValues[key] = value
				}
				missing = append(missing, copyValues)
			}
			return
		}
		dimension := dimensions[index]
		for _, value := range dimension.Values {
			current[dimension.Name] = value
			walk(index+1, current)
		}
	}
	walk(0, map[string]string{})
	return SKUWorkspaceMissingCombinations{Status: "calculated", Combinations: missing}
}

// GetSKUWorkspace is a read-only Owner/task view. It reports only variants in
// the latest immutable detail observation and never synthesizes SKU rows.
func (s *Service) GetSKUWorkspace(ownerID, sourceID, taskLinkID int64) (*SKUWorkspace, error) {
	if ownerID <= 0 || sourceID <= 0 || taskLinkID <= 0 {
		return nil, ErrInvalidWorkflow
	}
	link, err := findOwnedTaskLink(s.db, sourceID, ownerID, taskLinkID)
	if err != nil {
		return nil, err
	}
	workspace := &SKUWorkspace{SourceID: sourceID, TaskLinkID: taskLinkID, Dimensions: []SKUWorkspaceDimension{}, Combinations: []SKUWorkspaceCombination{}, DuplicateCombinations: []SKUWorkspaceDuplicate{}, MissingPrice: []string{}, MissingStock: []string{}, CanonicalMappings: []SourcingSKUMapping{}, Status: "no_detail_observation", Blockers: []string{"尚无详情页SKU观察，请打开1688详情页补采"}, MissingCombinations: SKUWorkspaceMissingCombinations{Status: "unknown", Reason: "尚无详情页观察"}, Target: SKUWorkspaceTarget{ProductOpportunityID: link.ProductOpportunityID, PlatformIDs: []int64{}}}
	var demand demandCaseRow
	if err := s.db.Where("id = ? AND owner_id = ?", link.DemandCaseID, ownerID).First(&demand).Error; err == nil {
		workspace.Target.SalesChannel, workspace.Target.TargetLocale = demand.SalesChannel, demand.TargetLocale
	}
	if err := s.db.Where("owner_id = ? AND sourcing_product_id = ? AND task_link_id = ?", ownerID, sourceID, taskLinkID).Order("version DESC, supplier_sku ASC").Find(&workspace.CanonicalMappings).Error; err != nil {
		return nil, err
	}
	platformSeen := map[int64]bool{}
	for _, mapping := range workspace.CanonicalMappings {
		if !platformSeen[mapping.PlatformID] {
			platformSeen[mapping.PlatformID] = true
			workspace.Target.PlatformIDs = append(workspace.Target.PlatformIDs, mapping.PlatformID)
		}
	}
	sort.Slice(workspace.Target.PlatformIDs, func(i, j int) bool { return workspace.Target.PlatformIDs[i] < workspace.Target.PlatformIDs[j] })

	var snapshots []Sourcing1688Snapshot
	if err := s.db.Where("sourcing_product_id = ?", sourceID).Order("collected_at DESC, id DESC").Find(&snapshots).Error; err != nil {
		return nil, err
	}
	var selected *Sourcing1688Snapshot
	for i := range snapshots {
		var raw collectionRawPayload
		_ = json.Unmarshal(snapshots[i].RawPayload, &raw)
		if collectionPageKind(snapshots[i], raw) == CollectionPageDetail {
			selected = &snapshots[i]
			break
		}
	}
	if selected == nil {
		return workspace, nil
	}
	var payload skuWorkspacePayload
	if err := json.Unmarshal(selected.RawPayload, &payload); err != nil {
		return nil, fmt.Errorf("%w: latest detail snapshot SKU payload is invalid", ErrInvalidWorkflow)
	}
	workspace.SnapshotID, workspace.ObservedAt = selected.ID, &selected.CollectedAt
	workspace.Dimensions = deriveSKUDimensions(payload)
	mappingBySupplier := map[string]*SourcingSKUMapping{}
	for i := range workspace.CanonicalMappings {
		mapping := workspace.CanonicalMappings[i]
		if _, exists := mappingBySupplier[mapping.SupplierSKU]; !exists {
			copyMapping := mapping
			mappingBySupplier[mapping.SupplierSKU] = &copyMapping
		}
	}
	indexes, observed := map[string][]int{}, map[string]bool{}
	for index, variant := range payload.Variants {
		values := variantValues(variant, workspace.Dimensions)
		key := skuCombinationKey(values, variant.Spec, workspace.Dimensions)
		supplierSKU := cleanSKUValue(variant.SupplierSKU)
		if supplierSKU == "" {
			supplierSKU = cleanSKUValue(variant.SKUID)
		}
		combination := SKUWorkspaceCombination{Key: key, SupplierSKU: supplierSKU, Spec: cleanSKUValue(variant.Spec), Values: values, QuotedPrice: variant.Price, StockStatus: "unknown", QuotedStock: variant.Stock, Issues: []string{}}
		if variant.Stock != nil {
			combination.StockStatus = "observed"
		} else {
			combination.Issues = append(combination.Issues, "missing_stock")
			workspace.MissingStock = append(workspace.MissingStock, key)
		}
		if variant.Price == nil {
			combination.Issues = append(combination.Issues, "missing_price")
			workspace.MissingPrice = append(workspace.MissingPrice, key)
		}
		if supplierSKU == "" {
			combination.Issues = append(combination.Issues, "missing_supplier_sku")
		} else if mapping := mappingBySupplier[supplierSKU]; mapping != nil {
			combination.Mapping = mapping
		}
		workspace.Combinations = append(workspace.Combinations, combination)
		indexes[key] = append(indexes[key], index)
		observed[key] = true
	}
	for key, duplicateIndexes := range indexes {
		if key != "" && len(duplicateIndexes) > 1 {
			workspace.DuplicateCombinations = append(workspace.DuplicateCombinations, SKUWorkspaceDuplicate{Key: key, Indexes: duplicateIndexes})
			for _, index := range duplicateIndexes {
				workspace.Combinations[index].Duplicate = true
				workspace.Combinations[index].Issues = append(workspace.Combinations[index].Issues, "duplicate_combination")
			}
		}
	}
	sort.Slice(workspace.DuplicateCombinations, func(i, j int) bool {
		return workspace.DuplicateCombinations[i].Key < workspace.DuplicateCombinations[j].Key
	})
	workspace.MissingCombinations = missingSKUCartesian(workspace.Dimensions, observed)
	workspace.Status, workspace.Blockers = "ready", []string{}
	if len(workspace.Combinations) == 0 {
		workspace.Status = "needs_attention"
		workspace.Blockers = append(workspace.Blockers, "详情页没有取得SKU组合")
	}
	if len(workspace.MissingPrice) > 0 || len(workspace.MissingStock) > 0 || len(workspace.DuplicateCombinations) > 0 || (workspace.MissingCombinations.Status == "calculated" && len(workspace.MissingCombinations.Combinations) > 0) {
		workspace.Status = "needs_attention"
		workspace.Blockers = append(workspace.Blockers, "SKU存在重复或字段/组合缺失，请Owner核对")
	}
	return workspace, nil
}
