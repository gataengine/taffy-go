// Package taffy implements the CSS Grid placement algorithm.
// This file is a port of compute/grid/placement.rs from the Taffy Rust layout engine.
package taffy

// axisIsReversed returns whether placement/search should run in reverse for this axis.
func axisIsReversed(direction Direction, axis AbsoluteAxis) bool {
	return direction.IsRTL() && axis == AbsoluteAxisHorizontal
}

// advancePosition advances the cursor by one track in the active search direction.
func advancePosition(position OriginZeroLine, isReversed bool) OriginZeroLine {
	if isReversed {
		return OriginZeroLine(int16(position) - 1)
	}
	return OriginZeroLine(int16(position) + 1)
}

// searchStartLine returns the initial search line for sparse/dense placement.
func searchStartLine(gridStartLine, gridEndLine OriginZeroLine, isReversed bool) OriginZeroLine {
	if isReversed {
		return gridEndLine.SubU16(1)
	}
	return gridStartLine
}

// resolveIndefiniteGridSpan resolves an indefinite span at position, respecting the active axis direction.
func resolveIndefiniteGridSpan(position OriginZeroLine, span uint16, isReversed bool) Line[OriginZeroLine] {
	if isReversed {
		return Line[OriginZeroLine]{
			Start: (position.SubU16(span)).AddU16(1),
			End:   position.AddU16(1),
		}
	}
	return Line[OriginZeroLine]{
		Start: position,
		End:   position.AddU16(span),
	}
}

// mirrorHorizontalSpan mirrors a horizontal span around the explicit grid width.
func mirrorHorizontalSpan(span Line[OriginZeroLine], explicitColCount uint16) Line[OriginZeroLine] {
	explicitColEndLine := int16(explicitColCount)
	return Line[OriginZeroLine]{
		Start: OriginZeroLine(explicitColEndLine - int16(span.End)),
		End:   OriginZeroLine(explicitColEndLine - int16(span.Start)),
	}
}

// maybeMirrorSpan mirrors horizontal spans for RTL while leaving all other spans unchanged.
func maybeMirrorSpan(span Line[OriginZeroLine], axis AbsoluteAxis, direction Direction, explicitColCount uint16) Line[OriginZeroLine] {
	if axis == AbsoluteAxisHorizontal && direction.IsRTL() {
		return mirrorHorizontalSpan(span, explicitColCount)
	}
	return span
}

