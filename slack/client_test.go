package slack

import (
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
