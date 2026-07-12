package xiaoq

import (
	"errors"
	"fmt"
)

const (
	AgentID          = "xiao_q"
	TargetDemandCase = "demand_case"
	TargetExperiment = "experiment"
	TruthMock        = "mock"
	TruthInferred    = "inferred"
	MaxMessageRunes  = 2000
)

var (
	ErrInvalidInput          = errors.New("invalid xiao-q input")
	ErrTraceNotFound         = errors.New("xiao-q trace not found")
	ErrCapabilityUnavailable = errors.New("required xiao-q capability unavailable")
)

type MessageInput struct {
	Message      string `json:"message" binding:"required"`
	TargetType   string `json:"target_type,omitempty"`
	DemandCaseID int64  `json:"demand_case_id,omitempty"`
	ExperimentID string `json:"experiment_id,omitempty"`
}

type MessageResponse struct {
	TraceID      string         `json:"trace_id"`
	AgentID      string         `json:"agent_id"`
	Mode         string         `json:"mode"`
	TargetType   string         `json:"target_type"`
	DemandCaseID int64          `json:"demand_case_id"`
	ExperimentID string         `json:"experiment_id,omitempty"`
	Answer       string         `json:"answer"`
	TruthStatus  string         `json:"truth_status"`
	Trusted      bool           `json:"trusted"`
	Provider     string         `json:"provider"`
	Model        string         `json:"model"`
	TokensIn     int            `json:"tokens_in"`
	TokensOut    int            `json:"tokens_out"`
	LatencyMs    int            `json:"latency_ms"`
	Evidence     []EvidenceItem `json:"evidence"`
	Unknowns     []string       `json:"unknowns"`
	Links        []ResponseLink `json:"links"`
	Provenance   Provenance     `json:"provenance"`
}

type EvidenceItem struct {
	ID             int64  `json:"id"`
	Title          string `json:"title"`
	TruthStatus    string `json:"truth_status"`
	SourceURL      string `json:"source_url,omitempty"`
	ObservedAt     string `json:"observed_at,omitempty"`
	Summary        string `json:"summary"`
	RunID          string `json:"run_id"`
	SnapshotID     int64  `json:"snapshot_id"`
	SnapshotSHA256 string `json:"snapshot_sha256,omitempty"`
	VerifiedBy     int64  `json:"verified_by,omitempty"`
	VerifiedAt     string `json:"verified_at,omitempty"`
}

type ResponseLink struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

type Provenance struct {
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	TokensIn  int    `json:"tokens_in"`
	TokensOut int    `json:"tokens_out"`
	LatencyMs int    `json:"latency_ms"`
}

type Identity struct {
	AgentID     string `json:"agent_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Mode        string `json:"mode"`
}

type RunError struct {
	TraceID string
	Err     error
}

func (e *RunError) Error() string { return fmt.Sprintf("xiao-q run %s failed: %v", e.TraceID, e.Err) }
func (e *RunError) Unwrap() error { return e.Err }
