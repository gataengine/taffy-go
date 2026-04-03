// Package taffy implements the CSS Flexbox layout algorithm.
// This file is a port of compute/flexbox.rs from the Taffy Rust layout engine.
package taffy

import "math"

// ---------------------------------------------------------------------------
// Flex-direction-aware axis helpers for Size, Rect, AvailableSpace, and Point.
// These map main/cross axis concepts to width/height based on FlexDirection.
// ---------------------------------------------------------------------------

// sizeMain returns the main-axis component of a Size[T] for numeric types.
func sizeMain[T interface{ float32 | ~int }](s Size[T], dir FlexDirection) T {
	if dir.IsRow() {
		return s.Width
	}
	return s.Height
}

// sizeCross returns the cross-axis component of a Size[T] for numeric types.
func sizeCross[T interface{ float32 | ~int }](s Size[T], dir FlexDirection) T {
	if dir.IsRow() {
		return s.Height
	}
	return s.Width
}

// sizeSetMain sets the main-axis component of a Size.
func sizeSetMain[T any](s *Size[T], dir FlexDirection, val T) {
	if dir.IsRow() {
		s.Width = val
	} else {
		s.Height = val
	}
}

// sizeSetCross sets the cross-axis component of a Size.
func sizeSetCross[T any](s *Size[T], dir FlexDirection, val T) {
	if dir.IsRow() {
		s.Height = val
	} else {
		s.Width = val
	}
}

// sizeWithMain returns a copy with the main axis value replaced.
func sizeWithMain[T any](s Size[T], dir FlexDirection, val T) Size[T] {
	r := s
	sizeSetMain(&r, dir, val)
	return r
}

// sizeWithCross returns a copy with the cross axis value replaced.
func sizeWithCross[T any](s Size[T], dir FlexDirection, val T) Size[T] {
	r := s
	sizeSetCross(&r, dir, val)
	return r
}

// sizeFromCross creates a Size[Option[float32]] with only the cross axis set.
func sizeFromCross(dir FlexDirection, val Option[float32]) Size[Option[float32]] {
	s := SizeOptionNone
	sizeSetCross(&s, dir, val)
	return s
}

// sizeOptionMain returns the main-axis option component.
func sizeOptionMain(s Size[Option[float32]], dir FlexDirection) Option[float32] {
	if dir.IsRow() {
		return s.Width
	}
	return s.Height
}

// sizeOptionCross returns the cross-axis option component.
func sizeOptionCross(s Size[Option[float32]], dir FlexDirection) Option[float32] {
	if dir.IsRow() {
		return s.Height
	}
	return s.Width
}

// sizeOptionSetMain sets the main-axis option.
func sizeOptionSetMain(s *Size[Option[float32]], dir FlexDirection, val Option[float32]) {
	sizeSetMain(s, dir, val)
}

// sizeOptionSetCross sets the cross-axis option.
func sizeOptionSetCross(s *Size[Option[float32]], dir FlexDirection, val Option[float32]) {
	sizeSetCross(s, dir, val)
}

// sizeOptionWithMain returns a copy with main axis replaced.
func sizeOptionWithMain(s Size[Option[float32]], dir FlexDirection, val Option[float32]) Size[Option[float32]] {
	return sizeWithMain(s, dir, val)
}

// avMain returns the main-axis AvailableSpace.
func avMain(s Size[AvailableSpace], dir FlexDirection) AvailableSpace {
	if dir.IsRow() {
		return s.Width
	}
	return s.Height
}

// avCross returns the cross-axis AvailableSpace.
func avCross(s Size[AvailableSpace], dir FlexDirection) AvailableSpace {
	if dir.IsRow() {
		return s.Height
	}
	return s.Width
}

// avWithMain returns a copy with the main axis replaced.
func avWithMain(s Size[AvailableSpace], dir FlexDirection, val AvailableSpace) Size[AvailableSpace] {
	r := s
	if dir.IsRow() {
		r.Width = val
	} else {
		r.Height = val
	}
	return r
}

// avWithCross returns a copy with the cross axis replaced.
func avWithCross(s Size[AvailableSpace], dir FlexDirection, val AvailableSpace) Size[AvailableSpace] {
	r := s
	if dir.IsRow() {
		r.Height = val
	} else {
		r.Width = val
	}
	return r
}

// rectMainAxisSum returns the sum of the main-axis sides of a Rect[float32].
func rectMainAxisSum(r Rect[float32], dir FlexDirection) float32 {
	if dir.IsRow() {
		return r.Left + r.Right
	}
	return r.Top + r.Bottom
}

// rectCrossAxisSum returns the sum of the cross-axis sides of a Rect[float32].
func rectCrossAxisSum(r Rect[float32], dir FlexDirection) float32 {
	if dir.IsRow() {
		return r.Top + r.Bottom
	}
	return r.Left + r.Right
}

// rectMainStart returns the start (flex-direction-aware) value on the main axis.
func rectMainStart(r Rect[float32], dir FlexDirection) float32 {
	switch dir {
	case FlexDirectionRow:
		return r.Left
	case FlexDirectionRowReverse:
		return r.Right
	case FlexDirectionColumn:
		return r.Top
	default: // ColumnReverse
		return r.Bottom
	}
}

// rectMainEnd returns the end (flex-direction-aware) value on the main axis.
func rectMainEnd(r Rect[float32], dir FlexDirection) float32 {
	switch dir {
	case FlexDirectionRow:
		return r.Right
	case FlexDirectionRowReverse:
		return r.Left
	case FlexDirectionColumn:
		return r.Bottom
	default: // ColumnReverse
		return r.Top
	}
}

// rectCrossStart returns the start value on the cross axis.
func rectCrossStart(r Rect[float32], dir FlexDirection) float32 {
	switch dir {
	case FlexDirectionRow:
		return r.Top
	case FlexDirectionRowReverse:
		return r.Top
	case FlexDirectionColumn:
		return r.Left
	default: // ColumnReverse
		return r.Left
	}
}

// rectCrossEnd returns the end value on the cross axis.
func rectCrossEnd(r Rect[float32], dir FlexDirection) float32 {
	switch dir {
	case FlexDirectionRow:
		return r.Bottom
	case FlexDirectionRowReverse:
		return r.Bottom
	case FlexDirectionColumn:
		return r.Right
	default: // ColumnReverse
		return r.Right
	}
}

// rectOptionMainStart returns the main-start Option value.
func rectOptionMainStart(r Rect[Option[float32]], dir FlexDirection) Option[float32] {
	switch dir {
	case FlexDirectionRow:
		return r.Left
	case FlexDirectionRowReverse:
		return r.Right
	case FlexDirectionColumn:
		return r.Top
	default:
		return r.Bottom
	}
}

// rectOptionMainEnd returns the main-end Option value.
func rectOptionMainEnd(r Rect[Option[float32]], dir FlexDirection) Option[float32] {
	switch dir {
	case FlexDirectionRow:
		return r.Right
	case FlexDirectionRowReverse:
		return r.Left
	case FlexDirectionColumn:
		return r.Bottom
	default:
		return r.Top
	}
}

// rectOptionCrossStart returns the cross-start Option value.
func rectOptionCrossStart(r Rect[Option[float32]], dir FlexDirection) Option[float32] {
	switch dir {
	case FlexDirectionRow, FlexDirectionRowReverse:
		return r.Top
	default:
		return r.Left
	}
}

// rectOptionCrossEnd returns the cross-end Option value.
func rectOptionCrossEnd(r Rect[Option[float32]], dir FlexDirection) Option[float32] {
	switch dir {
	case FlexDirectionRow, FlexDirectionRowReverse:
		return r.Bottom
	default:
		return r.Right
	}
}

// rectBoolMainStart returns the main-start bool.
func rectBoolMainStart(r Rect[bool], dir FlexDirection) bool {
	switch dir {
	case FlexDirectionRow:
		return r.Left
	case FlexDirectionRowReverse:
		return r.Right
	case FlexDirectionColumn:
		return r.Top
	default:
		return r.Bottom
	}
}

// rectBoolMainEnd returns the main-end bool.
func rectBoolMainEnd(r Rect[bool], dir FlexDirection) bool {
	switch dir {
	case FlexDirectionRow:
		return r.Right
	case FlexDirectionRowReverse:
		return r.Left
	case FlexDirectionColumn:
		return r.Bottom
	default:
		return r.Top
	}
}

// rectBoolCrossStart returns the cross-start bool.
func rectBoolCrossStart(r Rect[bool], dir FlexDirection) bool {
	switch dir {
	case FlexDirectionRow, FlexDirectionRowReverse:
		return r.Top
	default:
		return r.Left
	}
}

// rectBoolCrossEnd returns the cross-end bool.
func rectBoolCrossEnd(r Rect[bool], dir FlexDirection) bool {
	switch dir {
	case FlexDirectionRow, FlexDirectionRowReverse:
		return r.Bottom
	default:
		return r.Right
	}
}

// pointMain returns the main-axis component of a Point.
func pointMain(p Point[float32], dir FlexDirection) float32 {
	if dir.IsRow() {
		return p.X
	}
	return p.Y
}

// pointCross returns the cross-axis component of a Point.
func pointCross(p Point[float32], dir FlexDirection) float32 {
	if dir.IsRow() {
		return p.Y
	}
	return p.X
}

// dimIsAuto checks if a Dimension's cross-axis component is auto.
func dimCrossIsAuto(s Size[Dimension], dir FlexDirection) bool {
	if dir.IsRow() {
		return s.Height.IsAuto()
	}
	return s.Width.IsAuto()
}

// asClampF32 clamps an AvailableSpace with concrete min/max float32 values.
func asClampF32(a AvailableSpace, min, max float32) AvailableSpace {
	if a.Tag == AvailableSpaceDefinite {
		return AvailableSpaceFromDefinite(f32Max(f32Min(a.Value, max), min))
	}
	return a
}

// asClampOption clamps an AvailableSpace with optional min/max.
func asClampOption(a AvailableSpace, min_, max_ Option[float32]) AvailableSpace {
	if a.Tag == AvailableSpaceDefinite {
		v := a.Value
		if min_.IsSome && max_.IsSome {
			return AvailableSpaceFromDefinite(f32Max(f32Min(v, max_.Value), min_.Value))
		}
		if min_.IsSome {
			return AvailableSpaceFromDefinite(f32Max(v, min_.Value))
		}
		if max_.IsSome {
			return AvailableSpaceFromDefinite(f32Min(v, max_.Value))
		}
		return a
	}
	return a
}

// computeContentSizeContribution computes how much width/height a node contributes to its parent's content size.
func computeContentSizeContribution(
	location Point[float32],
	size Size[float32],
	contentSize Size[float32],
	overflow Point[Overflow],
) Size[float32] {
	sizeContrib := Size[float32]{
		Width: func() float32 {
			if overflow.X == OverflowVisible {
				return f32Max(size.Width, contentSize.Width)
			}
			return size.Width
		}(),
		Height: func() float32 {
			if overflow.Y == OverflowVisible {
				return f32Max(size.Height, contentSize.Height)
			}
			return size.Height
		}(),
	}
	if sizeContrib.Width > 0 && sizeContrib.Height > 0 {
		maxX := f32Max(location.X+sizeContrib.Width, 0.0)
		minX := f32Min(location.X, 0.0)
		maxY := f32Max(location.Y+sizeContrib.Height, 0.0)
		minY := f32Min(location.Y, 0.0)
		return Size[float32]{Width: maxX - minX, Height: maxY - minY}
	}
	return SizeZeroF32
}

// resolveRectLPAToOption resolves a Rect[LengthPercentageAuto] to Rect[Option[float32]]
// where all four sides are resolved against the same context value.
func resolveRectLPAToOption(rect Rect[LengthPercentageAuto], context float32, calc CalcResolver) Rect[Option[float32]] {
	return Rect[Option[float32]]{
		Left:   MaybeResolveLPAF32(rect.Left, context, calc),
		Right:  MaybeResolveLPAF32(rect.Right, context, calc),
		Top:    MaybeResolveLPAF32(rect.Top, context, calc),
		Bottom: MaybeResolveLPAF32(rect.Bottom, context, calc),
	}
}

