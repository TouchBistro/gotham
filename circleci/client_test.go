package circleci

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClientBuilder_DefaultToken(t *testing.T) {
	c, err := NewClientBuilder().
		WithToken("test-token-123").
		Build()
	if err != nil {
		t.Fatalf("Build returned unexpected error: %v", err)
	}

	if c.token != "test-token-123" {
		t.Errorf("token = %q; want %q", c.token, "test-token-123")
	}
}

func TestNewClientBuilder_Defaults(t *testing.T) {
	c, err := NewClientBuilder().
		WithToken("tok").
		Build()
	if err != nil {
		t.Fatalf("Build returned unexpected error: %v", err)
	}

	if c.v1BaseURL != "https://circleci.com/api/v1.1" {
		t.Errorf("v1BaseURL = %q; want %q", c.v1BaseURL, "https://circleci.com/api/v1.1")
	}
	if c.v2BaseURL != "https://circleci.com/api/v2" {
		t.Errorf("v2BaseURL = %q; want %q", c.v2BaseURL, "https://circleci.com/api/v2")
	}
	if c.httpClient == nil {
		t.Fatal("httpClient is nil; want non-nil")
	}
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("httpClient.Timeout = %v; want %v", c.httpClient.Timeout, 30*time.Second)
	}
}

func TestNewClientBuilder_CustomHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 60 * time.Second}
	c, err := NewClientBuilder().
		WithToken("tok").
		WithHTTPClient(custom).
		Build()
	if err != nil {
		t.Fatalf("Build returned unexpected error: %v", err)
	}

	if c.httpClient != custom {
		t.Error("httpClient is not the custom client provided to WithHTTPClient")
	}
}

func TestNewClientBuilder_CustomBaseURLs(t *testing.T) {
	c, err := NewClientBuilder().
		WithToken("tok").
		WithBaseURLs("http://v1.local", "http://v2.local").
		Build()
	if err != nil {
		t.Fatalf("Build returned unexpected error: %v", err)
	}

	if c.v1BaseURL != "http://v1.local" {
		t.Errorf("v1BaseURL = %q; want %q", c.v1BaseURL, "http://v1.local")
	}
	if c.v2BaseURL != "http://v2.local" {
		t.Errorf("v2BaseURL = %q; want %q", c.v2BaseURL, "http://v2.local")
	}
}

func TestNewClientBuilder_EmptyTokenError(t *testing.T) {
	_, err := NewClientBuilder().Build()
	if err == nil {
		t.Fatal("Build returned nil error; want non-nil for empty token")
	}
	if !strings.Contains(err.Error(), "token") {
		t.Errorf("error message %q does not mention token", err.Error())
	}
}

func TestClient_doRequest_SetsCircleTokenHeader(t *testing.T) {
	var gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("Circle-Token")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c, err := NewClientBuilder().
		WithToken("secret-token").
		WithBaseURLs(srv.URL, srv.URL).
		Build()
	if err != nil {
		t.Fatalf("Build returned unexpected error: %v", err)
	}

	_, err = c.doRequest(context.Background(), http.MethodGet, srv.URL+"/test", nil)
	if err != nil {
		t.Fatalf("doRequest returned unexpected error: %v", err)
	}

	if gotToken != "secret-token" {
		t.Errorf("Circle-Token header = %q; want %q", gotToken, "secret-token")
	}
}

func TestClient_doRequest_SetsAcceptHeader(t *testing.T) {
	var gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, err := NewClientBuilder().
		WithToken("tok").
		WithBaseURLs(srv.URL, srv.URL).
		Build()
	if err != nil {
		t.Fatalf("Build returned unexpected error: %v", err)
	}

	_, err = c.doRequest(context.Background(), http.MethodGet, srv.URL+"/test", nil)
	if err != nil {
		t.Fatalf("doRequest returned unexpected error: %v", err)
	}

	if gotAccept != "application/json" {
		t.Errorf("Accept header = %q; want %q", gotAccept, "application/json")
	}
}

func TestClient_doRequest_ReturnsBodyOnSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"ok"}`))
	}))
	defer srv.Close()

	c, err := NewClientBuilder().
		WithToken("tok").
		WithBaseURLs(srv.URL, srv.URL).
		Build()
	if err != nil {
		t.Fatalf("Build returned unexpected error: %v", err)
	}

	body, err := c.doRequest(context.Background(), http.MethodGet, srv.URL+"/test", nil)
	if err != nil {
		t.Fatalf("doRequest returned unexpected error: %v", err)
	}

	if string(body) != `{"result":"ok"}` {
		t.Errorf("body = %q; want %q", string(body), `{"result":"ok"}`)
	}
}

func TestClient_doRequest_ErrorOnNon2xx(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{"404 Not Found", http.StatusNotFound, `{"message":"not found"}`},
		{"500 Internal Server Error", http.StatusInternalServerError, `{"message":"server error"}`},
		{"403 Forbidden", http.StatusForbidden, `{"message":"forbidden"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c, err := NewClientBuilder().
				WithToken("tok").
				WithBaseURLs(srv.URL, srv.URL).
				Build()
			if err != nil {
				t.Fatalf("Build returned unexpected error: %v", err)
			}

			_, err = c.doRequest(context.Background(), http.MethodGet, srv.URL+"/test", nil)
			if err == nil {
				t.Fatal("doRequest returned nil error; want non-nil for non-2xx status")
			}

			errMsg := err.Error()
			if !strings.Contains(errMsg, strings.TrimSpace(tt.body[:min(len(tt.body), 50)])) {
				t.Errorf("error message %q does not contain body excerpt", errMsg)
			}
		})
	}
}

func TestClient_doRequest_PropagatesContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow response to ensure context cancellation takes effect
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, err := NewClientBuilder().
		WithToken("tok").
		WithBaseURLs(srv.URL, srv.URL).
		Build()
	if err != nil {
		t.Fatalf("Build returned unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = c.doRequest(ctx, http.MethodGet, srv.URL+"/test", nil)
	if err == nil {
		t.Fatal("doRequest returned nil error; want context cancellation error")
	}
}

func TestClient_doRequest_SendsRequestBody(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, err := NewClientBuilder().
		WithToken("tok").
		WithBaseURLs(srv.URL, srv.URL).
		Build()
	if err != nil {
		t.Fatalf("Build returned unexpected error: %v", err)
	}

	_, err = c.doRequest(context.Background(), http.MethodPost, srv.URL+"/test", strings.NewReader(`{"key":"val"}`))
	if err != nil {
		t.Fatalf("doRequest returned unexpected error: %v", err)
	}

	if gotBody != `{"key":"val"}` {
		t.Errorf("request body = %q; want %q", gotBody, `{"key":"val"}`)
	}
}
