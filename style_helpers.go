// Package taffy provides helper/constructor functions for style types.
// This file is a port of style_helpers.rs from the Taffy Rust layout engine.
package taffy

// ---------------------------------------------------------------------------
// LengthPercentage constructors
// ---------------------------------------------------------------------------

// ZeroLP returns a zero LengthPercentage (absolute length of 0).
func ZeroLP() LengthPercentage {
	return LengthPercentage{Tag: LPLength, Value: 0}
}

// LengthLP creates a LengthPercentage from an absolute length value.
func LengthLP(val float32) LengthPercentage {
	return LPFromLength(val)
}

// PercentLP creates a LengthPercentage from a percentage value (0.0–1.0).
func PercentLP(val float32) LengthPercentage {
	return LPFromPercent(val)
}

// ---------------------------------------------------------------------------
// LengthPercentageAuto constructors
// ---------------------------------------------------------------------------

// ZeroLPA returns a zero LengthPercentageAuto (absolute length of 0).
func ZeroLPA() LengthPercentageAuto {
	return LengthPercentageAuto{Tag: LPALength, Value: 0}
}

// AutoLPA returns the Auto variant of LengthPercentageAuto.
func AutoLPA() LengthPercentageAuto {
	return LengthPercentageAuto{Tag: LPAAuto, Value: 0}
}

// LengthLPA creates a LengthPercentageAuto from an absolute length value.
func LengthLPA(val float32) LengthPercentageAuto {
	return LPAFromLength(val)
}

// PercentLPA creates a LengthPercentageAuto from a percentage value (0.0–1.0).
func PercentLPA(val float32) LengthPercentageAuto {
	return LPAFromPercent(val)
}

// ---------------------------------------------------------------------------
// Dimension constructors
// ---------------------------------------------------------------------------

// ZeroDimension returns a zero Dimension (absolute length of 0).
func ZeroDimension() Dimension {
	return Dimension{Tag: DimensionLength, Value: 0}
}

// AutoDimension returns the Auto variant of Dimension.
func AutoDimension() Dimension {
	return Dimension{Tag: DimensionAuto, Value: 0}
}

// LengthDimension creates a Dimension from an absolute length value.
func LengthDimension(val float32) Dimension {
	return DimensionFromLength(val)
}

// PercentDimension creates a Dimension from a percentage value (0.0–1.0).
func PercentDimension(val float32) Dimension {
	return DimensionFromPercent(val)
}

// ---------------------------------------------------------------------------
// AvailableSpace constructors
// ---------------------------------------------------------------------------

// AvailableSpaceFromLength creates a definite AvailableSpace from a pixel length value.
func AvailableSpaceFromLength(val float32) AvailableSpace {
	return AvailableSpaceFromDefinite(val)
}

// AvailableSpaceFromMinContent returns the MinContent variant of AvailableSpace.
func AvailableSpaceFromMinContent() AvailableSpace {
	return AvailableSpaceMinContentValue
}

// AvailableSpaceFromMaxContent returns the MaxContent variant of AvailableSpace.
func AvailableSpaceFromMaxContent() AvailableSpace {
	return AvailableSpaceMaxContentValue
}

// ---------------------------------------------------------------------------
// GridPlacement constructors
// ---------------------------------------------------------------------------

// GridLine_ creates a GridPlacement::Line with the given CSS Grid Line index.
// Positive indices count from the start, negative from the end. Zero is invalid.
func GridLine_(index int16) GridPlacement {
	return GridPlacementFromLine(index)
}

// GridSpan creates a GridPlacement::Span making the item span the given number of tracks.
func GridSpan(span uint16) GridPlacement {
	return GridPlacementFromSpan(span)
}

// GridNamedLine creates a GridPlacement::NamedLine at the nth occurrence of the named line.
func GridNamedLine(name string, index int16) GridPlacement {
	return GridPlacementFromNamedLine(name, index)
}

// ---------------------------------------------------------------------------
// TrackSizingFunction constructors
// ---------------------------------------------------------------------------

// GridMinMax creates a TrackSizingFunction with the given min and max sizing functions.
// This is the CSS minmax() function.
func GridMinMax(min MinTrackSizingFunction, max MaxTrackSizingFunction) TrackSizingFunction {
	return TrackSizingFunction{Min: min, Max: max}
}

// GridFr creates a fractional TrackSizingFunction equivalent to the CSS fr unit.
// It is shorthand for minmax(0, Nfr) — the minimum is a zero-length, not auto.
func GridFr(flex float32) TrackSizingFunction {
	return TrackSizingFunction{
		Min: MinTrackSizingFunction{Tag: MinTrackLength, Value: 0},
		Max: MaxTrackFromFr(flex),
	}
}

// ---------------------------------------------------------------------------
// MinTrackSizingFunction constructors
// ---------------------------------------------------------------------------

// MinContentTrackMin returns the MinContent variant of MinTrackSizingFunction.
func MinContentTrackMin() MinTrackSizingFunction {
	return MinTrackMinContentVal
}

// MaxContentTrackMin returns the MaxContent variant of MinTrackSizingFunction.
func MaxContentTrackMin() MinTrackSizingFunction {
	return MinTrackMaxContentVal
}

// ---------------------------------------------------------------------------
// MaxTrackSizingFunction constructors
// ---------------------------------------------------------------------------

// MinContentTrackMax returns the MinContent variant of MaxTrackSizingFunction.
func MinContentTrackMax() MaxTrackSizingFunction {
	return MaxTrackMinContentVal
}

// MaxContentTrackMax returns the MaxContent variant of MaxTrackSizingFunction.
func MaxContentTrackMax() MaxTrackSizingFunction {
	return MaxTrackMaxContentVal
}

// FitContentPxTrackMax creates a fit-content(px) MaxTrackSizingFunction with the given pixel limit.
func FitContentPxTrackMax(limit float32) MaxTrackSizingFunction {
	return MaxTrackFitContentPxVal(limit)
}

// FitContentPercentTrackMax creates a fit-content(%) MaxTrackSizingFunction with the given percent limit (0.0–1.0).
func FitContentPercentTrackMax(limit float32) MaxTrackSizingFunction {
	return MaxTrackFitContentPercentVal(limit)
}

// FrTrackMax creates a fractional MaxTrackSizingFunction from the given flex factor.
func FrTrackMax(flex float32) MaxTrackSizingFunction {
	return MaxTrackFromFr(flex)
}

// ---------------------------------------------------------------------------
// GridTemplateComponent constructors
// ---------------------------------------------------------------------------

// GridRepeat creates a repeat() GridTemplateComponent with the given repetition count and tracks.
func GridRepeat(count RepetitionCount, tracks []TrackSizingFunction) GridTemplateComponent {
	return GridTemplateRepeatVal(GridTemplateRepetition{
		Count:     count,
		Tracks:    tracks,
		LineNames: [][]string{},
	})
}

// EvenlySpacedTracks returns a grid template definition consisting of count evenly sized fr tracks.
// Equivalent to [repeat(count, [1fr])].
func EvenlySpacedTracks(count uint16) []GridTemplateComponent {
	repeatedTracks := []TrackSizingFunction{GridFr(1.0)}
	return []GridTemplateComponent{
		GridRepeat(RepetitionCountFromU16(count), repeatedTracks),
	}
}
