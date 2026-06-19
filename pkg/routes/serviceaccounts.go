package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navikt/union-api/pkg/config"
	"github.com/navikt/union-api/pkg/handlers"
	"github.com/navikt/union-api/pkg/middleware"
)

func ServiceAccountsRouter(cfg *config.Config, saHandler handlers.ServiceAccountsHandler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.NewSessionMiddleware(cfg))
	r.Get("/", saHandler.GetServiceAccounts)
	return r
}
