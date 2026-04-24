package ecl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSet_Empty(t *testing.T) {
	s := NewSet()
	require.NotNil(t, s)
	assert.Equal(t, 0, s.Len())
	assert.Empty(t, s.Slice())
	assert.False(t, s.Contains("123"))
}

func TestNewSetFromSlice(t *testing.T) {
	s := NewSetFromSlice([]string{"1", "2", "3", "2", "1"}) // with duplicates
	assert.Equal(t, 3, s.Len())
	assert.True(t, s.Contains("1"))
	assert.True(t, s.Contains("2"))
	assert.True(t, s.Contains("3"))
	assert.False(t, s.Contains("4"))
}

func TestSet_Contains(t *testing.T) {
	s := NewSetFromSlice([]string{"a", "b"})
	assert.True(t, s.Contains("a"))
	assert.True(t, s.Contains("b"))
	assert.False(t, s.Contains("c"))
	assert.False(t, s.Contains(""))
}

func TestSet_Len(t *testing.T) {
	assert.Equal(t, 0, NewSet().Len())
	assert.Equal(t, 1, NewSetFromSlice([]string{"x"}).Len())
	assert.Equal(t, 3, NewSetFromSlice([]string{"1", "2", "3"}).Len())
}

func TestSet_Intersect(t *testing.T) {
	a := NewSetFromSlice([]string{"1", "2", "3", "4"})
	b := NewSetFromSlice([]string{"3", "4", "5", "6"})

	got := a.Intersect(b)
	assert.Equal(t, []string{"3", "4"}, got.Slice())

	// Intersect with empty.
	empty := NewSet()
	assert.Equal(t, 0, a.Intersect(empty).Len())
	assert.Equal(t, 0, empty.Intersect(a).Len())

	// Self-intersect.
	assert.Equal(t, a.Slice(), a.Intersect(a).Slice())
}

func TestSet_Union(t *testing.T) {
	a := NewSetFromSlice([]string{"1", "2"})
	b := NewSetFromSlice([]string{"2", "3"})

	got := a.Union(b)
	assert.Equal(t, []string{"1", "2", "3"}, got.Slice())

	// Union with empty.
	empty := NewSet()
	assert.Equal(t, a.Slice(), a.Union(empty).Slice())
	assert.Equal(t, a.Slice(), empty.Union(a).Slice())
}

func TestSet_Minus(t *testing.T) {
	a := NewSetFromSlice([]string{"1", "2", "3", "4"})
	b := NewSetFromSlice([]string{"2", "4"})

	got := a.Minus(b)
	assert.Equal(t, []string{"1", "3"}, got.Slice())

	// Minus empty.
	empty := NewSet()
	assert.Equal(t, a.Slice(), a.Minus(empty).Slice())

	// Empty minus anything.
	assert.Equal(t, 0, empty.Minus(a).Len())

	// Minus self.
	assert.Equal(t, 0, a.Minus(a).Len())
}

func TestSet_Slice_Sorted(t *testing.T) {
	s := NewSetFromSlice([]string{"9", "1", "5", "3", "7"})
	assert.Equal(t, []string{"1", "3", "5", "7", "9"}, s.Slice())
}

func TestSet_Iter_All(t *testing.T) {
	s := NewSetFromSlice([]string{"a", "b", "c"})
	seen := make(map[string]bool)
	s.Iter(func(id string) bool {
		seen[id] = true
		return true
	})
	assert.Equal(t, map[string]bool{"a": true, "b": true, "c": true}, seen)
}

func TestSet_Iter_EarlyExit(t *testing.T) {
	s := NewSetFromSlice([]string{"a", "b", "c", "d", "e"})
	count := 0
	s.Iter(func(id string) bool {
		count++
		return count < 2 // stop after the second element
	})
	assert.Equal(t, 2, count)
}

func TestSet_Immutability(t *testing.T) {
	// Set operations must return new sets without mutating the originals.
	a := NewSetFromSlice([]string{"1", "2"})
	b := NewSetFromSlice([]string{"2", "3"})

	_ = a.Union(b)
	_ = a.Intersect(b)
	_ = a.Minus(b)

	assert.Equal(t, []string{"1", "2"}, a.Slice())
	assert.Equal(t, []string{"2", "3"}, b.Slice())
}
