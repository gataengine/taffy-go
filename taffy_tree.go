// Package taffy provides TaffyTree, the default implementation of the layout tree.
// This file is a port of tree/taffy_tree.rs from the Taffy Rust layout engine.
package taffy

import (
	"fmt"

	"taffy-go/compute"
	"taffy-go/geometry"
	"taffy-go/style"
	"taffy-go/tree"
)

// nodeData stores style + layout data for a single node.
type nodeData struct {
	style           style.Style
	unroundedLayout tree.Layout
	finalLayout     tree.Layout
	cache           tree.Cache
	hasNodeContext  bool
}

// newNodeData creates node data with the given style and an empty cache.
func newNodeData(s style.Style) *nodeData {
	return &nodeData{
		style:           s,
		cache:           tree.NewCache(),
		unroundedLayout: tree.NewLayout(),
		finalLayout:     tree.NewLayout(),
		hasNodeContext:  false,
	}
}

// markDirty clears the node's cache and reports whether it was already empty.
func (nd *nodeData) markDirty() tree.ClearState {
	return nd.cache.Clear()
}

// NodeContext is a user-provided measure function for leaf nodes.
// It computes the intrinsic size of the node given constraints.
type NodeContext = func(
	knownDimensions geometry.Size[geometry.Option[float32]],
	parentSize geometry.Size[geometry.Option[float32]],
	availableSpace geometry.Size[style.AvailableSpace],
	sizingMode tree.SizingMode,
) geometry.Size[float32]

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
	nodes        map[tree.NodeId]*nodeData
	nodeContexts map[tree.NodeId]NodeContext
	children     map[tree.NodeId][]tree.NodeId
	parents      map[tree.NodeId]*tree.NodeId
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
		nodes:        make(map[tree.NodeId]*nodeData, capacity),
		nodeContexts: make(map[tree.NodeId]NodeContext, capacity),
		children:     make(map[tree.NodeId][]tree.NodeId, capacity),
		parents:      make(map[tree.NodeId]*tree.NodeId, capacity),
		config:       defaultTaffyConfig(),
		nextId:       0,
	}
}

// allocateId allocates and returns a new unique NodeId.
func (t *TaffyTree) allocateId() tree.NodeId {
	id := tree.NodeId(t.nextId)
	t.nextId++
	return id
}

