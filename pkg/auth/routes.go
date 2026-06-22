package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Router returns a handler for the OAuth2 login flow.
// Mount this at /oauth2 to expose /oauth2/login and /oauth2/callback.
func Router(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/login", h.Login)
	r.Get("/callback", h.Callback)
	return r
}
