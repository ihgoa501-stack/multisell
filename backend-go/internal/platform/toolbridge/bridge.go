package toolbridge

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ToolDriver is the abstraction over different data collection methods.
type ToolDriver interface {
	// FetchPage collects structured data from a URL. Returns structured PageData.
	FetchPage(ctx context.Context, url string) (*PageData, error)
	// Health checks if this driver is available and returns latency.
	Health() (available bool, latency time.Duration, err error)
	// Category returns the tool category for security classification.
	Category() ToolCategory
	// Execute runs the tool with the given input and returns a structured result.
	Execute(input map[string]interface{}) (*ToolResult, error)
}

// ListPageDriver is an optional capability implemented by browser/search
// drivers that can discover multiple product opportunities from one page.
type ListPageDriver interface {
	FetchListPage(ctx context.Context, url string) (*ListPageData, error)
}

// Bridge defines the public API for fetching page data through the bridge.
type Bridge interface {
	FetchPage(ctx context.Context, url string) (*PageData, error)
}

// ToolBridgeOption is a functional option for ToolBridge construction.
type ToolBridgeOption func(*ToolBridge)

// ToolBridge routes fetch requests to the best available driver.
// It tries drivers in weight order (lower weight = higher priority) and
// falls through to the next driver when the preferred one fails.
type ToolBridge struct {
	mu      sync.RWMutex
	drivers []DriverEntry
	tools   map[string]ToolDriver
	timeout time.Duration
	logger  *zap.Logger
	tracker *ExternalCallTracker
}

// DriverEntry pairs a named driver with its preference weight.
type DriverEntry struct {
	Name   string
	Driver ToolDriver
	Weight int // lower = preferred first
}

// NewToolBridge creates a ToolBridge with the given drivers sorted by weight.
// If timeout is zero or negative, it defaults to 60 seconds.
func NewToolBridge(drivers []DriverEntry, timeout time.Duration, logger *zap.Logger, opts ...ToolBridgeOption) *ToolBridge {
	sorted := make([]DriverEntry, len(drivers))
	copy(sorted, drivers)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Weight < sorted[j].Weight
	})
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	b := &ToolBridge{
		drivers: sorted,
		timeout: timeout,
		logger:  logger,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// WithTracker sets the ExternalCallTracker for monitoring platform health.
func WithTracker(t *ExternalCallTracker) ToolBridgeOption {
	return func(b *ToolBridge) {
		b.tracker = t
	}
}

// FetchPage tries drivers in weight order. If the preferred driver fails,
// it falls through to the next. Returns ErrNoDrivers if no drivers are
// registered.
func (b *ToolBridge) FetchPage(ctx context.Context, url string) (*PageData, error) {
	b.mu.RLock()
	drivers := make([]DriverEntry, len(b.drivers))
	copy(drivers, b.drivers)
	b.mu.RUnlock()

	if len(drivers) == 0 {
		return nil, ErrNoDrivers
	}

	var lastErr error
	for _, entry := range drivers {
		// Check if the caller has cancelled the request.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		fetchCtx, cancel := context.WithTimeout(ctx, b.timeout)
		data, err := entry.Driver.FetchPage(fetchCtx, url)
		cancel()
		if err == nil {
			b.logger.Info("toolbridge: driver succeeded",
				zap.String("driver", entry.Name),
				zap.String("url", url))
			return data, nil
		}
		b.logger.Warn("toolbridge: driver failed, falling through",
			zap.String("driver", entry.Name),
			zap.Error(err))
		lastErr = err
	}
	return nil, lastErr
}

// FetchListPage routes list/search collection to the first capable driver and
// falls back when a driver fails. Detail-only drivers are skipped.
func (b *ToolBridge) FetchListPage(ctx context.Context, url string) (*ListPageData, error) {
	b.mu.RLock()
	drivers := make([]DriverEntry, len(b.drivers))
	copy(drivers, b.drivers)
	b.mu.RUnlock()

	if len(drivers) == 0 {
		return nil, ErrNoDrivers
	}
	var lastErr error
	capable := false
	for _, entry := range drivers {
		driver, ok := entry.Driver.(ListPageDriver)
		if !ok {
			continue
		}
		capable = true
		fetchCtx, cancel := context.WithTimeout(ctx, b.timeout)
		data, err := driver.FetchListPage(fetchCtx, url)
		cancel()
		if err == nil {
			return data, nil
		}
		lastErr = err
	}
	if !capable {
		return nil, ErrNoListDrivers
	}
	return nil, lastErr
}

// AddDriver registers an additional driver after construction. The bridge
// re-sorts all drivers by weight after adding.
func (b *ToolBridge) AddDriver(entry DriverEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.drivers = append(b.drivers, entry)
	sort.SliceStable(b.drivers, func(i, j int) bool {
		return b.drivers[i].Weight < b.drivers[j].Weight
	})
}

// Route is an alias for FetchPage provided to satisfy the sourcing.ToolBridge
// interface. It delegates to FetchPage with the same arguments.
func (b *ToolBridge) Route(ctx context.Context, url string) (*PageData, error) {
	return b.FetchPage(ctx, url)
}

// RegisterTool registers a tool driver by name for use with ExecuteTool.
// Tools are stored separately from page-fetching drivers.
func (b *ToolBridge) RegisterTool(name string, driver ToolDriver) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.tools == nil {
		b.tools = make(map[string]ToolDriver)
	}
	b.tools[name] = driver
}

// ExecuteTool runs a tool by name with the given input.
// For mutation tools, approvalID must be non-empty in production.
func (b *ToolBridge) ExecuteTool(name string, input map[string]interface{}, approvalID string) (*ToolResult, error) {
	b.mu.RLock()
	driver, ok := b.tools[name]
	b.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrToolNotRegistered, name)
	}
	if driver.Category() == ToolCategoryMutation && approvalID == "" {
		return nil, ErrApprovalRequired
	}
	return driver.Execute(input)
}

// ErrNoDrivers is returned when FetchPage is called with no registered drivers.
var ErrNoDrivers = errors.New("toolbridge: no drivers registered")

// ErrNoListDrivers is returned when registered drivers only support detail pages.
var ErrNoListDrivers = errors.New("toolbridge: no list-page drivers registered")

// ErrToolNotRegistered is returned when ExecuteTool is called for an unknown tool.
var ErrToolNotRegistered = errors.New("toolbridge: tool not registered")

// ErrApprovalRequired is returned when a mutation tool is executed without approval.
var ErrApprovalRequired = errors.New("toolbridge: mutation tool requires approval")
