// Package block provides the CSS block layout algorithm.
// This file is a port of compute/block.rs from the Taffy Rust layout engine.
package block

import (
	"github.com/gataengine/taffy-go/lib"
	"math"

	"github.com/gataengine/taffy-go/geometry"
	"github.com/gataengine/taffy-go/style"
	"github.com/gataengine/taffy-go/tree"
)

// ---------------------------------------------------------------------------
// Block Formatting Context
// ---------------------------------------------------------------------------

// BlockFormattingContext manages float positioning within a Block Formatting Context.
type BlockFormattingContext struct {
	floatCtx *floatContext
}

// NewBlockFormattingContext creates a new BlockFormattingContext.
func NewBlockFormattingContext() *BlockFormattingContext {
	return &BlockFormattingContext{
		floatCtx: newFloatContext(),
	}
}

// RootBlockContext returns the root BlockContext for this BFC.
func (bfc *BlockFormattingContext) RootBlockContext() *BlockContext {
	return &BlockContext{
		bfc:                      bfc,
		yOffset:                  0,
		insets:                   [2]float32{0, 0},
		contentBoxInsets:         [2]float32{0, 0},
		floatContentContribution: 0,
		isRoot:                   true,
	}
}

// ---------------------------------------------------------------------------
// BlockContext — context for each individual Block within a BFC
// ---------------------------------------------------------------------------

// BlockContext is the context for each individual Block within a Block Formatting Context.
// It contains a reference to the root BlockFormattingContext plus block-specific data.
type BlockContext struct {
	bfc                      *BlockFormattingContext
	yOffset                  float32
	insets                   [2]float32
	contentBoxInsets         [2]float32
	floatContentContribution float32
	isRoot                   bool
}

// SubContext creates a child BlockContext for a nested block.
func (bc *BlockContext) SubContext(additionalYOffset float32, insets [2]float32) *BlockContext {
	newInsets := [2]float32{bc.insets[0] + insets[0], bc.insets[1] + insets[1]}
	return &BlockContext{
		bfc:                      bc.bfc,
		yOffset:                  bc.yOffset + additionalYOffset,
		insets:                   newInsets,
		contentBoxInsets:         newInsets,
		floatContentContribution: 0,
		isRoot:                   false,
	}
}

func (bc *BlockContext) isBfcRoot() bool {
	return bc.isRoot
}

func (bc *BlockContext) setWidth(availableWidth float32) {
	bc.bfc.floatCtx.setWidth(availableWidth)
}

func (bc *BlockContext) applyContentBoxInset(contentBoxXInsets [2]float32) {
	bc.contentBoxInsets[0] = bc.insets[0] + contentBoxXInsets[0]
	bc.contentBoxInsets[1] = bc.insets[1] + contentBoxXInsets[1]
}

func (bc *BlockContext) hasFloats() bool {
	return bc.bfc.floatCtx.hasFloats()
}

func (bc *BlockContext) hasActiveFloats(minY float32) bool {
	return bc.bfc.floatCtx.hasActiveFloats(minY + bc.yOffset)
}

func (bc *BlockContext) placeFloatedBox(
	floatedBox geometry.Size[float32],
	minY float32,
	direction style.FloatDirection,
	clear style.Clear,
) geometry.Point[float32] {
	pos := bc.bfc.floatCtx.placeFloatedBox(
		floatedBox,
		minY+bc.yOffset,
		bc.contentBoxInsets,
		direction,
		clear,
	)
	pos.Y -= bc.yOffset
	pos.X -= bc.insets[0]

	bc.floatContentContribution = geometry.F32Max(bc.floatContentContribution, pos.Y+floatedBox.Height)

	return pos
}

func (bc *BlockContext) findContentSlot(minY float32, clear style.Clear, after *int) contentSlot {
	slot := bc.bfc.floatCtx.findContentSlot(minY+bc.yOffset, bc.contentBoxInsets, clear, after)
	slot.y -= bc.yOffset
	slot.x -= bc.insets[0]
	return slot
}

func (bc *BlockContext) clearedThreshold(clear style.Clear) (float32, bool) {
	threshold, ok := bc.bfc.floatCtx.clearedThreshold(clear)
	if !ok {
		return 0, false
	}
	return threshold - bc.yOffset, true
}

// AddChildFloatedContentHeightContribution adds a child's float content height contribution.
func (bc *BlockContext) AddChildFloatedContentHeightContribution(childContribution float32) {
	bc.floatContentContribution = geometry.F32Max(bc.floatContentContribution, childContribution)
}

// FloatedContentHeightContribution returns the float content height contribution.
func (bc *BlockContext) FloatedContentHeightContribution() float32 {
	return bc.floatContentContribution
}

// ---------------------------------------------------------------------------
// blockItem — per-child data for the block layout algorithm
// ---------------------------------------------------------------------------

type blockItem struct {
	nodeID tree.NodeId
	order  uint32

	isTable      bool
	isInSameBfc  bool
	float_       style.Float
	clear_       style.Clear

	size    geometry.Size[lib.Option[float32]]
	minSize geometry.Size[lib.Option[float32]]
	maxSize geometry.Size[lib.Option[float32]]

	overflow       geometry.Point[style.Overflow]
	scrollbarWidth float32

	position style.Position
	inset    geometry.Rect[style.LengthPercentageAuto]
	margin   geometry.Rect[style.LengthPercentageAuto]
	padding  geometry.Rect[float32]
	border   geometry.Rect[float32]
	paddingBorderSum geometry.Size[float32]

	computedSize          geometry.Size[float32]
	staticPosition        geometry.Point[float32]
	canBeCollapsedThrough bool
}

// ---------------------------------------------------------------------------
// computeContentSizeContribution — local helper
// ---------------------------------------------------------------------------

func computeContentSizeContribution(
	location geometry.Point[float32],
	size geometry.Size[float32],
	contentSize geometry.Size[float32],
	overflow geometry.Point[style.Overflow],
) geometry.Size[float32] {
	sizeContrib := geometry.Size[float32]{
		Width: func() float32 {
			if overflow.X == style.OverflowVisible {
				return geometry.F32Max(size.Width, contentSize.Width)
			}
			return size.Width
		}(),
		Height: func() float32 {
			if overflow.Y == style.OverflowVisible {
				return geometry.F32Max(size.Height, contentSize.Height)
			}
			return size.Height
		}(),
	}
	if sizeContrib.Width > 0 && sizeContrib.Height > 0 {
		maxX := geometry.F32Max(location.X+sizeContrib.Width, 0.0)
		minX := geometry.F32Min(location.X, 0.0)
		maxY := geometry.F32Max(location.Y+sizeContrib.Height, 0.0)
		minY := geometry.F32Min(location.Y, 0.0)
		return geometry.Size[float32]{Width: maxX - minX, Height: maxY - minY}
	}
	return geometry.SizeZeroF32
}

