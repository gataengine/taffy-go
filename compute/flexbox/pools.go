package flexbox

import "sync"

var flexItemsPool = sync.Pool{
	New: func() any {
		s := make([]flexItem, 0, 8)
		return &s
	},
}

func getFlexItems(capacity int) []flexItem {
	sp := flexItemsPool.Get().(*[]flexItem)
	s := *sp
	if cap(s) < capacity {
		s = make([]flexItem, 0, capacity)
	}
	return s[:0]
}

func putFlexItems(s []flexItem) {
	s = s[:0]
	flexItemsPool.Put(&s)
}
