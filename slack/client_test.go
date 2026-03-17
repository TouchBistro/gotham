package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestNewClient_FieldsSet verifies that NewClient correctly assigns the
// supplied string values to the Client struct's pointer fields.
func TestNewClient_FieldsSet(t *testing.T) {
	botToken := "xoxb-test-token"
	webhookURL := "https://hooks.slack.com/services/TEST"
	channelID := "C0123456"

	client := NewClient(botToken, webhookURL, channelID)

	if client.BotToken == nil {
		t.Fatal("NewClient().BotToken = nil; want non-nil")
	}
	if *client.BotToken != botToken {
		t.Errorf("*BotToken = %q; want %q", *client.BotToken, botToken)
	}

	if client.WebhookURL == nil {
		t.Fatal("NewClient().WebhookURL = nil; want non-nil")
	}
	if *client.WebhookURL != webhookURL {
		t.Errorf("*WebhookURL = %q; want %q", *client.WebhookURL, webhookURL)
	}

	if client.DefaultChannelID == nil {
		t.Fatal("NewClient().DefaultChannelID = nil; want non-nil")
	}
	if *client.DefaultChannelID != channelID {
		t.Errorf("*DefaultChannelID = %q; want %q", *client.DefaultChannelID, channelID)
	}
}

// TestNewClient_EmptyStrings verifies that NewClient works with empty string values.
func TestNewClient_EmptyStrings(t *testing.T) {
	client := NewClient("", "", "")

	if client.BotToken == nil {
		t.Fatal("NewClient(\"\", ...).BotToken = nil; want non-nil")
	}
	if *client.BotToken != "" {
		t.Errorf("*BotToken = %q; want empty string", *client.BotToken)
	}
}

// TestClient_ToStringPtr verifies that ToStringPtr on a Client delegates
// to the unexported helper and returns a pointer to the given value.
func TestClient_ToStringPtr(t *testing.T) {
	c := NewClient("", "", "")
	val := "hello"
	ptr := c.ToStringPtr(val)
	if ptr == nil {
		t.Fatal("ToStringPtr returned nil")
	}
	if *ptr != val {
		t.Errorf("*ToStringPtr(%q) = %q; want %q", val, *ptr, val)
	}
}

// TestClient_ToInt64Ptr verifies that ToInt64Ptr on a Client delegates
// to the unexported helper and returns a pointer to the given value.
func TestClient_ToInt64Ptr(t *testing.T) {
	c := NewClient("", "", "")
	var val int64 = 42
	ptr := c.ToInt64Ptr(val)
	if ptr == nil {
		t.Fatal("ToInt64Ptr returned nil")
	}
	if *ptr != val {
		t.Errorf("*ToInt64Ptr(%d) = %d; want %d", val, *ptr, val)
	}
}

// TestGetChannels_Success verifies that GetChannels correctly parses a successful
// response from the Slack conversations.list API using an httptest server.
func TestGetChannels_Success(t *testing.T) {
	mockResp := GetChannelsResponse{
		OK: true,
		Channels: []Channel{
			{ID: "C001", Name: "general", IsChannel: true},
			{ID: "C002", Name: "random", IsChannel: true},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the Authorization header is forwarded
		auth := r.Header.Get("Authorization")
		if auth != "Bearer xoxb-real-token" {
			t.Errorf("Authorization = %q; want %q", auth, "Bearer xoxb-real-token")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer srv.Close()

	// Override the conversations URL template to target the test server.
	old := conversationsListURLTemplate
	conversationsListURLTemplate = srv.URL + "?limit=%v&types=%v%v"
	defer func() { conversationsListURLTemplate = old }()

	c := NewClient("xoxb-real-token", "", "C001")
	resp, err := c.GetChannels(nil)
	if err != nil {
		t.Fatalf("GetChannels returned unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("GetChannels returned nil response")
	}
	if len(resp.Channels) != 2 {
		t.Errorf("len(Channels) = %d; want 2", len(resp.Channels))
	}
	if resp.Channels[0].ID != "C001" {
		t.Errorf("Channels[0].ID = %q; want %q", resp.Channels[0].ID, "C001")
	}
}

// TestGetChannels_NonOKStatus verifies that GetChannels returns a non-nil empty
// response when the mock server returns a non-200 status code.
func TestGetChannels_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	old := conversationsListURLTemplate
	conversationsListURLTemplate = srv.URL + "?limit=%v&types=%v%v"
	defer func() { conversationsListURLTemplate = old }()

	c := NewClient("xoxb-token", "", "")
	resp, err := c.GetChannels(nil)
	if err != nil {
		t.Fatalf("GetChannels with non-200 returned unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("GetChannels with non-200 returned nil response")
	}
	if len(resp.Channels) != 0 {
		t.Errorf("len(Channels) = %d; want 0 for non-200 response", len(resp.Channels))
	}
}

// TestGetChannels_RequestParams verifies that limit, types, and cursor query
// parameters from the GetChannelsRequest are forwarded in the HTTP request URL.
func TestGetChannels_RequestParams(t *testing.T) {
	var capturedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(GetChannelsResponse{OK: true})
	}))
	defer srv.Close()

	old := conversationsListURLTemplate
	conversationsListURLTemplate = srv.URL + "?limit=%v&types=%v%v"
	defer func() { conversationsListURLTemplate = old }()

	c := NewClient("xoxb-token", "", "")
	req := &GetChannelsRequest{
		Limit:      toInt64Ptr(50),
		Types:      toStringPtr("private_channel"),
		NextCursor: toStringPtr("dXNlcjpVMEc5V"),
	}
	_, err := c.GetChannels(req)
	if err != nil {
		t.Fatalf("GetChannels returned error: %v", err)
	}
	if !strings.Contains(capturedURL, "cursor=dXNlcjpVMEc5V") {
		t.Errorf("URL %q does not contain cursor param", capturedURL)
	}
}
