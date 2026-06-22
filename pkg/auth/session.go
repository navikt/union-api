package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

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

// VerifySessionToken verifies the HMAC signature and expiry of a session token
// and returns the authenticated Principal if valid.
func VerifySessionToken(secret []byte, token string) (*Principal, error) {
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

	return &Principal{Email: claims.Email, Name: claims.Name}, nil
}
