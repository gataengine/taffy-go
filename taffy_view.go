// Package taffy provides the taffyView adapter that implements all tree interfaces
// and the dispatch logic for routing to flexbox/grid/block/leaf algorithms.
package taffy

import (
	"taffy-go/compute"
	"taffy-go/compute/block"
	"taffy-go/compute/flexbox"
	"taffy-go/compute/grid"
	"taffy-go/geometry"
	"taffy-go/style"
	"taffy-go/tree"
)

// taffyView is an unexported wrapper combining TaffyTree with a measure function.
// It implements all the layout interfaces needed to run layout algorithms.
type taffyView struct {
	taffy           *TaffyTree
	measureFunction MeasureFunction
}

// --- TraversePartialTree / TraverseTree interface on *taffyView ---

func (v *taffyView) ChildIds(parent tree.NodeId) []tree.NodeId {
	return v.taffy.children[parent]
}

func (v *taffyView) ChildCount(parent tree.NodeId) int {
	return len(v.taffy.children[parent])
}

func (v *taffyView) GetChildId(parent tree.NodeId, index int) tree.NodeId {
	return v.taffy.children[parent][index]
}

// --- LayoutPartialTree interface on *taffyView ---

func (v *taffyView) GetCoreContainerStyle(nodeId tree.NodeId) style.CoreStyle {
	return &v.taffy.nodes[nodeId].style
}

func (v *taffyView) ResolveCalcValue(_ uint64, _ float32) float32 {
	return 0.0
}

func (v *taffyView) SetUnroundedLayout(nodeId tree.NodeId, layout *tree.Layout) {
	v.taffy.nodes[nodeId].unroundedLayout = *layout
}

// measureFunc returns a closure that calls the tree's measure function for the given node.
func (v *taffyView) measureFunc(nodeId tree.NodeId) func(geometry.Size[geometry.Option[float32]], geometry.Size[style.AvailableSpace]) geometry.Size[float32] {
	return func(knownDimensions geometry.Size[geometry.Option[float32]], availableSpace geometry.Size[style.AvailableSpace]) geometry.Size[float32] {
		ctx := v.taffy.nodeContexts[nodeId]
		if v.measureFunction != nil {
			return v.measureFunction(knownDimensions, availableSpace, nodeId, &ctx, &v.taffy.nodes[nodeId].style)
		}
		if ctx != nil {
			return ctx(knownDimensions, geometry.SizeOptionNone, availableSpace, tree.SizingModeInherentSize)
		}
		return geometry.SizeZeroF32
	}
}

// ComputeChildLayout dispatches to the correct algorithm based on Display.
func (v *taffyView) ComputeChildLayout(nodeId tree.NodeId, inputs tree.LayoutInput) tree.LayoutOutput {
	node := v.taffy.nodes[nodeId]
	hasChildren := len(v.taffy.children[nodeId]) > 0

	return compute.ComputeCachedLayout(v, nodeId, inputs, func(nodeId tree.NodeId, inputs tree.LayoutInput) tree.LayoutOutput {
		st := &node.style
		display := st.Display

		if inputs.RunMode == tree.RunModePerformHiddenLayout {
			return compute.ComputeHiddenLayout(v, nodeId)
		}

		switch display {
		case style.DisplayNone:
			return compute.ComputeHiddenLayout(v, nodeId)
		case style.DisplayFlex:
			if hasChildren {
				return flexbox.ComputeFlexboxLayout(v, nodeId, inputs)
			}
			return compute.ComputeLeafLayout(inputs, v.GetCoreContainerStyle(nodeId), v.ResolveCalcValue, v.measureFunc(nodeId))
		case style.DisplayGrid:
			if hasChildren {
				return grid.ComputeGridLayout(v, nodeId, inputs)
			}
			return compute.ComputeLeafLayout(inputs, v.GetCoreContainerStyle(nodeId), v.ResolveCalcValue, v.measureFunc(nodeId))
		case style.DisplayBlock:
			if hasChildren {
				return block.ComputeBlockLayout(v, nodeId, inputs, nil)
			}
			return compute.ComputeLeafLayout(inputs, v.GetCoreContainerStyle(nodeId), v.ResolveCalcValue, v.measureFunc(nodeId))
		default:
			return compute.ComputeLeafLayout(inputs, v.GetCoreContainerStyle(nodeId), v.ResolveCalcValue, v.measureFunc(nodeId))
		}
	})
}

// --- CacheTree interface on *taffyView ---

func (v *taffyView) CacheGet(nodeId tree.NodeId, input *tree.LayoutInput) *tree.LayoutOutput {
	return v.taffy.nodes[nodeId].cache.Get(input)
}

func (v *taffyView) CacheStore(nodeId tree.NodeId, input *tree.LayoutInput, output tree.LayoutOutput) {
	v.taffy.nodes[nodeId].cache.Store(input, output)
}

func (v *taffyView) CacheClear(nodeId tree.NodeId) {
	v.taffy.nodes[nodeId].cache.Clear()
}

// --- RoundTree interface on *taffyView ---

