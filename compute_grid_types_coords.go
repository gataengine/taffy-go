// Package taffy provides grid coordinate types and conversion functions.
// This file is a port of compute/grid/types/grid_track_counts.rs from the Taffy Rust layout engine.
// Note: GridLine and OriginZeroLine are already defined in style_grid.go.
// This file adds TrackCounts and coordinate conversion methods.
package taffy

// TrackCounts stores the number of tracks in a given dimension,
// separating implicit (before and after) from explicit tracks.
type TrackCounts struct {
	// NegativeImplicit is the number of tracks in the implicit grid before the explicit grid.
	NegativeImplicit uint16
	// Explicit is the number of tracks in the explicit grid.
	Explicit uint16
	// PositiveImplicit is the number of tracks in the implicit grid after the explicit grid.
	PositiveImplicit uint16
}

// NewTrackCounts creates a TrackCounts from raw track count numbers.
func NewTrackCounts(negativeImplicit, explicit, positiveImplicit uint16) TrackCounts {
	return TrackCounts{
		NegativeImplicit: negativeImplicit,
		Explicit:         explicit,
		PositiveImplicit: positiveImplicit,
	}
}

// Len returns the total number of tracks in the axis.
func (tc TrackCounts) Len() int {
	return int(tc.NegativeImplicit) + int(tc.Explicit) + int(tc.PositiveImplicit)
}

// ImplicitStartLine returns the OriginZeroLine representing the start of the implicit grid.
func (tc TrackCounts) ImplicitStartLine() OriginZeroLine {
	return OriginZeroLine(-int16(tc.NegativeImplicit))
}

// ImplicitEndLine returns the OriginZeroLine representing the end of the implicit grid.
func (tc TrackCounts) ImplicitEndLine() OriginZeroLine {
	return OriginZeroLine(int16(tc.Explicit) + int16(tc.PositiveImplicit))
}

// --- Conversion functions between OriginZero coordinates and CellOccupancyMatrix track indexes ---

// OzLineToNextTrack converts a grid line in OriginZero coordinates into the track immediately
// following that grid line as an index into the CellOccupancyMatrix.
func (tc TrackCounts) OzLineToNextTrack(index OriginZeroLine) int16 {
	return int16(index) + int16(tc.NegativeImplicit)
}

// OzLineRangeToTrackRange converts start and end grid lines in OriginZero coordinates into
// a range of tracks as indexes into the CellOccupancyMatrix.
func (tc TrackCounts) OzLineRangeToTrackRange(input Line[OriginZeroLine]) (start, end int16) {
	s := tc.OzLineToNextTrack(input.Start)
	e := tc.OzLineToNextTrack(input.End) // Don't subtract 1 as output range is exclusive
	return s, e
}

// TrackToPrevOzLine converts a track as an index into the CellOccupancyMatrix into the grid line
// immediately preceding that track in OriginZero coordinates.
func (tc TrackCounts) TrackToPrevOzLine(index uint16) OriginZeroLine {
	return OriginZeroLine(int16(index) - int16(tc.NegativeImplicit))
}

// TrackRangeToOzLineRange converts a range of tracks as indexes into the CellOccupancyMatrix into
// start and end grid lines in OriginZero coordinates.
func (tc TrackCounts) TrackRangeToOzLineRange(start, end int16) Line[OriginZeroLine] {
	s := tc.TrackToPrevOzLine(uint16(start))
	e := tc.TrackToPrevOzLine(uint16(end)) // Don't add 1 as input range is exclusive
	return Line[OriginZeroLine]{Start: s, End: e}
}

// --- Conversion functions between OriginZero coordinates and GridTrackVec indexes ---

// IntoTrackVecIndex converts a grid line in OriginZero coordinates into the index of that
// same grid line in the GridTrackVec. Panics if out of range.
func (o OriginZeroLine) IntoTrackVecIndex(trackCounts TrackCounts) int {
	idx, ok := o.TryIntoTrackVecIndex(trackCounts)
	if !ok {
		if int16(o) > 0 {
			panic("OriginZero grid line cannot be more than the number of positive grid lines")
		} else {
			panic("OriginZero grid line cannot be less than the number of negative grid lines")
		}
	}
	return idx
}

// TryIntoTrackVecIndex converts a grid line in OriginZero coordinates into the index of that
// same grid line in the GridTrackVec. Returns false if out of range.
// This fallible version is used for absolutely positioned grid items.
func (o OriginZeroLine) TryIntoTrackVecIndex(trackCounts TrackCounts) (int, bool) {
	val := int16(o)
	// OriginZero grid line cannot be less than the number of negative grid lines
	if val < -int16(trackCounts.NegativeImplicit) {
		return 0, false
	}
	// OriginZero grid line cannot be more than the number of positive grid lines
	if val > int16(trackCounts.Explicit)+int16(trackCounts.PositiveImplicit) {
		return 0, false
	}

	return 2 * int(val+int16(trackCounts.NegativeImplicit)), true
}
