// Package flexbox implements the CSS Flexbox layout algorithm.
// This file is a port of compute/flexbox.rs from the Taffy Rust layout engine.
package flexbox

import (
	"taffy-go/lib"
	"math"

	"taffy-go/compute"
	"taffy-go/geometry"
	"taffy-go/style"
	"taffy-go/tree"
)

// ---------------------------------------------------------------------------
// Flex-direction-aware axis helpers for Size, Rect, style.AvailableSpace, and Point.
// These map main/cross axis concepts to width/height based on style.FlexDirection.
// ---------------------------------------------------------------------------

// sizeMain returns the main-axis component of a geometry.Size[T] for numeric types.
func sizeMain[T interface{ float32 | ~int }](s geometry.Size[T], dir style.FlexDirection) T {
	if dir.IsRow() {
		return s.Width
	}
	return s.Height
}

// sizeCross returns the cross-axis component of a geometry.Size[T] for numeric types.
func sizeCross[T interface{ float32 | ~int }](s geometry.Size[T], dir style.FlexDirection) T {
	if dir.IsRow() {
		return s.Height
	}
	return s.Width
}

// sizeSetMain sets the main-axis component of a Size.
func sizeSetMain[T any](s *geometry.Size[T], dir style.FlexDirection, val T) {
	if dir.IsRow() {
		s.Width = val
	} else {
		s.Height = val
	}
}

// sizeSetCross sets the cross-axis component of a Size.
func sizeSetCross[T any](s *geometry.Size[T], dir style.FlexDirection, val T) {
	if dir.IsRow() {
		s.Height = val
	} else {
		s.Width = val
	}
}

// sizeWithMain returns a copy with the main axis value replaced.
func sizeWithMain[T any](s geometry.Size[T], dir style.FlexDirection, val T) geometry.Size[T] {
	r := s
	sizeSetMain(&r, dir, val)
	return r
}

// sizeWithCross returns a copy with the cross axis value replaced.
func sizeWithCross[T any](s geometry.Size[T], dir style.FlexDirection, val T) geometry.Size[T] {
	r := s
	sizeSetCross(&r, dir, val)
	return r
}

// sizeFromCross creates a geometry.Size[lib.Option[float32]] with only the cross axis set.
func sizeFromCross(dir style.FlexDirection, val lib.Option[float32]) geometry.Size[lib.Option[float32]] {
	s := geometry.SizeOptionNone
	sizeSetCross(&s, dir, val)
	return s
}

// sizeOptionMain returns the main-axis option component.
func sizeOptionMain(s geometry.Size[lib.Option[float32]], dir style.FlexDirection) lib.Option[float32] {
	if dir.IsRow() {
		return s.Width
	}
	return s.Height
}

// sizeOptionCross returns the cross-axis option component.
func sizeOptionCross(s geometry.Size[lib.Option[float32]], dir style.FlexDirection) lib.Option[float32] {
	if dir.IsRow() {
		return s.Height
	}
	return s.Width
}

// sizeOptionSetMain sets the main-axis option.
func sizeOptionSetMain(s *geometry.Size[lib.Option[float32]], dir style.FlexDirection, val lib.Option[float32]) {
	sizeSetMain(s, dir, val)
}

// sizeOptionSetCross sets the cross-axis option.
func sizeOptionSetCross(s *geometry.Size[lib.Option[float32]], dir style.FlexDirection, val lib.Option[float32]) {
	sizeSetCross(s, dir, val)
}

// sizeOptionWithMain returns a copy with main axis replaced.
func sizeOptionWithMain(s geometry.Size[lib.Option[float32]], dir style.FlexDirection, val lib.Option[float32]) geometry.Size[lib.Option[float32]] {
	return sizeWithMain(s, dir, val)
}

// avMain returns the main-axis style.AvailableSpace.
func avMain(s geometry.Size[style.AvailableSpace], dir style.FlexDirection) style.AvailableSpace {
	if dir.IsRow() {
		return s.Width
	}
	return s.Height
}

// avCross returns the cross-axis style.AvailableSpace.
func avCross(s geometry.Size[style.AvailableSpace], dir style.FlexDirection) style.AvailableSpace {
	if dir.IsRow() {
		return s.Height
	}
	return s.Width
}

// avWithMain returns a copy with the main axis replaced.
func avWithMain(s geometry.Size[style.AvailableSpace], dir style.FlexDirection, val style.AvailableSpace) geometry.Size[style.AvailableSpace] {
	r := s
	if dir.IsRow() {
		r.Width = val
	} else {
		r.Height = val
	}
	return r
}

// avWithCross returns a copy with the cross axis replaced.
func avWithCross(s geometry.Size[style.AvailableSpace], dir style.FlexDirection, val style.AvailableSpace) geometry.Size[style.AvailableSpace] {
	r := s
	if dir.IsRow() {
		r.Height = val
	} else {
		r.Width = val
	}
	return r
}

// rectMainAxisSum returns the sum of the main-axis sides of a geometry.Rect[float32].
func rectMainAxisSum(r geometry.Rect[float32], dir style.FlexDirection) float32 {
	if dir.IsRow() {
		return r.Left + r.Right
	}
	return r.Top + r.Bottom
}

// rectCrossAxisSum returns the sum of the cross-axis sides of a geometry.Rect[float32].
func rectCrossAxisSum(r geometry.Rect[float32], dir style.FlexDirection) float32 {
	if dir.IsRow() {
		return r.Top + r.Bottom
	}
	return r.Left + r.Right
}

// rectMainStart returns the start (flex-direction-aware) value on the main axis.
func rectMainStart(r geometry.Rect[float32], dir style.FlexDirection) float32 {
	switch dir {
	case style.FlexDirectionRow:
		return r.Left
	case style.FlexDirectionRowReverse:
		return r.Right
	case style.FlexDirectionColumn:
		return r.Top
	default: // ColumnReverse
		return r.Bottom
	}
}

// rectMainEnd returns the end (flex-direction-aware) value on the main axis.
func rectMainEnd(r geometry.Rect[float32], dir style.FlexDirection) float32 {
	switch dir {
	case style.FlexDirectionRow:
		return r.Right
	case style.FlexDirectionRowReverse:
		return r.Left
	case style.FlexDirectionColumn:
		return r.Bottom
	default: // ColumnReverse
		return r.Top
	}
}

// rectCrossStart returns the start value on the cross axis.
func rectCrossStart(r geometry.Rect[float32], dir style.FlexDirection) float32 {
	switch dir {
	case style.FlexDirectionRow:
		return r.Top
	case style.FlexDirectionRowReverse:
		return r.Top
	case style.FlexDirectionColumn:
		return r.Left
	default: // ColumnReverse
		return r.Left
	}
}

// rectCrossEnd returns the end value on the cross axis.
func rectCrossEnd(r geometry.Rect[float32], dir style.FlexDirection) float32 {
	switch dir {
	case style.FlexDirectionRow:
		return r.Bottom
	case style.FlexDirectionRowReverse:
		return r.Bottom
	case style.FlexDirectionColumn:
		return r.Right
	default: // ColumnReverse
		return r.Right
	}
}

// rectOptionMainStart returns the main-start Option value.
func rectOptionMainStart(r geometry.Rect[lib.Option[float32]], dir style.FlexDirection) lib.Option[float32] {
	switch dir {
	case style.FlexDirectionRow:
		return r.Left
	case style.FlexDirectionRowReverse:
		return r.Right
	case style.FlexDirectionColumn:
		return r.Top
	default:
		return r.Bottom
	}
}

