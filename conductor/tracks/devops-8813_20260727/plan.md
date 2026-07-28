# Implementation Plan: Trailing-`*` Prefix Matching in Authr Policy URL Paths

**Track ID:** devops-8813_20260727
**Jira:** DEVOPS-8813
**Branch:** `feat/devops-8813`

## Overview

Small, single-file behaviour change plus first-ever test coverage for
`http/auth/policy_rules.go`. Two phases:

1. **Phase 1 — Path matching helper and `Policies.Match` integration:** TDD the `pathMatches`
   helper, wire it into `Policies.Match`, and prove first-match-wins ordering plus backward
   compatibility are unchanged.
2. **Phase 2 — Documentation and release:** update godoc + `CHANGELOG.md`, tag the release so
   review-bot can bump off `v0.1.0`.

All tasks follow Red → Green → Refactor per `conductor/workflow.md`. Tests use the standard library
`testing` package, table-driven, co-located as `http/auth/policy_rules_test.go` (new file — none
exists today).

---

## Phase 1: Path Matching Helper and `Policies.Match` Integration

**Goal:** `pathMatches` exists, is used by `Policies.Match`, and supports global `*`, exact
case-insensitive equality, and trailing-`*` prefix — with no change to method, subject, or ordering
behaviour.

### Tasks

- [ ] **Task 1.1: TDD `pathMatches` helper** (FR-1, FR-2, NFR-3)
  - **Red:** create `http/auth/policy_rules_test.go` with a table-driven `TestPathMatches`
    covering:
    - global sentinel: rule `*` vs `/anything`, `/`, `` (empty)
    - exact equality: `/cr7/reviews` vs `/cr7/reviews` → true; vs `/cr7/reviews/` → false
    - case-insensitive equality: `/CR7/Reviews` vs `/cr7/reviews` → true (and reverse casing)
    - prefix: `/cr7/reviews/*` vs `/cr7/reviews/`, `/cr7/reviews/abc`,
      `/cr7/reviews/<uuid>/content` → true
    - prefix negatives: `/cr7/reviews/*` vs `/cr7/reviews` (no trailing slash), `/cr7/review`,
      `/cr7`, `/other/cr7/reviews/x` → false
    - bare-prefix edge: `/x*` vs `/x` → true; `/x*` vs `/xyz` → true; `/x*` vs `/y` → false;
      `/x*` vs `` → false
    - prefix case-insensitivity: `/CR7/Reviews/*` vs `/cr7/reviews/abc` → true
    - empty-prefix guard: request path shorter than prefix → false; and confirm a rule of exactly
      `*` resolves through the sentinel branch (not the prefix branch) — i.e. `*` still matches
      everything and no rule can produce an empty prefix match
    - non-wildcard rule that merely *contains* `*` mid-string (e.g. `/a*b`) → only exact-equality
      semantics apply → false against `/aXb`
  - Run `go test ./http/auth/...` and confirm compile failure / test failure (helper absent).
  - **Green:** add `pathMatches(rulePath, reqPath string) bool` to `http/auth/policy_rules.go`
    using `strings.CutSuffix` with the mandatory `prefix != ""` guard, per FR-2. Add GoDoc comment
    documenting the three supported url forms.
  - **Refactor:** keep the helper allocation-free (no `ToLower`, no `Split`); confirm `strings` is
    the only import touched.

- [ ] **Task 1.2: Wire `pathMatches` into `Policies.Match`** (FR-2, FR-3)
  - **Red:** add `TestPolicies_Match` to `http/auth/policy_rules_test.go` driving the real
    `Policies.Match(ctx, principal, req)` path with `BasicPrincipal` + `httptest.NewRequest`:
    - prefix rule `/cr7/reviews/*` + method `GET` + subjects `["*"]` matches
      `GET /cr7/reviews/<uuid>/content` and returns that `*Item`
    - same rule does not match `GET /cr7/review` → error returned, `*Item` nil
    - method still gates: rule method `GET` vs request `POST` on a matching prefix → no match
    - subject still gates: rule subjects `["admin"]` vs principal roles `["viewer"]` on a matching
      prefix → no match
    - global `*` rule still matches any method/path combination
    - exact-path rule still matches case-insensitively
  - Confirm the prefix-rule assertions fail against the current inline conditional.
  - **Green:** replace the inline
    `item.HttpPath == AllPaths || strings.EqualFold(item.HttpPath, req.URL.Path)` check in
    `Policies.Match` with `pathMatches(item.HttpPath, req.URL.Path)`. Change nothing else in
    `Match` — same trace/debug logging, same returned `*Item`, same no-match error text.

