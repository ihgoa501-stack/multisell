package {{ModuleName}}

import "time"

// {{ModuleName}} maps to the PostgreSQL "{{module_name}}" table.
type {{ModuleName}} struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"column:name;not null;unique" json:"name"`
	Status    int16     `gorm:"column:status;default:1" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func ({{ModuleName}}) TableName() string { return "{{module_name}}" }
