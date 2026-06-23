package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// shutdownTimeout bounds graceful draining of in-flight requests on SIGTERM.
// It sits above the per-request Timeout middleware and under the Kubernetes
// default termination grace period (30s) so the process exits cleanly on its own.
const shutdownTimeout = 25 * time.Second

func NewServer(handler http.Handler) (*http.Server, error) {
	return &http.Server{
		Addr:    ":8080",
		Handler: handler,
		// Timeouts protect against slow or stalled clients (e.g. slowloris).
		// WriteTimeout sits above the request Timeout middleware so the handler,
		// not the server, controls when a slow upstream call is cut off.
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      25 * time.Second,
		IdleTimeout:       60 * time.Second,
	}, nil
}

// Run starts the HTTP server and blocks until it exits or a termination signal
// is received, in which case it drains in-flight requests within shutdownTimeout.
// It returns nil on a clean shutdown.
func Run(srv *http.Server) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", srv.Addr)
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		slog.Info("shutdown signal received, draining connections")
		stop() // restore default signal handling; a second signal hard-kills.

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
		slog.Info("server stopped cleanly")
		return nil
	}
}
