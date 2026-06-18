package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/navikt/union-api/pkg/config"
	"golang.org/x/oauth2"
)

const (
	sessionCookieName = "session"
	stateCookieName   = "oauth_state"
)

// oauthState is encoded in the OAuth2 state parameter to carry both the CSRF
// nonce and the URL to redirect back to after a successful login.
type oauthState struct {
	Nonce    string `json:"nonce"`
	Redirect string `json:"redirect"`
}

// AuthHandler holds the OIDC provider and OAuth2 config used by the login and
// callback handlers.
type AuthHandler struct {
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier
}

// Verifier returns the OIDC ID token verifier, for use by middleware that
// needs to validate tokens outside of this handler.
func (a *AuthHandler) Verifier() *oidc.IDTokenVerifier {
	return a.verifier
}

// NewAuthHandler creates an AuthHandler by discovering the EntraID OIDC
// endpoints and building the OAuth2 config. It must be called after the
// provider is reachable (i.e. at server startup, not in init()).
func NewAuthHandler(ctx context.Context, cfg *config.Config) (*AuthHandler, error) {
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL())
	if err != nil {
		return nil, err
	}

	oauth2Config := &oauth2.Config{
		ClientID:     cfg.EntraIDClientID,
		ClientSecret: cfg.EntraIDClientSecret,
		RedirectURL:  cfg.RedirectURL(),
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.EntraIDClientID})

	return &AuthHandler{
		oauth2Config: oauth2Config,
		verifier:     verifier,
	}, nil
}

// Login redirects the browser to the EntraID authorization endpoint.
// It encodes the original request path and a CSRF nonce into the state parameter.
func (a *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	nonce, err := generateNonce()
	if err != nil {
		http.Error(w, "failed to generate nonce", http.StatusInternalServerError)
		return
	}

	// Capture the originally requested URL so we can send the user there after login.
	redirect := r.URL.Query().Get("redirect")
	if redirect == "" {
		redirect = "/"
	}

	stateBytes, err := json.Marshal(oauthState{Nonce: nonce, Redirect: redirect})
	if err != nil {
		http.Error(w, "failed to encode state", http.StatusInternalServerError)
		return
	}
	state := base64.URLEncoding.EncodeToString(stateBytes)

	// Store the nonce in a short-lived cookie so the callback can verify it.
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    nonce,
		Path:     "/",
		MaxAge:   int((10 * time.Minute).Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, a.oauth2Config.AuthCodeURL(state), http.StatusFound)
}

// Callback handles the redirect back from EntraID, validates the state/CSRF
// cookie, exchanges the authorization code for tokens, and stores the raw ID
// token in a session cookie before redirecting to the originally-requested URL.
func (a *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	// Validate state parameter.
	rawState := r.URL.Query().Get("state")
	if rawState == "" {
		http.Error(w, "missing state parameter", http.StatusBadRequest)
		return
	}

	stateBytes, err := base64.URLEncoding.DecodeString(rawState)
	if err != nil {
		http.Error(w, "invalid state parameter", http.StatusBadRequest)
		return
	}

	var state oauthState
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		http.Error(w, "invalid state parameter", http.StatusBadRequest)
		return
	}

	// Compare nonce from cookie to nonce in state to prevent CSRF.
	nonceCookie, err := r.Cookie(stateCookieName)
	if err != nil || nonceCookie.Value != state.Nonce {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}

	// Clear the state cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	// Exchange the authorization code for tokens.
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code parameter", http.StatusBadRequest)
		return
	}

	token, err := a.oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "failed to exchange token", http.StatusInternalServerError)
		return
	}

	// Extract and validate the raw ID token.
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "missing id_token in response", http.StatusInternalServerError)
		return
	}

	idToken, err := a.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		http.Error(w, "invalid id_token", http.StatusInternalServerError)
		return
	}

	// Store the raw ID token in the session cookie. The expiry matches the token.
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    rawIDToken,
		Path:     "/",
		Expires:  idToken.Expiry,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	redirect := state.Redirect
	if redirect == "" {
		redirect = "/"
	}
	http.Redirect(w, r, redirect, http.StatusFound)
}

func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
