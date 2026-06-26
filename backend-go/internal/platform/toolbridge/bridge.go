// Package toolbridge provides an abstraction layer for fetching structured product
// data from external sourcing platforms (e.g. 1688). It manages multiple drivers
// with automatic fallback and dead-letter-queue routing.
//
// Architecture:
//
//	A8 Agent calls Route(url) → ToolBridge:
//	  1. Try PluginDriver (primary — WebSocket to Chrome Extension, user login)
//	  2. Fall back to PlaywrightDriver (server-side headless browser)
//	  3. Both failed → Dead Letter Queue
package toolbridge

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Error sentinel
// ---------------------------------------------------------------------------

var (
	// ErrAllDriversFailed is returned when no driver could fetch the page.
	ErrAllDriversFailed = errors.New("all tool drivers failed")
	// ErrNoDriversRegistered is returned when no drivers have been registered.
	ErrNoDriversRegistered = errors.New("no tool drivers registered")
)

// ---------------------------------------------------------------------------
// Data model
// ---------------------------------------------------------------------------

// PageData is the unified return structure for a fetched product page.
// All drivers (plugin, playwright, api1688) return this same shape.
type PageData struct {
	// Meta
	SourceURL   string    `json:"source_url"`
	CollectedAt time.Time `json:"collected_at"`
	Driver      string    `json:"driver"` // "plugin", "playwright", "api1688"

	// Core fields
	Title string  `json:"title"`
	Price float64 `json:"price_1688"`
	// PriceMin/Max are set when the product has spec variants with differing prices.
	PriceMin *float64 `json:"price_min,omitempty"`
	PriceMax *float64 `json:"price_max,omitempty"`
	Currency string   `json:"currency"`      // default "CNY"
	MOQ      int      `json:"min_order_qty"` // minimum order quantity

	// Images
	Images []string `json:"images"` // main image URLs

	// Spec variants
	SpecVariants []SpecVariant `json:"spec_variants,omitempty"`

	// Supplier
	SupplierName  string `json:"supplier_name"`
	SupplierID    string `json:"supplier_id_1688"`
	SupplierScore *int   `json:"supplier_score,omitempty"` // 0-100

	// Description & attributes
	Description string            `json:"description,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`

	// Logistics (optional — populated when available)
	PackageWeight *float64 `json:"package_weight_kg,omitempty"`
	PackageLength *float64 `json:"package_length_cm,omitempty"`
	PackageWidth  *float64 `json:"package_width_cm,omitempty"`
	PackageHeight *float64 `json:"package_height_cm,omitempty"`
	FreightCNY    *float64 `json:"freight_cny,omitempty"`
}

// SpecVariant represents one variant in a multi-spec product (e.g. color, size).
type SpecVariant struct {
	Spec     string  `json:"spec"`      // e.g. "color:red;size:L"
	Price    float64 `json:"price"`
	Stock    int     `json:"stock"`
	ImageURL string  `json:"image_url,omitempty"`
}

// ---------------------------------------------------------------------------
// Driver interface
// ---------------------------------------------------------------------------

// ToolDriver fetches structured product data from a sourcing platform.
type ToolDriver interface {
	// FetchPage fetches and parses the product page at the given URL, returning
	// a structured PageData. The context is used for cancellation and deadlines.
	FetchPage(ctx context.Context, url string) (*PageData, error)

	// Health returns a human-readable status string ("online"/"offline") and the
	// round-trip latency of the last health check. Returns 0 latency if never
	// checked.
	Health() (status string, latency time.Duration)

	// Name returns a unique identifier for this driver (e.g. "plugin", "playwright").
	Name() string
}

// ---------------------------------------------------------------------------
// Dead Letter Queue entry
// ---------------------------------------------------------------------------

// DLQEntry records a failed fetch attempt that exhausted all available drivers.
type DLQEntry struct {
	SourceURL   string
	AttemptedAt time.Time
	PrimaryErr  string
	FallbackErr string
}

// ---------------------------------------------------------------------------
// ToolBridge
// ---------------------------------------------------------------------------

// ToolBridge manages multiple ToolDrivers with automatic fallback routing.
//
// The bridge tries the primary driver first. If it fails (offline, timeout, parse
// error), the fallback driver is used. If both fail, an entry is written to the
// in-memory dead letter queue and ErrAllDriversFailed is returned.
type ToolBridge struct {
	mu          sync.RWMutex
	primary     ToolDriver
	fallback    ToolDriver
	deadLetterQ []DLQEntry
	maxDLQSize  int
	logger      *zap.Logger
}

