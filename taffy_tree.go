// Package taffy provides TaffyTree, the default implementation of the layout tree.
// This file is a port of tree/taffy_tree.rs from the Taffy Rust layout engine.
package taffy

import "fmt"

// nodeData stores style + layout data for a single node.
type nodeData struct {
	style           Style
	unroundedLayout Layout
	finalLayout     Layout
	cache           cache
	hasNodeContext   bool
}

// newNodeData creates node data with the given style and an empty cache.
func newNodeData(style Style) *nodeData {
	return &nodeData{
		style:           style,
		cache:           newCache(),
		unroundedLayout: NewLayout(),
		finalLayout:     NewLayout(),
		hasNodeContext:   false,
	}
}

// markDirty clears the node's cache and reports whether it was already empty.
func (nd *nodeData) markDirty() ClearState {
	return nd.cache.clear()
}

// NodeContext is a user-provided measure function for leaf nodes.
// It computes the intrinsic size of the node given constraints.
type NodeContext = func(
	knownDimensions Size[Option[float32]],
	parentSize Size[Option[float32]],
	availableSpace Size[AvailableSpace],
	sizingMode SizingMode,
) Size[float32]

// taffyConfig holds tree-wide configuration.
type taffyConfig struct {
	useRounding bool
}

// defaultTaffyConfig returns the default configuration (rounding enabled).
func defaultTaffyConfig() taffyConfig {
	return taffyConfig{useRounding: true}
}

// TaffyTree is an entire tree of UI nodes and the entry point to Taffy's high-level API.
// It allows building a tree of UI nodes, running layout algorithms, and accessing results.
type TaffyTree struct {
	nodes        map[NodeId]*nodeData
	nodeContexts map[NodeId]NodeContext
	children     map[NodeId][]NodeId
	parents      map[NodeId]*NodeId
	config       taffyConfig
	nextId       uint64
}

// New creates a new TaffyTree with default capacity.
func New() *TaffyTree {
	return NewWithCapacity(16)
}

// NewWithCapacity creates a new TaffyTree that can store capacity nodes before reallocation.
func NewWithCapacity(capacity int) *TaffyTree {
	return &TaffyTree{
		nodes:        make(map[NodeId]*nodeData, capacity),
		nodeContexts: make(map[NodeId]NodeContext, capacity),
		children:     make(map[NodeId][]NodeId, capacity),
		parents:      make(map[NodeId]*NodeId, capacity),
		config:       defaultTaffyConfig(),
		nextId:       0,
	}
}

// allocateId allocates and returns a new unique NodeId.
func (t *TaffyTree) allocateId() NodeId {
	id := NodeId(t.nextId)
	t.nextId++
	return id
}

// requireNode panics if the given node does not exist in the tree.
func (t *TaffyTree) requireNode(node NodeId) {
	if _, ok := t.nodes[node]; !ok {
		panic(fmt.Sprintf("taffy: node %v is not in the TaffyTree", node))
	}
}

// EnableRounding enables rounding of layout values. Rounding is enabled by default.
func (t *TaffyTree) EnableRounding() {
	t.config.useRounding = true
}

// DisableRounding disables rounding of layout values. Rounding is enabled by default.
func (t *TaffyTree) DisableRounding() {
	t.config.useRounding = false
}

// NewLeaf creates and adds a new unattached leaf node to the tree.
// Returns the NodeId of the new node.
func (t *TaffyTree) NewLeaf(style Style) NodeId {
	id := t.allocateId()
	t.nodes[id] = newNodeData(style)
	t.children[id] = []NodeId{}
	t.parents[id] = nil
	return id
}

// NewLeafWithContext creates and adds a new unattached leaf node with a supplied measure context.
// Returns the NodeId of the new node.
func (t *TaffyTree) NewLeafWithContext(style Style, context NodeContext) NodeId {
	id := t.allocateId()
	data := newNodeData(style)
	data.hasNodeContext = true
	t.nodes[id] = data
	t.nodeContexts[id] = context
	t.children[id] = []NodeId{}
	t.parents[id] = nil
	return id
}

