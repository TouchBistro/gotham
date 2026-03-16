package qb

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/lib/pq"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type SqlRowsToEntityMapperFn[T Entity[T]] func(rows *sql.Rows) (*T, error)

// type ObjectWithMetadata interface {
// 	Metadata() TableMetadata
// }

type TableMetadata struct {
	table      string
	schema     string
	numKeyCols int
	typ        string  // type of struct this TableMetadata is derived from
	field      *string // if this is a field, thrn field name,
	metadata   []ColumnMetadata
}

// reflectTypeToColumnMetadata converts a reflect.Type to TableMetadata
func reflectTypeToTableMetadata(ty reflect.Type) (*TableMetadata, error) {

	var numKeyCols int
	tmd := &TableMetadata{
		typ: ty.Name(),
	}

	mdat := make([]ColumnMetadata, 0)

	kindOfEle := ty.Kind()
	log.Debugf("t.Rows[] type: %v kind: %v", ty, kindOfEle)

	if kindOfEle == reflect.Struct {

		// infer the name from type
		schema, table := tableNameFromType(ty)
		tmd.schema = schema
		tmd.table = table

		// detect column metadata from fields
		for i := 0; i < ty.NumField(); i++ {
			m := parseColumnMetadataFromStructField(ty.Field(i))
			if m.IsKey {
				numKeyCols++
			}
			mdat = append(mdat, m)
		}

		if len(mdat) == 0 {
			return nil, fmt.Errorf("no column metadata defined")
		}

		tmd.numKeyCols = numKeyCols
		tmd.metadata = mdat
		return tmd, nil

	} else {
		return nil, fmt.Errorf("supplied Entity type %v is not a struct", kindOfEle)
	}
}

// initialize a new Table of type Entity by supplied either schema, tablename, schema.table or just table name (schema public assumed)
func ForTable[T Entity[T]](tableName ...string) (*Table[T], error) {

	// generate table metadata with schema & table name
	tmd := &TableMetadata{} //chema: *schema, table: *table}
	t := &Table[T]{tableMetadata: tmd}

	// we generate & save table metadata
	if err := t.generateColsMetadata(tableName...); err != nil {
		return nil, err
	}

	t.generateSelectStatement()
	t.generateSqlRowsToEntityMapperFn()

	t.insertBatchSize = 100 // TODO parameterize OR auto-detect
	t.generateInsertStatement()

	t.updateBatchSize = 100 // TODO parameterize OR auto-detect
	t.generateUpdateStatement()

	t.deleteBatchSize = 100 // TODO parameterize OR auto-detect
	t.generateDeleteStatement()

	return t, nil
}

type Table[T Entity[T]] struct {

	// contains table metadata
	tableMetadata *TableMetadata

	// table name; supplied at initialization
	// table string
	// schema name; supplied at initialization
	// schema string
	// column metadata inspected from struct tags on the supplied runtime type of T
	// colsMetadata []ColumnMetadata

	// select statement template based on column metadata, generated at initialization
	selectStmt string
	// *sql.Rows -> rutnime T type mapper generated at initialization based on default SELECT statement & column metadata
	mapper SqlRowsToEntityMapperFn[T]
	// insert statement template based on column metadata, generated at initialization
	insertStmt string
	// batch size to use for batch inserts
	insertBatchSize int
	// update statement template based on column metadata, generated at initialization
	updateStmt string
	// batch size to use for batch updates
	updateBatchSize int
	// delete statement template based on column metadata, generated at initialization
	deleteStmt string
	// items to delete in 1 batch
	deleteBatchSize int
	// number of pkey columns
	// numKeyCols int
	// number of selectable columns
	numSelectableCols int
	// number of insertable columns
	numInsertableCols int
	// indicates if an insertable column at index (insertable only) is an array
	insertableColTypeArray []bool
	// number of updateable columns
	numUpdatableCols int
	// indicates if an updatable column at index (updatable only) is an array
	updatableColTypeArray []bool
}

// Metadata returns column metadata for this Table & satisfied ObjectWithMetadata interface
// func (t Table[T]) Metadata() TableMetadata {
// 	return TableMetadata{
// 		table: t.table, schema: t.schema, metadata: t.colsMetadata,
// 	}
// }

// SelectWhere selects all rows from the underlying database table after applying the supplied "Where" condition & arguments into this Table object
func (t Table[T]) SelectWhere(ctx context.Context, conn *sql.DB, where WhereClause, args ...any) ([]T, error) {

	// start transaction
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}

	if rows, err := t.SelectWhereTx(ctx, tx, where, args...); err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			log.Error(err2)
		}
		return nil, err
	} else {
		if err := tx.Commit(); err != nil {
			log.Error(err)
			return nil, err
		}
		return rows, nil
	}
}

