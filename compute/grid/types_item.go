// Package grid provides the gridItem type used during grid layout.
// This file is a port of compute/grid/types/grid_item.rs from the Taffy Rust layout engine.
package grid

import (
	"github.com/gataengine/taffy-go/geometry"
	"github.com/gataengine/taffy-go/lib"
	"github.com/gataengine/taffy-go/style"
	"github.com/gataengine/taffy-go/tree"
)

// gridItem represents a single grid item during layout.
type gridItem struct {
	// node is the id of the node that this item represents.
	node tree.NodeId

	// sourceOrder is the order of the item in the children array.
	// Used to sort back to the original order for final positioning.
	sourceOrder uint16

	// row is the item's definite row-start and row-end (origin-zero coordinates).
	row geometry.Line[style.OriginZeroLine]
	// column is the item's definite column-start and column-end (origin-zero coordinates).
	column geometry.Line[style.OriginZeroLine]

	// isCompressibleReplaced indicates if this is a compressible replaced element.
	isCompressibleReplaced bool
	// overflow is the item's overflow style.
	overflow geometry.Point[style.Overflow]
	// boxSizingStyle is the item's box_sizing style.
	boxSizingStyle style.BoxSizing
	// size is the item's size style.
	size geometry.Size[style.Dimension]
	// minSize is the item's min_size style.
	minSize geometry.Size[style.Dimension]
	// maxSize is the item's max_size style.
	maxSize geometry.Size[style.Dimension]
	// aspectRatio is the item's aspect_ratio style.
	aspectRatio lib.Option[float32]
	// paddingStyle is the item's padding style.
	paddingStyle geometry.Rect[style.LengthPercentage]
	// borderStyle is the item's border style.
	borderStyle geometry.Rect[style.LengthPercentage]
	// marginStyle is the item's margin style.
	marginStyle geometry.Rect[style.LengthPercentageAuto]
	// alignSelf is the item's align_self, or the parent's align_items if not set.
	alignSelf style.AlignSelf
	// justifySelf is the item's justify_self, or the parent's justify_items if not set.
	justifySelf style.AlignSelf
	// baseline is the items first baseline (horizontal).
	baseline lib.Option[float32]
	// baselineShim is a shim for baseline alignment that acts like an extra top margin.
	baselineShim float32

	// rowIndexes is the item's row-start and row-end as indexes into the Vec<gridTrack>.
	rowIndexes geometry.Line[uint16]
	// columnIndexes is the item's column-start and column-end as indexes into the Vec<gridTrack>.
	columnIndexes geometry.Line[uint16]

	// crossesFlexibleRow indicates whether the item crosses a flexible row.
	crossesFlexibleRow bool
	// crossesFlexibleColumn indicates whether the item crosses a flexible column.
	crossesFlexibleColumn bool
	// crossesIntrinsicRow indicates whether the item crosses an intrinsic row.
	crossesIntrinsicRow bool
	// crossesIntrinsicColumn indicates whether the item crosses an intrinsic column.
	crossesIntrinsicColumn bool

	// availableSpaceCache is a cache for the available space input to intrinsic sizing computation.
	availableSpaceCache lib.Option[geometry.Size[lib.Option[float32]]]
	// minContentContributionCache is a cache for the min-content size.
	minContentContributionCache geometry.Size[lib.Option[float32]]
	// minimumContributionCache is a cache for the minimum contribution.
	minimumContributionCache geometry.Size[lib.Option[float32]]
	// maxContentContributionCache is a cache for the max-content size.
	maxContentContributionCache geometry.Size[lib.Option[float32]]

	// yPosition is the final y position. Used to compute baseline alignment for the container.
	yPosition float32
	// height is the final height. Used to compute baseline alignment for the container.
	height float32
}

