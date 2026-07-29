# Specification: Trailing-`*` Prefix Matching in Authr Policy URL Paths

**Track ID:** devops-8813_20260727
**Jira:** DEVOPS-8813
**Type:** Feature
**Created:** 2026-07-27

## Overview

Extend the authr policy engine in `http/auth` so a rule `url` ending in `*` is treated as a
case-insensitive **prefix** match against the request path. Today `Policies.Match` supports only
two forms: the bare `*` global sentinel, and full-path case-insensitive equality. This track adds a
third form — trailing-`*` prefix — via a single `pathMatches(rulePath, reqPath string) bool` helper
used by `Policies.Match`.

## Background

`http/auth/policy_rules.go`, `Policies.Match` currently matches a rule's path with an inline check:

```go
if item.HttpPath == AllPaths || strings.EqualFold(item.HttpPath, req.URL.Path) {
```

so exactly two behaviours exist:

1. `item.HttpPath == "*"` — the *entire* rule url is the literal `*` sentinel (global catch-all).
2. `strings.EqualFold(item.HttpPath, req.URL.Path)` — case-insensitive *full-path equality*.

There is no prefix, wildcard, or regex support. A rule such as `"url": "/cr7/reviews/*"` falls into
the equality branch, is compared literally, and can never match a real request path.

**Problem:** consumers cannot write one rule covering a parameterized route family. Concrete case:
review-bot (DEVOPS-8697) wants `/cr7/reviews` and all its sub-paths
(`/cr7/reviews/<uuid>/content`, ...) readable by any authenticated user, but the uuid segment makes
exact-match rules impossible. review-bot's `auth.json` already carries an (inert) `/cr7/reviews/*`
rule that will activate when this ships.

## Functional Requirements

### FR-1: Trailing-`*` prefix matching

**Description:** A rule url whose last character is `*` (and which is not the bare `*` sentinel)
matches any request path that begins with the rule url minus the trailing `*`, compared
case-insensitively.

**Acceptance Criteria:**
- Rule `/cr7/reviews/*` matches `/cr7/reviews/`, `/cr7/reviews/abc`, and
  `/cr7/reviews/6f1c.../content`.
- Rule `/cr7/reviews/*` does **not** match `/cr7/review`, `/cr7`, or `/other/cr7/reviews/x`.
- Matching is case-insensitive on the prefix portion: rule `/CR7/Reviews/*` matches
  `/cr7/reviews/abc`.
- Bare-prefix edge: rule `/x*` matches `/x` (request path exactly equal to the prefix) and `/xyz`.
- A request path shorter than the prefix never matches.
- The empty-prefix case (rule url is exactly `*`) is handled by the existing global sentinel branch
  and must not be reachable through the prefix branch.

**Priority:** P0

### FR-2: `pathMatches` helper extracted and used by `Policies.Match`

**Description:** The path decision moves out of the inline conditional in `Policies.Match` into a
single named helper in `http/auth/policy_rules.go`.

**Acceptance Criteria:**
- A helper with signature `func pathMatches(rulePath, reqPath string) bool` exists in
  `http/auth/policy_rules.go`.
- Semantics, in order: global sentinel (`rulePath == AllPaths`) → case-insensitive full equality →
  trailing-`*` prefix (only when the prefix is non-empty) → `false`.
- Reference implementation from the ticket:

  ```go
  func pathMatches(rulePath, reqPath string) bool {
      if rulePath == AllPaths || strings.EqualFold(rulePath, reqPath) {
          return true
      }
      if prefix, ok := strings.CutSuffix(rulePath, "*"); ok && prefix != "" {
          return len(reqPath) >= len(prefix) && strings.EqualFold(prefix, reqPath[:len(prefix)])
      }
      return false
  }
  ```

- `Policies.Match` calls `pathMatches(item.HttpPath, req.URL.Path)` in place of the inline path
  check. No other part of `Match` changes.

**Priority:** P0

### FR-3: Backward compatibility

**Description:** No existing policy file changes behaviour.

**Acceptance Criteria:**
- Global `*` rules still match every path.
- Exact-path rules still match case-insensitively and only on full equality.
- Method matching (`AllMethods` / `strings.EqualFold`) is unchanged.
- Subject matching (`containsSetWithWildcard`) is unchanged.
- First-match-wins ordering over the `Policies` slice is unchanged; the returned `*Item` and the
  no-match error text are unchanged.
- Rationale: today a url ending in `*` (other than the bare `*` sentinel) can never match anything,
  so activating the prefix branch cannot alter the outcome of any rule that previously matched.

**Priority:** P0

### FR-4: Unit test coverage

**Description:** Tests covering the matching matrix listed in the ticket's acceptance criteria.

