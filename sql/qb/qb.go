// Package qb provides a reflection-based SQL query builder for PostgreSQL-backed
// Go microservices. It generates SELECT, INSERT, UPDATE, and DELETE statement
// templates at initialization time by inspecting struct tags, and executes them
// using pq.Array-style UNNEST batch parameters at runtime.
//
// Import path: github.com/TouchBistro/gotham/sql/qb
//
// # Core Types
//
//   - [Table] — wraps a single database table; supports CRUD operations.
//   - [Query] — wraps a multi-table read-only join query.
//   - [Entity] — interface that entity structs must implement (Key, Equals).
//   - [WhereClause] — interface for WHERE predicates ([WhereAll], [WhereNone], [WhereString]).
//
// # Struct Tags
//
// Column behaviour is controlled via the `qb` struct tag. Tokens are
// comma-separated:
//
//	ID   int64  `qb:"pk,id,ops=raw"`   // primary key, column name "id", read+insert+update
//	Name string `qb:"ops=r,as=full_name"` // SELECT only, aliased
//
// For multi-table join queries, embed [LeftJoin], [RightJoin], or [InnerJoin]
// marker fields and annotate them with a `qbon` tag containing the ON clause:
//
//	J qb.LeftJoin `qbon:"a.id = b.a_id"`
//
// # Initialization
//
//	// Single-table
//	t, err := qb.ForTable[MyEntity]("schema.table")
//
//	// Multi-table join (read-only)
//	q, err := qb.ForQuery[MyCompositeEntity]()
package qb
