# Implementation Plan: CircleCI Client Package

**Track ID:** devops-8525_20260404
**Jira:** DEVOPS-8525
**Branch:** `feat/devops-8525`

## Overview

The implementation is split into four phases:

1. **Phase 1 -- Types, Builder, and Client Foundation:** Define response types, the builder pattern constructor, and shared HTTP request helpers.
2. **Phase 2 -- Project Endpoints:** Implement `GetProject`, `GetProjectInsights`, `FollowProject`, and `UnfollowProject`.
3. **Phase 3 -- Pipeline and Workflow Endpoints:** Implement `ListPipelines`, `ListWorkflows`, and `CancelWorkflow`.
4. **Phase 4 -- Final Verification and Housekeeping:** Full coverage check, documentation review, tech-stack update.

All phases follow TDD (Red-Green-Refactor). Tests use `net/http/httptest` to mock the CircleCI API.

---

## Phase 1: Types, Builder, and Client Foundation [checkpoint: 8f8be67]

**Goal:** Establish the package structure, define response types, implement the builder-style constructor, and create shared HTTP helper methods.

### Tasks

- [x] **Task 1.1: Create package structure and response types** [02131c3]
  - Create `circleci/` directory with `doc.go` (package comment).
  - Create `circleci/types.go` with response structs for all endpoints:
    - `Project` -- fields from CircleCI project response (slug, name, organization, VCS info, etc.).
    - `ProjectInsights` -- wrapper for workflow insights (workflow metrics items).
    - `FollowProjectResponse` / `UnfollowProjectResponse` -- fields from v1.1 follow/unfollow responses.
    - `Pipeline` and `ListPipelinesResponse` -- pipeline object and paginated list wrapper with `Items` and `NextPageToken`.
    - `Workflow` and `ListWorkflowsResponse` -- workflow object and list wrapper.
  - Use snake_case JSON tags matching CircleCI API field names.
  - TDD: Write `circleci/types_test.go` that unmarshals sample JSON fixtures into each struct and verifies key fields are correctly populated.

- [x] **Task 1.2: Implement ClientBuilder and Client** [8a28631]
  - Create `circleci/client.go` with:
    - `Client` struct containing v1 base URL, v2 base URL, API token, and `*http.Client`.
    - `ClientBuilder` struct with `WithToken(token string)`, `WithHTTPClient(c *http.Client)`, `WithBaseURLs(v1, v2 string)` chainable methods.
    - `NewClientBuilder() *ClientBuilder` constructor.
    - `Build() *Client` terminal method that returns a configured client with sensible defaults (CircleCI cloud URLs, 30s timeout HTTP client).
  - Add unexported helper: `doRequest(ctx context.Context, method, url string, body io.Reader) ([]byte, error)` that sets the `Circle-Token` header, executes the request, reads the body, and returns an error on non-2xx status.
  - TDD: Write `circleci/client_test.go` tests:
    - Builder sets token and produces a client with correct defaults.
    - Builder with custom HTTP client uses the provided client.
    - Builder with custom base URLs overrides defaults.
    - `doRequest` sets the `Circle-Token` header correctly (verify via httptest server).
    - `doRequest` returns error with status code and body on non-2xx response.
    - `doRequest` propagates context cancellation.

- [x] **Task 1.3: Verification** -- Phase 1 [8f8be67]
  - Run `go test -cover ./circleci/...` and verify >90% coverage for new code.
  - Run `go vet ./circleci/...` and `gofmt -l circleci/`.
  - Confirm all exported symbols have GoDoc comments. [checkpoint marker]

---

## Phase 2: Project Endpoints [checkpoint: d4b1560]

**Goal:** Implement `GetProject`, `GetProjectInsights`, `FollowProject`, and `UnfollowProject`.

### Tasks

- [x] **Task 2.1: Implement GetProject** [b540d8b]
  - Add `GetProject(ctx context.Context, owner, repo string) (*Project, error)` method on `*Client`.
  - Calls `GET {v2BaseURL}/project/gh/{owner}/{repo}`.
  - Decodes response JSON into `*Project`.
  - TDD: Write tests using `httptest.NewServer`:
    - Successful response -- returns populated Project struct.
    - HTTP error response (404) -- returns descriptive error.
    - Verify request path and auth header.

- [x] **Task 2.2: Implement GetProjectInsights** [f6d1c82]
  - Add `GetProjectInsights(ctx context.Context, owner, repo string) (*ProjectInsights, error)` method on `*Client`.
  - Calls `GET {v2BaseURL}/insights/gh/{owner}/{repo}/workflows`.
  - Decodes response JSON into `*ProjectInsights`.
  - TDD: Write tests:
    - Successful response -- returns populated insights struct with workflow metrics.
    - HTTP error response -- returns descriptive error.

