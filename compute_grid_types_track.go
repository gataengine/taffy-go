// Package taffy provides grid track types used during grid layout.
// This file is a port of compute/grid/types/grid_track.rs from the Taffy Rust layout engine.
package taffy

import "math"

// gridTrackKind indicates whether a GridTrack represents an actual track or a gutter.
type gridTrackKind int

const (
	// gridTrackKindTrack is an actual track.
	gridTrackKindTrack gridTrackKind = iota
	// gridTrackKindGutter is a gutter (aka grid line / gap).
	gridTrackKindGutter
)

// gridTrack stores internal sizing information for a single grid track (row/column).
// Gutters between tracks are sized similarly to actual tracks, so they are also
// represented by this struct.
type gridTrack struct {
	// kind is whether the track is a full track or a gutter.
	kind gridTrackKind

	// isCollapsed indicates whether the track is collapsed. Collapsed tracks are
	// effectively treated as if they don't exist for sizing purposes.
	isCollapsed bool

	// minTrackSizingFunction is the minimum track sizing function of the track.
	minTrackSizingFunction MinTrackSizingFunction

	// maxTrackSizingFunction is the maximum track sizing function of the track.
	maxTrackSizingFunction MaxTrackSizingFunction

	// offset is the distance of the start of the track from the start of the grid container.
	offset float32

	// baseSize is the size (width/height as applicable) of the track.
	baseSize float32

	// growthLimit is a temporary scratch value when sizing tracks.
	// Note: can be infinity.
	growthLimit float32

	// contentAlignmentAdjustment is a temporary scratch value when sizing tracks. Used as
	// an additional amount to add to the estimate for the available space in the opposite
	// axis when content sizing items.
	contentAlignmentAdjustment float32

	// itemIncurredIncrease is a temporary scratch value when "distributing space" to avoid
	// clobbering planned increase variable.
	itemIncurredIncrease float32

	// baseSizePlannedIncrease is a temporary scratch value when "distributing space" to
	// avoid clobbering the main variable.
	baseSizePlannedIncrease float32

	// growthLimitPlannedIncrease is a temporary scratch value when "distributing space"
	// to avoid clobbering the main variable.
	growthLimitPlannedIncrease float32

	// infinitelyGrowable is a temporary scratch value when "distributing space".
	// See: https://www.w3.org/TR/css3-grid-layout/#infinitely-growable
	infinitelyGrowable bool
}

// newGridTrack creates a new GridTrack representing an actual track (not a gutter).
func newGridTrack(minFunc MinTrackSizingFunction, maxFunc MaxTrackSizingFunction) gridTrack {
	return newGridTrackWithKind(gridTrackKindTrack, minFunc, maxFunc)
}

// newGridTrackWithKind creates a GridTrack with all configuration parameters.
func newGridTrackWithKind(kind gridTrackKind, minFunc MinTrackSizingFunction, maxFunc MaxTrackSizingFunction) gridTrack {
	return gridTrack{
		kind:                       kind,
		isCollapsed:                false,
		minTrackSizingFunction:     minFunc,
		maxTrackSizingFunction:     maxFunc,
		offset:                     0.0,
		baseSize:                   0.0,
		growthLimit:                0.0,
		contentAlignmentAdjustment: 0.0,
		itemIncurredIncrease:       0.0,
		baseSizePlannedIncrease:    0.0,
		growthLimitPlannedIncrease: 0.0,
		infinitelyGrowable:         false,
	}
}

// gutter creates a new GridTrack representing a gutter.
func gridTrackGutter(size LengthPercentage) gridTrack {
	return newGridTrackWithKind(
		gridTrackKindGutter,
		MinTrackSizingFunctionFromLengthPercentage(size),
		MaxTrackSizingFunctionFromLengthPercentage(size),
	)
}

// collapse marks a GridTrack as collapsed. Also sets both of the track's sizing
// functions to fixed zero-sized sizing functions.
func (t *gridTrack) collapse() {
	t.isCollapsed = true
	t.minTrackSizingFunction = MinTrackSizingFunction{Tag: MinTrackLength, Value: 0}
	t.maxTrackSizingFunction = MaxTrackSizingFunction{Tag: MaxTrackLength, Value: 0}
}

// isFlexible returns true if the track has a Flex MaxTrackSizingFunction.
func (t *gridTrack) isFlexible() bool {
	return t.maxTrackSizingFunction.IsFr()
}

// usesPercentage returns true if the track uses percentage sizing.
func (t *gridTrack) usesPercentage() bool {
	return t.minTrackSizingFunction.UsesPercentage() || t.maxTrackSizingFunction.UsesPercentage()
}

// hasIntrinsicSizingFunction returns true if the track has an intrinsic min and/or max sizing function.
func (t *gridTrack) hasIntrinsicSizingFunction() bool {
	return t.minTrackSizingFunction.IsIntrinsic() || t.maxTrackSizingFunction.IsIntrinsic()
}

// fitContentLimit returns the fit-content limit for this track.
func (t *gridTrack) fitContentLimit(axisAvailableGridSpace Option[float32]) float32 {
	switch t.maxTrackSizingFunction.Tag {
	case MaxTrackFitContentPx:
		return t.maxTrackSizingFunction.Value
	case MaxTrackFitContentPercent:
		if axisAvailableGridSpace.IsSome {
			return axisAvailableGridSpace.Value * t.maxTrackSizingFunction.Value
		}
		return float32(math.Inf(1))
	default:
		return float32(math.Inf(1))
	}
}

// fitContentLimitedGrowthLimit returns the growth limit clamped by the fit-content limit.
func (t *gridTrack) fitContentLimitedGrowthLimit(axisAvailableGridSpace Option[float32]) float32 {
	limit := t.fitContentLimit(axisAvailableGridSpace)
	if t.growthLimit < limit {
		return t.growthLimit
	}
	return limit
}

// flexFactor returns the track's flex factor if it is a flex track, else 0.
func (t *gridTrack) flexFactor() float32 {
	if t.maxTrackSizingFunction.IsFr() {
		return t.maxTrackSizingFunction.Value
	}
	return 0.0
}
