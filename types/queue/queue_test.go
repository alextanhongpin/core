package queue_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/queue"
)

func TestPriorityQueue(t *testing.T) {
	type Leaderboard struct {
		Name  string
		Score int
	}

	a := Leaderboard{Name: "alice", Score: 10}
	b := Leaderboard{Name: "bob", Score: 1}
	c := Leaderboard{Name: "charles", Score: 100}
	pq := queue.NewPriorityQueue[Leaderboard, int]()
	pq.Push(a, a.Score)
	pq.Push(b, b.Score)
	pq.Push(c, c.Score)

	// Sequence will be bob, alice, charles
	d, _, _ := pq.Pop()
	e, _, _ := pq.Pop()
	f, _, _ := pq.Pop()
	if b != d {
		t.Fatalf("want %s, got %s", b.Name, d.Name)
	}
	if a != e {
		t.Fatalf("want %s, got %s", a.Name, e.Name)
	}
	if c != f {
		t.Fatalf("want %s, got %s", c.Name, f.Name)
	}
}

func TestPriorityQueue_TopK(t *testing.T) {
	type Leaderboard struct {
		Name  string
		Score int
	}

	a := Leaderboard{Name: "alice", Score: 10}
	b := Leaderboard{Name: "bob", Score: 1}
	c := Leaderboard{Name: "charles", Score: 100}
	pq := queue.NewPriorityQueue[Leaderboard, int]()
	pq.TopK = 2
	pq.Push(a, a.Score)
	pq.Push(b, b.Score)
	pq.Push(c, c.Score)

	if want, got := 2, pq.Len(); want != got {
		t.Fatalf("want %d, got %d", want, got)
	}
	d, _, ok := pq.Pop()
	if a != d || !ok {
		t.Fatalf("want %s, got %s", a.Name, d.Name)
	}
	e, _, ok := pq.Pop()
	if c != e || !ok {
		t.Fatalf("want %s, got %s", c.Name, e.Name)
	}
	_, _, ok = pq.Pop()
	if ok {
		t.Fatal("want false")
	}
}
