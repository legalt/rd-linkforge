// Package topk retains the best N items offered to it.
package topk

import (
	"container/heap"
	"errors"
	"slices"
)

var ErrInvalidN = errors.New("topk: n must be positive")

type Item struct {
	Key   string
	Count int64
}

type itemHeap []Item

func (h itemHeap) Len() int { return len(h) }

func (h itemHeap) Less(i, j int) bool { return worse(h[i], h[j]) }

func (h itemHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *itemHeap) Push(value any) {
	*h = append(*h, value.(Item))
}

func (h *itemHeap) Pop() any {
	old := *h
	last := len(old) - 1
	item := old[last]
	*h = old[:last]
	return item
}

type TopK struct {
	n     int
	items itemHeap
}

func New(n int) (*TopK, error) {
	if n <= 0 {
		return nil, ErrInvalidN
	}
	return &TopK{n: n, items: make(itemHeap, 0, n)}, nil
}

// Add offers an item without merging equal keys.
func (t *TopK) Add(key string, count int64) {
	item := Item{Key: key, Count: count}
	if len(t.items) < t.n {
		heap.Push(&t.items, item)
		return
	}
	if !better(item, t.items[0]) {
		return
	}

	heap.Pop(&t.items)
	heap.Push(&t.items, item)
}

// Result returns retained items from best to worst without changing the heap.
func (t *TopK) Result() []Item {
	result := slices.Clone(t.items)
	slices.SortFunc(result, func(a, b Item) int {
		switch {
		case better(a, b):
			return -1
		case better(b, a):
			return 1
		default:
			return 0
		}
	})
	return result
}

func better(a, b Item) bool {
	return a.Count > b.Count || a.Count == b.Count && a.Key < b.Key
}

func worse(a, b Item) bool {
	return a.Count < b.Count || a.Count == b.Count && a.Key > b.Key
}
