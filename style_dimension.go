// Package taffy provides dimension types for CSS layout.
// This file is a port of dimension.rs from the Taffy Rust layout engine.
package taffy

import "math"

// LengthPercentageTag identifies the variant of a LengthPercentage value.
type LengthPercentageTag int

const (
	// LPLength represents an absolute length.
	LPLength LengthPercentageTag = iota
	// LPPercent represents a percentage.
	LPPercent
	// LPCalc represents a calc() expression (stored as uint64 bits in Value).
	LPCalc
)

// LengthPercentage is a unit of linear measurement that can be a length or percentage.
type LengthPercentage struct {
	Tag   LengthPercentageTag
	Value float32
}

// LPFromLength creates a LengthPercentage from an absolute length value.
func LPFromLength(val float32) LengthPercentage {
	return LengthPercentage{Tag: LPLength, Value: val}
}

// LPFromPercent creates a LengthPercentage from a percentage value.
// NOTE: percentages are represented as a float32 value in the range [0.0, 1.0] NOT the range [0.0, 100.0]
func LPFromPercent(val float32) LengthPercentage {
	return LengthPercentage{Tag: LPPercent, Value: val}
}

// LPFromCalc creates a LengthPercentage from a calc() ID.
// The id is stored in the Value field as float32 bits.
func LPFromCalc(id uint64) LengthPercentage {
	return LengthPercentage{Tag: LPCalc, Value: math.Float32frombits(uint32(id))}
}

// LengthPercentageZero is a LengthPercentage with value 0.
var LengthPercentageZero = LengthPercentage{Tag: LPLength, Value: 0}

// LPATag identifies the variant of a LengthPercentageAuto value.
type LPATag int

const (
	// LPALength represents an absolute length.
	LPALength LPATag = iota
	// LPAPercent represents a percentage.
	LPAPercent
	// LPAAuto represents an automatic size.
	LPAAuto
	// LPACalc represents a calc() expression (stored as uint64 bits in Value).
	LPACalc
)

// LengthPercentageAuto is a unit of linear measurement that can be a length, percentage, or auto.
type LengthPercentageAuto struct {
	Tag   LPATag
	Value float32
}

// LPAFromLength creates a LengthPercentageAuto from an absolute length value.
func LPAFromLength(val float32) LengthPercentageAuto {
	return LengthPercentageAuto{Tag: LPALength, Value: val}
}

// LPAFromPercent creates a LengthPercentageAuto from a percentage value.
// NOTE: percentages are represented as a float32 value in the range [0.0, 1.0] NOT the range [0.0, 100.0]
func LPAFromPercent(val float32) LengthPercentageAuto {
	return LengthPercentageAuto{Tag: LPAPercent, Value: val}
}

// LPAFromCalc creates a LengthPercentageAuto from a calc() ID.
// The id is stored in the Value field as float32 bits.
func LPAFromCalc(id uint64) LengthPercentageAuto {
	return LengthPercentageAuto{Tag: LPACalc, Value: math.Float32frombits(uint32(id))}
}

// LPAFromLengthPercentage converts a LengthPercentage to a LengthPercentageAuto.
func LPAFromLengthPercentage(lp LengthPercentage) LengthPercentageAuto {
	switch lp.Tag {
	case LPLength:
		return LengthPercentageAuto{Tag: LPALength, Value: lp.Value}
	case LPPercent:
		return LengthPercentageAuto{Tag: LPAPercent, Value: lp.Value}
	default:
		return LengthPercentageAuto{Tag: LPACalc, Value: lp.Value}
	}
}

// IsAuto returns true if this value is Auto.
func (lpa LengthPercentageAuto) IsAuto() bool {
	return lpa.Tag == LPAAuto
}

// LengthPercentageAutoZero is a LengthPercentageAuto with value 0.
var LengthPercentageAutoZero = LengthPercentageAuto{Tag: LPALength, Value: 0}

// LengthPercentageAutoAuto is the auto variant of LengthPercentageAuto.
var LengthPercentageAutoAuto = LengthPercentageAuto{Tag: LPAAuto, Value: 0}

// DimensionTag identifies the variant of a Dimension value.
type DimensionTag int

const (
	// DimensionAuto represents an automatic size.
	// NOTE: Auto is placed first so that the zero-value of Dimension is Auto,
	// matching CSS defaults for width/height/min-size/max-size.
	DimensionAuto DimensionTag = iota
	// DimensionLength represents an absolute length.
	DimensionLength
	// DimensionPercent represents a percentage.
	DimensionPercent
	// DimensionCalc represents a calc() expression (stored as uint64 bits in Value).
	DimensionCalc
)

// Dimension is a unit of linear measurement that can be a length, percentage, auto, or calc.
type Dimension struct {
	Tag   DimensionTag
	Value float32
}

// DimensionFromLength creates a Dimension from an absolute length value.
func DimensionFromLength(val float32) Dimension {
	return Dimension{Tag: DimensionLength, Value: val}
}

// DimensionFromPercent creates a Dimension from a percentage value.
// NOTE: percentages are represented as a float32 value in the range [0.0, 1.0] NOT the range [0.0, 100.0]
func DimensionFromPercent(val float32) Dimension {
	return Dimension{Tag: DimensionPercent, Value: val}
}

// DimensionFromCalc creates a Dimension from a calc() ID.
// The id is stored in the Value field as float32 bits.
func DimensionFromCalc(id uint64) Dimension {
	return Dimension{Tag: DimensionCalc, Value: math.Float32frombits(uint32(id))}
}

// DimensionFromLengthPercentage converts a LengthPercentage to a Dimension.
func DimensionFromLengthPercentage(lp LengthPercentage) Dimension {
	switch lp.Tag {
	case LPLength:
		return Dimension{Tag: DimensionLength, Value: lp.Value}
	case LPPercent:
		return Dimension{Tag: DimensionPercent, Value: lp.Value}
	default:
		return Dimension{Tag: DimensionCalc, Value: lp.Value}
	}
}

// DimensionFromLengthPercentageAuto converts a LengthPercentageAuto to a Dimension.
func DimensionFromLengthPercentageAuto(lpa LengthPercentageAuto) Dimension {
	switch lpa.Tag {
	case LPALength:
		return Dimension{Tag: DimensionLength, Value: lpa.Value}
	case LPAPercent:
		return Dimension{Tag: DimensionPercent, Value: lpa.Value}
	case LPAAuto:
		return Dimension{Tag: DimensionAuto, Value: 0}
	default:
		return Dimension{Tag: DimensionCalc, Value: lpa.Value}
	}
}

// IsAuto returns true if this value is Auto.
func (d Dimension) IsAuto() bool {
	return d.Tag == DimensionAuto
}

// IntoOption returns Some(value) if this is a Length variant, otherwise None.
func (d Dimension) IntoOption() Option[float32] {
	if d.Tag == DimensionLength {
		return Some(d.Value)
	}
	return None[float32]()
}

// DimensionZero is a Dimension with value 0.
var DimensionZero = Dimension{Tag: DimensionLength, Value: 0}

// DimensionAuto is the auto variant of Dimension.
var DimensionAutoValue = Dimension{Tag: DimensionAuto, Value: 0}
