// Package compute provides the leaf node layout computation.
// This file is a port of compute/leaf.rs from the Taffy Rust layout engine.
package compute

import (
	"github.com/gataengine/taffy-go/lib"
	"github.com/gataengine/taffy-go/geometry"
	"github.com/gataengine/taffy-go/style"
	"github.com/gataengine/taffy-go/tree"
)

// ComputeLeafLayout computes the layout for a leaf node (a node with no children).
//
// The measureFunction is called when the node's size cannot be determined from style
// alone. It receives optional known dimensions and available space, and returns the
// measured size of the node's content.
//
// If measureFunction is nil, the node is assumed to have zero content size.
func ComputeLeafLayout(
	inputs tree.LayoutInput,
	st style.CoreStyle,
	calc style.CalcResolver,
	measureFunction func(geometry.Size[lib.Option[float32]], geometry.Size[style.AvailableSpace]) geometry.Size[float32],
) tree.LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	parentSize := inputs.ParentSize
	availableSpace := inputs.AvailableSpace
	sizingMode := inputs.SizingMode
	runMode := inputs.RunMode

	// Note: both horizontal and vertical percentage padding/border are resolved
	// against the container's inline size (i.e. width). This is how CSS is specified.
	// See: https://developer.mozilla.org/en-US/docs/Web/CSS/padding#values
	margin := style.ResolveRectLPAWidth(st.GetMargin(), parentSize.Width, calc)
	padding := style.ResolveRectLPWidth(st.GetPadding(), parentSize.Width, calc)
	border := style.ResolveRectLPWidth(st.GetBorder(), parentSize.Width, calc)
	paddingBorder := geometry.RectAddF32(padding, border)
	pbSum := geometry.RectSumAxes(paddingBorder)

	boxSizingAdjustment := geometry.SizeZeroF32
	if st.GetBoxSizing() == style.BoxSizingContentBox {
		boxSizingAdjustment = pbSum
	}

	// Resolve node's preferred/min/max sizes against the available space.
	// For ContentSize mode, we pretend the node has no size styles.
	var (
		nodeSize    geometry.Size[lib.Option[float32]]
		nodeMinSize geometry.Size[lib.Option[float32]]
		nodeMaxSize geometry.Size[lib.Option[float32]]
		aspectRatio lib.Option[float32]
	)

	switch sizingMode {
	case tree.SizingModeContentSize:
		nodeSize = knownDimensions
		nodeMinSize = geometry.SizeOptionNone
		nodeMaxSize = geometry.SizeOptionNone
		aspectRatio = lib.None[float32]()

	case tree.SizingModeInherentSize:
		aspectRatio = st.GetAspectRatio()

		styleSize := geometry.SizeOptionMaybeApplyAspectRatio(
			geometry.SizeMaybeAdd(
				style.MaybeResolveSizeDimOption(st.GetSize(), parentSize, calc),
				geometry.Size[lib.Option[float32]]{
					Width:  lib.Some(boxSizingAdjustment.Width),
					Height: lib.Some(boxSizingAdjustment.Height),
				},
			),
			aspectRatio,
		)

		styleMinSize := geometry.SizeOptionMaybeApplyAspectRatio(
			geometry.SizeMaybeAdd(
				style.MaybeResolveSizeDimOption(st.GetMinSize(), parentSize, calc),
				geometry.Size[lib.Option[float32]]{
					Width:  lib.Some(boxSizingAdjustment.Width),
					Height: lib.Some(boxSizingAdjustment.Height),
				},
			),
			aspectRatio,
		)

		styleMaxSize := geometry.SizeMaybeAdd(
			style.MaybeResolveSizeDimOption(st.GetMaxSize(), parentSize, calc),
			geometry.Size[lib.Option[float32]]{
				Width:  lib.Some(boxSizingAdjustment.Width),
				Height: lib.Some(boxSizingAdjustment.Height),
			},
		)

		nodeSize = geometry.SizeOptionOr(knownDimensions, styleSize)
		nodeMinSize = styleMinSize
		nodeMaxSize = styleMaxSize
	}

	// Scrollbar gutters are reserved when overflow == Scroll.
	// The axes are transposed because a node that scrolls vertically needs
	// horizontal space reserved for a scrollbar.
	overflow := st.GetOverflow()
	scrollbarGutter := geometry.Point[float32]{
		X: func() float32 {
			if overflow.Y == style.OverflowScroll {
				return st.GetScrollbarWidth()
			}
			return 0.0
		}(),
		Y: func() float32 {
			if overflow.X == style.OverflowScroll {
				return st.GetScrollbarWidth()
			}
			return 0.0
		}(),
	}

	contentBoxInset := paddingBorder
	contentBoxInset.Right += scrollbarGutter.X
	contentBoxInset.Bottom += scrollbarGutter.Y

	hasStylesPreventingBeingCollapsedThrough := !st.GetIsBlock() ||
		overflow.X.IsScrollContainer() ||
		overflow.Y.IsScrollContainer() ||
		st.GetPosition() == style.PositionAbsolute ||
		padding.Top > 0.0 ||
		padding.Bottom > 0.0 ||
		border.Top > 0.0 ||
		border.Bottom > 0.0 ||
		(nodeSize.Height.IsSome && nodeSize.Height.Value > 0.0) ||
		(nodeMinSize.Height.IsSome && nodeMinSize.Height.Value > 0.0)

	// Return early if both dimensions are known and styles prevent collapsing through.
	if runMode == tree.RunModeComputeSize && hasStylesPreventingBeingCollapsedThrough {
		if nodeSize.Width.IsSome && nodeSize.Height.IsSome {
			w := nodeSize.Width.Value
			h := nodeSize.Height.Value
			size := geometry.Size[float32]{Width: w, Height: h}
			size = geometry.Size[float32]{
				Width:  geometry.MaybeClampF32(size.Width, nodeMinSize.Width, nodeMaxSize.Width),
				Height: geometry.MaybeClampF32(size.Height, nodeMinSize.Height, nodeMaxSize.Height),
			}
			pbSumOptionW := lib.Some(pbSum.Width)
			pbSumOptionH := lib.Some(pbSum.Height)
			size = geometry.Size[float32]{
				Width:  geometry.MaybeMaxF32(size.Width, pbSumOptionW),
				Height: geometry.MaybeMaxF32(size.Height, pbSumOptionH),
			}
			return tree.LayoutOutput{
				Size:                      size,
				ContentSize:               geometry.SizeZeroF32,
				FirstBaselines:            geometry.PointNoneF32,
				TopMargin:                 tree.ZeroCollapsibleMarginSet,
				BottomMargin:              tree.ZeroCollapsibleMarginSet,
				MarginsCanCollapseThrough: false,
			}
		}
	}

	// Compute available space for the measure function.
	cbiHSum := contentBoxInset.Left + contentBoxInset.Right
	cbiVSum := contentBoxInset.Top + contentBoxInset.Bottom

	computedAvailableSpace := geometry.Size[style.AvailableSpace]{
		Width: func() style.AvailableSpace {
			as := style.AvailableSpaceFromOptionFloat32(knownDimensions.Width).Or(availableSpace.Width)
			as = style.AsMaybeSubOption(as, lib.Some(geometry.RectHorizontalAxisSum(margin)))
			as = as.MaybeSet(knownDimensions.Width)
			as = as.MaybeSet(nodeSize.Width)
			return as.MapDefiniteValue(func(size float32) float32 {
				clamped := geometry.MaybeClampF32(size, nodeMinSize.Width, nodeMaxSize.Width)
				return clamped - cbiHSum
			})
		}(),
		Height: func() style.AvailableSpace {
			as := style.AvailableSpaceFromOptionFloat32(knownDimensions.Height).Or(availableSpace.Height)
			as = style.AsMaybeSubOption(as, lib.Some(geometry.RectVerticalAxisSum(margin)))
			as = as.MaybeSet(knownDimensions.Height)
			as = as.MaybeSet(nodeSize.Height)
			return as.MapDefiniteValue(func(size float32) float32 {
				clamped := geometry.MaybeClampF32(size, nodeMinSize.Height, nodeMaxSize.Height)
				return clamped - cbiVSum
			})
		}(),
	}

	// Determine the known dimensions to pass to the measure function.
	var measureKnownDimensions geometry.Size[lib.Option[float32]]
	switch runMode {
	case tree.RunModeComputeSize:
		measureKnownDimensions = knownDimensions
	case tree.RunModePerformLayout:
		measureKnownDimensions = geometry.SizeOptionNone
	case tree.RunModePerformHiddenLayout:
		panic("ComputeLeafLayout: RunModePerformHiddenLayout is not valid for leaf layout")
	}

	// Call the measure function (or return zero if none provided).
	var measuredSize geometry.Size[float32]
	if measureFunction != nil {
		measuredSize = measureFunction(measureKnownDimensions, computedAvailableSpace)
	}

	contentBoxInsetSum := geometry.Size[float32]{Width: cbiHSum, Height: cbiVSum}
	clampedSize := geometry.Size[lib.Option[float32]]{
		Width: geometry.SizeOptionOr(knownDimensions, nodeSize).Width.Or(
			lib.Some(measuredSize.Width + contentBoxInsetSum.Width),
		),
		Height: geometry.SizeOptionOr(knownDimensions, nodeSize).Height.Or(
			lib.Some(measuredSize.Height + contentBoxInsetSum.Height),
		),
	}
	clampedSizeF32 := geometry.Size[float32]{
		Width:  geometry.MaybeClampF32(clampedSize.Width.UnwrapOr(0.0), nodeMinSize.Width, nodeMaxSize.Width),
		Height: geometry.MaybeClampF32(clampedSize.Height.UnwrapOr(0.0), nodeMinSize.Height, nodeMaxSize.Height),
	}

	// Apply aspect ratio to height if needed.
	aspectRatioHeight := float32(0.0)
	if aspectRatio.IsSome {
		aspectRatioHeight = clampedSizeF32.Width / aspectRatio.Value
	}
	size := geometry.Size[float32]{
		Width:  clampedSizeF32.Width,
		Height: geometry.F32Max(clampedSizeF32.Height, aspectRatioHeight),
	}

	// Ensure the size is at least the padding+border sum.
	size = geometry.Size[float32]{
		Width:  geometry.MaybeMaxF32(size.Width, lib.Some(pbSum.Width)),
		Height: geometry.MaybeMaxF32(size.Height, lib.Some(pbSum.Height)),
	}

	// Compute the content size (measured size + padding sum axes).
	paddingSum := geometry.RectSumAxes(padding)
	contentSize := geometry.SizeAddF32(measuredSize, paddingSum)

	marginsCanCollapseThrough := !hasStylesPreventingBeingCollapsedThrough &&
		size.Height == 0.0 &&
		measuredSize.Height == 0.0

	return tree.LayoutOutput{
		Size:                      size,
		ContentSize:               contentSize,
		FirstBaselines:            geometry.PointNoneF32,
		TopMargin:                 tree.ZeroCollapsibleMarginSet,
		BottomMargin:              tree.ZeroCollapsibleMarginSet,
		MarginsCanCollapseThrough: marginsCanCollapseThrough,
	}
}
