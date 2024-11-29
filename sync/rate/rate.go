package rate

import (
	"sync"
	"time"
)

type counter interface {
	Add(float64) float64
	Inc() float64
	Count() float64
}

var _ counter = (*Rate)(nil)

type Rate struct {
	Now    func() time.Time
	count  float64
	last   int64
	mu     sync.Mutex
	period int64
}

func New() *Rate {
	return NewRate(time.Second)
}

func NewRate(period time.Duration) *Rate {
	return &Rate{
		Now:    time.Now,
		period: period.Nanoseconds(),
	}
}

func (r *Rate) Reset() {
	r.mu.Lock()
	r.reset()
	r.mu.Unlock()
}

func (r *Rate) Inc() float64 {
	return r.Add(1)
}

func (r *Rate) Add(n float64) float64 {
	r.mu.Lock()
	f := r.add(n)
	r.mu.Unlock()

	return f
}

func (r *Rate) Count() float64 {
	return r.Add(0)
}

func (r *Rate) Per(t time.Duration) float64 {
	return r.Count() * float64(t.Nanoseconds()) / float64(r.period)
}

func (r *Rate) reset() {
	r.count = 0
	r.last = 0
}

func (r *Rate) add(n float64) float64 {
	now := r.Now().UnixNano()
	ratio := 1 - float64(min(now-r.last, r.period))/float64(r.period)
	r.count = r.count*ratio + n
	r.last = now

	return r.count
}
