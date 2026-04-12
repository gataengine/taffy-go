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

var flexLinesPool = sync.Pool{
	New: func() any {
		s := make([]flexLine, 0, 1)
		return &s
	},
}

func getFlexLines(capacity int) []flexLine {
	sp := flexLinesPool.Get().(*[]flexLine)
	s := *sp
	if cap(s) < capacity {
		s = make([]flexLine, 0, capacity)
	}
	return s[:0]
}

func putFlexLines(s []flexLine) {
	s = s[:0]
	flexLinesPool.Put(&s)
}