// newGridItemWithPlacementStyleAndOrder creates a new grid item given a concrete placement in both axes.
func newGridItemWithPlacementStyleAndOrder(
	node tree.NodeId,
	colSpan geometry.Line[style.OriginZeroLine],
	rowSpan geometry.Line[style.OriginZeroLine],
	childStyle style.GridItemStyle,
	coreStyle style.CoreStyle,
	parentAlignItems style.AlignItems,
	parentJustifyItems style.AlignItems,
	sourceOrder uint16,
) gridItem {
	alignSelf := childStyle.GetAlignSelf().UnwrapOr(parentAlignItems)
	justifySelf := childStyle.GetJustifySelf().UnwrapOr(parentJustifyItems)

	return gridItem{
		node:                        node,
		sourceOrder:                 sourceOrder,
		row:                         rowSpan,
		column:                      colSpan,
		isCompressibleReplaced:      coreStyle.GetIsCompressibleReplaced(),
		overflow:                    coreStyle.GetOverflow(),
		boxSizingStyle:              coreStyle.GetBoxSizing(),
		size:                        coreStyle.GetSize(),
		minSize:                     coreStyle.GetMinSize(),
		maxSize:                     coreStyle.GetMaxSize(),
		aspectRatio:                 coreStyle.GetAspectRatio(),
		paddingStyle:                coreStyle.GetPadding(),
		borderStyle:                 coreStyle.GetBorder(),
		marginStyle:                 coreStyle.GetMargin(),
		alignSelf:                   alignSelf,
		justifySelf:                 justifySelf,
		baseline:                    lib.None[float32](),
		baselineShim:                0.0,
		rowIndexes:                  geometry.Line[uint16]{Start: 0, End: 0},
		columnIndexes:               geometry.Line[uint16]{Start: 0, End: 0},
		crossesFlexibleRow:          false,
		crossesFlexibleColumn:       false,
		crossesIntrinsicRow:         false,
		crossesIntrinsicColumn:      false,
		availableSpaceCache:         lib.Option[geometry.Size[lib.Option[float32]]]{},
		minContentContributionCache: geometry.SizeOptionNone,
		maxContentContributionCache: geometry.SizeOptionNone,
		minimumContributionCache:    geometry.SizeOptionNone,
		yPosition:                   0.0,
		height:                      0.0,
	}
}

// placement returns this item's placement in the specified axis in OriginZero coordinates.
func (gi *gridItem) placement(axis geometry.AbstractAxis) geometry.Line[style.OriginZeroLine] {
	switch axis {
	case geometry.AbstractAxisBlock:
		return gi.row
	default:
		return gi.column
	}
}

// placementIndexes returns this item's placement in the specified axis as GridTrackVec indices.
func (gi *gridItem) placementIndexes(axis geometry.AbstractAxis) geometry.Line[uint16] {
	switch axis {
	case geometry.AbstractAxisBlock:
		return gi.rowIndexes
	default:
		return gi.columnIndexes
	}
}

// trackRangeExcludingLines returns a range which can be used as an index into the GridTrackVec
// covering all tracks and lines that this item spans, excluding the bounding lines.
func (gi *gridItem) trackRangeExcludingLines(axis geometry.AbstractAxis) (start, end int) {
	indexes := gi.placementIndexes(axis)
	return int(indexes.Start) + 1, int(indexes.End)
}

// span returns the number of tracks that this item spans in the specified axis.
func (gi *gridItem) span(axis geometry.AbstractAxis) uint16 {
	switch axis {
	case geometry.AbstractAxisBlock:
		return style.OriginZeroLineSpan(gi.row)
	default:
		return style.OriginZeroLineSpan(gi.column)
	}
}

// crossesFlexibleTrack returns the pre-computed value indicating whether the grid item
// crosses a flexible track in the specified axis.
func (gi *gridItem) crossesFlexibleTrack(axis geometry.AbstractAxis) bool {
	switch axis {
	case geometry.AbstractAxisInline:
		return gi.crossesFlexibleColumn
	default:
		return gi.crossesFlexibleRow
	}
}

// crossesIntrinsicTrack returns the pre-computed value indicating whether the grid item
// crosses an intrinsic track in the specified axis.
func (gi *gridItem) crossesIntrinsicTrack(axis geometry.AbstractAxis) bool {
	switch axis {
	case geometry.AbstractAxisInline:
		return gi.crossesIntrinsicColumn
	default:
		return gi.crossesIntrinsicRow
	}
}

// spannedTrackLimit computes the upper limit for limited min-/max-content contribution.
// It is the sum of the fixed max track sizing functions of any tracks it spans,
// and is applied if it only spans such tracks.
func (gi *gridItem) spannedTrackLimit(
	axis geometry.AbstractAxis,
	axisTracks []gridTrack,
	axisParentSize lib.Option[float32],
	calcResolver style.CalcResolver,
) lib.Option[float32] {
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
		return lib.Some(limit)
	}
	return lib.None[float32]()
}

// spannedFixedTrackLimit is similar to spannedTrackLimit but excludes FitContent arguments.
// Used to clamp the automatic minimum contributions of an item.
func (gi *gridItem) spannedFixedTrackLimit(
	axis geometry.AbstractAxis,
	axisTracks []gridTrack,
	axisParentSize lib.Option[float32],
	calcResolver style.CalcResolver,
) lib.Option[float32] {
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
		return lib.Some(limit)
	}
	return lib.None[float32]()
}

