// Package taffy provides the namedLineResolver for resolving named grid lines and areas.
// This file is a port of compute/grid/types/named.rs from the Taffy Rust layout engine.
package taffy

import "fmt"

// namedLineResolver takes grid line names and area names as input and can then be used
// to resolve line names of grid placement properties into line numbers.
type namedLineResolver struct {
	// rowLines maps row line names to line numbers.
	rowLines map[string][]uint16
	// columnLines maps column line names to line numbers.
	columnLines map[string][]uint16
	// areas maps area names to area definitions.
	areas map[string]GridTemplateArea
	// areaColumnCount is the number of columns implied by grid area definitions.
	areaColumnCount uint16
	// areaRowCount is the number of rows implied by grid area definitions.
	areaRowCount uint16
	// explicitColumnCount is the number of explicit columns in the grid.
	explicitColumnCount uint16
	// explicitRowCount is the number of explicit rows in the grid.
	explicitRowCount uint16
}

// upsertLineNameMap creates or updates an entry in a line name map.
func upsertLineNameMap(m map[string][]uint16, key string, value uint16) {
	if existing, ok := m[key]; ok {
		m[key] = append(existing, value)
	} else {
		m[key] = []uint16{value}
	}
}

// newNamedLineResolver creates and initialises a new namedLineResolver.
func newNamedLineResolver(
	style gridContainerStyle,
	gridTemplateColumnNames [][]string,
	gridTemplateRowNames [][]string,
	gridTemplateAreas []GridTemplateArea,
	columnAutoRepetitions uint16,
	rowAutoRepetitions uint16,
) namedLineResolver {
	areas := make(map[string]GridTemplateArea)
	columnLines := make(map[string][]uint16)
	rowLines := make(map[string][]uint16)

	var areaColumnCount uint16
	var areaRowCount uint16

	for _, area := range gridTemplateAreas {
		areas[area.Name] = area

		colEnd := area.ColumnEnd
		if colEnd < 1 {
			colEnd = 1
		}
		if colEnd-1 > areaColumnCount {
			areaColumnCount = colEnd - 1
		}

		rowEnd := area.RowEnd
		if rowEnd < 1 {
			rowEnd = 1
		}
		if rowEnd-1 > areaRowCount {
			areaRowCount = rowEnd - 1
		}

		upsertLineNameMap(columnLines, fmt.Sprintf("%s-start", area.Name), area.ColumnStart)
		upsertLineNameMap(columnLines, fmt.Sprintf("%s-end", area.Name), area.ColumnEnd)
		upsertLineNameMap(rowLines, fmt.Sprintf("%s-start", area.Name), area.RowStart)
		upsertLineNameMap(rowLines, fmt.Sprintf("%s-end", area.Name), area.RowEnd)
	}

	// Process column template names and tracks
	columnTracks := style.gridTemplateColumns()
	if columnTemplateColumnNames := gridTemplateColumnNames; columnTemplateColumnNames != nil {
		var currentLine uint16
		trackIdx := 0
		for _, lineNames := range columnTemplateColumnNames {
			currentLine++
			for _, lineName := range lineNames {
				upsertLineNameMap(columnLines, lineName, currentLine)
			}

			if trackIdx < len(columnTracks) {
				track := columnTracks[trackIdx]
				trackIdx++
				if track.Tag == GridTemplateRepeat {
					rep := track.Repetition
					var repeatCount uint16
					switch rep.Count.Tag {
					case RepetitionCountFixed:
						repeatCount = rep.Count.Count
					case RepetitionAutoFill, RepetitionAutoFit:
						repeatCount = columnAutoRepetitions
					}

					for range repeatCount {
						if rep.LineNames != nil {
							for _, lineNameSet := range rep.LineNames {
								for _, lineName := range lineNameSet {
									upsertLineNameMap(columnLines, lineName, currentLine)
								}
								currentLine++
							}
							// Last line name set collapses with following line name set
							currentLine--
						}
					}
					// Last line name set collapses with following line name set
					currentLine--
				}
			}
		}
	}
	// Sort and dedup column lines
	for key, lines := range columnLines {
		columnLines[key] = sortAndDedup(lines)
	}

	// Process row template names and tracks
	rowTracks := style.gridTemplateRows()
	if templateRowNames := gridTemplateRowNames; templateRowNames != nil {
		var currentLine uint16
		trackIdx := 0
		for _, lineNames := range templateRowNames {
			currentLine++
			for _, lineName := range lineNames {
				upsertLineNameMap(rowLines, lineName, currentLine)
			}

			if trackIdx < len(rowTracks) {
				track := rowTracks[trackIdx]
				trackIdx++
				if track.Tag == GridTemplateRepeat {
					rep := track.Repetition
					var repeatCount uint16
					switch rep.Count.Tag {
					case RepetitionCountFixed:
						repeatCount = rep.Count.Count
					case RepetitionAutoFill, RepetitionAutoFit:
						repeatCount = rowAutoRepetitions
					}

					for range repeatCount {
						if rep.LineNames != nil {
							for _, lineNameSet := range rep.LineNames {
								for _, lineName := range lineNameSet {
									upsertLineNameMap(rowLines, lineName, currentLine)
								}
								currentLine++
							}
							// Last line name set collapses with following line name set
							currentLine--
						}
					}
					// Last line name set collapses with following line name set
					currentLine--
				}
			}
		}
	}
	// Sort and dedup row lines
	for key, lines := range rowLines {
		rowLines[key] = sortAndDedup(lines)
	}

	return namedLineResolver{
		areaColumnCount:     areaColumnCount,
		areaRowCount:        areaRowCount,
		explicitColumnCount: 0, // Overwritten later
		explicitRowCount:    0, // Overwritten later
		areas:               areas,
		rowLines:            rowLines,
		columnLines:         columnLines,
	}
}

