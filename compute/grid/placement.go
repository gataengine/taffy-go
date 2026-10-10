// Package grid implements the CSS Grid placement algorithm.
// This file is a port of compute/grid/placement.rs from the Taffy Rust layout engine.
package grid

import (
	"github.com/gataengine/taffy-go/geometry"
	"github.com/gataengine/taffy-go/style"
	"github.com/gataengine/taffy-go/tree"
)

// axisIsReversed returns whether placement/search should run in reverse for this axis.
func axisIsReversed(direction style.Direction, axis geometry.AbsoluteAxis) bool {
	return direction.IsRTL() && axis == geometry.AbsoluteAxisHorizontal
}

// advancePosition advances the cursor by one track in the active search direction.
func advancePosition(position style.OriginZeroLine, isReversed bool) style.OriginZeroLine {
	if isReversed {
		return style.OriginZeroLine(int16(position) - 1)
	}
	return style.OriginZeroLine(int16(position) + 1)
}

// searchStartLine returns the initial search line for sparse/dense placement.
func searchStartLine(gridStartLine, gridEndLine style.OriginZeroLine, isReversed bool) style.OriginZeroLine {
	if isReversed {
		return gridEndLine.SubU16(1)
	}
	return gridStartLine
}

// resolveIndefiniteGridSpan resolves an indefinite span at position, respecting the active axis direction.
func resolveIndefiniteGridSpan(position style.OriginZeroLine, span uint16, isReversed bool) geometry.Line[style.OriginZeroLine] {
	if isReversed {
		return geometry.Line[style.OriginZeroLine]{
			Start: (position.SubU16(span)).AddU16(1),
			End:   position.AddU16(1),
		}
	}
	return geometry.Line[style.OriginZeroLine]{
		Start: position,
		End:   position.AddU16(span),
	}
}

// mirrorHorizontalSpan mirrors a horizontal span around the explicit grid width.
func mirrorHorizontalSpan(span geometry.Line[style.OriginZeroLine], explicitColCount uint16) geometry.Line[style.OriginZeroLine] {
	explicitColEndLine := int16(explicitColCount)
	return geometry.Line[style.OriginZeroLine]{
		Start: style.OriginZeroLine(explicitColEndLine - int16(span.End)),
		End:   style.OriginZeroLine(explicitColEndLine - int16(span.Start)),
	}
}

// maybeMirrorSpan mirrors horizontal spans for RTL while leaving all other spans unchanged.
func maybeMirrorSpan(span geometry.Line[style.OriginZeroLine], axis geometry.AbsoluteAxis, direction style.Direction, explicitColCount uint16) geometry.Line[style.OriginZeroLine] {
	if axis == geometry.AbsoluteAxisHorizontal && direction.IsRTL() {
		return mirrorHorizontalSpan(span, explicitColCount)
	}
	return span
}

