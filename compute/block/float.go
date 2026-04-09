// Package block provides float positioning within a block formatting context.
// This file is a port of compute/float.rs from the Taffy Rust layout engine.
package block

import (
	"math"

	"taffy-go/geometry"
	"taffy-go/lib"
	"taffy-go/style"
)

// contentSlot is an empty "slot" that avoids floats, suitable for non-floated content
// to be laid out into.
type contentSlot struct {
	// segmentID is the id of the segment that the slot starts in (None means no segment).
	segmentID lib.Option[int]
	// x is the x position of the start of the slot.
	x float32
	// y is the y position of the start of the slot.
	y float32
	// width is the width of the slot.
	width float32
	// height is the height of the slot.
	height float32
}

// placedFloatedBox represents a floated box that has been placed in the context.
type placedFloatedBox struct {
	// width is the width of the box.
	width float32
	// height is the height of the box.
	height float32
	// xInset is the horizontal distance from the edge of the container that the box
	// is floated towards (from the left for left floats, from the right for right floats).
	xInset float32
	// y is the vertical distance from top edge of the container.
	y float32
}

// segment is a non-overlapping horizontal segment of the Block Formatting Context container.
type segment struct {
	// yStart and yEnd define the vertical extent of the segment.
	yStart float32
	yEnd   float32
	// insets holds the left inset (index 0) and right inset (index 1).
	insets [2]float32
}

// fitsFloatWidth reports whether the segment can fit the passed floated box horizontally.
func (s *segment) fitsFloatWidth(floatedBox geometry.Size[float32], direction style.FloatDirection, bfcWidth float32) bool {
	slot := int(direction)
	return s.insets[slot] == 0.0 || (bfcWidth-floatedBox.Width-s.insetSum()) >= 0.0
}

// insetSum returns the total space taken up by both insets.
func (s *segment) insetSum() float32 {
	return s.insets[0] + s.insets[1]
}

// floatFitter is a helper type for placing a single floated box.
// Given a pinned starting y position, it determines if there is enough horizontal
// and vertical space across a range of segments.
type floatFitter struct {
	bfcWidth   float32
	slotHeight float64
	insets     [2]float32
}

func newFloatFitter(bfcWidth float32, slotHeight float32, insets [2]float32) floatFitter {
	return floatFitter{
		bfcWidth:   bfcWidth,
		slotHeight: float64(slotHeight),
		insets:     insets,
	}
}

func (f *floatFitter) unionInsets(insets [2]float32) {
	f.insets[0] = geometry.F32Max(f.insets[0], insets[0])
	f.insets[1] = geometry.F32Max(f.insets[1], insets[1])
}

func (f *floatFitter) fitsHorizontally(width float32) bool {
	return f.insets == [2]float32{0.0, 0.0} || f.bfcWidth-f.insets[0]-f.insets[1]-width >= 0.0
}

func (f *floatFitter) addHeight(height float32) {
	f.slotHeight += float64(height)
}

func (f *floatFitter) fitsVertically(height float32) bool {
	return f.slotHeight >= float64(height)
}

// floatContext is the main context for placing floated boxes.
type floatContext struct {
	availableWidth float32
	hasFloatsFlag  bool
	leftFloats     []placedFloatedBox
	rightFloats    []placedFloatedBox
	segments       []segment
	// lastPlacedFloats holds closed-open ranges indicating which segment the last placed
	// float was placed in (index 0 = left, index 1 = right).
	lastPlacedFloats [2][2]int // [2]{start, end}
}

func newFloatContext() *floatContext {
	return &floatContext{}
}

func (fc *floatContext) hasFloats() bool {
	return fc.hasFloatsFlag
}

func (fc *floatContext) hasActiveFloats(minY float32) bool {
	if !fc.hasFloatsFlag {
		return false
	}
	if len(fc.segments) == 0 {
		return false
	}
	return fc.segments[len(fc.segments)-1].yEnd > minY
}

func (fc *floatContext) setWidth(availableWidth float32) {
	fc.availableWidth = availableWidth
}

