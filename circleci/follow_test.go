package circleci

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_FollowProject_Success(t *testing.T) {
	var gotMethod, gotPath, gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotToken = r.Header.Get("Circle-Token")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"followed": true}`))
	}))
	defer srv.Close()

	c := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()

	resp, err := c.FollowProject(context.Background(), "TouchBistro", "my-service")
	if err != nil {
		t.Fatalf("FollowProject returned unexpected error: %v", err)
	}

	if !resp.Followed {
		t.Errorf("Followed = %v; want true", resp.Followed)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("request method = %q; want %q", gotMethod, http.MethodPost)
	}

	expectedPath := "/project/github/TouchBistro/my-service/follow"
	if gotPath != expectedPath {
		t.Errorf("request path = %q; want %q", gotPath, expectedPath)
	}

	if gotToken != "test-token" {
		t.Errorf("Circle-Token = %q; want %q", gotToken, "test-token")
	}
}

func TestClient_FollowProject_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Unauthorized"}`))
	}))
	defer srv.Close()

	c := NewClientBuilder().
		WithToken("bad-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()

	resp, err := c.FollowProject(context.Background(), "TouchBistro", "my-service")
	if err == nil {
		t.Fatal("FollowProject returned nil error; want non-nil for 401 response")
	}
	if resp != nil {
		t.Errorf("FollowProject returned non-nil response on error; want nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "401") {
		t.Errorf("error message %q does not contain status code 401", errMsg)
	}
	if !strings.Contains(errMsg, "Unauthorized") {
		t.Errorf("error message %q does not contain body excerpt", errMsg)
	}
}

func TestClient_FollowProject_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not valid json`))
	}))
	defer srv.Close()

	c := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()

	resp, err := c.FollowProject(context.Background(), "TouchBistro", "my-service")
	if err == nil {
		t.Fatal("FollowProject returned nil error; want non-nil for invalid JSON")
	}
	if resp != nil {
		t.Errorf("FollowProject returned non-nil response on error; want nil")
	}
	if !strings.Contains(err.Error(), "decoding FollowProject response") {
		t.Errorf("error message %q does not mention decoding failure", err.Error())
	}
}

func TestClient_UnfollowProject_Success(t *testing.T) {
	var gotMethod, gotPath, gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotToken = r.Header.Get("Circle-Token")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"followed": false}`))
	}))
	defer srv.Close()

	c := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()

	resp, err := c.UnfollowProject(context.Background(), "TouchBistro", "my-service")
	if err != nil {
		t.Fatalf("UnfollowProject returned unexpected error: %v", err)
	}

	if resp.Followed {
		t.Errorf("Followed = %v; want false", resp.Followed)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("request method = %q; want %q", gotMethod, http.MethodPost)
	}

	expectedPath := "/project/github/TouchBistro/my-service/unfollow"
	if gotPath != expectedPath {
		t.Errorf("request path = %q; want %q", gotPath, expectedPath)
	}

	if gotToken != "test-token" {
		t.Errorf("Circle-Token = %q; want %q", gotToken, "test-token")
	}
}

func TestClient_UnfollowProject_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"Forbidden"}`))
	}))
	defer srv.Close()

	c := NewClientBuilder().
		WithToken("bad-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()

	resp, err := c.UnfollowProject(context.Background(), "TouchBistro", "my-service")
	if err == nil {
		t.Fatal("UnfollowProject returned nil error; want non-nil for 403 response")
	}
	if resp != nil {
		t.Errorf("UnfollowProject returned non-nil response on error; want nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "403") {
		t.Errorf("error message %q does not contain status code 403", errMsg)
	}
	if !strings.Contains(errMsg, "Forbidden") {
		t.Errorf("error message %q does not contain body excerpt", errMsg)
	}
}

func TestClient_UnfollowProject_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not valid json`))
	}))
	defer srv.Close()

	c := NewClientBuilder().
		WithToken("test-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()

	resp, err := c.UnfollowProject(context.Background(), "TouchBistro", "my-service")
	if err == nil {
		t.Fatal("UnfollowProject returned nil error; want non-nil for invalid JSON")
	}
	if resp != nil {
		t.Errorf("UnfollowProject returned non-nil response on error; want nil")
	}
	if !strings.Contains(err.Error(), "decoding UnfollowProject response") {
		t.Errorf("error message %q does not mention decoding failure", err.Error())
	}
}
