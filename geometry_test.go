package taffy_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	taffy "taffy-go"
)

var _ = Describe("Option", func() {
	It("Some holds a value", func() {
		o := taffy.Some(42)
		Expect(o.IsSome).To(BeTrue())
		Expect(o.Value).To(Equal(42))
	})

	It("None holds no value", func() {
		o := taffy.None[int]()
		Expect(o.IsSome).To(BeFalse())
	})

	It("UnwrapOr returns value when Some", func() {
		Expect(taffy.Some(7).UnwrapOr(0)).To(Equal(7))
	})

	It("UnwrapOr returns alt when None", func() {
		Expect(taffy.None[int]().UnwrapOr(99)).To(Equal(99))
	})

	It("Or returns self when Some", func() {
		a := taffy.Some(1)
		b := taffy.Some(2)
		Expect(a.Or(b)).To(Equal(a))
	})

	It("Or returns alt when None", func() {
		a := taffy.None[int]()
		b := taffy.Some(2)
		Expect(a.Or(b)).To(Equal(b))
	})
})

var _ = Describe("AbsoluteAxis", func() {
	It("OtherAxis of Horizontal is Vertical", func() {
		Expect(taffy.AbsoluteAxisHorizontal.OtherAxis()).To(Equal(taffy.AbsoluteAxisVertical))
	})

	It("OtherAxis of Vertical is Horizontal", func() {
		Expect(taffy.AbsoluteAxisVertical.OtherAxis()).To(Equal(taffy.AbsoluteAxisHorizontal))
	})
})

var _ = Describe("AbstractAxis", func() {
	It("Other of Inline is Block", func() {
		Expect(taffy.AbstractAxisInline.Other()).To(Equal(taffy.AbstractAxisBlock))
	})

	It("Other of Block is Inline", func() {
		Expect(taffy.AbstractAxisBlock.Other()).To(Equal(taffy.AbstractAxisInline))
	})

	It("AsAbsNaive converts Inline to Horizontal", func() {
		Expect(taffy.AbstractAxisInline.AsAbsNaive()).To(Equal(taffy.AbsoluteAxisHorizontal))
	})

	It("AsAbsNaive converts Block to Vertical", func() {
		Expect(taffy.AbstractAxisBlock.AsAbsNaive()).To(Equal(taffy.AbsoluteAxisVertical))
	})
})

var _ = Describe("Rect", func() {
	It("HorizontalComponents returns left and right as Line", func() {
		r := taffy.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		l := r.HorizontalComponents()
		Expect(l.Start).To(Equal(float32(1)))
		Expect(l.End).To(Equal(float32(2)))
	})

	It("VerticalComponents returns top and bottom as Line", func() {
		r := taffy.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		l := r.VerticalComponents()
		Expect(l.Start).To(Equal(float32(3)))
		Expect(l.End).To(Equal(float32(4)))
	})

	It("Map transforms all four sides", func() {
		r := taffy.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		r2 := r.Map(func(v float32) float32 { return v * 2 })
		Expect(r2.Left).To(Equal(float32(2)))
		Expect(r2.Right).To(Equal(float32(4)))
		Expect(r2.Top).To(Equal(float32(6)))
		Expect(r2.Bottom).To(Equal(float32(8)))
	})

	It("RectMapTo transforms to a different type", func() {
		r := taffy.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		r2 := taffy.RectMapTo(r, func(v float32) int { return int(v) })
		Expect(r2.Left).To(Equal(1))
		Expect(r2.Right).To(Equal(2))
		Expect(r2.Top).To(Equal(3))
		Expect(r2.Bottom).To(Equal(4))
	})

	It("RectHorizontalAxisSum returns left+right", func() {
		r := taffy.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		Expect(taffy.RectHorizontalAxisSum(r)).To(Equal(float32(3)))
	})

	It("RectVerticalAxisSum returns top+bottom", func() {
		r := taffy.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		Expect(taffy.RectVerticalAxisSum(r)).To(Equal(float32(7)))
	})

	It("RectSumAxes returns both axis sums", func() {
		r := taffy.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		s := taffy.RectSumAxes(r)
		Expect(s.Width).To(Equal(float32(3)))
		Expect(s.Height).To(Equal(float32(7)))
	})

	It("RectGridAxisSum returns correct sum for each axis", func() {
		r := taffy.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		Expect(taffy.RectGridAxisSum(r, taffy.AbsoluteAxisHorizontal)).To(Equal(float32(3)))
		Expect(taffy.RectGridAxisSum(r, taffy.AbsoluteAxisVertical)).To(Equal(float32(7)))
	})

	It("RectAddF32 adds component-wise", func() {
		a := taffy.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		b := taffy.Rect[float32]{Left: 10, Right: 20, Top: 30, Bottom: 40}
		c := taffy.RectAddF32(a, b)
		Expect(c.Left).To(Equal(float32(11)))
		Expect(c.Right).To(Equal(float32(22)))
		Expect(c.Top).To(Equal(float32(33)))
		Expect(c.Bottom).To(Equal(float32(44)))
	})

	It("RectZipSize applies f with corresponding axis sizes", func() {
		r := taffy.Rect[float32]{Left: 1, Right: 2, Top: 3, Bottom: 4}
		s := taffy.Size[float32]{Width: 10, Height: 100}
		r2 := taffy.RectZipSize(r, s, func(a, b float32) float32 { return a + b })
		Expect(r2.Left).To(Equal(float32(11)))
		Expect(r2.Right).To(Equal(float32(12)))
		Expect(r2.Top).To(Equal(float32(103)))
		Expect(r2.Bottom).To(Equal(float32(104)))
	})
})

