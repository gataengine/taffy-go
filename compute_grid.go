// Package taffy implements the CSS Grid layout algorithm.
// This file is a port of compute/grid/mod.rs from the Taffy Rust layout engine.
package taffy

import "sort"

// computeGridLayout is the main entry point for CSS Grid layout.
// It consists of several phases: resolving the explicit grid, placing items,
// track sizing, and alignment/final item placement.
func computeGridLayout(tree layoutGridContainer, node NodeId, inputs LayoutInput) LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	parentSize := inputs.ParentSize
	availableSpace := inputs.AvailableSpace
	runMode := inputs.RunMode

	style := tree.getGridContainerStyle(node)
	direction := style.direction()
	calc := tree.resolveCalcValue

	// 1. Compute "available grid space"
	aspectRatio := style.aspectRatio()
	padding := resolveRectLPWidth(style.padding(), parentSize.Width, calc)
	border := resolveRectLPWidth(style.border(), parentSize.Width, calc)
	paddingBorder := RectAddF32(padding, border)
	paddingBorderSize := RectSumAxes(paddingBorder)
	boxSizingAdjustment := SizeZeroF32
	if style.boxSizing() == BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSize
	}

	minSize := SizeMaybeAdd(
		SizeOptionMaybeApplyAspectRatio(
			MaybeResolveSizeDimOption(style.minSize(), parentSize, calc),
			aspectRatio,
		),
		Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)},
	)
	maxSize := SizeMaybeAdd(
		SizeOptionMaybeApplyAspectRatio(
			MaybeResolveSizeDimOption(style.maxSize(), parentSize, calc),
			aspectRatio,
		),
		Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)},
	)
	preferredSize := SizeOptionNone
	if inputs.SizingMode == SizingModeInherentSize {
		preferredSize = SizeMaybeAdd(
			SizeOptionMaybeApplyAspectRatio(
				MaybeResolveSizeDimOption(style.size(), parentSize, calc),
				aspectRatio,
			),
			Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)},
		)
	}

	// Scrollbar gutters
	overflowStyle := style.overflow()
	scrollbarGutter := Size[float32]{
		Width: func() float32 {
			if overflowStyle.Y == OverflowScroll {
				return style.scrollbarWidth()
			}
			return 0.0
		}(),
		Height: func() float32 {
			if overflowStyle.X == OverflowScroll {
				return style.scrollbarWidth()
			}
			return 0.0
		}(),
	}
	contentBoxInset := paddingBorder
	contentBoxInset.Bottom += scrollbarGutter.Height
	switch direction {
	case DirectionLTR:
		contentBoxInset.Right += scrollbarGutter.Width
	case DirectionRTL:
		contentBoxInset.Left += scrollbarGutter.Width
	}

	alignContent := style.alignContent().UnwrapOr(AlignContentStretch)
	justifyContent := style.justifyContent().UnwrapOr(AlignContentStretch)
	alignItems := style.alignItems()
	justifyItemsStyle := style.justifyItems()

	constrainedAS := computeConstrainedAvailableSpace(knownDimensions, preferredSize, availableSpace, minSize, maxSize, paddingBorderSize)

	availableGridSpace := Size[AvailableSpace]{
		Width:  constrainedAS.Width.MapDefiniteValue(func(space float32) float32 { return space - RectHorizontalAxisSum(contentBoxInset) }),
		Height: constrainedAS.Height.MapDefiniteValue(func(space float32) float32 { return space - RectVerticalAxisSum(contentBoxInset) }),
	}

	outerNodeSize := SizeMaybeMax(
		SizeMaybeClamp(SizeOptionOr(knownDimensions, preferredSize), minSize, maxSize),
		Size[Option[float32]]{Width: Some(paddingBorderSize.Width), Height: Some(paddingBorderSize.Height)},
	)
	innerNodeSize := Size[Option[float32]]{
		Width:  MaybeSubOption(outerNodeSize.Width, Some(RectHorizontalAxisSum(contentBoxInset))),
		Height: MaybeSubOption(outerNodeSize.Height, Some(RectVerticalAxisSum(contentBoxInset))),
	}

	if runMode == RunModeComputeSize && outerNodeSize.Width.IsSome && outerNodeSize.Height.IsSome {
		return LayoutOutputFromOuterSize(Size[float32]{Width: outerNodeSize.Width.Value, Height: outerNodeSize.Height.Value})
	}

	// 2. Resolve the explicit grid
	autoFitContainerSize := SizeMaybeSub(
		SizeMaybeMax(
			SizeMaybeClamp(
				SizeOptionOr(SizeOptionOr(outerNodeSize, maxSize), minSize),
				minSize, maxSize,
			),
			Size[Option[float32]]{Width: Some(paddingBorderSize.Width), Height: Some(paddingBorderSize.Height)},
		),
		Size[Option[float32]]{
			Width:  Some(RectHorizontalAxisSum(contentBoxInset)),
			Height: Some(RectVerticalAxisSum(contentBoxInset)),
		},
	)

	autoRepeatFitStrategy := Size[autoRepeatStrategy]{
		Width: func() autoRepeatStrategy {
			if SizeOptionOr(outerNodeSize, maxSize).Width.IsSome {
				return autoRepeatMaxRepsNoOverflow
			}
			return autoRepeatMinRepsOverflow
		}(),
		Height: func() autoRepeatStrategy {
			if SizeOptionOr(outerNodeSize, maxSize).Height.IsSome {
				return autoRepeatMaxRepsNoOverflow
			}
			return autoRepeatMinRepsOverflow
		}(),
	}

	colAutoRepetitionCount, gridTemplateColCount := computeExplicitGridSizeInAxis(
		style, autoFitContainerSize.Width, autoRepeatFitStrategy.Width, calc, AbsoluteAxisHorizontal,
	)
	rowAutoRepetitionCount, gridTemplateRowCount := computeExplicitGridSizeInAxis(
		style, autoFitContainerSize.Height, autoRepeatFitStrategy.Height, calc, AbsoluteAxisVertical,
	)

	nameResolver := newNamedLineResolver(
		style,
		style.GridTemplateColumnNames,
		style.GridTemplateRowNames,
		style.gridTemplateAreas(),
		colAutoRepetitionCount,
		rowAutoRepetitionCount,
	)

	explicitColCount := gridTemplateColCount
	if nameResolver.areaColumnCountVal() > explicitColCount {
		explicitColCount = nameResolver.areaColumnCountVal()
	}
	explicitRowCount := gridTemplateRowCount
	if nameResolver.areaRowCountVal() > explicitRowCount {
		explicitRowCount = nameResolver.areaRowCountVal()
	}

	nameResolver.setExplicitColumnCount(explicitColCount)
	nameResolver.setExplicitRowCount(explicitRowCount)

	// 3. Implicit Grid: Estimate Track Counts
	childStyleSlice := make([]*Style, 0, tree.childCount(node))
	for _, childNode := range tree.childIds(node) {
		childStyleSlice = append(childStyleSlice, tree.getGridChildStyle(childNode))
	}
	estColCounts, estRowCounts := computeGridSizeEstimate(explicitColCount, explicitRowCount, direction, childStyleSlice)

	// 4. Grid Item Placement
	items := make([]gridItem, 0, tree.childCount(node))
	cellMatrix := newCellOccupancyMatrix(estColCounts, estRowCounts)

	inFlowChildrenIter := func() []gridPlacementChild {
		var result []gridPlacementChild
		for index, childNode := range tree.childIds(node) {
			childStyle := tree.getGridChildStyle(childNode)
			if childStyle.boxGenerationMode() == BoxGenerationModeNone || childStyle.position() == PositionAbsolute {
				continue
			}
			result = append(result, gridPlacementChild{index: index, node: childNode, style: childStyle})
		}
		return result
	}

	placeGridItems(
		&cellMatrix, &items, inFlowChildrenIter,
		direction, style.gridAutoFlow(),
		alignItems.UnwrapOr(AlignItemsStretch),
		justifyItemsStyle.UnwrapOr(AlignItemsStretch),
		&nameResolver,
	)

	// Extract track counts
	finalColCounts := *cellMatrix.trackCounts(AbsoluteAxisHorizontal)
	finalRowCounts := *cellMatrix.trackCounts(AbsoluteAxisVertical)

	// 5. Initialize Tracks
	var columns []gridTrack
	var rows []gridTrack
	columnTrackCountsForInit := finalColCounts
	if direction.IsRTL() && finalColCounts.Explicit <= 1 {
		columnTrackCountsForInit.NegativeImplicit = finalColCounts.PositiveImplicit
		columnTrackCountsForInit.PositiveImplicit = finalColCounts.NegativeImplicit
	}
	initializeGridTracks(&columns, columnTrackCountsForInit, style, AbsoluteAxisHorizontal,
		func(columnIndex int) bool {
			occupancyIndex := columnIndex
			if direction.IsRTL() {
				occupancyIndex = rtlColumnOccupancyIndexForInit(columnIndex, finalColCounts)
			}
			return cellMatrix.columnIsOccupied(occupancyIndex)
		})
	initializeGridTracks(&rows, finalRowCounts, style, AbsoluteAxisVertical,
		func(rowIndex int) bool {
			return cellMatrix.rowIsOccupied(rowIndex)
		})
	if direction.IsRTL() {
		reverseNonGutterTracks(columns, finalColCounts)
	}

	// 6. Track Sizing
	resolveItemTrackIndexes(items, finalColCounts, finalRowCounts)
	determineIfItemCrossesFlexibleOrIntrinsicTracks(items, columns, rows)

	hasBaselineAlignedItem := false
	for i := range items {
		if items[i].alignSelf == AlignItemsBaseline {
			hasBaselineAlignedItem = true
			break
		}
	}

	// Run track sizing for Inline axis
	trackSizingAlgorithm(
		tree, AbstractAxisInline,
		minSize.GetAbstract(AbstractAxisInline),
		maxSize.GetAbstract(AbstractAxisInline),
		justifyContent, alignContent,
		availableGridSpace, innerNodeSize,
		columns, rows, items,
		func(track *gridTrack, parentSize Option[float32], t layoutPartialTree) Option[float32] {
			return track.maxTrackSizingFunction.DefiniteValue(parentSize, t.resolveCalcValue)
		},
		hasBaselineAlignedItem,
	)
	var initialColumnSum float32
	for i := range columns {
		initialColumnSum += columns[i].baseSize
	}
	if !innerNodeSize.Width.IsSome {
		innerNodeSize.Width = Some(initialColumnSum)
	}

	// Clear caches
	for i := range items {
		items[i].availableSpaceCache = nil
	}

	// Run track sizing for Block axis
	trackSizingAlgorithm(
		tree, AbstractAxisBlock,
		minSize.GetAbstract(AbstractAxisBlock),
		maxSize.GetAbstract(AbstractAxisBlock),
		alignContent, justifyContent,
		availableGridSpace, innerNodeSize,
		rows, columns, items,
		func(track *gridTrack, _ Option[float32], _ layoutPartialTree) Option[float32] {
			return Some(track.baseSize)
		},
		false,
	)
	var initialRowSum float32
	for i := range rows {
		initialRowSum += rows[i].baseSize
	}
	if !innerNodeSize.Height.IsSome {
		innerNodeSize.Height = Some(initialRowSum)
	}

	// 6. Compute container size
	resolvedStyleSize := SizeOptionOr(knownDimensions, preferredSize)
	containerBorderBox := Size[float32]{
		Width: f32Max(
			SizeMaybeClamp(
				Size[Option[float32]]{
					Width: func() Option[float32] {
						if resolvedStyleSize.Width.IsSome {
							return resolvedStyleSize.Width
						}
						return Some(initialColumnSum + RectHorizontalAxisSum(contentBoxInset))
					}(),
					Height: None[float32](),
				},
				minSize, maxSize,
			).Width.UnwrapOr(0.0),
			paddingBorderSize.Width,
		),
		Height: f32Max(
			SizeMaybeClamp(
				Size[Option[float32]]{
					Width:  None[float32](),
					Height: func() Option[float32] {
						if resolvedStyleSize.Height.IsSome {
							return resolvedStyleSize.Height
						}
						return Some(initialRowSum + RectVerticalAxisSum(contentBoxInset))
					}(),
				},
				minSize, maxSize,
			).Height.UnwrapOr(0.0),
			paddingBorderSize.Height,
		),
	}
	containerContentBox := Size[float32]{
		Width:  f32Max(0.0, containerBorderBox.Width-RectHorizontalAxisSum(contentBoxInset)),
		Height: f32Max(0.0, containerBorderBox.Height-RectVerticalAxisSum(contentBoxInset)),
	}

	if runMode == RunModeComputeSize {
		return LayoutOutputFromOuterSize(containerBorderBox)
	}

	// 7. Resolve percentage track base sizes
	if !availableGridSpace.Width.IsDefinite() {
		for i := range columns {
			min := columns[i].minTrackSizingFunction.ResolvedPercentageSize(containerContentBox.Width, calc)
			max := columns[i].maxTrackSizingFunction.ResolvedPercentageSize(containerContentBox.Width, calc)
			columns[i].baseSize = MaybeClampF32(columns[i].baseSize, min, max)
		}
	}
	if !availableGridSpace.Height.IsDefinite() {
		for i := range rows {
			min := rows[i].minTrackSizingFunction.ResolvedPercentageSize(containerContentBox.Height, calc)
			max := rows[i].maxTrackSizingFunction.ResolvedPercentageSize(containerContentBox.Height, calc)
			rows[i].baseSize = MaybeClampF32(rows[i].baseSize, min, max)
		}
	}

	// Check if column sizing needs to be re-run
	var rerunColumnSizing bool
	hasPercentageColumn := false
	for i := range columns {
		if columns[i].usesPercentage() {
			hasPercentageColumn = true
			break
		}
	}
	parentWidthIndefinite := !availableSpace.Width.IsDefinite()
	rerunColumnSizing = parentWidthIndefinite && hasPercentageColumn

	if !rerunColumnSizing {
		// Check if min content contribution changed
		for i := range items {
			if !items[i].crossesIntrinsicColumn {
				continue
			}
			availSpace := items[i].availableSpace(
				AbstractAxisInline, rows, innerNodeSize.Height,
				func(track *gridTrack, _ Option[float32]) Option[float32] { return Some(track.baseSize) },
			)
			newMinContentContrib := items[i].minContentContribution(AbstractAxisInline, tree, availSpace, innerNodeSize)
			if items[i].minContentContributionCache.Width.IsSome &&
				items[i].minContentContributionCache.Width.Value != newMinContentContrib {
				rerunColumnSizing = true
			}
			items[i].availableSpaceCache = &availSpace
			items[i].minContentContributionCache.Width = Some(newMinContentContrib)
			items[i].maxContentContributionCache.Width = None[float32]()
			items[i].minimumContributionCache.Width = None[float32]()
		}
	} else {
		for i := range items {
			items[i].availableSpaceCache = nil
			items[i].minContentContributionCache.Width = None[float32]()
			items[i].maxContentContributionCache.Width = None[float32]()
			items[i].minimumContributionCache.Width = None[float32]()
		}
	}

	if rerunColumnSizing {
		trackSizingAlgorithm(
			tree, AbstractAxisInline,
			minSize.GetAbstract(AbstractAxisInline),
			maxSize.GetAbstract(AbstractAxisInline),
			justifyContent, alignContent,
			availableGridSpace, innerNodeSize,
			columns, rows, items,
			func(track *gridTrack, _ Option[float32], _ layoutPartialTree) Option[float32] {
				return Some(track.baseSize)
			},
			hasBaselineAlignedItem,
		)

		// Check if row sizing needs re-run
		var rerunRowSizing bool
		hasPercentageRow := false
		for i := range rows {
			if rows[i].usesPercentage() {
				hasPercentageRow = true
				break
			}
		}
		parentHeightIndefinite := !availableSpace.Height.IsDefinite()
		rerunRowSizing = parentHeightIndefinite && hasPercentageRow

		if !rerunRowSizing {
			for i := range items {
				if !items[i].crossesIntrinsicColumn {
					continue
				}
				availSpace := items[i].availableSpace(
					AbstractAxisBlock, columns, innerNodeSize.Width,
					func(track *gridTrack, _ Option[float32]) Option[float32] { return Some(track.baseSize) },
				)
				newMinContentContrib := items[i].minContentContribution(AbstractAxisBlock, tree, availSpace, innerNodeSize)
				if items[i].minContentContributionCache.Height.IsSome &&
					items[i].minContentContributionCache.Height.Value != newMinContentContrib {
					rerunRowSizing = true
				}
				items[i].availableSpaceCache = &availSpace
				items[i].minContentContributionCache.Height = Some(newMinContentContrib)
				items[i].maxContentContributionCache.Height = None[float32]()
				items[i].minimumContributionCache.Height = None[float32]()
			}
		} else {
			for i := range items {
				items[i].availableSpaceCache = nil
				items[i].minContentContributionCache.Height = None[float32]()
				items[i].maxContentContributionCache.Height = None[float32]()
				items[i].minimumContributionCache.Height = None[float32]()
			}
		}

		if rerunRowSizing {
			trackSizingAlgorithm(
				tree, AbstractAxisBlock,
				minSize.GetAbstract(AbstractAxisBlock),
				maxSize.GetAbstract(AbstractAxisBlock),
				alignContent, justifyContent,
				availableGridSpace, innerNodeSize,
				rows, columns, items,
				func(track *gridTrack, _ Option[float32], _ layoutPartialTree) Option[float32] {
					return Some(track.baseSize)
				},
				false,
			)
		}
	}

	// 8. Track Alignment
	inlineSizeWithoutScrollbar := f32Max(containerBorderBox.Width-paddingBorderSize.Width, 0.0)
	inlineScrollbarGutterForAlignment := f32Min(scrollbarGutter.Width, inlineSizeWithoutScrollbar)
	alignGridTracks(
		containerContentBox.Width,
		Line[float32]{
			Start: padding.Left + func() float32 {
				if direction.IsRTL() {
					return inlineScrollbarGutterForAlignment
				}
				return 0.0
			}(),
			End: padding.Right + func() float32 {
				if direction.IsRTL() {
					return 0.0
				}
				return inlineScrollbarGutterForAlignment
			}(),
		},
		Line[float32]{Start: border.Left, End: border.Right},
		columns, justifyContent, direction.IsRTL(),
	)
	alignGridTracks(
		containerContentBox.Height,
		Line[float32]{Start: padding.Top, End: padding.Bottom},
		Line[float32]{Start: border.Top, End: border.Bottom},
		rows, alignContent, false,
	)

	// 9. Size, Align, and Position Grid Items
	itemContentSizeContribution := SizeZeroF32

	// Sort items back into original order
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].sourceOrder < items[j].sourceOrder
	})

	containerAlignmentStyles := inBothAbsAxis[Option[AlignItems]]{
		horizontal: justifyItemsStyle,
		vertical:   alignItems,
	}

	// Position in-flow children
	for index := range items {
		item := &items[index]
		gridArea := Rect[float32]{
			Top:    rows[item.rowIndexes.Start+1].offset,
			Bottom: rows[item.rowIndexes.End].offset,
			Left:   columns[item.columnIndexes.Start+1].offset,
			Right:  columns[item.columnIndexes.End].offset,
		}
		contentSizeContrib, yPos, h := alignAndPositionItem(
			tree, item.node, uint32(index), gridArea,
			containerAlignmentStyles, item.baselineShim, direction,
		)
		item.yPosition = yPos
		item.height = h
		itemContentSizeContribution = SizeF32Max(itemContentSizeContribution, contentSizeContrib)
	}

	// Position hidden and absolutely positioned children
	order := uint32(len(items))
	childCount := tree.childCount(node)
	for i := 0; i < childCount; i++ {
		child := tree.getChildId(node, i)
		childStyle := tree.getGridChildStyle(child)

		// Position hidden child
		if childStyle.boxGenerationMode() == BoxGenerationModeNone {
			tree.setUnroundedLayout(child, &Layout{Order: order})
			performChildLayout(tree, child, SizeOptionNone, SizeOptionNone,
				Size[AvailableSpace]{Width: AvailableSpaceMaxContentValue, Height: AvailableSpaceMaxContentValue}, SizingModeInherentSize, LineFalseBool)
			order++
			continue
		}

		// Position absolutely positioned child
		if childStyle.position() == PositionAbsolute {
			maybeColIndexes := ResolveAbsolutelyPositionedGridTracks(
				IntoOriginZeroNonNamed(nameResolver.resolveColumnNames(childStyle.gridColumn()), finalColCounts.Explicit),
			)
			// Apply direction mirroring
			if direction.IsRTL() {
				maybeColIndexes = Line[Option[OriginZeroLine]]{
					Start: func() Option[OriginZeroLine] {
						if maybeColIndexes.Start.IsSome {
							mirrored := OriginZeroLine(int16(finalColCounts.Explicit) - int16(maybeColIndexes.Start.Value))
							idx, ok := mirrored.TryIntoTrackVecIndex(finalColCounts)
							if ok {
								return Some(OriginZeroLine(idx))
							}
						}
						return None[OriginZeroLine]()
					}(),
					End: func() Option[OriginZeroLine] {
						if maybeColIndexes.End.IsSome {
							mirrored := OriginZeroLine(int16(finalColCounts.Explicit) - int16(maybeColIndexes.End.Value))
							idx, ok := mirrored.TryIntoTrackVecIndex(finalColCounts)
							if ok {
								return Some(OriginZeroLine(idx))
							}
						}
						return None[OriginZeroLine]()
					}(),
				}
				maybeColIndexes = Line[Option[OriginZeroLine]]{Start: maybeColIndexes.End, End: maybeColIndexes.Start}
			} else {
				maybeColIndexes = Line[Option[OriginZeroLine]]{
					Start: func() Option[OriginZeroLine] {
						if maybeColIndexes.Start.IsSome {
							idx, ok := maybeColIndexes.Start.Value.TryIntoTrackVecIndex(finalColCounts)
							if ok {
								return Some(OriginZeroLine(idx))
							}
						}
						return None[OriginZeroLine]()
					}(),
					End: func() Option[OriginZeroLine] {
						if maybeColIndexes.End.IsSome {
							idx, ok := maybeColIndexes.End.Value.TryIntoTrackVecIndex(finalColCounts)
							if ok {
								return Some(OriginZeroLine(idx))
							}
						}
						return None[OriginZeroLine]()
					}(),
				}
			}

			maybeRowIndexes := ResolveAbsolutelyPositionedGridTracks(
				IntoOriginZeroNonNamed(nameResolver.resolveRowNames(childStyle.gridRow()), finalRowCounts.Explicit),
			)
			maybeRowIndexes = Line[Option[OriginZeroLine]]{
				Start: func() Option[OriginZeroLine] {
					if maybeRowIndexes.Start.IsSome {
						idx, ok := maybeRowIndexes.Start.Value.TryIntoTrackVecIndex(finalRowCounts)
						if ok {
							return Some(OriginZeroLine(idx))
						}
					}
					return None[OriginZeroLine]()
				}(),
				End: func() Option[OriginZeroLine] {
					if maybeRowIndexes.End.IsSome {
						idx, ok := maybeRowIndexes.End.Value.TryIntoTrackVecIndex(finalRowCounts)
						if ok {
							return Some(OriginZeroLine(idx))
						}
					}
					return None[OriginZeroLine]()
				}(),
			}

			gridArea := Rect[float32]{
				Top: func() float32 {
					if maybeRowIndexes.Start.IsSome {
						return rows[int(maybeRowIndexes.Start.Value)].offset
					}
					return border.Top
				}(),
				Bottom: func() float32 {
					if maybeRowIndexes.End.IsSome {
						return rows[int(maybeRowIndexes.End.Value)].offset
					}
					return containerBorderBox.Height - border.Bottom - scrollbarGutter.Height
				}(),
				Left: func() float32 {
					if maybeColIndexes.Start.IsSome {
						return columns[int(maybeColIndexes.Start.Value)].offset
					}
					if direction.IsRTL() {
						return border.Left + scrollbarGutter.Width
					}
					return border.Left
				}(),
				Right: func() float32 {
					if maybeColIndexes.End.IsSome {
						return columns[int(maybeColIndexes.End.Value)].offset
					}
					if direction.IsRTL() {
						return containerBorderBox.Width - border.Right
					}
					return containerBorderBox.Width - border.Right - scrollbarGutter.Width
				}(),
			}

			contentSizeContrib, _, _ := alignAndPositionItem(
				tree, child, order, gridArea, containerAlignmentStyles, 0.0, direction,
			)
			itemContentSizeContribution = SizeF32Max(itemContentSizeContribution, contentSizeContrib)
			order++
		}
	}

	// If there are no items, return just the container size
	if len(items) == 0 {
		return LayoutOutputFromOuterSize(containerBorderBox)
	}

	// Determine grid container baseline
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].rowIndexes.Start < items[j].rowIndexes.Start
	})

	firstRow := items[0].rowIndexes.Start
	firstRowEnd := len(items)
	for i := range items {
		if items[i].rowIndexes.Start != firstRow {
			firstRowEnd = i
			break
		}
	}
	firstRowItems := items[:firstRowEnd]

	rowHasBaselineItem := false
	for i := range firstRowItems {
		if firstRowItems[i].alignSelf == AlignItemsBaseline {
			rowHasBaselineItem = true
			break
		}
	}

	var baselineItem *gridItem
	if rowHasBaselineItem {
		for i := range firstRowItems {
			if firstRowItems[i].alignSelf == AlignItemsBaseline {
				baselineItem = &firstRowItems[i]
				break
			}
		}
	} else {
		baselineItem = &firstRowItems[0]
	}

	gridContainerBaseline := baselineItem.yPosition + baselineItem.baseline.UnwrapOr(baselineItem.height)

	return gridLayoutOutputFromSizesAndBaselines(
		containerBorderBox,
		itemContentSizeContribution,
		Point[Option[float32]]{X: None[float32](), Y: Some(gridContainerBaseline)},
	)
}

