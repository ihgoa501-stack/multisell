package feedback

import (
	"encoding/json"
	"fmt"
)

// PushDispatcher manages pushing accepted feedback to external systems.
type PushDispatcher struct {
	svc         *Service
	feedbackURL string // base URL for feedback detail links
}

// NewPushDispatcher creates a new push dispatcher.
func NewPushDispatcher(svc *Service, feedbackURL string) *PushDispatcher {
	return &PushDispatcher{svc: svc, feedbackURL: feedbackURL}
}

// PushConfig holds all push-related configuration from feedback_projects.settings.
type PushConfig struct {
	GitHub *GitHubPushConfig `json:"push_github"`
}

// DispatchPush attempts to push a submission to all configured destinations.
func (d *PushDispatcher) DispatchPush(sub *Submission, settingsJSON string) {
	if settingsJSON == "" || settingsJSON == "{}" {
		return
	}

	var cfg PushConfig
	if err := json.Unmarshal([]byte(settingsJSON), &cfg); err != nil {
		return
	}

	if cfg.GitHub != nil {
		d.pushToGitHub(sub, cfg.GitHub)
	}
}

func (d *PushDispatcher) pushToGitHub(sub *Submission, ghCfg *GitHubPushConfig) {
	feedbackURL := fmt.Sprintf("%s/feedback/%d", d.feedbackURL, sub.ID)
	body := BuildGitHubIssueBody(sub, feedbackURL)

	issueURL, issueNumber, err := PushToGitHub(ghCfg, sub.Title, body)
	if err != nil {
		d.svc.RecordPush(sub.ID, "github", "", "", body, "failed", err.Error())
		return
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"issue_number": issueNumber,
		"issue_url":    issueURL,
	})
	d.svc.RecordPush(sub.ID, "github", issueURL, fmt.Sprintf("%d", issueNumber), string(payload), "success", "")
}

// ExtractGitHubPushConfig parses GitHub push config from a project's settings JSON.
func ExtractGitHubPushConfig(settingsJSON string) *GitHubPushConfig {
	if settingsJSON == "" || settingsJSON == "{}" {
		return nil
	}
	var cfg struct {
		GitHub *GitHubPushConfig `json:"push_github"`
	}
	if err := json.Unmarshal([]byte(settingsJSON), &cfg); err != nil {
		return nil
	}
	return cfg.GitHub
}
