package taffy

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/gataengine/taffy-go/geometry"
	"github.com/gataengine/taffy-go/style"
	"github.com/gataengine/taffy-go/tree"
)

func BenchmarkFlexbox(b *testing.B) {
	b.Run("HugeNested", benchFlexboxHugeNested)
	b.Run("Wide", benchFlexboxWide)
	b.Run("DeepRandom", benchFlexboxDeepRandom)
	b.Run("DeepAuto", benchFlexboxDeepAuto)
	b.Run("SuperDeep", benchFlexboxSuperDeep)
}

func benchFlexboxHugeNested(b *testing.B) {
	hugeNestedStyle := style.DefaultStyle()
	hugeNestedStyle.Size = geometry.Size[style.Dimension]{
		Width:  style.LengthDimension(10),
		Height: style.LengthDimension(10),
	}
	hugeNestedStyle.FlexGrow = 1.0
	gen := fixedStyleGen{style: hugeNestedStyle}

	for _, nodeCount := range []uint32{10_000} {
		b.Run(fmt.Sprintf("%d", nodeCount), func(b *testing.B) {
			benchmarkLayout(b, defaultAvailableSpace(), func(t *TaffyTree, rng *rand.Rand) tree.NodeId {
				children := buildDeepTree(t, nodeCount, 10, gen, rng)
				return t.NewWithChildren(style.DefaultStyle(), children)
			})
		})
	}
}

func benchFlexboxWide(b *testing.B) {
	for _, nodeCount := range []uint32{10_000} {
		b.Run(fmt.Sprintf("%d", nodeCount), func(b *testing.B) {
			benchmarkLayout(b, defaultAvailableSpace(), func(t *TaffyTree, rng *rand.Rand) tree.NodeId {
				children := buildFlatHierarchy(t, nodeCount, randomStyleGen{}, rng)
				return t.NewWithChildren(style.DefaultStyle(), children)
			})
		})
	}
}

func benchFlexboxDeepRandom(b *testing.B) {
	for _, nodeCount := range []uint32{4_000, 10_000} {
		b.Run(fmt.Sprintf("%d", nodeCount), func(b *testing.B) {
			benchmarkLayout(b, defaultAvailableSpace(), func(t *TaffyTree, rng *rand.Rand) tree.NodeId {
				children := buildDeepTree(t, nodeCount, 2, randomStyleGen{}, rng)
				return t.NewWithChildren(style.DefaultStyle(), children)
			})
		})
	}
}

func benchFlexboxDeepAuto(b *testing.B) {
	deepAutoStyle := style.DefaultStyle()
	deepAutoStyle.FlexGrow = 1.0
	deepAutoStyle.Margin = geometry.Rect[style.LengthPercentageAuto]{
		Left:   style.LengthLPA(10),
		Right:  style.LengthLPA(10),
		Top:    style.LengthLPA(10),
		Bottom: style.LengthLPA(10),
	}
	gen := fixedStyleGen{style: deepAutoStyle}

	for _, nodeCount := range []uint32{4_000, 10_000} {
		b.Run(fmt.Sprintf("%d", nodeCount), func(b *testing.B) {
			benchmarkLayout(b, defaultAvailableSpace(), func(t *TaffyTree, rng *rand.Rand) tree.NodeId {
				children := buildDeepTree(t, nodeCount, 2, gen, rng)
				return t.NewWithChildren(style.DefaultStyle(), children)
			})
		})
	}
}

func benchFlexboxSuperDeep(b *testing.B) {
	superDeepStyle := style.DefaultStyle()
	superDeepStyle.FlexDirection = style.FlexDirectionRow
	superDeepStyle.FlexGrow = 1.0
	superDeepStyle.Margin = geometry.Rect[style.LengthPercentageAuto]{
		Left:   style.LengthLPA(10),
		Right:  style.LengthLPA(10),
		Top:    style.LengthLPA(10),
		Bottom: style.LengthLPA(10),
	}
	gen := fixedStyleGen{style: superDeepStyle}

	for _, depth := range []uint32{100} {
		b.Run(fmt.Sprintf("%d", depth), func(b *testing.B) {
			benchmarkLayout(b, defaultAvailableSpace(), func(t *TaffyTree, rng *rand.Rand) tree.NodeId {
				children := buildSuperDeepHierarchy(t, depth, 3, gen, rng)
				return t.NewWithChildren(style.DefaultStyle(), children)
			})
		})
	}
}
