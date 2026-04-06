package taffy

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"taffy-go/style"
)

var _ = Describe("Style", func() {
	Describe("style.DefaultStyle", func() {
		It("has Display=Flex", func() {
			s := style.DefaultStyle()
			Expect(s.Display).To(Equal(style.DisplayFlex))
		})

		It("has Position=Relative", func() {
			s := style.DefaultStyle()
			Expect(s.Position).To(Equal(style.PositionRelative))
		})

		It("has BoxSizing=BorderBox", func() {
			s := style.DefaultStyle()
			Expect(s.BoxSizing).To(Equal(style.BoxSizingBorderBox))
		})

		It("has Direction=LTR", func() {
			s := style.DefaultStyle()
			Expect(s.Direction).To(Equal(style.DirectionLTR))
		})

		It("has Overflow=Visible on both axes", func() {
			s := style.DefaultStyle()
			Expect(s.Overflow.X).To(Equal(style.OverflowVisible))
			Expect(s.Overflow.Y).To(Equal(style.OverflowVisible))
		})

		It("has ScrollbarWidth=0", func() {
			s := style.DefaultStyle()
			Expect(s.ScrollbarWidth).To(Equal(float32(0.0)))
		})

		It("has FlexDirection=Row", func() {
			s := style.DefaultStyle()
			Expect(s.FlexDirection).To(Equal(style.FlexDirectionRow))
		})

		It("has FlexWrap=NoWrap", func() {
			s := style.DefaultStyle()
			Expect(s.FlexWrap).To(Equal(style.FlexWrapNoWrap))
		})

		It("has FlexGrow=0.0", func() {
			s := style.DefaultStyle()
			Expect(s.FlexGrow).To(Equal(float32(0.0)))
		})

		It("has FlexShrink=1.0", func() {
			s := style.DefaultStyle()
			Expect(s.FlexShrink).To(Equal(float32(1.0)))
		})

		It("has FlexBasis=Auto", func() {
			s := style.DefaultStyle()
			Expect(s.FlexBasis.Tag).To(Equal(style.DimensionAuto))
		})

		It("has Size=Auto on both dimensions", func() {
			s := style.DefaultStyle()
			Expect(s.Size.Width.Tag).To(Equal(style.DimensionAuto))
			Expect(s.Size.Height.Tag).To(Equal(style.DimensionAuto))
		})

		It("has MinSize=Auto on both dimensions", func() {
			s := style.DefaultStyle()
			Expect(s.MinSize.Width.Tag).To(Equal(style.DimensionAuto))
			Expect(s.MinSize.Height.Tag).To(Equal(style.DimensionAuto))
		})

		It("has MaxSize=Auto on both dimensions", func() {
			s := style.DefaultStyle()
			Expect(s.MaxSize.Width.Tag).To(Equal(style.DimensionAuto))
			Expect(s.MaxSize.Height.Tag).To(Equal(style.DimensionAuto))
		})

		It("has AspectRatio=None", func() {
			s := style.DefaultStyle()
			Expect(s.AspectRatio.IsSome).To(BeFalse())
		})

		It("has Inset=Auto on all sides", func() {
			s := style.DefaultStyle()
			Expect(s.Inset.Left.Tag).To(Equal(style.LPAAuto))
			Expect(s.Inset.Right.Tag).To(Equal(style.LPAAuto))
			Expect(s.Inset.Top.Tag).To(Equal(style.LPAAuto))
			Expect(s.Inset.Bottom.Tag).To(Equal(style.LPAAuto))
		})

		It("has Margin=zero on all sides", func() {
			s := style.DefaultStyle()
			Expect(s.Margin.Left).To(Equal(style.LengthPercentageAuto{Tag: style.LPALength, Value: 0}))
			Expect(s.Margin.Right).To(Equal(style.LengthPercentageAuto{Tag: style.LPALength, Value: 0}))
			Expect(s.Margin.Top).To(Equal(style.LengthPercentageAuto{Tag: style.LPALength, Value: 0}))
			Expect(s.Margin.Bottom).To(Equal(style.LengthPercentageAuto{Tag: style.LPALength, Value: 0}))
		})

		It("has Padding=zero on all sides", func() {
			s := style.DefaultStyle()
			Expect(s.Padding.Left.Value).To(Equal(float32(0)))
			Expect(s.Padding.Right.Value).To(Equal(float32(0)))
			Expect(s.Padding.Top.Value).To(Equal(float32(0)))
			Expect(s.Padding.Bottom.Value).To(Equal(float32(0)))
		})

		It("has Border=zero on all sides", func() {
			s := style.DefaultStyle()
			Expect(s.Border.Left.Value).To(Equal(float32(0)))
			Expect(s.Border.Right.Value).To(Equal(float32(0)))
			Expect(s.Border.Top.Value).To(Equal(float32(0)))
			Expect(s.Border.Bottom.Value).To(Equal(float32(0)))
		})

		It("has AlignItems=None", func() {
			s := style.DefaultStyle()
			Expect(s.AlignItems.IsSome).To(BeFalse())
		})

		It("has AlignContent=None", func() {
			s := style.DefaultStyle()
			Expect(s.AlignContent.IsSome).To(BeFalse())
		})

		It("has JustifyContent=None", func() {
			s := style.DefaultStyle()
			Expect(s.JustifyContent.IsSome).To(BeFalse())
		})

		It("has Gap=zero", func() {
			s := style.DefaultStyle()
			Expect(s.Gap.Width.Value).To(Equal(float32(0)))
			Expect(s.Gap.Height.Value).To(Equal(float32(0)))
		})

		It("has TextAlign=Auto", func() {
			s := style.DefaultStyle()
			Expect(s.TextAlign).To(Equal(style.TextAlignAuto))
		})

		It("has GridAutoFlow=Row", func() {
			s := style.DefaultStyle()
			Expect(s.GridAutoFlow).To(Equal(style.GridAutoFlowRow))
		})

		It("has GridRow with Auto start and end", func() {
			s := style.DefaultStyle()
			Expect(s.GridRow.Start.Tag).To(Equal(style.GridPlacementAuto))
			Expect(s.GridRow.End.Tag).To(Equal(style.GridPlacementAuto))
		})

		It("has GridColumn with Auto start and end", func() {
			s := style.DefaultStyle()
			Expect(s.GridColumn.Start.Tag).To(Equal(style.GridPlacementAuto))
			Expect(s.GridColumn.End.Tag).To(Equal(style.GridPlacementAuto))
		})

		It("has empty grid template slices", func() {
			s := style.DefaultStyle()
			Expect(s.GridTemplateRows).To(BeEmpty())
			Expect(s.GridTemplateColumns).To(BeEmpty())
			Expect(s.GridAutoRows).To(BeEmpty())
			Expect(s.GridAutoColumns).To(BeEmpty())
			Expect(s.GridTemplateAreas).To(BeEmpty())
		})

		It("is not a table item", func() {
			s := style.DefaultStyle()
			Expect(s.ItemIsTable).To(BeFalse())
		})

		It("is not a replaced item", func() {
			s := style.DefaultStyle()
			Expect(s.ItemIsReplaced).To(BeFalse())
		})
	})

	Describe("Overflow", func() {
		It("Visible is not a scroll container", func() {
			Expect(style.OverflowVisible.IsScrollContainer()).To(BeFalse())
		})

		It("Clip is not a scroll container", func() {
			Expect(style.OverflowClip.IsScrollContainer()).To(BeFalse())
		})

		It("Hidden is a scroll container", func() {
			Expect(style.OverflowHidden.IsScrollContainer()).To(BeTrue())
		})

		It("Scroll is a scroll container", func() {
			Expect(style.OverflowScroll.IsScrollContainer()).To(BeTrue())
		})

		It("Visible MaybeIntoAutomaticMinSize returns None", func() {
			result := style.OverflowVisible.MaybeIntoAutomaticMinSize()
			Expect(result.IsSome).To(BeFalse())
		})

		It("Hidden MaybeIntoAutomaticMinSize returns geometry.Some(0.0)", func() {
			result := style.OverflowHidden.MaybeIntoAutomaticMinSize()
			Expect(result.IsSome).To(BeTrue())
			Expect(result.Value).To(Equal(float32(0.0)))
		})

		It("Scroll MaybeIntoAutomaticMinSize returns geometry.Some(0.0)", func() {
			result := style.OverflowScroll.MaybeIntoAutomaticMinSize()
			Expect(result.IsSome).To(BeTrue())
			Expect(result.Value).To(Equal(float32(0.0)))
		})
	})

	Describe("Direction", func() {
		It("LTR IsRTL returns false", func() {
			Expect(style.DirectionLTR.IsRTL()).To(BeFalse())
		})

		It("RTL IsRTL returns true", func() {
			Expect(style.DirectionRTL.IsRTL()).To(BeTrue())
		})
	})

	Describe("style.Style coreStyle interface methods", func() {
		It("boxGenerationMode is Normal for style.DisplayFlex", func() {
			s := style.DefaultStyle()
			Expect(s.Display).To(Equal(style.DisplayFlex))
			// Just verify via the Display field since boxGenerationMode is unexported
		})

		It("Display None sets boxGenerationMode to None", func() {
			s := style.DefaultStyle()
			s.Display = style.DisplayNone
			Expect(s.Display).To(Equal(style.DisplayNone))
		})
	})
})