// placeGridItems implements the Grid Item Placement Algorithm (8.5).
// It places items into the grid, generating new rows/columns into the implicit grid as required.
func placeGridItems(
	cellOccupancyMatrix *cellOccupancyMatrix,
	items *[]gridItem,
	children []gridPlacementChild,
	direction style.Direction,
	gridAutoFlow style.GridAutoFlow,
	alignItems style.AlignItems,
	justifyItems style.AlignItems,
	nameResolver *namedLineResolver,
) {
	primaryAxis := gridAutoFlow.PrimaryAxis()
	secondaryAxis := primaryAxis.OtherAxis()
	explicitColCount := cellOccupancyMatrix.trackCounts(geometry.AbsoluteAxisHorizontal).Explicit
	explicitRowCount := cellOccupancyMatrix.trackCounts(geometry.AbsoluteAxisVertical).Explicit

	mapToOriginZero := func(child gridPlacementChild) gridPlacementChildOZ {
		placement := inBothAbsAxis[geometry.Line[style.OriginZeroGridPlacement]]{
			horizontal: style.IntoOriginZeroNonNamed(
				nameResolver.resolveColumnNames(child.gridItemStyle.GetGridColumn()),
				explicitColCount,
			),
			vertical: style.IntoOriginZeroNonNamed(
				nameResolver.resolveRowNames(child.gridItemStyle.GetGridRow()),
				explicitRowCount,
			),
		}
		return gridPlacementChildOZ{
			index:         child.index,
			node:          child.node,
			gridItemStyle: child.gridItemStyle,
			coreStyle:     child.coreStyle,
			placement:     placement,
		}
	}

	// 1. Place children with definite positions
	for _, child := range children {
		oz := mapToOriginZero(child)
		if !style.IsDefiniteOriginZero(oz.placement.horizontal) || !style.IsDefiniteOriginZero(oz.placement.vertical) {
			continue
		}
		rowSpan, colSpan := placeDefiniteGridItem(oz.placement, primaryAxis, direction, explicitColCount)
		recordGridPlacement(cellOccupancyMatrix, items, oz.node, oz.index, oz.gridItemStyle, oz.coreStyle, alignItems, justifyItems,
			primaryAxis, rowSpan, colSpan, cellDefinitelyPlaced)
	}

	// 2. Place remaining children with definite secondary axis positions
	for _, child := range children {
		oz := mapToOriginZero(child)
		secondaryIsDefinite := style.IsDefiniteOriginZero(oz.placement.get(secondaryAxis))
		primaryIsDefinite := style.IsDefiniteOriginZero(oz.placement.get(primaryAxis))
		if !secondaryIsDefinite || primaryIsDefinite {
			continue
		}
		primarySpan, secondarySpan := placeDefiniteSecondaryAxisItem(
			cellOccupancyMatrix, oz.placement, gridAutoFlow, direction, explicitColCount,
		)
		recordGridPlacement(cellOccupancyMatrix, items, oz.node, oz.index, oz.gridItemStyle, oz.coreStyle, alignItems, justifyItems,
			primaryAxis, primarySpan, secondarySpan, cellAutoPlaced)
	}

	// 4. Position the remaining grid items
	primaryAxisGridStartLine := cellOccupancyMatrix.trackCounts(primaryAxis).ImplicitStartLine()
	primaryAxisGridEndLine := cellOccupancyMatrix.trackCounts(primaryAxis).ImplicitEndLine()
	secondaryAxisGridStartLine := cellOccupancyMatrix.trackCounts(secondaryAxis).ImplicitStartLine()
	secondaryAxisGridEndLine := cellOccupancyMatrix.trackCounts(secondaryAxis).ImplicitEndLine()
	primaryAxisIsReversed := axisIsReversed(direction, primaryAxis)
	gridStartPosition := [2]style.OriginZeroLine{
		searchStartLine(primaryAxisGridStartLine, primaryAxisGridEndLine, primaryAxisIsReversed),
		searchStartLine(secondaryAxisGridStartLine, secondaryAxisGridEndLine, axisIsReversed(direction, secondaryAxis)),
	}
	gridPosition := gridStartPosition

	for _, child := range children {
		oz := mapToOriginZero(child)
		if style.IsDefiniteOriginZero(oz.placement.get(secondaryAxis)) {
			continue
		}

		primarySpan, secondarySpan := placeIndefinitelyPositionedItem(
			cellOccupancyMatrix, oz.placement, gridAutoFlow,
			gridPosition, direction, explicitColCount,
		)
		recordGridPlacement(cellOccupancyMatrix, items, oz.node, oz.index, oz.gridItemStyle, oz.coreStyle, alignItems, justifyItems,
			primaryAxis, primarySpan, secondarySpan, cellAutoPlaced)

		// Update grid position for next item
		if gridAutoFlow.IsDense() {
			gridPosition = gridStartPosition
		} else if primaryAxisIsReversed {
			gridPosition = [2]style.OriginZeroLine{primarySpan.Start, secondarySpan.Start}
		} else {
			gridPosition = [2]style.OriginZeroLine{primarySpan.End, secondarySpan.Start}
		}
	}
}

// placeDefiniteGridItem places a single definitely positioned item into the grid.
func placeDefiniteGridItem(
	placement inBothAbsAxis[geometry.Line[style.OriginZeroGridPlacement]],
	primaryAxis geometry.AbsoluteAxis,
	direction style.Direction,
	explicitColCount uint16,
) (geometry.Line[style.OriginZeroLine], geometry.Line[style.OriginZeroLine]) {
	primarySpan := maybeMirrorSpan(
		style.ResolveDefiniteGridLines(placement.get(primaryAxis)),
		primaryAxis, direction, explicitColCount,
	)
	secondarySpan := maybeMirrorSpan(
		style.ResolveDefiniteGridLines(placement.get(primaryAxis.OtherAxis())),
		primaryAxis.OtherAxis(), direction, explicitColCount,
	)
	return primarySpan, secondarySpan
}