// knownDimensions computes the known_dimensions to be passed to child sizing functions.
// The key thing being done here is applying stretch alignment, which is necessary to
// allow percentage sizes further down the tree to resolve properly in some cases.
func (gi *gridItem) knownDimensions(
	t tree.LayoutPartialTree,
	innerNodeSize geometry.Size[lib.Option[float32]],
	gridAreaSize geometry.Size[lib.Option[float32]],
) geometry.Size[lib.Option[float32]] {
	margins := gi.marginsAxisSumsWithBaselineShims(innerNodeSize.Width, t)

	aspectRatio := gi.aspectRatio
	calc := func(id uint64, basis float32) float32 {
		return t.ResolveCalcValue(id, basis)
	}
	padding := style.ResolveOrZeroRectLPSizeContext(gi.paddingStyle, gridAreaSize, calc)
	border := style.ResolveOrZeroRectLPSizeContext(gi.borderStyle, gridAreaSize, calc)
	paddingBorderSize := geometry.RectSumAxes(geometry.RectAddF32(padding, border))
	var boxSizingAdjustment geometry.Size[float32]
	if gi.boxSizingStyle == style.BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSize
	}

	inherentSize := geometry.SizeMaybeAdd(
		geometry.SizeOptionMaybeApplyAspectRatio(
			style.MaybeResolveSizeDimOption(gi.size, gridAreaSize, calc),
			aspectRatio,
		),
		geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)},
	)
	minSizeResolved := geometry.SizeMaybeAdd(
		geometry.SizeOptionMaybeApplyAspectRatio(
			style.MaybeResolveSizeDimOption(gi.minSize, gridAreaSize, calc),
			aspectRatio,
		),
		geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)},
	)
	maxSizeResolved := geometry.SizeMaybeAdd(
		geometry.SizeOptionMaybeApplyAspectRatio(
			style.MaybeResolveSizeDimOption(gi.maxSize, gridAreaSize, calc),
			aspectRatio,
		),
		geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)},
	)

	gridAreaMinusItemMarginsSize := geometry.SizeMaybeSub(gridAreaSize, geometry.Size[lib.Option[float32]]{
		Width:  lib.Some(margins.Width),
		Height: lib.Some(margins.Height),
	})

	// Compute width
	width := inherentSize.Width
	if !width.IsSome {
		// Apply width based on stretch alignment
		if !gi.marginStyle.Left.IsAuto() && !gi.marginStyle.Right.IsAuto() && gi.justifySelf == style.AlignItemsStretch {
			width = gridAreaMinusItemMarginsSize.Width
		}
	}
	// Reapply aspect ratio after stretch width adjustments
	stretchedSize := geometry.SizeOptionMaybeApplyAspectRatio(
		geometry.Size[lib.Option[float32]]{Width: width, Height: inherentSize.Height},
		aspectRatio,
	)
	width = stretchedSize.Width
	height := stretchedSize.Height

	if !height.IsSome {
		// Apply height based on stretch alignment
		if !gi.marginStyle.Top.IsAuto() && !gi.marginStyle.Bottom.IsAuto() && gi.alignSelf == style.AlignItemsStretch {
			height = gridAreaMinusItemMarginsSize.Height
		}
	}
	// Reapply aspect ratio after stretch height adjustments
	stretchedSize2 := geometry.SizeOptionMaybeApplyAspectRatio(
		geometry.Size[lib.Option[float32]]{Width: width, Height: height},
		aspectRatio,
	)
	width = stretchedSize2.Width
	height = stretchedSize2.Height

	// Clamp size by min and max width/height
	clamped := geometry.SizeMaybeClamp(geometry.Size[lib.Option[float32]]{Width: width, Height: height}, minSizeResolved, maxSizeResolved)
	_ = inherentSize // suppress unused
	return clamped
}

// availableSpace computes the available_space to be passed to child sizing functions.
// These are estimates based on either the max track sizing function or the provisional
// base size in the opposite axis to the one currently being sized.
func (gi *gridItem) availableSpace(
	axis geometry.AbstractAxis,
	otherAxisTracks []gridTrack,
	otherAxisAvailableSpace lib.Option[float32],
	getTrackSizeEstimate func(track *gridTrack, availableSpace lib.Option[float32]) lib.Option[float32],
) geometry.Size[lib.Option[float32]] {
	otherStart, otherEnd := gi.trackRangeExcludingLines(axis.Other())
	tracks := otherAxisTracks[otherStart:otherEnd]

	var itemOtherAxisSize lib.Option[float32]
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
		itemOtherAxisSize = lib.Some(sum)
	}

	result := geometry.SizeOptionNone
	result.SetAbstract(axis.Other(), itemOtherAxisSize)
	return result
}

