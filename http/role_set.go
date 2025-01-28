package http

import (
	"encoding/json"
	"slices"
)

// a set of roles allocated to a policy items
type Set map[string]struct{}

type GroupSet Set // GroupSet is an alias for set

// Contains returns a true value if the set contains any of the roles supplied in the list
func (s Set) Contains(roles ...string) bool {
	// if this role has Everyone "*" wildcard as a member defined in it, then we return 'true'
	if _, ok := s[Everyone]; ok {
		return ok
	}

	for _, r := range roles {
		if _, ok := s[r]; ok {
			return true
		}
	}
	return false
}

// ContainsSet returns a tru value if the set contain any of the roles supplied in the Set
func (s Set) ContainsSet(other Set) bool {
	// if this role has Everyone "*" wildcard as a member defined in it, then we return 'true'
	if _, ok := s[Everyone]; ok {
		return ok
	}

	for r, _ := range other {
		if _, ok := s[r]; ok {
			return true
		}
	}
	return false
}

// ToStringSlice converts the Set to a []string
func (s Set) ToStringSlice() []string {
	var slice []string
	for k, _ := range s {
		slice = append(slice, k)
	}
	slices.Sort(slice)
	return slice
}

// Insert the supplied rules in this set
func (s Set) Insert(roles ...string) {
	for _, role := range roles {
		s[role] = struct{}{}
	}
}

// UnmarshallJSON impl custom unmarshall logic for Set
func (r *Set) UnmarshalJSON(data []byte) error {

	tmp := make(map[string]struct{})

	var s []string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	for _, s1 := range s {
		tmp[s1] = struct{}{}
	}

	*r = Set(tmp)
	return nil
}

// MarshallJSON impl custom marshall logic for Set
func (r Set) MarshalJSON() ([]byte, error) {

	// convert this to string
	var s []string
	for k := range r {
		s = append(s, k)
	}

	return json.Marshal(s)
}

// RoleSetFrom returns a Subjects
func RoleSetFrom(roles ...string) Set {
	m := Set{}

	// TODO check if this makes sense, if [] is supplied, should be add Everyone to the role set ??
	if len(roles) == 0 {
		roles = []string{Everyone}
	}

	for _, s := range roles {
		m[s] = struct{}{}
	}
	return m
}