// sortAndDedup sorts and removes duplicates from a uint16 slice.
func sortAndDedup(lines []uint16) []uint16 {
	if len(lines) <= 1 {
		return lines
	}
	// Simple insertion sort (slices are small)
	for i := 1; i < len(lines); i++ {
		key := lines[i]
		j := i - 1
		for j >= 0 && lines[j] > key {
			lines[j+1] = lines[j]
			j--
		}
		lines[j+1] = key
	}
	// Dedup
	w := 1
	for r := 1; r < len(lines); r++ {
		if lines[r] != lines[r-1] {
			lines[w] = lines[r]
			w++
		}
	}
	return lines[:w]
}

// resolveRowNames resolves named lines for both start and end of a row-axis grid placement.
func (r *namedLineResolver) resolveRowNames(line Line[GridPlacement]) Line[NonNamedGridPlacement] {
	return r.resolveLineNames(line, GridAreaAxisRow)
}

// resolveColumnNames resolves named lines for both start and end of a column-axis grid placement.
func (r *namedLineResolver) resolveColumnNames(line Line[GridPlacement]) Line[NonNamedGridPlacement] {
	return r.resolveLineNames(line, GridAreaAxisColumn)
}

// resolveLineNames resolves named lines for both start and end of a grid placement.
func (r *namedLineResolver) resolveLineNames(line Line[GridPlacement], axis GridAreaAxis) Line[NonNamedGridPlacement] {
	// Resolve start
	startResolved := line.Start
	if line.Start.Tag == GridPlacementNamedLine {
		idx := r.findLineIndex(line.Start.LineName, line.Start.LineIndex, axis, GridAreaEndStart, func(lines []uint16) []uint16 {
			return lines
		})
		startResolved = GridPlacement{Tag: GridPlacementLine, LineIndex: idx.AsI16()}
	}

	// Resolve end
	endResolved := line.End
	if line.End.Tag == GridPlacementNamedLine {
		idx := r.findLineIndex(line.End.LineName, line.End.LineIndex, axis, GridAreaEndEnd, func(lines []uint16) []uint16 {
			return lines
		})
		endResolved = GridPlacement{Tag: GridPlacementLine, LineIndex: idx.AsI16()}
	}

	// Handle NamedSpan cases
	switch {
	case startResolved.Tag == GridPlacementLine && endResolved.Tag == GridPlacementNamedSpan:
		explicitTrackCount := r.explicitTrackCount(axis)
		startLine := startResolved.LineIndex
		var normalizedStartLine uint16
		if startLine > 0 {
			normalizedStartLine = uint16(startLine)
		} else {
			val := int16(explicitTrackCount) + 1 + startLine
			if val < 0 {
				val = 0
			}
			normalizedStartLine = uint16(val)
		}
		endLine := r.findLineIndex(endResolved.LineName, int16(endResolved.SpanSize), axis, GridAreaEndEnd, func(lines []uint16) []uint16 {
			point := partitionPoint(lines, normalizedStartLine)
			return lines[point:]
		})
		return Line[NonNamedGridPlacement]{
			Start: GenericGridPlacement[GridLine]{Tag: GenericGridPlacementLine, Line: GridLine(startResolved.LineIndex)},
			End:   GenericGridPlacement[GridLine]{Tag: GenericGridPlacementLine, Line: endLine},
		}

	case startResolved.Tag == GridPlacementNamedSpan && endResolved.Tag == GridPlacementLine:
		explicitTrackCount := r.explicitTrackCount(axis)
		endLine := endResolved.LineIndex
		var normalizedEndLine uint16
		if endLine > 0 {
			normalizedEndLine = uint16(endLine)
		} else {
			val := int16(explicitTrackCount) + 1 + endLine
			if val < 0 {
				val = 0
			}
			normalizedEndLine = uint16(val)
		}
		startLine := r.findLineIndex(startResolved.LineName, int16(startResolved.SpanSize), axis, GridAreaEndStart, func(lines []uint16) []uint16 {
			point := partitionPointLT(lines, normalizedEndLine)
			return lines[:point]
		})
		return Line[NonNamedGridPlacement]{
			Start: GenericGridPlacement[GridLine]{Tag: GenericGridPlacementLine, Line: startLine},
			End:   GenericGridPlacement[GridLine]{Tag: GenericGridPlacementLine, Line: GridLine(endResolved.LineIndex)},
		}

	default:
		return Line[NonNamedGridPlacement]{
			Start: convertToNonNamed(startResolved),
			End:   convertToNonNamed(endResolved),
		}
	}
}

