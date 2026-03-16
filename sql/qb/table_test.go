package qb

import (
	"context"
	"reflect"
	"testing"
)

type TestInner struct {
	InnerName  string
	InnerValue string
}

type Test struct {
	Id             int64  `qb:"id,pk,type=BIGINT,ops=r"`
	Name           string `qb:"name,type=VARCHAR,r,a"`
	Desc           string `qb:"description,as=description,r,w"`
	Link           string `qb:"linker,as=links"`           // wont be in select, insert & update list
	TransationType int64  `qb:"as=ttype,type=INTEGER,r,a"` // TransactionType field namei s assumed
	AnotherType    string `qb:"another,type=VARCHAR,a,w"`  // no select, only INSERT & UPDATE
	ThirdType      []string
	FourthType     string `qb:"as=fourth"`
	UnusedColumn   *string
	InnerObject    TestInner
}

func (t Test) Key() PrimaryKey        { return t.Id }
func (t Test) Equals(other Test) bool { return false }

type MasterData_EmployeeDetails struct {
	Id int64 `qb:"PK,r"`
}

func (m MasterData_EmployeeDetails) Key() PrimaryKey                              { return m.Id }
func (m MasterData_EmployeeDetails) Equals(other MasterData_EmployeeDetails) bool { return false }

func TestTableMetadata(t *testing.T) {

	table, err := ForTable[Test]()
	if err != nil {
		t.Errorf("err not expect, but returned: %v", err.Error())
	}

	if table.tableMetadata.schema != "public" {
		t.Errorf("schema name is incorrect, %v expected, %v received", "public", table.tableMetadata.schema)
	}

	if table.tableMetadata.table != "test" {
		t.Errorf("table name is incorrect, %v expected, %v received", "test", table.tableMetadata.schema)
	}

	table, err = ForTable[Test]("schem.table")
	if err != nil {
		t.Errorf("err not expected, but returned: %v", err.Error())
	}

	if table.tableMetadata.schema != "schem" {
		t.Errorf("schema name is incorrect, %v expected, %v received", "schem", table.tableMetadata.schema)
	}

	if table.tableMetadata.table != "table" {
		t.Errorf("table name is incorrect, %v expected, %v received", "table", table.tableMetadata.schema)
	}
}

func TestTableMetadata_Complex(t *testing.T) {

	// master_data.employe_details
	table, err := ForTable[MasterData_EmployeeDetails]()
	if err != nil {
		t.Errorf("err not expect, but returned: %v", err.Error())
	}

	md := table.tableMetadata
	expected_schema := "master_data"
	expected_table := "employee_details"

	if md.schema != expected_schema {
		t.Errorf("schema name is incorrect, %v expected, %v received", expected_schema, table.tableMetadata.schema)
	}

	if md.table != expected_table {
		t.Errorf("table name is incorrect, %v expected, %v received", expected_table, table.tableMetadata.schema)
	}

}

func TestColumnMetadata_SqlType(t *testing.T) {

	table, err := ForTable[Test]("schem.table")
	if err != nil {
		t.Errorf("err not expected, but returned: %v", err.Error())
	}

	cmd := table.tableMetadata.metadata
	expectedSize := 10
	if len(cmd) != expectedSize {
		t.Errorf("column metadata size %v expected, %v received", expectedSize, len(cmd))
	}

	var index = 0
	// Id
	if !cmd[index].IsKey {
		t.Errorf("column %v should be a PK", cmd[index].FieldName)
	}

	// Name
	index = 1

	if cmd[index].IsKey {
		t.Errorf("column %v should NOT be a PK", cmd[index].FieldName)
	}

	// Desc
	index = 2

	if !cmd[index].IsSelected {
		t.Errorf("column %v should be selectable", cmd[index].FieldName)
	}

	if cmd[index].IsInserted {
		t.Errorf("column %v should NOT be insertable", cmd[index].FieldName)
	}

	// Link
	index = 3
	if cmd[index].IsUpdated {
		t.Errorf("column %v should NOT be updtable", cmd[index].FieldName)
	}

	// AnotherType
	index = 5

	if !cmd[index].IsUpdated {
		t.Errorf("column %v should be updtable", cmd[index].FieldName)
	}

	// Third Type
	index = 6
	if cmd[index].SqlType != "VARCHAR[]" {
		t.Errorf("column %v should have sql type VARCHAR[], but found %v", cmd[index].FieldName, cmd[index].SqlType)
	}

	index = 8
	v8 := "unused_column"
	if cmd[index].SqlName != v8 {
		t.Errorf("column %v should have sqlName %v, found: %v", cmd[index].FieldName, v8, cmd[index].SqlName)
	}
}

