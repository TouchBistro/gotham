# Specification: Cost Explorer Saved-Report Library (`aws/cereport`)

**Track ID:** devops-8987_20260914
**Jira:** DEVOPS-8987 (sub-task of DEVOPS-8853)
**Type:** Feature
**Created:** 2026-09-14
**Branch:** `feat/devops-8987`

## Overview

Move the reusable half of `cereport` out of `devops-go-tools`
(`github.com/TouchBistro/goplayground/cost/savedreport`) into gotham as the new package
`aws/cereport`. The library loads checked-in report definitions (`Spec`), translates a `Spec` into a
`GetCostAndUsage` request, follows every result page into a `Result` grid, and writes that grid as
CSV. **Scope change 2026-09-14:** the console-URL → `Spec` capture path (`url.go`, `ParseURL`) is
*not* part of gotham; it belongs with the report data and its golden test (cerep). gotham holds only
report generation.

This is **Step 1** of the three-step split described in DEVOPS-8987. Step 2 (new repo `cerep`
holding the `cer` CLI and the 12 TouchBistro report definitions) and Step 3 (deleting
`cmd/cereport`, `cost/savedreport` and the tracked binary from `devops-go-tools`) are separate
pieces of work and are **out of scope** for this track.

## Background

- Source: `devops-go-tools` at commit `dfa45b1` ("commit cost explorer reports"). Step 0 of the
  ticket is already done at that commit: `@all` batch mode in the CLI, full-precision CSV in
  `run.go`, and `run_test.go` are all committed.
- `go list -deps ./cmd/cereport` confirms the CLI's only in-module dependency is
  `cost/savedreport`, and nothing else in `devops-go-tools` imports that package.
- `devops-go-tools` carries 26 direct dependencies, private modules, and a
  `replace github.com/TouchBistro/gotham => /Users/esiddiqui/...` local-path directive, so it
  builds on one laptop only. The library must live somewhere a CI runner can build.
- Library files pass golangci-lint defaults today. The only two `errcheck` findings are in
  `cmd/cereport/main.go` and `cost/savedreport/golden_test.go`, neither of which moves here.
- Measured baseline (2026-09-14): with the golden test excluded, `cost/savedreport` sits at
  **63.0%** statement coverage. Zero coverage: `LoadSpecs`, `Find`, `GrandTotal`, `SortedKeys`,
  `WriteCSV`, `groupHeader`. Below 70%: `expression` (55.6%), `resolveKey` (57.1%),
  `parseFilters` (65.4%). The golden test was the only exerciser of the tag-encoding paths.

### Deviations from the ticket text (agreed with requester, 2026-09-14)

| Ticket says | This track does | Why |
|-------------|-----------------|-----|
| Package path `aws/costexplorer` | Package path `aws/cereport`, package name `cereport` | Requester preference; keeps the tool's name. |
| Alias the SDK import as `ce` | SDK import left as `costexplorer`, unaliased | Package is no longer named `costexplorer`, so there is no name clash to disambiguate. Zero source diff beyond the `package` clause. |
| (implicit) `costexplorer.CostExplorerAPI` | `cereport.CostExplorerAPI`, name unchanged | No stutter with the new package name; honours "no logic changes". |
| Copy `url.go` + `url_test.go`; `ParseURL` exported from gotham | **Not copied.** `url.go` and the parser tests are excluded; `ParseURL` is not in gotham | Requester decision 2026-09-14: URL parsing is capture-time tooling for the spec dump, unrelated to running reports. It stays with `urls.txt` and the golden test in cerep. |
| Copy `Spec` verbatim | `ReportID`, `ReportARN`, `ChartStyle` removed from `Spec` | Requester decision 2026-09-14: written by the parser, never read by request building or CSV output. `TimeRange.Start`/`End` stay (read for CUSTOM ranges). |

## Functional Requirements

### FR-1: Package `aws/cereport` exposes the existing API unchanged

**Description:** Copy `types.go`, `translate.go`, `run.go`, `run_test.go` from `cost/savedreport`
into `aws/cereport/`. `url.go`/`url_test.go` are not copied (scope change above); the four translate
tests that lived in `url_test.go` move to `translate_test.go`. Request building and CSV output are
unchanged.

**Exported surface (must be identical):** `Spec`, `Group`, `Filter`, `TimeRange`,
`TimeRange.IsCustom`, `PeriodOption`, `ExcludeCurrentDay`, `Spec.ResolvePeriod`,
`Spec.GetCostAndUsageInput`, `Spec.Expression`, `LoadSpecs`, `Find`, `Result`, `Result.Total`,
`Result.GrandTotal`, `Result.SortedKeys`, `Result.WriteCSV`, `CostExplorerAPI`, `Run`.

**Acceptance Criteria:**
- `translate.go` and `run.go` differ from source only in the `package` line (`run.go` additionally
  in one `LoadSpecs` comment). `types.go` drops `ReportID`/`ReportARN`/`ChartStyle`, gains the
  `relativeCustom` constant (formerly in `url.go`), and its comments no longer describe URL capture.
- `gofmt -l aws/cereport` prints nothing; `go vet ./aws/cereport/...` is clean.
- The moved `url_test.go` and `run_test.go` pass unmodified.

**Priority:** P0

### FR-2: Dependencies

**Description:** gotham's first AWS dependency. The library accepts a `CostExplorerAPI` interface;
callers build the client, so `aws-sdk-go-v2/config` is **not** required.

**Acceptance Criteria:**
- `go.mod` gains exactly two new direct requires: `github.com/aws/aws-sdk-go-v2 v1.45.1` and
  `github.com/aws/aws-sdk-go-v2/service/costexplorer v1.69.1`.
