# Benchmark Suite Design

> **Status (audit 2026-10-11): Implemented.**

**Date:** 2026-04-06
**Status:** Approved

## Goal

Add benchmarks to taffy-go that mirror the Rust Taffy benchmark suite, enabling direct performance comparison between the Go port and the original Rust implementation in preparation for optimization work.

## Approach

Port the Rust Taffy programmatic tree builders and benchmark scenarios to Go using standard `testing.B` benchmarks. Results are compared manually against Rust Criterion output, and Go-to-Go regression tracking uses `benchstat`.

## File Structure

```
taffy-go/
├── bench_helpers_test.go       # Tree builders, style generators, seeded RNG
├── bench_flexbox_test.go       # 5 flexbox benchmark groups
├── bench_grid_test.go          # 3 grid benchmark groups
├── bench_tree_creation_test.go # Tree allocation benchmarks
```

All files are in the root `taffy` package (access to internals, idiomatic Go test layout).

Mixed flex+grid benchmarks are **out of scope** — the Rust version requires Parley (a text layout engine) for measure functions.

## Shared Helpers (`bench_helpers_test.go`)

### Style Generator Interface

```go
type styleGen interface {
    leafStyle(rng *rand.Rand) style.Style
    containerStyle(rng *rand.Rand) style.Style
}
```

### Implementations

- **`fixedStyleGen`** — returns the same style for all nodes
- **`randomStyleGen`** — random dimensions per node (20% auto, 60% length 0-500, 20% percent 0-100%)

### RNG

`rand.New(rand.NewPCG(12345, 0))` — deterministic, seeded to match Rust's seed value. The sequence will differ from ChaCha8 but that's irrelevant; what matters is determinism and the same distribution of random values.

### Tree Builders

Direct ports of the Rust `BuildTree`/`BuildTreeExt` methods:

- **`buildDeepTree(tree *TaffyTree, maxNodes, branchingFactor uint32, gen styleGen, rng *rand.Rand) []tree.NodeId`**
  Recursive balanced tree. If `maxNodes <= branchingFactor`, creates leaf nodes. Otherwise divides remaining nodes equally among `branchingFactor` children, each getting `(maxNodes - branchingFactor) / branchingFactor` nodes.

- **`buildFlatHierarchy(tree *TaffyTree, targetNodeCount uint32, gen styleGen, rng *rand.Rand) []tree.NodeId`**
  Two-level hierarchy. Creates container nodes with 1-4 random leaf children until target count is reached. Returns the container nodes as root children.

- **`buildSuperDeepHierarchy(tree *TaffyTree, depth, nodesPerLevel uint32, gen styleGen, rng *rand.Rand) []tree.NodeId`**
  Linear chain. At each level: wraps previous children in a container node, then adds `nodesPerLevel - 1` leaf siblings.

Each builder returns children for the root node. The benchmark function creates the root and sets children.

## Flexbox Benchmarks (`bench_flexbox_test.go`)

### BenchmarkFlexbox/HugeNested

Mirrors Rust's `yoga 'huge nested'` group. Deep balanced tree with branching factor 10.

| Sub-benchmark | Node Count | Style |
|---------------|-----------|-------|
| `/10000` | 10,000 | Fixed: `size: length(10), flex_grow: 1.0` |

### BenchmarkFlexbox/Wide

Two-level flat hierarchy with random dimensions.

| Sub-benchmark | Node Count | Style |
|---------------|-----------|-------|
| `/10000` | 10,000 | Random: 20% auto, 60% length(0-500), 20% percent(0-1) |

### BenchmarkFlexbox/DeepRandom

Deep binary tree (branching factor 2) with random dimensions.

| Sub-benchmark | Node Count | Style |
|---------------|-----------|-------|
| `/4000` | 4,000 | Random dimensions |
| `/10000` | 10,000 | Random dimensions |

### BenchmarkFlexbox/DeepAuto

Deep binary tree with auto-sized flex layout.

