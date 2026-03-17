package slack

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// conversationsListURLTemplate is the URL template used by GetChannels.
// It is a package-level variable so tests can override it with an httptest server URL.
var conversationsListURLTemplate = "https://slack.com/api/conversations.list?limit=%v&types=%v%v"

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
func NewClient(botToken, webhookURL, defaultChannelID string) Client {
	return Client{
		BotToken:         toStringPtr(botToken),
		WebhookURL:       toStringPtr(webhookURL),
		DefaultChannelID: toStringPtr(defaultChannelID),
	}
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
	httpReq.Header.Add("Authorization", fmt.Sprintf("Bearer %v", *s.BotToken))

	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	getChannelsResponse := &GetChannelsResponse{}
	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrap(err, "error reading response body")
		}
		if err = json.Unmarshal(body, getChannelsResponse); err != nil {
			return nil, errors.Wrapf(err, "error deserializing response: %v", err.Error())
		}
	}

	return getChannelsResponse, nil
}
