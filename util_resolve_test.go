// Tests for util_resolve.go — ported from util/resolve.rs #[cfg(test)] module.
package taffy

import (
	"github.com/gataengine/taffy-go/lib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/gataengine/taffy-go/geometry"
	"github.com/gataengine/taffy-go/style"
)

// noopCalc is a style.CalcResolver that always returns 42.42, matching the Rust test harness.
var noopCalc style.CalcResolver = func(_ uint64, _ float32) float32 { return 42.42 }

// Helper constructors matching the Rust test helpers.

func sizeDimensionAuto() geometry.Size[style.Dimension] {
	return geometry.Size[style.Dimension]{
		Width:  style.DimensionAutoValue,
		Height: style.DimensionAutoValue,
	}
}

func sizeFromLengths(w, h float32) geometry.Size[style.Dimension] {
	return geometry.Size[style.Dimension]{
		Width:  style.DimensionFromLength(w),
		Height: style.DimensionFromLength(h),
	}
}

func sizeFromPercent(w, h float32) geometry.Size[style.Dimension] {
	return geometry.Size[style.Dimension]{
		Width:  style.DimensionFromPercent(w),
		Height: style.DimensionFromPercent(h),
	}
}

func sizeOptionNew(w, h float32) geometry.Size[lib.Option[float32]] {
	return geometry.Size[lib.Option[float32]]{
		Width:  lib.Some(w),
		Height: lib.Some(h),
	}
}

func rectDimensionAuto() geometry.Rect[style.Dimension] {
	return geometry.Rect[style.Dimension]{
		Left:   style.DimensionAutoValue,
		Right:  style.DimensionAutoValue,
		Top:    style.DimensionAutoValue,
		Bottom: style.DimensionAutoValue,
	}
}

func rectFromLength(l, r, t, b float32) geometry.Rect[style.Dimension] {
	return geometry.Rect[style.Dimension]{
		Left:   style.DimensionFromLength(l),
		Right:  style.DimensionFromLength(r),
		Top:    style.DimensionFromLength(t),
		Bottom: style.DimensionFromLength(b),
	}
}

func rectFromPercent(l, r, t, b float32) geometry.Rect[style.Dimension] {
	return geometry.Rect[style.Dimension]{
		Left:   style.DimensionFromPercent(l),
		Right:  style.DimensionFromPercent(r),
		Top:    style.DimensionFromPercent(t),
		Bottom: style.DimensionFromPercent(b),
	}
}

func rectNewF32(l, r, t, b float32) geometry.Rect[float32] {
	return geometry.NewRectF32(l, r, t, b)
}

