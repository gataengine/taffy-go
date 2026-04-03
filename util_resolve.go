// Package taffy provides MaybeResolve and ResolveOrZero helpers.
// This file is a port of util/resolve.rs from the Taffy Rust layout engine.
package taffy

import "math"

// ---------------------------------------------------------------------------
// MaybeResolve for LengthPercentage
// ---------------------------------------------------------------------------

// MaybeResolveLPOption resolves a LengthPercentage to Option[float32] given an optional context.
// Length values resolve to Some(value). Percent values resolve to Some(context * value) if
// context is Some, or None if context is None. Calc values resolve similarly to Percent.
func MaybeResolveLPOption(lp LengthPercentage, context Option[float32], calc CalcResolver) Option[float32] {
	switch lp.Tag {
	case LPLength:
		return Some(lp.Value)
	case LPPercent:
		if !context.IsSome {
			return None[float32]()
		}
		return Some(context.Value * lp.Value)
	case LPCalc:
		if !context.IsSome {
			return None[float32]()
		}
		calcId := uint64(math.Float32bits(lp.Value))
		return Some(calc(calcId, context.Value))
	default:
		return None[float32]()
	}
}

// MaybeResolveLPF32 resolves a LengthPercentage to Option[float32] given a definite context.
func MaybeResolveLPF32(lp LengthPercentage, context float32, calc CalcResolver) Option[float32] {
	return MaybeResolveLPOption(lp, Some(context), calc)
}

// ---------------------------------------------------------------------------
// MaybeResolve for LengthPercentageAuto
// ---------------------------------------------------------------------------

// MaybeResolveLPAOption resolves a LengthPercentageAuto to Option[float32] given an optional context.
// Auto resolves to None. Length/Percent/Calc follow the same rules as LengthPercentage.
func MaybeResolveLPAOption(lpa LengthPercentageAuto, context Option[float32], calc CalcResolver) Option[float32] {
	switch lpa.Tag {
	case LPAAuto:
		return None[float32]()
	case LPALength:
		return Some(lpa.Value)
	case LPAPercent:
		if !context.IsSome {
			return None[float32]()
		}
		return Some(context.Value * lpa.Value)
	case LPACalc:
		if !context.IsSome {
			return None[float32]()
		}
		calcId := uint64(math.Float32bits(lpa.Value))
		return Some(calc(calcId, context.Value))
	default:
		return None[float32]()
	}
}

// MaybeResolveLPAF32 resolves a LengthPercentageAuto to Option[float32] given a definite context.
func MaybeResolveLPAF32(lpa LengthPercentageAuto, context float32, calc CalcResolver) Option[float32] {
	return MaybeResolveLPAOption(lpa, Some(context), calc)
}

// ---------------------------------------------------------------------------
// MaybeResolve for Dimension
// ---------------------------------------------------------------------------

// MaybeResolveDimOption resolves a Dimension to Option[float32] given an optional context.
// Auto resolves to None. Length/Percent/Calc follow the same rules as LengthPercentageAuto.
func MaybeResolveDimOption(dim Dimension, context Option[float32], calc CalcResolver) Option[float32] {
	switch dim.Tag {
	case DimensionAuto:
		return None[float32]()
	case DimensionLength:
		return Some(dim.Value)
	case DimensionPercent:
		if !context.IsSome {
			return None[float32]()
		}
		return Some(context.Value * dim.Value)
	case DimensionCalc:
		if !context.IsSome {
			return None[float32]()
		}
		calcId := uint64(math.Float32bits(dim.Value))
		return Some(calc(calcId, context.Value))
	default:
		return None[float32]()
	}
}

// MaybeResolveDimF32 resolves a Dimension to Option[float32] given a definite context.
func MaybeResolveDimF32(dim Dimension, context float32, calc CalcResolver) Option[float32] {
	return MaybeResolveDimOption(dim, Some(context), calc)
}

// ---------------------------------------------------------------------------
// MaybeResolve for Size
// ---------------------------------------------------------------------------

// MaybeResolveSizeDimOption resolves a Size[Dimension] given a Size[Option[float32]] context.
func MaybeResolveSizeDimOption(size Size[Dimension], context Size[Option[float32]], calc CalcResolver) Size[Option[float32]] {
	return Size[Option[float32]]{
		Width:  MaybeResolveDimOption(size.Width, context.Width, calc),
		Height: MaybeResolveDimOption(size.Height, context.Height, calc),
	}
}

// MaybeResolveSizeLPOption resolves a Size[LengthPercentage] given a Size[Option[float32]] context.
func MaybeResolveSizeLPOption(size Size[LengthPercentage], context Size[Option[float32]], calc CalcResolver) Size[Option[float32]] {
	return Size[Option[float32]]{
		Width:  MaybeResolveLPOption(size.Width, context.Width, calc),
		Height: MaybeResolveLPOption(size.Height, context.Height, calc),
	}
}

// MaybeResolveSizeLPAOption resolves a Size[LengthPercentageAuto] given a Size[Option[float32]] context.
func MaybeResolveSizeLPAOption(size Size[LengthPercentageAuto], context Size[Option[float32]], calc CalcResolver) Size[Option[float32]] {
	return Size[Option[float32]]{
		Width:  MaybeResolveLPAOption(size.Width, context.Width, calc),
		Height: MaybeResolveLPAOption(size.Height, context.Height, calc),
	}
}

