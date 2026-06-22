package serviceaccounts

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navikt/union-api/pkg/config"
	"github.com/navikt/union-api/pkg/middleware"
)

func Router(cfg *config.Config, h Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.NewSessionMiddleware(cfg))
	r.Get("/", h.GetServiceAccounts)
	return r
}
