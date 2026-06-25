package ipc

import "time"

// MsgType represents the type of inter-agent message.
type MsgType string

const (
	MsgTypeRequest   MsgType = "request"
	MsgTypeResponse  MsgType = "response"
	MsgTypeNotify    MsgType = "notify"
	MsgTypeDelegate  MsgType = "delegate"
	MsgTypeGather    MsgType = "gather"
	MsgTypeConsensus MsgType = "consensus"
)

// Message is the envelope for inter-agent communication.
type Message struct {
	ID        string                 `json:"id"`
	Type      MsgType                `json:"type"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	SessionID string                 `json:"session_id"`
	Payload   map[string]interface{} `json:"payload"`
	Priority  int                    `json:"priority"`
	TimeoutMs int                    `json:"timeout_ms"`
	CreatedAt time.Time              `json:"created_at"`
}

// TaskDef defines a task that can be delegated to an agent.
type TaskDef struct {
	ID            string                 `json:"id"`
	Description   string                 `json:"description"`
	Payload       map[string]interface{} `json:"payload"`
	DecisionPoint string                 `json:"decision_point"`
}
