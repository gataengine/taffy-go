# Phase 3: Grid Allocation Reduction

## Goal

Reduce grid layout allocations by ~53% (targeting ~357 MB of the 670 MB total in the grid deep benchmark) through three independent changes: replacing reflection-based sorting, pooling scratch slices, and eliminating cache heap escapes.

## Context

- **Current state**: Phase 1 (general alloc reduction) and Phase 2 (flexbox value-types) are complete. Go/Rust ratio is ~2.1-3.6x, target is 1.5-2x.
- **Profiling data**: `cpu_grid_deep.prof` and `mem_grid_deep.prof` from `BenchmarkGrid/Deep/2x2_levels7` (139 MB/op, 802k allocs/op).
- **Branch**: `refactor/structural-reorganization` (master).

## Top Allocation Hotspots (flat)

| Allocator | Flat MB | % | Root Cause |
|-----------|---------|---|------------|
| `grid.ComputeGridLayout` | 165 | 24.7% | Fresh slices per call (items, inFlowChildren, childStyles) |
| `reflectlite.unsafe_New` | 77 | 11.4% | `sort.SliceStable` uses reflection on 552-byte `gridItem` |
| `grid.ComputeGridLayout.func8` | 78 | 11.7% | `inFlowChildren` slice append in child loop |
| `compute.ComputeCachedLayout` | 75 | 11.1% | Cache wrapper overhead |
| `grid.initializeGridTracks` | 55 | 8.1% | `columns`/`rows` track slice allocation |
| `tree.(*Cache).Get` | 40 | 5.9% | `*LayoutOutput` return forces heap escape |
| `tree.(*Cache).Store` | 11.5 | 1.7% | Cache entry writes |

## Change 1: Replace `sort.SliceStable` with `slices.SortStableFunc`

### Problem

4 calls to `sort.SliceStable` on `[]gridItem` (552 bytes per element) use `reflectlite.unsafe_New` internally, allocating 77 MB (11.4%).

### Solution

Direct replacement at all 4 call sites with `slices.SortStableFunc` (Go 1.21+, no reflection):

| File | Line | Comparator |
|------|------|------------|
| `track_sizing.go` | 318 | By `placement(otherAxis).Start` |
| `track_sizing.go` | 393 | By `cmpByCrossFlexThenSpanThenStart(axis)` |
| `grid.go` | 495 | By `sourceOrder` |
| `grid.go` | 663 | By `rowIndexes.Start` |

`slices.SortStableFunc` takes `func(a, b T) int` — the existing `cmpByCrossFlexThenSpanThenStart` already returns `int`. The other 3 comparators need trivial conversion from `bool` to `cmp.Compare`.

### Expected Impact

~77 MB eliminated.

## Change 2: Pool grid scratch slices via `sync.Pool`

### Problem

`ComputeGridLayout` creates 5 slices fresh on every call:
- `items []gridItem` — part of the 165 MB
- `inFlowChildren []gridPlacementChild` — 78 MB (func8)
- `childStyles []style.GridItemStyle` — small
- `columns []gridTrack` — part of 55 MB via `initializeGridTracks`
- `rows []gridTrack` — part of 55 MB via `initializeGridTracks`

### Solution

A `gridScratch` struct pooled via `sync.Pool`:

```go
type gridScratch struct {
    items          []gridItem
    inFlowChildren []gridPlacementChild
    childStyles    []style.GridItemStyle
    columns        []gridTrack
    rows           []gridTrack
}

func (s *gridScratch) reset() {
    s.items = s.items[:0]
    s.inFlowChildren = s.inFlowChildren[:0]
    s.childStyles = s.childStyles[:0]
    s.columns = s.columns[:0]
    s.rows = s.rows[:0]
}

var gridScratchPool = sync.Pool{
    New: func() any { return new(gridScratch) },
}
```

Usage in `ComputeGridLayout`:

```go
scratch := gridScratchPool.Get().(*gridScratch)
defer gridScratchPool.Put(scratch)
scratch.reset()

// Use scratch.items, scratch.columns, etc. instead of local make() calls
// At end of function, save grown slices back to scratch
```

`initializeGridTracks` already takes `*[]gridTrack` and resets length to 0 (line 229). It only calls `make()` when capacity is insufficient (line 232). After the first pool cycle, capacity is sufficient and no allocation occurs.

Concurrency is safe: `sync.Pool` is goroutine-safe, and grid layout is depth-first — nested grid containers each call `Get()` and receive distinct scratch instances.

### Expected Impact

~230 MB eliminated on steady-state runs (first call still allocates, subsequent reuse).

## Change 3: Eliminate `Cache.Get` heap escape

### Problem

`Cache.Get` returns `*LayoutOutput`. In the `RunModeComputeSize` branch (`cache.go:152-153`):

```go
result := LayoutOutputFromOuterSize(cachedSize)
return &result  // heap escape — ~56 bytes per hit
```

The caller immediately dereferences: `return *cached`. The pointer is pure overhead.

`RunModeComputeSize` is the high-frequency path — called 3-6x per grid item (min-content, max-content, minimum contributions per axis) vs once for `PerformLayout`. Profile shows 434 MB cumulative through `MeasureChildSize` → `RunModeComputeSize`.

### Solution

Change return type from `*LayoutOutput` to `(LayoutOutput, bool)`:

**Files changed:**
- `tree/cache.go` — `func (c *Cache) Get(...)  (LayoutOutput, bool)`
- `tree/traits.go` — `CacheTree` interface: `CacheGet(...) (LayoutOutput, bool)`
- `taffy_view.go` — impl + 2 call sites (lines 68, 196)
- `compute/compute.go` — 1 call site (line 31)

Caller pattern becomes:
```go
if cached, ok := v.CacheGet(nodeId, &inputs); ok {
    return cached
}
```

### Expected Impact

~40 MB eliminated. The `PerformLayout` branch (`&entry.content`) was already non-escaping, so it becomes a simple value copy — same performance.

## Files Modified

| File | Changes |
|------|---------|
| `compute/grid/grid.go` | Pool acquire/release, use scratch slices, 2 sort replacements |
| `compute/grid/track_sizing.go` | 2 sort replacements |
| `compute/grid/scratch.go` (new) | `gridScratch` struct, pool, `reset()` method |
| `tree/cache.go` | `Get` returns `(LayoutOutput, bool)` |
| `tree/traits.go` | `CacheGet` interface signature |
| `taffy_view.go` | `CacheGet` impl + call sites |
| `compute/compute.go` | `CacheGet` call site |

## Testing

- All 4,441 conformance tests must pass (no behavioral change).
- `BenchmarkGrid/*` — verify alloc reduction. Target: allocs/op and B/op drop by ~50% for grid deep benchmarks.
- `BenchmarkFlexbox/*` — verify no regression (Change 3 touches shared cache path).

## Out of Scope

- Occupancy matrix flattening (6.5 MB / 1%) — deferred to a future phase.
- Named line resolver (14.5 MB / 2.2%) — deferred to a future phase.
- SlotMap-based TaffyTree (Phase 4) — separate effort.
- Index-based sorting (sorting index arrays instead of 552-byte structs) — not needed since eliminating reflection is the primary win.
