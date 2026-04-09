# Flexbox Value-Type Conversion + File Split

**Date:** 2026-04-09
**Status:** Approved
**Predecessor:** [Performance Phase 1](2026-04-09-performance-phase1-design.md)

## Problem

After Phase 1 allocation reduction, flexbox layout still allocates ~24.5MB (18% flat) in `generateAnonymousFlexItems` due to per-item heap allocations (`[]*flexItem`). The `flexLine` type adds a second level of pointer indirection (`*flexLine` containing `[]*flexItem`). Additionally, `flexbox.go` is 2,795 lines — the largest file in the codebase — making changes hard to review.

## Profiling Evidence

From `mem_flex_deep.prof` (FlexboxDeepAuto benchmark):

| Source | Flat alloc | Share |
|--------|-----------|-------|
| `ComputeCachedLayout` | 30.5 MB | 22% |
| `newNodeData` | 31.0 MB | 22% (tree construction, not layout) |
| `generateAnonymousFlexItems` | 24.5 MB | 18% |

## Design

### Step 1: Split `flexbox.go` into logical files

No behavior change. Extract into:

| File | Contents | Approx LOC |
|------|----------|------------|
| `flex_item.go` | `flexItem` struct, `generateAnonymousFlexItems()`, item-level helpers (`calculateFlexItem`, `determineFlexBaseSize`, `determineHypotheticalCrossSize`) | ~600 |
| `flex_line.go` | `flexLine` struct, `collectFlexLines()`, line-level operations (`resolveFlexibleLengths`, `determineUsedCrossSize`, `distributeRemainingFreeSpace`) | ~500 |
| `flexbox.go` | `ComputeFlexboxLayout`, `computePreliminary`, `finalLayoutPass`, alignment, wrapping orchestration | ~1,700 |

### Step 2: Convert `flexItem` from pointer to value type

Change `[]*flexItem` to `[]flexItem` throughout.

- Pre-allocate with `make([]flexItem, 0, len(childIds))` — capacity is upper bound (absolute/none children are skipped), slice never grows after initial creation
- Functions keep `*flexItem` parameter type; callers pass `&items[idx]`
- Safe because backing array never reallocates after creation

### Step 3: Convert `flexLine` from pointer to value type

Change `[]*flexLine` to `[]flexLine`. Change `flexLine.items` from `[]*flexItem` to `[]flexItem` (sub-slice of the flat items array).

Before:
```go
type flexLine struct {
    items       []*flexItem
    crossSize   float32
    offsetCross float32
}
```

After:
```go
type flexLine struct {
    items       []flexItem  // sub-slice of flat array
    crossSize   float32
    offsetCross float32
}
```

Sub-slice is safe because the backing array never reallocates. Items are accessed directly without pointer chasing.

### Step 4: Benchmark snapshot

Record results to `bench_after_flexbox_valuetypes.txt`.

## Commit Plan

| Step | Message | Behavioral change |
|------|---------|-------------------|
| 1 | `refactor: split flexbox.go into logical files` | None |
| 2 | `perf: convert flexItem from pointer to value type` | None |
| 3 | `perf: convert flexLine from pointer to value type` | None |
| 4 | `bench: record flexbox value-type results` | None |

## Invariants

- All 4,441 conformance tests pass after each step
- Benchmarks run after each perf step to measure incremental impact

## Not In Scope

- Grid optimizations (separate phase)
- TaffyTree SlotMap conversion (separate phase)
- Pooling / sync.Pool
- Any public API changes
