# Performance Phase 5: Allocation Micro-Optimizations

> **Status (audit 2026-10-11): Implemented.** Checkboxes below were never ticked.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate 87% of allocations in deep flexbox benchmarks by caching a closure and pooling two slice types.

**Architecture:** Three independent fixes targeting the top allocation sources profiled in FlexboxSuperDeep/100. Fix 1 caches the measureFunc closure on taffyView. Fixes 2-3 add sync.Pool for flexItem and flexLine slices in the flexbox package.

**Tech Stack:** Go 1.25, sync.Pool, standard benchmarking tools.

---

### Task 1: Baseline Benchmark

Capture before-numbers so we can measure improvement.

**Files:**
- None modified

- [ ] **Step 1: Run baseline benchmarks**

Run:
```bash
go test -run='^$' -bench='BenchmarkFlexbox/(SuperDeep/100$|DeepAuto/4000$|HugeNested/10000$)' -benchmem -count=5 2>&1 | tee bench_before_phase5.txt
```

Expected: Results showing ~7044 allocs/op for SuperDeep/100, ~20463 allocs/op for DeepAuto/4000.

- [ ] **Step 2: Commit baseline**

```bash
git add bench_before_phase5.txt
git commit -m "bench: record Phase 5 baseline"
```

---

### Task 2: Cache measureFunc Closure on taffyView

The `measureFunc(nodeId)` method creates a new heap-allocated closure on every leaf-node dispatch — 52.5% of all SuperDeep allocations. The closure captures `v` (constant) and `nodeId` (varies). Fix: create the closure once and store it; set `nodeId` via a field before each call.

**Files:**
- Modify: `taffy_view.go:18-21` (add fields to taffyView struct)
- Modify: `taffy_view.go:52-63` (replace measureFunc method)
- Modify: `taffy_view.go:87,93,99,102` (update 4 call sites in ComputeChildLayout)
- Modify: `taffy_tree.go:447` (initialize cached closure)

- [ ] **Step 1: Add fields to taffyView and initialize cached closure**

In `taffy_view.go`, change the struct definition:

```go
type taffyView struct {
	taffy              *TaffyTree
	measureFunction    MeasureFunction
	measureNodeId      tree.NodeId
	cachedMeasureFn    func(geometry.Size[lib.Option[float32]], geometry.Size[style.AvailableSpace]) geometry.Size[float32]
}
```

Add an initializer method right after the struct:

```go
// initMeasureFunc creates the reusable measure closure.
// Must be called once after taffyView is created.
func (v *taffyView) initMeasureFunc() {
	v.cachedMeasureFn = func(knownDimensions geometry.Size[lib.Option[float32]], availableSpace geometry.Size[style.AvailableSpace]) geometry.Size[float32] {
		nodeId := v.measureNodeId
		ctx := *v.taffy.nodeContexts.Get(nodeId)
		if v.measureFunction != nil {
			return v.measureFunction(knownDimensions, availableSpace, nodeId, &ctx, &v.taffy.nodes.Get(nodeId).style)
		}
		if ctx != nil {
			return ctx(knownDimensions, geometry.SizeOptionNone, availableSpace, tree.SizingModeInherentSize)
		}
		return geometry.SizeZeroF32
	}
}
```

- [ ] **Step 2: Delete the old measureFunc method**

Remove the `measureFunc` method (lines 51-63 of `taffy_view.go`):

```go
// DELETE this entire method:
// func (v *taffyView) measureFunc(nodeId tree.NodeId) func(...) ... { ... }
```

- [ ] **Step 3: Update the 4 call sites in ComputeChildLayout**

In `taffy_view.go` `ComputeChildLayout`, replace each occurrence of `v.measureFunc(nodeId)` with `v.setMeasureNode(nodeId)`. Add this helper:

```go
// setMeasureNode sets the node for the cached measure closure and returns it.
func (v *taffyView) setMeasureNode(nodeId tree.NodeId) func(geometry.Size[lib.Option[float32]], geometry.Size[style.AvailableSpace]) geometry.Size[float32] {
	v.measureNodeId = nodeId
	return v.cachedMeasureFn
}
```

