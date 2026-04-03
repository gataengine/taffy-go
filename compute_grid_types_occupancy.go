// Package taffy provides the CellOccupancyMatrix used to track occupied cells during grid placement.
// This file is a port of compute/grid/types/cell_occupancy.rs from the Taffy Rust layout engine.
package taffy

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
	// inner is the 2D grid of occupancy states. Stored as [row][col].
	inner [][]cellOccupancyState
	// columns is the counts of implicit and explicit columns.
	columns TrackCounts
	// rows is the counts of implicit and explicit rows.
	rows TrackCounts
}

// newCellOccupancyMatrix creates a CellOccupancyMatrix given a set of provisional track counts.
func newCellOccupancyMatrix(columns, rows TrackCounts) cellOccupancyMatrix {
	rowCount := rows.Len()
	colCount := columns.Len()
	inner := make([][]cellOccupancyState, rowCount)
	for i := range inner {
		inner[i] = make([]cellOccupancyState, colCount)
	}
	return cellOccupancyMatrix{
		inner:   inner,
		columns: columns,
		rows:    rows,
	}
}

// matrixRows returns the number of rows in the inner matrix.
func (m *cellOccupancyMatrix) matrixRows() int {
	return len(m.inner)
}

// matrixCols returns the number of columns in the inner matrix.
func (m *cellOccupancyMatrix) matrixCols() int {
	if len(m.inner) == 0 {
		return 0
	}
	return len(m.inner[0])
}

// isAreaInRange determines whether the specified area fits within the tracks
// currently represented by the matrix.
func (m *cellOccupancyMatrix) isAreaInRange(
	primaryAxis AbsoluteAxis,
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
	oldColCount := m.matrixCols()
	newRowCount := oldRowCount + int(reqNegativeRows) + int(reqPositiveRows)
	newColCount := oldColCount + int(reqNegativeCols) + int(reqPositiveCols)

	newInner := make([][]cellOccupancyState, newRowCount)

	rowIdx := 0
	// New negative rows
	for i := 0; i < int(reqNegativeRows); i++ {
		newInner[rowIdx] = make([]cellOccupancyState, newColCount)
		rowIdx++
	}

	// Existing rows
	for r := 0; r < oldRowCount; r++ {
		row := make([]cellOccupancyState, newColCount)
		// Copy existing columns, offset by reqNegativeCols
		for c := 0; c < oldColCount; c++ {
			row[int(reqNegativeCols)+c] = m.inner[r][c]
		}
		newInner[rowIdx] = row
		rowIdx++
	}

	// New positive rows
	for i := 0; i < int(reqPositiveRows); i++ {
		newInner[rowIdx] = make([]cellOccupancyState, newColCount)
		rowIdx++
	}

	// Update self with new data
	m.inner = newInner
	m.rows.NegativeImplicit += uint16(reqNegativeRows)
	m.rows.PositiveImplicit += uint16(reqPositiveRows)
	m.columns.NegativeImplicit += uint16(reqNegativeCols)
	m.columns.PositiveImplicit += uint16(reqPositiveCols)
}

// markAreaAs marks an area of the matrix as occupied, expanding the allocated space
// as necessary to accommodate the passed area.
func (m *cellOccupancyMatrix) markAreaAs(
	primaryAxis AbsoluteAxis,
	primarySpan Line[OriginZeroLine],
	secondarySpan Line[OriginZeroLine],
	value cellOccupancyState,
) {
	var rowSpan, columnSpan Line[OriginZeroLine]
	switch primaryAxis {
	case AbsoluteAxisHorizontal:
		rowSpan = secondarySpan
		columnSpan = primarySpan
	case AbsoluteAxisVertical:
		rowSpan = primarySpan
		columnSpan = secondarySpan
	}

	colStart, colEnd := m.columns.OzLineRangeToTrackRange(columnSpan)
	rowStart, rowEnd := m.rows.OzLineRangeToTrackRange(rowSpan)

	// Check that the resolved ranges fit within the allocated grid
	isInRange := m.isAreaInRange(AbsoluteAxisHorizontal, colStart, colEnd, rowStart, rowEnd)
	if !isInRange {
		m.expandToFitRange(rowStart, rowEnd, colStart, colEnd)
		colStart, colEnd = m.columns.OzLineRangeToTrackRange(columnSpan)
		rowStart, rowEnd = m.rows.OzLineRangeToTrackRange(rowSpan)
	}

	for x := rowStart; x < rowEnd; x++ {
		for y := colStart; y < colEnd; y++ {
			m.inner[int(x)][int(y)] = value
		}
	}
}

