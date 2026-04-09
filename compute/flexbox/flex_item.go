package flexbox

import (
	"taffy-go/geometry"
	"taffy-go/lib"
	"taffy-go/style"
	"taffy-go/tree"
)

// flexItem holds intermediate results for a single flex item.
type flexItem struct {
	node  tree.NodeId
	order uint32

	size    geometry.Size[lib.Option[float32]]
	minSize geometry.Size[lib.Option[float32]]
	maxSize geometry.Size[lib.Option[float32]]

	alignSelf      style.AlignSelf
	overflow       geometry.Point[style.Overflow]
	scrollbarWidth float32
	flexShrink     float32
	flexGrow       float32

	resolvedMinimumMainSize float32

	inset        geometry.Rect[lib.Option[float32]]
	margin       geometry.Rect[float32]
	marginIsAuto geometry.Rect[bool]
	padding      geometry.Rect[float32]
	border       geometry.Rect[float32]

	flexBasis      float32
	innerFlexBasis float32
	violation      float32
	frozen         bool

	contentFlexFraction float32

	hypotheticalInnerSize geometry.Size[float32]
	hypotheticalOuterSize geometry.Size[float32]
	targetSize            geometry.Size[float32]
	outerTargetSize       geometry.Size[float32]

	baseline float32

	offsetMain  float32
	offsetCross float32
}

// isScrollContainer returns true if the item is a scroll container.
func (fi *flexItem) isScrollContainer() bool {
	return fi.overflow.X.IsScrollContainer() || fi.overflow.Y.IsScrollContainer()
}

func generateAnonymousFlexItems(
	t tree.LayoutFlexboxContainer,
	node tree.NodeId,
	constants *algoConstants,
) []*flexItem {
	calc := t.ResolveCalcValue
	childIds := t.ChildIds(node)
	items := make([]*flexItem, 0, len(childIds))

	for index, child := range childIds {
		childStyle := t.GetFlexboxChildStyle(child)
		childCoreStyle := t.GetCoreContainerStyle(child)
		if childCoreStyle.GetPosition() == style.PositionAbsolute {
			continue
		}
		if childCoreStyle.GetBoxGenerationMode() == style.BoxGenerationModeNone {
			continue
		}

		aspectRatio := childCoreStyle.GetAspectRatio()
		padding := style.ResolveRectLPWidth(childCoreStyle.GetPadding(), constants.nodeInnerSize.Width, calc)
		border := style.ResolveRectLPWidth(childCoreStyle.GetBorder(), constants.nodeInnerSize.Width, calc)
		pbSum := geometry.SizeAddF32(geometry.RectSumAxes(padding), geometry.RectSumAxes(border))
		boxSizingAdj := geometry.SizeZeroF32
		if childCoreStyle.GetBoxSizing() == style.BoxSizingContentBox {
			boxSizingAdj = pbSum
		}

		items = append(items, &flexItem{
			node:  child,
			order: uint32(index),

			size: geometry.SizeMaybeAdd(
				geometry.SizeOptionMaybeApplyAspectRatio(
					style.MaybeResolveSizeDimOption(childCoreStyle.GetSize(), constants.nodeInnerSize, calc),
					aspectRatio,
				),
				geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdj.Width), Height: lib.Some(boxSizingAdj.Height)},
			),
			minSize: geometry.SizeMaybeAdd(
				geometry.SizeOptionMaybeApplyAspectRatio(
					style.MaybeResolveSizeDimOption(childCoreStyle.GetMinSize(), constants.nodeInnerSize, calc),
					aspectRatio,
				),
				geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdj.Width), Height: lib.Some(boxSizingAdj.Height)},
			),
			maxSize: geometry.SizeMaybeAdd(
				geometry.SizeOptionMaybeApplyAspectRatio(
					style.MaybeResolveSizeDimOption(childCoreStyle.GetMaxSize(), constants.nodeInnerSize, calc),
					aspectRatio,
				),
				geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdj.Width), Height: lib.Some(boxSizingAdj.Height)},
			),

			inset: geometry.RectZipSize(childCoreStyle.GetInset(), constants.nodeInnerSize,
				func(p style.LengthPercentageAuto, s lib.Option[float32]) lib.Option[float32] {
					return style.MaybeResolveLPAOption(p, s, calc)
				},
			),
			margin:       style.ResolveRectLPAWidth(childCoreStyle.GetMargin(), constants.nodeInnerSize.Width, calc),
			marginIsAuto: geometry.RectMapTo(childCoreStyle.GetMargin(), func(m style.LengthPercentageAuto) bool { return m.IsAuto() }),
			padding:      padding,
			border:       border,
			alignSelf:    childStyle.GetAlignSelf().UnwrapOr(constants.alignItems),
			overflow:     childCoreStyle.GetOverflow(),
			scrollbarWidth: childCoreStyle.GetScrollbarWidth(),
			flexGrow:     childStyle.GetFlexGrow(),
			flexShrink:   childStyle.GetFlexShrink(),

			flexBasis:      0.0,
			innerFlexBasis: 0.0,
			violation:      0.0,
			frozen:         false,

			resolvedMinimumMainSize: 0.0,
			hypotheticalInnerSize:   geometry.SizeZeroF32,
			hypotheticalOuterSize:   geometry.SizeZeroF32,
			targetSize:             geometry.SizeZeroF32,
			outerTargetSize:        geometry.SizeZeroF32,
			contentFlexFraction:    0.0,

			baseline:    0.0,
			offsetMain:  0.0,
			offsetCross: 0.0,
		})
	}
	return items
}

