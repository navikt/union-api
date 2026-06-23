package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log/slog"
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

type Handler struct {
	oauth2Config  *oauth2.Config
	verifier      *oidc.IDTokenVerifier
	secureCookies bool
	sessionSecret []byte
}

func (a *Handler) Verifier() *oidc.IDTokenVerifier {
	return a.verifier
}

// NewHandler creates a Handler by discovering the EntraID OIDC
// endpoints and building the OAuth2 config. It must be called after the
// provider is reachable (i.e. at server startup, not in init()).
func NewHandler(ctx context.Context, cfg *config.Config) (*Handler, error) {
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL())
	if err != nil {
		return nil, err
	}

	oauth2Config := &oauth2.Config{
		ClientID:     cfg.EntraID.ClientID,
		ClientSecret: cfg.EntraID.ClientSecret,
		RedirectURL:  cfg.RedirectURL(),
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.EntraID.ClientID})

	return &Handler{
		oauth2Config:  oauth2Config,
		verifier:      verifier,
		secureCookies: cfg.SecureCookies(),
		sessionSecret: []byte(cfg.SessionSecret),
	}, nil
}

// Login redirects the browser to the EntraID authorization endpoint.
// It encodes the original request path and a CSRF nonce into the state parameter.
func (a *Handler) Login(w http.ResponseWriter, r *http.Request) {
	nonce, err := generateNonce()
	if err != nil {
		slog.Error("login: failed to generate nonce", "err", err)
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
		slog.Error("login: failed to encode state", "err", err)
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
		Secure:   a.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	authURL := a.oauth2Config.AuthCodeURL(state)
	slog.Info("login: redirecting to EntraID", "redirect_after_login", redirect)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback handles the redirect back from EntraID, validates the state/CSRF
// cookie, exchanges the authorization code for tokens, and stores the raw ID
// token in a session cookie before redirecting to the originally-requested URL.
func (a *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	// Validate state parameter.
	rawState := r.URL.Query().Get("state")
	if rawState == "" {
		slog.Warn("callback: missing state parameter")
		http.Error(w, "missing state parameter", http.StatusBadRequest)
		return
	}

	stateBytes, err := base64.URLEncoding.DecodeString(rawState)
	if err != nil {
		slog.Warn("callback: failed to decode state", "err", err)
		http.Error(w, "invalid state parameter", http.StatusBadRequest)
		return
	}

	var state oauthState
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		slog.Warn("callback: failed to unmarshal state", "err", err)
		http.Error(w, "invalid state parameter", http.StatusBadRequest)
		return
	}

	// Compare nonce from cookie to nonce in state to prevent CSRF.
	nonceCookie, err := r.Cookie(stateCookieName)
	if err != nil {
		slog.Warn("callback: oauth_state cookie missing", "err", err)
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}
	if nonceCookie.Value != state.Nonce {
		slog.Warn("callback: nonce mismatch", "cookie", nonceCookie.Value, "state", state.Nonce)
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
		Secure:   a.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	// Exchange the authorization code for tokens.
	code := r.URL.Query().Get("code")
	if code == "" {
		slog.Warn("callback: missing code parameter")
		http.Error(w, "missing code parameter", http.StatusBadRequest)
		return
	}

	slog.Info("callback: exchanging code for tokens")
	token, err := a.oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		slog.Error("callback: token exchange failed", "err", err)
		http.Error(w, "failed to exchange token", http.StatusInternalServerError)
		return
	}

	// Extract and validate the raw ID token.
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		slog.Error("callback: id_token missing from token response")
		http.Error(w, "missing id_token in response", http.StatusInternalServerError)
		return
	}

	slog.Info("callback: verifying id_token", "len", len(rawIDToken))
	idToken, err := a.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		slog.Error("callback: id_token verification failed", "err", err)
		http.Error(w, "invalid id_token", http.StatusInternalServerError)
		return
	}

	// Extract only the claims we need from the (large) OIDC token.
	var oidcClaims struct {
		PreferredUsername string `json:"preferred_username"`
		Name              string `json:"name"`
	}
	if err := idToken.Claims(&oidcClaims); err != nil {
		slog.Error("callback: failed to parse id_token claims", "err", err)
		http.Error(w, "failed to parse token claims", http.StatusInternalServerError)
		return
	}

	// Create a compact signed session token — the raw ID token is too large
	// for a browser cookie (EntraID tokens can exceed 4 KB).
	sessionToken, err := CreateSessionToken(
		a.sessionSecret,
		oidcClaims.PreferredUsername,
		oidcClaims.Name,
		idToken.Expiry,
	)
	if err != nil {
		slog.Error("callback: failed to create session token", "err", err)
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	// Store the compact session token in the cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionToken,
		Path:     "/",
		Expires:  idToken.Expiry,
		HttpOnly: true,
		Secure:   a.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	redirect := state.Redirect
	if redirect == "" {
		redirect = "/"
	}
	slog.Info("callback: success, redirecting", "redirect", redirect)
	http.Redirect(w, r, redirect, http.StatusFound)
}

func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
