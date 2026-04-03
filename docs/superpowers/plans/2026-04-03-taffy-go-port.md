# Taffy Go Port Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port the Taffy Rust layout engine (v0.9.2) to Go with full feature parity across flexbox, grid, block, and float layout algorithms.

**Architecture:** Single flat `taffy` package in Go 1.25. Bottom-up file-by-file port mirroring Rust source structure. Unexported Go interfaces replace Rust traits for internal use; `TaffyTree` is the only public API. Node storage via `map[NodeId]*nodeData` behind an unexported interface.

**Tech Stack:** Go 1.25, Ginkgo/Gomega for testing, `encoding/xml` for fixture parsing.

**Source:** `../taffy/src/` (Rust)
**Target:** `/Users/wizard/workspace/gamedev/taffy3-go/` (Go)

---

## General Translation Rules

These rules apply to ALL tasks. Do not repeat them per-task.

### Rust → Go Type Mappings

| Rust | Go |
|---|---|
| `f32` | `float32` |
| `Option<f32>` in struct fields | `Option[float32]` (custom type, see Task 1) |
| `Option<T>` in params/returns | `*T` or `(T, bool)` |
| `Vec<T>` | `[]T` |
| `HashMap<K,V>` | `map[K]V` |
| `Box<T>` | `*T` |
| `&str` / `String` / `CustomIdent` | `string` |
| `usize` | `int` |
| `u16` | `int16` (but use `uint16` where negative values impossible) |
| `i16` | `int16` |
| `u32` | `uint32` |
| `u64` | `uint64` |
| `Result<T, E>` | `(T, error)` |
| `panic!("msg")` | `panic("msg")` |

### Rust → Go Pattern Mappings

| Rust Pattern | Go Pattern |
|---|---|
| `enum Foo { A, B, C }` (no data) | `type Foo uint8; const (FooA Foo = iota; FooB; FooC)` |
| `enum Foo { A(f32), B, C }` (with data) | `type Foo struct { Tag FooTag; Value float32 }` |
| `impl Struct { fn method(&self) }` | `func (s *Struct) method()` |
| `impl Trait for Struct` | Struct satisfies interface implicitly |
| `match x { ... }` | `switch x { ... }` |
| `x.unwrap()` | direct access or panic |
| `#[derive(Default)]` | zero value or `func defaultFoo() Foo` |
| `#[derive(Clone, Copy)]` | Go structs are value types by default |
| `pub` | Exported (uppercase) for public API, unexported (lowercase) for internal |
| `pub(crate)` | unexported (lowercase) |
| `impl Default for T` | `func defaultT() T { ... }` or use Go zero values |
| Iterator chains `.iter().map().filter().collect()` | `for` loops |
| `#[cfg(feature = "X")]` | Include all code unconditionally |
| `CompactLength` wrapper | Use plain structs with Tag+Value instead |
| Closures `\|x\| expr` | `func(x T) R { return expr }` |
| `trait Foo: Bar` | Embed `bar` interface in `foo` interface |
| `#[inline]` | Omit |
| Lifetime annotations | Omit |
| `core::marker::PhantomData` | Omit |

### Naming Conventions

- Rust `snake_case` functions → Go `camelCase` (unexported) or `PascalCase` (exported)
- Rust `SCREAMING_SNAKE` constants → Go `PascalCase` constants
- Rust `snake_case` fields → Go `PascalCase` fields (all struct fields exported for now to simplify porting)
- Enum variants: `EnumNameVariantName` (e.g., `DisplayFlex`, `PositionAbsolute`)

### Option[T] Type

Used throughout for struct fields where zero is a valid value:

```go
type Option[T any] struct {
    Value  T
    IsSome bool
}

func Some[T any](v T) Option[T] {
    return Option[T]{Value: v, IsSome: true}
}

func None[T any]() Option[T] {
    return Option[T]{}
}
```

### Calc Support

Since we're keeping calc(), port the calc function pointer pattern. In Go, use a `CalcResolver` function type:

```go
type CalcResolver func(calcId uint64, basis float32) float32
```

Dimension types that support calc store a calc ID instead of a raw pointer.

---

## Task 0: Project Setup

**Files:**
- Modify: `go.mod`
- Create: `taffy_suite_test.go`

- [ ] **Step 1: Initialize Ginkgo test suite**

```bash
cd /Users/wizard/workspace/gamedev/taffy3-go
go get github.com/onsi/ginkgo/v2
go get github.com/onsi/gomega
```

- [ ] **Step 2: Create Ginkgo bootstrap file**

Create `taffy_suite_test.go`:

```go
package taffy_test

import (
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestTaffy(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Taffy Suite")
}
```

- [ ] **Step 3: Copy test fixtures**

```bash
cp -r ../taffy/tests/xml/ test_fixtures/
```

- [ ] **Step 4: Create .gitignore**

```
# Go
*.exe
*.test
*.out
vendor/
```

- [ ] **Step 5: Verify setup compiles**

```bash
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum taffy_suite_test.go test_fixtures/ .gitignore
git commit -m "chore: initialize project with Ginkgo test suite and test fixtures"
```

---

## Task 1: Port geometry.go

**Rust source:** `../taffy/src/geometry.rs` (~730 lines)
**Files:**
- Create: `geometry.go`
- Create: `geometry_test.go`

Port the following types and all their methods/impls:

**Types to port:**
- `Option[T any]` — generic option type (defined here, used everywhere)
- `Point[T any]` — `{ X, Y T }`
- `Size[T any]` — `{ Width, Height T }`
- `Rect[T any]` — `{ Left, Right, Top, Bottom T }`
- `Line[T any]` — `{ Start, End T }`
- `MinMax[Min, Max any]` — `{ Min Min, Max Max }`
- `InBothAbsAxis[T any]` — `{ Horizontal, Vertical T }`
- `AbsoluteAxis` — enum: `Horizontal`, `Vertical`
- `AbstractAxis` — enum: `Inline`, `Block`

**Key methods to port (all of them from the Rust source):**

For `AbsoluteAxis`:
- `otherAxis() AbsoluteAxis`
- Methods to convert to AbstractAxis

For `AbstractAxis`:
- `otherAxis() AbstractAxis`
- `intoAbsoluteAxis(direction Direction) AbsoluteAxis` — note: needs Direction, forward-declare or use the type

For `Rect[T]`:
- `horizontalAxis() Line[T]`
- `verticalAxis() Line[T]`
- `getAxis(axis AbsoluteAxis) Line[T]`
- Mapping/zipping methods

For `Size[T]`:
- `getAxis(axis AbsoluteAxis) T`
- `setAxis(axis AbsoluteAxis, value T)`
- `map(f func(T) R) Size[R]` — use concrete types where needed
- `maybeApplyAspectRatio(...)` for `Size[Option[float32]]`

For `Point[T]`:
- `getAxis(axis AbsoluteAxis) T`

For `Line[T]`:
- `sum()` for numeric types

**Steps:**

- [ ] **Step 1: Read `../taffy/src/geometry.rs` completely**

- [ ] **Step 2: Create `geometry.go` with all types and methods**

Port every type, const, and method from `geometry.rs`. Replace `CompactLength`-based types with the simple Tag+Value pattern. Include the `Option[T]` generic type here.

- [ ] **Step 3: Verify it compiles**

```bash
go build ./...
```

- [ ] **Step 4: Create `geometry_test.go` with tests for key methods**

Port any `#[cfg(test)]` from geometry.rs. Add tests for axis operations, map/zip methods.

- [ ] **Step 5: Run tests**

```bash
go test -v ./...
```

- [ ] **Step 6: Commit**

```bash
git add geometry.go geometry_test.go
git commit -m "feat: port geometry types (Point, Size, Rect, Line, axes)"
```

---

## Task 2: Port style_dimension.go

**Rust source:** `../taffy/src/style/dimension.rs` (~380 lines)
**Files:**
- Create: `style_dimension.go`

Port the dimension types. **Do NOT use CompactLength** — use the Tag+Value struct pattern.

