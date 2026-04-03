// Package taffy provides grid layout style types.
// This file is a port of grid.rs from the Taffy Rust layout engine.
package taffy

import "math"

// CalcResolver is a function that resolves a calc() expression given a calc ID and a basis value.
type CalcResolver = func(calcId uint64, basis float32) float32

// ---------------------------------------------------------------------------
// GridAutoFlow
// ---------------------------------------------------------------------------

// GridAutoFlow controls whether grid items are placed row-wise or column-wise,
// and whether the sparse or dense packing algorithm is used.
// See: https://developer.mozilla.org/en-US/docs/Web/CSS/grid-auto-flow
type GridAutoFlow int

const (
	// GridAutoFlowRow places items by filling each row in turn, adding new rows as necessary.
	GridAutoFlowRow GridAutoFlow = iota
	// GridAutoFlowColumn places items by filling each column in turn, adding new columns as necessary.
	GridAutoFlowColumn
	// GridAutoFlowRowDense combines Row with the dense packing algorithm.
	GridAutoFlowRowDense
	// GridAutoFlowColumnDense combines Column with the dense packing algorithm.
	GridAutoFlowColumnDense
)

// IsDense reports whether the dense packing algorithm is used.
func (g GridAutoFlow) IsDense() bool {
	return g == GridAutoFlowRowDense || g == GridAutoFlowColumnDense
}

// PrimaryAxis returns the AbsoluteAxis that grid auto placement fills first.
func (g GridAutoFlow) PrimaryAxis() AbsoluteAxis {
	switch g {
	case GridAutoFlowColumn, GridAutoFlowColumnDense:
		return AbsoluteAxisVertical
	default:
		return AbsoluteAxisHorizontal
	}
}

// ---------------------------------------------------------------------------
// GridCoordinate interface and coordinate types
// ---------------------------------------------------------------------------

// gridCoordinate is a marker interface for types that can be used as grid line coordinates.
type gridCoordinate interface {
	isGridCoordinate()
}

// GridLine represents a grid line position in CSS Grid Line coordinates:
//   - The left/top edge of the explicit grid is line 1 (counts up)
//   - The right/bottom edge of the explicit grid is -1 (counts down)
//   - 0 is invalid
type GridLine int16

func (GridLine) isGridCoordinate() {}

// AsI16 returns the underlying int16 value.
func (g GridLine) AsI16() int16 {
	return int16(g)
}

// IntoOriginZeroLine converts this GridLine into OriginZero coordinates using the explicit track count.
func (g GridLine) IntoOriginZeroLine(explicitTrackCount uint16) OriginZeroLine {
	explicitLineCount := explicitTrackCount + 1
	val := int16(g)
	var oz int16
	if val > 0 {
		oz = val - 1
	} else if val < 0 {
		oz = val + int16(explicitLineCount)
	} else {
		panic("Grid line of zero is invalid")
	}
	return OriginZeroLine(oz)
}

// OriginZeroLine represents a grid line position in normalized "OriginZero" coordinates:
//   - The left/top edge of the explicit grid is 0
//   - The next line right/down is 1, etc.
//   - The next line left/up is -1, etc.
type OriginZeroLine int16

func (OriginZeroLine) isGridCoordinate() {}

// Add adds two OriginZeroLine values.
func (o OriginZeroLine) Add(rhs OriginZeroLine) OriginZeroLine {
	return OriginZeroLine(int16(o) + int16(rhs))
}

// Sub subtracts two OriginZeroLine values.
func (o OriginZeroLine) Sub(rhs OriginZeroLine) OriginZeroLine {
	return OriginZeroLine(int16(o) - int16(rhs))
}

// AddU16 adds a u16 offset to an OriginZeroLine.
func (o OriginZeroLine) AddU16(rhs uint16) OriginZeroLine {
	return OriginZeroLine(int16(o) + int16(rhs))
}

// SubU16 subtracts a u16 offset from an OriginZeroLine.
func (o OriginZeroLine) SubU16(rhs uint16) OriginZeroLine {
	return OriginZeroLine(int16(o) - int16(rhs))
}

// ImpliedNegativeImplicitTracks returns the minimum number of negative implicit tracks
// required if a grid item starts at this line.
func (o OriginZeroLine) ImpliedNegativeImplicitTracks() uint16 {
	if int16(o) < 0 {
		return uint16(-int16(o))
	}
	return 0
}

// ImpliedPositiveImplicitTracks returns the minimum number of positive implicit tracks
// required if a grid item ends at this line.
func (o OriginZeroLine) ImpliedPositiveImplicitTracks(explicitTrackCount uint16) uint16 {
	if int16(o) > int16(explicitTrackCount) {
		return uint16(int16(o)) - explicitTrackCount
	}
	return 0
}

// OriginZeroLineSpan returns the number of tracks between start and end lines.
func OriginZeroLineSpan(l Line[OriginZeroLine]) uint16 {
	diff := int16(l.End) - int16(l.Start)
	if diff < 0 {
		return 0
	}
	return uint16(diff)
}

// ---------------------------------------------------------------------------
// GenericGridPlacement
// ---------------------------------------------------------------------------

// GenericGridPlacementTag identifies the variant of a GenericGridPlacement.
type GenericGridPlacementTag int

const (
	// GenericGridPlacementAuto places the item according to the auto-placement algorithm.
	GenericGridPlacementAuto GenericGridPlacementTag = iota
	// GenericGridPlacementLine places the item at a specific line index.
	GenericGridPlacementLine
	// GenericGridPlacementSpan makes the item span a specified number of tracks.
	GenericGridPlacementSpan
)

// GenericGridPlacement is a grid line placement specification generic over the coordinate system.
// GenericGridPlacement[GridLine] is aliased as NonNamedGridPlacement.
// GenericGridPlacement[OriginZeroLine] is aliased as OriginZeroGridPlacement.
type GenericGridPlacement[T gridCoordinate] struct {
	Tag  GenericGridPlacementTag
	Line T
	Span uint16
}

// GenericGridPlacementAutoVal returns the Auto variant.
func GenericGridPlacementAutoVal[T gridCoordinate]() GenericGridPlacement[T] {
	return GenericGridPlacement[T]{Tag: GenericGridPlacementAuto}
}

