// Package taffy provides alignment types for CSS layout.
// This file is a port of alignment.rs from the Taffy Rust layout engine.
package taffy

// AlignItems controls how child nodes are aligned.
// For Flexbox it controls alignment in the cross axis.
// For Grid it controls alignment in the block axis.
// See: https://developer.mozilla.org/en-US/docs/Web/CSS/align-items
type AlignItems int

const (
	// AlignItemsStart packs items toward the start of the axis.
	AlignItemsStart AlignItems = iota
	// AlignItemsEnd packs items toward the end of the axis.
	AlignItemsEnd
	// AlignItemsFlexStart packs items towards the flex-relative start of the axis.
	// For flex containers with flex_direction RowReverse or ColumnReverse this is equivalent
	// to End. In all other cases it is equivalent to Start.
	AlignItemsFlexStart
	// AlignItemsFlexEnd packs items towards the flex-relative end of the axis.
	// For flex containers with flex_direction RowReverse or ColumnReverse this is equivalent
	// to Start. In all other cases it is equivalent to End.
	AlignItemsFlexEnd
	// AlignItemsCenter packs items along the center of the cross axis.
	AlignItemsCenter
	// AlignItemsBaseline aligns items such as their baselines align.
	AlignItemsBaseline
	// AlignItemsStretch stretches items to fill the container.
	AlignItemsStretch
)

// JustifyItems controls how child nodes are aligned in the inline axis for Grid.
// Does not apply to Flexbox, and will be ignored if specified on a flex container.
// See: https://developer.mozilla.org/en-US/docs/Web/CSS/justify-items
type JustifyItems = AlignItems

// AlignSelf controls alignment of an individual node.
// Overrides the parent Node's AlignItems property.
// For Flexbox it controls alignment in the cross axis.
// For Grid it controls alignment in the block axis.
// See: https://developer.mozilla.org/en-US/docs/Web/CSS/align-self
type AlignSelf = AlignItems

// JustifySelf controls alignment of an individual node.
// Overrides the parent Node's JustifyItems property.
// Does not apply to Flexbox, and will be ignored if specified on a flex child.
// For Grid it controls alignment in the inline axis.
// See: https://developer.mozilla.org/en-US/docs/Web/CSS/justify-self
type JustifySelf = AlignItems

// AlignContent sets the distribution of space between and around content items.
// For Flexbox it controls alignment in the cross axis.
// For Grid it controls alignment in the block axis.
// See: https://developer.mozilla.org/en-US/docs/Web/CSS/align-content
type AlignContent int

const (
	// AlignContentStart packs items toward the start of the axis.
	AlignContentStart AlignContent = iota
	// AlignContentEnd packs items toward the end of the axis.
	AlignContentEnd
	// AlignContentFlexStart packs items towards the flex-relative start of the axis.
	// For flex containers with flex_direction RowReverse or ColumnReverse this is equivalent
	// to End. In all other cases it is equivalent to Start.
	AlignContentFlexStart
	// AlignContentFlexEnd packs items towards the flex-relative end of the axis.
	// For flex containers with flex_direction RowReverse or ColumnReverse this is equivalent
	// to Start. In all other cases it is equivalent to End.
	AlignContentFlexEnd
	// AlignContentCenter centers items around the middle of the axis.
	AlignContentCenter
	// AlignContentStretch stretches items to fill the container.
	AlignContentStretch
	// AlignContentSpaceBetween aligns first and last items flush with the edges of the container (no gap).
	// The gap between items is distributed evenly.
	AlignContentSpaceBetween
	// AlignContentSpaceEvenly distributes gaps evenly, including before first and after last item.
	AlignContentSpaceEvenly
	// AlignContentSpaceAround makes the gap between the first and last items exactly half the gap between items.
	// The gaps are distributed evenly in proportion to these ratios.
	AlignContentSpaceAround
)

// Reversed returns the reversed alignment for RTL (right-to-left) contexts.
func (ac AlignContent) Reversed() AlignContent {
	switch ac {
	case AlignContentStart:
		return AlignContentEnd
	case AlignContentEnd:
		return AlignContentStart
	case AlignContentFlexStart:
		return AlignContentFlexEnd
	case AlignContentFlexEnd:
		return AlignContentFlexStart
	case AlignContentStretch:
		return AlignContentEnd
	default:
		return ac
	}
}

// JustifyContent sets the distribution of space between and around content items.
// For Flexbox it controls alignment in the main axis.
// For Grid it controls alignment in the inline axis.
// See: https://developer.mozilla.org/en-US/docs/Web/CSS/justify-content
type JustifyContent = AlignContent