var _ = Describe("MaybeResolve", func() {

	// -----------------------------------------------------------------------
	// maybe_resolve_dimension
	// -----------------------------------------------------------------------

	Describe("maybe_resolve_dimension", func() {
		It("resolve_auto", func() {
			Expect(style.MaybeResolveDimOption(style.DimensionAutoValue, lib.None[float32](), noopCalc)).To(Equal(lib.None[float32]()))
			Expect(style.MaybeResolveDimOption(style.DimensionAutoValue, lib.Some[float32](5.0), noopCalc)).To(Equal(lib.None[float32]()))
			Expect(style.MaybeResolveDimOption(style.DimensionAutoValue, lib.Some[float32](-5.0), noopCalc)).To(Equal(lib.None[float32]()))
			Expect(style.MaybeResolveDimOption(style.DimensionAutoValue, lib.Some[float32](0.0), noopCalc)).To(Equal(lib.None[float32]()))
		})

		It("resolve_length", func() {
			Expect(style.MaybeResolveDimOption(style.DimensionFromLength(1.0), lib.None[float32](), noopCalc)).To(Equal(lib.Some[float32](1.0)))
			Expect(style.MaybeResolveDimOption(style.DimensionFromLength(1.0), lib.Some[float32](5.0), noopCalc)).To(Equal(lib.Some[float32](1.0)))
			Expect(style.MaybeResolveDimOption(style.DimensionFromLength(1.0), lib.Some[float32](-5.0), noopCalc)).To(Equal(lib.Some[float32](1.0)))
			Expect(style.MaybeResolveDimOption(style.DimensionFromLength(1.0), lib.Some[float32](0.0), noopCalc)).To(Equal(lib.Some[float32](1.0)))
		})

		It("resolve_percent", func() {
			Expect(style.MaybeResolveDimOption(style.DimensionFromPercent(1.0), lib.None[float32](), noopCalc)).To(Equal(lib.None[float32]()))
			Expect(style.MaybeResolveDimOption(style.DimensionFromPercent(1.0), lib.Some[float32](5.0), noopCalc)).To(Equal(lib.Some[float32](5.0)))
			Expect(style.MaybeResolveDimOption(style.DimensionFromPercent(1.0), lib.Some[float32](-5.0), noopCalc)).To(Equal(lib.Some[float32](-5.0)))
			Expect(style.MaybeResolveDimOption(style.DimensionFromPercent(1.0), lib.Some[float32](50.0), noopCalc)).To(Equal(lib.Some[float32](50.0)))
		})
	})

	// -----------------------------------------------------------------------
	// maybe_resolve_size_dimension
	// -----------------------------------------------------------------------

	Describe("maybe_resolve_size_dimension", func() {
		It("maybe_resolve_auto", func() {
			Expect(style.MaybeResolveSizeDimOption(sizeDimensionAuto(), geometry.SizeOptionNone, noopCalc)).To(Equal(geometry.SizeOptionNone))
			Expect(style.MaybeResolveSizeDimOption(sizeDimensionAuto(), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(geometry.SizeOptionNone))
			Expect(style.MaybeResolveSizeDimOption(sizeDimensionAuto(), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(geometry.SizeOptionNone))
			Expect(style.MaybeResolveSizeDimOption(sizeDimensionAuto(), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(geometry.SizeOptionNone))
		})

		It("maybe_resolve_length", func() {
			Expect(style.MaybeResolveSizeDimOption(sizeFromLengths(5.0, 5.0), geometry.SizeOptionNone, noopCalc)).To(Equal(sizeOptionNew(5.0, 5.0)))
			Expect(style.MaybeResolveSizeDimOption(sizeFromLengths(5.0, 5.0), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(sizeOptionNew(5.0, 5.0)))
			Expect(style.MaybeResolveSizeDimOption(sizeFromLengths(5.0, 5.0), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(sizeOptionNew(5.0, 5.0)))
			Expect(style.MaybeResolveSizeDimOption(sizeFromLengths(5.0, 5.0), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(sizeOptionNew(5.0, 5.0)))
		})

		It("maybe_resolve_percent", func() {
			Expect(style.MaybeResolveSizeDimOption(sizeFromPercent(5.0, 5.0), geometry.SizeOptionNone, noopCalc)).To(Equal(geometry.SizeOptionNone))
			Expect(style.MaybeResolveSizeDimOption(sizeFromPercent(5.0, 5.0), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(sizeOptionNew(25.0, 25.0)))
			Expect(style.MaybeResolveSizeDimOption(sizeFromPercent(5.0, 5.0), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(sizeOptionNew(-25.0, -25.0)))
			Expect(style.MaybeResolveSizeDimOption(sizeFromPercent(5.0, 5.0), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(sizeOptionNew(0.0, 0.0)))
		})
	})
})

var _ = Describe("ResolveOrZero", func() {

	// -----------------------------------------------------------------------
	// resolve_or_zero_dimension_to_option_f32
	// -----------------------------------------------------------------------

	Describe("resolve_or_zero_dimension", func() {
		It("resolve_or_zero_auto", func() {
			Expect(style.ResolveOrZeroDim(style.DimensionAutoValue, lib.None[float32](), noopCalc)).To(Equal(float32(0.0)))
			Expect(style.ResolveOrZeroDim(style.DimensionAutoValue, lib.Some[float32](5.0), noopCalc)).To(Equal(float32(0.0)))
			Expect(style.ResolveOrZeroDim(style.DimensionAutoValue, lib.Some[float32](-5.0), noopCalc)).To(Equal(float32(0.0)))
			Expect(style.ResolveOrZeroDim(style.DimensionAutoValue, lib.Some[float32](0.0), noopCalc)).To(Equal(float32(0.0)))
		})

		It("resolve_or_zero_length", func() {
			Expect(style.ResolveOrZeroDim(style.DimensionFromLength(5.0), lib.None[float32](), noopCalc)).To(Equal(float32(5.0)))
			Expect(style.ResolveOrZeroDim(style.DimensionFromLength(5.0), lib.Some[float32](5.0), noopCalc)).To(Equal(float32(5.0)))
			Expect(style.ResolveOrZeroDim(style.DimensionFromLength(5.0), lib.Some[float32](-5.0), noopCalc)).To(Equal(float32(5.0)))
			Expect(style.ResolveOrZeroDim(style.DimensionFromLength(5.0), lib.Some[float32](0.0), noopCalc)).To(Equal(float32(5.0)))
		})

		It("resolve_or_zero_percent", func() {
			Expect(style.ResolveOrZeroDim(style.DimensionFromPercent(5.0), lib.None[float32](), noopCalc)).To(Equal(float32(0.0)))
			Expect(style.ResolveOrZeroDim(style.DimensionFromPercent(5.0), lib.Some[float32](5.0), noopCalc)).To(Equal(float32(25.0)))
			Expect(style.ResolveOrZeroDim(style.DimensionFromPercent(5.0), lib.Some[float32](-5.0), noopCalc)).To(Equal(float32(-25.0)))
			Expect(style.ResolveOrZeroDim(style.DimensionFromPercent(5.0), lib.Some[float32](0.0), noopCalc)).To(Equal(float32(0.0)))
		})
	})

	// -----------------------------------------------------------------------
	// resolve_or_zero_rect_dimension_to_rect (Size context)
	// -----------------------------------------------------------------------

	Describe("resolve_or_zero_rect_dimension_size_context", func() {
		It("resolve_or_zero_auto", func() {
			Expect(style.ResolveOrZeroRectDimSizeContext(rectDimensionAuto(), geometry.SizeOptionNone, noopCalc)).To(Equal(geometry.RectZeroF32))
			Expect(style.ResolveOrZeroRectDimSizeContext(rectDimensionAuto(), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(geometry.RectZeroF32))
			Expect(style.ResolveOrZeroRectDimSizeContext(rectDimensionAuto(), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(geometry.RectZeroF32))
			Expect(style.ResolveOrZeroRectDimSizeContext(rectDimensionAuto(), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(geometry.RectZeroF32))
		})

		It("resolve_or_zero_length", func() {
			Expect(style.ResolveOrZeroRectDimSizeContext(rectFromLength(5.0, 5.0, 5.0, 5.0), geometry.SizeOptionNone, noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(style.ResolveOrZeroRectDimSizeContext(rectFromLength(5.0, 5.0, 5.0, 5.0), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(style.ResolveOrZeroRectDimSizeContext(rectFromLength(5.0, 5.0, 5.0, 5.0), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(style.ResolveOrZeroRectDimSizeContext(rectFromLength(5.0, 5.0, 5.0, 5.0), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
		})

		It("resolve_or_zero_percent", func() {
			Expect(style.ResolveOrZeroRectDimSizeContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), geometry.SizeOptionNone, noopCalc)).To(Equal(geometry.RectZeroF32))
			Expect(style.ResolveOrZeroRectDimSizeContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), sizeOptionNew(5.0, 5.0), noopCalc)).To(Equal(rectNewF32(25.0, 25.0, 25.0, 25.0)))
			Expect(style.ResolveOrZeroRectDimSizeContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), sizeOptionNew(-5.0, -5.0), noopCalc)).To(Equal(rectNewF32(-25.0, -25.0, -25.0, -25.0)))
			Expect(style.ResolveOrZeroRectDimSizeContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), sizeOptionNew(0.0, 0.0), noopCalc)).To(Equal(geometry.RectZeroF32))
		})
	})

	// -----------------------------------------------------------------------
	// resolve_or_zero_rect_dimension_to_rect_f32_via_option (lib.Option[float32] context)
	// -----------------------------------------------------------------------

	Describe("resolve_or_zero_rect_dimension_option_context", func() {
		It("resolve_or_zero_auto", func() {
			Expect(style.ResolveOrZeroRectDimOptionContext(rectDimensionAuto(), lib.None[float32](), noopCalc)).To(Equal(geometry.RectZeroF32))
			Expect(style.ResolveOrZeroRectDimOptionContext(rectDimensionAuto(), lib.Some[float32](5.0), noopCalc)).To(Equal(geometry.RectZeroF32))
			Expect(style.ResolveOrZeroRectDimOptionContext(rectDimensionAuto(), lib.Some[float32](-5.0), noopCalc)).To(Equal(geometry.RectZeroF32))
			Expect(style.ResolveOrZeroRectDimOptionContext(rectDimensionAuto(), lib.Some[float32](0.0), noopCalc)).To(Equal(geometry.RectZeroF32))
		})

		It("resolve_or_zero_length", func() {
			Expect(style.ResolveOrZeroRectDimOptionContext(rectFromLength(5.0, 5.0, 5.0, 5.0), lib.None[float32](), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(style.ResolveOrZeroRectDimOptionContext(rectFromLength(5.0, 5.0, 5.0, 5.0), lib.Some[float32](5.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(style.ResolveOrZeroRectDimOptionContext(rectFromLength(5.0, 5.0, 5.0, 5.0), lib.Some[float32](-5.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
			Expect(style.ResolveOrZeroRectDimOptionContext(rectFromLength(5.0, 5.0, 5.0, 5.0), lib.Some[float32](0.0), noopCalc)).To(Equal(rectNewF32(5.0, 5.0, 5.0, 5.0)))
		})

		It("resolve_or_zero_percent", func() {
			Expect(style.ResolveOrZeroRectDimOptionContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), lib.None[float32](), noopCalc)).To(Equal(geometry.RectZeroF32))
			Expect(style.ResolveOrZeroRectDimOptionContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), lib.Some[float32](5.0), noopCalc)).To(Equal(rectNewF32(25.0, 25.0, 25.0, 25.0)))
			Expect(style.ResolveOrZeroRectDimOptionContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), lib.Some[float32](-5.0), noopCalc)).To(Equal(rectNewF32(-25.0, -25.0, -25.0, -25.0)))
			Expect(style.ResolveOrZeroRectDimOptionContext(rectFromPercent(5.0, 5.0, 5.0, 5.0), lib.Some[float32](0.0), noopCalc)).To(Equal(geometry.RectZeroF32))
		})
	})
})
