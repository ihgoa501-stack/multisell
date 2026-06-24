package feedback

import (
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides feedback business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new feedback Service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// ===================== Projects =====================

func (s *Service) ListProjects() ([]Project, error) {
	var projects []Project
	if err := s.db.Order("id desc").Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

func (s *Service) GetProject(id int64) (*Project, error) {
	var p Project
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) GetProjectBySlug(slug string) (*Project, error) {
	var p Project
	if err := s.db.Where("slug = ?", slug).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) CreateProject(req *CreateProjectRequest) (*Project, error) {
	p := &Project{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Settings:    req.Settings,
	}
	if p.Settings == "" {
		p.Settings = "{}"
	}
	if err := s.db.Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) UpdateProject(id int64, req *UpdateProjectRequest) (*Project, error) {
	p, err := s.GetProject(id)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		p.Name = req.Name
	}
	if req.Description != "" {
		p.Description = req.Description
	}
	if req.Settings != "" {
		p.Settings = req.Settings
	}
	if err := s.db.Save(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) DeleteProject(id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("project_id = ?", id).Delete(&Category{})
		tx.Where("project_id = ?", id).Delete(&Tag{})
		var subIDs []int64
		tx.Model(&Submission{}).Where("project_id = ?", id).Pluck("id", &subIDs)
		if len(subIDs) > 0 {
			tx.Where("submission_id IN ?", subIDs).Delete(&Vote{})
			tx.Where("submission_id IN ?", subIDs).Delete(&Comment{})
			tx.Where("submission_id IN ?", subIDs).Delete(&StatusLog{})
			tx.Where("submission_id IN ?", subIDs).Delete(&PushLog{})
			tx.Where("submission_id IN ?", subIDs).Delete(&SubmissionTag{})
			tx.Where("project_id = ?", id).Delete(&Submission{})
		}
		return tx.Delete(&Project{}, id).Error
	})
}

// ===================== Categories =====================

func (s *Service) ListCategories(projectID int64) ([]Category, error) {
	var cats []Category
	if err := s.db.Where("project_id = ?", projectID).Order("sort_order asc").Find(&cats).Error; err != nil {
		return nil, err
	}
	return cats, nil
}

