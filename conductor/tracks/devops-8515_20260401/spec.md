# Specification: Shipit Client Package

**Track ID:** devops-8515_20260401
**Jira:** DEVOPS-8515
**Type:** Feature
**Created:** 2026-04-01

## Overview

Add a `shipit` package to `github.com/TouchBistro/gotham` that provides a Go HTTP client for the Shipit deployment engine API (based on shipit-engine v0.39.0). The client enables callers to list stacks, lock/unlock individual stacks, and perform bulk lock/unlock operations across all stacks with Go concurrency.

## Background

Shipit is a deployment utility based on Shopify's shipit-engine, running at TouchBistro as a self-hosted deployment. The gotham library needs a reusable client so that multiple services (e.g., devops-api-service) can interact with the Shipit API without duplicating HTTP/auth logic. Only the gotham client package is in scope for this track.

## Functional Requirements

### FR-1: Client Constructor (Builder Pattern)

**Description:** Provide a builder-style constructor that accepts two required values: an API password and a base URI.

**Acceptance Criteria:**
- The constructor accepts an API password (string) and a base URI (string).
- The API password is used to create a Basic Auth header for all API calls. The username is always an empty string.
- The base URI is stored and used as the prefix for all API endpoint paths.
- The client is usable immediately after construction with no additional setup.

**Priority:** P0

### FR-2: ListAllStacks

**Description:** Retrieve all stacks from the Shipit API with automatic pagination.

**Acceptance Criteria:**
- Calls `GET {base_uri}/api/stacks` with `page_size=50` query parameter.
- Paginates using the `since=n` query parameter on subsequent calls, continuing until the response `Link` header has no `rel=next` value.
- Returns a slice of Stack structs with JSON field mapping as specified (snake_case JSON tags, PascalCase Go fields).
- Stack struct includes all fields from the schema: `id`, `repo_owner`, `repo_name`, `environment`, `html_url`, `url`, `tasks_url`, `deploy_url`, `merge_requests_url`, `undeployed_commits_count`, `is_locked`, `continuous_deployment`, `created_at`, `updated_at`, `last_deployed_at`, `branch`, `merge_queue_enabled`, `is_archived`, `archived_since`, `ignore_ci`.
- The stack identifier for use in other endpoints is the concatenation `repo_owner/repo_name/environment` (NOT the integer `id` field).

**Priority:** P0

### FR-3: LockStack

**Description:** Lock a specific stack by its identifier.

**Acceptance Criteria:**
- Calls `POST {base_uri}/api/stacks/{stack_id}/lock` where `stack_id` is `repo_owner/repo_name/environment`.
- Sends a JSON request body: `{"reason": "<reason>"}`.
- Sets `Content-Type: application/json` header.
- Returns an error if the API call fails.

**Priority:** P0

### FR-4: UnlockStack

**Description:** Unlock a specific stack by its identifier.

**Acceptance Criteria:**
- Calls `DELETE {base_uri}/api/stacks/{stack_id}/lock` where `stack_id` is `repo_owner/repo_name/environment`.
- Returns an error if the API call fails.

**Priority:** P0

### FR-5: LockAll

**Description:** Lock all stacks concurrently.

**Acceptance Criteria:**
- Calls `ListAllStacks()` to retrieve all stacks.
- Calls `LockStack()` on each stack using Go concurrency (errgroup).
- The `stack_id` passed to `LockStack` is constructed as `repo_owner/repo_name/environment`.
- Accepts a `reason` string that is passed to each `LockStack` call.
- Returns an error if any individual lock operation fails.

**Priority:** P0

### FR-6: UnlockAll

**Description:** Unlock all stacks concurrently.

**Acceptance Criteria:**
- Calls `ListAllStacks()` to retrieve all stacks.
- Calls `UnlockStack()` on each stack using Go concurrency (errgroup).
- The `stack_id` passed to `UnlockStack` is constructed as `repo_owner/repo_name/environment`.
- Returns an error if any individual unlock operation fails.

**Priority:** P0

## Non-Functional Requirements

### NFR-1: Authentication

- All API calls must include a Basic Auth header with an empty username and the configured API password.

### NFR-2: Error Handling

- All HTTP errors (non-2xx status codes) must be surfaced as Go errors with context (status code, response body excerpt).
- Network errors must be wrapped with descriptive context using `fmt.Errorf` with `%w`.

### NFR-3: Testability

- The client must be testable with `net/http/httptest` servers (no live API calls in tests).
- Test coverage must be >90% for the package.

### NFR-4: Concurrency Safety

- `LockAll` and `UnlockAll` must use `golang.org/x/sync/errgroup` for structured concurrency with proper error propagation.

### NFR-5: Code Style

- Follow the project Go style guide: `gofmt`, table-driven tests, GoDoc on all exported symbols.
- JSON struct tags use snake_case matching the Shipit API field names.

## User Stories

### US-1: Service Developer Uses Shipit Client

**As** a TouchBistro backend engineer,
**I want** a pre-built Shipit client in gotham,
**So that** I can integrate Shipit operations (list, lock, unlock) into my service without writing raw HTTP calls.

**Scenarios:**

**Given** valid Shipit credentials and base URI,
**When** I construct a client and call `ListAllStacks()`,
**Then** I receive a complete slice of Stack structs from all pages.

**Given** a valid stack identifier,
**When** I call `LockStack("touchbistro/myservice/production", "deploying hotfix")`,
**Then** the stack is locked in Shipit with the provided reason.

**Given** all stacks need to be locked for a maintenance window,
**When** I call `LockAll("maintenance window")`,
**Then** all stacks are locked concurrently and any failure is reported.

## Technical Considerations

- **Package location:** `shipit/` at the repository root, following the existing pattern (`slack/`, `cache/`, `sql/`).
- **HTTP client:** Use `net/http` standard library (consistent with the `slack` package pattern).
- **Error wrapping:** Use `fmt.Errorf` with `%w` verb (or `github.com/pkg/errors` to stay consistent with existing packages).
- **Link header parsing:** Parse the standard HTTP `Link` header to detect `rel=next` for pagination.
- **Concurrency:** Use `golang.org/x/sync/errgroup` (already referenced in the code style guide).
- **Time fields:** Use `string` for `created_at`, `updated_at`, `last_deployed_at` to avoid parsing issues, or `*time.Time` with proper JSON handling. Decision to be made during implementation.
- **Nullable fields:** `archived_since` can be `null`; use a pointer type (`*string` or `*time.Time`).

## Out of Scope

- devops-api-service endpoints (separate repository, separate track).
- Shipit API endpoints beyond stacks (tasks, deploys, merge requests, etc.).
- Retry/backoff logic (can be added later).
- Rate limiting.
- Context/cancellation support beyond what errgroup provides (can be enhanced later).

## Open Questions

1. Should `LockAll`/`UnlockAll` limit concurrency (e.g., max 10 goroutines) to avoid overwhelming the Shipit API? Recommendation: yes, use `errgroup.SetLimit()`.
2. Should time fields (`created_at`, `updated_at`, `last_deployed_at`) be parsed as `time.Time` or kept as raw strings? Recommendation: use `time.Time` for type safety.
3. Should the client accept a `context.Context` parameter on all methods for cancellation support? Recommendation: yes, follow Go conventions.
