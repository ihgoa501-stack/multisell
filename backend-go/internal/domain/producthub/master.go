package producthub

import "time"

// Product lifecycle status constants.
const (
	LifecycleIdea        = "idea"
	LifecycleResearching = "researching"
	LifecycleSampling    = "sampling"
	LifecycleApproved    = "approved"
	LifecycleCosted      = "costed"
	LifecycleReadyToList = "ready_to_list"
	LifecycleListed      = "listed"
	LifecycleActive      = "active"
	LifecycleSunset      = "sunset"
	LifecycleArchived    = "archived"
)

// ValidLifecycleStatuses returns all valid lifecycle states in order.
func ValidLifecycleStatuses() []string {
	return []string{
		LifecycleIdea, LifecycleResearching, LifecycleSampling,
		LifecycleApproved, LifecycleCosted, LifecycleReadyToList,
		LifecycleListed, LifecycleActive, LifecycleSunset, LifecycleArchived,
	}
}

// IsValidLifecycleStatus checks if s is a valid status.
func IsValidLifecycleStatus(s string) bool {
	for _, v := range ValidLifecycleStatuses() {
		if v == s {
			return true
		}
	}
	return false
}

// BusinessModel constants.
const (
	BusinessOEM          = "oem"
	BusinessODM          = "odm"
	BusinessCatalog      = "catalog"
	BusinessPrivateLabel = "private_label"
)

// ValidBusinessModels returns all valid business model values.
func ValidBusinessModels() []string {
    return []string{BusinessOEM, BusinessODM, BusinessCatalog, BusinessPrivateLabel}
}

// IsValidBusinessModel checks if s is a valid business model.
func IsValidBusinessModel(s string) bool {
    for _, v := range ValidBusinessModels() {
        if v == s {
            return true
        }
    }
    return false
}

// ProductMaster maps to the "product_master" table.
type ProductMaster struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductCode     string    `gorm:"column:product_code;uniqueIndex;size:64" json:"product_code"`
	Name            string    `gorm:"column:name;size:256;not null" json:"name"`
	BrandID         *int64    `gorm:"column:brand_id" json:"brand_id,omitempty"`
	CategoryID      *int64    `gorm:"column:category_id" json:"category_id,omitempty"`
	BusinessModel   string    `gorm:"column:business_model;size:32;default:catalog" json:"business_model"`
	LifecycleStatus string    `gorm:"column:lifecycle_status;size:32;default:idea" json:"lifecycle_status"`
	OwnerID         int64     `gorm:"column:owner_id" json:"owner_id"`
	TeamID          *int64    `gorm:"column:team_id" json:"team_id,omitempty"`
	Description     string    `gorm:"column:description;type:text" json:"description"`
	TargetMarket    string    `gorm:"column:target_market;size:128" json:"target_market"`
	TargetPrice     float64   `gorm:"column:target_price" json:"target_price"`
	TargetMargin    float64   `gorm:"column:target_margin" json:"target_margin"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (ProductMaster) TableName() string { return "product_master" }
