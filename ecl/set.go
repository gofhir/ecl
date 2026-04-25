package ecl

import "sort"

// Set is an efficient concept ID set used by the ECL evaluator.
// Implementations may be map-backed, bitmap-backed, or lazy/streaming.
type Set interface {
	// Contains reports whether id is in the set.
	Contains(id string) bool

	// Intersect returns a new set with elements in both sets.
	Intersect(other Set) Set

	// Union returns a new set with elements in either set.
	Union(other Set) Set

	// Minus returns a new set with elements in this set but not in other.
	Minus(other Set) Set

	// Len returns the number of elements.
	Len() int

	// Slice returns the elements as a sorted string slice (for deterministic output).
	Slice() []string

	// Iter calls fn for each element. Iteration stops if fn returns false.
	Iter(fn func(id string) bool)
}

// mapSet is the default map-backed Set implementation.
type mapSet struct {
	m map[string]struct{}
}

// NewSet creates an empty Set.
func NewSet() Set {
	return newMapSet()
}

// newMapSet creates an empty *mapSet for internal use where the concrete type is needed.
func newMapSet() *mapSet {
	return &mapSet{m: make(map[string]struct{})}
}

// NewSetFromSlice creates a Set from a slice of IDs (duplicates ignored).
func NewSetFromSlice(ids []string) Set {
	m := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		m[id] = struct{}{}
	}
	return &mapSet{m: m}
}

// Contains reports whether id is in the set.
func (s *mapSet) Contains(id string) bool {
	_, ok := s.m[id]
	return ok
}

// Len returns the number of elements.
func (s *mapSet) Len() int {
	return len(s.m)
}

// Intersect returns a new set with elements in both sets.
// Iterates over the smaller set for efficiency.
func (s *mapSet) Intersect(other Set) Set {
	out := make(map[string]struct{})
	if other == nil {
		return &mapSet{m: out}
	}
	// Iterate the smaller side.
	if o, ok := other.(*mapSet); ok && len(o.m) < len(s.m) {
		for id := range o.m {
			if _, ok := s.m[id]; ok {
				out[id] = struct{}{}
			}
		}
		return &mapSet{m: out}
	}
	for id := range s.m {
		if other.Contains(id) {
			out[id] = struct{}{}
		}
	}
	return &mapSet{m: out}
}

// Union returns a new set with elements in either set.
func (s *mapSet) Union(other Set) Set {
	out := make(map[string]struct{}, len(s.m))
	for id := range s.m {
		out[id] = struct{}{}
	}
	if other != nil {
		other.Iter(func(id string) bool {
			out[id] = struct{}{}
			return true
		})
	}
	return &mapSet{m: out}
}

// Minus returns a new set with elements in this set but not in other.
func (s *mapSet) Minus(other Set) Set {
	out := make(map[string]struct{})
	for id := range s.m {
		if other == nil || !other.Contains(id) {
			out[id] = struct{}{}
		}
	}
	return &mapSet{m: out}
}

// Slice returns the elements as a sorted string slice (for deterministic output).
func (s *mapSet) Slice() []string {
	ids := make([]string, 0, len(s.m))
	for id := range s.m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Iter calls fn for each element in unspecified order. Iteration stops early
// if fn returns false.
func (s *mapSet) Iter(fn func(id string) bool) {
	for id := range s.m {
		if !fn(id) {
			return
		}
	}
}