// subdivideSegment divides a segment into two so a new float can be placed at exact segment boundaries.
func (fc *floatContext) subdivideSegment(idx int, divideAtY float32) {
	old := &fc.segments[idx]
	newSeg := segment{
		insets: old.insets,
		yStart: divideAtY,
		yEnd:   old.yEnd,
	}
	old.yEnd = divideAtY

	// Insert newSeg after idx
	fc.segments = append(fc.segments, segment{})
	copy(fc.segments[idx+2:], fc.segments[idx+1:])
	fc.segments[idx+1] = newSeg
}

// updateLastPlacedFloat updates the last placed float start and end values.
func (fc *floatContext) updateLastPlacedFloat(direction style.FloatDirection, placementStart, placementEnd int) {
	slot := int(direction)
	if placementStart > fc.lastPlacedFloats[slot][0] {
		fc.lastPlacedFloats[slot][0] = placementStart
	}
	if placementEnd > fc.lastPlacedFloats[slot][1] {
		fc.lastPlacedFloats[slot][1] = placementEnd
	}
}

// placeFloatedBox positions a floated box within the context, returning the (x, y) coordinates.
func (fc *floatContext) placeFloatedBox(
	floatedBox geometry.Size[float32],
	minY float32,
	containingBlockInsets [2]float32,
	direction style.FloatDirection,
	clear style.Clear,
) geometry.Point[float32] {
	fc.hasFloatsFlag = true

	placed := fc.placeFloatedBoxInner(floatedBox, minY, containingBlockInsets, direction, clear)

	xInset := placed.xInset
	y := placed.y
	switch direction {
	case style.FloatDirectionLeft:
		fc.leftFloats = append(fc.leftFloats, placed)
		return geometry.Point[float32]{X: xInset, Y: y}
	default: // FloatDirectionRight
		fc.rightFloats = append(fc.rightFloats, placed)
		return geometry.Point[float32]{X: fc.availableWidth - xInset - floatedBox.Width, Y: y}
	}
}

