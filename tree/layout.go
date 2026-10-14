// Package tree provides final data structures representing high-level UI layout.
// This file is a port of tree/layout.rs from the Taffy Rust layout engine.
package tree

import (
	"github.com/gataengine/taffy-go/geometry"
	"github.com/gataengine/taffy-go/lib"
	"github.com/gataengine/taffy-go/style"
)

// RunMode indicates whether we are performing a full layout, or merely need to size a node.
type RunMode int

const (
	// RunModePerformLayout means a full layout for this node and all children should be computed.
	RunModePerformLayout RunMode = iota
	// RunModeComputeSize means the layout algorithm should execute to determine an accurate
	// container size. Layout steps unnecessary for determining size may be skipped.
	RunModeComputeSize
	// RunModePerformHiddenLayout means this node should have a null layout as it has been hidden
	// (e.g. using Display::None).
	RunModePerformHiddenLayout
)

// SizingMode indicates whether styles should be taken into account when computing size.
type SizingMode int

const (
	// SizingModeContentSize means only content contributions should be taken into account.
	SizingModeContentSize SizingMode = iota
	// SizingModeInherentSize means inherent size styles should be taken into account
	// in addition to content contributions.
	SizingModeInherentSize
)

// RequestedAxis is an axis that layout algorithms can be requested to compute a size for.
type RequestedAxis int

const (
	// RequestedAxisHorizontal is the horizontal axis.
	RequestedAxisHorizontal RequestedAxis = iota
	// RequestedAxisVertical is the vertical axis.
	RequestedAxisVertical
	// RequestedAxisBoth means both axes.
	RequestedAxisBoth
)

// RequestedAxisFromAbsolute converts an AbsoluteAxis to a RequestedAxis.
func RequestedAxisFromAbsolute(axis geometry.AbsoluteAxis) RequestedAxis {
	switch axis {
	case geometry.AbsoluteAxisHorizontal:
		return RequestedAxisHorizontal
	default:
		return RequestedAxisVertical
	}
}

// CollapsibleMarginSet is a set of margins available for collapsing with, for block layout's
// margin collapsing.
type CollapsibleMarginSet struct {
	// Positive is the largest positive margin.
	Positive float32
	// Negative is the smallest negative margin (with largest absolute value).
	Negative float32
}

// ZeroCollapsibleMarginSet is a default margin set with no collapsible margins.
var ZeroCollapsibleMarginSet = CollapsibleMarginSet{Positive: 0.0, Negative: 0.0}

// FromMargin creates a CollapsibleMarginSet from a single margin value.
func (CollapsibleMarginSet) FromMargin(margin float32) CollapsibleMarginSet {
	if margin >= 0.0 {
		return CollapsibleMarginSet{Positive: margin, Negative: 0.0}
	}
	return CollapsibleMarginSet{Positive: 0.0, Negative: margin}
}

// CollapsibleMarginSetFromMargin creates a CollapsibleMarginSet from a single margin value.
func CollapsibleMarginSetFromMargin(margin float32) CollapsibleMarginSet {
	if margin >= 0.0 {
		return CollapsibleMarginSet{Positive: margin, Negative: 0.0}
	}
	return CollapsibleMarginSet{Positive: 0.0, Negative: margin}
}

// CollapseWithMargin collapses a single margin value into this set and returns the result.
func (c CollapsibleMarginSet) CollapseWithMargin(margin float32) CollapsibleMarginSet {
	if margin >= 0.0 {
		if margin > c.Positive {
			c.Positive = margin
		}
	} else {
		if margin < c.Negative {
			c.Negative = margin
		}
	}
	return c
}

// CollapseWithSet collapses another CollapsibleMarginSet into this one and returns the result.
func (c CollapsibleMarginSet) CollapseWithSet(other CollapsibleMarginSet) CollapsibleMarginSet {
	if other.Positive > c.Positive {
		c.Positive = other.Positive
	}
	if other.Negative < c.Negative {
		c.Negative = other.Negative
	}
	return c
}

// Resolve resolves the resultant margin from this set once all collapsible margins
// have been collapsed into it.
func (c CollapsibleMarginSet) Resolve() float32 {
	return c.Positive + c.Negative
}

