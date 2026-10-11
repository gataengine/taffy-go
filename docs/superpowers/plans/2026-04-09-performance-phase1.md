# Performance Phase 1: Allocation Reduction — Implementation Plan

> **Status (audit 2026-10-11): Implemented.** Checkboxes below were never ticked.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate heap allocations in hot layout paths to reduce Go/Rust performance gap from 2.1-4.7x toward ~1.5-2x.

**Architecture:** Surgical fixes to cache, closure, and pointer-as-optional patterns. No algorithm changes. Each fix is independently testable via the existing 4,441-test conformance suite and benchmark suite.

**Tech Stack:** Go 1.25, no new dependencies.

**Baseline benchmark results:** `bench_results_go.txt` in repo root.

---

### Task 0: Save baseline benchmark snapshot

**Files:**
- None modified

- [ ] **Step 1: Run benchmarks and save baseline**

```bash
cd /Users/wizard/workspace/gamedev/taffy3-go
go test -bench=. -benchmem -count=3 -run='^$' . 2>&1 | tee bench_baseline_phase1.txt
```

Expected: benchmark results printed and saved to `bench_baseline_phase1.txt`.

- [ ] **Step 2: Commit baseline**

```bash
git add bench_baseline_phase1.txt
git commit -m "bench: save Phase 1 baseline snapshot"
```

---

### Task 1: Extract `Option[T]` to `lib/` package

**Files:**
- Create: `lib/option.go`
- Modify: `geometry/geometry.go` — remove `Option`, `Some`, `None`, `UnwrapOr`, `Or`
- Modify: all 26 files that reference `geometry.Option`, `geometry.Some`, `geometry.None`

- [ ] **Step 1: Create `lib/option.go`**

```go
// Package lib provides general-purpose types shared across the taffy-go codebase.
package lib

// Option represents an optional value, similar to Rust's Option<T>.
type Option[T any] struct {
	Value  T
	IsSome bool
}

// Some returns an Option containing the given value.
func Some[T any](v T) Option[T] {
	return Option[T]{Value: v, IsSome: true}
}

// None returns an empty Option.
func None[T any]() Option[T] {
	return Option[T]{}
}

// UnwrapOr returns the contained value if Some, otherwise returns the alt value.
func (o Option[T]) UnwrapOr(alt T) T {
	if o.IsSome {
		return o.Value
	}
	return alt
}

// Or returns this Option if Some, otherwise returns alt.
func (o Option[T]) Or(alt Option[T]) Option[T] {
	if o.IsSome {
		return o
	}
	return alt
}
```

- [ ] **Step 2: Remove Option from `geometry/geometry.go`**

Remove lines 5-35 (the `Option` struct, `Some`, `None`, `UnwrapOr`, `Or`). Add `import "taffy-go/lib"` to the file. Add type alias and constructor forwarding so in-package references still work during migration:

Actually — since we're doing a full rename, do NOT add aliases. Just delete the type and constructors from geometry.

Delete these from `geometry/geometry.go`:
```go
type Option[T any] struct { ... }
func Some[T any](v T) Option[T] { ... }
func None[T any]() Option[T] { ... }
func (o Option[T]) UnwrapOr(alt T) T { ... }
func (o Option[T]) Or(alt Option[T]) Option[T] { ... }
```

- [ ] **Step 3: Rename all references across the codebase**

In every `.go` file (non-test and test), replace:
- `geometry.Option[` → `lib.Option[`
- `geometry.Some[` → `lib.Some[`
- `geometry.Some(` → `lib.Some(`
- `geometry.None[` → `lib.None[`

Add `"taffy-go/lib"` to imports in every affected file. Remove `"taffy-go/geometry"` from imports only if no other `geometry.` references remain in that file.

The affected files (26 source + test files) can be found via:
```bash
grep -rl 'geometry\.Option\|geometry\.Some\|geometry\.None' --include='*.go' .
```

