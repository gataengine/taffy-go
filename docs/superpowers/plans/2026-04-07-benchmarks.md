# Benchmark Suite Implementation Plan

> **Status (audit 2026-10-11): Implemented.** Checkboxes below were never ticked.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Go benchmarks that mirror the Rust Taffy benchmark suite for direct performance comparison.

**Architecture:** Four `_test.go` files in the root `taffy` package — shared helpers, flexbox, grid, and tree creation benchmarks. Tree builders are direct ports of Rust's `BuildTree`/`BuildTreeExt` traits. Standard `testing.B` with `b.Run` sub-benchmarks.

**Tech Stack:** Go `testing` package, `math/rand/v2` for seeded RNG, `benchstat` for comparison.

---

### Task 1: Shared benchmark helpers

**Files:**
- Create: `bench_helpers_test.go`

- [ ] **Step 1: Create the style generator interface and fixed implementation**

Create `bench_helpers_test.go` with the `styleGen` interface and `fixedStyleGen`:

```go
package taffy

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"taffy-go/geometry"
	"taffy-go/style"
	"taffy-go/tree"
)

// styleGen generates styles for benchmark tree nodes.
type styleGen interface {
	leafStyle(rng *rand.Rand) style.Style
	containerStyle(rng *rand.Rand) style.Style
}

// fixedStyleGen returns the same style for all nodes.
type fixedStyleGen struct {
	style style.Style
}

func (f fixedStyleGen) leafStyle(_ *rand.Rand) style.Style      { return f.style }
func (f fixedStyleGen) containerStyle(_ *rand.Rand) style.Style  { return f.style }
```

- [ ] **Step 2: Add random style generator and random dimension helper**

Append the `randomDimension` helper and `randomStyleGen` to `bench_helpers_test.go`:

```go
// randomDimension returns a random Dimension: 20% auto, 60% length(0-500), 20% percent(0-1).
func randomDimension(rng *rand.Rand) style.Dimension {
	r := rng.Float32()
	switch {
	case r < 0.2:
		return style.AutoDimension()
	case r < 0.8:
		return style.LengthDimension(rng.Float32() * 500.0)
	default:
		return style.PercentDimension(rng.Float32())
	}
}

// randomStyleGen generates random dimensions for each node.
type randomStyleGen struct{}

func (randomStyleGen) leafStyle(rng *rand.Rand) style.Style {
	s := style.DefaultStyle()
	s.Size = geometry.Size[style.Dimension]{
		Width:  randomDimension(rng),
		Height: randomDimension(rng),
	}
	return s
}

func (randomStyleGen) containerStyle(rng *rand.Rand) style.Style {
	s := style.DefaultStyle()
	s.Size = geometry.Size[style.Dimension]{
		Width:  randomDimension(rng),
		Height: randomDimension(rng),
	}
	return s
}
```

- [ ] **Step 3: Add tree builder functions**

Append the three tree builders to `bench_helpers_test.go`:

```go
const benchRNGSeed = 12345

// newBenchRNG creates a deterministic RNG for benchmarks.
func newBenchRNG() *rand.Rand {
	return rand.New(rand.NewPCG(benchRNGSeed, 0))
}

// buildDeepTree builds a recursive balanced tree with the given branching factor.
// Returns the child node IDs for the caller to attach to a parent.
func buildDeepTree(t *TaffyTree, maxNodes, branchingFactor uint32, gen styleGen, rng *rand.Rand) []tree.NodeId {
	if maxNodes <= branchingFactor {
		nodes := make([]tree.NodeId, maxNodes)
		for i := range nodes {
			nodes[i] = t.NewLeaf(gen.leafStyle(rng))
		}
		return nodes
	}
	childMaxNodes := (maxNodes - branchingFactor) / branchingFactor
	nodes := make([]tree.NodeId, branchingFactor)
	for i := range nodes {
		children := buildDeepTree(t, childMaxNodes, branchingFactor, gen, rng)
		nodes[i] = t.NewWithChildren(gen.containerStyle(rng), children)
	}
	return nodes
}

// buildFlatHierarchy builds a 2-level hierarchy with random 1-4 leaf children per container.
func buildFlatHierarchy(t *TaffyTree, targetNodeCount uint32, gen styleGen, rng *rand.Rand) []tree.NodeId {
	var containers []tree.NodeId
	var count uint32
	for count < targetNodeCount {
		subCount := rng.IntN(4) + 1
		leaves := make([]tree.NodeId, subCount)
		for i := range leaves {
			leaves[i] = t.NewLeaf(gen.leafStyle(rng))
		}
		container := t.NewWithChildren(gen.containerStyle(rng), leaves)
		containers = append(containers, container)
		count += uint32(1 + subCount)
	}
	return containers
}

// buildSuperDeepHierarchy builds a linear chain of depth levels, each with nodesPerLevel nodes.
func buildSuperDeepHierarchy(t *TaffyTree, depth, nodesPerLevel uint32, gen styleGen, rng *rand.Rand) []tree.NodeId {
	children := make([]tree.NodeId, 0, nodesPerLevel)
	for level := uint32(0); level < depth; level++ {
		container := t.NewWithChildren(gen.containerStyle(rng), children)
		children = make([]tree.NodeId, 0, nodesPerLevel)
		children = append(children, container)
		for j := uint32(1); j < nodesPerLevel; j++ {
			children = append(children, t.NewLeaf(gen.leafStyle(rng)))
		}
	}
	return children
}
```