func TestGenerateSelectSql(t *testing.T) {
	statement := `SELECT id, name, description, transation_type AS "ttype" FROM schem.tab`
	table, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Errorf("err not expected, but returned: %v", err.Error())
	}

	if table.selectStmt != statement {
		t.Errorf("\nExpected: %v\nFound   : %v", statement, table.selectStmt)
	}
}

func TestGenerateInsertSqlTemplate(t *testing.T) {
	statement := `INSERT INTO schem.tab (name, transation_type, another) (SELECT * FROM UNNEST ($1::VARCHAR[], $2::INTEGER[], $3::VARCHAR[]));`
	table, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Errorf("err not expected, but returned: %v", err.Error())
	}

	if table.insertStmt != statement {
		t.Errorf("\nExpected: %v\nFound   : %v", statement, table.insertStmt)
	}
}

func TestGenerateUpdateSqlTemplate(t *testing.T) {
	statement := `UPDATE schem.tab SET description = dat.description, another = dat.another FROM ( SELECT UNNEST($1::BIGINT[]) AS id, UNNEST($2::VARCHAR[]) AS description, UNNEST($3::VARCHAR[]) AS another ) dat WHERE schem.tab.id = dat.id;`
	table, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Errorf("err not expected, but returned: %v", err.Error())
	}

	if table.updateStmt != statement {
		t.Errorf("\nExpected: %v\nFound   : %v", statement, table.updateStmt)
	}
}

func TestGenerateDeleteSqlTemplate(t *testing.T) {
	statement := `DELETE FROM schem.tab WHERE (id) IN ( SELECT * FROM UNNEST($1::BIGINT[]));`
	table, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Errorf("err not expected, but returned: %v", err.Error())
	}

	if table.deleteStmt != statement {
		t.Errorf("\nExpected: %v\nFound   : %v", statement, table.deleteStmt)
	}
}

func TestDetermineSchemaTableName_DotNotation(t *testing.T) {
	schema, table, err := determineSchemaTableName("myschema.mytable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *schema != "myschema" {
		t.Errorf("expected schema myschema, got %v", *schema)
	}
	if *table != "mytable" {
		t.Errorf("expected table mytable, got %v", *table)
	}
}

func TestDetermineSchemaTableName_TwoArgs(t *testing.T) {
	schema, table, err := determineSchemaTableName("myschema", "mytable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *schema != "myschema" {
		t.Errorf("expected schema myschema, got %v", *schema)
	}
	if *table != "mytable" {
		t.Errorf("expected table mytable, got %v", *table)
	}
}

func TestDetermineSchemaTableName_NoArgs_Error(t *testing.T) {
	_, _, err := determineSchemaTableName()
	if err == nil {
		t.Error("expected error for no args, got nil")
	}
}

func TestDetermineSchemaTableName_JustTableName(t *testing.T) {
	schema, table, err := determineSchemaTableName("mytable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *schema != "public" {
		t.Errorf("expected schema public, got %v", *schema)
	}
	if *table != "mytable" {
		t.Errorf("expected table mytable, got %v", *table)
	}
}