Within `geometry/geometry.go` itself, references like `Option[float32]` (unqualified) must be changed to `lib.Option[float32]`. Similarly `Some(` → `lib.Some(` and `None[` → `lib.None[` within the geometry package. Add `import "taffy-go/lib"` to `geometry/geometry.go`.

- [ ] **Step 4: Verify everything compiles and tests pass**

```bash
go build ./...
go test ./... 2>&1 | tail -20
```

Expected: all packages compile, all 4,441+ tests pass.

- [ ] **Step 5: Run benchmarks to confirm no regression**

```bash
go test -bench=. -benchmem -count=3 -run='^$' . 2>&1 | tee bench_after_task1.txt
```

Expected: ns/op and allocs/op numbers essentially unchanged from baseline (pure refactor).

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "refactor: extract Option[T] to lib/ package

Full rename of geometry.Option/Some/None to lib.Option/Some/None
across all 26 files. No behavioral change."
```

---

### Task 2: Inline `ComputeCachedLayout` (Fix 1)

**Files:**
- Modify: `taffy_view.go:65-99` — rewrite `ComputeChildLayout`
- Modify: `taffy_view.go:185-205` — rewrite `ComputeBlockChildLayout`

- [ ] **Step 1: Rewrite `ComputeChildLayout`**

Replace `taffy_view.go` lines 65-99 (the entire `ComputeChildLayout` method) with:

```go
func (v *taffyView) ComputeChildLayout(nodeId tree.NodeId, inputs tree.LayoutInput) tree.LayoutOutput {
	// Inline cache check — avoids closure allocation from ComputeCachedLayout
	if cached := v.CacheGet(nodeId, &inputs); cached != nil {
		return *cached
	}

	node := v.taffy.nodes[nodeId]
	hasChildren := len(v.taffy.children[nodeId]) > 0
	st := &node.style

	var result tree.LayoutOutput
	if inputs.RunMode == tree.RunModePerformHiddenLayout {
		result = compute.ComputeHiddenLayout(v, nodeId)
	} else {
		switch st.Display {
		case style.DisplayNone:
			result = compute.ComputeHiddenLayout(v, nodeId)
		case style.DisplayFlex:
			if hasChildren {
				result = flexbox.ComputeFlexboxLayout(v, nodeId, inputs)
			} else {
				result = compute.ComputeLeafLayout(inputs, v.GetCoreContainerStyle(nodeId), v.ResolveCalcValue, v.measureFunc(nodeId))
			}
		case style.DisplayGrid:
			if hasChildren {
				result = grid.ComputeGridLayout(v, nodeId, inputs)
			} else {
				result = compute.ComputeLeafLayout(inputs, v.GetCoreContainerStyle(nodeId), v.ResolveCalcValue, v.measureFunc(nodeId))
			}
		case style.DisplayBlock:
			if hasChildren {
				result = block.ComputeBlockLayout(v, nodeId, inputs, nil)
			} else {
				result = compute.ComputeLeafLayout(inputs, v.GetCoreContainerStyle(nodeId), v.ResolveCalcValue, v.measureFunc(nodeId))
			}
		default:
			result = compute.ComputeLeafLayout(inputs, v.GetCoreContainerStyle(nodeId), v.ResolveCalcValue, v.measureFunc(nodeId))
		}
	}

	v.CacheStore(nodeId, &inputs, result)
	return result
}
```

- [ ] **Step 2: Rewrite `ComputeBlockChildLayout`**

Replace `taffy_view.go` lines 185-205 (the entire `ComputeBlockChildLayout` method) with:

```go
func (v *taffyView) ComputeBlockChildLayout(nodeId tree.NodeId, inputs tree.LayoutInput, blockCtx any) tree.LayoutOutput {
	// Inline cache check — avoids closure allocation from ComputeCachedLayout
	if cached := v.CacheGet(nodeId, &inputs); cached != nil {
		return *cached
	}

	node := v.taffy.nodes[nodeId]
	hasChildren := len(v.taffy.children[nodeId]) > 0
	st := &node.style

	var result tree.LayoutOutput
	if inputs.RunMode == tree.RunModePerformHiddenLayout {
		result = compute.ComputeHiddenLayout(v, nodeId)
	} else if st.Display == style.DisplayBlock && hasChildren {
		result = block.ComputeBlockLayout(v, nodeId, inputs, blockCtx)
	} else {
		result = v.ComputeChildLayout(nodeId, inputs)
	}

	v.CacheStore(nodeId, &inputs, result)
	return result
}
```

Note: when `ComputeBlockChildLayout` falls through to `ComputeChildLayout`, the cache will be checked again inside that method. This is harmless (cache miss both times on uncached nodes, and the store in `ComputeChildLayout` will populate it). If this concerns you, extract the dispatch logic into a private `computeUncached` method — but the double-check is cheap (just a bool + struct comparison) and the fallthrough case is rare.

- [ ] **Step 3: Verify tests pass**

```bash
go test ./... 2>&1 | tail -20
```

Expected: all tests pass.

- [ ] **Step 4: Run benchmarks**

```bash
go test -bench=. -benchmem -count=3 -run='^$' . 2>&1 | tee bench_after_task2.txt
```

Expected: allocs/op should drop significantly for deep benchmarks. Compare with baseline.

- [ ] **Step 5: Commit**

```bash
git add taffy_view.go
git commit -m "perf: inline ComputeCachedLayout to eliminate closure allocations

