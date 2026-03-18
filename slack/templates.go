package slack

import "fmt"

// FormatSimpleMessage builds and returns a PostMessageRequest for a standard
// status notification message. The message contains a mrkdwn-formatted body,
// a color-coded attachment border, and a "See Details" link constructed from
// baseURL and detailsPageRelativePath.
//
// Unlike the original checkr implementation, this function does not read
// environment variables — the caller must supply baseURL directly.
func FormatSimpleMessage(title, color, message, detailsPageRelativePath, baseURL string) PostMessageRequest {
	detailsURL := baseURL + detailsPageRelativePath

	sectionType := MessageBlockType(Section)
	dividerType := MessageBlockType(Divider)
	mrkdwnType := MessageBlockTextType(MrkDwn)

	attachments := []MessageAttachment{
		{
			Color: toStringPtr(color),
			Blocks: []MessageBlock{
				{
					Type: &sectionType,
					Text: &MessageBlockText{
						Type: &mrkdwnType,
						Text: toStringPtr(message),
					},
				},
				{
					Type: &dividerType,
				},
				{
					Type: &sectionType,
					Text: &MessageBlockText{
						Type: &mrkdwnType,
						Text: toStringPtr(fmt.Sprintf("*<%v|See Details>*", detailsURL)),
					},
				},
			},
		},
	}

	return PostMessageRequest{
		Text:        toStringPtr(fmt.Sprintf("*%v:*", title)),
		Attachments: attachments,
	}
}
