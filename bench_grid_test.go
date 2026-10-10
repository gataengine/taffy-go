package taffy

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/gataengine/taffy-go/geometry"
	"github.com/gataengine/taffy-go/style"
	"github.com/gataengine/taffy-go/tree"
)

// randomGridTrack returns a random grid track sizing function.
// Distribution: 10% auto, 10% min-content, 10% max-content, 20% fr(1),
// 10% minmax(0, fr(1)), 20% length(40), 20% percent(0.3).
func randomGridTrack(rng *rand.Rand) style.GridTemplateComponent {
	r := rng.Float32()
	var tsf style.TrackSizingFunction
	switch {
	case r < 0.1:
		tsf = style.TrackSizingAuto
	case r < 0.2:
		tsf = style.TrackSizingFunction{Min: style.MinTrackMinContentVal, Max: style.MaxTrackMinContentVal}
	case r < 0.3:
		tsf = style.TrackSizingFunction{Min: style.MinTrackMaxContentVal, Max: style.MaxTrackMaxContentVal}
	case r < 0.5:
		tsf = style.GridFr(1.0)
	case r < 0.6:
		tsf = style.GridMinMax(
			style.MinTrackSizingFunction{Tag: style.MinTrackLength, Value: 0},
			style.MaxTrackFromFr(1.0),
		)
	case r < 0.8:
		tsf = style.TrackSizingFunction{
			Min: style.MinTrackSizingFunction{Tag: style.MinTrackLength, Value: 40},
			Max: style.MaxTrackFromLength(40),
		}
	default:
		tsf = style.TrackSizingFunction{
			Min: style.MinTrackSizingFunction{Tag: style.MinTrackPercent, Value: 0.3},
			Max: style.MaxTrackFromPercent(0.3),
		}
	}
	return style.GridTemplateSingleVal(tsf)
}

// randomNxNGridStyle creates a grid style with trackCount random columns and rows.
func randomNxNGridStyle(rng *rand.Rand, trackCount int) style.Style {
	s := style.DefaultStyle()
	s.Display = style.DisplayGrid
	cols := make([]style.GridTemplateComponent, trackCount)
	rows := make([]style.GridTemplateComponent, trackCount)
	for i := 0; i < trackCount; i++ {
		cols[i] = randomGridTrack(rng)
		rows[i] = randomGridTrack(rng)
	}
	s.GridTemplateColumns = cols
	s.GridTemplateRows = rows
	return s
}

// buildGridLeaf creates a leaf node with size 20x20.
func buildGridLeaf(t *TaffyTree) tree.NodeId {
	s := style.DefaultStyle()
	s.Size = geometry.Size[style.Dimension]{
		Width:  style.LengthDimension(20),
		Height: style.LengthDimension(20),
	}
	return t.NewLeaf(s)
}

// buildGridFlatHierarchy creates a flat colCount x rowCount grid with leaf children.
func buildGridFlatHierarchy(colCount, rowCount int) (*TaffyTree, tree.NodeId) {
	t := New()
	rng := newBenchRNG()

	s := style.DefaultStyle()
	s.Display = style.DisplayGrid
	cols := make([]style.GridTemplateComponent, colCount)
	rows := make([]style.GridTemplateComponent, rowCount)
	for i := 0; i < colCount; i++ {
		cols[i] = randomGridTrack(rng)
	}
	for i := 0; i < rowCount; i++ {
		rows[i] = randomGridTrack(rng)
	}
	s.GridTemplateColumns = cols
	s.GridTemplateRows = rows

	children := make([]tree.NodeId, colCount*rowCount)
	for i := range children {
		children[i] = buildGridLeaf(t)
	}
	root := t.NewWithChildren(s, children)
	return t, root
}

// buildDeepGridTree recursively constructs nested grids.
func buildDeepGridTree(t *TaffyTree, levels, trackCount int, rng *rand.Rand) []tree.NodeId {
	childCount := trackCount * trackCount
	if levels == 1 {
		nodes := make([]tree.NodeId, childCount)
		for i := range nodes {
			nodes[i] = buildGridLeaf(t)
		}
		return nodes
	}
	nodes := make([]tree.NodeId, childCount)
	for i := range nodes {
		subChildren := buildDeepGridTree(t, levels-1, trackCount, rng)
		nodes[i] = t.NewWithChildren(randomNxNGridStyle(rng, trackCount), subChildren)
	}
	return nodes
}

// buildDeepGridHierarchy creates a deep grid tree and returns (tree, root).
func buildDeepGridHierarchy(levels, trackCount int) (*TaffyTree, tree.NodeId) {
	t := New()
	rng := newBenchRNG()
	children := buildDeepGridTree(t, levels, trackCount, rng)
	root := t.NewWithChildren(style.DefaultStyle(), children)
	return t, root
}

func BenchmarkGrid(b *testing.B) {
	b.Run("Wide", benchGridWide)
	b.Run("Deep", benchGridDeep)
	b.Run("SuperDeep", benchGridSuperDeep)
}

func benchGridWide(b *testing.B) {
	availableSpace := lengthAvailableSpace(12000, 12000)
	for _, trackCount := range []int{31, 100} {
		b.Run(fmt.Sprintf("%dx%d", trackCount, trackCount), func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()
				t, root := buildGridFlatHierarchy(trackCount, trackCount)
				b.StartTimer()
				t.ComputeLayout(root, availableSpace)
			}
		})
	}
}

func benchGridDeep(b *testing.B) {
	availableSpace := lengthAvailableSpace(12000, 12000)
	benches := []struct {
		tracks int
		levels int
	}{
		{2, 5},
		{3, 4},
		{2, 7},
	}
	for _, bench := range benches {
		name := fmt.Sprintf("%dx%d_levels%d", bench.tracks, bench.tracks, bench.levels)
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()
				t, root := buildDeepGridHierarchy(bench.levels, bench.tracks)
				b.StartTimer()
				t.ComputeLayout(root, availableSpace)
			}
		})
	}
}

func benchGridSuperDeep(b *testing.B) {
	availableSpace := defaultAvailableSpace()
	for _, levels := range []int{100, 1000} {
		b.Run(fmt.Sprintf("%d", levels), func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()
				t, root := buildDeepGridHierarchy(levels, 1)
				b.StartTimer()
				t.ComputeLayout(root, availableSpace)
			}
		})
	}
}
