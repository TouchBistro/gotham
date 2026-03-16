# Specification: gotham — Add Query Builder (qb) Package

## Overview

Move the `qb` (query builder) package from `devops-api-service` into `gotham` as a reusable SQL utility library. The package provides automated CRUD query generation for PostgreSQL-backed Go microservices. It must be relocated to `sql/qb`, refactored to remove all `devops-api-service`-internal imports, and documented with a `README.md`.

## Background

The `qb` package currently lives in `github.com/TouchBistro/devops-api/db/qb` inside `devops-api-service`. To make it available as a shared library primitive across TouchBistro's Go microservice fleet, it is being relocated to `github.com/TouchBistro/gotham/sql/qb`. The package uses struct tags and reflection to generate SELECT, INSERT, UPDATE, and DELETE SQL templates at initialization time, supporting batch operations via PostgreSQL `UNNEST`-based array parameters.

---

## Functional Requirements

### FR-1: Relocate qb to sql/qb

**Description:** The package must be placed at `sql/qb` within the gotham module (`github.com/TouchBistro/gotham/sql/qb`).

**Acceptance Criteria:**
- Package declaration is `package qb`.
- Import path resolves to `github.com/TouchBistro/gotham/sql/qb`.
- All source files from the origin package are present, except files removed per FR-3.

**Priority:** High

---

### FR-2: Remove Internal devops-api-service Imports

**Description:** Any import referencing `github.com/TouchBistro/devops-api/**` must be removed from the migrated package. The only allowed imports are Go standard library packages and approved third-party libraries.

**Acceptance Criteria:**
- No import path starting with `github.com/TouchBistro/devops-api` exists in any file under `sql/qb/`.
- Functions previously calling `tbutil.ToStringPtr` are refactored; the `ToStringPtr` helper is moved to a `sql/qb/tmp` sub-package within gotham and referenced from there.
- The `tmp` package is marked in code comments as a temporary holding area pending future refactoring.

**Priority:** High

---

### FR-3: Remove Files With No Uncommented Code

**Description:** Source files whose entire code body consists of commented-out lines must not be migrated.

**Acceptance Criteria:**
- `table_join.go` is not present in `sql/qb/` (its entire Go body is inside a block comment).
- `functions.go` is not present in `sql/qb/` (its entire Go body is commented out).
- All remaining migrated files contain at least one uncommented, compilable Go declaration.

**Priority:** High

---

### FR-4: Retain qb.go for Go Doc

**Description:** The `qb.go` file, which provides the package-level godoc comment, must be kept.

**Acceptance Criteria:**
- `sql/qb/qb.go` is present and contains the package-level documentation comment.

**Priority:** Medium

---

### FR-5: Add New Third-Party Dependency (lib/pq)

**Description:** The migrated package depends on `github.com/lib/pq` for PostgreSQL array parameter support. This dependency is not present in gotham's current `go.mod` and must be added.

**Acceptance Criteria:**
- `github.com/lib/pq` is listed as a direct dependency in `go.mod`.
- `go.sum` is updated accordingly.
- `tech-stack.md` is updated to document the new dependency.

**Priority:** High

---

### FR-6: Create README.md

**Description:** A `README.md` must be created inside `sql/qb/` explaining the package features, struct tag conventions, and usage examples.

**Acceptance Criteria:**
- `sql/qb/README.md` exists.
- README covers: purpose of the package, supported operations (SELECT, INSERT, UPDATE, DELETE), struct tag syntax (`qb` tag keys: `pk`/`pkey`, `ops=r/a/w`, `type=`, `as=`, `qbon`), how to initialize a `Table` or `Query`, and a minimal usage code example.

**Priority:** Medium

---

### FR-7: Preserve and Adapt Existing Tests

**Description:** The existing test files (`table_test.go`, `column_metadata_test.go`) must be migrated and must continue to pass after refactoring.

**Acceptance Criteria:**
- `sql/qb/table_test.go` and `sql/qb/column_metadata_test.go` are present.
- `go test ./sql/qb/...` passes with no failures.

**Priority:** High

---

## Non-Functional Requirements

### NFR-1: Test Coverage

The `sql/qb` package must achieve >90% code coverage as measured by `go test -cover ./sql/qb/...`.

### NFR-2: Compilability

