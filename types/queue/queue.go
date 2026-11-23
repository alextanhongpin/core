package queue

import (
	"container/heap"
	"slices"
)

// Orderable is implemented by values that define their own priority order.
// a.Less(b) reports whether a has lower priority than b.
type Orderable[T any] interface {
	Less(other T) bool
}

type priorityQueue[T Orderable[T]] []T

func (pq priorityQueue[T]) Len() int           { return len(pq) }
func (pq priorityQueue[T]) Less(i, j int) bool { return pq[i].Less(pq[j]) }
func (pq priorityQueue[T]) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }

func (pq *priorityQueue[T]) Push(x any) {
	*pq = append(*pq, x.(T))
}

func (pq *priorityQueue[T]) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = slices.Delete(old, n-1, n) // clears the vacated slot
	return item
}

// PriorityQueue is a binary min-heap: Pop returns the lowest-priority item
// according to Less. The zero value is an empty, unbounded queue. It is not
// safe for concurrent use.
type PriorityQueue[T Orderable[T]] struct {
	q priorityQueue[T]

	// TopK, when positive, bounds the queue: after each Push the lowest items
	// are discarded until at most TopK remain, so the queue retains the TopK
	// highest items seen.
	TopK int
}

// NewPriorityQueue returns an empty, unbounded queue.
func NewPriorityQueue[T Orderable[T]]() *PriorityQueue[T] {
	return &PriorityQueue[T]{}
}

// Push adds vals, discarding the lowest items if TopK is exceeded.
func (pq *PriorityQueue[T]) Push(vals ...T) {
	for _, val := range vals {
		heap.Push(&pq.q, val)
		for pq.TopK > 0 && pq.q.Len() > pq.TopK {
			heap.Pop(&pq.q)
		}
	}
}

// Pop removes and returns the lowest item. ok is false if the queue is empty.
func (pq *PriorityQueue[T]) Pop() (val T, ok bool) {
	if pq.q.Len() == 0 {
		return val, false
	}
	return heap.Pop(&pq.q).(T), true
}

// Peek returns the lowest item without removing it.
func (pq *PriorityQueue[T]) Peek() (val T, ok bool) {
	if pq.q.Len() == 0 {
		return val, false
	}
	return pq.q[0], true
}

// Len returns the number of items in the queue.
func (pq *PriorityQueue[T]) Len() int {
	return pq.q.Len()
}

// Slice returns a copy of the items in ascending order. The queue is not
// modified.
func (pq *PriorityQueue[T]) Slice() []T {
	res := slices.Clone(pq.q)
	slices.SortStableFunc(res, func(a, b T) int {
		switch {
		case a.Less(b):
			return -1
		case b.Less(a):
			return 1
		default:
			return 0
		}
	})
	return res
}
