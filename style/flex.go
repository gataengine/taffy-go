// Package style provides flex layout style types.
// This file is a port of flex.rs from the Taffy Rust layout engine.
package style

import "taffy-go/geometry"

// FlexDirection controls the direction of the flexbox layout main axis.
//
// There are always two perpendicular layout axes: main (or primary) and cross (or secondary).
// Adding items will cause them to be positioned adjacent to each other along the main axis.
//
// The default behavior is FlexDirectionRow.
//
// See: https://www.w3.org/TR/css-flexbox-1/#flex-direction-property
type FlexDirection int

const (
	// FlexDirectionRow defines +x as the main axis.
	// Items will be added from left to right in a row.
	FlexDirectionRow FlexDirection = iota
	// FlexDirectionColumn defines +y as the main axis.
	// Items will be added from top to bottom in a column.
	FlexDirectionColumn
	// FlexDirectionRowReverse defines -x as the main axis.
	// Items will be added from right to left in a row.
	FlexDirectionRowReverse
	// FlexDirectionColumnReverse defines -y as the main axis.
	// Items will be added from bottom to top in a column.
	FlexDirectionColumnReverse
)

// IsRow reports whether the direction is FlexDirectionRow or FlexDirectionRowReverse.
func (fd FlexDirection) IsRow() bool {
	return fd == FlexDirectionRow || fd == FlexDirectionRowReverse
}

// IsColumn reports whether the direction is FlexDirectionColumn or FlexDirectionColumnReverse.
func (fd FlexDirection) IsColumn() bool {
	return fd == FlexDirectionColumn || fd == FlexDirectionColumnReverse
}

// IsReverse reports whether the direction is FlexDirectionRowReverse or FlexDirectionColumnReverse.
func (fd FlexDirection) IsReverse() bool {
	return fd == FlexDirectionRowReverse || fd == FlexDirectionColumnReverse
}

// MainAxis returns the AbsoluteAxis that corresponds to the main axis.
func (fd FlexDirection) MainAxis() geometry.AbsoluteAxis {
	switch fd {
	case FlexDirectionRow, FlexDirectionRowReverse:
		return geometry.AbsoluteAxisHorizontal
	default:
		return geometry.AbsoluteAxisVertical
	}
}

// CrossAxis returns the AbsoluteAxis that corresponds to the cross axis.
func (fd FlexDirection) CrossAxis() geometry.AbsoluteAxis {
	switch fd {
	case FlexDirectionRow, FlexDirectionRowReverse:
		return geometry.AbsoluteAxisVertical
	default:
		return geometry.AbsoluteAxisHorizontal
	}
}

// FlexWrap controls whether flex items are forced onto one line or can wrap onto multiple lines.
//
// Defaults to FlexWrapNoWrap.
//
// See: https://www.w3.org/TR/css-flexbox-1/#flex-wrap-property
type FlexWrap int

const (
	// FlexWrapNoWrap means items will not wrap and stay on a single line.
	FlexWrapNoWrap FlexWrap = iota
	// FlexWrapWrap means items will wrap according to the item's FlexDirection.
	FlexWrapWrap
	// FlexWrapWrapReverse means items will wrap in the opposite direction to the item's FlexDirection.
	FlexWrapWrapReverse
)

// FlexboxContainerStyle is the set of styles required for a Flexbox container.
type FlexboxContainerStyle interface {
	GetFlexDirection() FlexDirection
	GetFlexWrap() FlexWrap
	GetGap() geometry.Size[LengthPercentage]
	GetAlignContent() geometry.Option[AlignContent]
	GetAlignItems() geometry.Option[AlignItems]
	GetJustifyContent() geometry.Option[AlignContent]
}

// FlexboxItemStyle is the set of styles required for a Flexbox item (child of a Flexbox container).
type FlexboxItemStyle interface {
	GetFlexBasis() Dimension
	GetFlexGrow() float32
	GetFlexShrink() float32
	GetAlignSelf() geometry.Option[AlignItems]
}
