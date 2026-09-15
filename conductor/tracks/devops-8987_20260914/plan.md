# Implementation Plan: Cost Explorer Saved-Report Library (`aws/cereport`)

**Track ID:** devops-8987_20260914
**Jira:** DEVOPS-8987
**Branch:** `feat/devops-8987`
**Source:** `devops-go-tools` @ `dfa45b1`, `cost/savedreport/`

## Overview

Three phases. Phase 1 moves six files verbatim (package clause only) and adds the two SDK
dependencies. Phase 2 lifts coverage from the measured 63.0% baseline to ≥ 90% with synthetic,
offline fixtures. Phase 3 documents the package, bumps the release tag, and opens the PR.

Red → Green → Refactor per `conductor/workflow.md`. Phase 1's "Red" is the moved test files failing
to compile before their source files exist. Commits per task with the short SHA recorded here;
checkpoint commit per phase. Every `go` / `golangci-lint` invocation in this track uses
`GOCACHE=$TMPDIR/gocache GOLANGCI_LINT_CACHE=$TMPDIR/golangci` (sandbox constraint, see spec).

---

## Phase 1: Move the Library [checkpoint: 9be68bc]

**Goal:** `aws/cereport` compiles, its moved tests pass, `go.mod` carries exactly the two new direct
requires, and the moved files differ from source only in their `package` clause.

### Tasks

- [x] **Task 1.1: Add AWS SDK dependencies** (FR-2) [f70565f]
  - **Note (2026-09-14):** `go mod tidy` pruned the two requires because nothing imported them yet, so this task produced no standalone diff. The `go.mod`/`go.sum` change landed with Task 1.2 in `f70565f`.
  - `go get github.com/aws/aws-sdk-go-v2@v1.45.1 github.com/aws/aws-sdk-go-v2/service/costexplorer@v1.69.1`
  - `go mod tidy` — expect the two modules to drop to `// indirect` until Task 1.2 imports them;
    that is fine, Task 1.2 re-runs tidy.
  - Assert no `replace` directive; assert no `aws-sdk-go-v2/config` anywhere in `go.mod`.
  - Verification: `go build ./...` still green.

- [x] **Task 1.2: Copy library files, rename package, add doc.go** (FR-1, FR-3) [f70565f]
  - **Red:** `cp` `url_test.go` and `run_test.go` from
    `/Users/esiddiqui/Projects/devops/devops-go-tools/cost/savedreport/` into `aws/cereport/`,
    `sed` the `package savedreport` line to `package cereport`. Run
    `go test ./aws/cereport/` and confirm compile failure (undefined `ParseURL`, `Run`, ...).
  - **Green:** `cp` `types.go`, `url.go`, `translate.go`, `run.go`; same `package` edit. Move the
    package doc comment (top of `types.go`) into a new `doc.go`, rewritten to name
    `cereport` and extended with a "Basic usage" block (build client via
    `costexplorer.NewFromConfig`, `LoadSpecs`, `Find`, `Run`, `WriteCSV`) in the style of
    `circleci/doc.go`. Leave the SDK import unaliased (no clash with package name).
    `go mod tidy` so the two requires become direct.
  - Run `go test ./aws/cereport/` — moved tests pass unmodified.
  - **Refactor:** none. Confirm with
    `diff <src>/url.go aws/cereport/url.go` (and the other five) that only the `package` line
    differs (`types.go` additionally loses the doc comment).

- [x] **Task 1.3: Verification — Phase 1** [checkpoint marker] [9be68bc]
  - **Results (2026-09-14):** `gofmt -l aws/cereport` empty; `go vet` clean; `golangci-lint run ./aws/cereport/...` → 0 issues;
    `go build ./...` ok; `go test -count=1 ./...` ok for every package (aws/cereport, circleci, http/auth, shipit, slack, sql/qb, sql/qb/tmp).
  - **go.mod:** direct `+github.com/aws/aws-sdk-go-v2 v1.45.1`, `+github.com/aws/aws-sdk-go-v2/service/costexplorer v1.69.1`;
    indirect `+smithy-go v1.28.1`, `+internal/configsources v1.5.1`, `+internal/endpoints/v2 v2.8.1`. No `replace`, no `config`.
  - **Baseline coverage:** `go test -cover ./aws/cereport/` → 63.0% (matches the pre-move measurement without the golden test).
  - `gofmt -l aws/cereport` empty; `go vet ./aws/cereport/...` clean.
  - `golangci-lint run ./aws/cereport/...` zero issues.
  - `go build ./...` and `go test ./...` green module-wide.
  - `go.mod` review: direct requires = existing + `aws-sdk-go-v2` + `service/costexplorer`;
    indirect = `smithy-go`, `internal/configsources`, `internal/endpoints/v2`. Record the exact
    versions here.
  - `go test -cover ./aws/cereport/` — record baseline (expected ≈ 63%).

