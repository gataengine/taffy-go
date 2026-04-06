# Structural Reorganization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the flat single-package `taffy` codebase into `geometry/`, `style/`, `tree/`, `compute/` sub-packages with public trait interfaces for custom tree backends.

**Architecture:** Bottom-up migration following the dependency DAG: `geometry ← style ← tree ← compute/* ← taffy (root)`. Each task creates one package, makes it compile, and commits. Tests migrate last since they exercise the full integration and belong in the root package.

**Tech Stack:** Go 1.25, no new dependencies

**Module path:** `taffy-go` (from go.mod)

---

### Task 1: Create `geometry` Package

**Files:**
- Create: `geometry/geometry.go`
- Create: `geometry/math.go`
- Create: `geometry/sys.go`
- Source: `geometry.go`, `util_math.go` (partial), `util_sys.go`

This is the leaf package — no imports from other taffy packages. Contains all generic container types and pure math operations on `Option[T]`, `Size[T]`, `Rect[T]`, `Point[T]`.

- [ ] **Step 1: Create `geometry/geometry.go`**

Copy the entire contents of `geometry.go` into `geometry/geometry.go`. Change `package taffy` to `package geometry`. Everything in this file is already exported or unexported correctly — no renaming needed.

The file contains:
- Types: `Option[T]`, `AbsoluteAxis`, `AbstractAxis`, `inBothAbsAxis[T]`, `Rect[T]`, `Line[T]`, `Size[T]`, `Point[T]`, `MinMax[Min, Max]`
- Constants: `AbsoluteAxisHorizontal`, `AbsoluteAxisVertical`, `AbstractAxisInline`, `AbstractAxisBlock`
- Variables: `RectZeroF32`, `LineTrueBool`, `LineFalseBool`, `SizeZeroF32`, `SizeOptionNone`, `PointZeroF32`, `PointNoneF32`
- Functions: `Some`, `None`, constructors, map/zip helpers, axis accessors
- Unexported helpers: `f32Max`, `f32Min`, `inBothAbsAxis`

```go
// First line change only:
package geometry
```

- [ ] **Step 2: Create `geometry/sys.go`**

Copy `util_sys.go` into `geometry/sys.go`. Change package to `geometry`.

Contains unexported helpers: `roundF32`, `ceilF32`, `floorF32`, `absF32`.

```go
package geometry
```

- [ ] **Step 3: Create `geometry/math.go`**

Copy from `util_math.go` only the functions that do NOT reference `AvailableSpace`. These are the pure `Option[float32]`, `float32`, and `Size[Option[float32]]` math operations.

**Include these exported functions:**
- `MaybeMinOption`, `MaybeMaxOption`, `MaybeAddOption`, `MaybeSubOption`, `MaybeClampOption`
- `MaybeMinF32`, `MaybeMaxF32`, `MaybeAddF32`, `MaybeSubF32`, `MaybeClampF32`
- `SizeMaybeMin`, `SizeMaybeMax`, `SizeMaybeAdd`, `SizeMaybeSub`, `SizeMaybeClamp`

**Include these unexported functions:**
- `maybeMinOption`, `maybeMaxOption`, `maybeAddOption`, `maybeSubOption`, `maybeClampOption`
- `maybeMinF32`, `maybeMaxF32`, `maybeAddF32`, `maybeSubF32`, `maybeClampF32`
- `sizeMaybeMin`, `sizeMaybeMax`, `sizeMaybeAdd`, `sizeMaybeSub`, `sizeMaybeClamp`

**Do NOT include:** `AsMaybeMinF32`, `AsMaybeMaxF32`, `AsMaybeAddF32`, `AsMaybeSubF32`, `AsMaybeMinOption`, `AsMaybeMaxOption`, `AsMaybeAddOption`, `AsMaybeSubOption` and their unexported counterparts — these reference `AvailableSpace` and go to `style/math.go` later.

```go
package geometry
```

- [ ] **Step 4: Verify `geometry` package compiles**

Run: `cd /Users/wizard/workspace/gamedev/taffy3-go && go build ./geometry/`
Expected: success, no errors

- [ ] **Step 5: Commit**

```bash
git add geometry/
git commit -m "refactor: create geometry package with core types and math"
```

---

### Task 2: Create `style` Package

**Files:**
- Create: `style/style.go`
- Create: `style/dimension.go`
- Create: `style/alignment.go`
- Create: `style/flex.go`
- Create: `style/grid.go`
- Create: `style/block.go`
- Create: `style/float.go`
- Create: `style/available_space.go`
- Create: `style/helpers.go`
- Create: `style/math.go`
- Create: `style/resolve.go`
- Source: `style.go`, `style_dimension.go`, `style_alignment.go`, `style_flex.go`, `style_grid.go`, `style_block.go`, `style_float.go`, `style_available_space.go`, `style_helpers.go`, `util_math.go` (partial), `util_resolve.go`, `compute_leaf.go` (partial)

This package imports `geometry` but nothing else from taffy.

- [ ] **Step 1: Create `style/dimension.go`**

Copy `style_dimension.go` → `style/dimension.go`. Change package to `style`. Replace all geometry type references with `geometry.` prefix.

Changes needed:
- `package style`
- `import "taffy-go/geometry"`
- `Option[float32]` → `geometry.Option[float32]`
- `Some[float32]` → `geometry.Some[float32]`
- `None[float32]` → `geometry.None[float32]`