So lines 87, 93, 99, 102 change from:
```go
result = compute.ComputeLeafLayout(inputs, v.GetCoreContainerStyle(nodeId), v.ResolveCalcValue, v.measureFunc(nodeId))
```
to:
```go
result = compute.ComputeLeafLayout(inputs, v.GetCoreContainerStyle(nodeId), v.ResolveCalcValue, v.setMeasureNode(nodeId))
```

- [ ] **Step 4: Initialize the closure at taffyView creation sites**

In `taffy_tree.go:447`:
```go
view := &taffyView{taffy: t, measureFunction: measureFn}
view.initMeasureFunc()
```

In `taffy_tree.go:457`:
```go
view := &taffyView{taffy: t}
view.initMeasureFunc()
```

- [ ] **Step 5: Run tests**

Run: `go test ./... 2>&1 | tail -20`

Expected: All tests pass. No behavior change — the closure logic is identical, just created once instead of per-call.

- [ ] **Step 6: Verify allocation reduction**

Run:
```bash
go test -run='^$' -bench='BenchmarkFlexbox/SuperDeep/100$' -benchmem -count=3
```

Expected: allocs/op drops significantly (measureFunc was 52.5% of allocs).

- [ ] **Step 7: Commit**

```bash
git add taffy_view.go taffy_tree.go
git commit -m "perf: cache measureFunc closure on taffyView (eliminates per-call allocation)"
```

---

### Task 3: Pool flexItem Slices

`generateAnonymousFlexItems` calls `make([]flexItem, 0, cap)` per node — 17.3% of SuperDeep allocs. Pool these slices via `sync.Pool`.

**Files:**
- Modify: `compute/flexbox/flex_item.go:56-63` (get from pool instead of make)
- Modify: `compute/flexbox/flexbox.go:554-691` (return to pool at end of computePreliminary)
- Create: `compute/flexbox/pools.go` (pool definitions)

- [ ] **Step 1: Create pools.go with pool definitions**

Create `compute/flexbox/pools.go`:

```go
package flexbox

import "sync"

var flexItemsPool = sync.Pool{
	New: func() any {
		s := make([]flexItem, 0, 8)
		return &s
	},
}

func getFlexItems(capacity int) []flexItem {
	sp := flexItemsPool.Get().(*[]flexItem)
	s := *sp
	if cap(s) < capacity {
		s = make([]flexItem, 0, capacity)
	}
	return s[:0]
}

func putFlexItems(s []flexItem) {
	s = s[:0]
	flexItemsPool.Put(&s)
}
```

- [ ] **Step 2: Update generateAnonymousFlexItems to use pool**

In `flex_item.go:63`, change:
```go
items := make([]flexItem, 0, len(childIds))
```
to:
```go
items := getFlexItems(len(childIds))
```

- [ ] **Step 3: Return flexItems to pool at end of computePreliminary**

In `flexbox.go`, the function `computePreliminary` has two return paths:

1. Early return at line 634-636 (RunModeComputeSize):
```go
if runMode == tree.RunModeComputeSize {
	putFlexItems(flexItems)
	return tree.LayoutOutputFromOuterSize(constants.containerSize)
}
```

2. Final return at line 686-690:
```go
result := tree.LayoutOutputFromSizesAndBaselines(
	constants.containerSize,
	geometry.SizeF32Max(inflowContentSize, absoluteContentSize),
	geometry.Point[lib.Option[float32]]{X: lib.None[float32](), Y: firstVerticalBaseline},
)
putFlexItems(flexItems)
return result
```

Note: `flexLines[i].items` are sub-slices of `flexItems`, so `flexItems` must outlive `flexLines`. Both return paths are after all flexLine usage is complete.

- [ ] **Step 4: Run tests**

Run: `go test ./... 2>&1 | tail -20`

Expected: All tests pass.

- [ ] **Step 5: Commit**

```bash
git add compute/flexbox/pools.go compute/flexbox/flex_item.go compute/flexbox/flexbox.go
git commit -m "perf: pool flexItem slices via sync.Pool (17% of SuperDeep allocs)"
```

---

### Task 4: Pool flexLine Slices

`collectFlexLines` calls `make([]flexLine, ...)` per node — 18.0% of SuperDeep allocs. Same pooling pattern.