// measureChildSizeBoth computes both dimensions of a child (RunMode::ComputeSize + RequestedAxis::Both).
func measureChildSizeBoth(
	tree layoutPartialTree,
	nodeId NodeId,
	knownDimensions Size[Option[float32]],
	parentSize Size[Option[float32]],
	availableSpace Size[AvailableSpace],
	sizingMode SizingMode,
	verticalMarginsAreCollapsible Line[bool],
) Size[float32] {
	return tree.computeChildLayout(nodeId, LayoutInput{
		KnownDimensions:               knownDimensions,
		ParentSize:                    parentSize,
		AvailableSpace:                availableSpace,
		SizingMode:                    sizingMode,
		Axis:                          RequestedAxisBoth,
		RunMode:                       RunModeComputeSize,
		VerticalMarginsAreCollapsible: verticalMarginsAreCollapsible,
	}).Size
}

// ---------------------------------------------------------------------------
// Flexbox internal types
// ---------------------------------------------------------------------------

// flexItem holds intermediate results for a single flex item.
type flexItem struct {
	node  NodeId
	order uint32

	size    Size[Option[float32]]
	minSize Size[Option[float32]]
	maxSize Size[Option[float32]]

	alignSelf      AlignSelf
	overflow       Point[Overflow]
	scrollbarWidth float32
	flexShrink     float32
	flexGrow       float32

	resolvedMinimumMainSize float32

	inset        Rect[Option[float32]]
	margin       Rect[float32]
	marginIsAuto Rect[bool]
	padding      Rect[float32]
	border       Rect[float32]

	flexBasis      float32
	innerFlexBasis float32
	violation      float32
	frozen         bool

	contentFlexFraction float32

	hypotheticalInnerSize Size[float32]
	hypotheticalOuterSize Size[float32]
	targetSize            Size[float32]
	outerTargetSize       Size[float32]

	baseline float32

	offsetMain  float32
	offsetCross float32
}

// isScrollContainer returns true if the item is a scroll container.
func (fi *flexItem) isScrollContainer() bool {
	return fi.overflow.X.IsScrollContainer() || fi.overflow.Y.IsScrollContainer()
}

// flexLine holds a slice of flex items for one line plus cross-axis sizing info.
type flexLine struct {
	items       []*flexItem
	crossSize   float32
	offsetCross float32
}

// algoConstants holds computed constants for the flexbox algorithm.
type algoConstants struct {
	dir             FlexDirection
	layoutDirection Direction
	isRow           bool
	isColumn        bool
	isWrap          bool
	isWrapReverse   bool

	minSize Size[Option[float32]]
	maxSize Size[Option[float32]]

	margin          Rect[float32]
	border          Rect[float32]
	contentBoxInset Rect[float32]
	scrollbarGutter Point[float32]
	gap             Size[float32]

	alignItems     AlignItems
	alignContent   AlignContent
	justifyContent Option[AlignContent]

	nodeOuterSize      Size[Option[float32]]
	nodeInnerSize      Size[Option[float32]]
	containerSize      Size[float32]
	innerContainerSize Size[float32]
}

// ---------------------------------------------------------------------------
// Main entry point
// ---------------------------------------------------------------------------

// computeFlexboxLayout computes the layout of a box according to the flexbox algorithm.
func computeFlexboxLayout(
	tree layoutFlexboxContainer,
	node NodeId,
	inputs LayoutInput,
) LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	parentSize := inputs.ParentSize
	runMode := inputs.RunMode

	style := tree.getFlexboxContainerStyle(node)
	calc := tree.resolveCalcValue

	aspectRatio := style.aspectRatio()
	padding := resolveRectLPWidth(style.padding(), parentSize.Width, calc)
	border := resolveRectLPWidth(style.border(), parentSize.Width, calc)
	paddingBorderSum := SizeAddF32(RectSumAxes(padding), RectSumAxes(border))
	boxSizingAdjustment := SizeZeroF32
	if style.boxSizing() == BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSum
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

	// If both min and max in a given axis are set and max <= min then this determines the size.
	minMaxDefiniteSize := SizeZipMap(minSize, maxSize, func(min, max Option[float32]) Option[float32] {
		if min.IsSome && max.IsSome && max.Value <= min.Value {
			return min
		}
		return None[float32]()
	})

	styledBasedKnownDimensions := SizeMaybeMax(
		SizeOptionOr(knownDimensions,
			SizeOptionOr(minMaxDefiniteSize, clampedStyleSize),
		),
		Size[Option[float32]]{Width: Some(paddingBorderSum.Width), Height: Some(paddingBorderSum.Height)},
	)

	// Short-circuit if container size is fully known and we only need the size.
	if runMode == RunModeComputeSize {
		if styledBasedKnownDimensions.Width.IsSome && styledBasedKnownDimensions.Height.IsSome {
			return LayoutOutputFromOuterSize(Size[float32]{
				Width:  styledBasedKnownDimensions.Width.Value,
				Height: styledBasedKnownDimensions.Height.Value,
			})
		}
	}

	return computePreliminary(tree, node, LayoutInput{
		KnownDimensions:               styledBasedKnownDimensions,
		ParentSize:                    inputs.ParentSize,
		AvailableSpace:                inputs.AvailableSpace,
		SizingMode:                    inputs.SizingMode,
		Axis:                          inputs.Axis,
		RunMode:                       inputs.RunMode,
		VerticalMarginsAreCollapsible: inputs.VerticalMarginsAreCollapsible,
	})
}

// ---------------------------------------------------------------------------
// computePreliminary - the core flexbox algorithm
// ---------------------------------------------------------------------------

func computePreliminary(tree layoutFlexboxContainer, node NodeId, inputs LayoutInput) LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	availableSpace := inputs.AvailableSpace
	runMode := inputs.RunMode

	// Define constants.
	constants := computeConstants(tree, tree.getFlexboxContainerStyle(node), knownDimensions, inputs.ParentSize)

	// 9.1. Initial Setup
	// 1. Generate anonymous flex items.
	flexItems := generateAnonymousFlexItems(tree, node, &constants)

	// 9.2. Line Length Determination
	// 2. Determine the available main and cross space for the flex items.
	availableSpace = determineAvailableSpace(knownDimensions, availableSpace, &constants)

	// 3. Determine the flex base size and hypothetical main size of each item.
	determineFlexBaseSize(tree, &constants, availableSpace, flexItems)

	// 5. Collect flex items into flex lines.
	flexLines := collectFlexLines(&constants, availableSpace, flexItems)

	// Determine container main size and re-resolve gaps.
	if mainInner := sizeOptionMain(constants.nodeInnerSize, constants.dir); mainInner.IsSome {
		outerMainSize := mainInner.Value + rectMainAxisSum(constants.contentBoxInset, constants.dir)
		sizeSetMain(&constants.innerContainerSize, constants.dir, mainInner.Value)
		sizeSetMain(&constants.containerSize, constants.dir, outerMainSize)
	} else {
		determineContainerMainSize(tree, availableSpace, flexLines, &constants)
		sizeOptionSetMain(&constants.nodeInnerSize, constants.dir, Some(sizeMain(constants.innerContainerSize, constants.dir)))
		sizeOptionSetMain(&constants.nodeOuterSize, constants.dir, Some(sizeMain(constants.containerSize, constants.dir)))

		// Re-resolve percentage gaps.
		style := tree.getFlexboxContainerStyle(node)
		calc := tree.resolveCalcValue
		innerContainerSizeMain := sizeMain(constants.innerContainerSize, constants.dir)
		mainGapLP := func() LengthPercentage {
			if constants.dir.IsRow() {
				return style.gap().Width
			}
			return style.gap().Height
		}()
		newGap := MaybeResolveLPOption(mainGapLP, Some(innerContainerSizeMain), calc).UnwrapOr(0.0)
		sizeSetMain(&constants.gap, constants.dir, newGap)
	}

	// 6. Resolve the flexible lengths.
	for _, line := range flexLines {
		resolveFlexibleLengths(line, &constants)
	}

	// 9.4. Cross Size Determination
	// 7. Determine the hypothetical cross size of each item.
	for _, line := range flexLines {
		determineHypotheticalCrossSize(tree, line, &constants, availableSpace)
	}

	// Calculate child baselines.
	calculateChildrenBaseLines(tree, knownDimensions, availableSpace, flexLines, &constants)

	// 8. Calculate the cross size of each flex line.
	calculateCrossSize(flexLines, knownDimensions, &constants)

	// 9. Handle 'align-content: stretch'.
	handleAlignContentStretch(flexLines, knownDimensions, &constants)

	// 11. Determine the used cross size of each flex item.
	determineUsedCrossSize(tree, flexLines, &constants)

	// 9.5. Main-Axis Alignment
	// 12. Distribute any remaining free space.
	distributeRemainingFreeSpace(flexLines, &constants)

	// 9.6. Cross-Axis Alignment
	// 13. Resolve cross-axis auto margins.
	resolveCrossAxisAutoMargins(flexLines, &constants)

	// 15. Determine the flex container's used cross size.
	totalLineCrossSize := determineContainerCrossSize(flexLines, knownDimensions, &constants)

	if runMode == RunModeComputeSize {
		return LayoutOutputFromOuterSize(constants.containerSize)
	}

	// 16. Align all flex lines per align-content.
	alignFlexLinesPerAlignContent(flexLines, &constants, totalLineCrossSize)

	// Do a final layout pass.
	inflowContentSize := finalLayoutPass(tree, flexLines, &constants)

	// Perform absolute layout on absolute children.
	absoluteContentSize := performAbsoluteLayoutOnAbsoluteChildren(tree, node, &constants)

	// Hidden layout for display:none children.
	childLen := tree.childCount(node)
	for order := 0; order < childLen; order++ {
		child := tree.getChildId(node, order)
		if tree.getFlexboxChildStyle(child).boxGenerationMode() == BoxGenerationModeNone {
			tree.setUnroundedLayout(child, &Layout{Order: uint32(order)})
			performChildLayout(
				tree, child,
				SizeOptionNone,
				SizeOptionNone,
				Size[AvailableSpace]{Width: AvailableSpaceMaxContentValue, Height: AvailableSpaceMaxContentValue},
				SizingModeInherentSize,
				LineFalseBool,
			)
		}
	}

	// 8.5. Calculate flex container baselines.
	var firstVerticalBaseline Option[float32]
	if len(flexLines) > 0 {
		var baselineChild *flexItem
		for _, item := range flexLines[0].items {
			if constants.isColumn || item.alignSelf == AlignItemsBaseline {
				baselineChild = item
				break
			}
		}
		if baselineChild == nil && len(flexLines[0].items) > 0 {
			baselineChild = flexLines[0].items[0]
		}
		if baselineChild != nil {
			offsetVertical := baselineChild.offsetCross
			if !constants.isRow {
				offsetVertical = baselineChild.offsetMain
			}
			firstVerticalBaseline = Some(offsetVertical + baselineChild.baseline)
		}
	}

	return LayoutOutputFromSizesAndBaselines(
		constants.containerSize,
		SizeF32Max(inflowContentSize, absoluteContentSize),
		Point[Option[float32]]{X: None[float32](), Y: firstVerticalBaseline},
	)
}

// ---------------------------------------------------------------------------
// computeConstants
// ---------------------------------------------------------------------------

