// Package grid implements the CSS Grid track sizing algorithm.
// This file is a port of compute/grid/track_sizing.rs from the Taffy Rust layout engine.
package grid

import (
	"cmp"
	"math"
	"slices"

	"taffy-go/geometry"
	"taffy-go/lib"
	"taffy-go/style"
	"taffy-go/tree"
)

// itemBatcher batches items by span size for track sizing.
type itemBatcher struct {
	axis          geometry.AbstractAxis
	indexOffset   int
	currentSpan   uint16
	currentIsFlex bool
}

func newItemBatcher(axis geometry.AbstractAxis) itemBatcher {
	return itemBatcher{axis: axis, indexOffset: 0, currentSpan: 1, currentIsFlex: false}
}

// next returns the next batch of items and whether they are flex items.
// Returns nil, false when iteration is complete.
func (b *itemBatcher) next(items []gridItem) (batch []gridItem, isFlex bool) {
	if b.currentIsFlex || b.indexOffset >= len(items) {
		return nil, false
	}

	item := &items[b.indexOffset]
	b.currentSpan = item.span(b.axis)
	b.currentIsFlex = item.crossesFlexibleTrack(b.axis)

	var nextIndexOffset int
	if b.currentIsFlex {
		nextIndexOffset = len(items)
	} else {
		nextIndexOffset = len(items)
		for i := b.indexOffset; i < len(items); i++ {
			if items[i].crossesFlexibleTrack(b.axis) || items[i].span(b.axis) > b.currentSpan {
				nextIndexOffset = i
				break
			}
		}
	}

	batch = items[b.indexOffset:nextIndexOffset]
	isFlex = b.currentIsFlex
	b.indexOffset = nextIndexOffset
	return batch, isFlex
}

// intrinsicContributionType indicates whether it is a minimum or maximum size's space being distributed.
type intrinsicContributionType int

const (
	intrinsicContributionMinimum intrinsicContributionType = iota
	intrinsicContributionMaximum
)

// cmpByCrossFlexThenSpanThenStart returns a comparison function for sorting grid items.
func cmpByCrossFlexThenSpanThenStart(axis geometry.AbstractAxis) func(a, b *gridItem) int {
	return func(a, b *gridItem) int {
		aCrossFlex := a.crossesFlexibleTrack(axis)
		bCrossFlex := b.crossesFlexibleTrack(axis)
		if !aCrossFlex && bCrossFlex {
			return -1
		}
		if aCrossFlex && !bCrossFlex {
			return 1
		}
		placementA := a.placement(axis)
		placementB := b.placement(axis)
		spanA := style.OriginZeroLineSpan(placementA)
		spanB := style.OriginZeroLineSpan(placementB)
		if spanA < spanB {
			return -1
		}
		if spanA > spanB {
			return 1
		}
		if int16(placementA.Start) < int16(placementB.Start) {
			return -1
		}
		if int16(placementA.Start) > int16(placementB.Start) {
			return 1
		}
		return 0
	}
}

// computeAlignmentGutterAdjustment computes a per-gutter additional size adjustment for content sizing.
func computeAlignmentGutterAdjustment(
	alignment style.AlignContent,
	axisInnerNodeSize lib.Option[float32],
	getTrackSizeEstimate func(track *gridTrack, basis lib.Option[float32]) lib.Option[float32],
	tracks []gridTrack,
) float32 {
	if len(tracks) <= 1 {
		return 0.0
	}

	outerGutterWeight := 0
	switch alignment {
	case style.AlignContentStart, style.AlignContentFlexStart, style.AlignContentEnd, style.AlignContentFlexEnd, style.AlignContentCenter:
		outerGutterWeight = 1
	case style.AlignContentStretch, style.AlignContentSpaceBetween:
		outerGutterWeight = 0
	case style.AlignContentSpaceAround, style.AlignContentSpaceEvenly:
		outerGutterWeight = 1
	}

	innerGutterWeight := 0
	switch alignment {
	case style.AlignContentFlexStart, style.AlignContentStart, style.AlignContentFlexEnd, style.AlignContentEnd, style.AlignContentCenter, style.AlignContentStretch:
		innerGutterWeight = 0
	case style.AlignContentSpaceBetween:
		innerGutterWeight = 1
	case style.AlignContentSpaceAround:
		innerGutterWeight = 2
	case style.AlignContentSpaceEvenly:
		innerGutterWeight = 1
	}

	if innerGutterWeight == 0 {
		return 0.0
	}

	if axisInnerNodeSize.IsSome {
		hasAll := true
		var trackSizeSum float32
		for i := range tracks {
			est := getTrackSizeEstimate(&tracks[i], axisInnerNodeSize)
			if !est.IsSome {
				hasAll = false
				break
			}
			trackSizeSum += est.Value
		}
		var freeSpace float32
		if hasAll {
			freeSpace = geometry.F32Max(0.0, axisInnerNodeSize.Value-trackSizeSum)
		}

		weightedTrackCount := (((len(tracks) - 3) / 2) * innerGutterWeight) + (2 * outerGutterWeight)
		if weightedTrackCount == 0 {
			return 0.0
		}
		return (freeSpace / float32(weightedTrackCount)) * float32(innerGutterWeight)
	}

	return 0.0
}

// resolveItemTrackIndexes converts origin-zero placement to grid track vector indexes.
func resolveItemTrackIndexes(items []gridItem, columnCounts, rowCounts TrackCounts) {
	for i := range items {
		items[i].columnIndexes = geometry.Line[uint16]{
			Start: uint16(OzLineIntoTrackVecIndex(items[i].column.Start, columnCounts)),
			End:   uint16(OzLineIntoTrackVecIndex(items[i].column.End, columnCounts)),
		}
		items[i].rowIndexes = geometry.Line[uint16]{
			Start: uint16(OzLineIntoTrackVecIndex(items[i].row.Start, rowCounts)),
			End:   uint16(OzLineIntoTrackVecIndex(items[i].row.End, rowCounts)),
		}
	}
}