// LayoutInput contains the input constraints/hints for laying out a node,
// passed in by the parent.
type LayoutInput struct {
	// RunMode indicates whether we only need to know the node's size or need a full layout.
	RunMode RunMode
	// SizingMode indicates whether a node's style sizes should be taken into account or ignored.
	SizingMode SizingMode
	// Axis is which axis we need the size of.
	Axis RequestedAxis
	// KnownDimensions are dimensions (width/height) which should be taken as fixed when performing layout.
	KnownDimensions geometry.Size[lib.Option[float32]]
	// ParentSize dimensions are intended to be used for percentage resolution.
	ParentSize geometry.Size[lib.Option[float32]]
	// AvailableSpace represents an amount of space to layout into, used as a soft constraint for wrapping.
	AvailableSpace geometry.Size[style.AvailableSpace]
	// VerticalMarginsAreCollapsible is specific to CSS Block layout for correctly computing margin collapsing.
	VerticalMarginsAreCollapsible geometry.Line[bool]
}

// HiddenLayoutInput is a LayoutInput that can be used to request hidden layout.
var HiddenLayoutInput = LayoutInput{
	RunMode:                       RunModePerformHiddenLayout,
	KnownDimensions:               geometry.SizeOptionNone,
	ParentSize:                    geometry.SizeOptionNone,
	AvailableSpace:                geometry.Size[style.AvailableSpace]{Width: style.AvailableSpaceMaxContentValue, Height: style.AvailableSpaceMaxContentValue},
	SizingMode:                    SizingModeInherentSize,
	Axis:                          RequestedAxisBoth,
	VerticalMarginsAreCollapsible: geometry.LineFalseBool,
}

// LayoutOutput contains the result of laying a single node, returned up to the parent node.
type LayoutOutput struct {
	// Size is the size of the node.
	Size geometry.Size[float32]
	// ContentSize is the size of the content within the node.
	ContentSize geometry.Size[float32]
	// FirstBaselines is the first baseline of the node in each dimension, if any.
	FirstBaselines geometry.Point[lib.Option[float32]]
	// TopMargin is the top margin that can be collapsed with (for CSS block layout).
	TopMargin CollapsibleMarginSet
	// BottomMargin is the bottom margin that can be collapsed with (for CSS block layout).
	BottomMargin CollapsibleMarginSet
	// MarginsCanCollapseThrough indicates whether margins can be collapsed through this node.
	MarginsCanCollapseThrough bool
}

// HiddenLayoutOutput is an all-zero LayoutOutput for hidden nodes.
var HiddenLayoutOutput = LayoutOutput{
	Size:                      geometry.SizeZeroF32,
	ContentSize:               geometry.SizeZeroF32,
	FirstBaselines:            geometry.PointNoneF32,
	TopMargin:                 ZeroCollapsibleMarginSet,
	BottomMargin:              ZeroCollapsibleMarginSet,
	MarginsCanCollapseThrough: false,
}

// DefaultLayoutOutput is a blank layout output.
var DefaultLayoutOutput = HiddenLayoutOutput

// LayoutOutputFromSizesAndBaselines constructs a LayoutOutput from size, content size, and baselines.
func LayoutOutputFromSizesAndBaselines(size geometry.Size[float32], contentSize geometry.Size[float32], firstBaselines geometry.Point[lib.Option[float32]]) LayoutOutput {
	return LayoutOutput{
		Size:                      size,
		ContentSize:               contentSize,
		FirstBaselines:            firstBaselines,
		TopMargin:                 ZeroCollapsibleMarginSet,
		BottomMargin:              ZeroCollapsibleMarginSet,
		MarginsCanCollapseThrough: false,
	}
}

// LayoutOutputFromSizes constructs a LayoutOutput from just the container and content sizes.
func LayoutOutputFromSizes(size geometry.Size[float32], contentSize geometry.Size[float32]) LayoutOutput {
	return LayoutOutputFromSizesAndBaselines(size, contentSize, geometry.PointNoneF32)
}

