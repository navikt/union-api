package routes

import (
	"net/http"
	"github.com/go-chi/chi/v5"
)

func ServiceAccountsRouter() http.Handler {
	r := chi.NewRouter()
	r.Route("/serviceaccounts", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Service Accounts"))
		})
	})

	return r
}