// New creates a new ToolBridge.
//
// primary is the preferred driver (e.g. PluginDriver).
// fallback is used when the primary driver is unavailable or fails.
// maxDLQSize limits the number of dead-letter entries retained in memory.
func New(primary, fallback ToolDriver, logger *zap.Logger, maxDLQSize int) *ToolBridge {
	return &ToolBridge{
		primary:     primary,
		fallback:    fallback,
		maxDLQSize:  maxDLQSize,
		logger:      logger,
		deadLetterQ: make([]DLQEntry, 0, maxDLQSize),
	}
}

// Route fetches product data from the given URL.
//
// Routing strategy:
//  1. Try the primary driver.
//  2. If primary fails, try the fallback driver.
//  3. If both fail, write a DLQ entry and return ErrAllDriversFailed.
func (b *ToolBridge) Route(ctx context.Context, url string) (*PageData, error) {
	b.mu.RLock()
	primary := b.primary
	fallback := b.fallback
	b.mu.RUnlock()

	if primary == nil && fallback == nil {
		return nil, ErrNoDriversRegistered
	}

	var primaryErr, fallbackErr string

	// Try primary driver.
	if primary != nil {
		b.logger.Debug("toolbridge: trying primary driver",
			zap.String("driver", primary.Name()),
			zap.String("url", url))
		result, err := primary.FetchPage(ctx, url)
		if err == nil {
			b.logger.Info("toolbridge: primary driver succeeded",
				zap.String("driver", primary.Name()))
			return result, nil
		}
		primaryErr = err.Error()
		b.logger.Warn("toolbridge: primary driver failed",
			zap.String("driver", primary.Name()),
			zap.Error(err))
	}

	// Try fallback driver.
	if fallback != nil {
		b.logger.Debug("toolbridge: trying fallback driver",
			zap.String("driver", fallback.Name()),
			zap.String("url", url))
		result, err := fallback.FetchPage(ctx, url)
		if err == nil {
			b.logger.Info("toolbridge: fallback driver succeeded",
				zap.String("driver", fallback.Name()))
			return result, nil
		}
		fallbackErr = err.Error()
		b.logger.Warn("toolbridge: fallback driver failed",
			zap.String("driver", fallback.Name()),
			zap.Error(err))
	}

	// Both failed — record in dead letter queue.
	entry := DLQEntry{
		SourceURL:   url,
		AttemptedAt: time.Now(),
		PrimaryErr:  primaryErr,
		FallbackErr: fallbackErr,
	}

	b.mu.Lock()
	b.deadLetterQ = append(b.deadLetterQ, entry)
	if len(b.deadLetterQ) > b.maxDLQSize {
		b.deadLetterQ = b.deadLetterQ[len(b.deadLetterQ)-b.maxDLQSize:]
	}
	b.mu.Unlock()

	b.logger.Error("toolbridge: both drivers failed, sent to DLQ",
		zap.String("url", url),
		zap.String("primary_err", primaryErr),
		zap.String("fallback_err", fallbackErr))

	return nil, fmt.Errorf("%w: primary=%v fallback=%v",
		ErrAllDriversFailed, primaryErr, fallbackErr)
}

// Health returns the aggregated health status of all registered drivers.
// The result map is keyed by driver name.
func (b *ToolBridge) Health() map[string]DriverHealth {
	b.mu.RLock()
	primary := b.primary
	fallback := b.fallback
	b.mu.RUnlock()

	result := make(map[string]DriverHealth)

	if primary != nil {
		status, latency := primary.Health()
		result[primary.Name()] = DriverHealth{Status: status, Latency: latency}
	}
	if fallback != nil {
		status, latency := fallback.Health()
		result[fallback.Name()] = DriverHealth{Status: status, Latency: latency}
	}

	return result
}

// DriverHealth represents the health snapshot of a single driver.
type DriverHealth struct {
	Status  string        `json:"status"`
	Latency time.Duration `json:"latency_ms"`
}

// SetPrimary replaces the primary driver at runtime. Thread-safe.
func (b *ToolBridge) SetPrimary(driver ToolDriver) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.primary = driver
}

// SetFallback replaces the fallback driver at runtime. Thread-safe.
func (b *ToolBridge) SetFallback(driver ToolDriver) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.fallback = driver
}

// DLQEntries returns a copy of the current dead letter queue entries.
func (b *ToolBridge) DLQEntries() []DLQEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()
	entries := make([]DLQEntry, len(b.deadLetterQ))
	copy(entries, b.deadLetterQ)
	return entries
}