---

## Phase 2: Coverage to ≥ 90% [checkpoint: 2c81e64]

**Goal:** every path the departing golden test covered is pinned by synthetic fixtures, the
zero-coverage helpers are exercised, and the package reports ≥ 90% statement coverage offline.

### Tasks

- [x] **Task 2.1: Loader, lookup, grid, CSV and pagination tests** (FR-4) — `run_test.go` [2efc886]
  - **Note:** `fakeCE` gained `tokens` (records each request's NextPageToken) and `err` fields; otherwise the moved tests are untouched.
  - **Red:** add table-driven tests and confirm they fail or are uncovered before assertions are
    tightened (these are tests against existing code, so "Red" here means writing the assertion
    first and watching coverage move):
    - `TestLoadSpecs`: valid JSON array in `t.TempDir()`; missing file → error; malformed JSON →
      error mentioning the path.
    - `TestFind`: hit returns the pointer; miss error lists sorted names.
    - `TestRun_Pagination`: two pages via `NextPageToken`, same group in both → amounts summed
      per period; periods discovered on page 2 extend existing rows with zeros.
    - `TestRun_APIError`: fake returns an error → `Run` returns it, no partial `Result`.
    - `TestRun_MetricErrors`: missing metric key; nil `Amount`; unparsable amount.
    - `TestWriteCSV_Grouped`: two groups, two periods; assert header
      `<key>,<p1>,<p2>,Total`, rows in descending-total order, name tie broken alphabetically,
      per-row total, trailing `Total` row; amounts at full precision
      (`strconv.FormatFloat(v,'f',-1,64)`), e.g. `0.1+0.2` renders `0.30000000000000004`.
    - `TestWriteCSV_Ungrouped`: header first cell `Total`, single `(total)` row.
    - `TestResult_GrandTotal_SortedKeys`: direct unit tests including the tie-break branch.
    - `TestGroupHeader`: no groups → `Total`; two groups → `A / B`.
  - **Green:** no production change expected.

- [x] **Task 2.2: URL parser tests with synthetic fixtures** (FR-4) — `url_test.go` [e553148]
  - **Red:** add a small helper that builds a fragment URL from `url.Values` so fixtures stay
    readable, then:
    - `TestParseURL_TagGroupBy`: `groupBy=["TagKeyValue:repo"]` → one `TAG`/`repo` group
      (pins the golden `ytd_ecs_by_repo_amortized` behaviour).
    - `TestParseURL_TagFilterRow`: filter row `dimension.id=TagKey`, `growableValue.value=groupid`,
      `values=[venue]` → `Filter{Type:TAG, Key:groupid, Values:[venue]}`
      (pins `ytd_venue_ark_by_service_amortized`).
    - `TestParseURL_TrimsName`: `reportName=%20foo` → `foo`.
    - `TestParseURL_EmptyGroupByAndFilter`: no `groupBy`, `filter=[]` → nil slices.
    - `TestParseURL_NormalizedUnits`: `useNormalizedUnits=true&costAggregate=undefined` →
      `NormalizedUsageAmount`.
    - `TestParseURL_Rejects` additions: tag row without `growableValue`; `TagKeyValue:` with
      empty key; unknown operator; filter with no values; fragment without `?`; invalid query
      escape (`%zz`); malformed `groupBy` JSON; malformed `filter` JSON;
      `showOnlyUncategorized=true`; `reportMode=SAVINGS_PLANS`.
  - **Green:** no production change expected.

- [x] **Task 2.3: Translation tests** (FR-4) — new `translate_test.go` [5893223]
  - **Red:**
    - `TestExpression`: zero filters → nil; one filter → bare expression (no `And`); two filters →
      `And` with two elements; `TAG` → `Tags`; `COST_CATEGORY` → `CostCategories`; unknown type →
      error; `Exclude` on `TAG` → `Not{Tags}`.
    - `TestResolvePeriod_Errors`: `CUSTOM` without dates; `Relative` empty without dates;
      `LAST_WEEK` (unhandled) → error naming the report.
    - `TestResolvePeriod_LastNDays`: `LAST_30_DAYS` with `ExcludeCurrentDay`.
    - `TestGetCostAndUsageInput_MultiGroup`: two group-bys → two `GroupDefinition`s in order;
      propagation of `ResolvePeriod` and `Expression` errors.
  - **Green:** no production change expected.

- [x] **Task 2.4: Verification — Phase 2** [checkpoint marker] [2c81e64]
  - **Results (2026-09-14):** `go test -count=1 -cover ./aws/cereport/` → **99.3%** (target ≥ 90%). Every function at 100% except
    `addAt` 85.7% (the defensive `for len(row) < len(r.Periods)` growth loop: `Run` already pads rows when a new period appears) and
    `WriteCSV` 96.2% (write error on the Total row only; header, row and final-flush error paths are covered).
  - `golangci-lint run ./aws/cereport/...` → 0 issues; `go vet` clean; `gofmt -l` empty.
  - `go test -count=1 -cover ./aws/cereport/` ≥ 90%; record the figure and any function still
    below 80% from `go tool cover -func`.
  - `golangci-lint run ./aws/cereport/...` zero issues (watch `errcheck` on any `f.Close()` in new
    tests — check the error or use `t.Cleanup`).
  - `go test ./...` green module-wide.

---

## Phase 2b: Purge the console-URL capture path (scope change 2026-09-14) [checkpoint: afc10ee]

**Goal:** gotham's `aws/cereport` contains only report generation. No URL parsing, no
parser-only `Spec` fields, no capture-time wording in docs. Coverage stays ≥ 90%.

**Why:** requester review after Phase 2: `ParseURL` and friends are tooling for the one-time
spec dump (`urls.txt` → `specs.json`), unrelated to running reports; that belongs with the data in
cerep. `ReportID`/`ReportARN`/`ChartStyle` are written by the parser and never read.

### Tasks

- [x] **Task 2b.1: Remove `url.go`, parser tests and parser-only fields** (FR-1, FR-4) [9fbc3bd]
  - `git rm aws/cereport/url.go aws/cereport/url_test.go`.
  - Move `relativeCustom` into `types.go`; drop `ReportID`, `ReportARN`, `ChartStyle` from `Spec`;
    reword `Spec`/`Group`/`Filter`/`TimeRange` comments without the console-URL framing.
  - Move the four translate tests that lived in `url_test.go` (`TestResolvePeriod`,
    `TestResolvePeriod_ExcludeCurrentDay`, `TestResolvePeriod_Custom…`,
    `TestGetCostAndUsageInput_ExcludeBecomesNot`) into `translate_test.go`; the last one builds its
    input from a literal `Spec` instead of `ParseURL`.
  - Rewrite `doc.go` (spec JSON example replaces the capture snippet); fix the `LoadSpecs` comment.
  - Verify no reference to `ParseURL`, `ReportID`, `ReportARN`, `ChartStyle` remains.

- [x] **Task 2b.2: Verification — Phase 2b** [checkpoint marker] [afc10ee]
  - **Results (2026-09-14):** `go test -count=1 -cover ./aws/cereport/` → **98.9%**; only `addAt` (85.7%) and `WriteCSV` (96.2%) below 100%, same defensive branches as Phase 2.
  - `golangci-lint` 0 issues; `go vet` ok; `gofmt -l` empty; `go build ./...` ok. Package is now 6 files: `doc.go`, `types.go`, `translate.go`, `run.go` + two test files.
  - `tech-stack.md` tree line reads "checked-in Spec → GetCostAndUsage → CSV grid".
  - `go test -count=1 -cover ./aws/cereport/` ≥ 90%; `golangci-lint` 0 issues; `go vet`, `gofmt -l`
    clean; `go build ./...` ok.
  - `conductor/tech-stack.md` package tree line no longer mentions console URLs.

---

## Phase 3: Documentation and Release [checkpoint: 7e015cd]

**Goal:** the package is documented for consumers, the release bump is staged, the full Makefile
gate is green, and a PR is open against `master`.

### Tasks

- [x] **Task 3.1: Package README and root README line** (FR-3) [860f40a]
  - `aws/cereport/README.md` following `slack/README.md`: purpose (checked-in Spec → API → CSV),
    install, spec JSON example (fields, relative vs CUSTOM ranges),
    `LoadSpecs`/`Find`, building a client (`config.LoadDefaultConfig` + `costexplorer.NewFromConfig`
    — caller's responsibility, region `us-east-1`), `Run` + `WriteCSV`, `ExcludeCurrentDay`
    semantics, CSV shape (groups as rows, ISO-dated columns, trailing `Total`), required IAM
    action `ce:GetCostAndUsage`, testing with a fake `CostExplorerAPI`, and a pointer to cerep for
    capturing specs from console URLs.
  - Root `README.md`: one bullet naming `aws/cereport`.
  - Documentation-only; verified by review and `go vet`.

- [x] **Task 3.2: Release tag** (FR-5) [c142b2a]
  - Set `.circleci/tag.dat` to `v0.2.0` (single line, trailing newline).
  - Confirm with `git tag --sort=-v:refname | head -1` that `v0.2.0` > current (`v0.1.1`) so
    `tagger.sh` uses the proposed value instead of auto-bumping to `v0.1.2`.

- [x] **Task 3.3: Verification — Phase 3** [checkpoint marker] [7e015cd]
  - **Results (2026-09-14):** `make build` ok; `make lint` → 0 issues; `make test` → every package ok. `aws/cereport` alone: **98.9%**
    (`make test` prints module-wide `-coverpkg=./...` figures, 11.7% for this package, which are not comparable). `make test-ci` module total 68.4% (pre-existing baseline).
  - `gofmt -l .` lists four pre-existing files outside this track (`circleci/types.go`, `sql/qb/{db_mock,entity,query}_test.go`); `aws/cereport` is clean. Left untouched.
  - `git diff master --name-only`: only `aws/cereport/**`, `go.mod`, `go.sum`, `README.md`, `.circleci/tag.dat`, `conductor/**`.
  - Push + CircleCI confirmation recorded under Task 3.4.
  - `make build`, `make lint`, `make test` (with cache env overrides) all green; paste the
    `aws/cereport` coverage line from `make test-ci` here.
  - `git diff master --stat` review: only `aws/cereport/**`, `go.mod`, `go.sum`, `README.md`,
    `.circleci/tag.dat`, `conductor/**` touched.
  - Push branch; confirm CircleCI `build-lint-test` green.

- [x] **Task 3.4: Open PR** (FR-5) [ba6406a]
  - `gh pr create --base master` with title
    `feat(aws/cereport): add Cost Explorer saved-report library [DEVOPS-8987]`; body lists the
    file mapping table from the ticket (adjusted to `aws/cereport`), the two new direct requires,
    the coverage figure, the `v0.2.0` bump, and the deviations from the ticket text.
  - Record the PR URL here and on the Jira ticket. Mark `tracks.md` entry `[x]` on merge.
  - **PR:** https://github.com/TouchBistro/gotham/pull/17 — `feat/devops-8987` → `master`, opened 2026-09-14.
    Title: `feat(aws/cereport): add Cost Explorer report runner from devops-go-tools cereport [DEVOPS-8987]`.
  - Jira follow-up comment posted with the scope change and PR link.
  - **CircleCI:** `build-lint-test` passed on `ba6406a` (build 47, workflow `build-test-release`). The `release` job runs only on `master` after merge.

---

## Phase 4: API refinement after review (2026-09-14) [checkpoint: c7e6280]

**Goal:** gotham's `aws/cereport` is a pure report engine with a self-describing, validated
`Spec`. Spec loading and selection move to the client (cerep).

**Why:** requester review of PR #17: `LoadSpecs`/`Find` are the client's config layer (file
path, JSON array, terminal-formatted error) and add nothing over `json.Unmarshal`; `Spec` had no
validation, so bad specs failed late or cost an API request; consumers had to know magic strings.

### Tasks

- [x] **Task 4.1: Remove `LoadSpecs` and `Find`** (FR-1) [daaaf21]
  - Delete both from `run.go` (with the `os` and `encoding/json` imports) and their tests.
  - Hand the requester a drop-in snippet for `cerep/cmd/cer/main.go` (`loadSpecs`, `findSpec`).

- [x] **Task 4.2: Constants and `Spec.Validate()`** (FR-6) [0a17e80]
  - `types.go`: exported `Type*`, `Metric*`, `Granularity*`, `Range*` constants plus
    `RangeLastDays(n)` / `RangeLastMonths(n)`; `dateLayout`; field comments point at them.
  - `validate.go`: `Validate()` collects every problem with `errors.Join`; `validateKey`;
    `TimeRange.validate`. Metrics hand-kept (API takes CamelCase; SDK enum is SCREAMING_SNAKE);
    granularities, group types and the 35 dimensions come from the SDK enums' `Values()`.
    Exported `Metrics()`, `Granularities()`, `Dimensions()` for help text.
  - `translate.go`: `GetCostAndUsageInput` calls `Validate` first; literals replaced by constants.
  - **Red:** `validate_test.go` — valid specs (5), one test per rule (18), all-problems-at-once,
    range helpers, value lists, `GetCostAndUsageInput` rejects before building.
  - **Green:** implementation above.

- [x] **Task 4.3: README and doc.go rewrite** (FR-3) [454ebf2]
  - README: quick start; Spec field table with constants; four Spec examples (YTD by service,
    daily by region for one tag, single-number MTD by cost category, fixed window with two
    group-bys); Validation with sample output and rule list; loading from JSON via
    `json.Unmarshal`; `Result` field table; five Result examples (CSV, top-N share, single
    number, tag prefix stripping, period-over-period); time-range table; fake for tests;
    permissions/cost/limits.
  - `doc.go`: Spec literal in the usage example; mentions Validate and the constants.

- [x] **Task 4.4: Verification — Phase 4** [checkpoint marker] [c7e6280]
  - **Results (2026-09-14):** `go test -count=1 -cover ./aws/cereport/` → **98.3%**. Below 100%: `addAt` 85.7%, `WriteCSV` 96.2%
    (as before) and `GetCostAndUsageInput` 81.8% — its `ResolvePeriod`/`Expression` error returns are unreachable now that
    `Validate` runs first; kept as defensive code. `golangci-lint` 0 issues; `go vet` ok; `gofmt -l` empty; `go build ./...` ok.
  - Exported surface now: `Spec`, `Group`, `Filter`, `TimeRange`, `Result`, `CostExplorerAPI`, `PeriodOption`, `Run`, `ExcludeCurrentDay`,
    `Spec.{Validate,ResolvePeriod,GetCostAndUsageInput,Expression}`, `TimeRange.IsCustom`, `Result.{Total,GrandTotal,SortedKeys,WriteCSV}`,
    `Metrics`, `Granularities`, `Dimensions`, `RangeLastDays`, `RangeLastMonths`, and the `Type*`/`Metric*`/`Granularity*`/`Range*` constants.
  - `go test -cover ./aws/cereport/` ≥ 90%; lint 0; vet; gofmt; `make build lint test`.
  - Push to PR #17; CircleCI green; Jira follow-up.
  - **Done (2026-09-14):** `make build` / `make lint` (0 issues) / `make test` green; pushed `b45b52a`; CircleCI `build-lint-test` passed (build 49);
    PR #17 description updated; Jira round-2 comment posted; cerep `loadSpecs`/`findSpec` snippet handed to the requester.

---

## Risks and Dependencies

| Item | Notes |
|------|-------|
| Coverage gap | Baseline 63.0% without the golden test; ticket assumed a small delta. Phase 2 is sized for ~25 new test cases across three files. If a helper resists coverage without contortion (e.g. `indexOf` miss branch), document it here rather than adding production code. |
| First AWS dependency | Consumers see `go.sum` growth only. If anyone objects, the pruning argument is in the spec (NFR-2). |
| Tag bump | If another change merges first and tags `v0.1.2`, `v0.2.0` still wins in `tagger.sh` (`sort -V`). If someone else stages a `v0.2.0`, coordinate. |
| Downstream `cerep` (Step 2) | Blocked on `v0.2.0` being tagged. Until then it develops against `../gotham` via an uncommitted `go.work`. Import path will be `github.com/TouchBistro/gotham/aws/cereport` (not `aws/costexplorer` as the ticket text says) — the ticket needs a one-line correction. |
| `devops-go-tools` cleanup (Step 3) | Not started; the deleted-but-tracked `cereport` binary shows as ` D` in that repo's working tree. Leave for Step 3. |
| Sandbox caches | `~/Library/Caches` is read-only in this session; forgetting the `GOCACHE`/`GOLANGCI_LINT_CACHE` overrides yields `operation not permitted`, not a code problem. |
