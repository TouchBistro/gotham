package slack

import (
	"testing"
)

// TestPostMessageRequest_FieldAssignment verifies that PostMessageRequest
// fields can be assigned and read back correctly.
func TestPostMessageRequest_FieldAssignment(t *testing.T) {
	channel := "C12345"
	text := "hello world"

	req := PostMessageRequest{
		Channel: &channel,
		Text:    &text,
		Attachments: []MessageAttachment{
			{Color: strPtr("#37fd12")},
		},
	}

	if req.Channel == nil || *req.Channel != channel {
		t.Errorf("Channel = %v; want %v", req.Channel, channel)
	}
	if req.Text == nil || *req.Text != text {
		t.Errorf("Text = %v; want %v", req.Text, text)
	}
	if len(req.Attachments) != 1 {
		t.Fatalf("Attachments len = %d; want 1", len(req.Attachments))
	}
	if req.Attachments[0].Color == nil || *req.Attachments[0].Color != "#37fd12" {
		t.Errorf("Attachment Color = %v; want #37fd12", req.Attachments[0].Color)
	}
}

// TestMessageAttachment_FieldAssignment verifies that MessageAttachment fields
// are correctly set.
func TestMessageAttachment_FieldAssignment(t *testing.T) {
	id := "att1"
	color := "#d21404"
	title := "Test Title"
	pretext := "pretext"
	text := "attachment text"
	titleLink := "https://example.com"

	att := MessageAttachment{
		ID:        &id,
		Color:     &color,
		Title:     &title,
		Pretext:   &pretext,
		Text:      &text,
		TitleLink: &titleLink,
		Blocks:    []MessageBlock{},
	}

	if att.ID == nil || *att.ID != id {
		t.Errorf("ID = %v; want %v", att.ID, id)
	}
	if att.Color == nil || *att.Color != color {
		t.Errorf("Color = %v; want %v", att.Color, color)
	}
	if att.Title == nil || *att.Title != title {
		t.Errorf("Title = %v; want %v", att.Title, title)
	}
	if att.Pretext == nil || *att.Pretext != pretext {
		t.Errorf("Pretext = %v; want %v", att.Pretext, pretext)
	}
	if att.Text == nil || *att.Text != text {
		t.Errorf("Text = %v; want %v", att.Text, text)
	}
	if att.TitleLink == nil || *att.TitleLink != titleLink {
		t.Errorf("TitleLink = %v; want %v", att.TitleLink, titleLink)
	}
}

// TestMessageBlock_FieldAssignment verifies MessageBlock construction.
func TestMessageBlock_FieldAssignment(t *testing.T) {
	blockType := Section
	textType := MrkDwn
	emoji := true
	textVal := "some text"

	block := MessageBlock{
		Type: &blockType,
		Text: &MessageBlockText{
			Type:  &textType,
			Emoji: &emoji,
			Text:  &textVal,
		},
	}

	if block.Type == nil || *block.Type != Section {
		t.Errorf("Type = %v; want Section", block.Type)
	}
	if block.Text == nil {
		t.Fatal("Text is nil")
	}
	if block.Text.Type == nil || *block.Text.Type != MrkDwn {
		t.Errorf("Text.Type = %v; want MrkDwn", block.Text.Type)
	}
	if block.Text.Emoji == nil || *block.Text.Emoji != emoji {
		t.Errorf("Text.Emoji = %v; want %v", block.Text.Emoji, emoji)
	}
	if block.Text.Text == nil || *block.Text.Text != textVal {
		t.Errorf("Text.Text = %v; want %v", block.Text.Text, textVal)
	}
}

// TestMessageBlockType_Constants verifies the MessageBlockType constants.
func TestMessageBlockType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		got      MessageBlockType
		expected MessageBlockType
	}{
		{"Section", Section, "section"},
		{"Divider", Divider, "divider"},
		{"Context", Context, "context"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q; want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// TestMessageBlockTextType_Constants verifies the MessageBlockTextType constants.
func TestMessageBlockTextType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		got      MessageBlockTextType
		expected MessageBlockTextType
	}{
		{"PlainText", PlainText, "plain_text"},
		{"MrkDwn", MrkDwn, "mrkdwn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q; want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// TestPostMessageResponse_FieldAssignment verifies PostMessageResponse fields.
func TestPostMessageResponse_FieldAssignment(t *testing.T) {
	resp := PostMessageResponse{
		OK:    true,
		Error: "",
	}

	if !resp.OK {
		t.Errorf("OK = false; want true")
	}
}

// TestChannel_FieldAssignment verifies Channel struct field assignment.
func TestChannel_FieldAssignment(t *testing.T) {
	ch := Channel{
		ID:        "C001",
		Name:      "general",
		IsChannel: true,
		IsGroup:   false,
		IsPrivate: false,
	}

	if ch.ID != "C001" {
		t.Errorf("ID = %q; want %q", ch.ID, "C001")
	}
	if ch.Name != "general" {
		t.Errorf("Name = %q; want %q", ch.Name, "general")
	}
	if !ch.IsChannel {
		t.Errorf("IsChannel = false; want true")
	}
}

// TestGetChannelsResponse_FieldAssignment verifies GetChannelsResponse fields.
func TestGetChannelsResponse_FieldAssignment(t *testing.T) {
	cursor := "cursor123"
	resp := GetChannelsResponse{
		OK: true,
		Channels: []Channel{
			{ID: "C001", Name: "general"},
		},
		ResponseMetadata: ResponseMetadata{NextCursor: &cursor},
	}

	if !resp.OK {
		t.Errorf("OK = false; want true")
	}
	if len(resp.Channels) != 1 {
		t.Fatalf("Channels len = %d; want 1", len(resp.Channels))
	}
	if resp.Channels[0].ID != "C001" {
		t.Errorf("Channels[0].ID = %q; want %q", resp.Channels[0].ID, "C001")
	}
	if resp.ResponseMetadata.NextCursor == nil || *resp.ResponseMetadata.NextCursor != cursor {
		t.Errorf("NextCursor = %v; want %v", resp.ResponseMetadata.NextCursor, cursor)
	}
}

// TestGetChannelsRequest_FieldAssignment verifies GetChannelsRequest fields.
func TestGetChannelsRequest_FieldAssignment(t *testing.T) {
	types := "public_channel"
	limit := int64(50)
	cursor := "next_cursor"

	req := GetChannelsRequest{
		Types:      &types,
		Limit:      &limit,
		NextCursor: &cursor,
	}

	if req.Types == nil || *req.Types != types {
		t.Errorf("Types = %v; want %v", req.Types, types)
	}
	if req.Limit == nil || *req.Limit != limit {
		t.Errorf("Limit = %v; want %v", req.Limit, limit)
	}
	if req.NextCursor == nil || *req.NextCursor != cursor {
		t.Errorf("NextCursor = %v; want %v", req.NextCursor, cursor)
	}
}

// strPtr is a test helper to create a *string from a literal.
func strPtr(s string) *string {
	return &s
}
