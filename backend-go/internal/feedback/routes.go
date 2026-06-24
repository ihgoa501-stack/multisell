package feedback

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers feedback routes on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := NewService(db, logger)
	h := NewHandler(svc)

	fb := rg.Group("/feedback")
	{
		// Projects
		fb.GET("/projects", h.ListProjects)
		fb.GET("/projects/:id", h.GetProject)
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
		fb.GET("/submissions/:id", h.GetSubmission)
		fb.POST("/submissions", h.CreateSubmission)
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

		// Migration (admin)
		fb.POST("/migrate", h.Migrate)
	}
}