// GenericGridPlacementLineVal returns the Line variant with a given line index.
func GenericGridPlacementLineVal[T gridCoordinate](line T) GenericGridPlacement[T] {
	return GenericGridPlacement[T]{Tag: GenericGridPlacementLine, Line: line}
}

// GenericGridPlacementSpanVal returns the Span variant.
func GenericGridPlacementSpanVal[T gridCoordinate](span uint16) GenericGridPlacement[T] {
	return GenericGridPlacement[T]{Tag: GenericGridPlacementSpan, Span: span}
}

// IndefiniteSpan resolves the span for an indefinite placement.
// Panics if called on a definite placement (both ends are Lines).
func IndefiniteSpan[T gridCoordinate](l Line[GenericGridPlacement[T]]) uint16 {
	start := l.Start
	end := l.End
	switch {
	case start.Tag == GenericGridPlacementLine && end.Tag == GenericGridPlacementAuto:
		return 1
	case start.Tag == GenericGridPlacementAuto && end.Tag == GenericGridPlacementLine:
		return 1
	case start.Tag == GenericGridPlacementAuto && end.Tag == GenericGridPlacementAuto:
		return 1
	case start.Tag == GenericGridPlacementLine && end.Tag == GenericGridPlacementSpan:
		return end.Span
	case start.Tag == GenericGridPlacementSpan && end.Tag == GenericGridPlacementLine:
		return start.Span
	case start.Tag == GenericGridPlacementSpan && end.Tag == GenericGridPlacementAuto:
		return start.Span
	case start.Tag == GenericGridPlacementAuto && end.Tag == GenericGridPlacementSpan:
		return end.Span
	case start.Tag == GenericGridPlacementSpan && end.Tag == GenericGridPlacementSpan:
		return start.Span
	default:
		panic("indefiniteSpan should only be called on indefinite grid tracks")
	}
}

// OriginZeroGridPlacement is a GenericGridPlacement using OriginZero coordinates.
type OriginZeroGridPlacement = GenericGridPlacement[OriginZeroLine]

// NonNamedGridPlacement is a GenericGridPlacement using CSS GridLine coordinates (no named lines).
type NonNamedGridPlacement = GenericGridPlacement[GridLine]

// IsDefiniteOriginZero reports whether the placement is definite (has at least one Line position).
func IsDefiniteOriginZero(l Line[OriginZeroGridPlacement]) bool {
	return l.Start.Tag == GenericGridPlacementLine || l.End.Tag == GenericGridPlacementLine
}

// IsDefiniteNonNamed reports whether the NonNamed placement is definite.
func IsDefiniteNonNamed(l Line[NonNamedGridPlacement]) bool {
	start := l.Start
	end := l.End
	if start.Tag == GenericGridPlacementLine && start.Line.AsI16() != 0 {
		return true
	}
	if end.Tag == GenericGridPlacementLine && end.Line.AsI16() != 0 {
		return true
	}
	return false
}

// IntoOriginZeroPlacement converts a NonNamedGridPlacement to an OriginZeroGridPlacement.
func IntoOriginZeroPlacement(p NonNamedGridPlacement, explicitTrackCount uint16) OriginZeroGridPlacement {
	switch p.Tag {
	case GenericGridPlacementAuto:
		return GenericGridPlacement[OriginZeroLine]{Tag: GenericGridPlacementAuto}
	case GenericGridPlacementSpan:
		return GenericGridPlacement[OriginZeroLine]{Tag: GenericGridPlacementSpan, Span: p.Span}
	case GenericGridPlacementLine:
		if p.Line.AsI16() == 0 {
			return GenericGridPlacement[OriginZeroLine]{Tag: GenericGridPlacementAuto}
		}
		return GenericGridPlacement[OriginZeroLine]{
			Tag:  GenericGridPlacementLine,
			Line: p.Line.IntoOriginZeroLine(explicitTrackCount),
		}
	default:
		return GenericGridPlacement[OriginZeroLine]{Tag: GenericGridPlacementAuto}
	}
}

// IntoOriginZeroNonNamed converts a Line[NonNamedGridPlacement] to a Line[OriginZeroGridPlacement].
func IntoOriginZeroNonNamed(l Line[NonNamedGridPlacement], explicitTrackCount uint16) Line[OriginZeroGridPlacement] {
	return Line[OriginZeroGridPlacement]{
		Start: IntoOriginZeroPlacement(l.Start, explicitTrackCount),
		End:   IntoOriginZeroPlacement(l.End, explicitTrackCount),
	}
}

// ResolveDefiniteGridLines resolves an OriginZero placement where at least one end is a Line.
func ResolveDefiniteGridLines(l Line[OriginZeroGridPlacement]) Line[OriginZeroLine] {
	start := l.Start
	end := l.End
	switch {
	case start.Tag == GenericGridPlacementLine && end.Tag == GenericGridPlacementLine:
		line1, line2 := start.Line, end.Line
		if line1 == line2 {
			return Line[OriginZeroLine]{Start: line1, End: line1.AddU16(1)}
		}
		if int16(line1) < int16(line2) {
			return Line[OriginZeroLine]{Start: line1, End: line2}
		}
		return Line[OriginZeroLine]{Start: line2, End: line1}
	case start.Tag == GenericGridPlacementLine && end.Tag == GenericGridPlacementSpan:
		return Line[OriginZeroLine]{Start: start.Line, End: start.Line.AddU16(end.Span)}
	case start.Tag == GenericGridPlacementLine && end.Tag == GenericGridPlacementAuto:
		return Line[OriginZeroLine]{Start: start.Line, End: start.Line.AddU16(1)}
	case start.Tag == GenericGridPlacementSpan && end.Tag == GenericGridPlacementLine:
		return Line[OriginZeroLine]{Start: end.Line.SubU16(start.Span), End: end.Line}
	case start.Tag == GenericGridPlacementAuto && end.Tag == GenericGridPlacementLine:
		return Line[OriginZeroLine]{Start: end.Line.SubU16(1), End: end.Line}
	default:
		panic("ResolveDefiniteGridLines should only be called on definite grid tracks")
	}
}