func determineFlexBaseSize(
	t tree.LayoutFlexboxContainer,
	constants *algoConstants,
	availableSpace geometry.Size[style.AvailableSpace],
	flexItems []*flexItem,
) {
	dir := constants.dir
	calc := t.ResolveCalcValue

	for _, child := range flexItems {
		childStyle := t.GetFlexboxChildStyle(child.node)
		childCoreStyle := t.GetCoreContainerStyle(child.node)

		// Parent size for child sizing.
		crossAxisParentSize := sizeOptionCross(constants.nodeInnerSize, dir)
		childParentSize := sizeFromCross(dir, crossAxisParentSize)

		// Available space for child sizing.
		crossAxisMarginSum := rectCrossAxisSum(constants.margin, dir)
		childMinCross := maybeAddF32IfSome(sizeOptionCross(child.minSize, dir), crossAxisMarginSum)
		childMaxCross := maybeAddF32IfSome(sizeOptionCross(child.maxSize, dir), crossAxisMarginSum)

		// Clamp available space by min- and max- size.
		var crossAxisAvailableSpace style.AvailableSpace
		switch crossAv := avCross(availableSpace, dir); crossAv.Tag {
		case style.AvailableSpaceDefinite:
			val := crossAxisParentSize.UnwrapOr(crossAv.Value)
			crossAxisAvailableSpace = style.AvailableSpaceFromDefinite(geometry.MaybeClampF32(val, childMinCross, childMaxCross))
		case style.AvailableSpaceMinContent:
			if childMinCross.IsSome {
				crossAxisAvailableSpace = style.AvailableSpaceFromDefinite(childMinCross.Value)
			} else {
				crossAxisAvailableSpace = style.AvailableSpaceMinContentValue
			}
		default: // MaxContent
			if childMaxCross.IsSome {
				crossAxisAvailableSpace = style.AvailableSpaceFromDefinite(childMaxCross.Value)
			} else {
				crossAxisAvailableSpace = style.AvailableSpaceMaxContentValue
			}
		}

		// Known dimensions for child sizing.
		childKnownDimensions := sizeOptionWithMain(child.size, dir, lib.None[float32]())
		if child.alignSelf == style.AlignItemsStretch &&
			!rectBoolCrossStart(child.marginIsAuto, constants.dir) &&
			!rectBoolCrossEnd(child.marginIsAuto, constants.dir) &&
			!sizeOptionCross(childKnownDimensions, dir).IsSome {
			sizeSetCross(&childKnownDimensions, dir,
				geometry.MaybeSubOption(crossAxisAvailableSpace.IntoOption(), lib.Some(rectCrossAxisSum(child.margin, dir))),
			)
		}

		containerWidth := sizeOptionMain(constants.nodeInnerSize, dir)
		var boxSizingAdj float32
		if childCoreStyle.GetBoxSizing() == style.BoxSizingContentBox {
			p := style.ResolveRectLPWidth(childCoreStyle.GetPadding(), containerWidth, calc)
			b := style.ResolveRectLPWidth(childCoreStyle.GetBorder(), containerWidth, calc)
			pbS := geometry.SizeAddF32(geometry.RectSumAxes(p), geometry.RectSumAxes(b))
			boxSizingAdj = sizeMain(pbS, dir)
		}
		flexBasis := maybeAddF32IfSome(
			style.MaybeResolveDimOption(childStyle.GetFlexBasis(), containerWidth, calc),
			boxSizingAdj,
		)

		// Determine flex basis.
		mainSize := sizeOptionMain(child.size, dir)
		resolvedBasis := flexBasis.Or(mainSize)
		if resolvedBasis.IsSome {
			child.flexBasis = resolvedBasis.Value
		} else {
			// E. Size the item into the available space.
			childAvailableSpace := avWithCross(
				avWithMain(
					geometry.Size[style.AvailableSpace]{Width: style.AvailableSpaceMaxContentValue, Height: style.AvailableSpaceMaxContentValue},
					dir,
					func() style.AvailableSpace {
						if avMain(availableSpace, dir).Tag == style.AvailableSpaceMinContent {
							return style.AvailableSpaceMinContentValue
						}
						return style.AvailableSpaceMaxContentValue
					}(),
				),
				dir, crossAxisAvailableSpace,
			)

			child.flexBasis = tree.MeasureChildSize(
				t, child.node,
				childKnownDimensions, childParentSize, childAvailableSpace,
				tree.SizingModeContentSize, dir.MainAxis(), geometry.LineFalseBool,
			)
		}

		// Floor flex-basis by padding_border_sum.
		paddingBorderSum := rectMainAxisSum(child.padding, constants.dir) + rectMainAxisSum(child.border, constants.dir)
		if child.flexBasis < paddingBorderSum {
			child.flexBasis = paddingBorderSum
		}

		// Hypothetical main size.
		child.innerFlexBasis = child.flexBasis - rectMainAxisSum(child.padding, constants.dir) - rectMainAxisSum(child.border, constants.dir)

		paddingBorderAxesSums := geometry.Size[lib.Option[float32]]{
			Width:  lib.Some(geometry.RectSumAxes(geometry.RectAddF32(child.padding, child.border)).Width),
			Height: lib.Some(geometry.RectSumAxes(geometry.RectAddF32(child.padding, child.border)).Height),
		}

		// Resolve minimum main size.
		overflowMin := geometry.SizeZipMap(
			geometry.PointToSize(child.overflow),
			geometry.Size[style.Overflow]{Width: style.OverflowVisible, Height: style.OverflowVisible}, // placeholder
			func(o style.Overflow, _ style.Overflow) lib.Option[float32] {
				return o.MaybeIntoAutomaticMinSize()
			},
		)
		styleMinMainSize := sizeOptionMain(
			geometry.SizeOptionOr(child.minSize, overflowMin),
			dir,
		)

		if styleMinMainSize.IsSome {
			child.resolvedMinimumMainSize = styleMinMainSize.Value
		} else {
			// Compute min-content main size.
			childAvailableSpaceMin := avWithCross(
				geometry.Size[style.AvailableSpace]{Width: style.AvailableSpaceMinContentValue, Height: style.AvailableSpaceMinContentValue},
				dir, crossAxisAvailableSpace,
			)
			minContentMainSize := tree.MeasureChildSize(
				t, child.node,
				childKnownDimensions, childParentSize, childAvailableSpaceMin,
				tree.SizingModeContentSize, dir.MainAxis(), geometry.LineFalseBool,
			)

			// 4.5. Automatic Minimum Size of Flex Items
			clampedMinContentSize := geometry.MaybeMinF32(minContentMainSize, sizeOptionMain(child.size, dir))
			clampedMinContentSize = geometry.MaybeMinF32(clampedMinContentSize, sizeOptionMain(child.maxSize, dir))
			child.resolvedMinimumMainSize = geometry.MaybeMaxF32(clampedMinContentSize, sizeOptionMain(paddingBorderAxesSums, dir))
		}

		hypotheticalInnerMinMain := geometry.MaybeMaxF32(child.resolvedMinimumMainSize, sizeOptionMain(paddingBorderAxesSums, constants.dir))
		hypotheticalInnerSize := geometry.MaybeClampF32(child.flexBasis, lib.Some(hypotheticalInnerMinMain), sizeOptionMain(child.maxSize, constants.dir))
		hypotheticalOuterSize := hypotheticalInnerSize + rectMainAxisSum(child.margin, constants.dir)

		sizeSetMain(&child.hypotheticalInnerSize, constants.dir, hypotheticalInnerSize)
		sizeSetMain(&child.hypotheticalOuterSize, constants.dir, hypotheticalOuterSize)
	}
}

