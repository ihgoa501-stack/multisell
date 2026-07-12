package experiment

import (
	"testing"
	"time"
)

func TestClosureValidationSharedPredicates(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name, status, source string
		imported             *time.Time
		want                 bool
	}{
		{"platform reconciled", "reconciled", "platform_import", &now, true},
		{"api closed", "closed", "api_sync", &now, true},
		{"manual source", "reconciled", "manual", &now, false},
		{"not imported", "reconciled", "platform_import", nil, false},
		{"pending", "pending", "api_sync", &now, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTrustedSettlement(tc.status, tc.source, tc.imported); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}

	for _, tc := range []struct {
		name                           string
		items, unmatched, orderMatches int64
		want                           bool
	}{
		{"all matched", 2, 0, 1, true}, {"empty", 0, 0, 0, false}, {"one unmatched", 2, 1, 1, false}, {"wrong order", 2, 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isFullyReconciledSettlement(tc.items, tc.unmatched, tc.orderMatches); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}

	for _, tc := range []struct {
		name             string
		actual, expected int64
		status, missing  string
		want             bool
	}{
		{"final", 10, 10, "final", "", true}, {"wrong order", 11, 10, "final", "", false}, {"provisional", 10, 10, "provisional", "", false}, {"missing costs", 10, 10, "final", "shipping_fee", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isFinalProfitForOrder(tc.actual, tc.expected, tc.status, tc.missing); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
