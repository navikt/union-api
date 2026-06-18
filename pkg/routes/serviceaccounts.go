package routes

import (
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/navikt/union-api/pkg/config"
	"github.com/navikt/union-api/pkg/handlers"
	"github.com/navikt/union-api/pkg/middleware"
)

func ServiceAccountsRouter(cfg *config.Config, verifier *oidc.IDTokenVerifier) http.Handler {
	r := chi.NewRouter()

	sessionMiddleware := middleware.NewSessionMiddleware(cfg, verifier)

	r.Use(sessionMiddleware)
	r.Get("/", handlers.ServiceAccountsHandler)

	return r
}