// placeFloatedBoxInner is the inner implementation of float placement.
func (fc *floatContext) placeFloatedBoxInner(
	floatedBox geometry.Size[float32],
	minY float32,
	containingBlockInsets [2]float32,
	direction style.FloatDirection,
	clear style.Clear,
) placedFloatedBox {
	slot := int(direction)

	// Compute the high water mark based on clear and direction
	var hwm int
	switch clear {
	case style.ClearLeft:
		floatDirStart := fc.lastPlacedFloats[slot][0]
		leftEnd := fc.lastPlacedFloats[0][1]
		hwm = max(floatDirStart, leftEnd+1)
	case style.ClearRight:
		floatDirStart := fc.lastPlacedFloats[slot][0]
		rightEnd := fc.lastPlacedFloats[1][1]
		hwm = max(floatDirStart, rightEnd+1)
	case style.ClearBoth:
		leftEnd := fc.lastPlacedFloats[0][1]
		rightEnd := fc.lastPlacedFloats[1][1]
		hwm = max(leftEnd, rightEnd) + 1
	default: // ClearNone
		hwm = fc.lastPlacedFloats[slot][0]
	}

	// Find start segment at or below min_y
	startIdx := len(fc.segments)
	if hwm < len(fc.segments) {
		for i := hwm; i < len(fc.segments); i++ {
			if fc.segments[i].yEnd > minY {
				startIdx = i
				break
			}
		}
	}

	startY := minY
	endIdx := startIdx

	// placedInset and startOpt/endOpt track the result of the placement loop
	type placementResult struct {
		startExists bool
		startIdx    int
		endExists   bool
		endIdx      int
		placedInset float32
	}

	var result placementResult

	// Loop to find placement
outer:
	for {
		// Start segment does not exist
		if startIdx >= len(fc.segments) {
			result = placementResult{
				startExists: false,
				placedInset: containingBlockInsets[slot],
			}
			break
		}

		startSeg := &fc.segments[startIdx]

		// Check if start segment fits horizontally
		if !startSeg.fitsFloatWidth(floatedBox, direction, fc.availableWidth) {
			startIdx++
			if endIdx < startIdx {
				endIdx = startIdx
			}
			continue
		}

		startY = geometry.F32Max(startY, startSeg.yStart)
		availableHeight := startSeg.yEnd - startY
		fitter := newFloatFitter(fc.availableWidth, availableHeight, containingBlockInsets)
		fitter.unionInsets(startSeg.insets)

		// Inner loop to find end segment
		for {
			if endIdx >= len(fc.segments) {
				inset := fitter.insets[slot]
				result = placementResult{
					startExists: true,
					startIdx:    startIdx,
					endExists:   false,
					placedInset: inset,
				}
				break outer
			}

			endSeg := &fc.segments[endIdx]

			// Check horizontal fit
			fitter.unionInsets(endSeg.insets)
			if !fitter.fitsHorizontally(floatedBox.Width) {
				startIdx++
				if endIdx < startIdx {
					endIdx = startIdx
				}
				continue outer
			}

			// Check vertical fit
			if endIdx != startIdx {
				fitter.addHeight(endSeg.yEnd - endSeg.yStart)
			}
			if !fitter.fitsVertically(floatedBox.Height) {
				endIdx++
				continue
			}

			inset := fitter.insets[slot]
			result = placementResult{
				startExists: true,
				startIdx:    startIdx,
				endExists:   true,
				endIdx:      endIdx,
				placedInset: inset,
			}
			break outer
		}
	}

	placedInset := result.placedInset

	// Short-circuit for zero-sized boxes
	if floatedBox.Width == 0.0 || floatedBox.Height == 0.0 {
		return placedFloatedBox{
			width:  floatedBox.Width,
			height: floatedBox.Height,
			y:      startY,
			xInset: placedInset,
		}
	}

	// Handle case where floated box is placed after all existing segments
	if !result.startExists {
		lastYEnd := float32(0.0)
		if len(fc.segments) > 0 {
			lastYEnd = fc.segments[len(fc.segments)-1].yEnd
		}
		if startY > lastYEnd {
			fc.segments = append(fc.segments, segment{yStart: lastYEnd, yEnd: startY, insets: [2]float32{0, 0}})
		}

		finalStartY := geometry.F32Max(lastYEnd, startY)
		insets := containingBlockInsets
		insets[slot] += floatedBox.Width
		fc.segments = append(fc.segments, segment{yStart: finalStartY, yEnd: finalStartY + floatedBox.Height, insets: insets})

		sIdx := len(fc.segments) - 1
		fc.updateLastPlacedFloat(direction, sIdx, sIdx+1)

		return placedFloatedBox{
			width:  floatedBox.Width,
			height: floatedBox.Height,
			y:      finalStartY,
			xInset: containingBlockInsets[slot],
		}
	}

	// Unwrap start index
	si := result.startIdx

	// Subdivide if needed at start
	if startY != fc.segments[si].yStart {
		fc.subdivideSegment(si, startY)
		si++
		if result.endExists {
			result.endIdx++
		}
	}

	var ei int
	if !result.endExists {
		lastYEnd := float32(0.0)
		if len(fc.segments) > 0 {
			lastYEnd = fc.segments[len(fc.segments)-1].yEnd
		}
		if minY > lastYEnd {
			fc.segments = append(fc.segments, segment{yStart: lastYEnd, yEnd: minY, insets: [2]float32{0, 0}})
		}
		ei = len(fc.segments) - 1
	} else {
		ei = result.endIdx
		endY := startY + floatedBox.Height
		if endY != fc.segments[ei].yEnd {
			fc.subdivideSegment(ei, endY)
		}
	}

	// Update inset for the range of segments
	placedInsetPlusWidth := placedInset + floatedBox.Width
	for i := si; i <= ei; i++ {
		fc.segments[i].insets[slot] = placedInsetPlusWidth
	}

	// Update last placed float
	fc.updateLastPlacedFloat(direction, si, ei+1)

	return placedFloatedBox{
		width:  floatedBox.Width,
		height: floatedBox.Height,
		y:      startY,
		xInset: placedInset,
	}
}

