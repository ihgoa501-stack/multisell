package common

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Pagination holds pagination parameters.
type Pagination struct {
	Page int `json:"page" form:"page"`
	Size int `json:"size" form:"size"`
}

// Offset returns the offset for database queries.
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.Size
}

// DefaultPagination returns a Pagination with sensible defaults.
func DefaultPagination() Pagination {
	return Pagination{Page: 1, Size: 20}
}

// Sort holds sorting parameters.
type Sort struct {
	Field string `json:"field" form:"sort_field"`
	Order string `json:"order" form:"sort_order"` // asc | desc
}

// ParsePagination extracts pagination from query parameters.
func ParsePagination(c *gin.Context) Pagination {
	p := DefaultPagination()
	if pageStr := c.Query("page"); pageStr != "" {
		if v, err := strconv.Atoi(pageStr); err == nil && v > 0 {
			p.Page = v
		}
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		if v, err := strconv.Atoi(sizeStr); err == nil && v > 0 && v <= 100 {
			p.Size = v
		}
	}
	return p
}

// ParseSort extracts sorting from query parameters.
func ParseSort(c *gin.Context) Sort {
	s := Sort{Field: "id", Order: "desc"}
	if f := c.Query("sort_field"); f != "" {
		s.Field = f
	}
	if o := c.Query("sort_order"); o == "asc" || o == "desc" {
		s.Order = o
	}
	return s
}

// ToFloat64 converts an interface{} to float64. Handles float64, json.Number,
// string, int, and int64 — the common types JSON unmarshaling produces.
func ToFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case string:
		return strconv.ParseFloat(val, 64)
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	default:
		return 0, convertError("float64", v)
	}
}

// ToInt64 converts an interface{} to int64. Handles float64 (truncation), string,
// int, and int64.
func ToInt64(v interface{}) (int64, error) {
	switch val := v.(type) {
	case float64:
		return int64(val), nil
	case string:
		return strconv.ParseInt(val, 10, 64)
	case int:
		return int64(val), nil
	case int64:
		return val, nil
	default:
		return 0, convertError("int64", v)
	}
}

// ToString converts an interface{} to string. Returns the empty string for nil.
func ToString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// ReviewerFromCtx extracts the reviewer identity from JWT context.
// Prefers the username claim, falls back to user_id, then "unknown".
func ReviewerFromCtx(c *gin.Context) string {
	if v, ok := c.Get("username"); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	if v, ok := c.Get("user_id"); ok {
		switch x := v.(type) {
		case float64:
			return fmt.Sprintf("user:%d", int64(x))
		case int64:
			return fmt.Sprintf("user:%d", x)
		}
	}
	return "unknown"
}

// UserIDFromCtx extracts the user_id from JWT context as *int64.
// Returns nil if the claim is missing or not a numeric type.
func UserIDFromCtx(c *gin.Context) *int64 {
	v, ok := c.Get("user_id")
	if !ok {
		return nil
	}
	switch x := v.(type) {
	case float64:
		id := int64(x)
		return &id
	case int64:
		return &x
	case int:
		id := int64(x)
		return &id
	default:
		return nil
	}
}

func convertError(typ string, v interface{}) error {
	return fmt.Errorf("cannot convert %T to %s", v, typ)
}
