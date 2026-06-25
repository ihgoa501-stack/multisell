package productanalysis

// Service defines the public interface for product analysis operations.
// This interface exists so the A8 Agent (and future autonomous agents)
// can invoke analyses programmatically without depending on the HTTP layer.
type Service interface {
	// Analyze runs a full analysis on a sourced 1688 product.
	Analyze(in *AnalyzeInput, userID string) (*AnalysisResult, error)

	// GetAnalysis returns a single analysis by ID (scoped to user).
	GetAnalysis(id int64, userID string) (*ProductAnalysis, error)

	// ListAnalyses returns paginated analyses for a user.
	ListAnalyses(filter *ListFilter) ([]ProductAnalysis, int64, error)

	// RecordFeedback writes an immutable audit entry for a previous analysis.
	RecordFeedback(analysisID int64, in *FeedbackInput, userID string) error
}