Eliminates per-node closure heap allocation in ComputeChildLayout
and ComputeBlockChildLayout by inlining cache check + store."
```

---

### Task 3: Cache — inline entries and return `&entry.content` (Fix 2 + Fix 5)

**Combines Fix 2 and Fix 5** because both modify `Cache.Get` — doing them together avoids rewriting the same code twice. After inlining entries, `&c.finalLayoutEntry.content` points into the Cache struct (which lives inside nodeData on the heap).

**Files:**
- Modify: `tree/cache.go` — change `Cache` struct, `Store`, `Get`, `Clear`, `IsEmptyCheck`, `NewCache`

- [ ] **Step 1: Change Cache struct**

Replace `tree/cache.go` lines 37-45:

```go
type Cache struct {
	finalLayoutEntry *layoutCacheEntry
	measureEntries   [cacheSize]*measureCacheEntry
	isEmpty          bool
}
```

With:

```go
type Cache struct {
	finalLayoutEntry layoutCacheEntry
	hasFinalLayout   bool
	measureEntries   [cacheSize]measureCacheEntry
	measureValid     [cacheSize]bool
	isEmpty          bool
}
```

- [ ] **Step 2: Update `NewCache`**

Replace line 48-50:

```go
func NewCache() Cache {
	return Cache{isEmpty: true}
}
```

With:

```go
func NewCache() Cache {
	return Cache{isEmpty: true}
}
```

(No change needed — zero values for bool arrays and structs are already correct.)

- [ ] **Step 3: Update `Cache.Get`**

Replace the `RunModePerformLayout` branch (lines 108-128):

```go
	case RunModePerformLayout:
		if !c.hasFinalLayout {
			return nil
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
			return &entry.content
		}
		return nil
```

Replace the `RunModeComputeSize` branch (lines 130-151):

```go
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
				result := LayoutOutputFromOuterSize(cachedSize)
				return &result
			}
		}
		return nil