**Types to port:**
- `LengthPercentage` — `{ Tag LengthPercentageTag, Value float32 }`
  - Tags: `LengthPercentageLength`, `LengthPercentagePercent`, `LengthPercentageCalc`
  - Constructors: `lpLength(val)`, `lpPercent(val)`, `lpCalc(id)`
- `LengthPercentageAuto` — `{ Tag LengthPercentageAutoTag, Value float32 }`
  - Tags: `LPALength`, `LPAPercent`, `LPAAuto`, `LPACalc`
  - Methods: `resolveToOption(context float32, calc CalcResolver) Option[float32]`, `isAuto() bool`
- `Dimension` — `{ Tag DimensionTag, Value float32 }`
  - Tags: `DimensionLength`, `DimensionPercent`, `DimensionAuto`, `DimensionCalc`
  - Methods: `isAuto() bool`, `intoOption() Option[float32]`

Port all `From` impls as conversion functions (e.g., `lpaToDimension(lpa LengthPercentageAuto) Dimension`).

**Steps:**

- [ ] **Step 1: Read `../taffy/src/style/dimension.rs` and `../taffy/src/style/compact_length.rs`**

Understand the CompactLength encoding so you can map tag values correctly.

- [ ] **Step 2: Create `style_dimension.go`**

Port all types, constructors, methods, and conversion functions.

- [ ] **Step 3: Verify it compiles**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add style_dimension.go
git commit -m "feat: port dimension types (LengthPercentage, LengthPercentageAuto, Dimension)"
```

---

## Task 3: Port style_alignment.go

**Rust source:** `../taffy/src/style/alignment.rs` (~136 lines)
**Files:**
- Create: `style_alignment.go`

**Types to port:**
- `AlignItems` — enum: `AlignItemsStart`, `AlignItemsEnd`, `AlignItemsFlexStart`, `AlignItemsFlexEnd`, `AlignItemsCenter`, `AlignItemsBaseline`, `AlignItemsStretch`
- `AlignContent` — enum: `AlignContentStart`, `AlignContentEnd`, `AlignContentFlexStart`, `AlignContentFlexEnd`, `AlignContentCenter`, `AlignContentStretch`, `AlignContentSpaceBetween`, `AlignContentSpaceEvenly`, `AlignContentSpaceAround`
- Type aliases (in Go, just reuse the types): `JustifyItems = AlignItems`, `AlignSelf = AlignItems`, `JustifySelf = AlignItems`, `JustifyContent = AlignContent`
- Method: `AlignContent.reversed() AlignContent`

**Steps:**

- [ ] **Step 1: Read `../taffy/src/style/alignment.rs`**

- [ ] **Step 2: Create `style_alignment.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add style_alignment.go
git commit -m "feat: port alignment types (AlignItems, AlignContent, JustifyContent)"
```

---

## Task 4: Port style_available_space.go

**Rust source:** `../taffy/src/style/available_space.rs` (~163 lines)
**Files:**
- Create: `style_available_space.go`

**Types to port:**
- `AvailableSpace` — `{ Tag AvailableSpaceTag, Value float32 }`
  - Tags: `AvailableSpaceDefinite`, `AvailableSpaceMinContent`, `AvailableSpaceMaxContent`
  - All methods: `isDefinite()`, `intoOption()`, `unwrapOr()`, `unwrap()`, `or()`, `orElse()`, `unwrapOrElse()`, `maybeSet()`, `mapDefiniteValue()`, `computeFreeSpace()`, `isRoughlyEqual()`
- Methods on `Size[AvailableSpace]`: `intoOptions()`, `maybeSet()`

**Steps:**

- [ ] **Step 1: Read `../taffy/src/style/available_space.rs`**

- [ ] **Step 2: Create `style_available_space.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add style_available_space.go
git commit -m "feat: port AvailableSpace type"
```

---

## Task 5: Port style_flex.go

**Rust source:** `../taffy/src/style/flex.rs` (~203 lines)
**Files:**
- Create: `style_flex.go`
- Create: `style_flex_test.go`

**Types to port:**
- `FlexDirection` — enum: `FlexDirectionRow`, `FlexDirectionColumn`, `FlexDirectionRowReverse`, `FlexDirectionColumnReverse`
  - Methods: `isRow()`, `isColumn()`, `isReverse()`, `mainAxis()`, `crossAxis()`
  - Default: `FlexDirectionRow`
- `FlexWrap` — enum: `FlexWrapNoWrap`, `FlexWrapWrap`, `FlexWrapWrapReverse`
  - Default: `FlexWrapNoWrap`
- `flexboxContainerStyle` — unexported interface (methods: `flexDirection()`, `flexWrap()`, `gap()`, `alignContent()`, `alignItems()`, `justifyContent()`)
- `flexboxItemStyle` — unexported interface (methods: `flexBasis()`, `flexGrow()`, `flexShrink()`, `alignSelf()`)

**Steps:**

- [ ] **Step 1: Read `../taffy/src/style/flex.rs`**

- [ ] **Step 2: Create `style_flex.go`**

- [ ] **Step 3: Create `style_flex_test.go`** — port the `#[cfg(test)]` module

- [ ] **Step 4: Run tests**

```bash
go test -v -run FlexDirection ./...
```

- [ ] **Step 5: Commit**

```bash
git add style_flex.go style_flex_test.go
git commit -m "feat: port flexbox style types (FlexDirection, FlexWrap)"
```

---

## Task 6: Port style_float.go

**Rust source:** `../taffy/src/style/float.rs` (~80 lines)
**Files:**
- Create: `style_float.go`

**Types to port:**
- `Float` — enum: `FloatLeft`, `FloatRight`, `FloatNone` (default: `FloatNone`)
  - Methods: `isFloated()`, `floatDirection()`
- `FloatDirection` — enum: `FloatDirectionLeft`, `FloatDirectionRight`
- `Clear` — enum: `ClearLeft`, `ClearRight`, `ClearBoth`, `ClearNone` (default: `ClearNone`)

**Steps:**

- [ ] **Step 1: Read `../taffy/src/style/float.rs`**

- [ ] **Step 2: Create `style_float.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add style_float.go
git commit -m "feat: port float style types (Float, FloatDirection, Clear)"
```

---

## Task 7: Port style_block.go

**Rust source:** `../taffy/src/style/block.rs` (~57 lines)
**Files:**
- Create: `style_block.go`

**Types to port:**
- `TextAlign` — enum: `TextAlignAuto`, `TextAlignLegacyLeft`, `TextAlignLegacyRight`, `TextAlignLegacyCenter` (default: `TextAlignAuto`)
- `blockContainerStyle` — unexported interface: `textAlign() TextAlign`
- `blockItemStyle` — unexported interface: `isTable() bool`, `float() Float`, `clear() Clear`

**Steps:**

- [ ] **Step 1: Read `../taffy/src/style/block.rs`**

- [ ] **Step 2: Create `style_block.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add style_block.go
git commit -m "feat: port block style types (TextAlign, blockContainerStyle)"
```

---

## Task 8: Port style_grid.go

**Rust source:** `../taffy/src/style/grid.rs` (~1607 lines)
**Files:**
- Create: `style_grid.go`

This is the largest style file. Port all grid-related types.

**Types to port:**
- `GridAutoFlow` — enum: `GridAutoFlowRow`, `GridAutoFlowColumn`, `GridAutoFlowRowDense`, `GridAutoFlowColumnDense`
  - Methods: `isDense()`, `primaryAxis()`
- `MinTrackSizingFunction` — `{ Tag MinTrackSizingTag, Value float32 }`
  - Tags: Length, Percent, Auto, MinContent, MaxContent, Calc
  - Methods: `isIntrinsic()`, `isMinOrMaxContent()`, `isFr()`, `isAuto()`, `isMinContent()`, `isMaxContent()`, `definiteValue()`, `resolvedPercentageSize()`, `usesPercentage()`
