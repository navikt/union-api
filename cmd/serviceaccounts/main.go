package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/navikt/union-api/pkg/config"
	"github.com/navikt/union-api/pkg/handlers"
	"github.com/navikt/union-api/pkg/routes"
	"github.com/navikt/union-api/pkg/server"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("configuration error", "err", err)
		os.Exit(1)
	}

	var handler slog.Handler
	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, nil)
		slog.SetDefault(slog.New(handler))
	}

	r := chi.NewRouter()

	if !cfg.DevMode {
		auth, err := handlers.NewAuthHandler(context.Background(), cfg)
		if err != nil {
			slog.Error("failed to initialise auth handler", "err", err)
			os.Exit(1)
		}
		r.Mount("/oauth2", routes.OAuthRouter(auth))
	}

	r.Mount("/serviceaccounts", routes.ServiceAccountsRouter(cfg))

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
