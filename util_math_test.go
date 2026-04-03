// Tests for util_math.go — ported from util/math.rs #[cfg(test)] module.
package taffy_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	taffy "taffy-go"
)

var _ = Describe("MaybeMath", func() {

	// -----------------------------------------------------------------------
	// Option[float32] with Option[float32]
	// -----------------------------------------------------------------------

	Describe("lhs Option[float32] rhs Option[float32]", func() {
		It("maybe_min", func() {
			Expect(taffy.MaybeMinOption(taffy.Some[float32](3.0), taffy.Some[float32](5.0))).To(Equal(taffy.Some[float32](3.0)))
			Expect(taffy.MaybeMinOption(taffy.Some[float32](5.0), taffy.Some[float32](3.0))).To(Equal(taffy.Some[float32](3.0)))
			Expect(taffy.MaybeMinOption(taffy.Some[float32](3.0), taffy.None[float32]())).To(Equal(taffy.Some[float32](3.0)))
			Expect(taffy.MaybeMinOption(taffy.None[float32](), taffy.Some[float32](3.0))).To(Equal(taffy.None[float32]()))
			Expect(taffy.MaybeMinOption(taffy.None[float32](), taffy.None[float32]())).To(Equal(taffy.None[float32]()))
		})

		It("maybe_max", func() {
			Expect(taffy.MaybeMaxOption(taffy.Some[float32](3.0), taffy.Some[float32](5.0))).To(Equal(taffy.Some[float32](5.0)))
			Expect(taffy.MaybeMaxOption(taffy.Some[float32](5.0), taffy.Some[float32](3.0))).To(Equal(taffy.Some[float32](5.0)))
			Expect(taffy.MaybeMaxOption(taffy.Some[float32](3.0), taffy.None[float32]())).To(Equal(taffy.Some[float32](3.0)))
			Expect(taffy.MaybeMaxOption(taffy.None[float32](), taffy.Some[float32](3.0))).To(Equal(taffy.None[float32]()))
			Expect(taffy.MaybeMaxOption(taffy.None[float32](), taffy.None[float32]())).To(Equal(taffy.None[float32]()))
		})

		It("maybe_add", func() {
			Expect(taffy.MaybeAddOption(taffy.Some[float32](3.0), taffy.Some[float32](5.0))).To(Equal(taffy.Some[float32](8.0)))
			Expect(taffy.MaybeAddOption(taffy.Some[float32](5.0), taffy.Some[float32](3.0))).To(Equal(taffy.Some[float32](8.0)))
			Expect(taffy.MaybeAddOption(taffy.Some[float32](3.0), taffy.None[float32]())).To(Equal(taffy.Some[float32](3.0)))
			Expect(taffy.MaybeAddOption(taffy.None[float32](), taffy.Some[float32](3.0))).To(Equal(taffy.None[float32]()))
			Expect(taffy.MaybeAddOption(taffy.None[float32](), taffy.None[float32]())).To(Equal(taffy.None[float32]()))
		})

		It("maybe_sub", func() {
			Expect(taffy.MaybeSubOption(taffy.Some[float32](3.0), taffy.Some[float32](5.0))).To(Equal(taffy.Some[float32](-2.0)))
			Expect(taffy.MaybeSubOption(taffy.Some[float32](5.0), taffy.Some[float32](3.0))).To(Equal(taffy.Some[float32](2.0)))
			Expect(taffy.MaybeSubOption(taffy.Some[float32](3.0), taffy.None[float32]())).To(Equal(taffy.Some[float32](3.0)))
			Expect(taffy.MaybeSubOption(taffy.None[float32](), taffy.Some[float32](3.0))).To(Equal(taffy.None[float32]()))
			Expect(taffy.MaybeSubOption(taffy.None[float32](), taffy.None[float32]())).To(Equal(taffy.None[float32]()))
		})
	})

	// -----------------------------------------------------------------------
	// float32 with Option[float32]
	// -----------------------------------------------------------------------

	Describe("lhs float32 rhs Option[float32]", func() {
		It("maybe_min", func() {
			Expect(taffy.MaybeMinF32(3.0, taffy.Some[float32](5.0))).To(Equal(float32(3.0)))
			Expect(taffy.MaybeMinF32(5.0, taffy.Some[float32](3.0))).To(Equal(float32(3.0)))
			Expect(taffy.MaybeMinF32(3.0, taffy.None[float32]())).To(Equal(float32(3.0)))
		})

		It("maybe_max", func() {
			Expect(taffy.MaybeMaxF32(3.0, taffy.Some[float32](5.0))).To(Equal(float32(5.0)))
			Expect(taffy.MaybeMaxF32(5.0, taffy.Some[float32](3.0))).To(Equal(float32(5.0)))
			Expect(taffy.MaybeMaxF32(3.0, taffy.None[float32]())).To(Equal(float32(3.0)))
		})

		It("maybe_add", func() {
			Expect(taffy.MaybeAddF32(3.0, taffy.Some[float32](5.0))).To(Equal(float32(8.0)))
			Expect(taffy.MaybeAddF32(5.0, taffy.Some[float32](3.0))).To(Equal(float32(8.0)))
			Expect(taffy.MaybeAddF32(3.0, taffy.None[float32]())).To(Equal(float32(3.0)))
		})

		It("maybe_sub", func() {
			Expect(taffy.MaybeSubF32(3.0, taffy.Some[float32](5.0))).To(Equal(float32(-2.0)))
			Expect(taffy.MaybeSubF32(5.0, taffy.Some[float32](3.0))).To(Equal(float32(2.0)))
			Expect(taffy.MaybeSubF32(3.0, taffy.None[float32]())).To(Equal(float32(3.0)))
		})
	})
})