// determineIfItemCrossesFlexibleOrIntrinsicTracks determines (in each axis) whether items cross flexible tracks.
func determineIfItemCrossesFlexibleOrIntrinsicTracks(items []gridItem, columns, rows []gridTrack) {
	for i := range items {
		start, end := items[i].trackRangeExcludingLines(geometry.AbstractAxisInline)
		items[i].crossesFlexibleColumn = false
		items[i].crossesIntrinsicColumn = false
		for j := start; j < end; j++ {
			if columns[j].isFlexible() {
				items[i].crossesFlexibleColumn = true
			}
			if columns[j].hasIntrinsicSizingFunction() {
				items[i].crossesIntrinsicColumn = true
			}
		}

		start, end = items[i].trackRangeExcludingLines(geometry.AbstractAxisBlock)
		items[i].crossesFlexibleRow = false
		items[i].crossesIntrinsicRow = false
		for j := start; j < end; j++ {
			if rows[j].isFlexible() {
				items[i].crossesFlexibleRow = true
			}
			if rows[j].hasIntrinsicSizingFunction() {
				items[i].crossesIntrinsicRow = true
			}
		}
	}
}

// trackSizingAlgorithm runs the full track sizing algorithm for one axis.
func trackSizingAlgorithm(
	t tree.LayoutPartialTree,
	axis geometry.AbstractAxis,
	axisMinSize lib.Option[float32],
	axisMaxSize lib.Option[float32],
	axisAlignment style.AlignContent,
	otherAxisAlignment style.AlignContent,
	availableGridSpace geometry.Size[style.AvailableSpace],
	innerNodeSize geometry.Size[lib.Option[float32]],
	axisTracks []gridTrack,
	otherAxisTracks []gridTrack,
	items []gridItem,
	getTrackSizeEstimate func(track *gridTrack, parentSize lib.Option[float32], tree tree.LayoutPartialTree) lib.Option[float32],
	hasBaselineAlignedItem bool,
) {
	// 11.4 Initialise Track sizes
	percentageBasis := innerNodeSize.GetAbstract(axis)
	if !percentageBasis.IsSome {
		percentageBasis = axisMinSize
	}
	initializeTrackSizes(t, axisTracks, percentageBasis)

	// 11.5.1 Shim item baselines
	if hasBaselineAlignedItem {
		resolveItemBaselines(t, axis, items, innerNodeSize)
	}

	// If all tracks have base_size == growth_limit, skip the rest
	allFixed := true
	for i := range axisTracks {
		if axisTracks[i].baseSize != axisTracks[i].growthLimit {
			allFixed = false
			break
		}
	}
	if allFixed {
		return
	}

	// Pre-computations for 11.5 Resolve Intrinsic Track Sizes
	gutterAlignmentAdjustment := computeAlignmentGutterAdjustment(
		otherAxisAlignment,
		innerNodeSize.GetAbstract(axis.Other()),
		func(track *gridTrack, basis lib.Option[float32]) lib.Option[float32] {
			return getTrackSizeEstimate(track, basis, t)
		},
		otherAxisTracks,
	)
	if len(otherAxisTracks) > 3 {
		for i := 2; i < len(otherAxisTracks); i += 2 {
			otherAxisTracks[i].contentAlignmentAdjustment = gutterAlignmentAdjustment
		}
	}

	// 11.5 Resolve Intrinsic Track Sizes
	resolveIntrinsicTrackSizes(
		t, axis, axisTracks, otherAxisTracks, items,
		availableGridSpace.GetAbstract(axis),
		innerNodeSize, getTrackSizeEstimate,
	)

	// 11.6. Maximise Tracks
	maximiseTracks(axisTracks, innerNodeSize.GetAbstract(axis), availableGridSpace.GetAbstract(axis))

	// Compute available space for expansion
	var axisAvailableSpaceForExpansion style.AvailableSpace
	if innerNodeSize.GetAbstract(axis).IsSome {
		axisAvailableSpaceForExpansion = style.AvailableSpaceFromDefinite(innerNodeSize.GetAbstract(axis).Value)
	} else {
		avail := availableGridSpace.GetAbstract(axis)
		if avail.Tag == style.AvailableSpaceMinContent {
			axisAvailableSpaceForExpansion = style.AvailableSpaceMinContentValue
		} else {
			axisAvailableSpaceForExpansion = style.AvailableSpaceMaxContentValue
		}
	}

	// 11.7. Expand Flexible Tracks
	expandFlexibleTracks(
		t, axis, axisTracks, items,
		axisMinSize, axisMaxSize, axisAvailableSpaceForExpansion, innerNodeSize,
	)

	// 11.8. Stretch auto Tracks
	if axisAlignment == style.AlignContentStretch {
		stretchAutoTracks(axisTracks, axisMinSize, axisAvailableSpaceForExpansion)
	}
}

// initializeTrackSizes initializes each track's base size and growth limit.
func initializeTrackSizes(t tree.LayoutPartialTree, axisTracks []gridTrack, axisInnerNodeSize lib.Option[float32]) {
	calc := func(id uint64, basis float32) float32 {
		return t.ResolveCalcValue(id, basis)
	}
	for i := range axisTracks {
		track := &axisTracks[i]
		track.baseSize = track.minTrackSizingFunction.DefiniteValue(axisInnerNodeSize, calc).UnwrapOr(0.0)
		track.growthLimit = track.maxTrackSizingFunction.DefiniteValue(axisInnerNodeSize, calc).UnwrapOr(float32(math.Inf(1)))
		if track.growthLimit < track.baseSize {
			track.growthLimit = track.baseSize
		}
	}
}