// maybeAddF32IfSome adds addend to opt if opt is Some, returns opt unchanged if addend is 0 or opt is None.
func maybeAddF32IfSome(opt lib.Option[float32], addend float32) lib.Option[float32] {
	if !opt.IsSome {
		return opt
	}
	return lib.Some(opt.Value + addend)
}

func determineHypotheticalCrossSize(
	t tree.LayoutFlexboxContainer,
	line *flexLine,
	constants *algoConstants,
	availableSpace geometry.Size[style.AvailableSpace],
) {
	for _, child := range line.items {
		paddingBorderSum := rectCrossAxisSum(geometry.RectAddF32(child.padding, child.border), constants.dir)

		childKnownMain := style.AvailableSpaceFromDefinite(sizeMain(constants.containerSize, constants.dir))

		childCross := geometry.MaybeMaxF32(
			geometry.MaybeClampF32(
				sizeOptionCross(child.size, constants.dir).UnwrapOr(0),
				sizeOptionCross(child.minSize, constants.dir),
				sizeOptionCross(child.maxSize, constants.dir),
			),
			lib.Some(paddingBorderSum),
		)
		// Only use child cross if it was present in child.size.
		var childCrossOpt lib.Option[float32]
		if sizeOptionCross(child.size, constants.dir).IsSome {
			clampedVal := geometry.MaybeClampF32(
				sizeOptionCross(child.size, constants.dir).Value,
				sizeOptionCross(child.minSize, constants.dir),
				sizeOptionCross(child.maxSize, constants.dir),
			)
			childCrossOpt = lib.Some(geometry.F32Max(clampedVal, paddingBorderSum))
		} else {
			childCrossOpt = lib.None[float32]()
		}

		childAvailableCross := asClampOption(
			style.AsMaybeMaxF32(avCross(availableSpace, constants.dir), paddingBorderSum),
			sizeOptionCross(child.minSize, constants.dir),
			sizeOptionCross(child.maxSize, constants.dir),
		)

		var childInnerCross float32
		if childCrossOpt.IsSome {
			childInnerCross = childCrossOpt.Value
		} else {
			knownDims := geometry.Size[lib.Option[float32]]{
				Width: func() lib.Option[float32] {
					if constants.isRow {
						return lib.Some(sizeMain(child.targetSize, constants.dir))
					}
					return childCrossOpt
				}(),
				Height: func() lib.Option[float32] {
					if constants.isRow {
						return childCrossOpt
					}
					return lib.Some(sizeMain(child.targetSize, constants.dir))
				}(),
			}
			avSpace := geometry.Size[style.AvailableSpace]{
				Width: func() style.AvailableSpace {
					if constants.isRow {
						return childKnownMain
					}
					return childAvailableCross
				}(),
				Height: func() style.AvailableSpace {
					if constants.isRow {
						return childAvailableCross
					}
					return childKnownMain
				}(),
			}

			measured := tree.MeasureChildSize(
				t, child.node,
				knownDims,
				constants.nodeInnerSize,
				avSpace,
				tree.SizingModeContentSize,
				constants.dir.CrossAxis(),
				geometry.LineFalseBool,
			)
			childInnerCross = geometry.F32Max(
				geometry.MaybeClampF32(measured, sizeOptionCross(child.minSize, constants.dir), sizeOptionCross(child.maxSize, constants.dir)),
				paddingBorderSum,
			)
		}

		_ = childCross // suppress unused
		childOuterCross := childInnerCross + rectCrossAxisSum(child.margin, constants.dir)
		sizeSetCross(&child.hypotheticalInnerSize, constants.dir, childInnerCross)
		sizeSetCross(&child.hypotheticalOuterSize, constants.dir, childOuterCross)
	}
}