var _ = Describe("Line", func() {
	It("Map transforms both fields", func() {
		l := taffy.Line[float32]{Start: 1, End: 2}
		l2 := l.Map(func(v float32) float32 { return v + 10 })
		Expect(l2.Start).To(Equal(float32(11)))
		Expect(l2.End).To(Equal(float32(12)))
	})

	It("LineMapTo transforms to a different type", func() {
		l := taffy.Line[float32]{Start: 1.5, End: 2.5}
		l2 := taffy.LineMapTo(l, func(v float32) int { return int(v) })
		Expect(l2.Start).To(Equal(1))
		Expect(l2.End).To(Equal(2))
	})

	It("LineSumF32 adds start and end", func() {
		l := taffy.Line[float32]{Start: 3, End: 7}
		Expect(taffy.LineSumF32(l)).To(Equal(float32(10)))
	})

	It("LineTrueBool has both fields true", func() {
		Expect(taffy.LineTrueBool.Start).To(BeTrue())
		Expect(taffy.LineTrueBool.End).To(BeTrue())
	})

	It("LineFalseBool has both fields false", func() {
		Expect(taffy.LineFalseBool.Start).To(BeFalse())
		Expect(taffy.LineFalseBool.End).To(BeFalse())
	})
})

var _ = Describe("Size", func() {
	It("GetAbs returns Width for Horizontal", func() {
		s := taffy.Size[float32]{Width: 10, Height: 20}
		Expect(s.GetAbs(taffy.AbsoluteAxisHorizontal)).To(Equal(float32(10)))
	})

	It("GetAbs returns Height for Vertical", func() {
		s := taffy.Size[float32]{Width: 10, Height: 20}
		Expect(s.GetAbs(taffy.AbsoluteAxisVertical)).To(Equal(float32(20)))
	})

	It("GetAbstract returns Width for Inline", func() {
		s := taffy.Size[float32]{Width: 10, Height: 20}
		Expect(s.GetAbstract(taffy.AbstractAxisInline)).To(Equal(float32(10)))
	})

	It("GetAbstract returns Height for Block", func() {
		s := taffy.Size[float32]{Width: 10, Height: 20}
		Expect(s.GetAbstract(taffy.AbstractAxisBlock)).To(Equal(float32(20)))
	})

	It("SetAbstract sets Width for Inline", func() {
		s := taffy.Size[float32]{Width: 10, Height: 20}
		s.SetAbstract(taffy.AbstractAxisInline, 99)
		Expect(s.Width).To(Equal(float32(99)))
		Expect(s.Height).To(Equal(float32(20)))
	})

	It("SetAbstract sets Height for Block", func() {
		s := taffy.Size[float32]{Width: 10, Height: 20}
		s.SetAbstract(taffy.AbstractAxisBlock, 99)
		Expect(s.Width).To(Equal(float32(10)))
		Expect(s.Height).To(Equal(float32(99)))
	})

	It("Map transforms both fields", func() {
		s := taffy.Size[float32]{Width: 2, Height: 3}
		s2 := s.Map(func(v float32) float32 { return v * 2 })
		Expect(s2.Width).To(Equal(float32(4)))
		Expect(s2.Height).To(Equal(float32(6)))
	})

	It("MapWidth transforms only Width", func() {
		s := taffy.Size[float32]{Width: 2, Height: 3}
		s2 := s.MapWidth(func(v float32) float32 { return v * 10 })
		Expect(s2.Width).To(Equal(float32(20)))
		Expect(s2.Height).To(Equal(float32(3)))
	})

	It("MapHeight transforms only Height", func() {
		s := taffy.Size[float32]{Width: 2, Height: 3}
		s2 := s.MapHeight(func(v float32) float32 { return v * 10 })
		Expect(s2.Width).To(Equal(float32(2)))
		Expect(s2.Height).To(Equal(float32(30)))
	})

	It("SizeMapTo transforms to a different type", func() {
		s := taffy.Size[float32]{Width: 1.5, Height: 2.5}
		s2 := taffy.SizeMapTo(s, func(v float32) int { return int(v) })
		Expect(s2.Width).To(Equal(1))
		Expect(s2.Height).To(Equal(2))
	})

	It("SizeZipMap combines two sizes", func() {
		a := taffy.Size[float32]{Width: 1, Height: 2}
		b := taffy.Size[float32]{Width: 3, Height: 4}
		c := taffy.SizeZipMap(a, b, func(x, y float32) float32 { return x + y })
		Expect(c.Width).To(Equal(float32(4)))
		Expect(c.Height).To(Equal(float32(6)))
	})

	It("SizeAddF32 adds component-wise", func() {
		a := taffy.Size[float32]{Width: 1, Height: 2}
		b := taffy.Size[float32]{Width: 3, Height: 4}
		c := taffy.SizeAddF32(a, b)
		Expect(c.Width).To(Equal(float32(4)))
		Expect(c.Height).To(Equal(float32(6)))
	})

	It("SizeSubF32 subtracts component-wise", func() {
		a := taffy.Size[float32]{Width: 10, Height: 20}
		b := taffy.Size[float32]{Width: 3, Height: 4}
		c := taffy.SizeSubF32(a, b)
		Expect(c.Width).To(Equal(float32(7)))
		Expect(c.Height).To(Equal(float32(16)))
	})

	It("SizeF32Max returns max per component", func() {
		a := taffy.Size[float32]{Width: 1, Height: 10}
		b := taffy.Size[float32]{Width: 5, Height: 3}
		c := taffy.SizeF32Max(a, b)
		Expect(c.Width).To(Equal(float32(5)))
		Expect(c.Height).To(Equal(float32(10)))
	})

	It("SizeF32Min returns min per component", func() {
		a := taffy.Size[float32]{Width: 1, Height: 10}
		b := taffy.Size[float32]{Width: 5, Height: 3}
		c := taffy.SizeF32Min(a, b)
		Expect(c.Width).To(Equal(float32(1)))
		Expect(c.Height).To(Equal(float32(3)))
	})

	It("SizeHasNonZeroArea returns true when both > 0", func() {
		Expect(taffy.SizeHasNonZeroArea(taffy.Size[float32]{Width: 1, Height: 1})).To(BeTrue())
	})

	It("SizeHasNonZeroArea returns false when width is 0", func() {
		Expect(taffy.SizeHasNonZeroArea(taffy.Size[float32]{Width: 0, Height: 1})).To(BeFalse())
	})

	It("SizeHasNonZeroArea returns false when height is 0", func() {
		Expect(taffy.SizeHasNonZeroArea(taffy.Size[float32]{Width: 1, Height: 0})).To(BeFalse())
	})
})