// resolveItemBaselines resolves baselines for baseline-aligned items.
func resolveItemBaselines(
	t tree.LayoutPartialTree,
	axis geometry.AbstractAxis,
	items []gridItem,
	innerNodeSize geometry.Size[lib.Option[float32]],
) {
	otherAxis := axis.Other()
	slices.SortStableFunc(items, func(a, b gridItem) int {
		return cmp.Compare(int16(a.placement(otherAxis).Start), int16(b.placement(otherAxis).Start))
	})

	remaining := items
	for len(remaining) > 0 {
		currentRow := remaining[0].placement(otherAxis).Start

		// Find end of items in this row
		splitIdx := len(remaining)
		for i := 1; i < len(remaining); i++ {
			if remaining[i].placement(otherAxis).Start != currentRow {
				splitIdx = i
				break
			}
		}
		rowItems := remaining[:splitIdx]
		remaining = remaining[splitIdx:]

		// Count baseline-aligned items
		baselineCount := 0
		for i := range rowItems {
			if rowItems[i].alignSelf == style.AlignItemsBaseline {
				baselineCount++
			}
		}
		if baselineCount <= 1 {
			continue
		}

		// Compute baselines
		for i := range rowItems {
			output := tree.PerformChildLayout(
				t, rowItems[i].node,
				geometry.SizeOptionNone, innerNodeSize,
				geometry.Size[style.AvailableSpace]{Width: style.AvailableSpaceMinContentValue, Height: style.AvailableSpaceMinContentValue},
				tree.SizingModeInherentSize, geometry.LineFalseBool,
			)
			bl := output.FirstBaselines.Y
			height := output.Size.Height
			baselineVal := bl.UnwrapOr(height)
			baselineVal += style.ResolveOrZeroLPA(rowItems[i].marginStyle.Top, innerNodeSize.Width, func(id uint64, basis float32) float32 {
				return t.ResolveCalcValue(id, basis)
			})
			rowItems[i].baseline = lib.Some(baselineVal)
		}

		// Compute max baseline
		maxBaseline := float32(0.0)
		for i := range rowItems {
			if rowItems[i].baseline.IsSome && rowItems[i].baseline.Value > maxBaseline {
				maxBaseline = rowItems[i].baseline.Value
			}
		}

		// Compute baseline shim
		for i := range rowItems {
			rowItems[i].baselineShim = maxBaseline - rowItems[i].baseline.UnwrapOr(0.0)
		}
	}
}

