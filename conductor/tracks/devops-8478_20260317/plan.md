# Implementation Plan: gotham — Add Slack Package (from checkr)

## Overview

This track migrates the `slack` package from `github.com/TouchBistro/checkr/slack` into `github.com/TouchBistro/gotham/slack`. The work is organized into three phases:

- **Phase 1 — Package Scaffold & Types**: Create the package skeleton, migrate `types.go`, and establish local pointer helpers. No external dependencies are introduced.
- **Phase 2 — Client & Templates**: Migrate `client.go` with the refactored constructor and `templates.go` with the renamed function. Both must be free of `checkr` imports.
- **Phase 3 — Tests, Godoc & README**: Write tests using `httptest`, add godoc to all exported symbols, and write the `README.md`.

All tasks follow the Red → Green → Refactor TDD cycle. Commits are made per phase.

---

## Phase 1: Package Scaffold and Types

**Goal:** Establish the `slack/` package directory, migrate all request/response types from `types.go`, and define local unexported pointer helpers that replace `checkr/util` dependencies.

### Tasks

- [x] Task: Inspect source files — Read `~/Projects/devops/checkr/slack/client.go`, `types.go`, `templates.go`, and `slack.go` in full to understand all types, imports, and function signatures before writing any code. (TDD: prerequisite — no test yet, understanding only)

- [x] Task: Create `slack/types.go` — Migrate all request and response types (`PostMessageRequest`, `MessageAttachment`, `MessageBlock`, `GetChannelsRequest`, `GetChannelsResponse`, and any others present in source). Add godoc comments to every exported type and field. Write `slack/types_test.go` with construction tests that assert struct fields are correctly set. (TDD: Write failing test asserting field assignment → implement types → confirm tests pass) [9fd772d]

- [x] Task: Create `slack/helpers.go` — Define unexported `toStringPtr(val string) *string` and `toInt64Ptr(val int64) *int64` helpers. Write `slack/helpers_test.go` to verify each returns a pointer to the correct value. (TDD: Write failing tests → implement helpers → confirm pass) [8461294]

- [x] Verification: Run `go build ./slack/...` and `go vet ./slack/...` — confirm zero errors. Run `go test ./slack/...` — confirm all Phase 1 tests pass. [checkpoint marker]

---

## Phase 2: Client and Templates

**Goal:** Migrate `client.go` with a refactored `NewClient` constructor (accepting parameters instead of reading env vars), migrate `templates.go` with `FormatSimpleMessage` (accepting `baseURL` as a parameter), and confirm no `checkr` imports remain.

### Tasks

- [x] Task: Create `slack/client.go` — Define the `Client` struct with exported `*string` fields (`BotToken`, `WebhookURL`, `DefaultChannelID`). Define the color constants (`Good`, `Danger`, `Warning`, `Blue`). Implement `NewClient(botToken, webhookURL, defaultChannelID string) Client` that assigns values directly to the struct without reading environment variables. Implement exported methods `ToStringPtr` and `ToInt64Ptr` on `Client` (delegating to the unexported helpers in `helpers.go`). Add godoc to all exported symbols. Write `slack/client_test.go` with tests for `NewClient` that assert each field is set to the expected pointer value. (TDD: Write failing test for constructor → implement `NewClient` → confirm pass) [805fa42]

- [x] Task: Implement `GetChannels` method on `Client` in `slack/client.go` [c4573b1] — Port the method from checkr source, replacing `ioutil.ReadAll` with `io.ReadAll` (Go 1.22 standard). Add godoc. Write tests in `slack/client_test.go` using `httptest.NewServer` to mock the Slack API endpoint — test both a successful response and an error response (non-200 or malformed JSON). (TDD: Write failing httptest-based tests → implement method → confirm pass)

- [x] Task: Implement `PostMessage` method on `Client` in `slack/client.go` [a1a5ab0] — Port the method from checkr source, replacing `ioutil.ReadAll` with `io.ReadAll`. Add godoc. Write tests in `slack/client_test.go` using `httptest.NewServer` to mock the Slack API endpoint — test both webhook-path and bot-token-path execution, a successful response, and an error response. (TDD: Write failing httptest-based tests → implement method → confirm pass)

- [x] Task: Create `slack/templates.go` [5a30a3e] — Port `FormatSimpleMessage` (renamed from `FormatSimpleCheckrMessage`). Replace the `env.CoalesceEnv` call with the `baseURL string` parameter. Replace any `checkr/util` calls with the local unexported helpers. Add godoc. Write `slack/templates_test.go` with tests that call `FormatSimpleMessage` with known inputs and assert the returned `PostMessageRequest` fields (attachment color, title link, text, etc.). (TDD: Write failing tests → implement function → confirm pass)

- [ ] Task: Verify no checkr imports remain — Confirm with `grep -r "TouchBistro/checkr" slack/` that the result is empty.

- [ ] Verification: Run `go build ./slack/...` and `go vet ./slack/...` — confirm zero errors. Run `go test -cover ./slack/...` — confirm all tests pass and coverage is reported. [checkpoint marker]

---

## Phase 3: Coverage, Godoc Audit, and README

**Goal:** Reach >90% test coverage, ensure every exported symbol has godoc, and produce a `README.md` for the package.

### Tasks

- [ ] Task: Coverage audit — Run `go test -coverprofile=coverage/slack.out ./slack/... && go tool cover -func=coverage/slack.out`. Identify any uncovered code paths. Write additional tests as needed to reach >90% coverage. (TDD: Identify uncovered branches → write targeted failing tests → implement coverage → confirm pass)

- [ ] Task: Godoc audit — Review all exported symbols across `client.go`, `types.go`, `templates.go`, and `helpers.go` (if any exported). Confirm every exported constant, type, field, function, and method has a godoc comment. Fix any missing or inadequate comments. No tests required for this task; verify with `go doc ./slack/...` producing readable output.

- [ ] Task: Create `slack/README.md` — Write README covering: package purpose, how to construct a `Client` with `NewClient`, how to call `PostMessage` and `GetChannels`, the available color constants (`Good`, `Danger`, `Warning`, `Blue`), and a minimal `FormatSimpleMessage` usage example with sample output. No tests required.

- [ ] Task: Update `conductor/tracks.md` — Add the new track entry for `devops-8478_20260317` with status in-progress.

- [ ] Verification: Run full test suite `go test -cover ./...` — confirm all packages pass and no existing packages are broken. Run `go vet ./...` — confirm zero warnings. Confirm `slack/README.md` exists and is readable. [checkpoint marker]