// lineAreaIsUnoccupied determines whether a grid area specified by bounding grid lines
// in OriginZero coordinates is entirely unoccupied.
func (m *cellOccupancyMatrix) lineAreaIsUnoccupied(
	primaryAxis AbsoluteAxis,
	primarySpan Line[OriginZeroLine],
	secondarySpan Line[OriginZeroLine],
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
	primaryAxis AbsoluteAxis,
	primaryStart, primaryEnd int16,
	secondaryStart, secondaryEnd int16,
) bool {
	var rowStart, rowEnd, colStart, colEnd int16
	switch primaryAxis {
	case AbsoluteAxisHorizontal:
		rowStart, rowEnd = secondaryStart, secondaryEnd
		colStart, colEnd = primaryStart, primaryEnd
	case AbsoluteAxisVertical:
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
			if m.inner[int(x)][int(y)] != cellUnoccupied {
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
	for _, cell := range m.inner[rowIndex] {
		if cell != cellUnoccupied {
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
	for _, row := range m.inner {
		if row[columnIndex] != cellUnoccupied {
			return true
		}
	}
	return false
}

// trackCounts returns the track counts of this CellOccupancyMatrix in the relevant axis.
func (m *cellOccupancyMatrix) trackCounts(trackType AbsoluteAxis) *TrackCounts {
	switch trackType {
	case AbsoluteAxisHorizontal:
		return &m.columns
	default:
		return &m.rows
	}
}

// lastOfType searches backwards from the end of the track and finds the last grid cell
// matching the specified state (if any). Returns the index as an OriginZeroLine, or (zero, false).
func (m *cellOccupancyMatrix) lastOfType(
	trackType AbsoluteAxis,
	startAt OriginZeroLine,
	kind cellOccupancyState,
) (OriginZeroLine, bool) {
	tc := m.trackCounts(trackType.OtherAxis())
	trackComputedIndex := tc.OzLineToNextTrack(startAt)

	var foundIndex int
	found := false

	switch trackType {
	case AbsoluteAxisHorizontal:
		if trackComputedIndex < 0 || int(trackComputedIndex) >= m.matrixRows() {
			return 0, false
		}
		row := m.inner[int(trackComputedIndex)]
		for i := len(row) - 1; i >= 0; i-- {
			if row[i] == kind {
				foundIndex = i
				found = true
				break
			}
		}
	case AbsoluteAxisVertical:
		if trackComputedIndex < 0 || int(trackComputedIndex) >= m.matrixCols() {
			return 0, false
		}
		for i := len(m.inner) - 1; i >= 0; i-- {
			if m.inner[i][int(trackComputedIndex)] == kind {
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
	trackType AbsoluteAxis,
	startAt OriginZeroLine,
	kind cellOccupancyState,
) (OriginZeroLine, bool) {
	tc := m.trackCounts(trackType.OtherAxis())
	trackComputedIndex := tc.OzLineToNextTrack(startAt)

	var foundIndex int
	found := false

	switch trackType {
	case AbsoluteAxisHorizontal:
		if trackComputedIndex < 0 || int(trackComputedIndex) >= m.matrixRows() {
			return 0, false
		}
		row := m.inner[int(trackComputedIndex)]
		for i := 0; i < len(row); i++ {
			if row[i] == kind {
				foundIndex = i
				found = true
				break
			}
		}
	case AbsoluteAxisVertical:
		if trackComputedIndex < 0 || int(trackComputedIndex) >= m.matrixCols() {
			return 0, false
		}
		for i := 0; i < len(m.inner); i++ {
			if m.inner[i][int(trackComputedIndex)] == kind {
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
