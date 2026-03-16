package qb

import "reflect"

type PrimaryKey any

type Entity[T any] interface {

	// must return the key for the Entity
	Key() PrimaryKey

	// provide an equals() logic for a supplied Entity type, can use
	// typof () to ignore any un-supported implementing types
	Equals(T) bool
}

// TODO replace this with reflectTypeToColumnMetadata after testing
// entityToTableMetadata creates a TableMetadata from a type implementing Entity interface
func entityToTableMetadata[T any](e Entity[T]) (*TableMetadata, error) {
	return reflectTypeToTableMetadata(reflect.TypeOf(e))
}

// Convert compares a slice of Entities "e" to the other slice of Entities "n" & returns
// 3 map[any]Entity that represent the items EntityKey->Entity that must be added, updated &
// deleted from this []Entity to make it the same as the other and returns a map of identities to
// add, update or delete
func Convert0[T Entity[T]](e, n []T) (add, upd, del map[PrimaryKey]T) {

	// we start with assumption that all existing items need to be deleted
	del = make(map[PrimaryKey]T)
	for _, e1 := range e {
		del[e1.Key()] = e1
	}

	add = make(map[PrimaryKey]T) // will hold items to be added
	upd = make(map[PrimaryKey]T) // will hold items to be updated

	for _, nv := range n {
		// if the new key already exists in existing list
		nk := nv.Key()
		if ev, ok := del[nk]; ok {
			delete(del, nk) //  we dont have to delete it, so remove from the delete list
			if !ev.Equals(nv) {
				// if the existing value doesn't match the new
				upd[nk] = nv // add this key/val to the update list
			}
		} else {
			add[nv.Key()] = nv
		}
	}
	return
}

// Convert compares a slice of Entities "e" to the other slice of Entities "n" & returns
// 3 []Entity that represent the items that must be added, updated & deleted from e []Entity
// to make it the same as the other entities list "n"
func Convert[T Entity[T]](e, n []T) (add, upd, del []T) {

	// we start with assumption that all existing items need to be deleted
	mdel := make(map[PrimaryKey]T)
	for _, e1 := range e {
		mdel[e1.Key()] = e1
	}

	add = make([]T, 0) // will hold items to be added
	upd = make([]T, 0) // will hold items to be updated

	for _, nv := range n {
		// if the new key already exists in existing list
		nk := nv.Key()
		if ev, ok := mdel[nk]; ok {
			delete(mdel, nk) //  we dont have to delete it, so remove from the delete list
			if !ev.Equals(nv) {
				// if the existing value doesn't match the new
				//upd[nk] = nv // add this key/val to the update list
				upd = append(upd, nv)
			}
		} else {
			// add =[nv.Key()] = nv
			add = append(add, nv)
		}
	}

	// build del
	del = make([]T, 0) // will hold items to be deleted
	for _, v := range mdel {
		del = append(del, v)
	}

	return
}
