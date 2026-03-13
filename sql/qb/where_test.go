package qb

import "testing"

func TestWhereAll_WhereClause(t *testing.T) {
	w := WhereAll{}
	got := w.WhereClause()
	want := "WHERE 1=1"
	if got != want {
		t.Errorf("WhereAll.WhereClause() = %q; want %q", got, want)
	}
}

func TestWhereNone_WhereClause(t *testing.T) {
	w := WhereNone{}
	got := w.WhereClause()
	want := "WHERE 1=0"
	if got != want {
		t.Errorf("WhereNone.WhereClause() = %q; want %q", got, want)
	}
}

func TestWhereString_WhereClause(t *testing.T) {
	w := WhereString("WHERE x=1")
	got := w.WhereClause()
	want := "WHERE x=1"
	if got != want {
		t.Errorf("WhereString.WhereClause() = %q; want %q", got, want)
	}
}

func TestWhereEq_WhereClause_SingleKey(t *testing.T) {
	table, err := ForTable[Test]("schem.tab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	eq := WhereEq[Test]{}
	got := eq.WhereClause(*table)
	want := "id = $1"
	if got != want {
		t.Errorf("WhereEq.WhereClause() = %q; want %q", got, want)
	}
}
