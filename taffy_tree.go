// Package taffy provides TaffyTree, the default implementation of the layout tree.
// This file is a port of tree/taffy_tree.rs from the Taffy Rust layout engine.
package taffy

import (
	"fmt"
	"taffy-go/lib"

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
func newNodeData(s style.Style) nodeData {
	return nodeData{
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
	knownDimensions geometry.Size[lib.Option[float32]],
	parentSize geometry.Size[lib.Option[float32]],
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
	nodes        lib.SlotMap[nodeData]
	nodeContexts lib.SlotMap[NodeContext]
	children     lib.SlotMap[[]tree.NodeId]
	parents      lib.SlotMap[lib.Option[tree.NodeId]]
	config       taffyConfig
}

// New creates a new TaffyTree with default capacity.
func New() *TaffyTree {
	return NewWithCapacity(16)
}

// NewWithCapacity creates a new TaffyTree that can store capacity nodes before reallocation.
func NewWithCapacity(capacity int) *TaffyTree {
	return &TaffyTree{
		nodes:        lib.NewSlotMapWithCapacity[nodeData](capacity),
		nodeContexts: lib.NewSlotMapWithCapacity[NodeContext](capacity),
		children:     lib.NewSlotMapWithCapacity[[]tree.NodeId](capacity),
		parents:      lib.NewSlotMapWithCapacity[lib.Option[tree.NodeId]](capacity),
		config:       defaultTaffyConfig(),
	}
}

// requireNode panics if the given node does not exist in the tree.
func (t *TaffyTree) requireNode(node tree.NodeId) {
	if !t.nodes.Contains(node) {
		panic(fmt.Sprintf("taffy: node {%d,%d} is not in the TaffyTree", node.Index, node.Generation))
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
	id := t.nodes.Insert(newNodeData(s))
	t.nodeContexts.Insert(nil)
	t.children.Insert([]tree.NodeId{})
	t.parents.Insert(lib.Option[tree.NodeId]{})
	return id
}

// NewLeafWithContext creates and adds a new unattached leaf node with a supplied measure context.
// Returns the NodeId of the new node.
func (t *TaffyTree) NewLeafWithContext(s style.Style, context NodeContext) tree.NodeId {
	data := newNodeData(s)
	data.hasNodeContext = true
	id := t.nodes.Insert(data)
	t.nodeContexts.Insert(context)
	t.children.Insert([]tree.NodeId{})
	t.parents.Insert(lib.Option[tree.NodeId]{})
	return id
}

// NewWithChildren creates and adds a new node with the given children.
// Returns the NodeId of the new node.
func (t *TaffyTree) NewWithChildren(s style.Style, children []tree.NodeId) tree.NodeId {
	id := t.nodes.Insert(newNodeData(s))
	t.nodeContexts.Insert(nil)

	childrenCopy := make([]tree.NodeId, len(children))
	copy(childrenCopy, children)
	t.children.Insert(childrenCopy)
	t.parents.Insert(lib.Option[tree.NodeId]{})

	for _, child := range children {
		*t.parents.Get(child) = lib.Some(id)
	}

	return id
}

// Clear drops all nodes in the tree.
func (t *TaffyTree) Clear() {
	t.nodes = lib.NewSlotMap[nodeData]()
	t.nodeContexts = lib.NewSlotMap[NodeContext]()
	t.children = lib.NewSlotMap[[]tree.NodeId]()
	t.parents = lib.NewSlotMap[lib.Option[tree.NodeId]]()
}

// Remove removes a specific node from the tree.
// Detaches it from its parent (if any) and clears parent references for its children.
// Returns the id of the node removed. Panics if node does not exist.
func (t *TaffyTree) Remove(node tree.NodeId) tree.NodeId {
	t.requireNode(node)

	// Remove node from its parent's children list
	if parentOpt := t.parents.Get(node); parentOpt != nil && parentOpt.IsSome {
		parent := parentOpt.Value
		if siblings := t.children.Get(parent); siblings != nil {
			filtered := (*siblings)[:0]
			for _, sibling := range *siblings {
				if sibling != node {
					filtered = append(filtered, sibling)
				}
			}
			*siblings = filtered
		}
	}

	// Clear parent references for this node's children
	if nodeChildren := t.children.Get(node); nodeChildren != nil {
		for _, child := range *nodeChildren {
			if p := t.parents.Get(child); p != nil {
				*p = lib.Option[tree.NodeId]{}
			}
		}
	}

	t.nodes.Remove(node)
	t.children.Remove(node)
	t.parents.Remove(node)
	t.nodeContexts.Remove(node)

	return node
}

// SetNodeContext sets the measure context associated with a node.
// Pass nil to remove the context.
func (t *TaffyTree) SetNodeContext(node tree.NodeId, measure *NodeContext) {
	t.requireNode(node)
	if measure != nil {
		t.nodes.Get(node).hasNodeContext = true
		*t.nodeContexts.Get(node) = *measure
	} else {
		t.nodes.Get(node).hasNodeContext = false
		*t.nodeContexts.Get(node) = nil
	}
	t.MarkDirty(node)
}

// GetNodeContext returns a pointer to the measure context for the given node, or nil if none.
func (t *TaffyTree) GetNodeContext(node tree.NodeId) *NodeContext {
	t.requireNode(node)
	ctx := t.nodeContexts.Get(node)
	if ctx == nil || *ctx == nil {
		return nil
	}
	return ctx
}

// AddChild adds a child node under the supplied parent.
func (t *TaffyTree) AddChild(parent, child tree.NodeId) {
	t.requireNode(parent)
	t.requireNode(child)
	*t.parents.Get(child) = lib.Some(parent)
	childrenPtr := t.children.Get(parent)
	*childrenPtr = append(*childrenPtr, child)
	t.MarkDirty(parent)
}

// InsertChildAtIndex inserts a child node at the given index under the supplied parent,
// shifting all children after it to the right. Panics if index > child count.
func (t *TaffyTree) InsertChildAtIndex(parent tree.NodeId, index int, child tree.NodeId) {
	t.requireNode(parent)
	t.requireNode(child)
	childrenPtr := t.children.Get(parent)
	childCount := len(*childrenPtr)
	if index > childCount {
		panic(fmt.Sprintf("taffy: index (%d) should be <= child_count (%d) for parent node {%d,%d}", index, childCount, parent.Index, parent.Generation))
	}
	*t.parents.Get(child) = lib.Some(parent)
	children := *childrenPtr
	children = append(children, tree.NodeId{})
	copy(children[index+1:], children[index:])
	children[index] = child
	*childrenPtr = children
	t.MarkDirty(parent)
}

// SetChildren directly sets the children of the supplied parent.
// Old children have their parent reference cleared; new children have their parent set.
func (t *TaffyTree) SetChildren(parent tree.NodeId, children []tree.NodeId) {
	t.requireNode(parent)

	// Remove parent ref from old children
	oldChildren := *t.children.Get(parent)
	for _, child := range oldChildren {
		*t.parents.Get(child) = lib.Option[tree.NodeId]{}
	}

	// Assign new parent refs, removing children from their previous parents first
	for _, child := range children {
		if prevParentOpt := t.parents.Get(child); prevParentOpt != nil && prevParentOpt.IsSome {
			prevParent := prevParentOpt.Value
			if prevParent != parent {
				t.removeChildFromParent(prevParent, child)
			}
		}
		*t.parents.Get(child) = lib.Some(parent)
	}

	childrenCopy := make([]tree.NodeId, len(children))
	copy(childrenCopy, children)
	*t.children.Get(parent) = childrenCopy

	t.MarkDirty(parent)
}

// removeChildFromParent is an internal helper that removes child from parent's children list
// without marking dirty or updating the child's parent pointer.
func (t *TaffyTree) removeChildFromParent(parent, child tree.NodeId) {
	if siblings := t.children.Get(parent); siblings != nil {
		filtered := (*siblings)[:0]
		for _, s := range *siblings {
			if s != child {
				filtered = append(filtered, s)
			}
		}
		*siblings = filtered
	}
}

// RemoveChild removes the child of the parent node.
// The child is not removed from the tree; it is simply detached from its parent.
// Returns the removed child's NodeId. Panics if child is not a child of parent.
func (t *TaffyTree) RemoveChild(parent, child tree.NodeId) tree.NodeId {
	t.requireNode(parent)
	t.requireNode(child)
	index := -1
	parentChildren := *t.children.Get(parent)
	for i, c := range parentChildren {
		if c == child {
			index = i
			break
		}
	}
	if index < 0 {
		panic(fmt.Sprintf("taffy: node {%d,%d} is not a child of node {%d,%d}", child.Index, child.Generation, parent.Index, parent.Generation))
	}
	return t.RemoveChildAtIndex(parent, index)
}

// RemoveChildAtIndex removes the child at the given index from the parent.
// The child is not removed from the tree; it is simply detached from its parent.
// Returns the removed child's NodeId. Panics if index is out of bounds.
func (t *TaffyTree) RemoveChildAtIndex(parent tree.NodeId, index int) tree.NodeId {
	t.requireNode(parent)
	childrenPtr := t.children.Get(parent)
	children := *childrenPtr
	childCount := len(children)
	if index >= childCount {
		panic(fmt.Sprintf("taffy: index (%d) should be < child_count (%d) for parent node {%d,%d}", index, childCount, parent.Index, parent.Generation))
	}
	child := children[index]
	*childrenPtr = append(children[:index], children[index+1:]...)
	*t.parents.Get(child) = lib.Option[tree.NodeId]{}
	t.MarkDirty(parent)
	return child
}

// ReplaceChildAtIndex replaces the child at the given index from the parent with a new child.
// The old child is detached from the parent. Returns the old child's NodeId.
// Panics if index is out of bounds.
func (t *TaffyTree) ReplaceChildAtIndex(parent tree.NodeId, index int, newChild tree.NodeId) tree.NodeId {
	t.requireNode(parent)
	t.requireNode(newChild)
	childrenPtr := t.children.Get(parent)
	children := *childrenPtr
	childCount := len(children)
	if index >= childCount {
		panic(fmt.Sprintf("taffy: index (%d) should be < child_count (%d) for parent node {%d,%d}", index, childCount, parent.Index, parent.Generation))
	}
	oldChild := children[index]
	*t.parents.Get(newChild) = lib.Some(parent)
	(*childrenPtr)[index] = newChild
	*t.parents.Get(oldChild) = lib.Option[tree.NodeId]{}
	t.MarkDirty(parent)
	return oldChild
}

// ChildAtIndex returns the child node of the parent at the provided index.
// Panics if index is out of bounds.
func (t *TaffyTree) ChildAtIndex(parent tree.NodeId, index int) tree.NodeId {
	t.requireNode(parent)
	children := *t.children.Get(parent)
	childCount := len(children)
	if index >= childCount {
		panic(fmt.Sprintf("taffy: index (%d) should be < child_count (%d) for parent node {%d,%d}", index, childCount, parent.Index, parent.Generation))
	}
	return children[index]
}

// TotalNodeCount returns the total number of nodes in the tree.
func (t *TaffyTree) TotalNodeCount() int {
	return t.nodes.Len()
}

// Parent returns the parent of the specified node as an Option (None if it has no parent).
// Panics if the node does not exist.
func (t *TaffyTree) Parent(child tree.NodeId) lib.Option[tree.NodeId] {
	t.requireNode(child)
	return *t.parents.Get(child)
}

// Children returns a list of children that belong to the parent node.
// Panics if the parent node does not exist.
func (t *TaffyTree) Children(parent tree.NodeId) []tree.NodeId {
	t.requireNode(parent)
	children := *t.children.Get(parent)
	result := make([]tree.NodeId, len(children))
	copy(result, children)
	return result
}

// SetStyle sets the Style of the provided node and marks it dirty. Panics if node does not exist.
func (t *TaffyTree) SetStyle(node tree.NodeId, s style.Style) {
	t.requireNode(node)
	t.nodes.Get(node).style = s
	t.MarkDirty(node)
}

// GetStyle returns a pointer to the Style of the provided node. Panics if node does not exist.
func (t *TaffyTree) GetStyle(node tree.NodeId) *style.Style {
	t.requireNode(node)
	return &t.nodes.Get(node).style
}

// GetLayout returns a pointer to the Layout of the provided node.
// Returns final_layout if rounding is enabled, unrounded_layout otherwise.
// Panics if node does not exist.
func (t *TaffyTree) GetLayout(node tree.NodeId) *tree.Layout {
	t.requireNode(node)
	if t.config.useRounding {
		return &t.nodes.Get(node).finalLayout
	}
	return &t.nodes.Get(node).unroundedLayout
}

// MarkDirty marks the layout of a node and all its ancestors as outdated.
// Uses an iterative approach to propagate up the tree, stopping early if a node
// was already dirty (since its ancestors must also already be dirty).
func (t *TaffyTree) MarkDirty(node tree.NodeId) {
	t.requireNode(node)
	current := node
	for {
		state := t.nodes.Get(current).markDirty()
		if state == tree.ClearStateAlreadyEmpty {
			// Already dirty - ancestors must also already be dirty
			break
		}
		parentOpt := *t.parents.Get(current)
		if !parentOpt.IsSome {
			break
		}
		current = parentOpt.Value
	}
}

// IsDirty indicates whether the layout of the given node needs to be recomputed.
// Panics if node does not exist.
func (t *TaffyTree) IsDirty(node tree.NodeId) bool {
	t.requireNode(node)
	return t.nodes.Get(node).cache.IsEmptyCheck()
}

// MeasureFunction is a function type used by taffyView during layout computation.
// It computes the intrinsic size of a leaf node given constraints.
type MeasureFunction = func(
	knownDimensions geometry.Size[lib.Option[float32]],
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
