package grid

import (
	"sync"

	"github.com/gataengine/taffy-go/style"
)

// gridScratch holds reusable slice buffers for ComputeGridLayout.
// Pooled via gridScratchPool to avoid repeated heap allocations.
type gridScratch struct {
	items          []gridItem
	inFlowChildren []gridPlacementChild
	childStyles    []style.GridItemStyle
	columns        []gridTrack
	rows           []gridTrack
}

// reset clears all slices to zero length while retaining backing arrays.
func (s *gridScratch) reset() {
	s.items = s.items[:0]
	s.inFlowChildren = s.inFlowChildren[:0]
	s.childStyles = s.childStyles[:0]
	s.columns = s.columns[:0]
	s.rows = s.rows[:0]
}

// gridScratchPool is a pool of reusable gridScratch instances.
var gridScratchPool = sync.Pool{
	New: func() any { return new(gridScratch) },
}
