package singleflight_test

import (
	"context"
	"errors"
	"github.com/alextanhongpin/core/dsync/singleflight"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLeaderJoinsCallbackAfterCancellation(t *testing.T) {
	g := singleflight.MustNew(redistest.Client(t), singleflight.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	release := make(chan struct{})
	exited := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		_, err := g.Do(ctx, t.Name(), func(work context.Context) error {
			defer close(exited)
			close(started)
			<-work.Done()
			<-release
			return context.Cause(work)
		})
		result <- err
	}()
	<-started
	cancel()
	select {
	case err := <-result:
		t.Fatalf("returned before callback exited: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	require.ErrorIs(t, <-result, context.Canceled)
	<-exited
}
func TestCanceledLocalFollowerReturns(t *testing.T) {
	g := singleflight.MustNew(redistest.Client(t), singleflight.Config{})
	started := make(chan struct{})
	release := make(chan struct{})
	leader := make(chan error, 1)
	go func() {
		_, err := g.Do(context.Background(), t.Name(), func(context.Context) error { close(started); <-release; return nil })
		leader <- err
	}()
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := g.Do(ctx, t.Name(), func(context.Context) error { t.Error("follower ran"); return nil })
	require.ErrorIs(t, err, context.Canceled)
	close(release)
	require.NoError(t, <-leader)
}
func TestRemoteWaiterObservesRelease(t *testing.T) {
	client := redistest.Client(t)
	g := singleflight.MustNew(client, singleflight.Config{})
	started := make(chan struct{})
	release := make(chan struct{})
	leader := make(chan error, 1)
	go func() {
		_, err := g.Do(context.Background(), t.Name(), func(context.Context) error { close(started); <-release; return errors.New("leader failed") })
		leader <- err
	}()
	<-started
	waiter := make(chan error, 1)
	go func() {
		did, err := singleflight.MustNew(client, singleflight.Config{}).Do(context.Background(), t.Name(), func(context.Context) error { t.Error("remote follower ran"); return nil })
		if did {
			t.Error("follower reported leader")
		}
		waiter <- err
	}()
	time.Sleep(30 * time.Millisecond)
	close(release)
	require.Error(t, <-leader)
	require.NoError(t, <-waiter)
}