- Indirect requires limited to `github.com/aws/smithy-go`,
  `github.com/aws/aws-sdk-go-v2/internal/configsources`,
  `github.com/aws/aws-sdk-go-v2/internal/endpoints/v2`.
- No `replace` directive. `go mod tidy` is a no-op afterwards.
- `conductor/tech-stack.md` documents the new dependencies and package before implementation.

**Priority:** P0

### FR-3: Package documentation

**Acceptance Criteria:**
- `aws/cereport/doc.go` holds the package comment (moved from `types.go`) plus a "Basic usage"
  example in the style of `circleci/doc.go` and `shipit/doc.go`.
- `aws/cereport/README.md` exists, shaped like `slack/README.md`: what it is, install, parse a
  console URL, load specs, run a report, write CSV, `ExcludeCurrentDay`, required IAM action
  `ce:GetCostAndUsage`.
- Root `README.md` gains one line naming the package.

**Priority:** P1

### FR-4: Test coverage ≥ 90%, fully offline

**Description:** All tests are offline against the in-package fake. No TouchBistro report
definitions are committed to gotham. (URL-parser fixtures written earlier in this track were
removed with `url.go` under the scope change.)

**Acceptance Criteria:**
- `go test -cover ./aws/cereport/` reports ≥ 90% statement coverage.
- All tests use the in-package fake `CostExplorerAPI`; no network, no AWS credentials.
- New tests cover at minimum: `LoadSpecs` (ok / missing file / bad JSON, via `t.TempDir()`),
  `Find` (hit / miss), `WriteCSV` (grouped and ungrouped: header, descending-total order,
  trailing `Total` row and column, full-precision amounts), `SortedKeys` tie-break,
  `GrandTotal`, `Run` pagination via `NextPageToken`, `Run` API error propagation,
  `metricValue` error paths, `Expression` with multiple filters (AND), `TAG` and `COST_CATEGORY` filter types,
  unknown filter type, custom range without dates, unhandled relative range, `LAST_N_DAYS`.

**Priority:** P0

### FR-5: Release

**Acceptance Criteria:**
- `.circleci/tag.dat` contains `v0.2.0` (new package = minor bump; file currently says `v0.1.0`,
  older than the current tag `v0.1.1`, which would otherwise auto-bump to `v0.1.2`).
- `make build lint test` green locally and in CircleCI `build-lint-test`.
- PR opened against `master` with title
  `feat(aws/cereport): add Cost Explorer saved-report library [DEVOPS-8987]`.
- On merge, CI tags `v0.2.0` and goreleaser publishes the GitHub release (verified post-merge,
  outside this track's commits).

**Priority:** P0

## Non-Functional Requirements

- **NFR-1 No behaviour change:** for the same `Spec`, `now`, and fake API responses, the moved
  code produces byte-identical CSV to `cost/savedreport` at `dfa45b1`.
- **NFR-2 Consumer isolation:** modules that depend on gotham but do not import `aws/cereport`
  compile no AWS code (Go module-graph pruning); only their `go.sum` grows.
- **NFR-3 Lint:** golangci-lint v2 defaults (`errcheck`, `govet`, `ineffassign`, `staticcheck`,
  `unused`) report zero issues for `aws/cereport`.
- **NFR-4 Style:** `conductor/code_styleguides/go.md`; table-driven tests with the standard
  `testing` package, co-located `*_test.go`.

## Constraints

- Go module directive stays `go 1.25.0`; local toolchain is go1.26.1.
- No `replace` directives in `go.mod`.
- Local sandbox denies writes to `~/Library/Caches/{go-build,golangci-lint}`; run `go` and
  `golangci-lint` with `GOCACHE` / `GOLANGCI_LINT_CACHE` under `$TMPDIR`. Artifacts unaffected.
- Copy files with plain `cp` from `devops-go-tools` at `dfa45b1`; no subtree split (single
  source commit, per ticket).

## Out of Scope

- Step 2: new repo `cerep`, `cer` CLI, `reports/specs.json`, `reports/urls.txt`, golden test.
- Step 3: deleting `cmd/cereport`, `cost/savedreport`, and the tracked binary from
  `devops-go-tools`.
- Runner credentials / assume-role (S3 scope).
- New library features: cost-category group-by, `showOnlyUntagged`, `showOnlyUncategorized`.
- Updating the rest of `tech-stack.md` (Go version line, missing `slack`/`shipit` entries) beyond
  what this track adds.

## Acceptance Criteria (from DEVOPS-8987, Step 1)

1. gotham `make build lint test` green.
2. `aws/cereport` at ≥ 90% coverage.
3. `go.mod` gains only core `aws-sdk-go-v2` + `service/costexplorer` as direct requires.
4. `.circleci/tag.dat` = `v0.2.0`; `v0.2.0` tagged and released on merge.

## Decision Log

| Date | Decision |
|------|----------|
| 2026-09-14 | Package path `aws/cereport` (not `aws/costexplorer`); library only, CLI and report data still go to `cerep`. |
| 2026-09-14 | Keep `CostExplorerAPI` name; leave SDK import unaliased. |
| 2026-09-14 | Conductor track created and plan posted to Jira; PR to be opened by the implementer at the end of Phase 3. |
| 2026-09-14 | Golden-test coverage replaced with synthetic URL fixtures rather than copying TouchBistro report data into gotham. |
| 2026-09-14 | **Scope change:** console-URL parsing (`url.go`, `ParseURL`, parser tests) purged from gotham; only report generation stays. Capture tooling belongs with the report data in cerep. |
| 2026-09-14 | `Spec` fields `ReportID`, `ReportARN`, `ChartStyle` removed (parser-written, never read). `Name` kept (lookup key, filenames, error messages); `TimeRange.Start`/`End` kept (CUSTOM ranges). |
