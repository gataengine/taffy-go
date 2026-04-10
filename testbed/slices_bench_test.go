package testbed

import (
	"cmp"
	"fmt"
	"math/rand/v2"
	"slices"
	"sort"
	"testing"
	"unsafe"

	"taffy-go/lib"
)

// largeStruct mimics gridItem (~350 bytes) with the same field layout that matters for sorting.
type largeStruct struct {
	node       uint64
	source     uint16
	rowStart   uint16
	rowEnd     uint16
	colStart   uint16
	colEnd     uint16
	crossFlex  bool
	_pad       [3]byte
	filler     [320]byte // pad to realistic gridItem size
}

func init() {
	var s largeStruct
	_ = s
	// Sanity: confirm our struct is in the right ballpark.
	if sz := unsafe.Sizeof(s); sz < 300 || sz > 500 {
		panic("largeStruct size out of expected range")
	}
}

// generateItems creates n items with random sort-relevant fields.
func generateItems(n int, seed uint64) []largeStruct {
	rng := rand.New(rand.NewPCG(seed, 0))
	items := make([]largeStruct, n)
	for i := range items {
		items[i] = largeStruct{
			node:      uint64(i),
			source:    uint16(i),
			crossFlex: rng.IntN(10) < 2, // 20% cross flex
			colStart:  uint16(rng.IntN(20)),
			rowStart:  uint16(rng.IntN(20)),
		}
		span := uint16(rng.IntN(4)) + 1
		items[i].colEnd = items[i].colStart + span
		items[i].rowEnd = items[i].rowStart + span
	}
	return items
}

// cmpByValue is the comparator taking values (current approach).
func cmpByValue(a, b largeStruct) int {
	// crossFlex: non-flex before flex
	if !a.crossFlex && b.crossFlex {
		return -1
	}
	if a.crossFlex && !b.crossFlex {
		return 1
	}
	// span
	spanA := a.colEnd - a.colStart
	spanB := b.colEnd - b.colStart
	if c := cmp.Compare(spanA, spanB); c != 0 {
		return c
	}
	// start position
	return cmp.Compare(int16(a.colStart), int16(b.colStart))
}

// cmpByPtr is the comparator taking pointers (avoids copy).
func cmpByPtr(a, b *largeStruct) int {
	if !a.crossFlex && b.crossFlex {
		return -1
	}
	if a.crossFlex && !b.crossFlex {
		return 1
	}
	spanA := a.colEnd - a.colStart
	spanB := b.colEnd - b.colStart
	if c := cmp.Compare(spanA, spanB); c != 0 {
		return c
	}
	return cmp.Compare(int16(a.colStart), int16(b.colStart))
}

// --- Approach 1: slices.SortStableFunc with value comparator (current code) ---

func sortByValue(items []largeStruct) {
	slices.SortStableFunc(items, cmpByValue)
}

// --- Approach 2: sort.SliceStable with index comparator ---

func sortSliceStable(items []largeStruct) {
	sort.SliceStable(items, func(i, j int) bool {
		return cmpByPtr(&items[i], &items[j]) < 0
	})
}

// --- Approach 3: sort []int indices, then permute ---

func sortByIndices(items []largeStruct) {
	n := len(items)
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	slices.SortStableFunc(indices, func(i, j int) int {
		return cmpByPtr(&items[i], &items[j])
	})
	permute(items, indices)
}

// permute reorders items in-place according to sorted indices.
func permute(items []largeStruct, indices []int) {
	for i := range indices {
		for indices[i] != i {
			j := indices[i]
			items[i], items[j] = items[j], items[i]
			indices[i], indices[j] = indices[j], indices[i]
		}
	}
}

// --- Approach 4: lib.SortStableByIndex (not in-place, new slice) ---

func sortByLib(items []largeStruct) {
	lib.SortStableByIndex(items, cmpByPtr)
}

// --- Benchmarks ---

func BenchmarkSort(b *testing.B) {
	for _, n := range []int{100, 1000, 6561} {
		base := generateItems(n, 42)

		b.Run(fmt.Sprintf("N=%d/slicesSortStableFunc_value", n), func(b *testing.B) {
			items := make([]largeStruct, n)
			for b.Loop() {
				copy(items, base)
				sortByValue(items)
			}
		})

		b.Run(fmt.Sprintf("N=%d/sortSliceStable_index", n), func(b *testing.B) {
			items := make([]largeStruct, n)
			for b.Loop() {
				copy(items, base)
				sortSliceStable(items)
			}
		})

		b.Run(fmt.Sprintf("N=%d/slicesSortStableFunc_indices_permute_inplace", n), func(b *testing.B) {
			items := make([]largeStruct, n)
			for b.Loop() {
				copy(items, base)
				sortByIndices(items)
			}
		})

		b.Run(fmt.Sprintf("N=%d/lib_SortStableByIndex", n), func(b *testing.B) {
			items := make([]largeStruct, n)
			for b.Loop() {
				copy(items, base)
				sortByLib(items)
			}
		})
	}
}