var _ = Describe("SizeOption", func() {
	It("SizeOptionMaybeApplyAspectRatio does nothing when aspect ratio is None", func() {
		s := taffy.NewSizeOptionF32(100, 50)
		result := taffy.SizeOptionMaybeApplyAspectRatio(s, taffy.None[float32]())
		Expect(result).To(Equal(s))
	})

	It("SizeOptionMaybeApplyAspectRatio computes height from width", func() {
		s := taffy.Size[taffy.Option[float32]]{
			Width:  taffy.Some(float32(100)),
			Height: taffy.None[float32](),
		}
		// aspect ratio = width/height = 2, so height = width/ratio = 50
		result := taffy.SizeOptionMaybeApplyAspectRatio(s, taffy.Some(float32(2)))
		Expect(result.Width.Value).To(Equal(float32(100)))
		Expect(result.Height.Value).To(Equal(float32(50)))
	})

	It("SizeOptionMaybeApplyAspectRatio computes width from height", func() {
		s := taffy.Size[taffy.Option[float32]]{
			Width:  taffy.None[float32](),
			Height: taffy.Some(float32(50)),
		}
		// aspect ratio = 2, so width = height * ratio = 100
		result := taffy.SizeOptionMaybeApplyAspectRatio(s, taffy.Some(float32(2)))
		Expect(result.Width.Value).To(Equal(float32(100)))
		Expect(result.Height.Value).To(Equal(float32(50)))
	})

	It("SizeOptionMaybeApplyAspectRatio does nothing when both are Some", func() {
		s := taffy.NewSizeOptionF32(100, 50)
		result := taffy.SizeOptionMaybeApplyAspectRatio(s, taffy.Some(float32(3)))
		Expect(result).To(Equal(s))
	})

	It("SizeOptionMaybeApplyAspectRatio does nothing when both are None", func() {
		s := taffy.SizeOptionNone
		result := taffy.SizeOptionMaybeApplyAspectRatio(s, taffy.Some(float32(2)))
		Expect(result).To(Equal(s))
	})

	It("SizeOptionUnwrapOr returns values where Some, alt where None", func() {
		s := taffy.Size[taffy.Option[float32]]{
			Width:  taffy.Some(float32(10)),
			Height: taffy.None[float32](),
		}
		alt := taffy.Size[float32]{Width: 99, Height: 88}
		result := taffy.SizeOptionUnwrapOr(s, alt)
		Expect(result.Width).To(Equal(float32(10)))
		Expect(result.Height).To(Equal(float32(88)))
	})

	It("SizeOptionBothAxisDefined returns true when both Some", func() {
		s := taffy.NewSizeOptionF32(1, 2)
		Expect(taffy.SizeOptionBothAxisDefined(s)).To(BeTrue())
	})

	It("SizeOptionBothAxisDefined returns false when one is None", func() {
		s := taffy.Size[taffy.Option[float32]]{
			Width:  taffy.Some(float32(1)),
			Height: taffy.None[float32](),
		}
		Expect(taffy.SizeOptionBothAxisDefined(s)).To(BeFalse())
	})
})

