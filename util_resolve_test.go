// Tests for util_resolve.go — ported from util/resolve.rs #[cfg(test)] module.
package taffy

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// noopCalc is a CalcResolver that always returns 42.42, matching the Rust test harness.
var noopCalc CalcResolver = func(_ uint64, _ float32) float32 { return 42.42 }

// Helper constructors matching the Rust test helpers.

func sizeDimensionAuto() Size[Dimension] {
	return Size[Dimension]{
		Width:  DimensionAutoValue,
		Height: DimensionAutoValue,
	}
}

func sizeFromLengths(w, h float32) Size[Dimension] {
	return Size[Dimension]{
		Width:  DimensionFromLength(w),
		Height: DimensionFromLength(h),
	}
}

func sizeFromPercent(w, h float32) Size[Dimension] {
	return Size[Dimension]{
		Width:  DimensionFromPercent(w),
		Height: DimensionFromPercent(h),
	}
}

func sizeOptionNew(w, h float32) Size[Option[float32]] {
	return Size[Option[float32]]{
		Width:  Some(w),
		Height: Some(h),
	}
}

func rectDimensionAuto() Rect[Dimension] {
	return Rect[Dimension]{
		Left:   DimensionAutoValue,
		Right:  DimensionAutoValue,
		Top:    DimensionAutoValue,
		Bottom: DimensionAutoValue,
	}
}

func rectFromLength(l, r, t, b float32) Rect[Dimension] {
	return Rect[Dimension]{
		Left:   DimensionFromLength(l),
		Right:  DimensionFromLength(r),
		Top:    DimensionFromLength(t),
		Bottom: DimensionFromLength(b),
	}
}

func rectFromPercent(l, r, t, b float32) Rect[Dimension] {
	return Rect[Dimension]{
		Left:   DimensionFromPercent(l),
		Right:  DimensionFromPercent(r),
		Top:    DimensionFromPercent(t),
		Bottom: DimensionFromPercent(b),
	}
}

func rectNewF32(l, r, t, b float32) Rect[float32] {
	return NewRectF32(l, r, t, b)
}

