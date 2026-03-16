package qb

// db_mock_test.go provides a minimal database/sql driver mock so that
// DB-interaction methods on Table and Query can be exercised in unit tests
// without a live PostgreSQL connection.

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"
)

// --- mock driver ---

const mockDriverName = "qb_test_mock"

func init() {
	sql.Register(mockDriverName, &mockDriver{})
}

type mockDriver struct {
	conn *mockConn
}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	if d.conn != nil {
		return d.conn, nil
	}
	return &mockConn{}, nil
}

type mockConn struct {
	execErr  error
	queryErr error
	rows     [][]driver.Value
	cols     []string
	// rowsAffected returned by Exec
	rowsAffected int64
	beginErr     error
	commitErr    error
	rollbackErr  error
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	return &mockStmt{conn: c}, nil
}

func (c *mockConn) Close() error  { return nil }
func (c *mockConn) Begin() (driver.Tx, error) {
	if c.beginErr != nil {
		return nil, c.beginErr
	}
	return &mockTx{conn: c}, nil
}

type mockTx struct {
	conn *mockConn
}

func (t *mockTx) Commit() error {
	if t.conn.commitErr != nil {
		return t.conn.commitErr
	}
	return nil
}

func (t *mockTx) Rollback() error {
	if t.conn.rollbackErr != nil {
		return t.conn.rollbackErr
	}
	return nil
}

type mockStmt struct {
	conn *mockConn
}

func (s *mockStmt) Close() error { return nil }
func (s *mockStmt) NumInput() int { return -1 } // variadic

func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) {
	if s.conn.execErr != nil {
		return nil, s.conn.execErr
	}
	return &mockResult{rowsAffected: s.conn.rowsAffected}, nil
}

func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error) {
	if s.conn.queryErr != nil {
		return nil, s.conn.queryErr
	}
	return &mockRows{
		cols:    s.conn.cols,
		data:    s.conn.rows,
		current: -1,
	}, nil
}

type mockResult struct {
	rowsAffected int64
}

func (r *mockResult) LastInsertId() (int64, error) { return 0, nil }
func (r *mockResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

type mockRows struct {
	cols    []string
	data    [][]driver.Value
	current int
}

func (r *mockRows) Columns() []string { return r.cols }

func (r *mockRows) Close() error { return nil }

func (r *mockRows) Next(dest []driver.Value) error {
	r.current++
	if r.current >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.current])
	return nil
}

// --- helper: open mock db ---

func newMockDB(conn *mockConn) *sql.DB {
	// Each test needs a unique DSN to get its own connection from sql's pool.
	// We use a shared global driver and inject via conn field.
	drv := sql.Drivers()
	_ = drv
	// Register a new driver name each call is messy; instead we use a
	// driver that always returns the supplied conn.
	drvName := mockDriverName + "_" + randomID()
	sql.Register(drvName, &mockDriver{conn: conn})
	db, _ := sql.Open(drvName, "test")
	db.SetMaxOpenConns(1)
	return db
}

var mockIDCounter int

func randomID() string {
	mockIDCounter++
	return string(rune('a' + mockIDCounter))
}

// --- tests using mock DB ---

