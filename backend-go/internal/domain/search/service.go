package search

import (
	"strconv"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// Service provides search business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new search service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// Search performs a LIKE search across multiple tables.
// Each table contributes at most limit/6 results; total is capped at limit.
func (s *Service) Search(keyword string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	if keyword == "" || len(keyword) < 2 {
		return []SearchResult{}, nil
	}
	like := "%" + keyword + "%"
	perTable := limit / 6
	if perTable < 1 {
		perTable = 1
	}

	out := make([]SearchResult, 0, limit)

	// product: name
	type prodRow struct {
		ID   int64
		Name string
	}
	var prods []prodRow
	if err := s.db.Table("product").
		Select("id, name").
		Where("LOWER(name) LIKE LOWER(?)", like).
		Order("id DESC").
		Limit(perTable).
		Scan(&prods).Error; err != nil {
		return nil, err
	}
	for _, p := range prods {
		out = append(out, SearchResult{
			Type:  "product",
			ID:    p.ID,
			Title: p.Name,
			URL:   "/products/" + itoa(p.ID),
		})
		if len(out) >= limit {
			return out, nil
		}
	}

	// sku: code, spec_desc
	type skuRow struct {
		ID       int64
		Code     string
		SpecDesc string
	}
	var skus []skuRow
	if err := s.db.Table("sku").
		Select("id, code, spec_desc").
		Where("LOWER(code) LIKE LOWER(?) OR LOWER(spec_desc) LIKE LOWER(?)", like, like).
		Order("id DESC").
		Limit(perTable).
		Scan(&skus).Error; err != nil {
		return nil, err
	}
	for _, sk := range skus {
		title := sk.Code
		if title == "" {
			title = sk.SpecDesc
		}
		out = append(out, SearchResult{
			Type:     "sku",
			ID:       sk.ID,
			Title:    title,
			Subtitle: sk.SpecDesc,
			URL:      "/skus/" + itoa(sk.ID),
		})
		if len(out) >= limit {
			return out, nil
		}
	}

	// sales_order: order_no, recipient_name
	type ordRow struct {
		ID            int64
		OrderNo       string
		RecipientName string
	}
	var ords []ordRow
	if err := s.db.Table("sales_order").
		Select("id, order_no, recipient_name").
		Where("LOWER(order_no) LIKE LOWER(?) OR LOWER(recipient_name) LIKE LOWER(?)", like, like).
		Order("id DESC").
		Limit(perTable).
		Scan(&ords).Error; err != nil {
		return nil, err
	}
	for _, o := range ords {
		out = append(out, SearchResult{
			Type:     "order",
			ID:       o.ID,
			Title:    o.OrderNo,
			Subtitle: o.RecipientName,
			URL:      "/orders/" + itoa(o.ID),
		})
		if len(out) >= limit {
			return out, nil
		}
	}

	// after_sales_order: reason
	type asRow struct {
		ID     int64
		Reason string
	}
	var ass []asRow
	if err := s.db.Table("after_sales_order").
		Select("id, reason").
		Where("LOWER(reason) LIKE LOWER(?)", like).
		Order("id DESC").
		Limit(perTable).
		Scan(&ass).Error; err != nil {
		return nil, err
	}
	for _, a := range ass {
		out = append(out, SearchResult{
			Type:  "aftersales",
			ID:    a.ID,
			Title: a.Reason,
			URL:   "/aftersales/" + itoa(a.ID),
		})
		if len(out) >= limit {
			return out, nil
		}
	}

	// exception_item: title
	type exRow struct {
		ID    int64
		Title string
	}
	var exs []exRow
	if err := s.db.Table("exception_item").
		Select("id, title").
		Where("LOWER(title) LIKE LOWER(?)", like).
		Order("id DESC").
		Limit(perTable).
		Scan(&exs).Error; err != nil {
		return nil, err
	}
	for _, e := range exs {
		out = append(out, SearchResult{
			Type:  "exception",
			ID:    e.ID,
			Title: e.Title,
			URL:   "/exceptions/" + itoa(e.ID),
		})
		if len(out) >= limit {
			return out, nil
		}
	}

	// settlement: settlement_no
	type stRow struct {
		ID           int64
		SettlementNo string
	}
	var sts []stRow
	if err := s.db.Table("settlement").
		Select("id, settlement_no").
		Where("LOWER(settlement_no) LIKE LOWER(?)", like).
		Order("id DESC").
		Limit(perTable).
		Scan(&sts).Error; err != nil {
		return nil, err
	}
	for _, st := range sts {
		out = append(out, SearchResult{
			Type:  "settlement",
			ID:    st.ID,
			Title: st.SettlementNo,
			URL:   "/settlements/" + itoa(st.ID),
		})
		if len(out) >= limit {
			return out, nil
		}
	}

	return out, nil
}

// RecordRecentSearch stores a search query for a user.
func (s *Service) RecordRecentSearch(userID, keyword string) {
	if err := s.db.Exec("INSERT INTO recent_search (user_id, query, searched_at) VALUES (?, ?, ?)",
		userID, keyword, time.Now().Format(time.RFC3339Nano)).Error; err != nil {
		s.logger.Warn("failed to record recent search", zap.Error(err))
	}
}

// Recent returns the recent searches for a user, ordered by time descending.
func (s *Service) Recent(userID string) []RecentSearch {
	var results []RecentSearch
	if err := s.db.Table("recent_search").
		Select("query, searched_at AS timestamp").
		Where("user_id = ?", userID).
		Order("searched_at DESC").
		Limit(20).
		Scan(&results).Error; err != nil {
		s.logger.Warn("failed to query recent searches", zap.Error(err))
		return []RecentSearch{}
	}
	if results == nil {
		return []RecentSearch{}
	}
	return results
}
