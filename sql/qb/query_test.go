package qb

import (
	"strings"
	"testing"
)

// QueryLeft and QueryRight are minimal entity types for join query tests.

type QueryLeft struct {
	LeftId   int64  `qb:"left_id,pk,type=BIGINT,ops=r"`
	LeftName string `qb:"left_name,type=VARCHAR,ops=r"`
}

func (q QueryLeft) Key() PrimaryKey           { return q.LeftId }
func (q QueryLeft) Equals(other QueryLeft) bool { return false }

type QueryRight struct {
	RightId   int64  `qb:"right_id,pk,type=BIGINT,ops=r"`
	RightName string `qb:"right_name,type=VARCHAR,ops=r"`
}

func (q QueryRight) Key() PrimaryKey            { return q.RightId }
func (q QueryRight) Equals(other QueryRight) bool { return false }

// CompositeLeftJoinEntity is a composite struct using LeftJoin.
type CompositeLeftJoinEntity struct {
	Left  QueryLeft
	Join1 LeftJoin  `qbon:"left.left_id = right.right_id"`
	Right QueryRight
}

func (c CompositeLeftJoinEntity) Key() PrimaryKey                                    { return nil }
func (c CompositeLeftJoinEntity) Equals(other CompositeLeftJoinEntity) bool          { return false }

// CompositeInnerJoinEntity is a composite struct using InnerJoin.
type CompositeInnerJoinEntity struct {
	Left  QueryLeft
	Join1 InnerJoin `qbon:"left.left_id = right.right_id"`
	Right QueryRight
}

func (c CompositeInnerJoinEntity) Key() PrimaryKey                                    { return nil }
func (c CompositeInnerJoinEntity) Equals(other CompositeInnerJoinEntity) bool          { return false }

func TestForQuery_LeftJoin_SelectStmtNonEmpty(t *testing.T) {
	q, err := ForQuery[CompositeLeftJoinEntity]()
	if err != nil {
		t.Fatalf("ForQuery returned unexpected error: %v", err)
	}

	if q == nil {
		t.Fatal("ForQuery returned nil query")
	}

	if q.selectStmt == "" {
		t.Error("selectStmt should be non-empty after ForQuery initialization")
	}
}

func TestForQuery_LeftJoin_SelectStmtContainsLeftJoin(t *testing.T) {
	q, err := ForQuery[CompositeLeftJoinEntity]()
	if err != nil {
		t.Fatalf("ForQuery returned unexpected error: %v", err)
	}

	if !strings.Contains(q.selectStmt, "LEFT JOIN") {
		t.Errorf("selectStmt should contain LEFT JOIN, got: %v", q.selectStmt)
	}
}

func TestForQuery_InnerJoin_SelectStmtContainsJoin(t *testing.T) {
	q, err := ForQuery[CompositeInnerJoinEntity]()
	if err != nil {
		t.Fatalf("ForQuery returned unexpected error: %v", err)
	}

	if !strings.Contains(q.selectStmt, "JOIN") {
		t.Errorf("selectStmt should contain JOIN, got: %v", q.selectStmt)
	}
}

func TestForQuery_SelectStmtContainsSelectedColumns(t *testing.T) {
	q, err := ForQuery[CompositeLeftJoinEntity]()
	if err != nil {
		t.Fatalf("ForQuery returned unexpected error: %v", err)
	}

	if !strings.Contains(q.selectStmt, "left_id") {
		t.Errorf("selectStmt should contain left_id, got: %v", q.selectStmt)
	}

	if !strings.Contains(q.selectStmt, "right_id") {
		t.Errorf("selectStmt should contain right_id, got: %v", q.selectStmt)
	}
}

// CompositeRightJoinEntity tests RightJoin path in isJoin
type CompositeRightJoinEntity struct {
	Left  QueryLeft
	Join1 RightJoin `qbon:"left.left_id = right.right_id"`
	Right QueryRight
}

func (c CompositeRightJoinEntity) Key() PrimaryKey                                    { return nil }
func (c CompositeRightJoinEntity) Equals(other CompositeRightJoinEntity) bool          { return false }

