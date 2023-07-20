package circuitbreaker_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/circuitbreaker"
	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker(t *testing.T) {
	assert := assert.New(t)

	cb := circuitbreaker.New()
	cb.Timeout = 100 * time.Millisecond

	assert.Equal(circuitbreaker.StatusClosed, cb.Status())

	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()

			err := cb.Do(func() error {
				return errors.New("bad request")
			})
			t.Log(err, cb.Status())
		}()
	}
	wg.Wait()

	conc := make(chan bool)
	done := make(chan bool)

	go func() {
		<-conc
		err := cb.Do(func() error {
			return errors.New("bad request")
		})
		t.Log(err, cb.Status())
		close(done)
	}()

	// Trigger concurrent run.
	close(conc)
	<-done
	err := cb.Do(func() error { return nil })

	assert.ErrorIs(err, circuitbreaker.Unavailable, err)
	assert.Equal(circuitbreaker.StatusOpen, cb.Status())

	time.Sleep(110 * time.Millisecond)

	_ = cb.Do(func() error { return nil })
	assert.Equal(circuitbreaker.StatusHalfOpen, cb.Status())

	for i := 0; i < 10; i++ {
		err := cb.Do(func() error {
			return nil
		})
		t.Log(err, cb.Status())
	}

	err = cb.Do(func() error { return nil })
	assert.Nil(err)
	assert.Equal(circuitbreaker.StatusClosed, cb.Status())
}
