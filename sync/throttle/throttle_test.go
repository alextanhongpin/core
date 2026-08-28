package throttle

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/evaltest"
)

// ThrottleInput describes a scenario for testing the Throttler.Do method.
type ThrottleInput struct {
	Limit            int `yaml:"limit"`
	BacklogLimit     int `yaml:"backlog_limit"`
	BacklogTimeoutMs int `yaml:"backlog_timeout_ms"`
	Calls            int `yaml:"calls"`
}

// ThrottleOutput captures the error from the last Do call.
type ThrottleOutput struct {
	LastErr string `json:"last_err"`
}

// TestThrottleDo uses evaltest to run data-driven scenarios for Throttler.Do.
// Each case creates a fresh Throttler with the given limits and performs
// `Calls` consecutive Do invocations. The test asserts the error from the
// final call.
func TestThrottleDo(t *testing.T) {
	evaltest.Run(t, func(t *testing.T, ctx context.Context, input ThrottleInput) (ThrottleOutput, error) {
		cfg := Config{
			Limit:          input.Limit,
			BacklogLimit:   input.BacklogLimit,
			BacklogTimeout: time.Duration(input.BacklogTimeoutMs) * time.Millisecond,
		}
		if err := cfg.Validate(); err != nil {
			return ThrottleOutput{}, err
		}

		th := New(&cfg)
		if input.Calls == 0 {
			return ThrottleOutput{}, nil
		}

		errs := make([]error, input.Calls)
		var wg sync.WaitGroup
		release := make(chan struct{})

		initialHold := min(input.Limit, input.Calls)
		var started sync.WaitGroup
		started.Add(initialHold)

		for i := 0; i < initialHold; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				errs[idx] = th.Do(context.Background(), func(ctx context.Context) error {
					started.Done()
					<-release
					return nil
				})
			}(i)
		}

		started.Wait()

		queuedInBacklog := min(input.BacklogLimit, max(0, input.Calls-input.Limit))
		for i := 0; i < queuedInBacklog; i++ {
			idx := input.Limit + i
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				errs[idx] = th.Do(context.Background(), func(ctx context.Context) error {
					return nil
				})
			}(idx)

			expectedBacklog := (input.Limit + input.BacklogLimit) - (idx + 1)
			for len(th.backlogCh) > expectedBacklog {
				time.Sleep(time.Millisecond)
			}
		}

		for i := input.Limit + queuedInBacklog; i < input.Calls; i++ {
			errs[i] = th.Do(context.Background(), func(ctx context.Context) error {
				return nil
			})
		}

		close(release)
		wg.Wait()

		var lastErrStr string
		if lastErr := errs[input.Calls-1]; lastErr != nil {
			lastErrStr = lastErr.Error()
		}

		return ThrottleOutput{
			LastErr: lastErrStr,
		}, nil
	})
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Limit:          10,
				BacklogLimit:   5,
				BacklogTimeout: time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid limit zero",
			cfg: Config{
				Limit: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid limit negative",
			cfg: Config{
				Limit: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid backlog limit negative",
			cfg: Config{
				Limit:        1,
				BacklogLimit: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid backlog timeout negative",
			cfg: Config{
				Limit:          1,
				BacklogTimeout: -1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOptions(t *testing.T) {
	cfg := DefaultConfig()
	WithLimit(50)(cfg)
	WithBacklogLimit(20)(cfg)
	WithBacklogTimeout(5 * time.Second)(cfg)

	if cfg.Limit != 50 {
		t.Fatalf("expected Limit 50, got %d", cfg.Limit)
	}
	if cfg.BacklogLimit != 20 {
		t.Fatalf("expected BacklogLimit 20, got %d", cfg.BacklogLimit)
	}
	if cfg.BacklogTimeout != 5*time.Second {
		t.Fatalf("expected BacklogTimeout 5s, got %v", cfg.BacklogTimeout)
	}
}

func TestNew_DefaultConfig(t *testing.T) {
	th := New(nil)
	if th == nil {
		t.Fatal("expected non-nil throttler")
	}
	if th.Limit != 1000 || th.BacklogLimit != 100 || th.BacklogTimeout != 10*time.Second {
		t.Fatalf("expected default config, got %+v", th.Config)
	}
}

func TestNew_PanicOnInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected New to panic on invalid config")
		}
	}()

	New(&Config{Limit: 0})
}

func TestThrottler_Do_FnError(t *testing.T) {
	th := New(&Config{
		Limit:          1,
		BacklogLimit:   0,
		BacklogTimeout: time.Second,
	})

	expectedErr := errors.New("custom work error")
	err := th.Do(context.Background(), func(ctx context.Context) error {
		return expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	// Verify token was released and subsequent call succeeds
	err = th.Do(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected subsequent call to succeed, got %v", err)
	}
}

func TestThrottler_Do_ContextCancelled(t *testing.T) {
	th := New(&Config{
		Limit:          1,
		BacklogLimit:   0,
		BacklogTimeout: time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := th.Do(ctx, func(ctx context.Context) error {
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestThrottler_Do_BacklogTimeout(t *testing.T) {
	th := New(&Config{
		Limit:          1,
		BacklogLimit:   1,
		BacklogTimeout: 20 * time.Millisecond,
	})

	started := make(chan struct{})
	release := make(chan struct{})

	go func() {
		_ = th.Do(context.Background(), func(ctx context.Context) error {
			close(started)
			<-release
			return nil
		})
	}()

	<-started

	// Second call enters backlog and times out waiting for primary token
	err := th.Do(context.Background(), func(ctx context.Context) error {
		return nil
	})

	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got %v", err)
	}

	close(release)
}

func TestFunc(t *testing.T) {
	th := New(&Config{
		Limit:          1,
		BacklogLimit:   0,
		BacklogTimeout: time.Second,
	})

	called := 0
	fn := Func(func(ctx context.Context, req int) (string, error) {
		called++
		return fmt.Sprintf("result-%d", req), nil
	}, th)

	res, err := fn(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != "result-42" {
		t.Fatalf("expected 'result-42', got %q", res)
	}
	if called != 1 {
		t.Fatalf("expected called 1, got %d", called)
	}

	// Test error propagation
	customErr := errors.New("fn error")
	errFn := Func(func(ctx context.Context, req int) (string, error) {
		return "", customErr
	}, th)

	_, err = errFn(context.Background(), 1)
	if !errors.Is(err, customErr) {
		t.Fatalf("expected %v, got %v", customErr, err)
	}
}
