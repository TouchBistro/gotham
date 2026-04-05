package circleci

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_GetProject_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"slug": "gh/TouchBistro/my-service",
			"name": "my-service",
			"id": "proj-123",
			"organization_name": "TouchBistro",
			"organization_slug": "gh/TouchBistro",
			"organization_id": "org-456",
			"vcs_info": {
				"vcs_url": "https://github.com/TouchBistro/my-service",
				"provider": "GitHub",
				"default_branch": "main"
			}
		}`))
	}))
	defer srv.Close()

	c := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()

	project, err := c.GetProject(context.Background(), "TouchBistro", "my-service")
	if err != nil {
		t.Fatalf("GetProject returned unexpected error: %v", err)
	}

	if project.Slug != "gh/TouchBistro/my-service" {
		t.Errorf("Slug = %q; want %q", project.Slug, "gh/TouchBistro/my-service")
	}
	if project.Name != "my-service" {
		t.Errorf("Name = %q; want %q", project.Name, "my-service")
	}
	if project.ID != "proj-123" {
		t.Errorf("ID = %q; want %q", project.ID, "proj-123")
	}
	if project.OrganizationName != "TouchBistro" {
		t.Errorf("OrganizationName = %q; want %q", project.OrganizationName, "TouchBistro")
	}
	if project.OrganizationSlug != "gh/TouchBistro" {
		t.Errorf("OrganizationSlug = %q; want %q", project.OrganizationSlug, "gh/TouchBistro")
	}
	if project.OrganizationID != "org-456" {
		t.Errorf("OrganizationID = %q; want %q", project.OrganizationID, "org-456")
	}
	if project.VCSInfo.VCSURL != "https://github.com/TouchBistro/my-service" {
		t.Errorf("VCSInfo.VCSURL = %q; want %q", project.VCSInfo.VCSURL, "https://github.com/TouchBistro/my-service")
	}
	if project.VCSInfo.Provider != "GitHub" {
		t.Errorf("VCSInfo.Provider = %q; want %q", project.VCSInfo.Provider, "GitHub")
	}
	if project.VCSInfo.DefaultBranch != "main" {
		t.Errorf("VCSInfo.DefaultBranch = %q; want %q", project.VCSInfo.DefaultBranch, "main")
	}
}

func TestClient_GetProject_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Project not found"}`))
	}))
	defer srv.Close()

	c := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()

	project, err := c.GetProject(context.Background(), "TouchBistro", "nonexistent")
	if err == nil {
		t.Fatal("GetProject returned nil error; want non-nil for 404 response")
	}
	if project != nil {
		t.Errorf("GetProject returned non-nil project on error; want nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "404") {
		t.Errorf("error message %q does not contain status code 404", errMsg)
	}
	if !strings.Contains(errMsg, "Project not found") {
		t.Errorf("error message %q does not contain body excerpt", errMsg)
	}
}

func TestClient_GetProject_VerifiesRequestPathAndAuth(t *testing.T) {
	var gotPath, gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("Circle-Token")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"slug":"gh/owner/repo","name":"repo","id":"1"}`))
	}))
	defer srv.Close()

	c := NewClientBuilder().
		WithToken("my-secret-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()

	_, err := c.GetProject(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("GetProject returned unexpected error: %v", err)
	}

	expectedPath := "/project/gh/owner/repo"
	if gotPath != expectedPath {
		t.Errorf("request path = %q; want %q", gotPath, expectedPath)
	}
	if gotToken != "my-secret-token" {
		t.Errorf("Circle-Token = %q; want %q", gotToken, "my-secret-token")
	}
}

