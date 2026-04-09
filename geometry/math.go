// Package geometry provides MaybeMath helpers for optional/constrained arithmetic.
// This file is a port of util/math.rs from the Taffy Rust layout engine.
package geometry

import "taffy-go/lib"

// ---------------------------------------------------------------------------
// lib.Option[float32] with lib.Option[float32] rhs
// ---------------------------------------------------------------------------

// MaybeMinOption returns the minimum of a and b.
// If a is None, returns None. If b is None, returns a unchanged.
func MaybeMinOption(a, b lib.Option[float32]) lib.Option[float32] {
	if !a.IsSome {
		return lib.None[float32]()
	}
	if !b.IsSome {
		return a
	}
	return lib.Some(f32Min(a.Value, b.Value))
}

// MaybeMaxOption returns the maximum of a and b.
// If a is None, returns None. If b is None, returns a unchanged.
func MaybeMaxOption(a, b lib.Option[float32]) lib.Option[float32] {
	if !a.IsSome {
		return lib.None[float32]()
	}
	if !b.IsSome {
		return a
	}
	return lib.Some(f32Max(a.Value, b.Value))
}

// MaybeAddOption adds a and b.
// If a is None, returns None. If b is None, returns a unchanged.
func MaybeAddOption(a, b lib.Option[float32]) lib.Option[float32] {
	if !a.IsSome {
		return lib.None[float32]()
	}
	if !b.IsSome {
		return a
	}
	return lib.Some(a.Value + b.Value)
}

// MaybeSubOption subtracts b from a.
// If a is None, returns None. If b is None, returns a unchanged.
func MaybeSubOption(a, b lib.Option[float32]) lib.Option[float32] {
	if !a.IsSome {
		return lib.None[float32]()
	}
	if !b.IsSome {
		return a
	}
	return lib.Some(a.Value - b.Value)
}

// MaybeClampOption clamps val between min_ and max_.
// If val is None, returns None. None min/max values are ignored.
func MaybeClampOption(val, min_, max_ lib.Option[float32]) lib.Option[float32] {
	if !val.IsSome {
		return lib.None[float32]()
	}
	v := val.Value
	if min_.IsSome && max_.IsSome {
		return lib.Some(f32Max(f32Min(v, max_.Value), min_.Value))
	}
	if min_.IsSome {
		return lib.Some(f32Max(v, min_.Value))
	}
	if max_.IsSome {
		return lib.Some(f32Min(v, max_.Value))
	}
	return val
}

// ---------------------------------------------------------------------------
// float32 with lib.Option[float32] rhs
// ---------------------------------------------------------------------------

// MaybeMinF32 returns the minimum of a and b.
// If b is None, returns a unchanged.
func MaybeMinF32(a float32, b lib.Option[float32]) float32 {
	if !b.IsSome {
		return a
	}
	return f32Min(a, b.Value)
}

// MaybeMaxF32 returns the maximum of a and b.
// If b is None, returns a unchanged.
func MaybeMaxF32(a float32, b lib.Option[float32]) float32 {
	if !b.IsSome {
		return a
	}
	return f32Max(a, b.Value)
}

// MaybeAddF32 adds a and b.
// If b is None, returns a unchanged.
func MaybeAddF32(a float32, b lib.Option[float32]) float32 {
	if !b.IsSome {
		return a
	}
	return a + b.Value
}

// MaybeSubF32 subtracts b from a.
// If b is None, returns a unchanged.
func MaybeSubF32(a float32, b lib.Option[float32]) float32 {
	if !b.IsSome {
		return a
	}
	return a - b.Value
}

// MaybeClampF32 clamps val between min_ and max_.
// None min/max values are ignored.
func MaybeClampF32(val float32, min_, max_ lib.Option[float32]) float32 {
	if min_.IsSome && max_.IsSome {
		return f32Max(f32Min(val, max_.Value), min_.Value)
	}
	if min_.IsSome {
		return f32Max(val, min_.Value)
	}
	if max_.IsSome {
		return f32Min(val, max_.Value)
	}
	return val
}