The migrated package must compile cleanly with `go build ./sql/qb/...` and `go vet ./sql/qb/...` producing no errors or warnings.

### NFR-3: No Breaking Changes to Existing Packages

Migrating `qb` must not alter or break any existing packages within gotham (`http/`, `cache/`, `util/`).

---

## User Stories

### US-1: Library Consumer Importing qb

**As** a TouchBistro backend engineer building a Go microservice,
**I want** to import `github.com/TouchBistro/gotham/sql/qb`,
**So that** I can generate type-safe SQL CRUD operations without writing boilerplate SQL.

**Scenario:** Initialize a Table and generate a SELECT statement
- **Given** I have a Go struct with `qb` struct tags defining column operations
- **When** I call `qb.ForTable[MyEntity]("schema.table")`
- **Then** I receive a `*Table[MyEntity]` with a pre-built SELECT statement matching the tagged fields

---

### US-2: Library Consumer Using Batch Insert

**As** a TouchBistro backend engineer,
**I want** to batch insert multiple entities using a single SQL call,
**So that** I can efficiently write large datasets to PostgreSQL.

**Scenario:** Batch insert via UNNEST arrays
- **Given** I have a `*Table[MyEntity]` initialized with `qb.ForTable`
- **When** I call `table.Insert(ctx, db, entity1, entity2, ...)`
- **Then** a single `INSERT INTO ... (SELECT * FROM UNNEST(...))` statement is executed and the number of rows affected is returned

---

### US-3: Library Consumer Using Join Queries

**As** a TouchBistro backend engineer,
**I want** to define multi-table join queries using struct composition and join type markers,
**So that** I can select across related tables without writing raw SQL.

**Scenario:** Define a Query with a LeftJoin
- **Given** I have a composite struct embedding two entity structs and a `qb.LeftJoin` field with a `qbon` tag
- **When** I call `qb.ForQuery[MyCompositeEntity]()`
- **Then** I receive a `*Query[MyCompositeEntity]` with a pre-built SELECT ... JOIN ... statement

---

## Technical Considerations

### Dependency: github.com/lib/pq

The package uses `pq.Array` for passing Go slices as PostgreSQL array parameters in batch INSERT, UPDATE, and DELETE operations. This is a hard runtime dependency. It must be added to `go.mod`.

### tbutil.ToStringPtr

Two call sites in `query.go` use `tbutil.ToStringPtr(...)` from `devops-api-service`. This is a trivial one-liner (`func ToStringPtr(val string) *string { return &val }`). Per Jira instructions:
- Create `sql/qb/tmp/` package in gotham.
- Define `ToStringPtr` (and `ToInt64Ptr` for completeness) there.
- Import `github.com/TouchBistro/gotham/sql/qb/tmp` inside `sql/qb/query.go`.
- Add a comment in `tmp/` noting it is a temporary holding package.

### WhereEq Generic Constraint

`where.go` contains `WhereEq[T]` whose `WhereClause` method takes a `Table[T]` argument, which is a different signature than the `WhereClause` interface. This is already in the source and must be migrated as-is (it does not implement the `WhereClause` interface); no change required unless compilation fails.

### result.go

`result.go` contains only `type Result interface{}`. This is a single-line uncommented declaration and must be migrated. It satisfies FR-3 (not empty).

---

## Out of Scope

- Implementing the commented-out `TableJoin` functionality from `table_join.go`.
- Implementing the commented-out free functions from `functions.go`.
- Refactoring the `tmp` package beyond housing the relocated `tbutil` helpers.
- Adding support for database engines other than PostgreSQL.
- Integration or end-to-end tests against a live database.

---

## Open Questions

1. Should `result.go` (containing only `type Result interface{}`) be kept given it has no existing usages? The Jira description states "files with no un-commented code must be removed" — `result.go` does have uncommented code (the interface declaration), so it is retained per FR-3. Confirm if this is the intended interpretation.

2. Should `tmp` be a sub-package of `sql/qb` (i.e., `sql/qb/tmp`) or a top-level gotham package (e.g., `util/ptr`)? The Jira description says `tmp` package — treating as `sql/qb/tmp` unless directed otherwise.

3. `go.mod` currently specifies `go 1.22`. The generics features used by `qb` require Go 1.18+, so no version constraint change is needed. Confirm.
