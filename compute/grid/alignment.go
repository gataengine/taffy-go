// Package grid implements alignment of grid tracks and final positioning of grid items.
// This file is a port of compute/grid/alignment.rs from the Taffy Rust layout engine.
package grid

import (
	"taffy-go/lib"
	"taffy-go/compute"
	"taffy-go/geometry"
	"taffy-go/style"
	"taffy-go/tree"
)

// alignGridTracks aligns the grid tracks within the grid container according to
// align-content (rows) or justify-content (columns).
func alignGridTracks(
	gridContainerContentBoxSize float32,
	padding geometry.Line[float32],
	border geometry.Line[float32],
	tracks []gridTrack,
	trackAlignmentStyle style.AlignContent,
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
	trackAlignment := compute.ApplyAlignmentFallback(freeSpace, numTracks, trackAlignmentStyle, isSafe)
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
			offset = compute.ComputeAlignmentOffset(freeSpace, numTracks, gap, trackAlignment, layoutIsReversed, isFirst)
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
	t tree.LayoutGridContainer,
	node tree.NodeId,
	order uint32,
	gridArea geometry.Rect[float32],
	containerAlignmentStyles inBothAbsAxis[lib.Option[style.AlignItems]],
	baselineShim float32,
	direction style.Direction,
) (geometry.Size[float32], float32, float32) {
	gridAreaSize := geometry.Size[float32]{
		Width:  gridArea.Right - gridArea.Left,
		Height: gridArea.Bottom - gridArea.Top,
	}

	coreStyle := t.GetCoreContainerStyle(node)
	calc := func(id uint64, basis float32) float32 {
		return t.ResolveCalcValue(id, basis)
	}

	overflow := coreStyle.GetOverflow()
	scrollbarWidth := coreStyle.GetScrollbarWidth()
	aspectRatio := coreStyle.GetAspectRatio()
	childGridStyle := t.GetGridChildStyle(node)
	justifySelf := childGridStyle.GetJustifySelf()
	alignSelf := childGridStyle.GetAlignSelf()

	position := coreStyle.GetPosition()
	insetHorizontal := geometry.Line[lib.Option[float32]]{
		Start: gridMaybeResolveDimOption(coreStyle.GetInset().Left, gridAreaSize.Width, calc),
		End:   gridMaybeResolveDimOption(coreStyle.GetInset().Right, gridAreaSize.Width, calc),
	}
	insetVertical := geometry.Line[lib.Option[float32]]{
		Start: gridMaybeResolveDimOption(coreStyle.GetInset().Top, gridAreaSize.Height, calc),
		End:   gridMaybeResolveDimOption(coreStyle.GetInset().Bottom, gridAreaSize.Height, calc),
	}
	padding := style.ResolveRectLPWidth(coreStyle.GetPadding(), lib.Some(gridAreaSize.Width), calc)
	border := style.ResolveRectLPWidth(coreStyle.GetBorder(), lib.Some(gridAreaSize.Width), calc)
	paddingBorderSize := geometry.RectSumAxes(geometry.RectAddF32(padding, border))

	boxSizingAdjustment := geometry.SizeZeroF32
	if coreStyle.GetBoxSizing() == style.BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSize
	}

	gridAreaSizeOpt := geometry.Size[lib.Option[float32]]{Width: lib.Some(gridAreaSize.Width), Height: lib.Some(gridAreaSize.Height)}
	inherentSize := geometry.SizeMaybeAdd(
		geometry.SizeOptionMaybeApplyAspectRatio(
			style.MaybeResolveSizeDimOption(coreStyle.GetSize(), gridAreaSizeOpt, calc),
			aspectRatio,
		),
		geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)},
	)
	minSize := geometry.SizeMaybeMax(
		geometry.SizeOptionOr(
			geometry.SizeMaybeAdd(
				geometry.SizeOptionMaybeApplyAspectRatio(
					style.MaybeResolveSizeDimOption(coreStyle.GetMinSize(), gridAreaSizeOpt, calc),
					aspectRatio,
				),
				geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)},
			),
			geometry.Size[lib.Option[float32]]{Width: lib.Some(paddingBorderSize.Width), Height: lib.Some(paddingBorderSize.Height)},
		),
		geometry.Size[lib.Option[float32]]{Width: lib.Some(paddingBorderSize.Width), Height: lib.Some(paddingBorderSize.Height)},
	)
	maxSize := geometry.SizeMaybeAdd(
		geometry.SizeOptionMaybeApplyAspectRatio(
			style.MaybeResolveSizeDimOption(coreStyle.GetMaxSize(), gridAreaSizeOpt, calc),
			aspectRatio,
		),
		geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)},
	)

	// Resolve alignment styles
	horizontalAlignment := justifySelf.UnwrapOr(containerAlignmentStyles.horizontal.UnwrapOr(func() style.AlignSelf {
		if inherentSize.Width.IsSome {
			return style.AlignItemsStart
		}
		return style.AlignItemsStretch
	}()))
	verticalAlignment := alignSelf.UnwrapOr(containerAlignmentStyles.vertical.UnwrapOr(func() style.AlignSelf {
		if inherentSize.Height.IsSome || aspectRatio.IsSome {
			return style.AlignItemsStart
		}
		return style.AlignItemsStretch
	}()))

	// Resolve margins (both horizontal and vertical against grid area WIDTH per CSS spec)
	margin := geometry.Rect[lib.Option[float32]]{
		Left:   style.MaybeResolveLPAOption(coreStyle.GetMargin().Left, lib.Some(gridAreaSize.Width), calc),
		Right:  style.MaybeResolveLPAOption(coreStyle.GetMargin().Right, lib.Some(gridAreaSize.Width), calc),
		Top:    style.MaybeResolveLPAOption(coreStyle.GetMargin().Top, lib.Some(gridAreaSize.Width), calc),
		Bottom: style.MaybeResolveLPAOption(coreStyle.GetMargin().Bottom, lib.Some(gridAreaSize.Width), calc),
	}

	gridAreaMinusItemMarginsSize := geometry.Size[float32]{
		Width:  gridAreaSize.Width - margin.Left.UnwrapOr(0.0) - margin.Right.UnwrapOr(0.0),
		Height: gridAreaSize.Height - margin.Top.UnwrapOr(0.0) - margin.Bottom.UnwrapOr(0.0) - baselineShim,
	}

	// Resolve width
	width := inherentSize.Width
	if !width.IsSome {
		if position == style.PositionAbsolute {
			if insetHorizontal.Start.IsSome && insetHorizontal.End.IsSome {
				width = lib.Some(geometry.F32Max(gridAreaMinusItemMarginsSize.Width-insetHorizontal.Start.Value-insetHorizontal.End.Value, 0.0))
			}
		}
		if !width.IsSome && margin.Left.IsSome && margin.Right.IsSome &&
			horizontalAlignment == style.AlignItemsStretch && position != style.PositionAbsolute {
			width = lib.Some(gridAreaMinusItemMarginsSize.Width)
		}
	}

	// Reapply aspect ratio
	sizeWithAR := geometry.SizeOptionMaybeApplyAspectRatio(
		geometry.Size[lib.Option[float32]]{Width: width, Height: inherentSize.Height},
		aspectRatio,
	)
	width = sizeWithAR.Width
	height := sizeWithAR.Height

	// Resolve height
	if !height.IsSome {
		if position == style.PositionAbsolute {
			if insetVertical.Start.IsSome && insetVertical.End.IsSome {
				height = lib.Some(geometry.F32Max(gridAreaMinusItemMarginsSize.Height-insetVertical.Start.Value-insetVertical.End.Value, 0.0))
			}
		}
		if !height.IsSome && margin.Top.IsSome && margin.Bottom.IsSome &&
			verticalAlignment == style.AlignItemsStretch && position != style.PositionAbsolute {
			height = lib.Some(gridAreaMinusItemMarginsSize.Height)
		}
	}

	// Reapply aspect ratio after height adjustments
	sizeWithAR2 := geometry.SizeOptionMaybeApplyAspectRatio(
		geometry.Size[lib.Option[float32]]{Width: width, Height: height},
		aspectRatio,
	)
	width = sizeWithAR2.Width
	height = sizeWithAR2.Height

	// Clamp by min/max
	clamped := geometry.SizeMaybeClamp(geometry.Size[lib.Option[float32]]{Width: width, Height: height}, minSize, maxSize)
	width = clamped.Width
	height = clamped.Height

	// Layout node
	size := geometry.Size[lib.Option[float32]]{Width: width, Height: height}
	if position == style.PositionAbsolute && (!width.IsSome || !height.IsSome) {
		measured := measureChildSizeBoth(
			t, node, size,
			gridAreaSizeOpt,
			geometry.SizeMapTo(geometry.Size[lib.Option[float32]]{
				Width:  lib.Some(gridAreaMinusItemMarginsSize.Width),
				Height: lib.Some(gridAreaMinusItemMarginsSize.Height),
			}, func(opt lib.Option[float32]) style.AvailableSpace {
				return style.AvailableSpaceFromDefinite(opt.Value)
			}),
			tree.SizingModeInherentSize,
			geometry.LineFalseBool,
		)
		size = geometry.Size[lib.Option[float32]]{Width: lib.Some(measured.Width), Height: lib.Some(measured.Height)}
	}

	layoutOutput := tree.PerformChildLayout(
		t, node, size, gridAreaSizeOpt,
		geometry.SizeMapTo(geometry.Size[lib.Option[float32]]{
			Width:  lib.Some(gridAreaMinusItemMarginsSize.Width),
			Height: lib.Some(gridAreaMinusItemMarginsSize.Height),
		}, func(opt lib.Option[float32]) style.AvailableSpace {
			return style.AvailableSpaceFromDefinite(opt.Value)
		}),
		tree.SizingModeInherentSize, geometry.LineFalseBool,
	)

	// Resolve final size
	finalSize := geometry.SizeMaybeClamp(
		geometry.Size[lib.Option[float32]]{
			Width:  lib.Some(size.Width.UnwrapOr(layoutOutput.Size.Width)),
			Height: lib.Some(size.Height.UnwrapOr(layoutOutput.Size.Height)),
		},
		minSize, maxSize,
	)
	finalWidth := finalSize.Width.UnwrapOr(0.0)
	finalHeight := finalSize.Height.UnwrapOr(0.0)

	x, xMargin := alignItemWithinArea(
		geometry.Line[float32]{Start: gridArea.Left, End: gridArea.Right},
		justifySelf.UnwrapOr(horizontalAlignment),
		finalWidth, position,
		insetHorizontal,
		geometry.Line[lib.Option[float32]]{Start: margin.Left, End: margin.Right},
		0.0, direction,
	)
	y, yMargin := alignItemWithinArea(
		geometry.Line[float32]{Start: gridArea.Top, End: gridArea.Bottom},
		alignSelf.UnwrapOr(verticalAlignment),
		finalHeight, position,
		insetVertical,
		geometry.Line[lib.Option[float32]]{Start: margin.Top, End: margin.Bottom},
		baselineShim, style.DirectionLTR,
	)

	scrollbarSize := geometry.Size[float32]{
		Width: func() float32 {
			if overflow.Y == style.OverflowScroll {
				return scrollbarWidth
			}
			return 0.0
		}(),
		Height: func() float32 {
			if overflow.X == style.OverflowScroll {
				return scrollbarWidth
			}
			return 0.0
		}(),
	}

	resolvedMargin := geometry.Rect[float32]{
		Left:   xMargin.Start,
		Right:  xMargin.End,
		Top:    yMargin.Start,
		Bottom: yMargin.End,
	}

	t.SetUnroundedLayout(node, &tree.Layout{
		Order:         order,
		Location:      geometry.Point[float32]{X: x, Y: y},
		Size:          geometry.Size[float32]{Width: finalWidth, Height: finalHeight},
		ContentSize:   layoutOutput.ContentSize,
		ScrollbarSize: scrollbarSize,
		Padding:       padding,
		Border:        border,
		Margin:        resolvedMargin,
	})

	// Compute content size contribution
	contribution := computeContentSizeContribution(
		geometry.Point[float32]{X: x - gridArea.Left, Y: y - gridArea.Top},
		geometry.Size[float32]{Width: finalWidth, Height: finalHeight},
		layoutOutput.ContentSize,
		overflow,
	)

	return contribution, y, finalHeight
}