// resolveIntrinsicTrackSizes implements step 11.5 of the grid sizing algorithm.
func resolveIntrinsicTrackSizes(
	t tree.LayoutPartialTree,
	axis geometry.AbstractAxis,
	axisTracks []gridTrack,
	otherAxisTracks []gridTrack,
	items []gridItem,
	axisAvailableGridSpace style.AvailableSpace,
	innerNodeSize geometry.Size[lib.Option[float32]],
	getTrackSizeEstimate func(track *gridTrack, parentSize lib.Option[float32], tree tree.LayoutPartialTree) lib.Option[float32],
) {
	// Sort items
	cmpFn := cmpByCrossFlexThenSpanThenStart(axis)
	slices.SortStableFunc(items, func(a, b gridItem) int {
		return cmpFn(&a, &b)
	})

	axisInnerNodeSize := innerNodeSize.GetAbstract(axis)
	var flexFactorSum float32
	for i := range axisTracks {
		flexFactorSum += axisTracks[i].flexFactor()
	}
	calc := func(id uint64, basis float32) float32 {
		return t.ResolveCalcValue(id, basis)
	}

	batcher := newItemBatcher(axis)
	for {
		batch, isFlex := batcher.next(items)
		if len(batch) == 0 {
			break
		}

		batchSpan := batch[0].span(axis)

		// Step 2: size tracks to fit non-spanning items
		if !isFlex && batchSpan == 1 {
			for i := range batch {
				item := &batch[i]
				trackIndex := int(item.placementIndexes(axis).Start) + 1

				// Compute available space and margins for this item
				availSpace := itemAvailableSpaceCached(item, axis, otherAxisTracks, innerNodeSize, getTrackSizeEstimate, t)
				marginAxisSums := item.marginsAxisSumsWithBaselineShims(innerNodeSize.Width, t)
				marginInAxis := marginAxisSums.GetAbstract(axis)

				track := &axisTracks[trackIndex]

				// Handle base sizes
				newBaseSize := track.baseSize
				if track.minTrackSizingFunction.IsMinContent() {
					contrib := itemMinContentContributionCached(item, axis, t, availSpace, innerNodeSize) + marginInAxis
					newBaseSize = geometry.F32Max(newBaseSize, contrib)
				} else if track.minTrackSizingFunction.UsesPercentage() {
					if !axisInnerNodeSize.IsSome {
						contrib := itemMinContentContributionCached(item, axis, t, availSpace, innerNodeSize) + marginInAxis
						newBaseSize = geometry.F32Max(newBaseSize, contrib)
					}
				} else if track.minTrackSizingFunction.IsMaxContent() {
					contrib := itemMaxContentContributionCached(item, axis, t, availSpace, innerNodeSize) + marginInAxis
					newBaseSize = geometry.F32Max(newBaseSize, contrib)
				} else if track.minTrackSizingFunction.IsAuto() {
					switch {
					case (axisAvailableGridSpace.Tag == style.AvailableSpaceMinContent || axisAvailableGridSpace.Tag == style.AvailableSpaceMaxContent) &&
						!item.overflow.GetAbstract(axis).IsScrollContainer():
						axisMinimumSize := itemMinimumContributionCached(item, t, axis, axisTracks, availSpace, innerNodeSize) + marginInAxis
						axisMinContentSize := itemMinContentContributionCached(item, axis, t, availSpace, innerNodeSize) + marginInAxis
						limit := track.maxTrackSizingFunction.DefiniteLimit(axisInnerNodeSize, calc)
						val := axisMinContentSize
						if limit.IsSome && limit.Value < val {
							val = limit.Value
						}
						if axisMinimumSize > val {
							val = axisMinimumSize
						}
						newBaseSize = geometry.F32Max(newBaseSize, val)
					default:
						contrib := itemMinimumContributionCached(item, t, axis, axisTracks, availSpace, innerNodeSize) + marginInAxis
						newBaseSize = geometry.F32Max(newBaseSize, contrib)
					}
				}
				// Length and calc tags - do nothing (not intrinsic)

				axisTracks[trackIndex].baseSize = newBaseSize

				// Handle growth limits
				if track.maxTrackSizingFunction.IsFitContent() {
					if !item.overflow.GetAbstract(axis).IsScrollContainer() {
						minContentContrib := itemMinContentContributionCached(item, axis, t, availSpace, innerNodeSize) + marginInAxis
						axisTracks[trackIndex].growthLimitPlannedIncrease = geometry.F32Max(axisTracks[trackIndex].growthLimitPlannedIncrease, minContentContrib)
					}
					fitContentLimit := axisTracks[trackIndex].fitContentLimit(axisInnerNodeSize)
					maxContentContrib := itemMaxContentContributionCached(item, axis, t, availSpace, innerNodeSize) + marginInAxis
					if maxContentContrib > fitContentLimit {
						maxContentContrib = fitContentLimit
					}
					axisTracks[trackIndex].growthLimitPlannedIncrease = geometry.F32Max(axisTracks[trackIndex].growthLimitPlannedIncrease, maxContentContrib)
				} else if track.maxTrackSizingFunction.IsMaxContentAlike() ||
					(track.maxTrackSizingFunction.UsesPercentage() && !axisInnerNodeSize.IsSome) {
					maxContrib := itemMaxContentContributionCached(item, axis, t, availSpace, innerNodeSize) + marginInAxis
					axisTracks[trackIndex].growthLimitPlannedIncrease = geometry.F32Max(axisTracks[trackIndex].growthLimitPlannedIncrease, maxContrib)
				} else if track.maxTrackSizingFunction.IsIntrinsic() {
					minContrib := itemMinContentContributionCached(item, axis, t, availSpace, innerNodeSize) + marginInAxis
					axisTracks[trackIndex].growthLimitPlannedIncrease = geometry.F32Max(axisTracks[trackIndex].growthLimitPlannedIncrease, minContrib)
				}
			}

			// Flush growth limit increases for step 2
			for i := range axisTracks {
				if axisTracks[i].growthLimitPlannedIncrease > 0.0 {
					if axisTracks[i].growthLimit == float32(math.Inf(1)) {
						axisTracks[i].growthLimit = axisTracks[i].growthLimitPlannedIncrease
					} else {
						axisTracks[i].growthLimit = geometry.F32Max(axisTracks[i].growthLimit, axisTracks[i].growthLimitPlannedIncrease)
					}
				}
				axisTracks[i].infinitelyGrowable = false
				axisTracks[i].growthLimitPlannedIncrease = 0.0
				if axisTracks[i].growthLimit < axisTracks[i].baseSize {
					axisTracks[i].growthLimit = axisTracks[i].baseSize
				}
			}
			continue
		}

		useFlexFactorForDistribution := isFlex && flexFactorSum != 0.0

		// 1. For intrinsic minimums
		for i := range batch {
			item := &batch[i]
			if !item.crossesIntrinsicTrack(axis) {
				continue
			}
			availSpace := itemAvailableSpaceCached(item, axis, otherAxisTracks, innerNodeSize, getTrackSizeEstimate, t)
			marginAxisSums := item.marginsAxisSumsWithBaselineShims(innerNodeSize.Width, t)
			marginInAxis := marginAxisSums.GetAbstract(axis)

			var space float32
			switch {
			case (axisAvailableGridSpace.Tag == style.AvailableSpaceMinContent || axisAvailableGridSpace.Tag == style.AvailableSpaceMaxContent) &&
				!item.overflow.GetAbstract(axis).IsScrollContainer():
				axisMinimumSize := itemMinimumContributionCached(item, t, axis, axisTracks, availSpace, innerNodeSize) + marginInAxis
				axisMinContentSize := itemMinContentContributionCached(item, axis, t, availSpace, innerNodeSize) + marginInAxis
				limit := item.spannedTrackLimit(axis, axisTracks, axisInnerNodeSize, calc)
				val := axisMinContentSize
				if limit.IsSome && limit.Value < val {
					val = limit.Value
				}
				if axisMinimumSize > val {
					val = axisMinimumSize
				}
				space = val
			default:
				space = itemMinimumContributionCached(item, t, axis, axisTracks, availSpace, innerNodeSize) + marginInAxis
			}

			if space > 0.0 {
				start, end := item.trackRangeExcludingLines(axis)
				tracks := axisTracks[start:end]
				hasIntrinsicMinTSF := func(t *gridTrack) bool {
					return !t.minTrackSizingFunction.DefiniteValue(axisInnerNodeSize, calc).IsSome
				}
				if item.overflow.GetAbstract(axis).IsScrollContainer() {
					fcLimit := func(t *gridTrack) float32 { return t.fitContentLimitedGrowthLimit(axisInnerNodeSize) }
					distributeItemSpaceToBaseSize(isFlex, useFlexFactorForDistribution, space, tracks, hasIntrinsicMinTSF, fcLimit, intrinsicContributionMinimum)
				} else {
					glLimit := func(t *gridTrack) float32 { return t.growthLimit }
					distributeItemSpaceToBaseSize(isFlex, useFlexFactorForDistribution, space, tracks, hasIntrinsicMinTSF, glLimit, intrinsicContributionMinimum)
				}
			}
		}
		flushPlannedBaseSizeIncreases(axisTracks)

		// 2. For content-based minimums
		hasMinOrMaxContentMinTSF := func(t *gridTrack) bool { return t.minTrackSizingFunction.IsMinOrMaxContent() }
		for i := range batch {
			item := &batch[i]
			availSpace := itemAvailableSpaceCached(item, axis, otherAxisTracks, innerNodeSize, getTrackSizeEstimate, t)
			marginAxisSums := item.marginsAxisSumsWithBaselineShims(innerNodeSize.Width, t)
			space := itemMinContentContributionCached(item, axis, t, availSpace, innerNodeSize) + marginAxisSums.GetAbstract(axis)
			if space > 0.0 {
				start, end := item.trackRangeExcludingLines(axis)
				tracks := axisTracks[start:end]
				if item.overflow.GetAbstract(axis).IsScrollContainer() {
					fcLimit := func(t *gridTrack) float32 { return t.fitContentLimitedGrowthLimit(axisInnerNodeSize) }
					distributeItemSpaceToBaseSize(isFlex, useFlexFactorForDistribution, space, tracks, hasMinOrMaxContentMinTSF, fcLimit, intrinsicContributionMinimum)
				} else {
					glLimit := func(t *gridTrack) float32 { return t.growthLimit }
					distributeItemSpaceToBaseSize(isFlex, useFlexFactorForDistribution, space, tracks, hasMinOrMaxContentMinTSF, glLimit, intrinsicContributionMinimum)
				}
			}
		}
		flushPlannedBaseSizeIncreases(axisTracks)

		// 3. For max-content minimums
		if axisAvailableGridSpace.Tag == style.AvailableSpaceMaxContent {
			hasAutoMinTSF := func(t *gridTrack) bool {
				return t.minTrackSizingFunction.IsAuto() && !t.maxTrackSizingFunction.IsMinContent()
			}
			hasMaxContentMinTSF := func(t *gridTrack) bool { return t.minTrackSizingFunction.IsMaxContent() }

			for i := range batch {
				item := &batch[i]
				availSpace := itemAvailableSpaceCached(item, axis, otherAxisTracks, innerNodeSize, getTrackSizeEstimate, t)
				marginAxisSums := item.marginsAxisSumsWithBaselineShims(innerNodeSize.Width, t)
				marginInAxis := marginAxisSums.GetAbstract(axis)
				axisMaxContentSize := itemMaxContentContributionCached(item, axis, t, availSpace, innerNodeSize) + marginInAxis
				limit := item.spannedTrackLimit(axis, axisTracks, axisInnerNodeSize, calc)
				space := axisMaxContentSize
				if limit.IsSome && limit.Value < space {
					space = limit.Value
				}
				if space > 0.0 {
					start, end := item.trackRangeExcludingLines(axis)
					tracks := axisTracks[start:end]
					hasMaxContentTrack := false
					for i := range tracks {
						if tracks[i].minTrackSizingFunction.IsMaxContent() {
							hasMaxContentTrack = true
							break
						}
					}
					if hasMaxContentTrack {
						distributeItemSpaceToBaseSize(isFlex, useFlexFactorForDistribution, space, tracks, hasMaxContentMinTSF,
							func(_ *gridTrack) float32 { return float32(math.Inf(1)) }, intrinsicContributionMaximum)
					} else {
						fcLimitedGrowthLimit := func(t *gridTrack) float32 { return t.fitContentLimitedGrowthLimit(axisInnerNodeSize) }
						distributeItemSpaceToBaseSize(isFlex, useFlexFactorForDistribution, space, tracks, hasAutoMinTSF, fcLimitedGrowthLimit, intrinsicContributionMaximum)
					}
				}
			}
			flushPlannedBaseSizeIncreases(axisTracks)
		}

		// In all cases: max-content base size
		hasMaxContentMinTSF2 := func(t *gridTrack) bool { return t.minTrackSizingFunction.IsMaxContent() }
		for i := range batch {
			item := &batch[i]
			availSpace := itemAvailableSpaceCached(item, axis, otherAxisTracks, innerNodeSize, getTrackSizeEstimate, t)
			marginAxisSums := item.marginsAxisSumsWithBaselineShims(innerNodeSize.Width, t)
			space := itemMaxContentContributionCached(item, axis, t, availSpace, innerNodeSize) + marginAxisSums.GetAbstract(axis)
			if space > 0.0 {
				start, end := item.trackRangeExcludingLines(axis)
				tracks := axisTracks[start:end]
				glLimit := func(t *gridTrack) float32 { return t.growthLimit }
				distributeItemSpaceToBaseSize(isFlex, useFlexFactorForDistribution, space, tracks, hasMaxContentMinTSF2, glLimit, intrinsicContributionMaximum)
			}
		}
		flushPlannedBaseSizeIncreases(axisTracks)

		// 4. Ensure growth limits are at least as large as base sizes
		for i := range axisTracks {
			if axisTracks[i].growthLimit < axisTracks[i].baseSize {
				axisTracks[i].growthLimit = axisTracks[i].baseSize
			}
		}

		if !isFlex {
			// 5. For intrinsic maximums
			hasIntrinsicMaxTSF := func(t *gridTrack) bool {
				return !t.maxTrackSizingFunction.HasDefiniteValue(axisInnerNodeSize)
			}
			for i := range batch {
				item := &batch[i]
				availSpace := itemAvailableSpaceCached(item, axis, otherAxisTracks, innerNodeSize, getTrackSizeEstimate, t)
				marginAxisSums := item.marginsAxisSumsWithBaselineShims(innerNodeSize.Width, t)
				space := itemMinContentContributionCached(item, axis, t, availSpace, innerNodeSize) + marginAxisSums.GetAbstract(axis)
				if space > 0.0 {
					start, end := item.trackRangeExcludingLines(axis)
					tracks := axisTracks[start:end]
					distributeItemSpaceToGrowthLimit(space, tracks, hasIntrinsicMaxTSF, axisInnerNodeSize)
				}
			}
			flushPlannedGrowthLimitIncreases(axisTracks, true)

			// 6. For max-content maximums
			hasMaxContentMaxTSF := func(t *gridTrack) bool {
				return t.maxTrackSizingFunction.IsMaxContentAlike() ||
					(t.maxTrackSizingFunction.UsesPercentage() && !axisInnerNodeSize.IsSome)
			}
			for i := range batch {
				item := &batch[i]
				availSpace := itemAvailableSpaceCached(item, axis, otherAxisTracks, innerNodeSize, getTrackSizeEstimate, t)
				marginAxisSums := item.marginsAxisSumsWithBaselineShims(innerNodeSize.Width, t)
				space := itemMaxContentContributionCached(item, axis, t, availSpace, innerNodeSize) + marginAxisSums.GetAbstract(axis)
				if space > 0.0 {
					start, end := item.trackRangeExcludingLines(axis)
					tracks := axisTracks[start:end]
					distributeItemSpaceToGrowthLimit(space, tracks, hasMaxContentMaxTSF, axisInnerNodeSize)
				}
			}
			flushPlannedGrowthLimitIncreases(axisTracks, false)
		}
	}

	// Step 5. Set infinite growth limits to base size
	for i := range axisTracks {
		if axisTracks[i].growthLimit == float32(math.Inf(1)) {
			axisTracks[i].growthLimit = axisTracks[i].baseSize
		}
	}
}

