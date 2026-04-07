package circleci

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_ListPipelines_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"items": [
				{"id": "pipe-1", "state": "created", "number": 42, "created_at": "2026-01-01T00:00:00Z"},
				{"id": "pipe-2", "state": "errored", "number": 43, "created_at": "2026-01-02T00:00:00Z"}
			],
			"next_page_token": "token-abc"
		}`))
	}))
	defer srv.Close()

	c := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()

	resp, err := c.ListPipelines(context.Background(), "TouchBistro", "my-service")
	if err != nil {
		t.Fatalf("ListPipelines returned unexpected error: %v", err)
	}

	if len(resp.Items) != 2 {
		t.Fatalf("len(Items) = %d; want 2", len(resp.Items))
	}
	if resp.Items[0].ID != "pipe-1" {
		t.Errorf("Items[0].ID = %q; want %q", resp.Items[0].ID, "pipe-1")
	}
	if resp.Items[0].State != "created" {
		t.Errorf("Items[0].State = %q; want %q", resp.Items[0].State, "created")
	}
	if resp.Items[0].Number != 42 {
		t.Errorf("Items[0].Number = %d; want 42", resp.Items[0].Number)
	}
	if resp.Items[1].ID != "pipe-2" {
		t.Errorf("Items[1].ID = %q; want %q", resp.Items[1].ID, "pipe-2")
	}
	if resp.NextPageToken != "token-abc" {
		t.Errorf("NextPageToken = %q; want %q", resp.NextPageToken, "token-abc")
	}
}

func TestClient_ListPipelines_EmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items": [], "next_page_token": ""}`))
	}))
	defer srv.Close()

	c := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()

	resp, err := c.ListPipelines(context.Background(), "TouchBistro", "my-service")
	if err != nil {
		t.Fatalf("ListPipelines returned unexpected error: %v", err)
	}

	if len(resp.Items) != 0 {
		t.Errorf("len(Items) = %d; want 0", len(resp.Items))
	}
	if resp.NextPageToken != "" {
		t.Errorf("NextPageToken = %q; want empty string", resp.NextPageToken)
	}
}

func TestClient_ListPipelines_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"Internal Server Error"}`))
	}))
	defer srv.Close()

	c := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()

	resp, err := c.ListPipelines(context.Background(), "TouchBistro", "my-service")
	if err == nil {
		t.Fatal("ListPipelines returned nil error; want non-nil for 500 response")
	}
	if resp != nil {
		t.Errorf("ListPipelines returned non-nil response on error; want nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "500") {
		t.Errorf("error message %q does not contain status code 500", errMsg)
	}
	if !strings.Contains(errMsg, "Internal Server Error") {
		t.Errorf("error message %q does not contain body excerpt", errMsg)
	}
}

func TestClient_ListPipelines_VerifiesRequestPathAndAuth(t *testing.T) {
	var gotPath, gotToken, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("Circle-Token")
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items": [], "next_page_token": ""}`))
	}))
	defer srv.Close()

	c := NewClientBuilder().
		WithToken("my-secret-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()

	_, err := c.ListPipelines(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("ListPipelines returned unexpected error: %v", err)
	}

	expectedPath := "/project/gh/owner/repo/pipeline"
	if gotPath != expectedPath {
		t.Errorf("request path = %q; want %q", gotPath, expectedPath)
	}
	if gotToken != "my-secret-token" {
		t.Errorf("Circle-Token = %q; want %q", gotToken, "my-secret-token")
	}
	if gotMethod != http.MethodGet {
		t.Errorf("request method = %q; want %q", gotMethod, http.MethodGet)
	}
}
