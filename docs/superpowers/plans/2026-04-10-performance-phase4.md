# Performance Phase 4 Implementation Plan

> **Status (audit 2026-10-11): Implemented.** Checkboxes below were never ticked.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce Go/Rust performance ratio from 2.2-4.1x toward 1.8-3.0x via SlotMap, sort optimization, and surgical allocation reductions.

**Architecture:** Five independent optimizations: (1) lib.SlotMap generic data structure + TaffyTree migration replacing hash maps with array-indexed slots, (2) lib.SortStableByIndex applied to grid sort, (3) Layout value semantics removing pointer indirection, (4) flat cellOccupancyMatrix removing per-row allocations, (5) itemBatcher sub-slice eliminating per-batch allocations.

**Tech Stack:** Go 1.25, generics, sync.Pool (existing), Ginkgo/Gomega test framework

---

### Task 1: lib.SlotMap — Implementation and Tests

**Files:**
- Create: `lib/slotmap.go`
- Create: `lib/slotmap_test.go`

- [ ] **Step 1: Write the failing test file**

Create `lib/slotmap_test.go`:

```go
package lib_test

import (
	"testing"

	"taffy-go/lib"
)

func TestSlotMap_InsertAndGet(t *testing.T) {
	sm := lib.NewSlotMap[string]()
	k1 := sm.Insert("hello")
	k2 := sm.Insert("world")

	if v := sm.Get(k1); v == nil || *v != "hello" {
		t.Fatalf("Get(k1) = %v, want 'hello'", v)
	}
	if v := sm.Get(k2); v == nil || *v != "world" {
		t.Fatalf("Get(k2) = %v, want 'world'", v)
	}
	if sm.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", sm.Len())
	}
}

func TestSlotMap_Remove(t *testing.T) {
	sm := lib.NewSlotMap[int]()
	k1 := sm.Insert(42)
	k2 := sm.Insert(99)

	val, ok := sm.Remove(k1)
	if !ok || val != 42 {
		t.Fatalf("Remove(k1) = (%v, %v), want (42, true)", val, ok)
	}
	if sm.Get(k1) != nil {
		t.Fatal("Get(k1) after remove should return nil")
	}
	if sm.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", sm.Len())
	}
	if v := sm.Get(k2); v == nil || *v != 99 {
		t.Fatalf("Get(k2) = %v, want 99", v)
	}
}

func TestSlotMap_GenerationReuse(t *testing.T) {
	sm := lib.NewSlotMap[string]()
	k1 := sm.Insert("first")
	sm.Remove(k1)

	// Insert into the freed slot
	k2 := sm.Insert("second")

	// k1 and k2 should share the same index but different generation
	if k1.Index != k2.Index {
		t.Fatalf("expected reuse of index %d, got %d", k1.Index, k2.Index)
	}
	if k1.Generation == k2.Generation {
		t.Fatal("expected different generations after reuse")
	}

	// Old key must not resolve
	if sm.Get(k1) != nil {
		t.Fatal("Get(k1) should return nil after slot was reused")
	}
	if v := sm.Get(k2); v == nil || *v != "second" {
		t.Fatalf("Get(k2) = %v, want 'second'", v)
	}
}

func TestSlotMap_Contains(t *testing.T) {
	sm := lib.NewSlotMap[int]()
	k := sm.Insert(1)
	if !sm.Contains(k) {
		t.Fatal("Contains(k) = false, want true")
	}
	sm.Remove(k)
	if sm.Contains(k) {
		t.Fatal("Contains(k) = true after remove, want false")
	}
}

func TestSlotMap_RemoveInvalidKey(t *testing.T) {
	sm := lib.NewSlotMap[int]()
	_, ok := sm.Remove(lib.SlotMapKey{Index: 999, Generation: 0})
	if ok {
		t.Fatal("Remove of invalid key should return false")
	}
}

func TestSlotMap_WithCapacity(t *testing.T) {
	sm := lib.NewSlotMapWithCapacity[int](100)
	if sm.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", sm.Len())
	}
	k := sm.Insert(7)
	if v := sm.Get(k); v == nil || *v != 7 {
		t.Fatalf("Get(k) = %v, want 7", v)
	}
}

func TestSlotMap_ManyInsertRemove(t *testing.T) {
	sm := lib.NewSlotMap[int]()
	keys := make([]lib.SlotMapKey, 1000)
	for i := range keys {
		keys[i] = sm.Insert(i)
	}
	if sm.Len() != 1000 {
		t.Fatalf("Len() = %d, want 1000", sm.Len())
	}

	// Remove even-indexed keys
	for i := 0; i < 1000; i += 2 {
		sm.Remove(keys[i])
	}
	if sm.Len() != 500 {
		t.Fatalf("Len() = %d, want 500", sm.Len())
	}

	// Re-insert 500 — should reuse freed slots
	for i := 0; i < 500; i++ {
		sm.Insert(i + 2000)
	}
	if sm.Len() != 1000 {
		t.Fatalf("Len() = %d, want 1000", sm.Len())
	}

	// Odd keys should still be valid
	for i := 1; i < 1000; i += 2 {
		if v := sm.Get(keys[i]); v == nil || *v != i {
			t.Fatalf("Get(keys[%d]) = %v, want %d", i, v, i)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./lib/ -run TestSlotMap -v`