// LayoutOutputFromOuterSize constructs a LayoutOutput from just the container's size.
func LayoutOutputFromOuterSize(size geometry.Size[float32]) LayoutOutput {
	return LayoutOutputFromSizes(size, geometry.SizeZeroF32)
}

// Layout is the final result of a layout algorithm for a single node.
type Layout struct {
	// Order is the relative ordering of the node. Nodes with a higher order should be rendered on top.
	Order uint32
	// Location is the top-left corner of the node.
	Location geometry.Point[float32]
	// Size is the width and height of the node.
	Size geometry.Size[float32]
	// ContentSize is the width and height of the content inside the node (may be larger due to overflow).
	ContentSize geometry.Size[float32]
	// ScrollbarSize is the size of the scrollbars in each dimension.
	ScrollbarSize geometry.Size[float32]
	// Border is the size of the borders of the node.
	Border geometry.Rect[float32]
	// Padding is the size of the padding of the node.
	Padding geometry.Rect[float32]
	// Margin is the size of the margin of the node.
	Margin geometry.Rect[float32]
}

// NewLayout creates a new zero Layout.
// The order is set to 0 (rendered below all other layouts).
func NewLayout() Layout {
	return Layout{
		Order:         0,
		Location:      geometry.PointZeroF32,
		Size:          geometry.SizeZeroF32,
		ContentSize:   geometry.SizeZeroF32,
		ScrollbarSize: geometry.SizeZeroF32,
		Border:        geometry.RectZeroF32,
		Padding:       geometry.RectZeroF32,
		Margin:        geometry.RectZeroF32,
	}
}

// NewLayoutWithOrder creates a new zero Layout with the supplied order value.
func NewLayoutWithOrder(order uint32) Layout {
	return Layout{
		Order:         order,
		Location:      geometry.PointZeroF32,
		Size:          geometry.SizeZeroF32,
		ContentSize:   geometry.SizeZeroF32,
		ScrollbarSize: geometry.SizeZeroF32,
		Border:        geometry.RectZeroF32,
		Padding:       geometry.RectZeroF32,
		Margin:        geometry.RectZeroF32,
	}
}

// ContentBoxWidth returns the width of the node's content box.
func (l *Layout) ContentBoxWidth() float32 {
	return l.Size.Width - l.Padding.Left - l.Padding.Right - l.Border.Left - l.Border.Right
}

// ContentBoxHeight returns the height of the node's content box.
func (l *Layout) ContentBoxHeight() float32 {
	return l.Size.Height - l.Padding.Top - l.Padding.Bottom - l.Border.Top - l.Border.Bottom
}

// ContentBoxSize returns the size of the node's content box.
func (l *Layout) ContentBoxSize() geometry.Size[float32] {
	return geometry.Size[float32]{Width: l.ContentBoxWidth(), Height: l.ContentBoxHeight()}
}

// ContentBoxX returns the x offset of the node's content box relative to its parent's border box.
func (l *Layout) ContentBoxX() float32 {
	return l.Location.X + l.Border.Left + l.Padding.Left
}

// ContentBoxY returns the y offset of the node's content box relative to its parent's border box.
func (l *Layout) ContentBoxY() float32 {
	return l.Location.Y + l.Border.Top + l.Padding.Top
}

// ScrollWidth returns the scroll width of the node.
// The scroll width is the difference between the content width and node width, floored at zero.
func (l *Layout) ScrollWidth() float32 {
	scrollbarWidth := l.ScrollbarSize.Width
	if scrollbarWidth > l.Size.Width {
		scrollbarWidth = l.Size.Width
	}
	result := l.ContentSize.Width + scrollbarWidth - l.Size.Width + l.Border.Right
	if result < 0 {
		return 0
	}
	return result
}

// ScrollHeight returns the scroll height of the node.
// The scroll height is the difference between the content height and node height, floored at zero.
func (l *Layout) ScrollHeight() float32 {
	scrollbarHeight := l.ScrollbarSize.Height
	if scrollbarHeight > l.Size.Height {
		scrollbarHeight = l.Size.Height
	}
	result := l.ContentSize.Height + scrollbarHeight - l.Size.Height + l.Border.Bottom
	if result < 0 {
		return 0
	}
	return result
}