```

- [ ] **Step 4: Update `Cache.Store`**

Replace lines 160-184:

```go
func (c *Cache) Store(input *LayoutInput, output LayoutOutput) {
	knownDimensions := input.KnownDimensions
	availableSpace := input.AvailableSpace

	switch input.RunMode {
	case RunModePerformLayout:
		c.isEmpty = false
		c.hasFinalLayout = true
		c.finalLayoutEntry = layoutCacheEntry{
			knownDimensions: knownDimensions,
			availableSpace:  availableSpace,
			content:         output,
		}
	case RunModeComputeSize:
		c.isEmpty = false
		slot := computeCacheSlot(knownDimensions, availableSpace)
		c.measureValid[slot] = true
		c.measureEntries[slot] = measureCacheEntry{
			knownDimensions: knownDimensions,
			availableSpace:  availableSpace,
			content:         output.Size,
		}
	case RunModePerformHiddenLayout:
		// Do not cache hidden layout results.
	}
}
```

- [ ] **Step 5: Update `Cache.Clear`**

Replace lines 187-197:

```go
func (c *Cache) Clear() ClearState {
	if c.isEmpty {
		return ClearStateAlreadyEmpty
	}
	c.isEmpty = true
	c.hasFinalLayout = false
	c.measureValid = [cacheSize]bool{}
	return ClearStateCleared
}
```

- [ ] **Step 6: Update `Cache.IsEmptyCheck`**

Replace lines 200-210:

```go
func (c *Cache) IsEmptyCheck() bool {
	if c.hasFinalLayout {
		return false
	}
	for _, valid := range c.measureValid {
		if valid {
			return false
		}
	}
	return true
}
```

- [ ] **Step 7: Verify tests pass**

```bash
go test ./... 2>&1 | tail -20
```

Expected: all tests pass.

- [ ] **Step 8: Run benchmarks**

```bash
go test -bench=. -benchmem -count=3 -run='^$' . 2>&1 | tee bench_after_task3.txt
```

Expected: significant allocs/op drop from eliminated `&layoutCacheEntry{}` and `&measureCacheEntry{}` allocations.

- [ ] **Step 9: Commit**

```bash
git add tree/cache.go
git commit -m "perf: inline cache entries to eliminate Store heap allocations

Replace pointer-based *layoutCacheEntry and *measureCacheEntry with
inline value types and bool validity flags. Zero heap allocations
in Cache.Store."
```

---

### Task 4: `gridItem.availableSpaceCache` — use `lib.Option[T]` (Fix 3)

**Files:**
- Modify: `compute/grid/types_item.go:69,122,364-377`
- Modify: `compute/grid/grid.go:270,380,387,434,441`

- [ ] **Step 1: Add lib import to types_item.go**

Add `"taffy-go/lib"` to the import block in `compute/grid/types_item.go` (if not already present from Task 1).

- [ ] **Step 2: Change the field type**

In `compute/grid/types_item.go`, replace line 69:

```go
	availableSpaceCache *geometry.Size[lib.Option[float32]]
```

With:

```go
	availableSpaceCache lib.Option[geometry.Size[lib.Option[float32]]]
```

- [ ] **Step 3: Update constructor**

In `compute/grid/types_item.go`, replace line 122:

```go
		availableSpaceCache:    nil,
```

With:

```go
		availableSpaceCache:    lib.Option[geometry.Size[lib.Option[float32]]]{},
```

- [ ] **Step 4: Update `availableSpaceCached` method**

In `compute/grid/types_item.go`, replace lines 364-377:

```go
func (gi *gridItem) availableSpaceCached(
	axis geometry.AbstractAxis,
	otherAxisTracks []gridTrack,
	otherAxisAvailableSpace lib.Option[float32],
	getTrackSizeEstimate func(track *gridTrack, availableSpace lib.Option[float32]) lib.Option[float32],
) geometry.Size[lib.Option[float32]] {
	if gi.availableSpaceCache.IsSome {
		return gi.availableSpaceCache.Value
	}
	availableSpaces := gi.availableSpace(axis, otherAxisTracks, otherAxisAvailableSpace, getTrackSizeEstimate)
	gi.availableSpaceCache = lib.Some(availableSpaces)
	return availableSpaces
}
```

- [ ] **Step 5: Update clear sites in `grid.go`**

In `compute/grid/grid.go`, replace line 270:

```go
		items[i].availableSpaceCache = nil
