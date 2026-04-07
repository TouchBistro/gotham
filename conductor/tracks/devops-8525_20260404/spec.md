# Specification: CircleCI Client Package

**Track ID:** devops-8525_20260404
**Jira:** DEVOPS-8525
**Type:** Feature
**Created:** 2026-04-04

## Overview

Add a `circleci` package to `github.com/TouchBistro/gotham` that provides a Go HTTP client for the CircleCI API (v1 and v2). The client uses raw `net/http` calls (no third-party CircleCI SDK) and exposes methods for querying projects, managing pipelines/workflows, and following/unfollowing projects. A builder-style constructor is used for future extensibility.

## Background

TouchBistro services need to interact with CircleCI for CI/CD operations such as listing pipelines, inspecting workflows, and managing project follow state. Rather than duplicating HTTP/auth logic across services, a reusable client in gotham standardizes the interaction. The CircleCI API spans two versions: v1.1 (legacy endpoints like follow/unfollow, project details) and v2 (pipelines, workflows, project insights). The client will abstract over both API versions behind a single unified interface.

## Functional Requirements

### FR-1: Builder-Style Client Constructor

**Description:** Provide a builder-style constructor for the CircleCI client that accepts an API token and allows future extensibility (e.g., custom HTTP client, base URL override).

**Acceptance Criteria:**
- A `ClientBuilder` (or equivalent builder type) is provided with a method to set the CircleCI API token.
- The builder produces a `Client` struct via a `Build()` method.
- The API token is used as a `Circle-Token` header on all API requests.
- The client defaults to the CircleCI cloud base URLs (`https://circleci.com/api/v1.1` for v1 and `https://circleci.com/api/v2` for v2).
- The client is usable immediately after `Build()` with no additional setup.
- The builder allows optional configuration (e.g., custom `*http.Client`) for testability.

**Priority:** P0

### FR-2: GetProject

**Description:** Retrieve project information for a given owner/repo from CircleCI.

**Acceptance Criteria:**
- Method signature: `GetProject(ctx context.Context, owner, repo string) (*Project, error)`.
- Calls the appropriate CircleCI API endpoint to retrieve project details.
- Returns a `Project` struct with JSON field mapping matching the API response.
- Returns a descriptive error on non-2xx responses.

**Priority:** P0

### FR-3: GetProjectInsights

**Description:** Retrieve project workflow insights/summary metrics for a given owner/repo.

**Acceptance Criteria:**
- Method signature: `GetProjectInsights(ctx context.Context, owner, repo string) (*ProjectInsights, error)`.
- Calls the CircleCI v2 insights endpoint for the project.
- Returns a struct representing the insights response (workflow metrics, success rates, etc.).
- Returns a descriptive error on non-2xx responses.

**Priority:** P0

### FR-4: FollowProject

**Description:** Follow a project on CircleCI for the authenticated user.

**Acceptance Criteria:**
- Method signature: `FollowProject(ctx context.Context, owner, repo string) (*FollowProjectResponse, error)`.
- Calls the CircleCI v1.1 follow endpoint: `POST /api/v1.1/project/github/{owner}/{repo}/follow`.
- Returns a struct representing the follow response.
- Returns a descriptive error on non-2xx responses.

**Priority:** P0

### FR-5: UnfollowProject

**Description:** Unfollow a project on CircleCI for the authenticated user.

**Acceptance Criteria:**
- Method signature: `UnfollowProject(ctx context.Context, owner, repo string) (*UnfollowProjectResponse, error)`.
- Calls the CircleCI v1.1 unfollow endpoint: `POST /api/v1.1/project/github/{owner}/{repo}/unfollow`.
- Returns a struct representing the unfollow response.
- Returns a descriptive error on non-2xx responses.

**Priority:** P0

### FR-6: ListPipelines

**Description:** List pipelines for a given project (owner/repo).

**Acceptance Criteria:**
- Method signature: `ListPipelines(ctx context.Context, owner, repo string) (*ListPipelinesResponse, error)`.
- Calls the CircleCI v2 endpoint: `GET /api/v2/project/gh/{owner}/{repo}/pipeline`.
- Returns a struct containing the list of pipelines and any pagination token.
- Returns a descriptive error on non-2xx responses.

