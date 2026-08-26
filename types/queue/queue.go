package queue

import (
	"cmp"
	"container/heap"
	"slices"
)

// An item is something we manage in a priorityQueue queue.
type item[T any, V cmp.Ordered] struct {
	Value    T // The Value of the item; arbitrary.
	Priority V // The priorityQueue of the item in the queue.
}

// A priorityQueue implements heap.Interface and holds Items.
type priorityQueue[T any, V cmp.Ordered] []*item[T, V]

func (pq priorityQueue[T, V]) Len() int { return len(pq) }

func (pq priorityQueue[T, V]) Less(i, j int) bool {
	// If we want Pop to give us the highest, not lowest, priorityQueue we can use greater than here.
	return pq[i].Priority < pq[j].Priority
}

func (pq priorityQueue[T, V]) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *priorityQueue[T, V]) Push(x any) {
	item := x.(*item[T, V])
	*pq = append(*pq, item)
}

func (pq *priorityQueue[T, V]) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // don't stop the GC from reclaiming the item eventually
	*pq = old[0 : n-1]
	return item
}

type PriorityQueue[T any, V cmp.Ordered] struct {
	q    priorityQueue[T, V]
	TopK int
}

func NewPriorityQueue[T any, V cmp.Ordered]() *PriorityQueue[T, V] {
	return &PriorityQueue[T, V]{
		q: make(priorityQueue[T, V], 0),
	}
}

func (pq *PriorityQueue[T, V]) Push(val T, p V) {
	heap.Push(&pq.q, &item[T, V]{Value: val, Priority: p})
	for pq.TopK > 0 && pq.q.Len() > pq.TopK {
		_ = heap.Pop(&pq.q)
	}
}

func (pq *PriorityQueue[T, V]) Pop() (val T, p V, ok bool) {
	if pq.q.Len() == 0 {
		return
	}
	item, ok := heap.Pop(&pq.q).(*item[T, V])
	return item.Value, item.Priority, true
}

func (pq *PriorityQueue[T, V]) Len() int {
	return pq.q.Len()
}

func (pq *PriorityQueue[T, V]) Slice() []T {
	q := slices.Clone(pq.q)
	slices.SortFunc(q, func(a, b *item[T, V]) int {
		return cmp.Compare(a.Priority, b.Priority)
	})
	res := make([]T, len(q))
	for i, v := range q {
		res[i] = v.Value
	}
	return res
}
