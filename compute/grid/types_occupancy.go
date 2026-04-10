// Package grid provides the CellOccupancyMatrix used to track occupied cells during grid placement.
// This file is a port of compute/grid/types/cell_occupancy.rs from the Taffy Rust layout engine.
package grid

import (
	"taffy-go/geometry"
	"taffy-go/style"
)

// cellOccupancyState represents the occupancy state of a single grid cell.
type cellOccupancyState int

const (
	// cellUnoccupied indicates that a grid cell is unoccupied.
	cellUnoccupied cellOccupancyState = iota
	// cellDefinitelyPlaced indicates that a grid cell is occupied by a definitely placed item.
	cellDefinitelyPlaced
	// cellAutoPlaced indicates that a grid cell is occupied by an auto-placed item.
	cellAutoPlaced
)

// cellOccupancyMatrix is a dynamically sized matrix (2d grid) which tracks the occupancy
// of each grid cell during auto-placement. It also keeps tabs on how many tracks there are
// and which tracks are implicit and which are explicit.
type cellOccupancyMatrix struct {
	// inner is the flat grid of occupancy states stored in row-major order.
	inner []cellOccupancyState
	// colCount is the number of columns (stride for row-major indexing).
	colCount int
	// columns is the counts of implicit and explicit columns.
	columns TrackCounts
	// rows is the counts of implicit and explicit rows.
	rows TrackCounts
}

// newCellOccupancyMatrix creates a CellOccupancyMatrix given a set of provisional track counts.
func newCellOccupancyMatrix(columns, rows TrackCounts) cellOccupancyMatrix {
	rowCount := rows.Len()
	colCount := columns.Len()
	return cellOccupancyMatrix{
		inner:    make([]cellOccupancyState, rowCount*colCount),
		colCount: colCount,
		columns:  columns,
		rows:     rows,
	}
}

// matrixRows returns the number of rows in the inner matrix.
func (m *cellOccupancyMatrix) matrixRows() int {
	if m.colCount == 0 {
		return 0
	}
	return len(m.inner) / m.colCount
}

// matrixCols returns the number of columns in the inner matrix.
func (m *cellOccupancyMatrix) matrixCols() int {
	return m.colCount
}

// isAreaInRange determines whether the specified area fits within the tracks
// currently represented by the matrix.
func (m *cellOccupancyMatrix) isAreaInRange(
	primaryAxis geometry.AbsoluteAxis,
	primaryStart, primaryEnd int16,
	secondaryStart, secondaryEnd int16,
) bool {
	primaryLen := int16(m.trackCounts(primaryAxis).Len())
	if primaryStart < 0 || primaryEnd > primaryLen {
		return false
	}
	secondaryLen := int16(m.trackCounts(primaryAxis.OtherAxis()).Len())
	if secondaryStart < 0 || secondaryEnd > secondaryLen {
		return false
	}
	return true
}

// expandToFitRange expands the grid (potentially in all 4 directions) to ensure
// that the specified range fits within the allocated space.
func (m *cellOccupancyMatrix) expandToFitRange(rowStart, rowEnd, colStart, colEnd int16) {
	// Calculate number of rows and columns missing to accommodate ranges
	reqNegativeRows := max16(0, -rowStart)
	reqPositiveRows := max16(0, rowEnd-int16(m.rows.Len()))
	reqNegativeCols := max16(0, -colStart)
	reqPositiveCols := max16(0, colEnd-int16(m.columns.Len()))

	oldRowCount := m.matrixRows()
	oldColCount := m.colCount
	newRowCount := oldRowCount + int(reqNegativeRows) + int(reqPositiveRows)
	newColCount := oldColCount + int(reqNegativeCols) + int(reqPositiveCols)

	newInner := make([]cellOccupancyState, newRowCount*newColCount)

	for r := 0; r < oldRowCount; r++ {
		for c := 0; c < oldColCount; c++ {
			newRow := r + int(reqNegativeRows)
			newCol := c + int(reqNegativeCols)
			newInner[newRow*newColCount+newCol] = m.inner[r*oldColCount+c]
		}
	}

	m.inner = newInner
	m.colCount = newColCount
	m.rows.NegativeImplicit += uint16(reqNegativeRows)
	m.rows.PositiveImplicit += uint16(reqPositiveRows)
	m.columns.NegativeImplicit += uint16(reqNegativeCols)
	m.columns.PositiveImplicit += uint16(reqPositiveCols)
}

