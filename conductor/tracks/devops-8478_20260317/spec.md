# Specification: gotham — Add Slack Package (from checkr)

## Overview

Move the `slack` package from `github.com/TouchBistro/checkr/slack` into `github.com/TouchBistro/gotham/slack`, adapting it for use as a standalone gotham library package. All `checkr`-specific dependencies and naming must be removed. The package must compile cleanly, be fully documented, have a README, and be covered by tests that do not call the real Slack API.

## Background

The `checkr` service contains a `slack` package that provides a Slack HTTP client and message formatting helpers. As part of merging `checkr` into `devops-api-service`, the reusable Slack client is being promoted to `gotham` so it can be shared across TouchBistro's Go microservice fleet. This track covers only the `gotham` repository changes — migrating and adapting the package so it has no dependency on `checkr`.

---

## Functional Requirements

### FR-1: Create the slack Package at github.com/TouchBistro/gotham/slack

**Description:** The package must be placed at `slack/` within the gotham module, with import path `github.com/TouchBistro/gotham/slack`.

**Acceptance Criteria:**
- Package declaration in all files is `package slack`.
- Import path resolves to `github.com/TouchBistro/gotham/slack`.
- The following source files are present: `client.go`, `types.go`, `templates.go`.
- The stub file `slack.go` (containing only the comment "this file is to be deleted") is not included.

**Priority:** High

---

### FR-2: Remove All checkr Dependencies

**Description:** No import referencing `github.com/TouchBistro/checkr` may exist in any file under `slack/`.

**Acceptance Criteria:**
- No import path starting with `github.com/TouchBistro/checkr` exists in any file under `slack/`.
- `util.ToStringPtr` and `util.ToInt64Ptr` calls from `checkr/util` are replaced with local unexported helpers defined within the `slack` package (not in a separate sub-package).
- `env.CoalesceEnv` calls from `checkr/env` are eliminated by the constructor changes described in FR-3 and FR-4.

**Priority:** High

---

### FR-3: Refactor Client Constructor to Accept Parameters Directly

**Description:** `NewClient()` in `client.go` currently reads configuration from environment variables via `env.CoalesceEnv`. The constructor must be changed to accept configuration values as parameters instead.

**Acceptance Criteria:**
- `NewClient` accepts `botToken`, `webhookURL`, and `defaultChannelID` as `string` parameters (or pointer equivalents consistent with the `Client` struct fields).
- `NewClient` does not read from environment variables.
- The `Client` struct fields `BotToken`, `WebhookURL`, and `DefaultChannelID` remain as `*string`.
- The constructor signature is documented with godoc.

**Priority:** High

---

### FR-4: Refactor FormatSimpleMessage to Accept baseURL as Parameter

**Description:** `templates.go` contains a function currently named `FormatSimpleCheckrMessage` that calls `env.CoalesceEnv` to read a base URL. This function must be renamed and its dependency on env vars removed.

**Acceptance Criteria:**
- Function is renamed from `FormatSimpleCheckrMessage` to `FormatSimpleMessage`.
- `FormatSimpleMessage` accepts `baseURL string` as an explicit parameter instead of reading it from an environment variable.
- No call to `env.CoalesceEnv` or any other env-var reading helper exists in `templates.go`.
- The function signature is documented with godoc.

**Priority:** High

---

### FR-5: Write Godoc on All Exported Symbols

**Description:** Every exported type, function, method, and constant in the package must have a godoc comment.

**Acceptance Criteria:**
- All exported constants (`Good`, `Danger`, `Warning`, `Blue`) have godoc comments.
- `Client` struct and all its exported fields have godoc comments.
- `NewClient` and all exported methods on `Client` (`GetChannels`, `PostMessage`, `ToStringPtr`, `ToInt64Ptr`) have godoc comments.
- All exported types in `types.go` (`PostMessageRequest`, `MessageAttachment`, `MessageBlock`, `GetChannelsRequest`, `GetChannelsResponse`, and any others present) have godoc comments.
- `FormatSimpleMessage` in `templates.go` has a godoc comment.

**Priority:** Medium

---

### FR-6: Write Tests Without Calling the Real Slack API

**Description:** A `slack_test.go` (or multiple `*_test.go` files) must be provided that cover the package behavior without making real HTTP calls to Slack.

**Acceptance Criteria:**
- Tests for `FormatSimpleMessage` verify that the returned message struct is correctly populated given known inputs.
- Tests for `Client` struct initialization verify that `NewClient` correctly assigns the supplied values to the struct fields.
- Tests for `PostMessage` use `httptest.NewServer` to mock the Slack HTTP endpoint and assert the correct request payload and handling of both success and error responses.
- Tests for `GetChannels` use `httptest.NewServer` to mock the Slack HTTP endpoint and assert correct request and response parsing.
- No test requires a real `SLACK_BOT_TOKEN` environment variable to be set.
- `go test ./slack/...` passes with no failures.

**Priority:** High

---

### FR-7: Write README.md for the slack Package

**Description:** A `README.md` must be created inside `slack/` explaining the package purpose, construction of the client, and usage examples.

**Acceptance Criteria:**
- `slack/README.md` exists.
- README covers: purpose of the package, how to initialize a `Client` with `NewClient`, how to call `PostMessage` and `GetChannels`, the available color constants, and a minimal `FormatSimpleMessage` usage example.