// requireNode panics if the given node does not exist in the tree.
func (t *TaffyTree) requireNode(node tree.NodeId) {
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
func (t *TaffyTree) NewLeaf(s style.Style) tree.NodeId {
	id := t.allocateId()
	t.nodes[id] = newNodeData(s)
	t.children[id] = []tree.NodeId{}
	t.parents[id] = nil
	return id
}

// NewLeafWithContext creates and adds a new unattached leaf node with a supplied measure context.
// Returns the NodeId of the new node.
func (t *TaffyTree) NewLeafWithContext(s style.Style, context NodeContext) tree.NodeId {
	id := t.allocateId()
	data := newNodeData(s)
	data.hasNodeContext = true
	t.nodes[id] = data
	t.nodeContexts[id] = context
	t.children[id] = []tree.NodeId{}
	t.parents[id] = nil
	return id
}

// NewWithChildren creates and adds a new node with the given children.
// Returns the NodeId of the new node.
func (t *TaffyTree) NewWithChildren(s style.Style, children []tree.NodeId) tree.NodeId {
	id := t.allocateId()
	t.nodes[id] = newNodeData(s)

	childrenCopy := make([]tree.NodeId, len(children))
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
	t.nodes = make(map[tree.NodeId]*nodeData)
	t.nodeContexts = make(map[tree.NodeId]NodeContext)
	t.children = make(map[tree.NodeId][]tree.NodeId)
	t.parents = make(map[tree.NodeId]*tree.NodeId)
	t.nextId = 0
}

// Remove removes a specific node from the tree.
// Detaches it from its parent (if any) and clears parent references for its children.
// Returns the id of the node removed. Panics if node does not exist.
func (t *TaffyTree) Remove(node tree.NodeId) tree.NodeId {
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
func (t *TaffyTree) SetNodeContext(node tree.NodeId, measure *NodeContext) {
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
func (t *TaffyTree) GetNodeContext(node tree.NodeId) *NodeContext {
	t.requireNode(node)
	if ctx, ok := t.nodeContexts[node]; ok {
		return &ctx
	}
	return nil
}

// AddChild adds a child node under the supplied parent.
func (t *TaffyTree) AddChild(parent, child tree.NodeId) {
	t.requireNode(parent)
	t.requireNode(child)
	parentCopy := parent
	t.parents[child] = &parentCopy
	t.children[parent] = append(t.children[parent], child)
	t.MarkDirty(parent)
}

// InsertChildAtIndex inserts a child node at the given index under the supplied parent,
// shifting all children after it to the right. Panics if index > child count.
func (t *TaffyTree) InsertChildAtIndex(parent tree.NodeId, index int, child tree.NodeId) {
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
func (t *TaffyTree) SetChildren(parent tree.NodeId, children []tree.NodeId) {
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

	childrenCopy := make([]tree.NodeId, len(children))
	copy(childrenCopy, children)
	t.children[parent] = childrenCopy

	t.MarkDirty(parent)
}

// removeChildFromParent is an internal helper that removes child from parent's children list
// without marking dirty or updating the child's parent pointer.
func (t *TaffyTree) removeChildFromParent(parent, child tree.NodeId) {
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
func (t *TaffyTree) RemoveChild(parent, child tree.NodeId) tree.NodeId {
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
func (t *TaffyTree) RemoveChildAtIndex(parent tree.NodeId, index int) tree.NodeId {
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
func (t *TaffyTree) ReplaceChildAtIndex(parent tree.NodeId, index int, newChild tree.NodeId) tree.NodeId {
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
func (t *TaffyTree) ChildAtIndex(parent tree.NodeId, index int) tree.NodeId {
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
func (t *TaffyTree) Parent(child tree.NodeId) *tree.NodeId {
	t.requireNode(child)
	return t.parents[child]
}

// Children returns a list of children that belong to the parent node.
// Panics if the parent node does not exist.
func (t *TaffyTree) Children(parent tree.NodeId) []tree.NodeId {
	t.requireNode(parent)
	children := t.children[parent]
	result := make([]tree.NodeId, len(children))
	copy(result, children)
	return result
}

// SetStyle sets the Style of the provided node and marks it dirty. Panics if node does not exist.
func (t *TaffyTree) SetStyle(node tree.NodeId, s style.Style) {
	t.requireNode(node)
	t.nodes[node].style = s
	t.MarkDirty(node)
}

// GetStyle returns a pointer to the Style of the provided node. Panics if node does not exist.
func (t *TaffyTree) GetStyle(node tree.NodeId) *style.Style {
	t.requireNode(node)
	return &t.nodes[node].style
}

// GetLayout returns a pointer to the Layout of the provided node.
// Returns final_layout if rounding is enabled, unrounded_layout otherwise.
// Panics if node does not exist.
func (t *TaffyTree) GetLayout(node tree.NodeId) *tree.Layout {
	t.requireNode(node)
	if t.config.useRounding {
		return &t.nodes[node].finalLayout
	}
	return &t.nodes[node].unroundedLayout
}

// MarkDirty marks the layout of a node and all its ancestors as outdated.
// Uses an iterative approach to propagate up the tree, stopping early if a node
// was already dirty (since its ancestors must also already be dirty).
func (t *TaffyTree) MarkDirty(node tree.NodeId) {
	t.requireNode(node)
	current := node
	for {
		state := t.nodes[current].markDirty()
		if state == tree.ClearStateAlreadyEmpty {
			// Already dirty - ancestors must also already be dirty
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
func (t *TaffyTree) IsDirty(node tree.NodeId) bool {
	t.requireNode(node)
	return t.nodes[node].cache.IsEmptyCheck()
}

// MeasureFunction is a function type used by taffyView during layout computation.
// It computes the intrinsic size of a leaf node given constraints.
type MeasureFunction = func(
	knownDimensions geometry.Size[geometry.Option[float32]],
	availableSpace geometry.Size[style.AvailableSpace],
	nodeId tree.NodeId,
	context *NodeContext,
	style *style.Style,
) geometry.Size[float32]

// ComputeLayout updates the stored layout of the provided node and its children.
// Uses a no-op measure function (all leaf nodes measure as zero).
func (t *TaffyTree) ComputeLayout(node tree.NodeId, availableSpace geometry.Size[style.AvailableSpace]) {
	t.requireNode(node)
	t.ComputeLayoutWithMeasure(node, availableSpace, nil)
}

// ComputeLayoutWithMeasure updates the stored layout of the provided node and its children
// using the given measure function for leaf nodes.
func (t *TaffyTree) ComputeLayoutWithMeasure(node tree.NodeId, availableSpace geometry.Size[style.AvailableSpace], measureFn MeasureFunction) {
	t.requireNode(node)
	view := &taffyView{taffy: t, measureFunction: measureFn}
	computeRootLayout(view, node, availableSpace)
	if t.config.useRounding {
		compute.RoundLayout(view, node)
	}
}

// PrintTree prints a debug representation of the tree's layout starting from root.
func (t *TaffyTree) PrintTree(root tree.NodeId) {
	t.requireNode(root)
	view := &taffyView{taffy: t}
	PrintTreeFn(view, root)
}