// measureChildSizeBoth computes both dimensions of a child (RunMode::ComputeSize + RequestedAxis::Both).
func measureChildSizeBoth(
	t tree.LayoutPartialTree,
	nodeId tree.NodeId,
	knownDimensions geometry.Size[lib.Option[float32]],
	parentSize geometry.Size[lib.Option[float32]],
	availableSpace geometry.Size[style.AvailableSpace],
	sizingMode tree.SizingMode,
	verticalMarginsAreCollapsible geometry.Line[bool],
) geometry.Size[float32] {
	return t.ComputeChildLayout(nodeId, tree.LayoutInput{
		KnownDimensions:               knownDimensions,
		ParentSize:                    parentSize,
		AvailableSpace:                availableSpace,
		SizingMode:                    sizingMode,
		Axis:                          tree.RequestedAxisBoth,
		RunMode:                       tree.RunModeComputeSize,
		VerticalMarginsAreCollapsible: verticalMarginsAreCollapsible,
	}).Size
}

// ---------------------------------------------------------------------------
// ComputeBlockLayout — main entry point
// ---------------------------------------------------------------------------

// ComputeBlockLayout computes the block layout for the given node.
func ComputeBlockLayout(
	t tree.LayoutBlockContainer,
	nodeID tree.NodeId,
	inputs tree.LayoutInput,
	blockCtx any,
) tree.LayoutOutput {
	var bc *BlockContext
	if blockCtx != nil {
		bc = blockCtx.(*BlockContext)
	}
	return computeBlockLayout(t, nodeID, inputs, bc)
}

func computeBlockLayout(
	t tree.LayoutBlockContainer,
	nodeID tree.NodeId,
	inputs tree.LayoutInput,
	blockCtx *BlockContext,
) tree.LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	parentSize := inputs.ParentSize
	runMode := inputs.RunMode

	coreStyle := t.GetCoreContainerStyle(nodeID)
	calc := t.ResolveCalcValue

	overflow := coreStyle.GetOverflow()
	isScrollContainer := overflow.X.IsScrollContainer() || overflow.Y.IsScrollContainer()
	aspectRatio := coreStyle.GetAspectRatio()
	padding := style.ResolveRectLPWidth(coreStyle.GetPadding(), parentSize.Width, calc)
	border := style.ResolveRectLPWidth(coreStyle.GetBorder(), parentSize.Width, calc)
	paddingBorderSize := geometry.RectSumAxes(geometry.RectAddF32(padding, border))

	boxSizingAdjustment := geometry.SizeZeroF32
	if coreStyle.GetBoxSizing() == style.BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSize
	}

	minSize := geometry.SizeMaybeAdd(
		geometry.SizeOptionMaybeApplyAspectRatio(
			style.MaybeResolveSizeDimOption(coreStyle.GetMinSize(), parentSize, calc),
			aspectRatio,
		),
		geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)},
	)
	maxSize := geometry.SizeMaybeAdd(
		geometry.SizeOptionMaybeApplyAspectRatio(
			style.MaybeResolveSizeDimOption(coreStyle.GetMaxSize(), parentSize, calc),
			aspectRatio,
		),
		geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)},
	)

	var clampedStyleSize geometry.Size[lib.Option[float32]]
	if inputs.SizingMode == tree.SizingModeInherentSize {
		clampedStyleSize = geometry.SizeMaybeClamp(
			geometry.SizeMaybeAdd(
				geometry.SizeOptionMaybeApplyAspectRatio(
					style.MaybeResolveSizeDimOption(coreStyle.GetSize(), parentSize, calc),
					aspectRatio,
				),
				geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)},
			),
			minSize,
			maxSize,
		)
	} else {
		clampedStyleSize = geometry.SizeOptionNone
	}

	// If both min and max in a given axis are set and max <= min then this determines the size in that axis
	minMaxDefiniteSize := geometry.SizeZipMap(minSize, maxSize, func(mn, mx lib.Option[float32]) lib.Option[float32] {
		if mn.IsSome && mx.IsSome && mx.Value <= mn.Value {
			return mn
		}
		return lib.None[float32]()
	})

	styledBasedKnownDimensions := geometry.SizeMaybeMax(
		geometry.SizeOptionOr(
			geometry.SizeOptionOr(knownDimensions, minMaxDefiniteSize),
			clampedStyleSize,
		),
		geometry.Size[lib.Option[float32]]{Width: lib.Some(paddingBorderSize.Width), Height: lib.Some(paddingBorderSize.Height)},
	)

	// Short-circuit if both dimensions known and only computing size
	if runMode == tree.RunModeComputeSize {
		if styledBasedKnownDimensions.Width.IsSome && styledBasedKnownDimensions.Height.IsSome {
			return tree.LayoutOutputFromOuterSize(geometry.Size[float32]{
				Width:  styledBasedKnownDimensions.Width.Value,
				Height: styledBasedKnownDimensions.Height.Value,
			})
		}
	}

	// Dispatch to computeBlockInner with either the inherited or a new BFC
	newInputs := tree.LayoutInput{
		KnownDimensions:               styledBasedKnownDimensions,
		ParentSize:                    inputs.ParentSize,
		AvailableSpace:                inputs.AvailableSpace,
		SizingMode:                    inputs.SizingMode,
		Axis:                          inputs.Axis,
		RunMode:                       inputs.RunMode,
		VerticalMarginsAreCollapsible: inputs.VerticalMarginsAreCollapsible,
	}

	if blockCtx != nil && !isScrollContainer {
		return computeBlockInner(t, nodeID, newInputs, blockCtx)
	}

	rootBfc := NewBlockFormattingContext()
	rootCtx := rootBfc.RootBlockContext()
	return computeBlockInner(t, nodeID, newInputs, rootCtx)
}

// ---------------------------------------------------------------------------
// computeBlockInner — core block layout algorithm
// ---------------------------------------------------------------------------

