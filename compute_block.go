// Package taffy provides the CSS block layout algorithm.
// This file is a port of compute/block.rs from the Taffy Rust layout engine.
package taffy

import "math"

// ---------------------------------------------------------------------------
// Block Formatting Context
// ---------------------------------------------------------------------------

// blockFormattingContext manages float positioning within a Block Formatting Context.
type blockFormattingContext struct {
	floatCtx *floatContext
}

func newBlockFormattingContext() *blockFormattingContext {
	return &blockFormattingContext{
		floatCtx: newFloatContext(),
	}
}

func (bfc *blockFormattingContext) rootBlockContext() *blockContext {
	return &blockContext{
		bfc:                      bfc,
		yOffset:                  0,
		insets:                   [2]float32{0, 0},
		contentBoxInsets:         [2]float32{0, 0},
		floatContentContribution: 0,
		isRoot:                   true,
	}
}

// ---------------------------------------------------------------------------
// blockContext — context for each individual Block within a BFC
// ---------------------------------------------------------------------------

// blockContext is the context for each individual Block within a Block Formatting Context.
// It contains a reference to the root blockFormattingContext plus block-specific data.
type blockContext struct {
	bfc                      *blockFormattingContext
	yOffset                  float32
	insets                   [2]float32
	contentBoxInsets         [2]float32
	floatContentContribution float32
	isRoot                   bool
}

func (bc *blockContext) subContext(additionalYOffset float32, insets [2]float32) *blockContext {
	newInsets := [2]float32{bc.insets[0] + insets[0], bc.insets[1] + insets[1]}
	return &blockContext{
		bfc:                      bc.bfc,
		yOffset:                  bc.yOffset + additionalYOffset,
		insets:                   newInsets,
		contentBoxInsets:         newInsets,
		floatContentContribution: 0,
		isRoot:                   false,
	}
}

func (bc *blockContext) isBfcRoot() bool {
	return bc.isRoot
}

func (bc *blockContext) setWidth(availableWidth float32) {
	bc.bfc.floatCtx.setWidth(availableWidth)
}

func (bc *blockContext) applyContentBoxInset(contentBoxXInsets [2]float32) {
	bc.contentBoxInsets[0] = bc.insets[0] + contentBoxXInsets[0]
	bc.contentBoxInsets[1] = bc.insets[1] + contentBoxXInsets[1]
}

func (bc *blockContext) hasFloats() bool {
	return bc.bfc.floatCtx.hasFloats()
}

func (bc *blockContext) hasActiveFloats(minY float32) bool {
	return bc.bfc.floatCtx.hasActiveFloats(minY + bc.yOffset)
}

func (bc *blockContext) placeFloatedBox(
	floatedBox Size[float32],
	minY float32,
	direction FloatDirection,
	clear Clear,
) Point[float32] {
	pos := bc.bfc.floatCtx.placeFloatedBox(
		floatedBox,
		minY+bc.yOffset,
		bc.contentBoxInsets,
		direction,
		clear,
	)
	pos.Y -= bc.yOffset
	pos.X -= bc.insets[0]

	bc.floatContentContribution = f32Max(bc.floatContentContribution, pos.Y+floatedBox.Height)

	return pos
}

func (bc *blockContext) findContentSlot(minY float32, clear Clear, after *int) contentSlot {
	slot := bc.bfc.floatCtx.findContentSlot(minY+bc.yOffset, bc.contentBoxInsets, clear, after)
	slot.y -= bc.yOffset
	slot.x -= bc.insets[0]
	return slot
}

func (bc *blockContext) clearedThreshold(clear Clear) (float32, bool) {
	threshold, ok := bc.bfc.floatCtx.clearedThreshold(clear)
	if !ok {
		return 0, false
	}
	return threshold - bc.yOffset, true
}

func (bc *blockContext) addChildFloatedContentHeightContribution(childContribution float32) {
	bc.floatContentContribution = f32Max(bc.floatContentContribution, childContribution)
}

func (bc *blockContext) floatedContentHeightContribution() float32 {
	return bc.floatContentContribution
}

// ---------------------------------------------------------------------------
// blockItem — per-child data for the block layout algorithm
// ---------------------------------------------------------------------------

type blockItem struct {
	nodeID NodeId
	order  uint32

	isTable      bool
	isInSameBfc  bool
	float_       Float
	clear_       Clear

	size    Size[Option[float32]]
	minSize Size[Option[float32]]
	maxSize Size[Option[float32]]

	overflow       Point[Overflow]
	scrollbarWidth float32

	position Position
	inset    Rect[LengthPercentageAuto]
	margin   Rect[LengthPercentageAuto]
	padding  Rect[float32]
	border   Rect[float32]
	paddingBorderSum Size[float32]

	computedSize          Size[float32]
	staticPosition        Point[float32]
	canBeCollapsedThrough bool
}

// ---------------------------------------------------------------------------
// computeBlockLayout — main entry point
// ---------------------------------------------------------------------------