// flushPlannedBaseSizeIncreases adds planned base size increases and resets them.
func flushPlannedBaseSizeIncreases(tracks []gridTrack) {
	for i := range tracks {
		tracks[i].baseSize += tracks[i].baseSizePlannedIncrease
		tracks[i].baseSizePlannedIncrease = 0.0
	}
}

// flushPlannedGrowthLimitIncreases adds planned growth limit increases and resets them.
func flushPlannedGrowthLimitIncreases(tracks []gridTrack, setInfinitelyGrowable bool) {
	for i := range tracks {
		if tracks[i].growthLimitPlannedIncrease > 0.0 {
			if tracks[i].growthLimit == float32(math.Inf(1)) {
				tracks[i].growthLimit = tracks[i].baseSize + tracks[i].growthLimitPlannedIncrease
			} else {
				tracks[i].growthLimit += tracks[i].growthLimitPlannedIncrease
			}
			tracks[i].infinitelyGrowable = setInfinitelyGrowable
		} else {
			tracks[i].infinitelyGrowable = false
		}
		tracks[i].growthLimitPlannedIncrease = 0.0
	}
}

// distributeItemSpaceToBaseSize distributes extra space across spanned tracks to base sizes.
func distributeItemSpaceToBaseSize(
	isFlex bool,
	useFlexFactorForDistribution bool,
	space float32,
	tracks []gridTrack,
	trackIsAffected func(*gridTrack) bool,
	trackLimit func(*gridTrack) float32,
	contributionType intrinsicContributionType,
) {
	if isFlex {
		filter := func(t *gridTrack) bool { return t.isFlexible() && trackIsAffected(t) }
		if useFlexFactorForDistribution {
			distributeItemSpaceToBaseSizeInner(space, tracks, filter,
				func(t *gridTrack) float32 { return t.flexFactor() }, trackLimit, contributionType)
		} else {
			distributeItemSpaceToBaseSizeInner(space, tracks, filter,
				func(_ *gridTrack) float32 { return 1.0 }, trackLimit, contributionType)
		}
	} else {
		distributeItemSpaceToBaseSizeInner(space, tracks, trackIsAffected,
			func(_ *gridTrack) float32 { return 1.0 }, trackLimit, contributionType)
	}
}

