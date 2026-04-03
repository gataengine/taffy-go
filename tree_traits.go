// Package taffy provides the abstractions that make up the core of Taffy's low-level API.
// This file is a port of tree/traits.rs from the Taffy Rust layout engine.
package taffy

// traversePartialTree is Taffy's abstraction for downward tree traversal.
// It does not require access to any nodes other than a single container node's
// immediate children.
type traversePartialTree interface {
	childIds(parent NodeId) []NodeId
	childCount(parent NodeId) int
	getChildId(parent NodeId, childIndex int) NodeId
}

// traverseTree is a marker interface which extends traversePartialTree.
// Implementing this interface implies the additional guarantee that the child
// methods can be used to recurse infinitely down the tree. It is required by
// the roundTree and printTree interfaces.
type traverseTree interface {
	traversePartialTree
}

// layoutPartialTree is the interface any type must implement to be laid out
// using Taffy's algorithms.
//
// Note that this interface extends traversePartialTree (not traverseTree).
// Taffy's algorithm implementations have been designed such that they can be
// used for laying out a single node that only has access to its immediate children.
type layoutPartialTree interface {
	traversePartialTree
	// getCoreContainerStyle returns a reference to the Style for this node.
	getCoreContainerStyle(nodeId NodeId) *Style
	// resolveCalcValue resolves a calc value given a basis.
	resolveCalcValue(calcId uint64, basis float32) float32
	// setUnroundedLayout sets the node's unrounded layout.
	setUnroundedLayout(nodeId NodeId, layout *Layout)
	// computeChildLayout computes the specified node's size or full layout given constraints.
	computeChildLayout(nodeId NodeId, inputs LayoutInput) LayoutOutput
}

// cacheTree is the interface used by the computeCachedLayout function to allow
// cached layout results to be stored and retrieved.
type cacheTree interface {
	cacheGet(nodeId NodeId, input *LayoutInput) *LayoutOutput
	cacheStore(nodeId NodeId, input *LayoutInput, output LayoutOutput)
	cacheClear(nodeId NodeId)
}

// roundTree is the interface used by the roundLayout function which takes a tree
// of unrounded float-valued layouts and performs rounding to snap values to the pixel grid.
//
// As indicated by its dependence on traverseTree, it requires full recursive access to the tree.
type roundTree interface {
	traverseTree
	getUnroundedLayout(nodeId NodeId) *Layout
	setFinalLayout(nodeId NodeId, layout *Layout)
}

// printTree is the interface used by the printTree function which prints a debug representation.
//
// As indicated by its dependence on traverseTree, it requires full recursive access to the tree.
type printTree interface {
	traverseTree
	getDebugLabel(nodeId NodeId) string
	getFinalLayout(nodeId NodeId) *Layout
}

// layoutFlexboxContainer extends layoutPartialTree with getters for the styles
// required for Flexbox layout.
type layoutFlexboxContainer interface {
	layoutPartialTree
	getFlexboxContainerStyle(nodeId NodeId) *Style
	getFlexboxChildStyle(childNodeId NodeId) *Style
}

// layoutGridContainer extends layoutPartialTree with getters for the styles
// required for CSS Grid layout.
type layoutGridContainer interface {
	layoutPartialTree
	getGridContainerStyle(nodeId NodeId) *Style
	getGridChildStyle(childNodeId NodeId) *Style
}

// blockContext is a placeholder type for the block layout context.
// It will be fully defined in compute_block.go.
type blockContext struct{}

// layoutBlockContainer extends layoutPartialTree with getters for the styles
// required for CSS Block layout.
type layoutBlockContainer interface {
	layoutPartialTree
	getBlockContainerStyle(nodeId NodeId) *Style
	getBlockChildStyle(childNodeId NodeId) *Style
	computeBlockChildLayout(nodeId NodeId, inputs LayoutInput, blockCtx *blockContext) LayoutOutput
}

// measureChildSize computes the size of a child node along the given axis
// given the specified constraints.
func measureChildSize(
	tree layoutPartialTree,
	nodeId NodeId,
	knownDimensions Size[Option[float32]],
	parentSize Size[Option[float32]],
	availableSpace Size[AvailableSpace],
	sizingMode SizingMode,
	axis AbsoluteAxis,
	verticalMarginsAreCollapsible Line[bool],
) float32 {
	output := tree.computeChildLayout(nodeId, LayoutInput{
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

// performChildLayout performs a full layout on a child node given the specified constraints.
func performChildLayout(
	tree layoutPartialTree,
	nodeId NodeId,
	knownDimensions Size[Option[float32]],
	parentSize Size[Option[float32]],
	availableSpace Size[AvailableSpace],
	sizingMode SizingMode,
	verticalMarginsAreCollapsible Line[bool],
) LayoutOutput {
	return tree.computeChildLayout(nodeId, LayoutInput{
		KnownDimensions:               knownDimensions,
		ParentSize:                    parentSize,
		AvailableSpace:                availableSpace,
		SizingMode:                    sizingMode,
		Axis:                          RequestedAxisBoth,
		RunMode:                       RunModePerformLayout,
		VerticalMarginsAreCollapsible: verticalMarginsAreCollapsible,
	})
}
