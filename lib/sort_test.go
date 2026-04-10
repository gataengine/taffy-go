package lib_test

import (
	"cmp"
	"slices"
	"testing"

	"taffy-go/lib"
)

func TestSortStableByIndex_Correctness(t *testing.T) {
	type item struct {
		key     int
		origIdx int
	}

	items := []item{{3, 0}, {1, 1}, {2, 2}, {1, 3}}
	lib.SortStableByIndex(items, func(a, b *item) int {
		return cmp.Compare(a.key, b.key)
	})

	// Expected: sorted by key, stable (original order preserved for equal keys)
	want := []item{{1, 1}, {1, 3}, {2, 2}, {3, 0}}
	for i, got := range items {
		if got != want[i] {
			t.Fatalf("items[%d] = %v, want %v\nfull result: %v", i, got, want[i], items)
		}
	}
}

func TestSortStableByIndex_AlreadySorted(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	lib.SortStableByIndex(items, func(a, b *int) int {
		return cmp.Compare(*a, *b)
	})
	want := []int{1, 2, 3, 4, 5}
	if !slices.Equal(items, want) {
		t.Fatalf("got %v, want %v", items, want)
	}
}

func TestSortStableByIndex_Reversed(t *testing.T) {
	items := []int{5, 4, 3, 2, 1}
	lib.SortStableByIndex(items, func(a, b *int) int {
		return cmp.Compare(*a, *b)
	})
	want := []int{1, 2, 3, 4, 5}
	if !slices.Equal(items, want) {
		t.Fatalf("got %v, want %v", items, want)
	}
}

func TestSortStableByIndex_Empty(t *testing.T) {
	var items []int
	lib.SortStableByIndex(items, func(a, b *int) int {
		return cmp.Compare(*a, *b)
	})
	if len(items) != 0 {
		t.Fatalf("expected empty, got %v", items)
	}
}

func TestSortStableByIndex_Single(t *testing.T) {
	items := []int{42}
	lib.SortStableByIndex(items, func(a, b *int) int {
		return cmp.Compare(*a, *b)
	})
	if items[0] != 42 {
		t.Fatalf("got %v, want [42]", items)
	}
}