func computeConstants(
	tree layoutFlexboxContainer,
	style *Style,
	knownDimensions Size[Option[float32]],
	parentSize Size[Option[float32]],
) algoConstants {
	calc := tree.resolveCalcValue

	dir := style.flexDirection()
	isRow := dir.IsRow()
	isColumn := dir.IsColumn()
	isWrap := style.flexWrap() == FlexWrapWrap || style.flexWrap() == FlexWrapWrapReverse
	isWrapReverse := style.flexWrap() == FlexWrapWrapReverse

	aspectRatio := style.aspectRatio()
	margin := resolveRectLPAWidth(style.margin(), parentSize.Width, calc)
	padding := resolveRectLPWidth(style.padding(), parentSize.Width, calc)
	border := resolveRectLPWidth(style.border(), parentSize.Width, calc)
	paddingBorderSum := SizeAddF32(RectSumAxes(padding), RectSumAxes(border))
	boxSizingAdjustment := SizeZeroF32
	if style.boxSizing() == BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSum
	}

	alignItems := style.alignItems().UnwrapOr(AlignItemsStretch)
	alignContent := style.alignContent().UnwrapOr(AlignContentStretch)
	justifyContent := style.justifyContent()
	layoutDirection := style.direction()

	// Scrollbar gutters.
	overflowTransposed := style.overflow().Transpose()
	scrollbarGutter := Point[float32]{
		X: func() float32 {
			if overflowTransposed.X == OverflowScroll {
				return style.scrollbarWidth()
			}
			return 0.0
		}(),
		Y: func() float32 {
			if overflowTransposed.Y == OverflowScroll {
				return style.scrollbarWidth()
			}
			return 0.0
		}(),
	}

	contentBoxInset := RectAddF32(padding, border)
	contentBoxInset.Bottom += scrollbarGutter.Y
	switch layoutDirection {
	case DirectionLTR:
		contentBoxInset.Right += scrollbarGutter.X
	case DirectionRTL:
		contentBoxInset.Left += scrollbarGutter.X
	}

	nodeOuterSize := knownDimensions
	nodeInnerSize := SizeMaybeSub(nodeOuterSize, Size[Option[float32]]{
		Width:  Some(RectHorizontalAxisSum(contentBoxInset)),
		Height: Some(RectVerticalAxisSum(contentBoxInset)),
	})

	nodeInnerSizeOr := SizeOptionOr(nodeInnerSize, Size[Option[float32]]{Width: Some[float32](0), Height: Some[float32](0)})
	gap := Size[float32]{
		Width:  MaybeResolveLPOption(style.gap().Width, nodeInnerSizeOr.Width, calc).UnwrapOr(0.0),
		Height: MaybeResolveLPOption(style.gap().Height, nodeInnerSizeOr.Height, calc).UnwrapOr(0.0),
	}

	return algoConstants{
		dir:             dir,
		layoutDirection: layoutDirection,
		isRow:           isRow,
		isColumn:        isColumn,
		isWrap:          isWrap,
		isWrapReverse:   isWrapReverse,
		minSize: SizeMaybeAdd(
			SizeOptionMaybeApplyAspectRatio(
				MaybeResolveSizeDimOption(style.minSize(), parentSize, calc),
				aspectRatio,
			),
			Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)},
		),
		maxSize: SizeMaybeAdd(
			SizeOptionMaybeApplyAspectRatio(
				MaybeResolveSizeDimOption(style.maxSize(), parentSize, calc),
				aspectRatio,
			),
			Size[Option[float32]]{Width: Some(boxSizingAdjustment.Width), Height: Some(boxSizingAdjustment.Height)},
		),
		margin:             margin,
		border:             border,
		gap:                gap,
		contentBoxInset:    contentBoxInset,
		scrollbarGutter:    scrollbarGutter,
		alignItems:         alignItems,
		alignContent:       alignContent,
		justifyContent:     justifyContent,
		nodeOuterSize:      nodeOuterSize,
		nodeInnerSize:      nodeInnerSize,
		containerSize:      SizeZeroF32,
		innerContainerSize: SizeZeroF32,
	}
}

// ---------------------------------------------------------------------------
// generateAnonymousFlexItems
// ---------------------------------------------------------------------------

func generateAnonymousFlexItems(
	tree layoutFlexboxContainer,
	node NodeId,
	constants *algoConstants,
) []*flexItem {
	calc := tree.resolveCalcValue
	childIds := tree.childIds(node)
	items := make([]*flexItem, 0, len(childIds))

	for index, child := range childIds {
		childStyle := tree.getFlexboxChildStyle(child)
		if childStyle.position() == PositionAbsolute {
			continue
		}
		if childStyle.boxGenerationMode() == BoxGenerationModeNone {
			continue
		}

		aspectRatio := childStyle.aspectRatio()
		padding := resolveRectLPWidth(childStyle.padding(), constants.nodeInnerSize.Width, calc)
		border := resolveRectLPWidth(childStyle.border(), constants.nodeInnerSize.Width, calc)
		pbSum := SizeAddF32(RectSumAxes(padding), RectSumAxes(border))
		boxSizingAdj := SizeZeroF32
		if childStyle.boxSizing() == BoxSizingContentBox {
			boxSizingAdj = pbSum
		}

		items = append(items, &flexItem{
			node:  child,
			order: uint32(index),

			size: SizeMaybeAdd(
				SizeOptionMaybeApplyAspectRatio(
					MaybeResolveSizeDimOption(childStyle.size(), constants.nodeInnerSize, calc),
					aspectRatio,
				),
				Size[Option[float32]]{Width: Some(boxSizingAdj.Width), Height: Some(boxSizingAdj.Height)},
			),
			minSize: SizeMaybeAdd(
				SizeOptionMaybeApplyAspectRatio(
					MaybeResolveSizeDimOption(childStyle.minSize(), constants.nodeInnerSize, calc),
					aspectRatio,
				),
				Size[Option[float32]]{Width: Some(boxSizingAdj.Width), Height: Some(boxSizingAdj.Height)},
			),
			maxSize: SizeMaybeAdd(
				SizeOptionMaybeApplyAspectRatio(
					MaybeResolveSizeDimOption(childStyle.maxSize(), constants.nodeInnerSize, calc),
					aspectRatio,
				),
				Size[Option[float32]]{Width: Some(boxSizingAdj.Width), Height: Some(boxSizingAdj.Height)},
			),

			inset: RectZipSize(childStyle.inset(), constants.nodeInnerSize,
				func(p LengthPercentageAuto, s Option[float32]) Option[float32] {
					return MaybeResolveLPAOption(p, s, calc)
				},
			),
			margin:       resolveRectLPAWidth(childStyle.margin(), constants.nodeInnerSize.Width, calc),
			marginIsAuto: RectMapTo(childStyle.margin(), func(m LengthPercentageAuto) bool { return m.IsAuto() }),
			padding:      padding,
			border:       border,
			alignSelf:    childStyle.alignSelf().UnwrapOr(constants.alignItems),
			overflow:     childStyle.overflow(),
			scrollbarWidth: childStyle.scrollbarWidth(),
			flexGrow:     childStyle.flexGrow(),
			flexShrink:   childStyle.flexShrink(),

			flexBasis:      0.0,
			innerFlexBasis: 0.0,
			violation:      0.0,
			frozen:         false,

			resolvedMinimumMainSize: 0.0,
			hypotheticalInnerSize:   SizeZeroF32,
			hypotheticalOuterSize:   SizeZeroF32,
			targetSize:             SizeZeroF32,
			outerTargetSize:        SizeZeroF32,
			contentFlexFraction:    0.0,

			baseline:    0.0,
			offsetMain:  0.0,
			offsetCross: 0.0,
		})
	}
	return items
}

// ---------------------------------------------------------------------------
// determineAvailableSpace
// ---------------------------------------------------------------------------

func determineAvailableSpace(
	knownDimensions Size[Option[float32]],
	outerAvailableSpace Size[AvailableSpace],
	constants *algoConstants,
) Size[AvailableSpace] {
	var width AvailableSpace
	if knownDimensions.Width.IsSome {
		width = AvailableSpaceFromDefinite(knownDimensions.Width.Value - RectHorizontalAxisSum(constants.contentBoxInset))
	} else {
		width = AsMaybeSubF32(
			AsMaybeSubF32(outerAvailableSpace.Width, RectHorizontalAxisSum(constants.margin)),
			RectHorizontalAxisSum(constants.contentBoxInset),
		)
	}

	var height AvailableSpace
	if knownDimensions.Height.IsSome {
		height = AvailableSpaceFromDefinite(knownDimensions.Height.Value - RectVerticalAxisSum(constants.contentBoxInset))
	} else {
		height = AsMaybeSubF32(
			AsMaybeSubF32(outerAvailableSpace.Height, RectVerticalAxisSum(constants.margin)),
			RectVerticalAxisSum(constants.contentBoxInset),
		)
	}

	return Size[AvailableSpace]{Width: width, Height: height}
}

// ---------------------------------------------------------------------------
// determineFlexBaseSize
// ---------------------------------------------------------------------------

func determineFlexBaseSize(
	tree layoutFlexboxContainer,
	constants *algoConstants,
	availableSpace Size[AvailableSpace],
	flexItems []*flexItem,
) {
	dir := constants.dir
	calc := tree.resolveCalcValue

	for _, child := range flexItems {
		childStyle := tree.getFlexboxChildStyle(child.node)

		// Parent size for child sizing.
		crossAxisParentSize := sizeOptionCross(constants.nodeInnerSize, dir)
		childParentSize := sizeFromCross(dir, crossAxisParentSize)

		// Available space for child sizing.
		crossAxisMarginSum := rectCrossAxisSum(constants.margin, dir)
		childMinCross := maybeAddF32IfSome(sizeOptionCross(child.minSize, dir), crossAxisMarginSum)
		childMaxCross := maybeAddF32IfSome(sizeOptionCross(child.maxSize, dir), crossAxisMarginSum)

		// Clamp available space by min- and max- size.
		var crossAxisAvailableSpace AvailableSpace
		switch crossAv := avCross(availableSpace, dir); crossAv.Tag {
		case AvailableSpaceDefinite:
			val := crossAxisParentSize.UnwrapOr(crossAv.Value)
			crossAxisAvailableSpace = AvailableSpaceFromDefinite(maybeClampF32(val, childMinCross, childMaxCross))
		case AvailableSpaceMinContent:
			if childMinCross.IsSome {
				crossAxisAvailableSpace = AvailableSpaceFromDefinite(childMinCross.Value)
			} else {
				crossAxisAvailableSpace = AvailableSpaceMinContentValue
			}
		default: // MaxContent
			if childMaxCross.IsSome {
				crossAxisAvailableSpace = AvailableSpaceFromDefinite(childMaxCross.Value)
			} else {
				crossAxisAvailableSpace = AvailableSpaceMaxContentValue
			}
		}

		// Known dimensions for child sizing.
		childKnownDimensions := sizeOptionWithMain(child.size, dir, None[float32]())
		if child.alignSelf == AlignItemsStretch &&
			!rectBoolCrossStart(child.marginIsAuto, constants.dir) &&
			!rectBoolCrossEnd(child.marginIsAuto, constants.dir) &&
			!sizeOptionCross(childKnownDimensions, dir).IsSome {
			sizeSetCross(&childKnownDimensions, dir,
				maybeSubOption(crossAxisAvailableSpace.IntoOption(), Some(rectCrossAxisSum(child.margin, dir))),
			)
		}

		containerWidth := sizeOptionMain(constants.nodeInnerSize, dir)
		var boxSizingAdj float32
		if childStyle.boxSizing() == BoxSizingContentBox {
			p := resolveRectLPWidth(childStyle.padding(), containerWidth, calc)
			b := resolveRectLPWidth(childStyle.border(), containerWidth, calc)
			pbS := SizeAddF32(RectSumAxes(p), RectSumAxes(b))
			boxSizingAdj = sizeMain(pbS, dir)
		}
		flexBasis := maybeAddF32IfSome(
			MaybeResolveDimOption(childStyle.flexBasis(), containerWidth, calc),
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
					Size[AvailableSpace]{Width: AvailableSpaceMaxContentValue, Height: AvailableSpaceMaxContentValue},
					dir,
					func() AvailableSpace {
						if avMain(availableSpace, dir).Tag == AvailableSpaceMinContent {
							return AvailableSpaceMinContentValue
						}
						return AvailableSpaceMaxContentValue
					}(),
				),
				dir, crossAxisAvailableSpace,
			)

			child.flexBasis = measureChildSize(
				tree, child.node,
				childKnownDimensions, childParentSize, childAvailableSpace,
				SizingModeContentSize, dir.MainAxis(), LineFalseBool,
			)
		}

		// Floor flex-basis by padding_border_sum.
		paddingBorderSum := rectMainAxisSum(child.padding, constants.dir) + rectMainAxisSum(child.border, constants.dir)
		if child.flexBasis < paddingBorderSum {
			child.flexBasis = paddingBorderSum
		}

		// Hypothetical main size.
		child.innerFlexBasis = child.flexBasis - rectMainAxisSum(child.padding, constants.dir) - rectMainAxisSum(child.border, constants.dir)

		paddingBorderAxesSums := Size[Option[float32]]{
			Width:  Some(RectSumAxes(RectAddF32(child.padding, child.border)).Width),
			Height: Some(RectSumAxes(RectAddF32(child.padding, child.border)).Height),
		}

		// Resolve minimum main size.
		overflowMin := SizeZipMap(
			PointToSize(child.overflow),
			Size[Overflow]{Width: OverflowVisible, Height: OverflowVisible}, // placeholder
			func(o Overflow, _ Overflow) Option[float32] {
				return o.MaybeIntoAutomaticMinSize()
			},
		)
		styleMinMainSize := sizeOptionMain(
			SizeOptionOr(child.minSize, overflowMin),
			dir,
		)

		if styleMinMainSize.IsSome {
			child.resolvedMinimumMainSize = styleMinMainSize.Value
		} else {
			// Compute min-content main size.
			childAvailableSpaceMin := avWithCross(
				Size[AvailableSpace]{Width: AvailableSpaceMinContentValue, Height: AvailableSpaceMinContentValue},
				dir, crossAxisAvailableSpace,
			)
			minContentMainSize := measureChildSize(
				tree, child.node,
				childKnownDimensions, childParentSize, childAvailableSpaceMin,
				SizingModeContentSize, dir.MainAxis(), LineFalseBool,
			)

			// 4.5. Automatic Minimum Size of Flex Items
			clampedMinContentSize := maybeMinF32(minContentMainSize, sizeOptionMain(child.size, dir))
			clampedMinContentSize = maybeMinF32(clampedMinContentSize, sizeOptionMain(child.maxSize, dir))
			child.resolvedMinimumMainSize = maybeMaxF32(clampedMinContentSize, sizeOptionMain(paddingBorderAxesSums, dir))
		}

		hypotheticalInnerMinMain := maybeMaxF32(child.resolvedMinimumMainSize, sizeOptionMain(paddingBorderAxesSums, constants.dir))
		hypotheticalInnerSize := maybeClampF32(child.flexBasis, Some(hypotheticalInnerMinMain), sizeOptionMain(child.maxSize, constants.dir))
		hypotheticalOuterSize := hypotheticalInnerSize + rectMainAxisSum(child.margin, constants.dir)

		sizeSetMain(&child.hypotheticalInnerSize, constants.dir, hypotheticalInnerSize)
		sizeSetMain(&child.hypotheticalOuterSize, constants.dir, hypotheticalOuterSize)
	}
}