```

With:

```go
		items[i].availableSpaceCache = lib.Option[geometry.Size[lib.Option[float32]]]{}
```

Replace line 387:

```go
			items[i].availableSpaceCache = nil
```

With:

```go
			items[i].availableSpaceCache = lib.Option[geometry.Size[lib.Option[float32]]]{}
```

Replace line 441:

```go
			items[i].availableSpaceCache = nil
```

With:

```go
			items[i].availableSpaceCache = lib.Option[geometry.Size[lib.Option[float32]]]{}
```

- [ ] **Step 6: Update set sites in `grid.go`**

Replace line 380:

```go
			items[i].availableSpaceCache = &availSpace
```

With:

```go
			items[i].availableSpaceCache = lib.Some(availSpace)
```

Replace line 434:

```go
			items[i].availableSpaceCache = &availSpace
```

With:

```go
			items[i].availableSpaceCache = lib.Some(availSpace)
```

- [ ] **Step 7: Verify tests pass**

```bash
go test ./... 2>&1 | tail -20
```

Expected: all tests pass.

- [ ] **Step 8: Run benchmarks**

```bash
go test -bench=. -benchmem -count=3 -run='^$' . 2>&1 | tee bench_after_task4.txt
```

Expected: allocs/op drop in grid benchmarks (~590K fewer).

- [ ] **Step 9: Commit**

```bash
git add compute/grid/types_item.go compute/grid/grid.go
git commit -m "perf: replace pointer-as-optional availableSpaceCache with lib.Option

Eliminates heap allocation per grid item cache write."
```

---

### Task 5: `inFlowChildrenIter` — build once, pass slice (Fix 4)

**Files:**
- Modify: `compute/grid/grid.go:184-202`
- Modify: `compute/grid/placement.go:65-68,101,112,138`

- [ ] **Step 1: Change `placeGridItems` signature**

In `compute/grid/placement.go`, replace line 68:

```go
	childrenIter func() []gridPlacementChild,
```

With:

```go
	children []gridPlacementChild,