// placeDefiniteSecondaryAxisItem places an item with a definite secondary axis position.
func placeDefiniteSecondaryAxisItem(
	matrix *cellOccupancyMatrix,
	placement inBothAbsAxis[geometry.Line[style.OriginZeroGridPlacement]],
	autoFlow style.GridAutoFlow,
	direction style.Direction,
	explicitColCount uint16,
) (geometry.Line[style.OriginZeroLine], geometry.Line[style.OriginZeroLine]) {
	primaryAxis := autoFlow.PrimaryAxis()
	secondaryAxis := primaryAxis.OtherAxis()
	primaryAxisIsReversed := axisIsReversed(direction, primaryAxis)
	primaryAxisGridStartLine := matrix.trackCounts(primaryAxis).ImplicitStartLine()
	primaryAxisGridEndLine := matrix.trackCounts(primaryAxis).ImplicitEndLine()

	secondaryAxisPlacement := maybeMirrorSpan(
		style.ResolveDefiniteGridLines(placement.get(secondaryAxis)),
		secondaryAxis, direction, explicitColCount,
	)

	var startingPosition style.OriginZeroLine
	if autoFlow.IsDense() {
		startingPosition = searchStartLine(primaryAxisGridStartLine, primaryAxisGridEndLine, primaryAxisIsReversed)
	} else {
		var found bool
		var lookupResult style.OriginZeroLine
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

	primaryAxisSpan := style.IndefiniteSpan(placement.get(primaryAxis))

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
	placement inBothAbsAxis[geometry.Line[style.OriginZeroGridPlacement]],
	autoFlow style.GridAutoFlow,
	gridPosition [2]style.OriginZeroLine,
	direction style.Direction,
	explicitColCount uint16,
) (geometry.Line[style.OriginZeroLine], geometry.Line[style.OriginZeroLine]) {
	primaryAxis := autoFlow.PrimaryAxis()
	secondaryAxis := primaryAxis.OtherAxis()
	primaryAxisIsReversed := axisIsReversed(direction, primaryAxis)
	secondaryAxisIsReversed := axisIsReversed(direction, secondaryAxis)

	primaryPlacementStyle := placement.get(primaryAxis)
	secondaryPlacementStyle := placement.get(secondaryAxis)

	secondarySpan := style.IndefiniteSpan(secondaryPlacementStyle)
	hasDefinitePrimaryAxisPosition := style.IsDefiniteOriginZero(primaryPlacementStyle)
	primaryAxisGridStartLine := matrix.trackCounts(primaryAxis).ImplicitStartLine()
	primaryAxisGridEndLine := matrix.trackCounts(primaryAxis).ImplicitEndLine()
	secondaryAxisGridStartLine := matrix.trackCounts(secondaryAxis).ImplicitStartLine()
	secondaryAxisGridEndLine := matrix.trackCounts(secondaryAxis).ImplicitEndLine()
	primaryStartPosition := searchStartLine(primaryAxisGridStartLine, primaryAxisGridEndLine, primaryAxisIsReversed)
	secondaryStartPosition := searchStartLine(secondaryAxisGridStartLine, secondaryAxisGridEndLine, secondaryAxisIsReversed)

	lineAreaIsOccupied := func(pSpan, sSpan geometry.Line[style.OriginZeroLine]) bool {
		return !matrix.lineAreaIsUnoccupied(primaryAxis, pSpan, sSpan)
	}

	primaryIdx, secondaryIdx := gridPosition[0], gridPosition[1]

	if hasDefinitePrimaryAxisPosition {
		primarySpan := maybeMirrorSpan(
			style.ResolveDefiniteGridLines(primaryPlacementStyle),
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
		primarySpanSize := style.IndefiniteSpan(primaryPlacementStyle)

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
	node tree.NodeId,
	index int,
	gridItemSt style.GridItemStyle,
	coreSt style.CoreStyle,
	parentAlignItems style.AlignItems,
	parentJustifyItems style.AlignItems,
	primaryAxis geometry.AbsoluteAxis,
	primarySpan geometry.Line[style.OriginZeroLine],
	secondarySpan geometry.Line[style.OriginZeroLine],
	placementType cellOccupancyState,
) {
	// Mark area of grid as occupied
	matrix.markAreaAs(primaryAxis, primarySpan, secondarySpan, placementType)

	// Create grid item
	var colSpan, rowSpan geometry.Line[style.OriginZeroLine]
	switch primaryAxis {
	case geometry.AbsoluteAxisHorizontal:
		colSpan = primarySpan
		rowSpan = secondarySpan
	case geometry.AbsoluteAxisVertical:
		colSpan = secondarySpan
		rowSpan = primarySpan
	}
	*items = append(*items, newGridItemWithPlacementStyleAndOrder(
		node, colSpan, rowSpan, gridItemSt, coreSt, parentAlignItems, parentJustifyItems, uint16(index),
	))
}

// gridPlacementChild is a helper struct for the placement algorithm.
type gridPlacementChild struct {
	index         int
	node          tree.NodeId
	gridItemStyle style.GridItemStyle
	coreStyle     style.CoreStyle
}

// gridPlacementChildOZ is the origin-zero version of gridPlacementChild.
type gridPlacementChildOZ struct {
	index         int
	node          tree.NodeId
	gridItemStyle style.GridItemStyle
	coreStyle     style.CoreStyle
	placement     inBothAbsAxis[geometry.Line[style.OriginZeroGridPlacement]]
}

// inBothAbsAxis holds values for both absolute axes.
type inBothAbsAxis[T any] struct {
	horizontal T
	vertical   T
}

// get returns the value for the specified absolute axis.
func (a inBothAbsAxis[T]) get(axis geometry.AbsoluteAxis) T {
	switch axis {
	case geometry.AbsoluteAxisHorizontal:
		return a.horizontal
	default:
		return a.vertical
	}
}