func (s *Service) CreateCategory(req *CreateCategoryRequest) (*Category, error) {
	c := &Category{
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Color:     req.Color,
		Icon:      req.Icon,
		SortOrder: req.SortOrder,
	}
	if err := s.db.Create(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) UpdateCategory(id int64, req *UpdateCategoryRequest) (*Category, error) {
	var c Category
	if err := s.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	if req.Name != "" {
		c.Name = req.Name
	}
	if req.Color != "" {
		c.Color = req.Color
	}
	if req.Icon != "" {
		c.Icon = req.Icon
	}
	c.SortOrder = req.SortOrder
	if err := s.db.Save(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Service) DeleteCategory(id int64) error {
	return s.db.Delete(&Category{}, id).Error
}

// ===================== Tags =====================

func (s *Service) ListTags(projectID int64) ([]Tag, error) {
	var tags []Tag
	if err := s.db.Where("project_id = ?", projectID).Order("name asc").Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

func (s *Service) CreateTag(req *CreateTagRequest) (*Tag, error) {
	t := &Tag{
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Color:     req.Color,
	}
	if err := s.db.Create(t).Error; err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) DeleteTag(id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("tag_id = ?", id).Delete(&SubmissionTag{})
		return tx.Delete(&Tag{}, id).Error
	})
}

// ===================== Submissions =====================

func (s *Service) ListSubmissions(projectID int64, status, feedbackType, severity string, page, size int) ([]SubmissionResponse, int64, error) {
	q := s.db.Model(&Submission{}).Where("project_id = ?", projectID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if feedbackType != "" {
		q = q.Where("feedback_type = ?", feedbackType)
	}
	if severity != "" {
		q = q.Where("severity = ?", severity)
	}

	var total int64
	q.Count(&total)

	var submissions []Submission
	if err := q.Preload("Category").Preload("Tags").Order("priority desc, created_at desc").
		Offset((page - 1) * size).Limit(size).Find(&submissions).Error; err != nil {
		return nil, 0, err
	}

	responses := make([]SubmissionResponse, 0, len(submissions))
	for _, sub := range submissions {
		responses = append(responses, s.toResponse(&sub))
	}
	return responses, total, nil
}

func (s *Service) ListSubmissionsByUser(userID, projectID int64, page, size int) ([]SubmissionResponse, int64, error) {
	q := s.db.Model(&Submission{}).Where("user_id = ?", userID)
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}

	var total int64
	q.Count(&total)

	var submissions []Submission
	if err := q.Preload("Category").Preload("Tags").Order("created_at desc").
		Offset((page - 1) * size).Limit(size).Find(&submissions).Error; err != nil {
		return nil, 0, err
	}

	responses := make([]SubmissionResponse, 0, len(submissions))
	for _, sub := range submissions {
		responses = append(responses, s.toResponse(&sub))
	}
	return responses, total, nil
}

func (s *Service) GetSubmission(id int64) (*Submission, error) {
	var sub Submission
	if err := s.db.Preload("Project").Preload("Category").Preload("Tags").
		First(&sub, id).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func (s *Service) GetSubmissionDetail(id int64, currentUserID *int64) (*SubmissionDetailResponse, error) {
	sub, err := s.GetSubmission(id)
	if err != nil {
		return nil, err
	}

	var comments []Comment
	s.db.Where("submission_id = ?", id).Order("created_at asc").Find(&comments)

	var statusLogs []StatusLog
	s.db.Where("submission_id = ?", id).Order("created_at asc").Find(&statusLogs)

	var voteCount int64
	s.db.Model(&Vote{}).Where("submission_id = ? AND vote_type = 'upvote'", id).Count(&voteCount)

	detail := &SubmissionDetailResponse{
		SubmissionResponse: s.toResponse(sub),
		VoteCount:          voteCount,
	}

	for _, c := range comments {
		detail.Comments = append(detail.Comments, CommentResponse{
			ID:           c.ID,
			SubmissionID: c.SubmissionID,
			UserID:       c.UserID,
			Body:         c.Body,
			IsInternal:   c.IsInternal,
			CreatedAt:    c.CreatedAt.Format(time.RFC3339),
		})
	}
	for _, sl := range statusLogs {
		detail.StatusLogs = append(detail.StatusLogs, StatusLogResponse{
			ID:         sl.ID,
			FromStatus: sl.FromStatus,
			ToStatus:   sl.ToStatus,
			ChangedBy:  sl.ChangedBy,
			Note:       sl.Note,
			CreatedAt:  sl.CreatedAt.Format(time.RFC3339),
		})
	}

	if currentUserID != nil {
		var vote Vote
		if err := s.db.Where("submission_id = ? AND user_id = ?", id, *currentUserID).First(&vote).Error; err == nil {
			detail.UserVote = vote.VoteType
		}
	}

	return detail, nil
}

func (s *Service) CreateSubmission(req *CreateSubmissionRequest, userID *int64) (*Submission, error) {
	sub := &Submission{
		ProjectID:    req.ProjectID,
		UserID:       userID,
		CategoryID:   req.CategoryID,
		Title:        req.Title,
		Description:  req.Description,
		FeedbackType: req.FeedbackType,
		Severity:     req.Severity,
		Status:       StatusPending,
		Source:       SourcePortal,
		URL:          req.URL,
		UserAgent:    req.UserAgent,
		Attachments:  req.Attachments,
	}
	if userID == nil {
		sub.Source = SourceWidget
	}
	if sub.Attachments == "" {
		sub.Attachments = "[]"
	}
	sub.Priority = s.calculatePriority(sub)

	if err := s.db.Create(sub).Error; err != nil {
		return nil, err
	}
	s.logStatusChange(sub.ID, "", StatusPending, userID, "Initial submission")
	return sub, nil
}

func (s *Service) UpdateSubmissionStatus(id int64, req *UpdateSubmissionStatusRequest, changedBy int64) (*Submission, error) {
	sub, err := s.GetSubmission(id)
	if err != nil {
		return nil, err
	}

	oldStatus := sub.Status
	sub.Status = req.Status
	sub.ReviewerNotes = req.ReviewerNotes
	sub.AssignedTo = req.AssignedTo

	if req.Status == StatusRejected || req.Status == StatusDeclined {
		sub.RejectReason = req.RejectReason
	}

	now := time.Now()
	if req.Status == StatusAccepted || req.Status == StatusRejected || req.Status == StatusDeclined {
		sub.ReviewedAt = &now
	}
	if req.Status == StatusShipped {
		sub.ShippedAt = &now
	}

	if err := s.db.Save(sub).Error; err != nil {
		return nil, err
	}

	note := req.ReviewerNotes
	if note == "" && req.RejectReason != "" {
		note = req.RejectReason
	}
	s.logStatusChange(sub.ID, oldStatus, req.Status, &changedBy, note)
	return sub, nil
}

func (s *Service) UpdateSubmission(id int64, req *UpdateSubmissionRequest) (*Submission, error) {
	sub, err := s.GetSubmission(id)
	if err != nil {
		return nil, err
	}
	if req.Title != "" {
		sub.Title = req.Title
	}
	if req.Description != "" {
		sub.Description = req.Description
	}
	if req.FeedbackType != "" {
		sub.FeedbackType = req.FeedbackType
	}
	if req.Severity != "" {
		sub.Severity = req.Severity
	}
	if req.CategoryID != nil {
		sub.CategoryID = req.CategoryID
	}
	if req.Priority != nil {
		sub.Priority = *req.Priority
	}
	if err := s.db.Save(sub).Error; err != nil {
		return nil, err
	}
	return sub, nil
}

func (s *Service) DeleteSubmission(id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("submission_id = ?", id).Delete(&Vote{})
		tx.Where("submission_id = ?", id).Delete(&Comment{})
		tx.Where("submission_id = ?", id).Delete(&StatusLog{})
		tx.Where("submission_id = ?", id).Delete(&PushLog{})
		tx.Where("submission_id = ?", id).Delete(&SubmissionTag{})
		return tx.Delete(&Submission{}, id).Error
	})
}