// placeGridItems implements the Grid Item Placement Algorithm (8.5).
// It places items into the grid, generating new rows/columns into the implicit grid as required.
func placeGridItems(
	cellOccupancyMatrix *cellOccupancyMatrix,
	items *[]gridItem,
	childrenIter func() []gridPlacementChild,
	direction Direction,
	gridAutoFlow GridAutoFlow,
	alignItems AlignItems,
	justifyItems AlignItems,
	nameResolver *namedLineResolver,
) {
	primaryAxis := gridAutoFlow.PrimaryAxis()
	secondaryAxis := primaryAxis.OtherAxis()
	explicitColCount := cellOccupancyMatrix.trackCounts(AbsoluteAxisHorizontal).Explicit
	explicitRowCount := cellOccupancyMatrix.trackCounts(AbsoluteAxisVertical).Explicit

	mapToOriginZero := func(child gridPlacementChild) gridPlacementChildOZ {
		placement := inBothAbsAxis[Line[OriginZeroGridPlacement]]{
			horizontal: IntoOriginZeroNonNamed(
				nameResolver.resolveColumnNames(child.style.gridColumn()),
				explicitColCount,
			),
			vertical: IntoOriginZeroNonNamed(
				nameResolver.resolveRowNames(child.style.gridRow()),
				explicitRowCount,
			),
		}
		return gridPlacementChildOZ{
			index:     child.index,
			node:      child.node,
			style:     child.style,
			placement: placement,
		}
	}

	// 1. Place children with definite positions
	for _, child := range childrenIter() {
		oz := mapToOriginZero(child)
		if !IsDefiniteOriginZero(oz.placement.horizontal) || !IsDefiniteOriginZero(oz.placement.vertical) {
			continue
		}
		rowSpan, colSpan := placeDefiniteGridItem(oz.placement, primaryAxis, direction, explicitColCount)
		recordGridPlacement(cellOccupancyMatrix, items, oz.node, oz.index, oz.style, alignItems, justifyItems,
			primaryAxis, rowSpan, colSpan, cellDefinitelyPlaced)
	}

	// 2. Place remaining children with definite secondary axis positions
	for _, child := range childrenIter() {
		oz := mapToOriginZero(child)
		secondaryIsDefinite := IsDefiniteOriginZero(oz.placement.get(secondaryAxis))
		primaryIsDefinite := IsDefiniteOriginZero(oz.placement.get(primaryAxis))
		if !secondaryIsDefinite || primaryIsDefinite {
			continue
		}
		primarySpan, secondarySpan := placeDefiniteSecondaryAxisItem(
			cellOccupancyMatrix, oz.placement, gridAutoFlow, direction, explicitColCount,
		)
		recordGridPlacement(cellOccupancyMatrix, items, oz.node, oz.index, oz.style, alignItems, justifyItems,
			primaryAxis, primarySpan, secondarySpan, cellAutoPlaced)
	}

	// 4. Position the remaining grid items
	primaryAxisGridStartLine := cellOccupancyMatrix.trackCounts(primaryAxis).ImplicitStartLine()
	primaryAxisGridEndLine := cellOccupancyMatrix.trackCounts(primaryAxis).ImplicitEndLine()
	secondaryAxisGridStartLine := cellOccupancyMatrix.trackCounts(secondaryAxis).ImplicitStartLine()
	secondaryAxisGridEndLine := cellOccupancyMatrix.trackCounts(secondaryAxis).ImplicitEndLine()
	primaryAxisIsReversed := axisIsReversed(direction, primaryAxis)
	gridStartPosition := [2]OriginZeroLine{
		searchStartLine(primaryAxisGridStartLine, primaryAxisGridEndLine, primaryAxisIsReversed),
		searchStartLine(secondaryAxisGridStartLine, secondaryAxisGridEndLine, axisIsReversed(direction, secondaryAxis)),
	}
	gridPosition := gridStartPosition

	for _, child := range childrenIter() {
		oz := mapToOriginZero(child)
		if IsDefiniteOriginZero(oz.placement.get(secondaryAxis)) {
			continue
		}

		primarySpan, secondarySpan := placeIndefinitelyPositionedItem(
			cellOccupancyMatrix, oz.placement, gridAutoFlow,
			gridPosition, direction, explicitColCount,
		)
		recordGridPlacement(cellOccupancyMatrix, items, oz.node, oz.index, oz.style, alignItems, justifyItems,
			primaryAxis, primarySpan, secondarySpan, cellAutoPlaced)

		// Update grid position for next item
		if gridAutoFlow.IsDense() {
			gridPosition = gridStartPosition
		} else if primaryAxisIsReversed {
			gridPosition = [2]OriginZeroLine{primarySpan.Start, secondarySpan.Start}
		} else {
			gridPosition = [2]OriginZeroLine{primarySpan.End, secondarySpan.Start}
		}
	}
}

// placeDefiniteGridItem places a single definitely positioned item into the grid.
func placeDefiniteGridItem(
	placement inBothAbsAxis[Line[OriginZeroGridPlacement]],
	primaryAxis AbsoluteAxis,
	direction Direction,
	explicitColCount uint16,
) (Line[OriginZeroLine], Line[OriginZeroLine]) {
	primarySpan := maybeMirrorSpan(
		ResolveDefiniteGridLines(placement.get(primaryAxis)),
		primaryAxis, direction, explicitColCount,
	)
	secondarySpan := maybeMirrorSpan(
		ResolveDefiniteGridLines(placement.get(primaryAxis.OtherAxis())),
		primaryAxis.OtherAxis(), direction, explicitColCount,
	)
	return primarySpan, secondarySpan
}

