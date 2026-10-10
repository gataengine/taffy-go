package lib_test

import (
	"testing"

	"github.com/gataengine/taffy-go/lib"
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

	k2 := sm.Insert("second")

	if k1.Index != k2.Index {
		t.Fatalf("expected reuse of index %d, got %d", k1.Index, k2.Index)
	}
	if k1.Generation == k2.Generation {
		t.Fatal("expected different generations after reuse")
	}
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

	for i := 0; i < 1000; i += 2 {
		sm.Remove(keys[i])
	}
	if sm.Len() != 500 {
		t.Fatalf("Len() = %d, want 500", sm.Len())
	}

	for i := range 500 {
		sm.Insert(i + 2000)
	}
	if sm.Len() != 1000 {
		t.Fatalf("Len() = %d, want 1000", sm.Len())
	}

	for i := 1; i < 1000; i += 2 {
		if v := sm.Get(keys[i]); v == nil || *v != i {
			t.Fatalf("Get(keys[%d]) = %v, want %d", i, v, i)
		}
	}
}
