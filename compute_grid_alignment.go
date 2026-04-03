// Package taffy implements alignment of grid tracks and final positioning of grid items.
// This file is a port of compute/grid/alignment.rs from the Taffy Rust layout engine.
package taffy


// alignGridTracks aligns the grid tracks within the grid container according to
// align-content (rows) or justify-content (columns).
func alignGridTracks(
	gridContainerContentBoxSize float32,
	padding Line[float32],
	border Line[float32],
	tracks []gridTrack,
	trackAlignmentStyle AlignContent,
	axisReversed bool,
) {
	var usedSize float32
	for i := range tracks {
		usedSize += tracks[i].baseSize
	}
	freeSpace := gridContainerContentBoxSize - usedSize
	origin := padding.Start + border.Start

	// Count non-collapsed tracks (skipping gutters)
	numTracks := 0
	for i := 1; i < len(tracks); i += 2 {
		if !tracks[i].isCollapsed {
			numTracks++
		}
	}

	gap := float32(0.0)
	layoutIsReversed := false
	isSafe := false
	trackAlignment := applyAlignmentFallback(freeSpace, numTracks, trackAlignmentStyle, isSafe)
	if axisReversed {
		trackAlignment = trackAlignment.Reversed()
	}

	// Compute offsets
	totalOffset := origin
	seenNonCollapsedTrack := false
	for i := range tracks {
		isGutter := i%2 == 0
		isNonCollapsedTrack := !isGutter && !tracks[i].isCollapsed

		isFirst := isNonCollapsedTrack && !seenNonCollapsedTrack

		offset := float32(0.0)
		if isNonCollapsedTrack {
			offset = computeAlignmentOffset(freeSpace, numTracks, gap, trackAlignment, layoutIsReversed, isFirst)
		}

		tracks[i].offset = totalOffset + offset
		totalOffset = totalOffset + offset + tracks[i].baseSize
		if isNonCollapsedTrack {
			seenNonCollapsedTrack = true
		}
	}
}