// maybeAddF32IfSome adds addend to opt if opt is Some, returns opt unchanged if addend is 0 or opt is None.
func maybeAddF32IfSome(opt Option[float32], addend float32) Option[float32] {
	if !opt.IsSome {
		return opt
	}
	return Some(opt.Value + addend)
}

// ---------------------------------------------------------------------------
// collectFlexLines
// ---------------------------------------------------------------------------

func collectFlexLines(
	constants *algoConstants,
	availableSpace Size[AvailableSpace],
	flexItems []*flexItem,
) []*flexLine {
	if !constants.isWrap {
		lines := make([]*flexLine, 0, 1)
		lines = append(lines, &flexLine{items: flexItems, crossSize: 0.0, offsetCross: 0.0})
		return lines
	}

	mainAxisAvailableSpace := func() AvailableSpace {
		maxMain := sizeOptionMain(constants.maxSize, constants.dir)
		if maxMain.IsSome {
			defVal := avMain(availableSpace, constants.dir).IntoOption().UnwrapOr(maxMain.Value)
			clamped := maybeMaxF32(defVal, sizeOptionMain(constants.minSize, constants.dir))
			return AvailableSpaceFromDefinite(clamped)
		}
		return avMain(availableSpace, constants.dir)
	}()

	switch mainAxisAvailableSpace.Tag {
	case AvailableSpaceMaxContent:
		// Items will never wrap.
		lines := make([]*flexLine, 0, 1)
		lines = append(lines, &flexLine{items: flexItems, crossSize: 0.0, offsetCross: 0.0})
		return lines

	case AvailableSpaceMinContent:
		// Each item in its own line.
		lines := make([]*flexLine, 0, len(flexItems))
		for _, item := range flexItems {
			lines = append(lines, &flexLine{
				items:     []*flexItem{item},
				crossSize: 0.0, offsetCross: 0.0,
			})
		}
		return lines

	default: // Definite
		mainAvail := mainAxisAvailableSpace.Value
		lines := make([]*flexLine, 0, 1)
		mainAxisGap := sizeMain(constants.gap, constants.dir)
		remaining := flexItems

		for len(remaining) > 0 {
			lineLength := float32(0.0)
			splitIdx := len(remaining)

			for idx, child := range remaining {
				gapContribution := float32(0.0)
				if idx != 0 {
					gapContribution = mainAxisGap
				}
				lineLength += sizeMain(child.hypotheticalOuterSize, constants.dir) + gapContribution
				if lineLength > mainAvail && idx != 0 {
					splitIdx = idx
					break
				}
			}

			lines = append(lines, &flexLine{
				items:     remaining[:splitIdx],
				crossSize: 0.0, offsetCross: 0.0,
			})
			remaining = remaining[splitIdx:]
		}
		return lines
	}
}

// ---------------------------------------------------------------------------
// determineContainerMainSize
// ---------------------------------------------------------------------------

func determineContainerMainSize(
	tree layoutFlexboxContainer,
	availableSpace Size[AvailableSpace],
	lines []*flexLine,
	constants *algoConstants,
) {
	dir := constants.dir
	mainContentBoxInset := rectMainAxisSum(constants.contentBoxInset, dir)

	outerMainSize := func() float32 {
		if nodeMain := sizeOptionMain(constants.nodeOuterSize, dir); nodeMain.IsSome {
			return nodeMain.Value
		}

		switch avMain(availableSpace, dir).Tag {
		case AvailableSpaceDefinite:
			mainAxisAvailSpace := avMain(availableSpace, dir).Value
			longestLineLength := float32(0.0)
			for _, line := range lines {
				lineMainAxisGap := sumAxisGaps(sizeMain(constants.gap, dir), len(line.items))
				totalTargetSize := float32(0.0)
				for _, child := range line.items {
					paddingBorderSum := rectMainAxisSum(RectAddF32(child.padding, child.border), dir)
					v := maybeMaxF32(child.flexBasis, sizeOptionMain(child.minSize, dir)) + rectMainAxisSum(child.margin, dir)
					if v < paddingBorderSum {
						v = paddingBorderSum
					}
					totalTargetSize += v
				}
				lineTotal := totalTargetSize + lineMainAxisGap
				if lineTotal > longestLineLength {
					longestLineLength = lineTotal
				}
			}
			size := longestLineLength + mainContentBoxInset
			if len(lines) > 1 {
				return f32Max(size, mainAxisAvailSpace)
			}
			return size

		case AvailableSpaceMinContent:
			if constants.isWrap {
				longestLineLength := float32(0.0)
				for _, line := range lines {
					lineMainAxisGap := sumAxisGaps(sizeMain(constants.gap, dir), len(line.items))
					totalTargetSize := float32(0.0)
					for _, child := range line.items {
						paddingBorderSum := rectMainAxisSum(RectAddF32(child.padding, child.border), dir)
						v := maybeMaxF32(child.flexBasis, sizeOptionMain(child.minSize, dir)) + rectMainAxisSum(child.margin, dir)
						if v < paddingBorderSum {
							v = paddingBorderSum
						}
						totalTargetSize += v
					}
					lineTotal := totalTargetSize + lineMainAxisGap
					if lineTotal > longestLineLength {
						longestLineLength = lineTotal
					}
				}
				return longestLineLength + mainContentBoxInset
			}
			fallthrough

		default: // MinContent (non-wrap) or MaxContent
			var mainSize float32

			for _, line := range lines {
				for _, item := range line.items {
					styleMin := sizeOptionMain(item.minSize, dir)
					stylePreferred := sizeOptionMain(item.size, dir)
					styleMax := sizeOptionMain(item.maxSize, dir)

					clampingBasis := maybeMaxOption(Some(item.flexBasis), stylePreferred)
					flexBasisMin := clampingBasis
					if item.flexShrink != 0.0 {
						flexBasisMin = None[float32]()
					}
					flexBasisMax := clampingBasis
					if item.flexGrow != 0.0 {
						flexBasisMax = None[float32]()
					}

					minMainSize := func() float32 {
						v := maybeMaxOption(styleMin, flexBasisMin)
						v = v.Or(flexBasisMin)
						result := v.UnwrapOr(item.resolvedMinimumMainSize)
						if result < item.resolvedMinimumMainSize {
							result = item.resolvedMinimumMainSize
						}
						return result
					}()
					maxMainSize := func() float32 {
						v := maybeMinOption(styleMax, flexBasisMax)
						v = v.Or(flexBasisMax)
						return v.UnwrapOr(float32(math.Inf(1)))
					}()

					contentContribution := func() float32 {
						// Short-circuit: max <= min or max <= pref
						if stylePreferred.IsSome && (maxMainSize <= minMainSize || maxMainSize <= stylePreferred.Value) {
							pref := stylePreferred.Value
							clamped := pref
							if clamped > maxMainSize {
								clamped = maxMainSize
							}
							if clamped < minMainSize {
								clamped = minMainSize
							}
							return clamped + rectMainAxisSum(item.margin, dir)
						}
						if maxMainSize <= minMainSize {
							return minMainSize + rectMainAxisSum(item.margin, dir)
						}

						if item.isScrollContainer() {
							return item.flexBasis + rectMainAxisSum(item.margin, dir)
						}

						// Else compute content size.
						crossAxisParentSize := sizeOptionCross(constants.nodeInnerSize, dir)
						crossAxisMarginSum := rectCrossAxisSum(constants.margin, dir)
						childMinCross := maybeAddF32IfSome(sizeOptionCross(item.minSize, dir), crossAxisMarginSum)
						childMaxCross := maybeAddF32IfSome(sizeOptionCross(item.maxSize, dir), crossAxisMarginSum)
						crossAxisAvailableSpace := avCross(availableSpace, dir).
							MapDefiniteValue(func(val float32) float32 {
								return crossAxisParentSize.UnwrapOr(val)
							})
						crossAxisAvailableSpace = asClampOption(crossAxisAvailableSpace, childMinCross, childMaxCross)

						childAvailableSpace := avWithCross(availableSpace, dir, crossAxisAvailableSpace)

						childKnownDimensions := sizeOptionWithMain(item.size, dir, None[float32]())
						if item.alignSelf == AlignItemsStretch && !sizeOptionCross(childKnownDimensions, dir).IsSome {
							sizeSetCross(&childKnownDimensions, dir,
								maybeSubOption(crossAxisAvailableSpace.IntoOption(), Some(rectCrossAxisSum(item.margin, dir))),
							)
						}

						contentMainSize := measureChildSize(
							tree, item.node,
							childKnownDimensions,
							constants.nodeInnerSize,
							childAvailableSpace,
							SizingModeInherentSize,
							dir.MainAxis(),
							LineFalseBool,
						) + rectMainAxisSum(item.margin, dir)

						if constants.isRow {
							return f32Max(maybeClampF32(contentMainSize, styleMin, styleMax), mainContentBoxInset)
						}
						return f32Max(maybeClampF32(f32Max(contentMainSize, item.flexBasis), styleMin, styleMax), mainContentBoxInset)
					}()

					// content_flex_fraction
					diff := contentContribution - item.flexBasis
					if diff > 0 {
						item.contentFlexFraction = diff / f32Max(1.0, item.flexGrow)
					} else if diff < 0 {
						scaledShrinkFactor := f32Max(1.0, item.flexShrink*item.innerFlexBasis)
						item.contentFlexFraction = diff / scaledShrinkFactor
					} else {
						item.contentFlexFraction = 0.0
					}
				}

				// Add each item's contribution.
				itemMainSizeSum := float32(0.0)
				for _, item := range line.items {
					flexFraction := item.contentFlexFraction
					var flexContribution float32
					if item.contentFlexFraction > 0 {
						flexContribution = f32Max(1.0, item.flexGrow) * flexFraction
					} else if item.contentFlexFraction < 0 {
						scaledShrinkFactor := f32Max(1.0, item.flexShrink) * item.innerFlexBasis
						flexContribution = scaledShrinkFactor * flexFraction
					}
					size := item.flexBasis + flexContribution
					sizeSetMain(&item.outerTargetSize, constants.dir, size)
					sizeSetMain(&item.targetSize, constants.dir, size)
					itemMainSizeSum += size
				}

				gapSum := sumAxisGaps(sizeMain(constants.gap, dir), len(line.items))
				mainSize = f32Max(mainSize, itemMainSizeSum+gapSum)
			}
			return mainSize + mainContentBoxInset
		}
	}()

	outerMainSize = maybeClampF32(outerMainSize, sizeOptionMain(constants.minSize, dir), sizeOptionMain(constants.maxSize, dir))
	scrollbarGutterMain := pointMain(constants.scrollbarGutter, dir)
	if outerMainSize < mainContentBoxInset-scrollbarGutterMain {
		outerMainSize = mainContentBoxInset - scrollbarGutterMain
	}

	innerMainSize := f32Max(outerMainSize-mainContentBoxInset, 0.0)
	sizeSetMain(&constants.containerSize, dir, outerMainSize)
	sizeSetMain(&constants.innerContainerSize, dir, innerMainSize)
	sizeOptionSetMain(&constants.nodeInnerSize, dir, Some(innerMainSize))
}