// availableSpaceCached retrieves the available_space from the cache or computes it.
func (gi *gridItem) availableSpaceCached(
	axis geometry.AbstractAxis,
	otherAxisTracks []gridTrack,
	otherAxisAvailableSpace lib.Option[float32],
	getTrackSizeEstimate func(track *gridTrack, availableSpace lib.Option[float32]) lib.Option[float32],
) geometry.Size[lib.Option[float32]] {
	if gi.availableSpaceCache.IsSome {
		return gi.availableSpaceCache.Value
	}
	availableSpaces := gi.availableSpace(axis, otherAxisTracks, otherAxisAvailableSpace, getTrackSizeEstimate)
	gi.availableSpaceCache = lib.Some(availableSpaces)
	return availableSpaces
}

// marginsAxisSumsWithBaselineShims computes the item's resolved margins for size contributions.
// Horizontal percentage margins always resolve to zero if the container size is indefinite.
func (gi *gridItem) marginsAxisSumsWithBaselineShims(
	innerNodeWidth lib.Option[float32],
	t tree.LayoutPartialTree,
) geometry.Size[float32] {
	calc := func(id uint64, basis float32) float32 {
		return t.ResolveCalcValue(id, basis)
	}
	r := geometry.Rect[float32]{
		Left:   style.ResolveOrZeroLPA(gi.marginStyle.Left, lib.Some[float32](0.0), calc),
		Right:  style.ResolveOrZeroLPA(gi.marginStyle.Right, lib.Some[float32](0.0), calc),
		Top:    style.ResolveOrZeroLPA(gi.marginStyle.Top, innerNodeWidth, calc) + gi.baselineShim,
		Bottom: style.ResolveOrZeroLPA(gi.marginStyle.Bottom, innerNodeWidth, calc),
	}
	return geometry.RectSumAxes(r)
}

// minContentContribution computes the item's min content contribution from the provided parameters.
func (gi *gridItem) minContentContribution(
	axis geometry.AbstractAxis,
	t tree.LayoutPartialTree,
	availSpace geometry.Size[lib.Option[float32]],
	innerNodeSize geometry.Size[lib.Option[float32]],
) float32 {
	knownDims := gi.knownDimensions(t, innerNodeSize, availSpace)
	asAvailableSpace := geometry.SizeMapTo(availSpace, func(opt lib.Option[float32]) style.AvailableSpace {
		if opt.IsSome {
			return style.AvailableSpaceFromDefinite(opt.Value)
		}
		return style.AvailableSpaceMinContentValue
	})
	return tree.MeasureChildSize(
		t,
		gi.node,
		knownDims,
		innerNodeSize,
		asAvailableSpace,
		tree.SizingModeInherentSize,
		axis.AsAbsNaive(),
		geometry.LineFalseBool,
	)
}

// minContentContributionCached retrieves the min content contribution from the cache or computes it.
func (gi *gridItem) minContentContributionCached(
	axis geometry.AbstractAxis,
	t tree.LayoutPartialTree,
	availSpace geometry.Size[lib.Option[float32]],
	innerNodeSize geometry.Size[lib.Option[float32]],
) float32 {
	cached := gi.minContentContributionCache.GetAbstract(axis)
	if cached.IsSome {
		return cached.Value
	}
	size := gi.minContentContribution(axis, t, availSpace, innerNodeSize)
	gi.minContentContributionCache.SetAbstract(axis, lib.Some(size))
	return size
}

// maxContentContribution computes the item's max content contribution from the provided parameters.
func (gi *gridItem) maxContentContribution(
	axis geometry.AbstractAxis,
	t tree.LayoutPartialTree,
	availSpace geometry.Size[lib.Option[float32]],
	innerNodeSize geometry.Size[lib.Option[float32]],
) float32 {
	knownDims := gi.knownDimensions(t, innerNodeSize, availSpace)
	asAvailableSpace := geometry.SizeMapTo(availSpace, func(opt lib.Option[float32]) style.AvailableSpace {
		if opt.IsSome {
			return style.AvailableSpaceFromDefinite(opt.Value)
		}
		return style.AvailableSpaceMaxContentValue
	})
	return tree.MeasureChildSize(
		t,
		gi.node,
		knownDims,
		innerNodeSize,
		asAvailableSpace,
		tree.SizingModeInherentSize,
		axis.AsAbsNaive(),
		geometry.LineFalseBool,
	)
}