// NewWithChildren creates and adds a new node with the given children.
// Returns the NodeId of the new node.
func (t *TaffyTree) NewWithChildren(style Style, children []NodeId) NodeId {
	id := t.allocateId()
	t.nodes[id] = newNodeData(style)

	childrenCopy := make([]NodeId, len(children))
	copy(childrenCopy, children)
	t.children[id] = childrenCopy
	t.parents[id] = nil

	for _, child := range children {
		childCopy := id
		t.parents[child] = &childCopy
	}

	return id
}

// Clear drops all nodes in the tree.
func (t *TaffyTree) Clear() {
	t.nodes = make(map[NodeId]*nodeData)
	t.nodeContexts = make(map[NodeId]NodeContext)
	t.children = make(map[NodeId][]NodeId)
	t.parents = make(map[NodeId]*NodeId)
	t.nextId = 0
}

// Remove removes a specific node from the tree.
// Detaches it from its parent (if any) and clears parent references for its children.
// Returns the id of the node removed. Panics if node does not exist.
func (t *TaffyTree) Remove(node NodeId) NodeId {
	t.requireNode(node)

	// Remove node from its parent's children list
	if parentPtr := t.parents[node]; parentPtr != nil {
		parent := *parentPtr
		if siblings, ok := t.children[parent]; ok {
			filtered := siblings[:0]
			for _, sibling := range siblings {
				if sibling != node {
					filtered = append(filtered, sibling)
				}
			}
			t.children[parent] = filtered
		}
	}

	// Clear parent references for this node's children
	if nodeChildren, ok := t.children[node]; ok {
		for _, child := range nodeChildren {
			t.parents[child] = nil
		}
	}

	delete(t.children, node)
	delete(t.parents, node)
	delete(t.nodes, node)
	delete(t.nodeContexts, node)

	return node
}

// SetNodeContext sets the measure context associated with a node.
// Pass nil to remove the context.
func (t *TaffyTree) SetNodeContext(node NodeId, measure *NodeContext) {
	t.requireNode(node)
	if measure != nil {
		t.nodes[node].hasNodeContext = true
		t.nodeContexts[node] = *measure
	} else {
		t.nodes[node].hasNodeContext = false
		delete(t.nodeContexts, node)
	}
	t.MarkDirty(node)
}

// GetNodeContext returns a pointer to the measure context for the given node, or nil if none.
func (t *TaffyTree) GetNodeContext(node NodeId) *NodeContext {
	t.requireNode(node)
	if ctx, ok := t.nodeContexts[node]; ok {
		return &ctx
	}
	return nil
}

// AddChild adds a child node under the supplied parent.
func (t *TaffyTree) AddChild(parent, child NodeId) {
	t.requireNode(parent)
	t.requireNode(child)
	parentCopy := parent
	t.parents[child] = &parentCopy
	t.children[parent] = append(t.children[parent], child)
	t.MarkDirty(parent)
}

// InsertChildAtIndex inserts a child node at the given index under the supplied parent,
// shifting all children after it to the right. Panics if index > child count.
func (t *TaffyTree) InsertChildAtIndex(parent NodeId, index int, child NodeId) {
	t.requireNode(parent)
	t.requireNode(child)
	childCount := len(t.children[parent])
	if index > childCount {
		panic(fmt.Sprintf("taffy: index (%d) should be <= child_count (%d) for parent node %v", index, childCount, parent))
	}
	parentCopy := parent
	t.parents[child] = &parentCopy
	children := t.children[parent]
	children = append(children, 0)
	copy(children[index+1:], children[index:])
	children[index] = child
	t.children[parent] = children
	t.MarkDirty(parent)
}

