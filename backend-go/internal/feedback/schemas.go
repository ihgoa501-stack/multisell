package feedback


// ===================== Requests =====================

type CreateSubmissionRequest struct {
	ProjectID    int64   `json:"project_id" binding:"required"`
	CategoryID   *int64  `json:"category_id"`
	Title        string  `json:"title" binding:"required,min=1,max=500"`
	Description  string  `json:"description" binding:"required"`
	FeedbackType string  `json:"feedback_type" binding:"required,oneof=bug feature improvement question other"`
	Severity     string  `json:"severity" binding:"oneof=critical major minor trivial"`
	URL          string  `json:"url"`
	UserAgent    string  `json:"user_agent"`
	Attachments  string  `json:"attachments"`
}

type UpdateSubmissionStatusRequest struct {
	Status        string `json:"status" binding:"required,oneof=pending under_review accepted rejected planned in_progress shipped declined"`
	ReviewerNotes string `json:"reviewer_notes"`
	RejectReason  string `json:"reject_reason"`
	AssignedTo    *int64 `json:"assigned_to"`
}

type UpdateSubmissionRequest struct {
	CategoryID   *int64 `json:"category_id"`
	Title        string `json:"title" binding:"min=1,max=500"`
	Description  string `json:"description"`
	FeedbackType string `json:"feedback_type" binding:"oneof=bug feature improvement question other"`
	Severity     string `json:"severity" binding:"oneof=critical major minor trivial"`
	Priority     *int   `json:"priority" binding:"min=0,max=100"`
}

type CreateVoteRequest struct {
	VoteType string `json:"vote_type" binding:"required,oneof=upvote downvote"`
}

type CreateCommentRequest struct {
	Body       string `json:"body" binding:"required"`
	IsInternal bool   `json:"is_internal"`
}

type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Slug        string `json:"slug" binding:"required,min=1,max=100"`
	Description string `json:"description"`
	Settings    string `json:"settings"`
}

type UpdateProjectRequest struct {
	Name        string `json:"name" binding:"min=1,max=255"`
	Description string `json:"description"`
	Settings    string `json:"settings"`
}

type CreateCategoryRequest struct {
	ProjectID int64  `json:"project_id" binding:"required"`
	Name      string `json:"name" binding:"required,min=1,max=100"`
	Color     string `json:"color"`
	Icon      string `json:"icon"`
	SortOrder int    `json:"sort_order"`
}

type UpdateCategoryRequest struct {
	Name      string `json:"name" binding:"min=1,max=100"`
	Color     string `json:"color"`
	Icon      string `json:"icon"`
	SortOrder int    `json:"sort_order"`
}

type CreateTagRequest struct {
	ProjectID int64  `json:"project_id" binding:"required"`
	Name      string `json:"name" binding:"required,min=1,max=100"`
	Color     string `json:"color"`
}

// ===================== Responses =====================

type SubmissionResponse struct {
	ID            int64             `json:"id"`
	ProjectID     int64             `json:"project_id"`
	UserID        *int64            `json:"user_id,omitempty"`
	CategoryID    *int64            `json:"category_id,omitempty"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	FeedbackType  string            `json:"feedback_type"`
	Priority      int               `json:"priority"`
	Status        string            `json:"status"`
	Severity      string            `json:"severity"`
	Source        string            `json:"source"`
	URL           string            `json:"url"`
	Attachments   string            `json:"attachments"`
	AssignedTo    *int64            `json:"assigned_to,omitempty"`
	ReviewerNotes string            `json:"reviewer_notes,omitempty"`
	RejectReason  string            `json:"reject_reason,omitempty"`
	VoteCount     int64             `json:"vote_count"`
	CommentCount  int64             `json:"comment_count"`
	CreatedAt     string            `json:"created_at"`
	UpdatedAt     string            `json:"updated_at"`
	ReviewedAt    string            `json:"reviewed_at,omitempty"`
	ShippedAt     string            `json:"shipped_at,omitempty"`
	Project       *ProjectResponse  `json:"project,omitempty"`
	Category      *CategoryResponse `json:"category,omitempty"`
	Tags          []TagResponse     `json:"tags,omitempty"`
}

type SubmissionDetailResponse struct {
	SubmissionResponse
	Comments   []CommentResponse   `json:"comments"`
	StatusLogs []StatusLogResponse `json:"status_logs"`
	VoteCount  int64               `json:"vote_count"`
	UserVote   string              `json:"user_vote,omitempty"`
}

type ProjectResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Settings    string `json:"settings"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CategoryResponse struct {
	ID        int64  `json:"id"`
	ProjectID int64  `json:"project_id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	Icon      string `json:"icon"`
	SortOrder int    `json:"sort_order"`
	CreatedAt string `json:"created_at"`
}

type CommentResponse struct {
	ID           int64  `json:"id"`
	SubmissionID int64  `json:"submission_id"`
	UserID       int64  `json:"user_id"`
	Body         string `json:"body"`
	IsInternal   bool   `json:"is_internal"`
	CreatedAt    string `json:"created_at"`
}

type StatusLogResponse struct {
	ID         int64  `json:"id"`
	FromStatus string `json:"from_status"`
	ToStatus   string `json:"to_status"`
	ChangedBy  *int64 `json:"changed_by,omitempty"`
	Note       string `json:"note"`
	CreatedAt  string `json:"created_at"`
}

type TagResponse struct {
	ID        int64  `json:"id"`
	ProjectID int64  `json:"project_id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	CreatedAt string `json:"created_at"`
}

type DashboardStatsResponse struct {
	TotalSubmissions int64            `json:"total_submissions"`
	PendingReview    int64            `json:"pending_review"`
	Accepted         int64            `json:"accepted"`
	Shipped          int64            `json:"shipped"`
	ByType           map[string]int64 `json:"by_type"`
	ByStatus         map[string]int64 `json:"by_status"`
	AvgPriority      float64          `json:"avg_priority"`
}