func computeBlockLayout(
	tree layoutBlockContainer,
	nodeID NodeId,
	inputs LayoutInput,
	blockCtx *blockContext,
) LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	parentSize := inputs.ParentSize
	runMode := inputs.RunMode

	style := tree.getBlockContainerStyle(nodeID)
	calc := tree.resolveCalcValue

	overflow := style.overflow()
	isScrollContainer := overflow.X.IsScrollContainer() || overflow.Y.IsScrollContainer()
	aspectRatio := style.aspectRatio()
	padding := resolveRectLPWidth(style.padding(), parentSize.Width, calc)
	border := resolveRectLPWidth(style.border(), parentSize.Width, calc)
	paddingBorderSize := RectSumAxes(RectAddF32(padding, border))

	boxSizingAdjustment := SizeZeroF32
	if style.boxSizing() == BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSize
	}

	minSize := SizeMaybeAdd(
		SizeOptionMaybeApplyAspectRatio(
			MaybeResolveSizeDimOption(style.minSize(), parentSize, calc),
			aspectRatio,
		),
		Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)},
	)
	maxSize := SizeMaybeAdd(
		SizeOptionMaybeApplyAspectRatio(
			MaybeResolveSizeDimOption(style.maxSize(), parentSize, calc),
			aspectRatio,
		),
		Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)},
	)

	var clampedStyleSize Size[Option[float32]]
	if inputs.SizingMode == SizingModeInherentSize {
		clampedStyleSize = SizeMaybeClamp(
			SizeMaybeAdd(
				SizeOptionMaybeApplyAspectRatio(
					MaybeResolveSizeDimOption(style.size(), parentSize, calc),
					aspectRatio,
				),
				Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)},
			),
			minSize,
			maxSize,
		)
	} else {
		clampedStyleSize = SizeOptionNone
	}

	// If both min and max in a given axis are set and max <= min then this determines the size in that axis
	minMaxDefiniteSize := SizeZipMap(minSize, maxSize, func(mn, mx Option[float32]) Option[float32] {
		if mn.IsSome && mx.IsSome && mx.Value <= mn.Value {
			return mn
		}
		return None[float32]()
	})

	styledBasedKnownDimensions := SizeMaybeMax(
		SizeOptionOr(
			SizeOptionOr(
				SizeOptionOr(knownDimensions, minMaxDefiniteSize),
				clampedStyleSize,
			),
			Size[Option[float32]]{Width: Some(paddingBorderSize.Width), Height: Some(paddingBorderSize.Height)},
		),
		Size[Option[float32]]{Width: Some(paddingBorderSize.Width), Height: Some(paddingBorderSize.Height)},
	)

	// Short-circuit if both dimensions known and only computing size
	if runMode == RunModeComputeSize {
		if styledBasedKnownDimensions.Width.IsSome && styledBasedKnownDimensions.Height.IsSome {
			return LayoutOutputFromOuterSize(Size[float32]{
				Width:  styledBasedKnownDimensions.Width.Value,
				Height: styledBasedKnownDimensions.Height.Value,
			})
		}
	}

	// Dispatch to computeBlockInner with either the inherited or a new BFC
	newInputs := LayoutInput{
		KnownDimensions:               styledBasedKnownDimensions,
		ParentSize:                    inputs.ParentSize,
		AvailableSpace:                inputs.AvailableSpace,
		SizingMode:                    inputs.SizingMode,
		Axis:                          inputs.Axis,
		RunMode:                       inputs.RunMode,
		VerticalMarginsAreCollapsible: inputs.VerticalMarginsAreCollapsible,
	}

	if blockCtx != nil && !isScrollContainer {
		return computeBlockInner(tree, nodeID, newInputs, blockCtx)
	}

	rootBfc := newBlockFormattingContext()
	rootCtx := rootBfc.rootBlockContext()
	return computeBlockInner(tree, nodeID, newInputs, rootCtx)
}

// ---------------------------------------------------------------------------
// computeBlockInner — core block layout algorithm
// ---------------------------------------------------------------------------