Expected: compilation error — `lib.SlotMapKey` and `lib.NewSlotMap` not defined

- [ ] **Step 3: Implement lib.SlotMap**

Create `lib/slotmap.go`:

```go
package lib

// SlotMapKey is a versioned handle into a SlotMap.
// Index identifies the slot; Generation guards against use-after-free.
type SlotMapKey struct {
	Index      uint32
	Generation uint32
}

type slot[V any] struct {
	value      V
	generation uint32
	occupied   bool
}

// SlotMap is a dense, generational arena. Lookup is O(1) via array index
// with a generation check to detect stale keys.
type SlotMap[V any] struct {
	slots    []slot[V]
	freeList []uint32
	count    int
}

// NewSlotMap creates an empty SlotMap.
func NewSlotMap[V any]() SlotMap[V] {
	return SlotMap[V]{}
}

// NewSlotMapWithCapacity creates an empty SlotMap with pre-allocated capacity.
func NewSlotMapWithCapacity[V any](capacity int) SlotMap[V] {
	return SlotMap[V]{
		slots:    make([]slot[V], 0, capacity),
		freeList: make([]uint32, 0, capacity/4),
	}
}

// Insert adds a value and returns its key.
func (s *SlotMap[V]) Insert(value V) SlotMapKey {
	if len(s.freeList) > 0 {
		idx := s.freeList[len(s.freeList)-1]
		s.freeList = s.freeList[:len(s.freeList)-1]
		sl := &s.slots[idx]
		sl.value = value
		sl.occupied = true
		s.count++
		return SlotMapKey{Index: idx, Generation: sl.generation}
	}
	gen := uint32(0)
	s.slots = append(s.slots, slot[V]{value: value, generation: gen, occupied: true})
	s.count++
	return SlotMapKey{Index: uint32(len(s.slots) - 1), Generation: gen}
}

// Get returns a pointer to the value for the given key, or nil if the key is
// invalid or has been removed.
func (s *SlotMap[V]) Get(key SlotMapKey) *V {
	if int(key.Index) >= len(s.slots) {
		return nil
	}
	sl := &s.slots[key.Index]
	if !sl.occupied || sl.generation != key.Generation {
		return nil
	}
	return &sl.value
}

// Remove removes the value associated with the key and returns it.
// Returns the zero value and false if the key is invalid.
func (s *SlotMap[V]) Remove(key SlotMapKey) (V, bool) {
	if int(key.Index) >= len(s.slots) {
		var zero V
		return zero, false
	}
	sl := &s.slots[key.Index]
	if !sl.occupied || sl.generation != key.Generation {
		var zero V
		return zero, false
	}
	val := sl.value
	var zero V
	sl.value = zero // clear to allow GC
	sl.occupied = false
	sl.generation++
	s.freeList = append(s.freeList, key.Index)
	s.count--
	return val, true
}

// Len returns the number of occupied slots.
func (s *SlotMap[V]) Len() int {
	return s.count
}

// Contains reports whether the key points to a live slot.
func (s *SlotMap[V]) Contains(key SlotMapKey) bool {
	return s.Get(key) != nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./lib/ -run TestSlotMap -v`
Expected: all 7 tests PASS

- [ ] **Step 5: Commit**

```bash
git add lib/slotmap.go lib/slotmap_test.go
git commit -m "feat: add lib.SlotMap generic generational arena"
```

---

### Task 2: TaffyTree Migration to SlotMap

