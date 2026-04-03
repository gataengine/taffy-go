// Package taffy provides the gridItem type used during grid layout.
// This file is a port of compute/grid/types/grid_item.rs from the Taffy Rust layout engine.
package taffy

// gridItem represents a single grid item during layout.
type gridItem struct {
	// node is the id of the node that this item represents.
	node NodeId

	// sourceOrder is the order of the item in the children array.
	// Used to sort back to the original order for final positioning.
	sourceOrder uint16

	// row is the item's definite row-start and row-end (origin-zero coordinates).
	row Line[OriginZeroLine]
	// column is the item's definite column-start and column-end (origin-zero coordinates).
	column Line[OriginZeroLine]

	// isCompressibleReplaced indicates if this is a compressible replaced element.
	isCompressibleReplaced bool
	// overflow is the item's overflow style.
	overflow Point[Overflow]
	// boxSizingStyle is the item's box_sizing style.
	boxSizingStyle BoxSizing
	// size is the item's size style.
	size Size[Dimension]
	// minSize is the item's min_size style.
	minSize Size[Dimension]
	// maxSize is the item's max_size style.
	maxSize Size[Dimension]
	// aspectRatio is the item's aspect_ratio style.
	aspectRatio Option[float32]
	// paddingStyle is the item's padding style.
	paddingStyle Rect[LengthPercentage]
	// borderStyle is the item's border style.
	borderStyle Rect[LengthPercentage]
	// marginStyle is the item's margin style.
	marginStyle Rect[LengthPercentageAuto]
	// alignSelf is the item's align_self, or the parent's align_items if not set.
	alignSelf AlignSelf
	// justifySelf is the item's justify_self, or the parent's justify_items if not set.
	justifySelf AlignSelf
	// baseline is the items first baseline (horizontal).
	baseline Option[float32]
	// baselineShim is a shim for baseline alignment that acts like an extra top margin.
	baselineShim float32

	// rowIndexes is the item's row-start and row-end as indexes into the Vec<gridTrack>.
	rowIndexes Line[uint16]
	// columnIndexes is the item's column-start and column-end as indexes into the Vec<gridTrack>.
	columnIndexes Line[uint16]

	// crossesFlexibleRow indicates whether the item crosses a flexible row.
	crossesFlexibleRow bool
	// crossesFlexibleColumn indicates whether the item crosses a flexible column.
	crossesFlexibleColumn bool
	// crossesIntrinsicRow indicates whether the item crosses an intrinsic row.
	crossesIntrinsicRow bool
	// crossesIntrinsicColumn indicates whether the item crosses an intrinsic column.
	crossesIntrinsicColumn bool

	// availableSpaceCache is a cache for the available space input to intrinsic sizing computation.
	availableSpaceCache *Size[Option[float32]]
	// minContentContributionCache is a cache for the min-content size.
	minContentContributionCache Size[Option[float32]]
	// minimumContributionCache is a cache for the minimum contribution.
	minimumContributionCache Size[Option[float32]]
	// maxContentContributionCache is a cache for the max-content size.
	maxContentContributionCache Size[Option[float32]]

	// yPosition is the final y position. Used to compute baseline alignment for the container.
	yPosition float32
	// height is the final height. Used to compute baseline alignment for the container.
	height float32
}