// ---------------------------------------------------------------------------
// resolveFlexibleLengths
// ---------------------------------------------------------------------------

func resolveFlexibleLengths(line *flexLine, constants *algoConstants) {
	totalMainAxisGap := sumAxisGaps(sizeMain(constants.gap, constants.dir), len(line.items))

	// 1. Determine the used flex factor.
	totalHypotheticalOuterMainSize := float32(0.0)
	for _, child := range line.items {
		totalHypotheticalOuterMainSize += sizeMain(child.hypotheticalOuterSize, constants.dir)
	}
	usedFlexFactor := totalMainAxisGap + totalHypotheticalOuterMainSize
	nodeInnerMain := sizeOptionMain(constants.nodeInnerSize, constants.dir).UnwrapOr(0.0)
	growing := usedFlexFactor < nodeInnerMain
	shrinking := usedFlexFactor > nodeInnerMain
	exactlySized := !growing && !shrinking

	// 2. Size inflexible items.
	for _, child := range line.items {
		innerTargetSize := sizeMain(child.hypotheticalInnerSize, constants.dir)
		sizeSetMain(&child.targetSize, constants.dir, innerTargetSize)

		if exactlySized ||
			(child.flexGrow == 0 && child.flexShrink == 0) ||
			(growing && child.flexBasis > sizeMain(child.hypotheticalInnerSize, constants.dir)) ||
			(shrinking && child.flexBasis < sizeMain(child.hypotheticalInnerSize, constants.dir)) {
			child.frozen = true
			outerTarget := innerTargetSize + rectMainAxisSum(child.margin, constants.dir)
			sizeSetMain(&child.outerTargetSize, constants.dir, outerTarget)
		}
	}

	if exactlySized {
		return
	}

	// 3. Calculate initial free space.
	usedSpace := totalMainAxisGap
	for _, child := range line.items {
		if child.frozen {
			usedSpace += sizeMain(child.outerTargetSize, constants.dir)
		} else {
			usedSpace += child.flexBasis + rectMainAxisSum(child.margin, constants.dir)
		}
	}

	initialFreeSpace := maybeSubF32(sizeOptionMain(constants.nodeInnerSize, constants.dir).UnwrapOr(0.0),
		Some(usedSpace))

	// 4. Loop.
	for {
		// a. Check for flexible items.
		allFrozen := true
		for _, child := range line.items {
			if !child.frozen {
				allFrozen = false
				break
			}
		}
		if allFrozen {
			break
		}

		// b. Calculate remaining free space.
		usedSpace = totalMainAxisGap
		for _, child := range line.items {
			if child.frozen {
				usedSpace += sizeMain(child.outerTargetSize, constants.dir)
			} else {
				usedSpace += child.flexBasis + rectMainAxisSum(child.margin, constants.dir)
			}
		}

		var unfrozen []*flexItem
		var sumFlexGrow, sumFlexShrink float32
		for _, child := range line.items {
			if !child.frozen {
				unfrozen = append(unfrozen, child)
				sumFlexGrow += child.flexGrow
				sumFlexShrink += child.flexShrink
			}
		}

		nodeInnerMainOpt := sizeOptionMain(constants.nodeInnerSize, constants.dir)
		var freeSpace float32
		if growing && sumFlexGrow < 1.0 {
			freeSpace = maybeMinF32(
				initialFreeSpace*sumFlexGrow-totalMainAxisGap,
				maybeSubOption(nodeInnerMainOpt, Some(usedSpace)),
			)
		} else if shrinking && sumFlexShrink < 1.0 {
			freeSpace = maybeMaxF32(
				initialFreeSpace*sumFlexShrink-totalMainAxisGap,
				maybeSubOption(nodeInnerMainOpt, Some(usedSpace)),
			)
		} else {
			maybeFS := maybeSubOption(nodeInnerMainOpt, Some(usedSpace))
			freeSpace = maybeFS.UnwrapOr(usedFlexFactor - usedSpace)
		}

		// c. Distribute free space.
		if isNormal(freeSpace) {
			if growing && sumFlexGrow > 0.0 {
				for _, child := range unfrozen {
					sizeSetMain(&child.targetSize, constants.dir,
						child.flexBasis+freeSpace*(child.flexGrow/sumFlexGrow))
				}
			} else if shrinking && sumFlexShrink > 0.0 {
				sumScaledShrinkFactor := float32(0.0)
				for _, child := range unfrozen {
					sumScaledShrinkFactor += child.innerFlexBasis * child.flexShrink
				}
				if sumScaledShrinkFactor > 0.0 {
					for _, child := range unfrozen {
						scaledShrinkFactor := child.innerFlexBasis * child.flexShrink
						sizeSetMain(&child.targetSize, constants.dir,
							child.flexBasis+freeSpace*(scaledShrinkFactor/sumScaledShrinkFactor))
					}
				}
			}
		}

		// d. Fix min/max violations.
		totalViolation := float32(0.0)
		for _, child := range unfrozen {
			resolvedMinMain := Some(child.resolvedMinimumMainSize)
			maxMain := sizeOptionMain(child.maxSize, constants.dir)
			clamped := f32Max(maybeClampF32(sizeMain(child.targetSize, constants.dir), resolvedMinMain, maxMain), 0.0)
			child.violation = clamped - sizeMain(child.targetSize, constants.dir)
			sizeSetMain(&child.targetSize, constants.dir, clamped)
			sizeSetMain(&child.outerTargetSize, constants.dir,
				sizeMain(child.targetSize, constants.dir)+rectMainAxisSum(child.margin, constants.dir))
			totalViolation += child.violation
		}

		// e. Freeze over-flexed items.
		for _, child := range unfrozen {
			if totalViolation > 0.0 {
				child.frozen = child.violation > 0.0
			} else if totalViolation < 0.0 {
				child.frozen = child.violation < 0.0
			} else {
				child.frozen = true
			}
		}
	}
}

// isNormal returns true if f is a normal float (not zero, subnormal, inf, or NaN).
// This matches Rust's f32::is_normal().
func isNormal(f float32) bool {
	bits := math.Float32bits(f)
	exp := (bits >> 23) & 0xFF
	// Normal numbers have exponent in [1, 254] (not 0 = zero/subnormal, not 255 = inf/NaN)
	return exp != 0 && exp != 0xFF
}

// ---------------------------------------------------------------------------
// determineHypotheticalCrossSize
// ---------------------------------------------------------------------------

func determineHypotheticalCrossSize(
	tree layoutFlexboxContainer,
	line *flexLine,
	constants *algoConstants,
	availableSpace Size[AvailableSpace],
) {
	for _, child := range line.items {
		paddingBorderSum := rectCrossAxisSum(RectAddF32(child.padding, child.border), constants.dir)

		childKnownMain := AvailableSpaceFromDefinite(sizeMain(constants.containerSize, constants.dir))

		childCross := maybeMaxF32(
			maybeClampF32(
				sizeOptionCross(child.size, constants.dir).UnwrapOr(0),
				sizeOptionCross(child.minSize, constants.dir),
				sizeOptionCross(child.maxSize, constants.dir),
			),
			Some(paddingBorderSum),
		)
		// Only use child cross if it was present in child.size.
		var childCrossOpt Option[float32]
		if sizeOptionCross(child.size, constants.dir).IsSome {
			clampedVal := maybeClampF32(
				sizeOptionCross(child.size, constants.dir).Value,
				sizeOptionCross(child.minSize, constants.dir),
				sizeOptionCross(child.maxSize, constants.dir),
			)
			childCrossOpt = Some(f32Max(clampedVal, paddingBorderSum))
		} else {
			childCrossOpt = None[float32]()
		}

		childAvailableCross := asClampOption(
			AsMaybeMaxF32(avCross(availableSpace, constants.dir), paddingBorderSum),
			sizeOptionCross(child.minSize, constants.dir),
			sizeOptionCross(child.maxSize, constants.dir),
		)

		var childInnerCross float32
		if childCrossOpt.IsSome {
			childInnerCross = childCrossOpt.Value
		} else {
			knownDims := Size[Option[float32]]{
				Width: func() Option[float32] {
					if constants.isRow {
						return Some(sizeMain(child.targetSize, constants.dir))
					}
					return childCrossOpt
				}(),
				Height: func() Option[float32] {
					if constants.isRow {
						return childCrossOpt
					}
					return Some(sizeMain(child.targetSize, constants.dir))
				}(),
			}
			avSpace := Size[AvailableSpace]{
				Width: func() AvailableSpace {
					if constants.isRow {
						return childKnownMain
					}
					return childAvailableCross
				}(),
				Height: func() AvailableSpace {
					if constants.isRow {
						return childAvailableCross
					}
					return childKnownMain
				}(),
			}

			measured := measureChildSize(
				tree, child.node,
				knownDims,
				constants.nodeInnerSize,
				avSpace,
				SizingModeContentSize,
				constants.dir.CrossAxis(),
				LineFalseBool,
			)
			childInnerCross = f32Max(
				maybeClampF32(measured, sizeOptionCross(child.minSize, constants.dir), sizeOptionCross(child.maxSize, constants.dir)),
				paddingBorderSum,
			)
		}

		_ = childCross // suppress unused
		childOuterCross := childInnerCross + rectCrossAxisSum(child.margin, constants.dir)
		sizeSetCross(&child.hypotheticalInnerSize, constants.dir, childInnerCross)
		sizeSetCross(&child.hypotheticalOuterSize, constants.dir, childOuterCross)
	}
}

// ---------------------------------------------------------------------------
// calculateChildrenBaseLines
// ---------------------------------------------------------------------------

func calculateChildrenBaseLines(
	tree layoutFlexboxContainer,
	nodeSize Size[Option[float32]],
	availableSpace Size[AvailableSpace],
	flexLines []*flexLine,
	constants *algoConstants,
) {
	if !constants.isRow {
		return
	}

	for _, line := range flexLines {
		lineBaselineChildCount := 0
		for _, child := range line.items {
			if child.alignSelf == AlignItemsBaseline {
				lineBaselineChildCount++
			}
		}
		if lineBaselineChildCount <= 1 {
			continue
		}

		for _, child := range line.items {
			if child.alignSelf != AlignItemsBaseline {
				continue
			}

			knownDims := Size[Option[float32]]{
				Width: func() Option[float32] {
					if constants.isRow {
						return Some(sizeMain(child.targetSize, constants.dir))
					}
					return Some(sizeCross(child.hypotheticalInnerSize, constants.dir))
				}(),
				Height: func() Option[float32] {
					if constants.isRow {
						return Some(sizeCross(child.hypotheticalInnerSize, constants.dir))
					}
					return Some(sizeMain(child.targetSize, constants.dir))
				}(),
			}

			avSpace := Size[AvailableSpace]{
				Width: func() AvailableSpace {
					if constants.isRow {
						return AvailableSpaceFromDefinite(constants.containerSize.Width)
					}
					return availableSpace.Width.MaybeSet(nodeSize.Width)
				}(),
				Height: func() AvailableSpace {
					if constants.isRow {
						return availableSpace.Height.MaybeSet(nodeSize.Height)
					}
					return AvailableSpaceFromDefinite(constants.containerSize.Height)
				}(),
			}

			result := performChildLayout(
				tree, child.node,
				knownDims,
				constants.nodeInnerSize,
				avSpace,
				SizingModeContentSize,
				LineFalseBool,
			)

			baseline := result.FirstBaselines.Y
			height := result.Size.Height
			child.baseline = baseline.UnwrapOr(height) + child.margin.Top
		}
	}
}

// ---------------------------------------------------------------------------
// calculateCrossSize
// ---------------------------------------------------------------------------

