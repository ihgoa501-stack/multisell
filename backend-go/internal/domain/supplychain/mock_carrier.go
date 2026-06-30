package supplychain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"time"
)

// This file implements a MOCK carrier API client for the supply chain tracking
// domain.
//
// Issue #38 requires integrating carrier APIs to fetch real shipment tracks
// (pickup → outbound → last-mile → delivery). Per the issue constraints, this
// group MUST NOT call any real carrier API; all tracking data is synthetic.
//
// The MockCarrierClient generates a deterministic, cross-border shipment
// lifecycle for a given tracking number. The TrackingService.SyncFromCarrier
// method polls the mock client, deduplicates against the existing
// status_history, appends new events, and updates the current status.
//
// Replace MockCarrierClient with a real adapter (e.g. 17track, Yanwen,
// YunExpress) before production rollout.

// ErrMockCarrierTrackingNotFound is returned by the mock carrier when the
// tracking number is empty or malformed.
var ErrMockCarrierTrackingNotFound = errors.New("mock carrier: tracking number not found")

// MockCarrierClient is a synthetic carrier API client. It produces deterministic
// tracking events for a given tracking number without any network calls.
//
// The lifecycle is determined by a stable hash of the tracking number so the
// same tracking number always yields the same progression. A small fraction of
// tracking numbers (those prefixed with "EXC-") return an "exception" status to
// exercise the escalation path.
type MockCarrierClient struct {
	// now is overridable for deterministic tests.
	now func() time.Time
}

// NewMockCarrierClient creates a new MockCarrierClient using the system clock.
func NewMockCarrierClient() *MockCarrierClient {
	return &MockCarrierClient{now: time.Now}
}

// FetchTrackingEvents returns the synthetic shipment lifecycle for the given
// tracking number. The events cover the cross-border journey:
//
//	picked_up → outbound → transit → customs → last_mile → delivered
//
// or, for exception-tagged tracking numbers:
//
//	picked_up → outbound → exception
//
// The returned events are chronological with monotonically increasing
// timestamps rooted at a deterministic base time derived from the tracking
// number hash. This ensures the same tracking number always produces the same
// event sequence, which keeps SyncFromCarrier idempotent.
func (c *MockCarrierClient) FetchTrackingEvents(trackingNo string) ([]TrackingEvent, error) {
	if trackingNo == "" {
		return nil, ErrMockCarrierTrackingNotFound
	}

	base := c.baseTime(trackingNo)
	steps := c.lifecycleFor(trackingNo)

	events := make([]TrackingEvent, 0, len(steps))
	offset := time.Duration(0)
	for _, s := range steps {
		events = append(events, TrackingEvent{
			Status:    s.status,
			Timestamp: base.Add(offset),
			Location:  s.location,
			Message:   s.message,
		})
		// Each subsequent event happens after the current step's duration
		// has elapsed, producing strictly increasing timestamps.
		offset += s.step
	}
	return events, nil
}

// lifecycleFor picks the synthetic lifecycle for a tracking number. Tracking
// numbers prefixed with "EXC-" return an exception path so escalation code
// paths can be exercised end-to-end.
func (c *MockCarrierClient) lifecycleFor(trackingNo string) []mockLifecycleStep {
	if len(trackingNo) >= 4 && trackingNo[:4] == "EXC-" {
		return exceptionLifecycle()
	}
	return happyLifecycle()
}

type mockLifecycleStep struct {
	status   string
	step     time.Duration
	location string
	message  string
}

// happyLifecycle returns the standard cross-border shipment progression.
// Step durations are synthetic (not real-time); they only space events apart
// so deduplication during SyncFromCarrier is well-defined.
func happyLifecycle() []mockLifecycleStep {
	return []mockLifecycleStep{
		{status: "picked_up", step: 1 * time.Hour, location: "Shenzhen Origin Hub", message: "Package picked up from supplier"},
		{status: "outbound", step: 6 * time.Hour, location: "Shenzhen Customs Export", message: "Export customs clearance completed"},
		{status: "transit", step: 24 * time.Hour, location: "International Transit", message: "In transit to destination country"},
		{status: "customs", step: 24 * time.Hour, location: "Destination Customs", message: "Import customs clearance in progress"},
		{status: "last_mile", step: 12 * time.Hour, location: "Local Delivery Hub", message: "Handed to last-mile carrier"},
		{status: "delivered", step: 8 * time.Hour, location: "Customer Address", message: "Package delivered to customer"},
	}
}

