package taffy

import (
	"github.com/gataengine/taffy-go/geometry"
	"github.com/gataengine/taffy-go/lib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Option", func() {
	It("Some holds a value", func() {
		o := lib.Some(42)
		Expect(o.IsSome).To(BeTrue())
		Expect(o.Value).To(Equal(42))
	})

	It("None holds no value", func() {
		o := lib.None[int]()
		Expect(o.IsSome).To(BeFalse())
	})

	It("UnwrapOr returns value when Some", func() {
		Expect(lib.Some(7).UnwrapOr(0)).To(Equal(7))
	})

	It("UnwrapOr returns alt when None", func() {
		Expect(lib.None[int]().UnwrapOr(99)).To(Equal(99))
	})

	It("Or returns self when Some", func() {
		a := lib.Some(1)
		b := lib.Some(2)
		Expect(a.Or(b)).To(Equal(a))
	})

	It("Or returns alt when None", func() {
		a := lib.None[int]()
		b := lib.Some(2)
		Expect(a.Or(b)).To(Equal(b))
	})
})

var _ = Describe("AbsoluteAxis", func() {
	It("OtherAxis of Horizontal is Vertical", func() {
		Expect(geometry.AbsoluteAxisHorizontal.OtherAxis()).To(Equal(geometry.AbsoluteAxisVertical))
	})

	It("OtherAxis of Vertical is Horizontal", func() {
		Expect(geometry.AbsoluteAxisVertical.OtherAxis()).To(Equal(geometry.AbsoluteAxisHorizontal))
	})
})

var _ = Describe("AbstractAxis", func() {
	It("Other of Inline is Block", func() {
		Expect(geometry.AbstractAxisInline.Other()).To(Equal(geometry.AbstractAxisBlock))
	})

	It("Other of Block is Inline", func() {
		Expect(geometry.AbstractAxisBlock.Other()).To(Equal(geometry.AbstractAxisInline))
	})

	It("AsAbsNaive converts Inline to Horizontal", func() {
		Expect(geometry.AbstractAxisInline.AsAbsNaive()).To(Equal(geometry.AbsoluteAxisHorizontal))
	})

	It("AsAbsNaive converts Block to Vertical", func() {
		Expect(geometry.AbstractAxisBlock.AsAbsNaive()).To(Equal(geometry.AbsoluteAxisVertical))
	})
})

var _ = Describe("Rect", func() {
	It("HorizontalComponents returns left and right as Line", func() {
		r := geometry.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		l := r.HorizontalComponents()
		Expect(l.Start).To(Equal(float32(1)))
		Expect(l.End).To(Equal(float32(2)))
	})

	It("VerticalComponents returns top and bottom as Line", func() {
		r := geometry.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		l := r.VerticalComponents()
		Expect(l.Start).To(Equal(float32(3)))
		Expect(l.End).To(Equal(float32(4)))
	})

	It("Map transforms all four sides", func() {
		r := geometry.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		r2 := r.Map(func(v float32) float32 { return v * 2 })
		Expect(r2.Left).To(Equal(float32(2)))
		Expect(r2.Right).To(Equal(float32(4)))
		Expect(r2.Top).To(Equal(float32(6)))
		Expect(r2.Bottom).To(Equal(float32(8)))
	})

	It("geometry.RectMapTo transforms to a different type", func() {
		r := geometry.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		r2 := geometry.RectMapTo(r, func(v float32) int { return int(v) })
		Expect(r2.Left).To(Equal(1))
		Expect(r2.Right).To(Equal(2))
		Expect(r2.Top).To(Equal(3))
		Expect(r2.Bottom).To(Equal(4))
	})

	It("geometry.RectHorizontalAxisSum returns left+right", func() {
		r := geometry.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		Expect(geometry.RectHorizontalAxisSum(r)).To(Equal(float32(3)))
	})

	It("geometry.RectVerticalAxisSum returns top+bottom", func() {
		r := geometry.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		Expect(geometry.RectVerticalAxisSum(r)).To(Equal(float32(7)))
	})

	It("geometry.RectSumAxes returns both axis sums", func() {
		r := geometry.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		s := geometry.RectSumAxes(r)
		Expect(s.Width).To(Equal(float32(3)))
		Expect(s.Height).To(Equal(float32(7)))
	})

	It("geometry.RectGridAxisSum returns correct sum for each axis", func() {
		r := geometry.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		Expect(geometry.RectGridAxisSum(r, geometry.AbsoluteAxisHorizontal)).To(Equal(float32(3)))
		Expect(geometry.RectGridAxisSum(r, geometry.AbsoluteAxisVertical)).To(Equal(float32(7)))
	})

	It("geometry.RectAddF32 adds component-wise", func() {
		a := geometry.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		b := geometry.Rect[float32]{Left: 10, Right: 20, Top: 30, Bottom: 40}
		c := geometry.RectAddF32(a, b)
		Expect(c.Left).To(Equal(float32(11)))
		Expect(c.Right).To(Equal(float32(22)))
		Expect(c.Top).To(Equal(float32(33)))
		Expect(c.Bottom).To(Equal(float32(44)))
	})

	It("geometry.RectZipSize applies f with corresponding axis sizes", func() {
		r := geometry.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		s := geometry.Size[float32]{Width: 10, Height: 100}
		r2 := geometry.RectZipSize(r, s, func(a, b float32) float32 { return a + b })
		Expect(r2.Left).To(Equal(float32(11)))
		Expect(r2.Right).To(Equal(float32(12)))
		Expect(r2.Top).To(Equal(float32(103)))
		Expect(r2.Bottom).To(Equal(float32(104)))
	})
})