**Files:**
- Modify: `tree/node.go` — change `NodeId` to `lib.SlotMapKey`
- Modify: `taffy_tree.go` — replace maps with SlotMaps
- Modify: `taffy_view.go` — update all map accesses to `SlotMap.Get()`

**Important:** `NodeId` is used throughout the entire codebase. Changing it to `lib.SlotMapKey` is a mechanical but wide-reaching change. The key insight is that `NodeId` was `type NodeId uint64` and is now `type NodeId = lib.SlotMapKey` (type alias). All existing code that passes `NodeId` around stays the same — only code that creates or compares `NodeId` values changes.

- [ ] **Step 1: Change NodeId type**

Edit `tree/node.go` to:

```go
package tree

import "taffy-go/lib"

// NodeId is a type representing the id of a single node in a tree of nodes.
// It is backed by a SlotMapKey for O(1) array-indexed lookup.
type NodeId = lib.SlotMapKey
```

Remove `invalidNodeId` and `NewNodeId` — they are unused (verify by grepping first).

- [ ] **Step 2: Update TaffyTree struct and constructors**

Edit `taffy_tree.go` — replace the struct and constructors:

```go
type TaffyTree struct {
	nodes        lib.SlotMap[nodeData]
	nodeContexts lib.SlotMap[NodeContext]
	children     lib.SlotMap[[]tree.NodeId]
	parents      lib.SlotMap[lib.Option[tree.NodeId]]
	config       taffyConfig
}
```

Update `newNodeData` to return by value (not pointer):

```go
func newNodeData(s style.Style) nodeData {
	return nodeData{
		style:           s,
		cache:           tree.NewCache(),
		unroundedLayout: tree.NewLayout(),
		finalLayout:     tree.NewLayout(),
		hasNodeContext:   false,
	}
}
```

Update `New()` and `NewWithCapacity()`:

```go
func New() *TaffyTree {
	return NewWithCapacity(16)
}

func NewWithCapacity(capacity int) *TaffyTree {
	return &TaffyTree{
		nodes:        lib.NewSlotMapWithCapacity[nodeData](capacity),
		nodeContexts: lib.NewSlotMapWithCapacity[NodeContext](capacity),
		children:     lib.NewSlotMapWithCapacity[[]tree.NodeId](capacity),
		parents:      lib.NewSlotMapWithCapacity[lib.Option[tree.NodeId]](capacity),
		config:       defaultTaffyConfig(),
	}
}
```

- [ ] **Step 3: Update all TaffyTree methods**

Replace every `map[tree.NodeId]X` access pattern. The key changes:

**`allocateId` is removed** — `SlotMap.Insert` returns the key directly.

**`NewLeaf`:**
```go
func (t *TaffyTree) NewLeaf(s style.Style) tree.NodeId {
	id := t.nodes.Insert(newNodeData(s))
	t.nodeContexts.Insert(nil)
	t.children.Insert([]tree.NodeId{})
	t.parents.Insert(lib.Option[tree.NodeId]{})
	return id
}
```

Note: All four SlotMaps must insert in lockstep so they share the same index. This is the simplest approach — an alternative would be storing everything in a single SlotMap with a combined struct. For this migration, use lockstep insertion (the four SlotMaps always have the same set of occupied indices).

**`NewLeafWithContext`:**
```go
func (t *TaffyTree) NewLeafWithContext(s style.Style, context NodeContext) tree.NodeId {
	data := newNodeData(s)
	data.hasNodeContext = true
	id := t.nodes.Insert(data)
	t.nodeContexts.Insert(context)
	t.children.Insert([]tree.NodeId{})
	t.parents.Insert(lib.Option[tree.NodeId]{})
	return id
}
```

**`NewWithChildren`:**
```go
func (t *TaffyTree) NewWithChildren(s style.Style, children []tree.NodeId) tree.NodeId {
	id := t.nodes.Insert(newNodeData(s))
	childrenCopy := make([]tree.NodeId, len(children))
	copy(childrenCopy, children)
	t.children.Insert(childrenCopy)
	t.parents.Insert(lib.Option[tree.NodeId]{})
	// Pad nodeContexts to keep in lockstep
	t.nodeContexts.Insert(nil)
	for _, child := range children {
		if p := t.parents.Get(child); p != nil {
			*p = lib.Some(id)
		}
	}
	return id
}
```

