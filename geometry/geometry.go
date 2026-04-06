// Package geometry provides geometric primitives useful for layout.
// This file is a port of geometry.rs from the Taffy Rust layout engine.
package geometry

// Option represents an optional value, similar to Rust's Option<T>.
type Option[T any] struct {
	Value  T
	IsSome bool
}

// Some returns an Option containing the given value.
func Some[T any](v T) Option[T] {
	return Option[T]{Value: v, IsSome: true}
}

// None returns an empty Option.
func None[T any]() Option[T] {
	return Option[T]{}
}

// UnwrapOr returns the contained value if Some, otherwise returns the alt value.
func (o Option[T]) UnwrapOr(alt T) T {
	if o.IsSome {
		return o.Value
	}
	return alt
}

// Or returns this Option if Some, otherwise returns alt.
func (o Option[T]) Or(alt Option[T]) Option[T] {
	if o.IsSome {
		return o
	}
	return alt
}

// AbsoluteAxis represents the simple absolute horizontal and vertical axis.
type AbsoluteAxis int

const (
	// AbsoluteAxisHorizontal is the horizontal axis.
	AbsoluteAxisHorizontal AbsoluteAxis = iota
	// AbsoluteAxisVertical is the vertical axis.
	AbsoluteAxisVertical
)

// OtherAxis returns the other variant of the enum.
func (a AbsoluteAxis) OtherAxis() AbsoluteAxis {
	switch a {
	case AbsoluteAxisHorizontal:
		return AbsoluteAxisVertical
	default:
		return AbsoluteAxisHorizontal
	}
}

// AbstractAxis represents the CSS abstract axis.
// See: https://www.w3.org/TR/css-writing-modes-3/#abstract-axes
type AbstractAxis int

const (
	// AbstractAxisInline is the axis in the inline dimension (horizontal in LTR writing modes).
	AbstractAxisInline AbstractAxis = iota
	// AbstractAxisBlock is the axis in the block dimension (vertical in LTR writing modes).
	AbstractAxisBlock
)

// Other returns the other variant of the enum.
func (a AbstractAxis) Other() AbstractAxis {
	switch a {
	case AbstractAxisInline:
		return AbstractAxisBlock
	default:
		return AbstractAxisInline
	}
}

// AsAbsNaive converts an AbstractAxis to an AbsoluteAxis, assuming Inline == Horizontal.
// This is currently always true but would change if writing_mode were implemented.
func (a AbstractAxis) AsAbsNaive() AbsoluteAxis {
	switch a {
	case AbstractAxisInline:
		return AbsoluteAxisHorizontal
	default:
		return AbsoluteAxisVertical
	}
}

// inBothAbsAxis holds an item in each absolute axis.
type inBothAbsAxis[T any] struct {
	horizontal T
	vertical   T
}

// get returns the contained item based on the AbsoluteAxis passed.
func (i inBothAbsAxis[T]) get(axis AbsoluteAxis) T {
	switch axis {
	case AbsoluteAxisHorizontal:
		return i.horizontal
	default:
		return i.vertical
	}
}

// Rect is an axis-aligned UI rectangle.
type Rect[T any] struct {
	// Left can represent either the x-coordinate of the starting edge or the amount of padding on the starting side.
	Left T
	// Right can represent either the x-coordinate of the ending edge or the amount of padding on the ending side.
	Right T
	// Top can represent either the y-coordinate of the top edge or the amount of padding on the top side.
	Top T
	// Bottom can represent either the y-coordinate of the bottom edge or the amount of padding on the bottom side.
	Bottom T
}

// Map applies the function f to all four sides of the rect.
func (r Rect[T]) Map(f func(T) T) Rect[T] {
	return Rect[T]{
		Left:   f(r.Left),
		Right:  f(r.Right),
		Top:    f(r.Top),
		Bottom: f(r.Bottom),
	}
}

// MapTo applies the function f to all four sides of the rect, returning a Rect of a different type.
// Since Go generics cannot add a new type parameter in a method, use the standalone RectMapTo function.

// HorizontalComponents returns a Line[T] representing the left and right properties of the Rect.
func (r Rect[T]) HorizontalComponents() Line[T] {
	return Line[T]{Start: r.Left, End: r.Right}
}

