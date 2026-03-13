# Implementation Plan: gotham — Add Query Builder (qb) Package

Track ID: devops-8473_20260313
Jira: DEVOPS-8473

## Overview

This track migrates the `qb` package from `devops-api-service` to gotham at `sql/qb`. The work is structured in three phases:

- **Phase 1** — Scaffolding: create the directory structure, add the `lib/pq` dependency, create the `tmp` helper package, and update `tech-stack.md`.
- **Phase 2** — Migration and Refactoring: copy active source files, replace internal imports, remove empty files, ensure the package compiles and all tests pass.
- **Phase 3** — Documentation: write `sql/qb/README.md` and verify godoc coverage via `qb.go`.

Commit strategy: one commit per phase (per-phase commits as configured in `workflow.md`).
Coverage target: >90% for `sql/qb/...`.

---

## Phase 1: Scaffolding and Dependency Setup

Goal: Establish the target package skeleton, add the required `lib/pq` dependency, create the `tmp` helper sub-package, and document the new dependency in `tech-stack.md`.

Tasks:

- [x] Task: Create directory `sql/qb/` and `sql/qb/tmp/` in the gotham repository. Create `sql/qb/tmp/ptr.go` containing `ToStringPtr` and `ToInt64Ptr` helpers with a comment noting this is a temporary holding package pending refactoring. (TDD: Write a test `sql/qb/tmp/ptr_test.go` that asserts both functions return a pointer to the supplied value — confirm tests fail, implement, confirm tests pass.) [43fbfad]

- [x] Task: Add `github.com/lib/pq` as a direct dependency to `go.mod` by running `go get github.com/lib/pq` from the gotham module root. Verify `go.mod` and `go.sum` are updated. (TDD: No logic test needed here; verification is that `go build ./...` succeeds after the dependency is added.) [43fbfad]

- [x] Task: Update `conductor/tech-stack.md` to document `github.com/lib/pq` as a new direct dependency with its purpose (PostgreSQL array parameter support via `pq.Array`). [43fbfad]

- [ ] Verification: Run `go build ./sql/qb/tmp/...` and `go test ./sql/qb/tmp/...` — both must pass. Confirm `go.mod` lists `github.com/lib/pq`. [checkpoint marker]

---

## Phase 2: Source Migration and Refactoring

Goal: Copy all active source files from the origin package into `sql/qb/`, update package declarations and import paths, replace the `tbutil` import with the `tmp` package, remove files that contain only commented-out code, and ensure the full package compiles and passes all tests.

Tasks:

- [x] Task: Copy `qb.go` to `sql/qb/qb.go`. Update `package` declaration if needed. No logic changes. (This is the godoc-only file — no test needed beyond compilation.) [f58c528]

- [x] Task: Copy `entity.go` to `sql/qb/entity.go`. Update package declaration. Verify no external imports are present. (TDD: Write `sql/qb/entity_test.go` testing `Convert` and `Convert0` with a minimal `Entity` stub — confirm tests fail, implement copy, confirm tests pass.) [f58c528]

- [x] Task: Copy `join.go` to `sql/qb/join.go`. Update package declaration. No external imports. (TDD: Write `sql/qb/join_test.go` asserting `LeftJoin.ToJoinString()`, `RightJoin.ToJoinString()`, and `InnerJoin.ToJoinString()` each return `""` — confirm tests fail, implement copy, confirm tests pass.) [f58c528]

- [x] Task: Copy `where.go` to `sql/qb/where.go`. Update package declaration. No external imports. (TDD: Write `sql/qb/where_test.go` asserting `WhereAll{}.WhereClause()` returns `"WHERE 1=1"`, `WhereNone{}.WhereClause()` returns `"WHERE 1=0"`, and `WhereString("WHERE x=1").WhereClause()` returns `"WHERE x=1"` — confirm tests fail, implement copy, confirm tests pass.) [f58c528]

- [x] Task: Copy `result.go` to `sql/qb/result.go`. Update package declaration. No external imports. (No dedicated test needed; compilation confirms the file is valid.) [f58c528]

