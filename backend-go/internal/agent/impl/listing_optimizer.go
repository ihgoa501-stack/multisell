// Package impl provides concrete agent implementations.
//
// ListingOptimizerAgent implements A2 Listing Optimizer business logic ported
// from backend/app/agent/agents/listing_optimizer.py (Python FastAPI codebase).
//
// Design docs: docs/aiagent/跨境电商AI_Agent深度调研报告.md §Agent2
//   - Keyword strategy + competitive deconstruction + copy generation
//   - Input: product info and competitor data
//   - Output: optimized listing (title, bullets, search terms, keyword research)
package impl

import (
	"fmt"
	"sort"
	"strings"
)

// ---------- Required context field names ----------

var listingOptimizerRequiredFields = []string{"product_name", "marketplace"}

// ---------- Helper types ----------

// keywordItem represents a keyword with its search volume.
type keywordItem struct {
	Word   string
	Volume float64
}

// ---------- ListingOptimizerAgent ----------

// ListingOptimizerAgent implements A2 Listing Optimizer logic.
//
// Decision points:
//   - "listing_optimize" — generates optimized title, bullets, search terms,
//     and actionable suggestions based on product features and keywords
//   - "keyword_research" — expands seed keywords into broader keyword candidates
//     by appending common suffixes (for, with, best)
type ListingOptimizerAgent struct{}

// NewListingOptimizerAgent creates a new ListingOptimizerAgent.
func NewListingOptimizerAgent() *ListingOptimizerAgent {
	return &ListingOptimizerAgent{}
}

// Decide dispatches to the correct decision handler based on decisionPoint.
//
// Supported decision points:
//   - "listing_optimize"
//   - "keyword_research"
//
// Returns: output map, confidence [0-1], riskLevel (low/medium/high), error.
func (a *ListingOptimizerAgent) Decide(decisionPoint string, ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	switch decisionPoint {
	case "listing_optimize":
		return a.optimize(ctx)
	case "keyword_research":
		return a.researchKeywords(ctx)
	default:
		return map[string]interface{}{
			"status":         "unknown",
			"decision_point": decisionPoint,
			"error":          fmt.Sprintf("unknown decision point: %s", decisionPoint),
		}, 0.0, "low", nil
	}
}

// ---------- Decision point: listing_optimize ----------

// optimize generates listing optimization suggestions based on context:
//
// Required context fields: product_name, marketplace
// Optional context fields: features, current_bullets, keywords
//
// Each keyword in the keywords list should be a map with "word" (string)
// and "volume" (float64) keys.
//
// Returns:
//   - title: optimized title with top-3 keywords prepended, capped at 200 chars
//   - bullets: up to 5 feature+keyword pairs
//   - search_terms: unique top-20 keywords not already in product name
//   - keyword_count: total number of input keywords
//   - suggestions: actionable improvement notes
func (a *ListingOptimizerAgent) optimize(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	// Validate required fields.
	if miss := missingFields(ctx, listingOptimizerRequiredFields); len(miss) > 0 {
		return insufficientData("listing_optimize", miss), 0.0, "low", nil
	}

	name := safeString(ctx["product_name"])
	mp := safeString(ctx["marketplace"], "US")
	features := parseStringList(ctx["features"])
	bulletsInput := parseStringList(ctx["current_bullets"])
	keywords := parseKeywords(ctx["keywords"])

	// Sort keywords by search volume descending, mirroring Python:
	//   sorted(k, key=lambda kw: _sf(kw.get("volume", 0)), reverse=True)
	sortedKW := sortKeywordsByVolume(keywords)

	// Title: Top 3 high-volume keywords + " - " + product_name, capped at 200 chars.
	topKW := topKeywords(sortedKW, 3)
	title := strings.Join(topKW, " ") + " - " + name
	if len(title) > 200 {
		title = title[:200]
	}

	// Bullets: Up to 5 features, each appended with a ranked keyword.
	var bullets []string
	for i, f := range features {
		if i >= 5 {
			break
		}
		kw := ""
		if i < len(sortedKW) {
			kw = sortedKW[i].Word
		}
		if kw != "" {
			bullets = append(bullets, fmt.Sprintf("%s — %s", f, kw))
		} else {
			bullets = append(bullets, f)
		}
	}

	// Search terms: Unique keywords, excluding those appearing in product name.
	nameLower := strings.ToLower(name)
	seenWord := make(map[string]struct{})
	var searchTerms []string
	for _, kw := range sortedKW {
		if len(searchTerms) >= 20 {
			break
		}
		w := strings.TrimSpace(kw.Word)
		if w == "" {
			continue
		}
		if _, seen := seenWord[w]; seen {
			continue
		}
		seenWord[w] = struct{}{}
		if strings.Contains(nameLower, strings.ToLower(w)) {
			continue
		}
		searchTerms = append(searchTerms, w)
	}

	// Suggestions.
	var suggestions []string
	if len(features) == 0 {
		suggestions = append(suggestions, "请补充产品卖点 features 以获得更精准的优化")
	}
	if len(bulletsInput) < 3 {
		suggestions = append(suggestions, "当前五点描述不足，建议补充至5条")
	}

	// Fall back to input bullets if no generated bullets.
	bulletsOutput := bullets
	if len(bulletsOutput) == 0 {
		bulletsOutput = bulletsInput
		if len(bulletsOutput) > 5 {
			bulletsOutput = bulletsOutput[:5]
		}
	}

	output = map[string]interface{}{
		"title":         title,
		"bullets":       bulletsOutput,
		"search_terms":  searchTerms,
		"marketplace":   mp,
		"keyword_count": len(sortedKW),
		"suggestions":   suggestions,
	}

	return output, 0.85, "low", nil
}

