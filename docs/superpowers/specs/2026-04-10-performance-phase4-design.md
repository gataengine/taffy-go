# Performance Phase 4: SlotMap, Sort, and Surgical Optimizations

## Context

Phases 1-3 reduced allocations and improved flexbox/grid performance significantly:
- Phase 1: Allocation reduction (Option[T], inline cache, closures) — allocs -40-62%, time -15-37%
- Phase 2: Flexbox value-type conversion (flexItem/flexLine pointer to value) — flexbox allocs -22-33%
- Phase 3: Grid allocation reduction (sort fix, sync.Pool scratch, Cache.Get escape) — grid deep B/op -75-85%

Current Go/Rust ratios: 2.2x-4.1x. Target: 1.5-2x.

Fresh profiling (2026-04-10) identified the remaining hot spots:
- `runtime.madvise` 13-17% flat CPU — memory pressure from allocations
- `runtime.mapaccess1_fast64` 3.5-7% flat CPU — TaffyTree hash map lookups
- `slices.insertionSortCmpFunc` 8.3% flat CPU (grid) — sorting ~350-byte gridItem structs by value
- GC overhead ~7-8%
- Grid `placeGridItems` 11.8% cum, `resolveIntrinsicTrackSizes` 45% cum
- Various per-call allocations in grid (cellOccupancyMatrix, itemBatcher)

## Design

### 1. lib.SortStableByIndex — Grid Sort Optimization

**Problem:** `resolveIntrinsicTrackSizes` calls `slices.SortStableFunc(items, func(a, b gridItem) int {...})`. Since `gridItem` is ~350 bytes, each comparison copies the struct twice. At N=6561 (3x3_levels4 benchmark), this costs 8.3% of CPU.

**Solution:** `lib.SortStableByIndex[S, E]` sorts an `[]int` index array with a pointer-based comparator, then permutes items in-place. Benchmarked at 3.8x faster than value sort at N=6561.

**Files:**
- `lib/sort.go` — already implemented and benchmarked
- `compute/grid/track_sizing.go` — replace `slices.SortStableFunc` call in `resolveIntrinsicTrackSizes` with `lib.SortStableByIndex`

**API:**
```go
func SortStableByIndex[S ~[]E, E any](items S, cmp func(a, b *E) int)
```

### 2. lib.SlotMap + TaffyTree Migration

**Problem:** TaffyTree uses `map[NodeId]*nodeData` (plus 3 more maps for contexts, children, parents). Every node access goes through hash map lookup at ~20-50ns with poor cache locality. This shows as 3.5-7% flat CPU (`mapaccess1_fast64`) and contributes to GC pressure.

**Solution:** Port Rust's SlotMap as `lib.SlotMap[V]` — a dense `[]slot[V]` with generation-counted keys. Lookup is array index + generation check: O(1) with cache-friendly access. Replace TaffyTree's four maps with SlotMaps.

**Files:**
- `lib/slotmap.go` — new generic SlotMap implementation

**API:**
```go
type SlotMapKey struct {
    Index      uint32
    Generation uint32
}

type SlotMap[V any] struct { ... }

func NewSlotMap[V any]() SlotMap[V]
func NewSlotMapWithCapacity[V any](capacity int) SlotMap[V]
func (s *SlotMap[V]) Insert(value V) SlotMapKey
func (s *SlotMap[V]) Get(key SlotMapKey) *V           // nil if invalid/removed
func (s *SlotMap[V]) Remove(key SlotMapKey) (V, bool)
func (s *SlotMap[V]) Len() int
func (s *SlotMap[V]) Contains(key SlotMapKey) bool
```

**TaffyTree migration:**
- `tree.NodeId` becomes `lib.SlotMapKey` (or a type alias)
- `taffy_tree.go`: replace `map[tree.NodeId]*nodeData` with `lib.SlotMap[nodeData]` (value storage, not pointer)
- Same for `nodeContexts`, `children`, `parents` maps
- `taffy_view.go`: all methods that do `v.taffy.nodes[nodeId]` become `v.taffy.nodes.Get(nodeId)`
- No interface changes — `LayoutPartialTree` etc. still use `NodeId`

### 3. Layout Value Semantics

