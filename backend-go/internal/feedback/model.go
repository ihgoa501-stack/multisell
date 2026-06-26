package feedback

import (
	"time"
)

// Project maps to the `feedback_project` table.
type Project struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"column:name;size:255;not null" json:"name"`
	Slug        string    `gorm:"column:slug;size:100;not null;uniqueIndex" json:"slug"`
	Description string    `gorm:"column:description;type:text" json:"description"`
	Settings    string    `gorm:"column:settings;type:jsonb;default:'{}'" json:"settings"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Project) TableName() string { return "feedback_project" }

// Category maps to the `feedback_category` table.
type Category struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProjectID int64     `gorm:"column:project_id;not null;index" json:"project_id"`
	Name      string    `gorm:"column:name;size:100;not null" json:"name"`
	Color     string    `gorm:"column:color;size:7" json:"color"`
	Icon      string    `gorm:"column:icon;size:50" json:"icon"`
	SortOrder int       `gorm:"column:sort_order;default:0" json:"sort_order"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Category) TableName() string { return "feedback_category" }

// Status constants.
const (
	StatusPending     = "pending"
	StatusUnderReview = "under_review"
	StatusAccepted    = "accepted"
	StatusRejected    = "rejected"
	StatusPlanned     = "planned"
	StatusInProgress  = "in_progress"
	StatusShipped     = "shipped"
	StatusDeclined    = "declined"
)

// Type constants.
const (
	TypeBug         = "bug"
	TypeFeature     = "feature"
	TypeImprovement = "improvement"
	TypeQuestion    = "question"
	TypeOther       = "other"
)

// Source constants.
const (
	SourceWidget   = "widget"
	SourcePortal   = "portal"
	SourceAPI      = "api"
	SourceInternal = "internal"
)

// Submission maps to the `feedback_submission` table.
type Submission struct {
	ID            int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProjectID     int64      `gorm:"column:project_id;not null;index" json:"project_id"`
	UserID        *int64     `gorm:"column:user_id;index" json:"user_id,omitempty"`
	CategoryID    *int64     `gorm:"column:category_id;index" json:"category_id,omitempty"`
	Title         string     `gorm:"column:title;size:500;not null" json:"title"`
	Description   string     `gorm:"column:description;type:text;not null" json:"description"`
	FeedbackType  string     `gorm:"column:feedback_type;size:20;default:feature" json:"feedback_type"`
	Priority      int        `gorm:"column:priority;default:0" json:"priority"`
	Status        string     `gorm:"column:status;size:20;default:pending;index" json:"status"`
	Severity      string     `gorm:"column:severity;size:20" json:"severity"`
	Source        string     `gorm:"column:source;size:50;default:widget" json:"source"`
	URL           string     `gorm:"column:url;type:text" json:"url"`
	UserAgent     string     `gorm:"column:user_agent;type:text" json:"user_agent"`
	Attachments   string     `gorm:"column:attachments;type:jsonb;default:'[]'" json:"attachments"`
	AssignedTo    *int64     `gorm:"column:assigned_to;index" json:"assigned_to,omitempty"`
	ReviewerNotes string     `gorm:"column:reviewer_notes;type:text" json:"reviewer_notes"`
	RejectReason  string     `gorm:"column:reject_reason;type:text" json:"reject_reason"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	ReviewedAt    *time.Time `gorm:"column:reviewed_at" json:"reviewed_at,omitempty"`
	ShippedAt     *time.Time `gorm:"column:shipped_at" json:"shipped_at,omitempty"`

	Project  *Project  `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Category *Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Tags     []Tag     `gorm:"many2many:feedback_submission_tag;" json:"tags,omitempty"`
}

func (Submission) TableName() string { return "feedback_submission" }

// Vote maps to the `feedback_vote` table.
type Vote struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SubmissionID int64     `gorm:"column:submission_id;not null;uniqueIndex:idx_submission_user" json:"submission_id"`
	UserID       int64     `gorm:"column:user_id;not null;uniqueIndex:idx_submission_user" json:"user_id"`
	VoteType     string    `gorm:"column:vote_type;size:10;default:upvote" json:"vote_type"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Vote) TableName() string { return "feedback_vote" }

// Comment maps to the `feedback_comment` table.
type Comment struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SubmissionID int64     `gorm:"column:submission_id;not null;index" json:"submission_id"`
	UserID       int64     `gorm:"column:user_id;not null" json:"user_id"`
	Body         string    `gorm:"column:body;type:text;not null" json:"body"`
	IsInternal   bool      `gorm:"column:is_internal;default:false" json:"is_internal"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Comment) TableName() string { return "feedback_comment" }

// StatusLog maps to the `feedback_status_log` table.
type StatusLog struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SubmissionID int64     `gorm:"column:submission_id;not null;index" json:"submission_id"`
	FromStatus   string    `gorm:"column:from_status;size:20" json:"from_status"`
	ToStatus     string    `gorm:"column:to_status;size:20;not null" json:"to_status"`
	ChangedBy    *int64    `gorm:"column:changed_by" json:"changed_by,omitempty"`
	Note         string    `gorm:"column:note;type:text" json:"note"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (StatusLog) TableName() string { return "feedback_status_log" }

// PushLog maps to the `feedback_push_log` table.
type PushLog struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SubmissionID   int64     `gorm:"column:submission_id;not null;index" json:"submission_id"`
	Destination    string    `gorm:"column:destination;size:50;not null" json:"destination"`
	DestinationURL string    `gorm:"column:destination_url;type:text" json:"destination_url"`
	ExternalID     string    `gorm:"column:external_id;size:255" json:"external_id"`
	Payload        string    `gorm:"column:payload;type:jsonb" json:"payload"`
	Status         string    `gorm:"column:status;size:20;default:pending" json:"status"`
	ErrorMessage   string    `gorm:"column:error_message;type:text" json:"error_message"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (PushLog) TableName() string { return "feedback_push_log" }

// Tag maps to the `feedback_tag` table.
type Tag struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProjectID int64     `gorm:"column:project_id;not null;index" json:"project_id"`
	Name      string    `gorm:"column:name;size:100;not null" json:"name"`
	Color     string    `gorm:"column:color;size:7" json:"color"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Tag) TableName() string { return "feedback_tag" }

// SubmissionTag maps to the `feedback_submission_tag` join table.
type SubmissionTag struct {
	SubmissionID int64 `gorm:"column:submission_id;primaryKey"`
	TagID        int64 `gorm:"column:tag_id;primaryKey"`
}

func (SubmissionTag) TableName() string { return "feedback_submission_tag" }
