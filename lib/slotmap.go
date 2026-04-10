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
