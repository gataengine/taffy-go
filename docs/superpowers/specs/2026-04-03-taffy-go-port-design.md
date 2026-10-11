# Taffy Go Port — Design Spec

> **Status (audit 2026-10-11): Implemented.**

## Overview

Port the [Taffy](https://github.com/DioxusLabs/taffy) Rust layout engine (v0.9.2) to Go with full feature parity across all four layout algorithms. The port follows a file-by-file, bottom-up approach.

**Source:** `../taffy` (Rust, ~21K lines across 51 source files)
**Target:** `taffy-go` (Go 1.25, single flat package)

## Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Layout algorithms | All four (flexbox, grid, block, float) | Full parity |
| API level | High-level only (TaffyTree) | Low-level interfaces kept as unexported internals to ease porting |
| Package structure | Single flat `taffy` package | Avoids circular imports, mirrors Rust's single-crate model, reorganize later |
| Node storage | `map[NodeId]*NodeData` behind unexported interface | Correctness over performance |
| `Style<CustomIdent>` generic | Hardcoded to `string` | Covers 99% of use cases, avoids generic proliferation |
| Test framework | Ginkgo/Gomega | User preference |
| Porting strategy | Bottom-up file-by-file (Approach A) | Natural dependency order, no stubs needed |

## Skipped Functionality

- **serde** — no serialization/deserialization
- **parse** — no CSS string parsing (users construct Style structs directly)
- **debug/profile** — no debug logging or profiling hooks
- **no_std/alloc** — not applicable in Go
- **CompactLength** — use plain Dimension/LengthPercentage structs

## Kept Non-Obvious Functionality

- **PrintTree** — useful for debugging
- **calc()** — CSS calc expressions (user-facing)

## File Mapping & Port Order

### Layer 1 — Geometry & Foundations

| Rust Source | Go File | ~Lines |
|---|---|---|
| `geometry.rs` | `geometry.go` | 730 |
| `style_helpers.rs` | `style_helpers.go` | 577 |

### Layer 2 — Style System

| Rust Source | Go File | ~Lines |
|---|---|---|
| `style/alignment.rs` | `style_alignment.go` | 136 |
| `style/dimension.rs` | `style_dimension.go` | 380 |
| `style/available_space.rs` | `style_available_space.go` | 163 |
| `style/flex.rs` | `style_flex.go` | 203 |
| `style/float.rs` | `style_float.go` | 80 |
| `style/block.rs` | `style_block.go` | 57 |
| `style/grid.rs` | `style_grid.go` | 1607 |
| `style/mod.rs` | `style.go` | 1338 |

### Layer 3 — Tree & Util

| Rust Source | Go File | ~Lines |
|---|---|---|
| `tree/node.rs` | `node.go` | 60 |
| `tree/layout.rs` | `layout.go` | 357 |
| `tree/cache.rs` | `cache.go` | 196 |
| `tree/traits.rs` | `tree_traits.go` | 418 |
| `tree/taffy_tree.rs` | `taffy_tree.go` | 1405 |
| `util/sys.rs` | `util_sys.go` | 301 |
| `util/math.rs` | `util_math.go` | 356 |
| `util/resolve.rs` | `util_resolve.go` | 351 |
| `util/print.rs` | `util_print.go` | 78 |

### Layer 4 — Compute Algorithms

| Rust Source | Go File | ~Lines |
|---|---|---|
| `compute/mod.rs` | `compute.go` | 337 |
| `compute/leaf.rs` | `compute_leaf.go` | 164 |
| `compute/common/alignment.rs` | `compute_alignment.go` | 100 |
| `compute/flexbox.rs` | `compute_flexbox.go` | 2517 |
| `compute/block.rs` | `compute_block.go` | 1311 |
| `compute/float.rs` | `compute_float.go` | 552 |
| `compute/grid/mod.rs` | `compute_grid.go` | 846 |
| `compute/grid/track_sizing.rs` | `compute_grid_track_sizing.go` | 1410 |
| `compute/grid/placement.rs` | `compute_grid_placement.go` | 764 |
| `compute/grid/explicit_grid.rs` | `compute_grid_explicit.go` | 697 |
| `compute/grid/implicit_grid.rs` | `compute_grid_implicit.go` | 253 |
| `compute/grid/alignment.rs` | `compute_grid_alignment.go` | 366 |
| `compute/grid/types/grid_track.rs` | `compute_grid_types_track.go` | 156 |
| `compute/grid/types/grid_item.rs` | `compute_grid_types_item.go` | 555 |
| `compute/grid/types/cell_occupancy.rs` | `compute_grid_types_occupancy.go` | 301 |
| `compute/grid/types/coordinates.rs` | `compute_grid_types_coords.go` | 150 |
| `compute/grid/types/named.rs` | `compute_grid_types_named.go` | 428 |

### Layer 5 — Tests

| Source | Go File | Notes |
|---|---|---|
| In-file `#[cfg(test)]` (~11 files) | `*_test.go` next to source | Ginkgo Describe blocks |
| `test_fixtures/flex/` (565 files) | `fixtures_flex_test.go` | XML fixture subtests |
| `test_fixtures/grid/` (274 files) | `fixtures_grid_test.go` | XML fixture subtests |
| `test_fixtures/block/` (207 files) | `fixtures_block_test.go` | XML fixture subtests |
| `test_fixtures/blockflex/` | `fixtures_blockflex_test.go` | XML fixture subtests |
| `test_fixtures/blockgrid/` | `fixtures_blockgrid_test.go` | XML fixture subtests |
| `test_fixtures/gridflex/` | `fixtures_gridflex_test.go` | XML fixture subtests |
| `test_fixtures/leaf/` | `fixtures_leaf_test.go` | XML fixture subtests |
| `test_fixtures/float/` | `fixtures_float_test.go` | XML fixture subtests |
| Shared helpers | `fixtures_helpers_test.go` | XML parsing, tree building, assertion helpers |

## Type System Translation

### Enums with Data

Rust enums with associated data become Go structs with a tag:

```go
type Dimension struct {
    Tag   DimensionTag
    Value float32
}
type DimensionTag uint8
const (
    DimensionAuto DimensionTag = iota
    DimensionLength
    DimensionPercent
)
```

### Generic Geometry

Go generics for geometry types:

```go
type Point[T any] struct { X, Y T }
type Size[T any] struct { Width, Height T }
type Rect[T any] struct { Left, Right, Top, Bottom T }
type Line[T any] struct { Start, End T }
```

### Option Types

- Go pointer `*T` for optional values in function parameters and return types
- Custom `Option[T]` struct for optional values stored in structs (avoids pointer indirection in hot paths):

```go
type Option[T any] struct {
    Value T
    IsSome bool
}
```

Used for fields like `Style.Size` where `None` means "auto" but `0` is a valid explicit value.

### Other Mappings

| Rust | Go |
|---|---|
| `f32` | `float32` |
| `NodeId` (u64 newtype) | `type NodeId uint64` |
| `Result<T, E>` | `(T, error)` |
| `panic!` | `panic()` for programmer errors |
| `Option<T>` | `*T` or `Option[T]` |
| `String` (CustomIdent) | `string` |
| `Vec<T>` | `[]T` |
| `HashMap<K, V>` | `map[K]V` |
| `#[inline]` | omitted (compiler decides) |
| `cfg` feature gates | all code included, no build tags |
| `derive(Clone, Debug)` | implement manually where needed |

## Node Storage

Unexported interface with map-based implementation:

```go
type nodeStore interface {
    getNode(NodeId) *nodeData
    addNode(nodeData) NodeId
    removeNode(NodeId)
    // etc.
}

type mapNodeStore struct {
    nodes  map[NodeId]*nodeData
    nextId uint64
}
```

## Public API

```go
// Core tree
type TaffyTree struct { /* unexported fields */ }

func New() *TaffyTree
func (t *TaffyTree) NewLeaf(style Style) NodeId
func (t *TaffyTree) NewLeafWithContext(style Style, context NodeContext) NodeId
func (t *TaffyTree) NewWithChildren(style Style, children []NodeId) NodeId
func (t *TaffyTree) SetStyle(node NodeId, style Style)
func (t *TaffyTree) GetStyle(node NodeId) *Style
func (t *TaffyTree) SetNodeContext(node NodeId, context *NodeContext)
func (t *TaffyTree) GetNodeContext(node NodeId) *NodeContext
func (t *TaffyTree) AddChild(parent NodeId, child NodeId)
func (t *TaffyTree) RemoveChild(parent NodeId, child NodeId) NodeId
func (t *TaffyTree) Children(node NodeId) []NodeId
func (t *TaffyTree) SetChildren(parent NodeId, children []NodeId)
func (t *TaffyTree) ComputeLayout(node NodeId, availableSpace Size[AvailableSpace])
func (t *TaffyTree) Layout(node NodeId) *Layout
func (t *TaffyTree) MarkDirty(node NodeId)
func (t *TaffyTree) PrintTree(node NodeId)

// Measure function type
type NodeContext = func(
    knownDimensions Size[*float32],
    parentSize Size[*float32],
    availableSpace Size[AvailableSpace],
    axis SizingMode,
) Size[float32]

// Node identifier
type NodeId uint64
```

## Unexported Interfaces (Internal)

Ported from Rust traits, used by compute functions:

```go
type layoutPartialTree interface {
    childCount(node NodeId) uint
    getChildId(node NodeId, index uint) NodeId
    getStyle(node NodeId) *Style
    setUnroundedLayout(node NodeId, layout Layout)
    getCacheEntry(node NodeId, index uint) *CacheEntry
    computeChildLayout(node NodeId, inputs LayoutInput) LayoutOutput
    // etc.
}

type traversePartialTree interface {
    childCount(node NodeId) uint
    getChildId(node NodeId, index uint) NodeId
}

type traverseTree interface {
    traversePartialTree
    childIds(node NodeId) []NodeId
}

type roundTree interface {
    traversePartialTree
    getUnroundedLayout(node NodeId) Layout
    setFinalLayout(node NodeId, layout Layout)
}

type printTree interface {
    traverseTree
    getFinalLayout(node NodeId) Layout
}
```

`TaffyTree` implements all of these.

## Testing Strategy

### In-File Unit Tests

Ported from Rust `#[cfg(test)]` modules into `*_test.go` files using Ginkgo:

```go
// util_math_test.go
var _ = Describe("Math Utils", func() {
    It("computes maybeMin correctly", func() {
        Expect(maybeMin(3.0, ptrFloat32(5.0))).To(Equal(float32(3.0)))
    })
})
```

### XML Fixture Tests

One test file per fixture directory. Each file loads fixtures from `test_fixtures/<dir>/` and runs them as parametrized Ginkgo entries across LTR/RTL and border-box/content-box variants.

Shared helpers in `fixtures_helpers_test.go`:
- `loadFixtures(dir string)` — parse XML files
- `buildAndComputeTree(fixture, direction, boxSizing)` — construct tree and run layout
- `expectLayoutMatches(actual, expected)` — recursive layout comparison

Fixture files copied from `../taffy/test_fixtures/` into the Go project.

### Test Dependencies

- `github.com/onsi/ginkgo/v2`
- `github.com/onsi/gomega`
- `encoding/xml` (stdlib) for fixture parsing

## Rust-Specific Translations

| Rust Pattern | Go Translation |
|---|---|
| `impl Struct { fn method(&self) }` | Method on struct `func (s *Struct) Method()` |
| `impl Trait for Struct` | Struct satisfies interface implicitly |
| `match` on enum | `switch` on tag field |
| `unwrap()` / `expect()` | Direct access or panic with message |
| Closures `\|x\| x + 1` | `func(x int) int { return x + 1 }` |
| Iterator chains `.iter().map().filter().collect()` | `for` loops or `slices` package functions |
| `#[derive(Default)]` | Zero values or explicit `Default()` constructors |
| Ownership/borrowing | Not applicable — Go has GC |
| Lifetime annotations | Not applicable |