// ResolveAbsolutelyPositionedGridTracks resolves placement for absolutely positioned items.
func ResolveAbsolutelyPositionedGridTracks(l Line[OriginZeroGridPlacement]) Line[Option[OriginZeroLine]] {
	start := l.Start
	end := l.End
	switch {
	case start.Tag == GenericGridPlacementLine && end.Tag == GenericGridPlacementLine:
		t1, t2 := start.Line, end.Line
		if t1 == t2 {
			return Line[Option[OriginZeroLine]]{Start: Some(t1), End: Some(t1.AddU16(1))}
		}
		if int16(t1) < int16(t2) {
			return Line[Option[OriginZeroLine]]{Start: Some(t1), End: Some(t2)}
		}
		return Line[Option[OriginZeroLine]]{Start: Some(t2), End: Some(t1)}
	case start.Tag == GenericGridPlacementLine && end.Tag == GenericGridPlacementSpan:
		return Line[Option[OriginZeroLine]]{Start: Some(start.Line), End: Some(start.Line.AddU16(end.Span))}
	case start.Tag == GenericGridPlacementLine && end.Tag == GenericGridPlacementAuto:
		return Line[Option[OriginZeroLine]]{Start: Some(start.Line), End: None[OriginZeroLine]()}
	case start.Tag == GenericGridPlacementSpan && end.Tag == GenericGridPlacementLine:
		return Line[Option[OriginZeroLine]]{Start: Some(end.Line.SubU16(start.Span)), End: Some(end.Line)}
	case start.Tag == GenericGridPlacementAuto && end.Tag == GenericGridPlacementLine:
		return Line[Option[OriginZeroLine]]{Start: None[OriginZeroLine](), End: Some(end.Line)}
	default:
		return Line[Option[OriginZeroLine]]{Start: None[OriginZeroLine](), End: None[OriginZeroLine]()}
	}
}

// ResolveIndefiniteGridTracks resolves an indefinite placement given an external start position.
func ResolveIndefiniteGridTracks(l Line[OriginZeroGridPlacement], start OriginZeroLine) Line[OriginZeroLine] {
	s := l.Start
	e := l.End
	switch {
	case s.Tag == GenericGridPlacementAuto && e.Tag == GenericGridPlacementAuto:
		return Line[OriginZeroLine]{Start: start, End: start.AddU16(1)}
	case s.Tag == GenericGridPlacementSpan && e.Tag == GenericGridPlacementAuto:
		return Line[OriginZeroLine]{Start: start, End: start.AddU16(s.Span)}
	case s.Tag == GenericGridPlacementAuto && e.Tag == GenericGridPlacementSpan:
		return Line[OriginZeroLine]{Start: start, End: start.AddU16(e.Span)}
	case s.Tag == GenericGridPlacementSpan && e.Tag == GenericGridPlacementSpan:
		return Line[OriginZeroLine]{Start: start, End: start.AddU16(s.Span)}
	default:
		panic("ResolveIndefiniteGridTracks should only be called on indefinite grid tracks")
	}
}

// ---------------------------------------------------------------------------
// GridPlacement (the public user-facing type with named line support)
// ---------------------------------------------------------------------------

// GridPlacementTag identifies the variant of a GridPlacement.
type GridPlacementTag int

const (
	// GridPlacementAuto places the item via the auto-placement algorithm.
	GridPlacementAuto GridPlacementTag = iota
	// GridPlacementLine places the item at a specific GridLine index.
	GridPlacementLine
	// GridPlacementNamedLine places the item at a named grid line.
	GridPlacementNamedLine
	// GridPlacementSpan makes the item span a specific number of tracks.
	GridPlacementSpan
	// GridPlacementNamedSpan makes the item span until the nth named line.
	GridPlacementNamedSpan
)

// GridPlacement is a grid line placement specification supporting named lines.
// Defaults to GridPlacementAuto.
type GridPlacement struct {
	Tag       GridPlacementTag
	LineIndex int16  // used for Line and NamedLine variants
	SpanSize  uint16 // used for Span and NamedSpan variants
	LineName  string // used for NamedLine and NamedSpan variants
}

// GridPlacementAutoVal is the Auto variant of GridPlacement.
var GridPlacementAutoVal = GridPlacement{Tag: GridPlacementAuto}

// GridPlacementFromLine creates a GridPlacement::Line.
func GridPlacementFromLine(index int16) GridPlacement {
	return GridPlacement{Tag: GridPlacementLine, LineIndex: index}
}

// GridPlacementFromNamedLine creates a GridPlacement::NamedLine.
func GridPlacementFromNamedLine(name string, index int16) GridPlacement {
	return GridPlacement{Tag: GridPlacementNamedLine, LineName: name, LineIndex: index}
}

// GridPlacementFromSpan creates a GridPlacement::Span.
func GridPlacementFromSpan(span uint16) GridPlacement {
	return GridPlacement{Tag: GridPlacementSpan, SpanSize: span}
}

// GridPlacementFromNamedSpan creates a GridPlacement::NamedSpan.
func GridPlacementFromNamedSpan(name string, span uint16) GridPlacement {
	return GridPlacement{Tag: GridPlacementNamedSpan, LineName: name, SpanSize: span}
}

// IntoOriginZeroPlacementIgnoringNamed converts a GridPlacement to an OriginZeroGridPlacement,
// ignoring named lines (treating them as Auto).
func (p GridPlacement) IntoOriginZeroPlacementIgnoringNamed(explicitTrackCount uint16) OriginZeroGridPlacement {
	switch p.Tag {
	case GridPlacementAuto:
		return GenericGridPlacement[OriginZeroLine]{Tag: GenericGridPlacementAuto}
	case GridPlacementSpan:
		return GenericGridPlacement[OriginZeroLine]{Tag: GenericGridPlacementSpan, Span: p.SpanSize}
	case GridPlacementLine:
		if p.LineIndex == 0 {
			return GenericGridPlacement[OriginZeroLine]{Tag: GenericGridPlacementAuto}
		}
		line := GridLine(p.LineIndex)
		return GenericGridPlacement[OriginZeroLine]{
			Tag:  GenericGridPlacementLine,
			Line: line.IntoOriginZeroLine(explicitTrackCount),
		}
	default: // NamedLine and NamedSpan resolve to Auto
		return GenericGridPlacement[OriginZeroLine]{Tag: GenericGridPlacementAuto}
	}
}