// rectOptionMainEnd returns the main-end Option value.
func rectOptionMainEnd(r geometry.Rect[lib.Option[float32]], dir style.FlexDirection) lib.Option[float32] {
	switch dir {
	case style.FlexDirectionRow:
		return r.Right
	case style.FlexDirectionRowReverse:
		return r.Left
	case style.FlexDirectionColumn:
		return r.Bottom
	default:
		return r.Top
	}
}

// rectOptionCrossStart returns the cross-start Option value.
func rectOptionCrossStart(r geometry.Rect[lib.Option[float32]], dir style.FlexDirection) lib.Option[float32] {
	switch dir {
	case style.FlexDirectionRow, style.FlexDirectionRowReverse:
		return r.Top
	default:
		return r.Left
	}
}

// rectOptionCrossEnd returns the cross-end Option value.
func rectOptionCrossEnd(r geometry.Rect[lib.Option[float32]], dir style.FlexDirection) lib.Option[float32] {
	switch dir {
	case style.FlexDirectionRow, style.FlexDirectionRowReverse:
		return r.Bottom
	default:
		return r.Right
	}
}

// rectBoolMainStart returns the main-start bool.
func rectBoolMainStart(r geometry.Rect[bool], dir style.FlexDirection) bool {
	switch dir {
	case style.FlexDirectionRow:
		return r.Left
	case style.FlexDirectionRowReverse:
		return r.Right
	case style.FlexDirectionColumn:
		return r.Top
	default:
		return r.Bottom
	}
}

// rectBoolMainEnd returns the main-end bool.
func rectBoolMainEnd(r geometry.Rect[bool], dir style.FlexDirection) bool {
	switch dir {
	case style.FlexDirectionRow:
		return r.Right
	case style.FlexDirectionRowReverse:
		return r.Left
	case style.FlexDirectionColumn:
		return r.Bottom
	default:
		return r.Top
	}
}

// rectBoolCrossStart returns the cross-start bool.
func rectBoolCrossStart(r geometry.Rect[bool], dir style.FlexDirection) bool {
	switch dir {
	case style.FlexDirectionRow, style.FlexDirectionRowReverse:
		return r.Top
	default:
		return r.Left
	}
}

// rectBoolCrossEnd returns the cross-end bool.
func rectBoolCrossEnd(r geometry.Rect[bool], dir style.FlexDirection) bool {
	switch dir {
	case style.FlexDirectionRow, style.FlexDirectionRowReverse:
		return r.Bottom
	default:
		return r.Right
	}
}

// pointMain returns the main-axis component of a Point.
func pointMain(p geometry.Point[float32], dir style.FlexDirection) float32 {
	if dir.IsRow() {
		return p.X
	}
	return p.Y
}

// pointCross returns the cross-axis component of a Point.
func pointCross(p geometry.Point[float32], dir style.FlexDirection) float32 {
	if dir.IsRow() {
		return p.Y
	}
	return p.X
}

// dimIsAuto checks if a style.Dimension's cross-axis component is auto.
func dimCrossIsAuto(s geometry.Size[style.Dimension], dir style.FlexDirection) bool {
	if dir.IsRow() {
		return s.Height.IsAuto()
	}
	return s.Width.IsAuto()
}

// asClampF32 clamps an style.AvailableSpace with concrete min/max float32 values.
func asClampF32(a style.AvailableSpace, min, max float32) style.AvailableSpace {
	if a.Tag == style.AvailableSpaceDefinite {
		return style.AvailableSpaceFromDefinite(geometry.F32Max(geometry.F32Min(a.Value, max), min))
	}
	return a
}

// asClampOption clamps an style.AvailableSpace with optional min/max.
func asClampOption(a style.AvailableSpace, min_, max_ lib.Option[float32]) style.AvailableSpace {
	if a.Tag == style.AvailableSpaceDefinite {
		v := a.Value
		if min_.IsSome && max_.IsSome {
			return style.AvailableSpaceFromDefinite(geometry.F32Max(geometry.F32Min(v, max_.Value), min_.Value))
		}
		if min_.IsSome {
			return style.AvailableSpaceFromDefinite(geometry.F32Max(v, min_.Value))
		}
		if max_.IsSome {
			return style.AvailableSpaceFromDefinite(geometry.F32Min(v, max_.Value))
		}
		return a
	}
	return a
}

// computeContentSizeContribution computes how much width/height a node contributes to its parent's content size.
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

// resolveRectLPAToOption resolves a geometry.Rect[style.LengthPercentageAuto] to geometry.Rect[lib.Option[float32]]
// where all four sides are resolved against the same context value.
func resolveRectLPAToOption(rect geometry.Rect[style.LengthPercentageAuto], context float32, calc style.CalcResolver) geometry.Rect[lib.Option[float32]] {
	return geometry.Rect[lib.Option[float32]]{
		Left:   style.MaybeResolveLPAF32(rect.Left, context, calc),
		Right:  style.MaybeResolveLPAF32(rect.Right, context, calc),
		Top:    style.MaybeResolveLPAF32(rect.Top, context, calc),
		Bottom: style.MaybeResolveLPAF32(rect.Bottom, context, calc),
	}
}

// measureChildSizeBoth computes both dimensions of a child (RunMode::ComputeSize + tree.RequestedAxis::Both).
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
// Flexbox internal types
// ---------------------------------------------------------------------------

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

// flexLine holds a slice of flex items for one line plus cross-axis sizing info.
type flexLine struct {
	items       []*flexItem
	crossSize   float32
	offsetCross float32
}

// algoConstants holds computed constants for the flexbox algorithm.
type algoConstants struct {
	dir             style.FlexDirection
	layoutDirection style.Direction
	isRow           bool
	isColumn        bool
	isWrap          bool
	isWrapReverse   bool

	minSize geometry.Size[lib.Option[float32]]
	maxSize geometry.Size[lib.Option[float32]]

	margin          geometry.Rect[float32]
	border          geometry.Rect[float32]
	contentBoxInset geometry.Rect[float32]
	scrollbarGutter geometry.Point[float32]
	gap             geometry.Size[float32]

	alignItems     style.AlignItems
	alignContent   style.AlignContent
	justifyContent lib.Option[style.AlignContent]

	nodeOuterSize      geometry.Size[lib.Option[float32]]
	nodeInnerSize      geometry.Size[lib.Option[float32]]
	containerSize      geometry.Size[float32]
	innerContainerSize geometry.Size[float32]
}

// ---------------------------------------------------------------------------
// Main entry point
// ---------------------------------------------------------------------------

