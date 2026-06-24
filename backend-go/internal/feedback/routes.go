package feedback

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RegisterRoutes registers feedback routes on the given router group.
// classifyFn is an optional LLM chat function for AI-assisted classification.
//   ctx: request context with timeout
//   systemPrompt: system message for the LLM
//   userMessage: the feedback text to classify
//   returns: the LLM response text, or error
// If nil, classification falls back to keyword matching.
func RegisterRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB, logger *zap.Logger,
	classifyFn func(ctx context.Context, systemPrompt, userMessage string) (string, error)) {

	svc := NewService(db, logger)
	sugar := logger.Sugar()

	// Wire up AI classifier if a chat function is provided
	if classifyFn != nil {
		classifier := NewAIClassifier(classifyFn, sugar)
		svc.SetClassifier(classifier)
		logger.Info("AI classifier enabled for feedback submissions")
	}

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
		fb.GET("/projects/:id/categories", h.ListCategories)
		fb.POST("/categories", h.CreateCategory)
		fb.PUT("/categories/:id", h.UpdateCategory)
		fb.DELETE("/categories/:id", h.DeleteCategory)
		fb.GET("/projects/:id/tags", h.ListTags)
		fb.POST("/tags", h.CreateTag)
		fb.DELETE("/tags/:id", h.DeleteTag)
		fb.POST("/projects", h.CreateProject)
		fb.PUT("/projects/:id", h.UpdateProject)
		fb.DELETE("/projects/:id", h.DeleteProject)
		fb.GET("/projects/:id/submissions", h.ListSubmissions)
		fb.PUT("/submissions/:id", h.UpdateSubmission)
		fb.PUT("/submissions/:id/status", h.UpdateSubmissionStatus)
		fb.DELETE("/submissions/:id", h.DeleteSubmission)
		fb.GET("/mine", h.ListMySubmissions)
		fb.POST("/submissions/:id/vote", h.Vote)
		fb.GET("/submissions/:id/comments", h.ListComments)
		fb.POST("/submissions/:id/comments", h.AddComment)
		fb.DELETE("/comments/:id", h.DeleteComment)
		fb.POST("/submissions/:id/tags/:tagId", h.AddTag)
		fb.DELETE("/submissions/:id/tags/:tagId", h.RemoveTag)
		fb.GET("/projects/:id/stats", h.GetDashboardStats)
		fb.POST("/migrate", h.Migrate)
	}
}
