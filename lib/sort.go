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
	// Apply the permutation in-place using cycle-chase.
	// indices[i] means "position i should get the element from original position indices[i]".
	for i := range indices {
		if indices[i] == i {
			continue
		}
		j := i
		tmp := items[j]
		for indices[j] != i {
			items[j] = items[indices[j]]
			next := indices[j]
			indices[j] = j
			j = next
		}
		items[j] = tmp
		indices[j] = j
	}
}