func TestSelectWhere_EmptyRows(t *testing.T) {
	conn := &mockConn{
		cols: []string{"id", "name", "description", "transation_type"},
		rows: [][]driver.Value{},
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	result, err := tbl.SelectWhere(ctx, db, WhereAll{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestSelect_EmptyRows(t *testing.T) {
	conn := &mockConn{
		cols: []string{"id", "name", "description", "transation_type"},
		rows: [][]driver.Value{},
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	result, err := tbl.Select(ctx, db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestSelectWhere_QueryError(t *testing.T) {
	conn := &mockConn{
		queryErr: errors.New("query error"),
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	_, err = tbl.SelectWhere(ctx, db, WhereAll{})
	if err == nil {
		t.Error("expected error from SelectWhere, got nil")
	}
}

func TestSelectWhere_WithArgs(t *testing.T) {
	conn := &mockConn{
		cols: []string{"id", "name", "description", "transation_type"},
		rows: [][]driver.Value{},
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	result, err := tbl.SelectWhere(ctx, db, WhereString("WHERE id=$1"), int64(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestSelectTx_EmptyRows(t *testing.T) {
	conn := &mockConn{
		cols: []string{"id", "name", "description", "transation_type"},
		rows: [][]driver.Value{},
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error beginning tx: %v", err)
	}

	result, err := tbl.SelectTx(ctx, tx)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("unexpected error: %v", err)
	}
	_ = tx.Commit()

	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestSelectWhereTx_WithArgs(t *testing.T) {
	conn := &mockConn{
		cols: []string{"id", "name", "description", "transation_type"},
		rows: [][]driver.Value{},
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error beginning tx: %v", err)
	}

	result, err := tbl.SelectWhereTx(ctx, tx, WhereString("WHERE id=$1"), int64(1))
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("unexpected error: %v", err)
	}
	_ = tx.Commit()

	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestInsert_NonEmptyEntities(t *testing.T) {
	conn := &mockConn{
		rowsAffected: 1,
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	entity := Test{Id: 1, Name: "test", Desc: "desc", TransationType: 1, AnotherType: "other"}
	rowsAffected, err := tbl.Insert(ctx, db, entity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rowsAffected != 1 {
		t.Errorf("expected 1 row affected, got %d", rowsAffected)
	}
}

func TestInsertTx_NonEmptyEntities(t *testing.T) {
	conn := &mockConn{
		rowsAffected: 1,
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error beginning tx: %v", err)
	}

	entity := Test{Id: 1, Name: "test", Desc: "desc", TransationType: 1, AnotherType: "other"}
	rowsAffected, err := tbl.InsertTx(ctx, tx, entity)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("unexpected error: %v", err)
	}
	_ = tx.Commit()

	if rowsAffected != 1 {
		t.Errorf("expected 1 row affected, got %d", rowsAffected)
	}
}

func TestUpdate_NonEmptyEntities(t *testing.T) {
	conn := &mockConn{
		rowsAffected: 1,
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	entity := Test{Id: 1, Name: "test", Desc: "desc", TransationType: 1, AnotherType: "other"}
	rowsAffected, err := tbl.Update(ctx, db, entity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rowsAffected != 1 {
		t.Errorf("expected 1 row affected, got %d", rowsAffected)
	}
}

func TestUpdateTx_NonEmptyEntities(t *testing.T) {
	conn := &mockConn{
		rowsAffected: 1,
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error beginning tx: %v", err)
	}

	entity := Test{Id: 1, Name: "test", Desc: "desc", TransationType: 1, AnotherType: "other"}
	rowsAffected, err := tbl.UpdateTx(ctx, tx, entity)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("unexpected error: %v", err)
	}
	_ = tx.Commit()

	if rowsAffected != 1 {
		t.Errorf("expected 1 row affected, got %d", rowsAffected)
	}
}

func TestDelete_NonEmptyEntities(t *testing.T) {
	conn := &mockConn{
		rowsAffected: 1,
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	entity := Test{Id: 1}
	rowsAffected, err := tbl.Delete(ctx, db, entity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rowsAffected != 1 {
		t.Errorf("expected 1 row affected, got %d", rowsAffected)
	}
}

func TestDeleteTx_NonEmptyEntities(t *testing.T) {
	conn := &mockConn{
		rowsAffected: 1,
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error beginning tx: %v", err)
	}

	entity := Test{Id: 1}
	rowsAffected, err := tbl.DeleteTx(ctx, tx, entity)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("unexpected error: %v", err)
	}
	_ = tx.Commit()

	if rowsAffected != 1 {
		t.Errorf("expected 1 row affected, got %d", rowsAffected)
	}
}

// Test error propagation from InsertTx -> Insert
func TestInsert_ExecError_PropagatesError(t *testing.T) {
	conn := &mockConn{
		execErr: errors.New("exec error"),
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	entity := Test{Id: 1, Name: "test", Desc: "desc", TransationType: 1, AnotherType: "other"}
	_, err = tbl.Insert(ctx, db, entity)
	if err == nil {
		t.Error("expected error from Insert with exec error, got nil")
	}
}

// Test error propagation from UpdateTx -> Update
func TestUpdate_ExecError_PropagatesError(t *testing.T) {
	conn := &mockConn{
		execErr: errors.New("exec error"),
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	entity := Test{Id: 1, Name: "test", Desc: "desc", TransationType: 1, AnotherType: "other"}
	_, err = tbl.Update(ctx, db, entity)
	if err == nil {
		t.Error("expected error from Update with exec error, got nil")
	}
}

// Test error propagation from DeleteTx -> Delete
func TestDelete_ExecError_PropagatesError(t *testing.T) {
	conn := &mockConn{
		execErr: errors.New("exec error"),
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	entity := Test{Id: 1}
	_, err = tbl.Delete(ctx, db, entity)
	if err == nil {
		t.Error("expected error from Delete with exec error, got nil")
	}
}

// Test SelectWhereTx with args (covers the len(args)>0 branch)
func TestSelectWhereTx_WithArgs_QueryError(t *testing.T) {
	conn := &mockConn{
		queryErr: errors.New("query error"),
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error beginning tx: %v", err)
	}

	_, err = tbl.SelectWhereTx(ctx, tx, WhereString("WHERE id=$1"), int64(1))
	if err == nil {
		_ = tx.Rollback()
		t.Error("expected error from SelectWhereTx with query error, got nil")
	}
}

// Test with actual rows returned to exercise the mapper
// We use a simple entity type that has just one selectable int64 column to keep
// the mock data setup minimal.

type SimpleEntity struct {
	Id int64 `qb:"id,pk,type=BIGINT,ops=r"`
}

func (s SimpleEntity) Key() PrimaryKey              { return s.Id }
func (s SimpleEntity) Equals(other SimpleEntity) bool { return s.Id == other.Id }

func TestSelectWhere_WithRows(t *testing.T) {
	conn := &mockConn{
		cols: []string{"id"},
		rows: [][]driver.Value{
			{int64(42)},
		},
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[SimpleEntity]("schem.simple")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	result, err := tbl.SelectWhere(ctx, db, WhereAll{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
	if result[0].Id != 42 {
		t.Errorf("expected Id=42, got %d", result[0].Id)
	}
}

// Test SelectWhere commit error path
func TestSelectWhere_CommitError(t *testing.T) {
	conn := &mockConn{
		cols:      []string{"id", "name", "description", "transation_type"},
		rows:      [][]driver.Value{},
		commitErr: errors.New("commit error"),
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	_, err = tbl.SelectWhere(ctx, db, WhereAll{})
	if err == nil {
		t.Error("expected error from SelectWhere with commit error, got nil")
	}
}

// Test Insert commit error path
func TestInsert_CommitError(t *testing.T) {
	conn := &mockConn{
		rowsAffected: 1,
		commitErr:    errors.New("commit error"),
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	entity := Test{Id: 1, Name: "test", Desc: "desc", TransationType: 1, AnotherType: "other"}
	_, err = tbl.Insert(ctx, db, entity)
	if err == nil {
		t.Error("expected commit error from Insert, got nil")
	}
}

// Test Update commit error path
func TestUpdate_CommitError(t *testing.T) {
	conn := &mockConn{
		rowsAffected: 1,
		commitErr:    errors.New("commit error"),
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	entity := Test{Id: 1, Name: "test", Desc: "desc", TransationType: 1, AnotherType: "other"}
	_, err = tbl.Update(ctx, db, entity)
	if err == nil {
		t.Error("expected commit error from Update, got nil")
	}
}

// Test Delete commit error path
func TestDelete_CommitError(t *testing.T) {
	conn := &mockConn{
		rowsAffected: 1,
		commitErr:    errors.New("commit error"),
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	entity := Test{Id: 1}
	_, err = tbl.Delete(ctx, db, entity)
	if err == nil {
		t.Error("expected commit error from Delete, got nil")
	}
}

// --- Query-level mock DB tests ---

func TestQuery_SelectWhere_EmptyRows(t *testing.T) {
	conn := &mockConn{
		cols: []string{"left.left_id", "left.left_name", "right.right_id", "right.right_name"},
		rows: [][]driver.Value{},
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	q, err := ForQuery[CompositeLeftJoinEntity]()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	result, err := q.SelectWhere(ctx, db, WhereAll{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestQuery_Select_EmptyRows(t *testing.T) {
	conn := &mockConn{
		cols: []string{"left.left_id", "left.left_name", "right.right_id", "right.right_name"},
		rows: [][]driver.Value{},
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	q, err := ForQuery[CompositeLeftJoinEntity]()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	result, err := q.Select(ctx, db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestQuery_SelectTx_EmptyRows(t *testing.T) {
	conn := &mockConn{
		cols: []string{"left.left_id", "left.left_name", "right.right_id", "right.right_name"},
		rows: [][]driver.Value{},
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	q, err := ForQuery[CompositeLeftJoinEntity]()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error beginning tx: %v", err)
	}

	result, err := q.SelectTx(ctx, tx)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("unexpected error: %v", err)
	}
	_ = tx.Commit()

	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestQuery_SelectWhere_QueryError(t *testing.T) {
	conn := &mockConn{
		queryErr: errors.New("query error"),
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	q, err := ForQuery[CompositeLeftJoinEntity]()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	_, err = q.SelectWhere(ctx, db, WhereAll{})
	if err == nil {
		t.Error("expected error from Query.SelectWhere with query error, got nil")
	}
}

func TestQuery_SelectWhere_CommitError(t *testing.T) {
	conn := &mockConn{
		cols:      []string{"left.left_id", "left.left_name", "right.right_id", "right.right_name"},
		rows:      [][]driver.Value{},
		commitErr: errors.New("commit error"),
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	q, err := ForQuery[CompositeLeftJoinEntity]()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	_, err = q.SelectWhere(ctx, db, WhereAll{})
	if err == nil {
		t.Error("expected commit error from Query.SelectWhere, got nil")
	}
}

func TestQuery_SelectWhereTx_WithArgs(t *testing.T) {
	conn := &mockConn{
		cols: []string{"left.left_id", "left.left_name", "right.right_id", "right.right_name"},
		rows: [][]driver.Value{},
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	q, err := ForQuery[CompositeLeftJoinEntity]()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error beginning tx: %v", err)
	}

	result, err := q.SelectWhereTx(ctx, tx, WhereString("WHERE id=$1"), int64(1))
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("unexpected error: %v", err)
	}
	_ = tx.Commit()

	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

// Test with rows returned to exercise the Query mapper function
func TestQuery_SelectWhere_WithRows(t *testing.T) {
	conn := &mockConn{
		cols: []string{"left_id", "left_name", "right_id", "right_name"},
		rows: [][]driver.Value{
			{int64(1), "left_val", int64(2), "right_val"},
		},
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	q, err := ForQuery[CompositeLeftJoinEntity]()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	result, err := q.SelectWhere(ctx, db, WhereAll{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
}

// Test generateColsMetadata error paths on table.go
// We test ForTable with a type that has no selectable struct fields (struct is a non-struct at ty.Kind() level)
// We can test it by calling reflectTypeToTableMetadata directly with a non-struct type.

// Use an empty struct type to cover the "no column metadata" path
// Actually the empty struct would result in len(mdat) == 0 error
type EmptyStruct struct{}

func (e EmptyStruct) Key() PrimaryKey              { return nil }
func (e EmptyStruct) Equals(other EmptyStruct) bool { return false }

func TestForTable_EmptyStruct_Error(t *testing.T) {
	_, err := ForTable[EmptyStruct]("schem.tab")
	if err == nil {
		t.Error("expected error for empty struct, got nil")
	}
}

// Test SelectWhere error path (rollback)
func TestSelectWhere_RollbackOnQueryError(t *testing.T) {
	conn := &mockConn{
		queryErr: errors.New("query error"),
	}
	db := newMockDB(conn)
	defer func() { _ = db.Close() }()

	tbl, err := ForTable[SimpleEntity]("schem.simple")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	_, err = tbl.SelectWhere(ctx, db, WhereAll{})
	if err == nil {
		t.Error("expected error from SelectWhere, got nil")
	}
}