// VerticalComponents returns a Line[T] containing the top and bottom properties of the Rect.
func (r Rect[T]) VerticalComponents() Line[T] {
	return Line[T]{Start: r.Top, End: r.Bottom}
}

// RectMapTo applies f to all four sides of a Rect[A], producing a Rect[B].
func RectMapTo[A, B any](r Rect[A], f func(A) B) Rect[B] {
	return Rect[B]{
		Left:   f(r.Left),
		Right:  f(r.Right),
		Top:    f(r.Top),
		Bottom: f(r.Bottom),
	}
}

// RectZipSize applies f to left/right with size.Width and top/bottom with size.Height.
func RectZipSize[T, U, R any](r Rect[T], size Size[U], f func(T, U) R) Rect[R] {
	return Rect[R]{
		Left:   f(r.Left, size.Width),
		Right:  f(r.Right, size.Width),
		Top:    f(r.Top, size.Height),
		Bottom: f(r.Bottom, size.Height),
	}
}

// RectAddF32 adds two Rect[float32] values component-wise.
func RectAddF32(a, b Rect[float32]) Rect[float32] {
	return Rect[float32]{
		Left:   a.Left + b.Left,
		Right:  a.Right + b.Right,
		Top:    a.Top + b.Top,
		Bottom: a.Bottom + b.Bottom,
	}
}

// RectHorizontalAxisSum returns left + right.
func RectHorizontalAxisSum[T interface {
	float32 | float64 | int | int32 | int64
}](r Rect[T]) T {
	return r.Left + r.Right
}

// RectVerticalAxisSum returns top + bottom.
func RectVerticalAxisSum[T interface {
	float32 | float64 | int | int32 | int64
}](r Rect[T]) T {
	return r.Top + r.Bottom
}

// RectSumAxes returns both axis sums as a Size.
func RectSumAxes[T interface {
	float32 | float64 | int | int32 | int64
}](r Rect[T]) Size[T] {
	return Size[T]{
		Width:  RectHorizontalAxisSum(r),
		Height: RectVerticalAxisSum(r),
	}
}

// RectGridAxisSum returns the sum of the two sides along the given AbsoluteAxis.
func RectGridAxisSum[T interface {
	float32 | float64 | int | int32 | int64
}](r Rect[T], axis AbsoluteAxis) T {
	switch axis {
	case AbsoluteAxisHorizontal:
		return r.Left + r.Right
	default:
		return r.Top + r.Bottom
	}
}

// RectZeroF32 is a Rect[float32] with all zeros.
var RectZeroF32 = Rect[float32]{Left: 0, Right: 0, Top: 0, Bottom: 0}

// NewRectF32 creates a new Rect[float32].
func NewRectF32(left, right, top, bottom float32) Rect[float32] {
	return Rect[float32]{Left: left, Right: right, Top: top, Bottom: bottom}
}

// Line represents an abstract "line" with a start and an end.
type Line[T any] struct {
	// Start is the start position of a line.
	Start T
	// End is the end position of a line.
	End T
}

// Map applies f to both Start and End.
func (l Line[T]) Map(f func(T) T) Line[T] {
	return Line[T]{Start: f(l.Start), End: f(l.End)}
}

// LineMapTo applies f to both fields of Line[A], producing a Line[B].
func LineMapTo[A, B any](l Line[A], f func(A) B) Line[B] {
	return Line[B]{Start: f(l.Start), End: f(l.End)}
}

// LineTrueBool is a Line with both Start and End set to true.
var LineTrueBool = Line[bool]{Start: true, End: true}

// LineFalseBool is a Line[bool] with both fields set to false.
var LineFalseBool = Line[bool]{Start: false, End: false}

// LineSumF32 adds the Start and End of a Line[float32].
func LineSumF32(l Line[float32]) float32 {
	return l.Start + l.End
}

// LineSumF64 adds the Start and End of a Line[float64].
func LineSumF64(l Line[float64]) float64 {
	return l.Start + l.End
}

// LineSumInt adds the Start and End of a Line[int].
func LineSumInt(l Line[int]) int {
	return l.Start + l.End
}

