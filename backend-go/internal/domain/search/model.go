package search

// SearchResult is one hit in global search.
type SearchResult struct {
	Type     string `json:"type"`
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	URL      string `json:"url"`
}

// RecentSearch is one entry in a user's recent search history (placeholder).
type RecentSearch struct {
	Query     string `json:"query"`
	Timestamp string `json:"timestamp,omitempty"`
}
