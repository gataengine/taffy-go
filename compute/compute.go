// Package compute provides the core layout computation entry points.
// This file is a port of compute/mod.rs from the Taffy Rust layout engine.
package compute

import (
	"github.com/gataengine/taffy-go/geometry"
	"github.com/gataengine/taffy-go/tree"
)

// LayoutAndCacheTree is a combined interface required by ComputeHiddenLayout.
// It extends both LayoutPartialTree (to drive child layout) and CacheTree (to
// clear the cache for hidden nodes).
type LayoutAndCacheTree interface {
	tree.LayoutPartialTree
	tree.CacheTree
}

// ComputeCachedLayout wraps a layout computation with cache lookup and store.
//
// It first checks whether a cached result exists for the given node and inputs.
// If a cache hit is found the cached value is returned immediately.
// Otherwise computeUncached is called, the result is stored in the cache, and
// then returned.
func ComputeCachedLayout(
	t tree.CacheTree,
	node tree.NodeId,
	inputs tree.LayoutInput,
	computeUncached func(tree.NodeId, tree.LayoutInput) tree.LayoutOutput,
) tree.LayoutOutput {
	// Check for a cached result.
	if cached, ok := t.CacheGet(node, &inputs); ok {
		return cached
	}

	// Compute the layout and store it in the cache.
	result := computeUncached(node, inputs)
	t.CacheStore(node, &inputs, result)
	return result
}

// RoundLayout rounds all layout values for the given node and its entire subtree
// to exact pixel values, writing the results to the final layout.
//
// Rounding is done based on cumulative viewport coordinates (rather than
// parent-relative coordinates) to avoid introducing gaps between adjacent elements.
// Widths/heights are computed by rounding the outer edges first and then
// taking the difference, rather than rounding the dimension directly.
//
// See https://github.com/facebook/yoga/commit/aa5b296ac78f7a22e1aeaf4891243c6bb76488e2
// for more context.
func RoundLayout(t tree.RoundTree, nodeId tree.NodeId) {
	roundLayoutInner(t, nodeId, 0.0, 0.0)
}

// roundLayoutInner is the recursive helper for RoundLayout.
func roundLayoutInner(t tree.RoundTree, nodeId tree.NodeId, cumulativeX, cumulativeY float32) {
	unrounded := t.GetUnroundedLayout(nodeId)
	layout := *unrounded // copy

	cumulativeX = cumulativeX + unrounded.Location.X
	cumulativeY = cumulativeY + unrounded.Location.Y

	layout.Location.X = geometry.RoundF32(unrounded.Location.X)
	layout.Location.Y = geometry.RoundF32(unrounded.Location.Y)
	layout.Size.Width = geometry.RoundF32(cumulativeX+unrounded.Size.Width) - geometry.RoundF32(cumulativeX)
	layout.Size.Height = geometry.RoundF32(cumulativeY+unrounded.Size.Height) - geometry.RoundF32(cumulativeY)

	layout.ScrollbarSize.Width = geometry.RoundF32(unrounded.ScrollbarSize.Width)
	layout.ScrollbarSize.Height = geometry.RoundF32(unrounded.ScrollbarSize.Height)

	layout.Border.Left = geometry.RoundF32(cumulativeX+unrounded.Border.Left) - geometry.RoundF32(cumulativeX)
	layout.Border.Right = geometry.RoundF32(cumulativeX+unrounded.Size.Width) -
		geometry.RoundF32(cumulativeX+unrounded.Size.Width-unrounded.Border.Right)
	layout.Border.Top = geometry.RoundF32(cumulativeY+unrounded.Border.Top) - geometry.RoundF32(cumulativeY)
	layout.Border.Bottom = geometry.RoundF32(cumulativeY+unrounded.Size.Height) -
		geometry.RoundF32(cumulativeY+unrounded.Size.Height-unrounded.Border.Bottom)

	layout.Padding.Left = geometry.RoundF32(cumulativeX+unrounded.Padding.Left) - geometry.RoundF32(cumulativeX)
	layout.Padding.Right = geometry.RoundF32(cumulativeX+unrounded.Size.Width) -
		geometry.RoundF32(cumulativeX+unrounded.Size.Width-unrounded.Padding.Right)
	layout.Padding.Top = geometry.RoundF32(cumulativeY+unrounded.Padding.Top) - geometry.RoundF32(cumulativeY)
	layout.Padding.Bottom = geometry.RoundF32(cumulativeY+unrounded.Size.Height) -
		geometry.RoundF32(cumulativeY+unrounded.Size.Height-unrounded.Padding.Bottom)

	roundContentSize(&layout, unrounded.ContentSize, cumulativeX, cumulativeY)

	t.SetFinalLayout(nodeId, layout)

	childCount := t.ChildCount(nodeId)
	for i := 0; i < childCount; i++ {
		child := t.GetChildId(nodeId, i)
		roundLayoutInner(t, child, cumulativeX, cumulativeY)
	}
}

// roundContentSize rounds the content size fields of the layout.
func roundContentSize(layout *tree.Layout, unroundedContentSize geometry.Size[float32], cumulativeX, cumulativeY float32) {
	layout.ContentSize.Width = geometry.RoundF32(cumulativeX+unroundedContentSize.Width) - geometry.RoundF32(cumulativeX)
	layout.ContentSize.Height = geometry.RoundF32(cumulativeY+unroundedContentSize.Height) - geometry.RoundF32(cumulativeY)
}

// ComputeHiddenLayout computes a zeroed-out layout for a Display::None node and
// all of its descendants, and clears their caches.
func ComputeHiddenLayout(t LayoutAndCacheTree, node tree.NodeId) tree.LayoutOutput {
	// Clear cache and set a zeroed layout for this node.
	t.CacheClear(node)
	t.SetUnroundedLayout(node, tree.Layout{Order: 0})

	// Recursively apply hidden layout to all children.
	childCount := t.ChildCount(node)
	for i := 0; i < childCount; i++ {
		childId := t.GetChildId(node, i)
		t.ComputeChildLayout(childId, tree.HiddenLayoutInput)
	}

	return tree.HiddenLayoutOutput
}
