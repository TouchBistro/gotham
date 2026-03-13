package qb

import (
	"reflect"
	"testing"
	"time"
)

func TestSankeCase(t *testing.T) {
	val := "ColumnName"
	exp := "column_name"
	snk := toSnakeCase(val)
	if snk != exp {
		t.Errorf("\nExpected: %v\nFound   : %v", exp, snk)
	}
}

func TestSankeCase_StartWithLowerCase(t *testing.T) {
	val := "anotherColumnName"
	exp := "another_column_name"
	snk := toSnakeCase(val)
	if snk != exp {
		t.Errorf("\nSupplied: %v\nExpected: %v\nFound   : %v", val, exp, snk)
	}
}

func TestSankeCase_StartWithNumberLowerCase(t *testing.T) {
	val := "33invalidColumnName"
	exp := "33invalid_column_name"
	snk := toSnakeCase(val)
	if snk != exp {
		t.Errorf("\nSupplied: %v\nExpected: %v\nFound   : %v", val, exp, snk)
	}
}

func TestSankeCase_StartWithNumbeUpperCase(t *testing.T) {
	val := "2Tricky3Column4Name"
	exp := "2_tricky3_column4_name"
	snk := toSnakeCase(val)
	if snk != exp {
		t.Errorf("\nSupplied: %v\nExpected: %v\nFound   : %v", val, exp, snk)
	}
}

func TestSankeCase_CamelSnakeeCase(t *testing.T) {
	val := "Camel_Snake_Case"
	exp := "camel_snake_case"
	snk := toSnakeCase(val)
	if snk != exp {
		t.Errorf("\nSupplied: %v\nExpected: %v\nFound   : %v", val, exp, snk)
	}
}

func TestSqlTypeForType_Int32(t *testing.T) {
	type sqlInt32Struct struct {
		Val int32
	}
	import_type := reflect.TypeOf(sqlInt32Struct{})
	f := import_type.Field(0)
	sqlType, isArray := SqlTypeForType(f)
	if sqlType != "INTEGER" {
		t.Errorf("expected INTEGER, got %v", sqlType)
	}
	if isArray {
		t.Errorf("expected isArray=false")
	}
}

func TestSqlTypeForType_Int64(t *testing.T) {
	type sqlInt64Struct struct {
		Val int64
	}
	import_type := reflect.TypeOf(sqlInt64Struct{})
	f := import_type.Field(0)
	sqlType, isArray := SqlTypeForType(f)
	if sqlType != "BIGINT" {
		t.Errorf("expected BIGINT, got %v", sqlType)
	}
	if isArray {
		t.Errorf("expected isArray=false")
	}
}

func TestSqlTypeForType_String(t *testing.T) {
	type sqlStringStruct struct {
		Val string
	}
	import_type := reflect.TypeOf(sqlStringStruct{})
	f := import_type.Field(0)
	sqlType, isArray := SqlTypeForType(f)
	if sqlType != "VARCHAR" {
		t.Errorf("expected VARCHAR, got %v", sqlType)
	}
	if isArray {
		t.Errorf("expected isArray=false")
	}
}

func TestSqlTypeForType_PointerString(t *testing.T) {
	type sqlPtrStruct struct {
		Val *string
	}
	import_type := reflect.TypeOf(sqlPtrStruct{})
	f := import_type.Field(0)
	sqlType, isArray := SqlTypeForType(f)
	if sqlType != "VARCHAR" {
		t.Errorf("expected VARCHAR for pointer string, got %v", sqlType)
	}
	if isArray {
		t.Errorf("expected isArray=false for pointer")
	}
}

func TestSqlTypeForType_SliceString(t *testing.T) {
	type sqlSliceStruct struct {
		Val []string
	}
	import_type := reflect.TypeOf(sqlSliceStruct{})
	f := import_type.Field(0)
	sqlType, isArray := SqlTypeForType(f)
	if sqlType != "VARCHAR[]" {
		t.Errorf("expected VARCHAR[] for slice string, got %v", sqlType)
	}
	if !isArray {
		t.Errorf("expected isArray=true for slice")
	}
}

func TestParseColumnMetadata_PkeyTag(t *testing.T) {
	type pkeyStruct struct {
		MyId int64 `qb:"my_id,pkey,ops=r"`
	}
	import_type := reflect.TypeOf(pkeyStruct{})
	f := import_type.Field(0)
	meta := parseColumnMetadataFromStructField(f)
	if !meta.IsKey {
		t.Errorf("expected IsKey=true for pkey tag")
	}
}

func TestParseColumnMetadata_AliasSupplied(t *testing.T) {
	type aliasStruct struct {
		MyName string `qb:"my_name,as=alias_name,ops=r"`
	}
	import_type := reflect.TypeOf(aliasStruct{})
	f := import_type.Field(0)
	meta := parseColumnMetadataFromStructField(f)
	if meta.Alias != "alias_name" {
		t.Errorf("expected Alias=alias_name, got %v", meta.Alias)
	}
	if meta.SqlName != "my_name" {
		t.Errorf("expected SqlName=my_name, got %v", meta.SqlName)
	}
}

func TestParseColumnMetadata_NoTag(t *testing.T) {
	type noTagStruct struct {
		MyField string
	}
	import_type := reflect.TypeOf(noTagStruct{})
	f := import_type.Field(0)
	meta := parseColumnMetadataFromStructField(f)
	if meta.SqlName != "my_field" {
		t.Errorf("expected SqlName=my_field, got %v", meta.SqlName)
	}
	if meta.IsSelected || meta.IsInserted || meta.IsUpdated || meta.IsKey {
		t.Errorf("expected all flags false for no-tag field")
	}
}

func TestParseColumnMetadata_OpsInsertUpdate(t *testing.T) {
	type opsAWStruct struct {
		MyField string `qb:"my_field,ops=aw"`
	}
	import_type := reflect.TypeOf(opsAWStruct{})
	f := import_type.Field(0)
	meta := parseColumnMetadataFromStructField(f)
	if !meta.IsInserted {
		t.Errorf("expected IsInserted=true for ops=aw")
	}
	if !meta.IsUpdated {
		t.Errorf("expected IsUpdated=true for ops=aw")
	}
	if meta.IsSelected {
		t.Errorf("expected IsSelected=false for ops=aw")
	}
}

func TestSqlTypeForType_TimeTime(t *testing.T) {
	type timeStruct struct {
		Val time.Time
	}
	import_type := reflect.TypeOf(timeStruct{})
	f := import_type.Field(0)
	sqlType, isArray := SqlTypeForType(f)
	if sqlType != "TIMESTAMP" {
		t.Errorf("expected TIMESTAMP for time.Time, got %v", sqlType)
	}
	if isArray {
		t.Errorf("expected isArray=false for time.Time")
	}
}
