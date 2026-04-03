# taffy-go

A Go port of the [Taffy](https://github.com/DioxusLabs/taffy) layout engine. Taffy-go computes CSS-style layouts for UI elements, supporting Flexbox, CSS Grid, Block, and Float layout algorithms.

## Features

- **Flexbox layout** - full CSS Flexbox specification support
- **CSS Grid layout** - full CSS Grid specification support including named lines, template areas, and auto-placement
- **Block layout** - CSS block layout with margin collapsing and float integration
- **Float layout** - CSS float positioning and clearing
- **High fidelity** - passes 4,441 layout conformance tests ported from Taffy's test suite
- **Simple API** - single `TaffyTree` type manages nodes, styles, and layout computation
- **No dependencies** - pure Go, no CGo

## Installation

```bash
go get taffy-go
```

Requires Go 1.25 or later.

## Usage

### Flexbox Layout

```go
package main

import (
    "fmt"
    taffy "taffy-go"
)

func main() {
    tree := taffy.New()

    // Create leaf nodes with styles
    header := tree.NewLeaf(taffy.Style{
        Display: taffy.DisplayFlex,
        Size: taffy.Size[taffy.Dimension]{
            Width:  taffy.LengthDimension(800),
            Height: taffy.LengthDimension(100),
        },
    })

    body := tree.NewLeaf(taffy.Style{
        Display: taffy.DisplayFlex,
        Size: taffy.Size[taffy.Dimension]{
            Width:  taffy.LengthDimension(800),
            Height: taffy.AutoDimension(),
        },
        FlexGrow: 1.0,
    })

    // Create root node with children
    root := tree.NewWithChildren(taffy.Style{
        Display:       taffy.DisplayFlex,
        FlexDirection: taffy.FlexDirectionColumn,
        Size: taffy.Size[taffy.Dimension]{
            Width:  taffy.LengthDimension(800),
            Height: taffy.LengthDimension(600),
        },
    }, []taffy.NodeId{header, body})

    // Compute layout
    tree.ComputeLayout(root, taffy.Size[taffy.AvailableSpace]{
        Width:  taffy.AvailableSpaceFromMaxContent(),
        Height: taffy.AvailableSpaceFromMaxContent(),
    })

    // Read computed layout
    rootLayout := tree.GetLayout(root)
    bodyLayout := tree.GetLayout(body)
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
tree := taffy.New()

// Create 6 child nodes
children := make([]taffy.NodeId, 6)
for i := range children {
    children[i] = tree.NewLeaf(taffy.Style{})
}

// Create a 3x2 grid
root := tree.NewWithChildren(taffy.Style{
    Display: taffy.DisplayGrid,
    Size: taffy.Size[taffy.Dimension]{
        Width:  taffy.LengthDimension(300),
        Height: taffy.LengthDimension(200),
    },
    GridTemplateColumns: []taffy.GridTemplateComponent{
        {Tag: taffy.GridTemplateSingle, Single: taffy.TrackSizingFunction{Min: taffy.MinTrackFromLength(100), Max: taffy.MaxTrackFromLength(100)}},
        {Tag: taffy.GridTemplateSingle, Single: taffy.TrackSizingFunction{Min: taffy.MinTrackFromLength(100), Max: taffy.MaxTrackFromLength(100)}},
        {Tag: taffy.GridTemplateSingle, Single: taffy.TrackSizingFunction{Min: taffy.MinTrackFromLength(100), Max: taffy.MaxTrackFromLength(100)}},
    },
    GridTemplateRows: []taffy.GridTemplateComponent{
        {Tag: taffy.GridTemplateSingle, Single: taffy.TrackSizingFunction{Min: taffy.MinTrackFromLength(100), Max: taffy.MaxTrackFromLength(100)}},
        {Tag: taffy.GridTemplateSingle, Single: taffy.TrackSizingFunction{Min: taffy.MinTrackFromLength(100), Max: taffy.MaxTrackFromLength(100)}},
    },
}, children)

tree.ComputeLayout(root, taffy.Size[taffy.AvailableSpace]{
    Width:  taffy.AvailableSpaceFromMaxContent(),
    Height: taffy.AvailableSpaceFromMaxContent(),
})
```

### Block Layout

```go
tree := taffy.New()

child0 := tree.NewLeaf(taffy.Style{
    Display: taffy.DisplayBlock,
    Size: taffy.Size[taffy.Dimension]{
        Height: taffy.LengthDimension(50),
    },
})
child1 := tree.NewLeaf(taffy.Style{
    Display: taffy.DisplayBlock,
    Size: taffy.Size[taffy.Dimension]{
        Height: taffy.LengthDimension(50),
    },
})

root := tree.NewWithChildren(taffy.Style{
    Display: taffy.DisplayBlock,
    Size: taffy.Size[taffy.Dimension]{
        Width: taffy.LengthDimension(200),
    },
}, []taffy.NodeId{child0, child1})

tree.ComputeLayout(root, taffy.Size[taffy.AvailableSpace]{
    Width:  taffy.AvailableSpaceFromMaxContent(),
    Height: taffy.AvailableSpaceFromMaxContent(),
})

// Root height is automatically computed as 100 (50 + 50)
```

### Custom Measure Functions

For leaf nodes with content (e.g., text), provide a measure function:

```go
tree := taffy.New()

measureFn := taffy.NodeContext(func(
    knownDimensions taffy.Size[taffy.Option[float32]],
    parentSize taffy.Size[taffy.Option[float32]],
    availableSpace taffy.Size[taffy.AvailableSpace],
    sizingMode taffy.SizingMode,
) taffy.Size[float32] {
    // Return the intrinsic size of your content
    return taffy.Size[float32]{Width: 100, Height: 50}
})

leaf := tree.NewLeafWithContext(taffy.Style{}, measureFn)
root := tree.NewWithChildren(taffy.DefaultStyle(), []taffy.NodeId{leaf})

tree.ComputeLayout(root, taffy.Size[taffy.AvailableSpace]{
    Width:  taffy.AvailableSpaceFromLength(400),
    Height: taffy.AvailableSpaceFromLength(300),
})
```

## API Overview

### Core Types

| Type | Description |
|------|-------------|
| `TaffyTree` | Main layout tree. Create nodes, set styles, compute and read layouts. |
| `NodeId` | Identifies a node within a `TaffyTree`. |
| `Style` | CSS-like styling properties for a node. |
| `Layout` | Computed layout result: position, size, padding, border, margin. |
| `NodeContext` | Measure function for leaf nodes with content. |

### Geometry Types

| Type | Description |
|------|-------------|
| `Size[T]` | Width and height pair. |
| `Point[T]` | X and Y coordinate pair. |
| `Rect[T]` | Left, right, top, bottom edges. |
| `Line[T]` | Start and end pair. |

### Style Value Types

| Type | Description |
|------|-------------|
| `Dimension` | Size value: length, percent, or auto. |
| `LengthPercentage` | Length or percentage (no auto). |
| `LengthPercentageAuto` | Length, percentage, or auto. |
| `AvailableSpace` | Available space: definite, min-content, or max-content. |

### Display Modes

| Value | Description |
|-------|-------------|
| `DisplayFlex` | Flexbox layout (default). |
| `DisplayGrid` | CSS Grid layout. |
| `DisplayBlock` | Block layout. |
| `DisplayNone` | Node is hidden. |

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
