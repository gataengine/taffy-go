# Phase 3: Grid Allocation Reduction — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce grid layout allocations by ~53% through three independent changes: reflection-free sorting, scratch slice pooling, and cache heap escape elimination.

**Architecture:** Three independent changes to the grid compute pipeline and cache layer. Each change is self-contained and can be verified independently. The sort replacement and cache fix are mechanical; the scratch pool requires threading pooled slices through `ComputeGridLayout`.

**Tech Stack:** Go 1.25, `slices.SortStableFunc`, `cmp.Compare`, `sync.Pool`

**Spec:** `docs/superpowers/specs/2026-04-10-grid-allocation-reduction-design.md`

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `compute/grid/grid.go` | Modify | Replace 2 `sort.SliceStable` calls, acquire/release scratch pool, use scratch slices |
| `compute/grid/track_sizing.go` | Modify | Replace 2 `sort.SliceStable` calls |
| `compute/grid/scratch.go` | Create | `gridScratch` struct, `reset()` method, `gridScratchPool` var |
| `tree/cache.go` | Modify | `Get` returns `(LayoutOutput, bool)` instead of `*LayoutOutput` |
| `tree/traits.go` | Modify | `CacheGet` interface signature |
| `taffy_view.go` | Modify | `CacheGet` impl + 2 call sites |
| `compute/compute.go` | Modify | `CacheGet` call site |

---

### Task 1: Replace `sort.SliceStable` in `track_sizing.go`

**Files:**
- Modify: `compute/grid/track_sizing.go:1-13` (imports), `:318-320`, `:391-395`

- [ ] **Step 1: Replace sort in `resolveItemBaselines` (line 318)**

In `compute/grid/track_sizing.go`, replace the import block:

```go
// Old imports (lines 5-13):
import (
	"taffy-go/lib"
	"math"
	"sort"

	"taffy-go/geometry"
	"taffy-go/style"
	"taffy-go/tree"
)

// New imports:
import (
	"cmp"
	"math"
	"slices"

	"taffy-go/geometry"
	"taffy-go/lib"
	"taffy-go/style"
	"taffy-go/tree"
)
```

Then replace lines 318-320:

```go
// Old:
sort.SliceStable(items, func(i, j int) bool {
	return int16(items[i].placement(otherAxis).Start) < int16(items[j].placement(otherAxis).Start)
})

// New:
slices.SortStableFunc(items, func(a, b gridItem) int {
	return cmp.Compare(int16(a.placement(otherAxis).Start), int16(b.placement(otherAxis).Start))
})
```

- [ ] **Step 2: Replace sort in `resolveIntrinsicTrackSizes` (line 393)**

Replace lines 392-395:

```go
// Old:
cmp := cmpByCrossFlexThenSpanThenStart(axis)
sort.SliceStable(items, func(i, j int) bool {
	return cmp(&items[i], &items[j]) < 0
})

// New:
cmpFn := cmpByCrossFlexThenSpanThenStart(axis)
slices.SortStableFunc(items, func(a, b gridItem) int {
	return cmpFn(&a, &b)
})
```

Note: the variable is renamed from `cmp` to `cmpFn` to avoid shadowing the `cmp` package import.

- [ ] **Step 3: Run tests**

Run: `go test taffy-go -count=1 -timeout=120s`

Expected: PASS (4441 specs)

- [ ] **Step 4: Commit**

```bash
git add compute/grid/track_sizing.go
git commit -m "perf: replace sort.SliceStable with slices.SortStableFunc in track_sizing"
```

---

### Task 2: Replace `sort.SliceStable` in `grid.go`

**Files:**
- Modify: `compute/grid/grid.go:1-12` (imports), `:495-497`, `:663-665`

- [ ] **Step 1: Replace imports**

In `compute/grid/grid.go`, replace the import block:

```go
// Old imports (lines 5-12):
import (
	"taffy-go/lib"
	"sort"

	"taffy-go/geometry"
	"taffy-go/style"
	"taffy-go/tree"
)

// New imports:
import (
	"cmp"
	"slices"

	"taffy-go/geometry"
	"taffy-go/lib"
	"taffy-go/style"
	"taffy-go/tree"
)
```

- [ ] **Step 2: Replace sort at line 495 (restore source order)**

```go
// Old:
sort.SliceStable(items, func(i, j int) bool {
	return items[i].sourceOrder < items[j].sourceOrder
})

// New:
slices.SortStableFunc(items, func(a, b gridItem) int {
	return cmp.Compare(a.sourceOrder, b.sourceOrder)
})
```

