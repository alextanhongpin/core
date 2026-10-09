package singleflight_test

import (
	"context"
	"errors"
	"github.com/alextanhongpin/core/dsync/singleflight"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLeaderJoinsCallbackAfterCancellation(t *testing.T) {
	g := singleflight.New(redistest.New(t).Client())
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
		}, time.Second, time.Second)
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
	g := singleflight.New(redistest.New(t).Client())
	started := make(chan struct{})
	release := make(chan struct{})
	leader := make(chan error, 1)
	go func() {
		_, err := g.Do(context.Background(), t.Name(), func(context.Context) error { close(started); <-release; return nil }, time.Second, time.Second)
		leader <- err
	}()
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := g.Do(ctx, t.Name(), func(context.Context) error { t.Error("follower ran"); return nil }, time.Second, time.Second)
	require.ErrorIs(t, err, context.Canceled)
	close(release)
	require.NoError(t, <-leader)
}
func TestRemoteWaiterObservesRelease(t *testing.T) {
	client := redistest.New(t).Client()
	g := singleflight.New(client)
	started := make(chan struct{})
	release := make(chan struct{})
	leader := make(chan error, 1)
	go func() {
		_, err := g.Do(context.Background(), t.Name(), func(context.Context) error { close(started); <-release; return errors.New("leader failed") }, time.Second, time.Second)
		leader <- err
	}()
	<-started
	waiter := make(chan error, 1)
	go func() {
		did, err := singleflight.New(client).Do(context.Background(), t.Name(), func(context.Context) error { t.Error("remote follower ran"); return nil }, time.Second, time.Second)
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
