package server

import (
	"net/http"
	"time"
)

type Option interface {
	Apply(s *http.Server)
}

type WriteTimeout time.Duration

func (r WriteTimeout) Apply(s *http.Server) {
	s.WriteTimeout = time.Duration(r)
}

type ReadTimeout time.Duration

func (r ReadTimeout) Apply(s *http.Server) {
	s.ReadTimeout = time.Duration(r)
}

type ReadHeaderTimeout time.Duration

func (r ReadHeaderTimeout) Apply(s *http.Server) {
	s.ReadHeaderTimeout = time.Duration(r)
}

type MaxBytes int64

func (r MaxBytes) Apply(s *http.Server) {
	s.Handler = http.MaxBytesHandler(s.Handler, int64(r))
}

type Timeout struct {
	Duration time.Duration
	Message  string
}

func (r Timeout) Apply(s *http.Server) {
	s.Handler = http.TimeoutHandler(s.Handler, r.Duration, r.Message)
}
