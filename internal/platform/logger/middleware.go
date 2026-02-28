package logger

import (
	"log/slog"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code and size.
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

// RequestLogger returns a chi-compatible middleware that logs each request
// with method, path, status, duration_ms, and response size.
func RequestLogger(log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrap := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(wrap, r)
			dur := time.Since(start)
			log.Info("request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", wrap.status),
				slog.Int("duration_ms", int(dur.Milliseconds())),
				slog.Int("size", wrap.size),
			)
		})
	}
}
