package gh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"

	ghAPI "github.com/cli/go-gh/v2/pkg/api"
)

// Client wraps a go-gh REST client for GitHub API access.
type Client struct {
	client *ghAPI.RESTClient
}

// NewClient creates a new Client using the default authenticated REST client.
func NewClient() (*Client, error) {
	client, err := ghAPI.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("creating GitHub API client: %w", err)
	}
	return &Client{client: client}, nil
}

// jsonBody encodes v as JSON and returns a bytes.Reader.
func jsonBody(v any) (*bytes.Reader, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// --- Types ---

// BranchProtectionSettings represents desired branch protection rules.
type BranchProtectionSettings struct {
	BlockDirectPushes             bool
	RequirePrReviews              bool
	RequiredApprovals             int
	RequireStatusChecks           bool
	RequireConversationResolution bool
	EnforceAdmins                 bool
	AllowForcePushes              bool
	BlockDeletion                 bool
}

// RepoSettings represents repository merge and cleanup settings.
type RepoSettings struct {
	DeleteBranchOnMerge bool
	AllowSquashMerge    bool
	AllowMergeCommit    bool
	AllowRebaseMerge    bool
}

// SecuritySettings represents security feature toggles.
type SecuritySettings struct {
	DependabotAlerts             bool
	DependabotSecurityUpdates    bool
	SecretScanning               bool
	SecretScanningPushProtection bool
}

// Milestone represents a GitHub milestone.
type Milestone struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Description string `json:"description"`
	DueOn       string `json:"due_on"`
	State       string `json:"state"`
}

// Label represents a GitHub label.
type Label struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// --- Repo ---

// GetRepoInfo returns the default branch and visibility for a repository.
func (c *Client) GetRepoInfo(repo string) (defaultBranch string, visibility string, err error) {
	var result struct {
		DefaultBranch string `json:"default_branch"`
		Visibility    string `json:"visibility"`
	}
	err = c.client.Get(fmt.Sprintf("repos/%s", repo), &result)
	if err != nil {
		return "", "", fmt.Errorf("getting repo info: %w", err)
	}
	return result.DefaultBranch, result.Visibility, nil
}

// UpdateRepoSettings applies repository settings via PATCH.
func (c *Client) UpdateRepoSettings(repo string, s RepoSettings) error {
	body, err := jsonBody(map[string]any{
		"delete_branch_on_merge": s.DeleteBranchOnMerge,
		"allow_squash_merge":    s.AllowSquashMerge,
		"allow_merge_commit":    s.AllowMergeCommit,
		"allow_rebase_merge":    s.AllowRebaseMerge,
	})
	if err != nil {
		return err
	}
	return c.client.Patch(fmt.Sprintf("repos/%s", repo), body, nil)
}

// UpdateBranchProtection sets branch protection rules.
func (c *Client) UpdateBranchProtection(repo string, branch string, s BranchProtectionSettings) error {
	var requiredStatusChecks any
	if s.RequireStatusChecks {
		requiredStatusChecks = map[string]any{
			"strict":   true,
			"contexts": []string{},
		}
	}

	var requiredPRReviews any
	if s.RequirePrReviews {
		requiredPRReviews = map[string]any{
			"required_approving_review_count": s.RequiredApprovals,
			"dismiss_stale_reviews":           false,
			"require_code_owner_reviews":      false,
		}
	} else if s.BlockDirectPushes {
		requiredPRReviews = map[string]any{
			"required_approving_review_count": 0,
			"dismiss_stale_reviews":           false,
			"require_code_owner_reviews":      false,
		}
	}

	body, err := jsonBody(map[string]any{
		"required_status_checks":           requiredStatusChecks,
		"enforce_admins":                   s.EnforceAdmins,
		"required_pull_request_reviews":    requiredPRReviews,
		"restrictions":                     nil,
		"allow_force_pushes":               s.AllowForcePushes,
		"allow_deletions":                  !s.BlockDeletion,
		"required_conversation_resolution": s.RequireConversationResolution,
		"block_creations":                  false,
		"lock_branch":                      false,
		"allow_fork_syncing":               false,
	})
	if err != nil {
		return err
	}

	path := fmt.Sprintf("repos/%s/branches/%s/protection", repo, url.PathEscape(branch))
	return c.client.Put(path, body, nil)
}

