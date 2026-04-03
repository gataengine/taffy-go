// Package taffy provides the leaf node layout computation.
// This file is a port of compute/leaf.rs from the Taffy Rust layout engine.
package taffy

// resolveRectLPWidth resolves a Rect[LengthPercentage] to Rect[float32] where all four sides
// are resolved against the same inline (width) context. This mirrors the CSS behaviour
// where both horizontal and vertical padding/border percentages are resolved against the
// container's inline size.
func resolveRectLPWidth(rect Rect[LengthPercentage], widthContext Option[float32], calc CalcResolver) Rect[float32] {
	return Rect[float32]{
		Left:   ResolveOrZeroLP(rect.Left, widthContext, calc),
		Right:  ResolveOrZeroLP(rect.Right, widthContext, calc),
		Top:    ResolveOrZeroLP(rect.Top, widthContext, calc),
		Bottom: ResolveOrZeroLP(rect.Bottom, widthContext, calc),
	}
}

// resolveRectLPAWidth resolves a Rect[LengthPercentageAuto] to Rect[float32] where all four
// sides are resolved against the same inline (width) context.
func resolveRectLPAWidth(rect Rect[LengthPercentageAuto], widthContext Option[float32], calc CalcResolver) Rect[float32] {
	return Rect[float32]{
		Left:   ResolveOrZeroLPA(rect.Left, widthContext, calc),
		Right:  ResolveOrZeroLPA(rect.Right, widthContext, calc),
		Top:    ResolveOrZeroLPA(rect.Top, widthContext, calc),
		Bottom: ResolveOrZeroLPA(rect.Bottom, widthContext, calc),
	}
}