- [ ] **Step 3: Replace sort at line 663 (baseline row sort)**

```go
// Old:
sort.SliceStable(items, func(i, j int) bool {
	return items[i].rowIndexes.Start < items[j].rowIndexes.Start
})

// New:
slices.SortStableFunc(items, func(a, b gridItem) int {
	return cmp.Compare(a.rowIndexes.Start, b.rowIndexes.Start)
})
```

- [ ] **Step 4: Run tests**

Run: `go test taffy-go -count=1 -timeout=120s`

Expected: PASS (4441 specs)

- [ ] **Step 5: Run grid benchmarks to verify sort allocation reduction**

Run: `go test -bench='BenchmarkGrid' -benchmem -count=1 -timeout=120s taffy-go 2>&1 | grep Bench`

Expected: B/op and allocs/op should decrease compared to pre-change baseline. The `reflectlite.unsafe_New` allocation (~77 MB in profiles) should be eliminated.

- [ ] **Step 6: Commit**

```bash
git add compute/grid/grid.go
git commit -m "perf: replace sort.SliceStable with slices.SortStableFunc in grid"
```

---

### Task 3: Create `gridScratch` struct and pool

**Files:**
- Create: `compute/grid/scratch.go`

- [ ] **Step 1: Create scratch.go**

Create `compute/grid/scratch.go` with the following content:

```go
package grid

import (
	"sync"

	"taffy-go/style"
)

// gridScratch holds reusable slice buffers for ComputeGridLayout.
// Pooled via gridScratchPool to avoid repeated heap allocations.
type gridScratch struct {
	items          []gridItem
	inFlowChildren []gridPlacementChild
	childStyles    []style.GridItemStyle
	columns        []gridTrack
	rows           []gridTrack
}

// reset clears all slices to zero length while retaining backing arrays.
func (s *gridScratch) reset() {
	s.items = s.items[:0]
	s.inFlowChildren = s.inFlowChildren[:0]
	s.childStyles = s.childStyles[:0]
	s.columns = s.columns[:0]
	s.rows = s.rows[:0]
}

// gridScratchPool is a pool of reusable gridScratch instances.
var gridScratchPool = sync.Pool{
	New: func() any { return new(gridScratch) },
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build taffy-go/compute/grid`

Expected: success, no errors

- [ ] **Step 3: Commit**

```bash
git add compute/grid/scratch.go
git commit -m "perf: add gridScratch pool for grid slice reuse"
```

---

### Task 4: Wire scratch pool into `ComputeGridLayout`

**Files:**
- Modify: `compute/grid/grid.go:174-182` (slice creation), `:207-209` (track vars), `:697-703` (return path)

This task replaces local `make()` calls with scratch-pooled slices.

- [ ] **Step 1: Add pool acquire/release at top of `ComputeGridLayout`**

At the very start of `ComputeGridLayout` (after the `containerStyle := ...` block, before the first computation — around line 23, right after `calc := func(...)`), insert:

```go
// Acquire scratch buffers
scratch := gridScratchPool.Get().(*gridScratch)
scratch.reset()
defer gridScratchPool.Put(scratch)
```

The `defer Put` is safe immediately because `reset()` already zeroed all slice lengths — if an early return fires (e.g. the `RunModeComputeSize` short-circuit at line 113), the scratch goes back to the pool with its backing arrays intact for future reuse.

To save grown slices back to scratch at the end of the function (before the final return at line 703), add a helper call. This is done in Step 5 below.

- [ ] **Step 2: Replace `childStyleSlice` allocation (line 175)**

```go
// Old (lines 175-178):
childStyleSlice := make([]style.GridItemStyle, 0, t.ChildCount(node))
for _, childNode := range t.ChildIds(node) {
	childStyleSlice = append(childStyleSlice, t.GetGridChildStyle(childNode))
}

// New:
childStyleSlice := scratch.childStyles
for _, childNode := range t.ChildIds(node) {
	childStyleSlice = append(childStyleSlice, t.GetGridChildStyle(childNode))
}
scratch.childStyles = childStyleSlice
```

- [ ] **Step 3: Replace `items` and `inFlowChildren` allocations (lines 182-185)**

