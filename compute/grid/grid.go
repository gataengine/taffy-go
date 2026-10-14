// Package grid implements the CSS Grid layout algorithm.
// This file is a port of compute/grid/mod.rs from the Taffy Rust layout engine.
package grid

import (
	"cmp"
	"slices"

	"github.com/gataengine/taffy-go/geometry"
	"github.com/gataengine/taffy-go/lib"
	"github.com/gataengine/taffy-go/style"
	"github.com/gataengine/taffy-go/tree"
)

// ComputeGridLayout is the main entry point for CSS Grid layout.
// It consists of several phases: resolving the explicit grid, placing items,
// track sizing, and alignment/final item placement.
func ComputeGridLayout(t tree.LayoutGridContainer, node tree.NodeId, inputs tree.LayoutInput) tree.LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	parentSize := inputs.ParentSize
	availableSpace := inputs.AvailableSpace
	runMode := inputs.RunMode

	containerStyle := t.GetGridContainerStyle(node)
	coreStyle := t.GetCoreContainerStyle(node)
	direction := coreStyle.GetDirection()
	calc := func(id uint64, basis float32) float32 {
		return t.ResolveCalcValue(id, basis)
	}

	scratch := gridScratchPool.Get().(*gridScratch)
	scratch.reset()
	defer gridScratchPool.Put(scratch)

	// 1. Compute "available grid space"
	aspectRatio := coreStyle.GetAspectRatio()
	padding := style.ResolveRectLPWidth(coreStyle.GetPadding(), parentSize.Width, calc)
	border := style.ResolveRectLPWidth(coreStyle.GetBorder(), parentSize.Width, calc)
	paddingBorder := geometry.RectAddF32(padding, border)
	paddingBorderSize := geometry.RectSumAxes(paddingBorder)
	boxSizingAdjustment := geometry.SizeZeroF32
	if coreStyle.GetBoxSizing() == style.BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSize
	}

	minSize := geometry.SizeMaybeAdd(
		geometry.SizeOptionMaybeApplyAspectRatio(
			style.MaybeResolveSizeDimOption(coreStyle.GetMinSize(), parentSize, calc),
			aspectRatio,
		),
		geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)},
	)
	maxSize := geometry.SizeMaybeAdd(
		geometry.SizeOptionMaybeApplyAspectRatio(
			style.MaybeResolveSizeDimOption(coreStyle.GetMaxSize(), parentSize, calc),
			aspectRatio,
		),
		geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)},
	)
	preferredSize := geometry.SizeOptionNone
	if inputs.SizingMode == tree.SizingModeInherentSize {
		preferredSize = geometry.SizeMaybeAdd(
			geometry.SizeOptionMaybeApplyAspectRatio(
				style.MaybeResolveSizeDimOption(coreStyle.GetSize(), parentSize, calc),
				aspectRatio,
			),
			geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)},
		)
	}

	// Scrollbar gutters
	overflowStyle := coreStyle.GetOverflow()
	scrollbarGutter := geometry.Size[float32]{
		Width: func() float32 {
			if overflowStyle.Y == style.OverflowScroll {
				return coreStyle.GetScrollbarWidth()
			}
			return 0.0
		}(),
		Height: func() float32 {
			if overflowStyle.X == style.OverflowScroll {
				return coreStyle.GetScrollbarWidth()
			}
			return 0.0
		}(),
	}
	contentBoxInset := paddingBorder
	contentBoxInset.Bottom += scrollbarGutter.Height
	switch direction {
	case style.DirectionLTR:
		contentBoxInset.Right += scrollbarGutter.Width
	case style.DirectionRTL:
		contentBoxInset.Left += scrollbarGutter.Width
	}

	alignContent := containerStyle.GetAlignContent().UnwrapOr(style.AlignContentStretch)
	justifyContent := containerStyle.GetJustifyContent().UnwrapOr(style.AlignContentStretch)
	alignItems := containerStyle.GetAlignItems()
	justifyItemsStyle := containerStyle.GetJustifyItems()

	constrainedAS := computeConstrainedAvailableSpace(knownDimensions, preferredSize, availableSpace, minSize, maxSize, paddingBorderSize)

	availableGridSpace := geometry.Size[style.AvailableSpace]{
		Width:  constrainedAS.Width.MapDefiniteValue(func(space float32) float32 { return space - geometry.RectHorizontalAxisSum(contentBoxInset) }),
		Height: constrainedAS.Height.MapDefiniteValue(func(space float32) float32 { return space - geometry.RectVerticalAxisSum(contentBoxInset) }),
	}

	outerNodeSize := geometry.SizeMaybeMax(
		geometry.SizeMaybeClamp(geometry.SizeOptionOr(knownDimensions, preferredSize), minSize, maxSize),
		geometry.Size[lib.Option[float32]]{Width: lib.Some(paddingBorderSize.Width), Height: lib.Some(paddingBorderSize.Height)},
	)
	innerNodeSize := geometry.Size[lib.Option[float32]]{
		Width:  geometry.MaybeSubOption(outerNodeSize.Width, lib.Some(geometry.RectHorizontalAxisSum(contentBoxInset))),
		Height: geometry.MaybeSubOption(outerNodeSize.Height, lib.Some(geometry.RectVerticalAxisSum(contentBoxInset))),
	}

	if runMode == tree.RunModeComputeSize && outerNodeSize.Width.IsSome && outerNodeSize.Height.IsSome {
		return tree.LayoutOutputFromOuterSize(geometry.Size[float32]{Width: outerNodeSize.Width.Value, Height: outerNodeSize.Height.Value})
	}

	// 2. Resolve the explicit grid
	autoFitContainerSize := geometry.SizeMaybeSub(
		geometry.SizeMaybeMax(
			geometry.SizeMaybeClamp(
				geometry.SizeOptionOr(geometry.SizeOptionOr(outerNodeSize, maxSize), minSize),
				minSize, maxSize,
			),
			geometry.Size[lib.Option[float32]]{Width: lib.Some(paddingBorderSize.Width), Height: lib.Some(paddingBorderSize.Height)},
		),
		geometry.Size[lib.Option[float32]]{
			Width:  lib.Some(geometry.RectHorizontalAxisSum(contentBoxInset)),
			Height: lib.Some(geometry.RectVerticalAxisSum(contentBoxInset)),
		},
	)

	autoRepeatFitStrategy := geometry.Size[autoRepeatStrategy]{
		Width: func() autoRepeatStrategy {
			if geometry.SizeOptionOr(outerNodeSize, maxSize).Width.IsSome {
				return autoRepeatMaxRepsNoOverflow
			}
			return autoRepeatMinRepsOverflow
		}(),
		Height: func() autoRepeatStrategy {
			if geometry.SizeOptionOr(outerNodeSize, maxSize).Height.IsSome {
				return autoRepeatMaxRepsNoOverflow
			}
			return autoRepeatMinRepsOverflow
		}(),
	}

	colAutoRepetitionCount, gridTemplateColCount := computeExplicitGridSizeInAxis(
		containerStyle, autoFitContainerSize.Width, autoRepeatFitStrategy.Width, calc, geometry.AbsoluteAxisHorizontal,
	)
	rowAutoRepetitionCount, gridTemplateRowCount := computeExplicitGridSizeInAxis(
		containerStyle, autoFitContainerSize.Height, autoRepeatFitStrategy.Height, calc, geometry.AbsoluteAxisVertical,
	)

	nameResolver := newNamedLineResolver(
		containerStyle,
		containerStyle.GetGridTemplateColumnNames(),
		containerStyle.GetGridTemplateRowNames(),
		containerStyle.GetGridTemplateAreas(),
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
	childStyleSlice := scratch.childStyles
	for _, childNode := range t.ChildIds(node) {
		childStyleSlice = append(childStyleSlice, t.GetGridChildStyle(childNode))
	}
	scratch.childStyles = childStyleSlice
	estColCounts, estRowCounts := computeGridSizeEstimate(explicitColCount, explicitRowCount, direction, childStyleSlice)

	// 4. Grid Item Placement
	items := scratch.items
	cellMatrix := newCellOccupancyMatrix(estColCounts, estRowCounts)

	inFlowChildren := scratch.inFlowChildren
	for index, childNode := range t.ChildIds(node) {
		childCoreStyle := t.GetCoreContainerStyle(childNode)
		childGridStyle := t.GetGridChildStyle(childNode)
		if childCoreStyle.GetBoxGenerationMode() == style.BoxGenerationModeNone || childCoreStyle.GetPosition() == style.PositionAbsolute {
			continue
		}
		inFlowChildren = append(inFlowChildren, gridPlacementChild{index: index, node: childNode, gridItemStyle: childGridStyle, coreStyle: childCoreStyle})
	}

	placeGridItems(
		&cellMatrix, &items, inFlowChildren,
		direction, containerStyle.GetGridAutoFlow(),
		alignItems.UnwrapOr(style.AlignItemsStretch),
		justifyItemsStyle.UnwrapOr(style.AlignItemsStretch),
		&nameResolver,
	)

	// Extract track counts
	finalColCounts := *cellMatrix.trackCounts(geometry.AbsoluteAxisHorizontal)
	finalRowCounts := *cellMatrix.trackCounts(geometry.AbsoluteAxisVertical)

	// 5. Initialize Tracks
	columns := scratch.columns
	rows := scratch.rows
	columnTrackCountsForInit := finalColCounts
	if direction.IsRTL() && finalColCounts.Explicit <= 1 {
		columnTrackCountsForInit.NegativeImplicit = finalColCounts.PositiveImplicit
		columnTrackCountsForInit.PositiveImplicit = finalColCounts.NegativeImplicit
	}
	initializeGridTracks(&columns, columnTrackCountsForInit, containerStyle, geometry.AbsoluteAxisHorizontal,
		func(columnIndex int) bool {
			occupancyIndex := columnIndex
			if direction.IsRTL() {
				occupancyIndex = rtlColumnOccupancyIndexForInit(columnIndex, finalColCounts)
			}
			return cellMatrix.columnIsOccupied(occupancyIndex)
		})
	initializeGridTracks(&rows, finalRowCounts, containerStyle, geometry.AbsoluteAxisVertical,
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
		if items[i].alignSelf == style.AlignItemsBaseline {
			hasBaselineAlignedItem = true
			break
		}
	}

	// Run track sizing for Inline axis
	trackSizingAlgorithm(
		t, geometry.AbstractAxisInline,
		minSize.GetAbstract(geometry.AbstractAxisInline),
		maxSize.GetAbstract(geometry.AbstractAxisInline),
		justifyContent, alignContent,
		availableGridSpace, innerNodeSize,
		columns, rows, items,
		func(track *gridTrack, parentSize lib.Option[float32], t tree.LayoutPartialTree) lib.Option[float32] {
			return track.maxTrackSizingFunction.DefiniteValue(parentSize, func(id uint64, basis float32) float32 {
				return t.ResolveCalcValue(id, basis)
			})
		},
		hasBaselineAlignedItem,
	)
	var initialColumnSum float32
	for i := range columns {
		initialColumnSum += columns[i].baseSize
	}
	if !innerNodeSize.Width.IsSome {
		innerNodeSize.Width = lib.Some(initialColumnSum)
	}

	// Clear caches
	for i := range items {
		items[i].availableSpaceCache = lib.Option[geometry.Size[lib.Option[float32]]]{}
	}

	// Run track sizing for Block axis
	trackSizingAlgorithm(
		t, geometry.AbstractAxisBlock,
		minSize.GetAbstract(geometry.AbstractAxisBlock),
		maxSize.GetAbstract(geometry.AbstractAxisBlock),
		alignContent, justifyContent,
		availableGridSpace, innerNodeSize,
		rows, columns, items,
		func(track *gridTrack, _ lib.Option[float32], _ tree.LayoutPartialTree) lib.Option[float32] {
			return lib.Some(track.baseSize)
		},
		false,
	)
	var initialRowSum float32
	for i := range rows {
		initialRowSum += rows[i].baseSize
	}
	if !innerNodeSize.Height.IsSome {
		innerNodeSize.Height = lib.Some(initialRowSum)
	}

	// 6. Compute container size
	resolvedStyleSize := geometry.SizeOptionOr(knownDimensions, preferredSize)
	containerBorderBox := geometry.Size[float32]{
		Width: geometry.F32Max(
			geometry.SizeMaybeClamp(
				geometry.Size[lib.Option[float32]]{
					Width: func() lib.Option[float32] {
						if resolvedStyleSize.Width.IsSome {
							return resolvedStyleSize.Width
						}
						return lib.Some(initialColumnSum + geometry.RectHorizontalAxisSum(contentBoxInset))
					}(),
					Height: lib.None[float32](),
				},
				minSize, maxSize,
			).Width.UnwrapOr(0.0),
			paddingBorderSize.Width,
		),
		Height: geometry.F32Max(
			geometry.SizeMaybeClamp(
				geometry.Size[lib.Option[float32]]{
					Width: lib.None[float32](),
					Height: func() lib.Option[float32] {
						if resolvedStyleSize.Height.IsSome {
							return resolvedStyleSize.Height
						}
						return lib.Some(initialRowSum + geometry.RectVerticalAxisSum(contentBoxInset))
					}(),
				},
				minSize, maxSize,
			).Height.UnwrapOr(0.0),
			paddingBorderSize.Height,
		),
	}
	containerContentBox := geometry.Size[float32]{
		Width:  geometry.F32Max(0.0, containerBorderBox.Width-geometry.RectHorizontalAxisSum(contentBoxInset)),
		Height: geometry.F32Max(0.0, containerBorderBox.Height-geometry.RectVerticalAxisSum(contentBoxInset)),
	}

	if runMode == tree.RunModeComputeSize {
		scratch.items = items
		scratch.columns = columns
		scratch.rows = rows
		return tree.LayoutOutputFromOuterSize(containerBorderBox)
	}

	// 7. Resolve percentage track base sizes
	if !availableGridSpace.Width.IsDefinite() {
		for i := range columns {
			min := columns[i].minTrackSizingFunction.ResolvedPercentageSize(containerContentBox.Width, calc)
			max := columns[i].maxTrackSizingFunction.ResolvedPercentageSize(containerContentBox.Width, calc)
			columns[i].baseSize = geometry.MaybeClampF32(columns[i].baseSize, min, max)
		}
	}
	if !availableGridSpace.Height.IsDefinite() {
		for i := range rows {
			min := rows[i].minTrackSizingFunction.ResolvedPercentageSize(containerContentBox.Height, calc)
			max := rows[i].maxTrackSizingFunction.ResolvedPercentageSize(containerContentBox.Height, calc)
			rows[i].baseSize = geometry.MaybeClampF32(rows[i].baseSize, min, max)
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
				geometry.AbstractAxisInline, rows, innerNodeSize.Height,
				func(track *gridTrack, _ lib.Option[float32]) lib.Option[float32] { return lib.Some(track.baseSize) },
			)
			newMinContentContrib := items[i].minContentContribution(geometry.AbstractAxisInline, t, availSpace, innerNodeSize)
			if items[i].minContentContributionCache.Width.IsSome &&
				items[i].minContentContributionCache.Width.Value != newMinContentContrib {
				rerunColumnSizing = true
			}
			items[i].availableSpaceCache = lib.Some(availSpace)
			items[i].minContentContributionCache.Width = lib.Some(newMinContentContrib)
			items[i].maxContentContributionCache.Width = lib.None[float32]()
			items[i].minimumContributionCache.Width = lib.None[float32]()
		}
	} else {
		for i := range items {
			items[i].availableSpaceCache = lib.Option[geometry.Size[lib.Option[float32]]]{}
			items[i].minContentContributionCache.Width = lib.None[float32]()
			items[i].maxContentContributionCache.Width = lib.None[float32]()
			items[i].minimumContributionCache.Width = lib.None[float32]()
		}
	}

	if rerunColumnSizing {
		trackSizingAlgorithm(
			t, geometry.AbstractAxisInline,
			minSize.GetAbstract(geometry.AbstractAxisInline),
			maxSize.GetAbstract(geometry.AbstractAxisInline),
			justifyContent, alignContent,
			availableGridSpace, innerNodeSize,
			columns, rows, items,
			func(track *gridTrack, _ lib.Option[float32], _ tree.LayoutPartialTree) lib.Option[float32] {
				return lib.Some(track.baseSize)
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
					geometry.AbstractAxisBlock, columns, innerNodeSize.Width,
					func(track *gridTrack, _ lib.Option[float32]) lib.Option[float32] { return lib.Some(track.baseSize) },
				)
				newMinContentContrib := items[i].minContentContribution(geometry.AbstractAxisBlock, t, availSpace, innerNodeSize)
				if items[i].minContentContributionCache.Height.IsSome &&
					items[i].minContentContributionCache.Height.Value != newMinContentContrib {
					rerunRowSizing = true
				}
				items[i].availableSpaceCache = lib.Some(availSpace)
				items[i].minContentContributionCache.Height = lib.Some(newMinContentContrib)
				items[i].maxContentContributionCache.Height = lib.None[float32]()
				items[i].minimumContributionCache.Height = lib.None[float32]()
			}
		} else {
			for i := range items {
				items[i].availableSpaceCache = lib.Option[geometry.Size[lib.Option[float32]]]{}
				items[i].minContentContributionCache.Height = lib.None[float32]()
				items[i].maxContentContributionCache.Height = lib.None[float32]()
				items[i].minimumContributionCache.Height = lib.None[float32]()
			}
		}

		if rerunRowSizing {
			trackSizingAlgorithm(
				t, geometry.AbstractAxisBlock,
				minSize.GetAbstract(geometry.AbstractAxisBlock),
				maxSize.GetAbstract(geometry.AbstractAxisBlock),
				alignContent, justifyContent,
				availableGridSpace, innerNodeSize,
				rows, columns, items,
				func(track *gridTrack, _ lib.Option[float32], _ tree.LayoutPartialTree) lib.Option[float32] {
					return lib.Some(track.baseSize)
				},
				false,
			)
		}
	}

	// 8. Track Alignment
	inlineSizeWithoutScrollbar := geometry.F32Max(containerBorderBox.Width-paddingBorderSize.Width, 0.0)
	inlineScrollbarGutterForAlignment := geometry.F32Min(scrollbarGutter.Width, inlineSizeWithoutScrollbar)
	alignGridTracks(
		containerContentBox.Width,
		geometry.Line[float32]{
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
		geometry.Line[float32]{Start: border.Left, End: border.Right},
		columns, justifyContent, direction.IsRTL(),
	)
	alignGridTracks(
		containerContentBox.Height,
		geometry.Line[float32]{Start: padding.Top, End: padding.Bottom},
		geometry.Line[float32]{Start: border.Top, End: border.Bottom},
		rows, alignContent, false,
	)

	// 9. Size, Align, and Position Grid Items
	itemContentSizeContribution := geometry.SizeZeroF32

	// Sort items back into original order
	slices.SortStableFunc(items, func(a, b gridItem) int {
		return cmp.Compare(a.sourceOrder, b.sourceOrder)
	})

	containerAlignmentStyles := inBothAbsAxis[lib.Option[style.AlignItems]]{
		horizontal: justifyItemsStyle,
		vertical:   alignItems,
	}

	// Position in-flow children
	for index := range items {
		item := &items[index]
		gridArea := geometry.Rect[float32]{
			Top:    rows[item.rowIndexes.Start+1].offset,
			Bottom: rows[item.rowIndexes.End].offset,
			Left:   columns[item.columnIndexes.Start+1].offset,
			Right:  columns[item.columnIndexes.End].offset,
		}
		contentSizeContrib, yPos, h := alignAndPositionItem(
			t, item.node, uint32(index), gridArea,
			containerAlignmentStyles, item.baselineShim, direction,
		)
		item.yPosition = yPos
		item.height = h
		itemContentSizeContribution = geometry.SizeF32Max(itemContentSizeContribution, contentSizeContrib)
	}

	// Position hidden and absolutely positioned children
	order := uint32(len(items))
	childCount := t.ChildCount(node)
	for i := 0; i < childCount; i++ {
		child := t.GetChildId(node, i)
		childCoreStyle := t.GetCoreContainerStyle(child)
		childGridStyle := t.GetGridChildStyle(child)

		// Position hidden child
		if childCoreStyle.GetBoxGenerationMode() == style.BoxGenerationModeNone {
			t.SetUnroundedLayout(child, tree.Layout{Order: order})
			tree.PerformChildLayout(t, child, geometry.SizeOptionNone, geometry.SizeOptionNone,
				geometry.Size[style.AvailableSpace]{Width: style.AvailableSpaceMaxContentValue, Height: style.AvailableSpaceMaxContentValue}, tree.SizingModeInherentSize, geometry.LineFalseBool)
			order++
			continue
		}

		// Position absolutely positioned child
		if childCoreStyle.GetPosition() == style.PositionAbsolute {
			maybeColIndexes := style.ResolveAbsolutelyPositionedGridTracks(
				style.IntoOriginZeroNonNamed(nameResolver.resolveColumnNames(childGridStyle.GetGridColumn()), finalColCounts.Explicit),
			)
			// Apply direction mirroring
			if direction.IsRTL() {
				maybeColIndexes = geometry.Line[lib.Option[style.OriginZeroLine]]{
					Start: func() lib.Option[style.OriginZeroLine] {
						if maybeColIndexes.Start.IsSome {
							mirrored := style.OriginZeroLine(int16(finalColCounts.Explicit) - int16(maybeColIndexes.Start.Value))
							idx, ok := OzLineTryIntoTrackVecIndex(mirrored, finalColCounts)
							if ok {
								return lib.Some(style.OriginZeroLine(idx))
							}
						}
						return lib.None[style.OriginZeroLine]()
					}(),
					End: func() lib.Option[style.OriginZeroLine] {
						if maybeColIndexes.End.IsSome {
							mirrored := style.OriginZeroLine(int16(finalColCounts.Explicit) - int16(maybeColIndexes.End.Value))
							idx, ok := OzLineTryIntoTrackVecIndex(mirrored, finalColCounts)
							if ok {
								return lib.Some(style.OriginZeroLine(idx))
							}
						}
						return lib.None[style.OriginZeroLine]()
					}(),
				}
				maybeColIndexes = geometry.Line[lib.Option[style.OriginZeroLine]]{Start: maybeColIndexes.End, End: maybeColIndexes.Start}
			} else {
				maybeColIndexes = geometry.Line[lib.Option[style.OriginZeroLine]]{
					Start: func() lib.Option[style.OriginZeroLine] {
						if maybeColIndexes.Start.IsSome {
							idx, ok := OzLineTryIntoTrackVecIndex(maybeColIndexes.Start.Value, finalColCounts)
							if ok {
								return lib.Some(style.OriginZeroLine(idx))
							}
						}
						return lib.None[style.OriginZeroLine]()
					}(),
					End: func() lib.Option[style.OriginZeroLine] {
						if maybeColIndexes.End.IsSome {
							idx, ok := OzLineTryIntoTrackVecIndex(maybeColIndexes.End.Value, finalColCounts)
							if ok {
								return lib.Some(style.OriginZeroLine(idx))
							}
						}
						return lib.None[style.OriginZeroLine]()
					}(),
				}
			}

			maybeRowIndexes := style.ResolveAbsolutelyPositionedGridTracks(
				style.IntoOriginZeroNonNamed(nameResolver.resolveRowNames(childGridStyle.GetGridRow()), finalRowCounts.Explicit),
			)
			maybeRowIndexes = geometry.Line[lib.Option[style.OriginZeroLine]]{
				Start: func() lib.Option[style.OriginZeroLine] {
					if maybeRowIndexes.Start.IsSome {
						idx, ok := OzLineTryIntoTrackVecIndex(maybeRowIndexes.Start.Value, finalRowCounts)
						if ok {
							return lib.Some(style.OriginZeroLine(idx))
						}
					}
					return lib.None[style.OriginZeroLine]()
				}(),
				End: func() lib.Option[style.OriginZeroLine] {
					if maybeRowIndexes.End.IsSome {
						idx, ok := OzLineTryIntoTrackVecIndex(maybeRowIndexes.End.Value, finalRowCounts)
						if ok {
							return lib.Some(style.OriginZeroLine(idx))
						}
					}
					return lib.None[style.OriginZeroLine]()
				}(),
			}

			gridArea := geometry.Rect[float32]{
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
				t, child, order, gridArea, containerAlignmentStyles, 0.0, direction,
			)
			itemContentSizeContribution = geometry.SizeF32Max(itemContentSizeContribution, contentSizeContrib)
			order++
		}
	}

	// If there are no items, return just the container size
	if len(items) == 0 {
		scratch.items = items
		scratch.columns = columns
		scratch.rows = rows
		return tree.LayoutOutputFromOuterSize(containerBorderBox)
	}

	// Determine grid container baseline
	slices.SortStableFunc(items, func(a, b gridItem) int {
		return cmp.Compare(a.rowIndexes.Start, b.rowIndexes.Start)
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
		if firstRowItems[i].alignSelf == style.AlignItemsBaseline {
			rowHasBaselineItem = true
			break
		}
	}

	var baselineItem *gridItem
	if rowHasBaselineItem {
		for i := range firstRowItems {
			if firstRowItems[i].alignSelf == style.AlignItemsBaseline {
				baselineItem = &firstRowItems[i]
				break
			}
		}
	} else {
		baselineItem = &firstRowItems[0]
	}

	gridContainerBaseline := baselineItem.yPosition + baselineItem.baseline.UnwrapOr(baselineItem.height)

	scratch.items = items
	scratch.columns = columns
	scratch.rows = rows
	return tree.LayoutOutputFromSizesAndBaselines(
		containerBorderBox,
		itemContentSizeContribution,
		geometry.Point[lib.Option[float32]]{X: lib.None[float32](), Y: lib.Some(gridContainerBaseline)},
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
	knownDimensions, preferredSize geometry.Size[lib.Option[float32]],
	availableSpace geometry.Size[style.AvailableSpace],
	minSize, maxSize geometry.Size[lib.Option[float32]],
	paddingBorderSize geometry.Size[float32],
) geometry.Size[style.AvailableSpace] {
	kOrP := geometry.SizeOptionOr(knownDimensions, preferredSize)
	result := geometry.Size[style.AvailableSpace]{
		Width: func() style.AvailableSpace {
			if kOrP.Width.IsSome {
				return style.AvailableSpaceFromDefinite(kOrP.Width.Value)
			}
			return availableSpace.Width
		}(),
		Height: func() style.AvailableSpace {
			if kOrP.Height.IsSome {
				return style.AvailableSpaceFromDefinite(kOrP.Height.Value)
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

func clampAvailableSpace(as style.AvailableSpace, min, max lib.Option[float32]) style.AvailableSpace {
	if as.Tag != style.AvailableSpaceDefinite {
		return as
	}
	val := as.Value
	if min.IsSome && val < min.Value {
		val = min.Value
	}
	if max.IsSome && val > max.Value {
		val = max.Value
	}
	return style.AvailableSpaceFromDefinite(val)
}

func maxAvailableSpace(as style.AvailableSpace, val float32) style.AvailableSpace {
	if as.Tag != style.AvailableSpaceDefinite {
		return as
	}
	if as.Value < val {
		return style.AvailableSpaceFromDefinite(val)
	}
	return as
}
