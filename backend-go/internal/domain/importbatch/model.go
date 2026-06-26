package importbatch

import (
	"encoding/json"
	"time"
)

// ImportBatch maps to the `import_batch` table.
type ImportBatch struct {
	ID           int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SourceType   string          `gorm:"column:source_type;size:30;not null" json:"source_type"`
	FileName     string          `gorm:"column:file_name;size:255" json:"file_name"`
	Status       string          `gorm:"column:status;size:20;not null;default:pending" json:"status"`
	TotalRows    int             `gorm:"column:total_rows;default:0" json:"total_rows"`
	SuccessCount int             `gorm:"column:success_count;default:0" json:"success_count"`
	ErrorCount   int             `gorm:"column:error_count;default:0" json:"error_count"`
	ErrorSummary string          `gorm:"column:error_summary;type:text" json:"error_summary"`
	CreatedBy    string          `gorm:"column:created_by;size:100" json:"created_by"`
	CreatedAt    time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (ImportBatch) TableName() string {
	return "import_batch"
}

// ImportBatchRow maps to the `import_batch_row` table.
type ImportBatchRow struct {
	ID           int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BatchID      int64           `gorm:"column:batch_id;not null;index" json:"batch_id"`
	RowIndex     int             `gorm:"column:row_index;not null" json:"row_index"`
	Status       string          `gorm:"column:status;size:20;not null;default:pending" json:"status"`
	RawData      json.RawMessage `gorm:"column:raw_data;type:jsonb" json:"raw_data"`
	ErrorMessage string          `gorm:"column:error_message;type:text" json:"error_message"`
	CreatedAt    time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default table name.
func (ImportBatchRow) TableName() string {
	return "import_batch_row"
}
