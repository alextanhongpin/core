package snapshot_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/alextanhongpin/core/sync/snapshot"
	"github.com/stretchr/testify/assert"
)

func TestSnapshot(t *testing.T) {
	policies := []snapshot.Policy{
		{Changes: 10_000, After: 10 * time.Millisecond},
		{Changes: 1_000, After: 20 * time.Millisecond},
		{Changes: 100, After: 30 * time.Millisecond},
	}

	synctest.Test(t, func(t *testing.T) {
		s, stop := snapshot.New(snapshot.Config{
			Policies: policies,
		})
		defer stop()

		var logs []snapshot.Policy
		s.Go(func(p snapshot.Policy) {
			logs = append(logs, p)
		})
		ch := s.Chan()

		is := assert.New(t)
		time.Sleep(11 * time.Millisecond)
		s.Add(10_000)
		is.Equal(policies[0], <-ch)

		time.Sleep(21 * time.Millisecond)
		is.Equal(policies[:1], logs)
		s.Add(1_000)
		is.Equal(policies[1], <-ch)

		time.Sleep(31 * time.Millisecond)
		is.Equal(policies[:2], logs)
		s.Add(100)
		is.Equal(policies[2], <-ch)
		time.Sleep(10 * time.Millisecond)
		is.Equal(policies, logs)
	})
}