// newGridItemWithPlacementStyleAndOrder creates a new grid item given a concrete placement in both axes.
func newGridItemWithPlacementStyleAndOrder(
	node NodeId,
	colSpan Line[OriginZeroLine],
	rowSpan Line[OriginZeroLine],
	style *Style,
	parentAlignItems AlignItems,
	parentJustifyItems AlignItems,
	sourceOrder uint16,
) gridItem {
	alignSelf := style.AlignSelf.UnwrapOr(parentAlignItems)
	justifySelf := style.JustifySelf.UnwrapOr(parentJustifyItems)

	return gridItem{
		node:                   node,
		sourceOrder:            sourceOrder,
		row:                    rowSpan,
		column:                 colSpan,
		isCompressibleReplaced: style.isCompressibleReplaced(),
		overflow:               style.overflow(),
		boxSizingStyle:         style.boxSizing(),
		size:                   style.size(),
		minSize:                style.minSize(),
		maxSize:                style.maxSize(),
		aspectRatio:            style.aspectRatio(),
		paddingStyle:           style.padding(),
		borderStyle:            style.border(),
		marginStyle:            style.margin(),
		alignSelf:              alignSelf,
		justifySelf:            justifySelf,
		baseline:               None[float32](),
		baselineShim:           0.0,
		rowIndexes:             Line[uint16]{Start: 0, End: 0},
		columnIndexes:          Line[uint16]{Start: 0, End: 0},
		crossesFlexibleRow:     false,
		crossesFlexibleColumn:  false,
		crossesIntrinsicRow:    false,
		crossesIntrinsicColumn: false,
		availableSpaceCache:    nil,
		minContentContributionCache: SizeOptionNone,
		maxContentContributionCache: SizeOptionNone,
		minimumContributionCache:    SizeOptionNone,
		yPosition: 0.0,
		height:    0.0,
	}
}

// placement returns this item's placement in the specified axis in OriginZero coordinates.
func (gi *gridItem) placement(axis AbstractAxis) Line[OriginZeroLine] {
	switch axis {
	case AbstractAxisBlock:
		return gi.row
	default:
		return gi.column
	}
}

// placementIndexes returns this item's placement in the specified axis as GridTrackVec indices.
func (gi *gridItem) placementIndexes(axis AbstractAxis) Line[uint16] {
	switch axis {
	case AbstractAxisBlock:
		return gi.rowIndexes
	default:
		return gi.columnIndexes
	}
}

// trackRangeExcludingLines returns a range which can be used as an index into the GridTrackVec
// covering all tracks and lines that this item spans, excluding the bounding lines.
func (gi *gridItem) trackRangeExcludingLines(axis AbstractAxis) (start, end int) {
	indexes := gi.placementIndexes(axis)
	return int(indexes.Start) + 1, int(indexes.End)
}

// span returns the number of tracks that this item spans in the specified axis.
func (gi *gridItem) span(axis AbstractAxis) uint16 {
	switch axis {
	case AbstractAxisBlock:
		return OriginZeroLineSpan(gi.row)
	default:
		return OriginZeroLineSpan(gi.column)
	}
}

// crossesFlexibleTrack returns the pre-computed value indicating whether the grid item
// crosses a flexible track in the specified axis.
func (gi *gridItem) crossesFlexibleTrack(axis AbstractAxis) bool {
	switch axis {
	case AbstractAxisInline:
		return gi.crossesFlexibleColumn
	default:
		return gi.crossesFlexibleRow
	}
}

// crossesIntrinsicTrack returns the pre-computed value indicating whether the grid item
// crosses an intrinsic track in the specified axis.
func (gi *gridItem) crossesIntrinsicTrack(axis AbstractAxis) bool {
	switch axis {
	case AbstractAxisInline:
		return gi.crossesIntrinsicColumn
	default:
		return gi.crossesIntrinsicRow
	}
}

// spannedTrackLimit computes the upper limit for limited min-/max-content contribution.
// It is the sum of the fixed max track sizing functions of any tracks it spans,
// and is applied if it only spans such tracks.
func (gi *gridItem) spannedTrackLimit(
	axis AbstractAxis,
	axisTracks []gridTrack,
	axisParentSize Option[float32],
	calcResolver CalcResolver,
) Option[float32] {
	start, end := gi.trackRangeExcludingLines(axis)
	spannedTracks := axisTracks[start:end]

	tracksAllFixed := true
	for i := range spannedTracks {
		if !spannedTracks[i].maxTrackSizingFunction.DefiniteLimit(axisParentSize, calcResolver).IsSome {
			tracksAllFixed = false
			break
		}
	}
	if tracksAllFixed {
		var limit float32
		for i := range spannedTracks {
			limit += spannedTracks[i].maxTrackSizingFunction.DefiniteLimit(axisParentSize, calcResolver).Value
		}
		return Some(limit)
	}
	return None[float32]()
}

