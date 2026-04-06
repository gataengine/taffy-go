# Taffy3-Go Structural Reorganization

## Goal

Reorganize the flat single-package `taffy` codebase into sub-packages with clear boundaries, and make the "trait" interfaces public so users can implement custom tree backends.

## Decisions

- **Custom tree backends** are the primary use case for public interfaces (option A)
- **Sub-package split** (option B → refined to option 1: full sub-package split)
- **Style interfaces over concrete struct** — algorithms work through interfaces (`CoreStyle`, `FlexboxContainerStyle`, etc.), not `*Style` directly
- **Monolithic default `Style` struct** — matches Rust; custom backends can optimize memory layout via interfaces
- **Public compute functions** — custom backends call `flexbox.ComputeFlexboxLayout(...)` etc. directly
- **Dispatch in root package** — `ComputeRootLayout` lives in root `taffy` package, not `compute/`

## Package Structure

```
taffy/                          # root — TaffyTree concrete impl + dispatch
├── geometry/                   # primitives: Option, Rect, Size, Point, Line, MinMax, axes
│   ├── geometry.go
│   └── math.go                 # Option/Size/Rect/float32 math (no style deps)
├── style/                      # Style struct, enums, dimension types, style interfaces
│   ├── style.go                # Style struct + CoreStyle interface + getter methods
│   ├── dimension.go            # LengthPercentage, LengthPercentageAuto, Dimension
│   ├── alignment.go            # AlignItems, AlignContent, JustifyItems, JustifyContent
│   ├── flex.go                 # FlexDirection, FlexWrap + FlexboxContainerStyle, FlexboxItemStyle
│   ├── grid.go                 # Grid types + GridContainerStyle, GridItemStyle
│   ├── block.go                # TextAlign + BlockContainerStyle, BlockItemStyle
│   ├── float.go                # Float, Clear enums
│   ├── available_space.go      # AvailableSpace type + AvailableSpace math helpers
│   ├── helpers.go              # constructor convenience functions
│   ├── resolve.go              # resolve LengthPercentage/Dimension against context (was util_resolve.go + resolve helpers from compute_leaf.go)
│   └── math.go                 # AvailableSpace and Size[AvailableSpace] math ops (split from util_math.go)
├── tree/                       # trait interfaces + layout data types (NO TaffyTree)
│   ├── traits.go               # public interfaces: LayoutPartialTree, TraversePartialTree, etc.
│   ├── layout.go               # Layout, LayoutInput, LayoutOutput, RunMode, SizingMode
│   ├── node.go                 # NodeId
│   └── cache.go                # Cache type
├── compute/                    # shared compute helpers (no dispatch)
│   ├── compute.go              # ComputeCachedLayout, RoundLayout, ComputeHiddenLayout
│   ├── leaf.go                 # ComputeLeafLayout
│   ├── alignment.go            # alignment offset helpers
│   ├── flexbox/
│   │   └── flexbox.go          # ComputeFlexboxLayout
│   ├── grid/
│   │   ├── grid.go             # ComputeGridLayout
│   │   ├── track_sizing.go
│   │   ├── placement.go
│   │   ├── alignment.go
│   │   ├── explicit.go
│   │   ├── implicit.go
│   │   ├── types_coords.go
│   │   ├── types_item.go
│   │   ├── types_named.go
│   │   ├── types_occupancy.go
│   │   └── types_track.go
│   └── block/
│       ├── block.go            # ComputeBlockLayout
│       └── float.go            # float context (only used by block)
├── taffy.go                    # TaffyTree, New(), public API methods
├── taffy_view.go               # taffyView adapter, interface impls, dispatch
└── print.go                    # PrintTree utility
```

## Dependency DAG

```
geometry  ←  style  ←  tree  ←  compute/*  ←  taffy (root)
```

No circular dependencies. `TaffyTree` in root `taffy` package is the only place that imports all sub-packages.

## Public Interfaces

### tree/traits.go

