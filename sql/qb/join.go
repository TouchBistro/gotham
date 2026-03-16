package qb

type Join interface {
	ToJoinString() string
}

type LeftJoin struct{}

func (j LeftJoin) ToJoinString() string { return "" }

type RightJoin struct{}

func (r RightJoin) ToJoinString() string { return "" }

type InnerJoin struct{}

func (i InnerJoin) ToJoinString() string { return "" }
