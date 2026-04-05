package circleci

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_GetProjectInsights_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"items": [
				{
					"name": "build-and-test",
					"metrics": {
						"total_runs": 150,
						"successful_runs": 140,
						"failed_runs": 10,
						"success_rate": 0.9333,
						"throughput": 5.0,
						"mean_duration_secs": 120.5,
						"total_duration_secs": 18075.0
					},
					"trends": {
						"total_runs": 0.05,
						"failed_runs": -0.02,
						"success_rate": 0.01
					}
				},
				{
					"name": "deploy",
					"metrics": {
						"total_runs": 50,
						"successful_runs": 48,
						"failed_runs": 2,
						"success_rate": 0.96,
						"throughput": 1.5,
						"mean_duration_secs": 300.0,
						"total_duration_secs": 15000.0
					},
					"trends": {
						"total_runs": 0.10,
						"failed_runs": 0.0,
						"success_rate": 0.02
					}
				}
			],
			"next_page_token": "abc123"
		}`))
	}))
	defer srv.Close()

	c := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()

	insights, err := c.GetProjectInsights(context.Background(), "TouchBistro", "my-service")
	if err != nil {
		t.Fatalf("GetProjectInsights returned unexpected error: %v", err)
	}

	if len(insights.Items) != 2 {
		t.Fatalf("Items length = %d; want 2", len(insights.Items))
	}

	first := insights.Items[0]
	if first.Name != "build-and-test" {
		t.Errorf("Items[0].Name = %q; want %q", first.Name, "build-and-test")
	}
	if first.Metrics.TotalRuns != 150 {
		t.Errorf("Items[0].Metrics.TotalRuns = %d; want 150", first.Metrics.TotalRuns)
	}
	if first.Metrics.SuccessfulRuns != 140 {
		t.Errorf("Items[0].Metrics.SuccessfulRuns = %d; want 140", first.Metrics.SuccessfulRuns)
	}
	if first.Metrics.FailedRuns != 10 {
		t.Errorf("Items[0].Metrics.FailedRuns = %d; want 10", first.Metrics.FailedRuns)
	}
	if first.Metrics.SuccessRate != 0.9333 {
		t.Errorf("Items[0].Metrics.SuccessRate = %f; want 0.9333", first.Metrics.SuccessRate)
	}
	if first.Metrics.Throughput != 5.0 {
		t.Errorf("Items[0].Metrics.Throughput = %f; want 5.0", first.Metrics.Throughput)
	}
	if first.Metrics.MeanDurationSecs != 120.5 {
		t.Errorf("Items[0].Metrics.MeanDurationSecs = %f; want 120.5", first.Metrics.MeanDurationSecs)
	}
	if first.Trends.SuccessRate != 0.01 {
		t.Errorf("Items[0].Trends.SuccessRate = %f; want 0.01", first.Trends.SuccessRate)
	}

	second := insights.Items[1]
	if second.Name != "deploy" {
		t.Errorf("Items[1].Name = %q; want %q", second.Name, "deploy")
	}
	if second.Metrics.TotalRuns != 50 {
		t.Errorf("Items[1].Metrics.TotalRuns = %d; want 50", second.Metrics.TotalRuns)
	}

	if insights.NextPageToken != "abc123" {
		t.Errorf("NextPageToken = %q; want %q", insights.NextPageToken, "abc123")
	}
}

func TestClient_GetProjectInsights_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Internal server error"}`))
	}))
	defer srv.Close()

	c := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()

	insights, err := c.GetProjectInsights(context.Background(), "TouchBistro", "my-service")
	if err == nil {
		t.Fatal("GetProjectInsights returned nil error; want non-nil for 500 response")
	}
	if insights != nil {
		t.Errorf("GetProjectInsights returned non-nil insights on error; want nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "500") {
		t.Errorf("error message %q does not contain status code 500", errMsg)
	}
	if !strings.Contains(errMsg, "Internal server error") {
		t.Errorf("error message %q does not contain body excerpt", errMsg)
	}
}

func TestClient_GetProjectInsights_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not valid json`))
	}))
	defer srv.Close()

	c := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()

	insights, err := c.GetProjectInsights(context.Background(), "TouchBistro", "my-service")
	if err == nil {
		t.Fatal("GetProjectInsights returned nil error; want non-nil for invalid JSON")
	}
	if insights != nil {
		t.Errorf("GetProjectInsights returned non-nil insights on error; want nil")
	}
	if !strings.Contains(err.Error(), "decoding GetProjectInsights response") {
		t.Errorf("error message %q does not mention decoding failure", err.Error())
	}
}

func TestClient_GetProjectInsights_VerifiesRequestPathAndAuth(t *testing.T) {
	var gotPath, gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("Circle-Token")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"items":[],"next_page_token":""}`))
	}))
	defer srv.Close()

	c := NewClientBuilder().
		WithToken("my-secret-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()

	_, err := c.GetProjectInsights(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("GetProjectInsights returned unexpected error: %v", err)
	}

	expectedPath := "/insights/gh/owner/repo/workflows"
	if gotPath != expectedPath {
		t.Errorf("request path = %q; want %q", gotPath, expectedPath)
	}
	if gotToken != "my-secret-token" {
		t.Errorf("Circle-Token = %q; want %q", gotToken, "my-secret-token")
	}
}