// SetChildren directly sets the children of the supplied parent.
// Old children have their parent reference cleared; new children have their parent set.
func (t *TaffyTree) SetChildren(parent NodeId, children []NodeId) {
	t.requireNode(parent)

	// Remove parent ref from old children
	for _, child := range t.children[parent] {
		t.parents[child] = nil
	}

	// Assign new parent refs, removing children from their previous parents first
	for _, child := range children {
		if prevParentPtr := t.parents[child]; prevParentPtr != nil {
			prevParent := *prevParentPtr
			if prevParent != parent {
				t.removeChildFromParent(prevParent, child)
			}
		}
		parentCopy := parent
		t.parents[child] = &parentCopy
	}

	childrenCopy := make([]NodeId, len(children))
	copy(childrenCopy, children)
	t.children[parent] = childrenCopy

	t.MarkDirty(parent)
}

// removeChildFromParent is an internal helper that removes child from parent's children list
// without marking dirty or updating the child's parent pointer.
func (t *TaffyTree) removeChildFromParent(parent, child NodeId) {
	if siblings, ok := t.children[parent]; ok {
		filtered := siblings[:0]
		for _, s := range siblings {
			if s != child {
				filtered = append(filtered, s)
			}
		}
		t.children[parent] = filtered
	}
}

// RemoveChild removes the child of the parent node.
// The child is not removed from the tree; it is simply detached from its parent.
// Returns the removed child's NodeId. Panics if child is not a child of parent.
func (t *TaffyTree) RemoveChild(parent, child NodeId) NodeId {
	t.requireNode(parent)
	t.requireNode(child)
	index := -1
	for i, c := range t.children[parent] {
		if c == child {
			index = i
			break
		}
	}
	if index < 0 {
		panic(fmt.Sprintf("taffy: node %v is not a child of node %v", child, parent))
	}
	return t.RemoveChildAtIndex(parent, index)
}

// RemoveChildAtIndex removes the child at the given index from the parent.
// The child is not removed from the tree; it is simply detached from its parent.
// Returns the removed child's NodeId. Panics if index is out of bounds.
func (t *TaffyTree) RemoveChildAtIndex(parent NodeId, index int) NodeId {
	t.requireNode(parent)
	children := t.children[parent]
	childCount := len(children)
	if index >= childCount {
		panic(fmt.Sprintf("taffy: index (%d) should be < child_count (%d) for parent node %v", index, childCount, parent))
	}
	child := children[index]
	t.children[parent] = append(children[:index], children[index+1:]...)
	t.parents[child] = nil
	t.MarkDirty(parent)
	return child
}

// ReplaceChildAtIndex replaces the child at the given index from the parent with a new child.
// The old child is detached from the parent. Returns the old child's NodeId.
// Panics if index is out of bounds.
func (t *TaffyTree) ReplaceChildAtIndex(parent NodeId, index int, newChild NodeId) NodeId {
	t.requireNode(parent)
	t.requireNode(newChild)
	children := t.children[parent]
	childCount := len(children)
	if index >= childCount {
		panic(fmt.Sprintf("taffy: index (%d) should be < child_count (%d) for parent node %v", index, childCount, parent))
	}
	oldChild := children[index]
	parentCopy := parent
	t.parents[newChild] = &parentCopy
	t.children[parent][index] = newChild
	t.parents[oldChild] = nil
	t.MarkDirty(parent)
	return oldChild
}

// ChildAtIndex returns the child node of the parent at the provided index.
// Panics if index is out of bounds.
func (t *TaffyTree) ChildAtIndex(parent NodeId, index int) NodeId {
	t.requireNode(parent)
	children := t.children[parent]
	childCount := len(children)
	if index >= childCount {
		panic(fmt.Sprintf("taffy: index (%d) should be < child_count (%d) for parent node %v", index, childCount, parent))
	}
	return children[index]
}

// TotalNodeCount returns the total number of nodes in the tree.
func (t *TaffyTree) TotalNodeCount() int {
	return len(t.nodes)
}

