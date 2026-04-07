package circleci

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_ListWorkflows_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"items": [
				{
					"id": "wf-1",
					"name": "build-and-test",
					"status": "success",
					"created_at": "2026-01-01T00:00:00Z",
					"stopped_at": "2026-01-01T00:05:00Z",
					"pipeline_id": "pipe-1",
					"pipeline_number": 42,
					"project_slug": "gh/TouchBistro/my-service"
				},
				{
					"id": "wf-2",
					"name": "deploy",
					"status": "running",
					"created_at": "2026-01-02T00:00:00Z",
					"stopped_at": null,
					"pipeline_id": "pipe-1",
					"pipeline_number": 42,
					"project_slug": "gh/TouchBistro/my-service"
				}
			],
			"next_page_token": "token-xyz"
		}`))
	}))
	defer srv.Close()

	c, err := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()
	if err != nil {
		t.Fatalf("Build returned unexpected error: %v", err)
	}

	resp, err := c.ListWorkflows(context.Background(), "pipe-1")
	if err != nil {
		t.Fatalf("ListWorkflows returned unexpected error: %v", err)
	}

	if len(resp.Items) != 2 {
		t.Fatalf("len(Items) = %d; want 2", len(resp.Items))
	}
	if resp.Items[0].ID != "wf-1" {
		t.Errorf("Items[0].ID = %q; want %q", resp.Items[0].ID, "wf-1")
	}
	if resp.Items[0].Name != "build-and-test" {
		t.Errorf("Items[0].Name = %q; want %q", resp.Items[0].Name, "build-and-test")
	}
	if resp.Items[0].Status != WorkflowStatusSuccess {
		t.Errorf("Items[0].Status = %q; want %q", resp.Items[0].Status, WorkflowStatusSuccess)
	}
	if resp.Items[0].PipelineID != "pipe-1" {
		t.Errorf("Items[0].PipelineID = %q; want %q", resp.Items[0].PipelineID, "pipe-1")
	}
	if resp.Items[0].PipelineNumber != 42 {
		t.Errorf("Items[0].PipelineNumber = %d; want 42", resp.Items[0].PipelineNumber)
	}
	if resp.Items[0].StoppedAt == nil {
		t.Error("Items[0].StoppedAt is nil; want non-nil for completed workflow")
	}
	if resp.Items[1].ID != "wf-2" {
		t.Errorf("Items[1].ID = %q; want %q", resp.Items[1].ID, "wf-2")
	}
	if resp.Items[1].Status != WorkflowStatusRunning {
		t.Errorf("Items[1].Status = %q; want %q", resp.Items[1].Status, WorkflowStatusRunning)
	}
	if resp.Items[1].StoppedAt != nil {
		t.Errorf("Items[1].StoppedAt = %v; want nil for running workflow", resp.Items[1].StoppedAt)
	}
	if resp.NextPageToken != "token-xyz" {
		t.Errorf("NextPageToken = %q; want %q", resp.NextPageToken, "token-xyz")
	}
}

func TestClient_ListWorkflows_EmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items": [], "next_page_token": ""}`))
	}))
	defer srv.Close()

	c, err := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()
	if err != nil {
		t.Fatalf("Build returned unexpected error: %v", err)
	}

	resp, err := c.ListWorkflows(context.Background(), "pipe-1")
	if err != nil {
		t.Fatalf("ListWorkflows returned unexpected error: %v", err)
	}

	if len(resp.Items) != 0 {
		t.Errorf("len(Items) = %d; want 0", len(resp.Items))
	}
	if resp.NextPageToken != "" {
		t.Errorf("NextPageToken = %q; want empty string", resp.NextPageToken)
	}
}

func TestClient_ListWorkflows_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Pipeline not found"}`))
	}))
	defer srv.Close()

	c, err := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()
	if err != nil {
		t.Fatalf("Build returned unexpected error: %v", err)
	}

	resp, err := c.ListWorkflows(context.Background(), "nonexistent-pipe")
	if err == nil {
		t.Fatal("ListWorkflows returned nil error; want non-nil for 404 response")
	}
	if resp != nil {
		t.Errorf("ListWorkflows returned non-nil response on error; want nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "404") {
		t.Errorf("error message %q does not contain status code 404", errMsg)
	}
	if !strings.Contains(errMsg, "Pipeline not found") {
		t.Errorf("error message %q does not contain body excerpt", errMsg)
	}
}

func TestClient_ListWorkflows_VerifiesRequestPath(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items": [], "next_page_token": ""}`))
	}))
	defer srv.Close()

	c, err := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()
	if err != nil {
		t.Fatalf("Build returned unexpected error: %v", err)
	}

	_, err = c.ListWorkflows(context.Background(), "pipe-abc-123")
	if err != nil {
		t.Fatalf("ListWorkflows returned unexpected error: %v", err)
	}

	expectedPath := "/pipeline/pipe-abc-123/workflow"
	if gotPath != expectedPath {
		t.Errorf("request path = %q; want %q", gotPath, expectedPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("request method = %q; want %q", gotMethod, http.MethodGet)
	}
}

func TestClient_CancelWorkflow_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"message":"Accepted."}`))
	}))
	defer srv.Close()

	c, err := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()
	if err != nil {
		t.Fatalf("Build returned unexpected error: %v", err)
	}

	err = c.CancelWorkflow(context.Background(), "wf-abc-123")
	if err != nil {
		t.Fatalf("CancelWorkflow returned unexpected error: %v", err)
	}
}

func TestClient_CancelWorkflow_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Workflow not found"}`))
	}))
	defer srv.Close()

	c, err := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()
	if err != nil {
		t.Fatalf("Build returned unexpected error: %v", err)
	}

	err = c.CancelWorkflow(context.Background(), "nonexistent-wf")
	if err == nil {
		t.Fatal("CancelWorkflow returned nil error; want non-nil for 404 response")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "404") {
		t.Errorf("error message %q does not contain status code 404", errMsg)
	}
	if !strings.Contains(errMsg, "Workflow not found") {
		t.Errorf("error message %q does not contain body excerpt", errMsg)
	}
}

func TestClient_CancelWorkflow_VerifiesRequestMethodAndPath(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, err := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()
	if err != nil {
		t.Fatalf("Build returned unexpected error: %v", err)
	}

	err = c.CancelWorkflow(context.Background(), "wf-xyz-789")
	if err != nil {
		t.Fatalf("CancelWorkflow returned unexpected error: %v", err)
	}

	expectedPath := "/workflow/wf-xyz-789/cancel"
	if gotPath != expectedPath {
		t.Errorf("request path = %q; want %q", gotPath, expectedPath)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("request method = %q; want %q", gotMethod, http.MethodPost)
	}
}
