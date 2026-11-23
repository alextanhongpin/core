package queue_test

import (
	"fmt"

	"github.com/alextanhongpin/core/types/queue"
)

type Task struct {
	Name     string
	Priority int
}

func (t Task) Less(o Task) bool { return t.Priority < o.Priority }

func ExamplePriorityQueue() {
	var pq queue.PriorityQueue[Task]
	pq.Push(Task{"write", 2}, Task{"deploy", 3}, Task{"lint", 1})
	for pq.Len() > 0 {
		t, _ := pq.Pop()
		fmt.Println(t.Name)
	}
	// Output:
	// lint
	// write
	// deploy
}

func ExamplePriorityQueue_topK() {
	pq := queue.PriorityQueue[Task]{TopK: 2}
	pq.Push(Task{"a", 5}, Task{"b", 1}, Task{"c", 9}, Task{"d", 3})
	fmt.Println(pq.Slice())
	// Output: [{a 5} {c 9}]
}