// ---------------------------------------------------------------------------
// ResolveOrZero for LengthPercentage
// ---------------------------------------------------------------------------

// ResolveOrZeroLP resolves a LengthPercentage to float32, returning 0 if the result would be None.
func ResolveOrZeroLP(lp LengthPercentage, context Option[float32], calc CalcResolver) float32 {
	return MaybeResolveLPOption(lp, context, calc).UnwrapOr(0)
}

// ---------------------------------------------------------------------------
// ResolveOrZero for LengthPercentageAuto
// ---------------------------------------------------------------------------

// ResolveOrZeroLPA resolves a LengthPercentageAuto to float32, returning 0 if the result would be None.
func ResolveOrZeroLPA(lpa LengthPercentageAuto, context Option[float32], calc CalcResolver) float32 {
	return MaybeResolveLPAOption(lpa, context, calc).UnwrapOr(0)
}

// ---------------------------------------------------------------------------
// ResolveOrZero for Dimension
// ---------------------------------------------------------------------------

// ResolveOrZeroDim resolves a Dimension to float32, returning 0 if the result would be None.
func ResolveOrZeroDim(dim Dimension, context Option[float32], calc CalcResolver) float32 {
	return MaybeResolveDimOption(dim, context, calc).UnwrapOr(0)
}

// ---------------------------------------------------------------------------
// ResolveOrZero for Size
// ---------------------------------------------------------------------------

// ResolveOrZeroSizeDim resolves a Size[Dimension] to Size[float32] against a Size[Option[float32]] context.
func ResolveOrZeroSizeDim(size Size[Dimension], context Size[Option[float32]], calc CalcResolver) Size[float32] {
	return Size[float32]{
		Width:  ResolveOrZeroDim(size.Width, context.Width, calc),
		Height: ResolveOrZeroDim(size.Height, context.Height, calc),
	}
}

// ResolveOrZeroSizeLPA resolves a Size[LengthPercentageAuto] to Size[float32] against a Size[Option[float32]] context.
func ResolveOrZeroSizeLPA(size Size[LengthPercentageAuto], context Size[Option[float32]], calc CalcResolver) Size[float32] {
	return Size[float32]{
		Width:  ResolveOrZeroLPA(size.Width, context.Width, calc),
		Height: ResolveOrZeroLPA(size.Height, context.Height, calc),
	}
}

// ---------------------------------------------------------------------------
// ResolveOrZero for Rect — resolved against Size context (l/r use width, t/b use height)
// ---------------------------------------------------------------------------

// ResolveOrZeroRectLPSizeContext resolves a Rect[LengthPercentage] to Rect[float32] against a Size context.
func ResolveOrZeroRectLPSizeContext(rect Rect[LengthPercentage], context Size[Option[float32]], calc CalcResolver) Rect[float32] {
	return Rect[float32]{
		Left:   ResolveOrZeroLP(rect.Left, context.Width, calc),
		Right:  ResolveOrZeroLP(rect.Right, context.Width, calc),
		Top:    ResolveOrZeroLP(rect.Top, context.Height, calc),
		Bottom: ResolveOrZeroLP(rect.Bottom, context.Height, calc),
	}
}

// ResolveOrZeroRectLPASizeContext resolves a Rect[LengthPercentageAuto] to Rect[float32] against a Size context.
func ResolveOrZeroRectLPASizeContext(rect Rect[LengthPercentageAuto], context Size[Option[float32]], calc CalcResolver) Rect[float32] {
	return Rect[float32]{
		Left:   ResolveOrZeroLPA(rect.Left, context.Width, calc),
		Right:  ResolveOrZeroLPA(rect.Right, context.Width, calc),
		Top:    ResolveOrZeroLPA(rect.Top, context.Height, calc),
		Bottom: ResolveOrZeroLPA(rect.Bottom, context.Height, calc),
	}
}

// ResolveOrZeroRectDimSizeContext resolves a Rect[Dimension] to Rect[float32] against a Size context.
func ResolveOrZeroRectDimSizeContext(rect Rect[Dimension], context Size[Option[float32]], calc CalcResolver) Rect[float32] {
	return Rect[float32]{
		Left:   ResolveOrZeroDim(rect.Left, context.Width, calc),
		Right:  ResolveOrZeroDim(rect.Right, context.Width, calc),
		Top:    ResolveOrZeroDim(rect.Top, context.Height, calc),
		Bottom: ResolveOrZeroDim(rect.Bottom, context.Height, calc),
	}
}

// ResolveOrZeroRectDimOptionContext resolves a Rect[Dimension] to Rect[float32] against an Option[float32] context.
// All four sides use the same context value.
func ResolveOrZeroRectDimOptionContext(rect Rect[Dimension], context Option[float32], calc CalcResolver) Rect[float32] {
	return Rect[float32]{
		Left:   ResolveOrZeroDim(rect.Left, context, calc),
		Right:  ResolveOrZeroDim(rect.Right, context, calc),
		Top:    ResolveOrZeroDim(rect.Top, context, calc),
		Bottom: ResolveOrZeroDim(rect.Bottom, context, calc),
	}
}