- [ ] **Step 2: Create `style/alignment.go`**

Copy `style_alignment.go` → `style/alignment.go`. Change package to `style`. No geometry imports needed — these are pure enum types.

- [ ] **Step 3: Create `style/float.go`**

Copy `style_float.go` → `style/float.go`. Change package to `style`. No geometry imports needed.

- [ ] **Step 4: Create `style/block.go`**

Copy `style_block.go` → `style/block.go`. Change package to `style`. 

**Make interfaces public** — rename and capitalize:
```go
type BlockContainerStyle interface {
    TextAlign() TextAlign
}

type BlockItemStyle interface {
    IsTable() bool
    Float_() Float
    Clear_() Clear
}
```

Note: Keep `Float_()` and `Clear_()` with trailing underscore to avoid collision with the `Float` and `Clear` type names.

- [ ] **Step 5: Create `style/available_space.go`**

Copy `style_available_space.go` → `style/available_space.go`. Change package to `style`. Add geometry imports for `Option[T]`, `Size[T]`.

Changes:
- `Option[float32]` → `geometry.Option[float32]`
- `Size[AvailableSpace]` → `geometry.Size[AvailableSpace]`
- `Size[Option[float32]]` → `geometry.Size[geometry.Option[float32]]`
- `Some[float32]` → `geometry.Some[float32]`
- `None[float32]` → `geometry.None[float32]`

- [ ] **Step 6: Create `style/flex.go`**

Copy `style_flex.go` → `style/flex.go`. Change package to `style`. Add geometry import for `AbsoluteAxis`, `Size`, `Option`.

**Make interfaces public:**
```go
type FlexboxContainerStyle interface {
    FlexDirection() FlexDirection
    FlexWrap() FlexWrap
    Gap() geometry.Size[LengthPercentage]
    AlignContent() geometry.Option[AlignContent]
    AlignItems() geometry.Option[AlignItems]
    JustifyContent() geometry.Option[AlignContent]
}

type FlexboxItemStyle interface {
    FlexBasis() Dimension
    FlexGrow() float32
    FlexShrink() float32
    AlignSelf() geometry.Option[AlignItems]
}
```

- [ ] **Step 7: Create `style/grid.go`**