// ---------------------------------------------------------------------------
// Size[lib.Option[float32]] versions
// ---------------------------------------------------------------------------

// SizeMaybeMin returns the component-wise minimum of a and b.
func SizeMaybeMin(a, b Size[lib.Option[float32]]) Size[lib.Option[float32]] {
	return Size[lib.Option[float32]]{
		Width:  MaybeMinOption(a.Width, b.Width),
		Height: MaybeMinOption(a.Height, b.Height),
	}
}

// SizeMaybeMax returns the component-wise maximum of a and b.
func SizeMaybeMax(a, b Size[lib.Option[float32]]) Size[lib.Option[float32]] {
	return Size[lib.Option[float32]]{
		Width:  MaybeMaxOption(a.Width, b.Width),
		Height: MaybeMaxOption(a.Height, b.Height),
	}
}

// SizeMaybeAdd adds b to a component-wise.
func SizeMaybeAdd(a, b Size[lib.Option[float32]]) Size[lib.Option[float32]] {
	return Size[lib.Option[float32]]{
		Width:  MaybeAddOption(a.Width, b.Width),
		Height: MaybeAddOption(a.Height, b.Height),
	}
}

// SizeMaybeSub subtracts b from a component-wise.
func SizeMaybeSub(a, b Size[lib.Option[float32]]) Size[lib.Option[float32]] {
	return Size[lib.Option[float32]]{
		Width:  MaybeSubOption(a.Width, b.Width),
		Height: MaybeSubOption(a.Height, b.Height),
	}
}

// SizeMaybeClamp clamps val component-wise between min_ and max_.
func SizeMaybeClamp(val, min_, max_ Size[lib.Option[float32]]) Size[lib.Option[float32]] {
	return Size[lib.Option[float32]]{
		Width:  MaybeClampOption(val.Width, min_.Width, max_.Width),
		Height: MaybeClampOption(val.Height, min_.Height, max_.Height),
	}
}

// ---------------------------------------------------------------------------
// Package-internal aliases (unexported) for use within this package.
// ---------------------------------------------------------------------------

func maybeMinOption(a, b lib.Option[float32]) lib.Option[float32]        { return MaybeMinOption(a, b) }
func maybeMaxOption(a, b lib.Option[float32]) lib.Option[float32]        { return MaybeMaxOption(a, b) }
func maybeAddOption(a, b lib.Option[float32]) lib.Option[float32]        { return MaybeAddOption(a, b) }
func maybeSubOption(a, b lib.Option[float32]) lib.Option[float32]        { return MaybeSubOption(a, b) }
func maybeClampOption(v, mn, mx lib.Option[float32]) lib.Option[float32] { return MaybeClampOption(v, mn, mx) }
func maybeMinF32(a float32, b lib.Option[float32]) float32           { return MaybeMinF32(a, b) }
func maybeMaxF32(a float32, b lib.Option[float32]) float32           { return MaybeMaxF32(a, b) }
func maybeAddF32(a float32, b lib.Option[float32]) float32           { return MaybeAddF32(a, b) }
func maybeSubF32(a float32, b lib.Option[float32]) float32           { return MaybeSubF32(a, b) }
func maybeClampF32(val float32, mn, mx lib.Option[float32]) float32  { return MaybeClampF32(val, mn, mx) }
func sizeMaybeMin(a, b Size[lib.Option[float32]]) Size[lib.Option[float32]] {
	return SizeMaybeMin(a, b)
}
func sizeMaybeMax(a, b Size[lib.Option[float32]]) Size[lib.Option[float32]] {
	return SizeMaybeMax(a, b)
}
func sizeMaybeAdd(a, b Size[lib.Option[float32]]) Size[lib.Option[float32]] {
	return SizeMaybeAdd(a, b)
}
func sizeMaybeSub(a, b Size[lib.Option[float32]]) Size[lib.Option[float32]] {
	return SizeMaybeSub(a, b)
}
func sizeMaybeClamp(v, mn, mx Size[lib.Option[float32]]) Size[lib.Option[float32]] {
	return SizeMaybeClamp(v, mn, mx)
}