// IsDefiniteGridPlacement reports whether the Line[GridPlacement] is definite.
// The placement is definite if at least one end is a non-zero Line or a NamedLine.
func IsDefiniteGridPlacement(l Line[GridPlacement]) bool {
	start, end := l.Start, l.End
	if start.Tag == GridPlacementLine && start.LineIndex != 0 {
		return true
	}
	if end.Tag == GridPlacementLine && end.LineIndex != 0 {
		return true
	}
	if start.Tag == GridPlacementNamedLine || end.Tag == GridPlacementNamedLine {
		return true
	}
	return false
}

// IntoOriginZeroIgnoringNamed converts a Line[GridPlacement] to a Line[OriginZeroGridPlacement],
// ignoring named lines.
func IntoOriginZeroIgnoringNamed(l Line[GridPlacement], explicitTrackCount uint16) Line[OriginZeroGridPlacement] {
	return Line[OriginZeroGridPlacement]{
		Start: l.Start.IntoOriginZeroPlacementIgnoringNamed(explicitTrackCount),
		End:   l.End.IntoOriginZeroPlacementIgnoringNamed(explicitTrackCount),
	}
}

// DefaultGridPlacementLine is the default Line[GridPlacement] (both ends Auto).
var DefaultGridPlacementLine = Line[GridPlacement]{
	Start: GridPlacementAutoVal,
	End:   GridPlacementAutoVal,
}

// ---------------------------------------------------------------------------
// MinTrackSizingFunction
// ---------------------------------------------------------------------------

// MinTrackSizingTag identifies the variant of a MinTrackSizingFunction.
type MinTrackSizingTag int

const (
	// MinTrackLength is an absolute length track size.
	MinTrackLength MinTrackSizingTag = iota
	// MinTrackPercent is a percentage track size.
	MinTrackPercent
	// MinTrackAuto is the auto minimum track size.
	MinTrackAuto
	// MinTrackMinContent is the min-content minimum track size.
	MinTrackMinContent
	// MinTrackMaxContent is the max-content minimum track size.
	MinTrackMaxContent
	// MinTrackCalc is a calc() minimum track size.
	MinTrackCalc
)

// MinTrackSizingFunction specifies the minimum size of a grid track.
type MinTrackSizingFunction struct {
	Tag   MinTrackSizingTag
	Value float32 // used for Length, Percent, and Calc (stored as float32 bits of uint32(calcId))
}

// MinTrackFromLength creates a MinTrackSizingFunction from an absolute length.
func MinTrackFromLength(val float32) MinTrackSizingFunction {
	return MinTrackSizingFunction{Tag: MinTrackLength, Value: val}
}

// MinTrackFromPercent creates a MinTrackSizingFunction from a percentage (0.0-1.0).
func MinTrackFromPercent(val float32) MinTrackSizingFunction {
	return MinTrackSizingFunction{Tag: MinTrackPercent, Value: val}
}

// MinTrackAutoVal is the Auto variant of MinTrackSizingFunction.
var MinTrackAutoVal = MinTrackSizingFunction{Tag: MinTrackAuto}

// MinTrackMinContentVal is the MinContent variant.
var MinTrackMinContentVal = MinTrackSizingFunction{Tag: MinTrackMinContent}

// MinTrackMaxContentVal is the MaxContent variant.
var MinTrackMaxContentVal = MinTrackSizingFunction{Tag: MinTrackMaxContent}

// MinTrackFromCalc creates a MinTrackSizingFunction from a calc() ID.
func MinTrackFromCalc(id uint64) MinTrackSizingFunction {
	return MinTrackSizingFunction{Tag: MinTrackCalc, Value: math.Float32frombits(uint32(id))}
}

// IsIntrinsic reports whether the min track sizing function is MinContent, MaxContent, or Auto.
func (m MinTrackSizingFunction) IsIntrinsic() bool {
	switch m.Tag {
	case MinTrackAuto, MinTrackMinContent, MinTrackMaxContent:
		return true
	}
	return false
}

// IsMinOrMaxContent reports whether the min track sizing function is MinContent or MaxContent.
func (m MinTrackSizingFunction) IsMinOrMaxContent() bool {
	return m.Tag == MinTrackMinContent || m.Tag == MinTrackMaxContent
}

// IsFr reports whether the min track sizing function is an fr value. Always false for min tracks.
func (m MinTrackSizingFunction) IsFr() bool {
	return false
}

// IsAuto reports whether the min track sizing function is Auto.
func (m MinTrackSizingFunction) IsAuto() bool {
	return m.Tag == MinTrackAuto
}

// IsMinContent reports whether the min track sizing function is MinContent.
func (m MinTrackSizingFunction) IsMinContent() bool {
	return m.Tag == MinTrackMinContent
}

// IsMaxContent reports whether the min track sizing function is MaxContent.
func (m MinTrackSizingFunction) IsMaxContent() bool {
	return m.Tag == MinTrackMaxContent
}

// DefiniteValue resolves this min track sizing function to a concrete value if possible.
// Returns None for intrinsic sizing functions.
func (m MinTrackSizingFunction) DefiniteValue(parentSize Option[float32], calcResolver CalcResolver) Option[float32] {
	switch m.Tag {
	case MinTrackLength:
		return Some(m.Value)
	case MinTrackPercent:
		if parentSize.IsSome {
			return Some(m.Value * parentSize.Value)
		}
		return None[float32]()
	case MinTrackCalc:
		if parentSize.IsSome && calcResolver != nil {
			id := uint64(math.Float32bits(m.Value))
			return Some(calcResolver(id, parentSize.Value))
		}
		return None[float32]()
	default:
		return None[float32]()
	}
}

// ResolvedPercentageSize resolves percentage values against the parent size.
// Non-percentage values return None.
func (m MinTrackSizingFunction) ResolvedPercentageSize(parentSize float32, calcResolver CalcResolver) Option[float32] {
	switch m.Tag {
	case MinTrackPercent:
		return Some(m.Value * parentSize)
	case MinTrackCalc:
		if calcResolver != nil {
			id := uint64(math.Float32bits(m.Value))
			return Some(calcResolver(id, parentSize))
		}
		return None[float32]()
	default:
		return None[float32]()
	}
}

// UsesPercentage reports whether the min track sizing function depends on the parent node size.
func (m MinTrackSizingFunction) UsesPercentage() bool {
	return m.Tag == MinTrackPercent || m.Tag == MinTrackCalc
}

