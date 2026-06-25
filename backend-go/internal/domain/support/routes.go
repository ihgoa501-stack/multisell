package support

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers support routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	conversations := rg.Group("/support/conversations")
	{
		conversations.GET("", h.ListConversations)
		conversations.GET("/:id", h.GetConversation)
		conversations.POST("", h.CreateConversation)
		conversations.PUT("/:id", h.UpdateConversation)
		conversations.DELETE("/:id", h.DeleteConversation)
		conversations.POST("/:id/reply", h.SendReply)
		conversations.POST("/:id/close", h.CloseConversation)
		conversations.GET("/:id/messages", h.GetMessages)
	}

	templates := rg.Group("/support/templates")
	{
		templates.GET("", h.ListTemplates)
		templates.GET("/:id", h.GetTemplate)
		templates.POST("", h.CreateTemplate)
		templates.PUT("/:id", h.UpdateTemplate)
		templates.DELETE("/:id", h.DeleteTemplate)
	}

	blacklist := rg.Group("/support/blacklist")
	{
		blacklist.GET("", h.ListBlacklist)
		blacklist.POST("", h.AddBlacklist)
		blacklist.GET("/check", h.CheckBlacklist)
		blacklist.DELETE("/:id", h.DeleteBlacklist)
	}
}