- `MaxTrackSizingFunction` — `{ Tag MaxTrackSizingTag, Value float32 }`
  - Tags: Length, Percent, Auto, MinContent, MaxContent, FitContentPx, FitContentPercent, Fr, Calc
  - Methods: `isIntrinsic()`, `isMaxContentAlike()`, `isFr()`, `isAuto()`, etc.
- `TrackSizingFunction` = `MinMax[MinTrackSizingFunction, MaxTrackSizingFunction]`
- `RepetitionCount` — enum: `AutoFill`, `AutoFit`, `Count(uint16)`
- `GridPlacement` — `{ Tag GridPlacementTag, ... }` with Auto, Line, NamedLine, Span, NamedSpan variants
  - Complex type: needs both `int16` index and `string` name fields
- `GridTemplateComponent` — enum with `Single(TrackSizingFunction)` and `Repeat(...)` variants
- `GridTemplateArea` — struct with name, row/col start/end
- `gridContainerStyle` — unexported interface
- `gridItemStyle` — unexported interface
- All coordinate types: `GridLine`, `OriginZeroLine`, `GridCoordinate` interface
- `GenericGridPlacement[T]` — generic placement with Line/Span/Auto

**Steps:**

- [ ] **Step 1: Read `../taffy/src/style/grid.rs` completely**

- [ ] **Step 2: Create `style_grid.go` — port all types, enums, and methods**

This is a large file. Port methodically top-to-bottom from the Rust source.

- [ ] **Step 3: Verify it compiles**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add style_grid.go
git commit -m "feat: port grid style types (GridPlacement, TrackSizing, GridAutoFlow)"
```

---

## Task 9: Port style.go (main Style struct + core enums)

**Rust source:** `../taffy/src/style/mod.rs` (~1338 lines)
**Files:**
- Create: `style.go`
- Create: `style_test.go`

**Types to port:**
- `Display` — enum: `DisplayBlock`, `DisplayFlex`, `DisplayGrid`, `DisplayNone`
  - Default: `DisplayFlex` (matching Taffy's default)
- `BoxGenerationMode` — enum: `BoxGenerationModeNormal`, `BoxGenerationModeNone`
- `Position` — enum: `PositionRelative`, `PositionAbsolute` (default: `PositionRelative`)
- `BoxSizing` — enum: `BoxSizingBorderBox`, `BoxSizingContentBox` (default: `BoxSizingBorderBox`)
- `Overflow` — enum: `OverflowVisible`, `OverflowClip`, `OverflowHidden`, `OverflowScroll` (default: `OverflowVisible`)
  - Methods: `isScrollContainer()`, `maybeIntoAutomaticMinSize()`
- `Direction` — enum: `DirectionLTR`, `DirectionRTL` (default: `DirectionLTR`)
  - Methods: `isRTL()`
- `coreStyle` — unexported interface with all CoreStyle methods
- `Style` — exported struct with ALL fields from Rust (no feature gates, include all):
  - `Display`, `ItemIsTable`, `ItemIsReplaced`, `BoxSizing`, `Direction`
  - `Overflow Point[Overflow]`, `ScrollbarWidth float32`
  - `Float`, `Clear`
  - `Position`, `Inset Rect[LengthPercentageAuto]`
  - `Size, MinSize, MaxSize Size[Dimension]`
  - `AspectRatio Option[float32]`
  - `Margin Rect[LengthPercentageAuto]`, `Padding, Border Rect[LengthPercentage]`
  - `AlignItems, AlignSelf, JustifyItems, JustifySelf Option[AlignItems]`
  - `AlignContent, JustifyContent Option[AlignContent]`
  - `Gap Size[LengthPercentage]`
  - `TextAlign`
  - `FlexDirection`, `FlexWrap`, `FlexBasis Dimension`, `FlexGrow, FlexShrink float32`
  - All grid fields
- `DefaultStyle() Style` — returns the default style matching Rust's `Style::DEFAULT`
- Implement all style trait interfaces on `*Style`

**Steps:**

- [ ] **Step 1: Read `../taffy/src/style/mod.rs` completely**

- [ ] **Step 2: Create `style.go` — port all enums, the Style struct, DefaultStyle(), and trait implementations**

- [ ] **Step 3: Create `style_test.go`** — port the `#[cfg(test)]` module (defaults_match test)

- [ ] **Step 4: Run tests**

```bash
go test -v -run Style ./...
```

- [ ] **Step 5: Commit**

```bash
git add style.go style_test.go
git commit -m "feat: port Style struct and core style enums (Display, Position, Overflow)"
```

---

## Task 10: Port style_helpers.go

**Rust source:** `../taffy/src/style_helpers.rs` (~577 lines)
**Files:**
- Create: `style_helpers.go`

