// Tests for util_math.go — ported from util/math.rs #[cfg(test)] module.
package taffy

import (
	"github.com/gataengine/taffy-go/lib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/gataengine/taffy-go/geometry"
)

var _ = Describe("MaybeMath", func() {

	// -----------------------------------------------------------------------
	// lib.Option[float32] with lib.Option[float32]
	// -----------------------------------------------------------------------

	Describe("lhs lib.Option[float32] rhs lib.Option[float32]", func() {
		It("maybe_min", func() {
			Expect(geometry.MaybeMinOption(lib.Some[float32](3.0), lib.Some[float32](5.0))).To(Equal(lib.Some[float32](3.0)))
			Expect(geometry.MaybeMinOption(lib.Some[float32](5.0), lib.Some[float32](3.0))).To(Equal(lib.Some[float32](3.0)))
			Expect(geometry.MaybeMinOption(lib.Some[float32](3.0), lib.None[float32]())).To(Equal(lib.Some[float32](3.0)))
			Expect(geometry.MaybeMinOption(lib.None[float32](), lib.Some[float32](3.0))).To(Equal(lib.None[float32]()))
			Expect(geometry.MaybeMinOption(lib.None[float32](), lib.None[float32]())).To(Equal(lib.None[float32]()))
		})

		It("maybe_max", func() {
			Expect(geometry.MaybeMaxOption(lib.Some[float32](3.0), lib.Some[float32](5.0))).To(Equal(lib.Some[float32](5.0)))
			Expect(geometry.MaybeMaxOption(lib.Some[float32](5.0), lib.Some[float32](3.0))).To(Equal(lib.Some[float32](5.0)))
			Expect(geometry.MaybeMaxOption(lib.Some[float32](3.0), lib.None[float32]())).To(Equal(lib.Some[float32](3.0)))
			Expect(geometry.MaybeMaxOption(lib.None[float32](), lib.Some[float32](3.0))).To(Equal(lib.None[float32]()))
			Expect(geometry.MaybeMaxOption(lib.None[float32](), lib.None[float32]())).To(Equal(lib.None[float32]()))
		})

		It("maybe_add", func() {
			Expect(geometry.MaybeAddOption(lib.Some[float32](3.0), lib.Some[float32](5.0))).To(Equal(lib.Some[float32](8.0)))
			Expect(geometry.MaybeAddOption(lib.Some[float32](5.0), lib.Some[float32](3.0))).To(Equal(lib.Some[float32](8.0)))
			Expect(geometry.MaybeAddOption(lib.Some[float32](3.0), lib.None[float32]())).To(Equal(lib.Some[float32](3.0)))
			Expect(geometry.MaybeAddOption(lib.None[float32](), lib.Some[float32](3.0))).To(Equal(lib.None[float32]()))
			Expect(geometry.MaybeAddOption(lib.None[float32](), lib.None[float32]())).To(Equal(lib.None[float32]()))
		})

		It("maybe_sub", func() {
			Expect(geometry.MaybeSubOption(lib.Some[float32](3.0), lib.Some[float32](5.0))).To(Equal(lib.Some[float32](-2.0)))
			Expect(geometry.MaybeSubOption(lib.Some[float32](5.0), lib.Some[float32](3.0))).To(Equal(lib.Some[float32](2.0)))
			Expect(geometry.MaybeSubOption(lib.Some[float32](3.0), lib.None[float32]())).To(Equal(lib.Some[float32](3.0)))
			Expect(geometry.MaybeSubOption(lib.None[float32](), lib.Some[float32](3.0))).To(Equal(lib.None[float32]()))
			Expect(geometry.MaybeSubOption(lib.None[float32](), lib.None[float32]())).To(Equal(lib.None[float32]()))
		})
	})

	// -----------------------------------------------------------------------
	// float32 with lib.Option[float32]
	// -----------------------------------------------------------------------

	Describe("lhs float32 rhs lib.Option[float32]", func() {
		It("maybe_min", func() {
			Expect(geometry.MaybeMinF32(3.0, lib.Some[float32](5.0))).To(Equal(float32(3.0)))
			Expect(geometry.MaybeMinF32(5.0, lib.Some[float32](3.0))).To(Equal(float32(3.0)))
			Expect(geometry.MaybeMinF32(3.0, lib.None[float32]())).To(Equal(float32(3.0)))
		})

		It("maybe_max", func() {
			Expect(geometry.MaybeMaxF32(3.0, lib.Some[float32](5.0))).To(Equal(float32(5.0)))
			Expect(geometry.MaybeMaxF32(5.0, lib.Some[float32](3.0))).To(Equal(float32(5.0)))
			Expect(geometry.MaybeMaxF32(3.0, lib.None[float32]())).To(Equal(float32(3.0)))
		})

		It("maybe_add", func() {
			Expect(geometry.MaybeAddF32(3.0, lib.Some[float32](5.0))).To(Equal(float32(8.0)))
			Expect(geometry.MaybeAddF32(5.0, lib.Some[float32](3.0))).To(Equal(float32(8.0)))
			Expect(geometry.MaybeAddF32(3.0, lib.None[float32]())).To(Equal(float32(3.0)))
		})

		It("maybe_sub", func() {
			Expect(geometry.MaybeSubF32(3.0, lib.Some[float32](5.0))).To(Equal(float32(-2.0)))
			Expect(geometry.MaybeSubF32(5.0, lib.Some[float32](3.0))).To(Equal(float32(2.0)))
			Expect(geometry.MaybeSubF32(3.0, lib.None[float32]())).To(Equal(float32(3.0)))
		})
	})
})
