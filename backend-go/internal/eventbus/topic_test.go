package eventbus

import (
	"testing"
)

func TestMatchTopic(t *testing.T) {
	tests := []struct {
		pattern string
		topic   string
		want    bool
	}{
		{"order.*", "order.created", true},
		{"order.*", "order.refund", true},
		{"order.*", "order.created.something", false},
		{"order.**", "order.created", true},
		{"order.**", "order.created.something", true},
		{"order.created", "order.created", true},
		{"order.created", "order.refund", false},
		{"*", "anything.goes", true},
		{"inventory.*", "inventory.stock_low", true},
		{"inventory.*", "order.created", false},
		{"agent.decided.*", "agent.decided.a5", true},
		{"agent.decided.a5.*", "agent.decided.a5.stock_alert", true},
		{"agent.decided.a5.*", "agent.decided.a6.stock_alert", false},
	}

	for _, tt := range tests {
		got := matchTopic(tt.pattern, tt.topic)
		if got != tt.want {
			t.Errorf("matchTopic(%q, %q) = %v, want %v", tt.pattern, tt.topic, got, tt.want)
		}
	}
}