func computeBlockInner(
	tree layoutBlockContainer,
	nodeID NodeId,
	inputs LayoutInput,
	blockCtx *blockContext,
) LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	parentSize := inputs.ParentSize
	availableSpace := inputs.AvailableSpace
	runMode := inputs.RunMode
	verticalMarginsAreCollapsible := inputs.VerticalMarginsAreCollapsible

	style := tree.getBlockContainerStyle(nodeID)
	calc := tree.resolveCalcValue

	rawPadding := style.padding()
	rawBorder := style.border()
	rawMargin := style.margin()
	aspectRatio := style.aspectRatio()
	padding := resolveRectLPWidth(rawPadding, parentSize.Width, calc)
	border := resolveRectLPWidth(rawBorder, parentSize.Width, calc)
	direction := style.direction()

	// Scrollbar gutters
	scrollbarGutter := func() Rect[float32] {
		offsets := style.overflow()
		// Transposed: scrolling vertically needs horizontal space for scrollbar
		xOverflow := offsets.Y
		yOverflow := offsets.X
		xGutter := float32(0)
		if xOverflow == OverflowScroll {
			xGutter = style.scrollbarWidth()
		}
		yGutter := float32(0)
		if yOverflow == OverflowScroll {
			yGutter = style.scrollbarWidth()
		}
		switch direction {
		case DirectionRTL:
			return Rect[float32]{Top: 0, Left: xGutter, Right: 0, Bottom: yGutter}
		default:
			return Rect[float32]{Top: 0, Left: 0, Right: xGutter, Bottom: yGutter}
		}
	}()

	paddingBorder := RectAddF32(padding, border)
	paddingBorderSize := RectSumAxes(paddingBorder)
	contentBoxInset := RectAddF32(paddingBorder, scrollbarGutter)
	containerContentBoxSize := SizeMaybeSub(knownDimensions, Size[Option[float32]]{
		Width:  Some(RectHorizontalAxisSum(contentBoxInset)),
		Height: Some(RectVerticalAxisSum(contentBoxInset)),
	})

	// Apply content box inset
	blockCtx.applyContentBoxInset([2]float32{contentBoxInset.Left, contentBoxInset.Right})

	boxSizingAdjustment := SizeZeroF32
	if style.boxSizing() == BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSize
	}

	size := SizeMaybeAdd(
		SizeOptionMaybeApplyAspectRatio(
			MaybeResolveSizeDimOption(style.size(), parentSize, calc),
			aspectRatio,
		),
		Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)},
	)
	minSize := SizeMaybeAdd(
		SizeOptionMaybeApplyAspectRatio(
			MaybeResolveSizeDimOption(style.minSize(), parentSize, calc),
			aspectRatio,
		),
		Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)},
	)
	maxSize := SizeMaybeAdd(
		SizeOptionMaybeApplyAspectRatio(
			MaybeResolveSizeDimOption(style.maxSize(), parentSize, calc),
			aspectRatio,
		),
		Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)},
	)

	overflow := style.overflow()
	isScrollContainer := overflow.X.IsScrollContainer() || overflow.Y.IsScrollContainer()

	// Determine margin collapsing behaviour
	ownMarginsCollapseWithChildren := Line[bool]{
		Start: verticalMarginsAreCollapsible.Start &&
			!isScrollContainer &&
			style.position() == PositionRelative &&
			padding.Top == 0 &&
			border.Top == 0,
		End: verticalMarginsAreCollapsible.End &&
			!isScrollContainer &&
			style.position() == PositionRelative &&
			padding.Bottom == 0 &&
			border.Bottom == 0 &&
			!size.Height.IsSome,
	}

	hasStylesPreventingBeingCollapsedThrough := !style.isBlock() ||
		blockCtx.isBfcRoot() ||
		isScrollContainer ||
		style.position() == PositionAbsolute ||
		padding.Top > 0 ||
		padding.Bottom > 0 ||
		border.Top > 0 ||
		border.Bottom > 0 ||
		(size.Height.IsSome && size.Height.Value > 0) ||
		(minSize.Height.IsSome && minSize.Height.Value > 0)

	textAlign := style.textAlign()

	// 1. Generate items
	items := generateBlockItemList(tree, nodeID, containerContentBoxSize)

	// 2. Compute container width
	containerOuterWidth := knownDimensions.Width.UnwrapOr(func() float32 {
		availWidth := AsMaybeSubF32(availableSpace.Width, RectHorizontalAxisSum(contentBoxInset))
		intrinsicWidth := determineContentBasedContainerWidth(tree, items, availWidth) +
			RectHorizontalAxisSum(contentBoxInset)
		return maybeMaxF32(
			maybeClampF32(intrinsicWidth, minSize.Width, maxSize.Width),
			Some(paddingBorderSize.Width),
		)
	}())

	// Short-circuit if computing size and both dimensions known
	if runMode == RunModeComputeSize && knownDimensions.Height.IsSome {
		return LayoutOutputFromOuterSize(Size[float32]{
			Width:  containerOuterWidth,
			Height: knownDimensions.Height.Value,
		})
	}

	containerPercentageResolutionHeight := func() Option[float32] {
		h := knownDimensions.Height
		if h.IsSome {
			return h
		}
		h2 := maybeMaxOption(size.Height, minSize.Height)
		if h2.IsSome {
			return h2
		}
		return minSize.Height
	}()

	// 3. Perform final item layout and return content height
	resolvedPadding := resolveRectLPWidth(rawPadding, Some(containerOuterWidth), calc)
	resolvedBorder := resolveRectLPWidth(rawBorder, Some(containerOuterWidth), calc)
	resolvedContentBoxInset := RectAddF32(RectAddF32(resolvedPadding, resolvedBorder), scrollbarGutter)

	inflowContentSize, intrinsicOuterHeight, firstChildTopMarginSet, lastChildBottomMarginSet :=
		performFinalLayoutOnInFlowChildren(
			tree,
			items,
			containerOuterWidth,
			containerPercentageResolutionHeight,
			contentBoxInset,
			resolvedContentBoxInset,
			textAlign,
			direction,
			ownMarginsCollapseWithChildren,
			blockCtx,
		)

	// Root BFCs contain floats
	if blockCtx.isBfcRoot() || isScrollContainer {
		intrinsicOuterHeight = f32Max(intrinsicOuterHeight, blockCtx.floatedContentHeightContribution())
	}

	containerOuterHeight := maybeMaxF32(
		maybeClampF32(
			knownDimensions.Height.UnwrapOr(intrinsicOuterHeight),
			minSize.Height,
			maxSize.Height,
		),
		Some(paddingBorderSize.Height),
	)
	finalOuterSize := Size[float32]{Width: containerOuterWidth, Height: containerOuterHeight}

	// Short-circuit if computing size
	if runMode == RunModeComputeSize {
		return LayoutOutputFromOuterSize(finalOuterSize)
	}

	// 4. Layout absolutely positioned children
	absolutePositionInset := RectAddF32(resolvedBorder, scrollbarGutter)
	absolutePositionArea := SizeSubF32(finalOuterSize, RectSumAxes(absolutePositionInset))
	absolutePositionOffset := Point[float32]{X: absolutePositionInset.Left, Y: absolutePositionInset.Top}
	absoluteContentSize := performBlockAbsoluteLayout(
		tree,
		items,
		absolutePositionArea,
		absolutePositionOffset,
		direction,
	)

	// 5. Perform hidden layout on hidden children
	childCount := tree.childCount(nodeID)
	for order := 0; order < childCount; order++ {
		child := tree.getChildId(nodeID, order)
		childStyle := tree.getBlockChildStyle(child)
		if childStyle.boxGenerationMode() == BoxGenerationModeNone {
			tree.setUnroundedLayout(child, &Layout{Order: uint32(order)})
			tree.computeChildLayout(child, LayoutInput{
				KnownDimensions:               SizeOptionNone,
				ParentSize:                    SizeOptionNone,
				AvailableSpace:                Size[AvailableSpace]{Width: AvailableSpaceMaxContentValue, Height: AvailableSpaceMaxContentValue},
				SizingMode:                    SizingModeInherentSize,
				Axis:                          RequestedAxisBoth,
				RunMode:                       RunModePerformLayout,
				VerticalMarginsAreCollapsible: LineFalseBool,
			})
		}
	}

	// 6. Compute output margins
	allInFlowChildrenCanBeCollapsedThrough := true
	for _, item := range items {
		if item.position != PositionAbsolute && !item.canBeCollapsedThrough {
			allInFlowChildrenCanBeCollapsedThrough = false
			break
		}
	}
	canBeCollapsedThrough := !hasStylesPreventingBeingCollapsedThrough && allInFlowChildrenCanBeCollapsedThrough

	contentSize := SizeF32Max(inflowContentSize, absoluteContentSize)

	var topMargin CollapsibleMarginSet
	if ownMarginsCollapseWithChildren.Start {
		topMargin = firstChildTopMarginSet
	} else {
		marginTop := ResolveOrZeroLPA(rawMargin.Top, parentSize.Width, calc)
		topMargin = CollapsibleMarginSetFromMargin(marginTop)
	}

	var bottomMargin CollapsibleMarginSet
	if ownMarginsCollapseWithChildren.End {
		bottomMargin = lastChildBottomMarginSet
	} else {
		marginBottom := ResolveOrZeroLPA(rawMargin.Bottom, parentSize.Width, calc)
		bottomMargin = CollapsibleMarginSetFromMargin(marginBottom)
	}

	return LayoutOutput{
		Size:                      finalOuterSize,
		ContentSize:               contentSize,
		FirstBaselines:            PointNoneF32,
		TopMargin:                 topMargin,
		BottomMargin:              bottomMargin,
		MarginsCanCollapseThrough: canBeCollapsedThrough,
	}
}

// ---------------------------------------------------------------------------
// generateBlockItemList — creates block items from children
// ---------------------------------------------------------------------------