// clearedSegment gets the end segment index for the given clear property.
func (fc *floatContext) clearedSegment(clear style.Clear) (int, bool) {
	switch clear {
	case style.ClearLeft:
		return fc.lastPlacedFloats[0][1], true
	case style.ClearRight:
		return fc.lastPlacedFloats[1][1], true
	case style.ClearBoth:
		leftEnd := fc.lastPlacedFloats[0][1]
		rightEnd := fc.lastPlacedFloats[1][1]
		return max(leftEnd, rightEnd), true
	default: // ClearNone
		return 0, false
	}
}

// clearedThreshold returns the bottom of the lowest relevant float for the specific clear property.
func (fc *floatContext) clearedThreshold(clear style.Clear) (float32, bool) {
	idx, ok := fc.clearedSegment(clear)
	if !ok {
		return 0, false
	}
	segIdx := max(idx, 1) - 1
	if segIdx >= len(fc.segments) {
		return 0, false
	}
	return fc.segments[segIdx].yEnd, true
}

// findContentSlot searches for a space suitable for laying out non-floated content into.
func (fc *floatContext) findContentSlot(
	minY float32,
	containingBlockInsets [2]float32,
	clear style.Clear,
	after *int,
) contentSlot {
	if !fc.hasActiveFloats(minY) {
		return contentSlot{
			x:      containingBlockInsets[0],
			y:      minY,
			width:  fc.availableWidth - containingBlockInsets[0] - containingBlockInsets[1],
			height: float32(math.Inf(1)),
		}
	}

	atLeast := 0
	if after != nil {
		atLeast = *after + 1
	}
	hwm := atLeast
	if idx, ok := fc.clearedSegment(clear); ok {
		if idx+1 > hwm {
			hwm = idx + 1
		}
	}

	startIdx := len(fc.segments)
	if hwm < len(fc.segments) {
		for i := hwm; i < len(fc.segments); i++ {
			if fc.segments[i].yEnd > minY {
				startIdx = i
				break
			}
		}
	}

	if startIdx < len(fc.segments) {
		seg := &fc.segments[startIdx]
		insetLeft := geometry.F32Max(seg.insets[0], containingBlockInsets[0])
		insetRight := geometry.F32Max(seg.insets[1], containingBlockInsets[1])
		return contentSlot{
			segmentID: lib.Some(startIdx),
			x:         insetLeft,
			y:         geometry.F32Max(seg.yStart, minY),
			width:     fc.availableWidth - insetLeft - insetRight,
			height:    float32(math.Inf(1)),
		}
	}

	return contentSlot{
		x:      containingBlockInsets[0],
		y:      minY,
		width:  fc.availableWidth - containingBlockInsets[0] - containingBlockInsets[1],
		height: float32(math.Inf(1)),
	}
}

// floatIntrinsicWidthCalculator computes the intrinsic width contribution of a set of floats.
type floatIntrinsicWidthCalculator struct {
	availableWidth style.AvailableSpace
	contribution   float32
}

func newFloatIntrinsicWidthCalculator(availableWidth style.AvailableSpace) *floatIntrinsicWidthCalculator {
	return &floatIntrinsicWidthCalculator{
		availableWidth: availableWidth,
		contribution:   0,
	}
}

func (c *floatIntrinsicWidthCalculator) addFloat(width float32, _ style.FloatDirection, _ style.Clear) {
	switch c.availableWidth.Tag {
	case style.AvailableSpaceDefinite:
		// We will never hit this code path with definite available space
	case style.AvailableSpaceMinContent:
		c.contribution = geometry.F32Max(c.contribution, width)
	case style.AvailableSpaceMaxContent:
		c.contribution += width
	}
}

func (c *floatIntrinsicWidthCalculator) result() float32 {
	return c.contribution
}