**`requireNode`:**
```go
func (t *TaffyTree) requireNode(node tree.NodeId) {
	if !t.nodes.Contains(node) {
		panic(fmt.Sprintf("taffy: node {%d,%d} is not in the TaffyTree", node.Index, node.Generation))
	}
}
```

**All `t.nodes[id]` becomes `t.nodes.Get(id)`**, `t.children[id]` becomes `*t.children.Get(id)`, etc.

**`Remove`:**
```go
func (t *TaffyTree) Remove(node tree.NodeId) tree.NodeId {
	t.requireNode(node)
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
```

**`Clear`:**
```go
func (t *TaffyTree) Clear() {
	t.nodes = lib.NewSlotMap[nodeData]()
	t.nodeContexts = lib.NewSlotMap[NodeContext]()
	t.children = lib.NewSlotMap[[]tree.NodeId]()
	t.parents = lib.NewSlotMap[lib.Option[tree.NodeId]]()
}
```

**`TotalNodeCount`:**
```go
func (t *TaffyTree) TotalNodeCount() int {
	return t.nodes.Len()
}
```

**`MarkDirty`** — replace `t.nodes[current]` with `t.nodes.Get(current)` and `t.parents[current]` with `*t.parents.Get(current)`:
```go
func (t *TaffyTree) MarkDirty(node tree.NodeId) {
	t.requireNode(node)
	current := node
	for {
		nd := t.nodes.Get(current)
		state := nd.markDirty()
		if state == tree.ClearStateAlreadyEmpty {
			break
		}
		parentOpt := t.parents.Get(current)
		if parentOpt == nil || !parentOpt.IsSome {
			break
		}
		current = parentOpt.Value
	}
}
```

Apply the same pattern to every remaining method (`SetNodeContext`, `GetNodeContext`, `AddChild`, `InsertChildAtIndex`, `SetChildren`, `RemoveChild`, `RemoveChildAtIndex`, `ReplaceChildAtIndex`, `ChildAtIndex`, `Parent`, `Children`, `SetStyle`, `GetStyle`, `GetLayout`, `IsDirty`).

- [ ] **Step 4: Update taffyView**

Every `v.taffy.nodes[nodeId]` becomes `v.taffy.nodes.Get(nodeId)`, every `v.taffy.children[nodeId]` becomes `*v.taffy.children.Get(nodeId)`.

Key methods in `taffy_view.go`:

```go
func (v *taffyView) ChildIds(parent tree.NodeId) []tree.NodeId {
	return *v.taffy.children.Get(parent)
}

func (v *taffyView) ChildCount(parent tree.NodeId) int {
	return len(*v.taffy.children.Get(parent))
}

func (v *taffyView) GetChildId(parent tree.NodeId, index int) tree.NodeId {
	return (*v.taffy.children.Get(parent))[index]
}

func (v *taffyView) GetCoreContainerStyle(nodeId tree.NodeId) style.CoreStyle {
	return &v.taffy.nodes.Get(nodeId).style
}

func (v *taffyView) SetUnroundedLayout(nodeId tree.NodeId, layout *tree.Layout) {
	v.taffy.nodes.Get(nodeId).unroundedLayout = *layout
}

func (v *taffyView) CacheGet(nodeId tree.NodeId, input *tree.LayoutInput) (tree.LayoutOutput, bool) {
	return v.taffy.nodes.Get(nodeId).cache.Get(input)
}

func (v *taffyView) CacheStore(nodeId tree.NodeId, input *tree.LayoutInput, output tree.LayoutOutput) {
	v.taffy.nodes.Get(nodeId).cache.Store(input, output)
}

func (v *taffyView) CacheClear(nodeId tree.NodeId) {
	v.taffy.nodes.Get(nodeId).cache.Clear()
}

func (v *taffyView) GetUnroundedLayout(nodeId tree.NodeId) *tree.Layout {
	return &v.taffy.nodes.Get(nodeId).unroundedLayout
}

func (v *taffyView) SetFinalLayout(nodeId tree.NodeId, layout *tree.Layout) {
	v.taffy.nodes.Get(nodeId).finalLayout = *layout
}

func (v *taffyView) GetFinalLayout(nodeId tree.NodeId) *tree.Layout {
	return &v.taffy.nodes.Get(nodeId).finalLayout
}

func (v *taffyView) GetFlexboxContainerStyle(nodeId tree.NodeId) style.FlexboxContainerStyle {
	return &v.taffy.nodes.Get(nodeId).style
}

func (v *taffyView) GetFlexboxChildStyle(childNodeId tree.NodeId) style.FlexboxItemStyle {
	return &v.taffy.nodes.Get(childNodeId).style
}

func (v *taffyView) GetGridContainerStyle(nodeId tree.NodeId) style.GridContainerStyle {
	return &v.taffy.nodes.Get(nodeId).style
}

func (v *taffyView) GetGridChildStyle(childNodeId tree.NodeId) style.GridItemStyle {
	return &v.taffy.nodes.Get(childNodeId).style
}

func (v *taffyView) GetBlockContainerStyle(nodeId tree.NodeId) style.BlockContainerStyle {
	return &v.taffy.nodes.Get(nodeId).style
}

func (v *taffyView) GetBlockChildStyle(childNodeId tree.NodeId) style.BlockItemStyle {
	return &v.taffy.nodes.Get(childNodeId).style
}
```

