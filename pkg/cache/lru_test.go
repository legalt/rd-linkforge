package cache

import (
	"errors"
	"reflect"
	"testing"
)

func TestNewLRURejectsInvalidCapacity(t *testing.T) {
	for _, capacity := range []int{-1, 0} {
		cache, err := NewLRU[string, int](capacity)
		if cache != nil || !errors.Is(err, ErrInvalidCapacity) {
			t.Errorf("NewLRU(%d) = %v, %v", capacity, cache, err)
		}
	}
}

func TestLRUOrderAndEviction(t *testing.T) {
	cache, err := NewLRU[string, int](2)
	if err != nil {
		t.Fatal(err)
	}

	cache.Put("a", 1)
	cache.Put("b", 2)
	assertKeys(t, cache, []string{"b", "a"})

	if value, ok := cache.Get("a"); !ok || value != 1 {
		t.Fatalf("Get(a) = %d, %v", value, ok)
	}
	assertKeys(t, cache, []string{"a", "b"})

	cache.Put("c", 3)
	if value, ok := cache.Get("b"); ok || value != 0 {
		t.Errorf("evicted Get(b) = %d, %v", value, ok)
	}
	if cache.Len() != 2 {
		t.Errorf("Len = %d, want 2", cache.Len())
	}
	assertKeys(t, cache, []string{"c", "a"})
}

func TestLRUUpdateAndDelete(t *testing.T) {
	cache, err := NewLRU[string, int](2)
	if err != nil {
		t.Fatal(err)
	}
	cache.Put("a", 1)
	cache.Put("b", 2)
	cache.Put("a", 10)

	if value, ok := cache.Get("a"); !ok || value != 10 {
		t.Errorf("updated Get(a) = %d, %v", value, ok)
	}
	cache.Delete("missing")
	cache.Delete("b")
	if cache.Len() != 1 {
		t.Errorf("Len = %d, want 1", cache.Len())
	}
	assertKeys(t, cache, []string{"a"})
}

func assertKeys(t *testing.T, cache *LRU[string, int], want []string) {
	t.Helper()
	if got := cache.Keys(); !reflect.DeepEqual(got, want) {
		t.Errorf("Keys = %v, want %v", got, want)
	}
}

func BenchmarkLRUGetHit(b *testing.B) {
	cache, err := NewLRU[string, int](128)
	if err != nil {
		b.Fatal(err)
	}
	for i := range 128 {
		cache.Put(string(rune(i)), i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		cache.Get("a")
	}
}