func TestMakeParameterArray(t *testing.T) {
	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr := tbl.makeParameterArray(3)
	if len(arr) != 3 {
		t.Errorf("expected length 3, got %d", len(arr))
	}
	for i, inner := range arr {
		if len(inner) != 0 {
			t.Errorf("inner slice %d should be empty, got length %d", i, len(inner))
		}
	}
}

func TestMakeParameterArray_Zero(t *testing.T) {
	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr := tbl.makeParameterArray(0)
	if len(arr) != 0 {
		t.Errorf("expected empty array, got length %d", len(arr))
	}
}

func TestForTable_TwoArgSchemaTable(t *testing.T) {
	table, err := ForTable[Test]("myschema", "mytable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if table.tableMetadata.schema != "myschema" {
		t.Errorf("expected schema myschema, got %v", table.tableMetadata.schema)
	}
	if table.tableMetadata.table != "mytable" {
		t.Errorf("expected table mytable, got %v", table.tableMetadata.table)
	}
}

// TableNameFromType tests: type with no underscore => public schema
func TestTableNameFromType_NoUnderscore(t *testing.T) {
	schema, table := tableNameFromType(reflect.TypeOf(Test{}))
	if schema != "public" {
		t.Errorf("expected schema public, got %v", schema)
	}
	if table != "test" {
		t.Errorf("expected table test, got %v", table)
	}
}

// Verify that Insert/Update/Delete with empty slice returns 0, nil without touching DB
func TestInsert_EmptyEntities_ReturnsZero(t *testing.T) {
	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Pass nil db — should return 0, nil before opening TX since entities is empty
	rowsAffected, err := tbl.Insert(context.TODO(), nil)
	if err != nil {
		t.Errorf("unexpected error for empty insert: %v", err)
	}
	if rowsAffected != 0 {
		t.Errorf("expected 0 rows affected for empty insert, got %d", rowsAffected)
	}
}

func TestUpdate_EmptyEntities_ReturnsZero(t *testing.T) {
	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rowsAffected, err := tbl.Update(context.TODO(), nil)
	if err != nil {
		t.Errorf("unexpected error for empty update: %v", err)
	}
	if rowsAffected != 0 {
		t.Errorf("expected 0 rows affected for empty update, got %d", rowsAffected)
	}
}

func TestDelete_EmptyEntities_ReturnsZero(t *testing.T) {
	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rowsAffected, err := tbl.Delete(context.TODO(), nil)
	if err != nil {
		t.Errorf("unexpected error for empty delete: %v", err)
	}
	if rowsAffected != 0 {
		t.Errorf("expected 0 rows affected for empty delete, got %d", rowsAffected)
	}
}

func TestInsertTx_EmptyEntities_ReturnsZero(t *testing.T) {
	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rowsAffected, err := tbl.InsertTx(context.TODO(), nil)
	if err != nil {
		t.Errorf("unexpected error for empty InsertTx: %v", err)
	}
	if rowsAffected != 0 {
		t.Errorf("expected 0 rows affected for empty InsertTx, got %d", rowsAffected)
	}
}

func TestUpdateTx_EmptyEntities_ReturnsZero(t *testing.T) {
	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rowsAffected, err := tbl.UpdateTx(context.TODO(), nil)
	if err != nil {
		t.Errorf("unexpected error for empty UpdateTx: %v", err)
	}
	if rowsAffected != 0 {
		t.Errorf("expected 0 rows affected for empty UpdateTx, got %d", rowsAffected)
	}
}

func TestDeleteTx_EmptyEntities_ReturnsZero(t *testing.T) {
	tbl, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rowsAffected, err := tbl.DeleteTx(context.TODO(), nil)
	if err != nil {
		t.Errorf("unexpected error for empty DeleteTx: %v", err)
	}
	if rowsAffected != 0 {
		t.Errorf("expected 0 rows affected for empty DeleteTx, got %d", rowsAffected)
	}
}
