package middleware

import (
	"net/http"
	"runtime/debug"
	"time"

	"golang.org/x/exp/slog"
)

func LogRequest(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ctx := r.Context()
			reqID, _ := RequestIDContext.Value(ctx)
			rw := newStatusResponseWriter(w)

			args := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.EscapedPath()),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", reqID),
				slog.String("user_agent", r.Header.Get("User-Agent")),
				slog.String("ip", r.RemoteAddr),
			}

			defer func() {
				if err := recover(); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					args = append(args,
						slog.Any("err", err),
						slog.String("trace", string(debug.Stack())),
					)
					logger.Error("server panic", args...)
				}
			}()

			next.ServeHTTP(rw, r)
			args = append(args, slog.Int("status", rw.statusCode))

			logger.With(args...).Info("server request")
		}

		return http.HandlerFunc(fn)
	}
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode    int
	headerWritten bool
}

func newStatusResponseWriter(w http.ResponseWriter) *statusResponseWriter {
	return &statusResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (w *statusResponseWriter) WriteHeader(statusCode int) {
	if w.headerWritten {
		return
	}

	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
	w.headerWritten = true

}

func (mw *statusResponseWriter) Write(b []byte) (int, error) {
	mw.headerWritten = true
	return mw.ResponseWriter.Write(b)
}

func (mw *statusResponseWriter) Unwrap() http.ResponseWriter {
	return mw.ResponseWriter
}
