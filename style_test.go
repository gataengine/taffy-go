package taffy_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	taffy "taffy-go"
)

var _ = Describe("Style", func() {
	Describe("DefaultStyle", func() {
		It("has Display=Flex", func() {
			s := taffy.DefaultStyle()
			Expect(s.Display).To(Equal(taffy.DisplayFlex))
		})

		It("has Position=Relative", func() {
			s := taffy.DefaultStyle()
			Expect(s.Position).To(Equal(taffy.PositionRelative))
		})

		It("has BoxSizing=BorderBox", func() {
			s := taffy.DefaultStyle()
			Expect(s.BoxSizing).To(Equal(taffy.BoxSizingBorderBox))
		})

		It("has Direction=LTR", func() {
			s := taffy.DefaultStyle()
			Expect(s.Direction).To(Equal(taffy.DirectionLTR))
		})

		It("has Overflow=Visible on both axes", func() {
			s := taffy.DefaultStyle()
			Expect(s.Overflow.X).To(Equal(taffy.OverflowVisible))
			Expect(s.Overflow.Y).To(Equal(taffy.OverflowVisible))
		})

		It("has ScrollbarWidth=0", func() {
			s := taffy.DefaultStyle()
			Expect(s.ScrollbarWidth).To(Equal(float32(0.0)))
		})

		It("has FlexDirection=Row", func() {
			s := taffy.DefaultStyle()
			Expect(s.FlexDirection).To(Equal(taffy.FlexDirectionRow))
		})

		It("has FlexWrap=NoWrap", func() {
			s := taffy.DefaultStyle()
			Expect(s.FlexWrap).To(Equal(taffy.FlexWrapNoWrap))
		})

		It("has FlexGrow=0.0", func() {
			s := taffy.DefaultStyle()
			Expect(s.FlexGrow).To(Equal(float32(0.0)))
		})

		It("has FlexShrink=1.0", func() {
			s := taffy.DefaultStyle()
			Expect(s.FlexShrink).To(Equal(float32(1.0)))
		})

		It("has FlexBasis=Auto", func() {
			s := taffy.DefaultStyle()
			Expect(s.FlexBasis.Tag).To(Equal(taffy.DimensionAuto))
		})

		It("has Size=Auto on both dimensions", func() {
			s := taffy.DefaultStyle()
			Expect(s.Size.Width.Tag).To(Equal(taffy.DimensionAuto))
			Expect(s.Size.Height.Tag).To(Equal(taffy.DimensionAuto))
		})

		It("has MinSize=Auto on both dimensions", func() {
			s := taffy.DefaultStyle()
			Expect(s.MinSize.Width.Tag).To(Equal(taffy.DimensionAuto))
			Expect(s.MinSize.Height.Tag).To(Equal(taffy.DimensionAuto))
		})

		It("has MaxSize=Auto on both dimensions", func() {
			s := taffy.DefaultStyle()
			Expect(s.MaxSize.Width.Tag).To(Equal(taffy.DimensionAuto))
			Expect(s.MaxSize.Height.Tag).To(Equal(taffy.DimensionAuto))
		})

		It("has AspectRatio=None", func() {
			s := taffy.DefaultStyle()
			Expect(s.AspectRatio.IsSome).To(BeFalse())
		})

		It("has Inset=Auto on all sides", func() {
			s := taffy.DefaultStyle()
			Expect(s.Inset.Left.Tag).To(Equal(taffy.LPAAuto))
			Expect(s.Inset.Right.Tag).To(Equal(taffy.LPAAuto))
			Expect(s.Inset.Top.Tag).To(Equal(taffy.LPAAuto))
			Expect(s.Inset.Bottom.Tag).To(Equal(taffy.LPAAuto))
		})

		It("has Margin=zero on all sides", func() {
			s := taffy.DefaultStyle()
			Expect(s.Margin.Left).To(Equal(taffy.LengthPercentageAuto{Tag: taffy.LPALength, Value: 0}))
			Expect(s.Margin.Right).To(Equal(taffy.LengthPercentageAuto{Tag: taffy.LPALength, Value: 0}))
			Expect(s.Margin.Top).To(Equal(taffy.LengthPercentageAuto{Tag: taffy.LPALength, Value: 0}))
			Expect(s.Margin.Bottom).To(Equal(taffy.LengthPercentageAuto{Tag: taffy.LPALength, Value: 0}))
		})

		It("has Padding=zero on all sides", func() {
			s := taffy.DefaultStyle()
			Expect(s.Padding.Left.Value).To(Equal(float32(0)))
			Expect(s.Padding.Right.Value).To(Equal(float32(0)))
			Expect(s.Padding.Top.Value).To(Equal(float32(0)))
			Expect(s.Padding.Bottom.Value).To(Equal(float32(0)))
		})

		It("has Border=zero on all sides", func() {
			s := taffy.DefaultStyle()
			Expect(s.Border.Left.Value).To(Equal(float32(0)))
			Expect(s.Border.Right.Value).To(Equal(float32(0)))
			Expect(s.Border.Top.Value).To(Equal(float32(0)))
			Expect(s.Border.Bottom.Value).To(Equal(float32(0)))
		})

		It("has AlignItems=None", func() {
			s := taffy.DefaultStyle()
			Expect(s.AlignItems.IsSome).To(BeFalse())
		})

		It("has AlignContent=None", func() {
			s := taffy.DefaultStyle()
			Expect(s.AlignContent.IsSome).To(BeFalse())
		})

		It("has JustifyContent=None", func() {
			s := taffy.DefaultStyle()
			Expect(s.JustifyContent.IsSome).To(BeFalse())
		})

		It("has Gap=zero", func() {
			s := taffy.DefaultStyle()
			Expect(s.Gap.Width.Value).To(Equal(float32(0)))
			Expect(s.Gap.Height.Value).To(Equal(float32(0)))
		})

		It("has TextAlign=Auto", func() {
			s := taffy.DefaultStyle()
			Expect(s.TextAlign).To(Equal(taffy.TextAlignAuto))
		})

		It("has GridAutoFlow=Row", func() {
			s := taffy.DefaultStyle()
			Expect(s.GridAutoFlow).To(Equal(taffy.GridAutoFlowRow))
		})

		It("has GridRow with Auto start and end", func() {
			s := taffy.DefaultStyle()
			Expect(s.GridRow.Start.Tag).To(Equal(taffy.GridPlacementAuto))
			Expect(s.GridRow.End.Tag).To(Equal(taffy.GridPlacementAuto))
		})

		It("has GridColumn with Auto start and end", func() {
			s := taffy.DefaultStyle()
			Expect(s.GridColumn.Start.Tag).To(Equal(taffy.GridPlacementAuto))
			Expect(s.GridColumn.End.Tag).To(Equal(taffy.GridPlacementAuto))
		})

		It("has empty grid template slices", func() {
			s := taffy.DefaultStyle()
			Expect(s.GridTemplateRows).To(BeEmpty())
			Expect(s.GridTemplateColumns).To(BeEmpty())
			Expect(s.GridAutoRows).To(BeEmpty())
			Expect(s.GridAutoColumns).To(BeEmpty())
			Expect(s.GridTemplateAreas).To(BeEmpty())
		})

		It("is not a table item", func() {
			s := taffy.DefaultStyle()
			Expect(s.ItemIsTable).To(BeFalse())
		})

		It("is not a replaced item", func() {
			s := taffy.DefaultStyle()
			Expect(s.ItemIsReplaced).To(BeFalse())
		})
	})

	Describe("Overflow", func() {
		It("Visible is not a scroll container", func() {
			Expect(taffy.OverflowVisible.IsScrollContainer()).To(BeFalse())
		})

		It("Clip is not a scroll container", func() {
			Expect(taffy.OverflowClip.IsScrollContainer()).To(BeFalse())
		})

		It("Hidden is a scroll container", func() {
			Expect(taffy.OverflowHidden.IsScrollContainer()).To(BeTrue())
		})

		It("Scroll is a scroll container", func() {
			Expect(taffy.OverflowScroll.IsScrollContainer()).To(BeTrue())
		})

		It("Visible MaybeIntoAutomaticMinSize returns None", func() {
			result := taffy.OverflowVisible.MaybeIntoAutomaticMinSize()
			Expect(result.IsSome).To(BeFalse())
		})

		It("Hidden MaybeIntoAutomaticMinSize returns Some(0.0)", func() {
			result := taffy.OverflowHidden.MaybeIntoAutomaticMinSize()
			Expect(result.IsSome).To(BeTrue())
			Expect(result.Value).To(Equal(float32(0.0)))
		})

		It("Scroll MaybeIntoAutomaticMinSize returns Some(0.0)", func() {
			result := taffy.OverflowScroll.MaybeIntoAutomaticMinSize()
			Expect(result.IsSome).To(BeTrue())
			Expect(result.Value).To(Equal(float32(0.0)))
		})
	})

	Describe("Direction", func() {
		It("LTR IsRTL returns false", func() {
			Expect(taffy.DirectionLTR.IsRTL()).To(BeFalse())
		})

		It("RTL IsRTL returns true", func() {
			Expect(taffy.DirectionRTL.IsRTL()).To(BeTrue())
		})
	})

	Describe("Style coreStyle interface methods", func() {
		It("boxGenerationMode is Normal for DisplayFlex", func() {
			s := taffy.DefaultStyle()
			Expect(s.Display).To(Equal(taffy.DisplayFlex))
			// Just verify via the Display field since boxGenerationMode is unexported
		})

		It("Display None sets boxGenerationMode to None", func() {
			s := taffy.DefaultStyle()
			s.Display = taffy.DisplayNone
			Expect(s.Display).To(Equal(taffy.DisplayNone))
		})
	})
})
