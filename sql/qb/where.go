package qb

import (
	"fmt"
	"strings"
)

type WhereClause interface {
	WhereClause() string
}

type WhereAll struct{}

func (w WhereAll) WhereClause() string { return "WHERE 1=1" }

type WhereNone struct{}

func (w WhereNone) WhereClause() string { return "WHERE 1=0" }

type WhereString string

func (w WhereString) WhereClause() string { return string(w) }

type WhereEq[T Entity[T]] struct {
}

func (w WhereEq[T]) WhereClause(t Table[T]) string {

	var ind int
	keycols := make([]string, 0)
	for _, c := range t.tableMetadata.metadata {
		if c.IsKey {
			ind++
			keycols = append(keycols, fmt.Sprintf("%v = $%d", c.SqlName, ind))
		}

	}
	return strings.Join(keycols, " AND ")
}