// ---------- Decision point: keyword_research ----------

// researchKeywords expands seed keywords into broader keyword candidates.
//
// Ported from Python _research_keywords:
//   Takes seed_keywords list and generates "S for", "S with", "S best" combos.
//
// Example: seed=["shoes","bags"] produces:
//   expanded=["shoes for","shoes with","shoes best","bags for","bags with","bags best"]
//   total_found = 6
func (a *ListingOptimizerAgent) researchKeywords(ctx map[string]interface{}) (output map[string]interface{}, confidence float64, riskLevel string, err error) {
	seed := parseStringList(ctx["seed_keywords"])
	if len(seed) == 0 {
		return insufficientData("keyword_research", []string{"seed_keywords"}), 0.0, "low", nil
	}

	suffixes := []string{"for", "with", "best"}
	expanded := make([]string, 0, len(seed)*len(suffixes))
	for _, s := range seed {
		for _, t := range suffixes {
			expanded = append(expanded, s+" "+t)
		}
	}

	output = map[string]interface{}{
		"seed":        seed,
		"expanded":    expanded,
		"total_found": len(seed) * 3,
	}

	return output, 0.80, "low", nil
}

// ---------- Helper: keyword parsing and sorting ----------

// parseKeywords extracts keyword items from an interface{} value.
//
// Input is expected to be []interface{} where each element is
// map[string]interface{} with at least "word" (string) and optionally
// "volume" (float64) keys (from JSON unmarshalling).
func parseKeywords(v interface{}) []keywordItem {
	if v == nil {
		return nil
	}
	list, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]keywordItem, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		out = append(out, keywordItem{
			Word:   safeString(m["word"]),
			Volume: safeFloat(m["volume"]),
		})
	}
	return out
}

// parseStringList extracts a []string from an interface{} value.
//
// Handles []interface{} (from JSON unmarshalling) where each element is a string.
// Returns nil when v is nil, not a list, or all elements are non-string.
func parseStringList(v interface{}) []string {
	if v == nil {
		return nil
	}
	list, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		s, ok := item.(string)
		if ok {
			out = append(out, s)
		}
	}
	return out
}

// sortKeywordsByVolume returns a new slice sorted by Volume descending.
// Does not modify the input slice (mirrors Python sorted()).
func sortKeywordsByVolume(items []keywordItem) []keywordItem {
	sorted := make([]keywordItem, len(items))
	copy(sorted, items)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Volume > sorted[j].Volume
	})
	return sorted
}

// topKeywords returns up to n keyword word strings from the front of items.
func topKeywords(items []keywordItem, n int) []string {
	if n > len(items) {
		n = len(items)
	}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = items[i].Word
	}
	return out
}
