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
	"github.com/navikt/union-api/pkg/uctl"
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

	r := chi.NewRouter()

	if !cfg.DevMode {
		auth, err := handlers.NewAuthHandler(context.Background(), cfg)
		if err != nil {
			slog.Error("failed to initialise auth handler", "err", err)
			os.Exit(1)
		}
		r.Mount("/oauth2", routes.OAuthRouter(auth))
	}

	uctlClient := uctl.NewUCTLClient(cfg.UnionConfig)
	saHandler := handlers.NewServiceAccountsHandler(uctlClient)
	r.Mount("/serviceaccounts", routes.ServiceAccountsRouter(cfg, saHandler))

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
