// Tests for util_math.go — ported from util/math.rs #[cfg(test)] module.
package taffy

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"taffy-go/geometry"
)

var _ = Describe("MaybeMath", func() {

	// -----------------------------------------------------------------------
	// geometry.Option[float32] with geometry.Option[float32]
	// -----------------------------------------------------------------------

	Describe("lhs geometry.Option[float32] rhs geometry.Option[float32]", func() {
		It("maybe_min", func() {
			Expect(geometry.MaybeMinOption(geometry.Some[float32](3.0), geometry.Some[float32](5.0))).To(Equal(geometry.Some[float32](3.0)))
			Expect(geometry.MaybeMinOption(geometry.Some[float32](5.0), geometry.Some[float32](3.0))).To(Equal(geometry.Some[float32](3.0)))
			Expect(geometry.MaybeMinOption(geometry.Some[float32](3.0), geometry.None[float32]())).To(Equal(geometry.Some[float32](3.0)))
			Expect(geometry.MaybeMinOption(geometry.None[float32](), geometry.Some[float32](3.0))).To(Equal(geometry.None[float32]()))
			Expect(geometry.MaybeMinOption(geometry.None[float32](), geometry.None[float32]())).To(Equal(geometry.None[float32]()))
		})

		It("maybe_max", func() {
			Expect(geometry.MaybeMaxOption(geometry.Some[float32](3.0), geometry.Some[float32](5.0))).To(Equal(geometry.Some[float32](5.0)))
			Expect(geometry.MaybeMaxOption(geometry.Some[float32](5.0), geometry.Some[float32](3.0))).To(Equal(geometry.Some[float32](5.0)))
			Expect(geometry.MaybeMaxOption(geometry.Some[float32](3.0), geometry.None[float32]())).To(Equal(geometry.Some[float32](3.0)))
			Expect(geometry.MaybeMaxOption(geometry.None[float32](), geometry.Some[float32](3.0))).To(Equal(geometry.None[float32]()))
			Expect(geometry.MaybeMaxOption(geometry.None[float32](), geometry.None[float32]())).To(Equal(geometry.None[float32]()))
		})

		It("maybe_add", func() {
			Expect(geometry.MaybeAddOption(geometry.Some[float32](3.0), geometry.Some[float32](5.0))).To(Equal(geometry.Some[float32](8.0)))
			Expect(geometry.MaybeAddOption(geometry.Some[float32](5.0), geometry.Some[float32](3.0))).To(Equal(geometry.Some[float32](8.0)))
			Expect(geometry.MaybeAddOption(geometry.Some[float32](3.0), geometry.None[float32]())).To(Equal(geometry.Some[float32](3.0)))
			Expect(geometry.MaybeAddOption(geometry.None[float32](), geometry.Some[float32](3.0))).To(Equal(geometry.None[float32]()))
			Expect(geometry.MaybeAddOption(geometry.None[float32](), geometry.None[float32]())).To(Equal(geometry.None[float32]()))
		})

		It("maybe_sub", func() {
			Expect(geometry.MaybeSubOption(geometry.Some[float32](3.0), geometry.Some[float32](5.0))).To(Equal(geometry.Some[float32](-2.0)))
			Expect(geometry.MaybeSubOption(geometry.Some[float32](5.0), geometry.Some[float32](3.0))).To(Equal(geometry.Some[float32](2.0)))
			Expect(geometry.MaybeSubOption(geometry.Some[float32](3.0), geometry.None[float32]())).To(Equal(geometry.Some[float32](3.0)))
			Expect(geometry.MaybeSubOption(geometry.None[float32](), geometry.Some[float32](3.0))).To(Equal(geometry.None[float32]()))
			Expect(geometry.MaybeSubOption(geometry.None[float32](), geometry.None[float32]())).To(Equal(geometry.None[float32]()))
		})
	})

	// -----------------------------------------------------------------------
	// float32 with geometry.Option[float32]
	// -----------------------------------------------------------------------

	Describe("lhs float32 rhs geometry.Option[float32]", func() {
		It("maybe_min", func() {
			Expect(geometry.MaybeMinF32(3.0, geometry.Some[float32](5.0))).To(Equal(float32(3.0)))
			Expect(geometry.MaybeMinF32(5.0, geometry.Some[float32](3.0))).To(Equal(float32(3.0)))
			Expect(geometry.MaybeMinF32(3.0, geometry.None[float32]())).To(Equal(float32(3.0)))
		})

		It("maybe_max", func() {
			Expect(geometry.MaybeMaxF32(3.0, geometry.Some[float32](5.0))).To(Equal(float32(5.0)))
			Expect(geometry.MaybeMaxF32(5.0, geometry.Some[float32](3.0))).To(Equal(float32(5.0)))
			Expect(geometry.MaybeMaxF32(3.0, geometry.None[float32]())).To(Equal(float32(3.0)))
		})

		It("maybe_add", func() {
			Expect(geometry.MaybeAddF32(3.0, geometry.Some[float32](5.0))).To(Equal(float32(8.0)))
			Expect(geometry.MaybeAddF32(5.0, geometry.Some[float32](3.0))).To(Equal(float32(8.0)))
			Expect(geometry.MaybeAddF32(3.0, geometry.None[float32]())).To(Equal(float32(3.0)))
		})

		It("maybe_sub", func() {
			Expect(geometry.MaybeSubF32(3.0, geometry.Some[float32](5.0))).To(Equal(float32(-2.0)))
			Expect(geometry.MaybeSubF32(5.0, geometry.Some[float32](3.0))).To(Equal(float32(2.0)))
			Expect(geometry.MaybeSubF32(3.0, geometry.None[float32]())).To(Equal(float32(3.0)))
		})
	})
})