// alignAndPositionItem aligns and sizes a grid item into its final position.
// Returns (contentSizeContribution, yPosition, height).
func alignAndPositionItem(
	tree layoutGridContainer,
	node NodeId,
	order uint32,
	gridArea Rect[float32],
	containerAlignmentStyles inBothAbsAxis[Option[AlignItems]],
	baselineShim float32,
	direction Direction,
) (Size[float32], float32, float32) {
	gridAreaSize := Size[float32]{
		Width:  gridArea.Right - gridArea.Left,
		Height: gridArea.Bottom - gridArea.Top,
	}

	style := tree.getGridChildStyle(node)
	calc := tree.resolveCalcValue

	overflow := style.overflow()
	scrollbarWidth := style.scrollbarWidth()
	aspectRatio := style.aspectRatio()
	justifySelf := style.justifySelf()
	alignSelf := style.alignSelf()

	position := style.position()
	insetHorizontal := Line[Option[float32]]{
		Start: gridMaybeResolveDimOption(style.inset().Left, gridAreaSize.Width, calc),
		End:   gridMaybeResolveDimOption(style.inset().Right, gridAreaSize.Width, calc),
	}
	insetVertical := Line[Option[float32]]{
		Start: gridMaybeResolveDimOption(style.inset().Top, gridAreaSize.Height, calc),
		End:   gridMaybeResolveDimOption(style.inset().Bottom, gridAreaSize.Height, calc),
	}
	padding := resolveRectLPWidth(style.padding(), Some(gridAreaSize.Width), calc)
	border := resolveRectLPWidth(style.border(), Some(gridAreaSize.Width), calc)
	paddingBorderSize := RectSumAxes(RectAddF32(padding, border))

	boxSizingAdjustment := SizeZeroF32
	if style.boxSizing() == BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSize
	}

	gridAreaSizeOpt := Size[Option[float32]]{Width: Some(gridAreaSize.Width), Height: Some(gridAreaSize.Height)}
	inherentSize := SizeMaybeAdd(
		SizeOptionMaybeApplyAspectRatio(
			MaybeResolveSizeDimOption(style.size(), gridAreaSizeOpt, calc),
			aspectRatio,
		),
		Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)},
	)
	minSize := SizeMaybeMax(
		SizeOptionOr(
			SizeMaybeAdd(
				SizeOptionMaybeApplyAspectRatio(
					MaybeResolveSizeDimOption(style.minSize(), gridAreaSizeOpt, calc),
					aspectRatio,
				),
				Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)},
			),
			Size[Option[float32]]{Width: Some(paddingBorderSize.Width), Height: Some(paddingBorderSize.Height)},
		),
		Size[Option[float32]]{Width: Some(paddingBorderSize.Width), Height: Some(paddingBorderSize.Height)},
	)
	maxSize := SizeMaybeAdd(
		SizeOptionMaybeApplyAspectRatio(
			MaybeResolveSizeDimOption(style.maxSize(), gridAreaSizeOpt, calc),
			aspectRatio,
		),
		Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)},
	)

	// Resolve alignment styles
	horizontalAlignment := justifySelf.UnwrapOr(containerAlignmentStyles.horizontal.UnwrapOr(func() AlignSelf {
		if inherentSize.Width.IsSome {
			return AlignItemsStart
		}
		return AlignItemsStretch
	}()))
	verticalAlignment := alignSelf.UnwrapOr(containerAlignmentStyles.vertical.UnwrapOr(func() AlignSelf {
		if inherentSize.Height.IsSome || aspectRatio.IsSome {
			return AlignItemsStart
		}
		return AlignItemsStretch
	}()))

	// Resolve margins (both horizontal and vertical against grid area WIDTH per CSS spec)
	margin := Rect[Option[float32]]{
		Left:   MaybeResolveLPAOption(style.margin().Left, Some(gridAreaSize.Width), calc),
		Right:  MaybeResolveLPAOption(style.margin().Right, Some(gridAreaSize.Width), calc),
		Top:    MaybeResolveLPAOption(style.margin().Top, Some(gridAreaSize.Width), calc),
		Bottom: MaybeResolveLPAOption(style.margin().Bottom, Some(gridAreaSize.Width), calc),
	}

	gridAreaMinusItemMarginsSize := Size[float32]{
		Width:  gridAreaSize.Width - margin.Left.UnwrapOr(0.0) - margin.Right.UnwrapOr(0.0),
		Height: gridAreaSize.Height - margin.Top.UnwrapOr(0.0) - margin.Bottom.UnwrapOr(0.0) - baselineShim,
	}

	// Resolve width
	width := inherentSize.Width
	if !width.IsSome {
		if position == PositionAbsolute {
			if insetHorizontal.Start.IsSome && insetHorizontal.End.IsSome {
				width = Some(f32Max(gridAreaMinusItemMarginsSize.Width-insetHorizontal.Start.Value-insetHorizontal.End.Value, 0.0))
			}
		}
		if !width.IsSome && margin.Left.IsSome && margin.Right.IsSome &&
			horizontalAlignment == AlignItemsStretch && position != PositionAbsolute {
			width = Some(gridAreaMinusItemMarginsSize.Width)
		}
	}

	// Reapply aspect ratio
	sizeWithAR := SizeOptionMaybeApplyAspectRatio(
		Size[Option[float32]]{Width: width, Height: inherentSize.Height},
		aspectRatio,
	)
	width = sizeWithAR.Width
	height := sizeWithAR.Height

	// Resolve height
	if !height.IsSome {
		if position == PositionAbsolute {
			if insetVertical.Start.IsSome && insetVertical.End.IsSome {
				height = Some(f32Max(gridAreaMinusItemMarginsSize.Height-insetVertical.Start.Value-insetVertical.End.Value, 0.0))
			}
		}
		if !height.IsSome && margin.Top.IsSome && margin.Bottom.IsSome &&
			verticalAlignment == AlignItemsStretch && position != PositionAbsolute {
			height = Some(gridAreaMinusItemMarginsSize.Height)
		}
	}

	// Reapply aspect ratio after height adjustments
	sizeWithAR2 := SizeOptionMaybeApplyAspectRatio(
		Size[Option[float32]]{Width: width, Height: height},
		aspectRatio,
	)
	width = sizeWithAR2.Width
	height = sizeWithAR2.Height

	// Clamp by min/max
	clamped := SizeMaybeClamp(Size[Option[float32]]{Width: width, Height: height}, minSize, maxSize)
	width = clamped.Width
	height = clamped.Height

	// Layout node
	size := Size[Option[float32]]{Width: width, Height: height}
	if position == PositionAbsolute && (!width.IsSome || !height.IsSome) {
		measured := measureChildSizeBoth(
			tree, node, size,
			gridAreaSizeOpt,
			SizeMapTo(Size[Option[float32]]{
				Width:  Some(gridAreaMinusItemMarginsSize.Width),
				Height: Some(gridAreaMinusItemMarginsSize.Height),
			}, func(opt Option[float32]) AvailableSpace {
				return AvailableSpaceFromDefinite(opt.Value)
			}),
			SizingModeInherentSize,
			LineFalseBool,
		)
		size = Size[Option[float32]]{Width: Some(measured.Width), Height: Some(measured.Height)}
	}

	layoutOutput := performChildLayout(
		tree, node, size, gridAreaSizeOpt,
		SizeMapTo(Size[Option[float32]]{
			Width:  Some(gridAreaMinusItemMarginsSize.Width),
			Height: Some(gridAreaMinusItemMarginsSize.Height),
		}, func(opt Option[float32]) AvailableSpace {
			return AvailableSpaceFromDefinite(opt.Value)
		}),
		SizingModeInherentSize, LineFalseBool,
	)

	// Resolve final size
	finalSize := SizeMaybeClamp(
		Size[Option[float32]]{
			Width:  Some(size.Width.UnwrapOr(layoutOutput.Size.Width)),
			Height: Some(size.Height.UnwrapOr(layoutOutput.Size.Height)),
		},
		minSize, maxSize,
	)
	finalWidth := finalSize.Width.UnwrapOr(0.0)
	finalHeight := finalSize.Height.UnwrapOr(0.0)

	x, xMargin := alignItemWithinArea(
		Line[float32]{Start: gridArea.Left, End: gridArea.Right},
		justifySelf.UnwrapOr(horizontalAlignment),
		finalWidth, position,
		insetHorizontal,
		Line[Option[float32]]{Start: margin.Left, End: margin.Right},
		0.0, direction,
	)
	y, yMargin := alignItemWithinArea(
		Line[float32]{Start: gridArea.Top, End: gridArea.Bottom},
		alignSelf.UnwrapOr(verticalAlignment),
		finalHeight, position,
		insetVertical,
		Line[Option[float32]]{Start: margin.Top, End: margin.Bottom},
		baselineShim, DirectionLTR,
	)

	scrollbarSize := Size[float32]{
		Width: func() float32 {
			if overflow.Y == OverflowScroll {
				return scrollbarWidth
			}
			return 0.0
		}(),
		Height: func() float32 {
			if overflow.X == OverflowScroll {
				return scrollbarWidth
			}
			return 0.0
		}(),
	}

	resolvedMargin := Rect[float32]{
		Left:   xMargin.Start,
		Right:  xMargin.End,
		Top:    yMargin.Start,
		Bottom: yMargin.End,
	}

	tree.setUnroundedLayout(node, &Layout{
		Order:         order,
		Location:      Point[float32]{X: x, Y: y},
		Size:          Size[float32]{Width: finalWidth, Height: finalHeight},
		ContentSize:   layoutOutput.ContentSize,
		ScrollbarSize: scrollbarSize,
		Padding:       padding,
		Border:        border,
		Margin:        resolvedMargin,
	})

	// Compute content size contribution
	contribution := computeContentSizeContribution(
		Point[float32]{X: x - gridArea.Left, Y: y - gridArea.Top},
		Size[float32]{Width: finalWidth, Height: finalHeight},
		layoutOutput.ContentSize,
		overflow,
	)

	return contribution, y, finalHeight
}

