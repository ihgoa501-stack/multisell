package feedback

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/httpx/middleware"
	"github.com/lingmirror/backend-go/internal/realtime"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var publicSubmissionLimiter = middleware.NewRateLimiter(20, time.Minute)

// RegisterRoutes registers feedback routes on the given router group.
//
//	classifyFn: optional LLM chat function for AI classification
//	hub: optional WebSocket hub for real-time notifications
//	actionCreator: optional function to create AgentOS UnifiedActions
func RegisterRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB, logger *zap.Logger,
	classifyFn func(ctx context.Context, systemPrompt, userMessage string) (string, error),
	hub *realtime.Hub,
	actionCreator func(table, sourceID, title, payload string) error) {

	svc := NewService(db, logger)
	sugar := logger.Sugar()

	// Wire up AI classifier if a chat function is provided
	if classifyFn != nil {
		classifier := NewAIClassifier(classifyFn, sugar)
		svc.SetClassifier(classifier)
		logger.Info("AI classifier enabled for feedback submissions")
	}

	// Wire up AgentOS triage handler
	if actionCreator != nil && hub != nil {
		svc.SetTriageHandler(func(data *TriageActionData) error {
			// Create notification payload
			payload := fmt.Sprintf(`{"submission_id":%d,"title":"%s","type":"%s","priority":%d}`,
				data.SubmissionID, data.Title, data.FeedbackType, data.Priority)

			// Create UnifiedAction in AgentOS
			if err := actionCreator("feedback_submission",
				fmt.Sprintf("%d", data.SubmissionID),
				fmt.Sprintf("审阅反馈: %s", data.Title),
				payload); err != nil {
				logger.Warn("Failed to create triage action", zap.Error(err))
			}

			// Broadcast WebSocket notification
			wsMsg, _ := json.Marshal(map[string]interface{}{
				"type": "feedback_new",
				"data": map[string]interface{}{
					"id":            data.SubmissionID,
					"title":         data.Title,
					"feedback_type": data.FeedbackType,
					"priority":      data.Priority,
				},
			})
			hub.Broadcast(wsMsg)
			return nil
		})
		logger.Info("AgentOS triage handler wired for feedback submissions")
	}

	h := NewHandler(svc)

	// Public routes (no auth required — for widget/portal submissions)
	fbPub := rg.Group("/feedback")
	{
		fbPub.POST("/submissions", publicSubmissionLimiter.Limit(), h.CreateSubmission)
		fbPub.GET("/projects", h.ListProjects)
		fbPub.GET("/projects/:id", h.GetProject)
		fbPub.GET("/submissions/:id", h.GetSubmission)
	}

	// Authenticated routes (require valid JWT)
	fb := rg.Group("/feedback", middleware.Auth(cfg), middleware.ApprovalRequired(db, logger))
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
		fb.GET("/projects/:id/analytics", h.GetAnalytics)
		fb.POST("/migrate", h.Migrate)

		// Agent-facing endpoints
		fb.GET("/pending-for-agent", h.ListSubmissionsForAgent)
		fb.GET("/assigned-to-me", h.ListSubmissionsForAgent)
	}
}