- [x] Task: Copy `column_metadata.go` to `sql/qb/column_metadata.go`. Update package declaration. Replace import of `github.com/sirupsen/logrus` (already in gotham's `go.mod`; no change to `go.mod` needed). (TDD: Migrate `column_metadata_test.go` to `sql/qb/column_metadata_test.go`, update package declaration — confirm tests fail, implement copy, confirm tests pass.) [f58c528]

- [x] Task: Copy `table.go` to `sql/qb/table.go`. Update package declaration. Replace `github.com/TouchBistro/devops-api/tbutil` import with `github.com/TouchBistro/gotham/sql/qb/tmp`. Replace all `tbutil.ToStringPtr(...)` call sites with `tmp.ToStringPtr(...)`. Retain `github.com/lib/pq`, `github.com/pkg/errors`, and `github.com/sirupsen/logrus` imports. (TDD: Migrate `table_test.go` to `sql/qb/table_test.go`, update package declaration — confirm tests fail, implement copy+refactor, confirm tests pass. Tests cover: `TestTableMetadata`, `TestTableMetadata_Complex`, `TestColumnMetadata_SqlType`, `TestGenerateSelectSql`, `TestGenerateInsertSqlTemplate`, `TestGenerateUpdateSqlTemplate`, `TestGenerateDeleteSqlTemplate`.) [f58c528]

- [x] Task: Copy `query.go` to `sql/qb/query.go`. Update package declaration. Replace `github.com/TouchBistro/devops-api/tbutil` import with `github.com/TouchBistro/gotham/sql/qb/tmp`. Replace all `tbutil.ToStringPtr(...)` call sites with `tmp.ToStringPtr(...)`. Retain `github.com/lib/pq`, `github.com/pkg/errors`, and `github.com/sirupsen/logrus` imports. (TDD: Write `sql/qb/query_test.go` testing `ForQuery` initialization with a valid composite struct containing `LeftJoin`, `RightJoin`, or `InnerJoin` fields and asserting the generated `selectStmt` is non-empty and structurally correct — confirm tests fail, implement copy+refactor, confirm tests pass.) [f58c528]

- [x] Task: Confirm `table_join.go` and `functions.go` are NOT present in `sql/qb/` (they contain only commented-out code per FR-3). This is a verification-only step; no files to create. [f58c528]

- [x] Task: Run `go vet ./sql/qb/...` and resolve any reported issues. [f58c528]

- [x] Task: Run `go test -cover ./sql/qb/...` and verify coverage is >90%. Identify any uncovered paths and add tests to close gaps. Coverage: 94.9% [f58c528]

- [ ] Verification: Run `go build ./...` and `go test ./...` from the repository root — all packages must compile and all tests must pass. Coverage for `sql/qb` must be >90%. [checkpoint marker]

---

## Phase 3: Documentation

Goal: Write the `sql/qb/README.md` explaining package features, struct tag conventions, initialization patterns, and usage examples.

Tasks:

- [x] Task: Write `sql/qb/README.md`. The document must cover:
  - Package purpose and target use case.
  - Supported CRUD operations and how they map to PostgreSQL.
  - `qb` struct tag syntax reference table: tag key, token values (`pk`, `pkey`, `ops=r/a/w`, `type=`, `as=`), and their meaning.
  - `qbon` struct tag usage for join ON clauses.
  - How to initialize a `Table[T]` using `ForTable` (with and without explicit table name override).
  - How to initialize a `Query[T]` using `ForQuery` for multi-table join scenarios.
  - A minimal self-contained Go code example showing struct definition with tags, `ForTable` call, and a `Select` call.
  - Note on the `tmp` sub-package and its temporary nature. [347b453]

- [x] Task: Verify `sql/qb/qb.go` godoc comment is present and accurate. Update the package-level comment if needed to reflect the new import path (`github.com/TouchBistro/gotham/sql/qb`). [347b453]

- [ ] Verification: Run `go doc github.com/TouchBistro/gotham/sql/qb` and confirm the package doc renders correctly. Confirm `sql/qb/README.md` is present. [checkpoint marker]