// markAreaAs marks an area of the matrix as occupied, expanding the allocated space
// as necessary to accommodate the passed area.
func (m *cellOccupancyMatrix) markAreaAs(
	primaryAxis geometry.AbsoluteAxis,
	primarySpan geometry.Line[style.OriginZeroLine],
	secondarySpan geometry.Line[style.OriginZeroLine],
	value cellOccupancyState,
) {
	var rowSpan, columnSpan geometry.Line[style.OriginZeroLine]
	switch primaryAxis {
	case geometry.AbsoluteAxisHorizontal:
		rowSpan = secondarySpan
		columnSpan = primarySpan
	case geometry.AbsoluteAxisVertical:
		rowSpan = primarySpan
		columnSpan = secondarySpan
	}

	colStart, colEnd := m.columns.OzLineRangeToTrackRange(columnSpan)
	rowStart, rowEnd := m.rows.OzLineRangeToTrackRange(rowSpan)

	// Check that the resolved ranges fit within the allocated grid
	isInRange := m.isAreaInRange(geometry.AbsoluteAxisHorizontal, colStart, colEnd, rowStart, rowEnd)
	if !isInRange {
		m.expandToFitRange(rowStart, rowEnd, colStart, colEnd)
		colStart, colEnd = m.columns.OzLineRangeToTrackRange(columnSpan)
		rowStart, rowEnd = m.rows.OzLineRangeToTrackRange(rowSpan)
	}

	for x := rowStart; x < rowEnd; x++ {
		for y := colStart; y < colEnd; y++ {
			m.inner[int(x)*m.colCount+int(y)] = value
		}
	}
}

// lineAreaIsUnoccupied determines whether a grid area specified by bounding grid lines
// in OriginZero coordinates is entirely unoccupied.
func (m *cellOccupancyMatrix) lineAreaIsUnoccupied(
	primaryAxis geometry.AbsoluteAxis,
	primarySpan geometry.Line[style.OriginZeroLine],
	secondarySpan geometry.Line[style.OriginZeroLine],
) bool {
	pCounts := m.trackCounts(primaryAxis)
	pStart, pEnd := pCounts.OzLineRangeToTrackRange(primarySpan)
	sCounts := m.trackCounts(primaryAxis.OtherAxis())
	sStart, sEnd := sCounts.OzLineRangeToTrackRange(secondarySpan)
	return m.trackAreaIsUnoccupied(primaryAxis, pStart, pEnd, sStart, sEnd)
}

// trackAreaIsUnoccupied determines whether a grid area specified by a range of indexes
// is entirely unoccupied.
func (m *cellOccupancyMatrix) trackAreaIsUnoccupied(
	primaryAxis geometry.AbsoluteAxis,
	primaryStart, primaryEnd int16,
	secondaryStart, secondaryEnd int16,
) bool {
	var rowStart, rowEnd, colStart, colEnd int16
	switch primaryAxis {
	case geometry.AbsoluteAxisHorizontal:
		rowStart, rowEnd = secondaryStart, secondaryEnd
		colStart, colEnd = primaryStart, primaryEnd
	case geometry.AbsoluteAxisVertical:
		rowStart, rowEnd = primaryStart, primaryEnd
		colStart, colEnd = secondaryStart, secondaryEnd
	}

	rows := m.matrixRows()
	cols := m.matrixCols()
	for x := rowStart; x < rowEnd; x++ {
		for y := colStart; y < colEnd; y++ {
			if int(x) < 0 || int(x) >= rows || int(y) < 0 || int(y) >= cols {
				continue // Out of bounds cells are considered unoccupied
			}
			if m.inner[int(x)*m.colCount+int(y)] != cellUnoccupied {
				return false
			}
		}
	}
	return true
}