// distributeItemSpaceToBaseSizeInner is the inner function for distributing space to base sizes.
func distributeItemSpaceToBaseSizeInner(
	space float32,
	tracks []gridTrack,
	trackIsAffected func(*gridTrack) bool,
	trackDistributionProportion func(*gridTrack) float32,
	trackLimit func(*gridTrack) float32,
	contributionType intrinsicContributionType,
) {
	if space == 0.0 {
		return
	}
	hasAffected := false
	for i := range tracks {
		if trackIsAffected(&tracks[i]) {
			hasAffected = true
			break
		}
	}
	if !hasAffected {
		return
	}

	getBaseSize := func(t *gridTrack) float32 { return t.baseSize }

	// 1. Find the space to distribute
	var trackSizes float32
	for i := range tracks {
		trackSizes += tracks[i].baseSize
	}
	extraSpace := geometry.F32Max(0.0, space-trackSizes)

	// 2. Distribute space up to limits
	const threshold float32 = 0.000001
	extraSpace = distributeSpaceUpToLimits(extraSpace, tracks, trackIsAffected, trackDistributionProportion, getBaseSize, trackLimit)

	// 3. Distribute remaining space beyond limits
	if extraSpace > threshold {
		var filter func(*gridTrack) bool
		switch contributionType {
		case intrinsicContributionMinimum:
			filter = func(t *gridTrack) bool { return t.maxTrackSizingFunction.IsIntrinsic() }
		case intrinsicContributionMaximum:
			filter = func(t *gridTrack) bool {
				return t.minTrackSizingFunction.IsMaxContent() || t.maxTrackSizingFunction.IsMaxOrFitContent()
			}
		}

		count := 0
		for i := range tracks {
			if trackIsAffected(&tracks[i]) && filter(&tracks[i]) {
				count++
			}
		}
		if count == 0 {
			filter = func(_ *gridTrack) bool { return true }
		}

		distributeSpaceUpToLimits(extraSpace, tracks, filter, trackDistributionProportion, getBaseSize, trackLimit)
	}

	// 4. Set planned increases
	for i := range tracks {
		if tracks[i].itemIncurredIncrease > tracks[i].baseSizePlannedIncrease {
			tracks[i].baseSizePlannedIncrease = tracks[i].itemIncurredIncrease
		}
		tracks[i].itemIncurredIncrease = 0.0
	}
}

// distributeItemSpaceToGrowthLimit distributes extra space across spanned tracks to growth limits.
func distributeItemSpaceToGrowthLimit(
	space float32,
	tracks []gridTrack,
	trackIsAffected func(*gridTrack) bool,
	axisInnerNodeSize lib.Option[float32],
) {
	if space == 0.0 {
		return
	}
	affectedCount := 0
	for i := range tracks {
		if trackIsAffected(&tracks[i]) {
			affectedCount++
		}
	}
	if affectedCount == 0 {
		return
	}

	// 1. Find the space to distribute
	var trackSizes float32
	for i := range tracks {
		if tracks[i].growthLimit == float32(math.Inf(1)) {
			trackSizes += tracks[i].baseSize
		} else {
			trackSizes += tracks[i].growthLimit
		}
	}
	extraSpace := geometry.F32Max(0.0, space-trackSizes)

	// 2. Distribute space up to limits
	growableCount := 0
	for i := range tracks {
		if trackIsAffected(&tracks[i]) &&
			(tracks[i].infinitelyGrowable || tracks[i].fitContentLimitedGrowthLimit(axisInnerNodeSize) == float32(math.Inf(1))) {
			growableCount++
		}
	}

	if growableCount > 0 {
		itemIncurredIncrease := extraSpace / float32(growableCount)
		for i := range tracks {
			if trackIsAffected(&tracks[i]) &&
				(tracks[i].infinitelyGrowable || tracks[i].fitContentLimitedGrowthLimit(axisInnerNodeSize) == float32(math.Inf(1))) {
				tracks[i].itemIncurredIncrease = itemIncurredIncrease
			}
		}
	} else {
		// 3. Distribute space beyond limits
		getGrowthLimitOrBase := func(t *gridTrack) float32 {
			if t.growthLimit == float32(math.Inf(1)) {
				return t.baseSize
			}
			return t.growthLimit
		}
		fcLimit := func(t *gridTrack) float32 { return t.fitContentLimit(axisInnerNodeSize) }
		distributeSpaceUpToLimits(extraSpace, tracks, trackIsAffected,
			func(_ *gridTrack) float32 { return 1.0 }, getGrowthLimitOrBase, fcLimit)
	}

	// 4. Set planned increases
	for i := range tracks {
		if tracks[i].itemIncurredIncrease > tracks[i].growthLimitPlannedIncrease {
			tracks[i].growthLimitPlannedIncrease = tracks[i].itemIncurredIncrease
		}
		tracks[i].itemIncurredIncrease = 0.0
	}
}

