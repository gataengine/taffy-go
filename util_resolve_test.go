// Tests for util_resolve.go — ported from util/resolve.rs #[cfg(test)] module.
package taffy_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	taffy "taffy-go"
)

// noopCalc is a CalcResolver that always returns 42.42, matching the Rust test harness.
var noopCalc taffy.CalcResolver = func(_ uint64, _ float32) float32 { return 42.42 }

// Helper constructors matching the Rust test helpers.

func sizeDimensionAuto() taffy.Size[taffy.Dimension] {
	return taffy.Size[taffy.Dimension]{
		Width:  taffy.DimensionAutoValue,
		Height: taffy.DimensionAutoValue,
	}
}

func sizeFromLengths(w, h float32) taffy.Size[taffy.Dimension] {
	return taffy.Size[taffy.Dimension]{
		Width:  taffy.DimensionFromLength(w),
		Height: taffy.DimensionFromLength(h),
	}
}

func sizeFromPercent(w, h float32) taffy.Size[taffy.Dimension] {
	return taffy.Size[taffy.Dimension]{
		Width:  taffy.DimensionFromPercent(w),
		Height: taffy.DimensionFromPercent(h),
	}
}

func sizeOptionNew(w, h float32) taffy.Size[taffy.Option[float32]] {
	return taffy.Size[taffy.Option[float32]]{
		Width:  taffy.Some(w),
		Height: taffy.Some(h),
	}
}

func rectDimensionAuto() taffy.Rect[taffy.Dimension] {
	return taffy.Rect[taffy.Dimension]{
		Left:   taffy.DimensionAutoValue,
		Right:  taffy.DimensionAutoValue,
		Top:    taffy.DimensionAutoValue,
		Bottom: taffy.DimensionAutoValue,
	}
}

func rectFromLength(l, r, t, b float32) taffy.Rect[taffy.Dimension] {
	return taffy.Rect[taffy.Dimension]{
		Left:   taffy.DimensionFromLength(l),
		Right:  taffy.DimensionFromLength(r),
		Top:    taffy.DimensionFromLength(t),
		Bottom: taffy.DimensionFromLength(b),
	}
}

func rectFromPercent(l, r, t, b float32) taffy.Rect[taffy.Dimension] {
	return taffy.Rect[taffy.Dimension]{
		Left:   taffy.DimensionFromPercent(l),
		Right:  taffy.DimensionFromPercent(r),
		Top:    taffy.DimensionFromPercent(t),
		Bottom: taffy.DimensionFromPercent(b),
	}
}

func rectNewF32(l, r, t, b float32) taffy.Rect[float32] {
	return taffy.NewRectF32(l, r, t, b)
}