// LineSumInt32 adds the Start and End of a Line[int32].
func LineSumInt32(l Line[int32]) int32 {
	return l.Start + l.End
}

// Size represents the width and height of a Rect.
type Size[T any] struct {
	// Width is the x extent of the rectangle.
	Width T
	// Height is the y extent of the rectangle.
	Height T
}

// GetAbs returns either the Width or Height depending on the AbsoluteAxis passed in.
func (s Size[T]) GetAbs(axis AbsoluteAxis) T {
	switch axis {
	case AbsoluteAxisHorizontal:
		return s.Width
	default:
		return s.Height
	}
}

// GetAbstract returns either the Width or Height depending on the AbstractAxis passed in.
func (s Size[T]) GetAbstract(axis AbstractAxis) T {
	switch axis {
	case AbstractAxisInline:
		return s.Width
	default:
		return s.Height
	}
}

// SetAbstract sets either the Width or Height depending on the AbstractAxis passed in.
func (s *Size[T]) SetAbstract(axis AbstractAxis, value T) {
	switch axis {
	case AbstractAxisInline:
		s.Width = value
	default:
		s.Height = value
	}
}

// Map applies f to both Width and Height.
func (s Size[T]) Map(f func(T) T) Size[T] {
	return Size[T]{Width: f(s.Width), Height: f(s.Height)}
}

// MapWidth applies f only to the Width.
func (s Size[T]) MapWidth(f func(T) T) Size[T] {
	return Size[T]{Width: f(s.Width), Height: s.Height}
}

// MapHeight applies f only to the Height.
func (s Size[T]) MapHeight(f func(T) T) Size[T] {
	return Size[T]{Width: s.Width, Height: f(s.Height)}
}

// SizeMapTo applies f to both fields of Size[A], producing a Size[B].
func SizeMapTo[A, B any](s Size[A], f func(A) B) Size[B] {
	return Size[B]{Width: f(s.Width), Height: f(s.Height)}
}

// SizeZipMap applies f to corresponding fields of two sizes, producing a new Size[R].
func SizeZipMap[T, Other, Ret any](s Size[T], other Size[Other], f func(T, Other) Ret) Size[Ret] {
	return Size[Ret]{
		Width:  f(s.Width, other.Width),
		Height: f(s.Height, other.Height),
	}
}

// SizeAddF32 adds two Size[float32] values component-wise.
func SizeAddF32(a, b Size[float32]) Size[float32] {
	return Size[float32]{Width: a.Width + b.Width, Height: a.Height + b.Height}
}

// SizeSubF32 subtracts two Size[float32] values component-wise.
func SizeSubF32(a, b Size[float32]) Size[float32] {
	return Size[float32]{Width: a.Width - b.Width, Height: a.Height - b.Height}
}

// SizeZeroF32 is a Size[float32] with zero width and height.
var SizeZeroF32 = Size[float32]{Width: 0, Height: 0}

