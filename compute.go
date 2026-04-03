// Package taffy provides the core layout computation entry points.
// This file is a port of compute/mod.rs from the Taffy Rust layout engine.
package taffy

// layoutAndCacheTree is a combined interface required by computeHiddenLayout.
// It extends both layoutPartialTree (to drive child layout) and cacheTree (to
// clear the cache for hidden nodes).
type layoutAndCacheTree interface {
	layoutPartialTree
	cacheTree
}

// computeRootLayout computes the layout for the root node of the tree.
//
// It dispatches the appropriate sizing logic based on the root node's display mode
// (Block nodes get special width-stretching treatment), then stores the resulting
// unrounded Layout on the root node.
func computeRootLayout(tree layoutPartialTree, root NodeId, availableSpace Size[AvailableSpace]) {
	knownDimensions := SizeOptionNone

	style := tree.getCoreContainerStyle(root)
	calc := tree.resolveCalcValue

	if style.isBlock() {
		parentSize := SizeAvailableSpaceIntoOptions(availableSpace)

		aspectRatio := style.aspectRatio()
		margin := resolveRectLPAWidth(style.margin(), parentSize.Width, calc)
		padding := resolveRectLPWidth(style.padding(), parentSize.Width, calc)
		border := resolveRectLPWidth(style.border(), parentSize.Width, calc)
		paddingBorderSize := RectSumAxes(RectAddF32(padding, border))

		boxSizingAdjustment := SizeZeroF32
		if style.boxSizing() == BoxSizingContentBox {
			boxSizingAdjustment = paddingBorderSize
		}

		minSize := SizeOptionMaybeApplyAspectRatio(
			SizeMaybeAdd(
				MaybeResolveSizeDimOption(style.minSize(), parentSize, calc),
				Size[Option[float32]]{
					Width:  Some(boxSizingAdjustment.Width),
					Height: Some(boxSizingAdjustment.Height),
				},
			),
			aspectRatio,
		)
		maxSize := SizeMaybeAdd(
			MaybeResolveSizeDimOption(style.maxSize(), parentSize, calc),
			Size[Option[float32]]{
				Width:  Some(boxSizingAdjustment.Width),
				Height: Some(boxSizingAdjustment.Height),
			},
		)
		clampedStyleSize := SizeMaybeClamp(
			SizeOptionMaybeApplyAspectRatio(
				SizeMaybeAdd(
					MaybeResolveSizeDimOption(style.size(), parentSize, calc),
					Size[Option[float32]]{
						Width:  Some(boxSizingAdjustment.Width),
						Height: Some(boxSizingAdjustment.Height),
					},
				),
				aspectRatio,
			),
			minSize,
			maxSize,
		)

		// If both min and max in a given axis are set and max <= min then this determines the size.
		minMaxDefiniteSize := SizeZipMap(minSize, maxSize, func(min, max Option[float32]) Option[float32] {
			if min.IsSome && max.IsSome && max.Value <= min.Value {
				return min
			}
			return None[float32]()
		})

		// Block nodes automatically stretch to fill available width if definite.
		marginHSum := RectHorizontalAxisSum(margin)
		availableSpaceBasedSize := Size[Option[float32]]{
			Width:  MaybeSubOption(availableSpace.Width.IntoOption(), Some(marginHSum)),
			Height: None[float32](),
		}

		styledBasedKnownDimensions := SizeMaybeMax(
			SizeOptionOr(
				SizeOptionOr(
					SizeOptionOr(knownDimensions, minMaxDefiniteSize),
					clampedStyleSize,
				),
				availableSpaceBasedSize,
			),
			Size[Option[float32]]{
				Width:  Some(paddingBorderSize.Width),
				Height: Some(paddingBorderSize.Height),
			},
		)

		knownDimensions = styledBasedKnownDimensions
	}

	// Recursively compute node layout.
	output := tree.computeChildLayout(root, LayoutInput{
		KnownDimensions:               knownDimensions,
		ParentSize:                    SizeAvailableSpaceIntoOptions(availableSpace),
		AvailableSpace:                availableSpace,
		SizingMode:                    SizingModeInherentSize,
		Axis:                          RequestedAxisBoth,
		RunMode:                       RunModePerformLayout,
		VerticalMarginsAreCollapsible: LineFalseBool,
	})

	// Re-fetch style (borrow is released after the block above).
	style = tree.getCoreContainerStyle(root)
	calc = tree.resolveCalcValue
	widthContext := availableSpace.Width.IntoOption()

	padding := resolveRectLPWidth(style.padding(), widthContext, calc)
	border := resolveRectLPWidth(style.border(), widthContext, calc)
	margin := resolveRectLPAWidth(style.margin(), widthContext, calc)

	scrollbarSize := Size[float32]{
		Width: func() float32 {
			if style.overflow().Y == OverflowScroll {
				return style.scrollbarWidth()
			}
			return 0.0
		}(),
		Height: func() float32 {
			if style.overflow().X == OverflowScroll {
				return style.scrollbarWidth()
			}
			return 0.0
		}(),
	}

	locationX := float32(0.0)
	if style.direction().IsRTL() {
		if opt := availableSpace.Width.IntoOption(); opt.IsSome {
			locationX = opt.Value - output.Size.Width
		}
	}

	tree.setUnroundedLayout(root, &Layout{
		Order:         0,
		Location:      Point[float32]{X: locationX, Y: 0.0},
		Size:          output.Size,
		ContentSize:   output.ContentSize,
		ScrollbarSize: scrollbarSize,
		Padding:       padding,
		Border:        border,
		Margin:        margin,
	})
}