// maximiseTracks distributes free space to tracks with finite growth limits, up to their limits.
func maximiseTracks(axisTracks []gridTrack, axisInnerNodeSize lib.Option[float32], axisAvailableGridSpace style.AvailableSpace) {
	var usedSpace float32
	for i := range axisTracks {
		usedSpace += axisTracks[i].baseSize
	}
	freeSpace := axisAvailableGridSpace.ComputeFreeSpace(usedSpace)
	if freeSpace == float32(math.Inf(1)) {
		for i := range axisTracks {
			axisTracks[i].baseSize = axisTracks[i].growthLimit
		}
	} else if freeSpace > 0.0 {
		fcLimitedGL := func(t *gridTrack) float32 { return t.fitContentLimitedGrowthLimit(axisInnerNodeSize) }
		distributeSpaceUpToLimits(
			freeSpace, axisTracks,
			func(_ *gridTrack) bool { return true },
			func(_ *gridTrack) float32 { return 1.0 },
			func(t *gridTrack) float32 { return t.baseSize },
			fcLimitedGL,
		)
		for i := range axisTracks {
			axisTracks[i].baseSize += axisTracks[i].itemIncurredIncrease
			axisTracks[i].itemIncurredIncrease = 0.0
		}
	}
}

// expandFlexibleTracks sizes flexible tracks using the largest value it can assign to an fr.
func expandFlexibleTracks(
	t tree.LayoutPartialTree,
	axis geometry.AbstractAxis,
	axisTracks []gridTrack,
	items []gridItem,
	axisMinSize lib.Option[float32],
	axisMaxSize lib.Option[float32],
	axisAvailableSpaceForExpansion style.AvailableSpace,
	innerNodeSize geometry.Size[lib.Option[float32]],
) {
	var flexFraction float32
	switch axisAvailableSpaceForExpansion.Tag {
	case style.AvailableSpaceDefinite:
		availableSpace := axisAvailableSpaceForExpansion.Value
		var usedSpace float32
		for i := range axisTracks {
			usedSpace += axisTracks[i].baseSize
		}
		freeSpace := availableSpace - usedSpace
		if freeSpace <= 0.0 {
			flexFraction = 0.0
		} else {
			flexFraction = findSizeOfFr(axisTracks, availableSpace)
		}
	case style.AvailableSpaceMinContent:
		flexFraction = 0.0
	case style.AvailableSpaceMaxContent:
		// Maximum of: track flex contributions and item flex contributions
		maxTrackFlex := float32(0.0)
		for i := range axisTracks {
			if axisTracks[i].maxTrackSizingFunction.IsFr() {
				ff := axisTracks[i].flexFactor()
				var val float32
				if ff > 1.0 {
					val = axisTracks[i].baseSize / ff
				} else {
					val = axisTracks[i].baseSize
				}
				if val > maxTrackFlex {
					maxTrackFlex = val
				}
			}
		}

		maxItemFlex := float32(0.0)
		for i := range items {
			if items[i].crossesFlexibleTrack(axis) {
				start, end := items[i].trackRangeExcludingLines(axis)
				itemTracks := axisTracks[start:end]
				maxContentContrib := items[i].maxContentContributionCached(axis, t, geometry.SizeOptionNone, innerNodeSize)
				fr := findSizeOfFr(itemTracks, maxContentContrib)
				if fr > maxItemFlex {
					maxItemFlex = fr
				}
			}
		}

		flexFraction = geometry.F32Max(maxTrackFlex, maxItemFlex)

		// Check min/max size constraints
		var hypotheticalGridSize float32
		for i := range axisTracks {
			if axisTracks[i].maxTrackSizingFunction.IsFr() {
				trackFF := axisTracks[i].maxTrackSizingFunction.Value
				hypotheticalGridSize += geometry.F32Max(axisTracks[i].baseSize, trackFF*flexFraction)
			} else {
				hypotheticalGridSize += axisTracks[i].baseSize
			}
		}
		axisMin := axisMinSize.UnwrapOr(0.0)
		axisMax := axisMaxSize.UnwrapOr(float32(math.Inf(1)))
		if hypotheticalGridSize < axisMin {
			flexFraction = findSizeOfFr(axisTracks, axisMin)
		} else if hypotheticalGridSize > axisMax {
			flexFraction = findSizeOfFr(axisTracks, axisMax)
		}
	}

	// Apply flex fraction to flexible tracks
	for i := range axisTracks {
		if axisTracks[i].maxTrackSizingFunction.IsFr() {
			trackFF := axisTracks[i].maxTrackSizingFunction.Value
			axisTracks[i].baseSize = geometry.F32Max(axisTracks[i].baseSize, trackFF*flexFraction)
		}
	}
}

