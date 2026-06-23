package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/navikt/union-api/pkg/auth"
	"github.com/navikt/union-api/pkg/config"
	"github.com/navikt/union-api/pkg/health"
	"github.com/navikt/union-api/pkg/k8s"
	appmw "github.com/navikt/union-api/pkg/middleware"
	"github.com/navikt/union-api/pkg/server"
	"github.com/navikt/union-api/pkg/serviceaccounts"
	"github.com/navikt/union-api/pkg/uctl"
	"github.com/navikt/union-api/web"
)

// requestTimeout bounds processing of a single request. The deadline propagates
// through the request context into the uctl exec and Kubernetes API calls, so a
// stalled upstream is cancelled rather than tying up a connection.
const requestTimeout = 20 * time.Second

// trustedProxyHops is the number of X-Forwarded-For entries appended by trusted
// infrastructure in front of this service. A GCP external Application Load
// Balancer (what GKE Gateway API provisions) always appends two entries, in
// order:
//
//	X-Forwarded-For: [<spoofable client-supplied values>,] <client-ip>, <load-balancer-ip>
//
// so the real client IP is the second entry from the right. chi walks the
// header right-to-left and returns that entry, ignoring any attacker-supplied
// values further left. If another trusted proxy is later added in front of the
// app (e.g. a service-mesh sidecar), bump this count to match, or it will
// silently start trusting a spoofable value.
const trustedProxyHops = 2

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("configuration error", "err", err)
		os.Exit(1)
	}

	if cfg.Logging.Format == "json" {
		handler := slog.NewJSONHandler(os.Stdout, nil)
		slog.SetDefault(slog.New(handler))
	}

	slog.SetLogLoggerLevel(cfg.Logging.Level)

	renderer, err := web.New()
	if err != nil {
		slog.Error("failed to initialise renderer", "err", err)
		os.Exit(1)
	}

	var authHandler *auth.Handler
	if !cfg.DevMode {
		authHandler, err = auth.NewHandler(context.Background(), cfg)
		if err != nil {
			slog.Error("failed to initialize auth handler", "err", err)
			os.Exit(1)
		}
	}

	uctlClient := uctl.NewUCTLClient(cfg.UnionConfig)

	// The k8s client relies on GCP Application Default Credentials. In the
	// cluster these come from the workload identity; for local dev mode run
	// `gcloud auth application-default login` first. A failure here is fatal so
	// we fail fast at startup instead of nil-panicking on the first request.
	k8sClient, err := k8s.NewK8sClient(context.Background(), cfg.GKEConfig)
	if err != nil {
		slog.Error("failed to initialize k8s client", "err", err)
		os.Exit(1)
	}

	saService := serviceaccounts.NewService(uctlClient, k8sClient)
	saHandler := serviceaccounts.NewHandler(saService, renderer)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	// chi's RealIP is deprecated (IP-spoofing risk: GHSA-3fxj-6jh8-hvhx) and is
	// intentionally not used. The real client IP is resolved from
	// X-Forwarded-For for the application routes below; see trustedProxyHops.
	r.Use(chimw.Recoverer)

	// Kubernetes probes: unauthenticated and outside the request logger so the
	// kubelet's frequent polling does not flood the access logs.
	r.Get("/isalive", health.Alive)
	r.Get("/isready", health.Ready)

	// Static files are served without session middleware so CSS loads correctly
	// on the login-redirect path.
	r.Handle("/static/*", http.StripPrefix("/static", http.FileServer(http.FS(web.StaticFS))))

	// Application routes get structured access logging and a per-request timeout.
	r.Group(func(r chi.Router) {
		// Resolve the real client IP from X-Forwarded-For (set before the logger
		// so it can record it). GetClientIP falls back to the TCP peer when the
		// header is absent or shorter than expected, e.g. local dev.
		r.Use(chimw.ClientIPFromXFFTrustedProxies(trustedProxyHops))
		r.Use(appmw.NewRequestLogger())
		r.Use(chimw.Timeout(requestTimeout))

		// Redirect root to the service accounts page.
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/serviceaccounts", http.StatusFound)
		})

		if !cfg.DevMode {
			r.Mount("/oauth2", auth.Router(authHandler))
		}

		r.Mount("/serviceaccounts", serviceaccounts.Router(cfg, saHandler))
	})

	srv, err := server.NewServer(r)
	if err != nil {
		slog.Error("failed to initialise server", "err", err)
		os.Exit(1)
	}

	if err := server.Run(srv); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