// computeCachedLayout wraps a layout computation with cache lookup and store.
//
// It first checks whether a cached result exists for the given node and inputs.
// If a cache hit is found the cached value is returned immediately.
// Otherwise computeUncached is called, the result is stored in the cache, and
// then returned.
func computeCachedLayout(
	tree cacheTree,
	node NodeId,
	inputs LayoutInput,
	computeUncached func(NodeId, LayoutInput) LayoutOutput,
) LayoutOutput {
	// Check for a cached result.
	if cached := tree.cacheGet(node, &inputs); cached != nil {
		return *cached
	}

	// Compute the layout and store it in the cache.
	result := computeUncached(node, inputs)
	tree.cacheStore(node, &inputs, result)
	return result
}

// roundLayout rounds all layout values for the given node and its entire subtree
// to exact pixel values, writing the results to the final layout.
//
// Rounding is done based on cumulative viewport coordinates (rather than
// parent-relative coordinates) to avoid introducing gaps between adjacent elements.
// Widths/heights are computed by rounding the outer edges first and then
// taking the difference, rather than rounding the dimension directly.
//
// See https://github.com/facebook/yoga/commit/aa5b296ac78f7a22e1aeaf4891243c6bb76488e2
// for more context.
func roundLayout(tree roundTree, nodeId NodeId) {
	roundLayoutInner(tree, nodeId, 0.0, 0.0)
}

// roundLayoutInner is the recursive helper for roundLayout.
func roundLayoutInner(tree roundTree, nodeId NodeId, cumulativeX, cumulativeY float32) {
	unrounded := tree.getUnroundedLayout(nodeId)
	layout := *unrounded // copy

	cumulativeX = cumulativeX + unrounded.Location.X
	cumulativeY = cumulativeY + unrounded.Location.Y

	layout.Location.X = roundF32(unrounded.Location.X)
	layout.Location.Y = roundF32(unrounded.Location.Y)
	layout.Size.Width = roundF32(cumulativeX+unrounded.Size.Width) - roundF32(cumulativeX)
	layout.Size.Height = roundF32(cumulativeY+unrounded.Size.Height) - roundF32(cumulativeY)

	layout.ScrollbarSize.Width = roundF32(unrounded.ScrollbarSize.Width)
	layout.ScrollbarSize.Height = roundF32(unrounded.ScrollbarSize.Height)

	layout.Border.Left = roundF32(cumulativeX+unrounded.Border.Left) - roundF32(cumulativeX)
	layout.Border.Right = roundF32(cumulativeX+unrounded.Size.Width) -
		roundF32(cumulativeX+unrounded.Size.Width-unrounded.Border.Right)
	layout.Border.Top = roundF32(cumulativeY+unrounded.Border.Top) - roundF32(cumulativeY)
	layout.Border.Bottom = roundF32(cumulativeY+unrounded.Size.Height) -
		roundF32(cumulativeY+unrounded.Size.Height-unrounded.Border.Bottom)

	layout.Padding.Left = roundF32(cumulativeX+unrounded.Padding.Left) - roundF32(cumulativeX)
	layout.Padding.Right = roundF32(cumulativeX+unrounded.Size.Width) -
		roundF32(cumulativeX+unrounded.Size.Width-unrounded.Padding.Right)
	layout.Padding.Top = roundF32(cumulativeY+unrounded.Padding.Top) - roundF32(cumulativeY)
	layout.Padding.Bottom = roundF32(cumulativeY+unrounded.Size.Height) -
		roundF32(cumulativeY+unrounded.Size.Height-unrounded.Padding.Bottom)

	roundContentSize(&layout, unrounded.ContentSize, cumulativeX, cumulativeY)

	tree.setFinalLayout(nodeId, &layout)

	childCount := tree.childCount(nodeId)
	for i := 0; i < childCount; i++ {
		child := tree.getChildId(nodeId, i)
		roundLayoutInner(tree, child, cumulativeX, cumulativeY)
	}
}

// roundContentSize rounds the content size fields of the layout.
func roundContentSize(layout *Layout, unroundedContentSize Size[float32], cumulativeX, cumulativeY float32) {
	layout.ContentSize.Width = roundF32(cumulativeX+unroundedContentSize.Width) - roundF32(cumulativeX)
	layout.ContentSize.Height = roundF32(cumulativeY+unroundedContentSize.Height) - roundF32(cumulativeY)
}

// computeHiddenLayout computes a zeroed-out layout for a Display::None node and
// all of its descendants, and clears their caches.
func computeHiddenLayout(tree layoutAndCacheTree, node NodeId) LayoutOutput {
	// Clear cache and set a zeroed layout for this node.
	tree.cacheClear(node)
	tree.setUnroundedLayout(node, &Layout{Order: 0})

	// Recursively apply hidden layout to all children.
	childCount := tree.childCount(node)
	for i := 0; i < childCount; i++ {
		childId := tree.getChildId(node, i)
		tree.computeChildLayout(childId, hiddenLayoutInput)
	}

	return hiddenLayoutOutput
}