// spannedFixedTrackLimit is similar to spannedTrackLimit but excludes FitContent arguments.
// Used to clamp the automatic minimum contributions of an item.
func (gi *gridItem) spannedFixedTrackLimit(
	axis AbstractAxis,
	axisTracks []gridTrack,
	axisParentSize Option[float32],
	calcResolver CalcResolver,
) Option[float32] {
	start, end := gi.trackRangeExcludingLines(axis)
	spannedTracks := axisTracks[start:end]

	tracksAllFixed := true
	for i := range spannedTracks {
		if !spannedTracks[i].maxTrackSizingFunction.DefiniteValue(axisParentSize, calcResolver).IsSome {
			tracksAllFixed = false
			break
		}
	}
	if tracksAllFixed {
		var limit float32
		for i := range spannedTracks {
			limit += spannedTracks[i].maxTrackSizingFunction.DefiniteValue(axisParentSize, calcResolver).Value
		}
		return Some(limit)
	}
	return None[float32]()
}

// knownDimensions computes the known_dimensions to be passed to child sizing functions.
// The key thing being done here is applying stretch alignment, which is necessary to
// allow percentage sizes further down the tree to resolve properly in some cases.
func (gi *gridItem) knownDimensions(
	tree layoutPartialTree,
	innerNodeSize Size[Option[float32]],
	gridAreaSize Size[Option[float32]],
) Size[Option[float32]] {
	margins := gi.marginsAxisSumsWithBaselineShims(innerNodeSize.Width, tree)

	aspectRatio := gi.aspectRatio
	padding := ResolveOrZeroRectLPSizeContext(gi.paddingStyle, gridAreaSize, func(id uint64, basis float32) float32 {
		return tree.resolveCalcValue(id, basis)
	})
	border := ResolveOrZeroRectLPSizeContext(gi.borderStyle, gridAreaSize, func(id uint64, basis float32) float32 {
		return tree.resolveCalcValue(id, basis)
	})
	paddingBorderSize := RectSumAxes(RectAddF32(padding, border))
	var boxSizingAdjustment Size[float32]
	if gi.boxSizingStyle == BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSize
	}

	inherentSize := SizeMaybeAdd(
		SizeOptionMaybeApplyAspectRatio(
			MaybeResolveSizeDimOption(gi.size, gridAreaSize, func(id uint64, basis float32) float32 {
				return tree.resolveCalcValue(id, basis)
			}),
			aspectRatio,
		),
		Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)},
	)
	minSizeResolved := SizeMaybeAdd(
		SizeOptionMaybeApplyAspectRatio(
			MaybeResolveSizeDimOption(gi.minSize, gridAreaSize, func(id uint64, basis float32) float32 {
				return tree.resolveCalcValue(id, basis)
			}),
			aspectRatio,
		),
		Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)},
	)
	maxSizeResolved := SizeMaybeAdd(
		SizeOptionMaybeApplyAspectRatio(
			MaybeResolveSizeDimOption(gi.maxSize, gridAreaSize, func(id uint64, basis float32) float32 {
				return tree.resolveCalcValue(id, basis)
			}),
			aspectRatio,
		),
		Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)},
	)

	gridAreaMinusItemMarginsSize := SizeMaybeSub(gridAreaSize, Size[Option[float32]]{
		Width:  Some(margins.Width),
		Height: Some(margins.Height),
	})

	// Compute width
	width := inherentSize.Width
	if !width.IsSome {
		// Apply width based on stretch alignment
		if !gi.marginStyle.Left.IsAuto() && !gi.marginStyle.Right.IsAuto() && gi.justifySelf == AlignItemsStretch {
			width = gridAreaMinusItemMarginsSize.Width
		}
	}
	// Reapply aspect ratio after stretch width adjustments
	stretchedSize := SizeOptionMaybeApplyAspectRatio(
		Size[Option[float32]]{Width: width, Height: inherentSize.Height},
		aspectRatio,
	)
	width = stretchedSize.Width
	height := stretchedSize.Height

	if !height.IsSome {
		// Apply height based on stretch alignment
		if !gi.marginStyle.Top.IsAuto() && !gi.marginStyle.Bottom.IsAuto() && gi.alignSelf == AlignItemsStretch {
			height = gridAreaMinusItemMarginsSize.Height
		}
	}
	// Reapply aspect ratio after stretch height adjustments
	stretchedSize2 := SizeOptionMaybeApplyAspectRatio(
		Size[Option[float32]]{Width: width, Height: height},
		aspectRatio,
	)
	width = stretchedSize2.Width
	height = stretchedSize2.Height

	// Clamp size by min and max width/height
	clamped := SizeMaybeClamp(Size[Option[float32]]{Width: width, Height: height}, minSizeResolved, maxSizeResolved)
	return clamped
}

