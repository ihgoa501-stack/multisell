package competitor

import "time"

// CompetitorProduct maps to the "competitor_product" table.
type CompetitorProduct struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"column:name;not null" json:"name"`
	Platform    string    `gorm:"column:platform;not null" json:"platform"`
	ProductURL  string    `gorm:"column:product_url" json:"product_url"`
	SkuCode     string    `gorm:"column:sku_code" json:"sku_code"`
	Category    string    `gorm:"column:category" json:"category"`
	Brand       string    `gorm:"column:brand" json:"brand"`
	Status      int16     `gorm:"column:status;default:1" json:"status"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (CompetitorProduct) TableName() string { return "competitor_product" }

// PriceSnapshot maps to the "price_snapshot" table.
type PriceSnapshot struct {
	ID                 int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CompetitorID       int64     `gorm:"column:competitor_id;not null;index" json:"competitor_id"`
	Price              float64   `gorm:"column:price;not null;type:numeric(10,2)" json:"price"`
	OriginalPrice      float64   `gorm:"column:original_price;type:numeric(10,2)" json:"original_price,omitempty"`
	Currency           string    `gorm:"column:currency;default:CNY" json:"currency"`
	SalesLast30d       int       `gorm:"column:sales_last_30d;default:0" json:"sales_last_30d"`
	Rating             float64   `gorm:"column:rating;default:0" json:"rating"`
	ReviewCount        int       `gorm:"column:review_count;default:0" json:"review_count"`
	IsInStock          bool      `gorm:"column:is_in_stock;default:true" json:"is_in_stock"`
	SnapshotDate       time.Time `gorm:"column:snapshot_date;not null" json:"snapshot_date"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (PriceSnapshot) TableName() string { return "price_snapshot" }

// PriceTrend is an API response struct showing price history for a competitor.
type PriceTrend struct {
	CompetitorID int64            `json:"competitor_id"`
	Snapshots    []PriceSnapshot  `json:"snapshots"`
	MinPrice     float64          `json:"min_price"`
	MaxPrice     float64          `json:"max_price"`
	AvgPrice     float64          `json:"avg_price"`
	CurrentPrice float64          `json:"current_price"`
	PriceChange7d float64         `json:"price_change_7d"`  // % change over 7 days
	PriceChange30d float64        `json:"price_change_30d"` // % change over 30 days
}

// CreateCompetitorInput is the payload for POST /competitors.
type CreateCompetitorInput struct {
	Name       string `json:"name" binding:"required"`
	Platform   string `json:"platform" binding:"required"`
	ProductURL string `json:"product_url"`
	SkuCode    string `json:"sku_code"`
	Category   string `json:"category"`
	Brand      string `json:"brand"`
}

// RecordPriceInput is the payload for POST /competitors/:id/prices.
type RecordPriceInput struct {
	Price         float64 `json:"price" binding:"required"`
	OriginalPrice float64 `json:"original_price"`
	Currency      string  `json:"currency"`
	SalesLast30d  int     `json:"sales_last_30d"`
	Rating        float64 `json:"rating"`
	ReviewCount   int     `json:"review_count"`
	IsInStock     bool    `json:"is_in_stock"`
	SnapshotDate  string  `json:"snapshot_date"` // RFC3339 date
}
