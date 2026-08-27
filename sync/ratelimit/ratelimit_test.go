package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alextanhongpin/evaltest"
)

func TestFixedWindow(t *testing.T) {
	evaltest.Run(t, func(t *testing.T, ctx context.Context, input FixedWindowInput) (FixedWindowOutput, error) {
		cfg := &Config{
			Limit:  input.Limit,
			Period: input.Period,
			Burst:  input.Burst,
		}
		if err := cfg.Validate(); err != nil {
			return FixedWindowOutput{}, err
		}
		r := NewFixedWindow(cfg)
		switch input.Action {
		case "Allow":
			res := r.Limit(input.Key)
			return FixedWindowOutput{Allow: res.Allow, Remaining: res.Remaining, ResetAfter: res.ResetAfter}, nil
		case "AllowN":
			res := r.LimitN(input.Key, input.N)
			return FixedWindowOutput{Allow: res.Allow, Remaining: res.Remaining, ResetAfter: res.ResetAfter}, nil
		case "Limit":
			res := r.Limit(input.Key)
			return FixedWindowOutput{
				Allow:      res.Allow,
				Remaining:  res.Remaining,
				ResetAfter: res.ResetAfter,
			}, nil
		case "LimitN":
			res := r.LimitN(input.Key, input.N)
			return FixedWindowOutput{
				Allow:      res.Allow,
				Remaining:  res.Remaining,
				ResetAfter: res.ResetAfter,
			}, nil
		default:
			return FixedWindowOutput{}, nil
		}
	})
}

func TestGCRA(t *testing.T) {
	evaltest.Run(t, func(t *testing.T, ctx context.Context, input GCRAInput) (GCRAOutput, error) {
		cfg := Config{
			Limit:  input.Limit,
			Period: input.Period,
			Burst:  input.Burst,
		}
		if err := cfg.Validate(); err != nil {
			return GCRAOutput{}, err
		}
		r := NewGCRA(&cfg)
		switch input.Action {
		case "Allow":
			res := r.Limit(input.Key)
			return GCRAOutput{Allow: res.Allow, Remaining: res.Remaining, ResetAfter: res.ResetAfter}, nil
		case "AllowN":
			res := r.LimitN(input.Key, input.N)
			return GCRAOutput{Allow: res.Allow, Remaining: res.Remaining, ResetAfter: res.ResetAfter}, nil
		case "Limit":
			res := r.Limit(input.Key)
			return GCRAOutput{
				Allow:      res.Allow,
				Remaining:  res.Remaining,
				ResetAfter: res.ResetAfter,
			}, nil
		case "LimitN":
			res := r.LimitN(input.Key, input.N)
			return GCRAOutput{
				Allow:      res.Allow,
				Remaining:  res.Remaining,
				ResetAfter: res.ResetAfter,
			}, nil
		default:
			return GCRAOutput{}, nil
		}
	})
}

type FixedWindowInput struct {
	Limit  int           `yaml:"limit"`
	Period time.Duration `yaml:"period"`
	Burst  int           `yaml:"burst"`
	Key    string        `yaml:"key"`
	Action string        `yaml:"action"`
	N      int           `yaml:"n,omitempty"`
}

type FixedWindowOutput struct {
	Allow      bool          `yaml:"allow"`
	Remaining  int           `yaml:"remaining"`
	ResetAfter time.Duration `yaml:"reset_after"`
}

type GCRAInput struct {
	Limit  int           `yaml:"limit"`
	Period time.Duration `yaml:"period"`
	Burst  int           `yaml:"burst"`
	Key    string        `yaml:"key"`
	Action string        `yaml:"action"`
	N      int           `yaml:"n,omitempty"`
}

type GCRAOutput struct {
	Allow      bool          `yaml:"allow"`
	Remaining  int           `yaml:"remaining"`
	ResetAfter time.Duration `yaml:"reset_after"`
}