Port the helper traits as concrete functions (Go doesn't need the trait-based overloading):

**Functions to port:**
- `zero[T]() T` — uses TaffyZero, make concrete per-type: `zeroLP()`, `zeroDimension()`, etc.
- `auto[T]() T` — similarly concrete per-type
- `length[T](val float32) T` — concrete per-type: `lpFromLength()`, `dimensionFromLength()`, etc.
- `percent[T](val float32) T` — concrete per-type
- `minContent[T]() T`, `maxContent[T]() T` — concrete per-type
- `fitContent[T](arg LengthPercentage) T` — for track sizing
- `fr(val float32)` — for grid track sizing
- `line(index int16)`, `span(span uint16)` — for grid placement
- `repeat(...)`, `evenlySpacedTracks(count)` — for grid templates
- `minmax(min, max)` — for track sizing

**Steps:**

- [ ] **Step 1: Read `../taffy/src/style_helpers.rs`**

- [ ] **Step 2: Create `style_helpers.go`** — port all helper functions

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add style_helpers.go
git commit -m "feat: port style helper functions (length, percent, auto, grid helpers)"
```

---

## Task 11: Port node.go

**Rust source:** `../taffy/src/tree/node.rs` (~60 lines)
**Files:**
- Create: `node.go`

**Types to port:**
- `NodeId` — `type NodeId uint64`
  - Constructor: `NewNodeId(val uint64) NodeId`
  - `invalidNodeId` constant for sentinel

**Steps:**

- [ ] **Step 1: Read `../taffy/src/tree/node.rs`**

- [ ] **Step 2: Create `node.go`**

```go
package taffy

type NodeId uint64

const invalidNodeId NodeId = ^NodeId(0)

func NewNodeId(val uint64) NodeId {
    return NodeId(val)
}
```

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add node.go
git commit -m "feat: port NodeId type"
```

---

## Task 12: Port layout.go

**Rust source:** `../taffy/src/tree/layout.rs` (~357 lines)
**Files:**
- Create: `layout.go`

**Types to port:**
- `RunMode` — enum: `RunModePerformLayout`, `RunModeComputeSize`, `RunModePerformHiddenLayout`
- `SizingMode` — enum: `SizingModeContentSize`, `SizingModeInherentSize`
- `RequestedAxis` — enum: `RequestedAxisHorizontal`, `RequestedAxisVertical`, `RequestedAxisBoth`
- `CollapsibleMarginSet` — struct with `Positive, Negative float32`
  - Methods: `fromMargin()`, `collapseWithMargin()`, `collapseWithSet()`, `resolve()`
  - Constant: `zeroCollapsibleMarginSet`
- `LayoutInput` — struct with all fields from Rust
  - Constant: `hiddenLayoutInput`
- `LayoutOutput` — struct with all fields
  - Constants: `hiddenLayoutOutput`, `defaultLayoutOutput`
  - Constructors: `layoutOutputFromSizesAndBaselines()`, `layoutOutputFromSizes()`, `layoutOutputFromOuterSize()`
- `Layout` — exported struct with all fields (order, location, size, contentSize, scrollbarSize, border, padding, margin)
  - Methods: `ContentBoxWidth()`, `ContentBoxHeight()`, `ContentBoxSize()`, `ContentBoxX()`, `ContentBoxY()`, `ScrollWidth()`, `ScrollHeight()`
  - Constructor: `NewLayout()`, `NewLayoutWithOrder(order uint32)`

**Steps:**

- [ ] **Step 1: Read `../taffy/src/tree/layout.rs` completely**

- [ ] **Step 2: Create `layout.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add layout.go
git commit -m "feat: port layout types (Layout, LayoutInput, LayoutOutput, CollapsibleMarginSet)"
```

---

## Task 13: Port cache.go

**Rust source:** `../taffy/src/tree/cache.rs` (~196 lines)
**Files:**
- Create: `cache.go`

**Types to port:**
- `ClearState` — enum: `ClearStateCleared`, `ClearStateAlreadyEmpty`
- `cacheEntry[T any]` — generic struct for cached layout results with input hash
- `cache` — struct with `finalLayoutEntry` and `measureEntries` array
  - Methods: `get(input *LayoutInput) *LayoutOutput`, `store(input *LayoutInput, output LayoutOutput)`, `clear() ClearState`, `isEmpty() bool`
- Cache size constant (typically 4-7 measure entries)

**Steps:**

- [ ] **Step 1: Read `../taffy/src/tree/cache.rs` completely**

- [ ] **Step 2: Create `cache.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add cache.go
git commit -m "feat: port layout cache"
```

---

## Task 14: Port tree_traits.go

**Rust source:** `../taffy/src/tree/traits.rs` (~418 lines)
**Files:**
- Create: `tree_traits.go`

Port all traits as **unexported Go interfaces**:

**Interfaces to port:**
- `traversePartialTree` — `childIds(parent NodeId) []NodeId`, `childCount(parent NodeId) int`, `getChildId(parent NodeId, childIndex int) NodeId`
- `traverseTree` — embeds `traversePartialTree` (marker trait)
- `layoutPartialTree` — embeds `traversePartialTree`, adds: `getCoreContainerStyle(nodeId NodeId) *Style`, `resolveCalcValue(calcId uint64, basis float32) float32`, `setUnroundedLayout(nodeId NodeId, layout *Layout)`, `computeChildLayout(nodeId NodeId, inputs LayoutInput) LayoutOutput`
- `cacheTree` — `cacheGet()`, `cacheStore()`, `cacheClear()`
- `roundTree` — embeds `traverseTree`, adds: `getUnroundedLayout()`, `setFinalLayout()`
- `printTree` — embeds `traverseTree`, adds: `getDebugLabel()`, `getFinalLayout()`
- `layoutFlexboxContainer` — embeds `layoutPartialTree`, adds: `getFlexboxContainerStyle()`, `getFlexboxChildStyle()`
- `layoutGridContainer` — embeds `layoutPartialTree`, adds: `getGridContainerStyle()`, `getGridChildStyle()`
- `layoutBlockContainer` — embeds `layoutPartialTree`, adds: `getBlockContainerStyle()`, `getBlockChildStyle()`, `computeBlockChildLayout()`
- `layoutPartialTreeExt` — helper methods: `measureChildSize()`, `measureChildSizeBoth()`, `performChildLayout()`

**Note:** `layoutPartialTreeExt` in Rust is a blanket trait impl. In Go, implement these as standalone functions that take a `layoutPartialTree` parameter:

```go
func measureChildSize(tree layoutPartialTree, nodeId NodeId, ...) float32
func performChildLayout(tree layoutPartialTree, nodeId NodeId, ...) LayoutOutput
```

**Steps:**

- [ ] **Step 1: Read `../taffy/src/tree/traits.rs` completely**

- [ ] **Step 2: Create `tree_traits.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add tree_traits.go
git commit -m "feat: port tree trait interfaces (layoutPartialTree, traverseTree, etc.)"
```

---

## Task 15: Port util_sys.go

**Rust source:** `../taffy/src/util/sys.rs` (~301 lines)
**Files:**
- Create: `util_sys.go`

Only port the `std` branch (we're in Go, always have std). Most of this maps to Go stdlib.

**Functions to port:**
- `f32Max(a, b float32) float32` — `max(a, b)` (Go 1.21+)
- `f32Min(a, b float32) float32` — `min(a, b)` (Go 1.21+)
- `round(value float32) float32` — `float32(math.Round(float64(value)))`
- `ceil(value float32) float32`
- `floor(value float32) float32`
- `abs(value float32) float32`

Most of these can be thin wrappers around Go's `math` package. Some can even be inlined at call sites. Keep them as unexported functions to maintain 1:1 mapping with Rust.

**Steps:**

- [ ] **Step 1: Read `../taffy/src/util/sys.rs`**

- [ ] **Step 2: Create `util_sys.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add util_sys.go
git commit -m "feat: port sys utilities (math wrappers)"
```

---

## Task 16: Port util_math.go

**Rust source:** `../taffy/src/util/math.rs` (~356 lines)
**Files:**
- Create: `util_math.go`
- Create: `util_math_test.go`

The Rust code uses trait-based overloading (`MaybeMath`) for operations on `Option<f32>`, `f32`, and `AvailableSpace`. In Go, port these as standalone functions:

**Functions to port:**
- `maybeMinOption(lhs Option[float32], rhs Option[float32]) Option[float32]`
- `maybeMaxOption(lhs Option[float32], rhs Option[float32]) Option[float32]`
- `maybeAddOption(lhs Option[float32], rhs Option[float32]) Option[float32]`
- `maybeSubOption(lhs Option[float32], rhs Option[float32]) Option[float32]`
- `maybeClampOption(lhs Option[float32], min Option[float32], max Option[float32]) Option[float32]`
- `maybeMinF32(lhs float32, rhs Option[float32]) float32` — f32 with Option<f32>
- `maybeMaxF32(lhs float32, rhs Option[float32]) float32`
- `maybeAddF32(lhs float32, rhs Option[float32]) float32`
- `maybeSubF32(lhs float32, rhs Option[float32]) float32`
- `maybeClampF32(lhs float32, min Option[float32], max Option[float32]) float32`
- `maybeMinAvailableSpace(lhs AvailableSpace, rhs float32) AvailableSpace`
- etc. for AvailableSpace variants
- `Size[T]` versions of the above

**Steps:**

- [ ] **Step 1: Read `../taffy/src/util/math.rs` completely**

- [ ] **Step 2: Create `util_math.go`**

- [ ] **Step 3: Create `util_math_test.go`** — port the `#[cfg(test)]` module (all test cases)

- [ ] **Step 4: Run tests**

```bash
go test -v -run MaybeMath ./...
```

- [ ] **Step 5: Commit**

```bash
git add util_math.go util_math_test.go
git commit -m "feat: port MaybeMath utility functions"
```

---

## Task 17: Port util_resolve.go

**Rust source:** `../taffy/src/util/resolve.rs` (~351 lines)
**Files:**
- Create: `util_resolve.go`
- Create: `util_resolve_test.go`

Port `MaybeResolve` and `ResolveOrZero` traits as functions:

**Functions to port:**
- `maybeResolveLPOption(lp LengthPercentage, context Option[float32], calc CalcResolver) Option[float32]`
- `maybeResolveLPAOption(lpa LengthPercentageAuto, context Option[float32], calc CalcResolver) Option[float32]`
- `maybeResolveDimensionOption(dim Dimension, context Option[float32], calc CalcResolver) Option[float32]`
- `maybeResolveLPF32(lp LengthPercentage, context float32, calc CalcResolver) Option[float32]`
- Same for Size and Rect versions
- `resolveOrZeroLP(lp LengthPercentage, context Option[float32], calc CalcResolver) float32`
- `resolveOrZeroLPA(lpa LengthPercentageAuto, context Option[float32], calc CalcResolver) float32`
- `resolveOrZeroDimension(dim Dimension, context Option[float32], calc CalcResolver) float32`
- Size and Rect versions

**Steps:**

- [ ] **Step 1: Read `../taffy/src/util/resolve.rs` completely**

- [ ] **Step 2: Create `util_resolve.go`**

- [ ] **Step 3: Create `util_resolve_test.go`** — port all `#[cfg(test)]` cases

- [ ] **Step 4: Run tests**

```bash
go test -v -run Resolve ./...
```

- [ ] **Step 5: Commit**

```bash
git add util_resolve.go util_resolve_test.go
git commit -m "feat: port resolution utilities (MaybeResolve, ResolveOrZero)"
```

---

## Task 18: Port util_print.go

**Rust source:** `../taffy/src/util/print.rs` (~78 lines)
**Files:**
- Create: `util_print.go`

**Functions to port:**
- `printTree(tree printTree, root NodeId)` — prints to stdout
- `writeTree(w io.Writer, tree printTree, root NodeId) error` — writes to any writer
- `writeNode(w io.Writer, tree printTree, nodeId NodeId, hasSibling bool, linesString string) error` — recursive helper

**Steps:**

- [ ] **Step 1: Read `../taffy/src/util/print.rs`**

- [ ] **Step 2: Create `util_print.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add util_print.go
git commit -m "feat: port tree printing utility"
```

---

## Task 19: Port taffy_tree.go

**Rust source:** `../taffy/src/tree/taffy_tree.rs` (~1405 lines)
**Files:**
- Create: `taffy_tree.go`
- Create: `taffy_tree_test.go`

This is the core public API. Port `TaffyTree`, `TaffyView`, and all trait implementations.

**Types to port:**

`nodeData` — unexported struct:
- `style Style`
- `unroundedLayout Layout`
- `finalLayout Layout`
- `cache cache`
- `hasNodeContext bool`

`nodeStore` — unexported interface:
```go
type nodeStore interface {
    getNode(id NodeId) *nodeData
    addNode(data nodeData) NodeId
    removeNode(id NodeId)
    has(id NodeId) bool
}
```

`mapNodeStore` — implementation:
```go
type mapNodeStore struct {
    nodes  map[NodeId]*nodeData
    nextId uint64
}
```

`TaffyTree` — exported struct:
```go
type TaffyTree struct {
    store        *mapNodeStore
    nodeContexts map[NodeId]NodeContext
    children     map[NodeId][]NodeId
    parents      map[NodeId]*NodeId
    config       taffyConfig
}
```

`NodeContext` — type alias:
```go
type NodeContext = func(
    knownDimensions Size[Option[float32]],
    parentSize Size[Option[float32]],
    availableSpace Size[AvailableSpace],
    sizingMode SizingMode,
) Size[float32]
```

**Public methods to port:**
- `New() *TaffyTree`
- `NewWithCapacity(capacity int) *TaffyTree`
- `EnableRounding()` / `DisableRounding()`
- `NewLeaf(style Style) NodeId`
- `NewLeafWithContext(style Style, context NodeContext) NodeId`
- `NewWithChildren(style Style, children []NodeId) NodeId`
- `Clear()`
- `Remove(node NodeId) NodeId`
- `SetNodeContext(node NodeId, measure *NodeContext)`
- `GetNodeContext(node NodeId) *NodeContext`
- `AddChild(parent, child NodeId)`
- `InsertChildAtIndex(parent NodeId, index int, child NodeId)`
- `SetChildren(parent NodeId, children []NodeId)`
- `RemoveChild(parent, child NodeId) NodeId`
- `RemoveChildAtIndex(parent NodeId, index int) NodeId`
- `ReplaceChildAtIndex(parent NodeId, index int, newChild NodeId) NodeId`
- `ChildAtIndex(parent NodeId, index int) NodeId`
- `TotalNodeCount() int`
- `Parent(child NodeId) *NodeId`
- `Children(parent NodeId) []NodeId`
- `SetStyle(node NodeId, style Style)`
- `GetStyle(node NodeId) *Style`
- `GetLayout(node NodeId) *Layout`
- `MarkDirty(node NodeId)`
- `IsDirty(node NodeId) bool`
- `ComputeLayout(node NodeId, availableSpace Size[AvailableSpace])`
- `ComputeLayoutWithMeasure(node NodeId, availableSpace Size[AvailableSpace], measureFn MeasureFunction)`
- `PrintTree(root NodeId)`

**Trait implementations:**
Port `impl TraversePartialTree`, `TraverseTree`, `CacheTree`, `PrintTree` as methods on `*TaffyTree`.

**taffyView:** Port as an unexported struct wrapping `*TaffyTree` + measure function. Implement all layout trait interfaces on it.

**Steps:**

- [ ] **Step 1: Read `../taffy/src/tree/taffy_tree.rs` completely**

- [ ] **Step 2: Create `taffy_tree.go` — port all types, methods, and trait implementations**

- [ ] **Step 3: Create `taffy_tree_test.go`** — port the `#[cfg(test)]` module

- [ ] **Step 4: Run tests**

```bash
go test -v -run TaffyTree ./...
```

- [ ] **Step 5: Commit**

```bash
git add taffy_tree.go taffy_tree_test.go
git commit -m "feat: port TaffyTree public API and node management"
```

---

## Task 20: Port compute.go

**Rust source:** `../taffy/src/compute/mod.rs` (~337 lines)
**Files:**
- Create: `compute.go`

**Functions to port:**
- `computeRootLayout(tree layoutPartialTree, root NodeId, availableSpace Size[AvailableSpace])`
- `computeCachedLayout(tree cacheTree, node NodeId, inputs LayoutInput, computeUncached func) LayoutOutput`
- `roundLayout(tree roundTree, nodeId NodeId)`
- `computeHiddenLayout(tree layoutPartialTree, node NodeId) LayoutOutput`
- `roundLayoutInner(tree roundTree, nodeId NodeId, cumulativeX, cumulativeY float32)` — recursive helper
- `roundContentSize(layout *Layout, unroundedContentSize Size[float32], cumulativeX, cumulativeY float32)` — helper

**Steps:**

- [ ] **Step 1: Read `../taffy/src/compute/mod.rs` completely**

- [ ] **Step 2: Create `compute.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add compute.go
git commit -m "feat: port compute dispatch (root layout, caching, rounding, hidden)"
```

---

## Task 21: Port compute_leaf.go

**Rust source:** `../taffy/src/compute/leaf.rs` (~164 lines)
**Files:**
- Create: `compute_leaf.go`

**Functions to port:**
- `computeLeafLayout(inputs LayoutInput, style coreStyle, resolveCalcValue CalcResolver, measureFunction func(Size[Option[float32]], Size[AvailableSpace]) Size[float32]) LayoutOutput`

**Steps:**

- [ ] **Step 1: Read `../taffy/src/compute/leaf.rs`**

- [ ] **Step 2: Create `compute_leaf.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add compute_leaf.go
git commit -m "feat: port leaf layout computation"
```

---

## Task 22: Port compute_alignment.go

**Rust source:** `../taffy/src/compute/common/alignment.rs` (~100 lines)
**Files:**
- Create: `compute_alignment.go`

**Functions to port:**
- `applyAlignmentFallback(freeSpace float32, numItems int, alignmentMode AlignContent, isSafe bool) AlignContent`
- `computeAlignmentOffset(freeSpace float32, numItems int, gap float32, alignmentMode AlignContent, layoutIsFlexReversed bool, isFirst bool) float32`

**Steps:**

- [ ] **Step 1: Read `../taffy/src/compute/common/alignment.rs`**

- [ ] **Step 2: Create `compute_alignment.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add compute_alignment.go
git commit -m "feat: port alignment computation utilities"
```

---

## Task 23: Port compute_flexbox.go

**Rust source:** `../taffy/src/compute/flexbox.rs` (~2517 lines)
**Files:**
- Create: `compute_flexbox.go`

This is the largest single algorithm file. Port all functions.

**Internal types to port:**
- `flexItem` — struct with all fields (node, size, minSize, maxSize, margins, padding, border, flexBasis, flexGrow, flexShrink, etc.)
- `flexLine` — struct (items slice, crossSize, offset)
- `algoConstants` — struct (direction, isRow, isColumn, isWrap, etc.)

**Functions to port (all unexported):**
- `computeFlexboxLayout(tree layoutFlexboxContainer, node NodeId, inputs LayoutInput) LayoutOutput`
- `computePreliminary(...)` — main algorithm
- `computeConstants(...)` — extract style constants
- `generateAnonymousFlexItems(...)` — build flex items from children
- `determineAvailableSpace(...)` — compute available space
- `determineFlexBaseSize(...)` — flex base size per item
- `collectFlexItemsIntoFlexLines(...)` — wrap items into lines
- `determineContainerMainSize(...)` — main axis size
- `resolveFlexibleLengths(...)` — the core flex algorithm
- `determineHypotheticalCrossSize(...)` — cross axis sizing
- `calculateChildrenBaseLines(...)` — baseline computation
- `calculateCrossSize(...)` — line cross sizes
- `handleAlignContentStretch(...)` — stretch alignment
- `determineUsedCrossSize(...)` — final cross sizes
- `distributeRemainingFreeSpace(...)` — main axis distribution
- `resolveCrossAxisAutoMargins(...)` — auto margin resolution
- `alignFlexItemsAlongCrossAxis(...)` — cross alignment
- `determineContainerCrossSize(...)` — container cross
- `alignFlexLinesPerAlignContent(...)` — multi-line alignment
- `calculateFlexItem(...)` — per-item layout
- `calculateLayoutLine(...)` — per-line layout
- `finalLayoutPass(...)` — final positioning
- `performAbsoluteLayoutOnAbsoluteChildren(...)` — absolute positioning
- `sumAxisGaps(...)` — gap calculation

**Steps:**

- [ ] **Step 1: Read `../taffy/src/compute/flexbox.rs` completely**

- [ ] **Step 2: Create `compute_flexbox.go` — port all types and functions**

Port methodically, function by function, maintaining the same structure as Rust.

- [ ] **Step 3: Verify it compiles**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add compute_flexbox.go
git commit -m "feat: port flexbox layout algorithm"
```

---

## Task 24: Port compute_float.go

**Rust source:** `../taffy/src/compute/float.rs` (~552 lines)
**Files:**
- Create: `compute_float.go`

**Types to port:**
- `floatSegment` — struct for vertical segments of available space
- `placedFloatedBox` — struct with position, size, direction
- `floatContext` — struct managing float placement
  - Methods: `setWidth()`, `hasFloats()`, `hasActiveFloats()`, `placeFloatedBox()`, `findContentSlot()`, `clearedThreshold()`, etc.
- `contentSlot` — struct with position, width, index
- `floatIntrinsicWidthCalculator` — struct for intrinsic width computation

**Steps:**

- [ ] **Step 1: Read `../taffy/src/compute/float.rs` completely**

- [ ] **Step 2: Create `compute_float.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add compute_float.go
git commit -m "feat: port float layout algorithm"
```

---

## Task 25: Port compute_block.go

**Rust source:** `../taffy/src/compute/block.rs` (~1311 lines)
**Files:**
- Create: `compute_block.go`

**Types to port:**
- `blockItem` — struct (node, order, size, minSize, maxSize, etc.)
- `blockContext` — struct managing block formatting context with float integration
  - Methods: `newBlockContext()`, `rootBlockContext()`, `subContext()`, `isBfcRoot()`, `setWidth()`, `placeFloatedBox()`, `findContentSlot()`, etc.

**Functions to port:**
- `computeBlockLayout(tree layoutBlockContainer, nodeId NodeId, inputs LayoutInput, blockCtx *blockContext) LayoutOutput`
- `computeInner(...)` — main algorithm
- `generateItemList(...)` — build block items
- `determineContentBasedContainerWidth(...)` — intrinsic width
- `performFinalLayoutOnInFlowChildren(...)` — position children
- `performAbsoluteLayoutOnAbsoluteChildren(...)` — absolute positioning

**Steps:**

- [ ] **Step 1: Read `../taffy/src/compute/block.rs` completely**

- [ ] **Step 2: Create `compute_block.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add compute_block.go
git commit -m "feat: port block layout algorithm"
```

---

## Task 26: Port compute_grid_types_coords.go

**Rust source:** `../taffy/src/compute/grid/types/coordinates.rs` (~150 lines)
**Files:**
- Create: `compute_grid_types_coords.go`

**Types to port:**
- `gridLine` — `type gridLine int16` (1-based grid line index)
  - Methods: `asI16()`, `intoOriginZeroLine(explicitTrackCount uint16)`
- `originZeroLine` — `type originZeroLine int16` (0-based grid line index)
  - Methods: `intoTrackVecIndex(trackCounts)`, `tryIntoTrackVecIndex()`, `impliedNegativeImplicitTracks()`, `impliedPositiveImplicitTracks()`
  - Arithmetic: implement Add/Sub as methods or functions
- `gridCoordinate` — not needed as interface in Go, use concrete types
- `trackCounts` — struct with `negativeImplicit`, `explicit`, `positiveImplicit uint16`
  - Methods to compute total counts and convert between coordinate systems

**Steps:**

- [ ] **Step 1: Read `../taffy/src/compute/grid/types/coordinates.rs`**

- [ ] **Step 2: Create `compute_grid_types_coords.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add compute_grid_types_coords.go
git commit -m "feat: port grid coordinate types (gridLine, originZeroLine, trackCounts)"
```

---

## Task 27: Port compute_grid_types_track.go

**Rust source:** `../taffy/src/compute/grid/types/grid_track.rs` (~156 lines)
**Files:**
- Create: `compute_grid_types_track.go`

**Types to port:**
- `gridTrackKind` — enum: `gridTrackKindTrack`, `gridTrackKindGutter`
- `gridTrack` — struct with all fields (kind, isCollapsed, minTrackSizingFunction, maxTrackSizingFunction, offset, baseSize, growthLimit, contentAlignmentAdjustment, itemIncurredIncrease, baseSizePlannedIncrease, growthLimitPlannedIncrease, infinitelyGrowable)
  - Methods: `newGridTrack()`, `newGridTrackWithKind()`, `gutter()`, `collapse()`, `isFlexible()`, `usesPercentage()`, `hasIntrinsicSizingFunction()`, `fitContentLimit()`, `fitContentLimitedGrowthLimit()`, `flexFactor()`

**Steps:**

- [ ] **Step 1: Read `../taffy/src/compute/grid/types/grid_track.rs`**

- [ ] **Step 2: Create `compute_grid_types_track.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add compute_grid_types_track.go
git commit -m "feat: port GridTrack type"
```

---

## Task 28: Port compute_grid_types_occupancy.go

**Rust source:** `../taffy/src/compute/grid/types/cell_occupancy.rs` (~301 lines)
**Files:**
- Create: `compute_grid_types_occupancy.go`

**Types to port:**
- `cellOccupancyState` — enum: `cellUnoccupied`, `cellDefinitelyPlaced`, `cellAutoPlaced`
- `cellOccupancyMatrix` — 2D grid tracking cell occupation
  - Uses a 2D slice or flat slice with row/column indexing
  - Methods: `withTrackCounts()`, `isAreaInRange()`, `expandToFitRange()`, `markAreaAs()`, `lineAreaIsUnoccupied()`, `trackAreaIsUnoccupied()`, `rowIsOccupied()`, `columnIsOccupied()`, `trackCounts()`, `lastOfType()`, `firstOfType()`

**Note:** Rust uses the `grid` crate for the 2D matrix. In Go, use a flat `[]cellOccupancyState` slice with manual row/column indexing, or a `[][]cellOccupancyState`.

**Steps:**

- [ ] **Step 1: Read `../taffy/src/compute/grid/types/cell_occupancy.rs`**

- [ ] **Step 2: Create `compute_grid_types_occupancy.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add compute_grid_types_occupancy.go
git commit -m "feat: port CellOccupancyMatrix"
```

---

## Task 29: Port compute_grid_types_item.go

**Rust source:** `../taffy/src/compute/grid/types/grid_item.rs` (~555 lines)
**Files:**
- Create: `compute_grid_types_item.go`

**Types to port:**
- `gridItem` — large struct with all fields (node, sourceOrder, row, column, style fields, caches, etc.)
  - All methods: `newWithPlacementStyleAndOrder()`, `placement()`, `placementIndexes()`, `trackRangeExcludingLines()`, `span()`, `crossesFlexibleTrack()`, `crossesIntrinsicTrack()`, `spannedTrackLimit()`, `spannedFixedTrackLimit()`, `knownDimensions()`, `availableSpace()`, `availableSpaceCached()`, `marginsAxisSumsWithBaselineShims()`, `minContentContribution()`, `minContentContributionCached()`, `maxContentContribution()`, `maxContentContributionCached()`, `minimumContribution()`, `minimumContributionCached()`

**Steps:**

- [ ] **Step 1: Read `../taffy/src/compute/grid/types/grid_item.rs` completely**

- [ ] **Step 2: Create `compute_grid_types_item.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add compute_grid_types_item.go
git commit -m "feat: port GridItem type with contribution computation"
```

---

## Task 30: Port compute_grid_types_named.go

**Rust source:** `../taffy/src/compute/grid/types/named.rs` (~428 lines)
**Files:**
- Create: `compute_grid_types_named.go`

**Types to port:**
- `namedLineResolver` — struct with maps for row/column line names and grid areas
  - Methods: `newNamedLineResolver()`, `resolveRowNames()`, `resolveColumnNames()`, `resolveLineNames()`, `findLineIndex()`, `areaColumnCount()`, `areaRowCount()`, `setExplicitColumnCount()`, `setExplicitRowCount()`

**Note:** The Rust code uses `StrHasher` for case-sensitive string hashing. In Go, just use `string` as map keys directly.

**Steps:**

- [ ] **Step 1: Read `../taffy/src/compute/grid/types/named.rs`**

- [ ] **Step 2: Create `compute_grid_types_named.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add compute_grid_types_named.go
git commit -m "feat: port named grid line resolver"
```

---

## Task 31: Port compute_grid_explicit.go

**Rust source:** `../taffy/src/compute/grid/explicit_grid.rs` (~697 lines)
**Files:**
- Create: `compute_grid_explicit.go`

**Functions to port:**
- `computeExplicitGridSizeInAxis(style gridContainerStyle, ...) (uint16, uint16)`
- `initializeGridTracks(tracks *[]gridTrack, counts trackCounts, style gridContainerStyle, ...)`
- `createImplicitTracks(tracks *[]gridTrack, count uint16, autoTracks []TrackSizingFunction, gap LengthPercentage)`

**Steps:**

- [ ] **Step 1: Read `../taffy/src/compute/grid/explicit_grid.rs`**

- [ ] **Step 2: Create `compute_grid_explicit.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add compute_grid_explicit.go
git commit -m "feat: port explicit grid construction"
```

---

## Task 32: Port compute_grid_implicit.go

**Rust source:** `../taffy/src/compute/grid/implicit_grid.rs` (~253 lines)
**Files:**
- Create: `compute_grid_implicit.go`

**Functions to port:**
- `computeGridSizeEstimate(explicitColCount, explicitRowCount uint16, direction Direction, childStyles ...) (trackCounts, trackCounts)`
- `getKnownChildPositions(...)` — scan children for explicit placements
- `childMinLineMaxLineSpan(line Line[GridPlacement], explicitTrackCount uint16) (originZeroLine, originZeroLine, uint16)`

**Steps:**

- [ ] **Step 1: Read `../taffy/src/compute/grid/implicit_grid.rs`**

- [ ] **Step 2: Create `compute_grid_implicit.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add compute_grid_implicit.go
git commit -m "feat: port implicit grid estimation"
```

---

## Task 33: Port compute_grid_placement.go

**Rust source:** `../taffy/src/compute/grid/placement.rs` (~764 lines)
**Files:**
- Create: `compute_grid_placement.go`

**Functions to port:**
- `placeGridItems(cellOccupancy *cellOccupancyMatrix, items *[]gridItem, ...)` — main placement algorithm
- `axisIsReversed(direction Direction, axis AbsoluteAxis) bool`
- `advancePosition(...)`, `searchStartLine(...)` — position helpers
- `resolveIndefiniteGridSpan(...)` — span resolution
- `mirrorHorizontalSpan(...)`, `maybeMirrorSpan(...)` — RTL mirroring
- `placeDefiniteGridItem(...)` — place fully-specified items
- `placeDefiniteSecondaryAxisItem(...)` — one axis definite
- `placeIndefinitelyPositionedItem(...)` — auto-placement

**Steps:**

- [ ] **Step 1: Read `../taffy/src/compute/grid/placement.rs` completely**

- [ ] **Step 2: Create `compute_grid_placement.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add compute_grid_placement.go
git commit -m "feat: port grid item placement algorithm"
```

---

## Task 34: Port compute_grid_track_sizing.go

**Rust source:** `../taffy/src/compute/grid/track_sizing.rs` (~1410 lines)
**Files:**
- Create: `compute_grid_track_sizing.go`

This is the most complex grid file. Port all functions.

**Types to port:**
- `itemBatcher` — struct for batching items by track span
- `intrinsicSizeMeasurer` — struct wrapping tree + measurement context
- `intrinsicContributionType` — enum for contribution types

**Functions to port:**
- `initializeTrackSizes(...)` — set initial base sizes and growth limits
- `resolveItemBaselines(...)` — baseline computation
- All `distributeItemSpace*` functions
- `maximiseTracks(...)` — maximize track sizes
- `expandFlexibleTracks(...)` — handle fr units
- `findSizeOfFr(...)` — determine fr unit size
- `stretchAutoTracks(...)` — stretch auto tracks
- `distributeSpaceUpToLimits(...)` — space distribution

**Steps:**

- [ ] **Step 1: Read `../taffy/src/compute/grid/track_sizing.rs` completely**

- [ ] **Step 2: Create `compute_grid_track_sizing.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add compute_grid_track_sizing.go
git commit -m "feat: port grid track sizing algorithm"
```

---

## Task 35: Port compute_grid_alignment.go

**Rust source:** `../taffy/src/compute/grid/alignment.rs` (~366 lines)
**Files:**
- Create: `compute_grid_alignment.go`

**Functions to port:**
- `alignTracks(gridContainerContentBoxSize float32, padding, border Line[float32], tracks *[]gridTrack, trackAlignmentStyle AlignContent, axisIsReversed bool)`
- `alignAndPositionItem(tree layoutGridContainer, node NodeId, order uint32, gridArea Rect[float32], containerAlignmentStyles InBothAbsAxis[Option[AlignItems]], baselineShim float32, direction Direction) (Size[float32], float32, float32)`
- `alignItemWithinArea(gridArea Line[float32], alignmentStyle AlignSelf, resolvedSize float32, position Position, inset, margin Line[Option[float32]], baselineShim float32, direction Direction) (float32, Line[float32])`

**Steps:**

- [ ] **Step 1: Read `../taffy/src/compute/grid/alignment.rs`**

- [ ] **Step 2: Create `compute_grid_alignment.go`**

- [ ] **Step 3: Verify it compiles**

- [ ] **Step 4: Commit**

```bash
git add compute_grid_alignment.go
git commit -m "feat: port grid alignment"
```

---

## Task 36: Port compute_grid.go

**Rust source:** `../taffy/src/compute/grid/mod.rs` (~846 lines)
**Files:**
- Create: `compute_grid.go`

This is the grid layout entry point that orchestrates all the grid sub-modules.

**Functions to port:**
- `computeGridLayout(tree layoutGridContainer, node NodeId, inputs LayoutInput) LayoutOutput` — main entry
- The main algorithm body which calls into explicit_grid, implicit_grid, placement, track_sizing, and alignment modules

**Steps:**

- [ ] **Step 1: Read `../taffy/src/compute/grid/mod.rs` completely**

- [ ] **Step 2: Create `compute_grid.go`**

- [ ] **Step 3: Verify it compiles**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add compute_grid.go
git commit -m "feat: port grid layout algorithm entry point"
```

---

## Task 37: Wire up ComputeLayout dispatch

**Files:**
- Modify: `taffy_tree.go`

After all compute algorithms are ported, wire up `TaffyTree.ComputeLayout` to dispatch to the correct algorithm based on `Display`:

```go
func (t *TaffyTree) ComputeLayout(node NodeId, availableSpace Size[AvailableSpace]) {
    // Create taffyView with default measure function
    view := &taffyView{taffy: t, measureFunction: defaultMeasureFunction}
    computeRootLayout(view, node, availableSpace)
    if t.config.useRounding {
        roundLayout(view, node)
    }
}
```

The dispatch in `computeRootLayout` → `computeChildLayout` should switch on `Display`:
- `DisplayFlex` → `computeFlexboxLayout`
- `DisplayGrid` → `computeGridLayout`
- `DisplayBlock` → `computeBlockLayout`
- `DisplayNone` → `computeHiddenLayout`

**Steps:**

- [ ] **Step 1: Read current `taffy_tree.go` to understand `taffyView` and `computeChildLayout`**

- [ ] **Step 2: Wire up the display-based dispatch in `taffyView.computeChildLayout`**

- [ ] **Step 3: Verify it compiles**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add taffy_tree.go
git commit -m "feat: wire up layout dispatch (flex, grid, block, float)"
```

---

## Task 38: Port XML fixture test infrastructure

**Files:**
- Create: `fixtures_helpers_test.go`

Build the test infrastructure that reads XML fixtures and runs them through the layout engine.

**Functions to create:**

```go
// loadFixture parses a single XML fixture file
func loadFixture(path string) *testFixture

// testFixture holds parsed test data
type testFixture struct {
    Name        string
    UseRounding bool
    Viewport    Size[AvailableSpace]
    InputRoot   *testNode
    Expected    *expectedNode
}

type testNode struct {
    Style    Style
    Children []*testNode
    Text     string // for leaf nodes with text content
}

type expectedNode struct {
    X, Y, Width, Height float32
    Children            []*expectedNode
}

// buildTree constructs a TaffyTree from a testNode
func buildTree(tree *TaffyTree, node *testNode) NodeId

// buildStyle parses XML attributes into a Style
func buildStyle(attrs map[string]string) Style

// assertLayoutMatches recursively checks computed layout against expected
func assertLayoutMatches(tree *TaffyTree, nodeId NodeId, expected *expectedNode)
```

**XML parsing:** Use `encoding/xml` with a custom decoder that reads `<div>` elements for input and `<node>` elements for expectations.

**Attribute parsing:** Port the attribute-to-style mapping from `../taffy/tests/generated_xml.rs`. This includes parsing:
- Dimensions: `"100px"`, `"50%"`, `"auto"`
- Grid templates: `"40px 40px 40px"`, `"1fr 2fr"`, `"repeat(3, 1fr)"`
- Enums: `"flex"`, `"grid"`, `"block"`, `"row"`, `"column"`, etc.
- Viewport: `"max-content"`, `"min-content"`, or px values

**Steps:**

- [ ] **Step 1: Read `../taffy/tests/generated_xml.rs` and `../taffy/tests/common/src/lib.rs` for parsing logic**

- [ ] **Step 2: Create `fixtures_helpers_test.go` with all parsing and assertion functions**

- [ ] **Step 3: Verify it compiles**

```bash
go test -c ./...
```

- [ ] **Step 4: Commit**

```bash
git add fixtures_helpers_test.go
git commit -m "feat: add XML fixture test infrastructure"
```

---

## Task 39: Port fixtures_leaf_test.go

**Files:**
- Create: `fixtures_leaf_test.go`

Start with leaf fixtures — these are the simplest and validate basic sizing.

```go
package taffy_test

import (
    "os"
    "path/filepath"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("Leaf Fixtures", func() {
    fixtureDir := "test_fixtures/leaf"
    entries, err := os.ReadDir(fixtureDir)
    Expect(err).NotTo(HaveOccurred())

    for _, entry := range entries {
        if filepath.Ext(entry.Name()) != ".xml" {
            continue
        }
        entry := entry
        It(entry.Name(), func() {
            fixture := loadFixture(filepath.Join(fixtureDir, entry.Name()))
            tree := New()
            rootId := buildTree(tree, fixture.InputRoot)
            tree.ComputeLayout(rootId, fixture.Viewport)
            assertLayoutMatches(tree, rootId, fixture.Expected)
        })
    }
})
```

**Steps:**

- [ ] **Step 1: Create `fixtures_leaf_test.go`**

- [ ] **Step 2: Run leaf fixture tests**

```bash
go test -v -run "Leaf Fixtures" ./...
```

- [ ] **Step 3: Debug and fix any failures**

- [ ] **Step 4: Commit**

```bash
git add fixtures_leaf_test.go
git commit -m "test: add leaf fixture tests"
```

---

## Task 40: Port fixtures_flex_test.go

**Files:**
- Create: `fixtures_flex_test.go`

Same pattern as leaf but for flex fixtures (565 test cases).

**Steps:**

- [ ] **Step 1: Create `fixtures_flex_test.go`** — same pattern, `fixtureDir = "test_fixtures/flex"`

- [ ] **Step 2: Run flex fixture tests**

```bash
go test -v -run "Flex Fixtures" ./... 2>&1 | head -100
```

- [ ] **Step 3: Debug and fix failures iteratively**

This is likely where the most bugs will surface. Fix issues in the compute and style code.

- [ ] **Step 4: Commit**

```bash
git add fixtures_flex_test.go
git commit -m "test: add flex fixture tests"
```

---

## Task 41: Port fixtures_block_test.go

**Files:**
- Create: `fixtures_block_test.go`

**Steps:**

- [ ] **Step 1: Create `fixtures_block_test.go`** — `fixtureDir = "test_fixtures/block"`

- [ ] **Step 2: Run block fixture tests and fix failures**

- [ ] **Step 3: Commit**

```bash
git add fixtures_block_test.go
git commit -m "test: add block fixture tests"
```

---

## Task 42: Port fixtures_grid_test.go

**Files:**
- Create: `fixtures_grid_test.go`

**Steps:**

- [ ] **Step 1: Create `fixtures_grid_test.go`** — `fixtureDir = "test_fixtures/grid"`

- [ ] **Step 2: Run grid fixture tests and fix failures**

- [ ] **Step 3: Commit**

```bash
git add fixtures_grid_test.go
git commit -m "test: add grid fixture tests"
```

---

## Task 43: Port fixtures_float_test.go

**Files:**
- Create: `fixtures_float_test.go`

**Steps:**

- [ ] **Step 1: Create `fixtures_float_test.go`** — `fixtureDir = "test_fixtures/float"`

- [ ] **Step 2: Run float fixture tests and fix failures**

- [ ] **Step 3: Commit**

```bash
git add fixtures_float_test.go
git commit -m "test: add float fixture tests"
```

---

## Task 44: Port remaining fixture tests

**Files:**
- Create: `fixtures_blockflex_test.go`
- Create: `fixtures_blockgrid_test.go`
- Create: `fixtures_gridflex_test.go`

**Steps:**

- [ ] **Step 1: Create all three test files** — same pattern with respective `fixtureDir` paths

- [ ] **Step 2: Run all fixture tests**

```bash
go test -v ./... 2>&1 | tail -20
```

- [ ] **Step 3: Fix any remaining failures**

- [ ] **Step 4: Commit**

```bash
git add fixtures_blockflex_test.go fixtures_blockgrid_test.go fixtures_gridflex_test.go
git commit -m "test: add mixed-mode fixture tests (blockflex, blockgrid, gridflex)"
```

---

## Task 45: Final validation and cleanup

**Steps:**

- [ ] **Step 1: Run full test suite**

```bash
go test -count=1 ./...
```

- [ ] **Step 2: Run go vet**

```bash
go vet ./...
```

- [ ] **Step 3: Check for any unused code**

```bash
# Review for any stub functions or dead code from the port
```

- [ ] **Step 4: Verify all fixture test directories are covered**

Count passing tests and compare with Rust fixture count (1,086 fixture files).

- [ ] **Step 5: Final commit**

```bash
git add -A
git commit -m "chore: final cleanup after complete Taffy Go port"
```
