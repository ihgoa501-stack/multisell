package toolbridge

import (
	"context"
	"errors"
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
}

// Bridge defines the public API for fetching page data through the bridge.
type Bridge interface {
	FetchPage(ctx context.Context, url string) (*PageData, error)
}

// ToolBridge routes fetch requests to the best available driver.
// It tries drivers in weight order (lower weight = higher priority) and
// falls through to the next driver when the preferred one fails.
type ToolBridge struct {
	mu      sync.RWMutex
	drivers []DriverEntry
	timeout time.Duration
	logger  *zap.Logger
}

// DriverEntry pairs a named driver with its preference weight.
type DriverEntry struct {
	Name   string
	Driver ToolDriver
	Weight int // lower = preferred first
}

// NewToolBridge creates a ToolBridge with the given drivers sorted by weight.
// If timeout is zero or negative, it defaults to 60 seconds.
func NewToolBridge(drivers []DriverEntry, timeout time.Duration, logger *zap.Logger) *ToolBridge {
	sorted := make([]DriverEntry, len(drivers))
	copy(sorted, drivers)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Weight < sorted[j].Weight
	})
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &ToolBridge{
		drivers: sorted,
		timeout: timeout,
		logger:  logger,
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

// ExecuteToolCall executes a ToolCall through the bridge, enforcing approval
// for production mutation calls. The actual mutation execution is a stub
// (returns success when validation passes); wire real handlers here when
// mutation tools are implemented.
func (b *ToolBridge) ExecuteToolCall(ctx context.Context, tc ToolCall) *ToolResult {
	if err := tc.Validate(); err != nil {
		return &ToolResult{
			Success:      false,
			ErrorMessage: err.Error(),
			Mode:         tc.Mode,
		}
	}
	// ponytail: mutation execution is not yet wired; this stub returns
	// success for any validated call. Wire real domain handlers when
	// the first mutation tool is implemented.
	b.logger.Info("toolbridge: execute tool call",
		zap.String("tool", tc.ToolName),
		zap.String("category", tc.Category.String()),
		zap.String("mode", tc.Mode.String()),
	)
	return &ToolResult{
		Success: true,
		Mode:    tc.Mode,
		Data:    tc.Input,
	}
}

// ErrNoDrivers is returned when FetchPage is called with no registered drivers.
var ErrNoDrivers = errors.New("toolbridge: no drivers registered")
