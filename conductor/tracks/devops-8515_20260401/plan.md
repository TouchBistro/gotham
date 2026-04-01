# Implementation Plan: Shipit Client Package

**Track ID:** devops-8515_20260401
**Jira:** DEVOPS-8515
**Branch:** `feat/devops-8515-shipit-client`

## Overview

The implementation is split into three phases:

1. **Phase 1 -- Types and Client Constructor:** Define the Stack struct, client struct, and builder-style constructor with Basic Auth.
2. **Phase 2 -- Core API Methods:** Implement `ListAllStacks`, `LockStack`, and `UnlockStack` with full pagination and error handling.
3. **Phase 3 -- Bulk Operations:** Implement `LockAll` and `UnlockAll` with concurrent execution using errgroup.

All phases follow TDD (Red-Green-Refactor). Tests use `net/http/httptest` to mock the Shipit API.

---

## Phase 1: Types and Client Constructor [checkpoint: fe3d08f]

**Goal:** Establish the package structure, define the Stack data type, and implement the client constructor with Basic Auth support.

### Tasks

- [x] **Task 1.1: Create package and Stack struct** [5b1fb2a]
  - Create `shipit/` directory with `doc.go` (package comment) and `types.go`.
  - Define the `Stack` struct with all fields from the Shipit API schema, using PascalCase field names and snake_case JSON tags.
  - Handle nullable fields (`archived_since`) with pointer types.
  - Use `time.Time` for timestamp fields (`created_at`, `updated_at`, `last_deployed_at`).
  - TDD: Write `types_test.go` that unmarshals sample JSON into a `Stack` and verifies all fields are correctly populated, including a null `archived_since`.
  - Add a `StackID()` method on `Stack` that returns `repo_owner/repo_name/environment`.
  - TDD: Test `StackID()` returns the correct concatenation.

- [x] **Task 1.2: Implement Client constructor** [17ab64a]
  - Create `client.go` with a `Client` struct containing the base URI and an `*http.Client`.
  - Implement a builder-style constructor: `NewClient(baseURI, apiPassword string) *Client`.
  - Store the Basic Auth credentials (empty username, apiPassword) for use in a helper method `setAuth(req *http.Request)` that sets the Authorization header.
  - TDD: Write `client_test.go` tests:
    - Constructing a client stores base URI and credentials.
    - The auth helper sets a correct Basic Auth header on a request (verify with `req.BasicAuth()`).

- [ ] **Task 1.3: Verification** -- Phase 1
  - Run `go test -cover ./shipit/...` and verify >90% coverage for new code.
  - Run `go vet ./shipit/...` and `gofmt -l shipit/`.
  - Confirm all exported symbols have GoDoc comments. [checkpoint marker]

---

## Phase 2: Core API Methods

**Goal:** Implement `ListAllStacks` with pagination, `LockStack`, and `UnlockStack`.

### Tasks

- [x] **Task 2.1: Implement ListAllStacks with pagination** [b73b443]
  - Add `ListAllStacks() ([]Stack, error)` method on `*Client`.
  - First request: `GET {base_uri}/api/stacks?page_size=50`.
  - Parse the `Link` response header; if `rel=next` is present, extract the `since` parameter and make subsequent calls with `GET {base_uri}/api/stacks?page_size=50&since=N`.
  - Stop when the `Link` header has no `rel=next`.
  - Accumulate all stacks across pages and return the full slice.
  - TDD: Write tests in `client_test.go` using `httptest.NewServer`:
    - Single page (no Link header) -- returns all stacks.
    - Multiple pages (Link header with rel=next on first response, no rel=next on second) -- returns combined stacks.
    - HTTP error response -- returns wrapped error.
    - Empty response (zero stacks) -- returns empty slice, no error.

- [x] **Task 2.2: Implement LockStack** [846febb]
  - Add `LockStack(stackID, reason string) error` method on `*Client`.
  - Sends `POST {base_uri}/api/stacks/{stack_id}/lock` with JSON body `{"reason": "<reason>"}`.
  - Sets `Content-Type: application/json` and Basic Auth headers.
  - Returns error on non-2xx response.
  - TDD: Write tests:
    - Successful lock (200 response) -- no error, verify request method/path/body/headers.
    - Failed lock (422 response) -- returns error with status code and body context.

- [x] **Task 2.3: Implement UnlockStack** [30636a1]
  - Add `UnlockStack(stackID string) error` method on `*Client`.
  - Sends `DELETE {base_uri}/api/stacks/{stack_id}/lock`.
  - Sets Basic Auth header.
  - Returns error on non-2xx response.
  - TDD: Write tests:
    - Successful unlock (204 response) -- no error, verify request method/path/headers.
    - Failed unlock (404 response) -- returns error with context.

- [ ] **Task 2.4: Verification** -- Phase 2
  - Run `go test -cover ./shipit/...` and verify >90% coverage.
  - Run `go vet ./shipit/...`.
  - Review all error paths have descriptive messages. [checkpoint marker]

---

## Phase 3: Bulk Operations

**Goal:** Implement `LockAll` and `UnlockAll` with concurrent execution.

### Tasks

- [ ] **Task 3.1: Add errgroup dependency**
  - Run `go get golang.org/x/sync` to add the errgroup dependency.
  - Update `tech-stack.md` to document the new dependency.

- [ ] **Task 3.2: Implement LockAll**
  - Add `LockAll(reason string) error` method on `*Client`.
  - Calls `ListAllStacks()`, then locks each stack concurrently using `errgroup.Group`.
  - Use `g.SetLimit(10)` to cap concurrency and avoid overwhelming the API.
  - Construct `stack_id` as `stack.RepoOwner + "/" + stack.RepoName + "/" + stack.Environment` (or use `stack.StackID()`).
  - TDD: Write tests:
    - All stacks locked successfully -- verify each stack received a lock request.
    - One stack fails to lock -- error is returned, other locks still attempted.
    - Zero stacks -- no error, no lock calls made.

- [ ] **Task 3.3: Implement UnlockAll**
  - Add `UnlockAll() error` method on `*Client`.
  - Calls `ListAllStacks()`, then unlocks each stack concurrently using `errgroup.Group`.
  - Use `g.SetLimit(10)` to cap concurrency.
  - TDD: Write tests:
    - All stacks unlocked successfully -- verify each stack received an unlock request.
    - One stack fails to unlock -- error is returned.
    - Zero stacks -- no error, no unlock calls made.

- [ ] **Task 3.4: Verification** -- Phase 3
  - Run full test suite: `go test -cover ./shipit/...`.
  - Run `go vet ./shipit/...` and `gofmt -l shipit/`.
  - Verify overall package coverage >90%.
  - Update `conductor/tracks.md` with the new track entry.
  - Update `conductor/tech-stack.md` if not already done. [checkpoint marker]
