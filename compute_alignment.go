// Package taffy provides generic CSS alignment utilities shared between
// Flexbox and CSS Grid layout algorithms.
// This file is a port of compute/common/alignment.rs from the Taffy Rust layout engine.
package taffy

// applyAlignmentFallback implements CSS alignment fallback behaviour.
//
// In addition to the spec at https://www.w3.org/TR/css-align-3/ this
// implementation follows the resolution of
// https://github.com/w3c/csswg-drafts/issues/10154
//
// Fallback occurs in two cases:
//  1. If there is only a single item being aligned (or no free space) and the
//     alignment mode is a distributed keyword, the mode collapses to a simpler one.
//  2. If free space is negative and the alignment is "safe", the mode falls back
//     to Start alignment.
func applyAlignmentFallback(freeSpace float32, numItems int, alignmentMode AlignContent, isSafe bool) AlignContent {
	// 1. Distributed alignment keywords fall back when numItems <= 1 or freeSpace <= 0.
	if numItems <= 1 || freeSpace <= 0.0 {
		switch alignmentMode {
		case AlignContentStretch:
			alignmentMode = AlignContentFlexStart
			isSafe = true
		case AlignContentSpaceBetween:
			alignmentMode = AlignContentFlexStart
			isSafe = true
		case AlignContentSpaceAround:
			alignmentMode = AlignContentCenter
			isSafe = true
		case AlignContentSpaceEvenly:
			alignmentMode = AlignContentCenter
			isSafe = true
		}
	}

	// 2. Negative free space with a "safe" mode falls back to Start.
	if freeSpace <= 0.0 && isSafe {
		alignmentMode = AlignContentStart
	}

	return alignmentMode
}

// computeAlignmentOffset computes the offset for a given item based on the
// alignment mode. It is used for both align-content and justify-content, and
// for both the Flexbox and CSS Grid algorithms.
//
// CSS Grid does not apply gaps as part of alignment, so pass gap=0 for Grid.
//
// isFirst should be true when computing the offset for the first item in the
// container; it should be false for subsequent items (where the gap contributes).
func computeAlignmentOffset(freeSpace float32, numItems int, gap float32, alignmentMode AlignContent, layoutIsFlexReversed bool, isFirst bool) float32 {
	if isFirst {
		switch alignmentMode {
		case AlignContentStart:
			return 0.0
		case AlignContentFlexStart:
			if layoutIsFlexReversed {
				return freeSpace
			}
			return 0.0
		case AlignContentEnd:
			return freeSpace
		case AlignContentFlexEnd:
			if layoutIsFlexReversed {
				return 0.0
			}
			return freeSpace
		case AlignContentCenter:
			return freeSpace / 2.0
		case AlignContentStretch:
			return 0.0
		case AlignContentSpaceBetween:
			return 0.0
		case AlignContentSpaceAround:
			if freeSpace >= 0.0 {
				return (freeSpace / float32(numItems)) / 2.0
			}
			return freeSpace / 2.0
		case AlignContentSpaceEvenly:
			if freeSpace >= 0.0 {
				return freeSpace / float32(numItems+1)
			}
			return freeSpace / 2.0
		default:
			return 0.0
		}
	}

	// For subsequent items the free space used is floored at 0.
	clampedFreeSpace := freeSpace
	if clampedFreeSpace < 0.0 {
		clampedFreeSpace = 0.0
	}

	switch alignmentMode {
	case AlignContentStart:
		return gap
	case AlignContentFlexStart:
		return gap
	case AlignContentEnd:
		return gap
	case AlignContentFlexEnd:
		return gap
	case AlignContentCenter:
		return gap
	case AlignContentStretch:
		return gap
	case AlignContentSpaceBetween:
		return gap + clampedFreeSpace/float32(numItems-1)
	case AlignContentSpaceAround:
		return gap + clampedFreeSpace/float32(numItems)
	case AlignContentSpaceEvenly:
		return gap + clampedFreeSpace/float32(numItems+1)
	default:
		return gap
	}
}
