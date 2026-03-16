# sql/qb — Query Builder

Import path: `github.com/TouchBistro/gotham/sql/qb`

## Purpose

`qb` is a reflection-based query builder for PostgreSQL-backed Go microservices. It generates SELECT, INSERT, UPDATE, and DELETE SQL statements at initialization time by inspecting Go struct tags. At runtime the generated templates are executed against a `*sql.DB` or `*sql.Tx` using `pq.Array`-style UNNEST batch parameters.

The package is designed for simple, single-table CRUD scenarios (`Table[T]`) and for multi-table read-only join scenarios (`Query[T]`).

## Supported CRUD Operations

| Operation | Method | PostgreSQL pattern |
|-----------|--------|--------------------|
| SELECT all rows | `Table.Select` / `Query.Select` | `SELECT c1, c2 … FROM schema.table WHERE 1=1` |
| SELECT with filter | `Table.SelectWhere` / `Query.SelectWhere` | `SELECT … WHERE <clause>` |
| SELECT in transaction | `Table.SelectTx` / `Table.SelectWhereTx` | same, on `*sql.Tx` |
| INSERT (batch) | `Table.Insert` / `Table.InsertTx` | `INSERT INTO … (cols) (SELECT * FROM UNNEST($1::type[], …))` |
| UPDATE (batch) | `Table.Update` / `Table.UpdateTx` | `UPDATE … SET … FROM (SELECT UNNEST(…)) dat WHERE pk = dat.pk` |
| DELETE (batch) | `Table.Delete` / `Table.DeleteTx` | `DELETE FROM … WHERE (pk) IN (SELECT * FROM UNNEST($1::type[]))` |

Batch operations collect all supplied entity values into per-column slices and pass them as PostgreSQL array parameters via `pq.Array`, enabling a single round-trip per batch (default batch size: 100).

## `qb` Struct Tag Reference

Struct fields are annotated with the `qb` struct tag. The tag value is a comma-separated list of tokens:

```
FieldName type `qb:"<token1>,<token2>,..."`
```

### Tag tokens

| Token | Form | Meaning |
|-------|------|---------|
| `pk` or `pkey` | keyless | Marks the field as a primary key column. Used in WHERE clauses for UPDATE and DELETE, and for `WhereEq`. |
| `ops=r` | key=value | Include this column in SELECT statements (`r` = read). |
| `ops=a` | key=value | Include this column in INSERT statements (`a` = append). |
| `ops=w` | key=value | Include this column in UPDATE statements (`w` = write). |
| `ops=raw` | key=value | Tokens can be combined: `ops=raw` means SELECT + INSERT + UPDATE. |
| `r` | keyless shorthand | Equivalent to `ops=r`. |
| `a` | keyless shorthand | Equivalent to `ops=a`. |
| `w` | keyless shorthand | Equivalent to `ops=w`. |
| `type=<sqltype>` | key=value | Override the inferred PostgreSQL type (e.g. `type=UUID`, `type=JSONB`). By default the type is inferred from the Go field type. |
| `as=<alias>` | key=value | Set a column alias used in SELECT output (e.g. `as=my_alias` generates `col AS "my_alias"`). |
| `<colname>` | keyless string | Override the SQL column name. If no `as=` is provided the alias is set to the same value. |

When no `qb` tag is present on a field, the field is still recorded in the metadata (with SQL name derived from the field name in snake_case) but is not selected, inserted, or updated.

### SQL type inference

If `type=` is not supplied, the SQL type is inferred from the Go field type:

| Go type | Inferred SQL type |
|---------|-------------------|
| `int`, `int32` | `INTEGER` |
| `int64` | `BIGINT` |
| `string`, `*string` | `VARCHAR` |
| `time.Time` | `TIMESTAMP` |
| `[]T` (slice) | `<T_type>[]` (array) |
| other | Go type string (e.g. `bool`) |

### Table name inference

When `ForTable` is called without an explicit name, the table name is derived from the Go type name using snake_case conversion. If the type name contains an underscore the part before the first underscore is used as the schema; otherwise the schema defaults to `public`.

Example: type `MyService_Order` → schema `my_service`, table `order`.

