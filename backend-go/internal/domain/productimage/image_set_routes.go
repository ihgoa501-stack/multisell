package productimage

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterImageSetRoutes adds the Listing image-set API to a JWT-protected
// /product-images group. Keeping it separate lets router wiring stay explicit.
func RegisterImageSetRoutes(group *gin.RouterGroup, db *gorm.DB, image ImageService) {
	h := NewImageSetHandler(db, image)
	group.POST("/image-sets", h.Create)
	group.GET("/image-sets/:set_id", h.Get)
	group.POST("/image-sets/:set_id/freeze", h.Freeze)
}
