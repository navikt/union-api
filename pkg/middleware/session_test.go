package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/navikt/union-api/pkg/auth"
	"github.com/navikt/union-api/pkg/config"
)

var testSecret = []byte("supersecrettestkey1234567890abcd")

// ---------------------------------------------------------------------------
// PrincipalFromContext
// ---------------------------------------------------------------------------

func TestPrincipalFromContext(t *testing.T) {
	t.Parallel()

	t.Run("returns principal when present", func(t *testing.T) {
		t.Parallel()

		want := &auth.Principal{Email: "alice@nav.no", Name: "Alice"}
		ctx := context.WithValue(context.Background(), principalKey, want)

		got, ok := PrincipalFromContext(ctx)
		if !ok {
			t.Fatal("ok: got false, want true")
		}
		if got.Email != want.Email || got.Name != want.Name {
			t.Errorf("principal: got %+v, want %+v", got, want)
		}
	})

	t.Run("returns nil and false when absent", func(t *testing.T) {
		t.Parallel()

		got, ok := PrincipalFromContext(context.Background())
		if ok {
			t.Error("ok: got true, want false")
		}
		if got != nil {
			t.Errorf("principal: got %+v, want nil", got)
		}
	})
}

// ---------------------------------------------------------------------------
// NewSessionMiddleware
// ---------------------------------------------------------------------------

func TestNewSessionMiddleware_DevMode(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{DevMode: true}

	var gotPrincipal *auth.Principal
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPrincipal, _ = PrincipalFromContext(r.Context())
	})

	// A corrupt session cookie must be ignored entirely in dev mode.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "garbage"})
	rr := httptest.NewRecorder()
	NewSessionMiddleware(cfg)(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rr.Code, http.StatusOK)
	}
	if gotPrincipal == nil {
		t.Fatal("expected stub principal in context, got nil")
	}
	if gotPrincipal.Email != "dev.user@nav.no" {
		t.Errorf("email: got %q, want %q", gotPrincipal.Email, "dev.user@nav.no")
	}
	if gotPrincipal.Name != "Dev User" {
		t.Errorf("name: got %q, want %q", gotPrincipal.Name, "Dev User")
	}
}

func TestNewSessionMiddleware_ProdMode(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{SessionSecret: string(testSecret)}

	validToken, _ := auth.CreateSessionToken(testSecret, "alice@nav.no", "Alice", time.Now().Add(time.Hour))

	tests := []struct {
		name           string
		cookie         *http.Cookie
		wantStatus     int
		wantLoginRedir bool
		wantEmail      string
	}{
		{
			name:           "no session cookie redirects to login",
			cookie:         nil,
			wantStatus:     http.StatusFound,
			wantLoginRedir: true,
		},
		{
			name:           "invalid token redirects to login",
			cookie:         &http.Cookie{Name: "session", Value: "bad.token"},
			wantStatus:     http.StatusFound,
			wantLoginRedir: true,
		},
		{
			name:       "valid token injects principal and calls next",
			cookie:     &http.Cookie{Name: "session", Value: validToken},
			wantStatus: http.StatusOK,
			wantEmail:  "alice@nav.no",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var gotPrincipal *auth.Principal
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPrincipal, _ = PrincipalFromContext(r.Context())
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			rr := httptest.NewRecorder()
			NewSessionMiddleware(cfg)(next).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", rr.Code, tt.wantStatus)
			}
			if tt.wantLoginRedir {
				loc := rr.Header().Get("Location")
				if !strings.HasPrefix(loc, "/oauth2/login?redirect=") {
					t.Errorf("Location: got %q, expected /oauth2/login?redirect=... prefix", loc)
				}
			}
			if tt.wantEmail != "" {
				if gotPrincipal == nil {
					t.Fatal("expected principal in context, got nil")
				}
				if gotPrincipal.Email != tt.wantEmail {
					t.Errorf("email: got %q, want %q", gotPrincipal.Email, tt.wantEmail)
				}
			}
		})
	}
}

func TestRedirectToLogin_EncodesRequestURI(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{SessionSecret: string(testSecret)}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequest(http.MethodGet, "/serviceaccounts?foo=bar", nil)
	rr := httptest.NewRecorder()
	NewSessionMiddleware(cfg)(next).ServeHTTP(rr, req)

	want := "/oauth2/login?redirect=%2Fserviceaccounts%3Ffoo%3Dbar"
	loc := rr.Header().Get("Location")
	if loc != want {
		t.Errorf("Location: got %q, want %q", loc, want)
	}
}
