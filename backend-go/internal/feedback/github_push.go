package feedback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GitHubPushConfig holds the configuration for pushing to a GitHub repo.
type GitHubPushConfig struct {
	Repo   string   `json:"repo"`   // "owner/repo"
	Token  string   `json:"token"`  // GitHub personal access token
	Labels []string `json:"labels"` // e.g., ["feedback"]
}

// GitHubIssueRequest is the GitHub Issues API request body.
type GitHubIssueRequest struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Labels    []string `json:"labels"`
}

// GitHubIssueResponse is the GitHub Issues API response (partial).
type GitHubIssueResponse struct {
	HTMLURL string `json:"html_url"`
	Number  int    `json:"number"`
}

// PushToGitHub creates a GitHub Issue for an accepted feedback submission.
// Returns the issue URL and issue number, or an error.
func PushToGitHub(cfg *GitHubPushConfig, title, body string) (string, int, error) {
	if cfg == nil || cfg.Repo == "" || cfg.Token == "" {
		return "", 0, fmt.Errorf("GitHub push not configured")
	}

	reqBody := GitHubIssueRequest{
		Title:  title,
		Body:   body,
		Labels: cfg.Labels,
	}
	if reqBody.Labels == nil {
		reqBody.Labels = []string{"feedback"}
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/issues", cfg.Repo)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", 0, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var result GitHubIssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", 0, fmt.Errorf("failed to decode GitHub response: %w", err)
	}

	return result.HTMLURL, result.Number, nil
}

// BuildGitHubIssueBody constructs the body for a GitHub Issue from a submission.
func BuildGitHubIssueBody(sub *Submission, feedbackURL string) string {
	body := fmt.Sprintf(`## %s

**类型**: %s
**严重程度**: %s
**优先级**: %d
**状态**: %s

---

%s

---

> 源自反馈系统: %s
`, sub.Title, sub.FeedbackType, sub.Severity, sub.Priority, sub.Status, sub.Description, feedbackURL)
	return body
}
