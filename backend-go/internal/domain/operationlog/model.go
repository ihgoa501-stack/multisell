package operationlog

import "time"

// OperationLog maps to the `operation_log` table.
type OperationLog struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Module     string    `gorm:"column:module;size:50" json:"module"`
	Action     string    `gorm:"column:action;size:50" json:"action"`
	ResourceID string    `gorm:"column:resource_id;size:100" json:"resource_id"`
	Content    string    `gorm:"column:content;type:text" json:"content"`
	Operator   string    `gorm:"column:operator;size:100" json:"operator"`
	UserID     int64     `gorm:"column:user_id" json:"user_id"`
	Result     string    `gorm:"column:result;size:20" json:"result"` // success/failure
	IP         string    `gorm:"column:ip;size:50" json:"ip"`
	Duration   int       `gorm:"column:duration" json:"duration"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default table name.
func (OperationLog) TableName() string {
	return "operation_log"
}