- [ ] **Step 4: Add benchmark helper that builds tree and computes layout**

Append a helper that encapsulates the stop-timer/build/start-timer/compute pattern:

```go
// benchmarkLayout is a helper that builds a tree and benchmarks layout computation.
// buildFn receives a fresh TaffyTree and RNG, and returns the root node ID.
func benchmarkLayout(b *testing.B, availableSpace geometry.Size[style.AvailableSpace], buildFn func(t *TaffyTree, rng *rand.Rand) tree.NodeId) {
	b.Helper()
	for b.Loop() {
		b.StopTimer()
		rng := newBenchRNG()
		t := New()
		root := buildFn(t, rng)
		b.StartTimer()
		t.ComputeLayout(root, availableSpace)
	}
}

// defaultAvailableSpace returns the default available space (max-content on both axes).
func defaultAvailableSpace() geometry.Size[style.AvailableSpace] {
	return geometry.Size[style.AvailableSpace]{
		Width:  style.AvailableSpaceMaxContentValue,
		Height: style.AvailableSpaceMaxContentValue,
	}
}

// lengthAvailableSpace returns definite available space on both axes.
func lengthAvailableSpace(width, height float32) geometry.Size[style.AvailableSpace] {
	return geometry.Size[style.AvailableSpace]{
		Width:  style.AvailableSpaceFromLength(width),
		Height: style.AvailableSpaceFromLength(height),
	}
}
```

- [ ] **Step 5: Verify the helpers compile**

Run: `go build ./...`
Expected: no errors (test files won't be built, but this catches import issues in non-test code)

Then run: `go vet ./...`
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add bench_helpers_test.go
git commit -m "bench: add shared benchmark helpers (tree builders, style generators)"
```

---

### Task 2: Flexbox benchmarks

**Files:**
- Create: `bench_flexbox_test.go`

- [ ] **Step 1: Create flexbox benchmark file with HugeNested benchmark**

Create `bench_flexbox_test.go`:

```go
package taffy

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"taffy-go/geometry"
	"taffy-go/style"
	"taffy-go/tree"
)

func BenchmarkFlexbox(b *testing.B) {
	b.Run("HugeNested", benchFlexboxHugeNested)
	b.Run("Wide", benchFlexboxWide)
	b.Run("DeepRandom", benchFlexboxDeepRandom)
	b.Run("DeepAuto", benchFlexboxDeepAuto)
	b.Run("SuperDeep", benchFlexboxSuperDeep)
}

func benchFlexboxHugeNested(b *testing.B) {
	hugeNestedStyle := style.DefaultStyle()
	hugeNestedStyle.Size = geometry.Size[style.Dimension]{
		Width:  style.LengthDimension(10),
		Height: style.LengthDimension(10),
	}
	hugeNestedStyle.FlexGrow = 1.0
	gen := fixedStyleGen{style: hugeNestedStyle}

	for _, nodeCount := range []uint32{10_000} {
		b.Run(fmt.Sprintf("%d", nodeCount), func(b *testing.B) {
			benchmarkLayout(b, defaultAvailableSpace(), func(t *TaffyTree, rng *rand.Rand) tree.NodeId {
				children := buildDeepTree(t, nodeCount, 10, gen, rng)
				return t.NewWithChildren(style.DefaultStyle(), children)
			})
		})
	}
}
```

- [ ] **Step 2: Add Wide benchmark**

Append to `bench_flexbox_test.go`:

```go
func benchFlexboxWide(b *testing.B) {
	for _, nodeCount := range []uint32{10_000} {
		b.Run(fmt.Sprintf("%d", nodeCount), func(b *testing.B) {
			benchmarkLayout(b, defaultAvailableSpace(), func(t *TaffyTree, rng *rand.Rand) tree.NodeId {
				children := buildFlatHierarchy(t, nodeCount, randomStyleGen{}, rng)
				return t.NewWithChildren(style.DefaultStyle(), children)
			})
		})
	}
}
```

- [ ] **Step 3: Add DeepRandom and DeepAuto benchmarks**

Append to `bench_flexbox_test.go`:

```go
func benchFlexboxDeepRandom(b *testing.B) {
	for _, nodeCount := range []uint32{4_000, 10_000} {
		b.Run(fmt.Sprintf("%d", nodeCount), func(b *testing.B) {
			benchmarkLayout(b, defaultAvailableSpace(), func(t *TaffyTree, rng *rand.Rand) tree.NodeId {
				children := buildDeepTree(t, nodeCount, 2, randomStyleGen{}, rng)
				return t.NewWithChildren(style.DefaultStyle(), children)
			})
		})
	}
}