// ComputeFlexboxLayout computes the layout of a box according to the flexbox algorithm.
func ComputeFlexboxLayout(
	t tree.LayoutFlexboxContainer,
	node tree.NodeId,
	inputs tree.LayoutInput,
) tree.LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	parentSize := inputs.ParentSize
	runMode := inputs.RunMode

	coreStyle := t.GetCoreContainerStyle(node)
	calc := t.ResolveCalcValue

	aspectRatio := coreStyle.GetAspectRatio()
	padding := style.ResolveRectLPWidth(coreStyle.GetPadding(), parentSize.Width, calc)
	border := style.ResolveRectLPWidth(coreStyle.GetBorder(), parentSize.Width, calc)
	paddingBorderSum := geometry.SizeAddF32(geometry.RectSumAxes(padding), geometry.RectSumAxes(border))
	boxSizingAdjustment := geometry.SizeZeroF32
	if coreStyle.GetBoxSizing() == style.BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSum
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

	// If both min and max in a given axis are set and max <= min then this determines the size.
	minMaxDefiniteSize := geometry.SizeZipMap(minSize, maxSize, func(min, max lib.Option[float32]) lib.Option[float32] {
		if min.IsSome && max.IsSome && max.Value <= min.Value {
			return min
		}
		return lib.None[float32]()
	})

	styledBasedKnownDimensions := geometry.SizeMaybeMax(
		geometry.SizeOptionOr(knownDimensions,
			geometry.SizeOptionOr(minMaxDefiniteSize, clampedStyleSize),
		),
		geometry.Size[lib.Option[float32]]{Width: lib.Some(paddingBorderSum.Width), Height: lib.Some(paddingBorderSum.Height)},
	)

	// Short-circuit if container size is fully known and we only need the size.
	if runMode == tree.RunModeComputeSize {
		if styledBasedKnownDimensions.Width.IsSome && styledBasedKnownDimensions.Height.IsSome {
			return tree.LayoutOutputFromOuterSize(geometry.Size[float32]{
				Width:  styledBasedKnownDimensions.Width.Value,
				Height: styledBasedKnownDimensions.Height.Value,
			})
		}
	}

	return computePreliminary(t, node, tree.LayoutInput{
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

func computePreliminary(t tree.LayoutFlexboxContainer, node tree.NodeId, inputs tree.LayoutInput) tree.LayoutOutput {
	knownDimensions := inputs.KnownDimensions
	availableSpace := inputs.AvailableSpace
	runMode := inputs.RunMode

	// Define constants.
	constants := computeConstants(t, node, knownDimensions, inputs.ParentSize)

	// 9.1. Initial Setup
	// 1. Generate anonymous flex items.
	flexItems := generateAnonymousFlexItems(t, node, &constants)

	// 9.2. Line Length Determination
	// 2. Determine the available main and cross space for the flex items.
	availableSpace = determineAvailableSpace(knownDimensions, availableSpace, &constants)

	// 3. Determine the flex base size and hypothetical main size of each item.
	determineFlexBaseSize(t, &constants, availableSpace, flexItems)

	// 5. Collect flex items into flex lines.
	flexLines := collectFlexLines(&constants, availableSpace, flexItems)

	// Determine container main size and re-resolve gaps.
	if mainInner := sizeOptionMain(constants.nodeInnerSize, constants.dir); mainInner.IsSome {
		outerMainSize := mainInner.Value + rectMainAxisSum(constants.contentBoxInset, constants.dir)
		sizeSetMain(&constants.innerContainerSize, constants.dir, mainInner.Value)
		sizeSetMain(&constants.containerSize, constants.dir, outerMainSize)
	} else {
		determineContainerMainSize(t, availableSpace, flexLines, &constants)
		sizeOptionSetMain(&constants.nodeInnerSize, constants.dir, lib.Some(sizeMain(constants.innerContainerSize, constants.dir)))
		sizeOptionSetMain(&constants.nodeOuterSize, constants.dir, lib.Some(sizeMain(constants.containerSize, constants.dir)))

		// Re-resolve percentage gaps.
		containerStyle := t.GetFlexboxContainerStyle(node)
		calc := t.ResolveCalcValue
		innerContainerSizeMain := sizeMain(constants.innerContainerSize, constants.dir)
		mainGapLP := func() style.LengthPercentage {
			if constants.dir.IsRow() {
				return containerStyle.GetGap().Width
			}
			return containerStyle.GetGap().Height
		}()
		newGap := style.MaybeResolveLPOption(mainGapLP, lib.Some(innerContainerSizeMain), calc).UnwrapOr(0.0)
		sizeSetMain(&constants.gap, constants.dir, newGap)
	}

	// 6. Resolve the flexible lengths.
	for _, line := range flexLines {
		resolveFlexibleLengths(line, &constants)
	}

	// 9.4. Cross Size Determination
	// 7. Determine the hypothetical cross size of each item.
	for _, line := range flexLines {
		determineHypotheticalCrossSize(t, line, &constants, availableSpace)
	}

	// Calculate child baselines.
	calculateChildrenBaseLines(t, knownDimensions, availableSpace, flexLines, &constants)

	// 8. Calculate the cross size of each flex line.
	calculateCrossSize(flexLines, knownDimensions, &constants)

	// 9. Handle 'align-content: stretch'.
	handleAlignContentStretch(flexLines, knownDimensions, &constants)

	// 11. Determine the used cross size of each flex item.
	determineUsedCrossSize(t, flexLines, &constants)

	// 9.5. Main-Axis Alignment
	// 12. Distribute any remaining free space.
	distributeRemainingFreeSpace(flexLines, &constants)

	// 9.6. Cross-Axis Alignment
	// 13. Resolve cross-axis auto margins.
	resolveCrossAxisAutoMargins(flexLines, &constants)

	// 15. Determine the flex container's used cross size.
	totalLineCrossSize := determineContainerCrossSize(flexLines, knownDimensions, &constants)

	if runMode == tree.RunModeComputeSize {
		return tree.LayoutOutputFromOuterSize(constants.containerSize)
	}

	// 16. Align all flex lines per align-content.
	alignFlexLinesPerAlignContent(flexLines, &constants, totalLineCrossSize)

	// Do a final layout pass.
	inflowContentSize := finalLayoutPass(t, flexLines, &constants)

	// Perform absolute layout on absolute children.
	absoluteContentSize := performAbsoluteLayoutOnAbsoluteChildren(t, node, &constants)

	// Hidden layout for display:none children.
	childLen := t.ChildCount(node)
	for order := 0; order < childLen; order++ {
		child := t.GetChildId(node, order)
		if t.GetCoreContainerStyle(child).GetBoxGenerationMode() == style.BoxGenerationModeNone {
			t.SetUnroundedLayout(child, &tree.Layout{Order: uint32(order)})
			tree.PerformChildLayout(
				t, child,
				geometry.SizeOptionNone,
				geometry.SizeOptionNone,
				geometry.Size[style.AvailableSpace]{Width: style.AvailableSpaceMaxContentValue, Height: style.AvailableSpaceMaxContentValue},
				tree.SizingModeInherentSize,
				geometry.LineFalseBool,
			)
		}
	}

	// 8.5. Calculate flex container baselines.
	var firstVerticalBaseline lib.Option[float32]
	if len(flexLines) > 0 {
		var baselineChild *flexItem
		for _, item := range flexLines[0].items {
			if constants.isColumn || item.alignSelf == style.AlignItemsBaseline {
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
			firstVerticalBaseline = lib.Some(offsetVertical + baselineChild.baseline)
		}
	}

	return tree.LayoutOutputFromSizesAndBaselines(
		constants.containerSize,
		geometry.SizeF32Max(inflowContentSize, absoluteContentSize),
		geometry.Point[lib.Option[float32]]{X: lib.None[float32](), Y: firstVerticalBaseline},
	)
}

// ---------------------------------------------------------------------------
// computeConstants
// ---------------------------------------------------------------------------

func computeConstants(
	t tree.LayoutFlexboxContainer,
	node tree.NodeId,
	knownDimensions geometry.Size[lib.Option[float32]],
	parentSize geometry.Size[lib.Option[float32]],
) algoConstants {
	containerStyle := t.GetFlexboxContainerStyle(node)
	coreStyle := t.GetCoreContainerStyle(node)
	calc := t.ResolveCalcValue

	dir := containerStyle.GetFlexDirection()
	isRow := dir.IsRow()
	isColumn := dir.IsColumn()
	isWrap := containerStyle.GetFlexWrap() == style.FlexWrapWrap || containerStyle.GetFlexWrap() == style.FlexWrapWrapReverse
	isWrapReverse := containerStyle.GetFlexWrap() == style.FlexWrapWrapReverse

	aspectRatio := coreStyle.GetAspectRatio()
	margin := style.ResolveRectLPAWidth(coreStyle.GetMargin(), parentSize.Width, calc)
	padding := style.ResolveRectLPWidth(coreStyle.GetPadding(), parentSize.Width, calc)
	border := style.ResolveRectLPWidth(coreStyle.GetBorder(), parentSize.Width, calc)
	paddingBorderSum := geometry.SizeAddF32(geometry.RectSumAxes(padding), geometry.RectSumAxes(border))
	boxSizingAdjustment := geometry.SizeZeroF32
	if coreStyle.GetBoxSizing() == style.BoxSizingContentBox {
		boxSizingAdjustment = paddingBorderSum
	}

	alignItems := containerStyle.GetAlignItems().UnwrapOr(style.AlignItemsStretch)
	alignContent := containerStyle.GetAlignContent().UnwrapOr(style.AlignContentStretch)
	justifyContent := containerStyle.GetJustifyContent()
	layoutDirection := coreStyle.GetDirection()

	// Scrollbar gutters.
	overflowTransposed := coreStyle.GetOverflow().Transpose()
	scrollbarGutter := geometry.Point[float32]{
		X: func() float32 {
			if overflowTransposed.X == style.OverflowScroll {
				return coreStyle.GetScrollbarWidth()
			}
			return 0.0
		}(),
		Y: func() float32 {
			if overflowTransposed.Y == style.OverflowScroll {
				return coreStyle.GetScrollbarWidth()
			}
			return 0.0
		}(),
	}

	contentBoxInset := geometry.RectAddF32(padding, border)
	contentBoxInset.Bottom += scrollbarGutter.Y
	switch layoutDirection {
	case style.DirectionLTR:
		contentBoxInset.Right += scrollbarGutter.X
	case style.DirectionRTL:
		contentBoxInset.Left += scrollbarGutter.X
	}

	nodeOuterSize := knownDimensions
	nodeInnerSize := geometry.SizeMaybeSub(nodeOuterSize, geometry.Size[lib.Option[float32]]{
		Width:  lib.Some(geometry.RectHorizontalAxisSum(contentBoxInset)),
		Height: lib.Some(geometry.RectVerticalAxisSum(contentBoxInset)),
	})

	nodeInnerSizeOr := geometry.SizeOptionOr(nodeInnerSize, geometry.Size[lib.Option[float32]]{Width: lib.Some[float32](0), Height: lib.Some[float32](0)})
	gap := geometry.Size[float32]{
		Width:  style.MaybeResolveLPOption(containerStyle.GetGap().Width, nodeInnerSizeOr.Width, calc).UnwrapOr(0.0),
		Height: style.MaybeResolveLPOption(containerStyle.GetGap().Height, nodeInnerSizeOr.Height, calc).UnwrapOr(0.0),
	}

	return algoConstants{
		dir:             dir,
		layoutDirection: layoutDirection,
		isRow:           isRow,
		isColumn:        isColumn,
		isWrap:          isWrap,
		isWrapReverse:   isWrapReverse,
		minSize: geometry.SizeMaybeAdd(
			geometry.SizeOptionMaybeApplyAspectRatio(
				style.MaybeResolveSizeDimOption(coreStyle.GetMinSize(), parentSize, calc),
				aspectRatio,
			),
			geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)},
		),
		maxSize: geometry.SizeMaybeAdd(
			geometry.SizeOptionMaybeApplyAspectRatio(
				style.MaybeResolveSizeDimOption(coreStyle.GetMaxSize(), parentSize, calc),
				aspectRatio,
			),
			geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdjustment.Width), Height: lib.Some(boxSizingAdjustment.Height)},
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
		containerSize:      geometry.SizeZeroF32,
		innerContainerSize: geometry.SizeZeroF32,
	}
}

