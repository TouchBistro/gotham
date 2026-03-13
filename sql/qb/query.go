package qb

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/TouchBistro/gotham/sql/qb/tmp"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// ForQuery creates a
func ForQuery[T Entity[T]]() (*Query[T], error) {

	q := &Query[T]{}
	if err := q.generateColsMetadata(); err != nil {
		return nil, err
	}

	q.selectBatchSize = 100 // TODO parameterize OR auto-detect
	q.generateSelectStatement()
	q.generateSqlRowsToEntityMapperFn()
	log.Info(q.selectStmt)

	return q, nil
}

type Query[T Entity[T]] struct {
	metadata        []TableMetadata
	aliases         []string
	joinNames       []string
	joinDefs        []string
	selectStmt      string
	selectBatchSize int
	mapper          SqlRowsToEntityMapperFn[T]
}

// SelectWhere selects all rows from the underlying database query after applying the supplied "Where" condition & arguments into this Table object
func (q Query[T]) SelectWhere(ctx context.Context, conn *sql.DB, where WhereClause, args ...any) ([]T, error) {

	// start transaction
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}

	if rows, err := q.SelectWhereTx(ctx, tx, where, args...); err != nil {
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

// SelectWhereTx selects all rows from the underlying database query after applying the supplied "Where" condition & arguments into this Table object  on the given transaction context
func (t Query[T]) SelectWhereTx(ctx context.Context, tx *sql.Tx, where WhereClause, args ...any) ([]T, error) {

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
	defer rs.Close()

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

// Selects all rows using the supplied database connection from the underlying Query
func (t Query[T]) Select(ctx context.Context, conn *sql.DB, args ...any) ([]T, error) {
	return t.SelectWhere(ctx, conn, WhereAll{}, args...)
}

// SelectsTx all rows using the supplied database connection from the underlying Query on the given transaction context
func (t Query[T]) SelectTx(ctx context.Context, tx *sql.Tx, args ...any) ([]T, error) {
	return t.SelectWhereTx(ctx, tx, WhereAll{}, args...)
}

// generateSelectStatement generates the SELECT statement for this query
// including any JOIN
func (q *Query[T]) generateSelectStatement() {
	tmpl := "SELECT %v FROM %v"

	from := ""
	selists := make([]string, 0)
	joinStrs := make([]string, 0)

	for n, tmd := range q.metadata {
		selists = append(selists, q.tableMetadataToSelectList(q.aliases[n], tmd)...)

		// for the first table, we make a FROM ...
		if n == 0 {
			from = fmt.Sprintf("%v.%v %v", tmd.schema, tmd.table, q.aliases[n])
		}
		if n < len(q.metadata)-1 {
			nextTmd := q.metadata[n+1]
			onString := q.joinDefs[n]
			joinStrs = append(joinStrs,
				fmt.Sprintf("%v %v.%v %v %v", q.joinNames[n], nextTmd.schema, nextTmd.table, q.aliases[n+1], onString))
		}
	}

	from = fmt.Sprintf("%v %v", from, strings.Join(joinStrs, " "))
	q.selectStmt = fmt.Sprintf(tmpl, strings.Join(selists, ","), from)
}

// generateColsMetadata for a query creates a column & join metadata that
// can be used to build a SELECT statement & row mapper
func (q *Query[T]) generateColsMetadata() error {

	tmds := make([]TableMetadata, 0)

	taliases := make([]string, 0)
	joinNames := make([]string, 0)
	// joinDefs := make([][][]string, 0)
	joinDefStrs := make([]string, 0)

	var e T
	typeOfEle := reflect.TypeOf(e)
	kindOfEle := typeOfEle.Kind()
	log.Debugf("type: %v kind: %v", typeOfEle, kindOfEle)

	if kindOfEle == reflect.Struct {
		for i := 0; i < typeOfEle.NumField(); i++ {

			field := typeOfEle.Field(i)
			log.Debugf("field_name=%v type=%v, kind=%v", field.Name, field.Type, field.Type.Kind())

			typeOfField := field.Type

			if joinStr, isJoin := isJoin(typeOfField); !isJoin {
				// generate table metadata from type
				tmd, err := reflectTypeToTableMetadata(field.Type)
				if err != nil {
					return err
				}
				tmd.field = tmp.ToStringPtr(field.Name) // here it's a field, so we add thi
				tmds = append(tmds, *tmd)
				taliases = append(taliases, toSnakeCase(field.Name))
			} else {
				joinNames = append(joinNames, *joinStr)
				qbVal := field.Tag.Get(QbStructMetaTagKey)
				if qbVal != "" {
					// TODO clean this mess..
					_, err := convertQbjTagValueToJoinClause(qbVal)
					if err != nil {
						return err
					}
					jval := ""
					joinDefStrs = append(joinDefStrs, jval)
				} else {
					qbonVal := field.Tag.Get(QbonStructMetaTagKey)
					joinDefStrs = append(joinDefStrs, fmt.Sprintf("ON %v", qbonVal))
				}

			}
		}
	} else {
		return fmt.Errorf("supplied Entity type %v is not a struct", kindOfEle)
	}

	if len(tmds) != len(taliases) {
		return fmt.Errorf("something went wrong, number of query sub-table metadatas and aliases are different")
	}

	q.aliases = taliases
	q.metadata = tmds
	q.joinNames = joinNames
	q.joinDefs = joinDefStrs
	return nil
}

// tableMetadataToSelectList converts the supplied table metadata object to a SELECT list
func (q *Query[T]) tableMetadataToSelectList(alias string, tmd TableMetadata) []string {
	selist := make([]string, 0)
	for _, m := range tmd.metadata {
		if m.IsSelected {
			if m.SqlName != m.Alias {
				selist = append(selist, fmt.Sprintf("%v.%v AS \"%v\"", alias, m.SqlName, m.Alias))
			} else {
				selist = append(selist, fmt.Sprintf("%v.%v", alias, m.SqlName))
			}
		}
	}
	return selist
}

// generateSqlRowsToEntityMapperFn returns a generic function that looks at
// the column metadata to map the supplied *sql.Rows to a slice of T
func (t *Query[T]) generateSqlRowsToEntityMapperFn() {
	t.mapper = func(rows *sql.Rows) (*T, error) {
		var e T
		ptr_e := reflect.ValueOf(&e)
		ve := ptr_e.Elem()

		holder := make([]any, 0)
		for _, tmd := range t.metadata {
			if tmd.field == nil {
				return nil, fmt.Errorf("table metadata does not have field names")
			}
			fld := ve.FieldByName(*tmd.field)
			fld_ptr := fld.Addr()
			fld_ptr_defref := fld_ptr.Elem() // value pointed to
			for _, c := range tmd.metadata {
				if c.IsSelected {
					f := fld_ptr_defref.FieldByName(c.FieldName).Addr().Interface() // get field by name
					if c.IsSqlArrayType {
						f = pq.Array(f)
					}
					holder = append(holder, f)
				}
			}
		}

		// now scan result set into holder slice with correct types
		if err := rows.Scan(holder...); err != nil {
			return nil, err
		}

		return &e, nil
	}
}

// isJoin looks at the reflect.Type supplied & returns the
// SQL string representtion
func isJoin(t reflect.Type) (*string, bool) {
	var lj LeftJoin
	var rj RightJoin
	var j InnerJoin

	var joinStr *string
	isJoin := false

	switch t {
	case reflect.TypeOf(lj):
		joinStr = tmp.ToStringPtr("LEFT JOIN")
		isJoin = true

	case reflect.TypeOf(rj):
		joinStr = tmp.ToStringPtr("RIGHT JOIN")
		isJoin = true

	case reflect.TypeOf(j):
		joinStr = tmp.ToStringPtr("JOIN")
		isJoin = true
	}

	return joinStr, isJoin
}

// convertQbjTagValuesToJoinClause takes the qbj tag values & converts them
// to a SQL string for a Join ON.
func convertQbjTagValueToJoinClause(val string) ([][]string, error) {

	jClause := make([][]string, 0)

	tokens := strings.Split(val, ",")
	for _, token := range tokens {
		if strings.Contains(token, ">=") {
			jClause = append(jClause, op_string(token, ">="))
		} else if strings.Contains(token, "<=") {
			jClause = append(jClause, op_string(token, "<="))
		} else if strings.Contains(token, "!=") {
			jClause = append(jClause, op_string(token, "!="))
		} else if strings.Contains(token, "=") {
			jClause = append(jClause, op_string(token, "="))
		} else if strings.Contains(token, ">") {
			jClause = append(jClause, op_string(token, ">"))
		} else if strings.Contains(token, "<") {
			jClause = append(jClause, op_string(token, "<"))
		} else {
			return nil, fmt.Errorf("error converting obj tag value %v to join clause", val)
		}
	}

	return jClause, nil

}

// op_string splits supplied token by the delim (op) into
// column_name op column_name2 OR column_name op $1 etc
func op_string(str, delim string) []string {

	tokens := strings.Split(str, delim)

	// invalid tokens
	if len(tokens) != 2 {
		return []string{"909", delim, "909"}
	}

	for n, token := range tokens {
		if !strings.Contains(token, "$") { // contains a param
			tokens[n] = toSnakeCase(token)
		}
	}
	return []string{tokens[0], delim, tokens[1]}
}
