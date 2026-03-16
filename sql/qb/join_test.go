package qb

import "testing"

func TestLeftJoin_ToJoinString(t *testing.T) {
	j := LeftJoin{}
	got := j.ToJoinString()
	if got != "" {
		t.Errorf("LeftJoin.ToJoinString() = %q; want %q", got, "")
	}
}

func TestRightJoin_ToJoinString(t *testing.T) {
	j := RightJoin{}
	got := j.ToJoinString()
	if got != "" {
		t.Errorf("RightJoin.ToJoinString() = %q; want %q", got, "")
	}
}

func TestInnerJoin_ToJoinString(t *testing.T) {
	j := InnerJoin{}
	got := j.ToJoinString()
	if got != "" {
		t.Errorf("InnerJoin.ToJoinString() = %q; want %q", got, "")
	}
}