// reverseNonGutterTracks reverses only non-gutter column tracks in-place for RTL.
func reverseNonGutterTracks(tracks []gridTrack, trackCounts TrackCounts) {
	if trackCounts.Explicit <= 1 {
		const minTrackVecLenToReverse = 5
		if len(tracks) < minTrackVecLenToReverse {
			return
		}
		left := 1
		right := len(tracks) - 2
		for left < right {
			tracks[left], tracks[right] = tracks[right], tracks[left]
			left += 2
			if right < 2 {
				break
			}
			right -= 2
		}
		return
	}

	explicitTrackCount := int(trackCounts.Explicit)
	if explicitTrackCount < 2 {
		return
	}

	left := int(trackCounts.NegativeImplicit)
	right := left + explicitTrackCount - 1
	for left < right {
		tracks[(2*left)+1], tracks[(2*right)+1] = tracks[(2*right)+1], tracks[(2*left)+1]
		left++
		if right == 0 {
			break
		}
		right--
	}
}

// rtlColumnOccupancyIndexForInit maps column indexes to occupancy-matrix indexes for auto-fit collapsing in RTL.
func rtlColumnOccupancyIndexForInit(columnIndex int, trackCounts TrackCounts) int {
	if trackCounts.Explicit <= 1 {
		return trackCounts.Len() - columnIndex - 1
	}

	explicitStart := int(trackCounts.NegativeImplicit)
	explicitEnd := explicitStart + int(trackCounts.Explicit)
	if columnIndex >= explicitStart && columnIndex < explicitEnd {
		return explicitStart + (explicitEnd - columnIndex - 1)
	}
	return columnIndex
}

