package platformtruth

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/platform-truth", func(c *gin.Context) {
		response.Success(c, CurrentContract())
	})
}