func TestForQuery_RightJoin_SelectStmtContainsRightJoin(t *testing.T) {
	q, err := ForQuery[CompositeRightJoinEntity]()
	if err != nil {
		t.Fatalf("ForQuery returned unexpected error: %v", err)
	}

	if !strings.Contains(q.selectStmt, "RIGHT JOIN") {
		t.Errorf("selectStmt should contain RIGHT JOIN, got: %v", q.selectStmt)
	}
}

// QueryWithAlias tests the alias path in tableMetadataToSelectList
type QueryWithAlias struct {
	MyId int64 `qb:"my_id,pk,type=BIGINT,ops=r,as=alias_id"`
}

func (q QueryWithAlias) Key() PrimaryKey              { return q.MyId }
func (q QueryWithAlias) Equals(other QueryWithAlias) bool { return false }

type QueryWithAlias2 struct {
	Id int64 `qb:"id,pk,type=BIGINT,ops=r"`
}

func (q QueryWithAlias2) Key() PrimaryKey               { return q.Id }
func (q QueryWithAlias2) Equals(other QueryWithAlias2) bool { return false }

type CompositeAliasJoinEntity struct {
	Left  QueryWithAlias
	Join1 LeftJoin       `qbon:"left.my_id = right.id"`
	Right QueryWithAlias2
}

func (c CompositeAliasJoinEntity) Key() PrimaryKey                                    { return nil }
func (c CompositeAliasJoinEntity) Equals(other CompositeAliasJoinEntity) bool          { return false }

func TestForQuery_AliasedColumn_SelectStmtContainsAS(t *testing.T) {
	q, err := ForQuery[CompositeAliasJoinEntity]()
	if err != nil {
		t.Fatalf("ForQuery returned unexpected error: %v", err)
	}

	if !strings.Contains(q.selectStmt, " AS ") {
		t.Errorf("selectStmt should contain AS for aliased column, got: %v", q.selectStmt)
	}
}

// Tests for convertQbjTagValueToJoinClause and op_string via qb tag path
type QbTagJoinEntity struct {
	Left  QueryLeft
	Join1 LeftJoin  `qb:"left_id=right_id"`
	Right QueryRight
}

func (c QbTagJoinEntity) Key() PrimaryKey                                 { return nil }
func (c QbTagJoinEntity) Equals(other QbTagJoinEntity) bool               { return false }

func TestForQuery_QbTagJoin(t *testing.T) {
	q, err := ForQuery[QbTagJoinEntity]()
	if err != nil {
		t.Fatalf("ForQuery with qb tag join returned unexpected error: %v", err)
	}
	if q == nil {
		t.Fatal("expected non-nil query")
	}
}

// Test error path: invalid qb tag value (no operator)
type InvalidQbTagJoinEntity struct {
	Left  QueryLeft
	Join1 LeftJoin  `qb:"invalidtoken"`
	Right QueryRight
}

func (c InvalidQbTagJoinEntity) Key() PrimaryKey                                       { return nil }
func (c InvalidQbTagJoinEntity) Equals(other InvalidQbTagJoinEntity) bool              { return false }

func TestForQuery_InvalidQbTag_ReturnsError(t *testing.T) {
	_, err := ForQuery[InvalidQbTagJoinEntity]()
	if err == nil {
		t.Error("expected error for invalid qb join tag, got nil")
	}
}

// Test op_string via convertQbjTagValueToJoinClause with various operators
func TestConvertQbjTagValueToJoinClause_Operators(t *testing.T) {
	tests := []struct {
		name    string
		val     string
		wantErr bool
	}{
		{"equals", "left_id=right_id", false},
		{"gte", "left_id>=right_id", false},
		{"lte", "left_id<=right_id", false},
		{"neq", "left_id!=right_id", false},
		{"gt", "left_id>right_id", false},
		{"lt", "left_id<right_id", false},
		{"invalid", "invalidtoken", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := convertQbjTagValueToJoinClause(tt.val)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %v, got nil", tt.val)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %v: %v", tt.val, err)
			}
		})
	}
}

func TestOpString_InvalidTokens(t *testing.T) {
	result := op_string("a=b=c", "=")
	if len(result) != 3 || result[0] != "909" {
		t.Errorf("expected invalid token result {909, =, 909}, got %v", result)
	}
}

func TestOpString_WithParam(t *testing.T) {
	result := op_string("left_id=$1", "=")
	if len(result) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(result))
	}
	if result[2] != "$1" {
		t.Errorf("expected $1 to be preserved, got %v", result[2])
	}
}