// explicitTrackCount returns the explicit track count for the given axis.
func (r *namedLineResolver) explicitTrackCount(axis GridAreaAxis) int16 {
	switch axis {
	case GridAreaAxisRow:
		return int16(r.explicitRowCount)
	default:
		return int16(r.explicitColumnCount)
	}
}

// convertToNonNamed converts a GridPlacement to a NonNamedGridPlacement.
func convertToNonNamed(p GridPlacement) NonNamedGridPlacement {
	switch p.Tag {
	case GridPlacementAuto:
		return GenericGridPlacement[GridLine]{Tag: GenericGridPlacementAuto}
	case GridPlacementLine:
		return GenericGridPlacement[GridLine]{Tag: GenericGridPlacementLine, Line: GridLine(p.LineIndex)}
	case GridPlacementSpan:
		return GenericGridPlacement[GridLine]{Tag: GenericGridPlacementSpan, Span: p.SpanSize}
	case GridPlacementNamedSpan:
		return GenericGridPlacement[GridLine]{Tag: GenericGridPlacementSpan, Span: 1}
	default:
		return GenericGridPlacement[GridLine]{Tag: GenericGridPlacementAuto}
	}
}

// partitionPoint returns the index of the first element in lines that is > threshold.
func partitionPoint(lines []uint16, threshold uint16) int {
	for i, v := range lines {
		if v > threshold {
			return i
		}
	}
	return len(lines)
}

// partitionPointLT returns the index of the first element in lines that is >= threshold.
func partitionPointLT(lines []uint16, threshold uint16) int {
	for i, v := range lines {
		if v >= threshold {
			return i
		}
	}
	return len(lines)
}

// findLineIndex resolves the grid line for a named grid line or span.
func (r *namedLineResolver) findLineIndex(
	name string,
	idx int16,
	axis GridAreaAxis,
	end GridAreaEnd,
	filterLines func([]uint16) []uint16,
) GridLine {
	explicitTrackCount := r.explicitTrackCount(axis)

	// An index of 0 is used to represent "no index specified".
	if idx == 0 {
		idx = 1
	}

	// Lookup lines
	var lineLookup map[string][]uint16
	switch axis {
	case GridAreaAxisRow:
		lineLookup = r.rowLines
	default:
		lineLookup = r.columnLines
	}

	if lines, ok := lineLookup[name]; ok {
		return GridLine(getLine(filterLines(lines), explicitTrackCount, idx))
	}

	// Try implicit names (e.g. "name-start" or "name-end")
	switch end {
	case GridAreaEndStart:
		implicitName := fmt.Sprintf("%s-start", name)
		if lines, ok := lineLookup[implicitName]; ok {
			return GridLine(getLine(filterLines(lines), explicitTrackCount, idx))
		}
	case GridAreaEndEnd:
		implicitName := fmt.Sprintf("%s-end", name)
		if lines, ok := lineLookup[implicitName]; ok {
			return GridLine(getLine(filterLines(lines), explicitTrackCount, idx))
		}
	}

	// Non-existent line names match to the first implicit line in the grid.
	var line int16
	if idx > 0 {
		line = (explicitTrackCount + 1) + idx
	} else {
		line = -((explicitTrackCount + 1) + idx)
	}
	return GridLine(line)
}

// getLine returns the line number for the given index into a filtered lines slice.
func getLine(lines []uint16, explicitTrackCount int16, idx int16) int16 {
	absIdx := idx
	if absIdx < 0 {
		absIdx = -absIdx
	}
	enoughLines := absIdx <= int16(len(lines))
	if enoughLines {
		if idx > 0 {
			return int16(lines[absIdx-1])
		}
		return int16(lines[len(lines)-int(absIdx)])
	}
	remainingLines := (absIdx - int16(len(lines)))
	if idx < 0 {
		remainingLines = -remainingLines
	}
	if idx > 0 {
		return (explicitTrackCount + 1) + remainingLines
	}
	return -((explicitTrackCount + 1) + remainingLines)
}

// areaColumnCountVal returns the number of columns defined by the grid areas.
func (r *namedLineResolver) areaColumnCountVal() uint16 {
	return r.areaColumnCount
}

// areaRowCountVal returns the number of rows defined by the grid areas.
func (r *namedLineResolver) areaRowCountVal() uint16 {
	return r.areaRowCount
}

// setExplicitColumnCount sets the number of columns in the explicit grid.
func (r *namedLineResolver) setExplicitColumnCount(count uint16) {
	r.explicitColumnCount = count
}

// setExplicitRowCount sets the number of rows in the explicit grid.
func (r *namedLineResolver) setExplicitRowCount(count uint16) {
	r.explicitRowCount = count
}
