// Package style provides float layout style types.
// This file is a port of float.rs from the Taffy Rust layout engine.
package style

// Float controls whether a box is floated to the left or right.
// This property only applies to children of a block layout.
//
// See: https://developer.mozilla.org/en-US/docs/Web/CSS/float
type Float int

const (
	// FloatNone means the box is not floated.
	FloatNone Float = iota
	// FloatLeft floats the box to the left.
	FloatLeft
	// FloatRight floats the box to the right.
	FloatRight
)

// FloatDirection represents the direction of a definitely-floated box.
type FloatDirection int

const (
	// FloatDirectionLeft means the box is floated to the left.
	FloatDirectionLeft FloatDirection = iota
	// FloatDirectionRight means the box is floated to the right.
	FloatDirectionRight
)

// IsFloated reports whether the box is floated (left or right).
func (f Float) IsFloated() bool {
	return f == FloatLeft || f == FloatRight
}

// FloatDirection converts Float into (FloatDirection, bool).
// Returns (FloatDirectionLeft, true) for FloatLeft,
// (FloatDirectionRight, true) for FloatRight,
// and (FloatDirectionLeft, false) for FloatNone.
func (f Float) GetFloatDirection() (FloatDirection, bool) {
	switch f {
	case FloatLeft:
		return FloatDirectionLeft, true
	case FloatRight:
		return FloatDirectionRight, true
	default:
		return FloatDirectionLeft, false
	}
}

// Clear gives a box "clearance", which moves it below floated boxes which precede it in the tree.
//
// See: https://developer.mozilla.org/en-US/docs/Web/CSS/clear
type Clear int

const (
	// ClearNone means the box does not clear floated boxes.
	ClearNone Clear = iota
	// ClearLeft clears left-floated boxes.
	ClearLeft
	// ClearRight clears right-floated boxes.
	ClearRight
	// ClearBoth clears boxes floated in either direction.
	ClearBoth
)
