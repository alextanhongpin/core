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

func LogRequest(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ctx := r.Context()
			reqID, _ := RequestIDContext.Value(ctx)
			rw := newStatusResponseWriter(w)

			body, _ := io.ReadAll(r.Body)

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
