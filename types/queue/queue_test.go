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
	want := []Leaderboard{{"charlie", 100}, {"alice", 1000}, {"zeta", 1000}}
	if len(list) != len(want) {
		t.Fatalf("len = %d, want %d", len(list), len(want))
	}
	for i := range want {
		if *list[i] != want[i] {
			t.Fatalf("list[%d] = %v, want %v", i, *list[i], want[i])
		}
	}
	if pq.Len() != 3 {
		t.Fatal("Slice modified the queue")
	}
}

func TestPriorityQueue_ZeroValue(t *testing.T) {
	var pq queue.PriorityQueue[*Leaderboard]
	if _, ok := pq.Pop(); ok {
		t.Fatal("empty Pop: want false")
	}
	if _, ok := pq.Peek(); ok {
		t.Fatal("empty Peek: want false")
	}
	pq.Push(&Leaderboard{Name: "b", Score: 2}, &Leaderboard{Name: "a", Score: 1})
	if v, ok := pq.Peek(); !ok || v.Name != "a" || pq.Len() != 2 {
		t.Fatalf("Peek = %v, %v", v, ok)
	}
	if !slices.Equal([]string{"a", "b"}, []string{pq.Slice()[0].Name, pq.Slice()[1].Name}) {
		t.Fatal("Slice not ascending")
	}
}
