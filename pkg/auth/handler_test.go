package auth

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

// newTestHandler builds a Handler with a fake oauth2.Config that
// uses a dummy auth URL. No network calls are made during construction.
func newTestHandler() *Handler {
	return &Handler{
		oauth2Config: &oauth2.Config{
			ClientID: "test-client",
			Endpoint: oauth2.Endpoint{
				AuthURL: "https://auth.example.com/authorize",
			},
		},
		secureCookies: false,
		sessionSecret: []byte("test-session-secret-123456789abc"),
	}
}

// mustEncodeState marshals an oauthState to base64(JSON), matching the
// encoding used by the Login handler.
func mustEncodeState(t *testing.T, state oauthState) string {
	t.Helper()
	b, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("mustEncodeState: %v", err)
	}
	return base64.URLEncoding.EncodeToString(b)
}

// ---------------------------------------------------------------------------
// generateNonce
// ---------------------------------------------------------------------------

func TestGenerateNonce(t *testing.T) {
	t.Parallel()

	n1, err := generateNonce()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// base64.URLEncoding of 16 bytes = 24 characters (including padding).
	if len(n1) != 24 {
		t.Errorf("length: got %d, want 24", len(n1))
	}

	n2, _ := generateNonce()
	if n1 == n2 {
		t.Error("two consecutive nonces should differ")
	}
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

func TestLogin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		query        string
		wantRedirect string // expected state.Redirect value
	}{
		{
			name:         "no redirect param defaults to /",
			query:        "",
			wantRedirect: "/",
		},
		{
			name:         "redirect param is preserved in state",
			query:        "?redirect=/dashboard",
			wantRedirect: "/dashboard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := newTestHandler()
			req := httptest.NewRequest(http.MethodGet, "/oauth2/login"+tt.query, nil)
			rr := httptest.NewRecorder()
			h.Login(rr, req)

			if rr.Code != http.StatusFound {
				t.Fatalf("status: got %d, want %d", rr.Code, http.StatusFound)
			}

			// Decode the state embedded in the redirect URL.
			loc, err := url.Parse(rr.Header().Get("Location"))
			if err != nil {
				t.Fatalf("parse Location: %v", err)
			}
			stateBytes, err := base64.URLEncoding.DecodeString(loc.Query().Get("state"))
			if err != nil {
				t.Fatalf("decode state: %v", err)
			}
			var state oauthState
			if err := json.Unmarshal(stateBytes, &state); err != nil {
				t.Fatalf("unmarshal state: %v", err)
			}

			if state.Redirect != tt.wantRedirect {
				t.Errorf("state.Redirect: got %q, want %q", state.Redirect, tt.wantRedirect)
			}

			// The oauth_state cookie must be set and its value must equal the nonce in state.
			var stateCookie *http.Cookie
			for _, c := range rr.Result().Cookies() {
				if c.Name == stateCookieName {
					stateCookie = c
					break
				}
			}
			if stateCookie == nil {
				t.Fatal("oauth_state cookie not set")
			}
			if stateCookie.Value != state.Nonce {
				t.Errorf("nonce: cookie=%q state=%q — should be equal", stateCookie.Value, state.Nonce)
			}
		})
	}
}

func TestLogin_SecureCookieFlag(t *testing.T) {
	t.Parallel()

	t.Run("secureCookies=true sets Secure flag", func(t *testing.T) {
		t.Parallel()

		h := newTestHandler()
		h.secureCookies = true
		req := httptest.NewRequest(http.MethodGet, "/oauth2/login", nil)
		rr := httptest.NewRecorder()
		h.Login(rr, req)

		for _, c := range rr.Result().Cookies() {
			if c.Name == stateCookieName {
				if !c.Secure {
					t.Error("expected Secure flag on oauth_state cookie, got false")
				}
				return
			}
		}
		t.Fatal("oauth_state cookie not found")
	})

	t.Run("secureCookies=false omits Secure flag", func(t *testing.T) {
		t.Parallel()

		h := newTestHandler() // secureCookies defaults to false
		req := httptest.NewRequest(http.MethodGet, "/oauth2/login", nil)
		rr := httptest.NewRecorder()
		h.Login(rr, req)

		for _, c := range rr.Result().Cookies() {
			if c.Name == stateCookieName {
				if c.Secure {
					t.Error("expected no Secure flag on oauth_state cookie, got true")
				}
				return
			}
		}
		t.Fatal("oauth_state cookie not found")
	})
}

// ---------------------------------------------------------------------------
// Callback — error paths that do not require a real token exchange
// ---------------------------------------------------------------------------

func TestCallback_ErrorCases(t *testing.T) {
	t.Parallel()

	nonce := "test-nonce"
	validState := mustEncodeState(t, oauthState{Nonce: nonce, Redirect: "/"})
	notJSON := base64.URLEncoding.EncodeToString([]byte("not-json"))

	tests := []struct {
		name       string
		rawURL     string
		cookie     *http.Cookie
		wantStatus int
		wantBody   string
	}{
		{
			name:       "missing state parameter",
			rawURL:     "/oauth2/callback",
			wantStatus: http.StatusBadRequest,
			wantBody:   "missing state parameter",
		},
		{
			name:       "state is not valid base64",
			rawURL:     "/oauth2/callback?state=" + url.QueryEscape("!!!not-base64!!!"),
			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid state parameter",
		},
		{
			name:       "state is valid base64 but not JSON",
			rawURL:     "/oauth2/callback?state=" + url.QueryEscape(notJSON),
			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid state parameter",
		},
		{
			name:       "oauth_state cookie missing",
			rawURL:     "/oauth2/callback?state=" + url.QueryEscape(validState),
			wantStatus: http.StatusBadRequest,
			wantBody:   "state mismatch",
		},
		{
			name:       "nonce mismatch between cookie and state",
			rawURL:     "/oauth2/callback?state=" + url.QueryEscape(validState),
			cookie:     &http.Cookie{Name: stateCookieName, Value: "wrong-nonce"},
			wantStatus: http.StatusBadRequest,
			wantBody:   "state mismatch",
		},
		{
			name:       "missing code parameter",
			rawURL:     "/oauth2/callback?state=" + url.QueryEscape(validState),
			cookie:     &http.Cookie{Name: stateCookieName, Value: nonce},
			wantStatus: http.StatusBadRequest,
			wantBody:   "missing code parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := &Handler{}
			req := httptest.NewRequest(http.MethodGet, tt.rawURL, nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			rr := httptest.NewRecorder()
			h.Callback(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", rr.Code, tt.wantStatus)
			}
			if !strings.Contains(rr.Body.String(), tt.wantBody) {
				t.Errorf("body %q does not contain %q", rr.Body.String(), tt.wantBody)
			}
		})
	}
}
