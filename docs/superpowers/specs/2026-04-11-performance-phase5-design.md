# Performance Phase 5: Allocation Micro-Optimizations

## Context

After 4 optimization phases, Go/Rust layout ratios are 1.6-3.0x. The GOGC=off test proves the remaining gap on deep benchmarks (~3x) is mostly inherent Go runtime overhead (interface dispatch, bounds checks, struct copying). However, fresh profiling reveals three allocation sites that account for 87% of all allocations in the worst benchmark (FlexboxSuperDeep/100) and can be eliminated cheaply.

## Goal

Reduce SuperDeep ratio from ~3.0x toward ~2.5x by eliminating the top three allocation sources. Low effort, low risk — surgical fixes only.

## Current Allocation Breakdown (SuperDeep/100)

| Source | % of allocs | Location |
|--------|-------------|----------|
| measureFunc closure | 52.5% | taffy_view.go:52-63 |
| collectFlexLines | 18.0% | flex_line.go:26 |
| generateAnonymousFlexItems | 17.3% | flex_item.go:63 |
| Tree creation (one-time) | ~7% | N/A |
| Other | ~5% | Various |

## Fix 1: Cache measureFunc Closure

**Problem**: `measureFunc(nodeId)` creates a new heap-allocated closure every time a leaf node is dispatched. The closure captures `v` (constant across the layout) and `nodeId` (varies per call). Called for every leaf-node layout dispatch — 4 call sites in `ComputeChildLayout`.

**Fix**: Store a single reusable closure on `taffyView`. Add a `currentMeasureNodeId` field that the closure reads instead of capturing a fresh `nodeId` each call. Set the field before calling `ComputeLeafLayout`.

```go
type taffyView struct {
    taffy              *TaffyTree
    measureFunction    MeasureFunc
    currentMeasureNodeId tree.NodeId          // NEW
    cachedMeasureFn    func(                  // NEW
        geometry.Size[lib.Option[float32]],
        geometry.Size[style.AvailableSpace],
    ) geometry.Size[float32]
}
```

Initialize `cachedMeasureFn` once when `taffyView` is created. The closure captures `v` (stable pointer) and reads `v.currentMeasureNodeId` on each invocation.

At each call site, replace:
```go
v.measureFunc(nodeId)
```
with:
```go
v.currentMeasureNodeId = nodeId
v.cachedMeasureFn
```

**Safety**: `ComputeLeafLayout` is non-recursive (leaf nodes have no children), so `currentMeasureNodeId` cannot be clobbered by re-entrant calls.

**Impact**: Eliminates 52.5% of SuperDeep allocs.

## Fix 2: Pool flexItem Slices

**Problem**: `generateAnonymousFlexItems` calls `make([]flexItem, 0, cap)` per node. Each flexItem is ~440 bytes, so a 10-child node allocates ~4.4KB that's immediately discarded after layout.

**Fix**: Use `sync.Pool` for `[]flexItem` slices. Get a slice from the pool, reset length to 0, use it, return it after the flex layout for that node completes.

The pool is package-level in `compute/flexbox`. The slice is obtained in `generateAnonymousFlexItems` and returned in `computePreliminary` after all flex phases complete (the slice must outlive `generateAnonymousFlexItems` since `flexLine.items` is a sub-slice of it).

**Impact**: Eliminates 17.3% of SuperDeep allocs.

## Fix 3: Pool flexLine Slices

**Problem**: `collectFlexLines` calls `make([]flexLine, 0, 1)` (or larger for wrap) per node. 18% of SuperDeep allocs.

**Fix**: Same `sync.Pool` pattern. Get from pool in `collectFlexLines`, return in `computePreliminary` after all flex phases complete.

**Impact**: Eliminates 18.0% of SuperDeep allocs.

## Expected Results

| Benchmark | Before | After (est.) | Notes |
|-----------|--------|-------------|-------|
| Flex SuperDeep/100 | 3.0x | ~2.5x | GOGC=off floor is 2.64x |
| Flex DeepAuto/4K | 2.8x | ~2.7x | Computation-bound, small alloc benefit |
| Flex DeepAuto/10K | 3.0x | ~2.8x | Same |
| Grid benchmarks | unchanged | unchanged | Different allocation sites |

## What This Won't Fix

- DeepAuto ratio (computation-bound, GOGC=off shows 0% GC tax)
- Grid deep ratios (different allocation patterns)
- The ~2.5x floor on recursive workloads (inherent Go runtime overhead)

## Verification

- Run benchmarks before/after with `-benchmem -count=5`
- Confirm alloc/op drops significantly for SuperDeep
- Confirm no test regressions
- Profile SuperDeep after to verify allocation sites are eliminated
