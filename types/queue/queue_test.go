package queue_test

import (
	"slices"
	"testing"

	"github.com/alextanhongpin/core/types/queue"
)

type Leaderboard struct {
	Name  string
	Score int
}

func (l *Leaderboard) Less(o *Leaderboard) bool {
	if l.Score != o.Score {
		return l.Score < o.Score
	}
	return l.Name < o.Name
	/*
		This does not work
		return cmp.Or(
			cmp.Less(l.Score, o.Score),
			cmp.Less(l.Name, o.Name),
		)
	*/
}

func TestPriorityQueue(t *testing.T) {
	a := &Leaderboard{Name: "alice", Score: 10}
	b := &Leaderboard{Name: "bob", Score: 1}
	c := &Leaderboard{Name: "charles", Score: 100}
	pq := queue.NewPriorityQueue[*Leaderboard]()
	pq.Push(a, b, c)
	// Sequence will be bob, alice, charles
	d, _ := pq.Pop()
	e, _ := pq.Pop()
	f, _ := pq.Pop()
	t.Log(d, e, f)
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
	a := &Leaderboard{Name: "alice", Score: 10}
	b := &Leaderboard{Name: "bob", Score: 1}
	c := &Leaderboard{Name: "charles", Score: 100}
	pq := queue.NewPriorityQueue[*Leaderboard]()
	pq.TopK = 2
	pq.Push(a, b, c)

	if want, got := 2, pq.Len(); want != got {
		t.Fatalf("want %d, got %d", want, got)
	}
	d, ok := pq.Pop()
	if a != d || !ok {
		t.Fatalf("want %s, got %s", a.Name, d.Name)
	}
	e, ok := pq.Pop()
	if c != e || !ok {
		t.Fatalf("want %s, got %s", c.Name, e.Name)
	}
	_, ok = pq.Pop()
	if ok {
		t.Fatal("want false")
	}
}

func TestPriorityQueue_Duplicate(t *testing.T) {
	pq := queue.NewPriorityQueue[*Leaderboard]()
	pq.TopK = 3
	pq.Push(&Leaderboard{Name: "alice", Score: 10})
	pq.Push(&Leaderboard{Name: "alice", Score: 1})
	pq.Push(&Leaderboard{Name: "alice", Score: 100})
	pq.Push(&Leaderboard{Name: "alice", Score: 1000})
	pq.Push(&Leaderboard{Name: "bob", Score: 1})
	pq.Push(&Leaderboard{Name: "charlie", Score: 1})
	pq.Push(&Leaderboard{Name: "charlie", Score: 10})
	pq.Push(&Leaderboard{Name: "charlie", Score: 100})
	pq.Push(&Leaderboard{Name: "zeta", Score: 1000})
	list := pq.Slice()
	slices.Reverse(list)
	for _, item := range list {
		t.Log(item)
	}
}