// alignItemWithinArea aligns and sizes a grid item along a single axis.
func alignItemWithinArea(
	gridArea Line[float32],
	alignmentStyle AlignSelf,
	resolvedSize float32,
	position Position,
	inset Line[Option[float32]],
	margin Line[Option[float32]],
	baselineShim float32,
	direction Direction,
) (float32, Line[float32]) {
	nonAutoMargin := Line[float32]{
		Start: margin.Start.UnwrapOr(0.0) + baselineShim,
		End:   margin.End.UnwrapOr(0.0),
	}
	gridAreaSize := f32Max(gridArea.End-gridArea.Start, 0.0)
	freeSpace := f32Max(gridAreaSize-resolvedSize-nonAutoMargin.Start-nonAutoMargin.End, 0.0)

	// Expand auto margins
	autoMarginCount := uint8(0)
	if !margin.Start.IsSome {
		autoMarginCount++
	}
	if !margin.End.IsSome {
		autoMarginCount++
	}
	autoMarginSize := float32(0.0)
	if autoMarginCount > 0 {
		autoMarginSize = freeSpace / float32(autoMarginCount)
	}
	resolvedMargin := Line[float32]{
		Start: margin.Start.UnwrapOr(autoMarginSize) + baselineShim,
		End:   margin.End.UnwrapOr(autoMarginSize),
	}

	// Compute alignment-based offset
	var alignmentBasedOffset float32
	switch alignmentStyle {
	case AlignItemsStart, AlignItemsFlexStart, AlignItemsBaseline, AlignItemsStretch:
		if direction.IsRTL() {
			alignmentBasedOffset = gridAreaSize - resolvedSize - resolvedMargin.End
		} else {
			alignmentBasedOffset = resolvedMargin.Start
		}
	case AlignItemsEnd, AlignItemsFlexEnd:
		if direction.IsRTL() {
			alignmentBasedOffset = resolvedMargin.Start
		} else {
			alignmentBasedOffset = gridAreaSize - resolvedSize - resolvedMargin.End
		}
	case AlignItemsCenter:
		alignmentBasedOffset = (gridAreaSize - resolvedSize + resolvedMargin.Start - resolvedMargin.End) / 2.0
	}

	var offsetWithinArea float32
	if position == PositionAbsolute {
		if inset.Start.IsSome && inset.End.IsSome {
			if direction.IsRTL() {
				offsetWithinArea = gridAreaSize - inset.End.Value - resolvedSize - nonAutoMargin.End
			} else {
				offsetWithinArea = inset.Start.Value + nonAutoMargin.Start
			}
		} else if inset.Start.IsSome {
			offsetWithinArea = inset.Start.Value + nonAutoMargin.Start
		} else if inset.End.IsSome {
			offsetWithinArea = gridAreaSize - inset.End.Value - resolvedSize - nonAutoMargin.End
		} else {
			offsetWithinArea = alignmentBasedOffset
		}
	} else {
		offsetWithinArea = alignmentBasedOffset
	}

	start := gridArea.Start + offsetWithinArea
	if position == PositionRelative {
		var relativeInset float32
		if direction.IsRTL() {
			if inset.End.IsSome {
				relativeInset = -inset.End.Value
			} else if inset.Start.IsSome {
				relativeInset = inset.Start.Value
			}
		} else {
			if inset.Start.IsSome {
				relativeInset = inset.Start.Value
			} else if inset.End.IsSome {
				relativeInset = -inset.End.Value
			}
		}
		start += relativeInset
	}

	return start, resolvedMargin
}

// gridMaybeResolveDimOption resolves a Dimension to an Option[float32] against a context.
func gridMaybeResolveDimOption(dim LengthPercentageAuto, context float32, calc CalcResolver) Option[float32] {
	return MaybeResolveLPAOption(dim, Some(context), calc)
}