// MinTrackSizingFunctionFromLengthPercentage converts a LengthPercentage to a MinTrackSizingFunction.
func MinTrackSizingFunctionFromLengthPercentage(lp LengthPercentage) MinTrackSizingFunction {
	switch lp.Tag {
	case LPLength:
		return MinTrackSizingFunction{Tag: MinTrackLength, Value: lp.Value}
	case LPPercent:
		return MinTrackSizingFunction{Tag: MinTrackPercent, Value: lp.Value}
	default: // LPCalc
		return MinTrackSizingFunction{Tag: MinTrackCalc, Value: lp.Value}
	}
}

// MinTrackSizingFunctionFromLengthPercentageAuto converts a LengthPercentageAuto to a MinTrackSizingFunction.
func MinTrackSizingFunctionFromLengthPercentageAuto(lpa LengthPercentageAuto) MinTrackSizingFunction {
	switch lpa.Tag {
	case LPALength:
		return MinTrackSizingFunction{Tag: MinTrackLength, Value: lpa.Value}
	case LPAPercent:
		return MinTrackSizingFunction{Tag: MinTrackPercent, Value: lpa.Value}
	case LPAAuto:
		return MinTrackAutoVal
	default: // LPACalc
		return MinTrackSizingFunction{Tag: MinTrackCalc, Value: lpa.Value}
	}
}

// MinTrackSizingFunctionFromDimension converts a Dimension to a MinTrackSizingFunction.
func MinTrackSizingFunctionFromDimension(d Dimension) MinTrackSizingFunction {
	switch d.Tag {
	case DimensionLength:
		return MinTrackSizingFunction{Tag: MinTrackLength, Value: d.Value}
	case DimensionPercent:
		return MinTrackSizingFunction{Tag: MinTrackPercent, Value: d.Value}
	case DimensionAuto:
		return MinTrackAutoVal
	default: // DimensionCalc
		return MinTrackSizingFunction{Tag: MinTrackCalc, Value: d.Value}
	}
}

// ---------------------------------------------------------------------------
// MaxTrackSizingFunction
// ---------------------------------------------------------------------------

// MaxTrackSizingTag identifies the variant of a MaxTrackSizingFunction.
type MaxTrackSizingTag int

const (
	// MaxTrackLength is an absolute length track size.
	MaxTrackLength MaxTrackSizingTag = iota
	// MaxTrackPercent is a percentage track size.
	MaxTrackPercent
	// MaxTrackAuto is the auto maximum track size.
	MaxTrackAuto
	// MaxTrackMinContent is the min-content maximum track size.
	MaxTrackMinContent
	// MaxTrackMaxContent is the max-content maximum track size.
	MaxTrackMaxContent
	// MaxTrackFitContentPx is a fit-content(px) maximum track size.
	MaxTrackFitContentPx
	// MaxTrackFitContentPercent is a fit-content(%) maximum track size.
	MaxTrackFitContentPercent
	// MaxTrackFr is a fractional maximum track size.
	MaxTrackFr
	// MaxTrackCalc is a calc() maximum track size.
	MaxTrackCalc
)

// MaxTrackSizingFunction specifies the maximum size of a grid track.
type MaxTrackSizingFunction struct {
	Tag   MaxTrackSizingTag
	Value float32 // used for Length, Percent, FitContentPx, FitContentPercent, Fr, Calc
}

// MaxTrackFromLength creates a MaxTrackSizingFunction from an absolute length.
func MaxTrackFromLength(val float32) MaxTrackSizingFunction {
	return MaxTrackSizingFunction{Tag: MaxTrackLength, Value: val}
}

// MaxTrackFromPercent creates a MaxTrackSizingFunction from a percentage (0.0-1.0).
func MaxTrackFromPercent(val float32) MaxTrackSizingFunction {
	return MaxTrackSizingFunction{Tag: MaxTrackPercent, Value: val}
}

// MaxTrackAutoVal is the Auto variant of MaxTrackSizingFunction.
var MaxTrackAutoVal = MaxTrackSizingFunction{Tag: MaxTrackAuto}

// MaxTrackMinContentVal is the MinContent variant.
var MaxTrackMinContentVal = MaxTrackSizingFunction{Tag: MaxTrackMinContent}

// MaxTrackMaxContentVal is the MaxContent variant.
var MaxTrackMaxContentVal = MaxTrackSizingFunction{Tag: MaxTrackMaxContent}

// MaxTrackFitContentPxVal creates a fit-content(px) MaxTrackSizingFunction.
func MaxTrackFitContentPxVal(limit float32) MaxTrackSizingFunction {
	return MaxTrackSizingFunction{Tag: MaxTrackFitContentPx, Value: limit}
}

// MaxTrackFitContentPercentVal creates a fit-content(%) MaxTrackSizingFunction.
func MaxTrackFitContentPercentVal(limit float32) MaxTrackSizingFunction {
	return MaxTrackSizingFunction{Tag: MaxTrackFitContentPercent, Value: limit}
}

// MaxTrackFitContentFromLP creates a fit-content MaxTrackSizingFunction from a LengthPercentage.
func MaxTrackFitContentFromLP(lp LengthPercentage) MaxTrackSizingFunction {
	switch lp.Tag {
	case LPLength:
		return MaxTrackFitContentPxVal(lp.Value)
	case LPPercent:
		return MaxTrackFitContentPercentVal(lp.Value)
	default:
		return MaxTrackFitContentPxVal(lp.Value)
	}
}

// MaxTrackFromFr creates a fractional MaxTrackSizingFunction.
func MaxTrackFromFr(val float32) MaxTrackSizingFunction {
	return MaxTrackSizingFunction{Tag: MaxTrackFr, Value: val}
}

// MaxTrackFromCalc creates a MaxTrackSizingFunction from a calc() ID.
func MaxTrackFromCalc(id uint64) MaxTrackSizingFunction {
	return MaxTrackSizingFunction{Tag: MaxTrackCalc, Value: math.Float32frombits(uint32(id))}
}

// IsIntrinsic reports whether the max track sizing function is MinContent, MaxContent, FitContent, or Auto.
func (m MaxTrackSizingFunction) IsIntrinsic() bool {
	switch m.Tag {
	case MaxTrackAuto, MaxTrackMinContent, MaxTrackMaxContent, MaxTrackFitContentPx, MaxTrackFitContentPercent:
		return true
	}
	return false
}