// placeDefiniteSecondaryAxisItem places an item with a definite secondary axis position.
func placeDefiniteSecondaryAxisItem(
	matrix *cellOccupancyMatrix,
	placement inBothAbsAxis[Line[OriginZeroGridPlacement]],
	autoFlow GridAutoFlow,
	direction Direction,
	explicitColCount uint16,
) (Line[OriginZeroLine], Line[OriginZeroLine]) {
	primaryAxis := autoFlow.PrimaryAxis()
	secondaryAxis := primaryAxis.OtherAxis()
	primaryAxisIsReversed := axisIsReversed(direction, primaryAxis)
	primaryAxisGridStartLine := matrix.trackCounts(primaryAxis).ImplicitStartLine()
	primaryAxisGridEndLine := matrix.trackCounts(primaryAxis).ImplicitEndLine()

	secondaryAxisPlacement := maybeMirrorSpan(
		ResolveDefiniteGridLines(placement.get(secondaryAxis)),
		secondaryAxis, direction, explicitColCount,
	)

	var startingPosition OriginZeroLine
	if autoFlow.IsDense() {
		startingPosition = searchStartLine(primaryAxisGridStartLine, primaryAxisGridEndLine, primaryAxisIsReversed)
	} else {
		var found bool
		var lookupResult OriginZeroLine
		if primaryAxisIsReversed {
			lookupResult, found = matrix.firstOfType(primaryAxis, secondaryAxisPlacement.Start, cellAutoPlaced)
		} else {
			lookupResult, found = matrix.lastOfType(primaryAxis, secondaryAxisPlacement.Start, cellAutoPlaced)
		}
		if found {
			startingPosition = lookupResult
		} else {
			startingPosition = searchStartLine(primaryAxisGridStartLine, primaryAxisGridEndLine, primaryAxisIsReversed)
		}
	}

	primaryAxisSpan := IndefiniteSpan(placement.get(primaryAxis))

	position := startingPosition
	for {
		primaryAxisPlacement := resolveIndefiniteGridSpan(position, primaryAxisSpan, primaryAxisIsReversed)
		doesFit := matrix.lineAreaIsUnoccupied(primaryAxis, primaryAxisPlacement, secondaryAxisPlacement)
		if doesFit {
			return primaryAxisPlacement, secondaryAxisPlacement
		}
		position = advancePosition(position, primaryAxisIsReversed)
	}
}

