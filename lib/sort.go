package lib

import "slices"

// SortStableByIndex performs a stable in-place sort using a pointer-based comparator.
// It sorts an index array instead of moving large structs during comparison,
// then permutes items in-place. This is significantly faster than
// slices.SortStableFunc for large element types (e.g. 300+ bytes).
func SortStableByIndex[S ~[]E, E any](items S, cmp func(a, b *E) int) {
	n := len(items)
	if n <= 1 {
		return
	}
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	slices.SortStableFunc(indices, func(i, j int) int {
		return cmp(&items[i], &items[j])
	})
	for i := range indices {
		for indices[i] != i {
			j := indices[i]
			items[i], items[j] = items[j], items[i]
			indices[i], indices[j] = indices[j], indices[i]
		}
	}
}
