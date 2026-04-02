package shipit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	got, err := c.ListAllStacks(context.Background())
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
	got, err := c.ListAllStacks(context.Background())
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
	_, err := c.ListAllStacks(context.Background())
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
	got, err := c.ListAllStacks(context.Background())
	if err != nil {
		t.Fatalf("ListAllStacks() error = %v; want nil", err)
	}
	if len(got) != 0 {
		t.Errorf("ListAllStacks() returned %d stacks; want 0", len(got))
	}
}

// TestListAllStacks_NetworkError verifies that a network-level failure (server closed
// before response) is returned as a wrapped error.
func TestListAllStacks_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hijack the connection and close it immediately to provoke a network error.
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijack not supported", http.StatusInternalServerError)
			return
		}
		conn, _, _ := hj.Hijack()
		_ = conn.Close()
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	_, err := c.ListAllStacks(context.Background())
	if err == nil {
		t.Fatal("ListAllStacks() error = nil; want non-nil error on network failure")
	}
}

// TestListAllStacks_InvalidJSON verifies that malformed JSON in the response body
// is returned as a wrapped error.
func TestListAllStacks_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `not-valid-json`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	_, err := c.ListAllStacks(context.Background())
	if err == nil {
		t.Fatal("ListAllStacks() error = nil; want non-nil error on invalid JSON")
	}
}

// TestListAllStacks_InvalidBaseURI verifies that an invalid base URI that prevents
// request creation is returned as an error.
func TestListAllStacks_InvalidBaseURI(t *testing.T) {
	c := NewClient("://invalid-uri", "secret")
	_, err := c.ListAllStacks(context.Background())
	if err == nil {
		t.Fatal("ListAllStacks() error = nil; want non-nil error for invalid base URI")
	}
}

// TestListAllStacks_CancelledContext verifies that a cancelled context aborts the request.
func TestListAllStacks_CancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "[]")
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	c := NewClient(srv.URL, "secret")
	_, err := c.ListAllStacks(ctx)
	if err == nil {
		t.Fatal("ListAllStacks() error = nil; want non-nil error for cancelled context")
	}
}

// --- LockStack tests ---

// TestLockStack_Success verifies that LockStack sends a correct POST request and
// returns no error on a 200 response.
func TestLockStack_Success(t *testing.T) {
	const stackID = "touchbistro/repo-a/production"
	const reason = "deploying hotfix"

	var gotMethod, gotPath, gotBody, gotContentType string
	var gotUser, gotPass string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		gotUser, gotPass, _ = r.BasicAuth()

		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	err := c.LockStack(context.Background(), stackID, reason)
	if err != nil {
		t.Fatalf("LockStack() error = %v; want nil", err)
	}

	wantPath := "/api/stacks/" + stackID + "/lock"
	if gotMethod != http.MethodPost {
		t.Errorf("request method = %q; want POST", gotMethod)
	}
	if gotPath != wantPath {
		t.Errorf("request path = %q; want %q", gotPath, wantPath)
	}
	if !strings.Contains(gotContentType, "application/json") {
		t.Errorf("Content-Type = %q; want application/json", gotContentType)
	}
	if gotUser != "" || gotPass != "secret" {
		t.Errorf("BasicAuth = (%q, %q); want (\"\", \"secret\")", gotUser, gotPass)
	}
	wantBody := `{"reason":"deploying hotfix"}`
	if gotBody != wantBody {
		t.Errorf("request body = %q; want %q", gotBody, wantBody)
	}
}

