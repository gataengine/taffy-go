# Performance Optimization Phase 1: Allocation Reduction

**Date:** 2026-04-09
**Goal:** Reduce Go/Rust performance gap from 2.1-4.7x to ~1.5-2x by eliminating heap allocations in hot paths.

## Context

Profiling of the worst benchmarks (deep grid 4.7x, deep flexbox 4.7x) reveals that ~30% of CPU is spent in GC + malloc. The top allocation sources are:

| Source | Allocs (grid) | Allocs (flexbox) | Root cause |
|--------|--------------|-----------------|------------|
| `ComputeCachedLayout` closure | 813K | 333K | Closure captures per node |
| `ComputeGridLayout.func8` (inFlowChildrenIter) | 724K | — | Slice rebuilt 3x per container |
| `Cache.Get` | 647K | 270K | `&result` escapes to heap |
| `gridItem.availableSpaceCache` | 590K | — | Pointer-as-optional |
| `newNamedLineResolver` | 317K | — | Deferred to Phase 2 |
| `newCellOccupancyMatrix` | 251K | — | Deferred to Phase 2 |
| `Cache.Store` | 146K | 119K | `&entry{}` escapes to heap |
| `generateAnonymousFlexItems` | — | 131K | Deferred to Phase 2 |
| `collectFlexLines` | — | 66K | Deferred to Phase 2 |

## Scope

Phase 1 targets fixes that eliminate allocations without changing algorithms or data structures beyond the cache and tree view layer. Phases 2-4 (grid internals, flexbox internals, SlotMap storage) are deferred.

## Prerequisite: Extract `Option[T]` to `lib/`

**Problem:** `geometry.Option[T]` is a general-purpose type living in a domain-specific package. Fix 3 and Fix 6 need `Option[T]` in packages that shouldn't depend on `geometry` semantically.

**Change:**
- Create `lib/` package with `Option[T]`, `Some[T]`, `None[T]`, `UnwrapOr`, `Or`
- Full rename across all 26 files (~367 `geometry.Option` occurrences, ~152 `geometry.Some`/`geometry.None` calls)
- Remove `Option`, `Some`, `None` from `geometry/geometry.go`
- Update all imports

**Validation:** all tests pass, benchmarks unchanged.

## Fix 1: Inline `ComputeCachedLayout`

**Problem:** `taffyView.ComputeChildLayout` (taffy_view.go:69) and `ComputeBlockChildLayout` (taffy_view.go:189) pass a closure to `compute.ComputeCachedLayout`. The closure captures `node` and `hasChildren`, forcing a heap allocation on every call — once per node per layout pass.

**Change:** Inline the cache check and store directly into both methods. The dispatch switch stays in the method body. `compute.ComputeCachedLayout` remains available for external callers but is no longer used internally.

```go
func (v *taffyView) ComputeChildLayout(nodeId tree.NodeId, inputs tree.LayoutInput) tree.LayoutOutput {
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

Same pattern for `ComputeBlockChildLayout`, keeping the `blockCtx` pass-through for block children.

**Eliminates:** ~813K allocs (grid), ~333K allocs (flexbox)

## Fix 2: `Cache.Get` PerformLayout — return `&entry.content`

**Problem:** `Cache.Get` in the `RunModePerformLayout` branch copies the cached `LayoutOutput` into a local variable, then returns its address. The local escapes to heap on every cache hit.

```go
// current (tree/cache.go:124-127)
result := entry.content
return &result  // heap alloc
```

**Change:** Return a pointer directly into the cache entry's storage.

```go
return &entry.content  // no alloc, points into existing entry
```

This is safe because `entry` is `*layoutCacheEntry`, already heap-allocated. `&entry.content` points into it.

The `RunModeComputeSize` branch is left unchanged — it constructs a `LayoutOutput` via `LayoutOutputFromOuterSize()` which cannot be returned by reference without storing the full `LayoutOutput` in measure entries.

**Eliminates:** portion of ~647K allocs (grid), ~270K allocs (flexbox) — only PerformLayout hits

## Fix 3: `gridItem.availableSpaceCache` — use `lib.Option[T]`

**Problem:** `gridItem.availableSpaceCache` is `*geometry.Size[geometry.Option[float32]]`. Setting the cache does `gi.availableSpaceCache = &availableSpaces`, which heap-allocates every time.

**Change:**
```go
// current (compute/grid/types_item.go:69)
availableSpaceCache *geometry.Size[lib.Option[float32]]

