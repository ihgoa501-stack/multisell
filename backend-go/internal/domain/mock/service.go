package mock

import (
	"math/rand"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides mock data and sync status business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new mock data service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// SeedMockData populates the database with demo mock orders, settlements, and sync statuses.
func (s *Service) SeedMockData() error {
	// Check if seed already exists
	var count int64
	s.db.Model(&MockOrder{}).Where("is_seed_data = ?", true).Count(&count)
	if count > 0 {
		return nil // already seeded
	}

	// Seed mock orders
	now := time.Now()
	products := []string{
		"Wireless Bluetooth Earbuds", "Smart Watch Fitness Tracker",
		"LED Desk Lamp", "Water Bottle 500ml", "Cotton T-Shirt",
		"Wireless Charger", "External SSD 1TB", "Bamboo Cutting Board",
		"Yoga Mat Premium", "Handheld Mini Fan",
	}
	statuses := []string{"pending", "shipped", "delivered", "delivered", "cancelled"}

	for i, p := range products {
		for j := 0; j < 3; j++ {
			orderDate := now.AddDate(0, 0, -(j*7 + i%5))
			s.db.Create(&MockOrder{
				PlatformID:  int64(i%3 + 1),
				OrderNo:     "MOCK-" + randString(8),
				ProductName: p,
				Quantity:    rand.Intn(5) + 1,
				TotalAmount: float64(rand.Intn(100) + 10),
				Currency:    "USD",
				Status:      statuses[rand.Intn(len(statuses))],
				OrderDate:   orderDate,
				IsSeedData:  true,
			})
		}
	}

	// Seed mock settlements
	settlementPeriods := []string{"2026-04", "2026-05", "2026-06"}
	for _, period := range settlementPeriods {
		for platformID := int64(1); platformID <= 3; platformID++ {
			s.db.Create(&MockSettlement{
				PlatformID:  platformID,
				Period:      period,
				TotalRevenue: float64(rand.Intn(50000) + 5000),
				TotalFee:    float64(rand.Intn(5000) + 500),
				NetAmount:   float64(rand.Intn(45000) + 4500),
				Currency:    "USD",
				OrderCount:  rand.Intn(200) + 20,
				IsSeedData:  true,
			})
		}
	}

	// Seed mock sync statuses
	platformNames := map[int64]string{1: "Ozon", 2: "Shopee", 3: "Lazada"}
	for _, platformID := range []int64{1, 2, 3} {
		lastSync := now.Add(-time.Duration(rand.Intn(300)) * time.Minute)
		s.db.Create(&MockSyncStatus{
			PlatformID:    platformID,
			PlatformName:  platformNames[platformID],
			SyncType:      "orders",
			Status:        "success",
			RecordsSynced: rand.Intn(500) + 50,
			LastSyncAt:    &lastSync,
			IsMockData:    true,
		})
		s.db.Create(&MockSyncStatus{
			PlatformID:    platformID,
			PlatformName:  platformNames[platformID],
			SyncType:      "products",
			Status:        "success",
			RecordsSynced: rand.Intn(100) + 10,
			LastSyncAt:    &lastSync,
			IsMockData:    true,
		})
		s.db.Create(&MockSyncStatus{
			PlatformID:    platformID,
			PlatformName:  platformNames[platformID],
			SyncType:      "fees",
			Status:        "success",
			RecordsSynced: rand.Intn(50) + 5,
			LastSyncAt:    &lastSync,
			IsMockData:    true,
		})
		// One failed sync for platform 1
		if platformID == 1 {
			s.db.Create(&MockSyncStatus{
				PlatformID:    platformID,
				PlatformName:  platformNames[platformID],
				SyncType:      "orders",
				Status:        "failed",
				RecordsSynced: 0,
				ErrorMessage:  "API rate limit exceeded",
				LastSyncAt:    &lastSync,
				IsMockData:    true,
			})
		}
	}

	return nil
}

// ListOrders returns paginated mock orders.
func (s *Service) ListOrders(page, size int) ([]MockOrder, int64, error) {
	var items []MockOrder
	var total int64
	q := s.db.Model(&MockOrder{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListSettlements returns paginated mock settlements.
func (s *Service) ListSettlements(page, size int) ([]MockSettlement, int64, error) {
	var items []MockSettlement
	var total int64
	q := s.db.Model(&MockSettlement{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetSyncStatuses returns the latest sync status grouped by platform.
func (s *Service) GetSyncStatuses() ([]MockSyncStatus, error) {
	var items []MockSyncStatus
	// Get latest sync per platform+type
	subQuery := s.db.Model(&MockSyncStatus{}).
		Select("DISTINCT ON (platform_id, sync_type) id").
		Order("platform_id, sync_type, id DESC")
	if err := s.db.Where("id IN (?)", subQuery).Order("platform_id, sync_type").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