## `qbon` Struct Tag

The `qbon` tag is used on `LeftJoin`, `RightJoin`, or `InnerJoin` marker fields inside a composite struct to specify the SQL JOIN ON condition as a raw string.

```go
type OrderWithCustomer struct {
    Order    Order
    J        qb.LeftJoin `qbon:"order.customer_id = customer.id"`
    Customer Customer
}
```

`ForQuery` reads the `qbon` tag value and emits `ON <value>` in the generated SELECT statement. Only one of `qb` or `qbon` should be set on a join marker field; `qbon` is the preferred form.

## Initializing a `Table[T]`

`ForTable` is a generic constructor that returns `*Table[T]` pre-loaded with all SQL templates.

```go
// Infer table name from the Go type name
t, err := qb.ForTable[MyEntity]()

// Explicit table name (schema inferred as "public")
t, err := qb.ForTable[MyEntity]("orders")

// Explicit schema.table
t, err := qb.ForTable[MyEntity]("commerce.orders")

// Schema and table as separate arguments
t, err := qb.ForTable[MyEntity]("commerce", "orders")
```

`T` must implement the `Entity[T]` interface:

```go
type Entity[T any] interface {
    Key() PrimaryKey
    Equals(T) bool
}
```

## Initializing a `Query[T]`

`ForQuery` is used for multi-table read-only join queries. The composite struct `T` must embed one struct per joined table, with `LeftJoin`, `RightJoin`, or `InnerJoin` marker fields between them carrying `qbon` tags.

```go
q, err := qb.ForQuery[OrderWithCustomer]()
```

`Query[T]` exposes `Select`, `SelectWhere`, `SelectTx`, and `SelectWhereTx`. It does not support INSERT, UPDATE, or DELETE.

## Minimal Usage Example

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"

    _ "github.com/lib/pq"
    "github.com/TouchBistro/gotham/sql/qb"
)

// Order maps to the "commerce.orders" table.
type Order struct {
    ID        int64  `qb:"pk,id,ops=raw"`
    ProductID int64  `qb:"product_id,ops=raw"`
    Quantity  int    `qb:"quantity,ops=raw"`
}

func (o Order) Key() qb.PrimaryKey { return o.ID }
func (o Order) Equals(other Order) bool {
    return o.ID == other.ID && o.ProductID == other.ProductID && o.Quantity == other.Quantity
}

func main() {
    db, err := sql.Open("postgres", "postgres://user:pass@localhost/mydb?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Initialize the Table once (at startup / in a constructor).
    orders, err := qb.ForTable[Order]("commerce.orders")
    if err != nil {
        log.Fatal(err)
    }

    // SELECT all rows.
    rows, err := orders.Select(context.Background(), db)
    if err != nil {
        log.Fatal(err)
    }
    for _, o := range rows {
        fmt.Printf("order id=%d product=%d qty=%d\n", o.ID, o.ProductID, o.Quantity)
    }

    // Batch insert.
    n, err := orders.Insert(context.Background(), db,
        Order{ID: 1, ProductID: 42, Quantity: 3},
        Order{ID: 2, ProductID: 99, Quantity: 1},
    )
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("%d rows inserted\n", n)
}
```

## `WhereClause` Helpers

| Type | `WhereClause()` output | Use |
|------|------------------------|-----|
| `WhereAll{}` | `WHERE 1=1` | Select all rows (default for `Select`) |
| `WhereNone{}` | `WHERE 1=0` | Select no rows |
| `WhereString("WHERE id=$1")` | verbatim string | Arbitrary SQL predicate |

`WhereEq[T]` is also provided; its `WhereClause(t Table[T])` method builds an equality predicate over all primary key columns. Note that `WhereEq` takes a `Table[T]` argument and does not implement the `WhereClause` interface.

## `tmp` Sub-package

`github.com/TouchBistro/gotham/sql/qb/tmp` is a **temporary** holding package containing two pointer helpers (`ToStringPtr`, `ToInt64Ptr`) relocated from `devops-api-service`. It is used internally by `qb` and is not part of the stable public API. This package will be merged into a proper shared utility package in a future refactoring effort.
