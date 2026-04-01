package shipit

import (
	"fmt"
	"time"
)

// Stack represents a single deployment stack returned by the Shipit API.
// The stack identifier used in API endpoints is the concatenation
// repo_owner/repo_name/environment, accessible via StackID().
type Stack struct {
	// ID is the internal integer identifier of the stack.
	ID int `json:"id"`
	// RepoOwner is the GitHub organisation or user that owns the repository.
	RepoOwner string `json:"repo_owner"`
	// RepoName is the name of the GitHub repository.
	RepoName string `json:"repo_name"`
	// Environment is the deployment environment name (e.g. "production", "staging").
	Environment string `json:"environment"`
	// HTMLURL is the human-readable Shipit web URL for this stack.
	HTMLURL string `json:"html_url"`
	// URL is the canonical API URL for this stack resource.
	URL string `json:"url"`
	// TasksURL is the API URL for listing tasks associated with this stack.
	TasksURL string `json:"tasks_url"`
	// DeployURL is the API URL for triggering deployments on this stack.
	DeployURL string `json:"deploy_url"`
	// MergeRequestsURL is the API URL for merge requests associated with this stack.
	MergeRequestsURL string `json:"merge_requests_url"`
	// UndeployedCommitsCount is the number of commits not yet deployed to this stack.
	UndeployedCommitsCount int `json:"undeployed_commits_count"`
	// IsLocked indicates whether the stack is currently locked against deployments.
	IsLocked bool `json:"is_locked"`
	// ContinuousDeployment indicates whether the stack auto-deploys on new commits.
	ContinuousDeployment bool `json:"continuous_deployment"`
	// CreatedAt is the timestamp when the stack was created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the timestamp when the stack was last updated.
	UpdatedAt time.Time `json:"updated_at"`
	// LastDeployedAt is the timestamp of the most recent deployment.
	LastDeployedAt time.Time `json:"last_deployed_at"`
	// Branch is the Git branch tracked by this stack.
	Branch string `json:"branch"`
	// MergeQueueEnabled indicates whether the merge queue feature is active.
	MergeQueueEnabled bool `json:"merge_queue_enabled"`
	// IsArchived indicates whether the stack has been archived.
	IsArchived bool `json:"is_archived"`
	// ArchivedSince is the timestamp when the stack was archived, or nil if not archived.
	ArchivedSince *string `json:"archived_since"`
	// IgnoreCI indicates whether CI checks are bypassed for this stack.
	IgnoreCI bool `json:"ignore_ci"`
}

// StackID returns the stack identifier used in Shipit API endpoint paths.
// It is the concatenation of RepoOwner, RepoName, and Environment separated by "/".
func (s Stack) StackID() string {
	return fmt.Sprintf("%s/%s/%s", s.RepoOwner, s.RepoName, s.Environment)
}
