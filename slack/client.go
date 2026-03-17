package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// conversationsListURLTemplate is the URL template used by GetChannels.
// It is a package-level variable so tests can override it with an httptest server URL.
var conversationsListURLTemplate = "https://slack.com/api/conversations.list?limit=%v&types=%v%v"

// chatPostMessageURL is the default URL used by PostMessage when no WebhookURL is set.
// It is a package-level variable so tests can override it with an httptest server URL.
var chatPostMessageURL = "https://slack.com/api/chat.postMessage"

// Color constants for Slack message attachment left-border colors.
const (
	// Good is the hex color for a success/green attachment border.
	Good = "#37fd12"
	// Danger is the hex color for an error/red attachment border.
	Danger = "#d21404"
	// Warning is the hex color for a warning/orange attachment border.
	Warning = "#e47200"
	// Blue is the hex color for an informational/blue attachment border.
	Blue = "#0083ff"
)

// Client holds the credentials and configuration required to interact
// with the Slack API. All fields are pointer types so that optional values
// (e.g. WebhookURL) can be left nil when not required.
type Client struct {
	// BotToken is the Slack bot OAuth token used for API calls that require
	// authorisation (e.g. chat.postMessage, conversations.list).
	BotToken *string
	// WebhookURL is an optional incoming-webhook URL. When set, PostMessage
	// sends the payload to this URL instead of chat.postMessage.
	WebhookURL *string
	// DefaultChannelID is the Slack channel ID used when a PostMessageRequest
	// does not specify a channel.
	DefaultChannelID *string
}

// NewClient constructs a Client from the supplied configuration values.
// It stores each value as a pointer so callers can later inspect or override
// individual fields. Unlike the checkr source, this constructor does not
// read environment variables — the caller is responsible for supplying them.
// webhookURL may be an empty string; when empty, PostMessage uses the
// chat.postMessage API endpoint instead.
func NewClient(botToken, webhookURL, defaultChannelID string) Client {
	c := Client{
		BotToken:         toStringPtr(botToken),
		DefaultChannelID: toStringPtr(defaultChannelID),
	}
	if webhookURL != "" {
		c.WebhookURL = toStringPtr(webhookURL)
	}
	return c
}

// ToStringPtr returns a pointer to a copy of val.
// It is the exported counterpart of the package-level toStringPtr helper
// and is provided so callers can build *string fields on request/response
// types without importing a separate utility package.
func (s *Client) ToStringPtr(val string) *string {
	return toStringPtr(val)
}

// ToInt64Ptr returns a pointer to a copy of val.
// It is the exported counterpart of the package-level toInt64Ptr helper
// and is provided so callers can build *int64 fields on request types
// without importing a separate utility package.
func (s *Client) ToInt64Ptr(val int64) *int64 {
	return toInt64Ptr(val)
}

// GetChannels returns a list of Slack channels by calling the conversations.list API.
// Optional parameters in req control pagination (limit, types, cursor); passing nil
// applies sensible defaults (limit=100, types="public_channel").
func (s *Client) GetChannels(req *GetChannelsRequest) (*GetChannelsResponse, error) {
	limit := int64(100)
	types := "public_channel"
	cursorQueryParam := ""

	if req != nil {
		if req.Limit != nil {
			limit = *req.Limit
		}
		if req.Types != nil {
			types = *req.Types
		}
		if req.NextCursor != nil {
			cursorQueryParam = fmt.Sprintf("&cursor=%v", *req.NextCursor)
		}
	}

	url := fmt.Sprintf(conversationsListURLTemplate, limit, types, cursorQueryParam)
	httpReq, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Add("Content-Type", "application/json")
	if s.BotToken == nil {
		return nil, errors.New("GetChannels requires a BotToken; client was constructed without one")
	}
	httpReq.Header.Add("Authorization", fmt.Sprintf("Bearer %v", *s.BotToken))

	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error reading response body")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("GetChannels: unexpected HTTP status %d: %s", resp.StatusCode, string(body))
	}

	getChannelsResponse := &GetChannelsResponse{}
	if err = json.Unmarshal(body, getChannelsResponse); err != nil {
		return nil, errors.Wrapf(err, "error deserializing response: %v", err.Error())
	}

	return getChannelsResponse, nil
}

// PostMessage posts a message to a Slack channel using either an incoming-webhook
// URL (when WebhookURL is set on the client) or the chat.postMessage API endpoint
// (when only BotToken is available).
//
// If message.Channel is nil, DefaultChannelID from the client is used as the target
// channel. The Authorization header is only added when BotToken is non-nil.
func (s *Client) PostMessage(message PostMessageRequest) (*PostMessageResponse, error) {
	url := chatPostMessageURL
	if s.WebhookURL != nil && *s.WebhookURL != "" {
		url = *s.WebhookURL
	}

	// Fall back to the client's default channel when the caller omits one.
	if message.Channel == nil {
		message.Channel = s.DefaultChannelID
	}

	body, err := json.Marshal(message)
	if err != nil {
		return nil, errors.Wrap(err, "error serializing message body")
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Add("Content-Type", "application/json")
	if s.BotToken != nil {
		httpReq.Header.Add("Authorization", fmt.Sprintf("Bearer %v", *s.BotToken))
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "error posting message to slack channel")
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error reading response body")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("PostMessage: unexpected HTTP status %d: %s", resp.StatusCode, string(respBody))
	}

	postMessageResponse := &PostMessageResponse{}
	if err = json.Unmarshal(respBody, postMessageResponse); err != nil {
		return nil, errors.Wrapf(err, "error deserializing response: %v", err.Error())
	}
	if !postMessageResponse.OK {
		log.Error(string(respBody))
		return postMessageResponse, errors.Errorf("error when sending message: %v", postMessageResponse.Error)
	}

	return postMessageResponse, nil
}