func computeBlockInner(
	t tree.LayoutBlockContainer,
	nodeID tree.NodeId,
	inputs tree.LayoutInput,
	blockCtx *BlockContext,
) tree.LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	parentSize := inputs.ParentSize
	availableSpace := inputs.AvailableSpace
	runMode := inputs.RunMode
	verticalMarginsAreCollapsible := inputs.VerticalMarginsAreCollapsible

	coreStyle := t.GetCoreContainerStyle(nodeID)
	containerStyle := t.GetBlockContainerStyle(nodeID)
	calc := t.ResolveCalcValue

	rawPadding := coreStyle.GetPadding()
	rawBorder := coreStyle.GetBorder()
	rawMargin := coreStyle.GetMargin()
	aspectRatio := coreStyle.GetAspectRatio()
	padding := style.ResolveRectLPWidth(rawPadding, parentSize.Width, calc)
	border := style.ResolveRectLPWidth(rawBorder, parentSize.Width, calc)
	direction := coreStyle.GetDirection()

	// Scrollbar gutters
	scrollbarGutter := func() geometry.Rect[float32] {
		offsets := coreStyle.GetOverflow()
		// Transposed: scrolling vertically needs horizontal space for scrollbar
		xOverflow := offsets.Y
		yOverflow := offsets.X
		xGutter := float32(0)
		if xOverflow == style.OverflowScroll {
			xGutter = coreStyle.GetScrollbarWidth()
		}
		yGutter := float32(0)
		if yOverflow == style.OverflowScroll {
			yGutter = coreStyle.GetScrollbarWidth()
		}
		switch direction {
		case style.DirectionRTL:
			return geometry.Rect[float32]{Top: 0, Left: xGutter, Right: 0, Bottom: yGutter}
		default:
			return geometry.Rect[float32]{Top: 0, Left: 0, Right: xGutter, Bottom: yGutter}
		}
	}()

	paddingBorder := geometry.RectAddF32(padding, border)
	paddingBorderSize := geometry.RectSumAxes(paddingBorder)
	contentBoxInset := geometry.RectAddF32(paddingBorder, scrollbarGutter)
	containerContentBoxSize := geometry.SizeMaybeSub(knownDimensions, geometry.Size[lib.Option[float32]]{
		Width:  lib.Some(geometry.RectHorizontalAxisSum(contentBoxInset)),
		Height: lib.Some(geometry.RectVerticalAxisSum(contentBoxInset)),
	})

	// Apply content box inset
	blockCtx.applyContentBoxInset([2]float32{contentBoxInset.Left, contentBoxInset.Right})

	boxSizingAdjustment := geometry.SizeZeroF32
	if coreStyle.GetBoxSizing() == style.BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSize
	}

	size := geometry.SizeMaybeAdd(
		geometry.SizeOptionMaybeApplyAspectRatio(
			style.MaybeResolveSizeDimOption(coreStyle.GetSize(), parentSize, calc),
			aspectRatio,
		),
		geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)},
	)
	minSize := geometry.SizeMaybeAdd(
		geometry.SizeOptionMaybeApplyAspectRatio(
			style.MaybeResolveSizeDimOption(coreStyle.GetMinSize(), parentSize, calc),
			aspectRatio,
		),
		geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)},
	)
	maxSize := geometry.SizeMaybeAdd(
		geometry.SizeOptionMaybeApplyAspectRatio(
			style.MaybeResolveSizeDimOption(coreStyle.GetMaxSize(), parentSize, calc),
			aspectRatio,
		),
		geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)},
	)

	overflow := coreStyle.GetOverflow()
	isScrollContainer := overflow.X.IsScrollContainer() || overflow.Y.IsScrollContainer()

	// Determine margin collapsing behaviour
	ownMarginsCollapseWithChildren := geometry.Line[bool]{
		Start: verticalMarginsAreCollapsible.Start &&
			!isScrollContainer &&
			coreStyle.GetPosition() == style.PositionRelative &&
			padding.Top == 0 &&
			border.Top == 0,
		End: verticalMarginsAreCollapsible.End &&
			!isScrollContainer &&
			coreStyle.GetPosition() == style.PositionRelative &&
			padding.Bottom == 0 &&
			border.Bottom == 0 &&
			!size.Height.IsSome,
	}

	hasStylesPreventingBeingCollapsedThrough := !coreStyle.GetIsBlock() ||
		blockCtx.isBfcRoot() ||
		isScrollContainer ||
		coreStyle.GetPosition() == style.PositionAbsolute ||
		padding.Top > 0 ||
		padding.Bottom > 0 ||
		border.Top > 0 ||
		border.Bottom > 0 ||
		(size.Height.IsSome && size.Height.Value > 0) ||
		(minSize.Height.IsSome && minSize.Height.Value > 0)

	textAlign := containerStyle.GetTextAlign()

	// 1. Generate items
	items := generateBlockItemList(t, nodeID, containerContentBoxSize)

	// 2. Compute container width
	var containerOuterWidth float32
	if knownDimensions.Width.IsSome {
		containerOuterWidth = knownDimensions.Width.Value
	} else {
		availWidth := style.AsMaybeSubF32(availableSpace.Width, geometry.RectHorizontalAxisSum(contentBoxInset))
		intrinsicWidth := determineContentBasedContainerWidth(t, items, availWidth) +
			geometry.RectHorizontalAxisSum(contentBoxInset)
		containerOuterWidth = geometry.MaybeMaxF32(
			geometry.MaybeClampF32(intrinsicWidth, minSize.Width, maxSize.Width),
			lib.Some(paddingBorderSize.Width),
		)
	}

	// Short-circuit if computing size and both dimensions known
	if runMode == tree.RunModeComputeSize && knownDimensions.Height.IsSome {
		return tree.LayoutOutputFromOuterSize(geometry.Size[float32]{
			Width:  containerOuterWidth,
			Height: knownDimensions.Height.Value,
		})
	}

	containerPercentageResolutionHeight := func() lib.Option[float32] {
		h := knownDimensions.Height
		if h.IsSome {
			return h
		}
		h2 := geometry.MaybeMaxOption(size.Height, minSize.Height)
		if h2.IsSome {
			return h2
		}
		return minSize.Height
	}()

	// 3. Perform final item layout and return content height
	resolvedPadding := style.ResolveRectLPWidth(rawPadding, lib.Some(containerOuterWidth), calc)
	resolvedBorder := style.ResolveRectLPWidth(rawBorder, lib.Some(containerOuterWidth), calc)
	resolvedContentBoxInset := geometry.RectAddF32(geometry.RectAddF32(resolvedPadding, resolvedBorder), scrollbarGutter)

	inflowContentSize, intrinsicOuterHeight, firstChildTopMarginSet, lastChildBottomMarginSet :=
		performFinalLayoutOnInFlowChildren(
			t,
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
		intrinsicOuterHeight = geometry.F32Max(intrinsicOuterHeight, blockCtx.FloatedContentHeightContribution())
	}

	containerOuterHeight := geometry.MaybeMaxF32(
		geometry.MaybeClampF32(
			knownDimensions.Height.UnwrapOr(intrinsicOuterHeight),
			minSize.Height,
			maxSize.Height,
		),
		lib.Some(paddingBorderSize.Height),
	)
	finalOuterSize := geometry.Size[float32]{Width: containerOuterWidth, Height: containerOuterHeight}

	// Short-circuit if computing size
	if runMode == tree.RunModeComputeSize {
		return tree.LayoutOutputFromOuterSize(finalOuterSize)
	}

	// 4. Layout absolutely positioned children
	absolutePositionInset := geometry.RectAddF32(resolvedBorder, scrollbarGutter)
	absolutePositionArea := geometry.SizeSubF32(finalOuterSize, geometry.RectSumAxes(absolutePositionInset))
	absolutePositionOffset := geometry.Point[float32]{X: absolutePositionInset.Left, Y: absolutePositionInset.Top}
	absoluteContentSize := performBlockAbsoluteLayout(
		t,
		items,
		absolutePositionArea,
		absolutePositionOffset,
		direction,
	)

	// 5. Perform hidden layout on hidden children
	childCount := t.ChildCount(nodeID)
	for order := 0; order < childCount; order++ {
		child := t.GetChildId(nodeID, order)
		childStyle := t.GetBlockChildStyle(child)
		childCoreStyle := t.GetCoreContainerStyle(child)
		if childCoreStyle.GetBoxGenerationMode() == style.BoxGenerationModeNone {
			t.SetUnroundedLayout(child, tree.Layout{Order: uint32(order)})
			t.ComputeChildLayout(child, tree.LayoutInput{
				KnownDimensions:               geometry.SizeOptionNone,
				ParentSize:                    geometry.SizeOptionNone,
				AvailableSpace:                geometry.Size[style.AvailableSpace]{Width: style.AvailableSpaceMaxContentValue, Height: style.AvailableSpaceMaxContentValue},
				SizingMode:                    tree.SizingModeInherentSize,
				Axis:                          tree.RequestedAxisBoth,
				RunMode:                       tree.RunModePerformLayout,
				VerticalMarginsAreCollapsible: geometry.LineFalseBool,
			})
			_ = childStyle // used via childCoreStyle
		}
	}

	// 6. Compute output margins
	allInFlowChildrenCanBeCollapsedThrough := true
	for _, item := range items {
		if item.position != style.PositionAbsolute && !item.canBeCollapsedThrough {
			allInFlowChildrenCanBeCollapsedThrough = false
			break
		}
	}
	canBeCollapsedThrough := !hasStylesPreventingBeingCollapsedThrough && allInFlowChildrenCanBeCollapsedThrough

	contentSize := geometry.SizeF32Max(inflowContentSize, absoluteContentSize)

	var topMargin tree.CollapsibleMarginSet
	if ownMarginsCollapseWithChildren.Start {
		topMargin = firstChildTopMarginSet
	} else {
		marginTop := style.ResolveOrZeroLPA(rawMargin.Top, parentSize.Width, calc)
		topMargin = tree.CollapsibleMarginSetFromMargin(marginTop)
	}

	var bottomMargin tree.CollapsibleMarginSet
	if ownMarginsCollapseWithChildren.End {
		bottomMargin = lastChildBottomMarginSet
	} else {
		marginBottom := style.ResolveOrZeroLPA(rawMargin.Bottom, parentSize.Width, calc)
		bottomMargin = tree.CollapsibleMarginSetFromMargin(marginBottom)
	}

	return tree.LayoutOutput{
		Size:                      finalOuterSize,
		ContentSize:               contentSize,
		FirstBaselines:            geometry.PointNoneF32,
		TopMargin:                 topMargin,
		BottomMargin:              bottomMargin,
		MarginsCanCollapseThrough: canBeCollapsedThrough,
	}
}

