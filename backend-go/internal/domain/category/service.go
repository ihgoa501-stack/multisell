package category

import (
	"context"
	"strings"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides category business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new category service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns a paginated list of categories with optional search.
func (s *Service) List(ctx context.Context, page, size int, search string) ([]Category, int64, error) {
	var items []Category
	var total int64

	q := s.db.WithContext(ctx).Model(&Category{})
	if search != "" {
		q = q.Where("name ILIKE ?", "%"+search+"%")
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	offset := (page - 1) * size
	if err := q.Order("sort_order ASC, id DESC").Offset(offset).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetTree builds a hierarchical tree of all categories.
func (s *Service) GetTree(ctx context.Context) ([]TreeNode, error) {
	var items []Category
	if err := s.db.WithContext(ctx).Order("sort_order ASC, id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return buildTree(items), nil
}

// TreeNode represents a category with children.
type TreeNode struct {
	Category
	Children []TreeNode `json:"children"`
}

func buildTree(items []Category) []TreeNode {
	nodeMap := make(map[int64]*TreeNode)
	var orderedRoots []int64

	for _, item := range items {
		node := &TreeNode{Category: item}
		nodeMap[item.ID] = node
	}

	for _, item := range items {
		node := nodeMap[item.ID]
		if item.ParentID == 0 {
			orderedRoots = append(orderedRoots, item.ID)
		} else if parent, ok := nodeMap[item.ParentID]; ok {
			parent.Children = append(parent.Children, *node)
		} else {
			orderedRoots = append(orderedRoots, item.ID)
		}
	}

	// Refresh root nodes to pick up children that were appended by value
	// (appending a struct copies, so we need to re-read from the map)
	var roots []TreeNode
	for _, id := range orderedRoots {
		roots = append(roots, *nodeMap[id])
	}
	return roots
}

// GetByID retrieves a single category by ID.
func (s *Service) GetByID(ctx context.Context, id int64) (*Category, error) {
	var c Category
	if err := s.db.WithContext(ctx).First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// Create inserts a new category.
func (s *Service) Create(ctx context.Context, c *Category) error {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Create(c).Error
}

// Update saves changes to an existing category.
func (s *Service) Update(ctx context.Context, c *Category) error {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return gorm.ErrInvalidData
	}
	return s.db.WithContext(ctx).Save(c).Error
}

// Delete removes a category by ID (hard delete — no soft-delete column).
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&Category{}, id).Error
}

// Tree is a public alias for GetTree (used by handler).
func (s *Service) Tree(ctx context.Context) ([]TreeNode, error) {
	return s.GetTree(ctx)
}

// ensure common import is used
var _ = common.Pagination{}
