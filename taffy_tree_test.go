package taffy

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"taffy-go/geometry"
	"taffy-go/style"
	treepkg "taffy-go/tree"
)

var _ = Describe("TaffyTree", func() {
	Describe("New", func() {
		It("creates an empty tree", func() {
			tree := New()
			Expect(tree).NotTo(BeNil())
			Expect(tree.TotalNodeCount()).To(Equal(0))
		})
	})

	Describe("NewWithCapacity", func() {
		It("creates an empty tree with given capacity", func() {
			tree := NewWithCapacity(8)
			Expect(tree).NotTo(BeNil())
			Expect(tree.TotalNodeCount()).To(Equal(0))
		})
	})

	Describe("EnableRounding / DisableRounding", func() {
		It("toggles rounding without panicking", func() {
			tree := New()
			tree.DisableRounding()
			tree.EnableRounding()
		})
	})

	Describe("NewLeaf", func() {
		It("creates a leaf node with no children", func() {
			tree := New()
			node := tree.NewLeaf(style.Style{})
			Expect(tree.TotalNodeCount()).To(Equal(1))
			Expect(tree.Children(node)).To(BeEmpty())
		})
	})

	Describe("NewLeafWithContext", func() {
		It("creates a leaf node with a measure context", func() {
			tree := New()
			ctx := NodeContext(func(
				_ geometry.Size[geometry.Option[float32]],
				_ geometry.Size[geometry.Option[float32]],
				_ geometry.Size[style.AvailableSpace],
				_ treepkg.SizingMode,
			) geometry.Size[float32] {
				return geometry.Size[float32]{Width: 42.0, Height: 42.0}
			})
			node := tree.NewLeafWithContext(style.Style{}, ctx)
			Expect(tree.TotalNodeCount()).To(Equal(1))
			Expect(tree.GetNodeContext(node)).NotTo(BeNil())
		})
	})

	Describe("NewWithChildren", func() {
		It("creates a node with parent-child relationships", func() {
			tree := New()
			child0 := tree.NewLeaf(style.Style{})
			child1 := tree.NewLeaf(style.Style{})
			parent := tree.NewWithChildren(style.Style{}, []treepkg.NodeId{child0, child1})

			Expect(tree.TotalNodeCount()).To(Equal(3))

			children := tree.Children(parent)
			Expect(children).To(HaveLen(2))
			Expect(children[0]).To(Equal(child0))
			Expect(children[1]).To(Equal(child1))

			Expect(tree.Parent(child0)).NotTo(BeNil())
			Expect(*tree.Parent(child0)).To(Equal(parent))
			Expect(tree.Parent(child1)).NotTo(BeNil())
			Expect(*tree.Parent(child1)).To(Equal(parent))
		})
	})

	Describe("Clear", func() {
		It("removes all nodes", func() {
			tree := New()
			tree.NewLeaf(style.Style{})
			tree.NewLeaf(style.Style{})
			Expect(tree.TotalNodeCount()).To(Equal(2))
			tree.Clear()
			Expect(tree.TotalNodeCount()).To(Equal(0))
		})
	})

	Describe("Remove", func() {
		It("removes a leaf node", func() {
			tree := New()
			node := tree.NewLeaf(style.Style{})
			Expect(tree.TotalNodeCount()).To(Equal(1))
			removed := tree.Remove(node)
			Expect(removed).To(Equal(node))
			Expect(tree.TotalNodeCount()).To(Equal(0))
		})

		It("removes node and detaches from parent", func() {
			tree := New()
			child := tree.NewLeaf(style.Style{})
			parent := tree.NewWithChildren(style.Style{}, []treepkg.NodeId{child})
			Expect(tree.Children(parent)).To(HaveLen(1))

			tree.Remove(child)
			Expect(tree.Children(parent)).To(BeEmpty())
		})

		It("removes node and detaches its children", func() {
			tree := New()
			node2 := tree.NewLeaf(style.Style{})
			node1 := tree.NewWithChildren(style.Style{}, []treepkg.NodeId{node2})
			node0 := tree.NewWithChildren(style.Style{}, []treepkg.NodeId{node1})

			Expect(tree.Children(node0)).To(Equal([]treepkg.NodeId{node1}))
			Expect(tree.Children(node1)).To(Equal([]treepkg.NodeId{node2}))

			tree.Remove(node1)

			Expect(tree.Children(node0)).To(BeEmpty())
			Expect(tree.Children(node2)).To(BeEmpty())
			Expect(tree.Parent(node2)).To(BeNil())
		})

		It("panics for invalid node", func() {
			tree := New()
			Expect(func() { tree.Remove(treepkg.NodeId(999)) }).To(Panic())
		})
	})

	Describe("SetNodeContext / GetNodeContext", func() {
		It("sets and retrieves measure context", func() {
			tree := New()
			node := tree.NewLeaf(style.Style{})
			Expect(tree.GetNodeContext(node)).To(BeNil())

			ctx := NodeContext(func(
				_ geometry.Size[geometry.Option[float32]],
				_ geometry.Size[geometry.Option[float32]],
				_ geometry.Size[style.AvailableSpace],
				_ treepkg.SizingMode,
			) geometry.Size[float32] {
				return geometry.Size[float32]{}
			})
			tree.SetNodeContext(node, &ctx)
			Expect(tree.GetNodeContext(node)).NotTo(BeNil())
		})

		It("removes context when nil is passed", func() {
			tree := New()
			ctx := NodeContext(func(
				_ geometry.Size[geometry.Option[float32]],
				_ geometry.Size[geometry.Option[float32]],
				_ geometry.Size[style.AvailableSpace],
				_ treepkg.SizingMode,
			) geometry.Size[float32] {
				return geometry.Size[float32]{}
			})
			node := tree.NewLeafWithContext(style.Style{}, ctx)
			Expect(tree.GetNodeContext(node)).NotTo(BeNil())
			tree.SetNodeContext(node, nil)
			Expect(tree.GetNodeContext(node)).To(BeNil())
		})
	})

	Describe("AddChild", func() {
		It("adds children incrementally", func() {
			tree := New()
			parent := tree.NewLeaf(style.Style{})
			Expect(tree.Children(parent)).To(BeEmpty())

			child0 := tree.NewLeaf(style.Style{})
			tree.AddChild(parent, child0)
			Expect(tree.Children(parent)).To(HaveLen(1))

			child1 := tree.NewLeaf(style.Style{})
			tree.AddChild(parent, child1)
			Expect(tree.Children(parent)).To(HaveLen(2))
		})
	})

	Describe("InsertChildAtIndex", func() {
		It("inserts a child at the specified index", func() {
			tree := New()
			child0 := tree.NewLeaf(style.Style{})
			child1 := tree.NewLeaf(style.Style{})
			parent := tree.NewWithChildren(style.Style{}, []treepkg.NodeId{child0, child1})

			child2 := tree.NewLeaf(style.Style{})
			tree.InsertChildAtIndex(parent, 1, child2)

			children := tree.Children(parent)
			Expect(children).To(HaveLen(3))
			Expect(children[0]).To(Equal(child0))
			Expect(children[1]).To(Equal(child2))
			Expect(children[2]).To(Equal(child1))
		})

		It("panics if index out of bounds", func() {
			tree := New()
			parent := tree.NewLeaf(style.Style{})
			child := tree.NewLeaf(style.Style{})
			Expect(func() { tree.InsertChildAtIndex(parent, 5, child) }).To(Panic())
		})
	})

	Describe("SetChildren", func() {
		It("replaces all children", func() {
			tree := New()
			oldChild := tree.NewLeaf(style.Style{})
			parent := tree.NewWithChildren(style.Style{}, []treepkg.NodeId{oldChild})
			Expect(tree.Children(parent)).To(HaveLen(1))

			newChild0 := tree.NewLeaf(style.Style{})
			newChild1 := tree.NewLeaf(style.Style{})
			tree.SetChildren(parent, []treepkg.NodeId{newChild0, newChild1})

			Expect(tree.Children(parent)).To(HaveLen(2))
			Expect(tree.Parent(oldChild)).To(BeNil())
			Expect(*tree.Parent(newChild0)).To(Equal(parent))
			Expect(*tree.Parent(newChild1)).To(Equal(parent))
		})
	})

	Describe("RemoveChild", func() {
		It("detaches child from parent", func() {
			tree := New()
			child := tree.NewLeaf(style.Style{})
			parent := tree.NewWithChildren(style.Style{}, []treepkg.NodeId{child})
			Expect(tree.Children(parent)).To(HaveLen(1))

			removed := tree.RemoveChild(parent, child)
			Expect(removed).To(Equal(child))
			Expect(tree.Children(parent)).To(BeEmpty())
			Expect(tree.Parent(child)).To(BeNil())
			// child still exists in the tree
			Expect(tree.TotalNodeCount()).To(Equal(2))
		})

		It("panics if child not found under parent", func() {
			tree := New()
			parent := tree.NewLeaf(style.Style{})
			notChild := tree.NewLeaf(style.Style{})
			Expect(func() { tree.RemoveChild(parent, notChild) }).To(Panic())
		})
	})

	Describe("RemoveChildAtIndex", func() {
		It("removes child by index", func() {
			tree := New()
			child0 := tree.NewLeaf(style.Style{})
			child1 := tree.NewLeaf(style.Style{})
			parent := tree.NewWithChildren(style.Style{}, []treepkg.NodeId{child0, child1})

			removed := tree.RemoveChildAtIndex(parent, 0)
			Expect(removed).To(Equal(child0))
			Expect(tree.Children(parent)).To(HaveLen(1))
			Expect(tree.Children(parent)[0]).To(Equal(child1))
		})

		It("panics if index out of bounds", func() {
			tree := New()
			parent := tree.NewLeaf(style.Style{})
			Expect(func() { tree.RemoveChildAtIndex(parent, 0) }).To(Panic())
		})
	})

	Describe("ReplaceChildAtIndex", func() {
		It("replaces child at index and returns old child", func() {
			tree := New()
			child0 := tree.NewLeaf(style.Style{})
			child1 := tree.NewLeaf(style.Style{})
			parent := tree.NewWithChildren(style.Style{}, []treepkg.NodeId{child0, child1})

			newChild := tree.NewLeaf(style.Style{})
			old := tree.ReplaceChildAtIndex(parent, 0, newChild)
			Expect(old).To(Equal(child0))
			Expect(tree.Children(parent)[0]).To(Equal(newChild))
			Expect(tree.Parent(child0)).To(BeNil())
			Expect(*tree.Parent(newChild)).To(Equal(parent))
		})
	})

	Describe("ChildAtIndex", func() {
		It("returns the child at the specified index", func() {
			tree := New()
			child0 := tree.NewLeaf(style.Style{})
			child1 := tree.NewLeaf(style.Style{})
			parent := tree.NewWithChildren(style.Style{}, []treepkg.NodeId{child0, child1})

			Expect(tree.ChildAtIndex(parent, 0)).To(Equal(child0))
			Expect(tree.ChildAtIndex(parent, 1)).To(Equal(child1))
		})

		It("panics if index out of bounds", func() {
			tree := New()
			parent := tree.NewLeaf(style.Style{})
			Expect(func() { tree.ChildAtIndex(parent, 0) }).To(Panic())
		})
	})

	Describe("TotalNodeCount", func() {
		It("returns the number of nodes in the tree", func() {
			tree := New()
			Expect(tree.TotalNodeCount()).To(Equal(0))
			tree.NewLeaf(style.Style{})
			Expect(tree.TotalNodeCount()).To(Equal(1))
			tree.NewLeaf(style.Style{})
			Expect(tree.TotalNodeCount()).To(Equal(2))
		})
	})

	Describe("Parent", func() {
		It("returns nil for root nodes", func() {
			tree := New()
			node := tree.NewLeaf(style.Style{})
			Expect(tree.Parent(node)).To(BeNil())
		})

		It("returns parent for child nodes", func() {
			tree := New()
			child := tree.NewLeaf(style.Style{})
			parent := tree.NewWithChildren(style.Style{}, []treepkg.NodeId{child})
			Expect(tree.Parent(child)).NotTo(BeNil())
			Expect(*tree.Parent(child)).To(Equal(parent))
		})
	})

	Describe("Children", func() {
		It("returns a copy of the children list", func() {
			tree := New()
			child0 := tree.NewLeaf(style.Style{})
			child1 := tree.NewLeaf(style.Style{})
			parent := tree.NewWithChildren(style.Style{}, []treepkg.NodeId{child0, child1})

			children := tree.Children(parent)
			Expect(children).To(Equal([]treepkg.NodeId{child0, child1}))
		})
	})

	Describe("SetStyle / GetStyle", func() {
		It("sets and retrieves style", func() {
			tree := New()
			node := tree.NewLeaf(style.Style{})

			st := style.DefaultStyle()
			st.Display = style.DisplayNone
			tree.SetStyle(node, st)

			retrieved := tree.GetStyle(node)
			Expect(retrieved.Display).To(Equal(style.DisplayNone))
		})
	})

	Describe("GetLayout", func() {
		It("returns the layout for a node", func() {
			tree := New()
			node := tree.NewLeaf(style.Style{})
			layout := tree.GetLayout(node)
			Expect(layout).NotTo(BeNil())
			Expect(layout.Size.Width).To(Equal(float32(0)))
		})
	})

	Describe("MarkDirty / IsDirty", func() {
		It("new nodes start as dirty", func() {
			tree := New()
			node := tree.NewLeaf(style.Style{})
			Expect(tree.IsDirty(node)).To(BeTrue())
		})

		It("MarkDirty does not panic and node remains dirty", func() {
			tree := New()
			node := tree.NewLeaf(style.Style{})
			tree.MarkDirty(node)
			Expect(tree.IsDirty(node)).To(BeTrue())
		})

		It("propagates dirty mark up to ancestors", func() {
			tree := New()
			grandchild := tree.NewLeaf(style.Style{})
			child := tree.NewWithChildren(style.Style{}, []treepkg.NodeId{grandchild})
			root := tree.NewWithChildren(style.Style{}, []treepkg.NodeId{child})

			// All nodes start dirty
			Expect(tree.IsDirty(root)).To(BeTrue())
			Expect(tree.IsDirty(child)).To(BeTrue())
			Expect(tree.IsDirty(grandchild)).To(BeTrue())

			// MarkDirty on grandchild should not panic; ancestors remain dirty
			tree.MarkDirty(grandchild)
			Expect(tree.IsDirty(grandchild)).To(BeTrue())
			Expect(tree.IsDirty(child)).To(BeTrue())
			Expect(tree.IsDirty(root)).To(BeTrue())
		})

		It("panics for invalid node", func() {
			tree := New()
			Expect(func() { tree.MarkDirty(treepkg.NodeId(999)) }).To(Panic())
		})
	})

	Describe("PrintTree", func() {
		It("prints without panicking", func() {
			tree := New()
			child := tree.NewLeaf(style.Style{})
			root := tree.NewWithChildren(style.Style{}, []treepkg.NodeId{child})
			Expect(func() { tree.PrintTree(root) }).NotTo(Panic())
		})
	})
})