// ---------------------------------------------------------------------------
// generateBlockItemList — creates block items from children
// ---------------------------------------------------------------------------

func generateBlockItemList(
	t tree.LayoutBlockContainer,
	node tree.NodeId,
	nodeInnerSize geometry.Size[lib.Option[float32]],
) []blockItem {
	childIDs := t.ChildIds(node)
	var items []blockItem
	order := 0

	for _, childNodeID := range childIDs {
		childStyle := t.GetBlockChildStyle(childNodeID)
		childCoreStyle := t.GetCoreContainerStyle(childNodeID)
		if childCoreStyle.GetBoxGenerationMode() == style.BoxGenerationModeNone {
			continue
		}

		calc := t.ResolveCalcValue
		aspectRatio := childCoreStyle.GetAspectRatio()
		padding := style.ResolveRectLPWidth(childCoreStyle.GetPadding(), nodeInnerSize.Width, calc)
		border := style.ResolveRectLPWidth(childCoreStyle.GetBorder(), nodeInnerSize.Width, calc)
		pbSum := geometry.RectSumAxes(geometry.RectAddF32(padding, border))
		boxSizingAdjustment := geometry.SizeZeroF32
		if childCoreStyle.GetBoxSizing() == style.BoxSizingContentBox {
			boxSizingAdjustment = pbSum
		}

		position := childCoreStyle.GetPosition()
		childOverflow := childCoreStyle.GetOverflow()
		childFloat := childStyle.GetFloat()
		isNotFloated := childFloat == style.FloatNone
		isBlock := childCoreStyle.GetIsBlock()
		isTable := childStyle.GetIsTable()
		isScrollContainer := childOverflow.X.IsScrollContainer() || childOverflow.Y.IsScrollContainer()
		isInSameBfc := isBlock && !isTable && position != style.PositionAbsolute && isNotFloated && !isScrollContainer

		bsa := geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)}

		items = append(items, blockItem{
			nodeID:      childNodeID,
			order:       uint32(order),
			isTable:     isTable,
			isInSameBfc: isInSameBfc,
			float_:      childFloat,
			clear_:      childStyle.GetClear(),
			size: geometry.SizeMaybeAdd(
				geometry.SizeOptionMaybeApplyAspectRatio(
					style.MaybeResolveSizeDimOption(childCoreStyle.GetSize(), nodeInnerSize, calc),
					aspectRatio,
				),
				bsa,
			),
			minSize: geometry.SizeMaybeAdd(
				geometry.SizeOptionMaybeApplyAspectRatio(
					style.MaybeResolveSizeDimOption(childCoreStyle.GetMinSize(), nodeInnerSize, calc),
					aspectRatio,
				),
				bsa,
			),
			maxSize: geometry.SizeMaybeAdd(
				geometry.SizeOptionMaybeApplyAspectRatio(
					style.MaybeResolveSizeDimOption(childCoreStyle.GetMaxSize(), nodeInnerSize, calc),
					aspectRatio,
				),
				bsa,
			),
			overflow:         childOverflow,
			scrollbarWidth:   childCoreStyle.GetScrollbarWidth(),
			position:         position,
			inset:            childCoreStyle.GetInset(),
			margin:           childCoreStyle.GetMargin(),
			padding:          padding,
			border:           border,
			paddingBorderSum: pbSum,
			computedSize:     geometry.SizeZeroF32,
			staticPosition:   geometry.PointZeroF32,
		})

		order++
	}

	return items
}

