// Package ds provides general-purpose data structures used across gotham.
package ds

import (
	"encoding/json"
	"slices"
)

// Set is a string-keyed set.
type Set map[string]struct{}

// GroupSet is an alias for Set, kept for readability at call sites that
// represent group membership.
type GroupSet = Set

// Contains returns true if the set contains any of the supplied items.
func (s Set) Contains(items ...string) bool {
	for _, item := range items {
		if _, ok := s[item]; ok {
			return true
		}
	}
	return false
}

// ContainsSet returns true if the set has any element in common with other.
func (s Set) ContainsSet(other Set) bool {
	for item := range other {
		if _, ok := s[item]; ok {
			return true
		}
	}
	return false
}

// ToStringSlice returns the set's elements as a sorted slice.
func (s Set) ToStringSlice() []string {
	var slice []string
	for k := range s {
		slice = append(slice, k)
	}
	slices.Sort(slice)
	return slice
}

// Insert adds the supplied items to the set.
func (s Set) Insert(items ...string) {
	for _, item := range items {
		s[item] = struct{}{}
	}
}

// From builds a Set from the supplied items.
func From(items ...string) Set {
	s := Set{}
	for _, item := range items {
		s[item] = struct{}{}
	}
	return s
}

// UnmarshalJSON decodes a JSON array of strings into the set.
func (s *Set) UnmarshalJSON(data []byte) error {
	var items []string
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}

	tmp := make(map[string]struct{}, len(items))
	for _, item := range items {
		tmp[item] = struct{}{}
	}
	*s = Set(tmp)
	return nil
}

// MarshalJSON encodes the set as a JSON array of strings.
func (s Set) MarshalJSON() ([]byte, error) {
	items := make([]string, 0, len(s))
	for k := range s {
		items = append(items, k)
	}
	slices.Sort(items)
	return json.Marshal(items)
}
