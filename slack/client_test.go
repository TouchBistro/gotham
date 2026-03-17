package slack

import (
	"encoding/json"
	"io"
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

// ---------------------------------------------------------------------------
// PostMessage tests
// ---------------------------------------------------------------------------

// TestPostMessage_WebhookPath verifies that PostMessage sends to WebhookURL
// when it is set on the client, and that the response is correctly parsed.
func TestPostMessage_WebhookPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q; want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewClient("", srv.URL, "C001")
	req := PostMessageRequest{
		Channel: toStringPtr("C001"),
		Text:    toStringPtr("hello"),
	}
	resp, err := c.PostMessage(req)
	if err != nil {
		t.Fatalf("PostMessage webhook path returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("PostMessage webhook path returned nil response")
	}
	if !resp.OK {
		t.Errorf("resp.OK = false; want true")
	}
}

// TestPostMessage_BotTokenPath verifies that PostMessage sets the Authorization
// header when BotToken is non-nil and WebhookURL is empty.
func TestPostMessage_BotTokenPath(t *testing.T) {
	var capturedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	// Override the chat.postMessage URL for testing
	old := chatPostMessageURL
	chatPostMessageURL = srv.URL
	defer func() { chatPostMessageURL = old }()

	c := NewClient("xoxb-bot-token", "", "C001")
	// Explicitly set WebhookURL to nil to force the bot-token path
	c.WebhookURL = nil
	req := PostMessageRequest{
		Channel: toStringPtr("C001"),
		Text:    toStringPtr("test message"),
	}
	resp, err := c.PostMessage(req)
	if err != nil {
		t.Fatalf("PostMessage bot-token path returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("PostMessage bot-token path returned nil response")
	}
	if capturedAuth != "Bearer xoxb-bot-token" {
		t.Errorf("Authorization = %q; want %q", capturedAuth, "Bearer xoxb-bot-token")
	}
}

// TestPostMessage_ErrorResponse verifies that PostMessage returns an error
// when the Slack API responds with {"ok":false,"error":"channel_not_found"}.
func TestPostMessage_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":false,"error":"channel_not_found"}`))
	}))
	defer srv.Close()

	old := chatPostMessageURL
	chatPostMessageURL = srv.URL
	defer func() { chatPostMessageURL = old }()

	c := NewClient("xoxb-token", "", "C001")
	c.WebhookURL = nil
	req := PostMessageRequest{
		Channel: toStringPtr("C001"),
		Text:    toStringPtr("test"),
	}
	resp, err := c.PostMessage(req)
	if err == nil {
		t.Fatal("PostMessage with error response returned nil error; want non-nil")
	}
	if resp == nil {
		t.Fatal("PostMessage with error response returned nil PostMessageResponse; want non-nil")
	}
	if resp.OK {
		t.Errorf("resp.OK = true; want false for error response")
	}
}

// TestGetChannels_MalformedJSON verifies that GetChannels returns an error
// when the server returns a 200 response with a non-JSON body.
func TestGetChannels_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not valid json`))
	}))
	defer srv.Close()

	old := conversationsListURLTemplate
	conversationsListURLTemplate = srv.URL + "?limit=%v&types=%v%v"
	defer func() { conversationsListURLTemplate = old }()

	c := NewClient("xoxb-token", "", "")
	_, err := c.GetChannels(nil)
	if err == nil {
		t.Fatal("GetChannels with malformed JSON returned nil error; want non-nil")
	}
}

// TestGetChannels_HTTPError verifies that GetChannels returns an error when
// the HTTP client cannot reach the target server (e.g. server already closed).
func TestGetChannels_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // Close immediately so the request will fail

	old := conversationsListURLTemplate
	conversationsListURLTemplate = srv.URL + "?limit=%v&types=%v%v"
	defer func() { conversationsListURLTemplate = old }()

	c := NewClient("xoxb-token", "", "")
	_, err := c.GetChannels(nil)
	if err == nil {
		t.Fatal("GetChannels with unreachable server returned nil error; want non-nil")
	}
}

// TestPostMessage_NonOKStatus verifies that PostMessage returns a nil response
// and no error when the server returns a non-200 HTTP status code.
func TestPostMessage_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	old := chatPostMessageURL
	chatPostMessageURL = srv.URL
	defer func() { chatPostMessageURL = old }()

	c := NewClient("xoxb-token", "", "C001")
	c.WebhookURL = nil
	req := PostMessageRequest{
		Channel: toStringPtr("C001"),
		Text:    toStringPtr("test"),
	}
	resp, err := c.PostMessage(req)
	if err != nil {
		t.Fatalf("PostMessage with non-200 status returned unexpected error: %v", err)
	}
	if resp != nil {
		t.Errorf("PostMessage with non-200 status returned non-nil response; want nil")
	}
}

// TestPostMessage_MalformedJSON verifies that PostMessage returns an error
// when the server returns a 200 response with non-JSON body.
func TestPostMessage_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not valid json`))
	}))
	defer srv.Close()

	old := chatPostMessageURL
	chatPostMessageURL = srv.URL
	defer func() { chatPostMessageURL = old }()

	c := NewClient("xoxb-token", "", "C001")
	c.WebhookURL = nil
	req := PostMessageRequest{
		Channel: toStringPtr("C001"),
		Text:    toStringPtr("test"),
	}
	_, err := c.PostMessage(req)
	if err == nil {
		t.Fatal("PostMessage with malformed JSON returned nil error; want non-nil")
	}
}

// TestPostMessage_HTTPError verifies that PostMessage returns an error when
// the HTTP client cannot reach the target server.
func TestPostMessage_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // Close immediately so the request will fail

	old := chatPostMessageURL
	chatPostMessageURL = srv.URL
	defer func() { chatPostMessageURL = old }()

	c := NewClient("xoxb-token", "", "C001")
	c.WebhookURL = nil
	req := PostMessageRequest{
		Channel: toStringPtr("C001"),
		Text:    toStringPtr("test"),
	}
	_, err := c.PostMessage(req)
	if err == nil {
		t.Fatal("PostMessage with unreachable server returned nil error; want non-nil")
	}
}

// TestPostMessage_DefaultChannelFallback verifies that when message.Channel is
// nil, PostMessage uses the client's DefaultChannelID instead.
func TestPostMessage_DefaultChannelFallback(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		capturedBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	old := chatPostMessageURL
	chatPostMessageURL = srv.URL
	defer func() { chatPostMessageURL = old }()

	c := NewClient("xoxb-token", "", "C-DEFAULT")
	c.WebhookURL = nil
	// Channel is nil — should fall back to DefaultChannelID
	req := PostMessageRequest{
		Text: toStringPtr("fallback test"),
	}
	_, err := c.PostMessage(req)
	if err != nil {
		t.Fatalf("PostMessage default channel fallback returned error: %v", err)
	}
	if !strings.Contains(string(capturedBody), "C-DEFAULT") {
		t.Errorf("request body %q does not contain default channel ID", string(capturedBody))
	}
}
