// Package taffy implements the CSS Grid track sizing algorithm.
// This file is a port of compute/grid/track_sizing.rs from the Taffy Rust layout engine.
package taffy

import (
	"math"
	"sort"
)

// itemBatcher batches items by span size for track sizing.
type itemBatcher struct {
	axis          AbstractAxis
	indexOffset   int
	currentSpan   uint16
	currentIsFlex bool
}

func newItemBatcher(axis AbstractAxis) itemBatcher {
	return itemBatcher{axis: axis, indexOffset: 0, currentSpan: 1, currentIsFlex: false}
}

// next returns the next batch of items and whether they are flex items.
// Returns nil, false when iteration is complete.
func (b *itemBatcher) next(items []gridItem) (batch []int, isFlex bool) {
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

	indices := make([]int, 0, nextIndexOffset-b.indexOffset)
	for i := b.indexOffset; i < nextIndexOffset; i++ {
		indices = append(indices, i)
	}
	isFlex = b.currentIsFlex
	b.indexOffset = nextIndexOffset
	return indices, isFlex
}

// intrinsicContributionType indicates whether it is a minimum or maximum size's space being distributed.
type intrinsicContributionType int

const (
	intrinsicContributionMinimum intrinsicContributionType = iota
	intrinsicContributionMaximum
)

// cmpByCrossFlexThenSpanThenStart returns a comparison function for sorting grid items.
func cmpByCrossFlexThenSpanThenStart(axis AbstractAxis) func(a, b *gridItem) int {
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
		spanA := OriginZeroLineSpan(placementA)
		spanB := OriginZeroLineSpan(placementB)
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
	alignment AlignContent,
	axisInnerNodeSize Option[float32],
	getTrackSizeEstimate func(track *gridTrack, basis Option[float32]) Option[float32],
	tracks []gridTrack,
) float32 {
	if len(tracks) <= 1 {
		return 0.0
	}

	outerGutterWeight := 0
	switch alignment {
	case AlignContentStart, AlignContentFlexStart, AlignContentEnd, AlignContentFlexEnd, AlignContentCenter:
		outerGutterWeight = 1
	case AlignContentStretch, AlignContentSpaceBetween:
		outerGutterWeight = 0
	case AlignContentSpaceAround, AlignContentSpaceEvenly:
		outerGutterWeight = 1
	}

	innerGutterWeight := 0
	switch alignment {
	case AlignContentFlexStart, AlignContentStart, AlignContentFlexEnd, AlignContentEnd, AlignContentCenter, AlignContentStretch:
		innerGutterWeight = 0
	case AlignContentSpaceBetween:
		innerGutterWeight = 1
	case AlignContentSpaceAround:
		innerGutterWeight = 2
	case AlignContentSpaceEvenly:
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
			freeSpace = f32Max(0.0, axisInnerNodeSize.Value-trackSizeSum)
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
		items[i].columnIndexes = Line[uint16]{
			Start: uint16(items[i].column.Start.IntoTrackVecIndex(columnCounts)),
			End:   uint16(items[i].column.End.IntoTrackVecIndex(columnCounts)),
		}
		items[i].rowIndexes = Line[uint16]{
			Start: uint16(items[i].row.Start.IntoTrackVecIndex(rowCounts)),
			End:   uint16(items[i].row.End.IntoTrackVecIndex(rowCounts)),
		}
	}
}