var _ = Describe("MaybeResolve", func() {

	// -----------------------------------------------------------------------
	// maybe_resolve_dimension
	// -----------------------------------------------------------------------

	Describe("maybe_resolve_dimension", func() {
		It("resolve_auto", func() {
			Expect(taffy.MaybeResolveDimOption(taffy.DimensionAutoValue, taffy.None[float32](), noopCalc)).To(Equal(taffy.None[float32]()))
			Expect(taffy.MaybeResolveDimOption(taffy.DimensionAutoValue, taffy.Some[float32](5.0), noopCalc)).To(Equal(taffy.None[float32]()))
			Expect(taffy.MaybeResolveDimOption(taffy.DimensionAutoValue, taffy.Some[float32](-5.0), noopCalc)).To(Equal(taffy.None[float32]()))
			Expect(taffy.MaybeResolveDimOption(taffy.DimensionAutoValue, taffy.Some[float32](0.0), noopCalc)).To(Equal(taffy.None[float32]()))
		})

		It("resolve_length", func() {
			Expect(taffy.MaybeResolveDimOption(taffy.DimensionFromLength(1.0), taffy.None[float32](), noopCalc)).To(Equal(taffy.Some[float32](1.0)))
			Expect(taffy.MaybeResolveDimOption(taffy.DimensionFromLength(1.0), taffy.Some[float32](5.0), noopCalc)).To(Equal(taffy.Some[float32](1.0)))
			Expect(taffy.MaybeResolveDimOption(taffy.DimensionFromLength(1.0), taffy.Some[float32](-5.0), noopCalc)).To(Equal(taffy.Some[float32](1.0)))
			Expect(taffy.MaybeResolveDimOption(taffy.DimensionFromLength(1.0), taffy.Some[float32](0.0), noopCalc)).To(Equal(taffy.Some[float32](1.0)))
		})

		It("resolve_percent", func() {
			Expect(taffy.MaybeResolveDimOption(taffy.DimensionFromPercent(1.0), taffy.None[float32](), noopCalc)).To(Equal(taffy.None[float32]()))
			Expect(taffy.MaybeResolveDimOption(taffy.DimensionFromPercent(1.0), taffy.Some[float32](5.0), noopCalc)).To(Equal(taffy.Some[float32](5.0)))
			Expect(taffy.MaybeResolveDimOption(taffy.DimensionFromPercent(1.0), taffy.Some[float32](-5.0), noopCalc)).To(Equal(taffy.Some[float32](-5.0)))
			Expect(taffy.MaybeResolveDimOption(taffy.DimensionFromPercent(1.0), taffy.Some[float32](50.0), noopCalc)).To(Equal(taffy.Some[float32](50.0)))
		})
	})

	// -----------------------------------------------------------------------
	// maybe_resolve_size_dimension
	// -----------------------------------------------------------------------

	Describe("maybe_resolve_size_dimension", func() {
		It("maybe_resolve_auto", func() {
			Expect(taffy.MaybeResolveSizeDimOption(sizeDimensionAuto(), taffy.SizeOptionNone, noopCalc)).To(Equal(taffy.SizeOptionNone))
			Expect(taffy.MaybeResolveSizeDimOption(sizeDimensionAuto(), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(taffy.SizeOptionNone))
			Expect(taffy.MaybeResolveSizeDimOption(sizeDimensionAuto(), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(taffy.SizeOptionNone))
			Expect(taffy.MaybeResolveSizeDimOption(sizeDimensionAuto(), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(taffy.SizeOptionNone))
		})

		It("maybe_resolve_length", func() {
			Expect(taffy.MaybeResolveSizeDimOption(sizeFromLengths(5.0, 5.0), taffy.SizeOptionNone, noopCalc)).To(Equal(sizeOptionNew(5.0, 5.0)))
			Expect(taffy.MaybeResolveSizeDimOption(sizeFromLengths(5.0, 5.0), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(sizeOptionNew(5.0, 5.0)))
			Expect(taffy.MaybeResolveSizeDimOption(sizeFromLengths(5.0, 5.0), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(sizeOptionNew(5.0, 5.0)))
			Expect(taffy.MaybeResolveSizeDimOption(sizeFromLengths(5.0, 5.0), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(sizeOptionNew(5.0, 5.0)))
		})

		It("maybe_resolve_percent", func() {
			Expect(taffy.MaybeResolveSizeDimOption(sizeFromPercent(5.0, 5.0), taffy.SizeOptionNone, noopCalc)).To(Equal(taffy.SizeOptionNone))
			Expect(taffy.MaybeResolveSizeDimOption(sizeFromPercent(5.0, 5.0), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(sizeOptionNew(25.0, 25.0)))
			Expect(taffy.MaybeResolveSizeDimOption(sizeFromPercent(5.0, 5.0), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(sizeOptionNew(-25.0, -25.0)))
			Expect(taffy.MaybeResolveSizeDimOption(sizeFromPercent(5.0, 5.0), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(sizeOptionNew(0.0, 0.0)))
		})
	})
})

var _ = Describe("ResolveOrZero", func() {

	// -----------------------------------------------------------------------
	// resolve_or_zero_dimension_to_option_f32
	// -----------------------------------------------------------------------

	Describe("resolve_or_zero_dimension", func() {
		It("resolve_or_zero_auto", func() {
			Expect(taffy.ResolveOrZeroDim(taffy.DimensionAutoValue, taffy.None[float32](), noopCalc)).To(Equal(float32(0.0)))
			Expect(taffy.ResolveOrZeroDim(taffy.DimensionAutoValue, taffy.Some[float32](5.0), noopCalc)).To(Equal(float32(0.0)))
			Expect(taffy.ResolveOrZeroDim(taffy.DimensionAutoValue, taffy.Some[float32](-5.0), noopCalc)).To(Equal(float32(0.0)))
			Expect(taffy.ResolveOrZeroDim(taffy.DimensionAutoValue, taffy.Some[float32](0.0), noopCalc)).To(Equal(float32(0.0)))
		})

		It("resolve_or_zero_length", func() {
			Expect(taffy.ResolveOrZeroDim(taffy.DimensionFromLength(5.0), taffy.None[float32](), noopCalc)).To(Equal(float32(5.0)))
			Expect(taffy.ResolveOrZeroDim(taffy.DimensionFromLength(5.0), taffy.Some[float32](5.0), noopCalc)).To(Equal(float32(5.0)))
			Expect(taffy.ResolveOrZeroDim(taffy.DimensionFromLength(5.0), taffy.Some[float32](-5.0), noopCalc)).To(Equal(float32(5.0)))
			Expect(taffy.ResolveOrZeroDim(taffy.DimensionFromLength(5.0), taffy.Some[float32](0.0), noopCalc)).To(Equal(float32(5.0)))
		})

		It("resolve_or_zero_percent", func() {
			Expect(taffy.ResolveOrZeroDim(taffy.DimensionFromPercent(5.0), taffy.None[float32](), noopCalc)).To(Equal(float32(0.0)))
			Expect(taffy.ResolveOrZeroDim(taffy.DimensionFromPercent(5.0), taffy.Some[float32](5.0), noopCalc)).To(Equal(float32(25.0)))
			Expect(taffy.ResolveOrZeroDim(taffy.DimensionFromPercent(5.0), taffy.Some[float32](-5.0), noopCalc)).To(Equal(float32(-25.0)))
			Expect(taffy.ResolveOrZeroDim(taffy.DimensionFromPercent(5.0), taffy.Some[float32](0.0), noopCalc)).To(Equal(float32(0.0)))
		})
	})

	// -----------------------------------------------------------------------
	// resolve_or_zero_rect_dimension_to_rect (Size context)
	// -----------------------------------------------------------------------

	Describe("resolve_or_zero_rect_dimension_size_context", func() {
		It("resolve_or_zero_auto", func() {
			Expect(taffy.ResolveOrZeroRectDimSizeContext(rectDimensionAuto(), taffy.SizeOptionNone, noopCalc)).To(Equal(taffy.RectZeroF32))
			Expect(taffy.ResolveOrZeroRectDimSizeContext(rectDimensionAuto(), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(taffy.RectZeroF32))
			Expect(taffy.ResolveOrZeroRectDimSizeContext(rectDimensionAuto(), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(taffy.RectZeroF32))
			Expect(taffy.ResolveOrZeroRectDimSizeContext(rectDimensionAuto(), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(taffy.RectZeroF32))
		})

		It("resolve_or_zero_length", func() {
			Expect(taffy.ResolveOrZeroRectDimSizeContext(rectFromLength(5.0, 5.0, 5.0, 5.0), taffy.SizeOptionNone, noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(taffy.ResolveOrZeroRectDimSizeContext(rectFromLength(5.0, 5.0, 5.0, 5.0), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(taffy.ResolveOrZeroRectDimSizeContext(rectFromLength(5.0, 5.0, 5.0, 5.0), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(taffy.ResolveOrZeroRectDimSizeContext(rectFromLength(5.0, 5.0, 5.0, 5.0), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
		})

		It("resolve_or_zero_percent", func() {
			Expect(taffy.ResolveOrZeroRectDimSizeContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), taffy.SizeOptionNone, noopCalc)).To(Equal(taffy.RectZeroF32))
			Expect(taffy.ResolveOrZeroRectDimSizeContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(rectNewF32(25.0, 25.0, 25.0, 25.0)))
			Expect(taffy.ResolveOrZeroRectDimSizeContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(rectNewF32(-25.0, -25.0, -25.0, -25.0)))
			Expect(taffy.ResolveOrZeroRectDimSizeContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(taffy.RectZeroF32))
		})
	})

	// -----------------------------------------------------------------------
	// resolve_or_zero_rect_dimension_to_rect_f32_via_option (Option[float32] context)
	// -----------------------------------------------------------------------

	Describe("resolve_or_zero_rect_dimension_option_context", func() {
		It("resolve_or_zero_auto", func() {
			Expect(taffy.ResolveOrZeroRectDimOptionContext(rectDimensionAuto(), taffy.None[float32](), noopCalc)).To(Equal(taffy.RectZeroF32))
			Expect(taffy.ResolveOrZeroRectDimOptionContext(rectDimensionAuto(), taffy.Some[float32](5.0), noopCalc)).To(Equal(taffy.RectZeroF32))
			Expect(taffy.ResolveOrZeroRectDimOptionContext(rectDimensionAuto(), taffy.Some[float32](-5.0), noopCalc)).To(Equal(taffy.RectZeroF32))
			Expect(taffy.ResolveOrZeroRectDimOptionContext(rectDimensionAuto(), taffy.Some[float32](0.0), noopCalc)).To(Equal(taffy.RectZeroF32))
		})

		It("resolve_or_zero_length", func() {
			Expect(taffy.ResolveOrZeroRectDimOptionContext(rectFromLength(5.0, 5.0, 5.0, 5.0), taffy.None[float32](), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(taffy.ResolveOrZeroRectDimOptionContext(rectFromLength(5.0, 5.0, 5.0, 5.0), taffy.Some[float32](5.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(taffy.ResolveOrZeroRectDimOptionContext(rectFromLength(5.0, 5.0, 5.0, 5.0), taffy.Some[float32](-5.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(taffy.ResolveOrZeroRectDimOptionContext(rectFromLength(5.0, 5.0, 5.0, 5.0), taffy.Some[float32](0.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
		})

		It("resolve_or_zero_percent", func() {
			Expect(taffy.ResolveOrZeroRectDimOptionContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), taffy.None[float32](), noopCalc)).To(Equal(taffy.RectZeroF32))
			Expect(taffy.ResolveOrZeroRectDimOptionContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), taffy.Some[float32](5.0), noopCalc)).To(Equal(rectNewF32(25.0, 25.0, 25.0, 25.0)))
			Expect(taffy.ResolveOrZeroRectDimOptionContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), taffy.Some[float32](-5.0), noopCalc)).To(Equal(rectNewF32(-25.0, -25.0, -25.0, -25.0)))
			Expect(taffy.ResolveOrZeroRectDimOptionContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), taffy.Some[float32](0.0), noopCalc)).To(Equal(taffy.RectZeroF32))
		})
	})
})