var _ = Describe("Point", func() {
	It("Map transforms both fields", func() {
		p := taffy.Point[float32]{X: 1, Y: 2}
		p2 := p.Map(func(v float32) float32 { return v * 3 })
		Expect(p2.X).To(Equal(float32(3)))
		Expect(p2.Y).To(Equal(float32(6)))
	})

	It("Transpose swaps X and Y", func() {
		p := taffy.Point[float32]{X: 1, Y: 2}
		p2 := p.Transpose()
		Expect(p2.X).To(Equal(float32(2)))
		Expect(p2.Y).To(Equal(float32(1)))
	})

	It("GetAbstract returns X for Inline", func() {
		p := taffy.Point[float32]{X: 5, Y: 10}
		Expect(p.GetAbstract(taffy.AbstractAxisInline)).To(Equal(float32(5)))
	})

	It("GetAbstract returns Y for Block", func() {
		p := taffy.Point[float32]{X: 5, Y: 10}
		Expect(p.GetAbstract(taffy.AbstractAxisBlock)).To(Equal(float32(10)))
	})

	It("SetAbstract sets X for Inline", func() {
		p := taffy.Point[float32]{X: 5, Y: 10}
		p.SetAbstract(taffy.AbstractAxisInline, 99)
		Expect(p.X).To(Equal(float32(99)))
		Expect(p.Y).To(Equal(float32(10)))
	})

	It("SetAbstract sets Y for Block", func() {
		p := taffy.Point[float32]{X: 5, Y: 10}
		p.SetAbstract(taffy.AbstractAxisBlock, 99)
		Expect(p.X).To(Equal(float32(5)))
		Expect(p.Y).To(Equal(float32(99)))
	})

	It("PointAddF32 adds component-wise", func() {
		a := taffy.Point[float32]{X: 1, Y: 2}
		b := taffy.Point[float32]{X: 3, Y: 4}
		c := taffy.PointAddF32(a, b)
		Expect(c.X).To(Equal(float32(4)))
		Expect(c.Y).To(Equal(float32(6)))
	})

	It("PointToSize converts Point to Size", func() {
		p := taffy.Point[float32]{X: 7, Y: 8}
		s := taffy.PointToSize(p)
		Expect(s.Width).To(Equal(float32(7)))
		Expect(s.Height).To(Equal(float32(8)))
	})

	It("PointMapTo transforms to a different type", func() {
		p := taffy.Point[float32]{X: 1.5, Y: 2.5}
		p2 := taffy.PointMapTo(p, func(v float32) int { return int(v) })
		Expect(p2.X).To(Equal(1))
		Expect(p2.Y).To(Equal(2))
	})
})

var _ = Describe("MinMax", func() {
	It("holds min and max values", func() {
		mm := taffy.MinMax[float32, float32]{Min: 1.0, Max: 10.0}
		Expect(mm.Min).To(Equal(float32(1.0)))
		Expect(mm.Max).To(Equal(float32(10.0)))
	})
})