// --- Security ---

// EnableDependabotAlerts enables Dependabot alerts for the repository.
func (c *Client) EnableDependabotAlerts(repo string) error {
	path := fmt.Sprintf("repos/%s/vulnerability-alerts", repo)
	return c.client.Put(path, nil, nil)
}

// EnableDependabotSecurityUpdates enables Dependabot security updates.
func (c *Client) EnableDependabotSecurityUpdates(repo string) error {
	path := fmt.Sprintf("repos/%s/automated-security-fixes", repo)
	return c.client.Put(path, nil, nil)
}

// EnableSecretScanning enables secret scanning for the repository.
func (c *Client) EnableSecretScanning(repo string) error {
	body, err := jsonBody(map[string]any{
		"security_and_analysis": map[string]any{
			"secret_scanning": map[string]string{
				"status": "enabled",
			},
		},
	})
	if err != nil {
		return err
	}
	return c.client.Patch(fmt.Sprintf("repos/%s", repo), body, nil)
}

// EnableSecretScanningPushProtection enables secret scanning push protection.
func (c *Client) EnableSecretScanningPushProtection(repo string) error {
	body, err := jsonBody(map[string]any{
		"security_and_analysis": map[string]any{
			"secret_scanning_push_protection": map[string]string{
				"status": "enabled",
			},
		},
	})
	if err != nil {
		return err
	}
	return c.client.Patch(fmt.Sprintf("repos/%s", repo), body, nil)
}

// --- Milestones ---

// ListMilestones returns all milestones for a repository.
func (c *Client) ListMilestones(repo string) ([]Milestone, error) {
	var all []Milestone
	page := 1
	for {
		path := fmt.Sprintf("repos/%s/milestones?state=all&per_page=100&page=%d", repo, page)
		var batch []Milestone
		err := c.client.Get(path, &batch)
		if err != nil {
			return nil, fmt.Errorf("listing milestones: %w", err)
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			break
		}
		page++
	}
	return all, nil
}

// CreateMilestone creates a new milestone.
func (c *Client) CreateMilestone(repo string, title, description, dueOn string) error {
	body, err := jsonBody(map[string]string{
		"title":       title,
		"description": description,
		"due_on":      dueOn,
	})
	if err != nil {
		return err
	}
	return c.client.Post(fmt.Sprintf("repos/%s/milestones", repo), body, nil)
}

// UpdateMilestone updates an existing milestone.
func (c *Client) UpdateMilestone(repo string, number int, title, description string) error {
	body, err := jsonBody(map[string]string{
		"title":       title,
		"description": description,
	})
	if err != nil {
		return err
	}
	return c.client.Patch(fmt.Sprintf("repos/%s/milestones/%d", repo, number), body, nil)
}

// --- Labels ---

// ListLabels returns all labels for a repository.
func (c *Client) ListLabels(repo string) ([]Label, error) {
	var all []Label
	page := 1
	for {
		path := fmt.Sprintf("repos/%s/labels?per_page=100&page=%d", repo, page)
		var batch []Label
		err := c.client.Get(path, &batch)
		if err != nil {
			return nil, fmt.Errorf("listing labels: %w", err)
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			break
		}
		page++
	}
	return all, nil
}

// CreateLabel creates a new label.
func (c *Client) CreateLabel(repo string, name, color, description string) error {
	body, err := jsonBody(map[string]string{
		"name":        name,
		"color":       color,
		"description": description,
	})
	if err != nil {
		return err
	}
	return c.client.Post(fmt.Sprintf("repos/%s/labels", repo), body, nil)
}

// UpdateLabel updates an existing label.
func (c *Client) UpdateLabel(repo string, name, color, description string) error {
	body, err := jsonBody(map[string]string{
		"color":       color,
		"description": description,
	})
	if err != nil {
		return err
	}
	path := fmt.Sprintf("repos/%s/labels/%s", repo, url.PathEscape(name))
	return c.client.Patch(path, body, nil)
}
