package brand

import "time"

// Brand maps to the PostgreSQL "brand" table.
type Brand struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"column:name;not null;unique" json:"name"`
	Logo        string    `gorm:"column:logo" json:"logo"`
	Description string    `gorm:"column:description" json:"description"`
	Status      int16     `gorm:"column:status;default:1" json:"status"`
	SortOrder   int       `gorm:"column:sort_order;default:0" json:"sort_order"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (Brand) TableName() string { return "brand" }
