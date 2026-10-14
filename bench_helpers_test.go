package taffy

import (
	"math/rand/v2"
	"testing"

	"github.com/gataengine/taffy-go/geometry"
	"github.com/gataengine/taffy-go/style"
	"github.com/gataengine/taffy-go/tree"
)

// styleGen generates styles for benchmark tree nodes.
type styleGen interface {
	leafStyle(rng *rand.Rand) style.Style
	containerStyle(rng *rand.Rand) style.Style
}

// fixedStyleGen returns the same style for all nodes.
type fixedStyleGen struct {
	style style.Style
}

func (f fixedStyleGen) leafStyle(_ *rand.Rand) style.Style      { return f.style }
func (f fixedStyleGen) containerStyle(_ *rand.Rand) style.Style { return f.style }

// randomDimension returns a random Dimension: 20% auto, 60% length(0-500), 20% percent(0-1).
func randomDimension(rng *rand.Rand) style.Dimension {
	r := rng.Float32()
	switch {
	case r < 0.2:
		return style.AutoDimension()
	case r < 0.8:
		return style.LengthDimension(rng.Float32() * 500.0)
	default:
		return style.PercentDimension(rng.Float32())
	}
}

// randomStyleGen generates random dimensions for each node.
type randomStyleGen struct{}

func (randomStyleGen) leafStyle(rng *rand.Rand) style.Style {
	s := style.DefaultStyle()
	s.Size = geometry.Size[style.Dimension]{
		Width:  randomDimension(rng),
		Height: randomDimension(rng),
	}
	return s
}

func (randomStyleGen) containerStyle(rng *rand.Rand) style.Style {
	s := style.DefaultStyle()
	s.Size = geometry.Size[style.Dimension]{
		Width:  randomDimension(rng),
		Height: randomDimension(rng),
	}
	return s
}

const benchRNGSeed = 12345

// newBenchRNG creates a deterministic RNG for benchmarks.
func newBenchRNG() *rand.Rand {
	return rand.New(rand.NewPCG(benchRNGSeed, 0))
}

// buildDeepTree builds a recursive balanced tree with the given branching factor.
// Returns the child node IDs for the caller to attach to a parent.
func buildDeepTree(t *TaffyTree, maxNodes, branchingFactor uint32, gen styleGen, rng *rand.Rand) []tree.NodeId {
	if maxNodes <= branchingFactor {
		nodes := make([]tree.NodeId, maxNodes)
		for i := range nodes {
			nodes[i] = t.NewLeaf(gen.leafStyle(rng))
		}
		return nodes
	}
	childMaxNodes := (maxNodes - branchingFactor) / branchingFactor
	nodes := make([]tree.NodeId, branchingFactor)
	for i := range nodes {
		children := buildDeepTree(t, childMaxNodes, branchingFactor, gen, rng)
		nodes[i] = t.NewWithChildren(gen.containerStyle(rng), children)
	}
	return nodes
}

// buildFlatHierarchy builds a 2-level hierarchy with random 1-4 leaf children per container.
func buildFlatHierarchy(t *TaffyTree, targetNodeCount uint32, gen styleGen, rng *rand.Rand) []tree.NodeId {
	var containers []tree.NodeId
	var count uint32
	for count < targetNodeCount {
		subCount := rng.IntN(4) + 1
		leaves := make([]tree.NodeId, subCount)
		for i := range leaves {
			leaves[i] = t.NewLeaf(gen.leafStyle(rng))
		}
		container := t.NewWithChildren(gen.containerStyle(rng), leaves)
		containers = append(containers, container)
		count += uint32(1 + subCount)
	}
	return containers
}

// buildSuperDeepHierarchy builds a linear chain of depth levels, each with nodesPerLevel nodes.
func buildSuperDeepHierarchy(t *TaffyTree, depth, nodesPerLevel uint32, gen styleGen, rng *rand.Rand) []tree.NodeId {
	children := make([]tree.NodeId, 0, nodesPerLevel)
	for level := uint32(0); level < depth; level++ {
		container := t.NewWithChildren(gen.containerStyle(rng), children)
		children = make([]tree.NodeId, 0, nodesPerLevel)
		children = append(children, container)
		for j := uint32(1); j < nodesPerLevel; j++ {
			children = append(children, t.NewLeaf(gen.leafStyle(rng)))
		}
	}
	return children
}

// benchmarkLayout is a helper that builds a tree and benchmarks layout computation.
// buildFn receives a fresh TaffyTree and RNG, and returns the root node ID.
func benchmarkLayout(b *testing.B, availableSpace geometry.Size[style.AvailableSpace], buildFn func(t *TaffyTree, rng *rand.Rand) tree.NodeId) {
	b.Helper()
	for b.Loop() {
		b.StopTimer()
		rng := newBenchRNG()
		t := New()
		root := buildFn(t, rng)
		b.StartTimer()
		t.ComputeLayout(root, availableSpace)
	}
}

// defaultAvailableSpace returns the default available space (max-content on both axes).
func defaultAvailableSpace() geometry.Size[style.AvailableSpace] {
	return geometry.Size[style.AvailableSpace]{
		Width:  style.AvailableSpaceMaxContentValue,
		Height: style.AvailableSpaceMaxContentValue,
	}
}

// lengthAvailableSpace returns definite available space on both axes.
func lengthAvailableSpace(width, height float32) geometry.Size[style.AvailableSpace] {
	return geometry.Size[style.AvailableSpace]{
		Width:  style.AvailableSpaceFromLength(width),
		Height: style.AvailableSpaceFromLength(height),
	}
}
