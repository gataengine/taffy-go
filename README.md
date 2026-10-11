# taffy-go

A Go port of the [Taffy](https://github.com/DioxusLabs/taffy) layout engine. Taffy-go computes CSS-style layouts for UI elements, supporting Flexbox, CSS Grid, Block, and Float layout algorithms.

## Features

- **Flexbox layout** - full CSS Flexbox specification support
- **CSS Grid layout** - full CSS Grid specification support including named lines, template areas, and auto-placement
- **Block layout** - CSS block layout with margin collapsing and float integration
- **Float layout** - CSS float positioning and clearing
- **High fidelity** - passes 4,441 layout conformance tests ported from Taffy's test suite
- **Custom tree backends** - public trait interfaces allow custom tree implementations
- **No dependencies** - pure Go, no CGo

## Installation

```bash
go get github.com/gataengine/taffy-go
```

Requires Go 1.27.1 or later.

## Package Structure

```
taffy-go/
├── geometry/    # Primitives: Option, Size, Rect, Point, Line, MinMax
├── style/       # Style struct, enums, dimension types, style interfaces
├── tree/        # Trait interfaces, Layout types, NodeId, Cache
├── compute/     # Shared compute helpers (cached layout, leaf, alignment)
│   ├── flexbox/ # Flexbox layout algorithm
│   ├── grid/    # CSS Grid layout algorithm
│   └── block/   # Block layout algorithm
└── (root)       # TaffyTree high-level API
```

## Usage

### Flexbox Layout

```go
package main

import (
    "fmt"
    taffy "github.com/gataengine/taffy-go"
    "github.com/gataengine/taffy-go/geometry"
    "github.com/gataengine/taffy-go/style"
    "github.com/gataengine/taffy-go/tree"
)

func main() {
    t := taffy.New()

    // Create leaf nodes with styles
    header := t.NewLeaf(style.Style{
        Display: style.DisplayFlex,
        Size: geometry.Size[style.Dimension]{
            Width:  style.LengthDimension(800),
            Height: style.LengthDimension(100),
        },
    })

    body := t.NewLeaf(style.Style{
        Display: style.DisplayFlex,
        Size: geometry.Size[style.Dimension]{
            Width:  style.LengthDimension(800),
            Height: style.AutoDimension(),
        },
        FlexGrow: 1.0,
    })

    // Create root node with children
    root := t.NewWithChildren(style.Style{
        Display:       style.DisplayFlex,
        FlexDirection: style.FlexDirectionColumn,
        Size: geometry.Size[style.Dimension]{
            Width:  style.LengthDimension(800),
            Height: style.LengthDimension(600),
        },
    }, []tree.NodeId{header, body})

    // Compute layout
    t.ComputeLayout(root, geometry.Size[style.AvailableSpace]{
        Width:  style.AvailableSpaceFromMaxContent(),
        Height: style.AvailableSpaceFromMaxContent(),
    })

    // Read computed layout
    rootLayout := t.GetLayout(root)
    bodyLayout := t.GetLayout(body)
    fmt.Printf("Root: %vx%v\n", rootLayout.Size.Width, rootLayout.Size.Height)
    fmt.Printf("Body: %vx%v (flex-grew to fill remaining space)\n",
        bodyLayout.Size.Width, bodyLayout.Size.Height)
    // Output:
    // Root: 800x600
    // Body: 800x500 (flex-grew to fill remaining space)
}
```

### CSS Grid Layout

```go
import (
    taffy "github.com/gataengine/taffy-go"
    "github.com/gataengine/taffy-go/geometry"
    "github.com/gataengine/taffy-go/style"
    "github.com/gataengine/taffy-go/tree"
)

t := taffy.New()

// Create 6 child nodes
children := make([]tree.NodeId, 6)
for i := range children {
    children[i] = t.NewLeaf(style.Style{})
}

// Create a 3x2 grid
root := t.NewWithChildren(style.Style{
    Display: style.DisplayGrid,
    Size: geometry.Size[style.Dimension]{
        Width:  style.LengthDimension(300),
        Height: style.LengthDimension(200),
    },
    GridTemplateColumns: []style.GridTemplateComponent{
        {Tag: style.GridTemplateSingle, Single: style.TrackSizingFunction{Min: style.MinTrackFromLength(100), Max: style.MaxTrackFromLength(100)}},
        {Tag: style.GridTemplateSingle, Single: style.TrackSizingFunction{Min: style.MinTrackFromLength(100), Max: style.MaxTrackFromLength(100)}},
        {Tag: style.GridTemplateSingle, Single: style.TrackSizingFunction{Min: style.MinTrackFromLength(100), Max: style.MaxTrackFromLength(100)}},
    },
    GridTemplateRows: []style.GridTemplateComponent{
        {Tag: style.GridTemplateSingle, Single: style.TrackSizingFunction{Min: style.MinTrackFromLength(100), Max: style.MaxTrackFromLength(100)}},
        {Tag: style.GridTemplateSingle, Single: style.TrackSizingFunction{Min: style.MinTrackFromLength(100), Max: style.MaxTrackFromLength(100)}},
    },
}, children)

t.ComputeLayout(root, geometry.Size[style.AvailableSpace]{
    Width:  style.AvailableSpaceFromMaxContent(),
    Height: style.AvailableSpaceFromMaxContent(),
})
```

### Block Layout

```go
import (
    taffy "github.com/gataengine/taffy-go"
    "github.com/gataengine/taffy-go/geometry"
    "github.com/gataengine/taffy-go/style"
    "github.com/gataengine/taffy-go/tree"
)

t := taffy.New()

child0 := t.NewLeaf(style.Style{
    Display: style.DisplayBlock,
    Size: geometry.Size[style.Dimension]{
        Height: style.LengthDimension(50),
    },
})
child1 := t.NewLeaf(style.Style{
    Display: style.DisplayBlock,
    Size: geometry.Size[style.Dimension]{
        Height: style.LengthDimension(50),
    },
})

root := t.NewWithChildren(style.Style{
    Display: style.DisplayBlock,
    Size: geometry.Size[style.Dimension]{
        Width: style.LengthDimension(200),
    },
}, []tree.NodeId{child0, child1})

t.ComputeLayout(root, geometry.Size[style.AvailableSpace]{
    Width:  style.AvailableSpaceFromMaxContent(),
    Height: style.AvailableSpaceFromMaxContent(),
})

// Root height is automatically computed as 100 (50 + 50)
```

### Custom Measure Functions

For leaf nodes with content (e.g., text), provide a measure function:

```go
import (
    taffy "github.com/gataengine/taffy-go"
    "github.com/gataengine/taffy-go/geometry"
    "github.com/gataengine/taffy-go/style"
    "github.com/gataengine/taffy-go/tree"
)

t := taffy.New()

measureFn := taffy.NodeContext(func(
    knownDimensions geometry.Size[geometry.Option[float32]],
    parentSize geometry.Size[geometry.Option[float32]],
    availableSpace geometry.Size[style.AvailableSpace],
    sizingMode tree.SizingMode,
) geometry.Size[float32] {
    // Return the intrinsic size of your content
    return geometry.Size[float32]{Width: 100, Height: 50}
})

leaf := t.NewLeafWithContext(style.Style{}, measureFn)
root := t.NewWithChildren(style.DefaultStyle(), []tree.NodeId{leaf})

t.ComputeLayout(root, geometry.Size[style.AvailableSpace]{
    Width:  style.AvailableSpaceFromLength(400),
    Height: style.AvailableSpaceFromLength(300),
})
```

### Custom Tree Backends

For advanced use cases, implement the trait interfaces from the `tree` package to create custom tree backends with your own storage strategy:

```go
import (
    "github.com/gataengine/taffy-go/tree"
    "github.com/gataengine/taffy-go/style"
    "github.com/gataengine/taffy-go/compute/flexbox"
    "github.com/gataengine/taffy-go/compute/grid"
    "github.com/gataengine/taffy-go/compute/block"
)

// Implement the interfaces your layout modes require:
// - tree.LayoutPartialTree (base)
// - tree.LayoutFlexboxContainer (for flexbox)
// - tree.LayoutGridContainer (for grid)
// - tree.LayoutBlockContainer (for block)

// Then call compute functions directly:
// flexbox.ComputeFlexboxLayout(myTree, node, inputs)
// grid.ComputeGridLayout(myTree, node, inputs)
// block.ComputeBlockLayout(myTree, node, inputs)
```

## API Overview

### High-Level API (root package)

| Type | Description |
|------|-------------|
| `TaffyTree` | Main layout tree. Create nodes, set styles, compute and read layouts. |
| `NodeContext` | Measure function for leaf nodes with content. |
| `MeasureFunction` | Extended measure function with node context. |

### Geometry Types (`geometry` package)

| Type | Description |
|------|-------------|
| `Size[T]` | Width and height pair. |
| `Point[T]` | X and Y coordinate pair. |
| `Rect[T]` | Left, right, top, bottom edges. |
| `Line[T]` | Start and end pair. |
| `Option[T]` | Optional value wrapper. |

### Style Types (`style` package)

| Type | Description |
|------|-------------|
| `Style` | CSS-like styling properties for a node. |
| `Dimension` | Size value: length, percent, or auto. |
| `LengthPercentage` | Length or percentage (no auto). |
| `LengthPercentageAuto` | Length, percentage, or auto. |
| `AvailableSpace` | Available space: definite, min-content, or max-content. |

### Style Interfaces (`style` package)

| Interface | Description |
|-----------|-------------|
| `CoreStyle` | Core style properties shared by all layout modes. |
| `FlexboxContainerStyle` | Flexbox container properties. |
| `FlexboxItemStyle` | Flexbox item properties. |
| `GridContainerStyle` | Grid container properties. |
| `GridItemStyle` | Grid item properties. |
| `BlockContainerStyle` | Block container properties. |
| `BlockItemStyle` | Block item properties. |

### Tree Interfaces (`tree` package)

| Interface | Description |
|-----------|-------------|
| `LayoutPartialTree` | Core layout interface for all algorithms. |
| `LayoutFlexboxContainer` | Extends LayoutPartialTree for flexbox. |
| `LayoutGridContainer` | Extends LayoutPartialTree for grid. |
| `LayoutBlockContainer` | Extends LayoutPartialTree for block. |
| `TraversePartialTree` | Downward tree traversal. |
| `CacheTree` | Layout result caching. |
| `RoundTree` | Layout rounding. |
| `PrintTree` | Debug printing. |

### Layout Types (`tree` package)

| Type | Description |
|------|-------------|
| `NodeId` | Identifies a node within a tree. |
| `Layout` | Computed layout result: position, size, padding, border, margin. |
| `LayoutInput` | Input parameters for layout computation. |
| `LayoutOutput` | Result of layout computation. |

### Display Modes

| Value | Description |
|-------|-------------|
| `style.DisplayFlex` | Flexbox layout (default). |
| `style.DisplayGrid` | CSS Grid layout. |
| `style.DisplayBlock` | Block layout. |
| `style.DisplayNone` | Node is hidden. |

## Learning Resources

Taffy-go implements the CSS layout specifications, so CSS documentation applies directly:

**Flexbox:**
- [Flexbox Froggy](https://flexboxfroggy.com/) - interactive tutorial
- [A Complete Guide to Flexbox](https://css-tricks.com/snippets/css/a-guide-to-flexbox/) - CSS-Tricks reference

**CSS Grid:**
- [CSS Grid Garden](https://cssgridgarden.com/) - interactive tutorial
- [A Complete Guide to CSS Grid](https://css-tricks.com/snippets/css/complete-guide-grid/) - CSS-Tricks reference

**General:**
- [MDN CSS Layout](https://developer.mozilla.org/en-US/docs/Learn/CSS/CSS_layout) - comprehensive CSS layout documentation

## Acknowledgements

This project is a Go port of [Taffy](https://github.com/DioxusLabs/taffy) by [DioxusLabs](https://github.com/DioxusLabs). Taffy is a collaborative, cross-platform UI layout library written in Rust that powers projects including Servo, Bevy, Zed, Slint, and others.

## License

MIT