// availableSpace computes the available_space to be passed to child sizing functions.
// These are estimates based on either the max track sizing function or the provisional
// base size in the opposite axis to the one currently being sized.
func (gi *gridItem) availableSpace(
	axis AbstractAxis,
	otherAxisTracks []gridTrack,
	otherAxisAvailableSpace Option[float32],
	getTrackSizeEstimate func(track *gridTrack, availableSpace Option[float32]) Option[float32],
) Size[Option[float32]] {
	otherStart, otherEnd := gi.trackRangeExcludingLines(axis.Other())
	tracks := otherAxisTracks[otherStart:otherEnd]

	var itemOtherAxisSize Option[float32]
	hasAll := true
	var sum float32
	for i := range tracks {
		est := getTrackSizeEstimate(&tracks[i], otherAxisAvailableSpace)
		if !est.IsSome {
			hasAll = false
			break
		}
		sum += est.Value + tracks[i].contentAlignmentAdjustment
	}
	if hasAll {
		itemOtherAxisSize = Some(sum)
	}

	result := SizeOptionNone
	result.SetAbstract(axis.Other(), itemOtherAxisSize)
	return result
}

// availableSpaceCached retrieves the available_space from the cache or computes it.
func (gi *gridItem) availableSpaceCached(
	axis AbstractAxis,
	otherAxisTracks []gridTrack,
	otherAxisAvailableSpace Option[float32],
	getTrackSizeEstimate func(track *gridTrack, availableSpace Option[float32]) Option[float32],
) Size[Option[float32]] {
	if gi.availableSpaceCache != nil {
		return *gi.availableSpaceCache
	}
	availableSpaces := gi.availableSpace(axis, otherAxisTracks, otherAxisAvailableSpace, getTrackSizeEstimate)
	gi.availableSpaceCache = &availableSpaces
	return availableSpaces
}

// marginsAxisSumsWithBaselineShims computes the item's resolved margins for size contributions.
// Horizontal percentage margins always resolve to zero if the container size is indefinite.
func (gi *gridItem) marginsAxisSumsWithBaselineShims(
	innerNodeWidth Option[float32],
	tree layoutPartialTree,
) Size[float32] {
	calc := func(id uint64, basis float32) float32 {
		return tree.resolveCalcValue(id, basis)
	}
	r := Rect[float32]{
		Left:   ResolveOrZeroLPA(gi.marginStyle.Left, Some[float32](0.0), calc),
		Right:  ResolveOrZeroLPA(gi.marginStyle.Right, Some[float32](0.0), calc),
		Top:    ResolveOrZeroLPA(gi.marginStyle.Top, innerNodeWidth, calc) + gi.baselineShim,
		Bottom: ResolveOrZeroLPA(gi.marginStyle.Bottom, innerNodeWidth, calc),
	}
	return RectSumAxes(r)
}