// ---------------------------------------------------------------------------
// generateAnonymousFlexItems
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// determineAvailableSpace
// ---------------------------------------------------------------------------

func determineAvailableSpace(
	knownDimensions geometry.Size[lib.Option[float32]],
	outerAvailableSpace geometry.Size[style.AvailableSpace],
	constants *algoConstants,
) geometry.Size[style.AvailableSpace] {
	var width style.AvailableSpace
	if knownDimensions.Width.IsSome {
		width = style.AvailableSpaceFromDefinite(knownDimensions.Width.Value - geometry.RectHorizontalAxisSum(constants.contentBoxInset))
	} else {
		width = style.AsMaybeSubF32(
			style.AsMaybeSubF32(outerAvailableSpace.Width, geometry.RectHorizontalAxisSum(constants.margin)),
			geometry.RectHorizontalAxisSum(constants.contentBoxInset),
		)
	}

	var height style.AvailableSpace
	if knownDimensions.Height.IsSome {
		height = style.AvailableSpaceFromDefinite(knownDimensions.Height.Value - geometry.RectVerticalAxisSum(constants.contentBoxInset))
	} else {
		height = style.AsMaybeSubF32(
			style.AsMaybeSubF32(outerAvailableSpace.Height, geometry.RectVerticalAxisSum(constants.margin)),
			geometry.RectVerticalAxisSum(constants.contentBoxInset),
		)
	}

	return geometry.Size[style.AvailableSpace]{Width: width, Height: height}
}

// ---------------------------------------------------------------------------
// determineFlexBaseSize
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// collectFlexLines
// ---------------------------------------------------------------------------

