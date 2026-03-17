package slack

import (
	"strings"
	"testing"
)

// TestFormatSimpleMessage_Text verifies that the returned PostMessageRequest.Text
// is formatted as "*<title>:*".
func TestFormatSimpleMessage_Text(t *testing.T) {
	req := FormatSimpleMessage("Deploy Success", Good, "Everything went fine.", "/deploys/42", "https://example.com")
	if req.Text == nil {
		t.Fatal("PostMessageRequest.Text = nil; want non-nil")
	}
	want := "*Deploy Success:*"
	if *req.Text != want {
		t.Errorf("Text = %q; want %q", *req.Text, want)
	}
}

// TestFormatSimpleMessage_AttachmentColor verifies that the attachment color
// matches the supplied color argument.
func TestFormatSimpleMessage_AttachmentColor(t *testing.T) {
	req := FormatSimpleMessage("title", Danger, "msg", "/path", "https://base.com")
	if len(req.Attachments) == 0 {
		t.Fatal("Attachments is empty; want at least one attachment")
	}
	att := req.Attachments[0]
	if att.Color == nil {
		t.Fatal("Attachments[0].Color = nil; want non-nil")
	}
	if *att.Color != Danger {
		t.Errorf("Attachments[0].Color = %q; want %q", *att.Color, Danger)
	}
}

// TestFormatSimpleMessage_Block0_SectionMessage verifies that Blocks[0] is a
// Section block whose mrkdwn text equals the supplied message argument.
func TestFormatSimpleMessage_Block0_SectionMessage(t *testing.T) {
	req := FormatSimpleMessage("title", Good, "Hello from gotham!", "/path", "https://base.com")
	if len(req.Attachments) == 0 {
		t.Fatal("Attachments is empty")
	}
	blocks := req.Attachments[0].Blocks
	if len(blocks) < 1 {
		t.Fatal("Blocks has fewer than 1 element")
	}
	b0 := blocks[0]
	if b0.Type == nil || *b0.Type != Section {
		t.Errorf("Blocks[0].Type = %v; want Section", b0.Type)
	}
	if b0.Text == nil {
		t.Fatal("Blocks[0].Text = nil; want non-nil")
	}
	if b0.Text.Text == nil || *b0.Text.Text != "Hello from gotham!" {
		t.Errorf("Blocks[0].Text.Text = %v; want %q", b0.Text.Text, "Hello from gotham!")
	}
	if b0.Text.Type == nil || *b0.Text.Type != MrkDwn {
		t.Errorf("Blocks[0].Text.Type = %v; want MrkDwn", b0.Text.Type)
	}
}

// TestFormatSimpleMessage_Block1_Divider verifies that Blocks[1] is a Divider block.
func TestFormatSimpleMessage_Block1_Divider(t *testing.T) {
	req := FormatSimpleMessage("title", Good, "msg", "/path", "https://base.com")
	blocks := req.Attachments[0].Blocks
	if len(blocks) < 2 {
		t.Fatal("Blocks has fewer than 2 elements")
	}
	b1 := blocks[1]
	if b1.Type == nil || *b1.Type != Divider {
		t.Errorf("Blocks[1].Type = %v; want Divider", b1.Type)
	}
}

// TestFormatSimpleMessage_Block2_SectionDetailsLink verifies that Blocks[2] is a
// Section block whose text contains the concatenation of baseURL and detailsPageRelativePath.
func TestFormatSimpleMessage_Block2_SectionDetailsLink(t *testing.T) {
	baseURL := "https://example.com"
	path := "/deploys/99"
	req := FormatSimpleMessage("title", Blue, "msg", path, baseURL)
	blocks := req.Attachments[0].Blocks
	if len(blocks) < 3 {
		t.Fatal("Blocks has fewer than 3 elements")
	}
	b2 := blocks[2]
	if b2.Type == nil || *b2.Type != Section {
		t.Errorf("Blocks[2].Type = %v; want Section", b2.Type)
	}
	if b2.Text == nil || b2.Text.Text == nil {
		t.Fatal("Blocks[2].Text or Blocks[2].Text.Text = nil; want non-nil")
	}
	fullURL := baseURL + path
	if !strings.Contains(*b2.Text.Text, fullURL) {
		t.Errorf("Blocks[2] text %q does not contain %q", *b2.Text.Text, fullURL)
	}
}
