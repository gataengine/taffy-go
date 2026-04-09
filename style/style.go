// Package style provides the main Style struct and core style enums.
// This file is a port of style/mod.rs from the Taffy Rust layout engine.
package style

import (
	"taffy-go/geometry"
	"taffy-go/lib"
)

// ---------------------------------------------------------------------------
// Display
// ---------------------------------------------------------------------------

// Display sets the layout used for the children of this node.
// The default is DisplayFlex (matching Taffy's default with flexbox enabled).
type Display int

const (
	// DisplayBlock means the children will follow the block layout algorithm.
	DisplayBlock Display = iota
	// DisplayFlex means the children will follow the flexbox layout algorithm.
	DisplayFlex
	// DisplayGrid means the children will follow the CSS Grid layout algorithm.
	DisplayGrid
	// DisplayNone means the node is hidden and its children will also be hidden.
	DisplayNone
)

// ---------------------------------------------------------------------------
// BoxGenerationMode
// ---------------------------------------------------------------------------

// BoxGenerationMode is an abstracted version of the CSS display property where
// any value other than "none" is represented by "normal".
// See: https://www.w3.org/TR/css-display-3/#box-generation
type BoxGenerationMode int

const (
	// BoxGenerationModeNormal means the node generates a box in the regular way.
	BoxGenerationModeNormal BoxGenerationMode = iota
	// BoxGenerationModeNone means the node and its descendants generate no boxes.
	BoxGenerationModeNone
)

// ---------------------------------------------------------------------------
// Position
// ---------------------------------------------------------------------------

// Position is the positioning strategy for this item.
// This controls both how the origin is determined and whether the item is
// controlled by flexbox layout.
type Position int

const (
	// PositionRelative means the offset is computed relative to the final
	// position given by the layout algorithm.
	PositionRelative Position = iota
	// PositionAbsolute means the offset is computed relative to this item's
	// closest positioned ancestor, or the origin if none exists.
	PositionAbsolute
)

// ---------------------------------------------------------------------------
// BoxSizing
// ---------------------------------------------------------------------------

// BoxSizing specifies whether size styles apply to the content box or border box.
// See: https://developer.mozilla.org/en-US/docs/Web/CSS/box-sizing
type BoxSizing int

const (
	// BoxSizingBorderBox means size styles specify the border box
	// (the size excluding margin but including padding/border).
	BoxSizingBorderBox BoxSizing = iota
	// BoxSizingContentBox means size styles specify the content box
	// (the size excluding padding/border/margin).
	BoxSizingContentBox
)

// ---------------------------------------------------------------------------
// Overflow
// ---------------------------------------------------------------------------

// Overflow controls how children overflowing their container affect layout.
// See: https://developer.mozilla.org/en-US/docs/Web/CSS/overflow
type Overflow int

const (
	// OverflowVisible means the automatic minimum size is based on content size.
	// Content that overflows should contribute to the scroll region of the parent.
	OverflowVisible Overflow = iota
	// OverflowClip means the automatic minimum size is based on content size.
	// Content that overflows should NOT contribute to the scroll region of the parent.
	OverflowClip
	// OverflowHidden means the automatic minimum size is 0.
	// Content that overflows should NOT contribute to the scroll region of the parent.
	OverflowHidden
	// OverflowScroll means the automatic minimum size is 0. Space is reserved for
	// a scrollbar. Content that overflows should NOT contribute to the scroll region of the parent.
	OverflowScroll
)

// IsScrollContainer returns true for overflow modes that contain their contents
// (Hidden, Scroll), or false for modes that allow contents to spill (Visible, Clip).
func (o Overflow) IsScrollContainer() bool {
	return o == OverflowHidden || o == OverflowScroll
}

// MaybeIntoAutomaticMinSize returns Some(0.0) if the overflow mode would cause
// the automatic minimum size of a Flexbox or CSS Grid item to be 0.
// Otherwise returns None.
func (o Overflow) MaybeIntoAutomaticMinSize() lib.Option[float32] {
	if o.IsScrollContainer() {
		return lib.Some[float32](0.0)
	}
	return lib.None[float32]()
}