func generateBlockItemList(
	tree layoutBlockContainer,
	node NodeId,
	nodeInnerSize Size[Option[float32]],
) []blockItem {
	childIDs := tree.childIds(node)
	var items []blockItem
	order := 0

	for _, childNodeID := range childIDs {
		childStyle := tree.getBlockChildStyle(childNodeID)
		if childStyle.boxGenerationMode() == BoxGenerationModeNone {
			continue
		}

		calc := tree.resolveCalcValue
		aspectRatio := childStyle.aspectRatio()
		padding := resolveRectLPWidth(childStyle.padding(), nodeInnerSize.Width, calc)
		border := resolveRectLPWidth(childStyle.border(), nodeInnerSize.Width, calc)
		pbSum := RectSumAxes(RectAddF32(padding, border))
		boxSizingAdjustment := SizeZeroF32
		if childStyle.boxSizing() == BoxSizingContentBox {
			boxSizingAdjustment = pbSum
		}

		position := childStyle.position()
		childOverflow := childStyle.overflow()
		childFloat := childStyle.float_()
		isNotFloated := childFloat == FloatNone
		isBlock := childStyle.isBlock()
		isTable := childStyle.isTable()
		isScrollContainer := childOverflow.X.IsScrollContainer() || childOverflow.Y.IsScrollContainer()
		isInSameBfc := isBlock && !isTable && position != PositionAbsolute && isNotFloated && !isScrollContainer

		bsa := Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)}

		items = append(items, blockItem{
			nodeID:      childNodeID,
			order:       uint32(order),
			isTable:     isTable,
			isInSameBfc: isInSameBfc,
			float_:      childFloat,
			clear_:      childStyle.clear_(),
			size: SizeMaybeAdd(
				SizeOptionMaybeApplyAspectRatio(
					MaybeResolveSizeDimOption(childStyle.size(), nodeInnerSize, calc),
					aspectRatio,
				),
				bsa,
			),
			minSize: SizeMaybeAdd(
				SizeOptionMaybeApplyAspectRatio(
					MaybeResolveSizeDimOption(childStyle.minSize(), nodeInnerSize, calc),
					aspectRatio,
				),
				bsa,
			),
			maxSize: SizeMaybeAdd(
				SizeOptionMaybeApplyAspectRatio(
					MaybeResolveSizeDimOption(childStyle.maxSize(), nodeInnerSize, calc),
					aspectRatio,
				),
				bsa,
			),
			overflow:         childOverflow,
			scrollbarWidth:   childStyle.scrollbarWidth(),
			position:         position,
			inset:            childStyle.inset(),
			margin:           childStyle.margin(),
			padding:          padding,
			border:           border,
			paddingBorderSum: pbSum,
			computedSize:     SizeZeroF32,
			staticPosition:   PointZeroF32,
		})

		order++
	}

	return items
}

// ---------------------------------------------------------------------------
// determineContentBasedContainerWidth
// ---------------------------------------------------------------------------

func determineContentBasedContainerWidth(
	tree layoutPartialTree,
	items []blockItem,
	availableWidth AvailableSpace,
) float32 {
	calc := tree.resolveCalcValue
	asAvailable := Size[AvailableSpace]{Width: availableWidth, Height: AvailableSpaceMinContentValue}

	maxChildWidth := float32(0)
	floatContribution := newFloatIntrinsicWidthCalculator(availableWidth)

	for i := range items {
		item := &items[i]
		if item.position == PositionAbsolute {
			continue
		}

		knownDimensions := SizeMaybeClamp(item.size, item.minSize, item.maxSize)

		itemXMarginSum := RectHorizontalAxisSum(
			ResolveOrZeroRectLPASizeContext(item.margin, SizeAvailableSpaceIntoOptions(asAvailable), calc),
		)

		width := knownDimensions.Width.UnwrapOr(func() float32 {
			sizeAndBaselines := performChildLayout(
				tree,
				item.nodeID,
				knownDimensions,
				SizeOptionNone,
				Size[AvailableSpace]{
					Width:  AsMaybeSubF32(asAvailable.Width, itemXMarginSum),
					Height: asAvailable.Height,
				},
				SizingModeInherentSize,
				LineTrueBool,
			)
			return sizeAndBaselines.Size.Width
		}())

		width = f32Max(width, item.paddingBorderSum.Width) + itemXMarginSum

		if dir, isFloated := item.float_.FloatDirection(); isFloated {
			floatContribution.addFloat(width, dir, item.clear_)
			continue
		}

		maxChildWidth = f32Max(maxChildWidth, width)
	}

	maxChildWidth = f32Max(maxChildWidth, floatContribution.result())
	return maxChildWidth
}

// ---------------------------------------------------------------------------
// performFinalLayoutOnInFlowChildren
// ---------------------------------------------------------------------------

