package integrations

import (
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	// adapters is a thread-safe registry of platform code → adapter.
	// Platform code is the lower-case short name, e.g. "shopify", "lazada".
	adapters sync.Map

	// realAdaptersInitialised tracks whether InitRealAdapters has been called,
	// to prevent duplicate initialisation.
	realAdaptersInitialised bool
)

func init() {
	// Register built-in stubs. These are used when no DB is available (e.g.,
	// tests). Production code should call InitRealAdapters to replace stubs
	// with real API-driven adapters.
	RegisterAdapter("ozon", NewOzonAdapter())
	RegisterAdapter("shopee", NewShopeeAdapter())
}

// RegisterAdapter registers a PlatformAdapter implementation under the given
// platform code. The code is normalised to lower case.
func RegisterAdapter(platformCode string, adapter PlatformAdapter) {
	code := strings.ToLower(platformCode)
	if _, loaded := adapters.LoadOrStore(code, adapter); loaded {
		panic(fmt.Sprintf("platform adapter already registered: %s", code))
	}
}

// InitRealAdapters replaces the stub platform adapters (registered at import
// time via init()) with real API-driven implementations. Must be called once
// during server startup after the database connection is available.
//
// Currently replaces:
//   - "ozon"  → OzonRealAdapter (full Ozon Seller API)
//   - "shopee" → ShopeeAdapter (full Shopee Open API)
//
// Calling InitRealAdapters multiple times is safe — subsequent calls are no-ops.
func InitRealAdapters(db *gorm.DB, logger *zap.Logger) {
	if realAdaptersInitialised {
		return
	}
	realAdaptersInitialised = true

	// Use Store instead of RegisterAdapter because init() already registered
	// the stub under these codes.
	adapters.Store("ozon", NewOzonRealAdapter(db, logger))
	adapters.Store("shopee", NewShopeeAdapter())
}

// GetAdapter returns the adapter registered for the given platform code.
// The second return value is false when no adapter is registered.
func GetAdapter(platformCode string) (PlatformAdapter, bool) {
	v, ok := adapters.Load(strings.ToLower(platformCode))
	if !ok {
		return nil, false
	}
	return v.(PlatformAdapter), true
}

// ListAdapters returns a snapshot of all registered adapters keyed by
// platform code.
func ListAdapters() map[string]PlatformAdapter {
	out := make(map[string]PlatformAdapter)
	adapters.Range(func(k, v interface{}) bool {
		out[k.(string)] = v.(PlatformAdapter)
		return true
	})
	return out
}
