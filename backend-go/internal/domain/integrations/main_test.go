package integrations

import "os"

// init ensures PLATFORM_TOKEN_ENCRYPTION_KEY is set for tests.
// ponytail: real env var replaces the hardcoded fallback that was deleted from crypto.go.
func init() {
	if os.Getenv("PLATFORM_TOKEN_ENCRYPTION_KEY") == "" {
		os.Setenv("PLATFORM_TOKEN_ENCRYPTION_KEY", "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=")
	}
}