// findSizeOfFr finds the largest size that an fr unit can be without exceeding the target size.
func findSizeOfFr(tracks []gridTrack, spaceToFill float32) float32 {
	if spaceToFill == 0.0 {
		return 0.0
	}

	hypotheticalFrSize := float32(math.Inf(1))
	var previousIterHypotheticalFrSize float32

	for {
		var usedSpace float32
		var naiveFlexFactorSum float32
		for i := range tracks {
			if tracks[i].maxTrackSizingFunction.IsFr() &&
				tracks[i].maxTrackSizingFunction.Value*hypotheticalFrSize >= tracks[i].baseSize {
				naiveFlexFactorSum += tracks[i].maxTrackSizingFunction.Value
			} else {
				usedSpace += tracks[i].baseSize
			}
		}
		leftoverSpace := spaceToFill - usedSpace
		flexFactor := geometry.F32Max(naiveFlexFactorSum, 1.0)

		previousIterHypotheticalFrSize = hypotheticalFrSize
		hypotheticalFrSize = leftoverSpace / flexFactor

		// Validate
		valid := true
		for i := range tracks {
			if tracks[i].maxTrackSizingFunction.IsFr() {
				ff := tracks[i].maxTrackSizingFunction.Value
				if !(ff*hypotheticalFrSize >= tracks[i].baseSize ||
					ff*previousIterHypotheticalFrSize < tracks[i].baseSize) {
					valid = false
					break
				}
			}
		}
		if valid {
			break
		}
	}

	return hypotheticalFrSize
}

// stretchAutoTracks stretches tracks with auto max sizing functions.
func stretchAutoTracks(
	axisTracks []gridTrack,
	axisMinSize lib.Option[float32],
	axisAvailableSpaceForExpansion style.AvailableSpace,
) {
	numAutoTracks := 0
	for i := range axisTracks {
		if axisTracks[i].maxTrackSizingFunction.IsAuto() {
			numAutoTracks++
		}
	}
	if numAutoTracks == 0 {
		return
	}

	var usedSpace float32
	for i := range axisTracks {
		usedSpace += axisTracks[i].baseSize
	}

	var freeSpace float32
	if axisAvailableSpaceForExpansion.IsDefinite() {
		freeSpace = axisAvailableSpaceForExpansion.ComputeFreeSpace(usedSpace)
	} else if axisMinSize.IsSome {
		freeSpace = axisMinSize.Value - usedSpace
	}

	if freeSpace > 0.0 {
		extraSpacePerAutoTrack := freeSpace / float32(numAutoTracks)
		for i := range axisTracks {
			if axisTracks[i].maxTrackSizingFunction.IsAuto() {
				axisTracks[i].baseSize += extraSpacePerAutoTrack
			}
		}
	}
}

// distributeSpaceUpToLimits distributes space to tracks evenly up to their limits.
func distributeSpaceUpToLimits(
	spaceToDistribute float32,
	tracks []gridTrack,
	trackIsAffected func(*gridTrack) bool,
	trackDistributionProportion func(*gridTrack) float32,
	trackAffectedProperty func(*gridTrack) float32,
	trackLimit func(*gridTrack) float32,
) float32 {
	const threshold float32 = 0.01
	remaining := spaceToDistribute

	for remaining > threshold {
		var proportionSum float32
		for i := range tracks {
			if trackAffectedProperty(&tracks[i])+tracks[i].itemIncurredIncrease < trackLimit(&tracks[i]) &&
				trackIsAffected(&tracks[i]) {
				proportionSum += trackDistributionProportion(&tracks[i])
			}
		}
		if proportionSum == 0.0 {
			break
		}

		// Compute min_increase_limit
		minIncreaseLimit := float32(math.Inf(1))
		for i := range tracks {
			if trackAffectedProperty(&tracks[i])+tracks[i].itemIncurredIncrease < trackLimit(&tracks[i]) &&
				trackIsAffected(&tracks[i]) {
				val := (trackLimit(&tracks[i]) - trackAffectedProperty(&tracks[i])) / trackDistributionProportion(&tracks[i])
				if val < minIncreaseLimit {
					minIncreaseLimit = val
				}
			}
		}

		iterIncrease := geometry.F32Min(minIncreaseLimit, remaining/proportionSum)

		for i := range tracks {
			if trackIsAffected(&tracks[i]) {
				increase := iterIncrease * trackDistributionProportion(&tracks[i])
				if increase > 0.0 && trackAffectedProperty(&tracks[i])+increase <= trackLimit(&tracks[i])+threshold {
					tracks[i].itemIncurredIncrease += increase
					remaining -= increase
				}
			}
		}
	}

	return remaining
}

// Helper functions to call item methods while keeping the track sizing code readable.

func itemAvailableSpaceCached(
	item *gridItem,
	axis geometry.AbstractAxis,
	otherAxisTracks []gridTrack,
	innerNodeSize geometry.Size[lib.Option[float32]],
	getTrackSizeEstimate func(track *gridTrack, parentSize lib.Option[float32], tree tree.LayoutPartialTree) lib.Option[float32],
	t tree.LayoutPartialTree,
) geometry.Size[lib.Option[float32]] {
	return item.availableSpaceCached(axis, otherAxisTracks, innerNodeSize.GetAbstract(axis.Other()),
		func(track *gridTrack, availableSpace lib.Option[float32]) lib.Option[float32] {
			return getTrackSizeEstimate(track, availableSpace, t)
		})
}

func itemMinContentContributionCached(
	item *gridItem,
	axis geometry.AbstractAxis,
	t tree.LayoutPartialTree,
	availSpace geometry.Size[lib.Option[float32]],
	innerNodeSize geometry.Size[lib.Option[float32]],
) float32 {
	return item.minContentContributionCached(axis, t, availSpace, innerNodeSize)
}

func itemMaxContentContributionCached(
	item *gridItem,
	axis geometry.AbstractAxis,
	t tree.LayoutPartialTree,
	availSpace geometry.Size[lib.Option[float32]],
	innerNodeSize geometry.Size[lib.Option[float32]],
) float32 {
	return item.maxContentContributionCached(axis, t, availSpace, innerNodeSize)
}

func itemMinimumContributionCached(
	item *gridItem,
	t tree.LayoutPartialTree,
	axis geometry.AbstractAxis,
	axisTracks []gridTrack,
	availSpace geometry.Size[lib.Option[float32]],
	innerNodeSize geometry.Size[lib.Option[float32]],
) float32 {
	return item.minimumContributionCached(t, axis, axisTracks, availSpace, innerNodeSize)
}