// ---------------------------------------------------------------------------
// Direction
// ---------------------------------------------------------------------------

// Direction sets the direction of text, table and grid columns, and horizontal overflow.
// See: https://developer.mozilla.org/en-US/docs/Web/CSS/direction
type Direction int

const (
	// DirectionLTR is left-to-right.
	DirectionLTR Direction = iota
	// DirectionRTL is right-to-left.
	DirectionRTL
)

// IsRTL returns true if the direction is right-to-left.
func (d Direction) IsRTL() bool {
	return d == DirectionRTL
}

// ---------------------------------------------------------------------------
// CoreStyle interface
// ---------------------------------------------------------------------------

// CoreStyle is the core set of styles shared between all CSS layout nodes.
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
	GetAspectRatio() lib.Option[float32]
	GetMargin() geometry.Rect[LengthPercentageAuto]
	GetPadding() geometry.Rect[LengthPercentage]
	GetBorder() geometry.Rect[LengthPercentage]
}

// ---------------------------------------------------------------------------
// Style struct
// ---------------------------------------------------------------------------

// Style is a typed representation of the CSS style information for a single node.
// It implements all the style interfaces used by the layout algorithms.
type Style struct {
	Display        Display
	ItemIsTable    bool
	ItemIsReplaced bool
	BoxSizing      BoxSizing
	Direction      Direction

	// Overflow properties
	Overflow       geometry.Point[Overflow]
	ScrollbarWidth float32

	// Float properties
	Float Float
	Clear Clear

	// Position properties
	Position Position
	Inset    geometry.Rect[LengthPercentageAuto]

	// Size properties
	Size        geometry.Size[Dimension]
	MinSize     geometry.Size[Dimension]
	MaxSize     geometry.Size[Dimension]
	AspectRatio lib.Option[float32]

	// Spacing properties
	Margin  geometry.Rect[LengthPercentageAuto]
	Padding geometry.Rect[LengthPercentage]
	Border  geometry.Rect[LengthPercentage]

	// Alignment properties
	AlignItems     lib.Option[AlignItems]
	AlignSelf      lib.Option[AlignItems]
	JustifyItems   lib.Option[AlignItems]
	JustifySelf    lib.Option[AlignItems]
	AlignContent   lib.Option[AlignContent]
	JustifyContent lib.Option[AlignContent]
	Gap            geometry.Size[LengthPercentage]

	// Block container properties
	TextAlign TextAlign

	// Flexbox container properties
	FlexDirection FlexDirection
	FlexWrap      FlexWrap

	// Flexbox item properties
	FlexBasis  Dimension
	FlexGrow   float32
	FlexShrink float32

	// Grid container properties
	GridTemplateRows    []GridTemplateComponent
	GridTemplateColumns []GridTemplateComponent
	GridAutoRows        []TrackSizingFunction
	GridAutoColumns     []TrackSizingFunction
	GridAutoFlow        GridAutoFlow

	// Grid container named properties
	GridTemplateAreas       []GridTemplateArea
	GridTemplateColumnNames [][]string
	GridTemplateRowNames    [][]string

	// Grid item properties
	GridRow    geometry.Line[GridPlacement]
	GridColumn geometry.Line[GridPlacement]
}

