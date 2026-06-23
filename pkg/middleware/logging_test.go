package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	chimw "github.com/go-chi/chi/v5/middleware"
)

func TestNewRequestLogger(t *testing.T) {
	// Not parallel: this test swaps the global default slog logger.
	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(orig) })

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	})

	req := httptest.NewRequest(http.MethodGet, "/serviceaccounts", nil)
	rr := httptest.NewRecorder()
	NewRequestLogger()(next).ServeHTTP(rr, req)

	if !called {
		t.Fatal("next handler was not called")
	}
	if rr.Code != http.StatusTeapot {
		t.Errorf("status passthrough: got %d, want %d", rr.Code, http.StatusTeapot)
	}

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("log output is not valid JSON: %v\n%s", err, buf.String())
	}
	if rec["msg"] != "http request" {
		t.Errorf("msg: got %v, want %q", rec["msg"], "http request")
	}
	if rec["method"] != http.MethodGet {
		t.Errorf("method: got %v, want %q", rec["method"], http.MethodGet)
	}
	if rec["path"] != "/serviceaccounts" {
		t.Errorf("path: got %v, want %q", rec["path"], "/serviceaccounts")
	}
	// JSON numbers decode to float64.
	if rec["status"] != float64(http.StatusTeapot) {
		t.Errorf("status: got %v, want %d", rec["status"], http.StatusTeapot)
	}
}

// TestNewRequestLogger_ClientIP verifies the logger records the trustworthy
// client IP resolved by a ClientIPFrom* middleware rather than a spoofable
// X-Forwarded-For value. Behind a GCP external load balancer the header is
// "[<spoofed>,] <client-ip>, <lb-ip>", so with 2 trusted hops the second-from-
// right entry (the real client) must win.
func TestNewRequestLogger_ClientIP(t *testing.T) {
	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(orig) })

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// Mirror the production chain: resolve client IP, then log it.
	handler := chimw.ClientIPFromXFFTrustedProxies(2)(NewRequestLogger()(next))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "9.9.9.9, 1.1.1.1, 10.0.0.1")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("log output is not valid JSON: %v\n%s", err, buf.String())
	}
	if rec["client_ip"] != "1.1.1.1" {
		t.Errorf("client_ip: got %v, want %q (spoofed 9.9.9.9 must be ignored)", rec["client_ip"], "1.1.1.1")
	}
}