// rowIsOccupied determines whether the specified row contains any items.
func (m *cellOccupancyMatrix) rowIsOccupied(rowIndex int) bool {
	if rowIndex >= m.matrixRows() {
		return false
	}
	base := rowIndex * m.colCount
	for c := 0; c < m.colCount; c++ {
		if m.inner[base+c] != cellUnoccupied {
			return true
		}
	}
	return false
}

// columnIsOccupied determines whether the specified column contains any items.
func (m *cellOccupancyMatrix) columnIsOccupied(columnIndex int) bool {
	if columnIndex >= m.matrixCols() {
		return false
	}
	rows := m.matrixRows()
	for r := 0; r < rows; r++ {
		if m.inner[r*m.colCount+columnIndex] != cellUnoccupied {
			return true
		}
	}
	return false
}

// trackCounts returns the track counts of this CellOccupancyMatrix in the relevant axis.
func (m *cellOccupancyMatrix) trackCounts(trackType geometry.AbsoluteAxis) *TrackCounts {
	switch trackType {
	case geometry.AbsoluteAxisHorizontal:
		return &m.columns
	default:
		return &m.rows
	}
}

// lastOfType searches backwards from the end of the track and finds the last grid cell
// matching the specified state (if any). Returns the index as an OriginZeroLine, or (zero, false).
func (m *cellOccupancyMatrix) lastOfType(
	trackType geometry.AbsoluteAxis,
	startAt style.OriginZeroLine,
	kind cellOccupancyState,
) (style.OriginZeroLine, bool) {
	tc := m.trackCounts(trackType.OtherAxis())
	trackComputedIndex := tc.OzLineToNextTrack(startAt)

	var foundIndex int
	found := false

	switch trackType {
	case geometry.AbsoluteAxisHorizontal:
		if trackComputedIndex < 0 || int(trackComputedIndex) >= m.matrixRows() {
			return 0, false
		}
		rowBase := int(trackComputedIndex) * m.colCount
		for i := m.colCount - 1; i >= 0; i-- {
			if m.inner[rowBase+i] == kind {
				foundIndex = i
				found = true
				break
			}
		}
	case geometry.AbsoluteAxisVertical:
		col := int(trackComputedIndex)
		if col < 0 || col >= m.matrixCols() {
			return 0, false
		}
		rows := m.matrixRows()
		for i := rows - 1; i >= 0; i-- {
			if m.inner[i*m.colCount+col] == kind {
				foundIndex = i
				found = true
				break
			}
		}
	}

	if !found {
		return 0, false
	}
	return tc.TrackToPrevOzLine(uint16(foundIndex)), true
}

// firstOfType searches forwards from the start of the track and finds the first grid cell
// matching the specified state (if any). Returns the index as an OriginZeroLine, or (zero, false).
func (m *cellOccupancyMatrix) firstOfType(
	trackType geometry.AbsoluteAxis,
	startAt style.OriginZeroLine,
	kind cellOccupancyState,
) (style.OriginZeroLine, bool) {
	tc := m.trackCounts(trackType.OtherAxis())
	trackComputedIndex := tc.OzLineToNextTrack(startAt)

	var foundIndex int
	found := false

	switch trackType {
	case geometry.AbsoluteAxisHorizontal:
		if trackComputedIndex < 0 || int(trackComputedIndex) >= m.matrixRows() {
			return 0, false
		}
		rowBase := int(trackComputedIndex) * m.colCount
		for i := 0; i < m.colCount; i++ {
			if m.inner[rowBase+i] == kind {
				foundIndex = i
				found = true
				break
			}
		}
	case geometry.AbsoluteAxisVertical:
		col := int(trackComputedIndex)
		if col < 0 || col >= m.matrixCols() {
			return 0, false
		}
		rows := m.matrixRows()
		for i := 0; i < rows; i++ {
			if m.inner[i*m.colCount+col] == kind {
				foundIndex = i
				found = true
				break
			}
		}
	}

	if !found {
		return 0, false
	}
	return tc.TrackToPrevOzLine(uint16(foundIndex)), true
}

// max16 returns the larger of two int16 values.
func max16(a, b int16) int16 {
	if a > b {
		return a
	}
	return b
}
