package metrics

import (
	"net/http"
)

// responseWriter captures the status code for metrics.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// RequestMiddleware returns chi-compatible middleware that records request count
// and error count (status >= 400) in the given Metrics.
func RequestMiddleware(m *Metrics) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrap := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(wrap, r)
			m.IncRequests()
			if wrap.status >= 400 {
				m.IncErrors()
			}
		})
	}
}