// determineIfItemCrossesFlexibleOrIntrinsicTracks determines (in each axis) whether items cross flexible tracks.
func determineIfItemCrossesFlexibleOrIntrinsicTracks(items []gridItem, columns, rows []gridTrack) {
	for i := range items {
		start, end := items[i].trackRangeExcludingLines(AbstractAxisInline)
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

		start, end = items[i].trackRangeExcludingLines(AbstractAxisBlock)
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
	tree layoutPartialTree,
	axis AbstractAxis,
	axisMinSize Option[float32],
	axisMaxSize Option[float32],
	axisAlignment AlignContent,
	otherAxisAlignment AlignContent,
	availableGridSpace Size[AvailableSpace],
	innerNodeSize Size[Option[float32]],
	axisTracks []gridTrack,
	otherAxisTracks []gridTrack,
	items []gridItem,
	getTrackSizeEstimate func(track *gridTrack, parentSize Option[float32], tree layoutPartialTree) Option[float32],
	hasBaselineAlignedItem bool,
) {
	// 11.4 Initialise Track sizes
	percentageBasis := innerNodeSize.GetAbstract(axis)
	if !percentageBasis.IsSome {
		percentageBasis = axisMinSize
	}
	initializeTrackSizes(tree, axisTracks, percentageBasis)

	// 11.5.1 Shim item baselines
	if hasBaselineAlignedItem {
		resolveItemBaselines(tree, axis, items, innerNodeSize)
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
		func(track *gridTrack, basis Option[float32]) Option[float32] {
			return getTrackSizeEstimate(track, basis, tree)
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
		tree, axis, axisTracks, otherAxisTracks, items,
		availableGridSpace.GetAbstract(axis),
		innerNodeSize, getTrackSizeEstimate,
	)

	// 11.6. Maximise Tracks
	maximiseTracks(axisTracks, innerNodeSize.GetAbstract(axis), availableGridSpace.GetAbstract(axis))

	// Compute available space for expansion
	var axisAvailableSpaceForExpansion AvailableSpace
	if innerNodeSize.GetAbstract(axis).IsSome {
		axisAvailableSpaceForExpansion = AvailableSpaceFromDefinite(innerNodeSize.GetAbstract(axis).Value)
	} else {
		avail := availableGridSpace.GetAbstract(axis)
		if avail.Tag == AvailableSpaceMinContent {
			axisAvailableSpaceForExpansion = AvailableSpaceMinContentValue
		} else {
			axisAvailableSpaceForExpansion = AvailableSpaceMaxContentValue
		}
	}

	// 11.7. Expand Flexible Tracks
	expandFlexibleTracks(
		tree, axis, axisTracks, items,
		axisMinSize, axisMaxSize, axisAvailableSpaceForExpansion, innerNodeSize,
	)

	// 11.8. Stretch auto Tracks
	if axisAlignment == AlignContentStretch {
		stretchAutoTracks(axisTracks, axisMinSize, axisAvailableSpaceForExpansion)
	}
}

// initializeTrackSizes initializes each track's base size and growth limit.
func initializeTrackSizes(tree layoutPartialTree, axisTracks []gridTrack, axisInnerNodeSize Option[float32]) {
	calc := tree.resolveCalcValue
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
	tree layoutPartialTree,
	axis AbstractAxis,
	items []gridItem,
	innerNodeSize Size[Option[float32]],
) {
	otherAxis := axis.Other()
	sort.SliceStable(items, func(i, j int) bool {
		return int16(items[i].placement(otherAxis).Start) < int16(items[j].placement(otherAxis).Start)
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
			if rowItems[i].alignSelf == AlignItemsBaseline {
				baselineCount++
			}
		}
		if baselineCount <= 1 {
			continue
		}

		// Compute baselines
		for i := range rowItems {
			output := performChildLayout(
				tree, rowItems[i].node,
				SizeOptionNone, innerNodeSize,
				Size[AvailableSpace]{Width: AvailableSpaceMinContentValue, Height: AvailableSpaceMinContentValue},
				SizingModeInherentSize, LineFalseBool,
			)
			bl := output.FirstBaselines.Y
			height := output.Size.Height
			baselineVal := bl.UnwrapOr(height)
			baselineVal += ResolveOrZeroLPA(rowItems[i].marginStyle.Top, innerNodeSize.Width, tree.resolveCalcValue)
			rowItems[i].baseline = Some(baselineVal)
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
	tree layoutPartialTree,
	axis AbstractAxis,
	axisTracks []gridTrack,
	otherAxisTracks []gridTrack,
	items []gridItem,
	axisAvailableGridSpace AvailableSpace,
	innerNodeSize Size[Option[float32]],
	getTrackSizeEstimate func(track *gridTrack, parentSize Option[float32], tree layoutPartialTree) Option[float32],
) {
	// Sort items
	cmp := cmpByCrossFlexThenSpanThenStart(axis)
	sort.SliceStable(items, func(i, j int) bool {
		return cmp(&items[i], &items[j]) < 0
	})

	axisInnerNodeSize := innerNodeSize.GetAbstract(axis)
	var flexFactorSum float32
	for i := range axisTracks {
		flexFactorSum += axisTracks[i].flexFactor()
	}
	calc := tree.resolveCalcValue

	batcher := newItemBatcher(axis)
	for {
		batchIndices, isFlex := batcher.next(items)
		if batchIndices == nil {
			break
		}

		batchSpan := items[batchIndices[0]].span(axis)

		// Step 2: size tracks to fit non-spanning items
		if !isFlex && batchSpan == 1 {
			for _, idx := range batchIndices {
				item := &items[idx]
				trackIndex := int(item.placementIndexes(axis).Start) + 1

				// Compute available space and margins for this item
				availSpace := itemAvailableSpaceCached(item, axis, otherAxisTracks, innerNodeSize, getTrackSizeEstimate, tree)
				marginAxisSums := item.marginsAxisSumsWithBaselineShims(innerNodeSize.Width, tree)
				marginInAxis := marginAxisSums.GetAbstract(axis)

				track := &axisTracks[trackIndex]

				// Handle base sizes
				newBaseSize := track.baseSize
				if track.minTrackSizingFunction.IsMinContent() {
					contrib := itemMinContentContributionCached(item, axis, tree, availSpace, innerNodeSize) + marginInAxis
					newBaseSize = f32Max(newBaseSize, contrib)
				} else if track.minTrackSizingFunction.UsesPercentage() {
					if !axisInnerNodeSize.IsSome {
						contrib := itemMinContentContributionCached(item, axis, tree, availSpace, innerNodeSize) + marginInAxis
						newBaseSize = f32Max(newBaseSize, contrib)
					}
				} else if track.minTrackSizingFunction.IsMaxContent() {
					contrib := itemMaxContentContributionCached(item, axis, tree, availSpace, innerNodeSize) + marginInAxis
					newBaseSize = f32Max(newBaseSize, contrib)
				} else if track.minTrackSizingFunction.IsAuto() {
					switch {
					case (axisAvailableGridSpace.Tag == AvailableSpaceMinContent || axisAvailableGridSpace.Tag == AvailableSpaceMaxContent) &&
						!item.overflow.GetAbstract(axis).IsScrollContainer():
						axisMinimumSize := itemMinimumContributionCached(item, tree, axis, axisTracks, availSpace, innerNodeSize) + marginInAxis
						axisMinContentSize := itemMinContentContributionCached(item, axis, tree, availSpace, innerNodeSize) + marginInAxis
						limit := track.maxTrackSizingFunction.DefiniteLimit(axisInnerNodeSize, calc)
						val := axisMinContentSize
						if limit.IsSome && limit.Value < val {
							val = limit.Value
						}
						if axisMinimumSize > val {
							val = axisMinimumSize
						}
						newBaseSize = f32Max(newBaseSize, val)
					default:
						contrib := itemMinimumContributionCached(item, tree, axis, axisTracks, availSpace, innerNodeSize) + marginInAxis
						newBaseSize = f32Max(newBaseSize, contrib)
					}
				}
				// Length and calc tags - do nothing (not intrinsic)

				axisTracks[trackIndex].baseSize = newBaseSize

				// Handle growth limits
				if track.maxTrackSizingFunction.IsFitContent() {
					if !item.overflow.GetAbstract(axis).IsScrollContainer() {
						minContentContrib := itemMinContentContributionCached(item, axis, tree, availSpace, innerNodeSize) + marginInAxis
						axisTracks[trackIndex].growthLimitPlannedIncrease = f32Max(axisTracks[trackIndex].growthLimitPlannedIncrease, minContentContrib)
					}
					fitContentLimit := axisTracks[trackIndex].fitContentLimit(axisInnerNodeSize)
					maxContentContrib := itemMaxContentContributionCached(item, axis, tree, availSpace, innerNodeSize) + marginInAxis
					if maxContentContrib > fitContentLimit {
						maxContentContrib = fitContentLimit
					}
					axisTracks[trackIndex].growthLimitPlannedIncrease = f32Max(axisTracks[trackIndex].growthLimitPlannedIncrease, maxContentContrib)
				} else if track.maxTrackSizingFunction.IsMaxContentAlike() ||
					(track.maxTrackSizingFunction.UsesPercentage() && !axisInnerNodeSize.IsSome) {
					maxContrib := itemMaxContentContributionCached(item, axis, tree, availSpace, innerNodeSize) + marginInAxis
					axisTracks[trackIndex].growthLimitPlannedIncrease = f32Max(axisTracks[trackIndex].growthLimitPlannedIncrease, maxContrib)
				} else if track.maxTrackSizingFunction.IsIntrinsic() {
					minContrib := itemMinContentContributionCached(item, axis, tree, availSpace, innerNodeSize) + marginInAxis
					axisTracks[trackIndex].growthLimitPlannedIncrease = f32Max(axisTracks[trackIndex].growthLimitPlannedIncrease, minContrib)
				}
			}

			// Flush growth limit increases for step 2
			for i := range axisTracks {
				if axisTracks[i].growthLimitPlannedIncrease > 0.0 {
					if axisTracks[i].growthLimit == float32(math.Inf(1)) {
						axisTracks[i].growthLimit = axisTracks[i].growthLimitPlannedIncrease
					} else {
						axisTracks[i].growthLimit = f32Max(axisTracks[i].growthLimit, axisTracks[i].growthLimitPlannedIncrease)
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
		for _, idx := range batchIndices {
			item := &items[idx]
			if !item.crossesIntrinsicTrack(axis) {
				continue
			}
			availSpace := itemAvailableSpaceCached(item, axis, otherAxisTracks, innerNodeSize, getTrackSizeEstimate, tree)
			marginAxisSums := item.marginsAxisSumsWithBaselineShims(innerNodeSize.Width, tree)
			marginInAxis := marginAxisSums.GetAbstract(axis)

			var space float32
			switch {
			case (axisAvailableGridSpace.Tag == AvailableSpaceMinContent || axisAvailableGridSpace.Tag == AvailableSpaceMaxContent) &&
				!item.overflow.GetAbstract(axis).IsScrollContainer():
				axisMinimumSize := itemMinimumContributionCached(item, tree, axis, axisTracks, availSpace, innerNodeSize) + marginInAxis
				axisMinContentSize := itemMinContentContributionCached(item, axis, tree, availSpace, innerNodeSize) + marginInAxis
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
				space = itemMinimumContributionCached(item, tree, axis, axisTracks, availSpace, innerNodeSize) + marginInAxis
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
		for _, idx := range batchIndices {
			item := &items[idx]
			availSpace := itemAvailableSpaceCached(item, axis, otherAxisTracks, innerNodeSize, getTrackSizeEstimate, tree)
			marginAxisSums := item.marginsAxisSumsWithBaselineShims(innerNodeSize.Width, tree)
			space := itemMinContentContributionCached(item, axis, tree, availSpace, innerNodeSize) + marginAxisSums.GetAbstract(axis)
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
		if axisAvailableGridSpace.Tag == AvailableSpaceMaxContent {
			hasAutoMinTSF := func(t *gridTrack) bool {
				return t.minTrackSizingFunction.IsAuto() && !t.maxTrackSizingFunction.IsMinContent()
			}
			hasMaxContentMinTSF := func(t *gridTrack) bool { return t.minTrackSizingFunction.IsMaxContent() }

			for _, idx := range batchIndices {
				item := &items[idx]
				availSpace := itemAvailableSpaceCached(item, axis, otherAxisTracks, innerNodeSize, getTrackSizeEstimate, tree)
				marginAxisSums := item.marginsAxisSumsWithBaselineShims(innerNodeSize.Width, tree)
				marginInAxis := marginAxisSums.GetAbstract(axis)
				axisMaxContentSize := itemMaxContentContributionCached(item, axis, tree, availSpace, innerNodeSize) + marginInAxis
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
		for _, idx := range batchIndices {
			item := &items[idx]
			availSpace := itemAvailableSpaceCached(item, axis, otherAxisTracks, innerNodeSize, getTrackSizeEstimate, tree)
			marginAxisSums := item.marginsAxisSumsWithBaselineShims(innerNodeSize.Width, tree)
			space := itemMaxContentContributionCached(item, axis, tree, availSpace, innerNodeSize) + marginAxisSums.GetAbstract(axis)
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
			for _, idx := range batchIndices {
				item := &items[idx]
				availSpace := itemAvailableSpaceCached(item, axis, otherAxisTracks, innerNodeSize, getTrackSizeEstimate, tree)
				marginAxisSums := item.marginsAxisSumsWithBaselineShims(innerNodeSize.Width, tree)
				space := itemMinContentContributionCached(item, axis, tree, availSpace, innerNodeSize) + marginAxisSums.GetAbstract(axis)
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
			for _, idx := range batchIndices {
				item := &items[idx]
				availSpace := itemAvailableSpaceCached(item, axis, otherAxisTracks, innerNodeSize, getTrackSizeEstimate, tree)
				marginAxisSums := item.marginsAxisSumsWithBaselineShims(innerNodeSize.Width, tree)
				space := itemMaxContentContributionCached(item, axis, tree, availSpace, innerNodeSize) + marginAxisSums.GetAbstract(axis)
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
	extraSpace := f32Max(0.0, space-trackSizes)

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
	axisInnerNodeSize Option[float32],
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
	extraSpace := f32Max(0.0, space-trackSizes)

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
func maximiseTracks(axisTracks []gridTrack, axisInnerNodeSize Option[float32], axisAvailableGridSpace AvailableSpace) {
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
	tree layoutPartialTree,
	axis AbstractAxis,
	axisTracks []gridTrack,
	items []gridItem,
	axisMinSize Option[float32],
	axisMaxSize Option[float32],
	axisAvailableSpaceForExpansion AvailableSpace,
	innerNodeSize Size[Option[float32]],
) {
	var flexFraction float32
	switch axisAvailableSpaceForExpansion.Tag {
	case AvailableSpaceDefinite:
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
	case AvailableSpaceMinContent:
		flexFraction = 0.0
	case AvailableSpaceMaxContent:
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
				maxContentContrib := items[i].maxContentContributionCached(axis, tree, SizeOptionNone, innerNodeSize)
				fr := findSizeOfFr(itemTracks, maxContentContrib)
				if fr > maxItemFlex {
					maxItemFlex = fr
				}
			}
		}

		flexFraction = f32Max(maxTrackFlex, maxItemFlex)

		// Check min/max size constraints
		var hypotheticalGridSize float32
		for i := range axisTracks {
			if axisTracks[i].maxTrackSizingFunction.IsFr() {
				trackFF := axisTracks[i].maxTrackSizingFunction.Value
				hypotheticalGridSize += f32Max(axisTracks[i].baseSize, trackFF*flexFraction)
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
			axisTracks[i].baseSize = f32Max(axisTracks[i].baseSize, trackFF*flexFraction)
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
		flexFactor := f32Max(naiveFlexFactorSum, 1.0)

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
	axisMinSize Option[float32],
	axisAvailableSpaceForExpansion AvailableSpace,
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

		iterIncrease := f32Min(minIncreaseLimit, remaining/proportionSum)

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
	axis AbstractAxis,
	otherAxisTracks []gridTrack,
	innerNodeSize Size[Option[float32]],
	getTrackSizeEstimate func(track *gridTrack, parentSize Option[float32], tree layoutPartialTree) Option[float32],
	tree layoutPartialTree,
) Size[Option[float32]] {
	return item.availableSpaceCached(axis, otherAxisTracks, innerNodeSize.GetAbstract(axis.Other()),
		func(track *gridTrack, availableSpace Option[float32]) Option[float32] {
			return getTrackSizeEstimate(track, availableSpace, tree)
		})
}

func itemMinContentContributionCached(
	item *gridItem,
	axis AbstractAxis,
	tree layoutPartialTree,
	availSpace Size[Option[float32]],
	innerNodeSize Size[Option[float32]],
) float32 {
	return item.minContentContributionCached(axis, tree, availSpace, innerNodeSize)
}

func itemMaxContentContributionCached(
	item *gridItem,
	axis AbstractAxis,
	tree layoutPartialTree,
	availSpace Size[Option[float32]],
	innerNodeSize Size[Option[float32]],
) float32 {
	return item.maxContentContributionCached(axis, tree, availSpace, innerNodeSize)
}

func itemMinimumContributionCached(
	item *gridItem,
	tree layoutPartialTree,
	axis AbstractAxis,
	axisTracks []gridTrack,
	availSpace Size[Option[float32]],
	innerNodeSize Size[Option[float32]],
) float32 {
	return item.minimumContributionCached(tree, axis, axisTracks, availSpace, innerNodeSize)
}