```go
package tree

type TraversePartialTree interface {
    ChildIds(parent NodeId) []NodeId
    ChildCount(parent NodeId) int
    GetChildId(parent NodeId, index int) NodeId
}

type TraverseTree interface {
    TraversePartialTree
}

type LayoutPartialTree interface {
    TraversePartialTree
    GetCoreContainerStyle(node NodeId) style.CoreStyle
    SetUnroundedLayout(node NodeId, layout *Layout)
    ComputeChildLayout(node NodeId, inputs LayoutInput) LayoutOutput
    ResolveCalcValue(val float32, basis float32) float32
}

type LayoutFlexboxContainer interface {
    LayoutPartialTree
    GetFlexboxContainerStyle(node NodeId) style.FlexboxContainerStyle
    GetFlexboxChildStyle(node NodeId) style.FlexboxItemStyle
}

type LayoutGridContainer interface {
    LayoutPartialTree
    GetGridContainerStyle(node NodeId) style.GridContainerStyle
    GetGridChildStyle(node NodeId) style.GridItemStyle
}

type LayoutBlockContainer interface {
    LayoutPartialTree
    GetBlockContainerStyle(node NodeId) style.BlockContainerStyle
    GetBlockChildStyle(node NodeId) style.BlockItemStyle
    ComputeBlockChildLayout(node NodeId, inputs LayoutInput) LayoutOutput
}

type CacheTree interface {
    CacheGet(node NodeId, inputs LayoutInput) LayoutOutput
    CacheStore(node NodeId, inputs LayoutInput, output LayoutOutput)
    CacheClear(node NodeId)
}

type RoundTree interface {
    TraverseTree
    GetUnroundedLayout(node NodeId) *Layout
    SetFinalLayout(node NodeId, layout *Layout)
}

type PrintTree interface {
    TraverseTree
    GetDebugLabel(node NodeId) string
    GetFinalLayout(node NodeId) *Layout
}
```

### style/ interfaces

```go
type CoreStyle interface {
    GetBoxGenerationMode() BoxGenerationMode
    GetBoxSizing() BoxSizing
    GetDirection() Direction
    GetOverflow() geometry.Size[Overflow]
    GetScrollbarWidth() float32
    GetPosition() Position
    GetInset() geometry.Rect[LengthPercentageAuto]
    GetSize() geometry.Size[Dimension]
    GetMinSize() geometry.Size[Dimension]
    GetMaxSize() geometry.Size[Dimension]
    GetAspectRatio() geometry.Option[float32]
    GetMargin() geometry.Rect[LengthPercentageAuto]
    GetPadding() geometry.Rect[LengthPercentage]
    GetBorder() geometry.Rect[LengthPercentage]
}

type FlexboxContainerStyle interface {
    CoreStyle
    GetFlexDirection() FlexDirection
    GetFlexWrap() FlexWrap
    GetGap() geometry.Size[LengthPercentage]
    GetAlignContent() geometry.Option[AlignContent]
    GetAlignItems() geometry.Option[AlignItems]
    GetJustifyContent() geometry.Option[JustifyContent]
}

type FlexboxItemStyle interface {
    GetFlexGrow() float32
    GetFlexShrink() float32
    GetFlexBasis() Dimension
    GetAlignSelf() geometry.Option[AlignSelf]
    GetJustifySelf() geometry.Option[JustifySelf]
}

type BlockContainerStyle interface {
    CoreStyle
    GetTextAlign() TextAlign
}

type BlockItemStyle interface {
    GetFloat() Float
    GetClear() Clear
}

type GridContainerStyle interface {
    CoreStyle
    GetGridTemplateRows() []GridTemplateComponent
    GetGridTemplateColumns() []GridTemplateComponent
    GetGridTemplateAreas() []GridTemplateArea
    GetGridAutoRows() []TrackSizingFunction
    GetGridAutoColumns() []TrackSizingFunction
    GetGridAutoFlow() GridAutoFlow
    GetGap() geometry.Size[LengthPercentage]
    GetAlignContent() geometry.Option[AlignContent]
    GetAlignItems() geometry.Option[AlignItems]
    GetJustifyContent() geometry.Option[JustifyContent]
}

type GridItemStyle interface {
    GetGridRow() geometry.Line[GridPlacement]
    GetGridColumn() geometry.Line[GridPlacement]
    GetAlignSelf() geometry.Option[AlignSelf]
    GetJustifySelf() geometry.Option[JustifySelf]
}
```

The concrete `Style` struct implements all interfaces via getter methods.

## Public Compute Functions

```go
// compute/compute.go
func ComputeCachedLayout(t tree.CacheTree, node tree.NodeId, inputs tree.LayoutInput, fn func() tree.LayoutOutput) tree.LayoutOutput
func ComputeHiddenLayout(t LayoutAndCacheTree, node tree.NodeId)
func RoundLayout(t tree.RoundTree, node tree.NodeId)

// compute/leaf.go
func ComputeLeafLayout(inputs tree.LayoutInput, style style.CoreStyle, measureFn MeasureFunc) tree.LayoutOutput

// compute/flexbox/flexbox.go
func ComputeFlexboxLayout(t tree.LayoutFlexboxContainer, node tree.NodeId, inputs tree.LayoutInput) tree.LayoutOutput

// compute/block/block.go
func ComputeBlockLayout(t tree.LayoutBlockContainer, node tree.NodeId, inputs tree.LayoutInput) tree.LayoutOutput

// compute/grid/grid.go
func ComputeGridLayout(t tree.LayoutGridContainer, node tree.NodeId, inputs tree.LayoutInput) tree.LayoutOutput
```