var _ = Describe("MaybeResolve", func() {

	// -----------------------------------------------------------------------
	// maybe_resolve_dimension
	// -----------------------------------------------------------------------

	Describe("maybe_resolve_dimension", func() {
		It("resolve_auto", func() {
			Expect(MaybeResolveDimOption(DimensionAutoValue, None[float32](), noopCalc)).To(Equal(None[float32]()))
			Expect(MaybeResolveDimOption(DimensionAutoValue, Some[float32](5.0), noopCalc)).To(Equal(None[float32]()))
			Expect(MaybeResolveDimOption(DimensionAutoValue, Some[float32](-5.0), noopCalc)).To(Equal(None[float32]()))
			Expect(MaybeResolveDimOption(DimensionAutoValue, Some[float32](0.0), noopCalc)).To(Equal(None[float32]()))
		})

		It("resolve_length", func() {
			Expect(MaybeResolveDimOption(DimensionFromLength(1.0), None[float32](), noopCalc)).To(Equal(Some[float32](1.0)))
			Expect(MaybeResolveDimOption(DimensionFromLength(1.0), Some[float32](5.0), noopCalc)).To(Equal(Some[float32](1.0)))
			Expect(MaybeResolveDimOption(DimensionFromLength(1.0), Some[float32](-5.0), noopCalc)).To(Equal(Some[float32](1.0)))
			Expect(MaybeResolveDimOption(DimensionFromLength(1.0), Some[float32](0.0), noopCalc)).To(Equal(Some[float32](1.0)))
		})

		It("resolve_percent", func() {
			Expect(MaybeResolveDimOption(DimensionFromPercent(1.0), None[float32](), noopCalc)).To(Equal(None[float32]()))
			Expect(MaybeResolveDimOption(DimensionFromPercent(1.0), Some[float32](5.0), noopCalc)).To(Equal(Some[float32](5.0)))
			Expect(MaybeResolveDimOption(DimensionFromPercent(1.0), Some[float32](-5.0), noopCalc)).To(Equal(Some[float32](-5.0)))
			Expect(MaybeResolveDimOption(DimensionFromPercent(1.0), Some[float32](50.0), noopCalc)).To(Equal(Some[float32](50.0)))
		})
	})

	// -----------------------------------------------------------------------
	// maybe_resolve_size_dimension
	// -----------------------------------------------------------------------

	Describe("maybe_resolve_size_dimension", func() {
		It("maybe_resolve_auto", func() {
			Expect(MaybeResolveSizeDimOption(sizeDimensionAuto(), SizeOptionNone, noopCalc)).To(Equal(SizeOptionNone))
			Expect(MaybeResolveSizeDimOption(sizeDimensionAuto(), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(SizeOptionNone))
			Expect(MaybeResolveSizeDimOption(sizeDimensionAuto(), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(SizeOptionNone))
			Expect(MaybeResolveSizeDimOption(sizeDimensionAuto(), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(SizeOptionNone))
		})

		It("maybe_resolve_length", func() {
			Expect(MaybeResolveSizeDimOption(sizeFromLengths(5.0, 5.0), SizeOptionNone, noopCalc)).To(Equal(sizeOptionNew(5.0, 5.0)))
			Expect(MaybeResolveSizeDimOption(sizeFromLengths(5.0, 5.0), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(sizeOptionNew(5.0, 5.0)))
			Expect(MaybeResolveSizeDimOption(sizeFromLengths(5.0, 5.0), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(sizeOptionNew(5.0, 5.0)))
			Expect(MaybeResolveSizeDimOption(sizeFromLengths(5.0, 5.0), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(sizeOptionNew(5.0, 5.0)))
		})

		It("maybe_resolve_percent", func() {
			Expect(MaybeResolveSizeDimOption(sizeFromPercent(5.0, 5.0), SizeOptionNone, noopCalc)).To(Equal(SizeOptionNone))
			Expect(MaybeResolveSizeDimOption(sizeFromPercent(5.0, 5.0), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(sizeOptionNew(25.0, 25.0)))
			Expect(MaybeResolveSizeDimOption(sizeFromPercent(5.0, 5.0), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(sizeOptionNew(-25.0, -25.0)))
			Expect(MaybeResolveSizeDimOption(sizeFromPercent(5.0, 5.0), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(sizeOptionNew(0.0, 0.0)))
		})
	})
})

var _ = Describe("ResolveOrZero", func() {

	// -----------------------------------------------------------------------
	// resolve_or_zero_dimension_to_option_f32
	// -----------------------------------------------------------------------

	Describe("resolve_or_zero_dimension", func() {
		It("resolve_or_zero_auto", func() {
			Expect(ResolveOrZeroDim(DimensionAutoValue, None[float32](), noopCalc)).To(Equal(float32(0.0)))
			Expect(ResolveOrZeroDim(DimensionAutoValue, Some[float32](5.0), noopCalc)).To(Equal(float32(0.0)))
			Expect(ResolveOrZeroDim(DimensionAutoValue, Some[float32](-5.0), noopCalc)).To(Equal(float32(0.0)))
			Expect(ResolveOrZeroDim(DimensionAutoValue, Some[float32](0.0), noopCalc)).To(Equal(float32(0.0)))
		})

		It("resolve_or_zero_length", func() {
			Expect(ResolveOrZeroDim(DimensionFromLength(5.0), None[float32](), noopCalc)).To(Equal(float32(5.0)))
			Expect(ResolveOrZeroDim(DimensionFromLength(5.0), Some[float32](5.0), noopCalc)).To(Equal(float32(5.0)))
			Expect(ResolveOrZeroDim(DimensionFromLength(5.0), Some[float32](-5.0), noopCalc)).To(Equal(float32(5.0)))
			Expect(ResolveOrZeroDim(DimensionFromLength(5.0), Some[float32](0.0), noopCalc)).To(Equal(float32(5.0)))
		})

		It("resolve_or_zero_percent", func() {
			Expect(ResolveOrZeroDim(DimensionFromPercent(5.0), None[float32](), noopCalc)).To(Equal(float32(0.0)))
			Expect(ResolveOrZeroDim(DimensionFromPercent(5.0), Some[float32](5.0), noopCalc)).To(Equal(float32(25.0)))
			Expect(ResolveOrZeroDim(DimensionFromPercent(5.0), Some[float32](-5.0), noopCalc)).To(Equal(float32(-25.0)))
			Expect(ResolveOrZeroDim(DimensionFromPercent(5.0), Some[float32](0.0), noopCalc)).To(Equal(float32(0.0)))
		})
	})

	// -----------------------------------------------------------------------
	// resolve_or_zero_rect_dimension_to_rect (Size context)
	// -----------------------------------------------------------------------

	Describe("resolve_or_zero_rect_dimension_size_context", func() {
		It("resolve_or_zero_auto", func() {
			Expect(ResolveOrZeroRectDimSizeContext(rectDimensionAuto(), SizeOptionNone, noopCalc)).To(Equal(RectZeroF32))
			Expect(ResolveOrZeroRectDimSizeContext(rectDimensionAuto(), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(RectZeroF32))
			Expect(ResolveOrZeroRectDimSizeContext(rectDimensionAuto(), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(RectZeroF32))
			Expect(ResolveOrZeroRectDimSizeContext(rectDimensionAuto(), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(RectZeroF32))
		})

		It("resolve_or_zero_length", func() {
			Expect(ResolveOrZeroRectDimSizeContext(rectFromLength(5.0, 5.0, 5.0, 5.0), SizeOptionNone, noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(ResolveOrZeroRectDimSizeContext(rectFromLength(5.0, 5.0, 5.0, 5.0), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(ResolveOrZeroRectDimSizeContext(rectFromLength(5.0, 5.0, 5.0, 5.0), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(ResolveOrZeroRectDimSizeContext(rectFromLength(5.0, 5.0, 5.0, 5.0), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
		})

		It("resolve_or_zero_percent", func() {
			Expect(ResolveOrZeroRectDimSizeContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), SizeOptionNone, noopCalc)).To(Equal(RectZeroF32))
			Expect(ResolveOrZeroRectDimSizeContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(rectNewF32(25.0, 25.0, 25.0, 25.0)))
			Expect(ResolveOrZeroRectDimSizeContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(rectNewF32(-25.0, -25.0, -25.0, -25.0)))
			Expect(ResolveOrZeroRectDimSizeContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(RectZeroF32))
		})
	})

	// -----------------------------------------------------------------------
	// resolve_or_zero_rect_dimension_to_rect_f32_via_option (Option[float32] context)
	// -----------------------------------------------------------------------

	Describe("resolve_or_zero_rect_dimension_option_context", func() {
		It("resolve_or_zero_auto", func() {
			Expect(ResolveOrZeroRectDimOptionContext(rectDimensionAuto(), None[float32](), noopCalc)).To(Equal(RectZeroF32))
			Expect(ResolveOrZeroRectDimOptionContext(rectDimensionAuto(), Some[float32](5.0), noopCalc)).To(Equal(RectZeroF32))
			Expect(ResolveOrZeroRectDimOptionContext(rectDimensionAuto(), Some[float32](-5.0), noopCalc)).To(Equal(RectZeroF32))
			Expect(ResolveOrZeroRectDimOptionContext(rectDimensionAuto(), Some[float32](0.0), noopCalc)).To(Equal(RectZeroF32))
		})

		It("resolve_or_zero_length", func() {
			Expect(ResolveOrZeroRectDimOptionContext(rectFromLength(5.0, 5.0, 5.0, 5.0), None[float32](), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(ResolveOrZeroRectDimOptionContext(rectFromLength(5.0, 5.0, 5.0, 5.0), Some[float32](5.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(ResolveOrZeroRectDimOptionContext(rectFromLength(5.0, 5.0, 5.0, 5.0), Some[float32](-5.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(ResolveOrZeroRectDimOptionContext(rectFromLength(5.0, 5.0, 5.0, 5.0), Some[float32](0.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
		})

		It("resolve_or_zero_percent", func() {
			Expect(ResolveOrZeroRectDimOptionContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), None[float32](), noopCalc)).To(Equal(RectZeroF32))
			Expect(ResolveOrZeroRectDimOptionContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), Some[float32](5.0), noopCalc)).To(Equal(rectNewF32(25.0, 25.0, 25.0, 25.0)))
			Expect(ResolveOrZeroRectDimOptionContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), Some[float32](-5.0), noopCalc)).To(Equal(rectNewF32(-25.0, -25.0, -25.0, -25.0)))
			Expect(ResolveOrZeroRectDimOptionContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), Some[float32](0.0), noopCalc)).To(Equal(RectZeroF32))
		})
	})
})