// ===================== Votes =====================

func (s *Service) Vote(submissionID, userID int64, voteType string) (int64, error) {
	var existing Vote
	result := s.db.Where("submission_id = ? AND user_id = ?", submissionID, userID).First(&existing)

	if result.Error == nil {
		if existing.VoteType == voteType {
			if err := s.db.Delete(&existing).Error; err != nil {
				return 0, err
			}
		} else {
			existing.VoteType = voteType
			if err := s.db.Save(&existing).Error; err != nil {
				return 0, err
			}
		}
	} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		v := &Vote{SubmissionID: submissionID, UserID: userID, VoteType: voteType}
		if err := s.db.Create(v).Error; err != nil {
			return 0, err
		}
	} else {
		return 0, result.Error
	}

	var count int64
	s.db.Model(&Vote{}).Where("submission_id = ? AND vote_type = 'upvote'", submissionID).Count(&count)
	return count, nil
}

// ===================== Comments =====================

func (s *Service) AddComment(submissionID, userID int64, req *CreateCommentRequest) (*Comment, error) {
	c := &Comment{
		SubmissionID: submissionID,
		UserID:       userID,
		Body:         req.Body,
		IsInternal:   req.IsInternal,
	}
	if err := s.db.Create(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) ListComments(submissionID int64, includeInternal bool) ([]Comment, error) {
	q := s.db.Where("submission_id = ?", submissionID)
	if !includeInternal {
		q = q.Where("is_internal = false")
	}
	var comments []Comment
	if err := q.Order("created_at asc").Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

func (s *Service) DeleteComment(id int64) error {
	return s.db.Delete(&Comment{}, id).Error
}

// ===================== Push Log =====================

func (s *Service) RecordPush(submissionID int64, destination, destinationURL, externalID, payload, status, errMsg string) (*PushLog, error) {
	pl := &PushLog{
		SubmissionID:   submissionID,
		Destination:    destination,
		DestinationURL: destinationURL,
		ExternalID:     externalID,
		Payload:        payload,
		Status:         status,
		ErrorMessage:   errMsg,
	}
	if err := s.db.Create(pl).Error; err != nil {
		return nil, err
	}
	return pl, nil
}

// ===================== Tag Management =====================

func (s *Service) AddTagToSubmission(submissionID, tagID int64) error {
	return s.db.Create(&SubmissionTag{SubmissionID: submissionID, TagID: tagID}).Error
}

func (s *Service) RemoveTagFromSubmission(submissionID, tagID int64) error {
	return s.db.Where("submission_id = ? AND tag_id = ?", submissionID, tagID).Delete(&SubmissionTag{}).Error
}

// ===================== Dashboard Stats =====================

func (s *Service) GetDashboardStats(projectID int64) (*DashboardStatsResponse, error) {
	stats := &DashboardStatsResponse{
		ByType:   make(map[string]int64),
		ByStatus: make(map[string]int64),
	}

	s.db.Model(&Submission{}).Where("project_id = ?", projectID).Count(&stats.TotalSubmissions)

	type kv struct {
		Key   string
		Count int64
	}

	var sc []kv
	s.db.Model(&Submission{}).Select("status as key, count(*) as count").
		Where("project_id = ?", projectID).Group("status").Find(&sc)
	for _, s := range sc {
		stats.ByStatus[s.Key] = s.Count
	}
	stats.PendingReview = stats.ByStatus[StatusPending] + stats.ByStatus[StatusUnderReview]

	var tc []kv
	s.db.Model(&Submission{}).Select("feedback_type as key, count(*) as count").
		Where("project_id = ?", projectID).Group("feedback_type").Find(&tc)
	for _, t := range tc {
		stats.ByType[t.Key] = t.Count
	}

	stats.Accepted = stats.ByStatus[StatusAccepted]
	stats.Shipped = stats.ByStatus[StatusShipped]

	type avgRes struct{ Avg float64 }
	var ar avgRes
	s.db.Model(&Submission{}).Select("COALESCE(AVG(priority), 0) as avg").
		Where("project_id = ?", projectID).Scan(&ar)
	stats.AvgPriority = ar.Avg

	return stats, nil
}

// ===================== Helpers =====================

func (s *Service) toResponse(sub *Submission) SubmissionResponse {
	resp := SubmissionResponse{
		ID:            sub.ID,
		ProjectID:     sub.ProjectID,
		UserID:        sub.UserID,
		CategoryID:    sub.CategoryID,
		Title:         sub.Title,
		Description:   sub.Description,
		FeedbackType:  sub.FeedbackType,
		Priority:      sub.Priority,
		Status:        sub.Status,
		Severity:      sub.Severity,
		Source:        sub.Source,
		URL:           sub.URL,
		Attachments:   sub.Attachments,
		AssignedTo:    sub.AssignedTo,
		ReviewerNotes: sub.ReviewerNotes,
		RejectReason:  sub.RejectReason,
		CreatedAt:     sub.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     sub.UpdatedAt.Format(time.RFC3339),
	}
	if sub.ReviewedAt != nil {
		resp.ReviewedAt = sub.ReviewedAt.Format(time.RFC3339)
	}
	if sub.ShippedAt != nil {
		resp.ShippedAt = sub.ShippedAt.Format(time.RFC3339)
	}
	if sub.Project != nil {
		resp.Project = &ProjectResponse{
			ID: sub.Project.ID, Name: sub.Project.Name, Slug: sub.Project.Slug,
			Description: sub.Project.Description, Settings: sub.Project.Settings,
			CreatedAt: sub.Project.CreatedAt.Format(time.RFC3339), UpdatedAt: sub.Project.UpdatedAt.Format(time.RFC3339),
		}
	}
	if sub.Category != nil {
		resp.Category = &CategoryResponse{
			ID: sub.Category.ID, ProjectID: sub.Category.ProjectID, Name: sub.Category.Name,
			Color: sub.Category.Color, Icon: sub.Category.Icon, SortOrder: sub.Category.SortOrder,
			CreatedAt: sub.Category.CreatedAt.Format(time.RFC3339),
		}
	}
	for _, t := range sub.Tags {
		resp.Tags = append(resp.Tags, TagResponse{
			ID: t.ID, ProjectID: t.ProjectID, Name: t.Name, Color: t.Color,
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
		})
	}

	var voteCount, commentCount int64
	s.db.Model(&Vote{}).Where("submission_id = ? AND vote_type = 'upvote'", sub.ID).Count(&voteCount)
	s.db.Model(&Comment{}).Where("submission_id = ?", sub.ID).Count(&commentCount)
	resp.VoteCount = voteCount
	resp.CommentCount = commentCount

	return resp
}

func (s *Service) logStatusChange(submissionID int64, from, to string, changedBy *int64, note string) {
	s.db.Create(&StatusLog{
		SubmissionID: submissionID,
		FromStatus:   from,
		ToStatus:     to,
		ChangedBy:    changedBy,
		Note:         note,
	})
}

func (s *Service) calculatePriority(sub *Submission) int {
	score := 0
	switch sub.Severity {
	case "critical":
		score += 40
	case "major":
		score += 25
	case "minor":
		score += 10
	case "trivial":
		score += 5
	default:
		score += 15
	}
	switch sub.FeedbackType {
	case TypeBug:
		score += 25
	case TypeFeature:
		score += 20
	case TypeImprovement:
		score += 15
	}
	if score > 100 {
		score = 100
	}
	return score
}

// AutoMigrate runs GORM auto-migration for all feedback tables.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Project{},
		&Category{},
		&Submission{},
		&Vote{},
		&Comment{},
		&StatusLog{},
		&PushLog{},
		&Tag{},
		&SubmissionTag{},
	)
}

// CreateDefaultProject creates a default project if none exist. Returns existing or new.
func (s *Service) CreateDefaultProject() (*Project, error) {
	var count int64
	s.db.Model(&Project{}).Count(&count)
	if count > 0 {
		var p Project
		s.db.First(&p)
		return &p, nil
	}
	return s.CreateProject(&CreateProjectRequest{
		Name: "MultiSell", Slug: "multisell",
		Description: "MultiSell 跨境电商 AI AgentOS", Settings: "{}",
	})
}
