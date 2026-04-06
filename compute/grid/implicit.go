// Package grid provides functions for estimating the implicit grid size.
// This file is a port of compute/grid/implicit_grid.rs from the Taffy Rust layout engine.
package grid

import (
	"taffy-go/geometry"
	"taffy-go/style"
)

// computeGridSizeEstimate estimates the number of rows and columns in the grid.
// This is used as a performance optimisation to pre-size vectors and reduce allocations.
// The estimates for negative implicit and explicit track counts are exact.
// The estimate for positive implicit track count is a lower bound.
func computeGridSizeEstimate(
	explicitColCount uint16,
	explicitRowCount uint16,
	direction style.Direction,
	childStyles []style.GridItemStyle,
) (TrackCounts, TrackCounts) {
	colMin, colMax, colMaxSpan, rowMin, rowMax, rowMaxSpan := getKnownChildPositions(
		childStyles, explicitColCount, explicitRowCount, direction,
	)

	negativeImplicitInlineTracks := colMin.ImpliedNegativeImplicitTracks()
	explicitInlineTracks := explicitColCount
	positiveImplicitInlineTracks := colMax.ImpliedPositiveImplicitTracks(explicitColCount)
	negativeImplicitBlockTracks := rowMin.ImpliedNegativeImplicitTracks()
	explicitBlockTracks := explicitRowCount
	positiveImplicitBlockTracks := rowMax.ImpliedPositiveImplicitTracks(explicitRowCount)

	// Adjust positive track estimate if any items have a span that doesn't fit
	totInlineTracks := negativeImplicitInlineTracks + explicitInlineTracks + positiveImplicitInlineTracks
	if totInlineTracks < colMaxSpan {
		positiveImplicitInlineTracks = colMaxSpan - explicitInlineTracks - negativeImplicitInlineTracks
	}

	totBlockTracks := negativeImplicitBlockTracks + explicitBlockTracks + positiveImplicitBlockTracks
	if totBlockTracks < rowMaxSpan {
		positiveImplicitBlockTracks = rowMaxSpan - explicitBlockTracks - negativeImplicitBlockTracks
	}

	columnCounts := NewTrackCounts(negativeImplicitInlineTracks, explicitInlineTracks, positiveImplicitInlineTracks)
	rowCounts := NewTrackCounts(negativeImplicitBlockTracks, explicitBlockTracks, positiveImplicitBlockTracks)

	return columnCounts, rowCounts
}

// getKnownChildPositions iterates over children, producing an estimate of the min and max grid lines
// along with the span of each item. Min and max grid lines are returned in origin-zero coordinates.
func getKnownChildPositions(
	childStyles []style.GridItemStyle,
	explicitColCount uint16,
	explicitRowCount uint16,
	direction style.Direction,
) (style.OriginZeroLine, style.OriginZeroLine, uint16, style.OriginZeroLine, style.OriginZeroLine, uint16) {
	colMin, colMax, colMaxSpan := style.OriginZeroLine(0), style.OriginZeroLine(0), uint16(0)
	rowMin, rowMax, rowMaxSpan := style.OriginZeroLine(0), style.OriginZeroLine(0), uint16(0)

	for _, childStyle := range childStyles {
		colLine := childStyle.GetGridColumn()
		rowLine := childStyle.GetGridRow()

		childColMin, childColMax, childColSpan := childMinLineMaxLineSpan(colLine, explicitColCount)
		childRowMin, childRowMax, childRowSpan := childMinLineMaxLineSpan(rowLine, explicitRowCount)

		// Placement mirrors horizontal spans in RTL
		if direction.IsRTL() && (childColMin != style.OriginZeroLine(0) || childColMax != style.OriginZeroLine(0)) {
			explicitColEndLine := int16(explicitColCount)
			mirroredMin := style.OriginZeroLine(explicitColEndLine - int16(childColMax))
			mirroredMax := style.OriginZeroLine(explicitColEndLine - int16(childColMin))
			childColMin = mirroredMin
			childColMax = mirroredMax
		}

		if childColMin < colMin {
			colMin = childColMin
		}
		if childColMax > colMax {
			colMax = childColMax
		}
		if childColSpan > colMaxSpan {
			colMaxSpan = childColSpan
		}
		if childRowMin < rowMin {
			rowMin = childRowMin
		}
		if childRowMax > rowMax {
			rowMax = childRowMax
		}
		if childRowSpan > rowMaxSpan {
			rowMaxSpan = childRowSpan
		}
	}

	return colMin, colMax, colMaxSpan, rowMin, rowMax, rowMaxSpan
}

// childMinLineMaxLineSpan produces a conservative estimate of the greatest and smallest grid lines
// used by a single grid item. Values are returned in origin-zero coordinates.
func childMinLineMaxLineSpan(
	line geometry.Line[style.GridPlacement],
	explicitTrackCount uint16,
) (style.OriginZeroLine, style.OriginZeroLine, uint16) {
	// Convert line into origin-zero coordinates, ignoring named lines
	ozLine := style.IntoOriginZeroIgnoringNamed(line, explicitTrackCount)

	start := ozLine.Start
	end := ozLine.End

	var minVal style.OriginZeroLine
	switch {
	case start.Tag == style.GenericGridPlacementLine && end.Tag == style.GenericGridPlacementLine:
		if start.Line == end.Line {
			minVal = start.Line
		} else if start.Line < end.Line {
			minVal = start.Line
		} else {
			minVal = end.Line
		}
	case start.Tag == style.GenericGridPlacementLine && end.Tag == style.GenericGridPlacementAuto:
		minVal = start.Line
	case start.Tag == style.GenericGridPlacementLine && end.Tag == style.GenericGridPlacementSpan:
		minVal = start.Line
	case start.Tag == style.GenericGridPlacementAuto && end.Tag == style.GenericGridPlacementLine:
		minVal = end.Line
	case start.Tag == style.GenericGridPlacementSpan && end.Tag == style.GenericGridPlacementLine:
		minVal = end.Line.SubU16(start.Span)
	default:
		minVal = style.OriginZeroLine(0)
	}

	var maxVal style.OriginZeroLine
	switch {
	case start.Tag == style.GenericGridPlacementLine && end.Tag == style.GenericGridPlacementLine:
		if start.Line == end.Line {
			maxVal = start.Line.AddU16(1)
		} else if start.Line > end.Line {
			maxVal = start.Line
		} else {
			maxVal = end.Line
		}
	case start.Tag == style.GenericGridPlacementLine && end.Tag == style.GenericGridPlacementAuto:
		maxVal = start.Line.AddU16(1)
	case start.Tag == style.GenericGridPlacementLine && end.Tag == style.GenericGridPlacementSpan:
		maxVal = start.Line.AddU16(end.Span)
	case start.Tag == style.GenericGridPlacementAuto && end.Tag == style.GenericGridPlacementLine:
		maxVal = end.Line
	case start.Tag == style.GenericGridPlacementSpan && end.Tag == style.GenericGridPlacementLine:
		maxVal = end.Line
	default:
		maxVal = style.OriginZeroLine(0)
	}

	// Calculate span only for indefinitely placed items
	var span uint16
	isStartAutoOrSpan := start.Tag == style.GenericGridPlacementAuto || start.Tag == style.GenericGridPlacementSpan
	isEndAutoOrSpan := end.Tag == style.GenericGridPlacementAuto || end.Tag == style.GenericGridPlacementSpan
	if isStartAutoOrSpan && isEndAutoOrSpan {
		span = style.IndefiniteSpan(ozLine)
	} else {
		span = 1
	}

	return minVal, maxVal, span
}