// minContentContribution computes the item's min content contribution from the provided parameters.
func (gi *gridItem) minContentContribution(
	axis AbstractAxis,
	tree layoutPartialTree,
	availSpace Size[Option[float32]],
	innerNodeSize Size[Option[float32]],
) float32 {
	knownDims := gi.knownDimensions(tree, innerNodeSize, availSpace)
	asAvailableSpace := SizeMapTo(availSpace, func(opt Option[float32]) AvailableSpace {
		if opt.IsSome {
			return AvailableSpaceFromDefinite(opt.Value)
		}
		return AvailableSpaceMinContentValue
	})
	return measureChildSize(
		tree,
		gi.node,
		knownDims,
		innerNodeSize,
		asAvailableSpace,
		SizingModeInherentSize,
		axis.AsAbsNaive(),
		LineFalseBool,
	)
}

// minContentContributionCached retrieves the min content contribution from the cache or computes it.
func (gi *gridItem) minContentContributionCached(
	axis AbstractAxis,
	tree layoutPartialTree,
	availSpace Size[Option[float32]],
	innerNodeSize Size[Option[float32]],
) float32 {
	cached := gi.minContentContributionCache.GetAbstract(axis)
	if cached.IsSome {
		return cached.Value
	}
	size := gi.minContentContribution(axis, tree, availSpace, innerNodeSize)
	gi.minContentContributionCache.SetAbstract(axis, Some(size))
	return size
}

// maxContentContribution computes the item's max content contribution from the provided parameters.
func (gi *gridItem) maxContentContribution(
	axis AbstractAxis,
	tree layoutPartialTree,
	availSpace Size[Option[float32]],
	innerNodeSize Size[Option[float32]],
) float32 {
	knownDims := gi.knownDimensions(tree, innerNodeSize, availSpace)
	asAvailableSpace := SizeMapTo(availSpace, func(opt Option[float32]) AvailableSpace {
		if opt.IsSome {
			return AvailableSpaceFromDefinite(opt.Value)
		}
		return AvailableSpaceMaxContentValue
	})
	return measureChildSize(
		tree,
		gi.node,
		knownDims,
		innerNodeSize,
		asAvailableSpace,
		SizingModeInherentSize,
		axis.AsAbsNaive(),
		LineFalseBool,
	)
}

// maxContentContributionCached retrieves the max content contribution from the cache or computes it.
func (gi *gridItem) maxContentContributionCached(
	axis AbstractAxis,
	tree layoutPartialTree,
	availSpace Size[Option[float32]],
	innerNodeSize Size[Option[float32]],
) float32 {
	cached := gi.maxContentContributionCache.GetAbstract(axis)
	if cached.IsSome {
		return cached.Value
	}
	size := gi.maxContentContribution(axis, tree, availSpace, innerNodeSize)
	gi.maxContentContributionCache.SetAbstract(axis, Some(size))
	return size
}

