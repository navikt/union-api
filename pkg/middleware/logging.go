package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// NewRequestLogger returns middleware that emits one structured slog record per
// HTTP request. It deliberately uses slog rather than chi's built-in text
// Logger so access logs share the same (optionally JSON) format as the rest of
// the service and stay parseable by log aggregators.
//
// It relies on chi's RequestID middleware running earlier in the chain to
// populate the request_id field; if absent, that field is simply empty.
func NewRequestLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			defer func() {
				status := ww.Status()
				if status == 0 {
					// Handler returned without writing a header; net/http will
					// send 200 OK implicitly.
					status = http.StatusOK
				}
				// GetClientIP is populated by a ClientIPFrom* middleware when one
				// is wired for the deployment's proxy topology; fall back to the
				// raw TCP peer address otherwise. We never trust X-Forwarded-For
				// directly here, so a spoofed header cannot poison the logs.
				clientIP := chimw.GetClientIP(r.Context())
				if clientIP == "" {
					clientIP = r.RemoteAddr
				}
				slog.LogAttrs(r.Context(), slog.LevelInfo, "http request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", status),
					slog.Int("bytes", ww.BytesWritten()),
					slog.Duration("duration", time.Since(start)),
					slog.String("request_id", chimw.GetReqID(r.Context())),
					slog.String("client_ip", clientIP),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
