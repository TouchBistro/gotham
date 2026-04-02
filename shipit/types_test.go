package shipit

import (
	"encoding/json"
	"testing"
	"time"
)

// sampleStackJSON is a representative Shipit API stack response with all fields populated.
// archived_since is set to null to verify pointer handling.
const sampleStackJSON = `{
	"id": 42,
	"repo_owner": "touchbistro",
	"repo_name": "myservice",
	"environment": "production",
	"html_url": "https://shipit.example.com/touchbistro/myservice/production",
	"url": "https://shipit.example.com/api/stacks/touchbistro/myservice/production",
	"tasks_url": "https://shipit.example.com/api/stacks/touchbistro/myservice/production/tasks",
	"deploy_url": "https://shipit.example.com/api/stacks/touchbistro/myservice/production/deploys",
	"merge_requests_url": "https://shipit.example.com/api/stacks/touchbistro/myservice/production/merge_requests",
	"undeployed_commits_count": 3,
	"is_locked": false,
	"continuous_deployment": true,
	"created_at": "2024-01-15T10:00:00Z",
	"updated_at": "2024-06-01T12:30:00Z",
	"last_deployed_at": "2024-05-30T08:45:00Z",
	"branch": "main",
	"merge_queue_enabled": false,
	"is_archived": false,
	"archived_since": null,
	"ignore_ci": false
}`

// TestStack_Unmarshal verifies that a Stack struct is correctly populated from
// a JSON payload matching the Shipit API schema.
func TestStack_Unmarshal(t *testing.T) {
	var s Stack
	if err := json.Unmarshal([]byte(sampleStackJSON), &s); err != nil {
		t.Fatalf("json.Unmarshal returned unexpected error: %v", err)
	}

	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"ID", s.ID, 42},
		{"RepoOwner", s.RepoOwner, "touchbistro"},
		{"RepoName", s.RepoName, "myservice"},
		{"Environment", s.Environment, "production"},
		{"HTMLURL", s.HTMLURL, "https://shipit.example.com/touchbistro/myservice/production"},
		{"URL", s.URL, "https://shipit.example.com/api/stacks/touchbistro/myservice/production"},
		{"TasksURL", s.TasksURL, "https://shipit.example.com/api/stacks/touchbistro/myservice/production/tasks"},
		{"DeployURL", s.DeployURL, "https://shipit.example.com/api/stacks/touchbistro/myservice/production/deploys"},
		{"MergeRequestsURL", s.MergeRequestsURL, "https://shipit.example.com/api/stacks/touchbistro/myservice/production/merge_requests"},
		{"UndeployedCommitsCount", s.UndeployedCommitsCount, 3},
		{"IsLocked", s.IsLocked, false},
		{"ContinuousDeployment", s.ContinuousDeployment, true},
		{"Branch", s.Branch, "main"},
		{"MergeQueueEnabled", s.MergeQueueEnabled, false},
		{"IsArchived", s.IsArchived, false},
		{"IgnoreCI", s.IgnoreCI, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v; want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

// TestStack_Unmarshal_TimestampFields verifies that time.Time fields are
// correctly parsed from RFC3339 JSON strings.
func TestStack_Unmarshal_TimestampFields(t *testing.T) {
	var s Stack
	if err := json.Unmarshal([]byte(sampleStackJSON), &s); err != nil {
		t.Fatalf("json.Unmarshal returned unexpected error: %v", err)
	}

	wantCreatedAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	wantUpdatedAt := time.Date(2024, 6, 1, 12, 30, 0, 0, time.UTC)
	wantLastDeployedAt := time.Date(2024, 5, 30, 8, 45, 0, 0, time.UTC)

	if !s.CreatedAt.Equal(wantCreatedAt) {
		t.Errorf("CreatedAt = %v; want %v", s.CreatedAt, wantCreatedAt)
	}
	if !s.UpdatedAt.Equal(wantUpdatedAt) {
		t.Errorf("UpdatedAt = %v; want %v", s.UpdatedAt, wantUpdatedAt)
	}
	if !s.LastDeployedAt.Equal(wantLastDeployedAt) {
		t.Errorf("LastDeployedAt = %v; want %v", s.LastDeployedAt, wantLastDeployedAt)
	}
}

// TestStack_Unmarshal_NullArchivedSince verifies that a null archived_since
// JSON value results in a nil pointer field on the Stack struct.
func TestStack_Unmarshal_NullArchivedSince(t *testing.T) {
	var s Stack
	if err := json.Unmarshal([]byte(sampleStackJSON), &s); err != nil {
		t.Fatalf("json.Unmarshal returned unexpected error: %v", err)
	}
	if s.ArchivedSince != nil {
		t.Errorf("ArchivedSince = %v; want nil for null JSON value", *s.ArchivedSince)
	}
}

// TestStack_Unmarshal_NonNullArchivedSince verifies that a non-null archived_since
// JSON string is correctly unmarshalled into a non-nil *string pointer.
func TestStack_Unmarshal_NonNullArchivedSince(t *testing.T) {
	jsonWithArchivedSince := `{
		"id": 1,
		"repo_owner": "touchbistro",
		"repo_name": "oldservice",
		"environment": "staging",
		"html_url": "",
		"url": "",
		"tasks_url": "",
		"deploy_url": "",
		"merge_requests_url": "",
		"undeployed_commits_count": 0,
		"is_locked": false,
		"continuous_deployment": false,
		"created_at": "2023-01-01T00:00:00Z",
		"updated_at": "2023-01-01T00:00:00Z",
		"last_deployed_at": "2023-01-01T00:00:00Z",
		"branch": "main",
		"merge_queue_enabled": false,
		"is_archived": true,
		"archived_since": "2024-01-01T00:00:00Z",
		"ignore_ci": false
	}`
	var s Stack
	if err := json.Unmarshal([]byte(jsonWithArchivedSince), &s); err != nil {
		t.Fatalf("json.Unmarshal returned unexpected error: %v", err)
	}
	if s.ArchivedSince == nil {
		t.Fatal("ArchivedSince = nil; want non-nil for non-null JSON value")
	}
	want := "2024-01-01T00:00:00Z"
	if *s.ArchivedSince != want {
		t.Errorf("*ArchivedSince = %q; want %q", *s.ArchivedSince, want)
	}
}

// TestStack_StackID verifies that StackID returns the correct
// repo_owner/repo_name/environment concatenation.
func TestStack_StackID(t *testing.T) {
	tests := []struct {
		name        string
		repoOwner   string
		repoName    string
		environment string
		want        string
	}{
		{
			name:        "standard production stack",
			repoOwner:   "touchbistro",
			repoName:    "myservice",
			environment: "production",
			want:        "touchbistro/myservice/production",
		},
		{
			name:        "staging stack",
			repoOwner:   "touchbistro",
			repoName:    "api",
			environment: "staging",
			want:        "touchbistro/api/staging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stack{
				RepoOwner:   tt.repoOwner,
				RepoName:    tt.repoName,
				Environment: tt.environment,
			}
			got := s.StackID()
			if got != tt.want {
				t.Errorf("StackID() = %q; want %q", got, tt.want)
			}
		})
	}
}