// IsMaxContentAlike reports whether the max track sizing function is MaxContent, FitContent, or Auto.
// "In all cases, treat auto and fit-content() as max-content, except where specified otherwise for fit-content()."
func (m MaxTrackSizingFunction) IsMaxContentAlike() bool {
	switch m.Tag {
	case MaxTrackAuto, MaxTrackMaxContent, MaxTrackFitContentPx, MaxTrackFitContentPercent:
		return true
	}
	return false
}

// IsFr reports whether the max track sizing function is a fractional value.
func (m MaxTrackSizingFunction) IsFr() bool {
	return m.Tag == MaxTrackFr
}

// IsAuto reports whether the max track sizing function is Auto.
func (m MaxTrackSizingFunction) IsAuto() bool {
	return m.Tag == MaxTrackAuto
}

// IsMinContent reports whether the max track sizing function is MinContent.
func (m MaxTrackSizingFunction) IsMinContent() bool {
	return m.Tag == MaxTrackMinContent
}

// IsMaxContent reports whether the max track sizing function is MaxContent.
func (m MaxTrackSizingFunction) IsMaxContent() bool {
	return m.Tag == MaxTrackMaxContent
}

// IsFitContent reports whether the max track sizing function is FitContent (either px or percent).
func (m MaxTrackSizingFunction) IsFitContent() bool {
	return m.Tag == MaxTrackFitContentPx || m.Tag == MaxTrackFitContentPercent
}

// IsMaxOrFitContent reports whether the max track sizing function is MaxContent or FitContent.
func (m MaxTrackSizingFunction) IsMaxOrFitContent() bool {
	switch m.Tag {
	case MaxTrackMaxContent, MaxTrackFitContentPx, MaxTrackFitContentPercent:
		return true
	}
	return false
}

// HasDefiniteValue reports whether the value can be resolved using DefiniteValue.
func (m MaxTrackSizingFunction) HasDefiniteValue(parentSize Option[float32]) bool {
	switch m.Tag {
	case MaxTrackLength:
		return true
	case MaxTrackPercent:
		return parentSize.IsSome
	case MaxTrackCalc:
		return parentSize.IsSome
	default:
		return false
	}
}

// DefiniteValue resolves this max track sizing function to a concrete value if possible.
func (m MaxTrackSizingFunction) DefiniteValue(parentSize Option[float32], calcResolver CalcResolver) Option[float32] {
	switch m.Tag {
	case MaxTrackLength:
		return Some(m.Value)
	case MaxTrackPercent:
		if parentSize.IsSome {
			return Some(m.Value * parentSize.Value)
		}
		return None[float32]()
	case MaxTrackCalc:
		if parentSize.IsSome && calcResolver != nil {
			id := uint64(math.Float32bits(m.Value))
			return Some(calcResolver(id, parentSize.Value))
		}
		return None[float32]()
	default:
		return None[float32]()
	}
}

// DefiniteLimit resolves the maximum size, also considering fit-content() sizing functions.
func (m MaxTrackSizingFunction) DefiniteLimit(parentSize Option[float32], calcResolver CalcResolver) Option[float32] {
	switch m.Tag {
	case MaxTrackFitContentPx:
		return Some(m.Value)
	case MaxTrackFitContentPercent:
		if parentSize.IsSome {
			return Some(m.Value * parentSize.Value)
		}
		return None[float32]()
	default:
		return m.DefiniteValue(parentSize, calcResolver)
	}
}

// ResolvedPercentageSize resolves percentage values against the parent size.
// Non-percentage values return None.
func (m MaxTrackSizingFunction) ResolvedPercentageSize(parentSize float32, calcResolver CalcResolver) Option[float32] {
	switch m.Tag {
	case MaxTrackPercent:
		return Some(m.Value * parentSize)
	case MaxTrackFitContentPercent:
		return Some(m.Value * parentSize)
	case MaxTrackCalc:
		if calcResolver != nil {
			id := uint64(math.Float32bits(m.Value))
			return Some(calcResolver(id, parentSize))
		}
		return None[float32]()
	default:
		return None[float32]()
	}
}

// UsesPercentage reports whether the max track sizing function depends on the parent node size.
func (m MaxTrackSizingFunction) UsesPercentage() bool {
	switch m.Tag {
	case MaxTrackPercent, MaxTrackFitContentPercent, MaxTrackCalc:
		return true
	}
	return false
}

// MaxTrackSizingFunctionFromLengthPercentage converts a LengthPercentage to a MaxTrackSizingFunction.
func MaxTrackSizingFunctionFromLengthPercentage(lp LengthPercentage) MaxTrackSizingFunction {
	switch lp.Tag {
	case LPLength:
		return MaxTrackSizingFunction{Tag: MaxTrackLength, Value: lp.Value}
	case LPPercent:
		return MaxTrackSizingFunction{Tag: MaxTrackPercent, Value: lp.Value}
	default: // LPCalc
		return MaxTrackSizingFunction{Tag: MaxTrackCalc, Value: lp.Value}
	}
}

// MaxTrackSizingFunctionFromLengthPercentageAuto converts a LengthPercentageAuto to a MaxTrackSizingFunction.
func MaxTrackSizingFunctionFromLengthPercentageAuto(lpa LengthPercentageAuto) MaxTrackSizingFunction {
	switch lpa.Tag {
	case LPALength:
		return MaxTrackSizingFunction{Tag: MaxTrackLength, Value: lpa.Value}
	case LPAPercent:
		return MaxTrackSizingFunction{Tag: MaxTrackPercent, Value: lpa.Value}
	case LPAAuto:
		return MaxTrackAutoVal
	default: // LPACalc
		return MaxTrackSizingFunction{Tag: MaxTrackCalc, Value: lpa.Value}
	}
}

// MaxTrackSizingFunctionFromDimension converts a Dimension to a MaxTrackSizingFunction.
func MaxTrackSizingFunctionFromDimension(d Dimension) MaxTrackSizingFunction {
	switch d.Tag {
	case DimensionLength:
		return MaxTrackSizingFunction{Tag: MaxTrackLength, Value: d.Value}
	case DimensionPercent:
		return MaxTrackSizingFunction{Tag: MaxTrackPercent, Value: d.Value}
	case DimensionAuto:
		return MaxTrackAutoVal
	default: // DimensionCalc
		return MaxTrackSizingFunction{Tag: MaxTrackCalc, Value: d.Value}
	}
}

