package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/navikt/union-api/pkg/config"
)

type contextKey string

const principalKey contextKey = "principal"

// Principal holds the identity of the authenticated user.
type Principal struct {
	Email string
	Name  string
}

// sessionClaims is the payload stored in the signed session cookie.
type sessionClaims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Exp   int64  `json:"exp"` // Unix timestamp
}

// CreateSessionToken creates a compact HMAC-SHA256 signed session token from
// the given claims. The returned string is safe to store in a cookie.
// Format: base64url(json payload) + "." + base64url(hmac signature)
func CreateSessionToken(secret []byte, email, name string, expiry time.Time) (string, error) {
	claims := sessionClaims{Email: email, Name: name, Exp: expiry.Unix()}
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session claims: %w", err)
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(encodedPayload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return encodedPayload + "." + sig, nil
}

func verifySessionToken(secret []byte, token string) (*sessionClaims, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Constant-time comparison to prevent timing attacks.
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(parts[0]))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expectedSig), []byte(parts[1])) {
		return nil, fmt.Errorf("invalid signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid payload encoding: %w", err)
	}

	var claims sessionClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	if time.Now().After(time.Unix(claims.Exp, 0)) {
		return nil, fmt.Errorf("session expired")
	}

	return &claims, nil
}

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
				ctx := context.WithValue(r.Context(), principalKey, &Principal{
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

			claims, err := verifySessionToken(secret, cookie.Value)
			if err != nil {
				slog.Warn("session: token verification failed, redirecting to login", "err", err)
				redirectToLogin(w, r)
				return
			}

			slog.Info("session: authenticated", "email", claims.Email)

			ctx := context.WithValue(r.Context(), principalKey, &Principal{
				Email: claims.Email,
				Name:  claims.Name,
			})
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
