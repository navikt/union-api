package auth

import (
	"strings"
	"testing"
	"time"
)

var testSecret = []byte("supersecrettestkey1234567890abcd")

func TestCreateAndVerifySessionToken_RoundTrip(t *testing.T) {
	t.Parallel()

	expiry := time.Now().Add(time.Hour)
	token, err := CreateSessionToken(testSecret, "alice@nav.no", "Alice", expiry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	principal, err := VerifySessionToken(testSecret, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if principal.Email != "alice@nav.no" {
		t.Errorf("email: got %q, want %q", principal.Email, "alice@nav.no")
	}
	if principal.Name != "Alice" {
		t.Errorf("name: got %q, want %q", principal.Name, "Alice")
	}
}

func TestVerifySessionToken_Errors(t *testing.T) {
	t.Parallel()

	validToken, _ := CreateSessionToken(testSecret, "alice@nav.no", "Alice", time.Now().Add(time.Hour))
	expiredToken, _ := CreateSessionToken(testSecret, "alice@nav.no", "Alice", time.Now().Add(-time.Minute))
	parts := strings.SplitN(validToken, ".", 2)

	tests := []struct {
		name    string
		secret  []byte
		token   string
		wantErr string
	}{
		{
			name:    "no dot separator",
			secret:  testSecret,
			token:   "nodottoken",
			wantErr: "invalid token format",
		},
		{
			// Swap in a different payload — the original signature no longer covers it.
			name:    "tampered payload",
			secret:  testSecret,
			token:   "dGFtcGVyZWQ." + parts[1], // "tampered" in RawURLEncoding
			wantErr: "invalid signature",
		},
		{
			// Keep the original payload but replace the signature.
			name:    "tampered signature",
			secret:  testSecret,
			token:   parts[0] + ".dGFtcGVyZWRzaWc", // "tamperedsig" in RawURLEncoding
			wantErr: "invalid signature",
		},
		{
			// Correct token but verified with a different secret.
			name:    "wrong secret",
			secret:  []byte("different-secret-123456789abcdef"),
			token:   validToken,
			wantErr: "invalid signature",
		},
		{
			name:    "expired token",
			secret:  testSecret,
			token:   expiredToken,
			wantErr: "session expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := VerifySessionToken(tt.secret, tt.token)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