// TestLockStack_Error verifies that LockStack returns an error containing the
// status code and body when the server responds with 422.
func TestLockStack_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = fmt.Fprint(w, `{"error":"stack already locked"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	err := c.LockStack(context.Background(), "touchbistro/repo-a/production", "reason")
	if err == nil {
		t.Fatal("LockStack() error = nil; want non-nil error for 422 response")
	}
	if !strings.Contains(err.Error(), "422") {
		t.Errorf("LockStack() error = %v; want error containing status code 422", err)
	}
	if !strings.Contains(err.Error(), "stack already locked") {
		t.Errorf("LockStack() error = %v; want error containing response body", err)
	}
}

// TestLockStack_NetworkError verifies that a network-level failure during LockStack
// is returned as a wrapped error.
func TestLockStack_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijack not supported", http.StatusInternalServerError)
			return
		}
		conn, _, _ := hj.Hijack()
		_ = conn.Close()
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	err := c.LockStack(context.Background(), "touchbistro/repo-a/production", "reason")
	if err == nil {
		t.Fatal("LockStack() error = nil; want non-nil error on network failure")
	}
}

// TestLockStack_InvalidBaseURI verifies that an invalid base URI that prevents
// request creation is returned as an error.
func TestLockStack_InvalidBaseURI(t *testing.T) {
	c := NewClient("://invalid-uri", "secret")
	err := c.LockStack(context.Background(), "touchbistro/repo-a/production", "reason")
	if err == nil {
		t.Fatal("LockStack() error = nil; want non-nil error for invalid base URI")
	}
}

// TestLockStack_InvalidStackID verifies that LockStack rejects stack IDs that
// do not match the expected owner/repo/environment format.
func TestLockStack_InvalidStackID(t *testing.T) {
	c := NewClient("https://shipit.example.com", "secret")

	tests := []struct {
		name    string
		stackID string
	}{
		{"empty string", ""},
		{"single segment", "repo-a"},
		{"two segments", "owner/repo-a"},
		{"four segments", "owner/repo-a/production/extra"},
		{"contains query char", "owner/repo-a/prod?x=1"},
		{"contains hash", "owner/repo-a/prod#frag"},
		{"contains space", "owner/repo a/production"},
		{"empty segment", "owner//production"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.LockStack(context.Background(), tt.stackID, "reason")
			if err == nil {
				t.Fatalf("LockStack(%q) error = nil; want invalid stack ID error", tt.stackID)
			}
			if !strings.Contains(err.Error(), "invalid stack ID") {
				t.Errorf("LockStack(%q) error = %v; want error containing 'invalid stack ID'", tt.stackID, err)
			}
		})
	}
}

// --- UnlockStack tests ---

// TestUnlockStack_Success verifies that UnlockStack sends a correct DELETE request
// and returns no error on a 204 response.
func TestUnlockStack_Success(t *testing.T) {
	const stackID = "touchbistro/repo-a/production"

	var gotMethod, gotPath string
	var gotUser, gotPass string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotUser, gotPass, _ = r.BasicAuth()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	err := c.UnlockStack(context.Background(), stackID)
	if err != nil {
		t.Fatalf("UnlockStack() error = %v; want nil", err)
	}

	wantPath := "/api/stacks/" + stackID + "/lock"
	if gotMethod != http.MethodDelete {
		t.Errorf("request method = %q; want DELETE", gotMethod)
	}
	if gotPath != wantPath {
		t.Errorf("request path = %q; want %q", gotPath, wantPath)
	}
	if gotUser != "" || gotPass != "secret" {
		t.Errorf("BasicAuth = (%q, %q); want (\"\", \"secret\")", gotUser, gotPass)
	}
}

// TestUnlockStack_Error verifies that UnlockStack returns an error with context
// when the server responds with 404.
func TestUnlockStack_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"error":"stack not found"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	err := c.UnlockStack(context.Background(), "touchbistro/repo-a/production")
	if err == nil {
		t.Fatal("UnlockStack() error = nil; want non-nil error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("UnlockStack() error = %v; want error containing status code 404", err)
	}
	if !strings.Contains(err.Error(), "stack not found") {
		t.Errorf("UnlockStack() error = %v; want error containing response body", err)
	}
}

// TestUnlockStack_NetworkError verifies that a network-level failure during UnlockStack
// is returned as a wrapped error.
func TestUnlockStack_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijack not supported", http.StatusInternalServerError)
			return
		}
		conn, _, _ := hj.Hijack()
		_ = conn.Close()
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	err := c.UnlockStack(context.Background(), "touchbistro/repo-a/production")
	if err == nil {
		t.Fatal("UnlockStack() error = nil; want non-nil error on network failure")
	}
}

// TestUnlockStack_InvalidBaseURI verifies that an invalid base URI that prevents
// request creation is returned as an error.
func TestUnlockStack_InvalidBaseURI(t *testing.T) {
	c := NewClient("://invalid-uri", "secret")
	err := c.UnlockStack(context.Background(), "touchbistro/repo-a/production")
	if err == nil {
		t.Fatal("UnlockStack() error = nil; want non-nil error for invalid base URI")
	}
}

// TestUnlockStack_InvalidStackID verifies that UnlockStack rejects stack IDs that
// do not match the expected owner/repo/environment format.
func TestUnlockStack_InvalidStackID(t *testing.T) {
	c := NewClient("https://shipit.example.com", "secret")

	tests := []struct {
		name    string
		stackID string
	}{
		{"empty string", ""},
		{"single segment", "repo-a"},
		{"two segments", "owner/repo-a"},
		{"four segments", "owner/repo-a/production/extra"},
		{"contains query char", "owner/repo-a/prod?x=1"},
		{"contains hash", "owner/repo-a/prod#frag"},
		{"contains space", "owner/repo a/production"},
		{"empty segment", "owner//production"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.UnlockStack(context.Background(), tt.stackID)
			if err == nil {
				t.Fatalf("UnlockStack(%q) error = nil; want invalid stack ID error", tt.stackID)
			}
			if !strings.Contains(err.Error(), "invalid stack ID") {
				t.Errorf("UnlockStack(%q) error = %v; want error containing 'invalid stack ID'", tt.stackID, err)
			}
		})
	}
}

// --- parseLinkNextSince tests ---

// --- LockAll tests ---

// TestLockAll_Success verifies that LockAll sends a lock request to every stack
// returned by ListAllStacks and returns no error when all succeed.
func TestLockAll_Success(t *testing.T) {
	stacks := []Stack{
		makeStack(1, "repo-a"),
		makeStack(2, "repo-b"),
	}
	stacksBody, _ := json.Marshal(stacks)

	lockedIDs := make(map[string]int)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(stacksBody)
			return
		}
		// POST /api/stacks/{stack_id}/lock
		lockedIDs[r.URL.Path]++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	err := c.LockAll(context.Background(), "maintenance window")
	if err != nil {
		t.Fatalf("LockAll() error = %v; want nil", err)
	}

	for _, s := range stacks {
		wantPath := "/api/stacks/" + s.StackID() + "/lock"
		if lockedIDs[wantPath] != 1 {
			t.Errorf("stack %q: lock request count = %d; want 1", s.StackID(), lockedIDs[wantPath])
		}
	}
}

// TestLockAll_OneStackFails verifies that LockAll returns an error when one stack
// fails to lock.
func TestLockAll_OneStackFails(t *testing.T) {
	stacks := []Stack{
		makeStack(1, "repo-a"),
		makeStack(2, "repo-b"),
	}
	stacksBody, _ := json.Marshal(stacks)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(stacksBody)
			return
		}
		// Fail the lock for repo-b
		if strings.Contains(r.URL.Path, "repo-b") {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = fmt.Fprint(w, `{"error":"already locked"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	err := c.LockAll(context.Background(), "reason")
	if err == nil {
		t.Fatal("LockAll() error = nil; want non-nil error when a stack fails to lock")
	}
}

// TestLockAll_MultipleStacksFail verifies that LockAll collects errors from all
// failed stacks rather than returning only the first failure.
func TestLockAll_MultipleStacksFail(t *testing.T) {
	stacks := []Stack{
		makeStack(1, "repo-a"),
		makeStack(2, "repo-b"),
		makeStack(3, "repo-c"),
	}
	stacksBody, _ := json.Marshal(stacks)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(stacksBody)
			return
		}
		// Fail repo-a and repo-c, succeed repo-b
		if strings.Contains(r.URL.Path, "repo-a") || strings.Contains(r.URL.Path, "repo-c") {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = fmt.Fprint(w, `{"error":"already locked"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	err := c.LockAll(context.Background(), "reason")
	if err == nil {
		t.Fatal("LockAll() error = nil; want non-nil error when multiple stacks fail")
	}

	unwrapped := errors.Unwrap(err)
	if unwrapped != nil {
		// errors.Join returns a type whose Unwrap returns []error; verify via the joined string
		t.Logf("LockAll() error = %v", err)
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "repo-a") {
		t.Errorf("LockAll() error missing repo-a failure: %v", err)
	}
	if !strings.Contains(errStr, "repo-c") {
		t.Errorf("LockAll() error missing repo-c failure: %v", err)
	}
}

// TestLockAll_ZeroStacks verifies that LockAll returns no error and makes no lock
// requests when ListAllStacks returns an empty slice.
func TestLockAll_ZeroStacks(t *testing.T) {
	lockCallCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "[]")
			return
		}
		lockCallCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	err := c.LockAll(context.Background(), "reason")
	if err != nil {
		t.Fatalf("LockAll() error = %v; want nil for zero stacks", err)
	}
	if lockCallCount != 0 {
		t.Errorf("lock request count = %d; want 0 for zero stacks", lockCallCount)
	}
}

// --- UnlockAll tests ---

// TestUnlockAll_Success verifies that UnlockAll sends an unlock request to every
// stack returned by ListAllStacks and returns no error when all succeed.
func TestUnlockAll_Success(t *testing.T) {
	stacks := []Stack{
		makeStack(1, "repo-a"),
		makeStack(2, "repo-b"),
	}
	stacksBody, _ := json.Marshal(stacks)

	unlockedIDs := make(map[string]int)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(stacksBody)
			return
		}
		// DELETE /api/stacks/{stack_id}/lock
		unlockedIDs[r.URL.Path]++
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	err := c.UnlockAll(context.Background())
	if err != nil {
		t.Fatalf("UnlockAll() error = %v; want nil", err)
	}

	for _, s := range stacks {
		wantPath := "/api/stacks/" + s.StackID() + "/lock"
		if unlockedIDs[wantPath] != 1 {
			t.Errorf("stack %q: unlock request count = %d; want 1", s.StackID(), unlockedIDs[wantPath])
		}
	}
}

// TestUnlockAll_OneStackFails verifies that UnlockAll returns an error when one
// stack fails to unlock.
func TestUnlockAll_OneStackFails(t *testing.T) {
	stacks := []Stack{
		makeStack(1, "repo-a"),
		makeStack(2, "repo-b"),
	}
	stacksBody, _ := json.Marshal(stacks)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(stacksBody)
			return
		}
		// Fail the unlock for repo-b
		if strings.Contains(r.URL.Path, "repo-b") {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"error":"stack not found"}`)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	err := c.UnlockAll(context.Background())
	if err == nil {
		t.Fatal("UnlockAll() error = nil; want non-nil error when a stack fails to unlock")
	}
}

// TestUnlockAll_MultipleStacksFail verifies that UnlockAll collects errors from all
// failed stacks rather than returning only the first failure.
func TestUnlockAll_MultipleStacksFail(t *testing.T) {
	stacks := []Stack{
		makeStack(1, "repo-a"),
		makeStack(2, "repo-b"),
		makeStack(3, "repo-c"),
	}
	stacksBody, _ := json.Marshal(stacks)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(stacksBody)
			return
		}
		// Fail repo-a and repo-c, succeed repo-b
		if strings.Contains(r.URL.Path, "repo-a") || strings.Contains(r.URL.Path, "repo-c") {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"error":"stack not found"}`)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	err := c.UnlockAll(context.Background())
	if err == nil {
		t.Fatal("UnlockAll() error = nil; want non-nil error when multiple stacks fail")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "repo-a") {
		t.Errorf("UnlockAll() error missing repo-a failure: %v", err)
	}
	if !strings.Contains(errStr, "repo-c") {
		t.Errorf("UnlockAll() error missing repo-c failure: %v", err)
	}
}

// TestUnlockAll_ZeroStacks verifies that UnlockAll returns no error and makes no
// unlock requests when ListAllStacks returns an empty slice.
func TestUnlockAll_ZeroStacks(t *testing.T) {
	unlockCallCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "[]")
			return
		}
		unlockCallCount++
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret")
	err := c.UnlockAll(context.Background())
	if err != nil {
		t.Fatalf("UnlockAll() error = %v; want nil for zero stacks", err)
	}
	if unlockCallCount != 0 {
		t.Errorf("unlock request count = %d; want 0 for zero stacks", unlockCallCount)
	}
}

// --- parseLinkNextSince tests ---

// TestParseLinkNextSince_NoNext verifies that an empty string is returned when
// there is no rel=next in the Link header.
func TestParseLinkNextSince_NoNext(t *testing.T) {
	result := parseLinkNextSince(`</api/stacks?page_size=50>; rel="prev"`)
	if result != "" {
		t.Errorf("parseLinkNextSince() = %q; want empty string", result)
	}
}

// TestParseLinkNextSince_WithNext verifies that the since value is extracted from
// a Link header containing rel=next.
func TestParseLinkNextSince_WithNext(t *testing.T) {
	result := parseLinkNextSince(`</api/stacks?page_size=50&since=42>; rel="next"`)
	if result != "42" {
		t.Errorf("parseLinkNextSince() = %q; want %q", result, "42")
	}
}

// TestParseLinkNextSince_EmptyHeader verifies that an empty header returns empty string.
func TestParseLinkNextSince_EmptyHeader(t *testing.T) {
	result := parseLinkNextSince("")
	if result != "" {
		t.Errorf("parseLinkNextSince() = %q; want empty string", result)
	}
}