func performFinalLayoutOnInFlowChildren(
	tree layoutBlockContainer,
	items []blockItem,
	containerOuterWidth float32,
	containerPercentageResolutionHeight Option[float32],
	contentBoxInset Rect[float32],
	resolvedContentBoxInset Rect[float32],
	textAlign TextAlign,
	direction Direction,
	ownMarginsCollapseWithChildren Line[bool],
	blockCtx *blockContext,
) (Size[float32], float32, CollapsibleMarginSet, CollapsibleMarginSet) {
	calc := tree.resolveCalcValue

	containerInnerWidth := containerOuterWidth - RectHorizontalAxisSum(resolvedContentBoxInset)
	containerPctResHeight := MaybeSubOption(containerPercentageResolutionHeight, Some(RectVerticalAxisSum(resolvedContentBoxInset)))
	parentSize := Size[Option[float32]]{Width: Some(containerInnerWidth), Height: containerPctResHeight}
	availableSpace := Size[AvailableSpace]{
		Width:  AvailableSpaceFromDefinite(containerInnerWidth),
		Height: AvailableSpaceMinContentValue,
	}

	if blockCtx.isBfcRoot() {
		blockCtx.setWidth(containerOuterWidth)
		blockCtx.applyContentBoxInset([2]float32{resolvedContentBoxInset.Left, resolvedContentBoxInset.Right})
	}

	inflowContentSize := SizeZeroF32
	committedYOffset := resolvedContentBoxInset.Top
	yOffsetForAbsolute := resolvedContentBoxInset.Top
	firstChildTopMarginSet := zeroCollapsibleMarginSet
	activeCollapsibleMarginSet := zeroCollapsibleMarginSet
	isCollapsingWithFirstMarginSet := true

	hasActiveFloats := blockCtx.hasActiveFloats(committedYOffset)
	yOffsetForFloat := resolvedContentBoxInset.Top

	for i := range items {
		item := &items[i]

		if item.position == PositionAbsolute {
			x := resolvedContentBoxInset.Left
			if direction == DirectionRTL {
				x = containerOuterWidth - resolvedContentBoxInset.Right
			}
			item.staticPosition = Point[float32]{X: x, Y: yOffsetForAbsolute}
			continue
		}

		itemMargin := Rect[Option[float32]]{
			Left:   MaybeResolveLPAF32(item.margin.Left, containerOuterWidth, calc),
			Right:  MaybeResolveLPAF32(item.margin.Right, containerOuterWidth, calc),
			Top:    MaybeResolveLPAF32(item.margin.Top, containerOuterWidth, calc),
			Bottom: MaybeResolveLPAF32(item.margin.Bottom, containerOuterWidth, calc),
		}
		itemNonAutoMargin := Rect[float32]{
			Left:   itemMargin.Left.UnwrapOr(0),
			Right:  itemMargin.Right.UnwrapOr(0),
			Top:    itemMargin.Top.UnwrapOr(0),
			Bottom: itemMargin.Bottom.UnwrapOr(0),
		}
		itemNonAutoXMarginSum := RectHorizontalAxisSum(itemNonAutoMargin)

		scrollbarSize := Size[float32]{
			Width: func() float32 {
				if item.overflow.Y == OverflowScroll {
					return item.scrollbarWidth
				}
				return 0
			}(),
			Height: func() float32 {
				if item.overflow.X == OverflowScroll {
					return item.scrollbarWidth
				}
				return 0
			}(),
		}

		// Handle floated boxes
		if floatDirection, isFloated := item.float_.FloatDirection(); isFloated {
			hasActiveFloats = true

			itemLayout := performChildLayout(
				tree,
				item.nodeID,
				SizeOptionNone,
				parentSize,
				Size[AvailableSpace]{Width: AvailableSpaceMaxContentValue, Height: AvailableSpaceMaxContentValue},
				SizingModeInherentSize,
				LineTrueBool,
			)
			marginBox := SizeAddF32(itemLayout.Size, RectSumAxes(itemNonAutoMargin))

			location := blockCtx.placeFloatedBox(marginBox, yOffsetForFloat, floatDirection, item.clear_)

			// Convert margin-box location to border-box location
			location.Y += itemNonAutoMargin.Top
			location.X += itemNonAutoMargin.Left

			tree.setUnroundedLayout(item.nodeID, &Layout{
				Order:         item.order,
				Size:          itemLayout.Size,
				ContentSize:   itemLayout.ContentSize,
				ScrollbarSize: scrollbarSize,
				Location:      location,
				Padding:       item.padding,
				Border:        item.border,
				Margin:        itemNonAutoMargin,
			})

			inflowContentSize = SizeF32Max(inflowContentSize, computeContentSizeContribution(
				location,
				itemLayout.Size,
				itemLayout.ContentSize,
				item.overflow,
			))

			continue
		}

		// Handle non-floated boxes
		var yMarginOffset float32

		var stretchWidth float32
		var floatAvoidingPosition Point[float32]
		var floatAvoidingWidth float32

		if item.isInSameBfc {
			stretchWidth = containerInnerWidth - itemNonAutoXMarginSum
			floatAvoidingPosition = PointZeroF32
			floatAvoidingWidth = 0
		} else {
			// Set y_margin_offset for different-bfc child
			if !isCollapsingWithFirstMarginSet || !ownMarginsCollapseWithChildren.Start {
				yMarginOffset = activeCollapsibleMarginSet.CollapseWithMargin(itemNonAutoMargin.Top).Resolve()
			}
			minY := committedYOffset + yMarginOffset

			if hasActiveFloats {
				slot := blockCtx.findContentSlot(minY, item.clear_, nil)
				hasActiveFloats = slot.segmentID != nil
				stretchWidth = slot.width - itemNonAutoXMarginSum
				floatAvoidingPosition = Point[float32]{X: slot.x, Y: slot.y}
				floatAvoidingWidth = slot.width
			} else {
				stretchWidth = containerInnerWidth - itemNonAutoXMarginSum
				floatAvoidingPosition = Point[float32]{X: resolvedContentBoxInset.Left, Y: minY}
				floatAvoidingWidth = containerInnerWidth
			}
		}

		var knownDimensions Size[Option[float32]]
		if item.isTable {
			knownDimensions = SizeOptionNone
		} else {
			w := item.size.Width.UnwrapOr(stretchWidth)
			w = maybeClampF32(w, item.minSize.Width, item.maxSize.Width)
			knownDimensions = SizeMaybeClamp(
				Size[Option[float32]]{Width: Some(w), Height: item.size.Height},
				item.minSize,
				item.maxSize,
			)
		}

		layoutInputs := LayoutInput{
			RunMode:    RunModePerformLayout,
			SizingMode: SizingModeInherentSize,
			Axis:       RequestedAxisBoth,
			KnownDimensions: knownDimensions,
			ParentSize:      parentSize,
			AvailableSpace: Size[AvailableSpace]{
				Width:  AvailableSpaceFromDefinite(stretchWidth),
				Height: availableSpace.Height,
			},
			VerticalMarginsAreCollapsible: func() Line[bool] {
				if item.isInSameBfc {
					return LineTrueBool
				}
				return LineFalseBool
			}(),
		}

		clearPos := float32(0)
		if cp, ok := blockCtx.clearedThreshold(item.clear_); ok {
			clearPos = cp
		}

		var itemLayout LayoutOutput
		if item.isInSameBfc {
			width := knownDimensions.Width.Value // same-bfc child always has defined width

			insetLeft := itemNonAutoMargin.Left + contentBoxInset.Left
			insetRight := containerOuterWidth - width - insetLeft
			insets := [2]float32{insetLeft, insetRight}

			childBlockCtx := blockCtx.subContext(
				f32Max(yOffsetForAbsolute+itemNonAutoMargin.Top, clearPos),
				insets,
			)
			output := tree.computeBlockChildLayout(item.nodeID, layoutInputs, childBlockCtx)

			childContribution := childBlockCtx.floatedContentHeightContribution()
			blockCtx.addChildFloatedContentHeightContribution(yOffsetForAbsolute + childContribution)

			itemLayout = output
		} else {
			itemLayout = tree.computeChildLayout(item.nodeID, layoutInputs)
		}

		finalSize := itemLayout.Size

		topMarginSet := itemLayout.TopMargin.CollapseWithMargin(itemMargin.Top.UnwrapOr(0))
		bottomMarginSet := itemLayout.BottomMargin.CollapseWithMargin(itemMargin.Bottom.UnwrapOr(0))

		// Expand auto margins to fill available space
		freeXSpace := f32Max(0, stretchWidth-finalSize.Width)
		xAxisAutoMarginSize := float32(0)
		autoMarginCount := 0
		if !itemMargin.Left.IsSome {
			autoMarginCount++
		}
		if !itemMargin.Right.IsSome {
			autoMarginCount++
		}
		if autoMarginCount > 0 {
			xAxisAutoMarginSize = freeXSpace / float32(autoMarginCount)
		}

		resolvedMargin := Rect[float32]{
			Left:   itemMargin.Left.UnwrapOr(xAxisAutoMarginSize),
			Right:  itemMargin.Right.UnwrapOr(xAxisAutoMarginSize),
			Top:    topMarginSet.Resolve(),
			Bottom: bottomMarginSet.Resolve(),
		}

		// Resolve item inset
		insetLeft := MaybeResolveLPAF32(item.inset.Left, containerInnerWidth, calc)
		insetRight := MaybeResolveLPAF32(item.inset.Right, containerInnerWidth, calc)
		insetTop := MaybeResolveLPAF32(item.inset.Top, 0, calc)
		insetBottom := MaybeResolveLPAF32(item.inset.Bottom, 0, calc)

		var insetOffsetX float32
		if direction.IsRTL() {
			if insetRight.IsSome {
				insetOffsetX = -insetRight.Value
			} else if insetLeft.IsSome {
				insetOffsetX = insetLeft.Value
			}
		} else {
			if insetLeft.IsSome {
				insetOffsetX = insetLeft.Value
			} else if insetRight.IsSome {
				insetOffsetX = -insetRight.Value
			}
		}

		var insetOffsetY float32
		if insetTop.IsSome {
			insetOffsetY = insetTop.Value
		} else if insetBottom.IsSome {
			insetOffsetY = -insetBottom.Value
		}

		insetOffset := Point[float32]{X: insetOffsetX, Y: insetOffsetY}

		// Set y_margin_offset for same bfc child
		if item.isInSameBfc &&
			(!isCollapsingWithFirstMarginSet || !ownMarginsCollapseWithChildren.Start) {
			yMarginOffset = activeCollapsibleMarginSet.CollapseWithMargin(resolvedMargin.Top).Resolve()
		}

		floatOrNotClear := item.float_.IsFloated() || item.clear_ == ClearNone

		item.computedSize = itemLayout.Size
		item.canBeCollapsedThrough = itemLayout.MarginsCanCollapseThrough && floatOrNotClear

		if item.isInSameBfc {
			unclearedY := committedYOffset + activeCollapsibleMarginSet.Resolve()
			x := resolvedContentBoxInset.Left
			if direction == DirectionRTL {
				x = containerOuterWidth - resolvedContentBoxInset.Right - finalSize.Width
			}
			item.staticPosition = Point[float32]{
				X: x,
				Y: f32Max(unclearedY, clearPos),
			}
		} else {
			x := floatAvoidingPosition.X
			if direction == DirectionRTL {
				x = floatAvoidingPosition.X + floatAvoidingWidth - finalSize.Width
			}
			item.staticPosition = Point[float32]{
				X: x,
				Y: floatAvoidingPosition.Y,
			}
		}

		var location Point[float32]
		if item.isInSameBfc {
			x := resolvedContentBoxInset.Left + insetOffset.X + resolvedMargin.Left
			if direction == DirectionRTL {
				x = containerOuterWidth - resolvedContentBoxInset.Right - finalSize.Width - resolvedMargin.Right + insetOffset.X
			}
			location = Point[float32]{
				X: x,
				Y: f32Max(committedYOffset, clearPos) + yMarginOffset + insetOffset.Y,
			}
		} else {
			x := floatAvoidingPosition.X + resolvedMargin.Left + insetOffset.X
			if direction == DirectionRTL {
				x = floatAvoidingPosition.X + floatAvoidingWidth - finalSize.Width - resolvedMargin.Right + insetOffset.X
			}
			location = Point[float32]{
				X: x,
				Y: floatAvoidingPosition.Y + insetOffset.Y,
			}
		}

		// Apply text alignment
		itemOuterWidth := itemLayout.Size.Width + RectHorizontalAxisSum(resolvedMargin)
		if itemOuterWidth < containerInnerWidth {
			freeSpace := containerInnerWidth - itemOuterWidth
			switch {
			case textAlign == TextAlignAuto:
				// Do nothing
			case textAlign == TextAlignLegacyLeft && direction == DirectionLTR:
				// Do nothing
			case textAlign == TextAlignLegacyLeft && direction == DirectionRTL:
				location.X -= freeSpace
			case textAlign == TextAlignLegacyRight && direction == DirectionLTR:
				location.X += freeSpace
			case textAlign == TextAlignLegacyRight && direction == DirectionRTL:
				// Do nothing
			case textAlign == TextAlignLegacyCenter && direction == DirectionLTR:
				location.X += freeSpace / 2
			case textAlign == TextAlignLegacyCenter && direction == DirectionRTL:
				location.X -= freeSpace / 2
			}
		}

		tree.setUnroundedLayout(item.nodeID, &Layout{
			Order:         item.order,
			Size:          itemLayout.Size,
			ContentSize:   itemLayout.ContentSize,
			ScrollbarSize: scrollbarSize,
			Location:      location,
			Padding:       item.padding,
			Border:        item.border,
			Margin:        resolvedMargin,
		})

		inflowContentSize = SizeF32Max(inflowContentSize, computeContentSizeContribution(
			Point[float32]{
				X: location.X - resolvedContentBoxInset.Left,
				Y: location.Y - resolvedContentBoxInset.Top,
			},
			finalSize,
			itemLayout.ContentSize,
			item.overflow,
		))

		// Update first_child_top_margin_set
		if isCollapsingWithFirstMarginSet {
			if item.canBeCollapsedThrough {
				firstChildTopMarginSet = firstChildTopMarginSet.
					CollapseWithSet(topMarginSet).
					CollapseWithSet(bottomMarginSet)
			} else {
				firstChildTopMarginSet = firstChildTopMarginSet.CollapseWithSet(topMarginSet)
				isCollapsingWithFirstMarginSet = false
			}
		}

		// Update active_collapsible_margin_set
		if item.canBeCollapsedThrough {
			activeCollapsibleMarginSet = activeCollapsibleMarginSet.
				CollapseWithSet(topMarginSet).
				CollapseWithSet(bottomMarginSet)
			yOffsetForAbsolute = committedYOffset + itemLayout.Size.Height + yMarginOffset
			yOffsetForFloat = committedYOffset + itemLayout.Size.Height + yMarginOffset
		} else {
			committedYOffset = location.Y - insetOffset.Y + itemLayout.Size.Height
			activeCollapsibleMarginSet = bottomMarginSet
			yOffsetForAbsolute = committedYOffset + activeCollapsibleMarginSet.Resolve()
			yOffsetForFloat = committedYOffset
		}
	}

	lastChildBottomMarginSet := activeCollapsibleMarginSet
	bottomYMarginOffset := float32(0)
	if !ownMarginsCollapseWithChildren.End {
		bottomYMarginOffset = lastChildBottomMarginSet.Resolve()
	}

	committedYOffset += resolvedContentBoxInset.Bottom + bottomYMarginOffset
	contentHeight := f32Max(0, committedYOffset)

	_ = yOffsetForFloat // suppress unused warning

	return inflowContentSize, contentHeight, firstChildTopMarginSet, lastChildBottomMarginSet
}

