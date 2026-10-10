// Package style provides available space types for CSS layout.
// This file is a port of available_space.rs from the Taffy Rust layout engine.
package style

import (
	"github.com/gataengine/taffy-go/lib"
	"math"

	"github.com/gataengine/taffy-go/geometry"
)

// AvailableSpaceTag identifies the variant of an AvailableSpace value.
type AvailableSpaceTag int

const (
	// AvailableSpaceDefinite represents a definite amount of available space (in pixels).
	AvailableSpaceDefinite AvailableSpaceTag = iota
	// AvailableSpaceMinContent represents indefinite space under a min-content constraint.
	AvailableSpaceMinContent
	// AvailableSpaceMaxContent represents indefinite space under a max-content constraint.
	AvailableSpaceMaxContent
)

// AvailableSpace represents the amount of space available to a node in a given axis.
// See: https://www.w3.org/TR/css-sizing-3/#available
type AvailableSpace struct {
	Tag   AvailableSpaceTag
	Value float32
}

// AvailableSpaceFromDefinite creates an AvailableSpace with a definite pixel value.
func AvailableSpaceFromDefinite(value float32) AvailableSpace {
	return AvailableSpace{Tag: AvailableSpaceDefinite, Value: value}
}

// AvailableSpaceFromFloat32 creates an AvailableSpace from a float32 value (always Definite).
func AvailableSpaceFromFloat32(value float32) AvailableSpace {
	return AvailableSpace{Tag: AvailableSpaceDefinite, Value: value}
}

// AvailableSpaceFromOptionFloat32 creates an AvailableSpace from an Option[float32].
// Some(value) maps to Definite(value), None maps to MaxContent.
func AvailableSpaceFromOptionFloat32(opt lib.Option[float32]) AvailableSpace {
	if opt.IsSome {
		return AvailableSpace{Tag: AvailableSpaceDefinite, Value: opt.Value}
	}
	return AvailableSpace{Tag: AvailableSpaceMaxContent}
}

// AvailableSpaceZero is an AvailableSpace with a definite value of 0.
var AvailableSpaceZero = AvailableSpace{Tag: AvailableSpaceDefinite, Value: 0}

// AvailableSpaceMinContentValue is the MinContent variant of AvailableSpace.
var AvailableSpaceMinContentValue = AvailableSpace{Tag: AvailableSpaceMinContent}

// AvailableSpaceMaxContentValue is the MaxContent variant of AvailableSpace.
var AvailableSpaceMaxContentValue = AvailableSpace{Tag: AvailableSpaceMaxContent}

// IsDefinite returns true for definite values, else false.
func (as AvailableSpace) IsDefinite() bool {
	return as.Tag == AvailableSpaceDefinite
}

// IntoOption converts to Option[float32].
// Definite values become Some(value). Constraints become None.
func (as AvailableSpace) IntoOption() lib.Option[float32] {
	if as.Tag == AvailableSpaceDefinite {
		return lib.Some(as.Value)
	}
	return lib.None[float32]()
}

// UnwrapOr returns the definite value or the provided default.
func (as AvailableSpace) UnwrapOr(defaultValue float32) float32 {
	if as.Tag == AvailableSpaceDefinite {
		return as.Value
	}
	return defaultValue
}

// Unwrap returns the definite value. Panics if the value is not definite.
func (as AvailableSpace) Unwrap() float32 {
	if as.Tag == AvailableSpaceDefinite {
		return as.Value
	}
	panic("called Unwrap on a non-Definite AvailableSpace")
}

// Or returns self if definite, otherwise returns the default value.
func (as AvailableSpace) Or(defaultValue AvailableSpace) AvailableSpace {
	if as.Tag == AvailableSpaceDefinite {
		return as
	}
	return defaultValue
}

// OrElse returns self if definite, otherwise returns the result of the callback.
func (as AvailableSpace) OrElse(defaultCb func() AvailableSpace) AvailableSpace {
	if as.Tag == AvailableSpaceDefinite {
		return as
	}
	return defaultCb()
}

// UnwrapOrElse returns the definite value or the result of the callback.
func (as AvailableSpace) UnwrapOrElse(defaultCb func() float32) float32 {
	if as.Tag == AvailableSpaceDefinite {
		return as.Value
	}
	return defaultCb()
}

// MaybeSet returns AvailableSpace::Definite containing the option value if Some, else returns self.
func (as AvailableSpace) MaybeSet(value lib.Option[float32]) AvailableSpace {
	if value.IsSome {
		return AvailableSpace{Tag: AvailableSpaceDefinite, Value: value.Value}
	}
	return as
}

// MapDefiniteValue applies the map function to the definite value, or returns self unchanged.
func (as AvailableSpace) MapDefiniteValue(mapFn func(float32) float32) AvailableSpace {
	if as.Tag == AvailableSpaceDefinite {
		return AvailableSpace{Tag: AvailableSpaceDefinite, Value: mapFn(as.Value)}
	}
	return as
}

// ComputeFreeSpace computes free space given the passed used_space.
func (as AvailableSpace) ComputeFreeSpace(usedSpace float32) float32 {
	switch as.Tag {
	case AvailableSpaceMaxContent:
		return float32(math.Inf(1))
	case AvailableSpaceMinContent:
		return 0.0
	default:
		return as.Value - usedSpace
	}
}

// IsRoughlyEqual compares equality with another AvailableSpace, treating definite values
// that are within float32 epsilon of each other as equal.
func (as AvailableSpace) IsRoughlyEqual(other AvailableSpace) bool {
	if as.Tag != other.Tag {
		return false
	}
	if as.Tag == AvailableSpaceDefinite {
		diff := as.Value - other.Value
		if diff < 0 {
			diff = -diff
		}
		return diff < float32(math.SmallestNonzeroFloat32)*1e38 // approximate float32 epsilon
	}
	return true
}

// SizeAvailableSpaceIntoOptions converts Size[AvailableSpace] into Size[Option[float32]].
func SizeAvailableSpaceIntoOptions(s geometry.Size[AvailableSpace]) geometry.Size[lib.Option[float32]] {
	return geometry.Size[lib.Option[float32]]{
		Width:  s.Width.IntoOption(),
		Height: s.Height.IntoOption(),
	}
}

// SizeAvailableSpaceMaybeSet returns a new Size[AvailableSpace] where each axis is set to
// a Definite value if the corresponding Option is Some, else unchanged.
func SizeAvailableSpaceMaybeSet(s geometry.Size[AvailableSpace], value geometry.Size[lib.Option[float32]]) geometry.Size[AvailableSpace] {
	return geometry.Size[AvailableSpace]{
		Width:  s.Width.MaybeSet(value.Width),
		Height: s.Height.MaybeSet(value.Height),
	}
}