Also update `measureFunc`, `ComputeChildLayout`, `ComputeBlockChildLayout`, and `GetDebugLabel` using the same pattern.

- [ ] **Step 5: Fix compilation errors**

Run: `go build ./...`

The `NodeId` type change from `uint64` to `lib.SlotMapKey` will cause compilation errors anywhere that:
- Compares `NodeId` with `==` or `!=` — struct comparison still works, no change needed
- Uses `NodeId` as a map key — any remaining maps keyed by `NodeId` need updating (should be none after this task)
- Casts `NodeId` to/from `uint64` — find and fix these

Fix each error. Common patterns:
- `tree.NodeId(t.nextId)` in `allocateId` — removed entirely
- `fmt.Sprintf("node %v", node)` — use `fmt.Sprintf("node {%d,%d}", node.Index, node.Generation)`

- [ ] **Step 6: Run all tests**

Run: `go test ./... -count=1`
Expected: all tests PASS

- [ ] **Step 7: Commit**

```bash
git add tree/node.go taffy_tree.go taffy_view.go
git commit -m "perf: migrate TaffyTree from hash maps to SlotMap"
```

---

### Task 3: Apply lib.SortStableByIndex to Grid Sort

**Files:**
- Modify: `compute/grid/track_sizing.go:392-396`

- [ ] **Step 1: Run grid tests to confirm baseline**

Run: `go test ./... -run 'Grid' -count=1`
Expected: all PASS

- [ ] **Step 2: Replace sort call**

In `compute/grid/track_sizing.go`, replace the existing sort block (around line 392-396):

```go
// Before:
cmpFn := cmpByCrossFlexThenSpanThenStart(axis)
slices.SortStableFunc(items, func(a, b gridItem) int {
    return cmpFn(&a, &b)
})

// After:
cmpFn := cmpByCrossFlexThenSpanThenStart(axis)
lib.SortStableByIndex(items, cmpFn)
```

Remove the `"slices"` import if it's no longer used elsewhere in this file. Keep the `"taffy-go/lib"` import (already present for `lib.Option`).

Note: `cmpFn` already has the signature `func(a, b *gridItem) int` — it matches `lib.SortStableByIndex`'s `cmp func(a, b *E) int` parameter directly.

- [ ] **Step 3: Also remove unused import if needed**

Check if `slices` is still used elsewhere in `track_sizing.go`. If not, remove the import. If the `cmp` package is also unused, remove it too.

- [ ] **Step 4: Run all tests**

