package retry_test

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func newRetry(t *testing.T, retries int) *retry.Retry {
	t.Helper()
	r, err := retry.New(retry.Config{MaxRetries: retries, Backoff: retry.NewConstantBackoff(0), Throttler: retry.NewNoopThrottler()})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestNonRetryableErrors(t *testing.T) {
	permanent := errors.New("permanent")
	joined := errors.Join(permanent, context.Canceled)
	if !errors.Is(joined, permanent) {
		t.Fatal("joined error must expose its member")
	}
	if errors.Is(joined, fmt.Errorf("wrapped: %w", permanent)) {
		t.Fatal("errors.Is must inspect the error, not its target")
	}
	for _, err := range []error{permanent, fmt.Errorf("wrapped: %w", permanent), errors.Join(errors.New("other"), fmt.Errorf("wrapped: %w", permanent))} {
		got, again := retry.NonRetryableErrors(permanent)(err)
		if again || got != err || errors.Is(got, retry.ErrCanceled) {
			t.Fatalf("classification: %v, %v", got, again)
		}
	}
}

func TestConfiguration(t *testing.T) {
	if _, err := retry.New(retry.Config{MaxRetries: -1}); err == nil {
		t.Fatal("negative budget accepted")
	}
	r, err := retry.New(retry.Config{})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	err = r.Do(context.Background(), func(context.Context) error { calls++; return errors.ErrUnsupported })
	if calls != 1 || !errors.Is(err, retry.ErrLimitExceeded) {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	cfg := retry.Config{MaxRetries: 1, Backoff: retry.NewConstantBackoff(0), Throttler: retry.NewNoopThrottler()}
	r, err = retry.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxRetries = 100
	calls = 0
	err = r.Do(context.Background(), func(context.Context) error { calls++; return errors.ErrUnsupported })
	if calls != 2 {
		t.Fatalf("configuration copy changed: calls=%d err=%v", calls, err)
	}
}

func TestClassifiedErrorAndLastResult(t *testing.T) {
	original := errors.New("original")
	classified := errors.New("classified")
	r, err := retry.New(retry.Config{MaxRetries: 2, Backoff: retry.NewConstantBackoff(0), Throttler: retry.NewNoopThrottler(), Retryable: func(err error) (error, bool) { return errors.Join(classified, err), true }})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	fn := retry.Func(func(context.Context, int) (int, error) { calls++; return calls, original }, r)
	got, err := fn(context.Background(), 0)
	if got != 3 || !errors.Is(err, classified) || !errors.Is(err, retry.ErrLimitExceeded) {
		t.Fatalf("got=%d err=%v", got, err)
	}
}

func TestCancellation(t *testing.T) {
	cause := errors.New("cause")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	calls := 0
	err := newRetry(t, 2).Do(ctx, func(context.Context) error { calls++; return nil })
	if calls != 0 || !errors.Is(err, retry.ErrCanceled) || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	fn := retry.Func(func(context.Context, int) (int, error) { t.Fatal("canceled callback invoked"); return 1, nil }, newRetry(t, 2))
	if got, _ := fn(ctx, 1); got != 0 {
		t.Fatal(got)
	}
	for _, target := range []error{context.Canceled, context.DeadlineExceeded} {
		calls = 0
		err = newRetry(t, 2).Do(context.Background(), func(context.Context) error { calls++; return fmt.Errorf("wrapped: %w", target) })
		if calls != 1 || !errors.Is(err, retry.ErrCanceled) || !errors.Is(err, target) {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	}
}

func TestCancellationDuringBackoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		r, err := retry.New(retry.Config{MaxRetries: 2, Backoff: retry.NewConstantBackoff(time.Hour), Throttler: retry.NewNoopThrottler()})
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		calls := 0
		go func() { done <- r.Do(ctx, func(context.Context) error { calls++; return errors.ErrUnsupported }) }()
		synctest.Wait()
		cancel()
		err = <-done
		if calls != 1 || !errors.Is(err, retry.ErrCanceled) {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	})
}

func TestThrottleBudgetAndConcurrency(t *testing.T) {
	limiter, err := retry.NewThrottler(retry.DefaultThrottlerConfig())
	if err != nil {
		t.Fatal(err)
	}
	r, err := retry.New(retry.Config{MaxRetries: 10, Backoff: retry.NewConstantBackoff(0), Throttler: limiter})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	err = r.Do(context.Background(), func(context.Context) error { calls++; return errors.ErrUnsupported })
	if calls != 6 || !errors.Is(err, retry.ErrThrottled) {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	for range 11 {
		if err := r.Do(context.Background(), func(context.Context) error { return nil }); err != nil {
			t.Fatal(err)
		}
	}
	if !limiter.Allow() {
		t.Fatal("success did not replenish budget")
	}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 100 {
				limiter.Allow()
				limiter.Success()
				_ = r.Do(context.Background(), func(context.Context) error { return nil })
			}
		})
	}
	wg.Wait()
	for _, cfg := range []retry.ThrottlerConfig{{MaxTokens: -1}, {MaxTokens: math.NaN()}, {TokenRatio: -1}, {TokenRatio: math.Inf(1)}} {
		if _, err := retry.NewThrottler(cfg); err == nil {
			t.Fatalf("invalid config accepted: %+v", cfg)
		}
	}
}

func TestBackoff(t *testing.T) {
	if got := retry.NewConstantBackoff(time.Second).At(0); got != time.Second {
		t.Fatal(got)
	}
	if got := retry.NewLinearBackoff(time.Second).At(3); got != 3*time.Second {
		t.Fatal(got)
	}
	for range 100 {
		if got := retry.NewExponentialBackoff(time.Second, 3*time.Second).At(1); got < 0 || got >= 2*time.Second {
			t.Fatal(got)
		}
		if got := retry.NewExponentialBackoff(time.Second, 3*time.Second).At(100); got < 0 || got >= 3*time.Second {
			t.Fatal(got)
		}
	}
}

func TestPolicyCannotRetryCancellation(t *testing.T) {
	r, err := retry.New(retry.Config{MaxRetries: 1, Retryable: func(error) (error, bool) {
		t.Fatal("context error reached policy")
		return errors.New("replacement"), true
	}})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Do(context.Background(), func(context.Context) error { return fmt.Errorf("operation: %w", context.Canceled) })
	if !errors.Is(err, retry.ErrCanceled) || !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