// SelectWhereTx selects all rows from the underlying database table after applying the supplied "Where" condition & arguments into this Table object  on the given transaction context
func (t Table[T]) SelectWhereTx(ctx context.Context, tx *sql.Tx, where WhereClause, args ...any) ([]T, error) {

	var err error
	var rs *sql.Rows
	stmt := fmt.Sprintf("%v %v", t.selectStmt, where.WhereClause())

	if len(args) > 0 {
		rs, err = tx.Query(stmt, args...)
	} else {
		rs, err = tx.Query(stmt)
	}

	if err != nil {
		return nil, errors.Wrapf(err, "error fetching entities from database")
	}
	defer func() { _ = rs.Close() }()

	mapper := t.mapper
	rows := make([]T, 0)
	for rs.Next() {
		row, err := mapper(rs)
		if err != nil {
			return nil, err
		}
		rows = append(rows, *row)
	}

	return rows, nil
}

// Selects all rows using the supplied database connection from the underlying Table
func (t Table[T]) Select(ctx context.Context, conn *sql.DB) ([]T, error) {
	return t.SelectWhere(ctx, conn, WhereAll{})
}

// SelectsTx all rows using the supplied database connection from the underlying Table on the given transaction context
func (t Table[T]) SelectTx(ctx context.Context, tx *sql.Tx) ([]T, error) {
	return t.SelectWhereTx(ctx, tx, WhereAll{})
}

// Insert uses an auto-generated batch INSERT DML template & executes it on the supplied entities as array parameters
func (t Table[T]) Insert(ctx context.Context, conn *sql.DB, entities ...T) (int64, error) {

	if len(entities) == 0 {
		return 0, nil
	}

	// start transaction
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return 0, err
	}

	if rowsAffected, err := t.InsertTx(ctx, tx, entities...); err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			log.Error(err2)
		}
		return 0, err
	} else {
		if err := tx.Commit(); err != nil {
			log.Error(err)
			return 0, err
		}
		return rowsAffected, nil
	}
}

// Insert uses an auto-generated batch INSERT DML template & executes it on the supplied entities as array parameters  on the given transaction context
func (t Table[T]) InsertTx(ctx context.Context, tx *sql.Tx, entities ...T) (int64, error) {

	if len(entities) == 0 {
		return 0, nil
	}

	var rowsAffected int64                            // collects num rows affected for all operations
	var rows_processed, rows_processed_this_batch int // total rows processed, and processed this batch

	parms := t.makeParameterArray(t.numInsertableCols)

	for _, e := range entities {
		rows_processed++
		rows_processed_this_batch++

		// append vals from the updatable fields/columns of this entity to the parm slices
		ind := 0
		for _, c := range t.tableMetadata.metadata {
			if c.IsInserted {
				v := reflect.ValueOf(e).FieldByName(c.FieldName).Interface()
				// if c.IsSqlArrayType {
				// 	v = pq.Array(v)
				// }
				parms[ind] = append(parms[ind], v)
				ind++
			}
		}

		// this batch is ready to send when rows processed is equal to batch size || or entitles are done
		if rows_processed_this_batch == t.insertBatchSize || rows_processed == len(entities) {

			stmt := t.insertStmt
			log.Debugf("sql: %v", stmt)

			// convert to []pg.Array
			args := make([]any, 0)
			for n, v := range parms {
				if t.insertableColTypeArray[n] {
					args = append(args, pq.Array(pq.Array(v)))
				} else {
					args = append(args, pq.Array(v))
				}
			}

			rs, err := tx.Exec(stmt, args...)
			if err != nil {
				return 0, err
			}

			ra, err := rs.RowsAffected()
			if err != nil {
				return 0, err
			}

			rowsAffected += ra //+ ra2 // combined idRow & membership rows
			rows_processed_this_batch = 0
			parms = t.makeParameterArray(t.numInsertableCols)
		}
	}

	log.Debugf("%v rows inserted", rowsAffected)
	return rowsAffected, nil
}

// Update uses an auto-generated batch UPDATE DML template for this table & executes it on the supplied entities as array parameters
func (t Table[T]) Update(ctx context.Context, conn *sql.DB, entities ...T) (int64, error) {

	if len(entities) == 0 {
		return 0, nil
	}

	// start transaction
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return 0, err
	}

	if rowsAffected, err := t.UpdateTx(ctx, tx, entities...); err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			log.Error(err2)
		}
		return 0, err
	} else {
		if err := tx.Commit(); err != nil {
			log.Error(err)
			return 0, err
		}
		return rowsAffected, nil
	}
}