// MaxTrackSizingFunctionFromMin converts a MinTrackSizingFunction to a MaxTrackSizingFunction.
func MaxTrackSizingFunctionFromMin(m MinTrackSizingFunction) MaxTrackSizingFunction {
	switch m.Tag {
	case MinTrackLength:
		return MaxTrackSizingFunction{Tag: MaxTrackLength, Value: m.Value}
	case MinTrackPercent:
		return MaxTrackSizingFunction{Tag: MaxTrackPercent, Value: m.Value}
	case MinTrackAuto:
		return MaxTrackAutoVal
	case MinTrackMinContent:
		return MaxTrackMinContentVal
	case MinTrackMaxContent:
		return MaxTrackMaxContentVal
	default: // MinTrackCalc
		return MaxTrackSizingFunction{Tag: MaxTrackCalc, Value: m.Value}
	}
}

// MinTrackSizingFunctionFromMax converts a MaxTrackSizingFunction to a MinTrackSizingFunction.
// Fr and FitContent values fall back to Auto.
func MinTrackSizingFunctionFromMax(m MaxTrackSizingFunction) MinTrackSizingFunction {
	if m.IsFr() || m.IsFitContent() {
		return MinTrackAutoVal
	}
	switch m.Tag {
	case MaxTrackLength:
		return MinTrackSizingFunction{Tag: MinTrackLength, Value: m.Value}
	case MaxTrackPercent:
		return MinTrackSizingFunction{Tag: MinTrackPercent, Value: m.Value}
	case MaxTrackAuto:
		return MinTrackAutoVal
	case MaxTrackMinContent:
		return MinTrackMinContentVal
	case MaxTrackMaxContent:
		return MinTrackMaxContentVal
	default: // MaxTrackCalc
		return MinTrackSizingFunction{Tag: MinTrackCalc, Value: m.Value}
	}
}

// ---------------------------------------------------------------------------
// TrackSizingFunction
// ---------------------------------------------------------------------------

// TrackSizingFunction is the sizing function for a grid track (row or column).
// It is a MinMax of MinTrackSizingFunction and MaxTrackSizingFunction.
type TrackSizingFunction = MinMax[MinTrackSizingFunction, MaxTrackSizingFunction]

// TrackSizingAuto is the auto TrackSizingFunction (both min and max are auto).
var TrackSizingAuto = TrackSizingFunction{
	Min: MinTrackAutoVal,
	Max: MaxTrackAutoVal,
}

// TrackSizingMinContent is the min-content TrackSizingFunction.
var TrackSizingMinContent = TrackSizingFunction{
	Min: MinTrackMinContentVal,
	Max: MaxTrackMinContentVal,
}

// TrackSizingMaxContent is the max-content TrackSizingFunction.
var TrackSizingMaxContent = TrackSizingFunction{
	Min: MinTrackMaxContentVal,
	Max: MaxTrackMaxContentVal,
}

// TrackSizingZero is a zero-length TrackSizingFunction.
var TrackSizingZero = TrackSizingFunction{
	Min: MinTrackSizingFunction{Tag: MinTrackLength, Value: 0},
	Max: MaxTrackSizingFunction{Tag: MaxTrackLength, Value: 0},
}

// TrackFromLength creates a TrackSizingFunction from an absolute length.
func TrackFromLength(val float32) TrackSizingFunction {
	return TrackSizingFunction{
		Min: MinTrackFromLength(val),
		Max: MaxTrackFromLength(val),
	}
}

// TrackFromPercent creates a TrackSizingFunction from a percentage.
func TrackFromPercent(val float32) TrackSizingFunction {
	return TrackSizingFunction{
		Min: MinTrackFromPercent(val),
		Max: MaxTrackFromPercent(val),
	}
}

// TrackFromFr creates a fractional TrackSizingFunction (min is auto, max is fr).
func TrackFromFr(val float32) TrackSizingFunction {
	return TrackSizingFunction{
		Min: MinTrackAutoVal,
		Max: MaxTrackFromFr(val),
	}
}

// TrackFitContent creates a fit-content TrackSizingFunction (min is auto).
func TrackFitContent(argument LengthPercentage) TrackSizingFunction {
	return TrackSizingFunction{
		Min: MinTrackAutoVal,
		Max: MaxTrackFitContentFromLP(argument),
	}
}

// TrackMinSizingFunction extracts the min track sizing function.
func TrackMinSizingFunction(t TrackSizingFunction) MinTrackSizingFunction {
	return t.Min
}

// TrackMaxSizingFunction extracts the max track sizing function.
func TrackMaxSizingFunction(t TrackSizingFunction) MaxTrackSizingFunction {
	return t.Max
}

// TrackHasFixedComponent reports whether at least one component is a fixed sizing function.
func TrackHasFixedComponent(t TrackSizingFunction) bool {
	minFixed := t.Min.Tag == MinTrackLength || t.Min.Tag == MinTrackPercent
	maxFixed := t.Max.Tag == MaxTrackLength || t.Max.Tag == MaxTrackPercent
	return minFixed || maxFixed
}

// TrackSizingFromLengthPercentage creates a TrackSizingFunction from a LengthPercentage.
func TrackSizingFromLengthPercentage(lp LengthPercentage) TrackSizingFunction {
	return TrackSizingFunction{
		Min: MinTrackSizingFunctionFromLengthPercentage(lp),
		Max: MaxTrackSizingFunctionFromLengthPercentage(lp),
	}
}

// TrackSizingFromLengthPercentageAuto creates a TrackSizingFunction from a LengthPercentageAuto.
func TrackSizingFromLengthPercentageAuto(lpa LengthPercentageAuto) TrackSizingFunction {
	return TrackSizingFunction{
		Min: MinTrackSizingFunctionFromLengthPercentageAuto(lpa),
		Max: MaxTrackSizingFunctionFromLengthPercentageAuto(lpa),
	}
}