// ---------------------------------------------------------------------------
// determineContentBasedContainerWidth
// ---------------------------------------------------------------------------

func determineContentBasedContainerWidth(
	t tree.LayoutPartialTree,
	items []blockItem,
	availableWidth style.AvailableSpace,
) float32 {
	calc := t.ResolveCalcValue
	asAvailable := geometry.Size[style.AvailableSpace]{Width: availableWidth, Height: style.AvailableSpaceMinContentValue}

	maxChildWidth := float32(0)
	floatContribution := newFloatIntrinsicWidthCalculator(availableWidth)

	for i := range items {
		item := &items[i]
		if item.position == style.PositionAbsolute {
			continue
		}

		knownDimensions := geometry.SizeMaybeClamp(item.size, item.minSize, item.maxSize)

		itemXMarginSum := geometry.RectHorizontalAxisSum(
			style.ResolveOrZeroRectLPASizeContext(item.margin, style.SizeAvailableSpaceIntoOptions(asAvailable), calc),
		)

		width := knownDimensions.Width.UnwrapOr(func() float32 {
			sizeAndBaselines := tree.PerformChildLayout(
				t,
				item.nodeID,
				knownDimensions,
				geometry.SizeOptionNone,
				geometry.Size[style.AvailableSpace]{
					Width:  style.AsMaybeSubF32(asAvailable.Width, itemXMarginSum),
					Height: asAvailable.Height,
				},
				tree.SizingModeInherentSize,
				geometry.LineTrueBool,
			)
			return sizeAndBaselines.Size.Width
		}())

		width = geometry.F32Max(width, item.paddingBorderSum.Width) + itemXMarginSum

		if dir, isFloated := item.float_.GetFloatDirection(); isFloated {
			floatContribution.addFloat(width, dir, item.clear_)
			continue
		}

		maxChildWidth = geometry.F32Max(maxChildWidth, width)
	}

	maxChildWidth = geometry.F32Max(maxChildWidth, floatContribution.result())
	return maxChildWidth
}

// ---------------------------------------------------------------------------
// performFinalLayoutOnInFlowChildren
// ---------------------------------------------------------------------------

