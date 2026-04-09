package flexbox

import (
	"math"

	"taffy-go/compute"
	"taffy-go/geometry"
	"taffy-go/lib"
	"taffy-go/style"
	"taffy-go/tree"
)

// flexLine holds a slice of flex items for one line plus cross-axis sizing info.
type flexLine struct {
	items       []*flexItem
	crossSize   float32
	offsetCross float32
}

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
