package circleci

import (
	"encoding/json"
	"testing"
)

// sampleProjectJSON is a representative CircleCI v2 GET /project/{slug} response.
const sampleProjectJSON = `{
	"slug": "gh/TouchBistro/my-service",
	"name": "my-service",
	"id": "abc-123-def",
	"organization_name": "TouchBistro",
	"organization_slug": "gh/TouchBistro",
	"organization_id": "org-456",
	"vcs_info": {
		"vcs_url": "https://github.com/TouchBistro/my-service",
		"provider": "GitHub",
		"default_branch": "main"
	}
}`

// TestProject_Unmarshal verifies that a Project struct is correctly populated
// from a JSON payload matching the CircleCI v2 project schema.
func TestProject_Unmarshal(t *testing.T) {
	var p Project
	if err := json.Unmarshal([]byte(sampleProjectJSON), &p); err != nil {
		t.Fatalf("json.Unmarshal returned unexpected error: %v", err)
	}

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"Slug", p.Slug, "gh/TouchBistro/my-service"},
		{"Name", p.Name, "my-service"},
		{"ID", p.ID, "abc-123-def"},
		{"OrganizationName", p.OrganizationName, "TouchBistro"},
		{"OrganizationSlug", p.OrganizationSlug, "gh/TouchBistro"},
		{"OrganizationID", p.OrganizationID, "org-456"},
		{"VCSInfo.VCSURL", p.VCSInfo.VCSURL, "https://github.com/TouchBistro/my-service"},
		{"VCSInfo.Provider", p.VCSInfo.Provider, "GitHub"},
		{"VCSInfo.DefaultBranch", p.VCSInfo.DefaultBranch, "main"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q; want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

// sampleProjectInsightsJSON is a representative CircleCI v2
// GET /insights/{slug}/workflows response.
const sampleProjectInsightsJSON = `{
	"items": [
		{
			"name": "build-and-test",
			"metrics": {
				"total_runs": 150,
				"successful_runs": 140,
				"failed_runs": 10,
				"success_rate": 0.9333,
				"throughput": 5.0,
				"mean_duration_secs": 320,
				"total_duration_secs": 48000
			},
			"trends": {
				"total_runs": 0.05,
				"failed_runs": -0.02,
				"success_rate": 0.01
			}
		}
	],
	"next_page_token": "token-abc"
}`

// TestProjectInsights_Unmarshal verifies that a ProjectInsights struct is
// correctly populated from a JSON payload matching the CircleCI v2 insights schema.
func TestProjectInsights_Unmarshal(t *testing.T) {
	var pi ProjectInsights
	if err := json.Unmarshal([]byte(sampleProjectInsightsJSON), &pi); err != nil {
		t.Fatalf("json.Unmarshal returned unexpected error: %v", err)
	}

	if len(pi.Items) != 1 {
		t.Fatalf("Items length = %d; want 1", len(pi.Items))
	}

	item := pi.Items[0]
	if item.Name != "build-and-test" {
		t.Errorf("Items[0].Name = %q; want %q", item.Name, "build-and-test")
	}
	if item.Metrics.TotalRuns != 150 {
		t.Errorf("Metrics.TotalRuns = %d; want 150", item.Metrics.TotalRuns)
	}
	if item.Metrics.SuccessfulRuns != 140 {
		t.Errorf("Metrics.SuccessfulRuns = %d; want 140", item.Metrics.SuccessfulRuns)
	}
	if item.Metrics.FailedRuns != 10 {
		t.Errorf("Metrics.FailedRuns = %d; want 10", item.Metrics.FailedRuns)
	}
	if item.Metrics.SuccessRate != 0.9333 {
		t.Errorf("Metrics.SuccessRate = %f; want 0.9333", item.Metrics.SuccessRate)
	}
	if item.Metrics.MeanDurationSecs != 320 {
		t.Errorf("Metrics.MeanDurationSecs = %f; want 320", item.Metrics.MeanDurationSecs)
	}
	if item.Trends.TotalRuns != 0.05 {
		t.Errorf("Trends.TotalRuns = %f; want 0.05", item.Trends.TotalRuns)
	}
	if pi.NextPageToken != "token-abc" {
		t.Errorf("NextPageToken = %q; want %q", pi.NextPageToken, "token-abc")
	}
}

// sampleFollowProjectResponseJSON is a representative CircleCI v1.1
// POST /project/github/{owner}/{repo}/follow response.
const sampleFollowProjectResponseJSON = `{
	"followed": true
}`

// TestFollowProjectResponse_Unmarshal verifies that a FollowProjectResponse
// struct is correctly populated from a JSON payload.
func TestFollowProjectResponse_Unmarshal(t *testing.T) {
	var r FollowProjectResponse
	if err := json.Unmarshal([]byte(sampleFollowProjectResponseJSON), &r); err != nil {
		t.Fatalf("json.Unmarshal returned unexpected error: %v", err)
	}
	if r.Followed != true {
		t.Errorf("Followed = %v; want true", r.Followed)
	}
}

// sampleUnfollowProjectResponseJSON is a representative CircleCI v1.1
// POST /project/github/{owner}/{repo}/unfollow response.
const sampleUnfollowProjectResponseJSON = `{
	"followed": false
}`

// TestUnfollowProjectResponse_Unmarshal verifies that an UnfollowProjectResponse
// struct is correctly populated from a JSON payload.
func TestUnfollowProjectResponse_Unmarshal(t *testing.T) {
	var r UnfollowProjectResponse
	if err := json.Unmarshal([]byte(sampleUnfollowProjectResponseJSON), &r); err != nil {
		t.Fatalf("json.Unmarshal returned unexpected error: %v", err)
	}
	if r.Followed != false {
		t.Errorf("Followed = %v; want false", r.Followed)
	}
}

// samplePipelineJSON is a representative CircleCI v2 pipeline object.
const sampleListPipelinesResponseJSON = `{
	"items": [
		{
			"id": "pipeline-001",
			"state": "created",
			"number": 42,
			"created_at": "2025-03-15T10:00:00Z"
		},
		{
			"id": "pipeline-002",
			"state": "errored",
			"number": 41,
			"created_at": "2025-03-14T09:00:00Z"
		}
	],
	"next_page_token": "page-token-xyz"
}`

// TestListPipelinesResponse_Unmarshal verifies that a ListPipelinesResponse
// struct is correctly populated from a JSON payload matching the CircleCI v2
// pipeline list schema.
func TestListPipelinesResponse_Unmarshal(t *testing.T) {
	var r ListPipelinesResponse
	if err := json.Unmarshal([]byte(sampleListPipelinesResponseJSON), &r); err != nil {
		t.Fatalf("json.Unmarshal returned unexpected error: %v", err)
	}

	if len(r.Items) != 2 {
		t.Fatalf("Items length = %d; want 2", len(r.Items))
	}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"Items[0].ID", r.Items[0].ID, "pipeline-001"},
		{"Items[0].State", r.Items[0].State, "created"},
		{"Items[0].Number", r.Items[0].Number, 42},
		{"Items[0].CreatedAt", r.Items[0].CreatedAt, "2025-03-15T10:00:00Z"},
		{"Items[1].ID", r.Items[1].ID, "pipeline-002"},
		{"Items[1].State", r.Items[1].State, "errored"},
		{"Items[1].Number", r.Items[1].Number, 41},
		{"NextPageToken", r.NextPageToken, "page-token-xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v; want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

// sampleListWorkflowsResponseJSON is a representative CircleCI v2
// GET /pipeline/{id}/workflow response.
const sampleListWorkflowsResponseJSON = `{
	"items": [
		{
			"id": "wf-001",
			"name": "build-and-test",
			"status": "success",
			"created_at": "2025-03-15T10:01:00Z",
			"stopped_at": "2025-03-15T10:05:00Z",
			"pipeline_id": "pipeline-001",
			"pipeline_number": 42,
			"project_slug": "gh/TouchBistro/my-service"
		}
	],
	"next_page_token": ""
}`

// TestListWorkflowsResponse_Unmarshal verifies that a ListWorkflowsResponse
// struct is correctly populated from a JSON payload matching the CircleCI v2
// workflow list schema.
func TestListWorkflowsResponse_Unmarshal(t *testing.T) {
	var r ListWorkflowsResponse
	if err := json.Unmarshal([]byte(sampleListWorkflowsResponseJSON), &r); err != nil {
		t.Fatalf("json.Unmarshal returned unexpected error: %v", err)
	}

	if len(r.Items) != 1 {
		t.Fatalf("Items length = %d; want 1", len(r.Items))
	}

	w := r.Items[0]
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"ID", w.ID, "wf-001"},
		{"Name", w.Name, "build-and-test"},
		{"Status", w.Status, "success"},
		{"CreatedAt", w.CreatedAt, "2025-03-15T10:01:00Z"},
		{"StoppedAt", w.StoppedAt, "2025-03-15T10:05:00Z"},
		{"PipelineID", w.PipelineID, "pipeline-001"},
		{"ProjectSlug", w.ProjectSlug, "gh/TouchBistro/my-service"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q; want %q", tt.name, tt.got, tt.want)
			}
		})
	}

	if w.PipelineNumber != 42 {
		t.Errorf("PipelineNumber = %d; want 42", w.PipelineNumber)
	}
}
