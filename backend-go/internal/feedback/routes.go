package feedback

import (
	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers feedback routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	// Public routes (no auth required — for widget/portal submissions)
	fbPub := rg.Group("/feedback")
	{
		fbPub.POST("/submissions", h.CreateSubmission)
		fbPub.GET("/projects", h.ListProjects)
		fbPub.GET("/projects/:id", h.GetProject)
		fbPub.GET("/submissions/:id", h.GetSubmission)
	}

	// Authenticated routes (require valid JWT)
	fb := rg.Group("/feedback", middleware.Auth(cfg))
	{
		// Projects
		fb.POST("/projects", h.CreateProject)
		fb.PUT("/projects/:id", h.UpdateProject)
		fb.DELETE("/projects/:id", h.DeleteProject)

		// Categories
		fb.GET("/projects/:id/categories", h.ListCategories)
		fb.POST("/categories", h.CreateCategory)
		fb.PUT("/categories/:id", h.UpdateCategory)
		fb.DELETE("/categories/:id", h.DeleteCategory)

		// Tags
		fb.GET("/projects/:id/tags", h.ListTags)
		fb.POST("/tags", h.CreateTag)
		fb.DELETE("/tags/:id", h.DeleteTag)

		// Submissions
		fb.GET("/projects/:id/submissions", h.ListSubmissions)
		fb.PUT("/submissions/:id", h.UpdateSubmission)
		fb.PUT("/submissions/:id/status", h.UpdateSubmissionStatus)
		fb.DELETE("/submissions/:id", h.DeleteSubmission)

		// My submissions
		fb.GET("/mine", h.ListMySubmissions)

		// Votes
		fb.POST("/submissions/:id/vote", h.Vote)

		// Comments
		fb.GET("/submissions/:id/comments", h.ListComments)
		fb.POST("/submissions/:id/comments", h.AddComment)
		fb.DELETE("/comments/:id", h.DeleteComment)

		// Tags on submissions
		fb.POST("/submissions/:id/tags/:tagId", h.AddTag)
		fb.DELETE("/submissions/:id/tags/:tagId", h.RemoveTag)

		// Dashboard stats
		fb.GET("/projects/:id/stats", h.GetDashboardStats)

		// Migration (admin only — requires auth)
		fb.POST("/migrate", h.Migrate)
	}
}
