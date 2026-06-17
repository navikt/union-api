package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navikt/union-api/pkg/middleware"
)

func ServiceAccountsRouter() http.Handler {
	r := chi.NewRouter()
	r.Route("/serviceaccounts", func(r chi.Router) {
		r.Use(middleware.SessionMiddleware)
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Service Accounts"))
		})
	})

	return r
}