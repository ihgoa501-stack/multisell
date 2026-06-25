package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Result is the standard API response envelope.
type Result struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageResult is the paginated API response envelope.
type PageResult struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	Size    int         `json:"size"`
}

// Success sends a 200 OK response with data.
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Result{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

// Error sends an error response with the given status code and message.
func Error(c *gin.Context, code int, message string) {
	c.JSON(code, Result{
		Code:    code,
		Message: message,
	})
}

// Paginated sends a paginated response.
func Paginated(c *gin.Context, data interface{}, total int64, page, size int) {
	c.JSON(http.StatusOK, PageResult{
		Code:    0,
		Message: "ok",
		Data:    data,
		Total:   total,
		Page:    page,
		Size:    size,
	})
}
