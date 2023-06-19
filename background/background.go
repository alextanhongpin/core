package background

import "sync"

type Task[T any] interface {
	Exec(T)
}

type Option interface {
	isOption()
}

type Buffer int

func (b Buffer) isOption() {}

// New returns a new background manager.
func New[T any](task Task[T], opts ...Option) (*Manager[T], func()) {
	var buf Buffer
	for _, opt := range opts {
		switch t := (opt).(type) {
		case Buffer:
			buf = t
		}
	}

	mgr := &Manager[T]{
		ch:   make(chan T, buf),
		done: make(chan struct{}),
		fn:   task,
	}
	return mgr, mgr.stop
}

type Manager[T any] struct {
	wg       sync.WaitGroup
	ch       chan T
	done     chan struct{}
	initOnce sync.Once
	stopOnce sync.Once
	fn       Task[T]
}

// Send sends a new message to the channel.
func (m *Manager[T]) Send(t T) {
	m.init()

	select {
	case <-m.done:
		m.exec(t)
	case m.ch <- t:
	}
}

// init inits the goroutine that listens for messages from the channel.
func (m *Manager[T]) init() {
	m.initOnce.Do(func() {
		m.wg.Add(1)

		go func() {
			defer m.wg.Done()
			m.subscribe()
		}()
	})
}

// stop stops the channel and waits for the channel messages to be flushed.
func (m *Manager[T]) stop() {
	m.stopOnce.Do(func() {
		close(m.done)
		m.wg.Wait()
	})
}

// subscribe listens to the channel for new messages.
func (m *Manager[T]) subscribe() {
	defer m.flush()

	for {
		select {
		case <-m.done:
			return
		case v := <-m.ch:
			m.exec(v)
		}
	}
}

// flush flushes the remaining items in the buffer.
// Only works with buffered channel.
func (m *Manager[T]) flush() {
	for len(m.ch) > 0 {
		m.exec(<-m.ch)
	}
}

// exec executes the function.
func (m *Manager[T]) exec(t T) {
	if m.fn == nil {
		return
	}

	m.fn.Exec(t)
}