// computeLeafLayout computes the layout for a leaf node (a node with no children).
//
// The measureFunction is called when the node's size cannot be determined from style
// alone. It receives optional known dimensions and available space, and returns the
// measured size of the node's content.
//
// If measureFunction is nil, the node is assumed to have zero content size.
func computeLeafLayout(
	inputs LayoutInput,
	style coreStyle,
	calc CalcResolver,
	measureFunction func(Size[Option[float32]], Size[AvailableSpace]) Size[float32],
) LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	parentSize := inputs.ParentSize
	availableSpace := inputs.AvailableSpace
	sizingMode := inputs.SizingMode
	runMode := inputs.RunMode

	// Note: both horizontal and vertical percentage padding/border are resolved
	// against the container's inline size (i.e. width). This is how CSS is specified.
	// See: https://developer.mozilla.org/en-US/docs/Web/CSS/padding#values
	margin := resolveRectLPAWidth(style.margin(), parentSize.Width, calc)
	padding := resolveRectLPWidth(style.padding(), parentSize.Width, calc)
	border := resolveRectLPWidth(style.border(), parentSize.Width, calc)
	paddingBorder := RectAddF32(padding, border)
	pbSum := RectSumAxes(paddingBorder)

	boxSizingAdjustment := SizeZeroF32
	if style.boxSizing() == BoxSizingContentBox {
		boxSizingAdjustment = pbSum
	}

	// Resolve node's preferred/min/max sizes against the available space.
	// For ContentSize mode, we pretend the node has no size styles.
	var (
		nodeSize    Size[Option[float32]]
		nodeMinSize Size[Option[float32]]
		nodeMaxSize Size[Option[float32]]
		aspectRatio Option[float32]
	)

	switch sizingMode {
	case SizingModeContentSize:
		nodeSize = knownDimensions
		nodeMinSize = SizeOptionNone
		nodeMaxSize = SizeOptionNone
		aspectRatio = None[float32]()

	case SizingModeInherentSize:
		aspectRatio = style.aspectRatio()

		styleSize := SizeOptionMaybeApplyAspectRatio(
			SizeMaybeAdd(
				MaybeResolveSizeDimOption(style.size(), parentSize, calc),
				Size[Option[float32]]{
					Width:  Some(boxSizingAdjustment.Width),
					Height: Some(boxSizingAdjustment.Height),
				},
			),
			aspectRatio,
		)

		styleMinSize := SizeOptionMaybeApplyAspectRatio(
			SizeMaybeAdd(
				MaybeResolveSizeDimOption(style.minSize(), parentSize, calc),
				Size[Option[float32]]{
					Width:  Some(boxSizingAdjustment.Width),
					Height: Some(boxSizingAdjustment.Height),
				},
			),
			aspectRatio,
		)

		styleMaxSize := SizeMaybeAdd(
			MaybeResolveSizeDimOption(style.maxSize(), parentSize, calc),
			Size[Option[float32]]{
				Width:  Some(boxSizingAdjustment.Width),
				Height: Some(boxSizingAdjustment.Height),
			},
		)

		nodeSize = SizeOptionOr(knownDimensions, styleSize)
		nodeMinSize = styleMinSize
		nodeMaxSize = styleMaxSize
	}

	// Scrollbar gutters are reserved when overflow == Scroll.
	// The axes are transposed because a node that scrolls vertically needs
	// horizontal space reserved for a scrollbar.
	overflow := style.overflow()
	scrollbarGutter := Point[float32]{
		X: func() float32 {
			if overflow.Y == OverflowScroll {
				return style.scrollbarWidth()
			}
			return 0.0
		}(),
		Y: func() float32 {
			if overflow.X == OverflowScroll {
				return style.scrollbarWidth()
			}
			return 0.0
		}(),
	}

	contentBoxInset := paddingBorder
	contentBoxInset.Right += scrollbarGutter.X
	contentBoxInset.Bottom += scrollbarGutter.Y

	hasStylesPreventingBeingCollapsedThrough := !style.isBlock() ||
		overflow.X.IsScrollContainer() ||
		overflow.Y.IsScrollContainer() ||
		style.position() == PositionAbsolute ||
		padding.Top > 0.0 ||
		padding.Bottom > 0.0 ||
		border.Top > 0.0 ||
		border.Bottom > 0.0 ||
		(nodeSize.Height.IsSome && nodeSize.Height.Value > 0.0) ||
		(nodeMinSize.Height.IsSome && nodeMinSize.Height.Value > 0.0)

	// Return early if both dimensions are known and styles prevent collapsing through.
	if runMode == RunModeComputeSize && hasStylesPreventingBeingCollapsedThrough {
		if nodeSize.Width.IsSome && nodeSize.Height.IsSome {
			w := nodeSize.Width.Value
			h := nodeSize.Height.Value
			size := Size[float32]{Width: w, Height: h}
			size = Size[float32]{
				Width:  maybeClampF32(size.Width, nodeMinSize.Width, nodeMaxSize.Width),
				Height: maybeClampF32(size.Height, nodeMinSize.Height, nodeMaxSize.Height),
			}
			pbSumOptionW := Some(pbSum.Width)
			pbSumOptionH := Some(pbSum.Height)
			size = Size[float32]{
				Width:  maybeMaxF32(size.Width, pbSumOptionW),
				Height: maybeMaxF32(size.Height, pbSumOptionH),
			}
			return LayoutOutput{
				Size:                      size,
				ContentSize:               SizeZeroF32,
				FirstBaselines:            PointNoneF32,
				TopMargin:                 zeroCollapsibleMarginSet,
				BottomMargin:              zeroCollapsibleMarginSet,
				MarginsCanCollapseThrough: false,
			}
		}
	}

	// Compute available space for the measure function.
	cbiHSum := contentBoxInset.Left + contentBoxInset.Right
	cbiVSum := contentBoxInset.Top + contentBoxInset.Bottom

	computedAvailableSpace := Size[AvailableSpace]{
		Width: func() AvailableSpace {
			as := AvailableSpaceFromOptionFloat32(knownDimensions.Width).Or(availableSpace.Width)
			as = AsMaybeSubOption(as, Some(RectHorizontalAxisSum(margin)))
			as = as.MaybeSet(knownDimensions.Width)
			as = as.MaybeSet(nodeSize.Width)
			return as.MapDefiniteValue(func(size float32) float32 {
				clamped := maybeClampF32(size, nodeMinSize.Width, nodeMaxSize.Width)
				return clamped - cbiHSum
			})
		}(),
		Height: func() AvailableSpace {
			as := AvailableSpaceFromOptionFloat32(knownDimensions.Height).Or(availableSpace.Height)
			as = AsMaybeSubOption(as, Some(RectVerticalAxisSum(margin)))
			as = as.MaybeSet(knownDimensions.Height)
			as = as.MaybeSet(nodeSize.Height)
			return as.MapDefiniteValue(func(size float32) float32 {
				clamped := maybeClampF32(size, nodeMinSize.Height, nodeMaxSize.Height)
				return clamped - cbiVSum
			})
		}(),
	}

	// Determine the known dimensions to pass to the measure function.
	var measureKnownDimensions Size[Option[float32]]
	switch runMode {
	case RunModeComputeSize:
		measureKnownDimensions = knownDimensions
	case RunModePerformLayout:
		measureKnownDimensions = SizeOptionNone
	case RunModePerformHiddenLayout:
		panic("computeLeafLayout: RunModePerformHiddenLayout is not valid for leaf layout")
	}

	// Call the measure function (or return zero if none provided).
	var measuredSize Size[float32]
	if measureFunction != nil {
		measuredSize = measureFunction(measureKnownDimensions, computedAvailableSpace)
	}

	contentBoxInsetSum := Size[float32]{Width: cbiHSum, Height: cbiVSum}
	clampedSize := Size[Option[float32]]{
		Width: SizeOptionOr(knownDimensions, nodeSize).Width.Or(
			Some(measuredSize.Width + contentBoxInsetSum.Width),
		),
		Height: SizeOptionOr(knownDimensions, nodeSize).Height.Or(
			Some(measuredSize.Height + contentBoxInsetSum.Height),
		),
	}
	clampedSizeF32 := Size[float32]{
		Width:  maybeClampF32(clampedSize.Width.UnwrapOr(0.0), nodeMinSize.Width, nodeMaxSize.Width),
		Height: maybeClampF32(clampedSize.Height.UnwrapOr(0.0), nodeMinSize.Height, nodeMaxSize.Height),
	}

	// Apply aspect ratio to height if needed.
	aspectRatioHeight := float32(0.0)
	if aspectRatio.IsSome {
		aspectRatioHeight = clampedSizeF32.Width / aspectRatio.Value
	}
	size := Size[float32]{
		Width:  clampedSizeF32.Width,
		Height: f32Max(clampedSizeF32.Height, aspectRatioHeight),
	}

	// Ensure the size is at least the padding+border sum.
	size = Size[float32]{
		Width:  maybeMaxF32(size.Width, Some(pbSum.Width)),
		Height: maybeMaxF32(size.Height, Some(pbSum.Height)),
	}

	// Compute the content size (measured size + padding sum axes).
	paddingSum := RectSumAxes(padding)
	contentSize := SizeAddF32(measuredSize, paddingSum)

	marginsCanCollapseThrough := !hasStylesPreventingBeingCollapsedThrough &&
		size.Height == 0.0 &&
		measuredSize.Height == 0.0

	return LayoutOutput{
		Size:                      size,
		ContentSize:               contentSize,
		FirstBaselines:            PointNoneF32,
		TopMargin:                 zeroCollapsibleMarginSet,
		BottomMargin:              zeroCollapsibleMarginSet,
		MarginsCanCollapseThrough: marginsCanCollapseThrough,
	}
}