// Parent returns a pointer to the NodeId of the parent of the specified node, or nil if it has none.
// Panics if the node does not exist.
func (t *TaffyTree) Parent(child NodeId) *NodeId {
	t.requireNode(child)
	return t.parents[child]
}

// Children returns a list of children that belong to the parent node.
// Panics if the parent node does not exist.
func (t *TaffyTree) Children(parent NodeId) []NodeId {
	t.requireNode(parent)
	children := t.children[parent]
	result := make([]NodeId, len(children))
	copy(result, children)
	return result
}

// SetStyle sets the Style of the provided node and marks it dirty. Panics if node does not exist.
func (t *TaffyTree) SetStyle(node NodeId, style Style) {
	t.requireNode(node)
	t.nodes[node].style = style
	t.MarkDirty(node)
}

// GetStyle returns a pointer to the Style of the provided node. Panics if node does not exist.
func (t *TaffyTree) GetStyle(node NodeId) *Style {
	t.requireNode(node)
	return &t.nodes[node].style
}

// GetLayout returns a pointer to the Layout of the provided node.
// Returns final_layout if rounding is enabled, unrounded_layout otherwise.
// Panics if node does not exist.
func (t *TaffyTree) GetLayout(node NodeId) *Layout {
	t.requireNode(node)
	if t.config.useRounding {
		return &t.nodes[node].finalLayout
	}
	return &t.nodes[node].unroundedLayout
}

// MarkDirty marks the layout of a node and all its ancestors as outdated.
// Uses an iterative approach to propagate up the tree, stopping early if a node
// was already dirty (since its ancestors must also already be dirty).
func (t *TaffyTree) MarkDirty(node NodeId) {
	t.requireNode(node)
	current := node
	for {
		state := t.nodes[current].markDirty()
		if state == ClearStateAlreadyEmpty {
			// Already dirty — ancestors must also already be dirty
			break
		}
		parentPtr := t.parents[current]
		if parentPtr == nil {
			break
		}
		current = *parentPtr
	}
}

// IsDirty indicates whether the layout of the given node needs to be recomputed.
// Panics if node does not exist.
func (t *TaffyTree) IsDirty(node NodeId) bool {
	t.requireNode(node)
	return t.nodes[node].cache.isEmptyCheck()
}

// MeasureFunction is a function type used by taffyView during layout computation.
// It computes the intrinsic size of a leaf node given constraints.
type MeasureFunction = func(
	knownDimensions Size[Option[float32]],
	availableSpace Size[AvailableSpace],
	nodeId NodeId,
	context *NodeContext,
	style *Style,
) Size[float32]

// ComputeLayout updates the stored layout of the provided node and its children.
// Uses a no-op measure function (all leaf nodes measure as zero).
func (t *TaffyTree) ComputeLayout(node NodeId, availableSpace Size[AvailableSpace]) {
	t.requireNode(node)
	t.ComputeLayoutWithMeasure(node, availableSpace, nil)
}

// ComputeLayoutWithMeasure updates the stored layout of the provided node and its children
// using the given measure function for leaf nodes.
func (t *TaffyTree) ComputeLayoutWithMeasure(node NodeId, availableSpace Size[AvailableSpace], measureFn MeasureFunction) {
	t.requireNode(node)
	view := &taffyView{taffy: t, measureFunction: measureFn}
	computeRootLayout(view, node, availableSpace)
	if t.config.useRounding {
		roundLayout(view, node)
	}
}

// PrintTree prints a debug representation of the tree's layout starting from root.
func (t *TaffyTree) PrintTree(root NodeId) {
	t.requireNode(root)
	PrintTreeFn(t, root)
}

// --- traversePartialTree interface implementation on *TaffyTree ---

func (t *TaffyTree) childIds(parent NodeId) []NodeId {
	return t.children[parent]
}

func (t *TaffyTree) childCount(parent NodeId) int {
	return len(t.children[parent])
}

func (t *TaffyTree) getChildId(parent NodeId, index int) NodeId {
	return t.children[parent][index]
}

// --- cacheTree interface implementation on *TaffyTree ---

