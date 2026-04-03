// Package taffy provides the main Style struct and core style enums.
// This file is a port of style/mod.rs from the Taffy Rust layout engine.
package taffy

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
func (o Overflow) MaybeIntoAutomaticMinSize() Option[float32] {
	if o.IsScrollContainer() {
		return Some[float32](0.0)
	}
	return None[float32]()
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
// coreStyle interface
// ---------------------------------------------------------------------------

// coreStyle is the core set of styles shared between all CSS layout nodes.
type coreStyle interface {
	boxGenerationMode() BoxGenerationMode
	isBlock() bool
	isCompressibleReplaced() bool
	boxSizing() BoxSizing
	direction() Direction
	overflow() Point[Overflow]
	scrollbarWidth() float32
	position() Position
	inset() Rect[LengthPercentageAuto]
	size() Size[Dimension]
	minSize() Size[Dimension]
	maxSize() Size[Dimension]
	aspectRatio() Option[float32]
	margin() Rect[LengthPercentageAuto]
	padding() Rect[LengthPercentage]
	border() Rect[LengthPercentage]
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
	Overflow       Point[Overflow]
	ScrollbarWidth float32

	// Float properties
	Float Float
	Clear Clear

	// Position properties
	Position Position
	Inset    Rect[LengthPercentageAuto]

	// Size properties
	Size        Size[Dimension]
	MinSize     Size[Dimension]
	MaxSize     Size[Dimension]
	AspectRatio Option[float32]

	// Spacing properties
	Margin  Rect[LengthPercentageAuto]
	Padding Rect[LengthPercentage]
	Border  Rect[LengthPercentage]

	// Alignment properties
	AlignItems     Option[AlignItems]
	AlignSelf      Option[AlignItems]
	JustifyItems   Option[AlignItems]
	JustifySelf    Option[AlignItems]
	AlignContent   Option[AlignContent]
	JustifyContent Option[AlignContent]
	Gap            Size[LengthPercentage]

	// Block container properties
	TextAlign TextAlign

	// Flexbox container properties
	FlexDirection FlexDirection
	FlexWrap      FlexWrap

	// Flexbox item properties
	FlexBasis   Dimension
	FlexGrow    float32
	FlexShrink  float32

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
	GridRow    Line[GridPlacement]
	GridColumn Line[GridPlacement]
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
		Overflow: Point[Overflow]{
			X: OverflowVisible,
			Y: OverflowVisible,
		},
		ScrollbarWidth: 0.0,
		Float:          FloatNone,
		Clear:          ClearNone,
		Position:       PositionRelative,
		Inset: Rect[LengthPercentageAuto]{
			Left:   autoLPA,
			Right:  autoLPA,
			Top:    autoLPA,
			Bottom: autoLPA,
		},
		Size: Size[Dimension]{
			Width:  autoDim,
			Height: autoDim,
		},
		MinSize: Size[Dimension]{
			Width:  autoDim,
			Height: autoDim,
		},
		MaxSize: Size[Dimension]{
			Width:  autoDim,
			Height: autoDim,
		},
		AspectRatio: None[float32](),
		Margin: Rect[LengthPercentageAuto]{
			Left:   LengthPercentageAuto{Tag: LPALength, Value: 0},
			Right:  LengthPercentageAuto{Tag: LPALength, Value: 0},
			Top:    LengthPercentageAuto{Tag: LPALength, Value: 0},
			Bottom: LengthPercentageAuto{Tag: LPALength, Value: 0},
		},
		Padding: Rect[LengthPercentage]{
			Left:   zeroLP,
			Right:  zeroLP,
			Top:    zeroLP,
			Bottom: zeroLP,
		},
		Border: Rect[LengthPercentage]{
			Left:   zeroLP,
			Right:  zeroLP,
			Top:    zeroLP,
			Bottom: zeroLP,
		},
		AlignItems:     None[AlignItems](),
		AlignSelf:      None[AlignItems](),
		JustifyItems:   None[AlignItems](),
		JustifySelf:    None[AlignItems](),
		AlignContent:   None[AlignContent](),
		JustifyContent: None[AlignContent](),
		Gap: Size[LengthPercentage]{
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
		GridRow: Line[GridPlacement]{
			Start: GridPlacement{Tag: GridPlacementAuto},
			End:   GridPlacement{Tag: GridPlacementAuto},
		},
		GridColumn: Line[GridPlacement]{
			Start: GridPlacement{Tag: GridPlacementAuto},
			End:   GridPlacement{Tag: GridPlacementAuto},
		},
	}
}

// ---------------------------------------------------------------------------
// coreStyle implementation on *Style
// ---------------------------------------------------------------------------

func (s *Style) boxGenerationMode() BoxGenerationMode {
	if s.Display == DisplayNone {
		return BoxGenerationModeNone
	}
	return BoxGenerationModeNormal
}

func (s *Style) isBlock() bool {
	return s.Display == DisplayBlock
}

func (s *Style) isCompressibleReplaced() bool {
	return s.ItemIsReplaced
}

func (s *Style) boxSizing() BoxSizing {
	return s.BoxSizing
}

func (s *Style) direction() Direction {
	return s.Direction
}

func (s *Style) overflow() Point[Overflow] {
	return s.Overflow
}

func (s *Style) scrollbarWidth() float32 {
	return s.ScrollbarWidth
}

func (s *Style) position() Position {
	return s.Position
}

func (s *Style) inset() Rect[LengthPercentageAuto] {
	return s.Inset
}

func (s *Style) size() Size[Dimension] {
	return s.Size
}

func (s *Style) minSize() Size[Dimension] {
	return s.MinSize
}

func (s *Style) maxSize() Size[Dimension] {
	return s.MaxSize
}

func (s *Style) aspectRatio() Option[float32] {
	return s.AspectRatio
}

func (s *Style) margin() Rect[LengthPercentageAuto] {
	return s.Margin
}

func (s *Style) padding() Rect[LengthPercentage] {
	return s.Padding
}

func (s *Style) border() Rect[LengthPercentage] {
	return s.Border
}

// ---------------------------------------------------------------------------
// blockContainerStyle implementation on *Style
// ---------------------------------------------------------------------------

func (s *Style) textAlign() TextAlign {
	return s.TextAlign
}

// ---------------------------------------------------------------------------
// blockItemStyle implementation on *Style
// ---------------------------------------------------------------------------

func (s *Style) isTable() bool {
	return s.ItemIsTable
}

func (s *Style) float_() Float {
	return s.Float
}

func (s *Style) clear_() Clear {
	return s.Clear
}

// ---------------------------------------------------------------------------
// flexboxContainerStyle implementation on *Style
// ---------------------------------------------------------------------------

func (s *Style) flexDirection() FlexDirection {
	return s.FlexDirection
}

func (s *Style) flexWrap() FlexWrap {
	return s.FlexWrap
}

func (s *Style) gap() Size[LengthPercentage] {
	return s.Gap
}

func (s *Style) alignContent() Option[AlignContent] {
	return s.AlignContent
}

func (s *Style) alignItems() Option[AlignItems] {
	return s.AlignItems
}

func (s *Style) justifyContent() Option[AlignContent] {
	return s.JustifyContent
}

// ---------------------------------------------------------------------------
// flexboxItemStyle implementation on *Style
// ---------------------------------------------------------------------------

func (s *Style) flexBasis() Dimension {
	return s.FlexBasis
}

func (s *Style) flexGrow() float32 {
	return s.FlexGrow
}

func (s *Style) flexShrink() float32 {
	return s.FlexShrink
}

func (s *Style) alignSelf() Option[AlignItems] {
	return s.AlignSelf
}

// ---------------------------------------------------------------------------
// gridContainerStyle implementation on *Style
// ---------------------------------------------------------------------------

func (s *Style) gridTemplateRows() []GridTemplateComponent {
	return s.GridTemplateRows
}

func (s *Style) gridTemplateColumns() []GridTemplateComponent {
	return s.GridTemplateColumns
}

func (s *Style) gridAutoRows() []TrackSizingFunction {
	return s.GridAutoRows
}

func (s *Style) gridAutoColumns() []TrackSizingFunction {
	return s.GridAutoColumns
}

func (s *Style) gridTemplateAreas() []GridTemplateArea {
	return s.GridTemplateAreas
}

func (s *Style) gridAutoFlow() GridAutoFlow {
	return s.GridAutoFlow
}

func (s *Style) justifyItems() Option[AlignItems] {
	return s.JustifyItems
}

// ---------------------------------------------------------------------------
// gridItemStyle implementation on *Style
// ---------------------------------------------------------------------------

func (s *Style) gridRow() Line[GridPlacement] {
	return s.GridRow
}

func (s *Style) gridColumn() Line[GridPlacement] {
	return s.GridColumn
}

func (s *Style) justifySelf() Option[AlignSelf] {
	return s.JustifySelf
}

// ---------------------------------------------------------------------------
// Compile-time interface satisfaction checks
// ---------------------------------------------------------------------------

var _ coreStyle = (*Style)(nil)
var _ blockContainerStyle = (*Style)(nil)
var _ blockItemStyle = (*Style)(nil)
var _ flexboxContainerStyle = (*Style)(nil)
var _ flexboxItemStyle = (*Style)(nil)
var _ gridContainerStyle = (*Style)(nil)
var _ gridItemStyle = (*Style)(nil)