var _ = Describe("Line", func() {
	It("Map transforms both fields", func() {
		l := geometry.Line[float32]{Start: 1, End: 2}
		l2 := l.Map(func(v float32) float32 { return v + 10 })
		Expect(l2.Start).To(Equal(float32(11)))
		Expect(l2.End).To(Equal(float32(12)))
	})

	It("geometry.LineMapTo transforms to a different type", func() {
		l := geometry.Line[float32]{Start: 1.5, End: 2.5}
		l2 := geometry.LineMapTo(l, func(v float32) int { return int(v) })
		Expect(l2.Start).To(Equal(1))
		Expect(l2.End).To(Equal(2))
	})

	It("geometry.LineSumF32 adds start and end", func() {
		l := geometry.Line[float32]{Start: 3, End: 7}
		Expect(geometry.LineSumF32(l)).To(Equal(float32(10)))
	})

	It("geometry.LineTrueBool has both fields true", func() {
		Expect(geometry.LineTrueBool.Start).To(BeTrue())
		Expect(geometry.LineTrueBool.End).To(BeTrue())
	})

	It("geometry.LineFalseBool has both fields false", func() {
		Expect(geometry.LineFalseBool.Start).To(BeFalse())
		Expect(geometry.LineFalseBool.End).To(BeFalse())
	})
})

