package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navikt/union-api/pkg/handlers"
)

// OAuthRouter returns a handler for the OAuth2 login flow.
// Mount this at /oauth2 to expose /oauth2/login and /oauth2/callback.
func OAuthRouter(auth *handlers.AuthHandler) http.Handler {
	r := chi.NewRouter()
	r.Get("/login", auth.Login)
	r.Get("/callback", auth.Callback)
	return r
}
