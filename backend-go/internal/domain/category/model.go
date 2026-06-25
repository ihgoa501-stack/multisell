package category

import "time"

// Category maps to the PostgreSQL "category" table.
type Category struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"column:name;not null" json:"name"`
	ParentID  int64     `gorm:"column:parent_id;default:0" json:"parent_id"`
	Level     int       `gorm:"column:level;default:0" json:"level"`
	SortOrder int       `gorm:"column:sort_order;default:0" json:"sort_order"`
	Status    int16     `gorm:"column:status;default:1" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (Category) TableName() string { return "category" }