- [x] **Task 2.3: Implement FollowProject** [bf1d347]
  - Add `FollowProject(ctx context.Context, owner, repo string) (*FollowProjectResponse, error)` method on `*Client`.
  - Calls `POST {v1BaseURL}/project/github/{owner}/{repo}/follow`.
  - Decodes response JSON into `*FollowProjectResponse`.
  - TDD: Write tests:
    - Successful follow -- returns response struct, verifies POST method and path.
    - HTTP error response -- returns descriptive error.

- [x] **Task 2.4: Implement UnfollowProject** [bf1d347]
  - Add `UnfollowProject(ctx context.Context, owner, repo string) (*UnfollowProjectResponse, error)` method on `*Client`.
  - Calls `POST {v1BaseURL}/project/github/{owner}/{repo}/unfollow`.
  - Decodes response JSON into `*UnfollowProjectResponse`.
  - TDD: Write tests:
    - Successful unfollow -- returns response struct, verifies POST method and path.
    - HTTP error response -- returns descriptive error.

- [x] **Task 2.5: Verification** -- Phase 2 [d4b1560]
  - Run `go test -cover ./circleci/...` and verify >90% coverage.
  - Run `go vet ./circleci/...`.
  - Review all error paths have descriptive messages. [checkpoint marker]

---

## Phase 3: Pipeline and Workflow Endpoints [checkpoint: ae23f31]

**Goal:** Implement `ListPipelines`, `ListWorkflows`, and `CancelWorkflow`.

### Tasks

- [x] **Task 3.1: Implement ListPipelines** [92924ab]
  - Add `ListPipelines(ctx context.Context, owner, repo string) (*ListPipelinesResponse, error)` method on `*Client`.
  - Calls `GET {v2BaseURL}/project/gh/{owner}/{repo}/pipeline`.
  - Decodes response JSON into `*ListPipelinesResponse` (contains `Items []Pipeline` and `NextPageToken string`).
  - TDD: Write tests:
    - Successful response with pipelines -- returns populated list.
    - Empty response (no pipelines) -- returns struct with empty items slice.
    - HTTP error response -- returns descriptive error.
    - Verify request path and auth header.

- [x] **Task 3.2: Implement ListWorkflows** [0cd223c]
  - Add `ListWorkflows(ctx context.Context, pipelineID string) (*ListWorkflowsResponse, error)` method on `*Client`.
  - Calls `GET {v2BaseURL}/pipeline/{pipelineID}/workflow`.
  - Decodes response JSON into `*ListWorkflowsResponse`.
  - TDD: Write tests:
    - Successful response with workflows -- returns populated list.
    - Empty response -- returns struct with empty items slice.
    - HTTP error response -- returns descriptive error.

- [x] **Task 3.3: Implement CancelWorkflow** [42cf823]
  - Add `CancelWorkflow(ctx context.Context, workflowID string) error` method on `*Client`.
  - Calls `POST {v2BaseURL}/workflow/{workflowID}/cancel`.
  - Returns nil on success (2xx).
  - Returns descriptive error on non-2xx.
  - TDD: Write tests:
    - Successful cancellation (200/202) -- returns nil.
    - Workflow not found (404) -- returns error.
    - Workflow not in cancellable state (409) -- returns error.
    - Verify request method (POST) and path.

- [x] **Task 3.4: Verification** -- Phase 3 [ae23f31]
  - Run `go test -cover ./circleci/...` and verify >90% coverage.
  - Run `go vet ./circleci/...` and `gofmt -l circleci/`.
  - Review all error paths have descriptive messages. [checkpoint marker]

---

## Phase 4: Final Verification and Housekeeping

**Goal:** Ensure full package quality, update project metadata, and finalize documentation.

### Tasks

- [ ] **Task 4.1: Full test suite and coverage report**
  - Run `go test -cover -coverprofile=coverage/circleci.out ./circleci/...`.
  - Verify overall package coverage >90%.
  - Run `go vet ./circleci/...` and `gofmt -l circleci/`.
  - Ensure no linting issues.

- [ ] **Task 4.2: Update project metadata**
  - Update `conductor/tech-stack.md` to add `circleci/` to the package structure.
  - Update `conductor/tracks.md` with the new track entry.

- [ ] **Task 4.3: Verification** -- Phase 4
  - Run full project test suite: `go test ./...`.
  - Verify no regressions in existing packages.
  - Confirm all GoDoc comments are present on exported symbols. [checkpoint marker]