// minimumContribution computes the minimum contribution of an item.
// The minimum contribution is the smallest outer size it can have.
// See: https://www.w3.org/TR/css-grid-1/#min-size-auto
func (gi *gridItem) minimumContribution(
	tree layoutPartialTree,
	axis AbstractAxis,
	axisTracks []gridTrack,
	knownDims Size[Option[float32]],
	innerNodeSize Size[Option[float32]],
) float32 {
	calc := func(id uint64, basis float32) float32 {
		return tree.resolveCalcValue(id, basis)
	}
	padding := ResolveOrZeroRectLPSizeContext(gi.paddingStyle, innerNodeSize, calc)
	border := ResolveOrZeroRectLPSizeContext(gi.borderStyle, innerNodeSize, calc)
	paddingBorderSize := RectSumAxes(RectAddF32(padding, border))
	var boxSizingAdjustment Size[float32]
	if gi.boxSizingStyle == BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSize
	}

	resolvedSize := MaybeResolveSizeDimOption(gi.size, innerNodeSize, calc)
	resolvedSize = SizeOptionMaybeApplyAspectRatio(resolvedSize, gi.aspectRatio)
	resolvedSize = SizeMaybeAdd(resolvedSize, Size[Option[float32]]{
		Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height),
	})
	sizeVal := resolvedSize.GetAbstract(axis)

	if !sizeVal.IsSome {
		resolvedMinSize := MaybeResolveSizeDimOption(gi.minSize, innerNodeSize, calc)
		resolvedMinSize = SizeOptionMaybeApplyAspectRatio(resolvedMinSize, gi.aspectRatio)
		resolvedMinSize = SizeMaybeAdd(resolvedMinSize, Size[Option[float32]]{
			Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height),
		})
		minSizeVal := resolvedMinSize.GetAbstract(axis)
		if minSizeVal.IsSome {
			sizeVal = minSizeVal
		}
	}

	if !sizeVal.IsSome {
		overflowVal := gi.overflow.GetAbstract(axis)
		autoMinSize := overflowVal.MaybeIntoAutomaticMinSize()
		if autoMinSize.IsSome {
			sizeVal = autoMinSize
		}
	}

	if !sizeVal.IsSome {
		// Automatic minimum size
		start, end := gi.trackRangeExcludingLines(axis)
		itemAxisTracks := axisTracks[start:end]

		// it spans at least one track in that axis whose min track sizing function is auto
		spansAutoMinTrack := false
		for i := range axisTracks {
			if axisTracks[i].minTrackSizingFunction.IsAuto() {
				spansAutoMinTrack = true
				break
			}
		}

		// if it spans more than one track, none of those tracks are flexible
		onlySpanOneTrack := len(itemAxisTracks) == 1
		spansAFlexibleTrack := false
		for i := range axisTracks {
			if axisTracks[i].maxTrackSizingFunction.IsFr() {
				spansAFlexibleTrack = true
				break
			}
		}

		useContentBasedMinimum := spansAutoMinTrack && (onlySpanOneTrack || !spansAFlexibleTrack)

		if useContentBasedMinimum {
			minimumContrib := gi.minContentContributionCached(axis, tree, knownDims, innerNodeSize)

			// If the item is a compressible replaced element, cap by preferred/max size
			if gi.isCompressibleReplaced {
				sizeOpt := MaybeResolveDimOption(gi.size.GetAbstract(axis), Some[float32](0.0), calc)
				maxSizeOpt := MaybeResolveDimOption(gi.maxSize.GetAbstract(axis), Some[float32](0.0), calc)
				minimumContrib = MaybeMinF32(minimumContrib, sizeOpt)
				minimumContrib = MaybeMinF32(minimumContrib, maxSizeOpt)
			}

			sizeVal = Some(minimumContrib)
		} else {
			sizeVal = Some[float32](0.0)
		}
	}

	// Clamp by spanned fixed track limit
	limit := gi.spannedFixedTrackLimit(axis, axisTracks, innerNodeSize.GetAbstract(axis), calc)
	return MaybeMinF32(sizeVal.Value, limit)
}

// minimumContributionCached retrieves the minimum contribution from the cache or computes it.
func (gi *gridItem) minimumContributionCached(
	tree layoutPartialTree,
	axis AbstractAxis,
	axisTracks []gridTrack,
	knownDims Size[Option[float32]],
	innerNodeSize Size[Option[float32]],
) float32 {
	cached := gi.minimumContributionCache.GetAbstract(axis)
	if cached.IsSome {
		return cached.Value
	}
	size := gi.minimumContribution(tree, axis, axisTracks, knownDims, innerNodeSize)
	gi.minimumContributionCache.SetAbstract(axis, Some(size))
	return size
}
