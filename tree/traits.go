// Package tree provides the abstractions that make up the core of Taffy's low-level API.
// This file is a port of tree/traits.rs from the Taffy Rust layout engine.
package tree

import (
	"github.com/gataengine/taffy-go/geometry"
	"github.com/gataengine/taffy-go/lib"
	"github.com/gataengine/taffy-go/style"
)

// TraversePartialTree is Taffy's abstraction for downward tree traversal.
// It does not require access to any nodes other than a single container node's
// immediate children.
type TraversePartialTree interface {
	ChildIds(parent NodeId) []NodeId
	ChildCount(parent NodeId) int
	GetChildId(parent NodeId, childIndex int) NodeId
}

// TraverseTree is a marker interface which extends TraversePartialTree.
// Implementing this interface implies the additional guarantee that the child
// methods can be used to recurse infinitely down the tree. It is required by
// the RoundTree and PrintTree interfaces.
type TraverseTree interface {
	TraversePartialTree
}

// LayoutPartialTree is the interface any type must implement to be laid out
// using Taffy's algorithms.
//
// Note that this interface extends TraversePartialTree (not TraverseTree).
// Taffy's algorithm implementations have been designed such that they can be
// used for laying out a single node that only has access to its immediate children.
type LayoutPartialTree interface {
	TraversePartialTree
	// GetCoreContainerStyle returns the core style for this node.
	GetCoreContainerStyle(nodeId NodeId) style.CoreStyle
	// ResolveCalcValue resolves a calc value given a basis.
	ResolveCalcValue(calcId uint64, basis float32) float32
	// SetUnroundedLayout sets the node's unrounded layout.
	SetUnroundedLayout(nodeId NodeId, layout Layout)
	// ComputeChildLayout computes the specified node's size or full layout given constraints.
	ComputeChildLayout(nodeId NodeId, inputs LayoutInput) LayoutOutput
}

// CacheTree is the interface used by the computeCachedLayout function to allow
// cached layout results to be stored and retrieved.
type CacheTree interface {
	CacheGet(nodeId NodeId, input *LayoutInput) (LayoutOutput, bool)
	CacheStore(nodeId NodeId, input *LayoutInput, output LayoutOutput)
	CacheClear(nodeId NodeId)
}

// RoundTree is the interface used by the roundLayout function which takes a tree
// of unrounded float-valued layouts and performs rounding to snap values to the pixel grid.
//
// As indicated by its dependence on TraverseTree, it requires full recursive access to the tree.
type RoundTree interface {
	TraverseTree
	GetUnroundedLayout(nodeId NodeId) *Layout
	SetFinalLayout(nodeId NodeId, layout Layout)
}

// PrintTree is the interface used by the printTree function which prints a debug representation.
//
// As indicated by its dependence on TraverseTree, it requires full recursive access to the tree.
type PrintTree interface {
	TraverseTree
	GetDebugLabel(nodeId NodeId) string
	GetFinalLayout(nodeId NodeId) *Layout
}

// LayoutFlexboxContainer extends LayoutPartialTree with getters for the styles
// required for Flexbox layout.
type LayoutFlexboxContainer interface {
	LayoutPartialTree
	GetFlexboxContainerStyle(nodeId NodeId) style.FlexboxContainerStyle
	GetFlexboxChildStyle(childNodeId NodeId) style.FlexboxItemStyle
}

// LayoutGridContainer extends LayoutPartialTree with getters for the styles
// required for CSS Grid layout.
type LayoutGridContainer interface {
	LayoutPartialTree
	GetGridContainerStyle(nodeId NodeId) style.GridContainerStyle
	GetGridChildStyle(childNodeId NodeId) style.GridItemStyle
}

// LayoutBlockContainer extends LayoutPartialTree with getters for the styles
// required for CSS Block layout.
type LayoutBlockContainer interface {
	LayoutPartialTree
	GetBlockContainerStyle(nodeId NodeId) style.BlockContainerStyle
	GetBlockChildStyle(childNodeId NodeId) style.BlockItemStyle
	ComputeBlockChildLayout(nodeId NodeId, inputs LayoutInput, blockCtx any) LayoutOutput
}

// MeasureChildSize computes the size of a child node along the given axis
// given the specified constraints.
func MeasureChildSize(
	tree LayoutPartialTree,
	nodeId NodeId,
	knownDimensions geometry.Size[lib.Option[float32]],
	parentSize geometry.Size[lib.Option[float32]],
	availableSpace geometry.Size[style.AvailableSpace],
	sizingMode SizingMode,
	axis geometry.AbsoluteAxis,
	verticalMarginsAreCollapsible geometry.Line[bool],
) float32 {
	output := tree.ComputeChildLayout(nodeId, LayoutInput{
		KnownDimensions:               knownDimensions,
		ParentSize:                    parentSize,
		AvailableSpace:                availableSpace,
		SizingMode:                    sizingMode,
		Axis:                          RequestedAxisFromAbsolute(axis),
		RunMode:                       RunModeComputeSize,
		VerticalMarginsAreCollapsible: verticalMarginsAreCollapsible,
	})
	return output.Size.GetAbs(axis)
}

// PerformChildLayout performs a full layout on a child node given the specified constraints.
func PerformChildLayout(
	tree LayoutPartialTree,
	nodeId NodeId,
	knownDimensions geometry.Size[lib.Option[float32]],
	parentSize geometry.Size[lib.Option[float32]],
	availableSpace geometry.Size[style.AvailableSpace],
	sizingMode SizingMode,
	verticalMarginsAreCollapsible geometry.Line[bool],
) LayoutOutput {
	return tree.ComputeChildLayout(nodeId, LayoutInput{
		KnownDimensions:               knownDimensions,
		ParentSize:                    parentSize,
		AvailableSpace:                availableSpace,
		SizingMode:                    sizingMode,
		Axis:                          RequestedAxisBoth,
		RunMode:                       RunModePerformLayout,
		VerticalMarginsAreCollapsible: verticalMarginsAreCollapsible,
	})
}