Copy `style_grid.go` → `style/grid.go`. Change package to `style`. Add geometry imports. Move `CalcResolver` type alias here (it's currently in `style_grid.go`).

**Make interfaces public:**
```go
type GridContainerStyle interface {
    GridTemplateRows() []GridTemplateComponent
    GridTemplateColumns() []GridTemplateComponent
    GridAutoRows() []TrackSizingFunction
    GridAutoColumns() []TrackSizingFunction
    GridTemplateAreas() []GridTemplateArea
    GridAutoFlow() GridAutoFlow
    Gap() geometry.Size[LengthPercentage]
    AlignContent() geometry.Option[AlignContent]
    JustifyContent() geometry.Option[JustifyContent]
    AlignItems() geometry.Option[AlignItems]
    JustifyItems() geometry.Option[AlignItems]
}

type GridItemStyle interface {
    GridRow() geometry.Line[GridPlacement]
    GridColumn() geometry.Line[GridPlacement]
    AlignSelf() geometry.Option[AlignSelf]
    JustifySelf() geometry.Option[AlignSelf]
}
```

Geometry type replacements throughout the file:
- `Line[T]` → `geometry.Line[T]`
- `Size[T]` → `geometry.Size[T]`
- `Option[T]` → `geometry.Option[T]`
- `MinMax[A, B]` → `geometry.MinMax[A, B]`
- `Some(...)` → `geometry.Some(...)`
- `None(...)` → `geometry.None(...)`
- `AbsoluteAxis*` → `geometry.AbsoluteAxis*`

- [ ] **Step 8: Create `style/style.go`**

Copy `style.go` → `style/style.go`. Change package to `style`. Add geometry imports.

**Make `CoreStyle` interface public** and rename methods to be exported:
```go
type CoreStyle interface {
    BoxGenerationMode() BoxGenerationMode
    IsBlock() bool
    IsCompressibleReplaced() bool
    BoxSizing() BoxSizing
    Direction() Direction
    Overflow() geometry.Point[Overflow]
    ScrollbarWidth() float32
    Position() Position
    Inset() geometry.Rect[LengthPercentageAuto]
    Size() geometry.Size[Dimension]
    MinSize() geometry.Size[Dimension]
    MaxSize() geometry.Size[Dimension]
    AspectRatio() geometry.Option[float32]
    Margin() geometry.Rect[LengthPercentageAuto]
    Padding() geometry.Rect[LengthPercentage]
    Border() geometry.Rect[LengthPercentage]
}
```

**Rename all `*Style` method implementations** from lowercase to exported:
```go
// Old: func (s *Style) boxGenerationMode() BoxGenerationMode
// New:
func (s *Style) BoxGenerationMode() BoxGenerationMode {
    if s.Display == DisplayNone {
        return BoxGenerationModeNone
    }
    return BoxGenerationModeNormal
}
```

Apply this pattern to ALL style getter methods on `*Style`:
- `boxGenerationMode` → `BoxGenerationMode`
- `isBlock` → `IsBlock`
- `isCompressibleReplaced` → `IsCompressibleReplaced`
- `boxSizing` → `BoxSizing_` (trailing underscore to avoid collision with `BoxSizing` type — OR rename method to `GetBoxSizing`)
- ...and so on for all methods

**IMPORTANT naming collision resolution:** Several method names collide with field names on the `Style` struct (e.g., `BoxSizing` is both a field and would-be method name). Use `Get` prefix for these methods to resolve collisions:

```go
func (s *Style) GetBoxSizing() BoxSizing      { return s.BoxSizing }
func (s *Style) GetDirection() Direction       { return s.Direction }
func (s *Style) GetOverflow() geometry.Point[Overflow] { return s.Overflow }
func (s *Style) GetScrollbarWidth() float32    { return s.ScrollbarWidth }
func (s *Style) GetPosition() Position         { return s.Position }
func (s *Style) GetInset() geometry.Rect[LengthPercentageAuto] { return s.Inset }
func (s *Style) GetSize() geometry.Size[Dimension] { return s.Size }
func (s *Style) GetMinSize() geometry.Size[Dimension] { return s.MinSize }
func (s *Style) GetMaxSize() geometry.Size[Dimension] { return s.MaxSize }
func (s *Style) GetAspectRatio() geometry.Option[float32] { return s.AspectRatio }
func (s *Style) GetMargin() geometry.Rect[LengthPercentageAuto] { return s.Margin }
func (s *Style) GetPadding() geometry.Rect[LengthPercentage] { return s.Padding }
func (s *Style) GetBorder() geometry.Rect[LengthPercentage] { return s.Border }
```

Update the `CoreStyle` interface to use `Get` prefix consistently:
```go
type CoreStyle interface {
    GetBoxGenerationMode() BoxGenerationMode
    GetIsBlock() bool
    GetIsCompressibleReplaced() bool
    GetBoxSizing() BoxSizing
    GetDirection() Direction
    GetOverflow() geometry.Point[Overflow]
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
```

And update ALL other style interfaces (`FlexboxContainerStyle`, `FlexboxItemStyle`, `BlockContainerStyle`, `BlockItemStyle`, `GridContainerStyle`, `GridItemStyle`) to use `Get` prefix on all methods for consistency.

Update `DefaultStyle()` to use `geometry.` prefixed types for all field values.

Update compile-time checks:
```go
var _ CoreStyle = (*Style)(nil)
var _ BlockContainerStyle = (*Style)(nil)
var _ BlockItemStyle = (*Style)(nil)
var _ FlexboxContainerStyle = (*Style)(nil)
var _ FlexboxItemStyle = (*Style)(nil)
var _ GridContainerStyle = (*Style)(nil)
var _ GridItemStyle = (*Style)(nil)
```

- [ ] **Step 9: Create `style/helpers.go`**

Copy `style_helpers.go` → `style/helpers.go`. Change package to `style`. Remove the `taffy.` prefix from all type references (they're now in the same package). Add `geometry.` prefix where geometry types are used.

- [ ] **Step 10: Create `style/resolve.go`**

Copy `util_resolve.go` → `style/resolve.go`. Change package to `style`. Update imports.

Changes:
- `Option[float32]` → `geometry.Option[float32]`
- `Size[T]` → `geometry.Size[T]`
- `Rect[T]` → `geometry.Rect[T]`
- `Some(...)` → `geometry.Some(...)`
- `None(...)` → `geometry.None(...)`

Also move the two helper functions from `compute_leaf.go` here and make them exported:

```go
// ResolveRectLPWidth resolves a Rect[LengthPercentage] using width as context for all sides.
func ResolveRectLPWidth(rect geometry.Rect[LengthPercentage], widthContext geometry.Option[float32], calc CalcResolver) geometry.Rect[float32] {
    // ... existing body from resolveRectLPWidth in compute_leaf.go
}

// ResolveRectLPAWidth resolves a Rect[LengthPercentageAuto] using width as context for all sides.
func ResolveRectLPAWidth(rect geometry.Rect[LengthPercentageAuto], widthContext geometry.Option[float32], calc CalcResolver) geometry.Rect[float32] {
    // ... existing body from resolveRectLPAWidth in compute_leaf.go
}
```

- [ ] **Step 11: Create `style/math.go`**

Copy the AvailableSpace-related functions from `util_math.go` into `style/math.go`:

**Exported functions:**
- `AsMaybeMinF32`, `AsMaybeMaxF32`, `AsMaybeAddF32`, `AsMaybeSubF32`
- `AsMaybeMinOption`, `AsMaybeMaxOption`, `AsMaybeAddOption`, `AsMaybeSubOption`

**Unexported functions:**
- `asMaybeMinF32`, `asMaybeMaxF32`, `asMaybeAddF32`, `asMaybeSubF32`
- `asMaybeMinOption`, `asMaybeMaxOption`, `asMaybeAddOption`, `asMaybeSubOption`

These functions reference `AvailableSpace` (from `available_space.go` in the same package) and `Option[float32]` (from `geometry`).

```go
package style

import "taffy-go/geometry"
```

- [ ] **Step 12: Verify `style` package compiles**

Run: `cd /Users/wizard/workspace/gamedev/taffy3-go && go build ./style/`
Expected: success

- [ ] **Step 13: Commit**

```bash
git add style/
git commit -m "refactor: create style package with types, interfaces, and resolve helpers"
```

---

### Task 3: Create `tree` Package

**Files:**
- Create: `tree/node.go`
- Create: `tree/layout.go`
- Create: `tree/cache.go`
- Create: `tree/traits.go`
- Source: `node.go`, `layout.go`, `cache.go`, `tree_traits.go`

This package imports `geometry` and `style`.

- [ ] **Step 1: Create `tree/node.go`**

Copy `node.go` → `tree/node.go`. Change package to `tree`.

```go
package tree

type NodeId = uint64

const invalidNodeId NodeId = ^NodeId(0)

func NewNodeId(val uint64) NodeId { return val }
```

- [ ] **Step 2: Create `tree/layout.go`**

Copy `layout.go` → `tree/layout.go`. Change package to `tree`. Add geometry and style imports.

Replace:
- `Size[T]` → `geometry.Size[T]`
- `Point[T]` → `geometry.Point[T]`
- `Rect[T]` → `geometry.Rect[T]`
- `Line[T]` → `geometry.Line[T]`
- `Option[T]` → `geometry.Option[T]`
- `AvailableSpace` → `style.AvailableSpace`
- `AbsoluteAxis` → `geometry.AbsoluteAxis`
- `SizeZeroF32` → `geometry.SizeZeroF32`
- `PointZeroF32` → `geometry.PointZeroF32`
- `RectZeroF32` → `geometry.RectZeroF32`

- [ ] **Step 3: Create `tree/cache.go`**

Copy `cache.go` → `tree/cache.go`. Change package to `tree`. Add geometry and style imports.

Same replacement pattern as layout.go. The `cache` type can stay unexported — it's an implementation detail. But `ClearState` and its constants are exported.

- [ ] **Step 4: Create `tree/traits.go`**

Copy `tree_traits.go` → `tree/traits.go`. Change package to `tree`. Add geometry and style imports.

**Make all interfaces public** — capitalize names and method names:

```go
package tree

import (
    "taffy-go/geometry"
    "taffy-go/style"
)

type TraversePartialTree interface {
    ChildIds(parent NodeId) []NodeId
    ChildCount(parent NodeId) int
    GetChildId(parent NodeId, childIndex int) NodeId
}

type TraverseTree interface {
    TraversePartialTree
}

type LayoutPartialTree interface {
    TraversePartialTree
    GetCoreContainerStyle(nodeId NodeId) style.CoreStyle
    ResolveCalcValue(calcId uint64, basis float32) float32
    SetUnroundedLayout(nodeId NodeId, layout *Layout)
    ComputeChildLayout(nodeId NodeId, inputs LayoutInput) LayoutOutput
}

type CacheTree interface {
    CacheGet(nodeId NodeId, input *LayoutInput) *LayoutOutput
    CacheStore(nodeId NodeId, input *LayoutInput, output LayoutOutput)
    CacheClear(nodeId NodeId)
}

type RoundTree interface {
    TraverseTree
    GetUnroundedLayout(nodeId NodeId) *Layout
    SetFinalLayout(nodeId NodeId, layout *Layout)
}

type PrintTree interface {
    TraverseTree
    GetDebugLabel(nodeId NodeId) string
    GetFinalLayout(nodeId NodeId) *Layout
}

type LayoutFlexboxContainer interface {
    LayoutPartialTree
    GetFlexboxContainerStyle(nodeId NodeId) style.FlexboxContainerStyle
    GetFlexboxChildStyle(childNodeId NodeId) style.FlexboxItemStyle
}

type LayoutGridContainer interface {
    LayoutPartialTree
    GetGridContainerStyle(nodeId NodeId) style.GridContainerStyle
    GetGridChildStyle(childNodeId NodeId) style.GridItemStyle
}

type LayoutBlockContainer interface {
    LayoutPartialTree
    GetBlockContainerStyle(nodeId NodeId) style.BlockContainerStyle
    GetBlockChildStyle(childNodeId NodeId) style.BlockItemStyle
    ComputeBlockChildLayout(nodeId NodeId, inputs LayoutInput, blockCtx any) LayoutOutput
}
```

Note: `blockContext` is currently defined in `compute_block.go`. For the interface, use `any` as the blockCtx type since the concrete type will be in `compute/block/`. This is a pragmatic choice — the block context is an opaque value passed through the tree interface.

**Make helper functions public:**
```go
func MeasureChildSize(tree LayoutPartialTree, nodeId NodeId, knownDimensions geometry.Size[geometry.Option[float32]], parentSize geometry.Size[geometry.Option[float32]], availableSpace geometry.Size[style.AvailableSpace], sizingMode SizingMode, axis geometry.AbsoluteAxis, verticalMarginsAreCollapsible geometry.Line[bool]) float32 {
    // ... existing body
}

func PerformChildLayout(tree LayoutPartialTree, nodeId NodeId, knownDimensions geometry.Size[geometry.Option[float32]], parentSize geometry.Size[geometry.Option[float32]], availableSpace geometry.Size[style.AvailableSpace], sizingMode SizingMode, verticalMarginsAreCollapsible geometry.Line[bool]) LayoutOutput {
    // ... existing body
}
```

- [ ] **Step 5: Verify `tree` package compiles**

Run: `cd /Users/wizard/workspace/gamedev/taffy3-go && go build ./tree/`
Expected: success

- [ ] **Step 6: Commit**

```bash
git add tree/
git commit -m "refactor: create tree package with public trait interfaces and layout types"
```

---

### Task 4: Create `compute` Package (Core)

**Files:**
- Create: `compute/compute.go`
- Create: `compute/leaf.go`
- Create: `compute/alignment.go`
- Source: `compute.go`, `compute_leaf.go`, `compute_alignment.go`

This package imports `geometry`, `style`, and `tree`.

- [ ] **Step 1: Create `compute/compute.go`**

Copy `compute.go` → `compute/compute.go`. Change package to `compute`. Add imports.

**Make functions public** (except `roundLayoutInner` and `roundContentSize` which are internal helpers):

```go
package compute

import (
    "taffy-go/geometry"
    "taffy-go/style"
    "taffy-go/tree"
)

// LayoutAndCacheTree combines LayoutPartialTree and CacheTree.
type LayoutAndCacheTree interface {
    tree.LayoutPartialTree
    tree.CacheTree
}

func ComputeCachedLayout(t tree.CacheTree, node tree.NodeId, inputs tree.LayoutInput, computeUncached func(tree.NodeId, tree.LayoutInput) tree.LayoutOutput) tree.LayoutOutput {
    // ... existing body with type prefixes
}

func RoundLayout(t tree.RoundTree, nodeId tree.NodeId) {
    // ... existing body
}

func ComputeHiddenLayout(t LayoutAndCacheTree, node tree.NodeId) tree.LayoutOutput {
    // ... existing body
}
```

**Remove `computeRootLayout`** from this file — it moves to root `taffy` package (it contains dispatch logic).

Replace all internal type references:
- `NodeId` → `tree.NodeId`
- `LayoutInput` → `tree.LayoutInput`
- `LayoutOutput` → `tree.LayoutOutput`
- `Layout` → `tree.Layout`
- `Size[T]` → `geometry.Size[T]`
- `Point[T]` → `geometry.Point[T]`
- `roundTree` → `tree.RoundTree`
- `cacheTree` → `tree.CacheTree`

- [ ] **Step 2: Create `compute/leaf.go`**

Copy `compute_leaf.go` → `compute/leaf.go`. Change package to `compute`. 

**Make `ComputeLeafLayout` public.** Remove `resolveRectLPWidth` and `resolveRectLPAWidth` (they moved to `style/resolve.go`).

```go
func ComputeLeafLayout(inputs tree.LayoutInput, coreStyle style.CoreStyle, calc style.CalcResolver, measureFunction func(geometry.Size[geometry.Option[float32]], geometry.Size[style.AvailableSpace]) geometry.Size[float32]) tree.LayoutOutput {
    // ... existing body, updated to use:
    // style.ResolveRectLPWidth(...) instead of resolveRectLPWidth(...)
    // style.ResolveRectLPAWidth(...) instead of resolveRectLPAWidth(...)
    // coreStyle.GetBoxSizing() instead of style.boxSizing()
    // etc.
}
```

Update all style access to use the public interface methods:
- `style.boxSizing()` → `coreStyle.GetBoxSizing()`
- `style.overflow()` → `coreStyle.GetOverflow()`
- `style.size()` → `coreStyle.GetSize()`
- `style.minSize()` → `coreStyle.GetMinSize()`
- `style.maxSize()` → `coreStyle.GetMaxSize()`
- etc.

Update all geometry/style type references with package prefixes.

- [ ] **Step 3: Create `compute/alignment.go`**

Copy `compute_alignment.go` → `compute/alignment.go`. Change package to `compute`.

**Make functions public:**
```go
func ApplyAlignmentFallback(freeSpace float32, numItems int, alignmentMode style.AlignContent, isSafe bool) style.AlignContent { ... }
func ComputeAlignmentOffset(freeSpace float32, numItems int, gap float32, alignmentMode style.AlignContent, layoutIsFlexReversed bool, isFirst bool) float32 { ... }
```

- [ ] **Step 4: Verify `compute` package compiles**

Run: `cd /Users/wizard/workspace/gamedev/taffy3-go && go build ./compute/`
Expected: success

- [ ] **Step 5: Commit**

```bash
git add compute/
git commit -m "refactor: create compute package with cached layout, leaf, and alignment"
```

---

### Task 5: Create `compute/flexbox` Package

**Files:**
- Create: `compute/flexbox/flexbox.go`
- Source: `compute_flexbox.go`

- [ ] **Step 1: Create `compute/flexbox/flexbox.go`**

Copy `compute_flexbox.go` → `compute/flexbox/flexbox.go`. Change package to `flexbox`.

**Make entry point public:**
```go
func ComputeFlexboxLayout(t tree.LayoutFlexboxContainer, node tree.NodeId, inputs tree.LayoutInput) tree.LayoutOutput {
    // ... existing computeFlexboxLayout body
}
```

All internal helper functions (`sizeMain`, `sizeCross`, `computePreliminary`, `computeConstants`, etc.) stay unexported.

Update all type references:
- Add `geometry.`, `style.`, `tree.` prefixes throughout
- Style interface access: `style.flexDirection()` → use the `tree.LayoutFlexboxContainer` interface which provides `GetFlexboxContainerStyle()` returning `style.FlexboxContainerStyle`
- Math functions: `maybeMinF32(...)` → `geometry.MaybeMinF32(...)` (now need to use exported versions since they're in a different package)
- `computeAlignmentOffset(...)` → `compute.ComputeAlignmentOffset(...)`
- `computeLeafLayout(...)` → `compute.ComputeLeafLayout(...)`
- `computeCachedLayout(...)` → `compute.ComputeCachedLayout(...)`
- `measureChildSize(...)` → `tree.MeasureChildSize(...)`
- `performChildLayout(...)` → `tree.PerformChildLayout(...)`

This is the largest file (~2782 lines) and will require the most mechanical changes. The logic stays identical — only package qualifiers and method name capitalization change.

- [ ] **Step 2: Verify `compute/flexbox` package compiles**

Run: `cd /Users/wizard/workspace/gamedev/taffy3-go && go build ./compute/flexbox/`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add compute/flexbox/
git commit -m "refactor: create compute/flexbox package"
```

---

### Task 6: Create `compute/block` Package

**Files:**
- Create: `compute/block/block.go`
- Create: `compute/block/float.go`
- Source: `compute_block.go`, `compute_float.go`

- [ ] **Step 1: Create `compute/block/float.go`**

Copy `compute_float.go` → `compute/block/float.go`. Change package to `block`.

All types (`contentSlot`, `placedFloatedBox`, `segment`, `floatFitter`) stay unexported — they're internal to the block package. But `floatContext` needs to be accessible from `block.go`, which is in the same package, so no export needed.

Update geometry type references with `geometry.` prefix.

- [ ] **Step 2: Create `compute/block/block.go`**

Copy `compute_block.go` → `compute/block/block.go`. Change package to `block`.

**Make entry point public:**
```go
func ComputeBlockLayout(t tree.LayoutBlockContainer, node tree.NodeId, inputs tree.LayoutInput) tree.LayoutOutput {
    // ... existing computeBlockLayout body
}
```

**Export `BlockContext`** type since it's passed through the tree interface:
```go
type BlockContext struct {
    // ... existing blockContext fields, capitalized
}
```

Update all type references with package prefixes. Same pattern as flexbox:
- Style access through interface methods
- Math functions use exported versions from `geometry` package
- `computeLeafLayout(...)` → `compute.ComputeLeafLayout(...)`
- `computeCachedLayout(...)` → `compute.ComputeCachedLayout(...)`

- [ ] **Step 3: Verify `compute/block` package compiles**

Run: `cd /Users/wizard/workspace/gamedev/taffy3-go && go build ./compute/block/`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add compute/block/
git commit -m "refactor: create compute/block package"
```

---

### Task 7: Create `compute/grid` Package

**Files:**
- Create: `compute/grid/grid.go`
- Create: `compute/grid/track_sizing.go`
- Create: `compute/grid/placement.go`
- Create: `compute/grid/alignment.go`
- Create: `compute/grid/explicit.go`
- Create: `compute/grid/implicit.go`
- Create: `compute/grid/types_coords.go`
- Create: `compute/grid/types_item.go`
- Create: `compute/grid/types_named.go`
- Create: `compute/grid/types_occupancy.go`
- Create: `compute/grid/types_track.go`
- Source: `compute_grid.go`, `compute_grid_track_sizing.go`, `compute_grid_placement.go`, `compute_grid_alignment.go`, `compute_grid_explicit.go`, `compute_grid_implicit.go`, `compute_grid_types_*.go`

- [ ] **Step 1: Create grid type files**

Copy each `compute_grid_types_*.go` to `compute/grid/types_*.go`. Change package to `grid`. Add geometry/style/tree imports and update type references.

These files define grid-internal types used across the grid algorithm. Types that are referenced only within `compute/grid/` stay unexported. Types that need to cross the package boundary (if any) get exported.

- [ ] **Step 2: Create grid algorithm files**

Copy each remaining `compute_grid_*.go` to the corresponding `compute/grid/*.go`. Change package to `grid`. Update imports and type references.

- [ ] **Step 3: Create `compute/grid/grid.go`**

Copy `compute_grid.go` → `compute/grid/grid.go`. Change package to `grid`.

**Make entry point public:**
```go
func ComputeGridLayout(t tree.LayoutGridContainer, node tree.NodeId, inputs tree.LayoutInput) tree.LayoutOutput {
    // ... existing computeGridLayout body
}
```

- [ ] **Step 4: Verify `compute/grid` package compiles**

Run: `cd /Users/wizard/workspace/gamedev/taffy3-go && go build ./compute/grid/`
Expected: success

- [ ] **Step 5: Commit**

```bash
git add compute/grid/
git commit -m "refactor: create compute/grid package"
```

---

### Task 8: Update Root `taffy` Package

**Files:**
- Create: `taffy_view.go` (new — split from `taffy_tree.go`)
- Modify: `taffy_tree.go` → `taffy.go` (rename + update)
- Modify: `print.go` (rename from `util_print.go`)
- Delete: all original source files that were copied to sub-packages

This is the integration task — the root `taffy` package ties everything together.

- [ ] **Step 1: Create `taffy_view.go`**

Extract `taffyView` and all its interface implementations from `taffy_tree.go` into `taffy_view.go`.

Update `taffyView` to implement the **public** interfaces from `tree/`:
```go
package taffy

import (
    "taffy-go/geometry"
    "taffy-go/style"
    "taffy-go/tree"
    "taffy-go/compute"
    "taffy-go/compute/flexbox"
    "taffy-go/compute/grid"
    "taffy-go/compute/block"
)

type taffyView struct {
    t         *TaffyTree
    measureFn MeasureFunction
}

// TraversePartialTree implementation
func (v *taffyView) ChildIds(parent tree.NodeId) []tree.NodeId { ... }
func (v *taffyView) ChildCount(parent tree.NodeId) int { ... }
func (v *taffyView) GetChildId(parent tree.NodeId, index int) tree.NodeId { ... }

// LayoutPartialTree implementation
func (v *taffyView) GetCoreContainerStyle(nodeId tree.NodeId) style.CoreStyle {
    return v.t.nodes[nodeId].style  // *style.Style implements style.CoreStyle
}
func (v *taffyView) ResolveCalcValue(_ uint64, _ float32) float32 { return 0 }
func (v *taffyView) SetUnroundedLayout(nodeId tree.NodeId, layout *tree.Layout) { ... }
func (v *taffyView) ComputeChildLayout(nodeId tree.NodeId, inputs tree.LayoutInput) tree.LayoutOutput {
    // DISPATCH LOGIC — the key integration point
    return compute.ComputeCachedLayout(v, nodeId, inputs, func(nodeId tree.NodeId, inputs tree.LayoutInput) tree.LayoutOutput {
        nodeData := v.t.nodes[nodeId]
        display := nodeData.style.Display
        switch {
        case display == style.DisplayNone:
            return compute.ComputeHiddenLayout(v, nodeId)
        case display == style.DisplayBlock:
            return block.ComputeBlockLayout(v, nodeId, inputs)
        case display == style.DisplayFlex:
            return flexbox.ComputeFlexboxLayout(v, nodeId, inputs)
        case display == style.DisplayGrid:
            return grid.ComputeGridLayout(v, nodeId, inputs)
        default:
            return compute.ComputeLeafLayout(inputs, v.GetCoreContainerStyle(nodeId), v.ResolveCalcValue, v.measureFunc(nodeId))
        }
    })
}

// CacheTree implementation
func (v *taffyView) CacheGet(nodeId tree.NodeId, input *tree.LayoutInput) *tree.LayoutOutput { ... }
func (v *taffyView) CacheStore(nodeId tree.NodeId, input *tree.LayoutInput, output tree.LayoutOutput) { ... }
func (v *taffyView) CacheClear(nodeId tree.NodeId) { ... }

// RoundTree implementation
func (v *taffyView) GetUnroundedLayout(nodeId tree.NodeId) *tree.Layout { ... }
func (v *taffyView) SetFinalLayout(nodeId tree.NodeId, layout *tree.Layout) { ... }

// PrintTree implementation
func (v *taffyView) GetDebugLabel(nodeId tree.NodeId) string { ... }
func (v *taffyView) GetFinalLayout(nodeId tree.NodeId) *tree.Layout { ... }

// Layout-mode-specific style accessors
func (v *taffyView) GetFlexboxContainerStyle(nodeId tree.NodeId) style.FlexboxContainerStyle {
    return v.t.nodes[nodeId].style
}
func (v *taffyView) GetFlexboxChildStyle(childNodeId tree.NodeId) style.FlexboxItemStyle {
    return v.t.nodes[childNodeId].style
}
// ... same pattern for Grid and Block style accessors
```

- [ ] **Step 2: Update `taffy_tree.go` → `taffy.go`**

Rename `taffy_tree.go` to `taffy.go`. Remove `taffyView` (now in `taffy_view.go`). Update `TaffyTree` to use sub-package types:

```go
package taffy

import (
    "taffy-go/style"
    "taffy-go/tree"
    "taffy-go/geometry"
    "taffy-go/compute"
)

type nodeData struct {
    style    style.Style
    layout   tree.Layout
    cache    tree.Cache  // or keep cache unexported and in tree/
    children []tree.NodeId
    parent   *tree.NodeId
    context  *NodeContext
    dirty    bool
}

type TaffyTree struct {
    nodes   map[tree.NodeId]nodeData
    nextId  tree.NodeId
    config  taffyConfig
}

func New() *TaffyTree { ... }
func (t *TaffyTree) NewLeaf(s style.Style) tree.NodeId { ... }
func (t *TaffyTree) SetStyle(node tree.NodeId, s style.Style) { ... }
func (t *TaffyTree) GetStyle(node tree.NodeId) *style.Style { ... }
func (t *TaffyTree) GetLayout(node tree.NodeId) *tree.Layout { ... }
func (t *TaffyTree) ComputeLayout(node tree.NodeId, availableSpace geometry.Size[style.AvailableSpace]) { ... }
// ... all other public methods updated with sub-package types
```

- [ ] **Step 3: Update `util_print.go` → `print.go`**

Rename and update to use `tree.PrintTree` interface:

```go
package taffy

import (
    "io"
    "taffy-go/tree"
)

func PrintTreeFn(t tree.PrintTree, root tree.NodeId) { ... }
func WriteTreeFn(w io.Writer, t tree.PrintTree, root tree.NodeId) { ... }
```

- [ ] **Step 4: Delete old source files**

Remove all files that have been migrated to sub-packages:
```bash
rm geometry.go util_math.go util_sys.go
rm style.go style_dimension.go style_alignment.go style_flex.go style_grid.go style_block.go style_float.go style_available_space.go style_helpers.go
rm node.go layout.go cache.go tree_traits.go
rm compute.go compute_leaf.go compute_alignment.go
rm compute_flexbox.go
rm compute_block.go compute_float.go
rm compute_grid.go compute_grid_track_sizing.go compute_grid_placement.go compute_grid_alignment.go compute_grid_explicit.go compute_grid_implicit.go
rm compute_grid_types_coords.go compute_grid_types_item.go compute_grid_types_named.go compute_grid_types_occupancy.go compute_grid_types_track.go
rm util_resolve.go util_print.go
```

- [ ] **Step 5: Verify root package compiles**

Run: `cd /Users/wizard/workspace/gamedev/taffy3-go && go build .`
Expected: success

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "refactor: update root taffy package to use sub-packages, add dispatch logic"
```

---

### Task 9: Migrate Tests

**Files:**
- Move: `geometry_test.go` → `geometry/geometry_test.go`
- Move: `util_math_test.go` → split across `geometry/math_test.go` and `style/math_test.go`
- Move: `util_resolve_test.go` → `style/resolve_test.go`
- Move: `style_test.go` → `style/style_test.go`
- Move: `style_flex_test.go` → `style/flex_test.go`
- Keep in root: `taffy_tree_test.go`, `taffy_suite_test.go`, `fixtures_*.go`, `fixtures_helpers_test.go`

- [ ] **Step 1: Move geometry tests**

Copy `geometry_test.go` → `geometry/geometry_test.go`. Change package to `geometry`. Remove `taffy.` prefixes (now in same package).

- [ ] **Step 2: Move style tests**

Copy `style_test.go` → `style/style_test.go` and `style_flex_test.go` → `style/flex_test.go`. Change package to `style`. Update type references.

- [ ] **Step 3: Move resolve tests**

Copy `util_resolve_test.go` → `style/resolve_test.go`. Change package to `style`. Update references.

- [ ] **Step 4: Split math tests**

Split `util_math_test.go`:
- Tests for `MaybeMin/Max/Add/Sub/Clamp` (Option and float32 variants) and `SizeMaybe*` → `geometry/math_test.go`
- Tests for `AsMaybe*` (AvailableSpace variants) → `style/math_test.go`

- [ ] **Step 5: Update integration tests in root**

Update `taffy_tree_test.go` to import sub-packages:
```go
import (
    "taffy-go/geometry"
    "taffy-go/style"
    "taffy-go/tree"
)
```

Replace all type references: `Style` → `style.Style`, `NodeId` → `tree.NodeId`, `Size` → `geometry.Size`, etc.

Update `fixtures_helpers_test.go` with same import changes.

- [ ] **Step 6: Delete old test files from root**

```bash
rm geometry_test.go util_math_test.go util_resolve_test.go style_test.go style_flex_test.go
```

- [ ] **Step 7: Run all tests**

Run: `cd /Users/wizard/workspace/gamedev/taffy3-go && go test ./...`
Expected: all tests pass

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "refactor: migrate tests to sub-packages"
```

---

### Task 10: Final Verification and Cleanup

- [ ] **Step 1: Run full test suite**

Run: `cd /Users/wizard/workspace/gamedev/taffy3-go && go test ./... -count=1`
Expected: all tests pass

- [ ] **Step 2: Verify no old files remain**

Run: `ls /Users/wizard/workspace/gamedev/taffy3-go/*.go`
Expected: only `taffy.go`, `taffy_view.go`, `print.go`, and test files remain in root

- [ ] **Step 3: Verify compile-time interface checks**

Ensure these compile-time assertions exist and pass:

In `style/style.go`:
```go
var _ CoreStyle = (*Style)(nil)
var _ BlockContainerStyle = (*Style)(nil)
var _ BlockItemStyle = (*Style)(nil)
var _ FlexboxContainerStyle = (*Style)(nil)
var _ FlexboxItemStyle = (*Style)(nil)
var _ GridContainerStyle = (*Style)(nil)
var _ GridItemStyle = (*Style)(nil)
```

In `taffy_view.go` (root):
```go
var _ tree.TraversePartialTree = (*taffyView)(nil)
var _ tree.LayoutPartialTree = (*taffyView)(nil)
var _ tree.CacheTree = (*taffyView)(nil)
var _ tree.RoundTree = (*taffyView)(nil)
var _ tree.PrintTree = (*taffyView)(nil)
var _ tree.LayoutFlexboxContainer = (*taffyView)(nil)
var _ tree.LayoutGridContainer = (*taffyView)(nil)
var _ tree.LayoutBlockContainer = (*taffyView)(nil)
```

- [ ] **Step 4: Run vet and build**

Run: `cd /Users/wizard/workspace/gamedev/taffy3-go && go vet ./... && go build ./...`
Expected: no warnings, success

- [ ] **Step 5: Commit any cleanup**

```bash
git add -A
git commit -m "refactor: final cleanup after structural reorganization"
```