**Files:**
- Modify: `compute/flexbox/pools.go` (add flexLine pool)
- Modify: `compute/flexbox/flex_line.go:20-89` (get from pool)
- Modify: `compute/flexbox/flexbox.go` (return to pool at same two return sites)

- [ ] **Step 1: Add flexLine pool to pools.go**

Append to `compute/flexbox/pools.go`:

```go
var flexLinesPool = sync.Pool{
	New: func() any {
		s := make([]flexLine, 0, 1)
		return &s
	},
}

func getFlexLines(capacity int) []flexLine {
	sp := flexLinesPool.Get().(*[]flexLine)
	s := *sp
	if cap(s) < capacity {
		s = make([]flexLine, 0, capacity)
	}
	return s[:0]
}

func putFlexLines(s []flexLine) {
	s = s[:0]
	flexLinesPool.Put(&s)
}
```

- [ ] **Step 2: Update collectFlexLines to use pool**

In `flex_line.go`, replace all 4 `make([]flexLine, ...)` calls:

Line 26: `lines := make([]flexLine, 0, 1)` → `lines := getFlexLines(1)`

Line 44: `lines := make([]flexLine, 0, 1)` → `lines := getFlexLines(1)`

Line 50: `lines := make([]flexLine, 0, len(flexItems))` → `lines := getFlexLines(len(flexItems))`

Line 61: `lines := make([]flexLine, 0, 1)` → `lines := getFlexLines(1)`

- [ ] **Step 3: Return flexLines to pool in computePreliminary**

At both return paths in `computePreliminary` (same locations as Task 3), add `putFlexLines(flexLines)` before each return. The early return becomes:

```go
if runMode == tree.RunModeComputeSize {
	putFlexLines(flexLines)
	putFlexItems(flexItems)
	return tree.LayoutOutputFromOuterSize(constants.containerSize)
}
```

And the final return becomes:

```go
result := tree.LayoutOutputFromSizesAndBaselines(
	constants.containerSize,
	geometry.SizeF32Max(inflowContentSize, absoluteContentSize),
	geometry.Point[lib.Option[float32]]{X: lib.None[float32](), Y: firstVerticalBaseline},
)
putFlexLines(flexLines)
putFlexItems(flexItems)
return result
```

Note: `putFlexLines` before `putFlexItems` because flexLine.items are sub-slices of flexItems.

- [ ] **Step 4: Run tests**

Run: `go test ./... 2>&1 | tail -20`

Expected: All tests pass.

- [ ] **Step 5: Commit**

```bash
git add compute/flexbox/pools.go compute/flexbox/flex_line.go compute/flexbox/flexbox.go
git commit -m "perf: pool flexLine slices via sync.Pool (18% of SuperDeep allocs)"
```

---

### Task 5: Final Benchmarks and Verification

**Files:**
- None modified

- [ ] **Step 1: Run final benchmarks**

Run:
```bash
go test -run='^$' -bench='BenchmarkFlexbox/(SuperDeep/100$|DeepAuto/4000$|HugeNested/10000$)' -benchmem -count=5 2>&1 | tee bench_after_phase5.txt
```

- [ ] **Step 2: Compare with baseline**

Run:
```bash
benchstat bench_before_phase5.txt bench_after_phase5.txt
```

Expected: SuperDeep/100 allocs/op drops significantly. Time improvement ~10-17% for SuperDeep. DeepAuto may show modest improvement.

- [ ] **Step 3: Run alloc profile to confirm top sources are eliminated**

Run:
```bash
go test -run='^$' -bench='BenchmarkFlexbox/SuperDeep/100' -memprofile=mem_phase5.prof -benchtime=2s -count=1
go tool pprof -top -alloc_objects mem_phase5.prof 2>&1 | head -15
```

Expected: measureFunc.func1, collectFlexLines, and generateAnonymousFlexItems no longer appear as top allocators.

- [ ] **Step 4: Run full test suite**

Run: `go test ./... 2>&1 | tail -20`

Expected: All tests pass.

- [ ] **Step 5: Commit benchmark results**

```bash
git add bench_before_phase5.txt bench_after_phase5.txt
git commit -m "bench: record Phase 5 optimization results"
```