var _ = Describe("Size", func() {
	It("GetAbs returns Width for Horizontal", func() {
		s := geometry.Size[float32]{Width: 10, Height: 20}
		Expect(s.GetAbs(geometry.AbsoluteAxisHorizontal)).To(Equal(float32(10)))
	})

	It("GetAbs returns Height for Vertical", func() {
		s := geometry.Size[float32]{Width: 10, Height: 20}
		Expect(s.GetAbs(geometry.AbsoluteAxisVertical)).To(Equal(float32(20)))
	})

	It("GetAbstract returns Width for Inline", func() {
		s := geometry.Size[float32]{Width: 10, Height: 20}
		Expect(s.GetAbstract(geometry.AbstractAxisInline)).To(Equal(float32(10)))
	})

	It("GetAbstract returns Height for Block", func() {
		s := geometry.Size[float32]{Width: 10, Height: 20}
		Expect(s.GetAbstract(geometry.AbstractAxisBlock)).To(Equal(float32(20)))
	})

	It("SetAbstract sets Width for Inline", func() {
		s := geometry.Size[float32]{Width: 10, Height: 20}
		s.SetAbstract(geometry.AbstractAxisInline, 99)
		Expect(s.Width).To(Equal(float32(99)))
		Expect(s.Height).To(Equal(float32(20)))
	})

	It("SetAbstract sets Height for Block", func() {
		s := geometry.Size[float32]{Width: 10, Height: 20}
		s.SetAbstract(geometry.AbstractAxisBlock, 99)
		Expect(s.Width).To(Equal(float32(10)))
		Expect(s.Height).To(Equal(float32(99)))
	})

	It("Map transforms both fields", func() {
		s := geometry.Size[float32]{Width: 2, Height: 3}
		s2 := s.Map(func(v float32) float32 { return v * 2 })
		Expect(s2.Width).To(Equal(float32(4)))
		Expect(s2.Height).To(Equal(float32(6)))
	})

	It("MapWidth transforms only Width", func() {
		s := geometry.Size[float32]{Width: 2, Height: 3}
		s2 := s.MapWidth(func(v float32) float32 { return v * 10 })
		Expect(s2.Width).To(Equal(float32(20)))
		Expect(s2.Height).To(Equal(float32(3)))
	})

	It("MapHeight transforms only Height", func() {
		s := geometry.Size[float32]{Width: 2, Height: 3}
		s2 := s.MapHeight(func(v float32) float32 { return v * 10 })
		Expect(s2.Width).To(Equal(float32(2)))
		Expect(s2.Height).To(Equal(float32(30)))
	})

	It("geometry.SizeMapTo transforms to a different type", func() {
		s := geometry.Size[float32]{Width: 1.5, Height: 2.5}
		s2 := geometry.SizeMapTo(s, func(v float32) int { return int(v) })
		Expect(s2.Width).To(Equal(1))
		Expect(s2.Height).To(Equal(2))
	})

	It("geometry.SizeZipMap combines two sizes", func() {
		a := geometry.Size[float32]{Width: 1, Height: 2}
		b := geometry.Size[float32]{Width: 3, Height: 4}
		c := geometry.SizeZipMap(a, b, func(x, y float32) float32 { return x + y })
		Expect(c.Width).To(Equal(float32(4)))
		Expect(c.Height).To(Equal(float32(6)))
	})

	It("geometry.SizeAddF32 adds component-wise", func() {
		a := geometry.Size[float32]{Width: 1, Height: 2}
		b := geometry.Size[float32]{Width: 3, Height: 4}
		c := geometry.SizeAddF32(a, b)
		Expect(c.Width).To(Equal(float32(4)))
		Expect(c.Height).To(Equal(float32(6)))
	})

	It("geometry.SizeSubF32 subtracts component-wise", func() {
		a := geometry.Size[float32]{Width: 10, Height: 20}
		b := geometry.Size[float32]{Width: 3, Height: 4}
		c := geometry.SizeSubF32(a, b)
		Expect(c.Width).To(Equal(float32(7)))
		Expect(c.Height).To(Equal(float32(16)))
	})

	It("geometry.SizeF32Max returns max per component", func() {
		a := geometry.Size[float32]{Width: 1, Height: 10}
		b := geometry.Size[float32]{Width: 5, Height: 3}
		c := geometry.SizeF32Max(a, b)
		Expect(c.Width).To(Equal(float32(5)))
		Expect(c.Height).To(Equal(float32(10)))
	})

	It("geometry.SizeF32Min returns min per component", func() {
		a := geometry.Size[float32]{Width: 1, Height: 10}
		b := geometry.Size[float32]{Width: 5, Height: 3}
		c := geometry.SizeF32Min(a, b)
		Expect(c.Width).To(Equal(float32(1)))
		Expect(c.Height).To(Equal(float32(3)))
	})

	It("geometry.SizeHasNonZeroArea returns true when both > 0", func() {
		Expect(geometry.SizeHasNonZeroArea(geometry.Size[float32]{Width: 1, Height: 1})).To(BeTrue())
	})

	It("geometry.SizeHasNonZeroArea returns false when width is 0", func() {
		Expect(geometry.SizeHasNonZeroArea(geometry.Size[float32]{Width: 0, Height: 1})).To(BeFalse())
	})

	It("geometry.SizeHasNonZeroArea returns false when height is 0", func() {
		Expect(geometry.SizeHasNonZeroArea(geometry.Size[float32]{Width: 1, Height: 0})).To(BeFalse())
	})
})

