package taffy

import (
	"fmt"
	"testing"
)

func TestBlockDebug(t *testing.T) {
	tree := New()

	child0 := tree.NewLeaf(Style{
		Display:   DisplayBlock,
		Direction: DirectionLTR,
		Size:      Size[Dimension]{Width: AutoDimension(), Height: LengthDimension(10)},
	})
	child1 := tree.NewLeaf(Style{
		Display:   DisplayBlock,
		Direction: DirectionLTR,
		Size:      Size[Dimension]{Width: AutoDimension(), Height: LengthDimension(10)},
	})
	root := tree.NewWithChildren(Style{
		Display:   DisplayBlock,
		Direction: DirectionLTR,
		Size:      Size[Dimension]{Width: LengthDimension(50), Height: AutoDimension()},
	}, []NodeId{child0, child1})

	tree.ComputeLayout(root, Size[AvailableSpace]{
		Width:  AvailableSpaceFromDefinite(50),
		Height: AvailableSpaceMaxContentValue,
	})

	rootLayout := tree.GetLayout(root)
	child0Layout := tree.GetLayout(child0)
	child1Layout := tree.GetLayout(child1)

	fmt.Printf("Root: size=%+v, location=%+v\n", rootLayout.Size, rootLayout.Location)
	fmt.Printf("Child0: size=%+v, location=%+v\n", child0Layout.Size, child0Layout.Location)
	fmt.Printf("Child1: size=%+v, location=%+v\n", child1Layout.Size, child1Layout.Location)

	// Verify
	if rootLayout.Size.Width != 50 || rootLayout.Size.Height != 20 {
		t.Errorf("Root size: got %+v, want {50, 20}", rootLayout.Size)
	}
	if child0Layout.Size.Width != 50 || child0Layout.Size.Height != 10 {
		t.Errorf("Child0 size: got %+v, want {50, 10}", child0Layout.Size)
	}
	if child0Layout.Location.X != 0 || child0Layout.Location.Y != 0 {
		t.Errorf("Child0 location: got %+v, want {0, 0}", child0Layout.Location)
	}
	if child1Layout.Size.Width != 50 || child1Layout.Size.Height != 10 {
		t.Errorf("Child1 size: got %+v, want {50, 10}", child1Layout.Size)
	}
	if child1Layout.Location.X != 0 || child1Layout.Location.Y != 10 {
		t.Errorf("Child1 location: got %+v, want {0, 10}", child1Layout.Location)
	}
}
