// Package impl provides concrete agent implementations.
package impl

import (
	"fmt"
	"math"
	"strings"

	"gorm.io/gorm"
)

// queryCount is a nil-safe shortcut for GORM Count().
func (a *AftersalesMgmtAgent) queryCount(tx *gorm.DB) (int64, error) {
	if a.db == nil {
		return 0, fmt.Errorf("db is nil")
	}
	var cnt int64
	if err := tx.Count(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}

// parsePeriodDays converts a period string like "7d", "30d", "week", "month"
// into a number of days (defaults to 30).
func parsePeriodDays(period string) int {
	switch {
	case strings.HasSuffix(period, "d"):
		var n int
		if _, err := fmt.Sscanf(period, "%dd", &n); err == nil && n > 0 {
			return n
		}
	case strings.EqualFold(period, "week") || strings.EqualFold(period, "7d"):
		return 7
	case strings.EqualFold(period, "month") || strings.EqualFold(period, "30d"):
		return 30
	case strings.EqualFold(period, "quarter") || strings.EqualFold(period, "90d"):
		return 90
	}
	return 30
}

// calculateTrend compares two counts to determine direction.
func calculateTrend(current, previous int64) string {
	if previous == 0 {
		if current == 0 {
			return "stable"
		}
		return "increasing"
	}
	ratio := float64(current) / float64(previous)
	switch {
	case ratio > 1.1:
		return "increasing"
	case ratio < 0.9:
		return "decreasing"
	default:
		return "stable"
	}
}

// categorizeReason maps a raw reason string to a standard category.
func categorizeReason(reason string) string {
	lower := strings.ToLower(reason)
	switch {
	case strings.Contains(lower, "quality") || strings.Contains(lower, "质量") ||
		strings.Contains(lower, "defect") || strings.Contains(lower, "瑕疵") ||
		strings.Contains(lower, "malfunction") || strings.Contains(lower, "故障") ||
		strings.Contains(lower, "broken") || strings.Contains(lower, "坏了"):
		return "product_quality"
	case strings.Contains(lower, "物流") || strings.Contains(lower, "运输") ||
		(strings.Contains(lower, "damage") && !strings.Contains(lower, "defect")) ||
		strings.Contains(lower, "shipping") || strings.Contains(lower, "delivery") ||
		strings.Contains(lower, "外包装"):
		return "logistics_damage"
	case strings.Contains(lower, "not want") || strings.Contains(lower, "不想要") ||
		strings.Contains(lower, "change mind") || strings.Contains(lower, "改变主意") ||
		strings.Contains(lower, "不需要") || strings.Contains(lower, "no longer") ||
		strings.Contains(lower, "ordered wrong") || strings.Contains(lower, "买错"):
		return "buyer_remorse"
	case strings.Contains(lower, "wrong") || strings.Contains(lower, "错发") ||
		strings.Contains(lower, "不对") || strings.Contains(lower, "incorrect") ||
		strings.Contains(lower, "not match") || strings.Contains(lower, "不匹配"):
		return "wrong_item"
	default:
		return "other"
	}
}

// reasonCategoryLabel returns a Chinese label for a reason category.
func reasonCategoryLabel(cat string) string {
	switch cat {
	case "product_quality":
		return "产品质量问题"
	case "logistics_damage":
		return "物流运输损坏"
	case "buyer_remorse":
		return "买家原因(改变主意/不需要)"
	case "wrong_item":
		return "错发/漏发"
	case "other":
		return "其他原因"
	default:
		return cat
	}
}

// safeDeltaPct computes the percentage change between two KPI values.
func safeDeltaPct(current, previous float64) float64 {
	if previous == 0 {
		if current == 0 {
			return 0
		}
		return 100
	}
	return math.Round((current-previous)/previous*10000) / 100
}

// safeMapFloat extracts a nested float from a map-of-maps chain.
func safeMapFloat(v interface{}, key string) float64 {
	if v == nil {
		return 0
	}
	if m, ok := v.(map[string]interface{}); ok {
		return safeFloat(m[key])
	}
	return 0
}