Run: `go test ./... -count=1`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add compute/grid/track_sizing.go
git commit -m "perf: use lib.SortStableByIndex for grid item sort (3.8x faster)"
```

---

### Task 4: Layout Value Semantics

**Files:**
- Modify: `tree/traits.go:41,61` — change signatures
- Modify: `taffy_view.go:47-48,130-131` — update implementation
- Modify: `compute/compute.go:87,107` — remove `&`
- Modify: `compute/flexbox/flex_item.go:508` — remove `&`
- Modify: `compute/flexbox/flexbox.go:652,1442` — remove `&`
- Modify: `compute/block/block.go:559,858,1099,1449` — remove `&`
- Modify: `compute/grid/alignment.go:297` — remove `&`
- Modify: `compute/grid/grid.go:541` — remove `&`
- Modify: `taffy_view.go:363` — remove `&` in `computeRootLayout`

- [ ] **Step 1: Change interface signatures**

In `tree/traits.go`, change:

```go
// Before:
SetUnroundedLayout(nodeId NodeId, layout *Layout)
```
to:
```go
SetUnroundedLayout(nodeId NodeId, layout Layout)
```

And:
```go
// Before:
SetFinalLayout(nodeId NodeId, layout *Layout)
```
to:
```go
SetFinalLayout(nodeId NodeId, layout Layout)
```

- [ ] **Step 2: Update taffyView implementation**

In `taffy_view.go`, change:

```go
// Before:
func (v *taffyView) SetUnroundedLayout(nodeId tree.NodeId, layout *tree.Layout) {
	v.taffy.nodes.Get(nodeId).unroundedLayout = *layout
}
```
to:
```go
func (v *taffyView) SetUnroundedLayout(nodeId tree.NodeId, layout tree.Layout) {
	v.taffy.nodes.Get(nodeId).unroundedLayout = layout
}
```

And:
```go
// Before:
func (v *taffyView) SetFinalLayout(nodeId tree.NodeId, layout *tree.Layout) {
	v.taffy.nodes.Get(nodeId).finalLayout = *layout
}
```
to:
```go
func (v *taffyView) SetFinalLayout(nodeId tree.NodeId, layout tree.Layout) {
	v.taffy.nodes.Get(nodeId).finalLayout = layout
}
```

- [ ] **Step 3: Update all callers — remove `&` from Layout literals**

In every file that calls `SetUnroundedLayout` or `SetFinalLayout`, remove the `&` prefix from the `tree.Layout{...}` argument. The full list:

`compute/compute.go:87`:
```go
// Before:
t.SetFinalLayout(nodeId, &layout)
// After:
t.SetFinalLayout(nodeId, layout)
```

`compute/compute.go:107`:
```go
// Before:
t.SetUnroundedLayout(node, &tree.Layout{Order: 0})
// After:
t.SetUnroundedLayout(node, tree.Layout{Order: 0})
```

`compute/flexbox/flex_item.go:508`:
```go
// Before:
t.SetUnroundedLayout(item.node, &tree.Layout{
// After:
t.SetUnroundedLayout(item.node, tree.Layout{
```

`compute/flexbox/flexbox.go:652`:
```go
// Before:
t.SetUnroundedLayout(child, &tree.Layout{Order: uint32(order)})
// After:
t.SetUnroundedLayout(child, tree.Layout{Order: uint32(order)})
```

`compute/flexbox/flexbox.go:1442`:
```go
// Before:
t.SetUnroundedLayout(child, &tree.Layout{
// After:
t.SetUnroundedLayout(child, tree.Layout{
```

`compute/block/block.go:559`:
```go
// Before:
t.SetUnroundedLayout(child, &tree.Layout{Order: uint32(order)})
// After:
t.SetUnroundedLayout(child, tree.Layout{Order: uint32(order)})
```

`compute/block/block.go:858`:
```go
// Before:
t.SetUnroundedLayout(item.nodeID, &tree.Layout{
// After:
t.SetUnroundedLayout(item.nodeID, tree.Layout{
```

`compute/block/block.go:1099`:
```go
// Before:
t.SetUnroundedLayout(item.nodeID, &tree.Layout{
// After:
t.SetUnroundedLayout(item.nodeID, tree.Layout{
```

`compute/block/block.go:1449`:
```go
// Before:
t.SetUnroundedLayout(item.nodeID, &tree.Layout{
// After:
t.SetUnroundedLayout(item.nodeID, tree.Layout{
```

`compute/grid/alignment.go:297`:
```go
// Before:
t.SetUnroundedLayout(node, &tree.Layout{
// After:
t.SetUnroundedLayout(node, tree.Layout{
```

`compute/grid/grid.go:541`:
```go
// Before:
t.SetUnroundedLayout(child, &tree.Layout{Order: order})
// After:
t.SetUnroundedLayout(child, tree.Layout{Order: order})
```

`taffy_view.go:363` (in `computeRootLayout`):
```go
// Before:
t.SetUnroundedLayout(root, &tree.Layout{
// After:
t.SetUnroundedLayout(root, tree.Layout{
```

- [ ] **Step 4: Run all tests**

Run: `go test ./... -count=1`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add tree/traits.go taffy_view.go compute/compute.go compute/flexbox/flex_item.go compute/flexbox/flexbox.go compute/block/block.go compute/grid/alignment.go compute/grid/grid.go
git commit -m "perf: Layout value semantics for SetUnroundedLayout/SetFinalLayout"
```

---

### Task 5: Flatten cellOccupancyMatrix

**Files:**
- Modify: `compute/grid/types_occupancy.go`

- [ ] **Step 1: Run grid tests to confirm baseline**

Run: `go test ./... -run 'Grid' -count=1`
Expected: all PASS

- [ ] **Step 2: Change struct and constructor**

In `compute/grid/types_occupancy.go`, replace the struct and `newCellOccupancyMatrix`:

```go
type cellOccupancyMatrix struct {
	inner    []cellOccupancyState
	colCount int
	columns  TrackCounts
	rows     TrackCounts
}

func newCellOccupancyMatrix(columns, rows TrackCounts) cellOccupancyMatrix {
	rowCount := rows.Len()
	colCount := columns.Len()
	return cellOccupancyMatrix{
		inner:    make([]cellOccupancyState, rowCount*colCount),
		colCount: colCount,
		columns:  columns,
		rows:     rows,
	}
}
```

- [ ] **Step 3: Update matrixRows and matrixCols**

```go
func (m *cellOccupancyMatrix) matrixRows() int {
	if m.colCount == 0 {
		return 0
	}
	return len(m.inner) / m.colCount
}

func (m *cellOccupancyMatrix) matrixCols() int {
	return m.colCount
}
```

- [ ] **Step 4: Update expandToFitRange**

This is the most complex method. Replace with flat-slice expansion:

```go
func (m *cellOccupancyMatrix) expandToFitRange(rowStart, rowEnd, colStart, colEnd int16) {
	reqNegativeRows := max16(0, -rowStart)
	reqPositiveRows := max16(0, rowEnd-int16(m.rows.Len()))
	reqNegativeCols := max16(0, -colStart)
	reqPositiveCols := max16(0, colEnd-int16(m.columns.Len()))

	oldRowCount := m.matrixRows()
	oldColCount := m.colCount
	newRowCount := oldRowCount + int(reqNegativeRows) + int(reqPositiveRows)
	newColCount := oldColCount + int(reqNegativeCols) + int(reqPositiveCols)

	newInner := make([]cellOccupancyState, newRowCount*newColCount)

	// Copy existing data into new grid, offset by new negative rows/cols
	for r := 0; r < oldRowCount; r++ {
		for c := 0; c < oldColCount; c++ {
			newRow := r + int(reqNegativeRows)
			newCol := c + int(reqNegativeCols)
			newInner[newRow*newColCount+newCol] = m.inner[r*oldColCount+c]
		}
	}

	m.inner = newInner
	m.colCount = newColCount
	m.rows.NegativeImplicit += uint16(reqNegativeRows)
	m.rows.PositiveImplicit += uint16(reqPositiveRows)
	m.columns.NegativeImplicit += uint16(reqNegativeCols)
	m.columns.PositiveImplicit += uint16(reqPositiveCols)
}
```

- [ ] **Step 5: Update markAreaAs**

Replace `m.inner[int(x)][int(y)]` with flat access:

```go
for x := rowStart; x < rowEnd; x++ {
	for y := colStart; y < colEnd; y++ {
		m.inner[int(x)*m.colCount+int(y)] = value
	}
}
```

- [ ] **Step 6: Update trackAreaIsUnoccupied**

Replace `m.inner[int(x)][int(y)]` with flat access:

```go
for x := rowStart; x < rowEnd; x++ {
	for y := colStart; y < colEnd; y++ {
		if int(x) < 0 || int(x) >= rows || int(y) < 0 || int(y) >= cols {
			continue
		}
		if m.inner[int(x)*m.colCount+int(y)] != cellUnoccupied {
			return false
		}
	}
}
```

- [ ] **Step 7: Update rowIsOccupied and columnIsOccupied**

```go
func (m *cellOccupancyMatrix) rowIsOccupied(rowIndex int) bool {
	if rowIndex >= m.matrixRows() {
		return false
	}
	base := rowIndex * m.colCount
	for c := 0; c < m.colCount; c++ {
		if m.inner[base+c] != cellUnoccupied {
			return true
		}
	}
	return false
}

func (m *cellOccupancyMatrix) columnIsOccupied(columnIndex int) bool {
	if columnIndex >= m.colCount {
		return false
	}
	rows := m.matrixRows()
	for r := 0; r < rows; r++ {
		if m.inner[r*m.colCount+columnIndex] != cellUnoccupied {
			return true
		}
	}
	return false
}
```

- [ ] **Step 8: Update lastOfType and firstOfType**

For `lastOfType`, replace `m.inner[int(trackComputedIndex)]` row access with stride arithmetic:

In the `AbsoluteAxisHorizontal` case:
```go
rowBase := int(trackComputedIndex) * m.colCount
for i := m.colCount - 1; i >= 0; i-- {
	if m.inner[rowBase+i] == kind {
		foundIndex = i
		found = true
		break
	}
}
```

In the `AbsoluteAxisVertical` case:
```go
col := int(trackComputedIndex)
rows := m.matrixRows()
for i := rows - 1; i >= 0; i-- {
	if m.inner[i*m.colCount+col] == kind {
		foundIndex = i
		found = true
		break
	}
}
```

Apply the same pattern for `firstOfType` (iterating forward instead of backward).

- [ ] **Step 9: Run all tests**

Run: `go test ./... -count=1`
Expected: all PASS

- [ ] **Step 10: Commit**

```bash
git add compute/grid/types_occupancy.go
git commit -m "perf: flatten cellOccupancyMatrix to single contiguous slice"
```

---

### Task 6: itemBatcher Returns Sub-slice

**Files:**
- Modify: `compute/grid/track_sizing.go`

- [ ] **Step 1: Change itemBatcher.next signature and implementation**

In `compute/grid/track_sizing.go`, replace `next`:

```go
// next returns the next batch of items as a sub-slice and whether they are flex items.
// Returns an empty slice and false when iteration is complete.
func (b *itemBatcher) next(items []gridItem) (batch []gridItem, isFlex bool) {
	if b.currentIsFlex || b.indexOffset >= len(items) {
		return nil, false
	}

	item := &items[b.indexOffset]
	b.currentSpan = item.span(b.axis)
	b.currentIsFlex = item.crossesFlexibleTrack(b.axis)

	var nextIndexOffset int
	if b.currentIsFlex {
		nextIndexOffset = len(items)
	} else {
		nextIndexOffset = len(items)
		for i := b.indexOffset; i < len(items); i++ {
			if items[i].crossesFlexibleTrack(b.axis) || items[i].span(b.axis) > b.currentSpan {
				nextIndexOffset = i
				break
			}
		}
	}

	batch = items[b.indexOffset:nextIndexOffset]
	isFlex = b.currentIsFlex
	b.indexOffset = nextIndexOffset
	return batch, isFlex
}
```

- [ ] **Step 2: Update all call sites in resolveIntrinsicTrackSizes**

Replace every `batchIndices` reference. The variable name changes from `batchIndices` to `batch`. Every `for _, idx := range batchIndices { item := &items[idx]` becomes `for i := range batch { item := &batch[i]`.

At the start of the loop (around line 409):
```go
// Before:
batchIndices, isFlex := batcher.next(items)
if batchIndices == nil {
    break
}
batchSpan := items[batchIndices[0]].span(axis)

// After:
batch, isFlex := batcher.next(items)
if len(batch) == 0 {
    break
}
batchSpan := batch[0].span(axis)
```

Then for every iteration block, change:
```go
// Before:
for _, idx := range batchIndices {
    item := &items[idx]

// After:
for i := range batch {
    item := &batch[i]
```

There are 8 such loops to update (lines approximately: 418, 509, 556, 582, 617, 643, 661, plus the span==1 block at 418).

- [ ] **Step 3: Run all tests**

Run: `go test ./... -count=1`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add compute/grid/track_sizing.go
git commit -m "perf: itemBatcher returns sub-slice instead of allocating index array"
```

---

### Task 7: Benchmark and Record Results

**Files:**
- Create: `bench_after_phase4.txt`

- [ ] **Step 1: Run full benchmark suite**

```bash
go test -run='^$' -bench='Benchmark' -benchmem -count=5 -timeout=300s 2>&1 | tee bench_after_phase4.txt
```

- [ ] **Step 2: Compare with previous results**

Compare key metrics from `bench_after_task5.txt` (Phase 3 final) vs `bench_after_phase4.txt`. Compute the improvement percentages for ns/op, B/op, and allocs/op for each benchmark.

- [ ] **Step 3: Commit benchmark results**

```bash
git add bench_after_phase4.txt
git commit -m "bench: record Phase 4 optimization results"
```