func calculateCrossSize(flexLines []*flexLine, nodeSize Size[Option[float32]], constants *algoConstants) {
	if !constants.isWrap && sizeOptionCross(nodeSize, constants.dir).IsSome {
		crossAxisPaddingBorder := rectCrossAxisSum(constants.contentBoxInset, constants.dir)
		crossMinSize := sizeOptionCross(constants.minSize, constants.dir)
		crossMaxSize := sizeOptionCross(constants.maxSize, constants.dir)
		v := maybeClampF32(
			sizeOptionCross(nodeSize, constants.dir).Value,
			crossMinSize,
			crossMaxSize,
		)
		v = f32Max(v-crossAxisPaddingBorder, 0.0)
		flexLines[0].crossSize = v
	} else {
		for _, line := range flexLines {
			maxBaseline := float32(0.0)
			for _, child := range line.items {
				if child.baseline > maxBaseline {
					maxBaseline = child.baseline
				}
			}

			crossSize := float32(0.0)
			for _, child := range line.items {
				var s float32
				if child.alignSelf == AlignItemsBaseline &&
					!rectBoolCrossStart(child.marginIsAuto, constants.dir) &&
					!rectBoolCrossEnd(child.marginIsAuto, constants.dir) {
					s = maxBaseline - child.baseline + sizeCross(child.hypotheticalOuterSize, constants.dir)
				} else {
					s = sizeCross(child.hypotheticalOuterSize, constants.dir)
				}
				if s > crossSize {
					crossSize = s
				}
			}
			line.crossSize = crossSize
		}

		// Single-line: clamp by min/max.
		if !constants.isWrap {
			crossAxisPaddingBorder := rectCrossAxisSum(constants.contentBoxInset, constants.dir)
			crossMinSize := sizeOptionCross(constants.minSize, constants.dir)
			crossMaxSize := sizeOptionCross(constants.maxSize, constants.dir)
			flexLines[0].crossSize = maybeClampF32(
				flexLines[0].crossSize,
				maybeSubOption(crossMinSize, Some(crossAxisPaddingBorder)),
				maybeSubOption(crossMaxSize, Some(crossAxisPaddingBorder)),
			)
		}
	}
}

// ---------------------------------------------------------------------------
// handleAlignContentStretch
// ---------------------------------------------------------------------------

func handleAlignContentStretch(flexLines []*flexLine, nodeSize Size[Option[float32]], constants *algoConstants) {
	if constants.alignContent != AlignContentStretch {
		return
	}

	crossAxisPaddingBorder := rectCrossAxisSum(constants.contentBoxInset, constants.dir)
	crossMinSize := sizeOptionCross(constants.minSize, constants.dir)
	crossMaxSize := sizeOptionCross(constants.maxSize, constants.dir)

	containerMinInnerCross := func() float32 {
		v := sizeOptionCross(nodeSize, constants.dir).Or(crossMinSize)
		v = maybeClampOption(v, crossMinSize, crossMaxSize)
		v = maybeSubOption(v, Some(crossAxisPaddingBorder))
		v = maybeMaxOption(v, Some[float32](0.0))
		return v.UnwrapOr(0.0)
	}()

	totalCrossAxisGap := sumAxisGaps(sizeCross(constants.gap, constants.dir), len(flexLines))
	linesTotalCross := float32(0.0)
	for _, line := range flexLines {
		linesTotalCross += line.crossSize
	}
	linesTotalCross += totalCrossAxisGap

	if linesTotalCross < containerMinInnerCross {
		remaining := containerMinInnerCross - linesTotalCross
		addition := remaining / float32(len(flexLines))
		for _, line := range flexLines {
			line.crossSize += addition
		}
	}
}

// ---------------------------------------------------------------------------
// determineUsedCrossSize
// ---------------------------------------------------------------------------

func determineUsedCrossSize(
	tree layoutFlexboxContainer,
	flexLines []*flexLine,
	constants *algoConstants,
) {
	calc := tree.resolveCalcValue
	for _, line := range flexLines {
		lineCrossSize := line.crossSize

		for _, child := range line.items {
			childStyle := tree.getFlexboxChildStyle(child.node)

			var crossTargetSize float32
			if child.alignSelf == AlignItemsStretch &&
				!rectBoolCrossStart(child.marginIsAuto, constants.dir) &&
				!rectBoolCrossEnd(child.marginIsAuto, constants.dir) &&
				dimCrossIsAuto(childStyle.size(), constants.dir) {

				// Compute max_size_ignoring_aspect_ratio.
				p := resolveRectLPWidth(childStyle.padding(), constants.nodeInnerSize.Width, calc)
				b := resolveRectLPWidth(childStyle.border(), constants.nodeInnerSize.Width, calc)
				pbSum := SizeAddF32(RectSumAxes(p), RectSumAxes(b))
				boxSizAdj := SizeZeroF32
				if childStyle.boxSizing() == BoxSizingContentBox {
					boxSizAdj = pbSum
				}
				maxSizeIgnoringAR := SizeMaybeAdd(
					MaybeResolveSizeDimOption(childStyle.maxSize(), constants.nodeInnerSize, calc),
					Size[Option[float32]]{Width: Some(boxSizAdj.Width), Height: Some(boxSizAdj.Height)},
				)

				crossTargetSize = maybeClampF32(
					lineCrossSize-rectCrossAxisSum(child.margin, constants.dir),
					sizeOptionCross(child.minSize, constants.dir),
					sizeOptionCross(maxSizeIgnoringAR, constants.dir),
				)
			} else {
				crossTargetSize = sizeCross(child.hypotheticalInnerSize, constants.dir)
			}

			sizeSetCross(&child.targetSize, constants.dir, crossTargetSize)
			sizeSetCross(&child.outerTargetSize, constants.dir,
				sizeCross(child.targetSize, constants.dir)+rectCrossAxisSum(child.margin, constants.dir))
		}
	}
}

// ---------------------------------------------------------------------------
// distributeRemainingFreeSpace
// ---------------------------------------------------------------------------