func collectFlexLines(
	constants *algoConstants,
	availableSpace geometry.Size[style.AvailableSpace],
	flexItems []*flexItem,
) []*flexLine {
	if !constants.isWrap {
		lines := make([]*flexLine, 0, 1)
		lines = append(lines, &flexLine{items: flexItems, crossSize: 0.0, offsetCross: 0.0})
		return lines
	}

	mainAxisAvailableSpace := func() style.AvailableSpace {
		maxMain := sizeOptionMain(constants.maxSize, constants.dir)
		if maxMain.IsSome {
			defVal := avMain(availableSpace, constants.dir).IntoOption().UnwrapOr(maxMain.Value)
			clamped := geometry.MaybeMaxF32(defVal, sizeOptionMain(constants.minSize, constants.dir))
			return style.AvailableSpaceFromDefinite(clamped)
		}
		return avMain(availableSpace, constants.dir)
	}()

	switch mainAxisAvailableSpace.Tag {
	case style.AvailableSpaceMaxContent:
		// Items will never wrap.
		lines := make([]*flexLine, 0, 1)
		lines = append(lines, &flexLine{items: flexItems, crossSize: 0.0, offsetCross: 0.0})
		return lines

	case style.AvailableSpaceMinContent:
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
	t tree.LayoutFlexboxContainer,
	availableSpace geometry.Size[style.AvailableSpace],
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
		case style.AvailableSpaceDefinite:
			mainAxisAvailSpace := avMain(availableSpace, dir).Value
			longestLineLength := float32(0.0)
			for _, line := range lines {
				lineMainAxisGap := sumAxisGaps(sizeMain(constants.gap, dir), len(line.items))
				totalTargetSize := float32(0.0)
				for _, child := range line.items {
					paddingBorderSum := rectMainAxisSum(geometry.RectAddF32(child.padding, child.border), dir)
					v := geometry.MaybeMaxF32(child.flexBasis, sizeOptionMain(child.minSize, dir)) + rectMainAxisSum(child.margin, dir)
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
				return geometry.F32Max(size, mainAxisAvailSpace)
			}
			return size

		case style.AvailableSpaceMinContent:
			if constants.isWrap {
				longestLineLength := float32(0.0)
				for _, line := range lines {
					lineMainAxisGap := sumAxisGaps(sizeMain(constants.gap, dir), len(line.items))
					totalTargetSize := float32(0.0)
					for _, child := range line.items {
						paddingBorderSum := rectMainAxisSum(geometry.RectAddF32(child.padding, child.border), dir)
						v := geometry.MaybeMaxF32(child.flexBasis, sizeOptionMain(child.minSize, dir)) + rectMainAxisSum(child.margin, dir)
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

					clampingBasis := geometry.MaybeMaxOption(lib.Some(item.flexBasis), stylePreferred)
					flexBasisMin := clampingBasis
					if item.flexShrink != 0.0 {
						flexBasisMin = lib.None[float32]()
					}
					flexBasisMax := clampingBasis
					if item.flexGrow != 0.0 {
						flexBasisMax = lib.None[float32]()
					}

					minMainSize := func() float32 {
						v := geometry.MaybeMaxOption(styleMin, flexBasisMin)
						v = v.Or(flexBasisMin)
						result := v.UnwrapOr(item.resolvedMinimumMainSize)
						if result < item.resolvedMinimumMainSize {
							result = item.resolvedMinimumMainSize
						}
						return result
					}()
					maxMainSize := func() float32 {
						v := geometry.MaybeMinOption(styleMax, flexBasisMax)
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

						childKnownDimensions := sizeOptionWithMain(item.size, dir, lib.None[float32]())
						if item.alignSelf == style.AlignItemsStretch && !sizeOptionCross(childKnownDimensions, dir).IsSome {
							sizeSetCross(&childKnownDimensions, dir,
								geometry.MaybeSubOption(crossAxisAvailableSpace.IntoOption(), lib.Some(rectCrossAxisSum(item.margin, dir))),
							)
						}

						contentMainSize := tree.MeasureChildSize(
							t, item.node,
							childKnownDimensions,
							constants.nodeInnerSize,
							childAvailableSpace,
							tree.SizingModeInherentSize,
							dir.MainAxis(),
							geometry.LineFalseBool,
						) + rectMainAxisSum(item.margin, dir)

						if constants.isRow {
							return geometry.F32Max(geometry.MaybeClampF32(contentMainSize, styleMin, styleMax), mainContentBoxInset)
						}
						return geometry.F32Max(geometry.MaybeClampF32(geometry.F32Max(contentMainSize, item.flexBasis), styleMin, styleMax), mainContentBoxInset)
					}()

					// content_flex_fraction
					diff := contentContribution - item.flexBasis
					if diff > 0 {
						item.contentFlexFraction = diff / geometry.F32Max(1.0, item.flexGrow)
					} else if diff < 0 {
						scaledShrinkFactor := geometry.F32Max(1.0, item.flexShrink*item.innerFlexBasis)
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
						flexContribution = geometry.F32Max(1.0, item.flexGrow) * flexFraction
					} else if item.contentFlexFraction < 0 {
						scaledShrinkFactor := geometry.F32Max(1.0, item.flexShrink) * item.innerFlexBasis
						flexContribution = scaledShrinkFactor * flexFraction
					}
					size := item.flexBasis + flexContribution
					sizeSetMain(&item.outerTargetSize, constants.dir, size)
					sizeSetMain(&item.targetSize, constants.dir, size)
					itemMainSizeSum += size
				}

				gapSum := sumAxisGaps(sizeMain(constants.gap, dir), len(line.items))
				mainSize = geometry.F32Max(mainSize, itemMainSizeSum+gapSum)
			}
			return mainSize + mainContentBoxInset
		}
	}()

	outerMainSize = geometry.MaybeClampF32(outerMainSize, sizeOptionMain(constants.minSize, dir), sizeOptionMain(constants.maxSize, dir))
	scrollbarGutterMain := pointMain(constants.scrollbarGutter, dir)
	if outerMainSize < mainContentBoxInset-scrollbarGutterMain {
		outerMainSize = mainContentBoxInset - scrollbarGutterMain
	}

	innerMainSize := geometry.F32Max(outerMainSize-mainContentBoxInset, 0.0)
	sizeSetMain(&constants.containerSize, dir, outerMainSize)
	sizeSetMain(&constants.innerContainerSize, dir, innerMainSize)
	sizeOptionSetMain(&constants.nodeInnerSize, dir, lib.Some(innerMainSize))
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

	initialFreeSpace := geometry.MaybeSubF32(sizeOptionMain(constants.nodeInnerSize, constants.dir).UnwrapOr(0.0),
		lib.Some(usedSpace))

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
			freeSpace = geometry.MaybeMinF32(
				initialFreeSpace*sumFlexGrow-totalMainAxisGap,
				geometry.MaybeSubOption(nodeInnerMainOpt, lib.Some(usedSpace)),
			)
		} else if shrinking && sumFlexShrink < 1.0 {
			freeSpace = geometry.MaybeMaxF32(
				initialFreeSpace*sumFlexShrink-totalMainAxisGap,
				geometry.MaybeSubOption(nodeInnerMainOpt, lib.Some(usedSpace)),
			)
		} else {
			maybeFS := geometry.MaybeSubOption(nodeInnerMainOpt, lib.Some(usedSpace))
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
			resolvedMinMain := lib.Some(child.resolvedMinimumMainSize)
			maxMain := sizeOptionMain(child.maxSize, constants.dir)
			clamped := geometry.F32Max(geometry.MaybeClampF32(sizeMain(child.targetSize, constants.dir), resolvedMinMain, maxMain), 0.0)
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

// ---------------------------------------------------------------------------
// calculateChildrenBaseLines
// ---------------------------------------------------------------------------

func calculateChildrenBaseLines(
	t tree.LayoutFlexboxContainer,
	nodeSize geometry.Size[lib.Option[float32]],
	availableSpace geometry.Size[style.AvailableSpace],
	flexLines []*flexLine,
	constants *algoConstants,
) {
	if !constants.isRow {
		return
	}

	for _, line := range flexLines {
		lineBaselineChildCount := 0
		for _, child := range line.items {
			if child.alignSelf == style.AlignItemsBaseline {
				lineBaselineChildCount++
			}
		}
		if lineBaselineChildCount <= 1 {
			continue
		}

		for _, child := range line.items {
			if child.alignSelf != style.AlignItemsBaseline {
				continue
			}

			knownDims := geometry.Size[lib.Option[float32]]{
				Width: func() lib.Option[float32] {
					if constants.isRow {
						return lib.Some(sizeMain(child.targetSize, constants.dir))
					}
					return lib.Some(sizeCross(child.hypotheticalInnerSize, constants.dir))
				}(),
				Height: func() lib.Option[float32] {
					if constants.isRow {
						return lib.Some(sizeCross(child.hypotheticalInnerSize, constants.dir))
					}
					return lib.Some(sizeMain(child.targetSize, constants.dir))
				}(),
			}

			avSpace := geometry.Size[style.AvailableSpace]{
				Width: func() style.AvailableSpace {
					if constants.isRow {
						return style.AvailableSpaceFromDefinite(constants.containerSize.Width)
					}
					return availableSpace.Width.MaybeSet(nodeSize.Width)
				}(),
				Height: func() style.AvailableSpace {
					if constants.isRow {
						return availableSpace.Height.MaybeSet(nodeSize.Height)
					}
					return style.AvailableSpaceFromDefinite(constants.containerSize.Height)
				}(),
			}

			result := tree.PerformChildLayout(
				t, child.node,
				knownDims,
				constants.nodeInnerSize,
				avSpace,
				tree.SizingModeContentSize,
				geometry.LineFalseBool,
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

func calculateCrossSize(flexLines []*flexLine, nodeSize geometry.Size[lib.Option[float32]], constants *algoConstants) {
	if !constants.isWrap && sizeOptionCross(nodeSize, constants.dir).IsSome {
		crossAxisPaddingBorder := rectCrossAxisSum(constants.contentBoxInset, constants.dir)
		crossMinSize := sizeOptionCross(constants.minSize, constants.dir)
		crossMaxSize := sizeOptionCross(constants.maxSize, constants.dir)
		v := geometry.MaybeClampF32(
			sizeOptionCross(nodeSize, constants.dir).Value,
			crossMinSize,
			crossMaxSize,
		)
		v = geometry.F32Max(v-crossAxisPaddingBorder, 0.0)
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
				if child.alignSelf == style.AlignItemsBaseline &&
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
			flexLines[0].crossSize = geometry.MaybeClampF32(
				flexLines[0].crossSize,
				geometry.MaybeSubOption(crossMinSize, lib.Some(crossAxisPaddingBorder)),
				geometry.MaybeSubOption(crossMaxSize, lib.Some(crossAxisPaddingBorder)),
			)
		}
	}
}

// ---------------------------------------------------------------------------
// handleAlignContentStretch
// ---------------------------------------------------------------------------

func handleAlignContentStretch(flexLines []*flexLine, nodeSize geometry.Size[lib.Option[float32]], constants *algoConstants) {
	if constants.alignContent != style.AlignContentStretch {
		return
	}

	crossAxisPaddingBorder := rectCrossAxisSum(constants.contentBoxInset, constants.dir)
	crossMinSize := sizeOptionCross(constants.minSize, constants.dir)
	crossMaxSize := sizeOptionCross(constants.maxSize, constants.dir)

	containerMinInnerCross := func() float32 {
		v := sizeOptionCross(nodeSize, constants.dir).Or(crossMinSize)
		v = geometry.MaybeClampOption(v, crossMinSize, crossMaxSize)
		v = geometry.MaybeSubOption(v, lib.Some(crossAxisPaddingBorder))
		v = geometry.MaybeMaxOption(v, lib.Some[float32](0.0))
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
	t tree.LayoutFlexboxContainer,
	flexLines []*flexLine,
	constants *algoConstants,
) {
	calc := t.ResolveCalcValue
	for _, line := range flexLines {
		lineCrossSize := line.crossSize

		for _, child := range line.items {
			childCoreStyle := t.GetCoreContainerStyle(child.node)

			var crossTargetSize float32
			if child.alignSelf == style.AlignItemsStretch &&
				!rectBoolCrossStart(child.marginIsAuto, constants.dir) &&
				!rectBoolCrossEnd(child.marginIsAuto, constants.dir) &&
				dimCrossIsAuto(childCoreStyle.GetSize(), constants.dir) {

				// Compute max_size_ignoring_aspect_ratio.
				p := style.ResolveRectLPWidth(childCoreStyle.GetPadding(), constants.nodeInnerSize.Width, calc)
				b := style.ResolveRectLPWidth(childCoreStyle.GetBorder(), constants.nodeInnerSize.Width, calc)
				pbSum := geometry.SizeAddF32(geometry.RectSumAxes(p), geometry.RectSumAxes(b))
				boxSizAdj := geometry.SizeZeroF32
				if childCoreStyle.GetBoxSizing() == style.BoxSizingContentBox {
					boxSizAdj = pbSum
				}
				maxSizeIgnoringAR := geometry.SizeMaybeAdd(
					style.MaybeResolveSizeDimOption(childCoreStyle.GetMaxSize(), constants.nodeInnerSize, calc),
					geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizAdj.Width), Height: lib.Some(boxSizAdj.Height)},
				)

				crossTargetSize = geometry.MaybeClampF32(
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
		rawJCMode := style.AlignContentFlexStart
		if constants.justifyContent.IsSome {
			rawJCMode = constants.justifyContent.Value
		}
		jcMode := compute.ApplyAlignmentFallback(freeSpace, numItems, rawJCMode, isSafe)

		if layoutReverse {
			for revI := len(line.items) - 1; revI >= 0; revI-- {
				enumI := len(line.items) - 1 - revI
				child := line.items[revI]
				child.offsetMain = compute.ComputeAlignmentOffset(freeSpace, numItems, gap, jcMode, layoutReverse, enumI == 0)
			}
		} else {
			for i, child := range line.items {
				child.offsetMain = compute.ComputeAlignmentOffset(freeSpace, numItems, gap, jcMode, layoutReverse, i == 0)
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
	case style.AlignItemsStart:
		if crossAxisShouldReverse {
			return freeSpace
		}
		return 0.0
	case style.AlignItemsFlexStart:
		if constants.isWrapReverse != crossAxisShouldReverse {
			return freeSpace
		}
		return 0.0
	case style.AlignItemsEnd:
		if crossAxisShouldReverse {
			return 0.0
		}
		return freeSpace
	case style.AlignItemsFlexEnd:
		if constants.isWrapReverse != crossAxisShouldReverse {
			return 0.0
		}
		return freeSpace
	case style.AlignItemsCenter:
		return freeSpace / 2.0
	case style.AlignItemsBaseline:
		if constants.isRow {
			return maxBaseline - child.baseline
		}
		baselineColumnShouldReverse := crossAxisShouldReverse && !constants.isWrap
		if constants.isWrapReverse != baselineColumnShouldReverse {
			return freeSpace
		}
		return 0.0
	case style.AlignItemsStretch:
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
	nodeSize geometry.Size[lib.Option[float32]],
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
	outerContainerSize = geometry.MaybeClampF32(outerContainerSize, minCrossSize, maxCrossSize)
	if outerContainerSize < paddingBorderSum-crossScrollbarGutter {
		outerContainerSize = paddingBorderSum - crossScrollbarGutter
	}
	innerContainerSize := geometry.F32Max(outerContainerSize-paddingBorderSum, 0.0)

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

	alignContentMode := compute.ApplyAlignmentFallback(freeSpace, numLines, constants.alignContent, isSafe)

	if constants.isWrapReverse {
		for revI := numLines - 1; revI >= 0; revI-- {
			enumI := numLines - 1 - revI
			line := flexLines[revI]
			line.offsetCross = compute.ComputeAlignmentOffset(freeSpace, numLines, gap, alignContentMode, constants.isWrapReverse, enumI == 0)
		}
	} else {
		for i, line := range flexLines {
			line.offsetCross = compute.ComputeAlignmentOffset(freeSpace, numLines, gap, alignContentMode, constants.isWrapReverse, i == 0)
		}
	}
}

// ---------------------------------------------------------------------------
// calculateFlexItem
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// calculateLayoutLine
// ---------------------------------------------------------------------------

func calculateLayoutLine(
	t tree.LayoutFlexboxContainer,
	line *flexLine,
	totalOffsetCross *float32,
	contentSize *geometry.Size[float32],
	containerSize geometry.Size[float32],
	nodeInnerSize geometry.Size[lib.Option[float32]],
	paddingBorder geometry.Rect[float32],
	direction style.FlexDirection,
	layoutDirection style.Direction,
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
				t, line.items[revI],
				&totalOffsetMain, *totalOffsetCross, lineOffsetCross,
				contentSize, containerSize, nodeInnerSize, direction, layoutDirection,
			)
		}
	} else {
		for _, item := range line.items {
			calculateFlexItem(
				t, item,
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
	t tree.LayoutFlexboxContainer,
	flexLines []*flexLine,
	constants *algoConstants,
) geometry.Size[float32] {
	var totalOffsetCross float32
	if constants.isColumn && constants.layoutDirection.IsRTL() {
		totalOffsetCross = constants.containerSize.Width - rectCrossEnd(constants.contentBoxInset, constants.dir)
	} else {
		totalOffsetCross = rectCrossStart(constants.contentBoxInset, constants.dir)
	}

	contentSize := geometry.SizeZeroF32

	if constants.isWrapReverse {
		for revI := len(flexLines) - 1; revI >= 0; revI-- {
			calculateLayoutLine(
				t, flexLines[revI],
				&totalOffsetCross, &contentSize,
				constants.containerSize, constants.nodeInnerSize, constants.contentBoxInset,
				constants.dir, constants.layoutDirection,
			)
		}
	} else {
		for _, line := range flexLines {
			calculateLayoutLine(
				t, line,
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
	t tree.LayoutFlexboxContainer,
	node tree.NodeId,
	constants *algoConstants,
) geometry.Size[float32] {
	containerWidth := constants.containerSize.Width
	containerHeight := constants.containerSize.Height
	insetRelativeSize := geometry.SizeSubF32(
		geometry.SizeSubF32(constants.containerSize, geometry.RectSumAxes(constants.border)),
		geometry.PointToSize(constants.scrollbarGutter),
	)

	contentSize := geometry.SizeZeroF32
	calc := t.ResolveCalcValue

	for order := 0; order < t.ChildCount(node); order++ {
		child := t.GetChildId(node, order)
		childStyle := t.GetFlexboxChildStyle(child)
		childCoreStyle := t.GetCoreContainerStyle(child)

		if childCoreStyle.GetBoxGenerationMode() == style.BoxGenerationModeNone || childCoreStyle.GetPosition() != style.PositionAbsolute {
			continue
		}

		overflow := childCoreStyle.GetOverflow()
		scrollbarWidth := childCoreStyle.GetScrollbarWidth()
		aspectRatio := childCoreStyle.GetAspectRatio()
		alignSelf := childStyle.GetAlignSelf().UnwrapOr(constants.alignItems)
		margin := resolveRectLPAToOption(childCoreStyle.GetMargin(), insetRelativeSize.Width, calc)
		padding := style.ResolveRectLPWidth(childCoreStyle.GetPadding(), lib.Some(insetRelativeSize.Width), calc)
		border := style.ResolveRectLPWidth(childCoreStyle.GetBorder(), lib.Some(insetRelativeSize.Width), calc)
		paddingBorderSum := geometry.SizeAddF32(geometry.RectSumAxes(padding), geometry.RectSumAxes(border))
		boxSizingAdj := geometry.SizeZeroF32
		if childCoreStyle.GetBoxSizing() == style.BoxSizingContentBox {
			boxSizingAdj = paddingBorderSum
		}

		// Resolve inset.
		insetS := childCoreStyle.GetInset()
		left := style.MaybeResolveLPAF32(insetS.Left, insetRelativeSize.Width, calc)
		right := style.MaybeResolveLPAF32(insetS.Right, insetRelativeSize.Width, calc)
		top := style.MaybeResolveLPAF32(insetS.Top, insetRelativeSize.Height, calc)
		bottom := style.MaybeResolveLPAF32(insetS.Bottom, insetRelativeSize.Height, calc)

		// Compute known dimensions from style.
		insetRelSizeOpt := geometry.Size[lib.Option[float32]]{Width: lib.Some(insetRelativeSize.Width), Height: lib.Some(insetRelativeSize.Height)}
		styleSize := geometry.SizeMaybeAdd(
			geometry.SizeOptionMaybeApplyAspectRatio(
				style.MaybeResolveSizeDimOption(childCoreStyle.GetSize(), insetRelSizeOpt, calc),
				aspectRatio,
			),
			geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdj.Width), Height: lib.Some(boxSizingAdj.Height)},
		)
		minSize := geometry.SizeMaybeMax(
			geometry.SizeOptionOr(
				geometry.SizeMaybeAdd(
					geometry.SizeOptionMaybeApplyAspectRatio(
						style.MaybeResolveSizeDimOption(childCoreStyle.GetMinSize(), insetRelSizeOpt, calc),
						aspectRatio,
					),
					geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdj.Width), Height: lib.Some(boxSizingAdj.Height)},
				),
				geometry.Size[lib.Option[float32]]{Width: lib.Some(paddingBorderSum.Width), Height: lib.Some(paddingBorderSum.Height)},
			),
			geometry.Size[lib.Option[float32]]{Width: lib.Some(paddingBorderSum.Width), Height: lib.Some(paddingBorderSum.Height)},
		)
		maxSize := geometry.SizeMaybeAdd(
			geometry.SizeOptionMaybeApplyAspectRatio(
				style.MaybeResolveSizeDimOption(childCoreStyle.GetMaxSize(), insetRelSizeOpt, calc),
				aspectRatio,
			),
			geometry.Size[lib.Option[float32]]{Width: lib.Some(boxSizingAdj.Width), Height: lib.Some(boxSizingAdj.Height)},
		)
		knownDimensions := geometry.SizeMaybeClamp(styleSize, minSize, maxSize)

		// Fill in width from left/right.
		if !knownDimensions.Width.IsSome && left.IsSome && right.IsSome {
			newWidthRaw := insetRelativeSize.Width
			newWidthRaw = geometry.MaybeSubF32(newWidthRaw, margin.Left)
			newWidthRaw = geometry.MaybeSubF32(newWidthRaw, margin.Right)
			newWidthRaw -= left.Value
			newWidthRaw -= right.Value
			knownDimensions.Width = lib.Some(geometry.F32Max(newWidthRaw, 0.0))
			knownDimensions = geometry.SizeMaybeClamp(geometry.SizeOptionMaybeApplyAspectRatio(knownDimensions, aspectRatio), minSize, maxSize)
		}

		// Fill in height from top/bottom.
		if !knownDimensions.Height.IsSome && top.IsSome && bottom.IsSome {
			newHeightRaw := insetRelativeSize.Height
			newHeightRaw = geometry.MaybeSubF32(newHeightRaw, margin.Top)
			newHeightRaw = geometry.MaybeSubF32(newHeightRaw, margin.Bottom)
			newHeightRaw -= top.Value
			newHeightRaw -= bottom.Value
			knownDimensions.Height = lib.Some(geometry.F32Max(newHeightRaw, 0.0))
			knownDimensions = geometry.SizeMaybeClamp(geometry.SizeOptionMaybeApplyAspectRatio(knownDimensions, aspectRatio), minSize, maxSize)
		}

		measuredSize := measureChildSizeBoth(
			t, child,
			knownDimensions,
			constants.nodeInnerSize,
			geometry.Size[style.AvailableSpace]{
				Width:  style.AvailableSpaceFromDefinite(geometry.MaybeClampF32(containerWidth, minSize.Width, maxSize.Width)),
				Height: style.AvailableSpaceFromDefinite(geometry.MaybeClampF32(containerHeight, minSize.Height, maxSize.Height)),
			},
			tree.SizingModeInherentSize,
			geometry.LineFalseBool,
		)
		finalSize := geometry.SizeMaybeClamp(
			geometry.SizeOptionOr(knownDimensions, geometry.Size[lib.Option[float32]]{Width: lib.Some(measuredSize.Width), Height: lib.Some(measuredSize.Height)}),
			minSize, maxSize,
		)
		finalSizeF32 := geometry.Size[float32]{Width: finalSize.Width.UnwrapOr(0), Height: finalSize.Height.UnwrapOr(0)}

		layoutOutput := tree.PerformChildLayout(
			t, child,
			finalSize,
			constants.nodeInnerSize,
			geometry.Size[style.AvailableSpace]{
				Width:  style.AvailableSpaceFromDefinite(geometry.MaybeClampF32(containerWidth, minSize.Width, maxSize.Width)),
				Height: style.AvailableSpaceFromDefinite(geometry.MaybeClampF32(containerHeight, minSize.Height, maxSize.Height)),
			},
			tree.SizingModeInherentSize,
			geometry.LineFalseBool,
		)

		nonAutoMargin := geometry.Rect[float32]{
			Left:   margin.Left.UnwrapOr(0),
			Right:  margin.Right.UnwrapOr(0),
			Top:    margin.Top.UnwrapOr(0),
			Bottom: margin.Bottom.UnwrapOr(0),
		}

		freeSpaceF := geometry.Size[float32]{
			Width:  geometry.F32Max(constants.containerSize.Width-finalSizeF32.Width-geometry.RectHorizontalAxisSum(nonAutoMargin), 0),
			Height: geometry.F32Max(constants.containerSize.Height-finalSizeF32.Height-geometry.RectVerticalAxisSum(nonAutoMargin), 0),
		}

		// Expand auto margins.
		resolvedMargin := func() geometry.Rect[float32] {
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
			return geometry.Rect[float32]{
				Left:   margin.Left.UnwrapOr(autoMarginW),
				Right:  margin.Right.UnwrapOr(autoMarginW),
				Top:    margin.Top.UnwrapOr(autoMarginH),
				Bottom: margin.Bottom.UnwrapOr(autoMarginH),
			}
		}()

		// Determine flex-relative insets.
		var startMain, endMain, startCross, endCross lib.Option[float32]
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
			jc := style.AlignContentStart
			if constants.justifyContent.IsSome {
				jc = constants.justifyContent.Value
			}
			switch {
			case jc == style.AlignContentSpaceBetween ||
				(!mainAxisFlexStartReversed && (jc == style.AlignContentStretch || jc == style.AlignContentFlexStart)) ||
				(mainAxisFlexStartReversed && jc == style.AlignContentFlexEnd):
				offsetMain = rectMainStart(constants.contentBoxInset, constants.dir) + rectMainStart(resolvedMargin, constants.dir)
			case !mainAxisFlexStartReversed && jc == style.AlignContentStart:
				offsetMain = rectMainStart(constants.contentBoxInset, constants.dir) + rectMainStart(resolvedMargin, constants.dir)
			case mainAxisFlexStartReversed && jc == style.AlignContentStart:
				offsetMain = sizeMain(constants.containerSize, constants.dir) -
					rectMainEnd(constants.contentBoxInset, constants.dir) -
					sizeMain(finalSizeF32, constants.dir) -
					rectMainEnd(resolvedMargin, constants.dir)
			case !mainAxisFlexStartReversed && jc == style.AlignContentEnd:
				offsetMain = sizeMain(constants.containerSize, constants.dir) -
					rectMainEnd(constants.contentBoxInset, constants.dir) -
					sizeMain(finalSizeF32, constants.dir) -
					rectMainEnd(resolvedMargin, constants.dir)
			case mainAxisFlexStartReversed && jc == style.AlignContentEnd:
				offsetMain = rectMainStart(constants.contentBoxInset, constants.dir) + rectMainStart(resolvedMargin, constants.dir)
			case (!mainAxisFlexStartReversed && jc == style.AlignContentFlexEnd) ||
				(mainAxisFlexStartReversed && (jc == style.AlignContentFlexStart || jc == style.AlignContentStretch)):
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
			case (!crossAxisFlexStartReversed && alignSelf == style.AlignItemsStart) ||
				(crossAxisFlexStartReversed && alignSelf == style.AlignItemsEnd):
				offsetCrossVal = rectCrossStart(constants.contentBoxInset, constants.dir) + rectCrossStart(resolvedMargin, constants.dir)
			case (crossAxisFlexStartReversed && alignSelf == style.AlignItemsStart) ||
				(!crossAxisFlexStartReversed && alignSelf == style.AlignItemsEnd):
				offsetCrossVal = sizeCross(constants.containerSize, constants.dir) -
					rectCrossEnd(constants.contentBoxInset, constants.dir) -
					sizeCross(finalSizeF32, constants.dir) -
					rectCrossEnd(resolvedMargin, constants.dir)
			case (!crossAxisFlexStartReversed && (alignSelf == style.AlignItemsBaseline || alignSelf == style.AlignItemsStretch || alignSelf == style.AlignItemsFlexStart)) ||
				(crossAxisFlexStartReversed && alignSelf == style.AlignItemsFlexEnd):
				offsetCrossVal = rectCrossStart(constants.contentBoxInset, constants.dir) + rectCrossStart(resolvedMargin, constants.dir)
			case (crossAxisFlexStartReversed && (alignSelf == style.AlignItemsBaseline || alignSelf == style.AlignItemsStretch || alignSelf == style.AlignItemsFlexStart)) ||
				(!crossAxisFlexStartReversed && alignSelf == style.AlignItemsFlexEnd):
				offsetCrossVal = sizeCross(constants.containerSize, constants.dir) -
					rectCrossEnd(constants.contentBoxInset, constants.dir) -
					sizeCross(finalSizeF32, constants.dir) -
					rectCrossEnd(resolvedMargin, constants.dir)
			case alignSelf == style.AlignItemsCenter:
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

		var location geometry.Point[float32]
		if constants.isRow {
			location = geometry.Point[float32]{X: offsetMain, Y: offsetCrossVal}
		} else {
			location = geometry.Point[float32]{X: offsetCrossVal, Y: offsetMain}
		}

		scrollbarSize := geometry.Size[float32]{
			Width: func() float32 {
				if overflow.Y == style.OverflowScroll {
					return scrollbarWidth
				}
				return 0.0
			}(),
			Height: func() float32 {
				if overflow.X == style.OverflowScroll {
					return scrollbarWidth
				}
				return 0.0
			}(),
		}

		t.SetUnroundedLayout(child, &tree.Layout{
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
		sizeContentSizeContrib := geometry.Size[float32]{
			Width: func() float32 {
				if overflow.X == style.OverflowVisible {
					return geometry.F32Max(finalSizeF32.Width, layoutOutput.ContentSize.Width)
				}
				return finalSizeF32.Width
			}(),
			Height: func() float32 {
				if overflow.Y == style.OverflowVisible {
					return geometry.F32Max(finalSizeF32.Height, layoutOutput.ContentSize.Height)
				}
				return finalSizeF32.Height
			}(),
		}
		if geometry.SizeHasNonZeroArea(sizeContentSizeContrib) {
			absoluteAreaOffset := geometry.Point[float32]{
				X: constants.border.Left + func() float32 {
					if constants.layoutDirection.IsRTL() {
						return constants.scrollbarGutter.X
					}
					return 0
				}(),
				Y: constants.border.Top,
			}
			relativeLocation := geometry.Point[float32]{
				X: location.X - absoluteAreaOffset.X,
				Y: location.Y - absoluteAreaOffset.Y,
			}
			var contentSizeContrib geometry.Size[float32]
			if constants.layoutDirection.IsRTL() {
				overflowExtraWidth := geometry.F32Max(sizeContentSizeContrib.Width-finalSizeF32.Width, 0)
				contentSizeContrib = geometry.Size[float32]{
					Width:  geometry.F32Max(insetRelativeSize.Width-relativeLocation.X, 0) + overflowExtraWidth,
					Height: relativeLocation.Y + sizeContentSizeContrib.Height,
				}
			} else {
				contentSizeContrib = geometry.Size[float32]{
					Width:  relativeLocation.X + sizeContentSizeContrib.Width,
					Height: relativeLocation.Y + sizeContentSizeContrib.Height,
				}
			}
			contentSize = geometry.SizeF32Max(contentSize, contentSizeContrib)
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

// pointMain returns the main-axis component of a geometry.Point[float32] based on direction.
// (Already defined above for float32, but we also need it for the scrollbar gutter.)

// geometry.MaybeSubOption subtracts b from a where both are lib.Option[float32].
// Already defined in util_math.go.