// exceptionLifecycle returns a progression that ends in an exception status,
// exercising the escalation path (Level 2 manual review).
func exceptionLifecycle() []mockLifecycleStep {
	return []mockLifecycleStep{
		{status: "picked_up", step: 1 * time.Hour, location: "Shenzhen Origin Hub", message: "Package picked up from supplier"},
		{status: "outbound", step: 6 * time.Hour, location: "Shenzhen Customs Export", message: "Export customs clearance completed"},
		{status: "exception", step: 24 * time.Hour, location: "International Transit", message: "Package held at transit hub — address undeliverable"},
	}
}

// baseTime derives a deterministic base timestamp from the tracking number so
// repeated SyncFromCarrier calls for the same tracking number produce stable
// event timestamps (idempotent appends).
func (c *MockCarrierClient) baseTime(trackingNo string) time.Time {
	h := fnv.New32a()
	_, _ = h.Write([]byte(trackingNo))
	seed := int64(h.Sum32())
	// Anchor to a fixed recent epoch so tests are deterministic across runs
	// even when c.now is the system clock.
	epoch := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return epoch.Add(time.Duration(seed%720) * time.Hour)
}

// SyncFromCarrier polls the mock carrier client for the latest tracking events
// and appends any new events to the tracking record's status_history. The
// tracking record's status is advanced to the latest event's status.
//
// Deduplication is based on (Status, Timestamp) pairs: events already present
// in status_history are not re-appended. This makes the operation safe to call
// repeatedly (idempotent).
//
// The estimated_delivery field is set to the delivered event's timestamp when
// the lifecycle reaches "delivered"; actual_delivery is also set to that
// timestamp.
func (s *TrackingService) SyncFromCarrier(ctx context.Context, id string, client *MockCarrierClient) (*SupplyChainTracking, error) {
	if client == nil {
		return nil, fmt.Errorf("tracking: SyncFromCarrier requires a non-nil carrier client")
	}

	var t SupplyChainTracking
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&t).Error; err != nil {
		return nil, err
	}

	events, err := client.FetchTrackingEvents(t.TrackingNo)
	if err != nil {
		return nil, err
	}

	// Load existing history.
	var history []TrackingEvent
	if t.StatusHistory != nil && len(*t.StatusHistory) > 0 {
		if err := json.Unmarshal(*t.StatusHistory, &history); err != nil {
			history = []TrackingEvent{}
		}
	}

	// Build a deduplication key set from existing history.
	seen := make(map[string]bool, len(history))
	for _, h := range history {
		seen[dedupKey(h)] = true
	}

	// Append only events not already present. Sort incoming events by
	// timestamp to keep history ordered (the mock already returns them in
	// order, but sorting defends against future carrier adapters).
	sorted := make([]TrackingEvent, len(events))
	copy(sorted, events)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	appended := 0
	for _, e := range sorted {
		if seen[dedupKey(e)] {
			continue
		}
		history = append(history, e)
		seen[dedupKey(e)] = true
		appended++
	}

	// If nothing changed, return the record as-is.
	if appended == 0 && len(history) > 0 {
		// Status may already be current; ensure status reflects the latest
		// history entry.
		latest := history[len(history)-1]
		if latest.Status != t.Status {
			_ = s.db.WithContext(ctx).Model(&t).Where("id = ?", id).Update("status", latest.Status).Error
			t.Status = latest.Status
		}
		return &t, nil
	}

	raw, err := json.Marshal(history)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"status_history": (*json.RawMessage)(&raw),
	}
	if len(history) > 0 {
		latest := history[len(history)-1]
		updates["status"] = latest.Status
		if latest.Status == "delivered" {
			updates["actual_delivery"] = latest.Timestamp
			updates["estimated_delivery"] = latest.Timestamp
		}
	}

	if err := s.db.WithContext(ctx).Model(&t).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Reload the updated record.
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// dedupKey returns a stable identifier for a tracking event used to detect
// duplicates during SyncFromCarrier.
func dedupKey(e TrackingEvent) string {
	return fmt.Sprintf("%s|%d", e.Status, e.Timestamp.UnixNano())
}