// maxContentContributionCached retrieves the max content contribution from the cache or computes it.
func (gi *gridItem) maxContentContributionCached(
	axis geometry.AbstractAxis,
	t tree.LayoutPartialTree,
	availSpace geometry.Size[lib.Option[float32]],
	innerNodeSize geometry.Size[lib.Option[float32]],
) float32 {
	cached := gi.maxContentContributionCache.GetAbstract(axis)
	if cached.IsSome {
		return cached.Value
	}
	size := gi.maxContentContribution(axis, t, availSpace, innerNodeSize)
	gi.maxContentContributionCache.SetAbstract(axis, lib.Some(size))
	return size
}

// minimumContribution computes the minimum contribution of an item.
// The minimum contribution is the smallest outer size it can have.
// See: https://www.w3.org/TR/css-grid-1/#min-size-auto
func (gi *gridItem) minimumContribution(
	t tree.LayoutPartialTree,
	axis geometry.AbstractAxis,
	axisTracks []gridTrack,
	knownDims geometry.Size[lib.Option[float32]],
	innerNodeSize geometry.Size[lib.Option[float32]],
) float32 {
	calc := func(id uint64, basis float32) float32 {
		return t.ResolveCalcValue(id, basis)
	}
	padding := style.ResolveOrZeroRectLPSizeContext(gi.paddingStyle, innerNodeSize, calc)
	border := style.ResolveOrZeroRectLPSizeContext(gi.borderStyle, innerNodeSize, calc)
	paddingBorderSize := geometry.RectSumAxes(geometry.RectAddF32(padding, border))
	var boxSizingAdjustment geometry.Size[float32]
	if gi.boxSizingStyle == style.BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSize
	}

	resolvedSize := style.MaybeResolveSizeDimOption(gi.size, innerNodeSize, calc)
	resolvedSize = geometry.SizeOptionMaybeApplyAspectRatio(resolvedSize, gi.aspectRatio)
	resolvedSize = geometry.SizeMaybeAdd(resolvedSize, geometry.Size[lib.Option[float32]]{
		Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height),
	})
	sizeVal := resolvedSize.GetAbstract(axis)

	if !sizeVal.IsSome {
		resolvedMinSize := style.MaybeResolveSizeDimOption(gi.minSize, innerNodeSize, calc)
		resolvedMinSize = geometry.SizeOptionMaybeApplyAspectRatio(resolvedMinSize, gi.aspectRatio)
		resolvedMinSize = geometry.SizeMaybeAdd(resolvedMinSize, geometry.Size[lib.Option[float32]]{
			Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height),
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
			minimumContrib := gi.minContentContributionCached(axis, t, knownDims, innerNodeSize)

			// If the item is a compressible replaced element, cap by preferred/max size
			if gi.isCompressibleReplaced {
				sizeOpt := style.MaybeResolveDimOption(gi.size.GetAbstract(axis), lib.Some[float32](0.0), calc)
				maxSizeOpt := style.MaybeResolveDimOption(gi.maxSize.GetAbstract(axis), lib.Some[float32](0.0), calc)
				minimumContrib = geometry.MaybeMinF32(minimumContrib, sizeOpt)
				minimumContrib = geometry.MaybeMinF32(minimumContrib, maxSizeOpt)
			}

			sizeVal = lib.Some(minimumContrib)
		} else {
			sizeVal = lib.Some[float32](0.0)
		}
	}

	// Clamp by spanned fixed track limit
	limit := gi.spannedFixedTrackLimit(axis, axisTracks, innerNodeSize.GetAbstract(axis), calc)
	return geometry.MaybeMinF32(sizeVal.Value, limit)
}

// minimumContributionCached retrieves the minimum contribution from the cache or computes it.
func (gi *gridItem) minimumContributionCached(
	t tree.LayoutPartialTree,
	axis geometry.AbstractAxis,
	axisTracks []gridTrack,
	knownDims geometry.Size[lib.Option[float32]],
	innerNodeSize geometry.Size[lib.Option[float32]],
) float32 {
	cached := gi.minimumContributionCache.GetAbstract(axis)
	if cached.IsSome {
		return cached.Value
	}
	size := gi.minimumContribution(t, axis, axisTracks, knownDims, innerNodeSize)
	gi.minimumContributionCache.SetAbstract(axis, lib.Some(size))
	return size
}