func performFinalLayoutOnInFlowChildren(
	t tree.LayoutBlockContainer,
	items []blockItem,
	containerOuterWidth float32,
	containerPercentageResolutionHeight lib.Option[float32],
	contentBoxInset geometry.Rect[float32],
	resolvedContentBoxInset geometry.Rect[float32],
	textAlign style.TextAlign,
	direction style.Direction,
	ownMarginsCollapseWithChildren geometry.Line[bool],
	blockCtx *BlockContext,
) (geometry.Size[float32], float32, tree.CollapsibleMarginSet, tree.CollapsibleMarginSet) {
	calc := t.ResolveCalcValue

	containerInnerWidth := containerOuterWidth - geometry.RectHorizontalAxisSum(resolvedContentBoxInset)
	containerPctResHeight := geometry.MaybeSubOption(containerPercentageResolutionHeight, lib.Some(geometry.RectVerticalAxisSum(resolvedContentBoxInset)))
	parentSize := geometry.Size[lib.Option[float32]]{Width: lib.Some(containerInnerWidth), Height: containerPctResHeight}
	availableSpace := geometry.Size[style.AvailableSpace]{
		Width:  style.AvailableSpaceFromDefinite(containerInnerWidth),
		Height: style.AvailableSpaceMinContentValue,
	}

	if blockCtx.isBfcRoot() {
		blockCtx.setWidth(containerOuterWidth)
		blockCtx.applyContentBoxInset([2]float32{resolvedContentBoxInset.Left, resolvedContentBoxInset.Right})
	}

	inflowContentSize := geometry.SizeZeroF32
	committedYOffset := resolvedContentBoxInset.Top
	yOffsetForAbsolute := resolvedContentBoxInset.Top
	firstChildTopMarginSet := tree.ZeroCollapsibleMarginSet
	activeCollapsibleMarginSet := tree.ZeroCollapsibleMarginSet
	isCollapsingWithFirstMarginSet := true

	hasActiveFloats := blockCtx.hasActiveFloats(committedYOffset)
	yOffsetForFloat := resolvedContentBoxInset.Top

	for i := range items {
		item := &items[i]

		if item.position == style.PositionAbsolute {
			x := resolvedContentBoxInset.Left
			if direction == style.DirectionRTL {
				x = containerOuterWidth - resolvedContentBoxInset.Right
			}
			item.staticPosition = geometry.Point[float32]{X: x, Y: yOffsetForAbsolute}
			continue
		}

		itemMargin := geometry.Rect[lib.Option[float32]]{
			Left:   style.MaybeResolveLPAF32(item.margin.Left, containerOuterWidth, calc),
			Right:  style.MaybeResolveLPAF32(item.margin.Right, containerOuterWidth, calc),
			Top:    style.MaybeResolveLPAF32(item.margin.Top, containerOuterWidth, calc),
			Bottom: style.MaybeResolveLPAF32(item.margin.Bottom, containerOuterWidth, calc),
		}
		itemNonAutoMargin := geometry.Rect[float32]{
			Left:   itemMargin.Left.UnwrapOr(0),
			Right:  itemMargin.Right.UnwrapOr(0),
			Top:    itemMargin.Top.UnwrapOr(0),
			Bottom: itemMargin.Bottom.UnwrapOr(0),
		}
		itemNonAutoXMarginSum := geometry.RectHorizontalAxisSum(itemNonAutoMargin)

		scrollbarSize := geometry.Size[float32]{
			Width: func() float32 {
				if item.overflow.Y == style.OverflowScroll {
					return item.scrollbarWidth
				}
				return 0
			}(),
			Height: func() float32 {
				if item.overflow.X == style.OverflowScroll {
					return item.scrollbarWidth
				}
				return 0
			}(),
		}

		// Handle floated boxes
		if floatDirection, isFloated := item.float_.GetFloatDirection(); isFloated {
			hasActiveFloats = true

			itemLayout := tree.PerformChildLayout(
				t,
				item.nodeID,
				geometry.SizeOptionNone,
				parentSize,
				geometry.Size[style.AvailableSpace]{Width: style.AvailableSpaceMaxContentValue, Height: style.AvailableSpaceMaxContentValue},
				tree.SizingModeInherentSize,
				geometry.LineTrueBool,
			)
			marginBox := geometry.SizeAddF32(itemLayout.Size, geometry.RectSumAxes(itemNonAutoMargin))

			location := blockCtx.placeFloatedBox(marginBox, yOffsetForFloat, floatDirection, item.clear_)

			// Convert margin-box location to border-box location
			location.Y += itemNonAutoMargin.Top
			location.X += itemNonAutoMargin.Left

			t.SetUnroundedLayout(item.nodeID, tree.Layout{
				Order:         item.order,
				Size:          itemLayout.Size,
				ContentSize:   itemLayout.ContentSize,
				ScrollbarSize: scrollbarSize,
				Location:      location,
				Padding:       item.padding,
				Border:        item.border,
				Margin:        itemNonAutoMargin,
			})

			inflowContentSize = geometry.SizeF32Max(inflowContentSize, computeContentSizeContribution(
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
		var floatAvoidingPosition geometry.Point[float32]
		var floatAvoidingWidth float32

		if item.isInSameBfc {
			stretchWidth = containerInnerWidth - itemNonAutoXMarginSum
			floatAvoidingPosition = geometry.PointZeroF32
			floatAvoidingWidth = 0
		} else {
			// Set y_margin_offset for different-bfc child
			if !isCollapsingWithFirstMarginSet || !ownMarginsCollapseWithChildren.Start {
				yMarginOffset = activeCollapsibleMarginSet.CollapseWithMargin(itemNonAutoMargin.Top).Resolve()
			}
			minY := committedYOffset + yMarginOffset

			if hasActiveFloats {
				slot := blockCtx.findContentSlot(minY, item.clear_, nil)
				hasActiveFloats = slot.segmentID.IsSome
				stretchWidth = slot.width - itemNonAutoXMarginSum
				floatAvoidingPosition = geometry.Point[float32]{X: slot.x, Y: slot.y}
				floatAvoidingWidth = slot.width
			} else {
				stretchWidth = containerInnerWidth - itemNonAutoXMarginSum
				floatAvoidingPosition = geometry.Point[float32]{X: resolvedContentBoxInset.Left, Y: minY}
				floatAvoidingWidth = containerInnerWidth
			}
		}

		var knownDimensions geometry.Size[lib.Option[float32]]
		if item.isTable {
			knownDimensions = geometry.SizeOptionNone
		} else {
			w := item.size.Width.UnwrapOr(stretchWidth)
			w = geometry.MaybeClampF32(w, item.minSize.Width, item.maxSize.Width)
			knownDimensions = geometry.SizeMaybeClamp(
				geometry.Size[lib.Option[float32]]{Width: lib.Some(w), Height: item.size.Height},
				item.minSize,
				item.maxSize,
			)
		}

		layoutInputs := tree.LayoutInput{
			RunMode:    tree.RunModePerformLayout,
			SizingMode: tree.SizingModeInherentSize,
			Axis:       tree.RequestedAxisBoth,
			KnownDimensions: knownDimensions,
			ParentSize:      parentSize,
			AvailableSpace: geometry.Size[style.AvailableSpace]{
				Width:  style.AvailableSpaceFromDefinite(stretchWidth),
				Height: availableSpace.Height,
			},
			VerticalMarginsAreCollapsible: func() geometry.Line[bool] {
				if item.isInSameBfc {
					return geometry.LineTrueBool
				}
				return geometry.LineFalseBool
			}(),
		}

		clearPos := float32(0)
		if cp, ok := blockCtx.clearedThreshold(item.clear_); ok {
			clearPos = cp
		}

		var itemLayout tree.LayoutOutput
		if item.isInSameBfc {
			width := knownDimensions.Width.Value // same-bfc child always has defined width

			insetLeft := itemNonAutoMargin.Left + contentBoxInset.Left
			insetRight := containerOuterWidth - width - insetLeft
			insets := [2]float32{insetLeft, insetRight}

			childBlockCtx := blockCtx.SubContext(
				geometry.F32Max(yOffsetForAbsolute+itemNonAutoMargin.Top, clearPos),
				insets,
			)
			output := t.ComputeBlockChildLayout(item.nodeID, layoutInputs, childBlockCtx)

			childContribution := childBlockCtx.FloatedContentHeightContribution()
			blockCtx.AddChildFloatedContentHeightContribution(yOffsetForAbsolute + childContribution)

			itemLayout = output
		} else {
			itemLayout = t.ComputeChildLayout(item.nodeID, layoutInputs)
		}

		finalSize := itemLayout.Size

		topMarginSet := itemLayout.TopMargin.CollapseWithMargin(itemMargin.Top.UnwrapOr(0))
		bottomMarginSet := itemLayout.BottomMargin.CollapseWithMargin(itemMargin.Bottom.UnwrapOr(0))

		// Expand auto margins to fill available space
		freeXSpace := geometry.F32Max(0, stretchWidth-finalSize.Width)
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

		resolvedMargin := geometry.Rect[float32]{
			Left:   itemMargin.Left.UnwrapOr(xAxisAutoMarginSize),
			Right:  itemMargin.Right.UnwrapOr(xAxisAutoMarginSize),
			Top:    topMarginSet.Resolve(),
			Bottom: bottomMarginSet.Resolve(),
		}

		// Resolve item inset
		insetLeft := style.MaybeResolveLPAF32(item.inset.Left, containerInnerWidth, calc)
		insetRight := style.MaybeResolveLPAF32(item.inset.Right, containerInnerWidth, calc)
		insetTop := style.MaybeResolveLPAF32(item.inset.Top, 0, calc)
		insetBottom := style.MaybeResolveLPAF32(item.inset.Bottom, 0, calc)

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

		insetOffset := geometry.Point[float32]{X: insetOffsetX, Y: insetOffsetY}

		// Set y_margin_offset for same bfc child
		if item.isInSameBfc &&
			(!isCollapsingWithFirstMarginSet || !ownMarginsCollapseWithChildren.Start) {
			yMarginOffset = activeCollapsibleMarginSet.CollapseWithMargin(resolvedMargin.Top).Resolve()
		}

		floatOrNotClear := item.float_.IsFloated() || item.clear_ == style.ClearNone

		item.computedSize = itemLayout.Size
		item.canBeCollapsedThrough = itemLayout.MarginsCanCollapseThrough && floatOrNotClear

		if item.isInSameBfc {
			unclearedY := committedYOffset + activeCollapsibleMarginSet.Resolve()
			x := resolvedContentBoxInset.Left
			if direction == style.DirectionRTL {
				x = containerOuterWidth - resolvedContentBoxInset.Right - finalSize.Width
			}
			item.staticPosition = geometry.Point[float32]{
				X: x,
				Y: geometry.F32Max(unclearedY, clearPos),
			}
		} else {
			x := floatAvoidingPosition.X
			if direction == style.DirectionRTL {
				x = floatAvoidingPosition.X + floatAvoidingWidth - finalSize.Width
			}
			item.staticPosition = geometry.Point[float32]{
				X: x,
				Y: floatAvoidingPosition.Y,
			}
		}

		var location geometry.Point[float32]
		if item.isInSameBfc {
			x := resolvedContentBoxInset.Left + insetOffset.X + resolvedMargin.Left
			if direction == style.DirectionRTL {
				x = containerOuterWidth - resolvedContentBoxInset.Right - finalSize.Width - resolvedMargin.Right + insetOffset.X
			}
			location = geometry.Point[float32]{
				X: x,
				Y: geometry.F32Max(committedYOffset, clearPos) + yMarginOffset + insetOffset.Y,
			}
		} else {
			x := floatAvoidingPosition.X + resolvedMargin.Left + insetOffset.X
			if direction == style.DirectionRTL {
				x = floatAvoidingPosition.X + floatAvoidingWidth - finalSize.Width - resolvedMargin.Right + insetOffset.X
			}
			location = geometry.Point[float32]{
				X: x,
				Y: floatAvoidingPosition.Y + insetOffset.Y,
			}
		}

		// Apply text alignment
		itemOuterWidth := itemLayout.Size.Width + geometry.RectHorizontalAxisSum(resolvedMargin)
		if itemOuterWidth < containerInnerWidth {
			freeSpace := containerInnerWidth - itemOuterWidth
			switch {
			case textAlign == style.TextAlignAuto:
				// Do nothing
			case textAlign == style.TextAlignLegacyLeft && direction == style.DirectionLTR:
				// Do nothing
			case textAlign == style.TextAlignLegacyLeft && direction == style.DirectionRTL:
				location.X -= freeSpace
			case textAlign == style.TextAlignLegacyRight && direction == style.DirectionLTR:
				location.X += freeSpace
			case textAlign == style.TextAlignLegacyRight && direction == style.DirectionRTL:
				// Do nothing
			case textAlign == style.TextAlignLegacyCenter && direction == style.DirectionLTR:
				location.X += freeSpace / 2
			case textAlign == style.TextAlignLegacyCenter && direction == style.DirectionRTL:
				location.X -= freeSpace / 2
			}
		}

		t.SetUnroundedLayout(item.nodeID, tree.Layout{
			Order:         item.order,
			Size:          itemLayout.Size,
			ContentSize:   itemLayout.ContentSize,
			ScrollbarSize: scrollbarSize,
			Location:      location,
			Padding:       item.padding,
			Border:        item.border,
			Margin:        resolvedMargin,
		})

		inflowContentSize = geometry.SizeF32Max(inflowContentSize, computeContentSizeContribution(
			geometry.Point[float32]{
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
	contentHeight := geometry.F32Max(0, committedYOffset)

	_ = yOffsetForFloat // suppress unused warning

	return inflowContentSize, contentHeight, firstChildTopMarginSet, lastChildBottomMarginSet
}

// ---------------------------------------------------------------------------
// performBlockAbsoluteLayout
// ---------------------------------------------------------------------------

func performBlockAbsoluteLayout(
	t tree.LayoutBlockContainer,
	items []blockItem,
	areaSize geometry.Size[float32],
	areaOffset geometry.Point[float32],
	direction style.Direction,
) geometry.Size[float32] {
	calc := t.ResolveCalcValue
	areaWidth := areaSize.Width
	areaHeight := areaSize.Height

	absoluteContentSize := geometry.SizeZeroF32

	for i := range items {
		item := &items[i]
		if item.position != style.PositionAbsolute {
			continue
		}

		childCoreStyle := t.GetCoreContainerStyle(item.nodeID)
		if childCoreStyle.GetBoxGenerationMode() == style.BoxGenerationModeNone || childCoreStyle.GetPosition() != style.PositionAbsolute {
			continue
		}

		aspectRatio := childCoreStyle.GetAspectRatio()
		margin := geometry.Rect[lib.Option[float32]]{
			Left:   style.MaybeResolveLPAF32(childCoreStyle.GetMargin().Left, areaWidth, calc),
			Right:  style.MaybeResolveLPAF32(childCoreStyle.GetMargin().Right, areaWidth, calc),
			Top:    style.MaybeResolveLPAF32(childCoreStyle.GetMargin().Top, areaWidth, calc),
			Bottom: style.MaybeResolveLPAF32(childCoreStyle.GetMargin().Bottom, areaWidth, calc),
		}
		padding := style.ResolveRectLPWidth(childCoreStyle.GetPadding(), lib.Some(areaWidth), calc)
		border := style.ResolveRectLPWidth(childCoreStyle.GetBorder(), lib.Some(areaWidth), calc)
		paddingBorderSum := geometry.RectSumAxes(geometry.RectAddF32(padding, border))
		boxSizingAdjustment := geometry.SizeZeroF32
		if childCoreStyle.GetBoxSizing() == style.BoxSizingContentBox {
			boxSizingAdjustment = paddingBorderSum
		}

		// Resolve inset
		left := style.MaybeResolveLPAF32(childCoreStyle.GetInset().Left, areaWidth, calc)
		right := style.MaybeResolveLPAF32(childCoreStyle.GetInset().Right, areaWidth, calc)
		top := style.MaybeResolveLPAF32(childCoreStyle.GetInset().Top, areaHeight, calc)
		bottom := style.MaybeResolveLPAF32(childCoreStyle.GetInset().Bottom, areaHeight, calc)

		bsa := geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)}
		areaSizeOpt := geometry.Size[lib.Option[float32]]{Width: lib.Some(areaSize.Width), Height: lib.Some(areaSize.Height)}

		// Compute known dimensions from size styles
		styleSize := geometry.SizeMaybeAdd(
			geometry.SizeOptionMaybeApplyAspectRatio(
				style.MaybeResolveSizeDimOption(childCoreStyle.GetSize(), areaSizeOpt, calc),
				aspectRatio,
			),
			bsa,
		)
		minSize := geometry.SizeMaybeMax(
			geometry.SizeOptionOr(
				geometry.SizeMaybeAdd(
					geometry.SizeOptionMaybeApplyAspectRatio(
						style.MaybeResolveSizeDimOption(childCoreStyle.GetMinSize(), areaSizeOpt, calc),
						aspectRatio,
					),
					bsa,
				),
				geometry.Size[lib.Option[float32]]{Width: lib.Some(paddingBorderSum.Width), Height: lib.Some(paddingBorderSum.Height)},
			),
			geometry.Size[lib.Option[float32]]{Width: lib.Some(paddingBorderSum.Width), Height: lib.Some(paddingBorderSum.Height)},
		)
		maxSize := geometry.SizeMaybeAdd(
			geometry.SizeOptionMaybeApplyAspectRatio(
				style.MaybeResolveSizeDimOption(childCoreStyle.GetMaxSize(), areaSizeOpt, calc),
				aspectRatio,
			),
			bsa,
		)
		knownDimensions := geometry.SizeMaybeClamp(styleSize, minSize, maxSize)

		// Fill in width from left/right
		if !knownDimensions.Width.IsSome && left.IsSome && right.IsSome {
			newWidthRaw := geometry.MaybeSubF32(geometry.MaybeSubF32(areaWidth, margin.Left), margin.Right) - left.Value - right.Value
			knownDimensions.Width = lib.Some(geometry.F32Max(newWidthRaw, 0))
			knownDimensions = geometry.SizeMaybeClamp(geometry.SizeOptionMaybeApplyAspectRatio(knownDimensions, aspectRatio), minSize, maxSize)
		}

		// Fill in height from top/bottom
		if !knownDimensions.Height.IsSome && top.IsSome && bottom.IsSome {
			newHeightRaw := geometry.MaybeSubF32(geometry.MaybeSubF32(areaHeight, margin.Top), margin.Bottom) - top.Value - bottom.Value
			knownDimensions.Height = lib.Some(geometry.F32Max(newHeightRaw, 0))
			knownDimensions = geometry.SizeMaybeClamp(geometry.SizeOptionMaybeApplyAspectRatio(knownDimensions, aspectRatio), minSize, maxSize)
		}

		measuredSize := measureChildSizeBoth(
			t,
			item.nodeID,
			knownDimensions,
			areaSizeOpt,
			geometry.Size[style.AvailableSpace]{
				Width:  style.AvailableSpaceFromDefinite(geometry.MaybeClampF32(areaWidth, minSize.Width, maxSize.Width)),
				Height: style.AvailableSpaceFromDefinite(geometry.MaybeClampF32(areaHeight, minSize.Height, maxSize.Height)),
			},
			tree.SizingModeContentSize,
			geometry.LineFalseBool,
		)

		finalSize := geometry.Size[float32]{
			Width:  geometry.MaybeClampF32(knownDimensions.Width.UnwrapOr(measuredSize.Width), minSize.Width, maxSize.Width),
			Height: geometry.MaybeClampF32(knownDimensions.Height.UnwrapOr(measuredSize.Height), minSize.Height, maxSize.Height),
		}

		layoutOutput := tree.PerformChildLayout(
			t,
			item.nodeID,
			geometry.Size[lib.Option[float32]]{Width: lib.Some(finalSize.Width), Height: lib.Some(finalSize.Height)},
			areaSizeOpt,
			geometry.Size[style.AvailableSpace]{
				Width:  style.AvailableSpaceFromDefinite(geometry.MaybeClampF32(areaWidth, minSize.Width, maxSize.Width)),
				Height: style.AvailableSpaceFromDefinite(geometry.MaybeClampF32(areaHeight, minSize.Height, maxSize.Height)),
			},
			tree.SizingModeContentSize,
			geometry.LineFalseBool,
		)

		nonAutoMargin := geometry.Rect[float32]{
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
		absoluteAutoMarginSpace := geometry.Point[float32]{
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
		freeSpace := geometry.Size[float32]{
			Width:  absoluteAutoMarginSpace.X - finalSize.Width - geometry.RectHorizontalAxisSum(nonAutoMargin),
			Height: absoluteAutoMarginSpace.Y - finalSize.Height - geometry.RectVerticalAxisSum(nonAutoMargin),
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

		autoMargin := geometry.Rect[float32]{
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

		resolvedMargin := geometry.Rect[float32]{
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

		location := geometry.Point[float32]{
			X: xOffset + areaOffset.X,
			Y: yPos,
		}

		scrollbarSize := geometry.Size[float32]{
			Width: func() float32 {
				if item.overflow.Y == style.OverflowScroll {
					return item.scrollbarWidth
				}
				return 0
			}(),
			Height: func() float32 {
				if item.overflow.X == style.OverflowScroll {
					return item.scrollbarWidth
				}
				return 0
			}(),
		}

		t.SetUnroundedLayout(item.nodeID, tree.Layout{
			Order:         item.order,
			Size:          finalSize,
			ContentSize:   layoutOutput.ContentSize,
			ScrollbarSize: scrollbarSize,
			Location:      location,
			Padding:       padding,
			Border:        border,
			Margin:        resolvedMargin,
		})

		relativeLocation := geometry.Point[float32]{
			X: location.X - areaOffset.X,
			Y: location.Y - areaOffset.Y,
		}
		absoluteContentSize = geometry.SizeF32Max(absoluteContentSize, computeContentSizeContribution(
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