func (t *TaffyTree) cacheGet(nodeId NodeId, input *LayoutInput) *LayoutOutput {
	return t.nodes[nodeId].cache.get(input)
}

func (t *TaffyTree) cacheStore(nodeId NodeId, input *LayoutInput, output LayoutOutput) {
	t.nodes[nodeId].cache.store(input, output)
}

func (t *TaffyTree) cacheClear(nodeId NodeId) {
	t.nodes[nodeId].cache.clear()
}

// --- printTree interface implementation on *TaffyTree ---

func (t *TaffyTree) getDebugLabel(nodeId NodeId) string {
	node := t.nodes[nodeId]
	numChildren := t.childCount(nodeId)
	switch {
	case node.style.Display == DisplayNone:
		return "NONE"
	case numChildren == 0:
		return "LEAF"
	case node.style.Display == DisplayBlock:
		return "BLOCK"
	case node.style.Display == DisplayFlex:
		if node.style.FlexDirection.IsRow() {
			return "FLEX ROW"
		}
		return "FLEX COL"
	case node.style.Display == DisplayGrid:
		return "GRID"
	default:
		return "NODE"
	}
}

func (t *TaffyTree) getFinalLayout(nodeId NodeId) *Layout {
	return &t.nodes[nodeId].finalLayout
}

// --- roundTree interface implementation on *TaffyTree (via taffyView) ---
// The TaffyTree itself does not directly implement roundTree;
// taffyView wraps it and provides those methods.

// taffyView is an unexported wrapper combining TaffyTree with a measure function.
// It implements all the layout interfaces needed to run layout algorithms.
type taffyView struct {
	taffy           *TaffyTree
	measureFunction MeasureFunction
}

// --- traversePartialTree interface on *taffyView ---

func (v *taffyView) childIds(parent NodeId) []NodeId {
	return v.taffy.children[parent]
}

func (v *taffyView) childCount(parent NodeId) int {
	return len(v.taffy.children[parent])
}

func (v *taffyView) getChildId(parent NodeId, index int) NodeId {
	return v.taffy.children[parent][index]
}

// --- layoutPartialTree interface on *taffyView ---

func (v *taffyView) getCoreContainerStyle(nodeId NodeId) *Style {
	return &v.taffy.nodes[nodeId].style
}

func (v *taffyView) resolveCalcValue(_ uint64, _ float32) float32 {
	return 0.0
}

func (v *taffyView) setUnroundedLayout(nodeId NodeId, layout *Layout) {
	v.taffy.nodes[nodeId].unroundedLayout = *layout
}

// measureFunc returns a closure that calls the tree's measure function for the given node.
func (v *taffyView) measureFunc(nodeId NodeId) func(Size[Option[float32]], Size[AvailableSpace]) Size[float32] {
	return func(knownDimensions Size[Option[float32]], availableSpace Size[AvailableSpace]) Size[float32] {
		ctx := v.taffy.nodeContexts[nodeId]
		if v.measureFunction != nil {
			return v.measureFunction(knownDimensions, availableSpace, nodeId, &ctx, &v.taffy.nodes[nodeId].style)
		}
		if ctx != nil {
			return ctx(knownDimensions, SizeOptionNone, availableSpace, SizingModeInherentSize)
		}
		return SizeZeroF32
	}
}

