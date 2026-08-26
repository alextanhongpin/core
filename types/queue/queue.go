package queue

import (
	"container/heap"
	"slices"
	"sort"
)

type Orderable[T any] interface {
	Less(other T) bool
}

// A priorityQueue implements heap.Interface and holds Items.
type priorityQueue[T Orderable[T]] []T

func (pq priorityQueue[T]) Len() int { return len(pq) }

func (pq priorityQueue[T]) Less(i, j int) bool {
	// If we want Pop to give us the highest, not lowest, priorityQueue we can use greater than here.
	return pq[i].Less(pq[j])
}

func (pq priorityQueue[T]) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *priorityQueue[T]) Push(x any) {
	item := x.(T)
	*pq = append(*pq, item)
}

func (pq *priorityQueue[T]) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = slices.Delete(old, n-1, n)
	return item
}

type PriorityQueue[T Orderable[T]] struct {
	q    priorityQueue[T]
	TopK int
}

func NewPriorityQueue[T Orderable[T]]() *PriorityQueue[T] {
	q := make(priorityQueue[T], 0)
	heap.Init(&q)
	return &PriorityQueue[T]{
		q: q,
	}
}

func (pq *PriorityQueue[T]) Push(vals ...T) {
	for _, val := range vals {
		heap.Push(&pq.q, val)
		for pq.TopK > 0 && pq.q.Len() > pq.TopK {
			_ = heap.Pop(&pq.q)
		}
	}
}

func (pq *PriorityQueue[T]) Pop() (val T, ok bool) {
	if pq.q.Len() == 0 {
		return
	}
	item, ok := heap.Pop(&pq.q).(T)
	return item, true
}

func (pq *PriorityQueue[T]) Len() int {
	return pq.q.Len()
}

func (pq *PriorityQueue[T]) Slice() []T {
	res := slices.Clone(pq.q)
	sort.Slice(res, func(i, j int) bool {
		return res[i].Less(res[j])
	})
	return res
}
