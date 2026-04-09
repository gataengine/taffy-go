// Package style provides MaybeMath helpers for AvailableSpace arithmetic.
// This file is a port of the AvailableSpace portions of util/math.rs from the Taffy Rust layout engine.
package style

import (
	"taffy-go/lib"
)

// ---------------------------------------------------------------------------
// Internal float32 min/max helpers
// ---------------------------------------------------------------------------

func f32Min(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func f32Max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// ---------------------------------------------------------------------------
// AvailableSpace with float32 rhs
// ---------------------------------------------------------------------------

// AsMaybeMinF32 returns the minimum of a and b.
// For MinContent/MaxContent, the result is Definite(b) since b is a hard cap.
func AsMaybeMinF32(a AvailableSpace, b float32) AvailableSpace {
	switch a.Tag {
	case AvailableSpaceDefinite:
		return AvailableSpaceFromDefinite(f32Min(a.Value, b))
	default:
		return AvailableSpaceFromDefinite(b)
	}
}

// AsMaybeMaxF32 returns the maximum of a and b.
// For MinContent/MaxContent, the space is returned unchanged.
func AsMaybeMaxF32(a AvailableSpace, b float32) AvailableSpace {
	switch a.Tag {
	case AvailableSpaceDefinite:
		return AvailableSpaceFromDefinite(f32Max(a.Value, b))
	default:
		return a
	}
}

// AsMaybeAddF32 adds b to a.
// For MinContent/MaxContent, the space is returned unchanged.
func AsMaybeAddF32(a AvailableSpace, b float32) AvailableSpace {
	switch a.Tag {
	case AvailableSpaceDefinite:
		return AvailableSpaceFromDefinite(a.Value + b)
	default:
		return a
	}
}

// AsMaybeSubF32 subtracts b from a.
// For MinContent/MaxContent, the space is returned unchanged.
func AsMaybeSubF32(a AvailableSpace, b float32) AvailableSpace {
	switch a.Tag {
	case AvailableSpaceDefinite:
		return AvailableSpaceFromDefinite(a.Value - b)
	default:
		return a
	}
}

// ---------------------------------------------------------------------------
// AvailableSpace with Option[float32] rhs
// ---------------------------------------------------------------------------

// AsMaybeMinOption returns the minimum of a and b.
// For Definite: applies min if b is Some, else unchanged.
// For MinContent/MaxContent: collapses to Definite(b) if b is Some, else unchanged.
func AsMaybeMinOption(a AvailableSpace, b lib.Option[float32]) AvailableSpace {
	switch a.Tag {
	case AvailableSpaceDefinite:
		if b.IsSome {
			return AvailableSpaceFromDefinite(f32Min(a.Value, b.Value))
		}
		return a
	case AvailableSpaceMinContent:
		if b.IsSome {
			return AvailableSpaceFromDefinite(b.Value)
		}
		return AvailableSpaceMinContentValue
	default: // MaxContent
		if b.IsSome {
			return AvailableSpaceFromDefinite(b.Value)
		}
		return AvailableSpaceMaxContentValue
	}
}

// AsMaybeMaxOption returns the maximum of a and b.
// For Definite: applies max if b is Some, else unchanged.
// For MinContent/MaxContent: unchanged regardless of b.
func AsMaybeMaxOption(a AvailableSpace, b lib.Option[float32]) AvailableSpace {
	switch a.Tag {
	case AvailableSpaceDefinite:
		if b.IsSome {
			return AvailableSpaceFromDefinite(f32Max(a.Value, b.Value))
		}
		return a
	default:
		return a
	}
}

// AsMaybeAddOption adds b to a.
// For Definite: adds b if Some, else unchanged.
// For MinContent/MaxContent: unchanged.
func AsMaybeAddOption(a AvailableSpace, b lib.Option[float32]) AvailableSpace {
	switch a.Tag {
	case AvailableSpaceDefinite:
		if b.IsSome {
			return AvailableSpaceFromDefinite(a.Value + b.Value)
		}
		return a
	default:
		return a
	}
}

// AsMaybeSubOption subtracts b from a.
// For Definite: subtracts b if Some, else unchanged.
// For MinContent/MaxContent: unchanged.
func AsMaybeSubOption(a AvailableSpace, b lib.Option[float32]) AvailableSpace {
	switch a.Tag {
	case AvailableSpaceDefinite:
		if b.IsSome {
			return AvailableSpaceFromDefinite(a.Value - b.Value)
		}
		return a
	default:
		return a
	}
}

// ---------------------------------------------------------------------------
// Package-internal aliases (unexported) for use within this package.
// ---------------------------------------------------------------------------

func asMaybeMinF32(a AvailableSpace, b float32) AvailableSpace { return AsMaybeMinF32(a, b) }
func asMaybeMaxF32(a AvailableSpace, b float32) AvailableSpace { return AsMaybeMaxF32(a, b) }
func asMaybeAddF32(a AvailableSpace, b float32) AvailableSpace { return AsMaybeAddF32(a, b) }
func asMaybeSubF32(a AvailableSpace, b float32) AvailableSpace { return AsMaybeSubF32(a, b) }
func asMaybeMinOption(a AvailableSpace, b lib.Option[float32]) AvailableSpace {
	return AsMaybeMinOption(a, b)
}
func asMaybeMaxOption(a AvailableSpace, b lib.Option[float32]) AvailableSpace {
	return AsMaybeMaxOption(a, b)
}
func asMaybeAddOption(a AvailableSpace, b lib.Option[float32]) AvailableSpace {
	return AsMaybeAddOption(a, b)
}
func asMaybeSubOption(a AvailableSpace, b lib.Option[float32]) AvailableSpace {
	return AsMaybeSubOption(a, b)
}