```

- [ ] **Step 2: Update all 3 call sites inside `placeGridItems`**

In `compute/grid/placement.go`, replace line 101:

```go
	for _, child := range childrenIter() {
```

With:

```go
	for _, child := range children {
```

Replace line 112:

```go
	for _, child := range childrenIter() {
```

With:

```go
	for _, child := range children {
```

Replace line 138:

```go
	for _, child := range childrenIter() {
```

With:

```go
	for _, child := range children {
```

- [ ] **Step 3: Build slice once in `ComputeGridLayout`**

In `compute/grid/grid.go`, replace lines 184-202:

```go
	inFlowChildrenIter := func() []gridPlacementChild {
		var result []gridPlacementChild
		for index, childNode := range t.ChildIds(node) {
			childCoreStyle := t.GetCoreContainerStyle(childNode)
			childGridStyle := t.GetGridChildStyle(childNode)
			if childCoreStyle.GetBoxGenerationMode() == style.BoxGenerationModeNone || childCoreStyle.GetPosition() == style.PositionAbsolute {
				continue
			}
			result = append(result, gridPlacementChild{index: index, node: childNode, gridItemStyle: childGridStyle, coreStyle: childCoreStyle})
		}
		return result
	}

	placeGridItems(
		&cellMatrix, &items, inFlowChildrenIter,
		direction, containerStyle.GetGridAutoFlow(),
		alignItems.UnwrapOr(style.AlignItemsStretch),
		justifyItemsStyle.UnwrapOr(style.AlignItemsStretch),
		&nameResolver,
	)
```

With:

```go
	inFlowChildren := make([]gridPlacementChild, 0, t.ChildCount(node))
	for index, childNode := range t.ChildIds(node) {
		childCoreStyle := t.GetCoreContainerStyle(childNode)
		childGridStyle := t.GetGridChildStyle(childNode)
		if childCoreStyle.GetBoxGenerationMode() == style.BoxGenerationModeNone || childCoreStyle.GetPosition() == style.PositionAbsolute {
			continue
		}
		inFlowChildren = append(inFlowChildren, gridPlacementChild{index: index, node: childNode, gridItemStyle: childGridStyle, coreStyle: childCoreStyle})
	}

	placeGridItems(
		&cellMatrix, &items, inFlowChildren,
		direction, containerStyle.GetGridAutoFlow(),
		alignItems.UnwrapOr(style.AlignItemsStretch),
		justifyItemsStyle.UnwrapOr(style.AlignItemsStretch),
		&nameResolver,
	)
```

- [ ] **Step 4: Verify tests pass**

```bash
go test ./... 2>&1 | tail -20
```

Expected: all tests pass.

- [ ] **Step 5: Run benchmarks**

```bash
go test -bench=. -benchmem -count=3 -run='^$' . 2>&1 | tee bench_after_task5.txt
```

Expected: allocs/op drop in grid benchmarks (~724K fewer from eliminating 3x slice rebuilds).

- [ ] **Step 6: Commit**

```bash
git add compute/grid/grid.go compute/grid/placement.go
git commit -m "perf: build inFlowChildren slice once instead of 3x via closure

Pass pre-built slice to placeGridItems instead of a closure that
rebuilds the same slice on each of its 3 call sites."
```

---

### Task 6: Replace all pointer-as-optional with `lib.Option[T]` (Fix 6)

**Files:**
- Modify: `taffy_tree.go:64,76-83,143-148,220-226,250-268,293-307,339,365-369,408-423`
- Modify: `taffy_view.go` — any methods that access `parents`
- Modify: `compute/block/float.go:14-16`

- [ ] **Step 1: Change `TaffyTree.parents` map type**

In `taffy_tree.go`, replace line 64:

```go
	parents      map[tree.NodeId]*tree.NodeId
```

With:

```go
	parents      map[tree.NodeId]lib.Option[tree.NodeId]
```

- [ ] **Step 2: Update `NewWithCapacity`**

In `taffy_tree.go`, replace line 80:

```go
		parents:      make(map[tree.NodeId]*tree.NodeId, capacity),
```

With:

```go
		parents:      make(map[tree.NodeId]lib.Option[tree.NodeId], capacity),
```

- [ ] **Step 3: Update `NewLeaf`**

In `taffy_tree.go`, replace line 116:

```go
	t.parents[id] = nil
```

With:

```go
	t.parents[id] = lib.Option[tree.NodeId]{}
```

Do the same for `NewLeafWithContext` (line 130) and `NewWithChildren` (line 142).

- [ ] **Step 4: Update parent assignment in `NewWithChildren`**

In `taffy_tree.go`, replace lines 144-147:

```go
	for _, child := range children {
		childCopy := id
		t.parents[child] = &childCopy
	}
```

With:

```go
	for _, child := range children {
		t.parents[child] = lib.Some(id)
	}
```

- [ ] **Step 5: Update `AddChild`**

In `taffy_tree.go`, replace lines 222-224:

```go
	parentCopy := parent
	t.parents[child] = &parentCopy
```

With:

```go
	t.parents[child] = lib.Some(parent)
```

Do the same pattern for `InsertChildAtIndex` (lines 238-239), `SetChildren` (line 267), `ReplaceChildAtIndex` (line 339).

- [ ] **Step 6: Update parent nil assignments**

In all methods that set `t.parents[child] = nil`, replace with `t.parents[child] = lib.Option[tree.NodeId]{}`. These are in:

- `Remove` (line 183)
- `RemoveChildAtIndex` (line 322)
- `SetChildren` (line 254)

- [ ] **Step 7: Update `Remove` parent check**

In `taffy_tree.go`, replace lines 168-179 (the parent removal block in `Remove`):

```go
	if parentPtr := t.parents[node]; parentPtr != nil {
		parent := *parentPtr
```

With:

```go
	if parentOpt := t.parents[node]; parentOpt.IsSome {
		parent := parentOpt.Value
```

- [ ] **Step 8: Update `Parent` return type**

In `taffy_tree.go`, change `Parent` method (line 366):

```go
func (t *TaffyTree) Parent(child tree.NodeId) *tree.NodeId {
	t.requireNode(child)
	return t.parents[child]
}
```

To:

```go
func (t *TaffyTree) Parent(child tree.NodeId) lib.Option[tree.NodeId] {
	t.requireNode(child)
	return t.parents[child]
}
```

- [ ] **Step 9: Update `MarkDirty` parent traversal**

In `taffy_tree.go`, replace lines 417-421:

```go
		parentPtr := t.parents[current]
		if parentPtr == nil {
			break
		}
		current = *parentPtr
```

With:

```go
		parentOpt := t.parents[current]
		if !parentOpt.IsSome {
			break
		}
		current = parentOpt.Value
```

- [ ] **Step 10: Update `Clear`**

In `taffy_tree.go`, replace line 158:

```go
	t.parents = make(map[tree.NodeId]*tree.NodeId)
```

With:

```go
	t.parents = make(map[tree.NodeId]lib.Option[tree.NodeId])
```

- [ ] **Step 11: Update callers of `Parent()` in tests**

Search for any test code that calls `Parent()` and checks `!= nil`:

```bash
grep -rn 'Parent(' --include='*_test.go' .
```

Update those to use `.IsSome` and `.Value` instead.

- [ ] **Step 12: Change `contentSlot.segmentID` in block/float.go**

In `compute/block/float.go`, replace line 16:

```go
	segmentID *int
```

With:

```go
	segmentID lib.Option[int]
```

Add `"taffy-go/lib"` to imports. Then search for all uses of `segmentID` in `float.go` and update:
- `== nil` → `!.IsSome`
- `!= nil` → `.IsSome`
- `*cs.segmentID` → `cs.segmentID.Value`
- `&someInt` → `lib.Some(someInt)`
- `= nil` → `= lib.Option[int]{}`

```bash
grep -n 'segmentID' compute/block/float.go
```

Update each reference accordingly.

- [ ] **Step 13: Verify tests pass**

```bash
go test ./... 2>&1 | tail -20
```

Expected: all tests pass.

- [ ] **Step 14: Run benchmarks**

```bash
go test -bench=. -benchmem -count=3 -run='^$' . 2>&1 | tee bench_after_task6.txt
```

Expected: allocs/op reduction in tree creation benchmarks (eliminated `&nodeId` per parent assignment).

- [ ] **Step 15: Commit**

```bash
git add taffy_tree.go compute/block/float.go
git commit -m "perf: replace pointer-as-optional with lib.Option[T]

Replace *tree.NodeId in TaffyTree.parents and *int in
contentSlot.segmentID with lib.Option to eliminate heap allocations.
Update Parent() return type to lib.Option[tree.NodeId]."
```

---

### Task 7: Final benchmark comparison

**Files:**
- None modified

- [ ] **Step 1: Run final benchmarks**

```bash
go test -bench=. -benchmem -count=5 -run='^$' . 2>&1 | tee bench_final_phase1.txt
```

- [ ] **Step 2: Compare against baseline**

```bash
go install golang.org/x/perf/cmd/benchstat@latest
benchstat bench_baseline_phase1.txt bench_final_phase1.txt
```

Review the delta. Expected improvements:
- Deep grid: ~40-60% fewer allocs/op, ~20-40% faster
- Deep flexbox: ~30-50% fewer allocs/op, ~20-30% faster
- Tree creation: ~10-20% fewer allocs/op

- [ ] **Step 3: Commit final results**

```bash
git add bench_final_phase1.txt
git commit -m "bench: save Phase 1 final benchmark results"
```
