package taffy

import (
	"fmt"
	"testing"

	"github.com/gataengine/taffy-go/style"
)

func BenchmarkTreeCreation(b *testing.B) {
	for _, nodeCount := range []uint32{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("%d", nodeCount), func(b *testing.B) {
			for b.Loop() {
				rng := newBenchRNG()
				t := New()
				gen := fixedStyleGen{style: style.DefaultStyle()}
				children := buildFlatHierarchy(t, nodeCount, gen, rng)
				root := t.NewWithChildren(style.DefaultStyle(), children)
				_ = root
			}
		})
	}
}