**Priority:** Medium

---

### FR-8: Patch-Level Version Bump Only

**Description:** Adding the `slack` package constitutes a backward-compatible addition. The module version must not receive a major or minor version bump.

**Acceptance Criteria:**
- `go.mod` module path remains `github.com/TouchBistro/gotham` (no `/v2` or similar suffix added).
- The `go` directive version in `go.mod` is not changed.
- If a `CHANGELOG.md` entry is added, it is categorized as a patch or minor addition, not a breaking change.

**Priority:** Low

---

## Non-Functional Requirements

### NFR-1: Test Coverage

The `slack` package must achieve >90% code coverage as measured by `go test -cover ./slack/...`.

### NFR-2: Compilability

The package must compile cleanly with `go build ./slack/...` and produce no errors from `go vet ./slack/...`.

### NFR-3: No Breaking Changes to Existing Packages

Adding the `slack` package must not alter or break any existing packages within gotham (`http/`, `cache/`, `util/`, `sql/qb/`).

### NFR-4: No New External Dependencies Required

The `slack` package implementation uses only Go standard library packages and dependencies already present in `go.mod` (specifically `github.com/pkg/errors` and `github.com/sirupsen/logrus`, both already declared). No new entries in `go.mod` should be required.

---

## User Stories

### US-1: Sending a Slack Message via Bot Token

**As** a TouchBistro backend engineer building a Go microservice,
**I want** to import `github.com/TouchBistro/gotham/slack` and post a message to a Slack channel,
**So that** I can send operational notifications without implementing an HTTP client from scratch.

**Scenario:** Post a message using a bot token
- **Given** I have a Slack bot token, webhook URL, and a default channel ID
- **When** I call `slack.NewClient(botToken, webhookURL, channelID)` and then `client.PostMessage(req)`
- **Then** the message is sent to the Slack API and a response is returned without error

---

### US-2: Formatting a Simple Notification Message

**As** a TouchBistro backend engineer,
**I want** to call `slack.FormatSimpleMessage(...)` to produce a consistently structured Slack message,
**So that** my service's notifications have a standard appearance without manually constructing message payloads.

**Scenario:** Format a message with a known base URL
- **Given** I have a title, link path, and base URL
- **When** I call `slack.FormatSimpleMessage(title, color, message, detailsPageRelativePath, baseURL)`
- **Then** I receive a `PostMessageRequest` with correctly set attachment fields

---

### US-3: Retrieving Available Channels

**As** a TouchBistro backend engineer,
**I want** to retrieve a list of Slack channels from the workspace,
**So that** my service can resolve channel names to IDs dynamically.

**Scenario:** Fetch channels
- **Given** I have a configured `Client` with a valid bot token
- **When** I call `client.GetChannels(req)`
- **Then** I receive a `GetChannelsResponse` containing the list of channels

---

## Technical Considerations

### Local Pointer Helpers

`checkr/util.ToStringPtr` and `ToInt64Ptr` are trivial one-liners. Rather than creating a separate sub-package (as was done for `sql/qb/tmp`), these should be defined as unexported helpers directly in `client.go` or a small `helpers.go` file within the `slack` package. The exported `ToStringPtr` and `ToInt64Ptr` methods on `Client` can delegate to these helpers or be implemented inline, preserving the existing public API surface.

### Existing Dependencies Already in go.mod

`github.com/pkg/errors` and `github.com/sirupsen/logrus` are both already direct dependencies in `go.mod`. No new `go.mod` entries are anticipated for this package.

### httptest for Tests

Tests must use `net/http/httptest.NewServer` to intercept outbound HTTP calls made by `PostMessage` and `GetChannels`. The `Client` struct's `WebhookURL` and/or `BotToken` fields can be pointed at the test server's URL to achieve this without modifying production code.

### ioutil.ReadAll Deprecation

The source `client.go` uses `ioutil.ReadAll` (from `io/ioutil`). Since gotham targets Go 1.22, this should be updated to `io.ReadAll` during migration.

---

## Out of Scope

- Making real calls to the Slack API in tests or CI.
- Adding retry logic, rate limiting, or circuit breaking to the Slack client.
- Supporting Slack API endpoints beyond those already present in the source (`chat.postMessage`, `conversations.list`).
- Modifying other gotham packages to use the new `slack` package.
- Any changes to the `devops-api-service` or `checkr` repositories (covered by separate Jira tickets).

---

## Open Questions

1. The exported `ToStringPtr` and `ToInt64Ptr` methods on `Client` exist in the source. Should these remain exported methods on `Client`, or should they be promoted to package-level exported functions? The Jira description does not explicitly change their visibility — they should be kept as-is (exported methods on `Client`) unless directed otherwise.

2. The exact signatures of `GetChannelsRequest`, `GetChannelsResponse`, and other types in `types.go` are not fully specified in the Jira ticket. They must be migrated as-is from the checkr source without structural changes.

3. Should `FormatSimpleMessage` in `templates.go` retain any other parameters that were previously sourced from `checkr/util` beyond the `baseURL` change? This depends on the full content of `templates.go` which was not fully reproduced in the Jira description — the implementer must inspect the source file directly.
