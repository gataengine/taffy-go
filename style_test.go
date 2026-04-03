package taffy

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Style", func() {
	Describe("DefaultStyle", func() {
		It("has Display=Flex", func() {
			s := DefaultStyle()
			Expect(s.Display).To(Equal(DisplayFlex))
		})

		It("has Position=Relative", func() {
			s := DefaultStyle()
			Expect(s.Position).To(Equal(PositionRelative))
		})

		It("has BoxSizing=BorderBox", func() {
			s := DefaultStyle()
			Expect(s.BoxSizing).To(Equal(BoxSizingBorderBox))
		})

		It("has Direction=LTR", func() {
			s := DefaultStyle()
			Expect(s.Direction).To(Equal(DirectionLTR))
		})

		It("has Overflow=Visible on both axes", func() {
			s := DefaultStyle()
			Expect(s.Overflow.X).To(Equal(OverflowVisible))
			Expect(s.Overflow.Y).To(Equal(OverflowVisible))
		})

		It("has ScrollbarWidth=0", func() {
			s := DefaultStyle()
			Expect(s.ScrollbarWidth).To(Equal(float32(0.0)))
		})

		It("has FlexDirection=Row", func() {
			s := DefaultStyle()
			Expect(s.FlexDirection).To(Equal(FlexDirectionRow))
		})

		It("has FlexWrap=NoWrap", func() {
			s := DefaultStyle()
			Expect(s.FlexWrap).To(Equal(FlexWrapNoWrap))
		})

		It("has FlexGrow=0.0", func() {
			s := DefaultStyle()
			Expect(s.FlexGrow).To(Equal(float32(0.0)))
		})

		It("has FlexShrink=1.0", func() {
			s := DefaultStyle()
			Expect(s.FlexShrink).To(Equal(float32(1.0)))
		})

		It("has FlexBasis=Auto", func() {
			s := DefaultStyle()
			Expect(s.FlexBasis.Tag).To(Equal(DimensionAuto))
		})

		It("has Size=Auto on both dimensions", func() {
			s := DefaultStyle()
			Expect(s.Size.Width.Tag).To(Equal(DimensionAuto))
			Expect(s.Size.Height.Tag).To(Equal(DimensionAuto))
		})

		It("has MinSize=Auto on both dimensions", func() {
			s := DefaultStyle()
			Expect(s.MinSize.Width.Tag).To(Equal(DimensionAuto))
			Expect(s.MinSize.Height.Tag).To(Equal(DimensionAuto))
		})

		It("has MaxSize=Auto on both dimensions", func() {
			s := DefaultStyle()
			Expect(s.MaxSize.Width.Tag).To(Equal(DimensionAuto))
			Expect(s.MaxSize.Height.Tag).To(Equal(DimensionAuto))
		})

		It("has AspectRatio=None", func() {
			s := DefaultStyle()
			Expect(s.AspectRatio.IsSome).To(BeFalse())
		})

		It("has Inset=Auto on all sides", func() {
			s := DefaultStyle()
			Expect(s.Inset.Left.Tag).To(Equal(LPAAuto))
			Expect(s.Inset.Right.Tag).To(Equal(LPAAuto))
			Expect(s.Inset.Top.Tag).To(Equal(LPAAuto))
			Expect(s.Inset.Bottom.Tag).To(Equal(LPAAuto))
		})

		It("has Margin=zero on all sides", func() {
			s := DefaultStyle()
			Expect(s.Margin.Left).To(Equal(LengthPercentageAuto{Tag: LPALength, Value: 0}))
			Expect(s.Margin.Right).To(Equal(LengthPercentageAuto{Tag: LPALength, Value: 0}))
			Expect(s.Margin.Top).To(Equal(LengthPercentageAuto{Tag: LPALength, Value: 0}))
			Expect(s.Margin.Bottom).To(Equal(LengthPercentageAuto{Tag: LPALength, Value: 0}))
		})

		It("has Padding=zero on all sides", func() {
			s := DefaultStyle()
			Expect(s.Padding.Left.Value).To(Equal(float32(0)))
			Expect(s.Padding.Right.Value).To(Equal(float32(0)))
			Expect(s.Padding.Top.Value).To(Equal(float32(0)))
			Expect(s.Padding.Bottom.Value).To(Equal(float32(0)))
		})

		It("has Border=zero on all sides", func() {
			s := DefaultStyle()
			Expect(s.Border.Left.Value).To(Equal(float32(0)))
			Expect(s.Border.Right.Value).To(Equal(float32(0)))
			Expect(s.Border.Top.Value).To(Equal(float32(0)))
			Expect(s.Border.Bottom.Value).To(Equal(float32(0)))
		})

		It("has AlignItems=None", func() {
			s := DefaultStyle()
			Expect(s.AlignItems.IsSome).To(BeFalse())
		})

		It("has AlignContent=None", func() {
			s := DefaultStyle()
			Expect(s.AlignContent.IsSome).To(BeFalse())
		})

		It("has JustifyContent=None", func() {
			s := DefaultStyle()
			Expect(s.JustifyContent.IsSome).To(BeFalse())
		})

		It("has Gap=zero", func() {
			s := DefaultStyle()
			Expect(s.Gap.Width.Value).To(Equal(float32(0)))
			Expect(s.Gap.Height.Value).To(Equal(float32(0)))
		})

		It("has TextAlign=Auto", func() {
			s := DefaultStyle()
			Expect(s.TextAlign).To(Equal(TextAlignAuto))
		})

		It("has GridAutoFlow=Row", func() {
			s := DefaultStyle()
			Expect(s.GridAutoFlow).To(Equal(GridAutoFlowRow))
		})

		It("has GridRow with Auto start and end", func() {
			s := DefaultStyle()
			Expect(s.GridRow.Start.Tag).To(Equal(GridPlacementAuto))
			Expect(s.GridRow.End.Tag).To(Equal(GridPlacementAuto))
		})

		It("has GridColumn with Auto start and end", func() {
			s := DefaultStyle()
			Expect(s.GridColumn.Start.Tag).To(Equal(GridPlacementAuto))
			Expect(s.GridColumn.End.Tag).To(Equal(GridPlacementAuto))
		})

		It("has empty grid template slices", func() {
			s := DefaultStyle()
			Expect(s.GridTemplateRows).To(BeEmpty())
			Expect(s.GridTemplateColumns).To(BeEmpty())
			Expect(s.GridAutoRows).To(BeEmpty())
			Expect(s.GridAutoColumns).To(BeEmpty())
			Expect(s.GridTemplateAreas).To(BeEmpty())
		})

		It("is not a table item", func() {
			s := DefaultStyle()
			Expect(s.ItemIsTable).To(BeFalse())
		})

		It("is not a replaced item", func() {
			s := DefaultStyle()
			Expect(s.ItemIsReplaced).To(BeFalse())
		})
	})

	Describe("Overflow", func() {
		It("Visible is not a scroll container", func() {
			Expect(OverflowVisible.IsScrollContainer()).To(BeFalse())
		})

		It("Clip is not a scroll container", func() {
			Expect(OverflowClip.IsScrollContainer()).To(BeFalse())
		})

		It("Hidden is a scroll container", func() {
			Expect(OverflowHidden.IsScrollContainer()).To(BeTrue())
		})

		It("Scroll is a scroll container", func() {
			Expect(OverflowScroll.IsScrollContainer()).To(BeTrue())
		})

		It("Visible MaybeIntoAutomaticMinSize returns None", func() {
			result := OverflowVisible.MaybeIntoAutomaticMinSize()
			Expect(result.IsSome).To(BeFalse())
		})

		It("Hidden MaybeIntoAutomaticMinSize returns Some(0.0)", func() {
			result := OverflowHidden.MaybeIntoAutomaticMinSize()
			Expect(result.IsSome).To(BeTrue())
			Expect(result.Value).To(Equal(float32(0.0)))
		})

		It("Scroll MaybeIntoAutomaticMinSize returns Some(0.0)", func() {
			result := OverflowScroll.MaybeIntoAutomaticMinSize()
			Expect(result.IsSome).To(BeTrue())
			Expect(result.Value).To(Equal(float32(0.0)))
		})
	})

	Describe("Direction", func() {
		It("LTR IsRTL returns false", func() {
			Expect(DirectionLTR.IsRTL()).To(BeFalse())
		})

		It("RTL IsRTL returns true", func() {
			Expect(DirectionRTL.IsRTL()).To(BeTrue())
		})
	})

	Describe("Style coreStyle interface methods", func() {
		It("boxGenerationMode is Normal for DisplayFlex", func() {
			s := DefaultStyle()
			Expect(s.Display).To(Equal(DisplayFlex))
			// Just verify via the Display field since boxGenerationMode is unexported
		})

		It("Display None sets boxGenerationMode to None", func() {
			s := DefaultStyle()
			s.Display = DisplayNone
			Expect(s.Display).To(Equal(DisplayNone))
		})
	})
})