// TrackSizingFromDimension creates a TrackSizingFunction from a Dimension.
func TrackSizingFromDimension(d Dimension) TrackSizingFunction {
	return TrackSizingFunction{
		Min: MinTrackSizingFunctionFromDimension(d),
		Max: MaxTrackSizingFunctionFromDimension(d),
	}
}

// ---------------------------------------------------------------------------
// RepetitionCount
// ---------------------------------------------------------------------------

// RepetitionCountTag identifies the variant of a RepetitionCount.
type RepetitionCountTag int

const (
	// RepetitionAutoFill generates tracks to fill the container.
	RepetitionAutoFill RepetitionCountTag = iota
	// RepetitionAutoFit generates tracks to fit the container.
	RepetitionAutoFit
	// RepetitionCountFixed repeats tracks a fixed number of times.
	RepetitionCountFixed
)

// RepetitionCount is the first argument to a repeat() in a grid-template definition.
type RepetitionCount struct {
	Tag   RepetitionCountTag
	Count uint16 // used for RepetitionCountFixed
}

// RepetitionCountFromU16 creates a RepetitionCount with a fixed count.
func RepetitionCountFromU16(n uint16) RepetitionCount {
	return RepetitionCount{Tag: RepetitionCountFixed, Count: n}
}

// RepetitionAutoFillVal is the AutoFill variant.
var RepetitionAutoFillVal = RepetitionCount{Tag: RepetitionAutoFill}

// RepetitionAutoFitVal is the AutoFit variant.
var RepetitionAutoFitVal = RepetitionCount{Tag: RepetitionAutoFit}

// ---------------------------------------------------------------------------
// GridTemplateRepetition
// ---------------------------------------------------------------------------

// GridTemplateRepetition is a typed representation of a repeat(..) in grid-template-* values.
type GridTemplateRepetition struct {
	// Count is the number of times the tracks are repeated.
	Count RepetitionCount
	// Tracks are the track sizing functions to repeat.
	Tracks []TrackSizingFunction
	// LineNames are the line names for the repeated tracks.
	LineNames [][]string
}

// TrackCount returns the number of repeated tracks.
func (r GridTemplateRepetition) TrackCount() uint16 {
	return uint16(len(r.Tracks))
}

// IsAutoRepetition reports whether this is an auto-repetition (AutoFill or AutoFit).
func (r GridTemplateRepetition) IsAutoRepetition() bool {
	return r.Count.Tag == RepetitionAutoFill || r.Count.Tag == RepetitionAutoFit
}

// ---------------------------------------------------------------------------
// GridTemplateComponent
// ---------------------------------------------------------------------------

// GridTemplateComponentTag identifies the variant of a GridTemplateComponent.
type GridTemplateComponentTag int

const (
	// GridTemplateSingle is a single track sizing function.
	GridTemplateSingle GridTemplateComponentTag = iota
	// GridTemplateRepeat is a repeat() block.
	GridTemplateRepeat
)

// GridTemplateComponent is an element in a grid-template-columns or grid-template-rows definition.
type GridTemplateComponent struct {
	Tag        GridTemplateComponentTag
	Single     TrackSizingFunction
	Repetition GridTemplateRepetition
}

// GridTemplateSingleVal creates a GridTemplateComponent with a single track sizing function.
func GridTemplateSingleVal(t TrackSizingFunction) GridTemplateComponent {
	return GridTemplateComponent{Tag: GridTemplateSingle, Single: t}
}

// GridTemplateRepeatVal creates a GridTemplateComponent with a repeat() block.
func GridTemplateRepeatVal(r GridTemplateRepetition) GridTemplateComponent {
	return GridTemplateComponent{Tag: GridTemplateRepeat, Repetition: r}
}

// IsAutoRepetition reports whether the track definition is an auto-repeated fragment.
func (c GridTemplateComponent) IsAutoRepetition() bool {
	if c.Tag != GridTemplateRepeat {
		return false
	}
	return c.Repetition.IsAutoRepetition()
}

// ---------------------------------------------------------------------------
// GridTemplateArea
// ---------------------------------------------------------------------------

// GridTemplateArea defines a named grid area and its extent in grid coordinates.
type GridTemplateArea struct {
	// Name is the name of the grid area.
	Name string
	// RowStart is the index of the row at which the grid area starts in grid coordinates.
	RowStart uint16
	// RowEnd is the index of the row at which the grid area ends in grid coordinates.
	RowEnd uint16
	// ColumnStart is the index of the column at which the grid area starts in grid coordinates.
	ColumnStart uint16
	// ColumnEnd is the index of the column at which the grid area ends in grid coordinates.
	ColumnEnd uint16
}

// ---------------------------------------------------------------------------
// GridAreaAxis and GridAreaEnd
// ---------------------------------------------------------------------------

// GridAreaAxis identifies a Row or Column axis for grid area operations.
type GridAreaAxis int

const (
	// GridAreaAxisRow is the Row axis.
	GridAreaAxisRow GridAreaAxis = iota
	// GridAreaAxisColumn is the Column axis.
	GridAreaAxisColumn
)

// GridAreaEnd identifies the Start or End of a grid area edge.
type GridAreaEnd int

const (
	// GridAreaEndStart is the Start edge.
	GridAreaEndStart GridAreaEnd = iota
	// GridAreaEndEnd is the End edge.
	GridAreaEndEnd
)

// ---------------------------------------------------------------------------
// gridContainerStyle and gridItemStyle interfaces
// ---------------------------------------------------------------------------

// gridContainerStyle defines the style methods required for a CSS Grid container.
type gridContainerStyle interface {
	GridTemplateRows() []GridTemplateComponent
	GridTemplateColumns() []GridTemplateComponent
	GridAutoRows() []TrackSizingFunction
	GridAutoColumns() []TrackSizingFunction
	GridTemplateAreas() []GridTemplateArea
	GridAutoFlow() GridAutoFlow
	Gap() Size[LengthPercentage]
	AlignContent() Option[AlignContent]
	JustifyContent() Option[JustifyContent]
	AlignItems() Option[AlignItems]
	JustifyItems() Option[AlignItems]
}

// gridItemStyle defines the style methods required for a CSS Grid item.
type gridItemStyle interface {
	GridRow() Line[GridPlacement]
	GridColumn() Line[GridPlacement]
	AlignSelf() Option[AlignSelf]
	JustifySelf() Option[AlignSelf]
}