// DefaultStyle returns Taffy's default style values.
// This matches Style::DEFAULT in the Rust implementation.
func DefaultStyle() Style {
	autoLPA := LengthPercentageAuto{Tag: LPAAuto}
	zeroLP := LengthPercentage{Tag: LPLength, Value: 0}
	autoDim := Dimension{Tag: DimensionAuto}

	return Style{
		Display:        DisplayFlex,
		ItemIsTable:    false,
		ItemIsReplaced: false,
		BoxSizing:      BoxSizingBorderBox,
		Direction:      DirectionLTR,
		Overflow: geometry.Point[Overflow]{
			X: OverflowVisible,
			Y: OverflowVisible,
		},
		ScrollbarWidth: 0.0,
		Float:          FloatNone,
		Clear:          ClearNone,
		Position:       PositionRelative,
		Inset: geometry.Rect[LengthPercentageAuto]{
			Left:   autoLPA,
			Right:  autoLPA,
			Top:    autoLPA,
			Bottom: autoLPA,
		},
		Size: geometry.Size[Dimension]{
			Width:  autoDim,
			Height: autoDim,
		},
		MinSize: geometry.Size[Dimension]{
			Width:  autoDim,
			Height: autoDim,
		},
		MaxSize: geometry.Size[Dimension]{
			Width:  autoDim,
			Height: autoDim,
		},
		AspectRatio: lib.None[float32](),
		Margin: geometry.Rect[LengthPercentageAuto]{
			Left:   LengthPercentageAuto{Tag: LPALength, Value: 0},
			Right:  LengthPercentageAuto{Tag: LPALength, Value: 0},
			Top:    LengthPercentageAuto{Tag: LPALength, Value: 0},
			Bottom: LengthPercentageAuto{Tag: LPALength, Value: 0},
		},
		Padding: geometry.Rect[LengthPercentage]{
			Left:   zeroLP,
			Right:  zeroLP,
			Top:    zeroLP,
			Bottom: zeroLP,
		},
		Border: geometry.Rect[LengthPercentage]{
			Left:   zeroLP,
			Right:  zeroLP,
			Top:    zeroLP,
			Bottom: zeroLP,
		},
		AlignItems:     lib.None[AlignItems](),
		AlignSelf:      lib.None[AlignItems](),
		JustifyItems:   lib.None[AlignItems](),
		JustifySelf:    lib.None[AlignItems](),
		AlignContent:   lib.None[AlignContent](),
		JustifyContent: lib.None[AlignContent](),
		Gap: geometry.Size[LengthPercentage]{
			Width:  zeroLP,
			Height: zeroLP,
		},
		TextAlign:     TextAlignAuto,
		FlexDirection: FlexDirectionRow,
		FlexWrap:      FlexWrapNoWrap,
		FlexBasis:     autoDim,
		FlexGrow:      0.0,
		FlexShrink:    1.0,
		// Grid fields are nil slices and zero values by default
		GridAutoFlow: GridAutoFlowRow,
		GridRow: geometry.Line[GridPlacement]{
			Start: GridPlacement{Tag: GridPlacementAuto},
			End:   GridPlacement{Tag: GridPlacementAuto},
		},
		GridColumn: geometry.Line[GridPlacement]{
			Start: GridPlacement{Tag: GridPlacementAuto},
			End:   GridPlacement{Tag: GridPlacementAuto},
		},
	}
}

// ---------------------------------------------------------------------------
// CoreStyle implementation on *Style
// ---------------------------------------------------------------------------

func (s *Style) GetBoxGenerationMode() BoxGenerationMode {
	if s.Display == DisplayNone {
		return BoxGenerationModeNone
	}
	return BoxGenerationModeNormal
}

func (s *Style) GetIsBlock() bool {
	return s.Display == DisplayBlock
}

func (s *Style) GetIsCompressibleReplaced() bool {
	return s.ItemIsReplaced
}

func (s *Style) GetBoxSizing() BoxSizing {
	return s.BoxSizing
}

func (s *Style) GetDirection() Direction {
	return s.Direction
}

func (s *Style) GetOverflow() geometry.Point[Overflow] {
	return s.Overflow
}

func (s *Style) GetScrollbarWidth() float32 {
	return s.ScrollbarWidth
}

func (s *Style) GetPosition() Position {
	return s.Position
}

func (s *Style) GetInset() geometry.Rect[LengthPercentageAuto] {
	return s.Inset
}

func (s *Style) GetSize() geometry.Size[Dimension] {
	return s.Size
}

func (s *Style) GetMinSize() geometry.Size[Dimension] {
	return s.MinSize
}

func (s *Style) GetMaxSize() geometry.Size[Dimension] {
	return s.MaxSize
}

func (s *Style) GetAspectRatio() lib.Option[float32] {
	return s.AspectRatio
}

func (s *Style) GetMargin() geometry.Rect[LengthPercentageAuto] {
	return s.Margin
}

