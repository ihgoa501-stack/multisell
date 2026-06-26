package metabolism

import (
	"time"
)

// DigestRecord tracks which agent has consumed which event.
type DigestRecord struct {
	EventID   int64
	Source    string
	AgentID   string
	Status    string // "digesting" | "digested" | "failed"
	CreatedAt time.Time
}

// DigestRequest is the payload an agent sends when declaring digestion.
type DigestRequest struct {
	EventID int64  `json:"event_id"`
	AgentID string `json:"agent_id"`
	Source  string `json:"source"`
}

// DigestTopic returns the event bus topic for an agent's digest declaration.
func DigestTopic(agentID string) string {
	return "metabolism.digested." + agentID
}
