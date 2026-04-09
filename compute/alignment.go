// Package compute provides generic CSS alignment utilities shared between
// Flexbox and CSS Grid layout algorithms.
// This file is a port of compute/common/alignment.rs from the Taffy Rust layout engine.
package compute

import "taffy-go/style"

// ApplyAlignmentFallback implements CSS alignment fallback behaviour.
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
func ApplyAlignmentFallback(freeSpace float32, numItems int, alignmentMode style.AlignContent, isSafe bool) style.AlignContent {
	// 1. Distributed alignment keywords fall back when numItems <= 1 or freeSpace <= 0.
	if numItems <= 1 || freeSpace <= 0.0 {
		switch alignmentMode {
		case style.AlignContentStretch:
			alignmentMode = style.AlignContentFlexStart
			isSafe = true
		case style.AlignContentSpaceBetween:
			alignmentMode = style.AlignContentFlexStart
			isSafe = true
		case style.AlignContentSpaceAround:
			alignmentMode = style.AlignContentCenter
			isSafe = true
		case style.AlignContentSpaceEvenly:
			alignmentMode = style.AlignContentCenter
			isSafe = true
		}
	}

	// 2. Negative free space with a "safe" mode falls back to Start.
	if freeSpace <= 0.0 && isSafe {
		alignmentMode = style.AlignContentStart
	}

	return alignmentMode
}

// ComputeAlignmentOffset computes the offset for a given item based on the
// alignment mode. It is used for both align-content and justify-content, and
// for both the Flexbox and CSS Grid algorithms.
//
// CSS Grid does not apply gaps as part of alignment, so pass gap=0 for Grid.
//
// isFirst should be true when computing the offset for the first item in the
// container; it should be false for subsequent items (where the gap contributes).
func ComputeAlignmentOffset(freeSpace float32, numItems int, gap float32, alignmentMode style.AlignContent, layoutIsFlexReversed bool, isFirst bool) float32 {
	if isFirst {
		switch alignmentMode {
		case style.AlignContentStart:
			return 0.0
		case style.AlignContentFlexStart:
			if layoutIsFlexReversed {
				return freeSpace
			}
			return 0.0
		case style.AlignContentEnd:
			return freeSpace
		case style.AlignContentFlexEnd:
			if layoutIsFlexReversed {
				return 0.0
			}
			return freeSpace
		case style.AlignContentCenter:
			return freeSpace / 2.0
		case style.AlignContentStretch:
			return 0.0
		case style.AlignContentSpaceBetween:
			return 0.0
		case style.AlignContentSpaceAround:
			if freeSpace >= 0.0 {
				return (freeSpace / float32(numItems)) / 2.0
			}
			return freeSpace / 2.0
		case style.AlignContentSpaceEvenly:
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
	case style.AlignContentStart:
		return gap
	case style.AlignContentFlexStart:
		return gap
	case style.AlignContentEnd:
		return gap
	case style.AlignContentFlexEnd:
		return gap
	case style.AlignContentCenter:
		return gap
	case style.AlignContentStretch:
		return gap
	case style.AlignContentSpaceBetween:
		return gap + clampedFreeSpace/float32(numItems-1)
	case style.AlignContentSpaceAround:
		return gap + clampedFreeSpace/float32(numItems)
	case style.AlignContentSpaceEvenly:
		return gap + clampedFreeSpace/float32(numItems+1)
	default:
		return gap
	}
}