func (v *taffyView) GetUnroundedLayout(nodeId tree.NodeId) *tree.Layout {
	return &v.taffy.nodes[nodeId].unroundedLayout
}

func (v *taffyView) SetFinalLayout(nodeId tree.NodeId, layout *tree.Layout) {
	v.taffy.nodes[nodeId].finalLayout = *layout
}

// --- PrintTree interface on *taffyView ---

func (v *taffyView) GetDebugLabel(nodeId tree.NodeId) string {
	node := v.taffy.nodes[nodeId]
	numChildren := v.ChildCount(nodeId)
	switch {
	case node.style.Display == style.DisplayNone:
		return "NONE"
	case numChildren == 0:
		return "LEAF"
	case node.style.Display == style.DisplayBlock:
		return "BLOCK"
	case node.style.Display == style.DisplayFlex:
		if node.style.FlexDirection.IsRow() {
			return "FLEX ROW"
		}
		return "FLEX COL"
	case node.style.Display == style.DisplayGrid:
		return "GRID"
	default:
		return "NODE"
	}
}

func (v *taffyView) GetFinalLayout(nodeId tree.NodeId) *tree.Layout {
	return &v.taffy.nodes[nodeId].finalLayout
}

// --- LayoutFlexboxContainer interface on *taffyView ---

func (v *taffyView) GetFlexboxContainerStyle(nodeId tree.NodeId) style.FlexboxContainerStyle {
	return &v.taffy.nodes[nodeId].style
}

func (v *taffyView) GetFlexboxChildStyle(childNodeId tree.NodeId) style.FlexboxItemStyle {
	return &v.taffy.nodes[childNodeId].style
}

// --- LayoutGridContainer interface on *taffyView ---

func (v *taffyView) GetGridContainerStyle(nodeId tree.NodeId) style.GridContainerStyle {
	return &v.taffy.nodes[nodeId].style
}

func (v *taffyView) GetGridChildStyle(childNodeId tree.NodeId) style.GridItemStyle {
	return &v.taffy.nodes[childNodeId].style
}

// --- LayoutBlockContainer interface on *taffyView ---

func (v *taffyView) GetBlockContainerStyle(nodeId tree.NodeId) style.BlockContainerStyle {
	return &v.taffy.nodes[nodeId].style
}

func (v *taffyView) GetBlockChildStyle(childNodeId tree.NodeId) style.BlockItemStyle {
	return &v.taffy.nodes[childNodeId].style
}

// ComputeBlockChildLayout dispatches to the correct algorithm for block children,
// passing through the block context for children in the same BFC.
func (v *taffyView) ComputeBlockChildLayout(nodeId tree.NodeId, inputs tree.LayoutInput, blockCtx any) tree.LayoutOutput {
	node := v.taffy.nodes[nodeId]
	hasChildren := len(v.taffy.children[nodeId]) > 0

	return compute.ComputeCachedLayout(v, nodeId, inputs, func(nodeId tree.NodeId, inputs tree.LayoutInput) tree.LayoutOutput {
		st := &node.style
		display := st.Display

		if inputs.RunMode == tree.RunModePerformHiddenLayout {
			return compute.ComputeHiddenLayout(v, nodeId)
		}

		// For block children in the same BFC, pass the block context through
		if display == style.DisplayBlock && hasChildren {
			return block.ComputeBlockLayout(v, nodeId, inputs, blockCtx)
		}

		// For non-block children, fall through to normal dispatch
		return v.ComputeChildLayout(nodeId, inputs)
	})
}

// Compile-time interface checks
var _ tree.TraversePartialTree = (*taffyView)(nil)
var _ tree.TraverseTree = (*taffyView)(nil)
var _ tree.LayoutPartialTree = (*taffyView)(nil)
var _ tree.CacheTree = (*taffyView)(nil)
var _ tree.RoundTree = (*taffyView)(nil)
var _ tree.PrintTree = (*taffyView)(nil)
var _ tree.LayoutFlexboxContainer = (*taffyView)(nil)
var _ tree.LayoutGridContainer = (*taffyView)(nil)
var _ tree.LayoutBlockContainer = (*taffyView)(nil)

// ---------------------------------------------------------------------------
// computeRootLayout
// ---------------------------------------------------------------------------

