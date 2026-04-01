package shipit

import (
	"net/http"
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
