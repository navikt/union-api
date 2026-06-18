package middleware

import (
	"context"
	"net/http"
	"net/url"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/navikt/union-api/pkg/config"
)

type contextKey string

const principalKey contextKey = "principal"

// Principal holds the identity of the authenticated user.
type Principal struct {
	Email string
	Name  string
}

// entraidClaims maps the EntraID ID token claims we care about.
type entraidClaims struct {
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name"`
}

// NewSessionMiddleware returns a middleware that enforces authentication.
//
// In dev mode (cfg.DevMode == true) it bypasses all token validation and
// injects a stub principal so the service can be run locally without
// EntraID credentials. Never enable dev mode in production.
//
// In production mode it reads the raw ID token from the "session" cookie,
// validates it against the EntraID JWKS (via the provided verifier), and
// injects the resulting Principal into the request context. Requests without
// a valid token are redirected to /login.
func NewSessionMiddleware(cfg *config.Config, verifier *oidc.IDTokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.DevMode {
				ctx := context.WithValue(r.Context(), principalKey, &Principal{
					Email: "dev.user@nav.no",
					Name:  "Dev User",
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			cookie, err := r.Cookie("session")
			if err != nil {
				redirectToLogin(w, r)
				return
			}

			idToken, err := verifier.Verify(r.Context(), cookie.Value)
			if err != nil {
				redirectToLogin(w, r)
				return
			}

			var claims entraidClaims
			if err := idToken.Claims(&claims); err != nil {
				http.Error(w, "failed to parse token claims", http.StatusInternalServerError)
				return
			}

			principal := &Principal{
				Email: claims.PreferredUsername,
				Name:  claims.Name,
			}

			ctx := context.WithValue(r.Context(), principalKey, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// PrincipalFromContext retrieves the authenticated Principal from the context.
// Returns nil, false if no principal is present.
func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(principalKey).(*Principal)
	return p, ok
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	loginURL := "/oauth2/login?redirect=" + url.QueryEscape(r.RequestURI)
	http.Redirect(w, r, loginURL, http.StatusFound)
}