// ---------------------------------------------------------------------------
// performBlockAbsoluteLayout
// ---------------------------------------------------------------------------

func performBlockAbsoluteLayout(
	tree layoutBlockContainer,
	items []blockItem,
	areaSize Size[float32],
	areaOffset Point[float32],
	direction Direction,
) Size[float32] {
	calc := tree.resolveCalcValue
	areaWidth := areaSize.Width
	areaHeight := areaSize.Height

	absoluteContentSize := SizeZeroF32

	for i := range items {
		item := &items[i]
		if item.position != PositionAbsolute {
			continue
		}

		childStyle := tree.getBlockChildStyle(item.nodeID)
		if childStyle.boxGenerationMode() == BoxGenerationModeNone || childStyle.position() != PositionAbsolute {
			continue
		}

		aspectRatio := childStyle.aspectRatio()
		margin := Rect[Option[float32]]{
			Left:   MaybeResolveLPAF32(childStyle.margin().Left, areaWidth, calc),
			Right:  MaybeResolveLPAF32(childStyle.margin().Right, areaWidth, calc),
			Top:    MaybeResolveLPAF32(childStyle.margin().Top, areaWidth, calc),
			Bottom: MaybeResolveLPAF32(childStyle.margin().Bottom, areaWidth, calc),
		}
		padding := resolveRectLPWidth(childStyle.padding(), Some(areaWidth), calc)
		border := resolveRectLPWidth(childStyle.border(), Some(areaWidth), calc)
		paddingBorderSum := RectSumAxes(RectAddF32(padding, border))
		boxSizingAdjustment := SizeZeroF32
		if childStyle.boxSizing() == BoxSizingContentBox {
			boxSizingAdjustment = paddingBorderSum
		}

		// Resolve inset
		left := MaybeResolveLPAF32(childStyle.inset().Left, areaWidth, calc)
		right := MaybeResolveLPAF32(childStyle.inset().Right, areaWidth, calc)
		top := MaybeResolveLPAF32(childStyle.inset().Top, areaHeight, calc)
		bottom := MaybeResolveLPAF32(childStyle.inset().Bottom, areaHeight, calc)

		bsa := Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)}
		areaSizeOpt := Size[Option[float32]]{Width: Some(areaSize.Width), Height: Some(areaSize.Height)}

		// Compute known dimensions from size styles
		styleSize := SizeMaybeAdd(
			SizeOptionMaybeApplyAspectRatio(
				MaybeResolveSizeDimOption(childStyle.size(), areaSizeOpt, calc),
				aspectRatio,
			),
			bsa,
		)
		minSize := SizeMaybeMax(
			SizeOptionOr(
				SizeMaybeAdd(
					SizeOptionMaybeApplyAspectRatio(
						MaybeResolveSizeDimOption(childStyle.minSize(), areaSizeOpt, calc),
						aspectRatio,
					),
					bsa,
				),
				Size[Option[float32]]{Width: Some(paddingBorderSum.Width), Height: Some(paddingBorderSum.Height)},
			),
			Size[Option[float32]]{Width: Some(paddingBorderSum.Width), Height: Some(paddingBorderSum.Height)},
		)
		maxSize := SizeMaybeAdd(
			SizeOptionMaybeApplyAspectRatio(
				MaybeResolveSizeDimOption(childStyle.maxSize(), areaSizeOpt, calc),
				aspectRatio,
			),
			bsa,
		)
		knownDimensions := SizeMaybeClamp(styleSize, minSize, maxSize)

		// Fill in width from left/right
		if !knownDimensions.Width.IsSome && left.IsSome && right.IsSome {
			newWidthRaw := maybeSubF32(maybeSubF32(areaWidth, margin.Left), margin.Right) - left.Value - right.Value
			knownDimensions.Width = Some(f32Max(newWidthRaw, 0))
			knownDimensions = SizeMaybeClamp(SizeOptionMaybeApplyAspectRatio(knownDimensions, aspectRatio), minSize, maxSize)
		}

		// Fill in height from top/bottom
		if !knownDimensions.Height.IsSome && top.IsSome && bottom.IsSome {
			newHeightRaw := maybeSubF32(maybeSubF32(areaHeight, margin.Top), margin.Bottom) - top.Value - bottom.Value
			knownDimensions.Height = Some(f32Max(newHeightRaw, 0))
			knownDimensions = SizeMaybeClamp(SizeOptionMaybeApplyAspectRatio(knownDimensions, aspectRatio), minSize, maxSize)
		}

		measuredSize := measureChildSizeBoth(
			tree,
			item.nodeID,
			knownDimensions,
			areaSizeOpt,
			Size[AvailableSpace]{
				Width:  AvailableSpaceFromDefinite(maybeClampF32(areaWidth, minSize.Width, maxSize.Width)),
				Height: AvailableSpaceFromDefinite(maybeClampF32(areaHeight, minSize.Height, maxSize.Height)),
			},
			SizingModeContentSize,
			LineFalseBool,
		)

		finalSize := Size[float32]{
			Width:  maybeClampF32(knownDimensions.Width.UnwrapOr(measuredSize.Width), minSize.Width, maxSize.Width),
			Height: maybeClampF32(knownDimensions.Height.UnwrapOr(measuredSize.Height), minSize.Height, maxSize.Height),
		}

		layoutOutput := performChildLayout(
			tree,
			item.nodeID,
			Size[Option[float32]]{Width: Some(finalSize.Width), Height: Some(finalSize.Height)},
			areaSizeOpt,
			Size[AvailableSpace]{
				Width:  AvailableSpaceFromDefinite(maybeClampF32(areaWidth, minSize.Width, maxSize.Width)),
				Height: AvailableSpaceFromDefinite(maybeClampF32(areaHeight, minSize.Height, maxSize.Height)),
			},
			SizingModeContentSize,
			LineFalseBool,
		)

		nonAutoMargin := Rect[float32]{
			Left: func() float32 {
				if left.IsSome {
					return margin.Left.UnwrapOr(0)
				}
				return 0
			}(),
			Right: func() float32 {
				if right.IsSome {
					return margin.Right.UnwrapOr(0)
				}
				return 0
			}(),
			Top: func() float32 {
				if top.IsSome {
					return margin.Top.UnwrapOr(0)
				}
				return 0
			}(),
			Bottom: func() float32 {
				if bottom.IsSome {
					return margin.Bottom.UnwrapOr(0)
				}
				return 0
			}(),
		}

		// Compute auto margins for absolutely positioned elements
		absoluteAutoMarginSpace := Point[float32]{
			X: func() float32 {
				if right.IsSome {
					return areaSize.Width - right.Value - left.UnwrapOr(0)
				}
				return finalSize.Width
			}(),
			Y: func() float32 {
				if bottom.IsSome {
					return areaSize.Height - bottom.Value - top.UnwrapOr(0)
				}
				return finalSize.Height
			}(),
		}
		freeSpace := Size[float32]{
			Width:  absoluteAutoMarginSpace.X - finalSize.Width - RectHorizontalAxisSum(nonAutoMargin),
			Height: absoluteAutoMarginSpace.Y - finalSize.Height - RectVerticalAxisSum(nonAutoMargin),
		}

		autoMarginWidth := float32(0)
		{
			count := 0
			if !margin.Left.IsSome {
				count++
			}
			if !margin.Right.IsSome {
				count++
			}
			if count == 2 && (!styleSize.Width.IsSome || styleSize.Width.Value >= freeSpace.Width) {
				autoMarginWidth = 0
			} else if count > 0 {
				autoMarginWidth = freeSpace.Width / float32(count)
			}
		}

		autoMarginHeight := float32(0)
		{
			count := 0
			if !margin.Top.IsSome {
				count++
			}
			if !margin.Bottom.IsSome {
				count++
			}
			if count == 2 && (!styleSize.Height.IsSome || styleSize.Height.Value >= freeSpace.Height) {
				autoMarginHeight = 0
			} else if count > 0 {
				autoMarginHeight = freeSpace.Height / float32(count)
			}
		}

		autoMargin := Rect[float32]{
			Left: func() float32 {
				if margin.Left.IsSome {
					return 0
				}
				return autoMarginWidth
			}(),
			Right: func() float32 {
				if margin.Right.IsSome {
					return 0
				}
				return autoMarginWidth
			}(),
			Top: func() float32 {
				if margin.Top.IsSome {
					return 0
				}
				return autoMarginHeight
			}(),
			Bottom: func() float32 {
				if margin.Bottom.IsSome {
					return 0
				}
				return autoMarginHeight
			}(),
		}

		resolvedMargin := Rect[float32]{
			Left:   margin.Left.UnwrapOr(autoMargin.Left),
			Right:  margin.Right.UnwrapOr(autoMargin.Right),
			Top:    margin.Top.UnwrapOr(autoMargin.Top),
			Bottom: margin.Bottom.UnwrapOr(autoMargin.Bottom),
		}

		var xOffset float32
		if left.IsSome && right.IsSome {
			if direction.IsRTL() {
				xOffset = areaSize.Width - finalSize.Width - right.Value - resolvedMargin.Right
			} else {
				xOffset = left.Value + resolvedMargin.Left
			}
		} else if left.IsSome {
			xOffset = left.Value + resolvedMargin.Left
		} else if right.IsSome {
			xOffset = areaSize.Width - finalSize.Width - right.Value - resolvedMargin.Right
		} else {
			if direction.IsRTL() {
				xOffset = item.staticPosition.X - finalSize.Width - resolvedMargin.Right - areaOffset.X
			} else {
				xOffset = item.staticPosition.X + resolvedMargin.Left - areaOffset.X
			}
		}

		var yPos float32
		if top.IsSome {
			yPos = top.Value + resolvedMargin.Top + areaOffset.Y
		} else if bottom.IsSome {
			yPos = areaSize.Height - finalSize.Height - bottom.Value - resolvedMargin.Bottom + areaOffset.Y
		} else {
			yPos = item.staticPosition.Y + resolvedMargin.Top
		}

		location := Point[float32]{
			X: xOffset + areaOffset.X,
			Y: yPos,
		}

		scrollbarSize := Size[float32]{
			Width: func() float32 {
				if item.overflow.Y == OverflowScroll {
					return item.scrollbarWidth
				}
				return 0
			}(),
			Height: func() float32 {
				if item.overflow.X == OverflowScroll {
					return item.scrollbarWidth
				}
				return 0
			}(),
		}

		tree.setUnroundedLayout(item.nodeID, &Layout{
			Order:         item.order,
			Size:          finalSize,
			ContentSize:   layoutOutput.ContentSize,
			ScrollbarSize: scrollbarSize,
			Location:      location,
			Padding:       padding,
			Border:        border,
			Margin:        resolvedMargin,
		})

		relativeLocation := Point[float32]{
			X: location.X - areaOffset.X,
			Y: location.Y - areaOffset.Y,
		}
		absoluteContentSize = SizeF32Max(absoluteContentSize, computeContentSizeContribution(
			relativeLocation,
			finalSize,
			layoutOutput.ContentSize,
			item.overflow,
		))
	}

	return absoluteContentSize
}

// suppress unused import
var _ = math.Inf
