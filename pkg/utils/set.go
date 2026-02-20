package utils

import (
	"maps"
	"slices"
)

// Set is a generic collection of unique elements.
type Set[T comparable] map[T]struct{}

// NewSet creates a new empty Set.
func NewSet[T comparable]() Set[T] {
	return make(Set[T])
}

// Add adds an element to the set.
func (s Set[T]) Add(item T) {
	s[item] = struct{}{}
}

// AddAll adds all elements from a slice to the set.
func (s Set[T]) AddAll(items []T) {
	for _, item := range items {
		s[item] = struct{}{}
	}
}

// Has checks if an element exists in the set.
func (s Set[T]) Has(item T) bool {
	_, ok := s[item]
	return ok
}

// Remove removes an element from the set.
func (s Set[T]) Remove(item T) {
	delete(s, item)
}

// ToSlice returns all elements in the set as a slice.
func (s Set[T]) ToSlice() []T {
	return slices.Collect(maps.Keys(s))
}

// Size returns the number of elements in the set.
func (s Set[T]) Size() int {
	return len(s)
}