func distributeRemainingFreeSpace(flexLines []*flexLine, constants *algoConstants) {
	for _, line := range flexLines {
		totalMainAxisGap := sumAxisGaps(sizeMain(constants.gap, constants.dir), len(line.items))
		usedSpace := totalMainAxisGap
		for _, child := range line.items {
			usedSpace += sizeMain(child.outerTargetSize, constants.dir)
		}
		freeSpace := sizeMain(constants.innerContainerSize, constants.dir) - usedSpace
		numAutoMargins := 0

		for _, child := range line.items {
			if rectBoolMainStart(child.marginIsAuto, constants.dir) {
				numAutoMargins++
			}
			if rectBoolMainEnd(child.marginIsAuto, constants.dir) {
				numAutoMargins++
			}
		}

		if freeSpace > 0 && numAutoMargins > 0 {
			marginVal := freeSpace / float32(numAutoMargins)
			for _, child := range line.items {
				if rectBoolMainStart(child.marginIsAuto, constants.dir) {
					if constants.isRow {
						child.margin.Left = marginVal
					} else {
						child.margin.Top = marginVal
					}
				}
				if rectBoolMainEnd(child.marginIsAuto, constants.dir) {
					if constants.isRow {
						child.margin.Right = marginVal
					} else {
						child.margin.Bottom = marginVal
					}
				}
			}
		}

		numItems := len(line.items)
		layoutReverse := constants.dir.IsReverse()
		gap := sizeMain(constants.gap, constants.dir)
		isSafe := false
		rawJCMode := AlignContentFlexStart
		if constants.justifyContent.IsSome {
			rawJCMode = constants.justifyContent.Value
		}
		jcMode := applyAlignmentFallback(freeSpace, numItems, rawJCMode, isSafe)

		if layoutReverse {
			for revI := len(line.items) - 1; revI >= 0; revI-- {
				enumI := len(line.items) - 1 - revI
				child := line.items[revI]
				child.offsetMain = computeAlignmentOffset(freeSpace, numItems, gap, jcMode, layoutReverse, enumI == 0)
			}
		} else {
			for i, child := range line.items {
				child.offsetMain = computeAlignmentOffset(freeSpace, numItems, gap, jcMode, layoutReverse, i == 0)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// resolveCrossAxisAutoMargins
// ---------------------------------------------------------------------------

func resolveCrossAxisAutoMargins(flexLines []*flexLine, constants *algoConstants) {
	for _, line := range flexLines {
		lineCrossSize := line.crossSize
		maxBaseline := float32(0.0)
		for _, child := range line.items {
			if child.baseline > maxBaseline {
				maxBaseline = child.baseline
			}
		}

		for _, child := range line.items {
			freeSpace := lineCrossSize - sizeCross(child.outerTargetSize, constants.dir)

			if rectBoolCrossStart(child.marginIsAuto, constants.dir) && rectBoolCrossEnd(child.marginIsAuto, constants.dir) {
				if constants.isRow {
					child.margin.Top = freeSpace / 2.0
					child.margin.Bottom = freeSpace / 2.0
				} else {
					child.margin.Left = freeSpace / 2.0
					child.margin.Right = freeSpace / 2.0
				}
			} else if rectBoolCrossStart(child.marginIsAuto, constants.dir) {
				if constants.isRow {
					child.margin.Top = freeSpace
				} else {
					child.margin.Left = freeSpace
				}
			} else if rectBoolCrossEnd(child.marginIsAuto, constants.dir) {
				if constants.isRow {
					child.margin.Bottom = freeSpace
				} else {
					child.margin.Right = freeSpace
				}
			} else {
				// 14. Align all flex items along the cross-axis.
				child.offsetCross = alignFlexItemsAlongCrossAxis(child, freeSpace, maxBaseline, constants)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// alignFlexItemsAlongCrossAxis
// ---------------------------------------------------------------------------

func alignFlexItemsAlongCrossAxis(
	child *flexItem,
	freeSpace float32,
	maxBaseline float32,
	constants *algoConstants,
) float32 {
	crossAxisShouldReverse := constants.isColumn && constants.layoutDirection.IsRTL()

	switch child.alignSelf {
	case AlignItemsStart:
		if crossAxisShouldReverse {
			return freeSpace
		}
		return 0.0
	case AlignItemsFlexStart:
		if constants.isWrapReverse != crossAxisShouldReverse {
			return freeSpace
		}
		return 0.0
	case AlignItemsEnd:
		if crossAxisShouldReverse {
			return 0.0
		}
		return freeSpace
	case AlignItemsFlexEnd:
		if constants.isWrapReverse != crossAxisShouldReverse {
			return 0.0
		}
		return freeSpace
	case AlignItemsCenter:
		return freeSpace / 2.0
	case AlignItemsBaseline:
		if constants.isRow {
			return maxBaseline - child.baseline
		}
		baselineColumnShouldReverse := crossAxisShouldReverse && !constants.isWrap
		if constants.isWrapReverse != baselineColumnShouldReverse {
			return freeSpace
		}
		return 0.0
	case AlignItemsStretch:
		if constants.isWrapReverse != crossAxisShouldReverse {
			return freeSpace
		}
		return 0.0
	default:
		return 0.0
	}
}

// ---------------------------------------------------------------------------
// determineContainerCrossSize
// ---------------------------------------------------------------------------

func determineContainerCrossSize(
	flexLines []*flexLine,
	nodeSize Size[Option[float32]],
	constants *algoConstants,
) float32 {
	totalCrossAxisGap := sumAxisGaps(sizeCross(constants.gap, constants.dir), len(flexLines))
	totalLineCrossSize := float32(0.0)
	for _, line := range flexLines {
		totalLineCrossSize += line.crossSize
	}

	paddingBorderSum := rectCrossAxisSum(constants.contentBoxInset, constants.dir)
	crossScrollbarGutter := pointCross(constants.scrollbarGutter, constants.dir)
	minCrossSize := sizeOptionCross(constants.minSize, constants.dir)
	maxCrossSize := sizeOptionCross(constants.maxSize, constants.dir)

	outerContainerSize := sizeOptionCross(nodeSize, constants.dir).
		UnwrapOr(totalLineCrossSize + totalCrossAxisGap + paddingBorderSum)
	outerContainerSize = maybeClampF32(outerContainerSize, minCrossSize, maxCrossSize)
	if outerContainerSize < paddingBorderSum-crossScrollbarGutter {
		outerContainerSize = paddingBorderSum - crossScrollbarGutter
	}
	innerContainerSize := f32Max(outerContainerSize-paddingBorderSum, 0.0)

	sizeSetCross(&constants.containerSize, constants.dir, outerContainerSize)
	sizeSetCross(&constants.innerContainerSize, constants.dir, innerContainerSize)

	return totalLineCrossSize
}

// ---------------------------------------------------------------------------
// alignFlexLinesPerAlignContent
// ---------------------------------------------------------------------------

func alignFlexLinesPerAlignContent(flexLines []*flexLine, constants *algoConstants, totalCrossSize float32) {
	numLines := len(flexLines)
	gap := sizeCross(constants.gap, constants.dir)
	totalCrossAxisGap := sumAxisGaps(gap, numLines)
	freeSpace := sizeCross(constants.innerContainerSize, constants.dir) - totalCrossSize - totalCrossAxisGap
	isSafe := false

	alignContentMode := applyAlignmentFallback(freeSpace, numLines, constants.alignContent, isSafe)

	if constants.isWrapReverse {
		for revI := numLines - 1; revI >= 0; revI-- {
			enumI := numLines - 1 - revI
			line := flexLines[revI]
			line.offsetCross = computeAlignmentOffset(freeSpace, numLines, gap, alignContentMode, constants.isWrapReverse, enumI == 0)
		}
	} else {
		for i, line := range flexLines {
			line.offsetCross = computeAlignmentOffset(freeSpace, numLines, gap, alignContentMode, constants.isWrapReverse, i == 0)
		}
	}
}

// ---------------------------------------------------------------------------
// calculateFlexItem
// ---------------------------------------------------------------------------

func calculateFlexItem(
	tree layoutFlexboxContainer,
	item *flexItem,
	totalOffsetMain *float32,
	totalOffsetCross float32,
	lineOffsetCross float32,
	totalContentSize *Size[float32],
	containerSize Size[float32],
	nodeInnerSize Size[Option[float32]],
	direction FlexDirection,
	layoutDirection Direction,
) {
	layoutOutput := performChildLayout(
		tree, item.node,
		SizeMapTo(item.targetSize, func(s float32) Option[float32] { return Some(s) }),
		nodeInnerSize,
		SizeMapTo(containerSize, func(s float32) AvailableSpace { return AvailableSpaceFromDefinite(s) }),
		SizingModeContentSize,
		LineFalseBool,
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

	var location Point[float32]
	if direction.IsRow() {
		location = Point[float32]{X: offsetMain, Y: offsetCross}
	} else {
		location = Point[float32]{X: offsetCross, Y: offsetMain}
	}
	scrollbarSize := Size[float32]{
		Width: func() float32 {
			if item.overflow.Y == OverflowScroll {
				return item.scrollbarWidth
			}
			return 0.0
		}(),
		Height: func() float32 {
			if item.overflow.X == OverflowScroll {
				return item.scrollbarWidth
			}
			return 0.0
		}(),
	}

	tree.setUnroundedLayout(item.node, &Layout{
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
	var contributionLocation Point[float32]
	if layoutDirection.IsRTL() {
		contributionLocation = Point[float32]{X: containerSize.Width - (location.X + size.Width), Y: location.Y}
	} else {
		contributionLocation = location
	}
	*totalContentSize = SizeF32Max(*totalContentSize, computeContentSizeContribution(
		contributionLocation, size, contentSize, item.overflow,
	))
}

// ---------------------------------------------------------------------------
// calculateLayoutLine
// ---------------------------------------------------------------------------

func calculateLayoutLine(
	tree layoutFlexboxContainer,
	line *flexLine,
	totalOffsetCross *float32,
	contentSize *Size[float32],
	containerSize Size[float32],
	nodeInnerSize Size[Option[float32]],
	paddingBorder Rect[float32],
	direction FlexDirection,
	layoutDirection Direction,
) {
	var totalOffsetMain float32
	if layoutDirection.IsRTL() && direction.IsRow() {
		totalOffsetMain = containerSize.Width - rectMainEnd(paddingBorder, direction)
	} else {
		totalOffsetMain = rectMainStart(paddingBorder, direction)
	}
	lineOffsetCross := line.offsetCross

	isRTLColumn := layoutDirection.IsRTL() && direction.IsColumn()
	if isRTLColumn {
		*totalOffsetCross -= lineOffsetCross + line.crossSize
	}

	if direction.IsReverse() {
		for revI := len(line.items) - 1; revI >= 0; revI-- {
			calculateFlexItem(
				tree, line.items[revI],
				&totalOffsetMain, *totalOffsetCross, lineOffsetCross,
				contentSize, containerSize, nodeInnerSize, direction, layoutDirection,
			)
		}
	} else {
		for _, item := range line.items {
			calculateFlexItem(
				tree, item,
				&totalOffsetMain, *totalOffsetCross, lineOffsetCross,
				contentSize, containerSize, nodeInnerSize, direction, layoutDirection,
			)
		}
	}

	if !isRTLColumn {
		*totalOffsetCross += lineOffsetCross + line.crossSize
	}
}

// ---------------------------------------------------------------------------
// finalLayoutPass
// ---------------------------------------------------------------------------

func finalLayoutPass(
	tree layoutFlexboxContainer,
	flexLines []*flexLine,
	constants *algoConstants,
) Size[float32] {
	var totalOffsetCross float32
	if constants.isColumn && constants.layoutDirection.IsRTL() {
		totalOffsetCross = constants.containerSize.Width - rectCrossEnd(constants.contentBoxInset, constants.dir)
	} else {
		totalOffsetCross = rectCrossStart(constants.contentBoxInset, constants.dir)
	}

	contentSize := SizeZeroF32

	if constants.isWrapReverse {
		for revI := len(flexLines) - 1; revI >= 0; revI-- {
			calculateLayoutLine(
				tree, flexLines[revI],
				&totalOffsetCross, &contentSize,
				constants.containerSize, constants.nodeInnerSize, constants.contentBoxInset,
				constants.dir, constants.layoutDirection,
			)
		}
	} else {
		for _, line := range flexLines {
			calculateLayoutLine(
				tree, line,
				&totalOffsetCross, &contentSize,
				constants.containerSize, constants.nodeInnerSize, constants.contentBoxInset,
				constants.dir, constants.layoutDirection,
			)
		}
	}

	if constants.layoutDirection.IsRTL() {
		contentSize.Width += constants.contentBoxInset.Left - constants.border.Left - constants.scrollbarGutter.X
	} else {
		contentSize.Width += constants.contentBoxInset.Right - constants.border.Right - constants.scrollbarGutter.X
	}
	contentSize.Height += constants.contentBoxInset.Bottom - constants.border.Bottom - constants.scrollbarGutter.Y

	return contentSize
}

// ---------------------------------------------------------------------------
// performAbsoluteLayoutOnAbsoluteChildren
// ---------------------------------------------------------------------------

func performAbsoluteLayoutOnAbsoluteChildren(
	tree layoutFlexboxContainer,
	node NodeId,
	constants *algoConstants,
) Size[float32] {
	containerWidth := constants.containerSize.Width
	containerHeight := constants.containerSize.Height
	insetRelativeSize := SizeSubF32(
		SizeSubF32(constants.containerSize, RectSumAxes(constants.border)),
		PointToSize(constants.scrollbarGutter),
	)

	contentSize := SizeZeroF32
	calc := tree.resolveCalcValue

	for order := 0; order < tree.childCount(node); order++ {
		child := tree.getChildId(node, order)
		childStyle := tree.getFlexboxChildStyle(child)

		if childStyle.boxGenerationMode() == BoxGenerationModeNone || childStyle.position() != PositionAbsolute {
			continue
		}

		overflow := childStyle.overflow()
		scrollbarWidth := childStyle.scrollbarWidth()
		aspectRatio := childStyle.aspectRatio()
		alignSelf := childStyle.alignSelf().UnwrapOr(constants.alignItems)
		margin := resolveRectLPAToOption(childStyle.margin(), insetRelativeSize.Width, calc)
		padding := resolveRectLPWidth(childStyle.padding(), Some(insetRelativeSize.Width), calc)
		border := resolveRectLPWidth(childStyle.border(), Some(insetRelativeSize.Width), calc)
		paddingBorderSum := SizeAddF32(RectSumAxes(padding), RectSumAxes(border))
		boxSizingAdj := SizeZeroF32
		if childStyle.boxSizing() == BoxSizingContentBox {
			boxSizingAdj = paddingBorderSum
		}

		// Resolve inset.
		insetS := childStyle.inset()
		left := MaybeResolveLPAF32(insetS.Left, insetRelativeSize.Width, calc)
		right := MaybeResolveLPAF32(insetS.Right, insetRelativeSize.Width, calc)
		top := MaybeResolveLPAF32(insetS.Top, insetRelativeSize.Height, calc)
		bottom := MaybeResolveLPAF32(insetS.Bottom, insetRelativeSize.Height, calc)

		// Compute known dimensions from style.
		insetRelSizeOpt := Size[Option[float32]]{Width: Some(insetRelativeSize.Width), Height: Some(insetRelativeSize.Height)}
		styleSize := SizeMaybeAdd(
			SizeOptionMaybeApplyAspectRatio(
				MaybeResolveSizeDimOption(childStyle.size(), insetRelSizeOpt, calc),
				aspectRatio,
			),
			Size[Option[float32]]{Width: Some(boxSizingAdj.Width), Height: Some(boxSizingAdj.Height)},
		)
		minSize := SizeMaybeMax(
			SizeOptionOr(
				SizeMaybeAdd(
					SizeOptionMaybeApplyAspectRatio(
						MaybeResolveSizeDimOption(childStyle.minSize(), insetRelSizeOpt, calc),
						aspectRatio,
					),
					Size[Option[float32]]{Width: Some(boxSizingAdj.Width), Height: Some(boxSizingAdj.Height)},
				),
				Size[Option[float32]]{Width: Some(paddingBorderSum.Width), Height: Some(paddingBorderSum.Height)},
			),
			Size[Option[float32]]{Width: Some(paddingBorderSum.Width), Height: Some(paddingBorderSum.Height)},
		)
		maxSize := SizeMaybeAdd(
			SizeOptionMaybeApplyAspectRatio(
				MaybeResolveSizeDimOption(childStyle.maxSize(), insetRelSizeOpt, calc),
				aspectRatio,
			),
			Size[Option[float32]]{Width: Some(boxSizingAdj.Width), Height: Some(boxSizingAdj.Height)},
		)
		knownDimensions := SizeMaybeClamp(styleSize, minSize, maxSize)

		// Fill in width from left/right.
		if !knownDimensions.Width.IsSome && left.IsSome && right.IsSome {
			newWidthRaw := insetRelativeSize.Width
			newWidthRaw = maybeSubF32(newWidthRaw, margin.Left)
			newWidthRaw = maybeSubF32(newWidthRaw, margin.Right)
			newWidthRaw -= left.Value
			newWidthRaw -= right.Value
			knownDimensions.Width = Some(f32Max(newWidthRaw, 0.0))
			knownDimensions = SizeMaybeClamp(SizeOptionMaybeApplyAspectRatio(knownDimensions, aspectRatio), minSize, maxSize)
		}

		// Fill in height from top/bottom.
		if !knownDimensions.Height.IsSome && top.IsSome && bottom.IsSome {
			newHeightRaw := insetRelativeSize.Height
			newHeightRaw = maybeSubF32(newHeightRaw, margin.Top)
			newHeightRaw = maybeSubF32(newHeightRaw, margin.Bottom)
			newHeightRaw -= top.Value
			newHeightRaw -= bottom.Value
			knownDimensions.Height = Some(f32Max(newHeightRaw, 0.0))
			knownDimensions = SizeMaybeClamp(SizeOptionMaybeApplyAspectRatio(knownDimensions, aspectRatio), minSize, maxSize)
		}

		measuredSize := measureChildSizeBoth(
			tree, child,
			knownDimensions,
			constants.nodeInnerSize,
			Size[AvailableSpace]{
				Width:  AvailableSpaceFromDefinite(maybeClampF32(containerWidth, minSize.Width, maxSize.Width)),
				Height: AvailableSpaceFromDefinite(maybeClampF32(containerHeight, minSize.Height, maxSize.Height)),
			},
			SizingModeInherentSize,
			LineFalseBool,
		)
		finalSize := SizeMaybeClamp(
			SizeOptionOr(knownDimensions, Size[Option[float32]]{Width: Some(measuredSize.Width), Height: Some(measuredSize.Height)}),
			minSize, maxSize,
		)
		finalSizeF32 := Size[float32]{Width: finalSize.Width.UnwrapOr(0), Height: finalSize.Height.UnwrapOr(0)}

		layoutOutput := performChildLayout(
			tree, child,
			finalSize,
			constants.nodeInnerSize,
			Size[AvailableSpace]{
				Width:  AvailableSpaceFromDefinite(maybeClampF32(containerWidth, minSize.Width, maxSize.Width)),
				Height: AvailableSpaceFromDefinite(maybeClampF32(containerHeight, minSize.Height, maxSize.Height)),
			},
			SizingModeInherentSize,
			LineFalseBool,
		)

		nonAutoMargin := Rect[float32]{
			Left:   margin.Left.UnwrapOr(0),
			Right:  margin.Right.UnwrapOr(0),
			Top:    margin.Top.UnwrapOr(0),
			Bottom: margin.Bottom.UnwrapOr(0),
		}

		freeSpaceF := Size[float32]{
			Width:  f32Max(constants.containerSize.Width-finalSizeF32.Width-RectHorizontalAxisSum(nonAutoMargin), 0),
			Height: f32Max(constants.containerSize.Height-finalSizeF32.Height-RectVerticalAxisSum(nonAutoMargin), 0),
		}

		// Expand auto margins.
		resolvedMargin := func() Rect[float32] {
			autoMarginW := func() float32 {
				count := uint8(0)
				if !margin.Left.IsSome {
					count++
				}
				if !margin.Right.IsSome {
					count++
				}
				if count > 0 {
					return freeSpaceF.Width / float32(count)
				}
				return 0.0
			}()
			autoMarginH := func() float32 {
				count := uint8(0)
				if !margin.Top.IsSome {
					count++
				}
				if !margin.Bottom.IsSome {
					count++
				}
				if count > 0 {
					return freeSpaceF.Height / float32(count)
				}
				return 0.0
			}()
			return Rect[float32]{
				Left:   margin.Left.UnwrapOr(autoMarginW),
				Right:  margin.Right.UnwrapOr(autoMarginW),
				Top:    margin.Top.UnwrapOr(autoMarginH),
				Bottom: margin.Bottom.UnwrapOr(autoMarginH),
			}
		}()

		// Determine flex-relative insets.
		var startMain, endMain, startCross, endCross Option[float32]
		if constants.isRow {
			startMain, endMain = left, right
			startCross, endCross = top, bottom
		} else {
			startMain, endMain = top, bottom
			startCross, endCross = left, right
		}

		mainAxisIsHorizontal := constants.isRow
		crossAxisIsHorizontal := !constants.isRow
		mainIsRTL := mainAxisIsHorizontal && constants.layoutDirection.IsRTL()
		crossIsRTL := crossAxisIsHorizontal && constants.layoutDirection.IsRTL()
		mainAxisFlexStartReversed := constants.dir.IsReverse() != mainIsRTL
		crossAxisFlexStartReversed := constants.isWrapReverse != crossIsRTL

		mainStartScrollbarOffset := float32(0.0)
		if mainIsRTL {
			mainStartScrollbarOffset = pointMain(constants.scrollbarGutter, constants.dir)
		}
		crossStartScrollbarOffset := float32(0.0)
		if crossIsRTL {
			crossStartScrollbarOffset = pointCross(constants.scrollbarGutter, constants.dir)
		}
		mainEndScrollbarOffset := float32(0.0)
		if !mainIsRTL {
			mainEndScrollbarOffset = pointMain(constants.scrollbarGutter, constants.dir)
		}
		crossEndScrollbarOffset := float32(0.0)
		if !crossIsRTL {
			crossEndScrollbarOffset = pointCross(constants.scrollbarGutter, constants.dir)
		}

		// Apply main-axis alignment.
		var offsetMain float32
		if startMain.IsSome || endMain.IsSome {
			if mainIsRTL && endMain.IsSome {
				offsetMain = sizeMain(constants.containerSize, constants.dir) -
					rectMainEnd(constants.border, constants.dir) -
					mainEndScrollbarOffset -
					sizeMain(finalSizeF32, constants.dir) -
					endMain.UnwrapOr(0) -
					rectMainEnd(resolvedMargin, constants.dir)
			} else if startMain.IsSome {
				offsetMain = startMain.Value +
					rectMainStart(constants.border, constants.dir) +
					mainStartScrollbarOffset +
					rectMainStart(resolvedMargin, constants.dir)
			} else {
				offsetMain = sizeMain(constants.containerSize, constants.dir) -
					rectMainEnd(constants.border, constants.dir) -
					mainEndScrollbarOffset -
					sizeMain(finalSizeF32, constants.dir) -
					endMain.UnwrapOr(0) -
					rectMainEnd(resolvedMargin, constants.dir)
			}
		} else {
			jc := AlignContentStart
			if constants.justifyContent.IsSome {
				jc = constants.justifyContent.Value
			}
			switch {
			case jc == AlignContentSpaceBetween ||
				(!mainAxisFlexStartReversed && (jc == AlignContentStretch || jc == AlignContentFlexStart)) ||
				(mainAxisFlexStartReversed && jc == AlignContentFlexEnd):
				offsetMain = rectMainStart(constants.contentBoxInset, constants.dir) + rectMainStart(resolvedMargin, constants.dir)
			case !mainAxisFlexStartReversed && jc == AlignContentStart:
				offsetMain = rectMainStart(constants.contentBoxInset, constants.dir) + rectMainStart(resolvedMargin, constants.dir)
			case mainAxisFlexStartReversed && jc == AlignContentStart:
				offsetMain = sizeMain(constants.containerSize, constants.dir) -
					rectMainEnd(constants.contentBoxInset, constants.dir) -
					sizeMain(finalSizeF32, constants.dir) -
					rectMainEnd(resolvedMargin, constants.dir)
			case !mainAxisFlexStartReversed && jc == AlignContentEnd:
				offsetMain = sizeMain(constants.containerSize, constants.dir) -
					rectMainEnd(constants.contentBoxInset, constants.dir) -
					sizeMain(finalSizeF32, constants.dir) -
					rectMainEnd(resolvedMargin, constants.dir)
			case mainAxisFlexStartReversed && jc == AlignContentEnd:
				offsetMain = rectMainStart(constants.contentBoxInset, constants.dir) + rectMainStart(resolvedMargin, constants.dir)
			case (!mainAxisFlexStartReversed && jc == AlignContentFlexEnd) ||
				(mainAxisFlexStartReversed && (jc == AlignContentFlexStart || jc == AlignContentStretch)):
				offsetMain = sizeMain(constants.containerSize, constants.dir) -
					rectMainEnd(constants.contentBoxInset, constants.dir) -
					sizeMain(finalSizeF32, constants.dir) -
					rectMainEnd(resolvedMargin, constants.dir)
			default: // SpaceEvenly, SpaceAround, Center
				offsetMain = (sizeMain(constants.containerSize, constants.dir) +
					rectMainStart(constants.contentBoxInset, constants.dir) -
					rectMainEnd(constants.contentBoxInset, constants.dir) -
					sizeMain(finalSizeF32, constants.dir) +
					rectMainStart(resolvedMargin, constants.dir) -
					rectMainEnd(resolvedMargin, constants.dir)) / 2.0
			}
		}

		// Apply cross-axis alignment.
		var offsetCrossVal float32
		if startCross.IsSome || endCross.IsSome {
			if crossIsRTL && endCross.IsSome {
				offsetCrossVal = sizeCross(constants.containerSize, constants.dir) -
					rectCrossEnd(constants.border, constants.dir) -
					crossEndScrollbarOffset -
					sizeCross(finalSizeF32, constants.dir) -
					endCross.UnwrapOr(0) -
					rectCrossEnd(resolvedMargin, constants.dir)
			} else if startCross.IsSome {
				offsetCrossVal = startCross.Value +
					rectCrossStart(constants.border, constants.dir) +
					crossStartScrollbarOffset +
					rectCrossStart(resolvedMargin, constants.dir)
			} else {
				offsetCrossVal = sizeCross(constants.containerSize, constants.dir) -
					rectCrossEnd(constants.border, constants.dir) -
					crossEndScrollbarOffset -
					sizeCross(finalSizeF32, constants.dir) -
					endCross.UnwrapOr(0) -
					rectCrossEnd(resolvedMargin, constants.dir)
			}
		} else {
			switch {
			case (!crossAxisFlexStartReversed && alignSelf == AlignItemsStart) ||
				(crossAxisFlexStartReversed && alignSelf == AlignItemsEnd):
				offsetCrossVal = rectCrossStart(constants.contentBoxInset, constants.dir) + rectCrossStart(resolvedMargin, constants.dir)
			case (crossAxisFlexStartReversed && alignSelf == AlignItemsStart) ||
				(!crossAxisFlexStartReversed && alignSelf == AlignItemsEnd):
				offsetCrossVal = sizeCross(constants.containerSize, constants.dir) -
					rectCrossEnd(constants.contentBoxInset, constants.dir) -
					sizeCross(finalSizeF32, constants.dir) -
					rectCrossEnd(resolvedMargin, constants.dir)
			case (!crossAxisFlexStartReversed && (alignSelf == AlignItemsBaseline || alignSelf == AlignItemsStretch || alignSelf == AlignItemsFlexStart)) ||
				(crossAxisFlexStartReversed && alignSelf == AlignItemsFlexEnd):
				offsetCrossVal = rectCrossStart(constants.contentBoxInset, constants.dir) + rectCrossStart(resolvedMargin, constants.dir)
			case (crossAxisFlexStartReversed && (alignSelf == AlignItemsBaseline || alignSelf == AlignItemsStretch || alignSelf == AlignItemsFlexStart)) ||
				(!crossAxisFlexStartReversed && alignSelf == AlignItemsFlexEnd):
				offsetCrossVal = sizeCross(constants.containerSize, constants.dir) -
					rectCrossEnd(constants.contentBoxInset, constants.dir) -
					sizeCross(finalSizeF32, constants.dir) -
					rectCrossEnd(resolvedMargin, constants.dir)
			case alignSelf == AlignItemsCenter:
				offsetCrossVal = (sizeCross(constants.containerSize, constants.dir) +
					rectCrossStart(constants.contentBoxInset, constants.dir) -
					rectCrossEnd(constants.contentBoxInset, constants.dir) -
					sizeCross(finalSizeF32, constants.dir) +
					rectCrossStart(resolvedMargin, constants.dir) -
					rectCrossEnd(resolvedMargin, constants.dir)) / 2.0
			default:
				offsetCrossVal = rectCrossStart(constants.contentBoxInset, constants.dir) + rectCrossStart(resolvedMargin, constants.dir)
			}
		}

		var location Point[float32]
		if constants.isRow {
			location = Point[float32]{X: offsetMain, Y: offsetCrossVal}
		} else {
			location = Point[float32]{X: offsetCrossVal, Y: offsetMain}
		}

		scrollbarSize := Size[float32]{
			Width: func() float32 {
				if overflow.Y == OverflowScroll {
					return scrollbarWidth
				}
				return 0.0
			}(),
			Height: func() float32 {
				if overflow.X == OverflowScroll {
					return scrollbarWidth
				}
				return 0.0
			}(),
		}

		tree.setUnroundedLayout(child, &Layout{
			Order:         uint32(order),
			Size:          finalSizeF32,
			ContentSize:   layoutOutput.ContentSize,
			ScrollbarSize: scrollbarSize,
			Location:      location,
			Padding:       padding,
			Border:        border,
			Margin:        resolvedMargin,
		})

		// Content size contribution for absolute children.
		sizeContentSizeContrib := Size[float32]{
			Width: func() float32 {
				if overflow.X == OverflowVisible {
					return f32Max(finalSizeF32.Width, layoutOutput.ContentSize.Width)
				}
				return finalSizeF32.Width
			}(),
			Height: func() float32 {
				if overflow.Y == OverflowVisible {
					return f32Max(finalSizeF32.Height, layoutOutput.ContentSize.Height)
				}
				return finalSizeF32.Height
			}(),
		}
		if SizeHasNonZeroArea(sizeContentSizeContrib) {
			absoluteAreaOffset := Point[float32]{
				X: constants.border.Left + func() float32 {
					if constants.layoutDirection.IsRTL() {
						return constants.scrollbarGutter.X
					}
					return 0
				}(),
				Y: constants.border.Top,
			}
			relativeLocation := Point[float32]{
				X: location.X - absoluteAreaOffset.X,
				Y: location.Y - absoluteAreaOffset.Y,
			}
			var contentSizeContrib Size[float32]
			if constants.layoutDirection.IsRTL() {
				overflowExtraWidth := f32Max(sizeContentSizeContrib.Width-finalSizeF32.Width, 0)
				contentSizeContrib = Size[float32]{
					Width:  f32Max(insetRelativeSize.Width-relativeLocation.X, 0) + overflowExtraWidth,
					Height: relativeLocation.Y + sizeContentSizeContrib.Height,
				}
			} else {
				contentSizeContrib = Size[float32]{
					Width:  relativeLocation.X + sizeContentSizeContrib.Width,
					Height: relativeLocation.Y + sizeContentSizeContrib.Height,
				}
			}
			contentSize = SizeF32Max(contentSize, contentSizeContrib)
		}
	}

	return contentSize
}

// ---------------------------------------------------------------------------
// sumAxisGaps
// ---------------------------------------------------------------------------

func sumAxisGaps(gap float32, numItems int) float32 {
	if numItems <= 1 {
		return 0.0
	}
	return gap * float32(numItems-1)
}

// pointMain returns the main-axis component of a Point[float32] based on direction.
// (Already defined above for float32, but we also need it for the scrollbar gutter.)

// maybeSubOption subtracts b from a where both are Option[float32].
// Already defined in util_math.go.
