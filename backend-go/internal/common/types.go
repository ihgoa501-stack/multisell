package common

import (
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