var _ = Describe("SizeOption", func() {
	It("geometry.SizeOptionMaybeApplyAspectRatio does nothing when aspect ratio is None", func() {
		s := geometry.NewSizeOptionF32(100, 50)
		result := geometry.SizeOptionMaybeApplyAspectRatio(s, lib.None[float32]())
		Expect(result).To(Equal(s))
	})

	It("geometry.SizeOptionMaybeApplyAspectRatio computes height from width", func() {
		s := geometry.Size[lib.Option[float32]]{
			Width:  lib.Some(float32(100)),
			Height: lib.None[float32](),
		}
		// aspect ratio = width/height = 2, so height = width/ratio = 50
		result := geometry.SizeOptionMaybeApplyAspectRatio(s, lib.Some(float32(2)))
		Expect(result.Width.Value).To(Equal(float32(100)))
		Expect(result.Height.Value).To(Equal(float32(50)))
	})

	It("geometry.SizeOptionMaybeApplyAspectRatio computes width from height", func() {
		s := geometry.Size[lib.Option[float32]]{
			Width:  lib.None[float32](),
			Height: lib.Some(float32(50)),
		}
		// aspect ratio = 2, so width = height * ratio = 100
		result := geometry.SizeOptionMaybeApplyAspectRatio(s, lib.Some(float32(2)))
		Expect(result.Width.Value).To(Equal(float32(100)))
		Expect(result.Height.Value).To(Equal(float32(50)))
	})

	It("geometry.SizeOptionMaybeApplyAspectRatio does nothing when both are Some", func() {
		s := geometry.NewSizeOptionF32(100, 50)
		result := geometry.SizeOptionMaybeApplyAspectRatio(s, lib.Some(float32(3)))
		Expect(result).To(Equal(s))
	})

	It("geometry.SizeOptionMaybeApplyAspectRatio does nothing when both are None", func() {
		s := geometry.SizeOptionNone
		result := geometry.SizeOptionMaybeApplyAspectRatio(s, lib.Some(float32(2)))
		Expect(result).To(Equal(s))
	})

	It("geometry.SizeOptionUnwrapOr returns values where Some, alt where None", func() {
		s := geometry.Size[lib.Option[float32]]{
			Width:  lib.Some(float32(10)),
			Height: lib.None[float32](),
		}
		alt := geometry.Size[float32]{Width: 99, Height: 88}
		result := geometry.SizeOptionUnwrapOr(s, alt)
		Expect(result.Width).To(Equal(float32(10)))
		Expect(result.Height).To(Equal(float32(88)))
	})

	It("geometry.SizeOptionBothAxisDefined returns true when both Some", func() {
		s := geometry.NewSizeOptionF32(1, 2)
		Expect(geometry.SizeOptionBothAxisDefined(s)).To(BeTrue())
	})

	It("geometry.SizeOptionBothAxisDefined returns false when one is None", func() {
		s := geometry.Size[lib.Option[float32]]{
			Width:  lib.Some(float32(1)),
			Height: lib.None[float32](),
		}
		Expect(geometry.SizeOptionBothAxisDefined(s)).To(BeFalse())
	})
})

var _ = Describe("Point", func() {
	It("Map transforms both fields", func() {
		p := geometry.Point[float32]{X: 1, Y: 2}
		p2 := p.Map(func(v float32) float32 { return v * 3 })
		Expect(p2.X).To(Equal(float32(3)))
		Expect(p2.Y).To(Equal(float32(6)))
	})

	It("Transpose swaps X and Y", func() {
		p := geometry.Point[float32]{X: 1, Y: 2}
		p2 := p.Transpose()
		Expect(p2.X).To(Equal(float32(2)))
		Expect(p2.Y).To(Equal(float32(1)))
	})

	It("GetAbstract returns X for Inline", func() {
		p := geometry.Point[float32]{X: 5, Y: 10}
		Expect(p.GetAbstract(geometry.AbstractAxisInline)).To(Equal(float32(5)))
	})

	It("GetAbstract returns Y for Block", func() {
		p := geometry.Point[float32]{X: 5, Y: 10}
		Expect(p.GetAbstract(geometry.AbstractAxisBlock)).To(Equal(float32(10)))
	})

	It("SetAbstract sets X for Inline", func() {
		p := geometry.Point[float32]{X: 5, Y: 10}
		p.SetAbstract(geometry.AbstractAxisInline, 99)
		Expect(p.X).To(Equal(float32(99)))
		Expect(p.Y).To(Equal(float32(10)))
	})

	It("SetAbstract sets Y for Block", func() {
		p := geometry.Point[float32]{X: 5, Y: 10}
		p.SetAbstract(geometry.AbstractAxisBlock, 99)
		Expect(p.X).To(Equal(float32(5)))
		Expect(p.Y).To(Equal(float32(99)))
	})

	It("geometry.PointAddF32 adds component-wise", func() {
		a := geometry.Point[float32]{X: 1, Y: 2}
		b := geometry.Point[float32]{X: 3, Y: 4}
		c := geometry.PointAddF32(a, b)
		Expect(c.X).To(Equal(float32(4)))
		Expect(c.Y).To(Equal(float32(6)))
	})

	It("PointToSize converts Point to Size", func() {
		p := geometry.Point[float32]{X: 7, Y: 8}
		s := geometry.PointToSize(p)
		Expect(s.Width).To(Equal(float32(7)))
		Expect(s.Height).To(Equal(float32(8)))
	})

	It("PointMapTo transforms to a different type", func() {
		p := geometry.Point[float32]{X: 1.5, Y: 2.5}
		p2 := geometry.PointMapTo(p, func(v float32) int { return int(v) })
		Expect(p2.X).To(Equal(1))
		Expect(p2.Y).To(Equal(2))
	})
})

var _ = Describe("MinMax", func() {
	It("holds min and max values", func() {
		mm := geometry.MinMax[float32, float32]{Min: 1.0, Max: 10.0}
		Expect(mm.Min).To(Equal(float32(1.0)))
		Expect(mm.Max).To(Equal(float32(10.0)))
	})
})