| Sub-benchmark | Node Count | Style |
|---------------|-----------|-------|
| `/4000` | 4,000 | Fixed: `flex_grow: 1.0, margin: length(10)` |
| `/10000` | 10,000 | Fixed: `flex_grow: 1.0, margin: length(10)` |

### BenchmarkFlexbox/SuperDeep

Linear depth chain, 3 nodes per level.

| Sub-benchmark | Depth | Style |
|---------------|-------|-------|
| `/100` | 100 | Fixed: `flex_direction: Row, flex_grow: 1.0, margin: length(10)` |

### Benchmark Pattern

Each benchmark follows `b.Run` with sub-benchmarks. Each iteration:
1. Creates a fresh `TaffyTree`
2. Builds the tree with the specified builder
3. Calls `ComputeLayout` (this is the measured operation)

Tree building is done in `b.StopTimer()`/`b.StartTimer()` blocks so only layout computation is measured.

## Grid Benchmarks (`bench_grid_test.go`)

Grid benchmarks use their own tree builders (matching the Rust `grid.rs` which doesn't use the generic `BuildTree` trait).

### Helper Functions

- **`randomGridTrack(rng)`** — returns a random track sizing: 10% auto, 10% min-content, 10% max-content, 20% fr(1), 10% minmax(0, fr(1)), 20% length(40), 20% percent(0.3)
- **`randomNxNGridStyle(rng, trackCount)`** — grid style with random column and row templates
- **`buildGridFlatHierarchy(colCount, rowCount)`** — flat NxN grid with leaf children
- **`buildDeepGridTree(tree, levels, trackCount, ...)`** — recursive nested grids

### BenchmarkGrid/Wide

Flat NxN grids. Available space: `length(12000)`.

| Sub-benchmark | Grid Size | Total Cells |
|---------------|----------|-------------|
| `/31x31` | 31×31 | 961 |
| `/100x100` | 100×100 | 10,000 |

### BenchmarkGrid/Deep

Nested grids where each cell contains a sub-grid.

| Sub-benchmark | Tracks | Levels | Total Nodes |
|---------------|--------|--------|-------------|
| `/2x2` | 2×2 | 5 | 1,024 |
| `/3x3` | 3×3 | 4 | 6,561 |
| `/2x2_deep` | 2×2 | 7 | 16,384 |

### BenchmarkGrid/SuperDeep

Deeply nested 1×1 grids. Available space: `max-content`.

| Sub-benchmark | Levels |
|---------------|--------|
| `/100` | 100 |
| `/1000` | 1,000 |

## Tree Creation Benchmarks (`bench_tree_creation_test.go`)

Measures node allocation speed (not layout computation).

### BenchmarkTreeCreation

Builds a flat hierarchy (same as flexbox wide benchmark's tree shape) and measures total creation time.

| Sub-benchmark | Node Count |
|---------------|-----------|
| `/1000` | 1,000 |
| `/10000` | 10,000 |
| `/100000` | 100,000 |

Each iteration creates a fresh `TaffyTree`, builds the flat hierarchy, and discards the result. The entire creation is measured (no stop/start timer).

## Running Benchmarks

### Go

```bash
# All benchmarks
go test -bench=. -benchmem -count=5 -timeout=30m ./... | tee go_bench.txt

# Specific suite
go test -bench=BenchmarkFlexbox -benchmem -count=5 ./...

# Compare runs
benchstat old.txt new.txt
```

### Rust (for comparison)

```bash
cd /Users/wizard/workspace/gamedev/taffy
cargo bench
```

### Comparing Go vs Rust

Criterion (Rust) reports time in ns/µs/ms. Go's `testing.B` reports ns/op. Compare the median/mean times directly. A simple comparison script or manual table is sufficient for the initial pass.

## Out of Scope

- Mixed flex+grid benchmarks (requires text measurement engine)
- 100,000 node variants (Rust gates behind `large` feature — can add later)
- Yoga comparison benchmarks
- Automated cross-language comparison tooling
