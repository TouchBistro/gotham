package qb

import (
	"testing"
)

// minEntity is a minimal stub implementing Entity[minEntity] for testing Convert and Convert0.
type minEntity struct {
	id  int
	val string
}

func (m minEntity) Key() PrimaryKey        { return m.id }
func (m minEntity) Equals(other minEntity) bool { return m.val == other.val }

func TestConvert_AddUpdateDelete(t *testing.T) {
	existing := []minEntity{
		{id: 1, val: "a"},
		{id: 2, val: "b"},
	}
	newEntities := []minEntity{
		{id: 1, val: "a_changed"},
		{id: 3, val: "c"},
	}

	add, upd, del := Convert(existing, newEntities)

	if len(add) != 1 {
		t.Errorf("expected 1 add, got %d", len(add))
	}
	if len(upd) != 1 {
		t.Errorf("expected 1 update, got %d", len(upd))
	}
	if len(del) != 1 {
		t.Errorf("expected 1 delete, got %d", len(del))
	}
}

func TestConvert_EmptyExisting(t *testing.T) {
	var existing []minEntity
	newEntities := []minEntity{
		{id: 1, val: "a"},
	}

	add, upd, del := Convert(existing, newEntities)

	if len(add) != 1 {
		t.Errorf("expected 1 add, got %d", len(add))
	}
	if len(upd) != 0 {
		t.Errorf("expected 0 updates, got %d", len(upd))
	}
	if len(del) != 0 {
		t.Errorf("expected 0 deletes, got %d", len(del))
	}
}

func TestConvert_EmptyNew(t *testing.T) {
	existing := []minEntity{
		{id: 1, val: "a"},
	}
	var newEntities []minEntity

	add, upd, del := Convert(existing, newEntities)

	if len(add) != 0 {
		t.Errorf("expected 0 adds, got %d", len(add))
	}
	if len(upd) != 0 {
		t.Errorf("expected 0 updates, got %d", len(upd))
	}
	if len(del) != 1 {
		t.Errorf("expected 1 delete, got %d", len(del))
	}
}

func TestConvert0_AddUpdateDelete(t *testing.T) {
	existing := []minEntity{
		{id: 1, val: "a"},
		{id: 2, val: "b"},
	}
	newEntities := []minEntity{
		{id: 1, val: "a_changed"},
		{id: 3, val: "c"},
	}

	add, upd, del := Convert0(existing, newEntities)

	if len(add) != 1 {
		t.Errorf("expected 1 add, got %d", len(add))
	}
	if len(upd) != 1 {
		t.Errorf("expected 1 update, got %d", len(upd))
	}
	if len(del) != 1 {
		t.Errorf("expected 1 delete, got %d", len(del))
	}
}

func TestConvert0_NoChange(t *testing.T) {
	existing := []minEntity{
		{id: 1, val: "a"},
	}
	newEntities := []minEntity{
		{id: 1, val: "a"},
	}

	add, upd, del := Convert0(existing, newEntities)

	if len(add) != 0 {
		t.Errorf("expected 0 adds, got %d", len(add))
	}
	if len(upd) != 0 {
		t.Errorf("expected 0 updates, got %d", len(upd))
	}
	if len(del) != 0 {
		t.Errorf("expected 0 deletes, got %d", len(del))
	}
}