func benchFlexboxDeepAuto(b *testing.B) {
	deepAutoStyle := style.DefaultStyle()
	deepAutoStyle.FlexGrow = 1.0
	deepAutoStyle.Margin = geometry.Rect[style.LengthPercentageAuto]{
		Left:   style.LengthLPA(10),
		Right:  style.LengthLPA(10),
		Top:    style.LengthLPA(10),
		Bottom: style.LengthLPA(10),
	}
	gen := fixedStyleGen{style: deepAutoStyle}

	for _, nodeCount := range []uint32{4_000, 10_000} {
		b.Run(fmt.Sprintf("%d", nodeCount), func(b *testing.B) {
			benchmarkLayout(b, defaultAvailableSpace(), func(t *TaffyTree, rng *rand.Rand) tree.NodeId {
				children := buildDeepTree(t, nodeCount, 2, gen, rng)
				return t.NewWithChildren(style.DefaultStyle(), children)
			})
		})
	}
}
```

- [ ] **Step 4: Add SuperDeep benchmark**

Append to `bench_flexbox_test.go`:

```go
func benchFlexboxSuperDeep(b *testing.B) {
	superDeepStyle := style.DefaultStyle()
	superDeepStyle.FlexDirection = style.FlexDirectionRow
	superDeepStyle.FlexGrow = 1.0
	superDeepStyle.Margin = geometry.Rect[style.LengthPercentageAuto]{
		Left:   style.LengthLPA(10),
		Right:  style.LengthLPA(10),
		Top:    style.LengthLPA(10),
		Bottom: style.LengthLPA(10),
	}
	gen := fixedStyleGen{style: superDeepStyle}

	for _, depth := range []uint32{100} {
		b.Run(fmt.Sprintf("%d", depth), func(b *testing.B) {
			benchmarkLayout(b, defaultAvailableSpace(), func(t *TaffyTree, rng *rand.Rand) tree.NodeId {
				children := buildSuperDeepHierarchy(t, depth, 3, gen, rng)
				return t.NewWithChildren(style.DefaultStyle(), children)
			})
		})
	}
}
```

- [ ] **Step 5: Run the flexbox benchmarks to verify they work**

Run: `go test -bench=BenchmarkFlexbox -benchmem -count=1 -timeout=10m ./...`
Expected: All benchmarks run and produce output like:
```
BenchmarkFlexbox/HugeNested/10000-N    XX    XXXXXXX ns/op    XXXXX B/op    XXXX allocs/op
...
```

- [ ] **Step 6: Commit**

```bash
git add bench_flexbox_test.go
git commit -m "bench: add flexbox benchmarks (huge nested, wide, deep, super deep)"
```

---

### Task 3: Grid benchmarks

**Files:**
- Create: `bench_grid_test.go`

- [ ] **Step 1: Create grid benchmark file with helpers**

Create `bench_grid_test.go` with grid-specific helper functions:

```go
package taffy

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"taffy-go/geometry"
	"taffy-go/style"
	"taffy-go/tree"
)

