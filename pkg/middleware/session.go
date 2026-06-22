package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/navikt/union-api/pkg/auth"
	"github.com/navikt/union-api/pkg/config"
)

type contextKey string

const principalKey contextKey = "principal"

// NewSessionMiddleware returns a middleware that enforces authentication.
//
// In dev mode (cfg.DevMode == true) it bypasses all token validation and
// injects a stub principal so the service can be run locally without
// EntraID credentials. Never enable dev mode in production.
//
// In production mode it reads the session cookie, verifies its HMAC signature
// and expiry, and injects the resulting Principal into the request context.
// Requests without a valid token are redirected to /oauth2/login.
func NewSessionMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	secret := []byte(cfg.SessionSecret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.DevMode {
				ctx := context.WithValue(r.Context(), principalKey, &auth.Principal{
					Email: "dev.user@nav.no",
					Name:  "Dev User",
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			cookie, err := r.Cookie("session")
			if err != nil {
				slog.Info("session: no session cookie, redirecting to login", "uri", r.RequestURI)
				redirectToLogin(w, r)
				return
			}

			principal, err := auth.VerifySessionToken(secret, cookie.Value)
			if err != nil {
				slog.Warn("session: token verification failed, redirecting to login", "err", err)
				redirectToLogin(w, r)
				return
			}

			slog.Info("session: authenticated", "email", principal.Email)

			ctx := context.WithValue(r.Context(), principalKey, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// PrincipalFromContext retrieves the authenticated Principal from the context.
// Returns nil, false if no principal is present.
func PrincipalFromContext(ctx context.Context) (*auth.Principal, bool) {
	p, ok := ctx.Value(principalKey).(*auth.Principal)
	return p, ok
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	loginURL := "/oauth2/login?redirect=" + url.QueryEscape(r.RequestURI)
	http.Redirect(w, r, loginURL, http.StatusFound)
}
