package integrations

import (
	"fmt"
	"strings"
	"sync"
)

var (
	// adapters is a thread-safe registry of platform code → adapter.
	// Platform code is the lower-case short name, e.g. "shopify", "lazada".
	adapters sync.Map
)

func init() {
	// Register built-in adapters.
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