- [ ] **Task 1.3: First-match-wins ordering and backward-compatibility regression tests**
      (FR-3, FR-4)
  - **Red:** add `TestPolicies_Match_Ordering` and `TestPolicies_Match_BackwardCompatible`:
    - ordering: a `deny` exact-path rule listed before an `allow` prefix rule wins for the exact
      path; the prefix rule wins for sub-paths — asserts the first matching rule in slice order is
      returned regardless of the new prefix capability
    - ordering: an `allow` prefix rule listed before a `deny` exact rule wins for the overlapping
      path (proves no reordering / no specificity preference was introduced)
    - backward compatibility: build a policy list containing only exact-path and global `*` rules
      (mirroring the shape of `defaultConfig()` in `http/auth/policy.go`) and assert every case
      resolves to the same rule it would have pre-change
    - backward compatibility: a legacy rule ending in `*` that previously matched nothing (e.g.
      `/legacy/*`) now matches sub-paths — assert this is the *only* class of behaviour delta
  - **Green:** no production change expected. If a test fails, fix `pathMatches` / `Match` and
    document the deviation in this plan.

- [ ] **Task 1.4: Verification — Phase 1** [checkpoint marker]
  - Run `go test -cover ./http/auth/...`; confirm >90% statement coverage on
    `http/auth/policy_rules.go`.
  - Run `go test ./...` to confirm no regression elsewhere in the module.
  - Run `go vet ./http/auth/...` and `gofmt -l http/auth/`.
  - Confirm `pathMatches` has a GoDoc comment and that `Policies.Match` behaviour outside the path
    check is byte-for-byte unchanged (`git diff http/auth/policy_rules.go` review).

---

## Phase 2: Documentation and Release

**Goal:** the new url semantics are documented for consumers, and a tagged version exists that
review-bot can bump to.

### Tasks

- [ ] **Task 2.1: Document path-matching semantics** (FR-2, FR-5)
  - Update the `Policies.Match` GoDoc in `http/auth/policy_rules.go` to enumerate the three
    supported `url` forms: global `*`, exact case-insensitive path, trailing-`*` case-insensitive
    prefix.
  - Update the "Config and policy matching" section of `http/auth/doc.go` with the same three
    forms and a `/cr7/reviews/*` example.
  - Note explicitly that no other wildcard form (mid-path `*`, `**`, regex, `:param`) is supported.
  - TDD note: documentation-only task; verified by `go vet ./http/auth/...` and doc review, no new
    tests required.

- [ ] **Task 2.2: CHANGELOG entry and release tag** (FR-5)
  - Add a `CHANGELOG.md` entry for the new version describing trailing-`*` prefix matching in
    authr policy urls and calling out that the change is backward compatible.
  - Confirm the version number with the requester (spec OQ-2; recommendation `v0.2.0`).
  - Tag and push the release once the branch is merged to `master`.
  - Record the tag in this plan so DEVOPS-8697 (review-bot) can reference it when bumping from
    `v0.1.0`.

- [ ] **Task 2.3: Verification — Phase 2** [checkpoint marker]
  - Run `go test ./...` on `master` at the tagged commit.
  - Verify the tag resolves: `go list -m github.com/TouchBistro/gotham@<tag>`.
  - Manual verification: in a scratch module, load a policy containing
    `{ "method": "GET", "url": "/cr7/reviews/*", "effect": "allow", "subjects": ["*"] }` via
    `auth.LoadConfigFromFile` and confirm `Policies.Match` returns that rule for
    `GET /cr7/reviews/6f1c9e2a-0000-0000-0000-000000000000/content` and returns an error for
    `GET /cr7/review`.

---

## Risks and Dependencies

| Item | Notes |
|------|-------|
| Downstream dependency | DEVOPS-8697 (review-bot) is blocked on the Phase 2 tag; its `/cr7/reviews/*` rule is already deployed and inert. |
| Accidental widening | An empty prefix would turn a malformed rule into a global allow. Guarded by `prefix != ""` and covered explicitly in Task 1.1. |
| No existing test file | `http/auth/policy_rules_test.go` is new; `Policies.Match` has zero coverage today, so Phase 1 also establishes the baseline for method/subject/ordering behaviour. |
| Version choice | OQ-2 unresolved — confirm `v0.2.0` vs `v0.1.1` before tagging. |
