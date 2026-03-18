// Package slack provides a Slack HTTP client and message-formatting helpers
// for posting messages and retrieving channel information from the Slack API.
package slack

// PostMessageRequest encapsulates the payload for a Slack chat.postMessage API call.
type PostMessageRequest struct {
	// Channel is the Slack channel ID to post the message to.
	Channel *string `json:"channel,omitempty"`
	// Text is the plain-text body of the message.
	Text *string `json:"text,omitempty"`
	// Attachments is the list of rich message attachments to include.
	Attachments []MessageAttachment `json:"attachments,omitempty"`
}

// MessageAttachment represents a single Slack message attachment.
type MessageAttachment struct {
	// ID is an optional identifier for the attachment.
	ID *string `json:"id,omitempty"`
	// Color is the left-border color of the attachment (hex string, e.g. "#37fd12").
	Color *string `json:"color,omitempty"`
	// Title is the bold title text shown at the top of the attachment.
	Title *string `json:"title,omitempty"`
	// Pretext is optional text shown above the attachment block.
	Pretext *string `json:"pretext,omitempty"`
	// Text is the main body text of the attachment.
	Text *string `json:"text,omitempty"`
	// TitleLink is an optional URL that the title links to.
	TitleLink *string `json:"title_link,omitempty"`
	// Blocks is the list of Block Kit blocks contained within this attachment.
	Blocks []MessageBlock `json:"blocks,omitempty"`
}

// MessageBlockType is the type identifier for a Slack Block Kit block.
type MessageBlockType string

const (
	// Section is the "section" block type.
	Section MessageBlockType = "section"
	// Divider is the "divider" block type.
	Divider MessageBlockType = "divider"
	// Context is the "context" block type.
	Context MessageBlockType = "context"
)

// MessageBlockTextType is the type identifier for text within a Slack block.
type MessageBlockTextType string

const (
	// PlainText indicates unformatted plain text.
	PlainText MessageBlockTextType = "plain_text"
	// MrkDwn indicates Slack's mrkdwn-formatted text.
	MrkDwn MessageBlockTextType = "mrkdwn"
)

// MessageBlock represents a single Block Kit block within a message attachment.
type MessageBlock struct {
	// Type identifies the kind of block (section, divider, context, etc.).
	Type *MessageBlockType `json:"type,omitempty"`
	// Text is the text element of the block (used in section and context blocks).
	Text *MessageBlockText `json:"text,omitempty"`
	// Accessory is an optional interactive element appended to the block.
	Accessory *MessageAccessory `json:"accessory,omitempty"`
}

// MessageBlockText represents the text object inside a Slack block.
type MessageBlockText struct {
	// Type specifies the text formatting type (plain_text or mrkdwn).
	Type *MessageBlockTextType `json:"type,omitempty"`
	// Emoji indicates whether emoji shortcodes should be rendered (plain_text only).
	Emoji *bool `json:"emoji,omitempty"`
	// Text is the text content.
	Text *string `json:"text,omitempty"`
}

// MessageAccessory represents an optional accessory element in a Slack block.
type MessageAccessory struct{}

// PostMessageResponse is the response returned by the Slack chat.postMessage API.
type PostMessageResponse struct {
	// OK indicates whether the API call succeeded.
	OK bool `json:"ok"`
	// Error contains the Slack error code when OK is false.
	Error string `json:"error"`
}

// Channel represents a Slack channel returned by the conversations.list API.
type Channel struct {
	// ID is the unique Slack channel identifier.
	ID string `json:"id"`
	// Name is the human-readable channel name.
	Name string `json:"name"`
	// IsChannel indicates whether this is a public channel.
	IsChannel bool `json:"is_channel"`
	// IsGroup indicates whether this is a private group channel.
	IsGroup bool `json:"is_group"`
	// IsPrivate indicates whether the channel is private.
	IsPrivate bool `json:"is_private"`
}

// ResponseMetadata holds pagination metadata returned by Slack list APIs.
type ResponseMetadata struct {
	// NextCursor is the cursor token to use for fetching the next page of results.
	NextCursor *string `json:"next_cursor"`
}

// GetChannelsResponse is the response returned by the Slack conversations.list API.
type GetChannelsResponse struct {
	// OK indicates whether the API call succeeded.
	OK bool `json:"ok"`
	// Channels is the list of channels returned in this response page.
	Channels []Channel `json:"channels"`
	// ResponseMetadata contains pagination cursor information.
	ResponseMetadata ResponseMetadata `json:"response_metadata"`
}

// GetChannelsRequest holds optional parameters for a conversations.list API call.
type GetChannelsRequest struct {
	// Types filters the channel types to return (e.g. "public_channel").
	Types *string
	// Limit is the maximum number of channels to return per page.
	Limit *int64
	// NextCursor is the pagination cursor from a previous response.
	NextCursor *string
}
