package main

import (
	"context"
	"fmt"
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
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	r := chi.NewRouter()

	if !cfg.DevMode {
		auth, err := handlers.NewAuthHandler(context.Background(), cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialise auth handler: %v\n", err)
			os.Exit(1)
		}
		r.Mount("/oauth2", routes.OAuthRouter(auth))
	}

	r.Mount("/serviceaccounts", routes.ServiceAccountsRouter(cfg))

	srv, err := server.NewServer(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialise server: %v\n", err)
		os.Exit(1)
	}

	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