## Dispatch in Root Package

`TaffyTree` in the root `taffy` package owns dispatch logic:

```go
func (v *taffyView) computeChildLayout(node tree.NodeId, inputs tree.LayoutInput) tree.LayoutOutput {
    display := v.getCoreContainerStyle(node).GetBoxGenerationMode()
    switch {
    case display == style.DisplayFlex:
        return flexbox.ComputeFlexboxLayout(v, node, inputs)
    case display == style.DisplayGrid:
        return grid.ComputeGridLayout(v, node, inputs)
    case display == style.DisplayBlock:
        return block.ComputeBlockLayout(v, node, inputs)
    default:
        return compute.ComputeLeafLayout(...)
    }
}
```

Custom backends write their own dispatch, calling algorithm packages directly.

## File Migration Map

| Current file | Target |
|---|---|
| `geometry.go` | `geometry/geometry.go` |
| `util_math.go` (Option/Size/Rect/float32 ops) | `geometry/math.go` |
| `util_math.go` (AvailableSpace ops) | `style/math.go` |
| `util_resolve.go` | `style/resolve.go` |
| `style.go` | `style/style.go` |
| `style_dimension.go` | `style/dimension.go` |
| `style_alignment.go` | `style/alignment.go` |
| `style_flex.go` | `style/flex.go` |
| `style_grid.go` | `style/grid.go` |
| `style_block.go` | `style/block.go` |
| `style_float.go` | `style/float.go` |
| `style_available_space.go` | `style/available_space.go` |
| `style_helpers.go` | `style/helpers.go` |
| `node.go` | `tree/node.go` |
| `layout.go` | `tree/layout.go` |
| `cache.go` | `tree/cache.go` |
| `tree_traits.go` | `tree/traits.go` (make public) |
| `compute.go` (minus dispatch) | `compute/compute.go` |
| `compute_leaf.go` | `compute/leaf.go` |
| `compute_alignment.go` | `compute/alignment.go` |
| `compute_flexbox.go` | `compute/flexbox/flexbox.go` |
| `compute_block.go` | `compute/block/block.go` |
| `compute_float.go` | `compute/block/float.go` |
| `compute_grid.go` | `compute/grid/grid.go` |
| `compute_grid_track_sizing.go` | `compute/grid/track_sizing.go` |
| `compute_grid_placement.go` | `compute/grid/placement.go` |
| `compute_grid_alignment.go` | `compute/grid/alignment.go` |
| `compute_grid_explicit.go` | `compute/grid/explicit.go` |
| `compute_grid_implicit.go` | `compute/grid/implicit.go` |
| `compute_grid_types_coords.go` | `compute/grid/types_coords.go` |
| `compute_grid_types_item.go` | `compute/grid/types_item.go` |
| `compute_grid_types_named.go` | `compute/grid/types_named.go` |
| `compute_grid_types_occupancy.go` | `compute/grid/types_occupancy.go` |
| `compute_grid_types_track.go` | `compute/grid/types_track.go` |
| `taffy_tree.go` | `taffy.go` + `taffy_view.go` (root) |
| `util_print.go` | `print.go` (root) |
| `util_sys.go` | `geometry/sys.go` (if still needed) |

## Migration Challenges

### 1. resolve.go depends on style types
`util_resolve.go` resolves `LengthPercentage`, `Dimension`, etc. — moves to `style/resolve.go`.

### 2. math.go split
`util_math.go` splits: pure geometry math → `geometry/math.go`, AvailableSpace math → `style/math.go`.

### 3. compute_leaf helpers used by compute.go
`resolveRectLPWidth` / `resolveRectLPAWidth` from `compute_leaf.go` are also used in root layout computation. Move to `style/resolve.go` as public functions.

### 4. measureChildSize / performChildLayout
Convenience functions in `tree_traits.go` — stay in `tree/` as public utility functions.

### 5. Tests
- `geometry_test.go` → `geometry/`
- `style_test.go`, `style_flex_test.go` → `style/`
- `util_math_test.go` → split across `geometry/` and `style/`
- `util_resolve_test.go` → `style/`
- `taffy_tree_test.go` → root `taffy/`
- `fixtures_*` + `fixtures_helpers_test.go` → root `taffy/` (integration tests)

## User-Facing API Impact

For users of `TaffyTree` (high-level API): imports change from `taffy.Style` to `style.Style`, `taffy.NodeId` to `tree.NodeId`, etc. The methods on `TaffyTree` remain the same.

For custom backend implementers (new capability): import `tree` for interfaces, `compute/*` for algorithms, `style` for types, and implement the relevant interfaces.