// computeConstrainedAvailableSpace computes the constrained available space for the grid.
func computeConstrainedAvailableSpace(
	knownDimensions, preferredSize Size[Option[float32]],
	availableSpace Size[AvailableSpace],
	minSize, maxSize Size[Option[float32]],
	paddingBorderSize Size[float32],
) Size[AvailableSpace] {
	kOrP := SizeOptionOr(knownDimensions, preferredSize)
	result := Size[AvailableSpace]{
		Width: func() AvailableSpace {
			if kOrP.Width.IsSome {
				return AvailableSpaceFromDefinite(kOrP.Width.Value)
			}
			return availableSpace.Width
		}(),
		Height: func() AvailableSpace {
			if kOrP.Height.IsSome {
				return AvailableSpaceFromDefinite(kOrP.Height.Value)
			}
			return availableSpace.Height
		}(),
	}

	// Clamp by min/max
	result.Width = clampAvailableSpace(result.Width, minSize.Width, maxSize.Width)
	result.Height = clampAvailableSpace(result.Height, minSize.Height, maxSize.Height)

	// Max with padding border size
	result.Width = maxAvailableSpace(result.Width, paddingBorderSize.Width)
	result.Height = maxAvailableSpace(result.Height, paddingBorderSize.Height)

	return result
}

func clampAvailableSpace(as AvailableSpace, min, max Option[float32]) AvailableSpace {
	if as.Tag != AvailableSpaceDefinite {
		return as
	}
	val := as.Value
	if min.IsSome && val < min.Value {
		val = min.Value
	}
	if max.IsSome && val > max.Value {
		val = max.Value
	}
	return AvailableSpaceFromDefinite(val)
}

func maxAvailableSpace(as AvailableSpace, val float32) AvailableSpace {
	if as.Tag != AvailableSpaceDefinite {
		return as
	}
	if as.Value < val {
		return AvailableSpaceFromDefinite(val)
	}
	return as
}

// gridLayoutOutputFromSizesAndBaselines creates a LayoutOutput from sizes and baselines.
func gridLayoutOutputFromSizesAndBaselines(size Size[float32], contentSize Size[float32], baselines Point[Option[float32]]) LayoutOutput {
	return LayoutOutput{
		Size:           size,
		ContentSize:    contentSize,
		FirstBaselines: baselines,
	}
}