// computeChildLayout dispatches to the correct algorithm based on Display.
func (v *taffyView) computeChildLayout(nodeId NodeId, inputs LayoutInput) LayoutOutput {
	node := v.taffy.nodes[nodeId]
	hasChildren := len(v.taffy.children[nodeId]) > 0

	return computeCachedLayout(v, nodeId, inputs, func(nodeId NodeId, inputs LayoutInput) LayoutOutput {
		style := &node.style
		display := style.Display

		if inputs.RunMode == RunModePerformHiddenLayout {
			return computeHiddenLayout(v, nodeId)
		}

		switch display {
		case DisplayNone:
			return computeHiddenLayout(v, nodeId)
		case DisplayFlex:
			if hasChildren {
				return computeFlexboxLayout(v, nodeId, inputs)
			}
			return computeLeafLayout(inputs, style, v.resolveCalcValue, v.measureFunc(nodeId))
		case DisplayGrid:
			if hasChildren {
				return computeGridLayout(v, nodeId, inputs)
			}
			return computeLeafLayout(inputs, style, v.resolveCalcValue, v.measureFunc(nodeId))
		case DisplayBlock:
			if hasChildren {
				return computeBlockLayout(v, nodeId, inputs, nil)
			}
			return computeLeafLayout(inputs, style, v.resolveCalcValue, v.measureFunc(nodeId))
		default:
			return computeLeafLayout(inputs, style, v.resolveCalcValue, v.measureFunc(nodeId))
		}
	})
}

// --- cacheTree interface on *taffyView ---

func (v *taffyView) cacheGet(nodeId NodeId, input *LayoutInput) *LayoutOutput {
	return v.taffy.nodes[nodeId].cache.get(input)
}

func (v *taffyView) cacheStore(nodeId NodeId, input *LayoutInput, output LayoutOutput) {
	v.taffy.nodes[nodeId].cache.store(input, output)
}

func (v *taffyView) cacheClear(nodeId NodeId) {
	v.taffy.nodes[nodeId].cache.clear()
}

// --- roundTree interface on *taffyView ---

func (v *taffyView) getUnroundedLayout(nodeId NodeId) *Layout {
	return &v.taffy.nodes[nodeId].unroundedLayout
}

func (v *taffyView) setFinalLayout(nodeId NodeId, layout *Layout) {
	v.taffy.nodes[nodeId].finalLayout = *layout
}

// --- printTree interface on *taffyView ---

func (v *taffyView) getDebugLabel(nodeId NodeId) string {
	return v.taffy.getDebugLabel(nodeId)
}

func (v *taffyView) getFinalLayout(nodeId NodeId) *Layout {
	return &v.taffy.nodes[nodeId].finalLayout
}

// --- layoutFlexboxContainer interface on *taffyView ---

func (v *taffyView) getFlexboxContainerStyle(nodeId NodeId) *Style {
	return &v.taffy.nodes[nodeId].style
}

func (v *taffyView) getFlexboxChildStyle(childNodeId NodeId) *Style {
	return &v.taffy.nodes[childNodeId].style
}

// --- layoutGridContainer interface on *taffyView ---

func (v *taffyView) getGridContainerStyle(nodeId NodeId) *Style {
	return &v.taffy.nodes[nodeId].style
}

func (v *taffyView) getGridChildStyle(childNodeId NodeId) *Style {
	return &v.taffy.nodes[childNodeId].style
}

// --- layoutBlockContainer interface on *taffyView ---

func (v *taffyView) getBlockContainerStyle(nodeId NodeId) *Style {
	return &v.taffy.nodes[nodeId].style
}

func (v *taffyView) getBlockChildStyle(childNodeId NodeId) *Style {
	return &v.taffy.nodes[childNodeId].style
}

func (v *taffyView) computeBlockChildLayout(nodeId NodeId, inputs LayoutInput, blockCtx *blockContext) LayoutOutput {
	node := v.taffy.nodes[nodeId]
	hasChildren := len(v.taffy.children[nodeId]) > 0

	return computeCachedLayout(v, nodeId, inputs, func(nodeId NodeId, inputs LayoutInput) LayoutOutput {
		style := &node.style
		display := style.Display

		if inputs.RunMode == RunModePerformHiddenLayout {
			return computeHiddenLayout(v, nodeId)
		}

		// For block children in the same BFC, pass the block context through
		if display == DisplayBlock && hasChildren {
			return computeBlockLayout(v, nodeId, inputs, blockCtx)
		}

		// For non-block children, fall through to normal dispatch
		return v.computeChildLayout(nodeId, inputs)
	})
}