// alignItemWithinArea aligns and sizes a grid item along a single axis.
func alignItemWithinArea(
	gridArea geometry.Line[float32],
	alignmentStyle style.AlignSelf,
	resolvedSize float32,
	position style.Position,
	inset geometry.Line[lib.Option[float32]],
	margin geometry.Line[lib.Option[float32]],
	baselineShim float32,
	direction style.Direction,
) (float32, geometry.Line[float32]) {
	nonAutoMargin := geometry.Line[float32]{
		Start: margin.Start.UnwrapOr(0.0) + baselineShim,
		End:   margin.End.UnwrapOr(0.0),
	}
	gridAreaSize := geometry.F32Max(gridArea.End-gridArea.Start, 0.0)
	freeSpace := geometry.F32Max(gridAreaSize-resolvedSize-nonAutoMargin.Start-nonAutoMargin.End, 0.0)

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
	resolvedMargin := geometry.Line[float32]{
		Start: margin.Start.UnwrapOr(autoMarginSize) + baselineShim,
		End:   margin.End.UnwrapOr(autoMarginSize),
	}

	// Compute alignment-based offset
	var alignmentBasedOffset float32
	switch alignmentStyle {
	case style.AlignItemsStart, style.AlignItemsFlexStart, style.AlignItemsBaseline, style.AlignItemsStretch:
		if direction.IsRTL() {
			alignmentBasedOffset = gridAreaSize - resolvedSize - resolvedMargin.End
		} else {
			alignmentBasedOffset = resolvedMargin.Start
		}
	case style.AlignItemsEnd, style.AlignItemsFlexEnd:
		if direction.IsRTL() {
			alignmentBasedOffset = resolvedMargin.Start
		} else {
			alignmentBasedOffset = gridAreaSize - resolvedSize - resolvedMargin.End
		}
	case style.AlignItemsCenter:
		alignmentBasedOffset = (gridAreaSize - resolvedSize + resolvedMargin.Start - resolvedMargin.End) / 2.0
	}

	var offsetWithinArea float32
	if position == style.PositionAbsolute {
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
	if position == style.PositionRelative {
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
func gridMaybeResolveDimOption(dim style.LengthPercentageAuto, context float32, calc style.CalcResolver) lib.Option[float32] {
	return style.MaybeResolveLPAOption(dim, lib.Some(context), calc)
}

// computeContentSizeContribution computes the content size contribution of a positioned item.
func computeContentSizeContribution(
	location geometry.Point[float32],
	size geometry.Size[float32],
	contentSize geometry.Size[float32],
	overflow geometry.Point[style.Overflow],
) geometry.Size[float32] {
	sizeContrib := geometry.Size[float32]{
		Width: func() float32 {
			if overflow.X == style.OverflowVisible {
				return geometry.F32Max(size.Width, contentSize.Width)
			}
			return size.Width
		}(),
		Height: func() float32 {
			if overflow.Y == style.OverflowVisible {
				return geometry.F32Max(size.Height, contentSize.Height)
			}
			return size.Height
		}(),
	}
	if sizeContrib.Width > 0 && sizeContrib.Height > 0 {
		maxX := geometry.F32Max(location.X+sizeContrib.Width, 0.0)
		minX := geometry.F32Min(location.X, 0.0)
		maxY := geometry.F32Max(location.Y+sizeContrib.Height, 0.0)
		minY := geometry.F32Min(location.Y, 0.0)
		return geometry.Size[float32]{Width: maxX - minX, Height: maxY - minY}
	}
	return geometry.SizeZeroF32
}

// measureChildSizeBoth computes both dimensions of a child (RunMode::ComputeSize + RequestedAxis::Both).
func measureChildSizeBoth(
	t tree.LayoutPartialTree,
	nodeId tree.NodeId,
	knownDimensions geometry.Size[lib.Option[float32]],
	parentSize geometry.Size[lib.Option[float32]],
	availableSpace geometry.Size[style.AvailableSpace],
	sizingMode tree.SizingMode,
	verticalMarginsAreCollapsible geometry.Line[bool],
) geometry.Size[float32] {
	return t.ComputeChildLayout(nodeId, tree.LayoutInput{
		KnownDimensions:               knownDimensions,
		ParentSize:                    parentSize,
		AvailableSpace:                availableSpace,
		SizingMode:                    sizingMode,
		Axis:                          tree.RequestedAxisBoth,
		RunMode:                       tree.RunModeComputeSize,
		VerticalMarginsAreCollapsible: verticalMarginsAreCollapsible,
	}).Size
}