// Update uses an auto-generated batch UPDATE DML template for this table & executes it on the supplied entities as array parameters on the given transaction context
func (t Table[T]) UpdateTx(ctx context.Context, tx *sql.Tx, entities ...T) (int64, error) {

	if len(entities) == 0 {
		return 0, nil
	}

	var rows_processed, rows_processed_this_batch int // total rows processed, and processed this batch
	var rowsAffected int64                            // collects num rows affected for all operations

	// init 2D slice to build FROM arrays for updatable columns
	parms := t.makeParameterArray(t.numUpdatableCols + t.tableMetadata.numKeyCols)

	for _, e := range entities {
		rows_processed++
		rows_processed_this_batch++

		// append vals from the updatable fields/columns of this entity to the parm slices
		ind := 0
		for _, c := range t.tableMetadata.metadata {
			if c.IsUpdated || c.IsKey {
				v := reflect.ValueOf(e).FieldByName(c.FieldName).Interface()
				// if c.IsSqlArrayType {
				// 	v = pq.Array(v)
				// }
				parms[ind] = append(parms[ind], v)
				ind++
			}
		}

		// this batch is ready to send when rows processed is equal to batch size || or entitles are done
		if rows_processed_this_batch == t.updateBatchSize || rows_processed == len(entities) {
			stmt := t.updateStmt
			log.Debugf("sql: %v", stmt)

			// convert to []pg.Array
			args := make([]any, 0)
			for _, v := range parms {
				args = append(args, pq.Array(v))
			}

			rs, err := tx.Exec(stmt, args...)
			if err != nil {
				return 0, err
			}

			ra, err := rs.RowsAffected()
			if err != nil {
				return 0, err
			}

			rowsAffected += ra
			rows_processed_this_batch = 0

			// re-init parameter array
			parms = t.makeParameterArray(t.numUpdatableCols + t.tableMetadata.numKeyCols)
		}
	}

	log.Debugf("%v rows inserted", rowsAffected)
	return rowsAffected, nil
}

// Delete uses an auto-generated batch DELETE DML statement & executes it on the supplied entitles as array parameters
func (t Table[T]) Delete(ctx context.Context, conn *sql.DB, entities ...T) (int64, error) {

	if len(entities) == 0 {
		return 0, nil
	}

	tx, err := conn.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return 0, err
	}

	if rowsAffected, err := t.DeleteTx(ctx, tx, entities...); err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			log.Error(err2)
		}
		return 0, err
	} else {
		if err := tx.Commit(); err != nil {
			log.Error(err)
			return 0, err
		}
		return rowsAffected, nil
	}
}

// DeleteTx uses an auto-generated batch DELETE DML statement & executes it on the supplied entities as array parametes on the given transaction context
func (t Table[T]) DeleteTx(ctx context.Context, tx *sql.Tx, entities ...T) (int64, error) {

	if len(entities) == 0 {
		return 0, nil
	}

	var rowsAffected int64
	var rows_processed, rows_processed_this_batch int

	// init 2D slice to build FROM arrays for updatable columns
	parms := t.makeParameterArray(t.tableMetadata.numKeyCols)

	for _, e := range entities {
		rows_processed++
		rows_processed_this_batch++

		// append vals from the key fields/columns of this entity to the parm slices
		ind := 0
		for _, c := range t.tableMetadata.metadata {
			if c.IsKey {
				v := reflect.ValueOf(e).FieldByName(c.FieldName).Interface()
				if c.IsSqlArrayType {
					v = pq.Array(v)
				}
				parms[ind] = append(parms[ind], v)
				ind++
			}
		}

		if rows_processed_this_batch == t.deleteBatchSize || rows_processed == len(entities) {
			stmt := t.deleteStmt
			log.Debugf("sql: %v", stmt)

			// convert to []pg.Array
			args := make([]any, 0)
			for _, v := range parms {
				args = append(args, pq.Array(v))
			}

			rs, err := tx.Exec(stmt, args...)
			if err != nil {
				return 0, err
			}

			ra, err := rs.RowsAffected()
			if err != nil {
				return 0, err
			}

			rowsAffected += ra // rows affected by this batch delete is added to total rows affected
			rows_processed_this_batch = 0
			parms = t.makeParameterArray(t.tableMetadata.numKeyCols)
		}
	}

	log.Debugf("%v rows deleted", rowsAffected)
	return rowsAffected, nil
}

// internal utility methods // // // //

