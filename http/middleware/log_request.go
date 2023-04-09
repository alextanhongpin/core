package middleware

import (
	"io"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
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

			var body []byte
			if r.Body != nil {
				body, _ = io.ReadAll(r.Body)
			}

			args := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.EscapedPath()),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", reqID),
				slog.String("user_agent", r.Header.Get("User-Agent")),
				slog.String("ip", clientIPFromRequest(r)),
				slog.String("body", string(body)),
				slog.Any("query", r.URL.Query()),
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

			switch {
			case rw.statusCode >= 200 && rw.statusCode < 300:
				logger.With(args...).Info("request success")
			case
				rw.statusCode == 400,
				rw.statusCode == 401:
				// Ignore bad request and unauthorized.
			case rw.statusCode > 401 && rw.statusCode < 500:
				args = append(args, slog.String("err", string(rw.body)))
				logger.With(args...).Error("request failed")
			default:
				// Ignore the rest.
			}
		}

		return http.HandlerFunc(fn)
	}
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode    int
	body          []byte
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

func (w *statusResponseWriter) Write(b []byte) (int, error) {
	w.headerWritten = true
	w.body = b
	return w.ResponseWriter.Write(b)
}

func (w *statusResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// The implementation is similar to gin-gonic's .ClientIP method.
func clientIPFromRequest(r *http.Request) string {
	clientIP := r.Header.Get("X-Forwarded-For")
	clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0])
	if clientIP == "" {
		clientIP = strings.TrimSpace(r.Header.Get("X-Real-Ip"))
	}

	if clientIP != "" {
		return clientIP
	}

	if ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err != nil {
		return ip
	}

	return ""
}
