package metabolism

// ScoringAdapter fetches scorable events from a source table and marks
// them as excreted after processing. Each adapter handles one source type
// (event_outbox, execution_logs, agent_decisions, etc.).
type ScoringAdapter interface {
	// ScorableEvents returns events whose status matches the given status.
	// An empty status returns all scorable events.
	ScorableEvents(status string) ([]ScorableEvent, error)

	// MarkExcreted records the excretion decision for the event.
	MarkExcreted(eventID int64, reason string) error
}

// SemanticScorer provides contextual scoring via an LLM or heuristic.
type SemanticScorer interface {
	// Score returns a value between 0 and 1 indicating semantic importance.
	Score(ev ScorableEvent) (float64, error)
}