// placeIndefinitelyPositionedItem places an item with no definite position in either axis.
func placeIndefinitelyPositionedItem(
	matrix *cellOccupancyMatrix,
	placement inBothAbsAxis[Line[OriginZeroGridPlacement]],
	autoFlow GridAutoFlow,
	gridPosition [2]OriginZeroLine,
	direction Direction,
	explicitColCount uint16,
) (Line[OriginZeroLine], Line[OriginZeroLine]) {
	primaryAxis := autoFlow.PrimaryAxis()
	secondaryAxis := primaryAxis.OtherAxis()
	primaryAxisIsReversed := axisIsReversed(direction, primaryAxis)
	secondaryAxisIsReversed := axisIsReversed(direction, secondaryAxis)

	primaryPlacementStyle := placement.get(primaryAxis)
	secondaryPlacementStyle := placement.get(secondaryAxis)

	secondarySpan := IndefiniteSpan(secondaryPlacementStyle)
	hasDefinitePrimaryAxisPosition := IsDefiniteOriginZero(primaryPlacementStyle)
	primaryAxisGridStartLine := matrix.trackCounts(primaryAxis).ImplicitStartLine()
	primaryAxisGridEndLine := matrix.trackCounts(primaryAxis).ImplicitEndLine()
	secondaryAxisGridStartLine := matrix.trackCounts(secondaryAxis).ImplicitStartLine()
	secondaryAxisGridEndLine := matrix.trackCounts(secondaryAxis).ImplicitEndLine()
	primaryStartPosition := searchStartLine(primaryAxisGridStartLine, primaryAxisGridEndLine, primaryAxisIsReversed)
	secondaryStartPosition := searchStartLine(secondaryAxisGridStartLine, secondaryAxisGridEndLine, secondaryAxisIsReversed)

	lineAreaIsOccupied := func(pSpan, sSpan Line[OriginZeroLine]) bool {
		return !matrix.lineAreaIsUnoccupied(primaryAxis, pSpan, sSpan)
	}

	primaryIdx, secondaryIdx := gridPosition[0], gridPosition[1]

	if hasDefinitePrimaryAxisPosition {
		primarySpan := maybeMirrorSpan(
			ResolveDefiniteGridLines(primaryPlacementStyle),
			primaryAxis, direction, explicitColCount,
		)

		// Compute secondary axis starting position for search
		if autoFlow.IsDense() {
			secondaryIdx = secondaryStartPosition
		} else {
			shouldAdvanceSecondary := false
			if primaryAxisIsReversed {
				shouldAdvanceSecondary = int16(primarySpan.Start) > int16(primaryIdx)
			} else {
				shouldAdvanceSecondary = int16(primarySpan.Start) < int16(primaryIdx)
			}
			if shouldAdvanceSecondary {
				secondaryIdx = advancePosition(secondaryIdx, secondaryAxisIsReversed)
			}
		}

		// Item has fixed primary axis position: search secondary axis
		for {
			sSpan := resolveIndefiniteGridSpan(secondaryIdx, secondarySpan, secondaryAxisIsReversed)
			if !lineAreaIsOccupied(primarySpan, sSpan) {
				return primarySpan, sSpan
			}
			secondaryIdx = advancePosition(secondaryIdx, secondaryAxisIsReversed)
		}
	} else {
		primarySpanSize := IndefiniteSpan(primaryPlacementStyle)

		// Item does not have any fixed axis
		for {
			pSpan := resolveIndefiniteGridSpan(primaryIdx, primarySpanSize, primaryAxisIsReversed)
			sSpan := resolveIndefiniteGridSpan(secondaryIdx, secondarySpan, secondaryAxisIsReversed)

			// Check if primary index is out of bounds
			primaryOutOfBounds := false
			if primaryAxisIsReversed {
				primaryOutOfBounds = int16(pSpan.Start) < int16(primaryAxisGridStartLine)
			} else {
				primaryOutOfBounds = int16(pSpan.End) > int16(primaryAxisGridEndLine)
			}
			if primaryOutOfBounds {
				secondaryIdx = advancePosition(secondaryIdx, secondaryAxisIsReversed)
				primaryIdx = primaryStartPosition
				continue
			}

			if lineAreaIsOccupied(pSpan, sSpan) {
				primaryIdx = advancePosition(primaryIdx, primaryAxisIsReversed)
				continue
			}

			return pSpan, sSpan
		}
	}
}

// recordGridPlacement records the grid item placement in both the occupancy matrix and items list.
func recordGridPlacement(
	matrix *cellOccupancyMatrix,
	items *[]gridItem,
	node NodeId,
	index int,
	style *Style,
	parentAlignItems AlignItems,
	parentJustifyItems AlignItems,
	primaryAxis AbsoluteAxis,
	primarySpan Line[OriginZeroLine],
	secondarySpan Line[OriginZeroLine],
	placementType cellOccupancyState,
) {
	// Mark area of grid as occupied
	matrix.markAreaAs(primaryAxis, primarySpan, secondarySpan, placementType)

	// Create grid item
	var colSpan, rowSpan Line[OriginZeroLine]
	switch primaryAxis {
	case AbsoluteAxisHorizontal:
		colSpan = primarySpan
		rowSpan = secondarySpan
	case AbsoluteAxisVertical:
		colSpan = secondarySpan
		rowSpan = primarySpan
	}
	*items = append(*items, newGridItemWithPlacementStyleAndOrder(
		node, colSpan, rowSpan, style, parentAlignItems, parentJustifyItems, uint16(index),
	))
}

// gridPlacementChild is a helper struct for the placement algorithm.
type gridPlacementChild struct {
	index int
	node  NodeId
	style *Style
}

// gridPlacementChildOZ is the origin-zero version of gridPlacementChild.
type gridPlacementChildOZ struct {
	index     int
	node      NodeId
	style     *Style
	placement inBothAbsAxis[Line[OriginZeroGridPlacement]]
}

// getAbsOZ returns the placement for the specified absolute axis.
func getAbsOZ(p inBothAbsAxis[Line[OriginZeroGridPlacement]], axis AbsoluteAxis) Line[OriginZeroGridPlacement] {
	return p.get(axis)
}