```go
// Old (lines 182-185):
items := make([]gridItem, 0, t.ChildCount(node))
cellMatrix := newCellOccupancyMatrix(estColCounts, estRowCounts)

inFlowChildren := make([]gridPlacementChild, 0, t.ChildCount(node))

// New:
items := scratch.items
cellMatrix := newCellOccupancyMatrix(estColCounts, estRowCounts)

inFlowChildren := scratch.inFlowChildren
```

After `placeGridItems` returns and `inFlowChildren` is no longer needed, save back `items` (which was grown by `placeGridItems`). No change needed — `items` is already the live variable used through the rest of the function.

- [ ] **Step 4: Replace `columns` and `rows` track declarations (lines 208-209)**

```go
// Old (lines 208-209):
var columns []gridTrack
var rows []gridTrack

// New:
columns := scratch.columns
rows := scratch.rows
```

`initializeGridTracks` already takes `*[]gridTrack`, resets length to 0 (line 229), and only calls `make()` when capacity is insufficient (line 232). No changes needed to `initializeGridTracks` — pooled slices with sufficient capacity will be reused automatically.

- [ ] **Step 5: Save grown slices back to scratch before returns**

Before the final return at line 703, add:

```go
scratch.items = items
scratch.columns = columns
scratch.rows = rows
```

There are 3 other return paths in `ComputeGridLayout`:
- Line 113: early return when both dimensions are known — slices haven't been created yet, scratch is already reset, no save needed.
- Line 332: `RunModeComputeSize` return — `items`, `columns`, `rows` are live at this point. Add the same 3 lines before this return.
- Line 659: empty items return — same, add the 3 lines before this return.

- [ ] **Step 6: Run tests**

Run: `go test taffy-go -count=1 -timeout=120s`

Expected: PASS (4441 specs)

- [ ] **Step 7: Run grid benchmarks to verify scratch pool allocation reduction**

Run: `go test -bench='BenchmarkGrid' -benchmem -count=1 -timeout=120s taffy-go 2>&1 | grep Bench`

Expected: Significant reduction in B/op and allocs/op for all grid benchmarks compared to post-Task 2 baseline.

- [ ] **Step 8: Commit**

```bash
git add compute/grid/grid.go
git commit -m "perf: pool grid scratch slices via sync.Pool"
```

---

### Task 5: Change `Cache.Get` to return `(LayoutOutput, bool)`

**Files:**
- Modify: `tree/cache.go:108-163`
- Modify: `tree/traits.go:49`
- Modify: `taffy_view.go:68-70`, `:112-113`, `:196-198`
- Modify: `compute/compute.go:31-33`

- [ ] **Step 1: Update `Cache.Get` signature and body**

In `tree/cache.go`, replace lines 106-163:

```go
// Get tries to retrieve a cached result from the cache.
// Returns the cached result and true if found, or a zero value and false otherwise.
func (c *Cache) Get(input *LayoutInput) (LayoutOutput, bool) {
	knownDimensions := input.KnownDimensions
	availableSpace := input.AvailableSpace

	switch input.RunMode {
	case RunModePerformLayout:
		if !c.hasFinalLayout {
			return LayoutOutput{}, false
		}
		entry := &c.finalLayoutEntry
		cachedSize := entry.content.Size

		widthMatch := knownDimensions.Width == entry.knownDimensions.Width ||
			(knownDimensions.Width.IsSome && knownDimensions.Width.Value == cachedSize.Width)
		heightMatch := knownDimensions.Height == entry.knownDimensions.Height ||
			(knownDimensions.Height.IsSome && knownDimensions.Height.Value == cachedSize.Height)
		widthSpaceMatch := knownDimensions.Width.IsSome ||
			entry.availableSpace.Width.IsRoughlyEqual(availableSpace.Width)
		heightSpaceMatch := knownDimensions.Height.IsSome ||
			entry.availableSpace.Height.IsRoughlyEqual(availableSpace.Height)

		if widthMatch && heightMatch && widthSpaceMatch && heightSpaceMatch {
			return entry.content, true
		}
		return LayoutOutput{}, false

	case RunModeComputeSize:
		for i, valid := range c.measureValid {
			if !valid {
				continue
			}
			entry := &c.measureEntries[i]
			cachedSize := entry.content

			widthMatch := knownDimensions.Width == entry.knownDimensions.Width ||
				(knownDimensions.Width.IsSome && knownDimensions.Width.Value == cachedSize.Width)
			heightMatch := knownDimensions.Height == entry.knownDimensions.Height ||
				(knownDimensions.Height.IsSome && knownDimensions.Height.Value == cachedSize.Height)
			widthSpaceMatch := knownDimensions.Width.IsSome ||
				entry.availableSpace.Width.IsRoughlyEqual(availableSpace.Width)
			heightSpaceMatch := knownDimensions.Height.IsSome ||
				entry.availableSpace.Height.IsRoughlyEqual(availableSpace.Height)

			if widthMatch && heightMatch && widthSpaceMatch && heightSpaceMatch {
				return LayoutOutputFromOuterSize(cachedSize), true
			}
		}
		return LayoutOutput{}, false

	case RunModePerformHiddenLayout:
		return LayoutOutput{}, false
	}

	return LayoutOutput{}, false
}
```