**Priority:** P0

### FR-7: ListWorkflows

**Description:** List all workflows for a given pipeline ID.

**Acceptance Criteria:**
- Method signature: `ListWorkflows(ctx context.Context, pipelineID string) (*ListWorkflowsResponse, error)`.
- Calls the CircleCI v2 endpoint: `GET /api/v2/pipeline/{pipelineID}/workflow`.
- Returns a struct containing the list of workflows.
- Returns a descriptive error on non-2xx responses.

**Priority:** P0

### FR-8: CancelWorkflow

**Description:** Cancel a running workflow by its ID.

**Acceptance Criteria:**
- Method signature: `CancelWorkflow(ctx context.Context, workflowID string) error`.
- Calls the CircleCI v2 endpoint: `POST /api/v2/workflow/{workflowID}/cancel`.
- Returns nil on success (2xx response).
- Returns a descriptive error on non-2xx responses.

**Priority:** P0

## Non-Functional Requirements

### NFR-1: Authentication

- All API calls must include a `Circle-Token` header with the configured API token.

### NFR-2: Error Handling

- All HTTP errors (non-2xx status codes) must be surfaced as Go errors with context (status code, response body excerpt).
- Network errors must be wrapped with descriptive context using `fmt.Errorf` with `%w`.

### NFR-3: Testability

- The client must be testable with `net/http/httptest` servers (no live API calls in tests).
- The builder must allow injecting a custom base URL for test servers.
- Test coverage must be >90% for the package.

### NFR-4: API Version Abstraction

- The client internally routes requests to the correct API version (v1.1 or v2) based on the endpoint, but callers do not need to know which version is used.

### NFR-5: Code Style

- Follow the project Go style guide: `gofmt`, table-driven tests, GoDoc on all exported symbols.
- JSON struct tags use snake_case matching the CircleCI API field names.

## User Stories

### US-1: DevOps Engineer Uses CircleCI Client

**As** a TouchBistro backend/DevOps engineer,
**I want** a pre-built CircleCI client in gotham,
**So that** I can interact with CircleCI (list pipelines, cancel workflows, follow projects) from Go services without writing raw HTTP calls.

**Scenarios:**

**Given** a valid CircleCI API token,
**When** I build a client and call `ListPipelines(ctx, "TouchBistro", "my-service")`,
**Then** I receive a list of pipeline objects for that project.

**Given** a running workflow ID,
**When** I call `CancelWorkflow(ctx, workflowID)`,
**Then** the workflow is cancelled and no error is returned.

**Given** a project I want to follow,
**When** I call `FollowProject(ctx, "TouchBistro", "my-service")`,
**Then** the project is followed on CircleCI for the authenticated user.

## Technical Considerations

- **Package location:** `circleci/` at the repository root, following the existing pattern (`shipit/`, `slack/`, `cache/`).
- **HTTP client:** Use `net/http` standard library. No third-party CircleCI SDK.
- **API versions:** v1.1 endpoints use `https://circleci.com/api/v1.1/...`, v2 endpoints use `https://circleci.com/api/v2/...`. The client stores both base URLs internally.
- **VCS slug:** CircleCI v2 uses `gh/{owner}/{repo}` as the project slug for GitHub projects. The v1.1 API uses `github/{owner}/{repo}`.
- **Error wrapping:** Use `fmt.Errorf` with `%w` verb for consistent error wrapping.
- **Builder pattern:** Use a `ClientBuilder` struct with chainable `With*` methods and a terminal `Build()` method. This keeps the constructor future-proof for adding options (custom HTTP client, custom base URLs, timeouts).
- **Response types:** Define dedicated Go structs for each API response. Only include fields that are useful to callers; unmapped JSON fields are silently ignored by `encoding/json`.

## Out of Scope

- Pagination support for `ListPipelines` and `ListWorkflows` (initial implementation returns the first page; pagination can be added later).
- Retry/backoff logic.
- Rate limiting.
- CircleCI API endpoints beyond those listed in the requirements (jobs, artifacts, orbs, etc.).
- CircleCI server (self-hosted) support (only CircleCI cloud URLs).

## Open Questions

None -- the Jira ticket provides clear requirements for all seven methods plus the builder constructor.
