package qb

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	QbStructMetaTagKey   string = "qb"
	QbonStructMetaTagKey string = "qbon"

	QbOpsValueSelect string = "r"
	QbOpsValueInsert string = "a"
	QbOpsValueUpdate string = "w"

	QbTagValuePk   string = "pk"
	QbTagValuePkey string = "pkey"
)

type ColumnMetadata struct {
	SqlName        string       // database column nmae
	SqlType        string       // database column type
	IsSqlArrayType bool         // database column is array
	Alias          string       // column alias to use for SELECTs
	FieldName      string       // struct field name
	Type           reflect.Type // struct field type
	IsSelected     bool         // is selected
	IsInserted     bool         // is inserted
	IsUpdated      bool         // is updated
	IsKey          bool         // is it a primary key
}

// ParseColumnMetadataFromStructField builds a column metadata struct for the the
// subpplied struct field. The following rules are used
//
// The value for the `qb` stuct tag  is used for hints to fill up Column Metadata from struct fields if cannot be inferred
// The value is parsed into token delimted by `,` & each token is either key-less or key-value pair with a `=` separator
//
//	FieldName `qb:"keyless1,key1=value1,key2=value2,key3=value3,keyless2,keyless3...."`
//
// # Here are the rules of how the metadata is extracted
//
// Name: initialized with the snake_case version of the field name; If supplied as a keyless value of the struct tag key `qb` then that value is used instea
// SqlType: initialized with a inteferred SQL equivalend, else the supplied value for type key; type=?
// Alias: initialized to be the same as SqlName, or read from value of the as key, as=?
// FieldName: name of the struct field
// Type: reflect.Type of the struct field
// isSelected: if the value of the ops key has r|R in it or a keyless r|R value exists for qb tag key
// isInserted: if the value of the ops key has a|A in it or a keyless a|A value exists for qb tag key
// isUpdated: if the value of the ops key has w|W in it or a keyless w|W value exists for qb tag key
// isKey: if the keyless value pk|PK|pkey|PKEY exists for qb tag key
func parseColumnMetadataFromStructField(f reflect.StructField) ColumnMetadata {

	sqltyp, isArray := SqlTypeForType(f)

	meta := ColumnMetadata{
		SqlName:        toSnakeCase(f.Name), // initialize to field name
		SqlType:        sqltyp,
		IsSqlArrayType: isArray,
		Alias:          toSnakeCase(f.Name),
		FieldName:      f.Name,
		Type:           f.Type,
	}

	var aliasSupplied bool
	tag := f.Tag.Get(QbStructMetaTagKey)

	if len(strings.TrimSpace(tag)) == 0 {
		return meta
	}

	tokens := strings.Split(tag, ",")
	for _, token := range tokens {
		if strings.ToLower(strings.TrimSpace(token)) == QbOpsValueSelect { // read or SELECT
			meta.IsSelected = true
		} else if strings.ToLower(strings.TrimSpace(token)) == QbOpsValueInsert { // append or INSERT
			meta.IsInserted = true
		} else if strings.ToLower(strings.TrimSpace(token)) == QbOpsValueUpdate { // write or UPDATE
			meta.IsUpdated = true
		} else if strings.ToLower(strings.TrimSpace(token)) == QbTagValuePk || strings.ToLower(strings.TrimSpace(token)) == QbTagValuePkey {
			meta.IsKey = true
		} else if strings.HasPrefix(token, "type=") {
			meta.SqlType = strings.TrimPrefix(token, "type=")
		} else if strings.HasPrefix(token, "as=") {
			meta.Alias = strings.TrimPrefix(token, "as=")
			aliasSupplied = true
		} else if strings.HasPrefix(token, "ops=") {
			ops := strings.TrimPrefix(token, "ops=")
			for _, c := range ops {
				switch c {
				case 'r', 'R':
					meta.IsSelected = true
				case 'a', 'A':
					meta.IsInserted = true
				case 'w', 'W':
					meta.IsUpdated = true
				}
			}
		} else {
			meta.SqlName = strings.TrimSpace(token)
		}
	}

	// if alias not supplied explicity, we
	// ensure the name & alias remain same
	if !aliasSupplied {
		meta.Alias = meta.SqlName
	}

	log.Debugf("%v", meta)
	return meta
}

// SqlTypeForType returns a SQL type inferred from the SQL type
func SqlTypeForType(f reflect.StructField) (string, bool) {

	var isArray bool

	t := f.Type
	k := t.Kind()

	rc := t.String()

	if t.Kind() == reflect.Pointer {
		rc = strings.TrimPrefix(rc, "*")
	} else if t.Kind() == reflect.Slice {
		rc = strings.TrimPrefix(rc, "[]")
		isArray = true
	}

	switch rc {
	case "int", "int32":
		rc = "INTEGER"
	case "int64":
		rc = "BIGINT"
	case "string":
		rc = "VARCHAR"
	case "time.Time":
		rc = "TIMESTAMP"
	}

	// if slice then we add [] to the type
	if t.Kind() == reflect.Slice {
		rc = fmt.Sprintf("%v[]", rc)
	}

	log.Debugf("result=%v, for=> %v %v %v", rc, f.Name, t, k)
	return rc, isArray
}

// tableNameFromType generate schema.table name from the supplied type
// the following rules are applied:
//
// Convert fully qualitified golang type name package.type to string
// Ignore the package part & use typeName only.
//
// if the typeName contains an "_", then it's assumed a schema name is also supplied
// If the schema is not inferred, `public` is the default value that is returned
// Each segment of the type name split by first `_` is converted to snake case
// First segement is schema, 2nd segment is table name
func tableNameFromType(ty reflect.Type) (string, string) {
	schema := "public"
	var table string

	name := fmt.Sprintf("%v", ty)
	// remove package name from the type name
	if strings.Contains(name, ".") {
		name = name[strings.Index(name, ".")+1:]
	}

	// check if an `_` exists, then split by first
	schemaSegment := ""
	tableSegment := name

	if strings.Contains(name, "_") {
		idx := strings.Index(name, "_")
		schemaSegment = name[0:idx]
		tableSegment = name[idx+1:]
	}

	if schemaSegment != "" {
		schema = toSnakeCase(schemaSegment)
	}

	table = toSnakeCase(tableSegment)
	return schema, table
}

// toSnakeCase converts the supplied string to snake case
func toSnakeCase(in string) string {
	var regexStrStartsWithLowerOrNumeric = regexp.MustCompile("(^[a-z0-9_]+)")     // match only 1st match if the str starts with a small/uscore/digit
	var regexpSubTokenStartingWithCaps = regexp.MustCompile("([A-Z]([a-z0-9_])+)") // match any substr that starts with a Caps & folled by small/uscore/digits
	var doubleUnderScoreMatcher = regexp.MustCompile("(__)")                       // match double uscore
	out := regexStrStartsWithLowerOrNumeric.ReplaceAllString(in, "${1}_")          // replace matching with match_
	out = regexpSubTokenStartingWithCaps.ReplaceAllString(out, "${1}_")            // replace matching with match_
	out = doubleUnderScoreMatcher.ReplaceAllString(out, "_")                       // replace matching with single _
	out = strings.TrimSuffix(out, "_")                                             //replace any underscore placed at the very end
	return strings.ToLower(out)
}