**Problem:** `SetUnroundedLayout(nodeId, *Layout)` and `SetFinalLayout(nodeId, *Layout)` take pointer parameters. Callers create `&tree.Layout{...}` which may escape to heap. Both methods immediately dereference and copy (`*layout`).

**Solution:** Change interface and implementation to accept `Layout` by value.

**Files:**
- `tree/traits.go` — change `SetUnroundedLayout` and `SetFinalLayout` signatures
- `taffy_view.go` — update implementation
- All callers in `compute/flexbox/`, `compute/grid/`, `compute/block/`, `compute/compute.go` — remove `&` from Layout literal arguments

**Before:**
```go
// Interface
SetUnroundedLayout(nodeId NodeId, layout *Layout)
SetFinalLayout(nodeId NodeId, layout *Layout)

// Caller
t.SetUnroundedLayout(item.node, &tree.Layout{Order: item.order, Size: size, ...})
```

**After:**
```go
// Interface
SetUnroundedLayout(nodeId NodeId, layout Layout)
SetFinalLayout(nodeId NodeId, layout Layout)

// Caller
t.SetUnroundedLayout(item.node, tree.Layout{Order: item.order, Size: size, ...})
```

### 4. Flatten cellOccupancyMatrix

**Problem:** `newCellOccupancyMatrix` allocates `[][]cellOccupancyState` — one outer slice plus N inner slices. For a 100x100 grid that's 101 allocations.

**Solution:** Single contiguous `[]cellOccupancyState` with row*colCount stride arithmetic.

**Files:**
- `compute/grid/types_occupancy.go` — change struct and all access methods

**Before:**
```go
type cellOccupancyMatrix struct {
    inner   [][]cellOccupancyState
    columns TrackCounts
    rows    TrackCounts
}
```

**After:**
```go
type cellOccupancyMatrix struct {
    inner    []cellOccupancyState  // flat: row * colCount + col
    colCount int
    columns  TrackCounts
    rows     TrackCounts
}
```

All methods that access `m.inner[row][col]` change to `m.inner[row*m.colCount+col]`. Row expansion (when grid grows) appends to the flat slice and updates colCount.

### 5. itemBatcher Returns Sub-slice

**Problem:** `itemBatcher.next()` allocates `make([]int, 0, cap)` and fills it with consecutive indices. Each batch creates a new slice allocation.

**Solution:** Return a sub-slice of the items slice instead of an index array.

**Files:**
- `compute/grid/track_sizing.go` — change `itemBatcher.next` signature and all call sites

**Before:**
```go
func (b *itemBatcher) next(items []gridItem) (batch []int, isFlex bool)
// caller: for _, idx := range batchIndices { item := &items[idx] }
```

**After:**
```go
func (b *itemBatcher) next(items []gridItem) (batch []gridItem, isFlex bool)
// caller: for i := range batch { item := &batch[i] }
```

Sub-slice shares backing array — mutations through `&batch[i]` affect the original items. Zero allocation.

## Expected Impact

| Optimization | Primary Benefit | Estimated Impact |
|---|---|---|
| SortStableByIndex | CPU: eliminates value-copy sort | ~8% grid CPU recovery |
| SlotMap | CPU: array vs hash lookup | eliminates 3.5-7% mapaccess |
| Layout value semantics | Allocs: removes potential heap escape | minor alloc reduction |
| Flatten cellOccupancyMatrix | Allocs: N+1 to 1 allocation | grid alloc reduction |
| itemBatcher sub-slice | Allocs: eliminates per-batch allocation | grid alloc reduction |

Combined with reduced GC/madvise pressure from fewer allocations, expected overall improvement: 15-25%, bringing Go/Rust ratios toward 1.8-3.0x range.

## Testing

- All existing fixture tests must pass (correctness guard)
- Benchmarks compared before/after for all grid and flexbox scenarios
- lib.SlotMap gets its own unit tests (insert, get, remove, generation reuse, iteration)
- lib.SortStableByIndex already benchmarked in testbed/slices_bench_test.go

## Out of Scope

- Flexbox `generateAnonymousFlexItems` pooling — deferred
- Production tree backend changes — SlotMap is for TaffyTree/benchmarks; production backends unaffected
