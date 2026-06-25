package toolregistry

import "fmt"

// ErrPermissionDenied is returned when the caller lacks a required permission
// for the requested tool.
type ErrPermissionDenied struct {
	// Name is the tool name.
	Name string
	// Permission is the specific permission that was missing.
	Permission string
}

func (e ErrPermissionDenied) Error() string {
	return fmt.Sprintf("tool %s: permission denied, missing required permission %q", e.Name, e.Permission)
}

// ErrRateLimited is returned when a tool call exceeds the configured rate limit.
type ErrRateLimited struct {
	// Name is the tool name.
	Name string
}

func (e ErrRateLimited) Error() string {
	return fmt.Sprintf("tool %s: rate limit exceeded", e.Name)
}

// ErrCircuitOpen is returned when the circuit breaker for a tool is open
// and requests are being rejected.
type ErrCircuitOpen struct {
	// Name is the tool name.
	Name string
}

func (e ErrCircuitOpen) Error() string {
	return fmt.Sprintf("tool %s: circuit breaker is open, request rejected", e.Name)
}