// generarteColsMetadata generates a ColumnMedatadata data structure for the supplied
// T (Entity) type for this `Table` & sets it to the internal field for reference.
// It uses the struct tags supplied to the fields on the struct T for this purpose
//
// Returns a non-nil error if any failure happens during parsing
func (t *Table[T]) generateColsMetadata(tableName ...string) error {
	var e T
	// reflectTypeToTableMetadata(e)
	if tmd, err := entityToTableMetadata(e); err != nil {
		return err
	} else {
		if t.tableMetadata != nil {
			t.tableMetadata = tmd

			// here we check if an override tableName was supplied
			// so we replace the inferred table schema/name with that
			if len(tableName) > 0 {
				schema, table, err := determineSchemaTableName(tableName...)
				if err != nil {
					return err
				}
				t.tableMetadata.schema = *schema
				t.tableMetadata.table = *table
			}
		} else {
			return fmt.Errorf("table metadata not initialized yet")
		}
	}

	return nil
}

// generateSelectStatement generates a template for SELECT statement based on the
// column metadata
// SELECT c1, c2, c3 .... c4 FROM schem.tab
func (t *Table[T]) generateSelectStatement() {
	tmpl := "SELECT %v FROM %v.%v"
	def := fmt.Sprintf(tmpl, "*", t.tableMetadata.schema, t.tableMetadata.table) // default

	if len(t.tableMetadata.metadata) == 0 {
		t.selectStmt = def
	}

	selist := make([]string, 0)
	for _, m := range t.tableMetadata.metadata {
		if m.IsSelected {
			if m.SqlName != m.Alias {
				selist = append(selist, fmt.Sprintf("%v AS \"%v\"", m.SqlName, m.Alias))
			} else {
				selist = append(selist, fmt.Sprintf("%v", m.SqlName))
			}
		}
	}

	t.numSelectableCols = len(selist)
	t.selectStmt = fmt.Sprintf(tmpl, strings.Join(selist, ", "), t.tableMetadata.schema, t.tableMetadata.table)
}

// generateSqlRowsToEntityMapperFn returns a generic function that looks at
// the column metadata to map the supplied *sql.Rows to a slice of T
func (t *Table[T]) generateSqlRowsToEntityMapperFn() {
	t.mapper = func(rows *sql.Rows) (*T, error) {
		var e T
		ptr_e := reflect.ValueOf(&e)
		ve := ptr_e.Elem()

		holder := make([]any, 0)
		for _, c := range t.tableMetadata.metadata {
			if c.IsSelected {
				f := ve.FieldByName(c.FieldName).Addr().Interface() // get field by name
				if c.IsSqlArrayType {
					f = pq.Array(f)
				}
				holder = append(holder, f)
			}
		}

		// now scan result set into holder slice with correct types
		if err := rows.Scan(holder...); err != nil {
			return nil, err
		}

		return &e, nil
	}
}

// generateInsertStatement generates a template for batch INSERT statement with a column list
// based on the column metadata, in order of definition where a column is tagged insertable `dbo: "ops=w"
//
// The insert would be made FROM the static tables/arrays that would generated using parameter substitution
// with values in the supplied entities passed to Insert() at runtime.
//
// Template of INSERT statement generated:
//
// ```
//
//	  INSERT INTO schem.tab
//		(c1, c2, c3, ... cn)
//		( SELECT * FROM UNNEST( $1::type[], $2::type[] .... coln::type[]))
//
// ```
func (t *Table[T]) generateInsertStatement() {
	batchInsertTemplate := `INSERT INTO %v.%v (%v) (SELECT * FROM UNNEST (%v));`

	t.insertableColTypeArray = make([]bool, 0)

	ind := 0
	colnames := make([]string, 0)
	coltypes := make([]string, 0)
	for _, c := range t.tableMetadata.metadata {
		if c.IsInserted {
			ind++
			colnames = append(colnames, c.SqlName)
			coltypes = append(coltypes, fmt.Sprintf("$%d::%v[]", ind, c.SqlType))
			t.insertableColTypeArray = append(t.insertableColTypeArray, c.IsSqlArrayType)
		}
	}

	insertCols := strings.Join(colnames, ", ")
	selectList := strings.Join(coltypes, ", ")

	t.numInsertableCols = len(colnames) // caching number of insertable columns
	t.insertStmt = fmt.Sprintf(batchInsertTemplate, t.tableMetadata.schema, t.tableMetadata.table, insertCols, selectList)
}