**Acceptance Criteria:** Tests exist and pass for:
- exact match
- global `*`
- trailing-`*` prefix, including the bare-prefix edge `"/x*"` vs `"/x"`
- case-insensitivity (rule and request differing in case, both directions)
- first-match-wins ordering unchanged
- no behaviour change for existing policy shapes (exact + global rules)

**Priority:** P0

### FR-5: Tagged release

**Description:** Cut a release so consumers can bump off `v0.1.0`.

**Acceptance Criteria:**
- `CHANGELOG.md` records the new path-matching capability.
- A git tag is pushed for the new version.
- review-bot (DEVOPS-8697) can bump `github.com/TouchBistro/gotham` from `v0.1.0` to the new tag and
  have its existing `/cr7/reviews/*` rule take effect with no code change on its side.

**Priority:** P0

## Non-Functional Requirements

### NFR-1: No new dependencies

Implementation uses only the standard library `strings` package (`strings.EqualFold`,
`strings.CutSuffix`). `strings.CutSuffix` is available in Go 1.20+; the module targets Go 1.22.

### NFR-2: Performance

`pathMatches` must be allocation-free and O(len(prefix)) per rule. No regex compilation, no path
splitting, no `strings.ToLower` allocations on the hot request path.

### NFR-3: Security — no accidental widening

The prefix branch must never be reachable with an empty prefix (which would silently turn a
malformed rule into a global allow). The `prefix != ""` guard is mandatory and must be covered by a
test.

### NFR-4: Coverage

>90% statement coverage for `http/auth/policy_rules.go` per `conductor/workflow.md`.

## User Stories

### US-1: Service owner writes one rule for a parameterized route family

**As a** service owner authoring `auth.json`
**I want** to write `"url": "/cr7/reviews/*"`
**So that** every sub-path under `/cr7/reviews` is covered by one rule without enumerating uuids.

- **Given** a policy rule `{ "method": "GET", "url": "/cr7/reviews/*", "effect": "allow", "subjects": ["*"] }`
- **When** an authenticated principal issues `GET /cr7/reviews/6f1c9e2a-.../content`
- **Then** `Policies.Match` returns that rule with `Effect == EffectAllow`.

### US-2: Existing deployments are unaffected

**As an** operator of a service already using gotham auth
**I want** the upgrade to be behaviour-neutral for my current policy file
**So that** I can bump the dependency without re-reviewing my rules.

- **Given** a policy file containing only exact-path and global `*` rules
- **When** the service upgrades to the new gotham version
- **Then** every request resolves to the same rule as before the upgrade.

### US-3: Prefix rules do not leak across sibling routes

**As a** security reviewer
**I want** `/cr7/reviews/*` to not match `/cr7/reviewsecret`... *(see Open Questions OQ-1)*

- **Given** a rule `/cr7/reviews/*` (trailing slash before the `*`)
- **When** a request hits `/cr7/reviewsomething`
- **Then** no match occurs, because the prefix `/cr7/reviews/` is not a prefix of that path.

## Technical Considerations

- **Single touch point:** the only production change is in `http/auth/policy_rules.go` — add
  `pathMatches`, replace the inline path conditional in `Policies.Match`.
- **New test file:** `http/auth/policy_rules_test.go` does not exist today; it must be created.
  Follow the existing style in `http/auth/basic_principal_test.go` /
  `http/auth/cache_principal_loader_test.go` (standard library `testing`, table-driven).
- **Test fixtures:** `Policies.Match` takes a `Principal` and an `http.Request`. Use
  `auth.BasicPrincipal` and `httptest.NewRequest` (or a hand-built `http.Request` with a parsed
  `*url.URL`) to drive `Match` tests.
- **Trailing-slash semantics:** matching operates on raw `req.URL.Path`; no normalization,
  cleaning, or trailing-slash folding is introduced by this change.
- **Godoc:** `Policies.Match` and `http/auth/doc.go` describe path matching; both should state the
  three supported url forms after this change.

## Out of Scope

- Mid-path or multi-segment wildcards (`/a/*/c`), `**`, glob, or regex rule urls.
- Named path parameters (`/reviews/:id`).
- Prefix or wildcard matching on `method`, or any change to subject/role matching.
- Path normalization (trailing slash folding, `path.Clean`, percent-decoding).
- Changing first-match-wins ordering, rule priority handling, or deny-precedence semantics.
- Any change to review-bot's `auth.json` (it already carries the `/cr7/reviews/*` rule).

## Open Questions

- **OQ-1:** Should a prefix rule be required to terminate on a path-segment boundary (i.e. should
  `/cr7/reviews/*` be prevented from matching a hypothetical `/cr7/reviews/../x` style path)? The
  ticket specifies plain string-prefix semantics; this spec implements exactly that. Segment-aware
  matching is deliberately out of scope.
- **OQ-2:** Release version. The ticket says only "tag a release so consumers can bump from
  v0.1.0". Recommendation: `v0.2.0` (additive capability, no breaking change). Confirm before
  tagging.
