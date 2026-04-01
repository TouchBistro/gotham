package shipit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestNewClient_StoresBaseURI verifies that NewClient stores the base URI.
func TestNewClient_StoresBaseURI(t *testing.T) {
	baseURI := "https://shipit.example.com"
	c := NewClient(baseURI, "secret")

	if c.baseURI != baseURI {
		t.Errorf("baseURI = %q; want %q", c.baseURI, baseURI)
	}
}

// TestNewClient_StoresAPIPassword verifies that NewClient stores the API password.
func TestNewClient_StoresAPIPassword(t *testing.T) {
	apiPassword := "s3cr3t-p4ssword"
	c := NewClient("https://shipit.example.com", apiPassword)

	if c.apiPassword != apiPassword {
		t.Errorf("apiPassword = %q; want %q", c.apiPassword, apiPassword)
	}
}

// TestNewClient_HasHTTPClient verifies that NewClient initialises a non-nil http.Client.
func TestNewClient_HasHTTPClient(t *testing.T) {
	c := NewClient("https://shipit.example.com", "secret")

	if c.httpClient == nil {
		t.Fatal("httpClient = nil; want non-nil *http.Client")
	}
}

// TestClient_SetAuth verifies that setAuth sets the correct Basic Auth header
// on a request — empty username and the configured API password.
func TestClient_SetAuth(t *testing.T) {
	apiPassword := "my-api-password"
	c := NewClient("https://shipit.example.com", apiPassword)

	req, err := http.NewRequest(http.MethodGet, "https://shipit.example.com/api/stacks", nil)
	if err != nil {
		t.Fatalf("http.NewRequest returned unexpected error: %v", err)
	}

	c.setAuth(req)

	username, password, ok := req.BasicAuth()
	if !ok {
		t.Fatal("req.BasicAuth() = (_, _, false); want (_, _, true) — Basic Auth header not set")
	}
	if username != "" {
		t.Errorf("BasicAuth username = %q; want empty string", username)
	}
	if password != apiPassword {
		t.Errorf("BasicAuth password = %q; want %q", password, apiPassword)
	}
}

// TestClient_SetAuth_EmptyPassword verifies that setAuth still sets Basic Auth
// correctly when the API password is an empty string.
func TestClient_SetAuth_EmptyPassword(t *testing.T) {
	c := NewClient("https://shipit.example.com", "")

	req, err := http.NewRequest(http.MethodGet, "https://shipit.example.com/api/stacks", nil)
	if err != nil {
		t.Fatalf("http.NewRequest returned unexpected error: %v", err)
	}

	c.setAuth(req)

	username, password, ok := req.BasicAuth()
	if !ok {
		t.Fatal("req.BasicAuth() = (_, _, false); want (_, _, true) — Basic Auth header not set")
	}
	if username != "" {
		t.Errorf("BasicAuth username = %q; want empty string", username)
	}
	if password != "" {
		t.Errorf("BasicAuth password = %q; want empty string", password)
	}
}

// --- ListAllStacks tests ---

// makeStack is a test helper that returns a Stack with the given id and repo name set.
func makeStack(id int, repoName string) Stack {
	return Stack{
		ID:          id,
		RepoOwner:   "touchbistro",
		RepoName:    repoName,
		Environment: "production",
	}
}

// TestListAllStacks_SinglePage verifies that a single-page response (no Link header)
// returns all stacks and no error.
func TestListAllStacks_SinglePage(t *testing.T) {
	stacks := []Stack{makeStack(1, "repo-a"), makeStack(2, "repo-b")}
	body, _ := json.Marshal(stacks)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	got, err := c.ListAllStacks()
	if err != nil {
		t.Fatalf("ListAllStacks() error = %v; want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListAllStacks() returned %d stacks; want 2", len(got))
	}
	if got[0].RepoName != "repo-a" || got[1].RepoName != "repo-b" {
		t.Errorf("ListAllStacks() stacks = %+v; unexpected values", got)
	}
}

// TestListAllStacks_MultiplePages verifies that pagination is followed when the
// first response contains a Link header with rel=next.
func TestListAllStacks_MultiplePages(t *testing.T) {
	page1 := []Stack{makeStack(1, "repo-a")}
	page2 := []Stack{makeStack(2, "repo-b")}

	body1, _ := json.Marshal(page1)
	body2, _ := json.Marshal(page2)

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			// Provide a rel=next link with since=2
			w.Header().Set("Link", `</api/stacks?page_size=50&since=2>; rel="next"`)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body1)
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body2)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	got, err := c.ListAllStacks()
	if err != nil {
		t.Fatalf("ListAllStacks() error = %v; want nil", err)
	}
	if callCount != 2 {
		t.Errorf("server called %d times; want 2", callCount)
	}
	if len(got) != 2 {
		t.Fatalf("ListAllStacks() returned %d stacks; want 2", len(got))
	}
	if got[0].RepoName != "repo-a" || got[1].RepoName != "repo-b" {
		t.Errorf("ListAllStacks() stacks = %+v; unexpected values", got)
	}
}

// TestListAllStacks_HTTPError verifies that a non-2xx response is returned as an error.
func TestListAllStacks_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	_, err := c.ListAllStacks()
	if err == nil {
		t.Fatal("ListAllStacks() error = nil; want non-nil error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("ListAllStacks() error = %v; want error containing status code 500", err)
	}
}

// TestListAllStacks_EmptyResponse verifies that a 200 response with an empty JSON
// array returns an empty slice and no error.
func TestListAllStacks_EmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "[]")
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	got, err := c.ListAllStacks()
	if err != nil {
		t.Fatalf("ListAllStacks() error = %v; want nil", err)
	}
	if len(got) != 0 {
		t.Errorf("ListAllStacks() returned %d stacks; want 0", len(got))
	}
}