// generateUpdateStatement generates a template for UPDATE statement with a column list
// based on the column metadata, in order of definition where a column is tagged insertable `dbo: "ops=a"
//
// The update would be made FROM the static tables/arrays that would generated using parameter substitution
// with values in the supplied entities passed to Insert() at runtime.
//
// Template of UPDATE statement generated:
//
// ```
//
//		UPDATE schem.tab
//		   SET
//		     c1 = dat.c1,
//		     c2 = dat.c2,
//		      :
//		     cn = dat.cn
//		   FROM (
//		       SELECT
//		              UNNEST(kc1::type[]) AS kc1,
//		              UNNEST(kc2::type[]) AS kc2,
//		              :
//		              UNNEST(c1::type[]) AS c1,
//		              UNNEST(c2::TYPE[]) AS c2
//	                  :
//		              UNNEST(ccn AS cn) dat
//		   ) dat
//		   WHERE schem.tab.k1 = dat.kc1, ... schem.tab.kcn = dat.kcn
//
// ````
func (t *Table[T]) generateUpdateStatement() {
	tmpl := `UPDATE %v.%v SET %v FROM ( SELECT %v ) dat WHERE %v;`

	setCols := make([]string, 0)
	selectCols := make([]string, 0)
	whereCols := make([]string, 0)
	ind := 0

	for _, c := range t.tableMetadata.metadata {

		if c.IsUpdated {
			setCols = append(setCols, fmt.Sprintf("%v = dat.%v", c.SqlName, c.SqlName))
			t.updatableColTypeArray = append(t.updatableColTypeArray, c.IsSqlArrayType)
		}

		if c.IsUpdated || c.IsKey {
			ind++
			selectCols = append(selectCols, fmt.Sprintf("UNNEST($%d::%v[]) AS %v", ind, c.SqlType, c.SqlName))
		}

		if c.IsKey {
			whereCols = append(whereCols, fmt.Sprintf("%v.%v.%v = dat.%v", t.tableMetadata.schema, t.tableMetadata.table, c.SqlName, c.SqlName))
		}

	}

	sets := strings.Join(setCols, ", ")
	selects := strings.Join(selectCols, ", ")
	wheres := strings.Join(whereCols, " AND ")

	t.updateStmt = fmt.Sprintf(tmpl, t.tableMetadata.schema, t.tableMetadata.table, sets, selects, wheres)
	t.numUpdatableCols = len(setCols)
}

// generateDeleteStatement generates a tempalate for the DELETE staement matching the columsn marked as `dbo:"pkey"".
// The update would be made FROM the static tables/arrays that would generated using parameter substitution
// with values in the supplied entities passed to Insert() at runtime.
//
// Template of DELETE statement generated:
//
// ```
//
// DELETE FROM schem.tab
// WHERE (kc1, kc2 ... kcn) IN
//
//	(SELECT * FROM UNNEST($1::type[], $2::type[] ... $n::type[]))
//
// ```
func (t *Table[T]) generateDeleteStatement() {

	batchDeleteTemplate :=
		`DELETE FROM %v.%v WHERE (%v) IN ( SELECT * FROM UNNEST(%v));`

	ind := 0
	keycolnames := make([]string, 0)
	keycoltypes := make([]string, 0)
	for _, c := range t.tableMetadata.metadata {
		if c.IsKey {
			ind++
			keycolnames = append(keycolnames, c.SqlName)
			keycoltypes = append(keycoltypes, fmt.Sprintf("$%d::%v[]", ind, c.SqlType))
		}
	}

	keyCols := strings.Join(keycolnames, ", ")
	selectList := strings.Join(keycoltypes, ", ")
	t.deleteStmt = fmt.Sprintf(batchDeleteTemplate, t.tableMetadata.schema, t.tableMetadata.table, keyCols, selectList)
}

// determineSchemaTableName uses the supplied var arg string parms to return
// schema & table names.
//
// the following rules are followed:
// if len(parm) == 0, return err
// if len(parm) == 1, then it is table_name or schema.table_name
// if len(parm) == 2, then it is schema & table
// default schema name when not inferred is `public`
func determineSchemaTableName(tableName ...string) (*string, *string, error) {
	var table string
	schema := "public"

	if len(tableName) == 0 {
		return nil, nil, fmt.Errorf("table name must be supplied")
	}

	if len(tableName) == 1 {
		table = tableName[0]
		if strings.Contains(table, ".") {
			//["schema.table"]
			tk := strings.Split(table, ".")
			schema = tk[0]
			table = tk[1]
		}
	} else if len(tableName) == 2 {
		//["schema","table"]
		schema = tableName[0]
		table = tableName[1]
	}

	return &schema, &table, nil
}

// create a 2D-array, [len][]any
func (t Table[T]) makeParameterArray(len int) [][]any {
	arr := make([][]any, 0)
	for i := 0; i < len; i++ {
		arr = append(arr, make([]any, 0))
	}
	return arr
}