// computeRootLayout computes the layout for the root node of the tree.
//
// It dispatches the appropriate sizing logic based on the root node's display mode
// (Block nodes get special width-stretching treatment), then stores the resulting
// unrounded Layout on the root node.
func computeRootLayout(t *taffyView, root tree.NodeId, availableSpace geometry.Size[style.AvailableSpace]) {
	knownDimensions := geometry.SizeOptionNone

	st := t.GetCoreContainerStyle(root)
	calc := t.ResolveCalcValue

	if st.GetIsBlock() {
		parentSize := style.SizeAvailableSpaceIntoOptions(availableSpace)

		aspectRatio := st.GetAspectRatio()
		margin := style.ResolveRectLPAWidth(st.GetMargin(), parentSize.Width, calc)
		padding := style.ResolveRectLPWidth(st.GetPadding(), parentSize.Width, calc)
		border := style.ResolveRectLPWidth(st.GetBorder(), parentSize.Width, calc)
		paddingBorderSize := geometry.RectSumAxes(geometry.RectAddF32(padding, border))

		boxSizingAdjustment := geometry.SizeZeroF32
		if st.GetBoxSizing() == style.BoxSizingContentBox {
			boxSizingAdjustment = paddingBorderSize
		}

		minSize := geometry.SizeOptionMaybeApplyAspectRatio(
			geometry.SizeMaybeAdd(
				style.MaybeResolveSizeDimOption(st.GetMinSize(), parentSize, calc),
				geometry.Size[geometry.Option[float32]]{
					Width:  geometry.Some(boxSizingAdjustment.Width),
					Height: geometry.Some(boxSizingAdjustment.Height),
				},
			),
			aspectRatio,
		)
		maxSize := geometry.SizeMaybeAdd(
			style.MaybeResolveSizeDimOption(st.GetMaxSize(), parentSize, calc),
			geometry.Size[geometry.Option[float32]]{
				Width:  geometry.Some(boxSizingAdjustment.Width),
				Height: geometry.Some(boxSizingAdjustment.Height),
			},
		)
		clampedStyleSize := geometry.SizeMaybeClamp(
			geometry.SizeOptionMaybeApplyAspectRatio(
				geometry.SizeMaybeAdd(
					style.MaybeResolveSizeDimOption(st.GetSize(), parentSize, calc),
					geometry.Size[geometry.Option[float32]]{
						Width:  geometry.Some(boxSizingAdjustment.Width),
						Height: geometry.Some(boxSizingAdjustment.Height),
					},
				),
				aspectRatio,
			),
			minSize,
			maxSize,
		)

		// If both min and max in a given axis are set and max <= min then this determines the size.
		minMaxDefiniteSize := geometry.SizeZipMap(minSize, maxSize, func(min, max geometry.Option[float32]) geometry.Option[float32] {
			if min.IsSome && max.IsSome && max.Value <= min.Value {
				return min
			}
			return geometry.None[float32]()
		})

		// Block nodes automatically stretch to fill available width if definite.
		marginHSum := geometry.RectHorizontalAxisSum(margin)
		availableSpaceBasedSize := geometry.Size[geometry.Option[float32]]{
			Width:  geometry.MaybeSubOption(availableSpace.Width.IntoOption(), geometry.Some(marginHSum)),
			Height: geometry.None[float32](),
		}

		styledBasedKnownDimensions := geometry.SizeMaybeMax(
			geometry.SizeOptionOr(
				geometry.SizeOptionOr(
					geometry.SizeOptionOr(knownDimensions, minMaxDefiniteSize),
					clampedStyleSize,
				),
				availableSpaceBasedSize,
			),
			geometry.Size[geometry.Option[float32]]{
				Width:  geometry.Some(paddingBorderSize.Width),
				Height: geometry.Some(paddingBorderSize.Height),
			},
		)

		knownDimensions = styledBasedKnownDimensions
	}

	// Recursively compute node layout.
	output := t.ComputeChildLayout(root, tree.LayoutInput{
		KnownDimensions:               knownDimensions,
		ParentSize:                    style.SizeAvailableSpaceIntoOptions(availableSpace),
		AvailableSpace:                availableSpace,
		SizingMode:                    tree.SizingModeInherentSize,
		Axis:                          tree.RequestedAxisBoth,
		RunMode:                       tree.RunModePerformLayout,
		VerticalMarginsAreCollapsible: geometry.LineFalseBool,
	})

	// Re-fetch style (borrow is released after the block above).
	st = t.GetCoreContainerStyle(root)
	calc = t.ResolveCalcValue
	widthContext := availableSpace.Width.IntoOption()

	padding := style.ResolveRectLPWidth(st.GetPadding(), widthContext, calc)
	border := style.ResolveRectLPWidth(st.GetBorder(), widthContext, calc)
	margin := style.ResolveRectLPAWidth(st.GetMargin(), widthContext, calc)

	scrollbarSize := geometry.Size[float32]{
		Width: func() float32 {
			if st.GetOverflow().Y == style.OverflowScroll {
				return st.GetScrollbarWidth()
			}
			return 0.0
		}(),
		Height: func() float32 {
			if st.GetOverflow().X == style.OverflowScroll {
				return st.GetScrollbarWidth()
			}
			return 0.0
		}(),
	}

	locationX := float32(0.0)
	if st.GetDirection().IsRTL() {
		if opt := availableSpace.Width.IntoOption(); opt.IsSome {
			locationX = opt.Value - output.Size.Width
		}
	}

	t.SetUnroundedLayout(root, &tree.Layout{
		Order:         0,
		Location:      geometry.Point[float32]{X: locationX, Y: 0.0},
		Size:          output.Size,
		ContentSize:   output.ContentSize,
		ScrollbarSize: scrollbarSize,
		Padding:       padding,
		Border:        border,
		Margin:        margin,
	})
}