// randomGridTrack returns a random grid track sizing function.
// Distribution: 10% auto, 10% min-content, 10% max-content, 20% fr(1),
// 10% minmax(0, fr(1)), 20% length(40), 20% percent(0.3).
func randomGridTrack(rng *rand.Rand) style.GridTemplateComponent {
	r := rng.Float32()
	var tsf style.TrackSizingFunction
	switch {
	case r < 0.1:
		tsf = style.TrackSizingAuto
	case r < 0.2:
		tsf = style.TrackSizingFunction{Min: style.MinTrackMinContentVal, Max: style.MaxTrackMinContentVal}
	case r < 0.3:
		tsf = style.TrackSizingFunction{Min: style.MinTrackMaxContentVal, Max: style.MaxTrackMaxContentVal}
	case r < 0.5:
		tsf = style.GridFr(1.0)
	case r < 0.6:
		tsf = style.GridMinMax(
			style.MinTrackSizingFunction{Tag: style.MinTrackLength, Value: 0},
			style.MaxTrackFromFr(1.0),
		)
	case r < 0.8:
		tsf = style.TrackSizingFunction{
			Min: style.MinTrackSizingFunction{Tag: style.MinTrackLength, Value: 40},
			Max: style.MaxTrackFromLength(40),
		}
	default:
		tsf = style.TrackSizingFunction{
			Min: style.MinTrackSizingFunction{Tag: style.MinTrackPercent, Value: 0.3},
			Max: style.MaxTrackFromPercent(0.3),
		}
	}
	return style.GridTemplateSingleVal(tsf)
}

// randomNxNGridStyle creates a grid style with trackCount random columns and rows.
func randomNxNGridStyle(rng *rand.Rand, trackCount int) style.Style {
	s := style.DefaultStyle()
	s.Display = style.DisplayGrid
	cols := make([]style.GridTemplateComponent, trackCount)
	rows := make([]style.GridTemplateComponent, trackCount)
	for i := 0; i < trackCount; i++ {
		cols[i] = randomGridTrack(rng)
		rows[i] = randomGridTrack(rng)
	}
	s.GridTemplateColumns = cols
	s.GridTemplateRows = rows
	return s
}

// buildGridLeaf creates a leaf node with size 20x20.
func buildGridLeaf(t *TaffyTree) tree.NodeId {
	s := style.DefaultStyle()
	s.Size = geometry.Size[style.Dimension]{
		Width:  style.LengthDimension(20),
		Height: style.LengthDimension(20),
	}
	return t.NewLeaf(s)
}

// buildGridFlatHierarchy creates a flat colCount x rowCount grid with leaf children.
func buildGridFlatHierarchy(colCount, rowCount int) (*TaffyTree, tree.NodeId) {
	t := New()
	rng := newBenchRNG()

	s := style.DefaultStyle()
	s.Display = style.DisplayGrid
	cols := make([]style.GridTemplateComponent, colCount)
	rows := make([]style.GridTemplateComponent, rowCount)
	for i := 0; i < colCount; i++ {
		cols[i] = randomGridTrack(rng)
	}
	for i := 0; i < rowCount; i++ {
		rows[i] = randomGridTrack(rng)
	}
	s.GridTemplateColumns = cols
	s.GridTemplateRows = rows

	children := make([]tree.NodeId, colCount*rowCount)
	for i := range children {
		children[i] = buildGridLeaf(t)
	}
	root := t.NewWithChildren(s, children)
	return t, root
}

// buildDeepGridTree recursively constructs nested grids.
func buildDeepGridTree(t *TaffyTree, levels, trackCount int, rng *rand.Rand) []tree.NodeId {
	childCount := trackCount * trackCount
	if levels == 1 {
		nodes := make([]tree.NodeId, childCount)
		for i := range nodes {
			nodes[i] = buildGridLeaf(t)
		}
		return nodes
	}
	nodes := make([]tree.NodeId, childCount)
	for i := range nodes {
		subChildren := buildDeepGridTree(t, levels-1, trackCount, rng)
		nodes[i] = t.NewWithChildren(randomNxNGridStyle(rng, trackCount), subChildren)
	}
	return nodes
}

// buildDeepGridHierarchy creates a deep grid tree and returns (tree, root).
func buildDeepGridHierarchy(levels, trackCount int) (*TaffyTree, tree.NodeId) {
	t := New()
	rng := newBenchRNG()
	children := buildDeepGridTree(t, levels, trackCount, rng)
	root := t.NewWithChildren(style.DefaultStyle(), children)
	return t, root
}
```

- [ ] **Step 2: Add Wide grid benchmarks**

Append to `bench_grid_test.go`:

```go
func BenchmarkGrid(b *testing.B) {
	b.Run("Wide", benchGridWide)
	b.Run("Deep", benchGridDeep)
	b.Run("SuperDeep", benchGridSuperDeep)
}

