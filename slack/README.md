# slack

Package `slack` provides a Slack HTTP client and message-formatting helpers for
posting messages and retrieving channel information from the Slack API. It is a
self-contained package with no dependency on external configuration systems —
all credentials and URLs are supplied by the caller at construction time,
making it straightforward to use in any TouchBistro Go service.

---

## Installation

```bash
go get github.com/TouchBistro/gotham/slack
```

---

## Creating a Client

Use `NewClient` to construct a `Client` from your bot token, incoming-webhook
URL, and default channel ID. All three values are stored as `*string` fields so
that optional values (e.g. `webhookURL`) can be left empty without special
handling.

```go
import "github.com/TouchBistro/gotham/slack"

client := slack.NewClient(
    "xoxb-your-bot-token",           // Slack bot OAuth token
    "https://hooks.slack.com/...",    // Incoming-webhook URL (may be empty)
    "C0123456789",                    // Default channel ID
)
```

When `webhookURL` is an empty string, `PostMessage` falls back to the
`chat.postMessage` API endpoint and uses the bot token for authorisation.

---

## Posting a Message

`PostMessage` sends a `PostMessageRequest` to Slack. If the request does not
include a `Channel`, the client's `DefaultChannelID` is used automatically.

```go
req := slack.PostMessageRequest{
    Text: client.ToStringPtr("Deployment complete!"),
}

resp, err := client.PostMessage(req)
if err != nil {
    log.Fatalf("PostMessage failed: %v", err)
}
fmt.Printf("Message sent, ok=%v\n", resp.OK)
```

To post to a specific channel, set `Channel` on the request:

```go
req := slack.PostMessageRequest{
    Channel: client.ToStringPtr("C9876543210"),
    Text:    client.ToStringPtr("Alert: something happened."),
}
```

---

## Getting Channels

`GetChannels` calls the Slack `conversations.list` API and returns a paginated
list of channels. Pass `nil` to use the defaults (limit=100, type=public_channel),
or supply a `*GetChannelsRequest` to customise the query.

```go
// Use defaults
resp, err := client.GetChannels(nil)
if err != nil {
    log.Fatalf("GetChannels failed: %v", err)
}
for _, ch := range resp.Channels {
    fmt.Printf("Channel: %s (%s)\n", ch.Name, ch.ID)
}

// Paginated request with custom type
req := &slack.GetChannelsRequest{
    Limit:      client.ToInt64Ptr(200),
    Types:      client.ToStringPtr("private_channel"),
    NextCursor: client.ToStringPtr(resp.ResponseMetadata.NextCursor),
}
nextPage, err := client.GetChannels(req)
```

---

## Color Constants

Use the pre-defined color constants for the left-border color of a Slack
message attachment:

| Constant        | Hex Value   | Usage               |
|-----------------|-------------|---------------------|
| `slack.Good`    | `#37fd12`   | Success / green     |
| `slack.Danger`  | `#d21404`   | Error / red         |
| `slack.Warning` | `#e47200`   | Warning / orange    |
| `slack.Blue`    | `#0083ff`   | Informational / blue|

```go
attachment := slack.MessageAttachment{
    Color: client.ToStringPtr(slack.Good),
    Text:  client.ToStringPtr("All checks passed."),
}
```

---

## Formatting a Message

`FormatSimpleMessage` builds a pre-structured `PostMessageRequest` containing a
color-coded attachment with a mrkdwn body block, a divider, and a "See Details"
link. This is useful for consistent-looking status notifications.

```go
req := slack.FormatSimpleMessage(
    "Deploy #42",                    // title
    slack.Good,                      // attachment border color
    "Service *api* deployed to prod.",// mrkdwn message body
    "/deploys/42",                   // relative path for the details link
    "https://dashboard.example.com", // base URL
)

// req.Text  => "*Deploy #42:*"
// req.Attachments[0].Color => "#37fd12"
// Blocks[2] contains a "See Details" link to https://dashboard.example.com/deploys/42

resp, err := client.PostMessage(req)
if err != nil {
    log.Fatalf("PostMessage failed: %v", err)
}
```

### ToStringPtr / ToInt64Ptr helpers

The `Client` exposes two convenience methods so callers can build `*string` and
`*int64` fields without importing a separate utility package:

```go
client.ToStringPtr("hello")  // returns *string pointing to "hello"
client.ToInt64Ptr(100)       // returns *int64 pointing to 100
```
