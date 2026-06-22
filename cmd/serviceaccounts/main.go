package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/navikt/union-api/pkg/auth"
	"github.com/navikt/union-api/pkg/config"
	"github.com/navikt/union-api/pkg/k8s"
	"github.com/navikt/union-api/pkg/server"
	"github.com/navikt/union-api/pkg/serviceaccounts"
	"github.com/navikt/union-api/pkg/uctl"
	"github.com/navikt/union-api/web"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("configuration error", "err", err)
		os.Exit(1)
	}

	if cfg.LogFormat == "json" {
		handler := slog.NewJSONHandler(os.Stdout, nil)
		slog.SetDefault(slog.New(handler))
	}

	slog.SetLogLoggerLevel(cfg.LogLevel)

	renderer, err := web.New()
	if err != nil {
		slog.Error("failed to initialise renderer", "err", err)
		os.Exit(1)
	}

	r := chi.NewRouter()

	// Static files are served without session middleware so CSS loads correctly
	// on the login-redirect path.
	r.Handle("/static/*", http.StripPrefix("/static", http.FileServer(http.FS(web.StaticFS))))

	// Redirect root to the service accounts page.
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/serviceaccounts", http.StatusFound)
	})

	if !cfg.DevMode {
		authHandler, err := auth.NewHandler(context.Background(), cfg)
		if err != nil {
			slog.Error("failed to initialize auth handler", "err", err)
			os.Exit(1)
		}
		r.Mount("/oauth2", auth.Router(authHandler))
	}

	uctlClient := uctl.NewUCTLClient(cfg.UnionConfig)
	k8sClient, err := k8s.NewK8sClient(context.Background(), cfg.K8sConfig)
	if err != nil {
		slog.Error("failed to initialize k8s client", "err", err)
	}
	saService := serviceaccounts.NewService(uctlClient, k8sClient)
	saHandler := serviceaccounts.NewHandler(saService, renderer)

	r.Mount("/serviceaccounts", serviceaccounts.Router(cfg, saHandler))

	srv, err := server.NewServer(r)
	if err != nil {
		slog.Error("failed to initialise server", "err", err)
		os.Exit(1)
	}

	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