func benchGridWide(b *testing.B) {
	availableSpace := lengthAvailableSpace(12000, 12000)
	for _, trackCount := range []int{31, 100} {
		b.Run(fmt.Sprintf("%dx%d", trackCount, trackCount), func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()
				t, root := buildGridFlatHierarchy(trackCount, trackCount)
				b.StartTimer()
				t.ComputeLayout(root, availableSpace)
			}
		})
	}
}
```

- [ ] **Step 3: Add Deep and SuperDeep grid benchmarks**

Append to `bench_grid_test.go`:

```go
func benchGridDeep(b *testing.B) {
	availableSpace := lengthAvailableSpace(12000, 12000)
	benches := []struct {
		tracks int
		levels int
	}{
		{2, 5},
		{3, 4},
		{2, 7},
	}
	for _, bench := range benches {
		name := fmt.Sprintf("%dx%d_levels%d", bench.tracks, bench.tracks, bench.levels)
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()
				t, root := buildDeepGridHierarchy(bench.levels, bench.tracks)
				b.StartTimer()
				t.ComputeLayout(root, availableSpace)
			}
		})
	}
}

func benchGridSuperDeep(b *testing.B) {
	availableSpace := defaultAvailableSpace()
	for _, levels := range []int{100, 1000} {
		b.Run(fmt.Sprintf("%d", levels), func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()
				t, root := buildDeepGridHierarchy(levels, 1)
				b.StartTimer()
				t.ComputeLayout(root, availableSpace)
			}
		})
	}
}
```

- [ ] **Step 4: Run the grid benchmarks to verify they work**

Run: `go test -bench=BenchmarkGrid -benchmem -count=1 -timeout=10m ./...`
Expected: All benchmarks run and produce timing output. The 100x100 and deep grid benchmarks may take several seconds each.

- [ ] **Step 5: Commit**

```bash
git add bench_grid_test.go
git commit -m "bench: add grid benchmarks (wide, deep, super deep)"
```

---

### Task 4: Tree creation benchmarks

**Files:**
- Create: `bench_tree_creation_test.go`

- [ ] **Step 1: Create tree creation benchmark file**

Create `bench_tree_creation_test.go`:

```go
package taffy

import (
	"fmt"
	"testing"

	"taffy-go/style"
)

func BenchmarkTreeCreation(b *testing.B) {
	for _, nodeCount := range []uint32{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("%d", nodeCount), func(b *testing.B) {
			for b.Loop() {
				rng := newBenchRNG()
				t := New()
				gen := fixedStyleGen{style: style.DefaultStyle()}
				children := buildFlatHierarchy(t, nodeCount, gen, rng)
				root := t.NewWithChildren(style.DefaultStyle(), children)
				_ = root
			}
		})
	}
}
```

- [ ] **Step 2: Run tree creation benchmarks to verify**

Run: `go test -bench=BenchmarkTreeCreation -benchmem -count=1 ./...`
Expected: All three sub-benchmarks produce output.

- [ ] **Step 3: Commit**

```bash
git add bench_tree_creation_test.go
git commit -m "bench: add tree creation benchmarks"
```

---

### Task 5: Run full benchmark suite and verify

**Files:** None (verification only)

- [ ] **Step 1: Run all benchmarks together**

Run: `go test -bench=. -benchmem -count=1 -timeout=30m ./...`
Expected: All benchmarks from all three files run successfully. Output includes:
- `BenchmarkFlexbox/HugeNested/10000`
- `BenchmarkFlexbox/Wide/10000`
- `BenchmarkFlexbox/DeepRandom/4000`
- `BenchmarkFlexbox/DeepRandom/10000`
- `BenchmarkFlexbox/DeepAuto/4000`
- `BenchmarkFlexbox/DeepAuto/10000`
- `BenchmarkFlexbox/SuperDeep/100`
- `BenchmarkGrid/Wide/31x31`
- `BenchmarkGrid/Wide/100x100`
- `BenchmarkGrid/Deep/2x2_levels5`
- `BenchmarkGrid/Deep/3x3_levels4`
- `BenchmarkGrid/Deep/2x2_levels7`
- `BenchmarkGrid/SuperDeep/100`
- `BenchmarkGrid/SuperDeep/1000`
- `BenchmarkTreeCreation/1000`
- `BenchmarkTreeCreation/10000`
- `BenchmarkTreeCreation/100000`

- [ ] **Step 2: Save baseline benchmark results**

Run: `go test -bench=. -benchmem -count=5 -timeout=30m ./... | tee bench_results.txt`

This saves the first baseline for future `benchstat` comparisons.

- [ ] **Step 3: Run Rust benchmarks for comparison**

Run: `cd /Users/wizard/workspace/gamedev/taffy && cargo bench 2>&1 | tee bench_results_rust.txt`

This produces Criterion output for the equivalent Rust benchmarks.

- [ ] **Step 4: Commit baseline results**

```bash
git add bench_results.txt
git commit -m "bench: save initial Go benchmark baseline results"
```