func (s *Style) GetPadding() geometry.Rect[LengthPercentage] {
	return s.Padding
}

func (s *Style) GetBorder() geometry.Rect[LengthPercentage] {
	return s.Border
}

// ---------------------------------------------------------------------------
// BlockContainerStyle implementation on *Style
// ---------------------------------------------------------------------------

func (s *Style) GetTextAlign() TextAlign {
	return s.TextAlign
}

// ---------------------------------------------------------------------------
// BlockItemStyle implementation on *Style
// ---------------------------------------------------------------------------

func (s *Style) GetIsTable() bool {
	return s.ItemIsTable
}

func (s *Style) GetFloat() Float {
	return s.Float
}

func (s *Style) GetClear() Clear {
	return s.Clear
}

// ---------------------------------------------------------------------------
// FlexboxContainerStyle implementation on *Style
// ---------------------------------------------------------------------------

func (s *Style) GetFlexDirection() FlexDirection {
	return s.FlexDirection
}

func (s *Style) GetFlexWrap() FlexWrap {
	return s.FlexWrap
}

func (s *Style) GetGap() geometry.Size[LengthPercentage] {
	return s.Gap
}

func (s *Style) GetAlignContent() lib.Option[AlignContent] {
	return s.AlignContent
}

func (s *Style) GetAlignItems() lib.Option[AlignItems] {
	return s.AlignItems
}

func (s *Style) GetJustifyContent() lib.Option[AlignContent] {
	return s.JustifyContent
}

// ---------------------------------------------------------------------------
// FlexboxItemStyle implementation on *Style
// ---------------------------------------------------------------------------

func (s *Style) GetFlexBasis() Dimension {
	return s.FlexBasis
}

func (s *Style) GetFlexGrow() float32 {
	return s.FlexGrow
}

func (s *Style) GetFlexShrink() float32 {
	return s.FlexShrink
}

func (s *Style) GetAlignSelf() lib.Option[AlignItems] {
	return s.AlignSelf
}

// ---------------------------------------------------------------------------
// GridContainerStyle implementation on *Style
// ---------------------------------------------------------------------------

func (s *Style) GetGridTemplateRows() []GridTemplateComponent {
	return s.GridTemplateRows
}

func (s *Style) GetGridTemplateColumns() []GridTemplateComponent {
	return s.GridTemplateColumns
}

func (s *Style) GetGridAutoRows() []TrackSizingFunction {
	return s.GridAutoRows
}

func (s *Style) GetGridAutoColumns() []TrackSizingFunction {
	return s.GridAutoColumns
}

func (s *Style) GetGridTemplateAreas() []GridTemplateArea {
	return s.GridTemplateAreas
}

func (s *Style) GetGridAutoFlow() GridAutoFlow {
	return s.GridAutoFlow
}

func (s *Style) GetJustifyItems() lib.Option[AlignItems] {
	return s.JustifyItems
}

func (s *Style) GetGridTemplateColumnNames() [][]string {
	return s.GridTemplateColumnNames
}

func (s *Style) GetGridTemplateRowNames() [][]string {
	return s.GridTemplateRowNames
}

// ---------------------------------------------------------------------------
// GridItemStyle implementation on *Style
// ---------------------------------------------------------------------------

func (s *Style) GetGridRow() geometry.Line[GridPlacement] {
	return s.GridRow
}

func (s *Style) GetGridColumn() geometry.Line[GridPlacement] {
	return s.GridColumn
}

func (s *Style) GetJustifySelf() lib.Option[AlignSelf] {
	return s.JustifySelf
}

// ---------------------------------------------------------------------------
// Compile-time interface satisfaction checks
// ---------------------------------------------------------------------------

var _ CoreStyle = (*Style)(nil)
var _ BlockContainerStyle = (*Style)(nil)
var _ BlockItemStyle = (*Style)(nil)
var _ FlexboxContainerStyle = (*Style)(nil)
var _ FlexboxItemStyle = (*Style)(nil)
var _ GridContainerStyle = (*Style)(nil)
var _ GridItemStyle = (*Style)(nil)
