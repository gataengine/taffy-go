// Tests for util_math.go — ported from util/math.rs #[cfg(test)] module.
package taffy

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MaybeMath", func() {

	// -----------------------------------------------------------------------
	// Option[float32] with Option[float32]
	// -----------------------------------------------------------------------

	Describe("lhs Option[float32] rhs Option[float32]", func() {
		It("maybe_min", func() {
			Expect(MaybeMinOption(Some[float32](3.0), Some[float32](5.0))).To(Equal(Some[float32](3.0)))
			Expect(MaybeMinOption(Some[float32](5.0), Some[float32](3.0))).To(Equal(Some[float32](3.0)))
			Expect(MaybeMinOption(Some[float32](3.0), None[float32]())).To(Equal(Some[float32](3.0)))
			Expect(MaybeMinOption(None[float32](), Some[float32](3.0))).To(Equal(None[float32]()))
			Expect(MaybeMinOption(None[float32](), None[float32]())).To(Equal(None[float32]()))
		})

		It("maybe_max", func() {
			Expect(MaybeMaxOption(Some[float32](3.0), Some[float32](5.0))).To(Equal(Some[float32](5.0)))
			Expect(MaybeMaxOption(Some[float32](5.0), Some[float32](3.0))).To(Equal(Some[float32](5.0)))
			Expect(MaybeMaxOption(Some[float32](3.0), None[float32]())).To(Equal(Some[float32](3.0)))
			Expect(MaybeMaxOption(None[float32](), Some[float32](3.0))).To(Equal(None[float32]()))
			Expect(MaybeMaxOption(None[float32](), None[float32]())).To(Equal(None[float32]()))
		})

		It("maybe_add", func() {
			Expect(MaybeAddOption(Some[float32](3.0), Some[float32](5.0))).To(Equal(Some[float32](8.0)))
			Expect(MaybeAddOption(Some[float32](5.0), Some[float32](3.0))).To(Equal(Some[float32](8.0)))
			Expect(MaybeAddOption(Some[float32](3.0), None[float32]())).To(Equal(Some[float32](3.0)))
			Expect(MaybeAddOption(None[float32](), Some[float32](3.0))).To(Equal(None[float32]()))
			Expect(MaybeAddOption(None[float32](), None[float32]())).To(Equal(None[float32]()))
		})

		It("maybe_sub", func() {
			Expect(MaybeSubOption(Some[float32](3.0), Some[float32](5.0))).To(Equal(Some[float32](-2.0)))
			Expect(MaybeSubOption(Some[float32](5.0), Some[float32](3.0))).To(Equal(Some[float32](2.0)))
			Expect(MaybeSubOption(Some[float32](3.0), None[float32]())).To(Equal(Some[float32](3.0)))
			Expect(MaybeSubOption(None[float32](), Some[float32](3.0))).To(Equal(None[float32]()))
			Expect(MaybeSubOption(None[float32](), None[float32]())).To(Equal(None[float32]()))
		})
	})

	// -----------------------------------------------------------------------
	// float32 with Option[float32]
	// -----------------------------------------------------------------------

	Describe("lhs float32 rhs Option[float32]", func() {
		It("maybe_min", func() {
			Expect(MaybeMinF32(3.0, Some[float32](5.0))).To(Equal(float32(3.0)))
			Expect(MaybeMinF32(5.0, Some[float32](3.0))).To(Equal(float32(3.0)))
			Expect(MaybeMinF32(3.0, None[float32]())).To(Equal(float32(3.0)))
		})

		It("maybe_max", func() {
			Expect(MaybeMaxF32(3.0, Some[float32](5.0))).To(Equal(float32(5.0)))
			Expect(MaybeMaxF32(5.0, Some[float32](3.0))).To(Equal(float32(5.0)))
			Expect(MaybeMaxF32(3.0, None[float32]())).To(Equal(float32(3.0)))
		})

		It("maybe_add", func() {
			Expect(MaybeAddF32(3.0, Some[float32](5.0))).To(Equal(float32(8.0)))
			Expect(MaybeAddF32(5.0, Some[float32](3.0))).To(Equal(float32(8.0)))
			Expect(MaybeAddF32(3.0, None[float32]())).To(Equal(float32(3.0)))
		})

		It("maybe_sub", func() {
			Expect(MaybeSubF32(3.0, Some[float32](5.0))).To(Equal(float32(-2.0)))
			Expect(MaybeSubF32(5.0, Some[float32](3.0))).To(Equal(float32(2.0)))
			Expect(MaybeSubF32(3.0, None[float32]())).To(Equal(float32(3.0)))
		})
	})
})