// f32Max returns the larger of two float32 values (NaN-safe: returns rhs if lhs is NaN).
func f32Max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// f32Min returns the smaller of two float32 values (NaN-safe: returns rhs if lhs is NaN).
func f32Min(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// SizeF32Max applies f32Max to each component separately.
func SizeF32Max(s, rhs Size[float32]) Size[float32] {
	return Size[float32]{
		Width:  f32Max(s.Width, rhs.Width),
		Height: f32Max(s.Height, rhs.Height),
	}
}

// SizeF32Min applies f32Min to each component separately.
func SizeF32Min(s, rhs Size[float32]) Size[float32] {
	return Size[float32]{
		Width:  f32Min(s.Width, rhs.Width),
		Height: f32Min(s.Height, rhs.Height),
	}
}

// SizeHasNonZeroArea returns true if both width and height are greater than 0.
func SizeHasNonZeroArea(s Size[float32]) bool {
	return s.Width > 0.0 && s.Height > 0.0
}

// SizeOptionNone is a Size[Option[float32]] with None width and height.
var SizeOptionNone = Size[Option[float32]]{
	Width:  None[float32](),
	Height: None[float32](),
}

// NewSizeOptionF32 creates a Size[Option[float32]] with Some(width) and Some(height).
func NewSizeOptionF32(width, height float32) Size[Option[float32]] {
	return Size[Option[float32]]{
		Width:  Some(width),
		Height: Some(height),
	}
}

// SizeOptionMaybeApplyAspectRatio applies aspect_ratio (if supplied) to a Size[Option[float32]].
//   - If width is Some but height is None, height is computed from width and aspect_ratio.
//   - If height is Some but width is None, width is computed from height and aspect_ratio.
//
// If aspect_ratio is None, this function returns size unchanged.
func SizeOptionMaybeApplyAspectRatio(size Size[Option[float32]], aspectRatio Option[float32]) Size[Option[float32]] {
	if !aspectRatio.IsSome {
		return size
	}
	ratio := aspectRatio.Value
	hasWidth := size.Width.IsSome
	hasHeight := size.Height.IsSome
	switch {
	case hasWidth && !hasHeight:
		w := size.Width.Value
		return Size[Option[float32]]{Width: Some(w), Height: Some(w / ratio)}
	case !hasWidth && hasHeight:
		h := size.Height.Value
		return Size[Option[float32]]{Width: Some(h * ratio), Height: Some(h)}
	default:
		return size
	}
}

// SizeOptionUnwrapOr performs Option.UnwrapOr on each component separately.
func SizeOptionUnwrapOr[T any](s Size[Option[T]], alt Size[T]) Size[T] {
	return Size[T]{
		Width:  s.Width.UnwrapOr(alt.Width),
		Height: s.Height.UnwrapOr(alt.Height),
	}
}

// SizeOptionOr performs Option.Or on each component separately.
func SizeOptionOr[T any](s Size[Option[T]], alt Size[Option[T]]) Size[Option[T]] {
	return Size[Option[T]]{
		Width:  s.Width.Or(alt.Width),
		Height: s.Height.Or(alt.Height),
	}
}

// SizeOptionBothAxisDefined returns true if both Width and Height are Some.
func SizeOptionBothAxisDefined[T any](s Size[Option[T]]) bool {
	return s.Width.IsSome && s.Height.IsSome
}

// Point is a 2-dimensional coordinate. When used with a Rect, represents the top-left corner.
type Point[T any] struct {
	// X is the x-coordinate.
	X T
	// Y is the y-coordinate.
	Y T
}

// PointZeroF32 is a Point[float32] at the origin (0, 0).
var PointZeroF32 = Point[float32]{X: 0, Y: 0}

// PointNoneF32 is a Point[Option[float32]] with (None, None).
var PointNoneF32 = Point[Option[float32]]{X: None[float32](), Y: None[float32]()}

// Map applies f to both X and Y.
func (p Point[T]) Map(f func(T) T) Point[T] {
	return Point[T]{X: f(p.X), Y: f(p.Y)}
}

// PointMapTo applies f to both fields of Point[A], producing a Point[B].
func PointMapTo[A, B any](p Point[A], f func(A) B) Point[B] {
	return Point[B]{X: f(p.X), Y: f(p.Y)}
}

// Transpose swaps X and Y components.
func (p Point[T]) Transpose() Point[T] {
	return Point[T]{X: p.Y, Y: p.X}
}

// GetAbstract returns either X or Y depending on the AbstractAxis passed in.
func (p Point[T]) GetAbstract(axis AbstractAxis) T {
	switch axis {
	case AbstractAxisInline:
		return p.X
	default:
		return p.Y
	}
}

// SetAbstract sets either X or Y depending on the AbstractAxis passed in.
func (p *Point[T]) SetAbstract(axis AbstractAxis, value T) {
	switch axis {
	case AbstractAxisInline:
		p.X = value
	default:
		p.Y = value
	}
}

// PointAddF32 adds two Point[float32] values component-wise.
func PointAddF32(a, b Point[float32]) Point[float32] {
	return Point[float32]{X: a.X + b.X, Y: a.Y + b.Y}
}

// PointToSize converts a Point[T] to a Size[T].
func PointToSize[T any](p Point[T]) Size[T] {
	return Size[T]{Width: p.X, Height: p.Y}
}

// MinMax holds a minimum and a maximum value.
type MinMax[Min, Max any] struct {
	// Min is the value representing the minimum.
	Min Min
	// Max is the value representing the maximum.
	Max Max
}
