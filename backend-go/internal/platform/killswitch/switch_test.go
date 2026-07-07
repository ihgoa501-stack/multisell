package killswitch

import (
	"testing"
)

func TestKillSwitch(t *testing.T) {
	// Start fresh
	Deactivate()

	if IsActive() {
		t.Fatal("expected inactive after Deactivate")
	}
	if Reason() != "" {
		t.Fatalf("expected empty reason, got %q", Reason())
	}

	Activate("emergency: security incident")

	if !IsActive() {
		t.Fatal("expected active after Activate")
	}
	if Reason() != "emergency: security incident" {
		t.Fatalf("expected 'emergency: security incident', got %q", Reason())
	}

	Deactivate()

	if IsActive() {
		t.Fatal("expected inactive after second Deactivate")
	}
	if Reason() != "" {
		t.Fatalf("expected empty reason after deactivation, got %q", Reason())
	}
}

func TestKillSwitchActivateThenDeactivate(t *testing.T) {
	Deactivate()
	Activate("testing")
	if !IsActive() {
		t.Fatal("expected active")
	}
	Activate("updated reason")
	// Last reason wins
	if Reason() != "updated reason" {
		t.Fatalf("expected 'updated reason', got %q", Reason())
	}
	Deactivate()
}