- [ ] **Step 2: Update `CacheTree` interface in `tree/traits.go`**

Replace line 49:

```go
// Old:
CacheGet(nodeId NodeId, input *LayoutInput) *LayoutOutput

// New:
CacheGet(nodeId NodeId, input *LayoutInput) (LayoutOutput, bool)
```

- [ ] **Step 3: Update `taffyView.CacheGet` in `taffy_view.go`**

Replace lines 112-113:

```go
// Old:
func (v *taffyView) CacheGet(nodeId tree.NodeId, input *tree.LayoutInput) *tree.LayoutOutput {
	return v.taffy.nodes[nodeId].cache.Get(input)
}

// New:
func (v *taffyView) CacheGet(nodeId tree.NodeId, input *tree.LayoutInput) (tree.LayoutOutput, bool) {
	return v.taffy.nodes[nodeId].cache.Get(input)
}
```

- [ ] **Step 4: Update call site in `taffyView.ComputeChildLayout` (line 68)**

Replace lines 68-70:

```go
// Old:
if cached := v.CacheGet(nodeId, &inputs); cached != nil {
	return *cached
}

// New:
if cached, ok := v.CacheGet(nodeId, &inputs); ok {
	return cached
}
```

- [ ] **Step 5: Update call site in `taffyView.ComputeBlockChildLayout` (line 196)**

Replace lines 196-198:

```go
// Old:
if cached := v.CacheGet(nodeId, &inputs); cached != nil {
	return *cached
}

// New:
if cached, ok := v.CacheGet(nodeId, &inputs); ok {
	return cached
}
```

- [ ] **Step 6: Update call site in `compute.ComputeCachedLayout` (line 31)**

Replace lines 31-33 in `compute/compute.go`:

```go
// Old:
if cached := t.CacheGet(node, &inputs); cached != nil {
	return *cached
}

// New:
if cached, ok := t.CacheGet(node, &inputs); ok {
	return cached
}
```

- [ ] **Step 7: Run tests**

Run: `go test taffy-go -count=1 -timeout=120s`

Expected: PASS (4441 specs)

- [ ] **Step 8: Run all benchmarks to verify no regression**

Run: `go test -bench='Benchmark' -benchmem -count=1 -timeout=120s taffy-go 2>&1 | grep Bench`

Expected: B/op reduction in both grid and flexbox benchmarks (cache path is shared). allocs/op should decrease.

- [ ] **Step 9: Commit**

```bash
git add tree/cache.go tree/traits.go taffy_view.go compute/compute.go
git commit -m "perf: eliminate Cache.Get heap escape via value return"
```

---

### Task 6: Final verification and benchmark recording

**Files:**
- Create: `bench_after_task5.txt` (benchmark record)

- [ ] **Step 1: Run full test suite**

Run: `go test taffy-go -count=1 -timeout=120s`

Expected: PASS (4441 specs)

- [ ] **Step 2: Run full benchmarks and save results**

Run: `go test -bench='Benchmark' -benchmem -count=3 -timeout=300s taffy-go 2>&1 | tee bench_after_task5.txt`

Expected: Significant reduction across grid benchmarks. Compare against the baseline numbers:
- `BenchmarkGrid/Wide/31x31`: was 876,719 B/op, 4,966 allocs/op
- `BenchmarkGrid/Deep/2x2_levels5`: was 8,209,211 B/op, 47,193 allocs/op
- `BenchmarkGrid/Deep/3x3_levels4`: was 39,672,464 B/op, 185,770 allocs/op
- `BenchmarkGrid/Deep/2x2_levels7`: was 139,079,200 B/op, 801,999 allocs/op

- [ ] **Step 3: Commit benchmark results**

```bash
git add bench_after_task5.txt
git commit -m "bench: record Phase 3 grid allocation reduction results"
```
