package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/navikt/union-api/pkg/config"
	"github.com/navikt/union-api/pkg/middleware"
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

type AuthHandler struct {
	oauth2Config  *oauth2.Config
	verifier      *oidc.IDTokenVerifier
	secureCookies bool
	sessionSecret []byte
}

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
		oauth2Config:  oauth2Config,
		verifier:      verifier,
		secureCookies: cfg.SecureCookies(),
		sessionSecret: []byte(cfg.SessionSecret),
	}, nil
}

// Login redirects the browser to the EntraID authorization endpoint.
// It encodes the original request path and a CSRF nonce into the state parameter.
func (a *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	nonce, err := generateNonce()
	if err != nil {
		log.Printf("login: failed to generate nonce: %v", err)
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
		log.Printf("login: failed to encode state: %v", err)
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
	log.Printf("login: redirecting to EntraID (redirect_after_login=%q)", redirect)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback handles the redirect back from EntraID, validates the state/CSRF
// cookie, exchanges the authorization code for tokens, and stores the raw ID
// token in a session cookie before redirecting to the originally-requested URL.
func (a *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	// Validate state parameter.
	rawState := r.URL.Query().Get("state")
	if rawState == "" {
		log.Printf("callback: missing state parameter")
		http.Error(w, "missing state parameter", http.StatusBadRequest)
		return
	}

	stateBytes, err := base64.URLEncoding.DecodeString(rawState)
	if err != nil {
		log.Printf("callback: failed to decode state: %v", err)
		http.Error(w, "invalid state parameter", http.StatusBadRequest)
		return
	}

	var state oauthState
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		log.Printf("callback: failed to unmarshal state: %v", err)
		http.Error(w, "invalid state parameter", http.StatusBadRequest)
		return
	}

	// Compare nonce from cookie to nonce in state to prevent CSRF.
	nonceCookie, err := r.Cookie(stateCookieName)
	if err != nil {
		log.Printf("callback: oauth_state cookie missing: %v", err)
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}
	if nonceCookie.Value != state.Nonce {
		log.Printf("callback: nonce mismatch (cookie=%q state=%q)", nonceCookie.Value, state.Nonce)
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
		log.Printf("callback: missing code parameter")
		http.Error(w, "missing code parameter", http.StatusBadRequest)
		return
	}

	log.Printf("callback: exchanging code for tokens")
	token, err := a.oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		log.Printf("callback: token exchange failed: %v", err)
		http.Error(w, "failed to exchange token", http.StatusInternalServerError)
		return
	}

	// Extract and validate the raw ID token.
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		log.Printf("callback: id_token missing from token response")
		http.Error(w, "missing id_token in response", http.StatusInternalServerError)
		return
	}

	log.Printf("callback: verifying id_token (len=%d)", len(rawIDToken))
	idToken, err := a.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		log.Printf("callback: id_token verification failed: %v", err)
		http.Error(w, "invalid id_token", http.StatusInternalServerError)
		return
	}

	// Extract only the claims we need from the (large) OIDC token.
	var oidcClaims struct {
		PreferredUsername string `json:"preferred_username"`
		Name              string `json:"name"`
	}
	if err := idToken.Claims(&oidcClaims); err != nil {
		log.Printf("callback: failed to parse id_token claims: %v", err)
		http.Error(w, "failed to parse token claims", http.StatusInternalServerError)
		return
	}

	// Create a compact signed session token — the raw ID token is too large
	// for a browser cookie (EntraID tokens can exceed 4 KB).
	sessionToken, err := middleware.CreateSessionToken(
		a.sessionSecret,
		oidcClaims.PreferredUsername,
		oidcClaims.Name,
		idToken.Expiry,
	)
	if err != nil {
		log.Printf("callback: failed to create session token: %v", err)
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
	log.Printf("callback: success, redirecting to %q", redirect)
	http.Redirect(w, r, redirect, http.StatusFound)
}

func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