// proposed
availableSpaceCache lib.Option[geometry.Size[lib.Option[float32]]]
```

Update all usage sites:
- `types_item.go:371-376`: check `.IsSome` instead of `!= nil`, set with `lib.Some()`
- `grid.go:270,387,441`: clear with `lib.None[...]()`, set with `lib.Some()`
- `grid.go:380,434`: set with `lib.Some(availSpace)`

**Eliminates:** ~590K allocs (grid)

## Fix 4: `inFlowChildrenIter` — build once, pass slice

**Problem:** `inFlowChildrenIter` is a closure that builds a `[]gridPlacementChild` slice. It is called 3 times inside `placeGridItems` (placement.go lines 101, 112, 138), rebuilding the identical slice each time. Each call allocates the slice and boxes interface values on append.

**Change:** Build the slice once in `ComputeGridLayout`, pass it directly to `placeGridItems`.

```go
// grid.go — build once
inFlowChildren := make([]gridPlacementChild, 0, t.ChildCount(node))
for index, childNode := range t.ChildIds(node) {
    childCoreStyle := t.GetCoreContainerStyle(childNode)
    childGridStyle := t.GetGridChildStyle(childNode)
    if childCoreStyle.GetBoxGenerationMode() == style.BoxGenerationModeNone ||
        childCoreStyle.GetPosition() == style.PositionAbsolute {
        continue
    }
    inFlowChildren = append(inFlowChildren, gridPlacementChild{
        index: index, node: childNode,
        gridItemStyle: childGridStyle, coreStyle: childCoreStyle,
    })
}

placeGridItems(&cellMatrix, &items, inFlowChildren, ...)
```

```go
// placement.go — change signature
func placeGridItems(
    cellOccupancyMatrix *cellOccupancyMatrix,
    items *[]gridItem,
    children []gridPlacementChild,  // was func() []gridPlacementChild
    ...
)
```

**Eliminates:** ~724K allocs (grid)

## Fix 5: `Cache.Store` — inline entries

**Problem:** `Cache.Store` heap-allocates `*layoutCacheEntry` and `*measureCacheEntry` on every store via `&layoutCacheEntry{...}` and `&measureCacheEntry{...}`.

**Change:** Store entries as values with validity bools instead of pointers.

```go
// current (tree/cache.go:38-45)
type Cache struct {
    finalLayoutEntry *layoutCacheEntry
    measureEntries   [cacheSize]*measureCacheEntry
    isEmpty          bool
}

// proposed
type Cache struct {
    finalLayoutEntry layoutCacheEntry
    hasFinalLayout   bool
    measureEntries   [cacheSize]measureCacheEntry
    measureValid     [cacheSize]bool
    isEmpty          bool
}
```

Store writes directly into the inline storage:
```go
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
```

`Cache.Get` checks `c.hasFinalLayout` / `c.measureValid[i]` instead of nil. `Cache.Clear` resets bools: `c.hasFinalLayout = false; c.measureValid = [cacheSize]bool{}`.

Fix 2 (`return &entry.content`) works with this change: `&c.finalLayoutEntry.content` points into the inline Cache struct which lives inside `nodeData` on the heap.

**Eliminates:** ~146K allocs (grid), ~119K allocs (flexbox)

## Fix 6: Replace all pointer-as-optional with `lib.Option[T]`

**Problem:** Several places use `*T` where nil means "absent" — each assignment heap-allocates.

**Known candidates from audit:**

| Location | Field/Return | Type | Hot path? |
|----------|-------------|------|-----------|
| `taffy_tree.go:64` | `TaffyTree.parents` map values | `*tree.NodeId` | Yes (tree building) |
| `compute/block/float.go:16` | `contentSlot.segmentID` | `*int` | Moderate |
| `taffy_tree.go:366` | `TaffyTree.Parent()` return | `*tree.NodeId` | Low (public API) |
| `taffy_tree.go:211` | `TaffyTree.GetNodeContext()` return | `*NodeContext` | Low (public API) |

**Change:** Replace `*T` fields with `lib.Option[T]`. For `TaffyTree.parents`:

```go
// current
parents map[tree.NodeId]*tree.NodeId

// proposed
parents map[tree.NodeId]lib.Option[tree.NodeId]
```

For public API returns (`Parent()`, `GetNodeContext()`), change return type to `lib.Option[T]` — this is an allowed minor API change per scope agreement.

**Eliminates:** variable — primarily helps tree creation benchmarks where parent pointers are allocated per node.

## Validation

After every individual fix (prerequisite, fix 1, fix 2, ... fix 6):
1. Run full test suite: `go test ./...` — must pass with zero failures
2. Run benchmarks: `go test -bench=. -benchmem -count=3 -run='^$' .`
3. Record allocs/op and ns/op, compare against baseline (`bench_results_go.txt`)
4. Commit the fix with benchmark delta in the commit message

## Expected Impact

| Benchmark | Current | Expected after Phase 1 |
|-----------|---------|----------------------|
| Grid Deep 2x2 levels7 | 4.7x Rust (1.5M allocs) | ~2.5-3x Rust (~200K allocs) |
| Flexbox DeepAuto 10K | 4.7x Rust (360K allocs) | ~2.5-3x Rust (~100K allocs) |
| Grid Wide 100x100 | 2.5x Rust (131K allocs) | ~2x Rust (~80K allocs) |
| Tree Creation 1K | 4.4x Rust | ~3x Rust |

Further gains from Phases 2-4 (grid internals, flexbox internals, SlotMap) are expected to bring the gap to 1.5-2x.

## Deferred to Later Phases

- `Cache.Get` ComputeSize branch allocation
- `newCellOccupancyMatrix` — flatten to single slice (Phase 2)
- `newNamedLineResolver` — lazy map creation (Phase 2)
- `generateAnonymousFlexItems` — value-type flexItem (Phase 3)
- `collectFlexLines` — value-type flexLine (Phase 3)
- SlotMap-based TaffyTree storage (Phase 4)
