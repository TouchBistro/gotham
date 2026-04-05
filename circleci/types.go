package circleci

// VCSInfo contains version control system information for a CircleCI project.
type VCSInfo struct {
	// VCSURL is the URL of the repository (e.g. "https://github.com/owner/repo").
	VCSURL string `json:"vcs_url"`
	// Provider is the VCS provider name (e.g. "GitHub").
	Provider string `json:"provider"`
	// DefaultBranch is the default branch of the repository (e.g. "main").
	DefaultBranch string `json:"default_branch"`
}

// Project represents a CircleCI project as returned by the v2
// GET /project/{slug} endpoint.
type Project struct {
	// Slug is the project slug (e.g. "gh/TouchBistro/my-service").
	Slug string `json:"slug"`
	// Name is the project name.
	Name string `json:"name"`
	// ID is the unique project identifier.
	ID string `json:"id"`
	// OrganizationName is the human-readable name of the organization.
	OrganizationName string `json:"organization_name"`
	// OrganizationSlug is the slug of the organization (e.g. "gh/TouchBistro").
	OrganizationSlug string `json:"organization_slug"`
	// OrganizationID is the unique identifier of the organization.
	OrganizationID string `json:"organization_id"`
	// VCSInfo contains version control information for the project.
	VCSInfo VCSInfo `json:"vcs_info"`
}

// WorkflowMetrics contains aggregate metrics for a workflow returned by the
// CircleCI v2 insights endpoint.
type WorkflowMetrics struct {
	// TotalRuns is the total number of workflow runs.
	TotalRuns int `json:"total_runs"`
	// SuccessfulRuns is the number of successful runs.
	SuccessfulRuns int `json:"successful_runs"`
	// FailedRuns is the number of failed runs.
	FailedRuns int `json:"failed_runs"`
	// SuccessRate is the ratio of successful runs to total runs.
	SuccessRate float64 `json:"success_rate"`
	// Throughput is the average number of runs per day.
	Throughput float64 `json:"throughput"`
	// MeanDurationSecs is the average duration of a run in seconds.
	MeanDurationSecs float64 `json:"mean_duration_secs"`
	// TotalDurationSecs is the total duration across all runs in seconds.
	TotalDurationSecs float64 `json:"total_duration_secs"`
}

// WorkflowTrends contains trend data for a workflow returned by the
// CircleCI v2 insights endpoint.
type WorkflowTrends struct {
	// TotalRuns is the trend delta for total runs.
	TotalRuns float64 `json:"total_runs"`
	// FailedRuns is the trend delta for failed runs.
	FailedRuns float64 `json:"failed_runs"`
	// SuccessRate is the trend delta for success rate.
	SuccessRate float64 `json:"success_rate"`
}

// WorkflowInsightItem represents a single workflow's insights summary,
// including its name, aggregate metrics, and trends.
type WorkflowInsightItem struct {
	// Name is the workflow name.
	Name string `json:"name"`
	// Metrics contains aggregate metrics for the workflow.
	Metrics WorkflowMetrics `json:"metrics"`
	// Trends contains trend data for the workflow.
	Trends WorkflowTrends `json:"trends"`
}

// ProjectInsights represents the response from the CircleCI v2
// GET /insights/{slug}/workflows endpoint.
type ProjectInsights struct {
	// Items is the list of workflow insight summaries.
	Items []WorkflowInsightItem `json:"items"`
	// NextPageToken is the pagination token for the next page of results.
	NextPageToken string `json:"next_page_token"`
}

// FollowProjectResponse represents the response from the CircleCI v1.1
// POST /project/github/{owner}/{repo}/follow endpoint.
type FollowProjectResponse struct {
	// Followed indicates whether the project is now followed.
	Followed bool `json:"followed"`
}

// UnfollowProjectResponse represents the response from the CircleCI v1.1
// POST /project/github/{owner}/{repo}/unfollow endpoint.
type UnfollowProjectResponse struct {
	// Followed indicates whether the project is still followed (expected false).
	Followed bool `json:"followed"`
}

// Pipeline represents a CircleCI pipeline as returned by the v2
// GET /project/{slug}/pipeline endpoint.
type Pipeline struct {
	// ID is the unique pipeline identifier.
	ID string `json:"id"`
	// State is the current pipeline state (e.g. "created", "errored").
	State string `json:"state"`
	// Number is the pipeline number within the project.
	Number int `json:"number"`
	// CreatedAt is the RFC3339 timestamp when the pipeline was created.
	CreatedAt string `json:"created_at"`
}

// ListPipelinesResponse represents the paginated response from the CircleCI v2
// GET /project/{slug}/pipeline endpoint.
type ListPipelinesResponse struct {
	// Items is the list of pipelines.
	Items []Pipeline `json:"items"`
	// NextPageToken is the pagination token for the next page of results.
	NextPageToken string `json:"next_page_token"`
}

// Workflow represents a CircleCI workflow as returned by the v2
// GET /pipeline/{id}/workflow endpoint.
type Workflow struct {
	// ID is the unique workflow identifier.
	ID string `json:"id"`
	// Name is the workflow name.
	Name string `json:"name"`
	// Status is the current workflow status (e.g. "success", "failed", "running").
	Status string `json:"status"`
	// CreatedAt is the RFC3339 timestamp when the workflow was created.
	CreatedAt string `json:"created_at"`
	// StoppedAt is the RFC3339 timestamp when the workflow stopped, or empty if still running.
	StoppedAt string `json:"stopped_at"`
	// PipelineID is the identifier of the pipeline this workflow belongs to.
	PipelineID string `json:"pipeline_id"`
	// PipelineNumber is the number of the pipeline this workflow belongs to.
	PipelineNumber int `json:"pipeline_number"`
	// ProjectSlug is the project slug (e.g. "gh/TouchBistro/my-service").
	ProjectSlug string `json:"project_slug"`
}

// ListWorkflowsResponse represents the paginated response from the CircleCI v2
// GET /pipeline/{id}/workflow endpoint.
type ListWorkflowsResponse struct {
	// Items is the list of workflows.
	Items []Workflow `json:"items"`
	// NextPageToken is the pagination token for the next page of results.
	NextPageToken string `json:"next_page_token"`
}