func calculateFlexItem(
	t tree.LayoutFlexboxContainer,
	item *flexItem,
	totalOffsetMain *float32,
	totalOffsetCross float32,
	lineOffsetCross float32,
	totalContentSize *geometry.Size[float32],
	containerSize geometry.Size[float32],
	nodeInnerSize geometry.Size[lib.Option[float32]],
	direction style.FlexDirection,
	layoutDirection style.Direction,
) {
	layoutOutput := tree.PerformChildLayout(
		t, item.node,
		geometry.SizeMapTo(item.targetSize, func(s float32) lib.Option[float32] { return lib.Some(s) }),
		nodeInnerSize,
		geometry.SizeMapTo(containerSize, func(s float32) style.AvailableSpace { return style.AvailableSpaceFromDefinite(s) }),
		tree.SizingModeContentSize,
		geometry.LineFalseBool,
	)
	size := layoutOutput.Size
	contentSize := layoutOutput.ContentSize

	isRTLRow := direction.IsRow() && layoutDirection.IsRTL()
	isRTLColumn := direction.IsColumn() && layoutDirection.IsRTL()

	mainRelativeInset := func() float32 {
		if isRTLRow {
			opt := rectOptionMainEnd(item.inset, direction)
			if !opt.IsSome {
				startOpt := rectOptionMainStart(item.inset, direction)
				if startOpt.IsSome {
					return -startOpt.Value
				}
			}
			return opt.UnwrapOr(0.0)
		}
		opt := rectOptionMainStart(item.inset, direction)
		if !opt.IsSome {
			endOpt := rectOptionMainEnd(item.inset, direction)
			if endOpt.IsSome {
				return -endOpt.Value
			}
		}
		return opt.UnwrapOr(0.0)
	}()

	crossRelativeInset := func() float32 {
		if isRTLColumn {
			opt := rectOptionCrossEnd(item.inset, direction)
			if opt.IsSome {
				return -opt.Value
			}
			return rectOptionCrossStart(item.inset, direction).UnwrapOr(0.0)
		}
		opt := rectOptionCrossStart(item.inset, direction)
		if !opt.IsSome {
			endOpt := rectOptionCrossEnd(item.inset, direction)
			if endOpt.IsSome {
				return -endOpt.Value
			}
		}
		return opt.UnwrapOr(0.0)
	}()

	effectiveLineOffsetCross := lineOffsetCross
	if isRTLColumn {
		effectiveLineOffsetCross = 0.0
	}

	var offsetMain float32
	if isRTLRow {
		offsetMain = *totalOffsetMain - item.offsetMain - rectMainEnd(item.margin, direction) - mainRelativeInset - size.Width
	} else {
		offsetMain = *totalOffsetMain + item.offsetMain + rectMainStart(item.margin, direction) + mainRelativeInset
	}

	offsetCross := totalOffsetCross + item.offsetCross + effectiveLineOffsetCross + rectCrossStart(item.margin, direction) + crossRelativeInset

	if direction.IsRow() {
		baselineOffsetCross := totalOffsetCross + item.offsetCross + effectiveLineOffsetCross + rectCrossStart(item.margin, direction)
		innerBaseline := layoutOutput.FirstBaselines.Y.UnwrapOr(size.Height)
		item.baseline = baselineOffsetCross + innerBaseline
	} else {
		baselineOffsetMain := *totalOffsetMain + item.offsetMain + rectMainStart(item.margin, direction)
		innerBaseline := layoutOutput.FirstBaselines.Y.UnwrapOr(size.Height)
		item.baseline = baselineOffsetMain + innerBaseline
	}

	var location geometry.Point[float32]
	if direction.IsRow() {
		location = geometry.Point[float32]{X: offsetMain, Y: offsetCross}
	} else {
		location = geometry.Point[float32]{X: offsetCross, Y: offsetMain}
	}
	scrollbarSize := geometry.Size[float32]{
		Width: func() float32 {
			if item.overflow.Y == style.OverflowScroll {
				return item.scrollbarWidth
			}
			return 0.0
		}(),
		Height: func() float32 {
			if item.overflow.X == style.OverflowScroll {
				return item.scrollbarWidth
			}
			return 0.0
		}(),
	}

	t.SetUnroundedLayout(item.node, &tree.Layout{
		Order:         item.order,
		Size:          size,
		ContentSize:   contentSize,
		ScrollbarSize: scrollbarSize,
		Location:      location,
		Padding:       item.padding,
		Border:        item.border,
		Margin:        item.margin,
	})

	if isRTLRow {
		*totalOffsetMain -= item.offsetMain + rectMainAxisSum(item.margin, direction) + sizeMain(size, direction)
	} else {
		*totalOffsetMain += item.offsetMain + rectMainAxisSum(item.margin, direction) + sizeMain(size, direction)
	}

	// Content size tracking.
	var contributionLocation geometry.Point[float32]
	if layoutDirection.IsRTL() {
		contributionLocation = geometry.Point[float32]{X: containerSize.Width - (location.X + size.Width), Y: location.Y}
	} else {
		contributionLocation = location
	}
	*totalContentSize = geometry.SizeF32Max(*totalContentSize, computeContentSizeContribution(
		contributionLocation, size, contentSize, item.overflow,
	))
}
